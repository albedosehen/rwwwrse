// Package health provides health monitoring and circuit breaker functionality.
package health

import (
	"context"
	"fmt"
	"maps"
	"net/http"
	"sync"
	"time"

	"github.com/albedosehen/rwwwrse/internal/config"
	"github.com/albedosehen/rwwwrse/internal/observability"
)

// healthChecker implements the HealthChecker interface.
type healthChecker struct {
	config      config.HealthConfig
	targets     map[string]HealthTarget
	status      map[string]HealthStatus
	mu          sync.RWMutex
	subscribers []chan<- HealthEvent
	subMu       sync.RWMutex
	logger      observability.Logger
	metrics     observability.MetricsCollector
	stopChan    chan struct{}
	running     bool
	runMu       sync.Mutex
	httpClient  *http.Client
}

// NewHealthChecker creates a new health checker instance.
func NewHealthChecker(
	config config.HealthConfig,
	logger observability.Logger,
	metrics observability.MetricsCollector,
) HealthChecker {
	return &healthChecker{
		config:   config,
		targets:  make(map[string]HealthTarget),
		status:   make(map[string]HealthStatus),
		logger:   logger,
		metrics:  metrics,
		stopChan: make(chan struct{}),
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// Check performs a single health check for the specified target.
func (hc *healthChecker) Check(ctx context.Context, target HealthTarget) HealthStatus {
	if target == nil {
		return HealthStatus{
			Healthy:   false,
			LastCheck: time.Now(),
			Error:     fmt.Errorf("target cannot be nil"),
		}
	}

	start := time.Now()

	// Create request with context timeout
	requestCtx, cancel := context.WithTimeout(ctx, target.Timeout())
	defer cancel()

	req, err := http.NewRequestWithContext(requestCtx, http.MethodGet, target.URL(), nil)
	if err != nil {
		status := HealthStatus{
			Healthy:      false,
			LastCheck:    start,
			ResponseTime: time.Since(start),
			Error:        fmt.Errorf("failed to create request: %w", err),
		}
		hc.recordHealthCheck(target.Name(), false, status.ResponseTime)
		return status
	}

	// Add custom headers
	for key, value := range target.Headers() {
		req.Header.Set(key, value)
	}

	// Perform the health check
	resp, err := hc.httpClient.Do(req)
	responseTime := time.Since(start)

	if err != nil {
		status := HealthStatus{
			Healthy:      false,
			LastCheck:    start,
			ResponseTime: responseTime,
			Error:        fmt.Errorf("health check request failed: %w", err),
		}
		hc.recordHealthCheck(target.Name(), false, responseTime)
		return status
	}
	defer resp.Body.Close()

	// Check if status code indicates health
	expectedStatus := target.ExpectedStatus()
	if expectedStatus == 0 {
		expectedStatus = http.StatusOK
	}

	healthy := resp.StatusCode == expectedStatus
	var healthErr error
	if !healthy {
		healthErr = fmt.Errorf("unexpected status code: %d, expected: %d", resp.StatusCode, expectedStatus)
	}

	status := HealthStatus{
		Healthy:      healthy,
		LastCheck:    start,
		ResponseTime: responseTime,
		StatusCode:   resp.StatusCode,
		Error:        healthErr,
	}

	hc.recordHealthCheck(target.Name(), healthy, responseTime)

	if hc.logger != nil {
		if healthy {
			hc.logger.Debug(ctx, "Health check passed",
				observability.String("target", target.Name()),
				observability.String("url", target.URL()),
				observability.Duration("response_time", responseTime),
				observability.Int("status_code", resp.StatusCode),
			)
		} else {
			hc.logger.Warn(ctx, "Health check failed",
				observability.String("target", target.Name()),
				observability.String("url", target.URL()),
				observability.Duration("response_time", responseTime),
				observability.Int("status_code", resp.StatusCode),
				observability.String("error", healthErr.Error()),
			)
		}
	}

	return status
}

// StartMonitoring begins periodic health monitoring for registered targets.
func (hc *healthChecker) StartMonitoring(ctx context.Context) error {
	hc.runMu.Lock()
	defer hc.runMu.Unlock()

	if hc.running {
		return fmt.Errorf("health monitoring is already running")
	}

	if !hc.config.Enabled {
		if hc.logger != nil {
			hc.logger.Info(ctx, "Health monitoring is disabled")
		}
		return nil
	}

	hc.running = true
	hc.stopChan = make(chan struct{})

	if hc.logger != nil {
		hc.logger.Info(ctx, "Starting health monitoring",
			observability.Duration("interval", hc.config.Interval),
			observability.Int("target_count", len(hc.targets)),
		)
	}

	// Publish health check started events
	hc.publishEvent(HealthEvent{
		Type:      HealthCheckStarted,
		Timestamp: time.Now(),
	})

	go hc.monitoringLoop(ctx)

	return nil
}

// StopMonitoring halts all health monitoring operations.
func (hc *healthChecker) StopMonitoring() error {
	hc.runMu.Lock()
	defer hc.runMu.Unlock()

	if !hc.running {
		return nil
	}

	if hc.logger != nil {
		hc.logger.Info(context.Background(), "Stopping health monitoring")
	}

	close(hc.stopChan)
	hc.running = false

	// Publish health check stopped events
	hc.publishEvent(HealthEvent{
		Type:      HealthCheckStopped,
		Timestamp: time.Now(),
	})

	return nil
}

// Subscribe registers a channel to receive health status updates.
func (hc *healthChecker) Subscribe(ch chan<- HealthEvent) error {
	if ch == nil {
		return fmt.Errorf("channel cannot be nil")
	}

	hc.subMu.Lock()
	defer hc.subMu.Unlock()

	hc.subscribers = append(hc.subscribers, ch)
	return nil
}

// Unsubscribe removes a channel from health event notifications.
func (hc *healthChecker) Unsubscribe(ch chan<- HealthEvent) error {
	if ch == nil {
		return fmt.Errorf("channel cannot be nil")
	}

	hc.subMu.Lock()
	defer hc.subMu.Unlock()

	for i, subscriber := range hc.subscribers {
		if subscriber == ch {
			// Remove channel from slice
			hc.subscribers = append(hc.subscribers[:i], hc.subscribers[i+1:]...)
			return nil
		}
	}

	return fmt.Errorf("channel not found in subscribers")
}

// RegisterTarget adds a health target for monitoring.
func (hc *healthChecker) RegisterTarget(target HealthTarget) error {
	if target == nil {
		return fmt.Errorf("target cannot be nil")
	}

	name := target.Name()
	if name == "" {
		return fmt.Errorf("target name cannot be empty")
	}

	hc.mu.Lock()
	defer hc.mu.Unlock()

	hc.targets[name] = target

	if hc.logger != nil {
		hc.logger.Info(context.Background(), "Registered health target",
			observability.String("name", name),
			observability.String("url", target.URL()),
			observability.Duration("timeout", target.Timeout()),
		)
	}

	return nil
}

// UnregisterTarget removes a health target from monitoring.
func (hc *healthChecker) UnregisterTarget(name string) error {
	if name == "" {
		return fmt.Errorf("target name cannot be empty")
	}

	hc.mu.Lock()
	defer hc.mu.Unlock()

	delete(hc.targets, name)
	delete(hc.status, name)

	if hc.logger != nil {
		hc.logger.Info(context.Background(), "Unregistered health target",
			observability.String("name", name),
		)
	}

	return nil
}

// GetTargetStatus returns the current health status of a target.
func (hc *healthChecker) GetTargetStatus(name string) (HealthStatus, bool) {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	status, exists := hc.status[name]
	return status, exists
}

// GetAllTargetStatus returns the health status of all targets.
func (hc *healthChecker) GetAllTargetStatus() map[string]HealthStatus {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	result := make(map[string]HealthStatus, len(hc.status))
	maps.Copy(result, hc.status)
	return result
}

// monitoringLoop runs the periodic health check monitoring.
func (hc *healthChecker) monitoringLoop(ctx context.Context) {
	ticker := time.NewTicker(hc.config.Interval)
	defer ticker.Stop()

	// Perform initial health checks
	hc.checkAllTargets(ctx)

	for {
		select {
		case <-ctx.Done():
			if hc.logger != nil {
				hc.logger.Info(ctx, "Health monitoring stopped due to context cancellation")
			}
			return
		case <-hc.stopChan:
			if hc.logger != nil {
				hc.logger.Info(ctx, "Health monitoring stopped")
			}
			return
		case <-ticker.C:
			hc.checkAllTargets(ctx)
		}
	}
}

// checkAllTargets performs health checks on all registered targets.
func (hc *healthChecker) checkAllTargets(ctx context.Context) {
	hc.mu.RLock()
	targets := make(map[string]HealthTarget, len(hc.targets))
	maps.Copy(targets, hc.targets)
	hc.mu.RUnlock()

	for name, target := range targets {
		// Get previous status for comparison
		hc.mu.RLock()
		previousStatus, hadPrevious := hc.status[name]
		hc.mu.RUnlock()

		// Perform health check
		newStatus := hc.Check(ctx, target)

		// Update consecutive counters
		if hadPrevious {
			if newStatus.Healthy {
				if previousStatus.Healthy {
					newStatus.ConsecutiveSuccesses = previousStatus.ConsecutiveSuccesses + 1
				} else {
					newStatus.ConsecutiveSuccesses = 1
				}
				newStatus.ConsecutiveFailures = 0
			} else {
				if !previousStatus.Healthy {
					newStatus.ConsecutiveFailures = previousStatus.ConsecutiveFailures + 1
				} else {
					newStatus.ConsecutiveFailures = 1
				}
				newStatus.ConsecutiveSuccesses = 0
			}
		} else {
			if newStatus.Healthy {
				newStatus.ConsecutiveSuccesses = 1
				newStatus.ConsecutiveFailures = 0
			} else {
				newStatus.ConsecutiveSuccesses = 0
				newStatus.ConsecutiveFailures = 1
			}
		}

		// Store updated status
		hc.mu.Lock()
		hc.status[name] = newStatus
		hc.mu.Unlock()

		// Publish events for status changes
		if !hadPrevious || previousStatus.Healthy != newStatus.Healthy {
			eventType := HealthCheckPassed
			if !newStatus.Healthy {
				eventType = HealthCheckFailed
			} else if hadPrevious && !previousStatus.Healthy {
				eventType = HealthRecovered
			}

			event := HealthEvent{
				Target:    name,
				NewStatus: newStatus,
				Timestamp: time.Now(),
				Type:      eventType,
			}

			if hadPrevious {
				event.OldStatus = previousStatus
			}

			hc.publishEvent(event)
		}

		// Check for degraded status (multiple failures but not yet unhealthy threshold)
		if newStatus.ConsecutiveFailures > 0 &&
			newStatus.ConsecutiveFailures < hc.config.UnhealthyThreshold &&
			(!hadPrevious || previousStatus.ConsecutiveFailures == 0) {
			hc.publishEvent(HealthEvent{
				Target:    name,
				OldStatus: previousStatus,
				NewStatus: newStatus,
				Timestamp: time.Now(),
				Type:      HealthDegraded,
			})
		}
	}
}

// publishEvent sends a health event to all subscribers.
func (hc *healthChecker) publishEvent(event HealthEvent) {
	hc.subMu.RLock()
	defer hc.subMu.RUnlock()

	for _, subscriber := range hc.subscribers {
		select {
		case subscriber <- event:
		default:
			// Channel is full or closed, skip this subscriber
			if hc.logger != nil {
				hc.logger.Warn(context.Background(), "Failed to send health event to subscriber")
			}
		}
	}
}

// recordHealthCheck records health check metrics if metrics collector is available.
func (hc *healthChecker) recordHealthCheck(target string, success bool, duration time.Duration) {
	if hc.metrics != nil {
		hc.metrics.RecordHealthCheck(target, success, duration)
	}
}
