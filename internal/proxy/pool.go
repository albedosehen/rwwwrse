package proxy

import (
	"context"
	"net/http"
	"sync"

	"github.com/albedosehen/rwwwrse/internal/config"
	proxyerrors "github.com/albedosehen/rwwwrse/internal/errors"
	"github.com/albedosehen/rwwwrse/internal/observability"
)

type connectionPool struct {
	// pools stores HTTP round trippers keyed by backend name for efficient connection reuse
	pools map[string]http.RoundTripper

	// mu protects concurrent access to the pools map using read-write semantics
	// for optimal performance with frequent reads and infrequent writes
	mu sync.RWMutex

	// logger provides structured logging with context awareness for debugging and monitoring
	logger observability.Logger

	// metrics collects and exports connection pool performance metrics
	metrics observability.MetricsCollector

	// stats tracks connection pool statistics for monitoring and debugging
	stats struct {
		mu                 sync.RWMutex
		connectionsCreated int64
		connectionsReused  int64
		connectionsActive  int64
	}
}

func NewConnectionPoolImpl(
	backendsConfig config.BackendsConfig,
	logger observability.Logger,
	metrics observability.MetricsCollector,
) (ConnectionPool, error) {
	if logger == nil {
		return nil, proxyerrors.NewConfigError(
			proxyerrors.ErrCodeConfigInvalid,
			"logger",
			nil,
		)
	}

	if metrics == nil {
		return nil, proxyerrors.NewConfigError(
			proxyerrors.ErrCodeConfigInvalid,
			"metrics",
			nil,
		)
	}

	cp := &connectionPool{
		pools:   make(map[string]http.RoundTripper),
		logger:  logger,
		metrics: metrics,
	}

	logger.Info(context.Background(), "Connection pool initialized",
		observability.Component("connection_pool"),
	)

	return cp, nil
}

func (cp *connectionPool) GetConnection(ctx context.Context, backend Backend) (http.RoundTripper, error) {
	if backend == nil {
		cp.logger.Error(ctx, nil, "GetConnection called with nil backend",
			observability.Component("connection_pool"),
		)
		return nil, proxyerrors.NewConfigError(
			proxyerrors.ErrCodeConfigInvalid,
			"backend",
			nil,
		)
	}

	backendName := backend.Name()

	// Fast path: check if connection exists (read lock)
	cp.mu.RLock()
	transport, exists := cp.pools[backendName]
	cp.mu.RUnlock()

	if exists {
		// Track connection reuse
		cp.incrementConnectionsReused()
		cp.metrics.IncActiveConnections()

		cp.logger.Debug(ctx, "Reusing existing connection",
			observability.Backend(backendName),
			observability.String("url", backend.URL().String()),
			observability.Component("connection_pool"),
		)

		return transport, nil
	}

	// Slow path: create new connection (write lock)
	cp.mu.Lock()
	// Double-check pattern to avoid race condition
	if transport, exists := cp.pools[backendName]; exists {
		cp.mu.Unlock()
		cp.incrementConnectionsReused()
		cp.metrics.IncActiveConnections()
		return transport, nil
	}

	// Create new transport using backend's configuration
	transport = backend.Transport()
	cp.pools[backendName] = transport
	cp.mu.Unlock()

	// Track new connection creation
	cp.incrementConnectionsCreated()
	cp.incrementActiveConnections()
	cp.metrics.IncActiveConnections()

	cp.logger.Debug(ctx, "Created new connection for backend",
		observability.Backend(backendName),
		observability.String("url", backend.URL().String()),
		observability.Component("connection_pool"),
	)

	return transport, nil
}

func (cp *connectionPool) ReleaseConnection(backend Backend, transport http.RoundTripper) error {
	if backend == nil {
		return proxyerrors.NewConfigError(
			proxyerrors.ErrCodeConfigInvalid,
			"backend",
			nil,
		)
	}

	// Decrement active connections when releasing
	cp.decrementActiveConnections()

	// For HTTP transports, we don't need to do anything special
	// The connection will be returned to the pool automatically
	return nil
}

func (cp *connectionPool) Close() error {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	for name, transport := range cp.pools {
		if httpTransport, ok := transport.(*http.Transport); ok {
			httpTransport.CloseIdleConnections()
		}
		cp.logger.Debug(context.Background(), "Closed connections for backend",
			observability.String("backend", name),
		)
	}

	cp.pools = make(map[string]http.RoundTripper)
	cp.logger.Info(context.Background(), "Connection pool closed")

	return nil
}

// Stats returns comprehensive connection pool statistics.
func (cp *connectionPool) Stats() ConnectionPoolStats {
	cp.mu.RLock()
	totalConnections := len(cp.pools)
	cp.mu.RUnlock()

	cp.stats.mu.RLock()
	connectionsActive := cp.stats.connectionsActive
	cp.stats.mu.RUnlock()

	return ConnectionPoolStats{
		TotalConnections:  totalConnections,
		ActiveConnections: int(connectionsActive),
		IdleConnections:   totalConnections - int(connectionsActive),
		ConnectionsInUse:  int(connectionsActive),
	}
}

func (cp *connectionPool) incrementConnectionsCreated() {
	cp.stats.mu.Lock()
	cp.stats.connectionsCreated++
	cp.stats.mu.Unlock()
}

func (cp *connectionPool) incrementConnectionsReused() {
	cp.stats.mu.Lock()
	cp.stats.connectionsReused++
	cp.stats.mu.Unlock()
}

func (cp *connectionPool) incrementActiveConnections() {
	cp.stats.mu.Lock()
	cp.stats.connectionsActive++
	cp.stats.mu.Unlock()
}

func (cp *connectionPool) decrementActiveConnections() {
	cp.stats.mu.Lock()
	if cp.stats.connectionsActive > 0 {
		cp.stats.connectionsActive--
	}
	cp.stats.mu.Unlock()
}
