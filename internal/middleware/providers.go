// Package middleware provides Wire dependency injection providers for middleware components.
package middleware

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/wire"

	"github.com/albedosehen/rwwwrse/internal/config"
	"github.com/albedosehen/rwwwrse/internal/observability"
)

// ProviderSet is a Wire provider set for middleware components.
var ProviderSet = wire.NewSet(
	NewDefaultTokenBucketRateLimiter,
	CreateCompleteMiddlewareChain,
)

// NewMiddlewareChain creates a new middleware chain.
func NewMiddlewareChain() Chain {
	return &chain{
		middlewares: make([]Middleware, 0),
	}
}

// NewDefaultRecoveryMiddleware creates a recovery middleware with Wire.
func NewDefaultRecoveryMiddleware(
	logger observability.Logger,
	metrics observability.MetricsCollector,
) Middleware {
	return &recoveryMiddleware{
		logger: logger,
	}
}

// NewDefaultLoggingMiddleware creates a logging middleware with Wire.
func NewDefaultLoggingMiddleware(
	logger observability.Logger,
	metrics observability.MetricsCollector,
) Middleware {
	return &loggingMiddleware{
		config:  DefaultLoggingConfig(),
		logger:  logger,
		metrics: metrics,
	}
}

// NewDefaultCORSMiddleware creates a CORS middleware with Wire.
func NewDefaultCORSMiddleware(
	logger observability.Logger,
	metrics observability.MetricsCollector,
) Middleware {
	return &corsMiddleware{
		config:  DefaultCORSConfig(),
		logger:  logger,
		metrics: metrics,
	}
}

// NewDefaultSecurityHeadersMiddleware creates a security headers middleware with Wire.
func NewDefaultSecurityHeadersMiddleware(
	logger observability.Logger,
	metrics observability.MetricsCollector,
) Middleware {
	return &securityHeadersMiddleware{
		config:  DefaultSecurityConfig(),
		logger:  logger,
		metrics: metrics,
	}
}

// NewDefaultRateLimitMiddleware creates a rate limiting middleware with Wire.
func NewDefaultRateLimitMiddleware(
	cfg *config.Config,
	logger observability.Logger,
	metrics observability.MetricsCollector,
) Middleware {
	// Create rate limiter with configuration
	limiter := NewDefaultTokenBucketRateLimiter(logger)

	return &rateLimitMiddleware{
		limiter: limiter,
		keyFunc: defaultKeyFunc,
		logger:  logger,
		metrics: metrics,
	}
}

// NewDefaultTokenBucketRateLimiter creates a token bucket rate limiter with Wire.
func NewDefaultTokenBucketRateLimiter(
	logger observability.Logger,
) RateLimiter {
	rl := &simpleBucketRateLimiter{
		requestsPerSec: 100,
		burstSize:      200,
		logger:         logger,
		limiters:       make(map[string]*simpleLimiter),
	}

	// Start background cleanup
	rl.startCleanup()

	return rl
}

// Constants for rate limiter configuration.
const (
	defaultCleanupInterval = 10 * time.Minute
	maxIdleTime            = 30 * time.Minute
	nanosPerSecond         = 1e9
)

// simpleBucketRateLimiter is a thread-safe token bucket rate limiter implementation.
type simpleBucketRateLimiter struct {
	requestsPerSec float64
	burstSize      int
	logger         observability.Logger
	limiters       map[string]*simpleLimiter
	mu             sync.RWMutex
	cleanupTicker  *time.Ticker
	stopCleanup    chan struct{}
	once           sync.Once
}

// simpleLimiter tracks rate limiting for a single key using token bucket algorithm.
type simpleLimiter struct {
	tokens     float64    // Current number of tokens (can be fractional)
	lastRefill time.Time  // Last time tokens were refilled
	maxTokens  int        // Maximum number of tokens (burst size)
	refillRate float64    // Tokens per second
	mu         sync.Mutex // Protects access to limiter state
}

// newSimpleLimiter creates a new rate limiter for a specific key.
func newSimpleLimiter(requestsPerSec float64, burstSize int) *simpleLimiter {
	now := time.Now()
	return &simpleLimiter{
		tokens:     float64(burstSize), // Start with full bucket
		lastRefill: now,
		maxTokens:  burstSize,
		refillRate: requestsPerSec,
	}
}

// allow checks if a request should be allowed and consumes a token if available.
func (sl *simpleLimiter) allow() bool {
	sl.mu.Lock()
	defer sl.mu.Unlock()

	now := time.Now()

	// Calculate tokens to add based on elapsed time
	elapsed := now.Sub(sl.lastRefill)
	tokensToAdd := sl.refillRate * elapsed.Seconds()

	// Add tokens, but don't exceed maximum
	sl.tokens = min(sl.tokens+tokensToAdd, float64(sl.maxTokens))
	sl.lastRefill = now

	// Check if we have at least one token
	if sl.tokens >= 1.0 {
		sl.tokens--
		return true
	}

	return false
}

// stats returns current statistics for this limiter.
func (sl *simpleLimiter) stats() RateLimitStats {
	sl.mu.Lock()
	defer sl.mu.Unlock()

	now := time.Now()

	// Calculate current tokens
	elapsed := now.Sub(sl.lastRefill)
	tokensToAdd := sl.refillRate * elapsed.Seconds()
	currentTokens := min(sl.tokens+tokensToAdd, float64(sl.maxTokens))

	remaining := int(currentTokens)
	requests := sl.maxTokens - remaining

	// Calculate when next token will be available
	var retryAfter time.Duration
	if remaining == 0 {
		retryAfter = time.Duration(nanosPerSecond / sl.refillRate)
	}

	return RateLimitStats{
		Requests:   requests,
		Remaining:  remaining,
		ResetTime:  now.Add(time.Duration(float64(sl.maxTokens)/sl.refillRate) * time.Second),
		RetryAfter: retryAfter,
	}
}

// isIdle checks if this limiter has been idle for too long.
func (sl *simpleLimiter) isIdle() bool {
	sl.mu.Lock()
	defer sl.mu.Unlock()

	return time.Since(sl.lastRefill) > maxIdleTime && sl.tokens >= float64(sl.maxTokens)
}

// Allow implements RateLimiter interface with proper token bucket algorithm.
func (rl *simpleBucketRateLimiter) Allow(ctx context.Context, key string) bool {
	if key == "" {
		return false // Invalid key
	}

	// Check context cancellation
	select {
	case <-ctx.Done():
		return false
	default:
	}

	limiter := rl.getLimiter(key)
	return limiter.allow()
}

// Reset implements RateLimiter interface with proper synchronization.
func (rl *simpleBucketRateLimiter) Reset(key string) error {
	if key == "" {
		return fmt.Errorf("invalid key: cannot be empty")
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	delete(rl.limiters, key)
	return nil
}

// Stats implements RateLimiter interface with accurate statistics.
func (rl *simpleBucketRateLimiter) Stats(key string) RateLimitStats {
	if key == "" {
		return RateLimitStats{
			Requests:   rl.burstSize,
			Remaining:  0,
			ResetTime:  time.Now().Add(time.Hour),
			RetryAfter: time.Second,
		}
	}

	limiter := rl.getLimiter(key)
	return limiter.stats()
}

// Cleanup implements RateLimiter interface with intelligent cleanup.
func (rl *simpleBucketRateLimiter) Cleanup(ctx context.Context) error {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Check context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Remove idle limiters to prevent memory leaks
	removed := 0
	for key, limiter := range rl.limiters {
		if limiter.isIdle() {
			delete(rl.limiters, key)
			removed++
		}
	}

	if rl.logger != nil && removed > 0 {
		rl.logger.Debug(ctx, "Rate limiter cleanup completed",
			observability.Int("removed_limiters", removed),
			observability.Int("active_limiters", len(rl.limiters)),
		)
	}

	return nil
}

// getLimiter returns or creates a rate limiter for the given key (thread-safe).
func (rl *simpleBucketRateLimiter) getLimiter(key string) *simpleLimiter {
	// First try with read lock for better performance
	rl.mu.RLock()
	if limiter, exists := rl.limiters[key]; exists {
		rl.mu.RUnlock()
		return limiter
	}
	rl.mu.RUnlock()

	// Need to create new limiter, acquire write lock
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Double-check pattern to avoid race condition
	if limiter, exists := rl.limiters[key]; exists {
		return limiter
	}

	// Create new limiter
	limiter := newSimpleLimiter(rl.requestsPerSec, rl.burstSize)
	rl.limiters[key] = limiter
	return limiter
}

// startCleanup starts the background cleanup goroutine.
func (rl *simpleBucketRateLimiter) startCleanup() {
	rl.once.Do(func() {
		rl.cleanupTicker = time.NewTicker(defaultCleanupInterval)
		rl.stopCleanup = make(chan struct{})

		go rl.cleanupLoop()
	})
}

// cleanupLoop runs periodic cleanup in a separate goroutine.
func (rl *simpleBucketRateLimiter) cleanupLoop() {
	defer rl.cleanupTicker.Stop()

	for {
		select {
		case <-rl.cleanupTicker.C:
			if err := rl.Cleanup(context.Background()); err != nil && rl.logger != nil {
				rl.logger.Error(context.Background(), err, "Rate limiter cleanup failed")
			}
		case <-rl.stopCleanup:
			return
		}
	}
}

// Stop gracefully stops the rate limiter and cleanup goroutine.
func (rl *simpleBucketRateLimiter) Stop() {
	if rl.stopCleanup != nil {
		close(rl.stopCleanup)
	}
}

// min returns the minimum of two float64 values.
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// MiddlewareConfig holds configuration for all middleware components.
type MiddlewareConfig struct {
	// Recovery middleware configuration
	Recovery struct {
		Enabled    bool `mapstructure:"enabled"`
		StackTrace bool `mapstructure:"stack_trace"`
	} `mapstructure:"recovery"`

	// Logging middleware configuration
	Logging LoggingConfig `mapstructure:"logging"`

	// CORS middleware configuration
	CORS CORSConfig `mapstructure:"cors"`

	// Security headers configuration
	Security SecurityConfig `mapstructure:"security"`

	// Rate limiting configuration
	RateLimit struct {
		Enabled           bool    `mapstructure:"enabled"`
		RequestsPerSecond float64 `mapstructure:"requests_per_second"`
		BurstSize         int     `mapstructure:"burst_size"`
	} `mapstructure:"rate_limit"`
}

// DefaultMiddlewareConfig returns a default middleware configuration.
func DefaultMiddlewareConfig() MiddlewareConfig {
	return MiddlewareConfig{
		Recovery: struct {
			Enabled    bool `mapstructure:"enabled"`
			StackTrace bool `mapstructure:"stack_trace"`
		}{
			Enabled:    true,
			StackTrace: true,
		},
		Logging:  DefaultLoggingConfig(),
		CORS:     DefaultCORSConfig(),
		Security: DefaultSecurityConfig(),
		RateLimit: struct {
			Enabled           bool    `mapstructure:"enabled"`
			RequestsPerSecond float64 `mapstructure:"requests_per_second"`
			BurstSize         int     `mapstructure:"burst_size"`
		}{
			Enabled:           true,
			RequestsPerSecond: 100,
			BurstSize:         200,
		},
	}
}

// CreateCompleteMiddlewareChain creates a complete middleware chain with all components.
func CreateCompleteMiddlewareChain(
	cfg *config.Config,
	logger observability.Logger,
	metrics observability.MetricsCollector,
) Chain {
	chain := NewMiddlewareChain()

	// Add middleware in order (they will be executed in reverse order)

	// 1. Recovery (outermost - catches panics from all other middleware)
	chain.Use(NewDefaultRecoveryMiddleware(logger, metrics))

	// 2. Logging (logs all requests/responses)
	chain.Use(NewDefaultLoggingMiddleware(logger, metrics))

	// 3. Security headers (sets security headers)
	chain.Use(NewDefaultSecurityHeadersMiddleware(logger, metrics))

	// 4. CORS (handles cross-origin requests)
	chain.Use(NewDefaultCORSMiddleware(logger, metrics))

	// 5. Rate limiting (innermost - closest to business logic)
	chain.Use(NewDefaultRateLimitMiddleware(cfg, logger, metrics))

	return chain
}

// ValidateMiddlewareConfig validates the middleware configuration.
func ValidateMiddlewareConfig(cfg MiddlewareConfig) error {
	// Validate CORS configuration
	corsValidator := NewCORSValidator()
	if err := corsValidator.Validate(cfg.CORS); err != nil {
		return err
	}

	// Validate security headers configuration
	securityValidator := NewSecurityHeadersValidator()
	if err := securityValidator.Validate(cfg.Security); err != nil {
		return err
	}

	// Validate rate limiting configuration
	if cfg.RateLimit.Enabled {
		if cfg.RateLimit.RequestsPerSecond <= 0 {
			return ErrInvalidRateLimit
		}
		if cfg.RateLimit.BurstSize <= 0 {
			return ErrInvalidBurstSize
		}
	}

	return nil
}

// Middleware configuration errors.
var (
	ErrInvalidRateLimit = fmt.Errorf("invalid rate limit: requests per second must be positive")
	ErrInvalidBurstSize = fmt.Errorf("invalid burst size: must be positive")
)
