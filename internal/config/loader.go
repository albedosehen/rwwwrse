package config

import (
	"context"
	"fmt"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
)

// ConfigLoader defines the interface for loading and validating configuration.
type ConfigLoader interface {
	Load() (*Config, error)
	Watch(ctx context.Context) (<-chan *Config, error)
	Validate(cfg *Config) error
}

// configLoader implements ConfigLoader using Viper for configuration management.
type configLoader struct {
	validator *validator.Validate
}

// NewConfigLoader creates a new configuration loader with validation.
func NewConfigLoader() ConfigLoader {
	return &configLoader{
		validator: validator.New(),
	}
}

// Load loads configuration from environment variables and config files.
// It follows the RWWWRSE_ environment variable prefix convention.
func (l *configLoader) Load() (*Config, error) {
	v := viper.New()

	// Set configuration file properties
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath("/etc/rwwwrse/")
	v.AddConfigPath("$HOME/.rwwwrse")
	v.AddConfigPath(".")

	// Environment variable configuration
	v.SetEnvPrefix("RWWWRSE")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Set default values
	setDefaults(v)

	// Read config file if it exists (optional)
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		// Config file not found is acceptable, continue with env vars and defaults
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if err := l.Validate(&cfg); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &cfg, nil
}

// Watch monitors configuration changes and returns a channel with updated configs.
// This enables live configuration reloading.
func (l *configLoader) Watch(ctx context.Context) (<-chan *Config, error) {
	v := viper.New()

	// Set configuration file properties
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath("/etc/rwwwrse/")
	v.AddConfigPath("$HOME/.rwwwrse")
	v.AddConfigPath(".")

	// Environment variable configuration
	v.SetEnvPrefix("RWWWRSE")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Set default values
	setDefaults(v)

	// Read initial config
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read initial config file: %w", err)
		}
	}

	configCh := make(chan *Config, 1)

	// Send initial configuration
	var initialCfg Config
	if err := v.Unmarshal(&initialCfg); err == nil {
		if err := l.Validate(&initialCfg); err == nil {
			select {
			case configCh <- &initialCfg:
			case <-ctx.Done():
				close(configCh)
				return nil, ctx.Err()
			}
		}
	}

	// Watch for configuration changes
	v.WatchConfig()
	v.OnConfigChange(func(in fsnotify.Event) {
		var newCfg Config
		if err := v.Unmarshal(&newCfg); err != nil {
			// Log error but continue watching
			return
		}

		if err := l.Validate(&newCfg); err != nil {
			// Log validation error but continue watching
			return
		}

		select {
		case configCh <- &newCfg:
		case <-ctx.Done():
			return
		}
	})

	// Monitor context cancellation
	go func() {
		defer close(configCh)
		<-ctx.Done()
	}()

	return configCh, nil
}

// Validate validates the configuration using struct tags and custom validation rules.
func (l *configLoader) Validate(cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("configuration cannot be nil")
	}

	if err := l.validator.Struct(cfg); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	// Custom validation rules
	if err := l.validateCustomRules(cfg); err != nil {
		return fmt.Errorf("custom validation failed: %w", err)
	}

	return nil
}

// validateCustomRules performs additional validation beyond struct tags.
func (l *configLoader) validateCustomRules(cfg *Config) error {
	// Validate TLS configuration
	if cfg.TLS.AutoCert {
		if cfg.TLS.Email == "" {
			return fmt.Errorf("TLS email is required when auto-cert is enabled")
		}
		if len(cfg.TLS.Domains) == 0 {
			return fmt.Errorf("TLS domains are required when auto-cert is enabled")
		}
	}

	// Validate backend routes
	if len(cfg.Backends.Routes) == 0 {
		return fmt.Errorf("at least one backend route must be configured")
	}

	// Validate port conflicts
	if cfg.Server.Port == cfg.Server.HTTPSPort {
		return fmt.Errorf("HTTP and HTTPS ports cannot be the same")
	}

	if cfg.Server.Port == cfg.Metrics.Port || cfg.Server.HTTPSPort == cfg.Metrics.Port {
		return fmt.Errorf("metrics port cannot conflict with server ports")
	}

	// Validate rate limiting
	if cfg.RateLimit.RequestsPerSecond <= 0 {
		return fmt.Errorf("rate limit requests per second must be positive")
	}

	if cfg.RateLimit.BurstSize <= 0 {
		return fmt.Errorf("rate limit burst size must be positive")
	}

	// Validate health check configuration
	if cfg.Health.UnhealthyThreshold <= 0 {
		return fmt.Errorf("unhealthy threshold must be positive")
	}

	if cfg.Health.HealthyThreshold <= 0 {
		return fmt.Errorf("healthy threshold must be positive")
	}

	return nil
}
