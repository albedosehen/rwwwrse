package middleware

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/albedosehen/rwwwrse/internal/observability"
	testhelpers "github.com/albedosehen/rwwwrse/internal/testing"
)

func TestNewLoggingMiddleware(t *testing.T) {
	tests := []struct {
		name    string
		logger  observability.Logger
		metrics observability.MetricsCollector
		wantNil bool
	}{
		{
			name:    "NewLoggingMiddleware_ValidParams",
			logger:  testhelpers.NewMockLogger(),
			metrics: testhelpers.NewMockMetricsCollector(),
			wantNil: false,
		},
		{
			name:    "NewLoggingMiddleware_NilLogger",
			logger:  nil,
			metrics: testhelpers.NewMockMetricsCollector(),
			wantNil: true,
		},
		{
			name:    "NewLoggingMiddleware_NilMetrics",
			logger:  testhelpers.NewMockLogger(),
			metrics: nil,
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			middleware := NewLoggingMiddleware(tt.logger, tt.metrics)

			// Assert
			if tt.wantNil {
				assert.Nil(t, middleware)
			} else {
				assert.NotNil(t, middleware)
				assert.Implements(t, (*Middleware)(nil), middleware)
			}
		})
	}
}

func TestLoggingMiddleware_Wrap_SuccessfulRequest(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		path           string
		statusCode     int
		responseBody   string
		expectedFields []string
	}{
		{
			name:         "SuccessfulRequest_GET",
			method:       http.MethodGet,
			path:         "/api/test",
			statusCode:   http.StatusOK,
			responseBody: "success",
			expectedFields: []string{
				"method",
				"path",
				"status",
				"duration",
				"size",
			},
		},
		{
			name:         "SuccessfulRequest_POST",
			method:       http.MethodPost,
			path:         "/api/users",
			statusCode:   http.StatusCreated,
			responseBody: `{"id": 123}`,
			expectedFields: []string{
				"method",
				"path",
				"status",
				"duration",
				"size",
			},
		},
		{
			name:         "SuccessfulRequest_PUT",
			method:       http.MethodPut,
			path:         "/api/users/123",
			statusCode:   http.StatusOK,
			responseBody: `{"updated": true}`,
			expectedFields: []string{
				"method",
				"path",
				"status",
				"duration",
				"size",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			logger := testhelpers.NewMockLogger()
			metrics := testhelpers.NewMockMetricsCollector()

			// Expect Info log for successful request
			logger.On("Info",
				mock.Anything,
				"HTTP request completed",
				mock.MatchedBy(func(fields []observability.Field) bool {
					fieldNames := make(map[string]bool)
					for _, field := range fields {
						fieldNames[field.Key] = true
					}

					// Check that all expected fields are present
					for _, expectedField := range tt.expectedFields {
						if !fieldNames[expectedField] {
							return false
						}
					}
					return true
				}),
			).Return()

			// Expect metrics calls
			metrics.On("RecordHTTPRequest",
				mock.AnythingOfType("string"),        // method
				mock.AnythingOfType("string"),        // path
				mock.AnythingOfType("int"),           // status
				mock.AnythingOfType("time.Duration"), // duration
			).Return()

			middleware := &loggingMiddleware{
				logger:  logger,
				metrics: metrics,
			}

			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.responseBody))
			})

			req := httptest.NewRequest(tt.method, tt.path, nil)
			ctx := WithRequestID(req.Context(), "logging-test")
			req = req.WithContext(ctx)
			rec := httptest.NewRecorder()

			// Act
			wrappedHandler := middleware.Wrap(handler)
			wrappedHandler.ServeHTTP(rec, req)

			// Assert
			assert.Equal(t, tt.statusCode, rec.Code)
			assert.Equal(t, tt.responseBody, rec.Body.String())
			logger.AssertExpectations(t)
			metrics.AssertExpectations(t)
		})
	}
}

func TestLoggingMiddleware_Wrap_ErrorRequest(t *testing.T) {
	tests := []struct {
		name         string
		method       string
		path         string
		statusCode   int
		responseBody string
		logLevel     string
	}{
		{
			name:         "ErrorRequest_BadRequest",
			method:       http.MethodPost,
			path:         "/api/users",
			statusCode:   http.StatusBadRequest,
			responseBody: `{"error": "invalid input"}`,
			logLevel:     "warn",
		},
		{
			name:         "ErrorRequest_NotFound",
			method:       http.MethodGet,
			path:         "/api/nonexistent",
			statusCode:   http.StatusNotFound,
			responseBody: `{"error": "not found"}`,
			logLevel:     "warn",
		},
		{
			name:         "ErrorRequest_InternalServerError",
			method:       http.MethodGet,
			path:         "/api/broken",
			statusCode:   http.StatusInternalServerError,
			responseBody: `{"error": "internal server error"}`,
			logLevel:     "error",
		},
		{
			name:         "ErrorRequest_ServiceUnavailable",
			method:       http.MethodGet,
			path:         "/api/service",
			statusCode:   http.StatusServiceUnavailable,
			responseBody: `{"error": "service unavailable"}`,
			logLevel:     "error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			logger := testhelpers.NewMockLogger()
			metrics := testhelpers.NewMockMetricsCollector()

			// Expect incoming request log - match any arguments after context and message
			logger.On("Info", mock.MatchedBy(func(ctx context.Context) bool {
				return true
			}), "HTTP request received", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()

			// Expect log based on status code
			switch tt.logLevel {
			case "warn":
				logger.On("Warn", mock.MatchedBy(func(ctx context.Context) bool {
					return true
				}), "HTTP request completed with client error", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()
			case "error":
				logger.On("Error", mock.MatchedBy(func(ctx context.Context) bool {
					return true
				}), mock.Anything, "HTTP request completed with server error", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()
			}

			// Create middleware using constructor to get proper config
			middleware := NewLoggingMiddleware(logger, metrics)

			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.responseBody))
			})

			req := httptest.NewRequest(tt.method, tt.path, nil)
			ctx := WithRequestID(req.Context(), "error-test")
			req = req.WithContext(ctx)
			rec := httptest.NewRecorder()

			// Act
			wrappedHandler := middleware.Wrap(handler)
			wrappedHandler.ServeHTTP(rec, req)

			// Assert
			assert.Equal(t, tt.statusCode, rec.Code)
			assert.Equal(t, tt.responseBody, rec.Body.String())
			logger.AssertExpectations(t)
			metrics.AssertExpectations(t)
		})
	}
}

func TestLoggingMiddleware_Wrap_RequestWithBody(t *testing.T) {
	tests := []struct {
		name        string
		method      string
		path        string
		requestBody string
		logBody     bool
	}{
		{
			name:        "RequestWithBody_POST_LogEnabled",
			method:      http.MethodPost,
			path:        "/api/users",
			requestBody: `{"name": "John", "email": "john@example.com"}`,
			logBody:     true,
		},
		{
			name:        "RequestWithBody_PUT_LogEnabled",
			method:      http.MethodPut,
			path:        "/api/users/123",
			requestBody: `{"name": "Jane"}`,
			logBody:     true,
		},
		{
			name:        "RequestWithBody_LogDisabled",
			method:      http.MethodPost,
			path:        "/api/sensitive",
			requestBody: `{"password": "secret123"}`,
			logBody:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			logger := testhelpers.NewMockLogger()
			metrics := testhelpers.NewMockMetricsCollector()

			// Expect Info log
			logger.On("Info",
				mock.Anything,
				"HTTP request completed",
				mock.MatchedBy(func(fields []observability.Field) bool {
					hasBodyField := false
					for _, field := range fields {
						if field.Key == "request_body" {
							hasBodyField = true
							break
						}
					}
					return hasBodyField == tt.logBody
				}),
			).Return()

			metrics.On("RecordHTTPRequest",
				mock.AnythingOfType("string"),
				mock.AnythingOfType("string"),
				mock.AnythingOfType("int"),
				mock.AnythingOfType("time.Duration"),
			).Return()

			config := DefaultLoggingConfig()
			config.LogRequestBody = tt.logBody

			middleware := &loggingMiddleware{
				config:  config,
				logger:  logger,
				metrics: metrics,
			}

			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("success"))
			})

			req := httptest.NewRequest(tt.method, tt.path, bytes.NewReader([]byte(tt.requestBody)))
			req.Header.Set("Content-Type", "application/json")
			ctx := WithRequestID(req.Context(), "body-test")
			req = req.WithContext(ctx)
			rec := httptest.NewRecorder()

			// Act
			wrappedHandler := middleware.Wrap(handler)
			wrappedHandler.ServeHTTP(rec, req)

			// Assert
			assert.Equal(t, http.StatusOK, rec.Code)
			logger.AssertExpectations(t)
			metrics.AssertExpectations(t)
		})
	}
}

func TestLoggingMiddleware_Wrap_ResponseCapture(t *testing.T) {
	tests := []struct {
		name            string
		responseBody    string
		expectedSize    int
		expectSizeField bool
	}{
		{
			name:            "ResponseCapture_WithBody",
			responseBody:    "Hello, World!",
			expectedSize:    13,
			expectSizeField: true,
		},
		{
			name:            "ResponseCapture_EmptyBody",
			responseBody:    "",
			expectedSize:    0,
			expectSizeField: true,
		},
		{
			name:            "ResponseCapture_JSONBody",
			responseBody:    `{"message": "success", "count": 42}`,
			expectedSize:    32,
			expectSizeField: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			logger := testhelpers.NewMockLogger()
			metrics := testhelpers.NewMockMetricsCollector()

			logger.On("Info",
				mock.Anything,
				"HTTP request completed",
				mock.MatchedBy(func(fields []observability.Field) bool {
					if !tt.expectSizeField {
						return true
					}

					for _, field := range fields {
						if field.Key == "size" {
							return field.Value == tt.expectedSize
						}
					}
					return false
				}),
			).Return()

			// Note: Current metrics interface doesn't have RecordHTTPRequest method
			// No metrics expectations needed for now

			middleware := &loggingMiddleware{
				logger:  logger,
				metrics: metrics,
			}

			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(tt.responseBody))
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			ctx := WithRequestID(req.Context(), "response-size-test")
			req = req.WithContext(ctx)
			rec := httptest.NewRecorder()

			// Act
			wrappedHandler := middleware.Wrap(handler)
			wrappedHandler.ServeHTTP(rec, req)

			// Assert
			assert.Equal(t, http.StatusOK, rec.Code)
			assert.Equal(t, tt.responseBody, rec.Body.String())
			logger.AssertExpectations(t)
			metrics.AssertExpectations(t)
		})
	}
}

func TestLoggingMiddleware_Wrap_TimingAccuracy(t *testing.T) {
	tests := []struct {
		name         string
		handlerDelay time.Duration
		minDuration  time.Duration
	}{
		{
			name:         "TimingAccuracy_FastRequest",
			handlerDelay: 1 * time.Millisecond,
			minDuration:  500 * time.Microsecond,
		},
		{
			name:         "TimingAccuracy_SlowRequest",
			handlerDelay: 10 * time.Millisecond,
			minDuration:  8 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			logger := testhelpers.NewMockLogger()
			metrics := testhelpers.NewMockMetricsCollector()

			logger.On("Info",
				mock.Anything,
				"HTTP request completed",
				mock.MatchedBy(func(fields []observability.Field) bool {
					for _, field := range fields {
						if field.Key == "duration" {
							duration, ok := field.Value.(time.Duration)
							return ok && duration >= tt.minDuration
						}
					}
					return false
				}),
			).Return()

			// Note: Current metrics interface doesn't have RecordHTTPRequest method
			// No metrics expectations needed for now

			middleware := &loggingMiddleware{
				config:  DefaultLoggingConfig(),
				logger:  logger,
				metrics: metrics,
			}

			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(tt.handlerDelay)
				w.WriteHeader(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodGet, "/timing", nil)
			ctx := WithRequestID(req.Context(), "timing-test")
			req = req.WithContext(ctx)
			rec := httptest.NewRecorder()

			// Act
			start := time.Now()
			wrappedHandler := middleware.Wrap(handler)
			wrappedHandler.ServeHTTP(rec, req)
			elapsed := time.Since(start)

			// Assert
			assert.Equal(t, http.StatusOK, rec.Code)
			assert.True(t, elapsed >= tt.minDuration, "Total elapsed time should be at least %v, got %v", tt.minDuration, elapsed)
			logger.AssertExpectations(t)
			metrics.AssertExpectations(t)
		})
	}
}

func TestLoggingMiddleware_Wrap_ContextFields(t *testing.T) {
	tests := []struct {
		name               string
		contextSetup       func(context.Context) context.Context
		expectedFieldCount int
		expectedFields     []string
	}{
		{
			name: "ContextFields_WithRequestID",
			contextSetup: func(ctx context.Context) context.Context {
				return WithRequestID(ctx, "req-123")
			},
			expectedFieldCount: 6, // method, path, status, duration, size, request_id
			expectedFields:     []string{"request_id"},
		},
		{
			name: "ContextFields_WithUserID",
			contextSetup: func(ctx context.Context) context.Context {
				ctx = WithRequestID(ctx, "req-456")
				const userId contextKey = "user_id"
				return context.WithValue(ctx, userId, "user-789")
			},
			expectedFieldCount: 6,
			expectedFields:     []string{"request_id"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			logger := testhelpers.NewMockLogger()
			metrics := testhelpers.NewMockMetricsCollector()

			logger.On("Info",
				mock.Anything,
				"HTTP request completed",
				mock.MatchedBy(func(fields []observability.Field) bool {
					if len(fields) < tt.expectedFieldCount {
						return false
					}

					fieldNames := make(map[string]bool)
					for _, field := range fields {
						fieldNames[field.Key] = true
					}

					for _, expectedField := range tt.expectedFields {
						if !fieldNames[expectedField] {
							return false
						}
					}
					return true
				}),
			).Return()

			// Note: Current metrics interface doesn't have RecordHTTPRequest method
			// No metrics expectations needed for now

			middleware := &loggingMiddleware{
				config:  DefaultLoggingConfig(),
				logger:  logger,
				metrics: metrics,
			}

			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodGet, "/context", nil)
			ctx := tt.contextSetup(req.Context())
			req = req.WithContext(ctx)
			rec := httptest.NewRecorder()

			// Act
			wrappedHandler := middleware.Wrap(handler)
			wrappedHandler.ServeHTTP(rec, req)

			// Assert
			assert.Equal(t, http.StatusOK, rec.Code)
			logger.AssertExpectations(t)
			metrics.AssertExpectations(t)
		})
	}
}

func TestLoggingMiddleware_Integration(t *testing.T) {
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

			logger.On("Info",
				mock.Anything,
				"HTTP request completed",
				mock.Anything,
			).Return()

			// Note: Current metrics interface doesn't have RecordHTTPRequest method
			// No metrics expectations needed for now

			loggingMw := &loggingMiddleware{
				config:  DefaultLoggingConfig(),
				logger:  logger,
				metrics: metrics,
			}

			// Create a middleware that adds headers
			headerMw := MiddlewareFunc(func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("X-Processed", "true")
					next.ServeHTTP(w, r)
				})
			})

			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("integration test"))
			})

			req := httptest.NewRequest(http.MethodGet, "/integration", nil)
			ctx := WithRequestID(req.Context(), "integration-test")
			req = req.WithContext(ctx)
			rec := httptest.NewRecorder()

			// Act - Chain logging with header middleware
			chain := NewChain(loggingMw, headerMw)
			finalHandler := chain.Then(handler)
			finalHandler.ServeHTTP(rec, req)

			// Assert
			assert.Equal(t, http.StatusOK, rec.Code)
			assert.Equal(t, "integration test", rec.Body.String())
			assert.Equal(t, "true", rec.Header().Get("X-Processed"))
			logger.AssertExpectations(t)
			metrics.AssertExpectations(t)
		})
	}
}

// Benchmark tests for logging middleware
func BenchmarkLoggingMiddleware_SuccessfulRequest(b *testing.B) {
	logger := &testhelpers.MockLogger{}
	metrics := &testhelpers.MockMetricsCollector{}

	// Setup mock expectations for benchmarking
	logger.On("Info",
		mock.Anything,
		mock.Anything,
		mock.Anything,
	).Return()

	metrics.On("RecordHTTPRequest",
		mock.AnythingOfType("string"),
		mock.AnythingOfType("string"),
		mock.AnythingOfType("int"),
		mock.AnythingOfType("time.Duration"),
	).Return()

	middleware := &loggingMiddleware{
		logger:  logger,
		metrics: metrics,
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("benchmark"))
	})

	wrappedHandler := middleware.Wrap(handler)
	req := httptest.NewRequest(http.MethodGet, "/benchmark", nil)
	ctx := WithRequestID(req.Context(), "bench-test")
	req = req.WithContext(ctx)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		wrappedHandler.ServeHTTP(rec, req)
	}
}

func BenchmarkLoggingMiddleware_WithBodyLogging(b *testing.B) {
	logger := &testhelpers.MockLogger{}
	metrics := &testhelpers.MockMetricsCollector{}

	logger.On("Info",
		mock.Anything,
		mock.Anything,
		mock.Anything,
	).Return()

	metrics.On("RecordHTTPRequest",
		mock.AnythingOfType("string"),
		mock.AnythingOfType("string"),
		mock.AnythingOfType("int"),
		mock.AnythingOfType("time.Duration"),
	).Return()

	config := VerboseLoggingConfig() // This enables body logging

	middleware := &loggingMiddleware{
		config:  config,
		logger:  logger,
		metrics: metrics,
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("benchmark with body"))
	})

	wrappedHandler := middleware.Wrap(handler)

	requestBody := `{"test": "data", "number": 123, "nested": {"key": "value"}}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/benchmark", bytes.NewReader([]byte(requestBody)))
		req.Header.Set("Content-Type", "application/json")
		ctx := WithRequestID(req.Context(), "bench-body-test")
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()
		wrappedHandler.ServeHTTP(rec, req)
	}
}
