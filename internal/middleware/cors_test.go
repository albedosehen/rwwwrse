package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	testhelpers "github.com/albedosehen/rwwwrse/internal/testing"
)

func TestDefaultCORSConfig(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "DefaultCORSConfig_ReturnsValidConfig",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			config := DefaultCORSConfig()

			// Assert
			assert.Equal(t, []string{"*"}, config.AllowedOrigins)
			assert.Contains(t, config.AllowedMethods, http.MethodGet)
			assert.Contains(t, config.AllowedMethods, http.MethodPost)
			assert.Contains(t, config.AllowedMethods, http.MethodOptions)
			assert.Contains(t, config.AllowedHeaders, "Accept")
			assert.Contains(t, config.AllowedHeaders, "Content-Type")
			assert.Equal(t, 12*time.Hour, config.MaxAge)
			assert.False(t, config.AllowCredentials)
			assert.False(t, config.OptionsPassthrough)
		})
	}
}

func TestRestrictiveCORSConfig(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "RestrictiveCORSConfig_ReturnsRestrictiveConfig",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			config := RestrictiveCORSConfig()

			// Assert
			assert.Empty(t, config.AllowedOrigins)
			assert.Contains(t, config.AllowedMethods, http.MethodGet)
			assert.Contains(t, config.AllowedMethods, http.MethodPost)
			assert.NotContains(t, config.AllowedMethods, http.MethodPut)
			assert.NotContains(t, config.AllowedMethods, http.MethodDelete)
			assert.Contains(t, config.AllowedHeaders, "Accept")
			assert.Contains(t, config.AllowedHeaders, "Content-Type")
			assert.NotContains(t, config.AllowedHeaders, "Authorization")
			assert.Equal(t, 1*time.Hour, config.MaxAge)
			assert.False(t, config.AllowCredentials)
			assert.False(t, config.OptionsPassthrough)
		})
	}
}

func TestNewCORSMiddleware(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "NewCORSMiddleware_ValidCreation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			logger := testhelpers.NewMockLogger()
			metrics := testhelpers.NewMockMetricsCollector()

			// Act
			middleware := NewCORSMiddleware(logger, metrics)

			// Assert
			assert.NotNil(t, middleware)
			assert.Implements(t, (*Middleware)(nil), middleware)
		})
	}
}

func TestNewCORSMiddlewareWithConfig(t *testing.T) {
	tests := []struct {
		name   string
		config CORSConfig
	}{
		{
			name:   "NewCORSMiddlewareWithConfig_CustomConfig",
			config: RestrictiveCORSConfig(),
		},
		{
			name: "NewCORSMiddlewareWithConfig_EmptyOrigins",
			config: CORSConfig{
				AllowedOrigins: []string{},
				AllowedMethods: []string{http.MethodGet},
				AllowedHeaders: []string{"Content-Type"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			logger := testhelpers.NewMockLogger()
			metrics := testhelpers.NewMockMetricsCollector()

			// Act
			middleware := NewCORSMiddlewareWithConfig(tt.config, logger, metrics)

			// Assert
			assert.NotNil(t, middleware)
			assert.Implements(t, (*Middleware)(nil), middleware)
		})
	}
}

func TestCORSMiddleware_Wrap_SimpleRequests(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		origin         string
		config         CORSConfig
		expectedStatus int
		checkHeaders   map[string]string
	}{
		{
			name:   "SimpleRequest_AllowedOrigin_Success",
			method: http.MethodGet,
			origin: "https://example.com",
			config: CORSConfig{
				AllowedOrigins: []string{"https://example.com"},
				AllowedMethods: []string{http.MethodGet},
				AllowedHeaders: []string{"Content-Type"},
			},
			expectedStatus: http.StatusOK,
			checkHeaders: map[string]string{
				"Access-Control-Allow-Origin": "https://example.com",
			},
		},
		{
			name:   "SimpleRequest_WildcardOrigin_Success",
			method: http.MethodGet,
			origin: "https://any-domain.com",
			config: CORSConfig{
				AllowedOrigins: []string{"*"},
				AllowedMethods: []string{http.MethodGet},
				AllowedHeaders: []string{"Content-Type"},
			},
			expectedStatus: http.StatusOK,
			checkHeaders: map[string]string{
				"Access-Control-Allow-Origin": "*",
			},
		},
		{
			name:           "SimpleRequest_NoOrigin_Success",
			method:         http.MethodGet,
			origin:         "",
			config:         DefaultCORSConfig(),
			expectedStatus: http.StatusOK,
			checkHeaders:   map[string]string{},
		},
		{
			name:   "SimpleRequest_DisallowedOrigin_Forbidden",
			method: http.MethodGet,
			origin: "https://malicious.com",
			config: CORSConfig{
				AllowedOrigins: []string{"https://trusted.com"},
				AllowedMethods: []string{http.MethodGet},
				AllowedHeaders: []string{"Content-Type"},
			},
			expectedStatus: http.StatusForbidden,
			checkHeaders:   map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			logger := testhelpers.NewMockLogger()
			metrics := testhelpers.NewMockMetricsCollector()

			if tt.expectedStatus == http.StatusForbidden {
				logger.On("Warn",
					mock.Anything,
					"CORS request blocked: origin not allowed",
					mock.Anything,
					mock.Anything,
					mock.Anything,
					mock.Anything,
				).Return()
			} else if tt.origin != "" {
				// Only expect debug log if there's an origin header
				logger.On("Debug",
					mock.Anything,
					"CORS headers applied",
					mock.Anything,
					mock.Anything,
					mock.Anything,
					mock.Anything,
				).Return()
			}

			middleware := NewCORSMiddlewareWithConfig(tt.config, logger, metrics)

			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("success"))
			})

			req := httptest.NewRequest(tt.method, "/test", nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			ctx := WithRequestID(req.Context(), "cors-test")
			req = req.WithContext(ctx)
			rec := httptest.NewRecorder()

			// Act
			wrappedHandler := middleware.Wrap(handler)
			wrappedHandler.ServeHTTP(rec, req)

			// Assert
			assert.Equal(t, tt.expectedStatus, rec.Code)

			for header, expectedValue := range tt.checkHeaders {
				assert.Equal(t, expectedValue, rec.Header().Get(header))
			}

			if tt.expectedStatus == http.StatusOK {
				assert.Equal(t, "success", rec.Body.String())
			}

			logger.AssertExpectations(t)
		})
	}
}

func TestCORSMiddleware_Wrap_PreflightRequests(t *testing.T) {
	tests := []struct {
		name               string
		origin             string
		requestMethod      string
		requestHeaders     string
		config             CORSConfig
		expectedStatus     int
		expectedHeaders    map[string]string
		optionsPassthrough bool
	}{
		{
			name:           "PreflightRequest_ValidOriginAndMethod_Success",
			origin:         "https://example.com",
			requestMethod:  "POST",
			requestHeaders: "Content-Type,Authorization",
			config: CORSConfig{
				AllowedOrigins: []string{"https://example.com"},
				AllowedMethods: []string{http.MethodPost, http.MethodOptions},
				AllowedHeaders: []string{"Content-Type", "Authorization"},
				MaxAge:         24 * time.Hour,
			},
			expectedStatus: http.StatusNoContent,
			expectedHeaders: map[string]string{
				"Access-Control-Allow-Origin":  "https://example.com",
				"Access-Control-Allow-Methods": "POST, OPTIONS",
				"Access-Control-Allow-Headers": "Content-Type,Authorization",
				"Access-Control-Max-Age":       "86400",
			},
		},
		{
			name:          "PreflightRequest_WithCredentials_Success",
			origin:        "https://trusted.com",
			requestMethod: "PUT",
			config: CORSConfig{
				AllowedOrigins:   []string{"https://trusted.com"},
				AllowedMethods:   []string{http.MethodPut, http.MethodOptions},
				AllowedHeaders:   []string{"Content-Type"},
				AllowCredentials: true,
			},
			expectedStatus: http.StatusNoContent,
			expectedHeaders: map[string]string{
				"Access-Control-Allow-Origin":      "https://trusted.com",
				"Access-Control-Allow-Methods":     "PUT, OPTIONS",
				"Access-Control-Allow-Credentials": "true",
			},
		},
		{
			name:          "PreflightRequest_OptionsPassthrough_Continues",
			origin:        "https://example.com",
			requestMethod: "POST",
			config: CORSConfig{
				AllowedOrigins:     []string{"https://example.com"},
				AllowedMethods:     []string{http.MethodPost, http.MethodOptions},
				AllowedHeaders:     []string{"Content-Type"},
				OptionsPassthrough: true,
			},
			expectedStatus: http.StatusOK,
			expectedHeaders: map[string]string{
				"Access-Control-Allow-Origin":  "https://example.com",
				"Access-Control-Allow-Methods": "POST, OPTIONS",
			},
			optionsPassthrough: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			logger := testhelpers.NewMockLogger()
			metrics := testhelpers.NewMockMetricsCollector()

			if tt.optionsPassthrough {
				// For options passthrough, expect both CORS headers applied and preflight handled logs
				logger.On("Debug",
					mock.Anything,
					"CORS headers applied",
					mock.Anything,
					mock.Anything,
					mock.Anything,
					mock.Anything,
				).Return()
				
				logger.On("Debug",
					mock.Anything,
					"CORS preflight request handled",
					mock.Anything,
					mock.Anything,
					mock.Anything,
					mock.Anything,
				).Return()
			} else {
				// For non-passthrough, expect only preflight handled log
				logger.On("Debug",
					mock.Anything,
					"CORS preflight request handled",
					mock.Anything,
					mock.Anything,
					mock.Anything,
					mock.Anything,
				).Return()
			}

			middleware := NewCORSMiddlewareWithConfig(tt.config, logger, metrics)

			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("handler called"))
			})

			req := httptest.NewRequest(http.MethodOptions, "/test", nil)
			req.Header.Set("Origin", tt.origin)
			req.Header.Set("Access-Control-Request-Method", tt.requestMethod)
			if tt.requestHeaders != "" {
				req.Header.Set("Access-Control-Request-Headers", tt.requestHeaders)
			}
			ctx := WithRequestID(req.Context(), "preflight-test")
			req = req.WithContext(ctx)
			rec := httptest.NewRecorder()

			// Act
			wrappedHandler := middleware.Wrap(handler)
			wrappedHandler.ServeHTTP(rec, req)

			// Assert
			assert.Equal(t, tt.expectedStatus, rec.Code)

			for header, expectedValue := range tt.expectedHeaders {
				assert.Equal(t, expectedValue, rec.Header().Get(header))
			}

			if tt.optionsPassthrough {
				assert.Equal(t, "handler called", rec.Body.String())
			} else {
				assert.Empty(t, rec.Body.String())
			}

			logger.AssertExpectations(t)
		})
	}
}

func TestCORSMiddleware_Wrap_ExposedHeaders(t *testing.T) {
	tests := []struct {
		name            string
		config          CORSConfig
		expectedHeaders map[string]string
	}{
		{
			name: "ExposedHeaders_SingleHeader",
			config: CORSConfig{
				AllowedOrigins: []string{"*"},
				AllowedMethods: []string{http.MethodGet},
				ExposedHeaders: []string{"X-Total-Count"},
			},
			expectedHeaders: map[string]string{
				"Access-Control-Expose-Headers": "X-Total-Count",
			},
		},
		{
			name: "ExposedHeaders_MultipleHeaders",
			config: CORSConfig{
				AllowedOrigins: []string{"*"},
				AllowedMethods: []string{http.MethodGet},
				ExposedHeaders: []string{"X-Total-Count", "X-Page-Size", "X-Current-Page"},
			},
			expectedHeaders: map[string]string{
				"Access-Control-Expose-Headers": "X-Total-Count, X-Page-Size, X-Current-Page",
			},
		},
		{
			name: "ExposedHeaders_None",
			config: CORSConfig{
				AllowedOrigins: []string{"*"},
				AllowedMethods: []string{http.MethodGet},
				ExposedHeaders: []string{},
			},
			expectedHeaders: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			logger := testhelpers.NewMockLogger()
			metrics := testhelpers.NewMockMetricsCollector()

			logger.On("Debug",
				mock.Anything,
				"CORS headers applied",
				mock.Anything,
				mock.Anything,
				mock.Anything,
				mock.Anything,
			).Return()

			middleware := NewCORSMiddlewareWithConfig(tt.config, logger, metrics)

			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("Origin", "https://example.com")
			ctx := WithRequestID(req.Context(), "expose-test")
			req = req.WithContext(ctx)
			rec := httptest.NewRecorder()

			// Act
			wrappedHandler := middleware.Wrap(handler)
			wrappedHandler.ServeHTTP(rec, req)

			// Assert
			assert.Equal(t, http.StatusOK, rec.Code)

			for header, expectedValue := range tt.expectedHeaders {
				assert.Equal(t, expectedValue, rec.Header().Get(header))
			}

			logger.AssertExpectations(t)
		})
	}
}

func TestCORSMiddleware_ConfigNormalization(t *testing.T) {
	tests := []struct {
		name           string
		inputConfig    CORSConfig
		expectedOrigin string
		expectedMethod string
		expectedHeader string
	}{
		{
			name: "ConfigNormalization_EmptyOrigins",
			inputConfig: CORSConfig{
				AllowedOrigins: []string{},
			},
			expectedOrigin: "*",
		},
		{
			name: "ConfigNormalization_EmptyMethods",
			inputConfig: CORSConfig{
				AllowedOrigins: []string{"*"},
				AllowedMethods: []string{},
			},
			expectedMethod: "GET",
		},
		{
			name: "ConfigNormalization_EmptyHeaders",
			inputConfig: CORSConfig{
				AllowedOrigins: []string{"*"},
				AllowedMethods: []string{http.MethodGet},
				AllowedHeaders: []string{},
			},
			expectedHeader: "Accept",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			logger := testhelpers.NewMockLogger()
			metrics := testhelpers.NewMockMetricsCollector()

			logger.On("Debug",
				mock.Anything,
				"CORS headers applied",
				mock.Anything,
				mock.Anything,
				mock.Anything,
				mock.Anything,
			).Return()

			middleware := NewCORSMiddlewareWithConfig(tt.inputConfig, logger, metrics)

			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("Origin", "https://example.com")
			ctx := WithRequestID(req.Context(), "normalize-test")
			req = req.WithContext(ctx)
			rec := httptest.NewRecorder()

			// Act
			wrappedHandler := middleware.Wrap(handler)
			wrappedHandler.ServeHTTP(rec, req)

			// Assert
			assert.Equal(t, http.StatusOK, rec.Code)

			if tt.expectedOrigin != "" {
				assert.Equal(t, tt.expectedOrigin, rec.Header().Get("Access-Control-Allow-Origin"))
			}

			logger.AssertExpectations(t)
		})
	}
}

func TestCORSMiddleware_OriginMatching(t *testing.T) {
	tests := []struct {
		name           string
		allowedOrigins []string
		testOrigin     string
		shouldAllow    bool
	}{
		{
			name:           "OriginMatching_ExactMatch",
			allowedOrigins: []string{"https://example.com", "https://test.com"},
			testOrigin:     "https://example.com",
			shouldAllow:    true,
		},
		{
			name:           "OriginMatching_Wildcard",
			allowedOrigins: []string{"*"},
			testOrigin:     "https://any-domain.com",
			shouldAllow:    true,
		},
		{
			name:           "OriginMatching_NoMatch",
			allowedOrigins: []string{"https://trusted.com"},
			testOrigin:     "https://untrusted.com",
			shouldAllow:    false,
		},
		{
			name:           "OriginMatching_CaseSensitive",
			allowedOrigins: []string{"https://Example.com"},
			testOrigin:     "https://example.com",
			shouldAllow:    false,
		},
		{
			name:           "OriginMatching_SubdomainNotAllowed",
			allowedOrigins: []string{"https://example.com"},
			testOrigin:     "https://sub.example.com",
			shouldAllow:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			logger := testhelpers.NewMockLogger()
			metrics := testhelpers.NewMockMetricsCollector()

			if tt.shouldAllow {
				logger.On("Debug",
					mock.Anything,
					"CORS headers applied",
					mock.Anything,
					mock.Anything,
					mock.Anything,
					mock.Anything,
				).Return()
			} else {
				logger.On("Warn",
					mock.Anything,
					"CORS request blocked: origin not allowed",
					mock.Anything,
					mock.Anything,
					mock.Anything,
					mock.Anything,
				).Return()
			}

			config := CORSConfig{
				AllowedOrigins: tt.allowedOrigins,
				AllowedMethods: []string{http.MethodGet},
				AllowedHeaders: []string{"Content-Type"},
			}

			middleware := NewCORSMiddlewareWithConfig(config, logger, metrics)

			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("success"))
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("Origin", tt.testOrigin)
			ctx := WithRequestID(req.Context(), "origin-test")
			req = req.WithContext(ctx)
			rec := httptest.NewRecorder()

			// Act
			wrappedHandler := middleware.Wrap(handler)
			wrappedHandler.ServeHTTP(rec, req)

			// Assert
			if tt.shouldAllow {
				assert.Equal(t, http.StatusOK, rec.Code)
				assert.Equal(t, "success", rec.Body.String())
			} else {
				assert.Equal(t, http.StatusForbidden, rec.Code)
				assert.Empty(t, rec.Body.String())
			}

			logger.AssertExpectations(t)
		})
	}
}

func TestCORSMiddleware_Integration(t *testing.T) {
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
				"CORS headers applied",
				mock.Anything,
				mock.Anything,
				mock.Anything,
				mock.Anything,
			).Return()

			cors := NewCORSMiddleware(logger, metrics)

			// Create a middleware that adds a custom header
			headerMw := MiddlewareFunc(func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("X-Custom", "middleware")
					next.ServeHTTP(w, r)
				})
			})

			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("integration success"))
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("Origin", "https://example.com")
			ctx := WithRequestID(req.Context(), "integration-test")
			req = req.WithContext(ctx)
			rec := httptest.NewRecorder()

			// Act - Chain CORS with custom middleware
			chain := NewChain(cors, headerMw)
			finalHandler := chain.Then(handler)
			finalHandler.ServeHTTP(rec, req)

			// Assert
			assert.Equal(t, http.StatusOK, rec.Code)
			assert.Equal(t, "integration success", rec.Body.String())
			assert.Equal(t, "*", rec.Header().Get("Access-Control-Allow-Origin"))
			assert.Equal(t, "middleware", rec.Header().Get("X-Custom"))

			logger.AssertExpectations(t)
		})
	}
}

// Benchmark tests for CORS middleware
func BenchmarkCORSMiddleware_SimpleRequest(b *testing.B) {
	logger := &testhelpers.MockLogger{}
	metrics := &testhelpers.MockMetricsCollector{}
	middleware := NewCORSMiddleware(logger, metrics)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := middleware.Wrap(handler)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "https://example.com")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		wrappedHandler.ServeHTTP(rec, req)
	}
}

func BenchmarkCORSMiddleware_PreflightRequest(b *testing.B) {
	logger := &testhelpers.MockLogger{}
	metrics := &testhelpers.MockMetricsCollector{}
	middleware := NewCORSMiddleware(logger, metrics)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := middleware.Wrap(handler)
	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		wrappedHandler.ServeHTTP(rec, req)
	}
}

func BenchmarkCORSMiddleware_OriginValidation(b *testing.B) {
	config := CORSConfig{
		AllowedOrigins: []string{"https://trusted1.com", "https://trusted2.com", "https://trusted3.com"},
		AllowedMethods: []string{http.MethodGet},
		AllowedHeaders: []string{"Content-Type"},
	}

	logger := &testhelpers.MockLogger{}
	metrics := &testhelpers.MockMetricsCollector{}
	middleware := NewCORSMiddlewareWithConfig(config, logger, metrics)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := middleware.Wrap(handler)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "https://trusted2.com")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		wrappedHandler.ServeHTTP(rec, req)
	}
}
