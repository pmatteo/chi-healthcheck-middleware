package healthcheck

import (
	"net/http"
)

// Default values for health check endpoints
const (
	DefaultLivenessEndpoint  = "/livez"
	DefaultReadinessEndpoint = "/readyz"
	DefaultStartupEndpoint   = "/startupz"
)

// Endpoint represents a single health check endpoint configuration.
type Endpoint struct {
	// Path is the URL path for this health check endpoint.
	Path string

	// Probe is the function used to check the health status for this endpoint.
	Probe HealthChecker
}

// config defines the configuration options for the health check middleware.
type config struct {
	// Next defines a function to skip this middleware when returned true.
	// Optional. Default: nil
	Next func(r *http.Request) bool

	// Endpoints is a list of health check endpoints with their associated probe functions.
	// At least one endpoint is required.
	Endpoints []Endpoint
}

type configFunc func(*config)

func WithEndpointDefaultProbe(path string) configFunc {

	return func(c *config) {
		c.Endpoints = append(c.Endpoints, Endpoint{
			Path:  path,
			Probe: defaultProbe,
		})
	}
}

func WithEndpoint(path string, probe HealthChecker) configFunc {

	return func(c *config) {
		c.Endpoints = append(c.Endpoints, Endpoint{
			Path:  path,
			Probe: probe,
		})
	}
}

func WithNext(n func(r *http.Request) bool) configFunc {

	return func(c *config) {
		c.Next = n
	}
}

// defaultProbe is a default probe function that always returns true.
func defaultProbe(r *http.Request) bool { return true }
