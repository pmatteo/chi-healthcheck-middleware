# Chi Healthcheck Middleware

[![Go Report Card](https://goreportcard.com/badge/github.com/pmatteo/chi-healthcheck-middleware)](https://goreportcard.com/report/github.com/pmatteo/chi-healthcheck-middleware)

![GitHub Release](https://img.shields.io/github/v/release/pmatteo/chi-healthcheck-middleware?display_name=tag&style=flat-square)

This Go package provides middleware for an HTTP server to manage health check endpoints, such as liveness, readiness, and startup checks. Inspired by [Fiber's health check middleware](https://github.com/gofiber/fiber), it brings similar functionality to the [`go-chi`](https://github.com/go-chi/chi) package in Go, allowing for easy integration with monitoring tools, load balancers, or orchestrators like Kubernetes.

> Due to `chi`'s design, which directly builds on Go's `net/http` package, this middleware is fully compatible with the standard library, just like any other `chi` middleware.

## Configuring the Middleware

The `NewHealthChecker` function is the primary way to create the health check middleware. It is highly configurable and supports various options that allow you to define custom health check endpoints, use default or custom probes, and specify conditions under which requests should bypass the middleware.

You can customize the middleware behavior by passing one or more configuration functions to `NewHealthChecker`. Here are the available options:

1. **Adding Health Check Endpoints**:

   You can add health check endpoints using `WithEndpointDefaultProbe` or `WithEndpoint`.

   - **`WithEndpointDefaultProbe`**:  
     This function creates a health check endpoint with a default probe function. You provide the URL path for the endpoint (e.g., `/health`), and the middleware will use a simple, built-in function to determine the health status. The default probe always returns a successful (`200 OK`) response, making it suitable for basic checks where no custom logic is required.

   - **`WithEndpoint`**:  
     This function allows you to define a custom health check endpoint with your own logic. You specify both the URL path and a custom probe function. The probe function is a `HealthChecker`, which is a function that takes an `*http.Request` and returns a `bool`. The middleware will call this function to determine if the application is healthy (`true`) or unhealthy (`false`). This option provides greater flexibility, allowing you to tailor the health check logic to your application's specific needs.

2. **Skipping the Middleware Based on Conditions**:

   You can specify conditions under which the request should skip the health check middleware using the `WithNext` function. This function takes a function that receives the incoming `*http.Request` and returns a `bool`. If this function returns `true`, the request will bypass the health check middleware and continue to the next handler in the chain. This is useful when certain requests, such as internal administrative endpoints, should not be subject to health checks.

If no configuration options are provided, the middleware by default define three endpoints: `/livez`, `/readyz` and `/startupz`. All of them hanlded by the `defaultProbe` function (which always returns true). There is no `Next` function defined if not provided.

## Example Usage

To illustrate how to use the `NewHealthChecker` middleware, consider the following example:

```Go
func main() {
  // Define a custom probe function that implements custom logic to check application health
  customProbe := func(r *http.Request) bool {
    // Example logic: return true if healthy, false otherwise
    return true
  }

  // Set up the health check middleware with different configurations
  healthMiddleware := healthcheck.NewHealthChecker(
    healthcheck.WithEndpointDefaultProbe("/health"), // Adds a default health check endpoint at /health
    healthcheck.WithEndpoint("/custom-health", customProbe), // Adds a custom health check endpoint at /custom-health
    healthcheck.WithNext(func(r *http.Request) bool {
    // Skips the middleware if the request path starts with /api prefix
    return strings.HasPrefix(r.URL.Path, "api/")
    }),
  )

 // Define your main HTTP handler
  router := chi.NewRouter()

  router.Use(NewHealthChecker(
    WithEndpointDefaultProbe(DefaultLivenessEndpoint),
    WithEndpointDefaultProbe(DefaultReadinessEndpoint),
    WithEndpointDefaultProbe(DefaultStartupEndpoint),
  ))
  router.Get("/", func(w http.ResponseWriter, r *http.Request) {
    _, _ = w.Write([]byte("Hello World"))
  })

  // Start the HTTP server
  http.ListenAndServe(":8080", router)
}
```

In this example:

- A default health check endpoint is added at /health using WithEndpointDefaultProbe.
- A custom endpoint is added at /custom-health with a custom probe function that implements specific health-check logic.
The middleware is configured to skip checks for requests matching the path /skip-health-check.
- By using these configuration functions, you can fully control how health checks are handled in your application, ensuring that they align with your specific operational requirements.
