package healthcheck_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/go-chi/chi/v5"
	healthcheck "github.com/pmatteo/chi-healthcheck-middleware"
	"github.com/stretchr/testify/require"
)

func testRequest(t *testing.T, ts *httptest.Server, method, path string, body io.Reader) (*http.Response, string) {

	req, err := http.NewRequest(method, ts.URL+path, body)
	if err != nil {
		t.Fatal(err)
		return nil, ""
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
		return nil, ""
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
		return nil, ""
	}
	defer resp.Body.Close()

	return resp, string(respBody)
}

// Helper function to check that the expected status is 200 OK
func testGiveStatus(t *testing.T, ts *httptest.Server, path string, expectedStatus int) {

	t.Helper()

	resp, _ := testRequest(t, ts, http.MethodGet, path, nil)
	require.Equal(t, expectedStatus, resp.StatusCode, "path: "+path+" should match "+strconv.Itoa(expectedStatus))
}

func Test_HealthCheck_Strict_Routing_Default(t *testing.T) {
	t.Parallel()

	router := chi.NewRouter()

	router.Use(healthcheck.NewHealthChecker())
	router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("Hello World"))
	})

	ts := httptest.NewServer(router)
	defer ts.Close()

	testGiveStatus(t, ts, "/readyz", http.StatusOK)
	testGiveStatus(t, ts, "/livez", http.StatusOK)
	testGiveStatus(t, ts, "/startupz", http.StatusOK)
	testGiveStatus(t, ts, "/readyz/", http.StatusOK)
	testGiveStatus(t, ts, "/livez/", http.StatusOK)
	testGiveStatus(t, ts, "/startupz/", http.StatusOK)
	testGiveStatus(t, ts, "/notDefined/readyz", http.StatusNotFound)
	testGiveStatus(t, ts, "/notDefined/livez", http.StatusNotFound)
	testGiveStatus(t, ts, "/notDefined/startupz", http.StatusNotFound)
}

func Test_HealthCheck_Default(t *testing.T) {
	t.Parallel()

	router := chi.NewRouter()

	router.Use(healthcheck.NewHealthChecker())
	router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("Hello World"))
	})

	ts := httptest.NewServer(router)
	defer ts.Close()

	testGiveStatus(t, ts, "/readyz", http.StatusOK)
	testGiveStatus(t, ts, "/livez", http.StatusOK)
	testGiveStatus(t, ts, "/startupz", http.StatusOK)
	testGiveStatus(t, ts, "/notDefined/readyz", http.StatusNotFound)
	testGiveStatus(t, ts, "/notDefined/livez", http.StatusNotFound)
	testGiveStatus(t, ts, "/notDefined/startupz", http.StatusNotFound)
}

func Test_HealthCheck_Custom(t *testing.T) {
	t.Parallel()

	router := chi.NewRouter()
	c1 := make(chan struct{}, 1)

	router.Use(healthcheck.NewHealthChecker(
		healthcheck.WithEndpointDefaultProbe("/live"),
		healthcheck.WithEndpoint("/ready", func(r *http.Request) bool {
			select {
			case <-c1:
				return true
			default:
				return false
			}
		}),
		healthcheck.WithEndpoint(healthcheck.DefaultStartupEndpoint, func(r *http.Request) bool {
			return false
		}),
	))

	router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("Hello World"))
	})

	ts := httptest.NewServer(router)
	defer ts.Close()

	testGiveStatus(t, ts, "/live", http.StatusOK)
	req := httptest.NewRequest(http.MethodPost, "/live", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	require.Equal(t, http.StatusNotFound, rr.Code)

	req = httptest.NewRequest(http.MethodPost, "/ready", nil)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	require.Equal(t, http.StatusNotFound, rr.Code)

	testGiveStatus(t, ts, "/ready", http.StatusServiceUnavailable)
	testGiveStatus(t, ts, "/startupz", http.StatusServiceUnavailable)

	c1 <- struct{}{}
	testGiveStatus(t, ts, "/ready", http.StatusOK)
}

func Test_HealthCheck_Custom_Nested(t *testing.T) {
	t.Parallel()

	router := chi.NewRouter()
	c1 := make(chan struct{}, 1)

	router.Use(healthcheck.NewHealthChecker(
		healthcheck.WithEndpointDefaultProbe("/probe/live"),
		healthcheck.WithEndpoint("/probe/ready", func(r *http.Request) bool {
			select {
			case <-c1:
				return true
			default:
				return false
			}
		}),
	))

	router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("Hello World"))
	})

	ts := httptest.NewServer(router)
	defer ts.Close()

	testGiveStatus(t, ts, "/probe/live", http.StatusOK)
	testGiveStatus(t, ts, "/probe/live/", http.StatusOK)
	testGiveStatus(t, ts, "/probe/ready", http.StatusServiceUnavailable)
	testGiveStatus(t, ts, "/probe/ready/", http.StatusServiceUnavailable)
	testGiveStatus(t, ts, "/probe/livez", http.StatusNotFound)
	testGiveStatus(t, ts, "/probe/readyz", http.StatusNotFound)
	testGiveStatus(t, ts, "/probe/livez/", http.StatusNotFound)
	testGiveStatus(t, ts, "/probe/readyz/", http.StatusNotFound)
	testGiveStatus(t, ts, "/livez", http.StatusNotFound)
	testGiveStatus(t, ts, "/readyz", http.StatusNotFound)
	testGiveStatus(t, ts, "/readyz/", http.StatusNotFound)
	testGiveStatus(t, ts, "/livez/", http.StatusNotFound)

	c1 <- struct{}{}
	testGiveStatus(t, ts, "/probe/ready", http.StatusOK)
	c1 <- struct{}{}
	testGiveStatus(t, ts, "/probe/ready/", http.StatusOK)
}

func Test_HealthCheck_Next(t *testing.T) {
	t.Parallel()

	router := chi.NewRouter()

	router.Use(
		healthcheck.NewHealthChecker(
			healthcheck.WithNext(func(r *http.Request) bool {
				return true
			}),
		),
	)

	router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("Hello World"))
	})

	ts := httptest.NewServer(router)
	defer ts.Close()

	testGiveStatus(t, ts, "/readyz", http.StatusNotFound)
	testGiveStatus(t, ts, "/livez", http.StatusNotFound)
	testGiveStatus(t, ts, "/startupz", http.StatusNotFound)
}

func Benchmark_HealthCheck(b *testing.B) {
	router := chi.NewRouter()
	router.Use(healthcheck.NewHealthChecker())

	req := httptest.NewRequest(http.MethodGet, healthcheck.DefaultLivenessEndpoint, nil)
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		require.Equal(b, http.StatusOK, rr.Code)
	}
}

func Benchmark_HealthCheck_Parallel(b *testing.B) {
	router := chi.NewRouter()

	router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("Hello World"))
	})

	router.Use(healthcheck.NewHealthChecker())

	req := httptest.NewRequest(http.MethodGet, healthcheck.DefaultLivenessEndpoint, nil)
	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)
			require.Equal(b, http.StatusOK, rr.Code)
		}
	})
}
