// Package health provides health check providers for dependency injection.
package health

import (
	"time"

	"github.com/google/wire"

	"github.com/albedosehen/rwwwrse/internal/config"
	"github.com/albedosehen/rwwwrse/internal/observability"
)

// ProviderSet is the Wire provider set for health components.
var ProviderSet = wire.NewSet(
	NewHealthChecker,
	NewCircuitBreaker,
	NewBackendTarget,
	NewHealthAggregator,
	NewHealthReporter,
	NewHealthTargetFactory,
	NewHealthSystem,
	ProvideHealthConfig,
	ProvideCircuitBreakerConfig,
)

// ProvideHealthConfig extracts health configuration from the main config.
func ProvideHealthConfig(cfg config.Config) config.HealthConfig {
	return cfg.Health
}

// ProvideCircuitBreakerConfig creates circuit breaker configuration from health config.
func ProvideCircuitBreakerConfig(healthCfg config.HealthConfig) CircuitBreakerConfig {
	return CircuitBreakerConfig{
		FailureThreshold: healthCfg.UnhealthyThreshold,
		SuccessThreshold: healthCfg.HealthyThreshold,
		Timeout:          60 * time.Second, // Default recovery timeout
	}
}

// NewHealthAggregator creates a new health aggregator instance.
func NewHealthAggregator(checker HealthChecker, logger observability.Logger, metrics observability.MetricsCollector) HealthAggregator {
	return &simpleHealthAggregator{
		checker: checker,
		logger:  logger,
		metrics: metrics,
	}
}

// NewHealthReporter creates a new health reporter instance.
func NewHealthReporter(aggregator HealthAggregator, logger observability.Logger, cfg config.Config) HealthReporter {
	return &simpleHealthReporter{
		aggregator: aggregator,
		logger:     logger,
		config:     cfg,
		startTime:  time.Now(),
	}
}

// HealthTargetFactory creates health targets for backends.
type HealthTargetFactory struct {
	checker HealthChecker
	breaker CircuitBreaker
	logger  observability.Logger
}

// NewHealthTargetFactory creates a new health target factory.
func NewHealthTargetFactory(checker HealthChecker, breaker CircuitBreaker, logger observability.Logger) *HealthTargetFactory {
	return &HealthTargetFactory{
		checker: checker,
		breaker: breaker,
		logger:  logger,
	}
}

// CreateTarget creates a new health target for the given backend route.
func (f *HealthTargetFactory) CreateTarget(name string, route config.BackendRoute) (HealthTarget, error) {
	return NewBackendTarget(name, route)
}

// HealthSystem represents the complete health monitoring system.
type HealthSystem struct {
	Checker    HealthChecker
	Aggregator HealthAggregator
	Reporter   HealthReporter
	Factory    *HealthTargetFactory
}

// NewHealthSystem creates a complete health monitoring system.
func NewHealthSystem(
	checker HealthChecker,
	aggregator HealthAggregator,
	reporter HealthReporter,
	factory *HealthTargetFactory,
) *HealthSystem {
	return &HealthSystem{
		Checker:    checker,
		Aggregator: aggregator,
		Reporter:   reporter,
		Factory:    factory,
	}
}

// Start initializes and starts the health monitoring system.
func (hs *HealthSystem) Start() error {
	// Health system is stateless and doesn't need explicit startup
	return nil
}

// Stop gracefully shuts down the health monitoring system.
func (hs *HealthSystem) Stop() error {
	// Health system is stateless and doesn't need explicit shutdown
	return nil
}

// HealthMonitorConfig represents comprehensive health monitoring configuration.
type HealthMonitorConfig struct {
	Enabled            bool
	Timeout            time.Duration
	Interval           time.Duration
	UnhealthyThreshold int
	HealthyThreshold   int
	CircuitBreaker     CircuitBreakerConfig
}

// NewHealthMonitorConfig creates health monitor configuration from the main config.
func NewHealthMonitorConfig(cfg config.Config) HealthMonitorConfig {
	return HealthMonitorConfig{
		Enabled:            cfg.Health.Enabled,
		Timeout:            cfg.Health.Timeout,
		Interval:           cfg.Health.Interval,
		UnhealthyThreshold: cfg.Health.UnhealthyThreshold,
		HealthyThreshold:   cfg.Health.HealthyThreshold,
		CircuitBreaker: CircuitBreakerConfig{
			FailureThreshold: cfg.Health.UnhealthyThreshold,
			SuccessThreshold: cfg.Health.HealthyThreshold,
			Timeout:          60 * time.Second,
		},
	}
}
