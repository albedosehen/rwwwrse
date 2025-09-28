package proxy

import (
	"context"
	"net/http"
	"net/url"
)

// Router defines request routing based on host header.
// It maintains a mapping between host names and their corresponding backend services.
type Router interface {
	// Route finds the appropriate backend for the given host.
	// Returns an error if no backend is configured for the host.
	Route(ctx context.Context, host string) (Backend, error)

	// Register associates a host with a backend service.
	// If the host is already registered, it updates the backend.
	Register(host string, backend Backend) error

	// Unregister removes a host from the routing table.
	// Returns an error if the host is not currently registered.
	Unregister(host string) error

	// Backends returns a copy of all currently registered backends.
	// The map is keyed by host name.
	Backends() map[string]Backend
}

// Backend represents a proxy backend service with health monitoring capabilities.
// It encapsulates the target URL, transport configuration, and health status.
type Backend interface {
	// URL returns the backend service URL.
	URL() *url.URL

	// Transport returns the HTTP transport for this backend.
	// This allows for custom connection pooling and timeout configuration.
	Transport() http.RoundTripper

	// IsHealthy checks if the backend is currently healthy.
	// This should be a fast, cached check rather than a live health probe.
	IsHealthy(ctx context.Context) bool

	// Name returns a human-readable identifier for this backend.
	Name() string
}

// ProxyHandler handles the core HTTP proxying logic.
// It processes incoming requests and forwards them to appropriate backends.
type ProxyHandler interface {
	// ServeHTTP processes an HTTP request through the proxy.
	// It performs routing, applies middleware, and forwards to the backend.
	ServeHTTP(w http.ResponseWriter, r *http.Request)

	// Shutdown gracefully shuts down the proxy handler.
	// It waits for existing requests to complete within the context timeout.
	Shutdown(ctx context.Context) error
}

// BackendManager provides lifecycle management for backend services.
type BackendManager interface {
	// AddBackend registers a new backend service.
	AddBackend(ctx context.Context, host string, backend Backend) error

	// RemoveBackend unregisters a backend service.
	RemoveBackend(ctx context.Context, host string) error

	// UpdateBackend modifies an existing backend configuration.
	UpdateBackend(ctx context.Context, host string, backend Backend) error

	// GetBackend retrieves a backend by host name.
	GetBackend(ctx context.Context, host string) (Backend, error)

	// ListBackends returns all registered backends.
	ListBackends(ctx context.Context) (map[string]Backend, error)
}

// RequestContext provides request-scoped information and utilities.
type RequestContext interface {
	// RequestID returns a unique identifier for this request.
	RequestID() string

	// Host returns the target host for this request.
	Host() string

	// Backend returns the backend service handling this request.
	Backend() Backend

	// StartTime returns when request processing began.
	StartTime() int64

	// SetAttribute stores a key-value pair in the request context.
	SetAttribute(key string, value interface{})

	// GetAttribute retrieves a value from the request context.
	GetAttribute(key string) (interface{}, bool)
}

// ConnectionPool manages HTTP connections to backend services.
type ConnectionPool interface {
	// GetConnection returns a connection for the specified backend.
	GetConnection(ctx context.Context, backend Backend) (http.RoundTripper, error)

	// ReleaseConnection returns a connection to the pool.
	ReleaseConnection(backend Backend, transport http.RoundTripper) error

	// Close closes all connections in the pool.
	Close() error

	// Stats returns connection pool statistics.
	Stats() ConnectionPoolStats
}

// ConnectionPoolStats provides metrics about connection pool usage.
type ConnectionPoolStats struct {
	ActiveConnections int
	IdleConnections   int
	TotalConnections  int
	ConnectionsInUse  int
}
