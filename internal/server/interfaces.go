package server

import (
	"context"
	"net/http"

	"github.com/albedosehen/rwwwrse/internal/tls"
)

// Server provides a common interface for HTTP and HTTPS servers.
type Server interface {
	// Start begins listening and serving requests.
	// This is a blocking call that runs until the server is stopped.
	Start(ctx context.Context) error

	// Stop gracefully shuts down the server.
	// It waits for active connections to finish within the timeout.
	Stop(ctx context.Context) error

	// ListenAddr returns the address the server is listening on.
	ListenAddr() string

	// Handler returns the HTTP handler for this server.
	Handler() http.Handler
}

// HTTPServer represents an HTTP server.
type HTTPServer interface {
	Server

	// IsHTTPS returns false for HTTP servers.
	IsHTTPS() bool
}

// HTTPSServer represents an HTTPS server with TLS capabilities.
type HTTPSServer interface {
	Server

	// IsHTTPS returns true for HTTPS servers.
	IsHTTPS() bool

	// GetCertificate returns the current TLS certificate function.
	GetCertificate() func(*http.Request) (*http.Request, error)
}

// ServerManager manages multiple servers (HTTP and HTTPS).
type ServerManager interface {
	// AddServer adds a server to be managed.
	AddServer(name string, server Server) error

	// RemoveServer removes a server from management.
	RemoveServer(name string) error

	// StartAll starts all managed servers.
	StartAll(ctx context.Context) error

	// StopAll gracefully stops all managed servers.
	StopAll(ctx context.Context) error

	// GetServer returns a server by name.
	GetServer(name string) (Server, bool)

	// ListServers returns all server names.
	ListServers() []string
}

// ServerConfig represents server configuration.
type ServerConfig struct {
	// Basic server settings
	Host string
	Port int

	// Timeout settings
	ReadTimeout    string
	WriteTimeout   string
	IdleTimeout    string
	RequestTimeout string

	// TLS settings
	TLSEnabled  bool
	TLSPort     int
	TLSCertFile string
	TLSKeyFile  string

	// Graceful shutdown settings
	ShutdownTimeout string

	// HTTP/2 settings
	HTTP2Enabled bool

	// Additional settings
	MaxHeaderBytes    int
	KeepAlivesEnabled bool
}

// ServerStats represents server runtime statistics.
type ServerStats struct {
	// Connection statistics
	ActiveConnections int64
	TotalConnections  int64
	TotalRequests     int64

	// Error statistics
	ConnectionErrors int64
	RequestErrors    int64

	// Performance statistics
	AverageResponseTime float64
	RequestsPerSecond   float64

	// Server state
	IsRunning bool
	UpTime    string
	StartTime string
}

// HealthChecker provides health checking for servers.
type HealthChecker interface {
	// IsHealthy returns true if the server is healthy.
	IsHealthy(ctx context.Context) bool

	// HealthDetails returns detailed health information.
	HealthDetails(ctx context.Context) map[string]interface{}
}

// ServerMetrics provides metrics collection for servers.
type ServerMetrics interface {
	// RecordRequest records a request metric.
	RecordRequest(method, path string, statusCode int, duration float64)

	// RecordConnection records a connection metric.
	RecordConnection(action string) // "open", "close"

	// RecordError records an error metric.
	RecordError(errorType, context string)

	// GetStats returns current server statistics.
	GetStats() ServerStats
}

// RequestContext provides per-request context and utilities.
type RequestContext interface {
	// RequestID returns the unique request identifier.
	RequestID() string

	// ClientIP returns the client IP address.
	ClientIP() string

	// UserAgent returns the User-Agent header.
	UserAgent() string

	// StartTime returns when the request started processing.
	StartTime() string

	// SetValue sets a value in the request context.
	SetValue(key string, value interface{})

	// GetValue gets a value from the request context.
	GetValue(key string) (interface{}, bool)
}

// MiddlewareFunc represents HTTP middleware.
type MiddlewareFunc func(http.Handler) http.Handler

// ServerBuilder provides a fluent interface for building servers.
type ServerBuilder interface {
	// WithHandler sets the HTTP handler.
	WithHandler(handler http.Handler) ServerBuilder

	// WithMiddleware adds middleware to the server.
	WithMiddleware(middleware ...MiddlewareFunc) ServerBuilder

	// WithTLS enables TLS for the server.
	WithTLS(certManager tls.Manager) ServerBuilder

	// WithMetrics enables metrics collection.
	WithMetrics(metrics ServerMetrics) ServerBuilder

	// WithHealthChecker sets the health checker.
	WithHealthChecker(checker HealthChecker) ServerBuilder

	// WithConfig sets the server configuration.
	WithConfig(config ServerConfig) ServerBuilder

	// Build creates and returns the configured server.
	Build() (Server, error)
}
