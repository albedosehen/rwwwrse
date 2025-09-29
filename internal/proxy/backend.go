package proxy

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"sync/atomic"
	"time"

	"github.com/albedosehen/rwwwrse/internal/config"
	proxyerrors "github.com/albedosehen/rwwwrse/internal/errors"
	"github.com/albedosehen/rwwwrse/internal/observability"
)

type backend struct {
	name      string
	url       *url.URL
	transport http.RoundTripper
	healthy   int64 // atomic boolean: 1 = healthy, 0 = unhealthy
	config    config.BackendRoute
	logger    observability.Logger
	metrics   observability.MetricsCollector
}

func NewBackendImpl(
	name string,
	route config.BackendRoute,
	logger observability.Logger,
	metrics observability.MetricsCollector,
) (Backend, error) {
	if name == "" {
		return nil, proxyerrors.NewConfigError(
			proxyerrors.ErrCodeConfigInvalid,
			"name",
			nil,
		)
	}

	if route.URL == "" {
		return nil, proxyerrors.NewConfigError(
			proxyerrors.ErrCodeConfigInvalid,
			"backend_url",
			nil,
		)
	}

	backendURL, err := url.Parse(route.URL)
	if err != nil {
		return nil, proxyerrors.NewConfigError(
			proxyerrors.ErrCodeConfigInvalid,
			"backend_url",
			err,
		)
	}
	
	// Validate that URL has scheme and host
	if backendURL.Scheme == "" || backendURL.Host == "" {
		return nil, proxyerrors.NewConfigError(
			proxyerrors.ErrCodeConfigInvalid,
			"backend_url",
			nil,
		)
	}

	if logger == nil {
		return nil, proxyerrors.NewConfigError(
			proxyerrors.ErrCodeConfigInvalid,
			"logger",
			nil,
		)
	}

	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   route.DialTimeout,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          route.MaxIdleConns,
		MaxIdleConnsPerHost:   route.MaxIdlePerHost,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: route.Timeout,
		ExpectContinueTimeout: 1 * time.Second,
		ForceAttemptHTTP2:     true,
	}

	b := &backend{
		name:      name,
		url:       backendURL,
		transport: transport,
		healthy:   1, // Start as healthy
		config:    route,
		logger:    logger,
		metrics:   metrics,
	}

	logger.Info(context.Background(), "Backend initialized",
		observability.String("name", name),
		observability.String("url", backendURL.String()),
		observability.Duration("timeout", route.Timeout),
		observability.Int("max_idle_conns", route.MaxIdleConns),
	)

	return b, nil
}

// URL returns the backend service URL.
func (b *backend) URL() *url.URL {
	return b.url
}

// Transport returns the HTTP transport for this backend.
func (b *backend) Transport() http.RoundTripper {
	return b.transport
}

func (b *backend) IsHealthy(ctx context.Context) bool {
	return atomic.LoadInt64(&b.healthy) == 1
}

func (b *backend) Name() string {
	return b.name
}

func (b *backend) SetHealthy(healthy bool) {
	var value int64
	if healthy {
		value = 1
	}

	oldValue := atomic.SwapInt64(&b.healthy, value)
	oldHealthy := oldValue == 1

	// Log health status changes
	if oldHealthy != healthy {
		if healthy {
			b.logger.Info(context.Background(), "Backend became healthy",
				observability.String("backend", b.name),
				observability.String("url", b.url.String()),
			)
		} else {
			b.logger.Warn(context.Background(), "Backend became unhealthy",
				observability.String("backend", b.name),
				observability.String("url", b.url.String()),
			)
		}

		// Record metrics for health status change
		if b.metrics != nil {
			b.metrics.RecordHealthCheck(b.name, healthy, 0)
		}
	}
}

func (b *backend) GetConfig() config.BackendRoute {
	return b.config
}

func (b *backend) Close() error {
	if transport, ok := b.transport.(*http.Transport); ok {
		transport.CloseIdleConnections()
	}

	b.logger.Info(context.Background(), "Backend closed",
		observability.String("backend", b.name),
		observability.String("url", b.url.String()),
	)

	return nil
}

type backendManager struct {
	backends map[string]Backend
	logger   observability.Logger
	metrics  observability.MetricsCollector
}

func NewBackendManagerImpl(
	backendsConfig config.BackendsConfig,
	logger observability.Logger,
	metrics observability.MetricsCollector,
) (BackendManager, error) {
	if logger == nil {
		return nil, proxyerrors.NewConfigError(
			proxyerrors.ErrCodeConfigInvalid,
			"logger",
			nil,
		)
	}

	bm := &backendManager{
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

		bm.backends[host] = backend
	}

	logger.Info(context.Background(), "Backend manager initialized",
		observability.Int("backend_count", len(bm.backends)),
	)

	return bm, nil
}

func (bm *backendManager) AddBackend(ctx context.Context, host string, backend Backend) error {
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

	bm.backends[host] = backend

	bm.logger.Info(ctx, "Backend added",
		observability.String("host", host),
		observability.String("backend", backend.Name()),
		observability.String("url", backend.URL().String()),
	)

	return nil
}

func (bm *backendManager) RemoveBackend(ctx context.Context, host string) error {
	if host == "" {
		return proxyerrors.NewConfigError(
			proxyerrors.ErrCodeConfigInvalid,
			"host",
			nil,
		)
	}

	backend, exists := bm.backends[host]
	if !exists {
		return proxyerrors.NewRoutingError(
			proxyerrors.ErrCodeHostNotConfigured,
			host,
			nil,
		)
	}

	// Close backend connections
	if closer, ok := backend.(interface{ Close() error }); ok {
		if err := closer.Close(); err != nil {
			bm.logger.Warn(ctx, "Error closing backend",
				observability.String("host", host),
				observability.String("backend", backend.Name()),
				observability.Error(err),
			)
		}
	}

	delete(bm.backends, host)

	bm.logger.Info(ctx, "Backend removed",
		observability.String("host", host),
		observability.String("backend", backend.Name()),
	)

	return nil
}

func (bm *backendManager) UpdateBackend(ctx context.Context, host string, backend Backend) error {
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

	// Remove old backend if it exists
	if oldBackend, exists := bm.backends[host]; exists {
		if closer, ok := oldBackend.(interface{ Close() error }); ok {
			if err := closer.Close(); err != nil {
				bm.logger.Warn(ctx, "Error closing old backend",
					observability.String("host", host),
					observability.String("backend", oldBackend.Name()),
					observability.Error(err),
				)
			}
		}
	}

	bm.backends[host] = backend

	bm.logger.Info(ctx, "Backend updated",
		observability.String("host", host),
		observability.String("backend", backend.Name()),
		observability.String("url", backend.URL().String()),
	)

	return nil
}

func (bm *backendManager) GetBackend(ctx context.Context, host string) (Backend, error) {
	if host == "" {
		return nil, proxyerrors.NewConfigError(
			proxyerrors.ErrCodeConfigInvalid,
			"host",
			nil,
		)
	}

	backend, exists := bm.backends[host]
	if !exists {
		return nil, proxyerrors.NewRoutingError(
			proxyerrors.ErrCodeHostNotConfigured,
			host,
			nil,
		)
	}

	return backend, nil
}

func (bm *backendManager) ListBackends(ctx context.Context) (map[string]Backend, error) {
	// Create a copy to avoid race conditions
	backends := make(map[string]Backend, len(bm.backends))
	for host, backend := range bm.backends {
		backends[host] = backend
	}

	return backends, nil
}
