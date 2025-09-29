// Package middleware implements CORS (Cross-Origin Resource Sharing) middleware.
package middleware

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/albedosehen/rwwwrse/internal/observability"
)

// CORSConfig holds configuration for CORS middleware.
type CORSConfig struct {
	// AllowedOrigins is a list of origins that may access the resource.
	// Use ["*"] to allow any origin.
	AllowedOrigins []string

	// AllowedMethods is a list of methods the client is allowed to use.
	AllowedMethods []string

	// AllowedHeaders is a list of headers the client is allowed to use.
	AllowedHeaders []string

	// ExposedHeaders indicates which headers are safe to expose.
	ExposedHeaders []string

	// AllowCredentials indicates whether the request can include credentials.
	AllowCredentials bool

	// MaxAge indicates how long the results of a preflight request can be cached.
	MaxAge time.Duration

	// OptionsPassthrough allows preflight requests to be passed to the next handler.
	OptionsPassthrough bool
}

// DefaultCORSConfig returns a secure default CORS configuration.
func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
			http.MethodHead,
			http.MethodOptions,
		},
		AllowedHeaders: []string{
			"Accept",
			"Authorization",
			"Content-Type",
			"X-CSRF-Token",
			"X-Requested-With",
		},
		ExposedHeaders:     []string{},
		AllowCredentials:   false,
		MaxAge:             12 * time.Hour,
		OptionsPassthrough: false,
	}
}

// RestrictiveCORSConfig returns a more restrictive CORS configuration.
func RestrictiveCORSConfig() CORSConfig {
	return CORSConfig{
		AllowedOrigins: []string{}, // Must be explicitly set
		AllowedMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodHead,
			http.MethodOptions,
		},
		AllowedHeaders: []string{
			"Accept",
			"Content-Type",
		},
		ExposedHeaders:     []string{},
		AllowCredentials:   false,
		MaxAge:             1 * time.Hour,
		OptionsPassthrough: false,
	}
}

// corsMiddleware implements CORS middleware.
type corsMiddleware struct {
	config  CORSConfig
	logger  observability.Logger
	metrics observability.MetricsCollector
}

// NewCORSMiddleware creates a new CORS middleware with default config.
func NewCORSMiddleware(
	logger observability.Logger,
	metrics observability.MetricsCollector,
) Middleware {
	return NewCORSMiddlewareWithConfig(
		DefaultCORSConfig(),
		logger,
		metrics,
	)
}

// NewCORSMiddlewareWithConfig creates a new CORS middleware with custom config.
func NewCORSMiddlewareWithConfig(
	config CORSConfig,
	logger observability.Logger,
	metrics observability.MetricsCollector,
) Middleware {
	// Validate and normalize config
	if len(config.AllowedOrigins) == 0 {
		config.AllowedOrigins = []string{"*"}
	}
	if len(config.AllowedMethods) == 0 {
		config.AllowedMethods = []string{http.MethodGet, http.MethodPost, http.MethodOptions}
	}
	if len(config.AllowedHeaders) == 0 {
		config.AllowedHeaders = []string{"Accept", "Content-Type"}
	}

	return &corsMiddleware{
		config:  config,
		logger:  logger,
		metrics: metrics,
	}
}

// Wrap implements the Middleware interface.
func (m *corsMiddleware) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		// Skip CORS handling if no Origin header
		if origin == "" {
			next.ServeHTTP(w, r)
			return
		}

		// Check if origin is allowed
		if !m.isOriginAllowed(origin) {
			if m.logger != nil {
				requestID := GetRequestID(r.Context())
				m.logger.Warn(r.Context(), "CORS request blocked: origin not allowed",
					observability.String("request_id", requestID),
					observability.String("origin", origin),
					observability.String("method", r.Method),
					observability.String("path", r.URL.Path),
				)
			}

			if m.metrics != nil {
				// TODO: Add CORS metrics when interface is extended
				// m.metrics.RecordCORSBlocked(origin, r.Method)
				_ = m.metrics // Acknowledge metrics is available but not used yet
			}

			w.WriteHeader(http.StatusForbidden)
			return
		}

		// Set CORS headers
		m.setCORSHeaders(w, r, origin)

		// Handle preflight requests
		if r.Method == http.MethodOptions {
			m.handlePreflight(w, r)

			if !m.config.OptionsPassthrough {
				w.WriteHeader(http.StatusNoContent)
				return
			}
		}

		// Log successful CORS handling
		if m.logger != nil {
			requestID := GetRequestID(r.Context())
			m.logger.Debug(r.Context(), "CORS headers applied",
				observability.String("request_id", requestID),
				observability.String("origin", origin),
				observability.String("method", r.Method),
				observability.String("path", r.URL.Path),
			)
		}

		next.ServeHTTP(w, r)
	})
}

// isOriginAllowed checks if the given origin is allowed.
func (m *corsMiddleware) isOriginAllowed(origin string) bool {
	for _, allowedOrigin := range m.config.AllowedOrigins {
		if allowedOrigin == "*" {
			return true
		}
		if allowedOrigin == origin {
			return true
		}
		// Support wildcard subdomains (e.g., "*.example.com")
		if strings.HasPrefix(allowedOrigin, "*.") {
			domain := allowedOrigin[2:]
			if strings.HasSuffix(origin, "."+domain) || origin == domain {
				return true
			}
		}
	}
	return false
}

// setCORSHeaders sets the appropriate CORS headers based on the request context.
// It validates the request method against allowed methods and sets headers accordingly.
func (m *corsMiddleware) setCORSHeaders(w http.ResponseWriter, r *http.Request, origin string) {
	headers := w.Header()

	// Validate request method against allowed methods (except for OPTIONS which is handled separately)
	if r.Method != http.MethodOptions && !m.isMethodAllowed(r.Method) {
		// Method not allowed, but we still set basic CORS headers for proper error handling
		if m.logger != nil {
			requestID := GetRequestID(r.Context())
			m.logger.Warn(r.Context(), "CORS request with disallowed method",
				observability.String("request_id", requestID),
				observability.String("origin", origin),
				observability.String("method", r.Method),
				observability.String("path", r.URL.Path),
			)
		}
	}

	// Access-Control-Allow-Origin
	if len(m.config.AllowedOrigins) == 1 && m.config.AllowedOrigins[0] == "*" {
		headers.Set("Access-Control-Allow-Origin", "*")
	} else {
		headers.Set("Access-Control-Allow-Origin", origin)
	}

	// Access-Control-Allow-Credentials
	if m.config.AllowCredentials {
		headers.Set("Access-Control-Allow-Credentials", "true")
	}

	// Access-Control-Expose-Headers
	if len(m.config.ExposedHeaders) > 0 {
		headers.Set("Access-Control-Expose-Headers", strings.Join(m.config.ExposedHeaders, ", "))
	}

	// Set Vary header based on request context and configuration
	m.setVaryHeader(headers, r)
}

// handlePreflight handles preflight OPTIONS requests.
func (m *corsMiddleware) handlePreflight(w http.ResponseWriter, r *http.Request) {
	headers := w.Header()

	// Access-Control-Allow-Methods
	if len(m.config.AllowedMethods) > 0 {
		headers.Set("Access-Control-Allow-Methods", strings.Join(m.config.AllowedMethods, ", "))
	}

	// Access-Control-Allow-Headers
	requestHeaders := r.Header.Get("Access-Control-Request-Headers")
	if requestHeaders != "" {
		if m.areHeadersAllowed(requestHeaders) {
			headers.Set("Access-Control-Allow-Headers", requestHeaders)
		} else {
			headers.Set("Access-Control-Allow-Headers", strings.Join(m.config.AllowedHeaders, ", "))
		}
	} else if len(m.config.AllowedHeaders) > 0 {
		headers.Set("Access-Control-Allow-Headers", strings.Join(m.config.AllowedHeaders, ", "))
	}

	// Access-Control-Max-Age
	if m.config.MaxAge > 0 {
		headers.Set("Access-Control-Max-Age", strconv.Itoa(int(m.config.MaxAge.Seconds())))
	}

	// Log preflight request
	if m.logger != nil {
		requestID := GetRequestID(r.Context())
		requestMethod := r.Header.Get("Access-Control-Request-Method")
		m.logger.Debug(r.Context(), "CORS preflight request handled",
			observability.String("request_id", requestID),
			observability.String("origin", r.Header.Get("Origin")),
			observability.String("requested_method", requestMethod),
			observability.String("requested_headers", requestHeaders),
		)
	}
}

// areHeadersAllowed checks if the requested headers are allowed.
func (m *corsMiddleware) areHeadersAllowed(requestHeaders string) bool {
	if len(m.config.AllowedHeaders) == 0 {
		return true
	}

	headers := strings.Split(requestHeaders, ",")
	for _, header := range headers {
		header = strings.TrimSpace(strings.ToLower(header))
		allowed := false

		for _, allowedHeader := range m.config.AllowedHeaders {
			if strings.ToLower(allowedHeader) == header {
				allowed = true
				break
			}
		}

		if !allowed {
			return false
		}
	}

	return true
}

// isMethodAllowed checks if the given HTTP method is allowed.
func (m *corsMiddleware) isMethodAllowed(method string) bool {
	for _, allowedMethod := range m.config.AllowedMethods {
		if allowedMethod == method {
			return true
		}
	}
	return false
}

// setVaryHeader sets the Vary header based on request context and configuration.
func (m *corsMiddleware) setVaryHeader(headers http.Header, r *http.Request) {
	vary := headers.Get("Vary")
	varyHeaders := []string{}

	// Always vary on Origin for CORS requests
	if vary == "" {
		varyHeaders = append(varyHeaders, "Origin")
	} else if !strings.Contains(vary, "Origin") {
		varyHeaders = append(varyHeaders, vary, "Origin")
	} else {
		// Origin already present, keep existing vary header
		return
	}

	// Add Access-Control-Request-Method to Vary for preflight requests
	if r.Method == http.MethodOptions {
		requestMethod := r.Header.Get("Access-Control-Request-Method")
		if requestMethod != "" && !strings.Contains(vary, "Access-Control-Request-Method") {
			varyHeaders = append(varyHeaders, "Access-Control-Request-Method")
		}

		// Add Access-Control-Request-Headers to Vary for preflight requests
		requestHeaders := r.Header.Get("Access-Control-Request-Headers")
		if requestHeaders != "" && !strings.Contains(vary, "Access-Control-Request-Headers") {
			varyHeaders = append(varyHeaders, "Access-Control-Request-Headers")
		}
	}

	if len(varyHeaders) > 0 {
		headers.Set("Vary", strings.Join(varyHeaders, ", "))
	}
}

// CORSValidator validates CORS configuration.
type CORSValidator struct{}

// NewCORSValidator creates a new CORS validator.
func NewCORSValidator() *CORSValidator {
	return &CORSValidator{}
}

// Validate validates the CORS configuration.
func (v *CORSValidator) Validate(config CORSConfig) error {
	// Validate origins
	if len(config.AllowedOrigins) == 0 {
		return ErrCORSNoOrigins
	}

	// Check for conflicting configuration
	hasWildcard := false
	for _, origin := range config.AllowedOrigins {
		if origin == "*" {
			hasWildcard = true
			break
		}
	}

	if hasWildcard && config.AllowCredentials {
		return ErrCORSWildcardWithCredentials
	}

	// Validate methods
	if len(config.AllowedMethods) == 0 {
		return ErrCORSNoMethods
	}

	for _, method := range config.AllowedMethods {
		if !isValidHTTPMethod(method) {
			return ErrCORSInvalidMethod
		}
	}

	// Validate max age
	if config.MaxAge < 0 {
		return ErrCORSInvalidMaxAge
	}

	return nil
}

// isValidHTTPMethod checks if the method is a valid HTTP method.
func isValidHTTPMethod(method string) bool {
	validMethods := []string{
		http.MethodGet,
		http.MethodHead,
		http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete,
		http.MethodConnect,
		http.MethodOptions,
		http.MethodTrace,
	}

	for _, validMethod := range validMethods {
		if method == validMethod {
			return true
		}
	}

	return false
}

// CORS error definitions.
var (
	ErrCORSNoOrigins               = fmt.Errorf("no allowed origins configured")
	ErrCORSWildcardWithCredentials = fmt.Errorf("wildcard origin cannot be used with credentials")
	ErrCORSNoMethods               = fmt.Errorf("no allowed methods configured")
	ErrCORSInvalidMethod           = fmt.Errorf("invalid HTTP method")
	ErrCORSInvalidMaxAge           = fmt.Errorf("invalid max age: must be non-negative")
)
