// Package middleware implements rate limiting middleware using token bucket algorithm.
package middleware

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"github.com/albedosehen/rwwwrse/internal/observability"
)

// tokenBucketRateLimiter implements RateLimiter using token bucket algorithm.
type tokenBucketRateLimiter struct {
	limiters       map[string]*rate.Limiter
	mu             sync.RWMutex
	requestsPerSec rate.Limit
	burstSize      int
	cleanupTicker  *time.Ticker
	stopCleanup    chan struct{}
	logger         observability.Logger
}

// NewTokenBucketRateLimiter creates a new token bucket rate limiter.
func NewTokenBucketRateLimiter(requestsPerSec float64, burstSize int, logger observability.Logger) RateLimiter {
	rl := &tokenBucketRateLimiter{
		limiters:       make(map[string]*rate.Limiter),
		requestsPerSec: rate.Limit(requestsPerSec),
		burstSize:      burstSize,
		cleanupTicker:  time.NewTicker(10 * time.Minute), // Cleanup every 10 minutes
		stopCleanup:    make(chan struct{}),
		logger:         logger,
	}

	// Start cleanup goroutine
	go rl.cleanupLoop()

	return rl
}

// Allow checks if a request should be allowed based on the key.
func (rl *tokenBucketRateLimiter) Allow(ctx context.Context, key string) bool {
	limiter := rl.getLimiter(key)
	return limiter.Allow()
}

// Reset resets the rate limit for a specific key.
func (rl *tokenBucketRateLimiter) Reset(key string) error {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	delete(rl.limiters, key)
	return nil
}

// Stats returns rate limiting statistics for a key.
func (rl *tokenBucketRateLimiter) Stats(key string) RateLimitStats {
	limiter := rl.getLimiter(key)

	// Calculate remaining tokens (approximate)
	reservation := limiter.Reserve()
	remaining := rl.burstSize
	if !reservation.OK() {
		remaining = 0
	} else {
		delay := reservation.Delay()
		reservation.Cancel() // Cancel the reservation since we're just checking

		if delay > 0 {
			remaining = 0
		}
	}

	return RateLimitStats{
		Requests:   rl.burstSize - remaining,
		Remaining:  remaining,
		ResetTime:  time.Now().Add(time.Second / time.Duration(rl.requestsPerSec)),
		RetryAfter: time.Second / time.Duration(rl.requestsPerSec),
	}
}

// Cleanup removes expired rate limit entries.
func (rl *tokenBucketRateLimiter) Cleanup(ctx context.Context) error {
	// For token bucket, we don't need explicit cleanup since limiters
	// will naturally reset based on time
	return nil
}

// getLimiter returns or creates a rate limiter for the given key.
func (rl *tokenBucketRateLimiter) getLimiter(key string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	limiter, exists := rl.limiters[key]
	if !exists {
		limiter = rate.NewLimiter(rl.requestsPerSec, rl.burstSize)
		rl.limiters[key] = limiter
	}

	return limiter
}

// cleanupLoop periodically removes unused limiters.
func (rl *tokenBucketRateLimiter) cleanupLoop() {
	for {
		select {
		case <-rl.cleanupTicker.C:
			rl.performCleanup()
		case <-rl.stopCleanup:
			rl.cleanupTicker.Stop()
			return
		}
	}
}

// performCleanup removes limiters that haven't been used recently.
func (rl *tokenBucketRateLimiter) performCleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Remove limiters that have full capacity (haven't been used recently)
	for key, limiter := range rl.limiters {
		// Check if limiter has full capacity
		if limiter.Tokens() >= float64(rl.burstSize) {
			delete(rl.limiters, key)
		}
	}

	if rl.logger != nil {
		rl.logger.Debug(context.Background(), "Rate limiter cleanup completed",
			observability.Int("active_limiters", len(rl.limiters)),
		)
	}
}

// Stop stops the cleanup goroutine.
func (rl *tokenBucketRateLimiter) Stop() {
	close(rl.stopCleanup)
}

// rateLimitMiddleware implements rate limiting middleware.
type rateLimitMiddleware struct {
	limiter RateLimiter
	keyFunc func(*http.Request) string
	logger  observability.Logger
	metrics observability.MetricsCollector
}

// NewRateLimitMiddleware creates a new rate limiting middleware.
func NewRateLimitMiddleware(
	limiter RateLimiter,
	logger observability.Logger,
	metrics observability.MetricsCollector,
) Middleware {
	return &rateLimitMiddleware{
		limiter: limiter,
		keyFunc: defaultKeyFunc,
		logger:  logger,
		metrics: metrics,
	}
}

// NewRateLimitMiddlewareWithKeyFunc creates a new rate limiting middleware with custom key function.
func NewRateLimitMiddlewareWithKeyFunc(
	limiter RateLimiter,
	keyFunc func(*http.Request) string,
	logger observability.Logger,
	metrics observability.MetricsCollector,
) Middleware {
	return &rateLimitMiddleware{
		limiter: limiter,
		keyFunc: keyFunc,
		logger:  logger,
		metrics: metrics,
	}
}

// Wrap implements the Middleware interface.
func (m *rateLimitMiddleware) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := m.keyFunc(r)

		if !m.limiter.Allow(r.Context(), key) {
			m.handleRateLimit(w, r, key)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// handleRateLimit handles rate limit exceeded responses.
func (m *rateLimitMiddleware) handleRateLimit(w http.ResponseWriter, r *http.Request, key string) {
	stats := m.limiter.Stats(key)
	requestID := GetRequestID(r.Context())

	// Log rate limit hit
	if m.logger != nil {
		m.logger.Warn(r.Context(), "Rate limit exceeded",
			observability.String("request_id", requestID),
			observability.String("key", key),
			observability.String("method", r.Method),
			observability.String("path", r.URL.Path),
			observability.Int("remaining", stats.Remaining),
		)
	}

	// Record metrics
	if m.metrics != nil {
		m.metrics.RecordRateLimitHit(key)
	}

	// Set rate limit headers
	w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", stats.Requests+stats.Remaining))
	w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", stats.Remaining))
	w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", stats.ResetTime.Unix()))
	w.Header().Set("Retry-After", fmt.Sprintf("%.0f", stats.RetryAfter.Seconds()))

	// Return 429 Too Many Requests
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusTooManyRequests)

	response := fmt.Sprintf(`{
		"error": "Rate limit exceeded",
		"status": 429,
		"request_id": "%s",
		"retry_after": %.0f,
		"timestamp": "%s"
	}`, requestID, stats.RetryAfter.Seconds(), time.Now().UTC().Format(time.RFC3339))

	_, _ = w.Write([]byte(response))
}

// defaultKeyFunc extracts client IP from request for rate limiting.
func defaultKeyFunc(r *http.Request) string {
	// Check for forwarded IP first
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP in the list
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			ip := strings.TrimSpace(ips[0])
			if ip != "" {
				return ip
			}
		}
	}

	// Check for real IP header
	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return realIP
	}

	// Fall back to remote address
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}

	return host
}
