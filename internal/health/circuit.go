// Package health provides circuit breaker implementation for service resilience.
package health

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/albedosehen/rwwwrse/internal/observability"
)

// circuitBreaker implements the CircuitBreaker interface.
type circuitBreaker struct {
	config   CircuitBreakerConfig
	circuits map[string]*circuitState
	mu       sync.RWMutex
	logger   observability.Logger
	metrics  observability.MetricsCollector
}

// circuitState holds the state of a single circuit.
type circuitState struct {
	state            CircuitState
	failures         int
	successes        int
	lastFailureTime  time.Time
	lastSuccessTime  time.Time
	halfOpenRequests int
	mu               sync.RWMutex
}

// NewCircuitBreaker creates a new circuit breaker instance.
func NewCircuitBreaker(
	config CircuitBreakerConfig,
	logger observability.Logger,
	metrics observability.MetricsCollector,
) CircuitBreaker {
	return &circuitBreaker{
		config:   config,
		circuits: make(map[string]*circuitState),
		logger:   logger,
		metrics:  metrics,
	}
}

// Allow determines if a request should be allowed based on current circuit state.
func (cb *circuitBreaker) Allow(ctx context.Context, target string) bool {
	if target == "" {
		return false
	}

	cb.mu.RLock()
	circuit, exists := cb.circuits[target]
	cb.mu.RUnlock()

	if !exists {
		// Create new circuit in closed state
		cb.mu.Lock()
		// Double-check pattern
		if circuit, exists = cb.circuits[target]; !exists {
			circuit = &circuitState{
				state: CircuitClosed,
			}
			cb.circuits[target] = circuit
		}
		cb.mu.Unlock()
	}

	circuit.mu.RLock()
	defer circuit.mu.RUnlock()

	switch circuit.state {
	case CircuitClosed:
		// Always allow requests when circuit is closed
		return true

	case CircuitOpen:
		// Check if timeout has elapsed to transition to half-open
		if time.Since(circuit.lastFailureTime) >= cb.config.Timeout {
			// Transition to half-open state
			cb.transitionToHalfOpen(target, circuit)
			return true
		}
		// Circuit is open, reject request
		cb.logCircuitAction(ctx, target, "request_rejected", "circuit_open")
		return false

	case CircuitHalfOpen:
		// Allow limited requests in half-open state
		if circuit.halfOpenRequests < cb.config.MaxRequests {
			return true
		}
		// Too many requests in half-open state, reject
		cb.logCircuitAction(ctx, target, "request_rejected", "half_open_limit_exceeded")
		return false

	default:
		// Unknown state, default to reject for safety
		return false
	}
}

// RecordSuccess records a successful operation for the circuit.
func (cb *circuitBreaker) RecordSuccess(ctx context.Context, target string) {
	if target == "" {
		return
	}

	cb.mu.RLock()
	circuit, exists := cb.circuits[target]
	cb.mu.RUnlock()

	if !exists {
		// Create new circuit if it doesn't exist
		cb.mu.Lock()
		if circuit, exists = cb.circuits[target]; !exists {
			circuit = &circuitState{
				state: CircuitClosed,
			}
			cb.circuits[target] = circuit
		}
		cb.mu.Unlock()
	}

	circuit.mu.Lock()
	defer circuit.mu.Unlock()

	circuit.failures = 0
	circuit.successes++
	circuit.lastSuccessTime = time.Now()

	switch circuit.state {
	case CircuitClosed:
		// Already in correct state, nothing to do

	case CircuitOpen:
		// This shouldn't happen as requests should be rejected in open state
		cb.logCircuitAction(ctx, target, "unexpected_success", "circuit_was_open")

	case CircuitHalfOpen:
		// Count this success towards closing the circuit
		if circuit.successes >= cb.config.SuccessThreshold {
			circuit.state = CircuitClosed
			circuit.halfOpenRequests = 0
			cb.logCircuitAction(ctx, target, "circuit_closed", "success_threshold_reached")
		}
	}

	if cb.metrics != nil {
		// Record success metrics
		// TODO: Implement circuit breaker metrics recording
		// cb.metrics.RecordCircuitBreakerEvent(target, "success")
		_ = cb.metrics // Acknowledge metrics is available but not used yet
	}
}

// RecordFailure records a failed operation for the circuit.
func (cb *circuitBreaker) RecordFailure(ctx context.Context, target string, err error) {
	if target == "" {
		return
	}

	cb.mu.RLock()
	circuit, exists := cb.circuits[target]
	cb.mu.RUnlock()

	if !exists {
		// Create new circuit if it doesn't exist
		cb.mu.Lock()
		if circuit, exists = cb.circuits[target]; !exists {
			circuit = &circuitState{
				state: CircuitClosed,
			}
			cb.circuits[target] = circuit
		}
		cb.mu.Unlock()
	}

	circuit.mu.Lock()
	defer circuit.mu.Unlock()

	circuit.successes = 0
	circuit.failures++
	circuit.lastFailureTime = time.Now()

	switch circuit.state {
	case CircuitClosed:
		// Check if we should open the circuit
		if circuit.failures >= cb.config.FailureThreshold {
			circuit.state = CircuitOpen
			cb.logCircuitAction(ctx, target, "circuit_opened", "failure_threshold_reached")
		}

	case CircuitOpen:
		// Circuit is already open, just record the failure

	case CircuitHalfOpen:
		// Any failure in half-open state opens the circuit
		circuit.state = CircuitOpen
		circuit.halfOpenRequests = 0
		cb.logCircuitAction(ctx, target, "circuit_opened", "failure_in_half_open")
	}

	if cb.metrics != nil {
		// Record failure metrics
		// TODO: Implement circuit breaker metrics recording
		// cb.metrics.RecordCircuitBreakerEvent(target, "failure")
		_ = cb.metrics // Acknowledge metrics is available but not used yet
	}

	if cb.logger != nil {
		cb.logger.Warn(ctx, "Circuit breaker recorded failure",
			observability.String("target", target),
			observability.String("state", circuit.state.String()),
			observability.Int("failures", circuit.failures),
			observability.String("error", err.Error()),
		)
	}
}

// State returns the current circuit breaker state for the target.
func (cb *circuitBreaker) State(ctx context.Context, target string) CircuitState {
	if target == "" {
		return CircuitClosed // Default state for invalid target
	}

	cb.mu.RLock()
	circuit, exists := cb.circuits[target]
	cb.mu.RUnlock()

	if !exists {
		return CircuitClosed // Default state for non-existent circuit
	}

	circuit.mu.RLock()
	defer circuit.mu.RUnlock()

	// Check if we need to transition from open to half-open
	if circuit.state == CircuitOpen && time.Since(circuit.lastFailureTime) >= cb.config.Timeout {
		// Don't transition here, just return current state
		// Transition will happen on next Allow() call
		// TODO: Consider if we should transition here for consistency
		_ = circuit.state // Acknowledge state check is intentional
	}

	return circuit.state
}

// Reset manually resets the circuit breaker to closed state.
func (cb *circuitBreaker) Reset(ctx context.Context, target string) error {
	if target == "" {
		return fmt.Errorf("target cannot be empty")
	}

	cb.mu.RLock()
	circuit, exists := cb.circuits[target]
	cb.mu.RUnlock()

	if !exists {
		// Circuit doesn't exist, nothing to reset
		return nil
	}

	circuit.mu.Lock()
	defer circuit.mu.Unlock()

	previousState := circuit.state
	circuit.state = CircuitClosed
	circuit.failures = 0
	circuit.successes = 0
	circuit.halfOpenRequests = 0
	circuit.lastFailureTime = time.Time{}
	circuit.lastSuccessTime = time.Now()

	cb.logCircuitAction(ctx, target, "circuit_reset", fmt.Sprintf("from_%s", previousState.String()))

	if cb.logger != nil {
		cb.logger.Info(ctx, "Circuit breaker manually reset",
			observability.String("target", target),
			observability.String("previous_state", previousState.String()),
		)
	}

	return nil
}

// GetCircuitInfo returns detailed information about a circuit (for monitoring/debugging).
func (cb *circuitBreaker) GetCircuitInfo(target string) (*CircuitInfo, error) {
	if target == "" {
		return nil, fmt.Errorf("target cannot be empty")
	}

	cb.mu.RLock()
	circuit, exists := cb.circuits[target]
	cb.mu.RUnlock()

	if !exists {
		return &CircuitInfo{
			Target: target,
			State:  CircuitClosed,
		}, nil
	}

	circuit.mu.RLock()
	defer circuit.mu.RUnlock()

	return &CircuitInfo{
		Target:           target,
		State:            circuit.state,
		Failures:         circuit.failures,
		Successes:        circuit.successes,
		LastFailureTime:  circuit.lastFailureTime,
		LastSuccessTime:  circuit.lastSuccessTime,
		HalfOpenRequests: circuit.halfOpenRequests,
		FailureThreshold: cb.config.FailureThreshold,
		SuccessThreshold: cb.config.SuccessThreshold,
		Timeout:          cb.config.Timeout,
		MaxRequests:      cb.config.MaxRequests,
	}, nil
}

// GetAllCircuits returns information about all circuits.
func (cb *circuitBreaker) GetAllCircuits() map[string]*CircuitInfo {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	result := make(map[string]*CircuitInfo, len(cb.circuits))

	for target, circuit := range cb.circuits {
		circuit.mu.RLock()
		result[target] = &CircuitInfo{
			Target:           target,
			State:            circuit.state,
			Failures:         circuit.failures,
			Successes:        circuit.successes,
			LastFailureTime:  circuit.lastFailureTime,
			LastSuccessTime:  circuit.lastSuccessTime,
			HalfOpenRequests: circuit.halfOpenRequests,
			FailureThreshold: cb.config.FailureThreshold,
			SuccessThreshold: cb.config.SuccessThreshold,
			Timeout:          cb.config.Timeout,
			MaxRequests:      cb.config.MaxRequests,
		}
		circuit.mu.RUnlock()
	}

	return result
}

// transitionToHalfOpen transitions a circuit from open to half-open state.
// This should be called with the circuit read lock held.
func (cb *circuitBreaker) transitionToHalfOpen(target string, circuit *circuitState) {
	// Upgrade to write lock
	circuit.mu.RUnlock()
	circuit.mu.Lock()
	defer func() {
		circuit.mu.Unlock()
		circuit.mu.RLock() // Downgrade back to read lock
	}()

	// Double-check the state hasn't changed
	if circuit.state == CircuitOpen && time.Since(circuit.lastFailureTime) >= cb.config.Timeout {
		circuit.state = CircuitHalfOpen
		circuit.halfOpenRequests = 0
		circuit.successes = 0

		cb.logCircuitAction(context.Background(), target, "circuit_half_opened", "timeout_elapsed")
	}
}

// logCircuitAction logs circuit breaker actions for monitoring.
func (cb *circuitBreaker) logCircuitAction(ctx context.Context, target, action, reason string) {
	if cb.logger != nil {
		cb.logger.Info(ctx, "Circuit breaker action",
			observability.String("target", target),
			observability.String("action", action),
			observability.String("reason", reason),
		)
	}
}

// CircuitInfo provides detailed information about a circuit breaker's state.
type CircuitInfo struct {
	Target           string        `json:"target"`
	State            CircuitState  `json:"state"`
	Failures         int           `json:"failures"`
	Successes        int           `json:"successes"`
	LastFailureTime  time.Time     `json:"last_failure_time"`
	LastSuccessTime  time.Time     `json:"last_success_time"`
	HalfOpenRequests int           `json:"half_open_requests"`
	FailureThreshold int           `json:"failure_threshold"`
	SuccessThreshold int           `json:"success_threshold"`
	Timeout          time.Duration `json:"timeout"`
	MaxRequests      int           `json:"max_requests"`
}
