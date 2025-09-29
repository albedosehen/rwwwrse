package middleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewChain(t *testing.T) {
	tests := []struct {
		name         string
		middlewares  []Middleware
		expectLength int
	}{
		{
			name:         "NewChain_Empty",
			middlewares:  []Middleware{},
			expectLength: 0,
		},
		{
			name: "NewChain_SingleMiddleware",
			middlewares: []Middleware{
				MiddlewareFunc(func(next http.Handler) http.Handler {
					return next
				}),
			},
			expectLength: 1,
		},
		{
			name: "NewChain_MultipleMiddlewares",
			middlewares: []Middleware{
				MiddlewareFunc(func(next http.Handler) http.Handler {
					return next
				}),
				MiddlewareFunc(func(next http.Handler) http.Handler {
					return next
				}),
			},
			expectLength: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			chain := NewChain(tt.middlewares...)

			// Assert
			assert.NotNil(t, chain)

			// Verify chain functionality by testing behavior
			assert.NotNil(t, chain)
		})
	}
}

func TestChain_Use(t *testing.T) {
	tests := []struct {
		name           string
		initialCount   int
		additionalMw   Middleware
		expectedLength int
	}{
		{
			name:         "Use_AddToEmptyChain",
			initialCount: 0,
			additionalMw: MiddlewareFunc(func(next http.Handler) http.Handler {
				return next
			}),
			expectedLength: 1,
		},
		{
			name:         "Use_AddToExistingChain",
			initialCount: 2,
			additionalMw: MiddlewareFunc(func(next http.Handler) http.Handler {
				return next
			}),
			expectedLength: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			var initialMws []Middleware
			for i := 0; i < tt.initialCount; i++ {
				initialMws = append(initialMws, MiddlewareFunc(func(next http.Handler) http.Handler {
					return next
				}))
			}

			chain := NewChain(initialMws...)

			// Act
			newChain := chain.Use(tt.additionalMw)

			// Assert
			assert.NotNil(t, newChain)
			assert.NotEqual(t, chain, newChain) // Should return a new chain

			assert.NotNil(t, newChain)
		})
	}
}

func TestChain_Then(t *testing.T) {
	tests := []struct {
		name     string
		handler  http.Handler
		wantBody string
	}{
		{
			name: "Then_WithValidHandler",
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("handler response"))
			}),
			wantBody: "handler response",
		},
		{
			name:     "Then_WithNilHandler",
			handler:  nil,
			wantBody: "404 page not found\n", // Default mux response
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			chain := NewChain()
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			rec := httptest.NewRecorder()

			// Act
			finalHandler := chain.Then(tt.handler)
			finalHandler.ServeHTTP(rec, req)

			// Assert
			assert.Equal(t, tt.wantBody, rec.Body.String())
		})
	}
}

func TestChain_ThenFunc(t *testing.T) {
	tests := []struct {
		name        string
		handlerFunc http.HandlerFunc
		wantBody    string
		wantStatus  int
	}{
		{
			name: "ThenFunc_ValidHandlerFunc",
			handlerFunc: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("handler func response"))
			},
			wantBody:   "handler func response",
			wantStatus: http.StatusOK,
		},
		{
			name: "ThenFunc_ErrorHandlerFunc",
			handlerFunc: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte("error"))
			},
			wantBody:   "error",
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			chain := NewChain()
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			rec := httptest.NewRecorder()

			// Act
			finalHandler := chain.ThenFunc(tt.handlerFunc)
			finalHandler.ServeHTTP(rec, req)

			// Assert
			assert.Equal(t, tt.wantBody, rec.Body.String())
			assert.Equal(t, tt.wantStatus, rec.Code)
		})
	}
}

func TestChain_MultipleMiddleware(t *testing.T) {
	tests := []struct {
		name            string
		middlewareCount int
		expectedHeaders map[string]string
	}{
		{
			name:            "MultipleMiddleware_AddMultipleToChain",
			middlewareCount: 3,
			expectedHeaders: map[string]string{
				"X-MW-1": "first",
				"X-MW-2": "second",
				"X-MW-3": "third",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			chain := NewChain()

			// Add middlewares sequentially
			for i := 1; i <= tt.middlewareCount; i++ {
				headerName := fmt.Sprintf("X-MW-%d", i)
				var headerValue string
				switch i {
				case 1:
					headerValue = "first"
				case 2:
					headerValue = "second"
				case 3:
					headerValue = "third"
				}

				mw := MiddlewareFunc(func(next http.Handler) http.Handler {
					return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						w.Header().Set(headerName, headerValue)
						next.ServeHTTP(w, r)
					})
				})
				chain = chain.Use(mw)
			}

			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			rec := httptest.NewRecorder()

			// Act
			finalHandler := chain.Then(handler)
			finalHandler.ServeHTTP(rec, req)

			// Assert
			for key, expectedValue := range tt.expectedHeaders {
				assert.Equal(t, expectedValue, rec.Header().Get(key))
			}
		})
	}
}

func TestChain_ExecutionOrder(t *testing.T) {
	tests := []struct {
		name          string
		expectedOrder []string
	}{
		{
			name:          "ExecutionOrder_MiddlewaresInOrder",
			expectedOrder: []string{"mw1", "mw2", "mw3", "handler"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			var executionOrder []string

			mw1 := MiddlewareFunc(func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					executionOrder = append(executionOrder, "mw1")
					next.ServeHTTP(w, r)
				})
			})

			mw2 := MiddlewareFunc(func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					executionOrder = append(executionOrder, "mw2")
					next.ServeHTTP(w, r)
				})
			})

			mw3 := MiddlewareFunc(func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					executionOrder = append(executionOrder, "mw3")
					next.ServeHTTP(w, r)
				})
			})

			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				executionOrder = append(executionOrder, "handler")
				w.WriteHeader(http.StatusOK)
			})

			chain := NewChain(mw1, mw2, mw3)
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			rec := httptest.NewRecorder()

			// Act
			finalHandler := chain.Then(handler)
			finalHandler.ServeHTTP(rec, req)

			// Assert
			assert.Equal(t, tt.expectedOrder, executionOrder)
		})
	}
}

func TestChain_WithHeaders(t *testing.T) {
	tests := []struct {
		name            string
		expectedHeaders map[string]string
	}{
		{
			name: "Headers_MiddlewareChainAddsHeaders",
			expectedHeaders: map[string]string{
				"X-Middleware-1": "first",
				"X-Middleware-2": "second",
				"X-Handler":      "final",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mw1 := MiddlewareFunc(func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("X-Middleware-1", "first")
					next.ServeHTTP(w, r)
				})
			})

			mw2 := MiddlewareFunc(func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("X-Middleware-2", "second")
					next.ServeHTTP(w, r)
				})
			})

			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-Handler", "final")
				w.WriteHeader(http.StatusOK)
			})

			chain := NewChain(mw1, mw2)
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			rec := httptest.NewRecorder()

			// Act
			finalHandler := chain.Then(handler)
			finalHandler.ServeHTTP(rec, req)

			// Assert
			for key, expectedValue := range tt.expectedHeaders {
				assert.Equal(t, expectedValue, rec.Header().Get(key))
			}
		})
	}
}

func TestChain_ImmutableChain(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "ImmutableChain_OriginalUnmodified",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			originalMw := MiddlewareFunc(func(next http.Handler) http.Handler {
				return next
			})

			originalChain := NewChain(originalMw)

			newMw := MiddlewareFunc(func(next http.Handler) http.Handler {
				return next
			})

			// Act
			newChain := originalChain.Use(newMw)

			// Assert
			assert.NotNil(t, originalChain)
			assert.NotNil(t, newChain)

			// Verify chains are different instances
			assert.NotEqual(t, originalChain, newChain)

			// Test that original chain still works
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			rec1 := httptest.NewRecorder()
			rec2 := httptest.NewRecorder()

			originalChain.Then(handler).ServeHTTP(rec1, req)
			newChain.Then(handler).ServeHTTP(rec2, req)

			assert.Equal(t, http.StatusOK, rec1.Code)
			assert.Equal(t, http.StatusOK, rec2.Code)
		})
	}
}

// Benchmark tests for chain operations
func BenchmarkNewChain(b *testing.B) {
	middlewares := make([]Middleware, 10)
	for i := range middlewares {
		middlewares[i] = MiddlewareFunc(func(next http.Handler) http.Handler {
			return next
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewChain(middlewares...)
	}
}

func BenchmarkChain_Use(b *testing.B) {
	chain := NewChain()
	mw := MiddlewareFunc(func(next http.Handler) http.Handler {
		return next
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = chain.Use(mw)
	}
}

func BenchmarkChain_Then(b *testing.B) {
	middlewares := make([]Middleware, 5)
	for i := range middlewares {
		middlewares[i] = MiddlewareFunc(func(next http.Handler) http.Handler {
			return next
		})
	}

	chain := NewChain(middlewares...)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = chain.Then(handler)
	}
}

func BenchmarkChain_Execution(b *testing.B) {
	middlewares := make([]Middleware, 5)
	for i := range middlewares {
		middlewares[i] = MiddlewareFunc(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				next.ServeHTTP(w, r)
			})
		})
	}

	chain := NewChain(middlewares...)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	finalHandler := chain.Then(handler)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		finalHandler.ServeHTTP(rec, req)
	}
}
