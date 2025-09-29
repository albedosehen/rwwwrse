package middleware

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	testhelpers "github.com/albedosehen/rwwwrse/internal/testing"
)

func TestDefaultSecurityConfig(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "DefaultSecurityConfig_ReturnsSecureDefaults",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			config := DefaultSecurityConfig()

			// Assert
			assert.NotEmpty(t, config.ContentSecurityPolicy)
			assert.Contains(t, config.ContentSecurityPolicy, "default-src 'self'")
			assert.Equal(t, "max-age=31536000; includeSubDomains; preload", config.StrictTransportSecurity)
			assert.Equal(t, "DENY", config.FrameOptions)
			assert.Equal(t, "nosniff", config.ContentTypeOptions)
			assert.Equal(t, "1; mode=block", config.XSSProtection)
			assert.Equal(t, "strict-origin-when-cross-origin", config.ReferrerPolicy)
			assert.NotEmpty(t, config.PermissionsPolicy)
			assert.Equal(t, "require-corp", config.CrossOriginEmbedderPolicy)
			assert.Equal(t, "same-origin", config.CrossOriginOpenerPolicy)
			assert.Equal(t, "same-origin", config.CrossOriginResourcePolicy)
		})
	}
}

func TestNewSecurityHeadersMiddleware(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "NewSecurityHeadersMiddleware_ValidCreation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			logger := testhelpers.NewMockLogger()
			metrics := testhelpers.NewMockMetricsCollector()

			// Act
			middleware := NewSecurityHeadersMiddleware(logger, metrics)

			// Assert
			assert.NotNil(t, middleware)
			assert.Implements(t, (*Middleware)(nil), middleware)
		})
	}
}

func TestNewSecurityHeadersMiddlewareWithConfig(t *testing.T) {
	tests := []struct {
		name   string
		config SecurityConfig
	}{
		{
			name:   "NewSecurityHeadersMiddlewareWithConfig_DefaultConfig",
			config: DefaultSecurityConfig(),
		},
		{
			name: "NewSecurityHeadersMiddlewareWithConfig_CustomConfig",
			config: SecurityConfig{
				ContentSecurityPolicy:     "default-src 'none'",
				StrictTransportSecurity:   "max-age=63072000",
				FrameOptions:              "SAMEORIGIN",
				ContentTypeOptions:        "nosniff",
				XSSProtection:             "0",
				ReferrerPolicy:            "no-referrer",
				PermissionsPolicy:         "geolocation=()",
				CrossOriginEmbedderPolicy: "unsafe-none",
				CrossOriginOpenerPolicy:   "unsafe-none",
				CrossOriginResourcePolicy: "cross-origin",
			},
		},
		{
			name:   "NewSecurityHeadersMiddlewareWithConfig_EmptyConfig",
			config: SecurityConfig{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			logger := testhelpers.NewMockLogger()
			metrics := testhelpers.NewMockMetricsCollector()

			// Act
			middleware := NewSecurityHeadersMiddlewareWithConfig(tt.config, logger, metrics)

			// Assert
			assert.NotNil(t, middleware)
			assert.Implements(t, (*Middleware)(nil), middleware)
		})
	}
}

func TestSecurityHeadersMiddleware_Wrap_AllHeaders(t *testing.T) {
	tests := []struct {
		name            string
		config          SecurityConfig
		isHTTPS         bool
		expectedHeaders map[string]string
	}{
		{
			name:    "AllHeaders_HTTPSRequest_AllHeadersSet",
			config:  DefaultSecurityConfig(),
			isHTTPS: true,
			expectedHeaders: map[string]string{
				"Content-Security-Policy":      DefaultSecurityConfig().ContentSecurityPolicy,
				"Strict-Transport-Security":    "max-age=31536000; includeSubDomains; preload",
				"X-Frame-Options":              "DENY",
				"X-Content-Type-Options":       "nosniff",
				"X-XSS-Protection":             "1; mode=block",
				"Referrer-Policy":              "strict-origin-when-cross-origin",
				"Permissions-Policy":           DefaultSecurityConfig().PermissionsPolicy,
				"Cross-Origin-Embedder-Policy": "require-corp",
				"Cross-Origin-Opener-Policy":   "same-origin",
				"Cross-Origin-Resource-Policy": "same-origin",
			},
		},
		{
			name:    "AllHeaders_HTTPRequest_NoHSTS",
			config:  DefaultSecurityConfig(),
			isHTTPS: false,
			expectedHeaders: map[string]string{
				"Content-Security-Policy":      DefaultSecurityConfig().ContentSecurityPolicy,
				"X-Frame-Options":              "DENY",
				"X-Content-Type-Options":       "nosniff",
				"X-XSS-Protection":             "1; mode=block",
				"Referrer-Policy":              "strict-origin-when-cross-origin",
				"Permissions-Policy":           DefaultSecurityConfig().PermissionsPolicy,
				"Cross-Origin-Embedder-Policy": "require-corp",
				"Cross-Origin-Opener-Policy":   "same-origin",
				"Cross-Origin-Resource-Policy": "same-origin",
			},
		},
		{
			name: "CustomHeaders_HTTPSRequest_CustomValues",
			config: SecurityConfig{
				ContentSecurityPolicy:     "default-src 'none'; script-src 'self'",
				StrictTransportSecurity:   "max-age=63072000; includeSubDomains",
				FrameOptions:              "SAMEORIGIN",
				ContentTypeOptions:        "nosniff",
				XSSProtection:             "0",
				ReferrerPolicy:            "no-referrer",
				PermissionsPolicy:         "geolocation=(), camera=()",
				CrossOriginEmbedderPolicy: "unsafe-none",
				CrossOriginOpenerPolicy:   "unsafe-none",
				CrossOriginResourcePolicy: "cross-origin",
			},
			isHTTPS: true,
			expectedHeaders: map[string]string{
				"Content-Security-Policy":      "default-src 'none'; script-src 'self'",
				"Strict-Transport-Security":    "max-age=63072000; includeSubDomains",
				"X-Frame-Options":              "SAMEORIGIN",
				"X-Content-Type-Options":       "nosniff",
				"X-XSS-Protection":             "0",
				"Referrer-Policy":              "no-referrer",
				"Permissions-Policy":           "geolocation=(), camera=()",
				"Cross-Origin-Embedder-Policy": "unsafe-none",
				"Cross-Origin-Opener-Policy":   "unsafe-none",
				"Cross-Origin-Resource-Policy": "cross-origin",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			logger := testhelpers.NewMockLogger()
			metrics := testhelpers.NewMockMetricsCollector()

			logger.On("Debug",
				mock.Anything,
				"Security headers applied",
				mock.Anything,
			).Return()

			middleware := NewSecurityHeadersMiddlewareWithConfig(tt.config, logger, metrics)

			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("success"))
			})

			var req *http.Request
			if tt.isHTTPS {
				req = httptest.NewRequest(http.MethodGet, "https://example.com/test", nil)
				req.TLS = &tls.ConnectionState{} // Set TLS to indicate HTTPS
			} else {
				req = httptest.NewRequest(http.MethodGet, "http://example.com/test", nil)
			}

			ctx := WithRequestID(req.Context(), "security-test")
			req = req.WithContext(ctx)
			rec := httptest.NewRecorder()

			// Act
			wrappedHandler := middleware.Wrap(handler)
			wrappedHandler.ServeHTTP(rec, req)

			// Assert
			assert.Equal(t, http.StatusOK, rec.Code)
			assert.Equal(t, "success", rec.Body.String())

			for header, expectedValue := range tt.expectedHeaders {
				assert.Equal(t, expectedValue, rec.Header().Get(header), "Header %s should match", header)
			}

			// Verify HSTS is only set for HTTPS
			if tt.isHTTPS {
				assert.NotEmpty(t, rec.Header().Get("Strict-Transport-Security"))
			} else {
				assert.Empty(t, rec.Header().Get("Strict-Transport-Security"))
			}

			logger.AssertExpectations(t)
		})
	}
}

func TestSecurityHeadersMiddleware_Wrap_EmptyConfig(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "EmptyConfig_NoHeadersSet",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			logger := testhelpers.NewMockLogger()
			metrics := testhelpers.NewMockMetricsCollector()

			logger.On("Debug",
				mock.Anything,
				"Security headers applied",
				mock.Anything,
			).Return()

			config := SecurityConfig{} // Empty config
			middleware := NewSecurityHeadersMiddlewareWithConfig(config, logger, metrics)

			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("success"))
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			ctx := WithRequestID(req.Context(), "empty-config-test")
			req = req.WithContext(ctx)
			rec := httptest.NewRecorder()

			// Act
			wrappedHandler := middleware.Wrap(handler)
			wrappedHandler.ServeHTTP(rec, req)

			// Assert
			assert.Equal(t, http.StatusOK, rec.Code)
			assert.Equal(t, "success", rec.Body.String())

			// No security headers should be set with empty config
			securityHeaders := []string{
				"Content-Security-Policy",
				"Strict-Transport-Security",
				"X-Frame-Options",
				"X-Content-Type-Options",
				"X-XSS-Protection",
				"Referrer-Policy",
				"Permissions-Policy",
				"Cross-Origin-Embedder-Policy",
				"Cross-Origin-Opener-Policy",
				"Cross-Origin-Resource-Policy",
			}

			for _, header := range securityHeaders {
				assert.Empty(t, rec.Header().Get(header), "Header %s should be empty", header)
			}

			logger.AssertExpectations(t)
		})
	}
}

func TestSecurityHeadersMiddleware_Wrap_IndividualHeaders(t *testing.T) {
	tests := []struct {
		name           string
		config         SecurityConfig
		expectedHeader string
		expectedValue  string
	}{
		{
			name: "IndividualHeader_CSP",
			config: SecurityConfig{
				ContentSecurityPolicy: "default-src 'self'; script-src 'unsafe-inline'",
			},
			expectedHeader: "Content-Security-Policy",
			expectedValue:  "default-src 'self'; script-src 'unsafe-inline'",
		},
		{
			name: "IndividualHeader_FrameOptions",
			config: SecurityConfig{
				FrameOptions: "SAMEORIGIN",
			},
			expectedHeader: "X-Frame-Options",
			expectedValue:  "SAMEORIGIN",
		},
		{
			name: "IndividualHeader_ContentTypeOptions",
			config: SecurityConfig{
				ContentTypeOptions: "nosniff",
			},
			expectedHeader: "X-Content-Type-Options",
			expectedValue:  "nosniff",
		},
		{
			name: "IndividualHeader_XSSProtection",
			config: SecurityConfig{
				XSSProtection: "1; mode=block",
			},
			expectedHeader: "X-XSS-Protection",
			expectedValue:  "1; mode=block",
		},
		{
			name: "IndividualHeader_ReferrerPolicy",
			config: SecurityConfig{
				ReferrerPolicy: "same-origin",
			},
			expectedHeader: "Referrer-Policy",
			expectedValue:  "same-origin",
		},
		{
			name: "IndividualHeader_PermissionsPolicy",
			config: SecurityConfig{
				PermissionsPolicy: "geolocation=()",
			},
			expectedHeader: "Permissions-Policy",
			expectedValue:  "geolocation=()",
		},
		{
			name: "IndividualHeader_CrossOriginEmbedder",
			config: SecurityConfig{
				CrossOriginEmbedderPolicy: "require-corp",
			},
			expectedHeader: "Cross-Origin-Embedder-Policy",
			expectedValue:  "require-corp",
		},
		{
			name: "IndividualHeader_CrossOriginOpener",
			config: SecurityConfig{
				CrossOriginOpenerPolicy: "same-origin",
			},
			expectedHeader: "Cross-Origin-Opener-Policy",
			expectedValue:  "same-origin",
		},
		{
			name: "IndividualHeader_CrossOriginResource",
			config: SecurityConfig{
				CrossOriginResourcePolicy: "same-site",
			},
			expectedHeader: "Cross-Origin-Resource-Policy",
			expectedValue:  "same-site",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			logger := testhelpers.NewMockLogger()
			metrics := testhelpers.NewMockMetricsCollector()

			logger.On("Debug",
				mock.Anything,
				"Security headers applied",
				mock.Anything,
			).Return()

			middleware := NewSecurityHeadersMiddlewareWithConfig(tt.config, logger, metrics)

			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			ctx := WithRequestID(req.Context(), "individual-header-test")
			req = req.WithContext(ctx)
			rec := httptest.NewRecorder()

			// Act
			wrappedHandler := middleware.Wrap(handler)
			wrappedHandler.ServeHTTP(rec, req)

			// Assert
			assert.Equal(t, http.StatusOK, rec.Code)
			assert.Equal(t, tt.expectedValue, rec.Header().Get(tt.expectedHeader))

			logger.AssertExpectations(t)
		})
	}
}

func TestSecurityHeadersMiddleware_Wrap_HSTS_OnlyHTTPS(t *testing.T) {
	tests := []struct {
		name     string
		scheme   string
		setTLS   bool
		wantHSTS bool
	}{
		{
			name:     "HSTS_HTTPSRequest_HSTS_Set",
			scheme:   "https",
			setTLS:   true,
			wantHSTS: true,
		},
		{
			name:     "HSTS_HTTPRequest_HSTS_NotSet",
			scheme:   "http",
			setTLS:   false,
			wantHSTS: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			logger := testhelpers.NewMockLogger()
			metrics := testhelpers.NewMockMetricsCollector()

			logger.On("Debug",
				mock.Anything,
				"Security headers applied",
				mock.Anything,
			).Return()

			config := SecurityConfig{
				StrictTransportSecurity: "max-age=31536000",
			}
			middleware := NewSecurityHeadersMiddlewareWithConfig(config, logger, metrics)

			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			url := tt.scheme + "://example.com/test"
			req := httptest.NewRequest(http.MethodGet, url, nil)
			if tt.setTLS {
				req.TLS = &tls.ConnectionState{} // Set TLS field
			}
			ctx := WithRequestID(req.Context(), "hsts-test")
			req = req.WithContext(ctx)
			rec := httptest.NewRecorder()

			// Act
			wrappedHandler := middleware.Wrap(handler)
			wrappedHandler.ServeHTTP(rec, req)

			// Assert
			assert.Equal(t, http.StatusOK, rec.Code)

			if tt.wantHSTS {
				assert.Equal(t, "max-age=31536000", rec.Header().Get("Strict-Transport-Security"))
			} else {
				assert.Empty(t, rec.Header().Get("Strict-Transport-Security"))
			}

			logger.AssertExpectations(t)
		})
	}
}

func TestSecurityHeadersMiddleware_Wrap_LogFields(t *testing.T) {
	tests := []struct {
		name     string
		setupReq func() *http.Request
		isHTTPS  bool
	}{
		{
			name: "LogFields_HTTPSRequest",
			setupReq: func() *http.Request {
				req := httptest.NewRequest(http.MethodPost, "https://api.example.com/users", nil)
				req.TLS = &tls.ConnectionState{}
				ctx := WithRequestID(req.Context(), "https-log-test")
				return req.WithContext(ctx)
			},
			isHTTPS: true,
		},
		{
			name: "LogFields_HTTPRequest",
			setupReq: func() *http.Request {
				req := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
				ctx := WithRequestID(req.Context(), "http-log-test")
				return req.WithContext(ctx)
			},
			isHTTPS: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			logger := testhelpers.NewMockLogger()
			metrics := testhelpers.NewMockMetricsCollector()

			middleware := NewSecurityHeadersMiddleware(logger, metrics)

			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			req := tt.setupReq()
			rec := httptest.NewRecorder()

			// Capture the log call to verify fields
			var loggedFields []interface{}
			logger.On("Debug",
				req.Context(),
				"Security headers applied",
				mock.Anything,
			).Run(func(args mock.Arguments) {
				if len(args) > 2 {
					loggedFields = args[2:]
				}
			}).Return()

			// Act
			wrappedHandler := middleware.Wrap(handler)
			wrappedHandler.ServeHTTP(rec, req)

			// Assert
			assert.Equal(t, http.StatusOK, rec.Code)

			// Verify log fields contain expected information
			foundHTTPS := false
			for i := 0; i < len(loggedFields); i += 2 {
				if i+1 < len(loggedFields) {
					if key, ok := loggedFields[i].(string); ok && key == "https" {
						if httpsValue, ok := loggedFields[i+1].(bool); ok {
							assert.Equal(t, tt.isHTTPS, httpsValue)
							foundHTTPS = true
							break
						}
					}
				}
			}
			assert.True(t, foundHTTPS, "HTTPS field should be logged")

			logger.AssertExpectations(t)
		})
	}
}

func TestSecurityHeadersMiddleware_Integration(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "Integration_ChainedWithOtherMiddleware",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			logger := testhelpers.NewMockLogger()
			metrics := testhelpers.NewMockMetricsCollector()

			logger.On("Debug",
				mock.Anything,
				"Security headers applied",
				mock.Anything,
			).Return()

			security := NewSecurityHeadersMiddleware(logger, metrics)

			// Create a middleware that adds a custom header
			customMw := MiddlewareFunc(func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("X-Custom-Middleware", "applied")
					next.ServeHTTP(w, r)
				})
			})

			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("integration success"))
			})

			req := httptest.NewRequest(http.MethodGet, "https://example.com/test", nil)
			req.TLS = &tls.ConnectionState{}
			ctx := WithRequestID(req.Context(), "integration-test")
			req = req.WithContext(ctx)
			rec := httptest.NewRecorder()

			// Act - Chain security middleware with custom middleware
			chain := NewChain(security, customMw)
			finalHandler := chain.Then(handler)
			finalHandler.ServeHTTP(rec, req)

			// Assert
			assert.Equal(t, http.StatusOK, rec.Code)
			assert.Equal(t, "integration success", rec.Body.String())

			// Verify security headers are set
			assert.NotEmpty(t, rec.Header().Get("X-Frame-Options"))
			assert.NotEmpty(t, rec.Header().Get("X-Content-Type-Options"))
			assert.NotEmpty(t, rec.Header().Get("Strict-Transport-Security"))

			// Verify custom middleware also worked
			assert.Equal(t, "applied", rec.Header().Get("X-Custom-Middleware"))

			logger.AssertExpectations(t)
		})
	}
}

// Benchmark tests for security headers middleware
func BenchmarkSecurityHeadersMiddleware_AllHeaders(b *testing.B) {
	logger := &testhelpers.MockLogger{}
	metrics := &testhelpers.MockMetricsCollector{}
	middleware := NewSecurityHeadersMiddleware(logger, metrics)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := middleware.Wrap(handler)
	req := httptest.NewRequest(http.MethodGet, "https://example.com/test", nil)
	req.TLS = &tls.ConnectionState{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		wrappedHandler.ServeHTTP(rec, req)
	}
}

func BenchmarkSecurityHeadersMiddleware_HTTPRequest(b *testing.B) {
	logger := &testhelpers.MockLogger{}
	metrics := &testhelpers.MockMetricsCollector{}
	middleware := NewSecurityHeadersMiddleware(logger, metrics)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := middleware.Wrap(handler)
	req := httptest.NewRequest(http.MethodGet, "http://example.com/test", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		wrappedHandler.ServeHTTP(rec, req)
	}
}

func BenchmarkSecurityHeadersMiddleware_EmptyConfig(b *testing.B) {
	logger := &testhelpers.MockLogger{}
	metrics := &testhelpers.MockMetricsCollector{}
	config := SecurityConfig{} // Empty config
	middleware := NewSecurityHeadersMiddlewareWithConfig(config, logger, metrics)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := middleware.Wrap(handler)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		wrappedHandler.ServeHTTP(rec, req)
	}
}
