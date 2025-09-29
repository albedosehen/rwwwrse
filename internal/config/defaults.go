package config

import (
	"time"

	"github.com/spf13/viper"
)

// setDefaults configures all default values for the application configuration.
// This ensures consistent behavior when configuration values are not explicitly set.
func setDefaults(v *viper.Viper) {
	// Server defaults
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.https_port", 8443)
	v.SetDefault("server.read_timeout", "30s")
	v.SetDefault("server.write_timeout", "30s")
	v.SetDefault("server.idle_timeout", "60s")
	v.SetDefault("server.graceful_timeout", "30s")

	// TLS defaults
	v.SetDefault("tls.enabled", true)
	v.SetDefault("tls.auto_cert", false) // Default to false to avoid validation issues
	v.SetDefault("tls.cache_dir", "/tmp/certs")
	v.SetDefault("tls.staging", false)
	v.SetDefault("tls.renew_before", "720h") // 30 days

	// Security defaults
	v.SetDefault("security.rate_limit_enabled", true)
	v.SetDefault("security.cors_enabled", true)
	v.SetDefault("security.cors_origins", []string{"*"})

	// Security headers defaults
	v.SetDefault("security.headers.content_type_nosniff", true)
	v.SetDefault("security.headers.frame_options", "DENY")
	v.SetDefault("security.headers.content_security_policy", "default-src 'self'")
	v.SetDefault("security.headers.strict_transport_security", "max-age=31536000; includeSubDomains")
	v.SetDefault("security.headers.referrer_policy", "strict-origin-when-cross-origin")

	// Logging defaults
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "json")
	v.SetDefault("logging.output", "stdout")

	// Metrics defaults
	v.SetDefault("metrics.enabled", true)
	v.SetDefault("metrics.path", "/metrics")
	v.SetDefault("metrics.port", 9090)

	// Health check defaults
	v.SetDefault("health.enabled", true)
	v.SetDefault("health.path", "/health")
	v.SetDefault("health.timeout", "5s")
	v.SetDefault("health.interval", "30s")
	v.SetDefault("health.unhealthy_threshold", 3)
	v.SetDefault("health.healthy_threshold", 2)

	// Rate limiting defaults
	v.SetDefault("ratelimit.requests_per_second", 100.0)
	v.SetDefault("ratelimit.burst_size", 200)
	v.SetDefault("ratelimit.cleanup_interval", "10m")

	// Backend route defaults (applied to each backend)
	setBackendDefaults(v)
}

// setBackendDefaults sets default values that apply to all backend routes.
// These can be overridden per-backend in the configuration.
func setBackendDefaults(v *viper.Viper) {
	v.SetDefault("backends.routes.*.health_path", "/health")
	v.SetDefault("backends.routes.*.health_interval", "30s")
	v.SetDefault("backends.routes.*.timeout", "30s")
	v.SetDefault("backends.routes.*.max_idle_conns", 100)
	v.SetDefault("backends.routes.*.max_idle_per_host", 10)
	v.SetDefault("backends.routes.*.dial_timeout", "10s")
}

// GetDefaultConfig returns a configuration object with all default values applied.
// This is useful for testing and documentation purposes.
func GetDefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Host:            "0.0.0.0",
			Port:            8080,
			HTTPSPort:       8443,
			ReadTimeout:     30 * time.Second,
			WriteTimeout:    30 * time.Second,
			IdleTimeout:     60 * time.Second,
			GracefulTimeout: 30 * time.Second,
		},
		TLS: TLSConfig{
			Enabled:     true,
			AutoCert:    false,
			CacheDir:    "/tmp/certs",
			Staging:     false,
			RenewBefore: 720 * time.Hour, // 30 days
		},
		Security: SecurityConfig{
			Headers: SecurityHeaders{
				ContentTypeNosniff:      true,
				FrameOptions:            "DENY",
				ContentSecurityPolicy:   "default-src 'self'",
				StrictTransportSecurity: "max-age=31536000; includeSubDomains",
				ReferrerPolicy:          "strict-origin-when-cross-origin",
			},
			RateLimitEnabled: true,
			CORSEnabled:      true,
			CORSOrigins:      []string{"*"},
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "json",
			Output: "stdout",
		},
		Metrics: MetricsConfig{
			Enabled: true,
			Path:    "/metrics",
			Port:    9090,
		},
		Health: HealthConfig{
			Enabled:            true,
			Path:               "/health",
			Timeout:            5 * time.Second,
			Interval:           30 * time.Second,
			UnhealthyThreshold: 3,
			HealthyThreshold:   2,
		},
		RateLimit: RateLimitConfig{
			RequestsPerSecond: 100.0,
			BurstSize:         200,
			CleanupInterval:   10 * time.Minute,
		},
		Backends: BackendsConfig{
			Routes: make(map[string]BackendRoute),
		},
	}
}

// GetDefaultBackendRoute returns a backend route configuration with default values.
func GetDefaultBackendRoute() BackendRoute {
	return BackendRoute{
		HealthPath:     "/health",
		HealthInterval: 30 * time.Second,
		Timeout:        30 * time.Second,
		MaxIdleConns:   100,
		MaxIdlePerHost: 10,
		DialTimeout:    10 * time.Second,
	}
}
