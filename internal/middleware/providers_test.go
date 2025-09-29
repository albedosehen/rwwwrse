package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/albedosehen/rwwwrse/internal/config"
	testhelpers "github.com/albedosehen/rwwwrse/internal/testing"
)

func TestNewMiddlewareChain(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "NewMiddlewareChain_CreatesValidChain",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			chain := NewMiddlewareChain()

			// Assert
			assert.NotNil(t, chain)
			assert.Implements(t, (*Chain)(nil), chain)
		})
	}
}

func TestNewDefaultRecoveryMiddleware(t *testing.T) {
	tests := []struct {
		name    string
		logger  bool
		metrics bool
	}{
		{
			name:    "NewDefaultRecoveryMiddleware_WithValidDeps",
			logger:  true,
			metrics: true,
		},
		{
			name:    "NewDefaultRecoveryMiddleware_WithNilLogger",
			logger:  false,
			metrics: true,
		},
		{
			name:    "NewDefaultRecoveryMiddleware_WithNilMetrics",
			logger:  true,
			metrics: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			var logger *testhelpers.MockLogger
			var metrics *testhelpers.MockMetricsCollector

			if tt.logger {
				logger = testhelpers.NewMockLogger()
			}
			if tt.metrics {
				metrics = testhelpers.NewMockMetricsCollector()
			}

			// Act
			middleware := NewDefaultRecoveryMiddleware(logger, metrics)

			// Assert
			assert.NotNil(t, middleware)
			assert.Implements(t, (*Middleware)(nil), middleware)
		})
	}
}

func TestNewDefaultLoggingMiddleware(t *testing.T) {
	tests := []struct {
		name    string
		logger  bool
		metrics bool
	}{
		{
			name:    "NewDefaultLoggingMiddleware_WithValidDeps",
			logger:  true,
			metrics: true,
		},
		{
			name:    "NewDefaultLoggingMiddleware_WithNilLogger",
			logger:  false,
			metrics: true,
		},
		{
			name:    "NewDefaultLoggingMiddleware_WithNilMetrics",
			logger:  true,
			metrics: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			var logger *testhelpers.MockLogger
			var metrics *testhelpers.MockMetricsCollector

			if tt.logger {
				logger = testhelpers.NewMockLogger()
			}
			if tt.metrics {
				metrics = testhelpers.NewMockMetricsCollector()
			}

			// Act
			middleware := NewDefaultLoggingMiddleware(logger, metrics)

			// Assert
			assert.NotNil(t, middleware)
			assert.Implements(t, (*Middleware)(nil), middleware)
		})
	}
}

func TestNewDefaultCORSMiddleware(t *testing.T) {
	tests := []struct {
		name    string
		logger  bool
		metrics bool
	}{
		{
			name:    "NewDefaultCORSMiddleware_WithValidDeps",
			logger:  true,
			metrics: true,
		},
		{
			name:    "NewDefaultCORSMiddleware_WithNilDeps",
			logger:  false,
			metrics: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			var logger *testhelpers.MockLogger
			var metrics *testhelpers.MockMetricsCollector

			if tt.logger {
				logger = testhelpers.NewMockLogger()
			}
			if tt.metrics {
				metrics = testhelpers.NewMockMetricsCollector()
			}

			// Act
			middleware := NewDefaultCORSMiddleware(logger, metrics)

			// Assert
			assert.NotNil(t, middleware)
			assert.Implements(t, (*Middleware)(nil), middleware)
		})
	}
}

func TestNewDefaultSecurityHeadersMiddleware(t *testing.T) {
	tests := []struct {
		name    string
		logger  bool
		metrics bool
	}{
		{
			name:    "NewDefaultSecurityHeadersMiddleware_WithValidDeps",
			logger:  true,
			metrics: true,
		},
		{
			name:    "NewDefaultSecurityHeadersMiddleware_WithNilDeps",
			logger:  false,
			metrics: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			var logger *testhelpers.MockLogger
			var metrics *testhelpers.MockMetricsCollector

			if tt.logger {
				logger = testhelpers.NewMockLogger()
			}
			if tt.metrics {
				metrics = testhelpers.NewMockMetricsCollector()
			}

			// Act
			middleware := NewDefaultSecurityHeadersMiddleware(logger, metrics)

			// Assert
			assert.NotNil(t, middleware)
			assert.Implements(t, (*Middleware)(nil), middleware)
		})
	}
}

func TestNewDefaultRateLimitMiddleware(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "NewDefaultRateLimitMiddleware_WithValidDeps",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			cfg := &config.Config{}
			logger := testhelpers.NewMockLogger()
			metrics := testhelpers.NewMockMetricsCollector()

			// Act
			middleware := NewDefaultRateLimitMiddleware(cfg, logger, metrics)

			// Assert
			assert.NotNil(t, middleware)
			assert.Implements(t, (*Middleware)(nil), middleware)
		})
	}
}

func TestNewDefaultTokenBucketRateLimiter(t *testing.T) {
	tests := []struct {
		name   string
		logger bool
	}{
		{
			name:   "NewDefaultTokenBucketRateLimiter_WithLogger",
			logger: true,
		},
		{
			name:   "NewDefaultTokenBucketRateLimiter_WithoutLogger",
			logger: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			var logger *testhelpers.MockLogger
			if tt.logger {
				logger = testhelpers.NewMockLogger()
			}

			// Act
			limiter := NewDefaultTokenBucketRateLimiter(logger)

			// Assert
			assert.NotNil(t, limiter)
			assert.Implements(t, (*RateLimiter)(nil), limiter)
		})
	}
}

func TestSimpleBucketRateLimiter_Allow(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		expected bool
	}{
		{
			name:     "Allow_ReturnsTrue",
			key:      "test-key",
			expected: true, // Simple implementation always returns true
		},
		{
			name:     "Allow_DifferentKey",
			key:      "another-key",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			logger := testhelpers.NewMockLogger()
			limiter := NewDefaultTokenBucketRateLimiter(logger)
			ctx := context.Background()

			// Act
			result := limiter.Allow(ctx, tt.key)

			// Assert
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSimpleBucketRateLimiter_Reset(t *testing.T) {
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
			limiter := NewDefaultTokenBucketRateLimiter(logger)

			// Act
			err := limiter.Reset(tt.key)

			// Assert
			assert.NoError(t, err)
		})
	}
}

func TestSimpleBucketRateLimiter_Stats(t *testing.T) {
	tests := []struct {
		name              string
		key               string
		expectedRemaining int
	}{
		{
			name:              "Stats_ReturnsDefaultValues",
			key:               "stats-key",
			expectedRemaining: 200, // Default burst size
		},
		{
			name:              "Stats_DifferentKey",
			key:               "another-stats-key",
			expectedRemaining: 200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			logger := testhelpers.NewMockLogger()
			limiter := NewDefaultTokenBucketRateLimiter(logger)

			// Act
			stats := limiter.Stats(tt.key)

			// Assert
			assert.Equal(t, 0, stats.Requests)
			assert.Equal(t, tt.expectedRemaining, stats.Remaining)
			assert.False(t, stats.ResetTime.IsZero())
			assert.Equal(t, time.Second, stats.RetryAfter)
		})
	}
}

func TestSimpleBucketRateLimiter_Cleanup(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "Cleanup_ClearsLimiters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			logger := testhelpers.NewMockLogger()
			limiter := NewDefaultTokenBucketRateLimiter(logger)
			ctx := context.Background()

			// Act
			err := limiter.Cleanup(ctx)

			// Assert
			assert.NoError(t, err)
		})
	}
}

func TestDefaultMiddlewareConfig(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "DefaultMiddlewareConfig_ReturnsValidConfig",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			cfg := DefaultMiddlewareConfig()

			// Assert
			assert.True(t, cfg.Recovery.Enabled)
			assert.True(t, cfg.Recovery.StackTrace)
			assert.True(t, cfg.Logging.LogRequests)
			assert.True(t, cfg.Logging.LogResponses)
			assert.True(t, cfg.RateLimit.Enabled)
			assert.Equal(t, float64(100), cfg.RateLimit.RequestsPerSecond)
			assert.Equal(t, 200, cfg.RateLimit.BurstSize)
		})
	}
}

func TestCreateCompleteMiddlewareChain(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "CreateCompleteMiddlewareChain_WithValidDeps",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			cfg := &config.Config{}
			logger := testhelpers.NewMockLogger()
			metrics := testhelpers.NewMockMetricsCollector()

			// Act
			chain := CreateCompleteMiddlewareChain(cfg, logger, metrics)

			// Assert
			assert.NotNil(t, chain)
			assert.Implements(t, (*Chain)(nil), chain)
		})
	}
}

func TestValidateMiddlewareConfig_Providers(t *testing.T) {
	tests := []struct {
		name        string
		config      MiddlewareConfig
		expectError bool
		errorType   error
	}{
		{
			name:        "ValidConfig",
			config:      DefaultMiddlewareConfig(),
			expectError: false,
		},
		{
			name: "InvalidRateLimit_ZeroRequestsPerSecond",
			config: func() MiddlewareConfig {
				cfg := DefaultMiddlewareConfig()
				cfg.RateLimit.RequestsPerSecond = 0
				return cfg
			}(),
			expectError: true,
			errorType:   ErrInvalidRateLimit,
		},
		{
			name: "InvalidBurstSize_NegativeBurstSize",
			config: func() MiddlewareConfig {
				cfg := DefaultMiddlewareConfig()
				cfg.RateLimit.BurstSize = -5
				return cfg
			}(),
			expectError: true,
			errorType:   ErrInvalidBurstSize,
		},
		{
			name: "RateLimitDisabled",
			config: func() MiddlewareConfig {
				cfg := DefaultMiddlewareConfig()
				cfg.RateLimit.Enabled = false
				cfg.RateLimit.RequestsPerSecond = 0
				cfg.RateLimit.BurstSize = 0
				return cfg
			}(),
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			err := ValidateMiddlewareConfig(tt.config)

			// Assert
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorType != nil {
					assert.ErrorIs(t, err, tt.errorType)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestProviderSetExists(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "ProviderSet_IsNotNil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Assert
			assert.NotNil(t, ProviderSet)
		})
	}
}

func TestMiddlewareConfigStructure(t *testing.T) {
	tests := []struct {
		name   string
		config MiddlewareConfig
	}{
		{
			name:   "MiddlewareConfig_DefaultStructure",
			config: DefaultMiddlewareConfig(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Assert structure fields exist
			assert.NotNil(t, tt.config.Recovery)
			assert.NotNil(t, tt.config.Logging)
			assert.NotNil(t, tt.config.CORS)
			assert.NotNil(t, tt.config.Security)
			assert.NotNil(t, tt.config.RateLimit)
		})
	}
}

func TestErrorConstants(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "ErrInvalidRateLimit_HasCorrectMessage",
			err:      ErrInvalidRateLimit,
			expected: "invalid rate limit: requests per second must be positive",
		},
		{
			name:     "ErrInvalidBurstSize_HasCorrectMessage",
			err:      ErrInvalidBurstSize,
			expected: "invalid burst size: must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Assert
			assert.Equal(t, tt.expected, tt.err.Error())
		})
	}
}

// Integration test for complete middleware chain
func TestCompleteMiddlewareChain_Integration(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "CompleteChain_CreatesWorkingChain",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			cfg := &config.Config{}
			logger := testhelpers.NewMockLogger()
			metrics := testhelpers.NewMockMetricsCollector()

			// Mock logger calls that may occur during middleware execution
			logger.On("Info",
				mock.Anything,
				mock.Anything,
				mock.Anything,
			).Return().Maybe()

			logger.On("Debug",
				mock.Anything,
				mock.Anything,
				mock.Anything,
			).Return().Maybe()

			// Act
			chain := CreateCompleteMiddlewareChain(cfg, logger, metrics)

			// Verify chain structure
			require.NotNil(t, chain)

			// Test that chain can be used (basic functionality)
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("OK"))
			})
			wrappedHandler := chain.Then(handler)

			// Assert
			assert.NotNil(t, wrappedHandler)

			// Test that the wrapped handler can handle a request
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			rec := httptest.NewRecorder()

			// This shouldn't panic
			assert.NotPanics(t, func() {
				wrappedHandler.ServeHTTP(rec, req)
			})
		})
	}
}

// Benchmark tests for provider functions
func BenchmarkNewMiddlewareChain(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = NewMiddlewareChain()
	}
}

func BenchmarkNewDefaultTokenBucketRateLimiter(b *testing.B) {
	logger := &testhelpers.MockLogger{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewDefaultTokenBucketRateLimiter(logger)
	}
}

func BenchmarkCreateCompleteMiddlewareChain(b *testing.B) {
	cfg := &config.Config{}
	logger := &testhelpers.MockLogger{}
	metrics := &testhelpers.MockMetricsCollector{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = CreateCompleteMiddlewareChain(cfg, logger, metrics)
	}
}

func BenchmarkValidateMiddlewareConfig(b *testing.B) {
	config := DefaultMiddlewareConfig()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ValidateMiddlewareConfig(config)
	}
}
