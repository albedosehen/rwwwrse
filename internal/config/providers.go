package config

import (
	"context"
	"fmt"

	"github.com/albedosehen/rwwwrse/internal/observability"
)

// ProvideConfig loads and returns the main configuration.
// It first attempts to load secrets from Doppler if available,
// then proceeds with the regular configuration loading process.
func ProvideConfig() (*Config, error) {
	// Create a basic logger for configuration loading
	// We can't use the full observability.Logger yet since it depends on configuration
	logger := &basicLogger{}

	// Try to load secrets from Doppler first
	dopplerProvider := NewDopplerProvider(logger)
	if err := dopplerProvider.LoadSecrets(); err != nil {
		// Log but don't fail if Doppler fails
		logger.Warn(context.Background(), fmt.Sprintf("Failed to load Doppler secrets: %v", err))
	}

	// Now proceed with regular configuration loading
	loader := NewConfigLoader()
	return loader.Load()
}

// basicLogger is a simple implementation of the observability.Logger interface
// used during initial configuration loading.
type basicLogger struct{}

func (l *basicLogger) Debug(ctx context.Context, msg string, fields ...observability.Field) {
	fmt.Printf("[DEBUG] %s\n", msg)
}

func (l *basicLogger) Info(ctx context.Context, msg string, fields ...observability.Field) {
	fmt.Printf("[INFO] %s\n", msg)
}

func (l *basicLogger) Warn(ctx context.Context, msg string, fields ...observability.Field) {
	fmt.Printf("[WARN] %s\n", msg)
}

func (l *basicLogger) Error(ctx context.Context, err error, msg string, fields ...observability.Field) {
	fmt.Printf("[ERROR] %s: %v\n", msg, err)
}

func (l *basicLogger) WithFields(fields ...observability.Field) observability.Logger {
	return l
}

func (l *basicLogger) WithContext(ctx context.Context) observability.Logger {
	return l
}

// ProvideServerConfig extracts server configuration from environment variables.
func ProvideServerConfig() (ServerConfig, error) {
	cfg, err := ProvideConfig()
	if err != nil {
		return ServerConfig{}, err
	}
	return cfg.Server, nil
}

// ProvideBackendsConfig extracts backends configuration from environment variables.
func ProvideBackendsConfig() (BackendsConfig, error) {
	cfg, err := ProvideConfig()
	if err != nil {
		return BackendsConfig{}, err
	}
	return cfg.Backends, nil
}

// ProvideTLSConfig extracts TLS configuration from environment variables.
func ProvideTLSConfig() (TLSConfig, error) {
	cfg, err := ProvideConfig()
	if err != nil {
		return TLSConfig{}, err
	}
	return cfg.TLS, nil
}

// ProvideHealthConfig extracts health configuration from environment variables.
func ProvideHealthConfig() (HealthConfig, error) {
	cfg, err := ProvideConfig()
	if err != nil {
		return HealthConfig{}, err
	}
	return cfg.Health, nil
}

// ProvideLoggingConfig extracts logging configuration from environment variables.
func ProvideLoggingConfig() (observability.LoggingConfig, error) {
	cfg, err := ProvideConfig()
	if err != nil {
		return observability.LoggingConfig{}, err
	}

	// Convert config.LoggingConfig to observability.LoggingConfig
	return observability.LoggingConfig{
		Level:      observability.ParseLogLevel(cfg.Logging.Level),
		Format:     observability.ParseLogFormat(cfg.Logging.Format),
		Output:     cfg.Logging.Output,
		AddSource:  false, // Default value, can be made configurable
		TimeFormat: "",    // Default value, can be made configurable
	}, nil
}

// ProvideMetricsConfig extracts metrics configuration from environment variables.
func ProvideMetricsConfig() (observability.MetricsConfig, error) {
	cfg, err := ProvideConfig()
	if err != nil {
		return observability.MetricsConfig{}, err
	}

	// Convert config.MetricsConfig to observability.MetricsConfig
	return observability.MetricsConfig{
		Enabled:   cfg.Metrics.Enabled,
		Address:   "",         // Default empty, can be made configurable
		Path:      "/metrics", // Default path
		Namespace: "rwwwrse",  // Default namespace
		Subsystem: "proxy",    // Default subsystem
	}, nil
}

// Provider functions that extract configs from an existing Config instance
// These are used when the Config is already loaded and passed to Wire

// ProvideServerConfigFromConfig extracts server config from a main config.
func ProvideServerConfigFromConfig(cfg *Config) ServerConfig {
	return cfg.Server
}

// ProvideBackendsConfigFromConfig extracts backends config from a main config.
func ProvideBackendsConfigFromConfig(cfg *Config) BackendsConfig {
	return cfg.Backends
}

// ProvideTLSConfigFromConfig extracts TLS config from a main config.
func ProvideTLSConfigFromConfig(cfg *Config) TLSConfig {
	return cfg.TLS
}

// ProvideHealthConfigFromConfig extracts health config from a main config.
func ProvideHealthConfigFromConfig(cfg *Config) HealthConfig {
	return cfg.Health
}

// ProvideLoggingConfigFromConfig extracts logging config from a main config.
func ProvideLoggingConfigFromConfig(cfg *Config) observability.LoggingConfig {
	return observability.LoggingConfig{
		Level:      observability.ParseLogLevel(cfg.Logging.Level),
		Format:     observability.ParseLogFormat(cfg.Logging.Format),
		Output:     cfg.Logging.Output,
		AddSource:  false, // Default value, can be made configurable
		TimeFormat: "",    // Default value, can be made configurable
	}
}

// ProvideMetricsConfigFromConfig extracts metrics config from a main config.
func ProvideMetricsConfigFromConfig(cfg *Config) observability.MetricsConfig {
	return observability.MetricsConfig{
		Enabled:   cfg.Metrics.Enabled,
		Address:   "",         // Default empty, can be made configurable
		Path:      "/metrics", // Default path
		Namespace: "rwwwrse",  // Default namespace
		Subsystem: "proxy",    // Default subsystem
	}
}
