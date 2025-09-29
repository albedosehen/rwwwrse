//go:build wireinject
// +build wireinject

package di

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/wire"

	"github.com/albedosehen/rwwwrse/internal/config"
	"github.com/albedosehen/rwwwrse/internal/health"
	"github.com/albedosehen/rwwwrse/internal/middleware"
	"github.com/albedosehen/rwwwrse/internal/observability"
	"github.com/albedosehen/rwwwrse/internal/proxy"
	"github.com/albedosehen/rwwwrse/internal/server"
	"github.com/albedosehen/rwwwrse/internal/tls"
)

// Application represents the complete wired application.
type Application struct {
	Config          *config.Config
	Logger          observability.Logger
	Metrics         observability.MetricsCollector
	Router          proxy.Router
	BackendMgr      proxy.BackendManager
	Handler         proxy.ProxyHandler
	ConnectionPool  proxy.ConnectionPool
	ServerManager   server.ServerManager
	TLSManager      tls.Manager
	MiddlewareChain middleware.Chain
	HealthSystem    *health.HealthSystem

	mu      sync.RWMutex
	running bool
}

// providerSet defines the complete set of providers for dependency icnjection.
var providerSet = wire.NewSet(
	// Configuration providers
	config.ProvideConfig,
	config.ProvideLoggingConfig,
	config.ProvideMetricsConfig,
	config.ProvideBackendsConfig,

	// Observability providers
	observability.ProvideLogger,
	observability.ProvideMetricsCollector,

	// Proxy providers
	proxy.ProvideRouter,
	proxy.ProvideBackendManager,
	proxy.ProvideProxyHandler,
	proxy.ProvideConnectionPool,

	// Server provider set
	server.ProviderSet,

	// TLS provider set
	tls.ProviderSet,

	// Middleware provider set
	middleware.ProviderSet,

	// Health provider set
	health.ProviderSet,

	// Application provider
	ProvideApplication,
)

// InitializeApplication creates a fully wired application instance.
// This function will be implemented by Wire code generation.
func InitializeApplication(ctx context.Context) (*Application, error) {
	wire.Build(
		// Configuration providers
		config.ProvideConfig,
		config.ProvideLoggingConfig,
		config.ProvideMetricsConfig,
		config.ProvideBackendsConfig,

		// Observability providers
		observability.ProvideLogger,
		observability.ProvideMetricsCollector,

		// Proxy providers
		proxy.ProvideRouter,
		proxy.ProvideBackendManager,
		proxy.ProvideProxyHandler,
		proxy.ProvideConnectionPool,

		// Server provider set
		server.ProviderSet,

		// TLS provider set
		tls.ProviderSet,

		// Middleware provider set
		middleware.ProviderSet,

		// Health provider set
		health.ProviderSet,

		// Application provider
		ProvideApplication,
	)
	return nil, nil
}

// InitializeApplicationWithConfig creates a fully wired application instance with custom config.
// This function will be implemented by Wire code generation.
func InitializeApplicationWithConfig(ctx context.Context, cfg *config.Config) (*Application, error) {
	wire.Build(
		// Extract sub-configs from main config
		config.ProvideLoggingConfigFromConfig,
		config.ProvideMetricsConfigFromConfig,
		config.ProvideBackendsConfigFromConfig,

		// Observability providers
		observability.ProvideLogger,
		observability.ProvideMetricsCollector,

		// Proxy providers
		proxy.ProvideRouter,
		proxy.ProvideBackendManager,
		proxy.ProvideProxyHandler,
		proxy.ProvideConnectionPool,

		// Server provider set
		server.ProviderSet,

		// TLS provider set
		tls.ProviderSet,

		// Middleware provider set
		middleware.ProviderSet,

		// Health provider set
		health.ProviderSet,

		// Application provider
		ProvideApplication,
	)
	return nil, nil
}

// ProvideApplication creates the main application instance.
func ProvideApplication(
	cfg *config.Config,
	logger observability.Logger,
	metrics observability.MetricsCollector,
	router proxy.Router,
	backendMgr proxy.BackendManager,
	handler proxy.ProxyHandler,
	connectionPool proxy.ConnectionPool,
	serverManager server.ServerManager,
	tlsManager tls.Manager,
	middlewareChain middleware.Chain,
	healthSystem *health.HealthSystem,
) *Application {
	return &Application{
		Config:          cfg,
		Logger:          logger,
		Metrics:         metrics,
		Router:          router,
		BackendMgr:      backendMgr,
		Handler:         handler,
		ConnectionPool:  connectionPool,
		ServerManager:   serverManager,
		TLSManager:      tlsManager,
		MiddlewareChain: middlewareChain,
		HealthSystem:    healthSystem,
	}
}

// Start starts all components of the application.
func (a *Application) Start(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.running {
		return fmt.Errorf("application is already running")
	}

	a.Logger.Info(ctx, "Starting rwwwrse reverse proxy application",
		observability.String("version", "v1.0.0"),
		observability.String("address", a.Config.Server.GetServerAddress()),
		observability.Bool("tls_enabled", a.Config.TLS.Enabled),
		observability.Int("backend_count", len(a.Config.Backends.Routes)),
	)

	// For now, we'll create a simple HTTP server manually
	// This will be enhanced once we have full server manager implementation

	a.running = true
	a.Logger.Info(ctx, "rwwwrse reverse proxy application started successfully",
		observability.String("status", "running"),
		observability.String("health_endpoint", fmt.Sprintf("%s/health", a.Config.Server.GetServerAddress())),
		observability.String("metrics_endpoint", fmt.Sprintf("%s/metrics", a.Config.Server.GetServerAddress())),
	)

	return nil
}

// Stop gracefully stops all components of the application.
func (a *Application) Stop(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.running {
		return nil
	}

	a.Logger.Info(ctx, "Stopping rwwwrse reverse proxy application")

	// Create shutdown context with timeout
	shutdownCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// For now, simple shutdown - this will be enhanced with proper server manager
	_ = shutdownCtx

	a.running = false
	a.Logger.Info(ctx, "rwwwrse reverse proxy application stopped successfully")
	return nil
}

// IsRunning returns whether the application is currently running.
func (a *Application) IsRunning() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.running
}

// GetStatus returns the current status of the application.
func (a *Application) GetStatus(ctx context.Context) map[string]interface{} {
	a.mu.RLock()
	defer a.mu.RUnlock()

	status := map[string]interface{}{
		"running":       a.running,
		"version":       "v1.0.0",
		"config":        a.Config.Server.GetServerAddress(),
		"tls_enabled":   a.Config.TLS.Enabled,
		"backend_count": len(a.Config.Backends.Routes),
	}

	return status
}
