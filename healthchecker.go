package healthcheck

import (
	"net/http"
	"strings"
)

// HealthChecker defines a function to check the liveness or readiness of the application.
type HealthChecker func(r *http.Request) bool

// NewHealthChecker endpoint middleware useful to setting up probeness endpoint (liveness, readness..)
// that load balancers or external services can leverage to.
func NewHealthChecker(configFuncs ...configFunc) func(http.Handler) http.Handler {

	cfg := &config{}
	for _, configFunc := range configFuncs {
		configFunc(cfg)
	}

	if len(cfg.Endpoints) == 0 {
		cfg.Endpoints = []Endpoint{
			{Path: DefaultLivenessEndpoint, Probe: defaultProbe},
			{Path: DefaultReadinessEndpoint, Probe: defaultProbe},
			{Path: DefaultStartupEndpoint, Probe: defaultProbe},
		}
	}

	h := func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {

			// If the Next function is defined and returns true, skip this middleware.
			if cfg.Next != nil && cfg.Next(r) {
				next.ServeHTTP(w, r)
				return
			}

			// Only handle GET and HEAD requests.
			if r.Method != http.MethodGet && r.Method != http.MethodHead {
				next.ServeHTTP(w, r)
				return
			}

			trimmedPath := strings.TrimSuffix(r.URL.Path, "/")

			// Iterate over configured endpoints to check if the request matches any endpoint.
			for _, endpoint := range cfg.Endpoints {

				if !strings.EqualFold(trimmedPath, endpoint.Path) {
					continue
				}

				w.Header().Set("Content-Type", "text/plain")

				// Execute the associated probe function.
				if endpoint.Probe(r) {
					w.WriteHeader(http.StatusOK)
				} else {
					w.WriteHeader(http.StatusServiceUnavailable)
				}

				return
			}

			// If no matching endpoint is found, continue to the next handler.
			next.ServeHTTP(w, r)
		}

		return http.HandlerFunc(fn)
	}

	return h
}
