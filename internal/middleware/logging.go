// Package middleware implements request logging middleware for observability.
package middleware

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"slices"
	"time"

	"github.com/albedosehen/rwwwrse/internal/observability"
)

// LoggingConfig holds configuration for logging middleware.
type LoggingConfig struct {
	// LogRequests enables request logging.
	LogRequests bool

	// LogResponses enables response logging.
	LogResponses bool

	// LogRequestBody enables request body logging.
	LogRequestBody bool

	// LogResponseBody enables response body logging.
	LogResponseBody bool

	// MaxBodySize limits the size of logged request/response bodies.
	MaxBodySize int64

	// ExcludePaths contains paths to exclude from logging.
	ExcludePaths []string

	// ExcludeUserAgents contains user agents to exclude from logging.
	ExcludeUserAgents []string

	// SkipSuccessfulRequests skips logging for successful requests (2xx status).
	SkipSuccessfulRequests bool
}

// DefaultLoggingConfig returns a default logging configuration.
func DefaultLoggingConfig() LoggingConfig {
	return LoggingConfig{
		LogRequests:            true,
		LogResponses:           true,
		LogRequestBody:         false,
		LogResponseBody:        false,
		MaxBodySize:            1024 * 1024, // 1MB
		ExcludePaths:           []string{"/health", "/metrics", "/favicon.ico"},
		ExcludeUserAgents:      []string{},
		SkipSuccessfulRequests: false,
	}
}

// VerboseLoggingConfig returns a verbose logging configuration.
func VerboseLoggingConfig() LoggingConfig {
	return LoggingConfig{
		LogRequests:            true,
		LogResponses:           true,
		LogRequestBody:         true,
		LogResponseBody:        true,
		MaxBodySize:            1024 * 1024, // 1MB
		ExcludePaths:           []string{"/health"},
		ExcludeUserAgents:      []string{},
		SkipSuccessfulRequests: false,
	}
}

// loggingMiddleware implements request/response logging middleware.
type loggingMiddleware struct {
	config  LoggingConfig
	logger  observability.Logger
	metrics observability.MetricsCollector
}

// NewLoggingMiddleware creates a new logging middleware with default config.
func NewLoggingMiddleware(
	logger observability.Logger,
	metrics observability.MetricsCollector,
) Middleware {
	if logger == nil || metrics == nil {
		return nil
	}
	return NewLoggingMiddlewareWithConfig(
		DefaultLoggingConfig(),
		logger,
		metrics,
	)
}

// NewLoggingMiddlewareWithConfig creates a new logging middleware with custom config.
func NewLoggingMiddlewareWithConfig(
	config LoggingConfig,
	logger observability.Logger,
	metrics observability.MetricsCollector,
) Middleware {
	if logger == nil || metrics == nil {
		return nil
	}
	return &loggingMiddleware{
		config:  config,
		logger:  logger,
		metrics: metrics,
	}
}

// Wrap implements the Middleware interface.
func (m *loggingMiddleware) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if we should skip logging for this request
		if m.shouldSkipLogging(r) {
			next.ServeHTTP(w, r)
			return
		}

		startTime := time.Now()
		requestID := GetRequestID(r.Context())

		// Log incoming request
		if m.config.LogRequests && m.logger != nil {
			m.logIncomingRequest(r, requestID)
		}

		// Wrap response writer to capture response data
		wrapped := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
			body:           &bytes.Buffer{},
			config:         m.config,
		}

		// Execute the request
		next.ServeHTTP(wrapped, r)

		duration := time.Since(startTime)

		// Log outgoing response
		if m.config.LogResponses && m.logger != nil {
			m.logOutgoingResponse(r, wrapped, requestID, duration)
		}

		// Record metrics
		if m.metrics != nil {
			// TODO: Add HTTP request metrics when interface is extended
			// m.metrics.RecordHTTPRequest(r.Method, wrapped.statusCode, duration)
			_ = m.metrics // Acknowledge metrics is available but not used yet
		}
	})
}

// shouldSkipLogging determines if logging should be skipped for this request.
func (m *loggingMiddleware) shouldSkipLogging(r *http.Request) bool {
	// Check excluded paths
	if slices.Contains(m.config.ExcludePaths, r.URL.Path) {
		return true
	}

	// Check excluded user agents
	userAgent := r.Header.Get("User-Agent")
	return slices.Contains(m.config.ExcludeUserAgents, userAgent)
}

// logIncomingRequest logs the incoming HTTP request.
func (m *loggingMiddleware) logIncomingRequest(r *http.Request, requestID string) {
	fields := []observability.Field{
		observability.String("request_id", requestID),
		observability.String("method", r.Method),
		observability.String("path", r.URL.Path),
		observability.String("query", r.URL.RawQuery),
		observability.String("remote_addr", r.RemoteAddr),
		observability.String("user_agent", r.Header.Get("User-Agent")),
		observability.String("referer", r.Header.Get("Referer")),
		observability.String("host", r.Host),
		observability.String("proto", r.Proto),
		observability.Int64("content_length", r.ContentLength),
	}

	// Add forwarded headers
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		fields = append(fields, observability.String("x_forwarded_for", xff))
	}
	if xrealip := r.Header.Get("X-Real-IP"); xrealip != "" {
		fields = append(fields, observability.String("x_real_ip", xrealip))
	}

	// Log request body if enabled
	if m.config.LogRequestBody && r.Body != nil && r.ContentLength > 0 && r.ContentLength <= m.config.MaxBodySize {
		body, err := m.readAndReplaceBody(r)
		if err == nil && len(body) > 0 {
			fields = append(fields, observability.String("request_body", string(body)))
		}
	}

	m.logger.Info(r.Context(), "HTTP request received", fields...)
}

// logOutgoingResponse logs the outgoing HTTP response.
func (m *loggingMiddleware) logOutgoingResponse(r *http.Request, wrapped *responseWriter, requestID string, duration time.Duration) {
	// Skip logging successful requests if configured
	if m.config.SkipSuccessfulRequests && wrapped.statusCode >= 200 && wrapped.statusCode < 300 {
		return
	}

	fields := []observability.Field{
		observability.String("request_id", requestID),
		observability.String("method", r.Method),
		observability.String("path", r.URL.Path),
		observability.Int("status_code", wrapped.statusCode),
		observability.Duration("duration", duration),
		observability.Int64("response_size", int64(wrapped.body.Len())),
	}

	// Log response body if enabled
	if m.config.LogResponseBody && wrapped.body.Len() > 0 && int64(wrapped.body.Len()) <= m.config.MaxBodySize {
		fields = append(fields, observability.String("response_body", wrapped.body.String()))
	}

	// Choose log level based on status code
	switch {
	case wrapped.statusCode >= 500:
		m.logger.Error(r.Context(), fmt.Errorf("server error: status %d", wrapped.statusCode), "HTTP request completed with server error", fields...)
	case wrapped.statusCode >= 400:
		m.logger.Warn(r.Context(), "HTTP request completed with client error", fields...)
	default:
		m.logger.Info(r.Context(), "HTTP request completed", fields...)
	}
}

// readAndReplaceBody reads the request body and replaces it for downstream handlers.
func (m *loggingMiddleware) readAndReplaceBody(r *http.Request) ([]byte, error) {
	if r.Body == nil {
		return nil, nil
	}

	// Read the body
	body, err := io.ReadAll(io.LimitReader(r.Body, m.config.MaxBodySize))
	if err != nil {
		return nil, err
	}

	// Close the original body
	r.Body.Close()

	// Replace the body with a new reader
	r.Body = io.NopCloser(bytes.NewReader(body))

	return body, nil
}

// responseWriter wraps http.ResponseWriter to capture response data.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	body       *bytes.Buffer
	config     LoggingConfig
}

// WriteHeader captures the status code.
func (rw *responseWriter) WriteHeader(statusCode int) {
	rw.statusCode = statusCode
	rw.ResponseWriter.WriteHeader(statusCode)
}

// Write captures the response body and writes to the underlying writer.
func (rw *responseWriter) Write(data []byte) (int, error) {
	// Capture response body if logging is enabled and within size limits
	if rw.config.LogResponseBody && int64(rw.body.Len()+len(data)) <= rw.config.MaxBodySize {
		rw.body.Write(data)
	}

	return rw.ResponseWriter.Write(data)
}

// Hijack implements http.Hijacker interface.
func (rw *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := rw.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("responseWriter does not implement http.Hijacker")
	}
	return hijacker.Hijack()
}

// Push implements http.Pusher interface for HTTP/2 server push.
func (rw *responseWriter) Push(target string, opts *http.PushOptions) error {
	pusher, ok := rw.ResponseWriter.(http.Pusher)
	if !ok {
		return fmt.Errorf("responseWriter does not implement http.Pusher")
	}
	return pusher.Push(target, opts)
}

// Flush implements http.Flusher interface.
func (rw *responseWriter) Flush() {
	flusher, ok := rw.ResponseWriter.(http.Flusher)
	if ok {
		flusher.Flush()
	}
}

// accessLogMiddleware provides simple access log functionality.
type accessLogMiddleware struct {
	logger  observability.Logger
	metrics observability.MetricsCollector
}

// NewAccessLogMiddleware creates a simple access log middleware.
func NewAccessLogMiddleware(
	logger observability.Logger,
	metrics observability.MetricsCollector,
) Middleware {
	return &accessLogMiddleware{
		logger:  logger,
		metrics: metrics,
	}
}

// Wrap implements the Middleware interface with simple access logging.
func (m *accessLogMiddleware) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()
		requestID := GetRequestID(r.Context())

		// Wrap response writer to capture status code
		wrapped := &simpleResponseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// Execute the request
		next.ServeHTTP(wrapped, r)

		duration := time.Since(startTime)

		// Log access
		if m.logger != nil {
			m.logger.Info(r.Context(), "HTTP access",
				observability.String("request_id", requestID),
				observability.String("method", r.Method),
				observability.String("path", r.URL.Path),
				observability.String("remote_addr", r.RemoteAddr),
				observability.Int("status", wrapped.statusCode),
				observability.Duration("duration", duration),
				observability.String("user_agent", r.Header.Get("User-Agent")),
			)
		}

		// Record metrics
		if m.metrics != nil {
			// TODO: Add HTTP request metrics when interface is extended
			// m.metrics.RecordHTTPRequest(r.Method, wrapped.statusCode, duration)
			_ = m.metrics // Acknowledge metrics is available but not used yet
		}
	})
}

// simpleResponseWriter wraps http.ResponseWriter to capture only status code.
type simpleResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader captures the status code.
func (rw *simpleResponseWriter) WriteHeader(statusCode int) {
	rw.statusCode = statusCode
	rw.ResponseWriter.WriteHeader(statusCode)
}

// Hijack implements http.Hijacker interface.
func (rw *simpleResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := rw.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("responseWriter does not implement http.Hijacker")
	}
	return hijacker.Hijack()
}

// Push implements http.Pusher interface.
func (rw *simpleResponseWriter) Push(target string, opts *http.PushOptions) error {
	pusher, ok := rw.ResponseWriter.(http.Pusher)
	if !ok {
		return fmt.Errorf("responseWriter does not implement http.Pusher")
	}
	return pusher.Push(target, opts)
}

// Flush implements http.Flusher interface.
func (rw *simpleResponseWriter) Flush() {
	flusher, ok := rw.ResponseWriter.(http.Flusher)
	if ok {
		flusher.Flush()
	}
}
