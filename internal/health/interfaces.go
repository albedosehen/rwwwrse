// Package health provides interfaces and types for health monitoring and status checking.
// It supports backend health monitoring, circuit breaker patterns, and health status aggregation.
package health

import (
	"context"
	"time"
)

// HealthChecker monitors service health and provides health status information.
// It performs periodic health checks and maintains health state for backends.
type HealthChecker interface {
	// Check performs a single health check for the specified target.
	// Returns the current health status including any error information.
	Check(ctx context.Context, target HealthTarget) HealthStatus

	// StartMonitoring begins periodic health monitoring for registered targets.
	// Health checks run according to the configured interval.
	StartMonitoring(ctx context.Context) error

	// StopMonitoring halts all health monitoring operations.
	// It waits for ongoing checks to complete within the context timeout.
	StopMonitoring() error

	// Subscribe registers a channel to receive health status updates.
	// The channel will receive HealthEvent notifications when status changes.
	Subscribe(ch chan<- HealthEvent) error

	// Unsubscribe removes a channel from health event notifications.
	Unsubscribe(ch chan<- HealthEvent) error
}

// HealthTarget represents a service or component that can be health checked.
type HealthTarget interface {
	// Name returns a unique identifier for this health target.
	Name() string

	// URL returns the health check endpoint URL.
	URL() string

	// Timeout returns the maximum time to wait for a health check response.
	Timeout() time.Duration

	// ExpectedStatus returns the HTTP status code that indicates good health.
	ExpectedStatus() int

	// Headers returns any custom headers to include in health check requests.
	Headers() map[string]string
}

// HealthStatus represents the current health state of a target.
type HealthStatus struct {
	// Healthy indicates whether the target is currently healthy.
	Healthy bool

	// LastCheck is the timestamp of the most recent health check.
	LastCheck time.Time

	// ResponseTime is the duration of the last health check request.
	ResponseTime time.Duration

	// ConsecutiveFailures is the number of consecutive failed health checks.
	ConsecutiveFailures int

	// ConsecutiveSuccesses is the number of consecutive successful health checks.
	ConsecutiveSuccesses int

	// Error contains details if the last health check failed.
	Error error

	// StatusCode is the HTTP status code from the last health check.
	StatusCode int

	// Metadata contains additional health check information.
	Metadata map[string]interface{}
}

// HealthEvent represents a health status change notification.
type HealthEvent struct {
	// Target is the health target that changed status.
	Target string

	// OldStatus is the previous health status.
	OldStatus HealthStatus

	// NewStatus is the current health status.
	NewStatus HealthStatus

	// Timestamp is when the status change occurred.
	Timestamp time.Time

	// Type describes the nature of the health event.
	Type HealthEventType
}

// HealthEventType categorizes different types of health events.
type HealthEventType int

const (
	HealthCheckStarted HealthEventType = iota
	HealthCheckPassed
	HealthCheckFailed
	HealthRecovered
	HealthDegraded
	HealthCheckStopped
)

// String returns a string representation of the health event type.
func (t HealthEventType) String() string {
	switch t {
	case HealthCheckStarted:
		return "started"
	case HealthCheckPassed:
		return "passed"
	case HealthCheckFailed:
		return "failed"
	case HealthRecovered:
		return "recovered"
	case HealthDegraded:
		return "degraded"
	case HealthCheckStopped:
		return "stopped"
	default:
		return "unknown"
	}
}

// CircuitBreaker implements the circuit breaker pattern for health management.
// It prevents requests to unhealthy services and allows for gradual recovery.
type CircuitBreaker interface {
	// Allow determines if a request should be allowed based on current circuit state.
	Allow(ctx context.Context, target string) bool

	// RecordSuccess records a successful operation for the circuit.
	RecordSuccess(ctx context.Context, target string)

	// RecordFailure records a failed operation for the circuit.
	RecordFailure(ctx context.Context, target string, err error)

	// State returns the current circuit breaker state for the target.
	State(ctx context.Context, target string) CircuitState

	// Reset manually resets the circuit breaker to closed state.
	Reset(ctx context.Context, target string) error
}

// CircuitState represents the current state of a circuit breaker.
type CircuitState int

const (
	CircuitClosed CircuitState = iota
	CircuitOpen
	CircuitHalfOpen
)

// String returns a string representation of the circuit state.
func (s CircuitState) String() string {
	switch s {
	case CircuitClosed:
		return "closed"
	case CircuitOpen:
		return "open"
	case CircuitHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// CircuitBreakerConfig configures circuit breaker behavior.
type CircuitBreakerConfig struct {
	// FailureThreshold is the number of failures needed to open the circuit.
	FailureThreshold int

	// SuccessThreshold is the number of successes needed to close the circuit.
	SuccessThreshold int

	// Timeout is how long the circuit stays open before allowing test requests.
	Timeout time.Duration

	// MaxRequests is the maximum number of requests allowed in half-open state.
	MaxRequests int
}

// HealthAggregator combines health status from multiple sources.
type HealthAggregator interface {
	// RegisterTarget adds a health target to the aggregator.
	RegisterTarget(target HealthTarget) error

	// UnregisterTarget removes a health target from the aggregator.
	UnregisterTarget(name string) error

	// OverallHealth returns the combined health status of all targets.
	OverallHealth(ctx context.Context) AggregatedHealth

	// TargetHealth returns the health status of a specific target.
	TargetHealth(ctx context.Context, name string) (HealthStatus, error)

	// AllTargets returns health status for all registered targets.
	AllTargets(ctx context.Context) map[string]HealthStatus
}

// AggregatedHealth represents the overall health of the system.
type AggregatedHealth struct {
	// Status indicates the overall system health.
	Status SystemHealthStatus

	// Healthy is the number of healthy targets.
	Healthy int

	// Unhealthy is the number of unhealthy targets.
	Unhealthy int

	// Total is the total number of targets being monitored.
	Total int

	// Details provides health information for each target.
	Details map[string]HealthStatus

	// Timestamp is when this health snapshot was taken.
	Timestamp time.Time
}

// SystemHealthStatus represents the overall system health state.
type SystemHealthStatus int

const (
	SystemHealthy SystemHealthStatus = iota
	SystemDegraded
	SystemUnhealthy
	SystemUnknown
)

// String returns a string representation of the system health status.
func (s SystemHealthStatus) String() string {
	switch s {
	case SystemHealthy:
		return "healthy"
	case SystemDegraded:
		return "degraded"
	case SystemUnhealthy:
		return "unhealthy"
	case SystemUnknown:
		return "unknown"
	default:
		return "unknown"
	}
}

// HealthReporter provides health status reporting capabilities.
type HealthReporter interface {
	// GetHealthReport generates a comprehensive health report.
	GetHealthReport(ctx context.Context) (*HealthReport, error)

	// GetReadinessReport generates a readiness report for startup checks.
	GetReadinessReport(ctx context.Context) (*ReadinessReport, error)

	// GetLivenessReport generates a liveness report for runtime checks.
	GetLivenessReport(ctx context.Context) (*LivenessReport, error)
}

// HealthReport contains comprehensive health information.
type HealthReport struct {
	Status    SystemHealthStatus      `json:"status"`
	Timestamp time.Time               `json:"timestamp"`
	Version   string                  `json:"version"`
	Uptime    time.Duration           `json:"uptime"`
	Targets   map[string]HealthStatus `json:"targets"`
	Summary   HealthSummary           `json:"summary"`
}

// ReadinessReport indicates if the service is ready to accept traffic.
type ReadinessReport struct {
	Ready     bool                   `json:"ready"`
	Timestamp time.Time              `json:"timestamp"`
	Checks    map[string]CheckResult `json:"checks"`
	Message   string                 `json:"message"`
}

// LivenessReport indicates if the service is alive and functioning.
type LivenessReport struct {
	Alive     bool                   `json:"alive"`
	Timestamp time.Time              `json:"timestamp"`
	Checks    map[string]CheckResult `json:"checks"`
	Message   string                 `json:"message"`
}

// CheckResult represents the result of an individual health check.
type CheckResult struct {
	Passed   bool          `json:"passed"`
	Duration time.Duration `json:"duration"`
	Error    string        `json:"error,omitempty"`
	Details  interface{}   `json:"details,omitempty"`
}

// HealthSummary provides aggregate health statistics.
type HealthSummary struct {
	TotalTargets     int     `json:"total_targets"`
	HealthyTargets   int     `json:"healthy_targets"`
	UnhealthyTargets int     `json:"unhealthy_targets"`
	SuccessRate      float64 `json:"success_rate"`
}
