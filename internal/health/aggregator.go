// Package health provides health aggregation functionality.
package health

import (
	"context"
	"fmt"
	"time"

	"github.com/albedosehen/rwwwrse/internal/config"
	"github.com/albedosehen/rwwwrse/internal/observability"
)

// simpleHealthAggregator implements HealthAggregator interface.
type simpleHealthAggregator struct {
	checker HealthChecker
	logger  observability.Logger
	metrics observability.MetricsCollector
}

// RegisterTarget adds a health target to the aggregator.
func (ha *simpleHealthAggregator) RegisterTarget(target HealthTarget) error {
	if hc, ok := ha.checker.(*healthChecker); ok {
		return hc.RegisterTarget(target)
	}
	return fmt.Errorf("health checker does not support target registration")
}

// UnregisterTarget removes a health target from the aggregator.
func (ha *simpleHealthAggregator) UnregisterTarget(name string) error {
	if hc, ok := ha.checker.(*healthChecker); ok {
		return hc.UnregisterTarget(name)
	}
	return fmt.Errorf("health checker does not support target unregistration")
}

// OverallHealth returns the combined health status of all targets.
func (ha *simpleHealthAggregator) OverallHealth(ctx context.Context) AggregatedHealth {
	var allStatus map[string]HealthStatus

	if hc, ok := ha.checker.(*healthChecker); ok {
		allStatus = hc.GetAllTargetStatus()
	} else {
		allStatus = make(map[string]HealthStatus)
	}

	healthy := 0
	unhealthy := 0
	total := len(allStatus)

	for _, status := range allStatus {
		if status.Healthy {
			healthy++
		} else {
			unhealthy++
		}
	}

	// Determine overall system health
	var systemStatus SystemHealthStatus
	if total == 0 {
		systemStatus = SystemUnknown
	} else if unhealthy == 0 {
		systemStatus = SystemHealthy
	} else if healthy > unhealthy {
		systemStatus = SystemDegraded
	} else {
		systemStatus = SystemUnhealthy
	}

	return AggregatedHealth{
		Status:    systemStatus,
		Healthy:   healthy,
		Unhealthy: unhealthy,
		Total:     total,
		Details:   allStatus,
		Timestamp: time.Now(),
	}
}

// TargetHealth returns the health status of a specific target.
func (ha *simpleHealthAggregator) TargetHealth(ctx context.Context, name string) (HealthStatus, error) {
	if hc, ok := ha.checker.(*healthChecker); ok {
		status, exists := hc.GetTargetStatus(name)
		if !exists {
			return HealthStatus{}, fmt.Errorf("target %s not found", name)
		}
		return status, nil
	}
	return HealthStatus{}, fmt.Errorf("health checker does not support target status lookup")
}

// AllTargets returns health status for all registered targets.
func (ha *simpleHealthAggregator) AllTargets(ctx context.Context) map[string]HealthStatus {
	if hc, ok := ha.checker.(*healthChecker); ok {
		return hc.GetAllTargetStatus()
	}
	return make(map[string]HealthStatus)
}

// simpleHealthReporter implements HealthReporter interface.
type simpleHealthReporter struct {
	aggregator HealthAggregator
	config     config.Config
	logger     observability.Logger
	startTime  time.Time
}

// GetHealthReport generates a comprehensive health report.
func (hr *simpleHealthReporter) GetHealthReport(ctx context.Context) (*HealthReport, error) {
	aggregated := hr.aggregator.OverallHealth(ctx)

	summary := HealthSummary{
		TotalTargets:     aggregated.Total,
		HealthyTargets:   aggregated.Healthy,
		UnhealthyTargets: aggregated.Unhealthy,
	}

	return &HealthReport{
		Status:    aggregated.Status,
		Timestamp: time.Now(),
		Version:   "1.0.0", // Could be made configurable
		Uptime:    time.Since(hr.startTime),
		Targets:   aggregated.Details,
		Summary:   summary,
	}, nil
}

// GetReadinessReport generates a readiness report for startup checks.
func (hr *simpleHealthReporter) GetReadinessReport(ctx context.Context) (*ReadinessReport, error) {
	aggregated := hr.aggregator.OverallHealth(ctx)

	checks := make(map[string]CheckResult)

	// Check if all critical services are healthy
	ready := aggregated.Status == SystemHealthy || aggregated.Status == SystemDegraded
	message := "Service is ready"

	if !ready {
		message = "Service is not ready - unhealthy backends detected"
	}

	// Add individual checks
	for name, status := range aggregated.Details {
		checks[name] = CheckResult{
			Passed:   status.Healthy,
			Duration: status.ResponseTime,
			Error:    getErrorString(status.Error),
		}
	}

	return &ReadinessReport{
		Ready:     ready,
		Timestamp: time.Now(),
		Checks:    checks,
		Message:   message,
	}, nil
}

// GetLivenessReport generates a liveness report for runtime checks.
func (hr *simpleHealthReporter) GetLivenessReport(ctx context.Context) (*LivenessReport, error) {
	// For liveness, we mainly check if the service itself is alive
	// This is simpler than readiness - just check basic functionality

	checks := make(map[string]CheckResult)
	alive := true
	message := "Service is alive"

	// Basic liveness check - ensure we can get health status
	start := time.Now()
	_ = hr.aggregator.OverallHealth(ctx)
	duration := time.Since(start)

	checks["health_aggregator"] = CheckResult{
		Passed:   true,
		Duration: duration,
	}

	return &LivenessReport{
		Alive:     alive,
		Timestamp: time.Now(),
		Checks:    checks,
		Message:   message,
	}, nil
}

// getErrorString safely converts an error to string.
func getErrorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
