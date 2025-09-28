package proxy

import (
	"github.com/albedosehen/rwwwrse/internal/config"
	"github.com/albedosehen/rwwwrse/internal/observability"
)

// ProvideRouter creates a new router instance.
func ProvideRouter(
	backendsConfig config.BackendsConfig,
	logger observability.Logger,
	metrics observability.MetricsCollector,
) Router {
	router, err := NewRouterImpl(backendsConfig, logger, metrics)
	if err != nil {
		// For dependency injection, we'll panic on configuration errors
		// since this indicates a fatal startup issue
		panic("failed to create router: " + err.Error())
	}
	return router
}

// ProvideBackendManager creates a new backend manager instance.
func ProvideBackendManager(
	backendsConfig config.BackendsConfig,
	logger observability.Logger,
	metrics observability.MetricsCollector,
) BackendManager {
	manager, err := NewBackendManagerImpl(backendsConfig, logger, metrics)
	if err != nil {
		panic("failed to create backend manager: " + err.Error())
	}
	return manager
}

// ProvideConnectionPool creates a new connection pool instance.
func ProvideConnectionPool(
	backendsConfig config.BackendsConfig,
	logger observability.Logger,
	metrics observability.MetricsCollector,
) ConnectionPool {
	pool, err := NewConnectionPoolImpl(backendsConfig, logger, metrics)
	if err != nil {
		panic("failed to create connection pool: " + err.Error())
	}
	return pool
}

// ProvideProxyHandler creates a new proxy handler instance.
func ProvideProxyHandler(
	router Router,
	logger observability.Logger,
	metrics observability.MetricsCollector,
) ProxyHandler {
	handler, err := NewProxyHandlerImpl(router, logger, metrics)
	if err != nil {
		panic("failed to create proxy handler: " + err.Error())
	}
	return handler
}
