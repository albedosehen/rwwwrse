// Package config provides configuration management for the rwwwrse reverse proxy server.
// It handles loading, validation, and parsing of configuration from environment variables
// and configuration files using the RWWWRSE_ prefix convention.
package config

import (
	"fmt"
	"time"
)

// Config represents the complete application configuration structure.
// All configuration uses the RWWWRSE_ prefix for environment variables.
type Config struct {
	Server    ServerConfig    `mapstructure:"server" validate:"required"`
	TLS       TLSConfig       `mapstructure:"tls" validate:"required"`
	Backends  BackendsConfig  `mapstructure:"backends" validate:"required"`
	Security  SecurityConfig  `mapstructure:"security"`
	Logging   LoggingConfig   `mapstructure:"logging"`
	Metrics   MetricsConfig   `mapstructure:"metrics"`
	Health    HealthConfig    `mapstructure:"health"`
	RateLimit RateLimitConfig `mapstructure:"ratelimit"`
}

// ServerConfig contains HTTP/HTTPS server configuration.
type ServerConfig struct {
	Host            string        `mapstructure:"host" default:"0.0.0.0"`
	Port            int           `mapstructure:"port" default:"8080" validate:"min=1,max=65535"`
	HTTPSPort       int           `mapstructure:"https_port" default:"8443" validate:"min=1,max=65535"`
	ReadTimeout     time.Duration `mapstructure:"read_timeout" default:"30s"`
	WriteTimeout    time.Duration `mapstructure:"write_timeout" default:"30s"`
	IdleTimeout     time.Duration `mapstructure:"idle_timeout" default:"60s"`
	GracefulTimeout time.Duration `mapstructure:"graceful_timeout" default:"30s"`
}

// TLSConfig contains TLS certificate management configuration.
type TLSConfig struct {
	Enabled     bool          `mapstructure:"enabled" default:"true"`
	AutoCert    bool          `mapstructure:"auto_cert" default:"true"`
	Email       string        `mapstructure:"email" validate:"required_if=AutoCert true,email"`
	Domains     []string      `mapstructure:"domains" validate:"required_if=AutoCert true,min=1"`
	CacheDir    string        `mapstructure:"cache_dir" default:"/tmp/certs"`
	Staging     bool          `mapstructure:"staging" default:"false"`
	RenewBefore time.Duration `mapstructure:"renew_before" default:"720h"` // 30 days
}

// BackendsConfig contains backend routing configuration.
type BackendsConfig struct {
	Routes map[string]BackendRoute `mapstructure:"routes" validate:"required,min=1"`
}

// BackendRoute defines configuration for a single backend service.
type BackendRoute struct {
	URL            string        `mapstructure:"url" validate:"required,url"`
	HealthPath     string        `mapstructure:"health_path" default:"/health"`
	HealthInterval time.Duration `mapstructure:"health_interval" default:"30s"`
	Timeout        time.Duration `mapstructure:"timeout" default:"30s"`
	MaxIdleConns   int           `mapstructure:"max_idle_conns" default:"100"`
	MaxIdlePerHost int           `mapstructure:"max_idle_per_host" default:"10"`
	DialTimeout    time.Duration `mapstructure:"dial_timeout" default:"10s"`
}

// SecurityConfig contains security-related configuration.
type SecurityConfig struct {
	Headers          SecurityHeaders `mapstructure:"headers"`
	RateLimitEnabled bool            `mapstructure:"rate_limit_enabled" default:"true"`
	CORSEnabled      bool            `mapstructure:"cors_enabled" default:"true"`
	CORSOrigins      []string        `mapstructure:"cors_origins" default:"*"`
}

// SecurityHeaders defines HTTP security headers configuration.
type SecurityHeaders struct {
	ContentTypeNosniff      bool   `mapstructure:"content_type_nosniff" default:"true"`
	FrameOptions            string `mapstructure:"frame_options" default:"DENY"`
	ContentSecurityPolicy   string `mapstructure:"content_security_policy" default:"default-src 'self'"`
	StrictTransportSecurity string `mapstructure:"strict_transport_security" default:"max-age=31536000; includeSubDomains"`
	ReferrerPolicy          string `mapstructure:"referrer_policy" default:"strict-origin-when-cross-origin"`
}

// LoggingConfig contains logging configuration.
type LoggingConfig struct {
	Level  string `mapstructure:"level" default:"info" validate:"oneof=debug info warn error"`
	Format string `mapstructure:"format" default:"json" validate:"oneof=json text"`
	Output string `mapstructure:"output" default:"stdout"`
}

// MetricsConfig contains metrics collection configuration.
type MetricsConfig struct {
	Enabled bool   `mapstructure:"enabled" default:"true"`
	Path    string `mapstructure:"path" default:"/metrics"`
	Port    int    `mapstructure:"port" default:"9090" validate:"min=1,max=65535"`
}

// HealthConfig contains health check configuration.
type HealthConfig struct {
	Enabled            bool          `mapstructure:"enabled" default:"true"`
	Path               string        `mapstructure:"path" default:"/health"`
	Timeout            time.Duration `mapstructure:"timeout" default:"5s"`
	Interval           time.Duration `mapstructure:"interval" default:"30s"`
	UnhealthyThreshold int           `mapstructure:"unhealthy_threshold" default:"3"`
	HealthyThreshold   int           `mapstructure:"healthy_threshold" default:"2"`
}

// RateLimitConfig contains rate limiting configuration.
type RateLimitConfig struct {
	RequestsPerSecond float64       `mapstructure:"requests_per_second" default:"100"`
	BurstSize         int           `mapstructure:"burst_size" default:"200"`
	CleanupInterval   time.Duration `mapstructure:"cleanup_interval" default:"10m"`
}

// GetCAEndpoint returns the appropriate ACME CA endpoint based on staging configuration.
func (t *TLSConfig) GetCAEndpoint() string {
	if t.Staging {
		return "https://acme-staging-v02.api.letsencrypt.org/directory"
	}
	return "https://acme-v02.api.letsencrypt.org/directory"
}

// IsProductionReady validates if the configuration is suitable for production deployment.
func (c *Config) IsProductionReady() bool {
	return !c.TLS.Staging && c.TLS.AutoCert && len(c.TLS.Domains) > 0 && c.TLS.Email != ""
}

// GetServerAddress returns the formatted server address for HTTP listening.
func (s *ServerConfig) GetServerAddress() string {
	if s.Host == "" {
		s.Host = "0.0.0.0"
	}
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}

// GetHTTPSAddress returns the formatted server address for HTTPS listening.
func (s *ServerConfig) GetHTTPSAddress() string {
	if s.Host == "" {
		s.Host = "0.0.0.0"
	}
	return fmt.Sprintf("%s:%d", s.Host, s.HTTPSPort)
}
