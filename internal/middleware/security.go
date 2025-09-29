// Package middleware implements security headers middleware for enhanced security.
package middleware

import (
	"fmt"
	"net/http"
	"slices"

	"github.com/albedosehen/rwwwrse/internal/observability"
)

// SecurityConfig holds configuration for security headers.
type SecurityConfig struct {
	// Content Security Policy
	ContentSecurityPolicy string

	// Strict Transport Security
	StrictTransportSecurity string

	// X-Frame-Options
	FrameOptions string

	// X-Content-Type-Options
	ContentTypeOptions string

	// X-XSS-Protection
	XSSProtection string

	// Referrer-Policy
	ReferrerPolicy string

	// Permissions-Policy
	PermissionsPolicy string

	// Cross-Origin-Embedder-Policy
	CrossOriginEmbedderPolicy string

	// Cross-Origin-Opener-Policy
	CrossOriginOpenerPolicy string

	// Cross-Origin-Resource-Policy
	CrossOriginResourcePolicy string
}

// DefaultSecurityConfig returns a secure default configuration.
func DefaultSecurityConfig() SecurityConfig {
	return SecurityConfig{
		ContentSecurityPolicy:     "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; font-src 'self' data:; connect-src 'self'; frame-ancestors 'none'",
		StrictTransportSecurity:   "max-age=31536000; includeSubDomains; preload",
		FrameOptions:              "DENY",
		ContentTypeOptions:        "nosniff",
		XSSProtection:             "1; mode=block",
		ReferrerPolicy:            "strict-origin-when-cross-origin",
		PermissionsPolicy:         "accelerometer=(), ambient-light-sensor=(), autoplay=(), battery=(), camera=(), cross-origin-isolated=(), display-capture=(), document-domain=(), encrypted-media=(), execution-while-not-rendered=(), execution-while-out-of-viewport=(), fullscreen=(), geolocation=(), gyroscope=(), keyboard-map=(), magnetometer=(), microphone=(), midi=(), navigation-override=(), payment=(), picture-in-picture=(), publickey-credentials-get=(), screen-wake-lock=(), sync-xhr=(), usb=(), web-share=(), xr-spatial-tracking=()",
		CrossOriginEmbedderPolicy: "require-corp",
		CrossOriginOpenerPolicy:   "same-origin",
		CrossOriginResourcePolicy: "same-origin",
	}
}

// securityHeadersMiddleware implements security headers middleware.
type securityHeadersMiddleware struct {
	config  SecurityConfig
	logger  observability.Logger
	metrics observability.MetricsCollector
}

// NewSecurityHeadersMiddleware creates a new security headers middleware with default config.
func NewSecurityHeadersMiddleware(
	logger observability.Logger,
	metrics observability.MetricsCollector,
) Middleware {
	return NewSecurityHeadersMiddlewareWithConfig(
		DefaultSecurityConfig(),
		logger,
		metrics,
	)
}

// NewSecurityHeadersMiddlewareWithConfig creates a new security headers middleware with custom config.
func NewSecurityHeadersMiddlewareWithConfig(
	config SecurityConfig,
	logger observability.Logger,
	metrics observability.MetricsCollector,
) Middleware {
	return &securityHeadersMiddleware{
		config:  config,
		logger:  logger,
		metrics: metrics,
	}
}

// Wrap implements the Middleware interface.
func (m *securityHeadersMiddleware) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set security headers
		m.setSecurityHeaders(w, r)

		// Record metrics
		if m.metrics != nil {
			// TODO: Add security headers metrics when interface is extended
			// m.metrics.RecordSecurityHeaders(r.Method, r.URL.Path)
			_ = m.metrics // Acknowledge metrics is available but not used yet
		}

		next.ServeHTTP(w, r)
	})
}

// setSecurityHeaders applies all security headers to the response.
func (m *securityHeadersMiddleware) setSecurityHeaders(w http.ResponseWriter, r *http.Request) {
	headers := w.Header()

	// Content Security Policy
	if m.config.ContentSecurityPolicy != "" {
		headers.Set("Content-Security-Policy", m.config.ContentSecurityPolicy)
	}

	// Strict Transport Security (only for HTTPS)
	if r.TLS != nil && m.config.StrictTransportSecurity != "" {
		headers.Set("Strict-Transport-Security", m.config.StrictTransportSecurity)
	}

	// X-Frame-Options
	if m.config.FrameOptions != "" {
		headers.Set("X-Frame-Options", m.config.FrameOptions)
	}

	// X-Content-Type-Options
	if m.config.ContentTypeOptions != "" {
		headers.Set("X-Content-Type-Options", m.config.ContentTypeOptions)
	}

	// X-XSS-Protection
	if m.config.XSSProtection != "" {
		headers.Set("X-XSS-Protection", m.config.XSSProtection)
	}

	// Referrer-Policy
	if m.config.ReferrerPolicy != "" {
		headers.Set("Referrer-Policy", m.config.ReferrerPolicy)
	}

	// Permissions-Policy
	if m.config.PermissionsPolicy != "" {
		headers.Set("Permissions-Policy", m.config.PermissionsPolicy)
	}

	// Cross-Origin-Embedder-Policy
	if m.config.CrossOriginEmbedderPolicy != "" {
		headers.Set("Cross-Origin-Embedder-Policy", m.config.CrossOriginEmbedderPolicy)
	}

	// Cross-Origin-Opener-Policy
	if m.config.CrossOriginOpenerPolicy != "" {
		headers.Set("Cross-Origin-Opener-Policy", m.config.CrossOriginOpenerPolicy)
	}

	// Cross-Origin-Resource-Policy
	if m.config.CrossOriginResourcePolicy != "" {
		headers.Set("Cross-Origin-Resource-Policy", m.config.CrossOriginResourcePolicy)
	}

	// Server header (hide server information)
	headers.Set("Server", "rwwwrse")

	// X-Powered-By header (remove if exists)
	headers.Del("X-Powered-By")

	if m.logger != nil {
		requestID := GetRequestID(r.Context())
		m.logger.Debug(r.Context(), "Security headers applied",
			observability.String("request_id", requestID),
			observability.String("method", r.Method),
			observability.String("path", r.URL.Path),
			observability.Bool("https", r.TLS != nil),
		)
	}
}

// secureHeadersOnlyMiddleware provides a minimal security headers middleware.
type secureHeadersOnlyMiddleware struct {
	logger  observability.Logger
	metrics observability.MetricsCollector
}

// NewSecureHeadersOnlyMiddleware creates a minimal security headers middleware.
func NewSecureHeadersOnlyMiddleware(
	logger observability.Logger,
	metrics observability.MetricsCollector,
) Middleware {
	return &secureHeadersOnlyMiddleware{
		logger:  logger,
		metrics: metrics,
	}
}

// Wrap implements the Middleware interface with minimal security headers.
func (m *secureHeadersOnlyMiddleware) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		headers := w.Header()

		// Essential security headers only
		headers.Set("X-Content-Type-Options", "nosniff")
		headers.Set("X-Frame-Options", "DENY")
		headers.Set("X-XSS-Protection", "1; mode=block")

		// HSTS for HTTPS only
		if r.TLS != nil {
			headers.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		// Hide server information
		headers.Set("Server", "rwwwrse")
		headers.Del("X-Powered-By")

		// Record metrics
		if m.metrics != nil {
			// TODO: Add security headers metrics when interface is extended
			// m.metrics.RecordSecurityHeaders(r.Method, r.URL.Path)
			_ = m.metrics // Acknowledge metrics is available but not used yet
		}

		next.ServeHTTP(w, r)
	})
}

// SecurityHeadersValidator validates security headers configuration.
type SecurityHeadersValidator struct{}

// NewSecurityHeadersValidator creates a new security headers validator.
func NewSecurityHeadersValidator() *SecurityHeadersValidator {
	return &SecurityHeadersValidator{}
}

// Validate validates the security configuration.
func (v *SecurityHeadersValidator) Validate(config SecurityConfig) error {
	// Validate CSP
	if config.ContentSecurityPolicy != "" {
		if err := v.validateCSP(config.ContentSecurityPolicy); err != nil {
			return err
		}
	}

	// Validate HSTS
	if config.StrictTransportSecurity != "" {
		if err := v.validateHSTS(config.StrictTransportSecurity); err != nil {
			return err
		}
	}

	// Validate Frame Options
	if config.FrameOptions != "" {
		if err := v.validateFrameOptions(config.FrameOptions); err != nil {
			return err
		}
	}

	return nil
}

// validateCSP validates Content Security Policy syntax.
func (v *SecurityHeadersValidator) validateCSP(csp string) error {
	// Basic CSP validation - check for required directives
	if csp == "" {
		return nil
	}

	// CSP should contain at least default-src
	if !contains(csp, "default-src") {
		return ErrInvalidCSP
	}

	return nil
}

// validateHSTS validates HTTP Strict Transport Security syntax.
func (v *SecurityHeadersValidator) validateHSTS(hsts string) error {
	if hsts == "" {
		return nil
	}

	// HSTS should contain max-age
	if !contains(hsts, "max-age=") {
		return ErrInvalidHSTS
	}

	return nil
}

// validateFrameOptions validates X-Frame-Options values.
func (v *SecurityHeadersValidator) validateFrameOptions(frameOptions string) error {
	validOptions := []string{"DENY", "SAMEORIGIN"}

	if slices.Contains(validOptions, frameOptions) {
		return nil
	}

	// Check for ALLOW-FROM syntax
	if len(frameOptions) > 10 && frameOptions[:10] == "ALLOW-FROM" {
		return nil
	}

	return ErrInvalidFrameOptions
}

// contains is a helper function to check if a string contains a substring.
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			(len(s) > len(substr) &&
				(s[:len(substr)] == substr ||
					s[len(s)-len(substr):] == substr ||
					containsMiddle(s, substr))))
}

// containsMiddle checks if substring exists in the middle of string.
func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Security error definitions.
var (
	ErrInvalidCSP          = fmt.Errorf("invalid Content Security Policy")
	ErrInvalidHSTS         = fmt.Errorf("invalid HTTP Strict Transport Security")
	ErrInvalidFrameOptions = fmt.Errorf("invalid X-Frame-Options")
)
