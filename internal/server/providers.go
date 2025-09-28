package server

import (
	"crypto/tls"
	"fmt"
	"net/http"

	"github.com/google/wire"

	"github.com/albedosehen/rwwwrse/internal/config"
	"github.com/albedosehen/rwwwrse/internal/observability"
	tlsManager "github.com/albedosehen/rwwwrse/internal/tls"
)

var ProviderSet = wire.NewSet(
	NewServerManager,
	NewHTTPServer,
	NewHTTPSServer,
	NewServerMetrics,
	NewBasicHealthChecker,
	NewServerBuilder,
	ProvideServerConfig,
)

func ProvideServerConfig(cfg config.Config) ServerConfig {
	return ServerConfig{
		Host:              cfg.Server.Host,
		Port:              cfg.Server.Port,
		TLSEnabled:        cfg.TLS.Enabled,
		TLSPort:           cfg.Server.HTTPSPort,
		ReadTimeout:       cfg.Server.ReadTimeout.String(),
		WriteTimeout:      cfg.Server.WriteTimeout.String(),
		IdleTimeout:       cfg.Server.IdleTimeout.String(),
		ShutdownTimeout:   cfg.Server.GracefulTimeout.String(),
		HTTP2Enabled:      true,    // Enable HTTP/2 by default
		MaxHeaderBytes:    1 << 20, // 1MB default
		KeepAlivesEnabled: true,
	}
}

type serverBuilder struct {
	config      ServerConfig
	handler     http.Handler
	middleware  []MiddlewareFunc
	tlsConfig   *tls.Config
	certManager tlsManager.Manager
	metrics     ServerMetrics
	healthCheck HealthChecker
	logger      observability.Logger
}

func NewServerBuilder(config ServerConfig, logger observability.Logger) ServerBuilder {
	return &serverBuilder{
		config: config,
		logger: logger,
	}
}

func (sb *serverBuilder) WithHandler(handler http.Handler) ServerBuilder {
	sb.handler = handler
	return sb
}

func (sb *serverBuilder) WithMiddleware(middleware ...MiddlewareFunc) ServerBuilder {
	sb.middleware = append(sb.middleware, middleware...)
	return sb
}

func (sb *serverBuilder) WithTLS(certManager tlsManager.Manager) ServerBuilder {
	if certManager == nil {
		return sb
	}

	sb.certManager = certManager
	sb.tlsConfig = certManager.GetTLSConfig()
	return sb
}

func (sb *serverBuilder) WithMetrics(metrics ServerMetrics) ServerBuilder {
	sb.metrics = metrics
	return sb
}

func (sb *serverBuilder) WithHealthChecker(checker HealthChecker) ServerBuilder {
	sb.healthCheck = checker
	return sb
}

func (sb *serverBuilder) WithConfig(config ServerConfig) ServerBuilder {
	sb.config = config
	return sb
}

func (sb *serverBuilder) Build() (Server, error) {
	if sb.handler == nil {
		return nil, fmt.Errorf("handler is required")
	}

	finalHandler := sb.handler
	for i := len(sb.middleware) - 1; i >= 0; i-- {
		finalHandler = sb.middleware[i](finalHandler)
	}

	if sb.config.TLSEnabled {
		if sb.certManager == nil {
			return nil, fmt.Errorf("TLS is enabled but no certificate manager provided")
		}

		if sb.tlsConfig == nil {
			return nil, fmt.Errorf("TLS is enabled but no TLS configuration available from certificate manager")
		}

		return NewHTTPSServer(sb.config, finalHandler, sb.tlsConfig, sb.logger), nil
	}

	return NewHTTPServer(sb.config, finalHandler, sb.logger), nil
}

type ServerSystem struct {
	Manager     ServerManager
	HTTPServer  HTTPServer
	HTTPSServer HTTPSServer
	Metrics     ServerMetrics
	Health      HealthChecker
}

func NewServerSystem(
	manager ServerManager,
	httpServer HTTPServer,
	httpsServer HTTPSServer,
	metrics ServerMetrics,
	health HealthChecker,
) *ServerSystem {
	return &ServerSystem{
		Manager:     manager,
		HTTPServer:  httpServer,
		HTTPSServer: httpsServer,
		Metrics:     metrics,
		Health:      health,
	}
}

func (ss *ServerSystem) Start() error {
	if ss.HTTPServer != nil {
		if err := ss.Manager.AddServer("http", ss.HTTPServer); err != nil {
			return fmt.Errorf("failed to add HTTP server: %w", err)
		}
	}

	if ss.HTTPSServer != nil {
		if err := ss.Manager.AddServer("https", ss.HTTPSServer); err != nil {
			return fmt.Errorf("failed to add HTTPS server: %w", err)
		}
	}

	return nil
}

func (ss *ServerSystem) Stop() error {
	return nil // Manager handles stopping servers
}

type requestContext struct {
	requestID string
	clientIP  string
	userAgent string
	startTime string
	values    map[string]interface{}
}

func NewRequestContext(requestID, clientIP, userAgent, startTime string) RequestContext {
	return &requestContext{
		requestID: requestID,
		clientIP:  clientIP,
		userAgent: userAgent,
		startTime: startTime,
		values:    make(map[string]interface{}),
	}
}

func (rc *requestContext) RequestID() string {
	return rc.requestID
}

func (rc *requestContext) ClientIP() string {
	return rc.clientIP
}

func (rc *requestContext) UserAgent() string {
	return rc.userAgent
}

func (rc *requestContext) StartTime() string {
	return rc.startTime
}

func (rc *requestContext) SetValue(key string, value interface{}) {
	rc.values[key] = value
}

func (rc *requestContext) GetValue(key string) (interface{}, bool) {
	value, exists := rc.values[key]
	return value, exists
}

func ConfigFromMain(mainConfig config.Config) ServerConfig {
	return ServerConfig{
		Host:              mainConfig.Server.Host,
		Port:              mainConfig.Server.Port,
		TLSEnabled:        mainConfig.TLS.Enabled,
		TLSPort:           mainConfig.Server.HTTPSPort,
		ReadTimeout:       mainConfig.Server.ReadTimeout.String(),
		WriteTimeout:      mainConfig.Server.WriteTimeout.String(),
		IdleTimeout:       mainConfig.Server.IdleTimeout.String(),
		ShutdownTimeout:   mainConfig.Server.GracefulTimeout.String(),
		HTTP2Enabled:      true,
		MaxHeaderBytes:    1 << 20, // 1MB
		KeepAlivesEnabled: true,
	}
}
