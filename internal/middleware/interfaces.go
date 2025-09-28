// Package middleware provides HTTP middleware components for the reverse proxy.
// It includes security headers, rate limiting, CORS, logging, and recovery middleware.
package middleware

import (
	"context"
	"net/http"
	"time"
)

// Middleware represents HTTP middleware that can wrap handlers.
type Middleware interface {
	// Wrap wraps an HTTP handler with middleware functionality.
	Wrap(next http.Handler) http.Handler
}

// Chain represents a collection of middleware that can be applied to handlers.
type Chain interface {
	// Use adds middleware to the chain.
	Use(middleware Middleware) Chain

	// Then applies the middleware chain to a handler.
	Then(handler http.Handler) http.Handler

	// ThenFunc applies the middleware chain to a handler function.
	ThenFunc(handlerFunc http.HandlerFunc) http.Handler
}

// RateLimiter provides rate limiting functionality.
type RateLimiter interface {
	// Allow checks if a request should be allowed based on the key.
	Allow(ctx context.Context, key string) bool

	// Reset resets the rate limit for a specific key.
	Reset(key string) error

	// Stats returns rate limiting statistics for a key.
	Stats(key string) RateLimitStats

	// Cleanup removes expired rate limit entries.
	Cleanup(ctx context.Context) error
}

// RateLimitStats provides statistics about rate limiting.
type RateLimitStats struct {
	Requests   int           `json:"requests"`
	Remaining  int           `json:"remaining"`
	ResetTime  time.Time     `json:"reset_time"`
	RetryAfter time.Duration `json:"retry_after"`
}

// SecurityHeaderProvider adds security headers to HTTP responses.
type SecurityHeaderProvider interface {
	// AddHeaders adds security headers to the response.
	AddHeaders(w http.ResponseWriter, r *http.Request)

	// Configure updates the security policy.
	Configure(policy SecurityPolicy) error
}

// SecurityPolicy defines security header configuration.
type SecurityPolicy struct {
	ContentTypeNosniff      bool   `json:"content_type_nosniff"`
	FrameOptions            string `json:"frame_options"`
	ContentSecurityPolicy   string `json:"content_security_policy"`
	StrictTransportSecurity string `json:"strict_transport_security"`
	ReferrerPolicy          string `json:"referrer_policy"`
	PermissionsPolicy       string `json:"permissions_policy"`
	CrossOriginEmbedder     string `json:"cross_origin_embedder"`
	CrossOriginOpener       string `json:"cross_origin_opener"`
	CrossOriginResource     string `json:"cross_origin_resource"`
}

// CORSProvider handles Cross-Origin Resource Sharing.
type CORSProvider interface {
	// HandleCORS processes CORS headers and preflight requests.
	HandleCORS(w http.ResponseWriter, r *http.Request) bool

	// Configure updates the CORS policy.
	Configure(policy CORSPolicy) error
}

// CORSPolicy defines CORS configuration.
type CORSPolicy struct {
	AllowedOrigins     []string `json:"allowed_origins"`
	AllowedMethods     []string `json:"allowed_methods"`
	AllowedHeaders     []string `json:"allowed_headers"`
	ExposedHeaders     []string `json:"exposed_headers"`
	AllowCredentials   bool     `json:"allow_credentials"`
	MaxAge             int      `json:"max_age"`
	OptionsPassthrough bool     `json:"options_passthrough"`
}

// RequestLogger logs HTTP requests and responses.
type RequestLogger interface {
	// LogRequest logs request details.
	LogRequest(ctx context.Context, r *http.Request)

	// LogResponse logs response details.
	LogResponse(ctx context.Context, r *http.Request, status int, size int64, duration time.Duration)
}

// RecoveryHandler handles panics in HTTP handlers.
type RecoveryHandler interface {
	// HandlePanic handles a panic that occurred during request processing.
	HandlePanic(w http.ResponseWriter, r *http.Request, err interface{})
}

// MiddlewareFunc is an adapter to allow ordinary functions to be used as middleware.
type MiddlewareFunc func(http.Handler) http.Handler

// Wrap implements the Middleware interface.
func (f MiddlewareFunc) Wrap(next http.Handler) http.Handler {
	return f(next)
}

// ContextKey represents keys used in request contexts.
type ContextKey string

const (
	// ContextKeyRequestID is the context key for request IDs.
	ContextKeyRequestID ContextKey = "request_id"

	// ContextKeyStartTime is the context key for request start time.
	ContextKeyStartTime ContextKey = "start_time"

	// ContextKeyUserAgent is the context key for user agent.
	ContextKeyUserAgent ContextKey = "user_agent"

	// ContextKeyRemoteAddr is the context key for remote address.
	ContextKeyRemoteAddr ContextKey = "remote_addr"
)

// GetRequestID extracts the request ID from the context.
func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(ContextKeyRequestID).(string); ok {
		return id
	}
	return ""
}

// GetStartTime extracts the start time from the context.
func GetStartTime(ctx context.Context) time.Time {
	if t, ok := ctx.Value(ContextKeyStartTime).(time.Time); ok {
		return t
	}
	return time.Time{}
}

// WithRequestID adds a request ID to the context.
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, ContextKeyRequestID, requestID)
}

// WithStartTime adds a start time to the context.
func WithStartTime(ctx context.Context, startTime time.Time) context.Context {
	return context.WithValue(ctx, ContextKeyStartTime, startTime)
}
