package middleware

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	testhelpers "github.com/albedosehen/rwwwrse/internal/testing"
)

func TestNewRecoveryMiddleware(t *testing.T) {
	tests := []struct {
		name   string
		logger *testhelpers.MockLogger
	}{
		{
			name:   "NewRecoveryMiddleware_ValidLogger",
			logger: testhelpers.NewMockLogger(),
		},
		{
			name:   "NewRecoveryMiddleware_NilLogger",
			logger: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			middleware := NewRecoveryMiddleware(tt.logger)

			// Assert
			assert.NotNil(t, middleware)
			assert.Implements(t, (*Middleware)(nil), middleware)
		})
	}
}

func TestRecoveryMiddleware_Wrap_NormalExecution(t *testing.T) {
	tests := []struct {
		name           string
		handler        http.HandlerFunc
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "Wrap_NormalHandler_Success",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("success"))
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "success",
		},
		{
			name: "Wrap_ErrorHandler_Success",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("error"))
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			logger := testhelpers.NewMockLogger()
			middleware := NewRecoveryMiddleware(logger)

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			rec := httptest.NewRecorder()

			// Act
			wrappedHandler := middleware.Wrap(tt.handler)
			wrappedHandler.ServeHTTP(rec, req)

			// Assert
			assert.Equal(t, tt.expectedStatus, rec.Code)
			assert.Equal(t, tt.expectedBody, rec.Body.String())

			// Verify no error logs were called since no panic occurred
			logger.AssertNotCalled(t, "Error")
		})
	}
}

func TestRecoveryMiddleware_Wrap_PanicRecovery(t *testing.T) {
	tests := []struct {
		name        string
		panicValue  interface{}
		setupReq    func() *http.Request
		expectedLog string
	}{
		{
			name:       "Wrap_StringPanic_Recovered",
			panicValue: "something went wrong",
			setupReq: func() *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/test", nil)
				ctx := WithRequestID(req.Context(), "test-req-123")
				return req.WithContext(ctx)
			},
			expectedLog: "panic: something went wrong",
		},
		{
			name:       "Wrap_ErrorPanic_Recovered",
			panicValue: fmt.Errorf("database connection failed"),
			setupReq: func() *http.Request {
				req := httptest.NewRequest(http.MethodPost, "/api/users", nil)
				ctx := WithRequestID(req.Context(), "req-456")
				return req.WithContext(ctx)
			},
			expectedLog: "panic: database connection failed",
		},
		{
			name:       "Wrap_NilPanic_Recovered",
			panicValue: nil,
			setupReq: func() *http.Request {
				req := httptest.NewRequest(http.MethodDelete, "/api/items/1", nil)
				ctx := WithRequestID(req.Context(), "req-789")
				return req.WithContext(ctx)
			},
			expectedLog: "panic: <nil>",
		},
		{
			name:       "Wrap_NoRequestID_Recovered",
			panicValue: "panic without request ID",
			setupReq: func() *http.Request {
				return httptest.NewRequest(http.MethodGet, "/test", nil)
			},
			expectedLog: "panic: panic without request ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			logger := testhelpers.NewMockLogger()
			middleware := NewRecoveryMiddleware(logger)

			panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				panic(tt.panicValue)
			})

			req := tt.setupReq()
			rec := httptest.NewRecorder()

			// Setup logger expectations
			logger.On("Error",
				req.Context(),
				mock.MatchedBy(func(err error) bool {
					return strings.Contains(err.Error(), tt.expectedLog)
				}),
				"Panic recovered",
				mock.Anything,
			).Return()

			// Act
			wrappedHandler := middleware.Wrap(panicHandler)
			wrappedHandler.ServeHTTP(rec, req)

			// Assert
			assert.Equal(t, http.StatusInternalServerError, rec.Code)
			assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

			// Check response body structure
			body := rec.Body.String()
			assert.Contains(t, body, `"error": "Internal server error"`)
			assert.Contains(t, body, `"status": 500`)
			assert.Contains(t, body, `"timestamp"`)

			// Verify logger was called
			logger.AssertExpectations(t)
		})
	}
}

func TestRecoveryMiddleware_Wrap_ResponseFormat(t *testing.T) {
	tests := []struct {
		name       string
		requestID  string
		wantFields []string
	}{
		{
			name:      "ResponseFormat_WithRequestID",
			requestID: "test-request-id",
			wantFields: []string{
				`"error": "Internal server error"`,
				`"status": 500`,
				`"request_id": "test-request-id"`,
				`"timestamp"`,
			},
		},
		{
			name:      "ResponseFormat_EmptyRequestID",
			requestID: "",
			wantFields: []string{
				`"error": "Internal server error"`,
				`"status": 500`,
				`"request_id": ""`,
				`"timestamp"`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			logger := testhelpers.NewMockLogger()
			middleware := NewRecoveryMiddleware(logger)

			panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				panic("test panic")
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tt.requestID != "" {
				ctx := WithRequestID(req.Context(), tt.requestID)
				req = req.WithContext(ctx)
			}
			rec := httptest.NewRecorder()

			// Setup logger expectations
			logger.On("Error",
				req.Context(),
				mock.MatchedBy(func(err error) bool {
					return strings.Contains(err.Error(), "panic: test panic")
				}),
				"Panic recovered",
				mock.Anything,
			).Return()

			// Act
			wrappedHandler := middleware.Wrap(panicHandler)
			wrappedHandler.ServeHTTP(rec, req)

			// Assert
			body := rec.Body.String()
			for _, field := range tt.wantFields {
				assert.Contains(t, body, field)
			}

			logger.AssertExpectations(t)
		})
	}
}

func TestRecoveryMiddleware_Wrap_LogFields(t *testing.T) {
	tests := []struct {
		name       string
		setupReq   func() *http.Request
		panicValue interface{}
		wantFields map[string]interface{}
	}{
		{
			name: "LogFields_FullRequest",
			setupReq: func() *http.Request {
				req := httptest.NewRequest(http.MethodPost, "/api/users?param=value", nil)
				req.RemoteAddr = "192.168.1.1:12345"
				ctx := WithRequestID(req.Context(), "req-full-123")
				return req.WithContext(ctx)
			},
			panicValue: "database error",
			wantFields: map[string]interface{}{
				"request_id":  "req-full-123",
				"method":      "POST",
				"path":        "/api/users",
				"remote_addr": "192.168.1.1:12345",
			},
		},
		{
			name: "LogFields_MinimalRequest",
			setupReq: func() *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/", nil)
				return req
			},
			panicValue: "simple panic",
			wantFields: map[string]interface{}{
				"request_id": "",
				"method":     "GET",
				"path":       "/",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			logger := testhelpers.NewMockLogger()
			middleware := NewRecoveryMiddleware(logger)

			panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				panic(tt.panicValue)
			})

			req := tt.setupReq()
			rec := httptest.NewRecorder()

			// Setup logger expectations with field matching
			logger.On("Error",
				req.Context(),
				mock.MatchedBy(func(err error) bool {
					return strings.Contains(err.Error(), fmt.Sprintf("panic: %v", tt.panicValue))
				}),
				"Panic recovered",
				mock.Anything,
			).Return()

			// Act
			wrappedHandler := middleware.Wrap(panicHandler)
			wrappedHandler.ServeHTTP(rec, req)

			// Assert
			assert.Equal(t, http.StatusInternalServerError, rec.Code)
			logger.AssertExpectations(t)
		})
	}
}

func TestRecoveryMiddleware_Wrap_StackTrace(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "StackTrace_IncludedInLog",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			logger := testhelpers.NewMockLogger()
			middleware := NewRecoveryMiddleware(logger)

			panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				panic("stack trace test")
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			ctx := WithRequestID(req.Context(), "stack-test")
			req = req.WithContext(ctx)
			rec := httptest.NewRecorder()

			// Capture the log call to verify stack trace
			var loggedFields []interface{}
			logger.On("Error",
				req.Context(),
				mock.MatchedBy(func(err error) bool {
					return strings.Contains(err.Error(), "panic: stack trace test")
				}),
				"Panic recovered",
				mock.Anything,
			).Run(func(args mock.Arguments) {
				// Store the fields for verification
				if len(args) > 3 {
					loggedFields = args[3:]
				}
			}).Return()

			// Act
			wrappedHandler := middleware.Wrap(panicHandler)
			wrappedHandler.ServeHTTP(rec, req)

			// Assert
			assert.Equal(t, http.StatusInternalServerError, rec.Code)

			// Verify stack trace was logged
			found := false
			for i := 0; i < len(loggedFields); i += 2 {
				if i+1 < len(loggedFields) {
					if key, ok := loggedFields[i].(string); ok && key == "stack" {
						if stack, ok := loggedFields[i+1].(string); ok && len(stack) > 0 {
							found = true
							// Verify it contains function information
							assert.Contains(t, stack, "recovery_test.go")
							break
						}
					}
				}
			}
			assert.True(t, found, "Stack trace should be included in log fields")

			logger.AssertExpectations(t)
		})
	}
}

func TestRecoveryMiddleware_Integration(t *testing.T) {
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
			recovery := NewRecoveryMiddleware(logger)

			// Create a middleware chain
			headerMw := MiddlewareFunc(func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("X-Test", "middleware")
					next.ServeHTTP(w, r)
				})
			})

			panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				panic("integration test panic")
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			ctx := WithRequestID(req.Context(), "integration-test")
			req = req.WithContext(ctx)
			rec := httptest.NewRecorder()

			// Setup logger expectations
			logger.On("Error",
				req.Context(),
				mock.MatchedBy(func(err error) bool {
					return strings.Contains(err.Error(), "panic: integration test panic")
				}),
				"Panic recovered",
				mock.Anything,
			).Return()

			// Act - Chain recovery middleware with header middleware
			chain := NewChain(recovery, headerMw)
			finalHandler := chain.Then(panicHandler)
			finalHandler.ServeHTTP(rec, req)

			// Assert
			assert.Equal(t, http.StatusInternalServerError, rec.Code)
			assert.Equal(t, "middleware", rec.Header().Get("X-Test"))

			logger.AssertExpectations(t)
		})
	}
}

// Benchmark tests for recovery middleware
func BenchmarkRecoveryMiddleware_Normal(b *testing.B) {
	logger := &testhelpers.MockLogger{}
	middleware := NewRecoveryMiddleware(logger)

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

func BenchmarkRecoveryMiddleware_WithPanic(b *testing.B) {
	var logBuffer bytes.Buffer
	logger := testhelpers.NewMockLogger()

	// Setup logger to avoid actual logging overhead in benchmark
	logger.On("Error",
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
	).Run(func(args mock.Arguments) {
		logBuffer.WriteString("logged panic\n")
	}).Return()

	middleware := NewRecoveryMiddleware(logger)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("benchmark panic")
	})

	wrappedHandler := middleware.Wrap(handler)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	ctx := WithRequestID(req.Context(), "bench-req")
	req = req.WithContext(ctx)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		wrappedHandler.ServeHTTP(rec, req)
	}
}

func BenchmarkRecoveryMiddleware_ContextOperations(b *testing.B) {
	ctx := context.Background()
	requestID := "benchmark-request-id"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		newCtx := WithRequestID(ctx, requestID)
		_ = GetRequestID(newCtx)
	}
}
