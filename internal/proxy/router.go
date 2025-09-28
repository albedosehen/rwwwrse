package proxy

import (
	"context"
	"strings"
	"sync"

	"github.com/albedosehen/rwwwrse/internal/config"
	proxyerrors "github.com/albedosehen/rwwwrse/internal/errors"
	"github.com/albedosehen/rwwwrse/internal/observability"
)

type router struct {
	backends map[string]Backend
	mu       sync.RWMutex
	logger   observability.Logger
	metrics  observability.MetricsCollector
}

func NewRouterImpl(
	backendsConfig config.BackendsConfig,
	logger observability.Logger,
	metrics observability.MetricsCollector,
) (Router, error) {
	if logger == nil {
		return nil, proxyerrors.NewConfigError(proxyerrors.ErrCodeConfigInvalid, "logger", nil)
	}

	r := &router{
		backends: make(map[string]Backend),
		logger:   logger,
		metrics:  metrics,
	}

	for host, route := range backendsConfig.Routes {
		backend, err := NewBackendImpl(host, route, logger, metrics)
		if err != nil {
			return nil, proxyerrors.WrapError(
				proxyerrors.ErrCodeConfigInvalid,
				"failed to initialize backend for host "+host,
				err,
			)
		}

		if err := r.Register(host, backend); err != nil {
			return nil, err
		}
	}

	logger.Info(context.Background(), "Router initialized",
		observability.Int("backend_count", len(r.backends)),
	)

	return r, nil
}

func (r *router) Route(ctx context.Context, host string) (Backend, error) {
	if host == "" {
		return nil, proxyerrors.NewRoutingError(
			proxyerrors.ErrCodeInvalidHost,
			host,
			nil,
		)
	}

	normalizedHost := r.normalizeHost(host)

	r.mu.RLock()
	backend, exists := r.backends[normalizedHost]
	r.mu.RUnlock()

	if !exists {
		r.logger.Warn(ctx, "Host not configured",
			observability.String("host", host),
			observability.String("normalized_host", normalizedHost),
		)
		return nil, proxyerrors.NewRoutingError(
			proxyerrors.ErrCodeHostNotConfigured,
			host,
			nil,
		)
	}

	// Check if backend is healthy before routing
	if !backend.IsHealthy(ctx) {
		r.logger.Warn(ctx, "Backend is unhealthy",
			observability.String("host", host),
			observability.String("backend", backend.Name()),
		)
		return nil, proxyerrors.NewBackendError(
			proxyerrors.ErrCodeBackendUnavailable,
			backend.Name(),
			nil,
		)
	}

	r.logger.Debug(ctx, "Routed request to backend",
		observability.String("host", host),
		observability.String("backend", backend.Name()),
		observability.String("backend_url", backend.URL().String()),
	)

	return backend, nil
}

// Register associates a host with a backend service.
func (r *router) Register(host string, backend Backend) error {
	if host == "" {
		return proxyerrors.NewConfigError(
			proxyerrors.ErrCodeConfigInvalid,
			"host",
			nil,
		)
	}

	if backend == nil {
		return proxyerrors.NewConfigError(
			proxyerrors.ErrCodeConfigInvalid,
			"backend",
			nil,
		)
	}

	normalizedHost := r.normalizeHost(host)

	r.mu.Lock()
	defer r.mu.Unlock()

	r.backends[normalizedHost] = backend

	r.logger.Info(context.Background(), "Registered backend for host",
		observability.String("host", host),
		observability.String("normalized_host", normalizedHost),
		observability.String("backend", backend.Name()),
		observability.String("backend_url", backend.URL().String()),
	)

	return nil
}

// Unregister removes a host from the routing table.
func (r *router) Unregister(host string) error {
	if host == "" {
		return proxyerrors.NewConfigError(
			proxyerrors.ErrCodeConfigInvalid,
			"host",
			nil,
		)
	}

	normalizedHost := r.normalizeHost(host)

	r.mu.Lock()
	defer r.mu.Unlock()

	backend, exists := r.backends[normalizedHost]
	if !exists {
		return proxyerrors.NewRoutingError(
			proxyerrors.ErrCodeHostNotConfigured,
			host,
			nil,
		)
	}

	delete(r.backends, normalizedHost)

	r.logger.Info(context.Background(), "Unregistered backend for host",
		observability.String("host", host),
		observability.String("normalized_host", normalizedHost),
		observability.String("backend", backend.Name()),
	)

	return nil
}

// Backends returns a copy of all currently registered backends.
func (r *router) Backends() map[string]Backend {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Create a copy to avoid race conditions
	backends := make(map[string]Backend, len(r.backends))
	for host, backend := range r.backends {
		backends[host] = backend
	}

	return backends
}

// normalizeHost removes the port from a host string if present.
// This allows routing based on hostname only.
func (r *router) normalizeHost(host string) string {
	if host == "" {
		return host
	}

	// Remove port if present (e.g., "example.com:8080" becomes "example.com")
	if colonIndex := strings.LastIndex(host, ":"); colonIndex != -1 {
		// Check if this is an IPv6 address
		if strings.Count(host, ":") > 1 && !strings.HasPrefix(host, "[") {
			// IPv6 without brackets, don't remove port
			return strings.ToLower(host)
		}
		// IPv4 or IPv6 with brackets, remove port
		return strings.ToLower(host[:colonIndex])
	}

	return strings.ToLower(host)
}
