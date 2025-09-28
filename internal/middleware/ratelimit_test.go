package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	testhelpers "github.com/albedosehen/rwwwrse/internal/testing"
)

func TestNewTokenBucketRateLimiter(t *testing.T) {
	tests := []struct {
		name           string
		requestsPerSec float64
		burstSize      int
		expectValid    bool
	}{
		{
			name:           "NewTokenBucketRateLimiter_ValidParams",
			requestsPerSec: 10.0,
			burstSize:      20,
			expectValid:    true,
		},
		{
			name:           "NewTokenBucketRateLimiter_SmallLimits",
			requestsPerSec: 1.0,
			burstSize:      1,
			expectValid:    true,
		},
		{
			name:           "NewTokenBucketRateLimiter_HighLimits",
			requestsPerSec: 1000.0,
			burstSize:      2000,
			expectValid:    true,
		},
		{
			name:           "NewTokenBucketRateLimiter_ZeroRequestsPerSec",
			requestsPerSec: 0.0,
			burstSize:      10,
			expectValid:    true, // Zero rate is valid (blocks all requests)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			logger := testhelpers.NewMockLogger()

			// Act
			limiter := NewTokenBucketRateLimiter(tt.requestsPerSec, tt.burstSize, logger)

			// Assert
			assert.NotNil(t, limiter)
			assert.Implements(t, (*RateLimiter)(nil), limiter)
		})
	}
}

func TestTokenBucketRateLimiter_Allow(t *testing.T) {
	tests := []struct {
		name           string
		requestsPerSec float64
		burstSize      int
		requestCount   int
		expectedAllows int
		key            string
	}{
		{
			name:           "Allow_WithinBurstLimit",
			requestsPerSec: 10.0,
			burstSize:      5,
			requestCount:   3,
			expectedAllows: 3,
			key:            "test-key-1",
		},
		{
			name:           "Allow_ExceedsBurstLimit",
			requestsPerSec: 10.0,
			burstSize:      2,
			requestCount:   5,
			expectedAllows: 2,
			key:            "test-key-2",
		},
		{
			name:           "Allow_ZeroRate",
			requestsPerSec: 0.0,
			burstSize:      5,
			requestCount:   3,
			expectedAllows: 0,
			key:            "test-key-3",
		},
		{
			name:           "Allow_SingleBurst",
			requestsPerSec: 1.0,
			burstSize:      1,
			requestCount:   2,
			expectedAllows: 1,
			key:            "test-key-4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			logger := testhelpers.NewMockLogger()
			limiter := NewTokenBucketRateLimiter(tt.requestsPerSec, tt.burstSize, logger)
			ctx := context.Background()

			// Act
			allowedCount := 0
			for i := 0; i < tt.requestCount; i++ {
				if limiter.Allow(ctx, tt.key) {
					allowedCount++
				}
			}

			// Assert
			assert.Equal(t, tt.expectedAllows, allowedCount)
		})
	}
}

func TestTokenBucketRateLimiter_Reset(t *testing.T) {
	tests := []struct {
		name string
		key  string
	}{
		{
			name: "Reset_ExistingKey",
			key:  "existing-key",
		},
		{
			name: "Reset_NonExistentKey",
			key:  "non-existent-key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			logger := testhelpers.NewMockLogger()
			limiter := NewTokenBucketRateLimiter(1.0, 1, logger)
			ctx := context.Background()

			// Use up the limit for existing key test
			if tt.name == "Reset_ExistingKey" {
				limiter.Allow(ctx, tt.key) // Use up the bucket
			}

			// Act
			err := limiter.Reset(tt.key)

			// Assert
			assert.NoError(t, err)

			// After reset, should be able to allow requests again
			if tt.name == "Reset_ExistingKey" {
				allowed := limiter.Allow(ctx, tt.key)
				assert.True(t, allowed, "Should allow request after reset")
			}
		})
	}
}

func TestTokenBucketRateLimiter_Stats(t *testing.T) {
	tests := []struct {
		name            string
		requestsPerSec  float64
		burstSize       int
		requestsToMake  int
		key             string
		expectRemaining int
		expectResetTime bool
	}{
		{
			name:            "Stats_UnusedLimiter",
			requestsPerSec:  10.0,
			burstSize:       5,
			requestsToMake:  0,
			key:             "unused-key",
			expectRemaining: 5,
			expectResetTime: true,
		},
		{
			name:            "Stats_PartiallyUsed",
			requestsPerSec:  10.0,
			burstSize:       5,
			requestsToMake:  2,
			key:             "partial-key",
			expectRemaining: 3,
			expectResetTime: true,
		},
		{
			name:            "Stats_FullyUsed",
			requestsPerSec:  10.0,
			burstSize:       3,
			requestsToMake:  5, // More than burst size
			key:             "full-key",
			expectRemaining: 0,
			expectResetTime: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			logger := testhelpers.NewMockLogger()
			limiter := NewTokenBucketRateLimiter(tt.requestsPerSec, tt.burstSize, logger)
			ctx := context.Background()

			// Make requests to use up tokens
			for i := 0; i < tt.requestsToMake; i++ {
				limiter.Allow(ctx, tt.key)
			}

			// Act
			stats := limiter.Stats(tt.key)

			// Assert
			assert.Equal(t, tt.expectRemaining, stats.Remaining)
			if tt.expectResetTime {
				assert.False(t, stats.ResetTime.IsZero())
			}
			assert.True(t, stats.RetryAfter > 0)
		})
	}
}

func TestTokenBucketRateLimiter_Cleanup(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "Cleanup_NoError",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			logger := testhelpers.NewMockLogger()
			limiter := NewTokenBucketRateLimiter(10.0, 10, logger)
			ctx := context.Background()

			// Act
			err := limiter.Cleanup(ctx)

			// Assert
			assert.NoError(t, err)
		})
	}
}

func TestTokenBucketRateLimiter_MultipleKeys(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "MultipleKeys_IndependentLimits",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			logger := testhelpers.NewMockLogger()
			limiter := NewTokenBucketRateLimiter(10.0, 2, logger)
			ctx := context.Background()

			key1 := "user-1"
			key2 := "user-2"

			// Act - Use up limit for key1
			assert.True(t, limiter.Allow(ctx, key1))
			assert.True(t, limiter.Allow(ctx, key1))
			assert.False(t, limiter.Allow(ctx, key1)) // Should be blocked

			// Act - key2 should still have its own limit
			assert.True(t, limiter.Allow(ctx, key2))
			assert.True(t, limiter.Allow(ctx, key2))
			assert.False(t, limiter.Allow(ctx, key2)) // Should be blocked

			// Assert - Stats should be independent
			stats1 := limiter.Stats(key1)
			stats2 := limiter.Stats(key2)

			assert.Equal(t, 0, stats1.Remaining)
			assert.Equal(t, 0, stats2.Remaining)
		})
	}
}

func TestRateLimitMiddleware_Wrap_AllowedRequests(t *testing.T) {
	tests := []struct {
		name           string
		requestsPerSec float64
		burstSize      int
		requestCount   int
		expectedStatus int
		keyFunc        func(*http.Request) string
	}{
		{
			name:           "AllowedRequests_WithinLimit",
			requestsPerSec: 10.0,
			burstSize:      5,
			requestCount:   3,
			expectedStatus: http.StatusOK,
			keyFunc: func(r *http.Request) string {
				return "test-client"
			},
		},
		{
			name:           "AllowedRequests_AtBurstLimit",
			requestsPerSec: 10.0,
			burstSize:      2,
			requestCount:   2,
			expectedStatus: http.StatusOK,
			keyFunc: func(r *http.Request) string {
				return "burst-client"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			logger := testhelpers.NewMockLogger()
			metrics := testhelpers.NewMockMetricsCollector()
			limiter := NewTokenBucketRateLimiter(tt.requestsPerSec, tt.burstSize, logger)

			// Create rate limit middleware
			middleware := &rateLimitMiddleware{
				limiter: limiter,
				keyFunc: tt.keyFunc,
				logger:  logger,
				metrics: metrics,
			}

			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("success"))
			})

			// Act & Assert - Make requests within limit
			for i := 0; i < tt.requestCount; i++ {
				req := httptest.NewRequest(http.MethodGet, "/test", nil)
				ctx := WithRequestID(req.Context(), "rate-limit-test")
				req = req.WithContext(ctx)
				rec := httptest.NewRecorder()

				wrappedHandler := middleware.Wrap(handler)
				wrappedHandler.ServeHTTP(rec, req)

				assert.Equal(t, tt.expectedStatus, rec.Code)
				if rec.Code == http.StatusOK {
					assert.Equal(t, "success", rec.Body.String())
				}
			}
		})
	}
}

func TestRateLimitMiddleware_Wrap_RateLimitExceeded(t *testing.T) {
	tests := []struct {
		name            string
		requestsPerSec  float64
		burstSize       int
		requestCount    int
		expectedBlocked int
		expectedHeaders map[string]string
	}{
		{
			name:            "RateLimitExceeded_BlocksExcess",
			requestsPerSec:  1.0,
			burstSize:       2,
			requestCount:    5,
			expectedBlocked: 3,
			expectedHeaders: map[string]string{
				"Content-Type": "application/json",
			},
		},
		{
			name:            "RateLimitExceeded_SingleRequest",
			requestsPerSec:  0.0, // Zero rate blocks all
			burstSize:       0,
			requestCount:    1,
			expectedBlocked: 1,
			expectedHeaders: map[string]string{
				"Content-Type": "application/json",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			logger := testhelpers.NewMockLogger()
			metrics := testhelpers.NewMockMetricsCollector()
			limiter := NewTokenBucketRateLimiter(tt.requestsPerSec, tt.burstSize, logger)

			// Expect warning logs for blocked requests
			logger.On("Warn",
				mock.Anything,
				"Rate limit exceeded",
				mock.Anything,
			).Return().Times(tt.expectedBlocked)

			// Expect metrics calls for blocked requests
			metrics.On("RecordRateLimitHit",
				mock.AnythingOfType("string"),
			).Return().Times(tt.expectedBlocked)

			middleware := &rateLimitMiddleware{
				limiter: limiter,
				keyFunc: func(r *http.Request) string {
					return "blocked-client"
				},
				logger:  logger,
				metrics: metrics,
			}

			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("success"))
			})

			blockedCount := 0

			// Act
			for i := 0; i < tt.requestCount; i++ {
				req := httptest.NewRequest(http.MethodGet, "/test", nil)
				ctx := WithRequestID(req.Context(), "rate-limit-blocked")
				req = req.WithContext(ctx)
				rec := httptest.NewRecorder()

				wrappedHandler := middleware.Wrap(handler)
				wrappedHandler.ServeHTTP(rec, req)

				if rec.Code == http.StatusTooManyRequests {
					blockedCount++

					// Verify rate limit headers
					for header, expectedValue := range tt.expectedHeaders {
						assert.Equal(t, expectedValue, rec.Header().Get(header))
					}

					// Verify rate limit specific headers are present
					assert.NotEmpty(t, rec.Header().Get("X-RateLimit-Limit"))
					assert.NotEmpty(t, rec.Header().Get("X-RateLimit-Remaining"))
					assert.NotEmpty(t, rec.Header().Get("Retry-After"))
				}
			}

			// Assert
			assert.Equal(t, tt.expectedBlocked, blockedCount)
			logger.AssertExpectations(t)
			metrics.AssertExpectations(t)
		})
	}
}

func TestRateLimitMiddleware_Wrap_KeyGeneration(t *testing.T) {
	tests := []struct {
		name     string
		keyFunc  func(*http.Request) string
		setupReq func() *http.Request
		wantKey  string
	}{
		{
			name: "KeyGeneration_ByIP",
			keyFunc: func(r *http.Request) string {
				return r.RemoteAddr
			},
			setupReq: func() *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/test", nil)
				req.RemoteAddr = "192.168.1.100:12345"
				return req
			},
			wantKey: "192.168.1.100:12345",
		},
		{
			name: "KeyGeneration_ByUserAgent",
			keyFunc: func(r *http.Request) string {
				return r.Header.Get("User-Agent")
			},
			setupReq: func() *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/test", nil)
				req.Header.Set("User-Agent", "test-client/1.0")
				return req
			},
			wantKey: "test-client/1.0",
		},
		{
			name: "KeyGeneration_ByCustomHeader",
			keyFunc: func(r *http.Request) string {
				return r.Header.Get("X-Client-ID")
			},
			setupReq: func() *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/test", nil)
				req.Header.Set("X-Client-ID", "client-123")
				return req
			},
			wantKey: "client-123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			logger := testhelpers.NewMockLogger()
			metrics := testhelpers.NewMockMetricsCollector()
			limiter := NewTokenBucketRateLimiter(1.0, 1, logger)

			var capturedKey string
			middleware := &rateLimitMiddleware{
				limiter: limiter,
				keyFunc: func(r *http.Request) string {
					key := tt.keyFunc(r)
					capturedKey = key
					return key
				},
				logger:  logger,
				metrics: metrics,
			}

			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			req := tt.setupReq()
			ctx := WithRequestID(req.Context(), "key-gen-test")
			req = req.WithContext(ctx)
			rec := httptest.NewRecorder()

			// Act
			wrappedHandler := middleware.Wrap(handler)
			wrappedHandler.ServeHTTP(rec, req)

			// Assert
			assert.Equal(t, tt.wantKey, capturedKey)
		})
	}
}

func TestRateLimitMiddleware_Wrap_ResponseFormat(t *testing.T) {
	tests := []struct {
		name         string
		expectedJSON []string
	}{
		{
			name: "ResponseFormat_JSONError",
			expectedJSON: []string{
				`"error"`,
				`"Rate limit exceeded"`,
				`"status"`,
				`429`,
				`"retry_after"`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			logger := testhelpers.NewMockLogger()
			metrics := testhelpers.NewMockMetricsCollector()
			limiter := NewTokenBucketRateLimiter(0.0, 0, logger) // Block all requests

			logger.On("Warn",
				mock.Anything,
				"Rate limit exceeded",
				mock.Anything,
			).Return()

			metrics.On("RecordRateLimitHit",
				mock.AnythingOfType("string"),
			).Return()

			middleware := &rateLimitMiddleware{
				limiter: limiter,
				keyFunc: func(r *http.Request) string {
					return "json-test-client"
				},
				logger:  logger,
				metrics: metrics,
			}

			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			ctx := WithRequestID(req.Context(), "json-format-test")
			req = req.WithContext(ctx)
			rec := httptest.NewRecorder()

			// Act
			wrappedHandler := middleware.Wrap(handler)
			wrappedHandler.ServeHTTP(rec, req)

			// Assert
			assert.Equal(t, http.StatusTooManyRequests, rec.Code)
			assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

			responseBody := rec.Body.String()
			for _, expectedField := range tt.expectedJSON {
				assert.Contains(t, responseBody, expectedField)
			}

			logger.AssertExpectations(t)
			metrics.AssertExpectations(t)
		})
	}
}

func TestRateLimitMiddleware_Integration(t *testing.T) {
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
			limiter := NewTokenBucketRateLimiter(10.0, 5, logger)

			rateLimit := &rateLimitMiddleware{
				limiter: limiter,
				keyFunc: func(r *http.Request) string {
					return "integration-client"
				},
				logger:  logger,
				metrics: metrics,
			}

			// Create a middleware that adds a custom header
			headerMw := MiddlewareFunc(func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("X-Integration", "test")
					next.ServeHTTP(w, r)
				})
			})

			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("integration success"))
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			ctx := WithRequestID(req.Context(), "integration-test")
			req = req.WithContext(ctx)
			rec := httptest.NewRecorder()

			// Act - Chain rate limit with header middleware
			chain := NewChain(rateLimit, headerMw)
			finalHandler := chain.Then(handler)
			finalHandler.ServeHTTP(rec, req)

			// Assert
			assert.Equal(t, http.StatusOK, rec.Code)
			assert.Equal(t, "integration success", rec.Body.String())
			assert.Equal(t, "test", rec.Header().Get("X-Integration"))
		})
	}
}

// Benchmark tests for rate limiting middleware
func BenchmarkTokenBucketRateLimiter_Allow(b *testing.B) {
	logger := &testhelpers.MockLogger{}
	limiter := NewTokenBucketRateLimiter(1000.0, 1000, logger)
	ctx := context.Background()
	key := "benchmark-key"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = limiter.Allow(ctx, key)
	}
}

func BenchmarkRateLimitMiddleware_AllowedRequests(b *testing.B) {
	logger := &testhelpers.MockLogger{}
	metrics := &testhelpers.MockMetricsCollector{}
	limiter := NewTokenBucketRateLimiter(float64(b.N), b.N, logger) // Allow all requests

	middleware := &rateLimitMiddleware{
		limiter: limiter,
		keyFunc: func(r *http.Request) string {
			return "bench-client"
		},
		logger:  logger,
		metrics: metrics,
	}

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

func BenchmarkRateLimitMiddleware_BlockedRequests(b *testing.B) {
	logger := &testhelpers.MockLogger{}
	metrics := &testhelpers.MockMetricsCollector{}
	limiter := NewTokenBucketRateLimiter(0.0, 0, logger) // Block all requests

	// Setup mocks to avoid logging overhead in benchmark
	logger.On("Warn",
		mock.Anything,
		mock.Anything,
		mock.Anything,
	).Return()

	metrics.On("RecordRateLimitHit",
		mock.AnythingOfType("string"),
	).Return()

	middleware := &rateLimitMiddleware{
		limiter: limiter,
		keyFunc: func(r *http.Request) string {
			return "blocked-bench-client"
		},
		logger:  logger,
		metrics: metrics,
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := middleware.Wrap(handler)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	ctx := WithRequestID(req.Context(), "bench-blocked")
	req = req.WithContext(ctx)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		wrappedHandler.ServeHTTP(rec, req)
	}
}
