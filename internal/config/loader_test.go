package config

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConfigLoader(t *testing.T) {
	loader := NewConfigLoader()
	require.NotNil(t, loader)
}

func TestConfigLoader_Load(t *testing.T) {
	tests := []struct {
		name     string
		envVars  map[string]string
		wantErr  bool
		validate func(*testing.T, *Config)
	}{
		{
			name: "load defaults when no env vars set",
			envVars: map[string]string{
				// Disable AutoCert to avoid validation requirements
				"RWWWRSE_TLS_AUTO_CERT": "false",
				// Add a backend route to satisfy validation
				"RWWWRSE_BACKENDS_ROUTES_EXAMPLE_URL": "http://example.com",
			},
			wantErr: false,
			validate: func(t *testing.T, config *Config) {
				assert.Equal(t, "0.0.0.0", config.Server.Host)
				assert.Equal(t, 8080, config.Server.Port)
				assert.Equal(t, 8443, config.Server.HTTPSPort)
				assert.False(t, config.TLS.AutoCert)
			},
		},
		{
			name: "override server configuration",
			envVars: map[string]string{
				"RWWWRSE_SERVER_HOST":                 "localhost",
				"RWWWRSE_SERVER_PORT":                 "9090",
				"RWWWRSE_SERVER_HTTPS_PORT":           "9443",
				"RWWWRSE_METRICS_PORT":                "8090", // Avoid port conflict
				"RWWWRSE_TLS_AUTO_CERT":               "false",
				"RWWWRSE_BACKENDS_ROUTES_EXAMPLE_URL": "http://example.com",
			},
			wantErr: false,
			validate: func(t *testing.T, config *Config) {
				assert.Equal(t, "localhost", config.Server.Host)
				assert.Equal(t, 9090, config.Server.Port)
				assert.Equal(t, 9443, config.Server.HTTPSPort)
			},
		},
		{
			name: "override TLS configuration",
			envVars: map[string]string{
				"RWWWRSE_TLS_ENABLED":                 "false",
				"RWWWRSE_TLS_AUTO_CERT":               "false",
				"RWWWRSE_TLS_CACHE_DIR":               "/tmp/test-certs",
				"RWWWRSE_TLS_EMAIL":                   "test@example.com",
				"RWWWRSE_TLS_STAGING":                 "true",
				"RWWWRSE_BACKENDS_ROUTES_EXAMPLE_URL": "http://example.com",
			},
			wantErr: false,
			validate: func(t *testing.T, config *Config) {
				assert.False(t, config.TLS.Enabled)
				assert.False(t, config.TLS.AutoCert)
				assert.Equal(t, "/tmp/test-certs", config.TLS.CacheDir)
				assert.Equal(t, "test@example.com", config.TLS.Email)
				assert.True(t, config.TLS.Staging)
			},
		},
		{
			name: "override logging configuration",
			envVars: map[string]string{
				"RWWWRSE_LOGGING_LEVEL":               "debug",
				"RWWWRSE_LOGGING_FORMAT":              "text",
				"RWWWRSE_LOGGING_OUTPUT":              "stderr",
				"RWWWRSE_TLS_AUTO_CERT":               "false",
				"RWWWRSE_BACKENDS_ROUTES_EXAMPLE_URL": "http://example.com",
			},
			wantErr: false,
			validate: func(t *testing.T, config *Config) {
				assert.Equal(t, "debug", config.Logging.Level)
				assert.Equal(t, "text", config.Logging.Format)
				assert.Equal(t, "stderr", config.Logging.Output)
			},
		},
		{
			name: "override metrics configuration",
			envVars: map[string]string{
				"RWWWRSE_METRICS_ENABLED":             "false",
				"RWWWRSE_METRICS_PORT":                "8090",
				"RWWWRSE_METRICS_PATH":                "/stats",
				"RWWWRSE_TLS_AUTO_CERT":               "false",
				"RWWWRSE_BACKENDS_ROUTES_EXAMPLE_URL": "http://example.com",
			},
			wantErr: false,
			validate: func(t *testing.T, config *Config) {
				assert.False(t, config.Metrics.Enabled)
				assert.Equal(t, 8090, config.Metrics.Port)
				assert.Equal(t, "/stats", config.Metrics.Path)
			},
		},
		{
			name: "override health configuration",
			envVars: map[string]string{
				"RWWWRSE_HEALTH_ENABLED":              "false",
				"RWWWRSE_HEALTH_PATH":                 "/status",
				"RWWWRSE_HEALTH_TIMEOUT":              "10s",
				"RWWWRSE_HEALTH_INTERVAL":             "60s",
				"RWWWRSE_HEALTH_UNHEALTHY_THRESHOLD":  "5",
				"RWWWRSE_HEALTH_HEALTHY_THRESHOLD":    "1",
				"RWWWRSE_TLS_AUTO_CERT":               "false",
				"RWWWRSE_BACKENDS_ROUTES_EXAMPLE_URL": "http://example.com",
			},
			wantErr: false,
			validate: func(t *testing.T, config *Config) {
				assert.False(t, config.Health.Enabled)
				assert.Equal(t, "/status", config.Health.Path)
				assert.Equal(t, 10*time.Second, config.Health.Timeout)
				assert.Equal(t, 60*time.Second, config.Health.Interval)
				assert.Equal(t, 5, config.Health.UnhealthyThreshold)
				assert.Equal(t, 1, config.Health.HealthyThreshold)
			},
		},
		{
			name: "override rate limit configuration",
			envVars: map[string]string{
				"RWWWRSE_RATELIMIT_REQUESTS_PER_SECOND": "50.5",
				"RWWWRSE_RATELIMIT_BURST_SIZE":          "100",
				"RWWWRSE_RATELIMIT_CLEANUP_INTERVAL":    "5m",
				"RWWWRSE_TLS_AUTO_CERT":                 "false",
				"RWWWRSE_BACKENDS_ROUTES_EXAMPLE_URL":   "http://example.com",
			},
			wantErr: false,
			validate: func(t *testing.T, config *Config) {
				assert.Equal(t, 50.5, config.RateLimit.RequestsPerSecond)
				assert.Equal(t, 100, config.RateLimit.BurstSize)
				assert.Equal(t, 5*time.Minute, config.RateLimit.CleanupInterval)
			},
		},
		{
			name: "override security configuration",
			envVars: map[string]string{
				"RWWWRSE_SECURITY_RATE_LIMIT_ENABLED": "false",
				"RWWWRSE_SECURITY_CORS_ENABLED":       "false",
				"RWWWRSE_TLS_AUTO_CERT":               "false",
				"RWWWRSE_BACKENDS_ROUTES_EXAMPLE_URL": "http://example.com",
			},
			wantErr: false,
			validate: func(t *testing.T, config *Config) {
				assert.False(t, config.Security.RateLimitEnabled)
				assert.False(t, config.Security.CORSEnabled)
			},
		},
		{
			name: "invalid port number",
			envVars: map[string]string{
				"RWWWRSE_SERVER_PORT": "invalid",
			},
			wantErr: true,
		},
		{
			name: "invalid boolean value",
			envVars: map[string]string{
				"RWWWRSE_TLS_ENABLED": "invalid",
			},
			wantErr: true,
		},
		{
			name: "invalid timeout duration",
			envVars: map[string]string{
				"RWWWRSE_SERVER_READ_TIMEOUT": "invalid",
			},
			wantErr: true,
		},
		{
			name: "invalid log level",
			envVars: map[string]string{
				"RWWWRSE_LOGGING_LEVEL": "invalid",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up environment variables
			cleanup := setEnvVars(t, tt.envVars)
			defer cleanup()

			loader := NewConfigLoader()
			config, err := loader.Load()

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, config)

			// Validate configuration values
			if tt.validate != nil {
				tt.validate(t, config)
			}
		})
	}
}

func TestConfigLoader_Validate(t *testing.T) {
	loader := NewConfigLoader()

	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name: "valid default config with AutoCert disabled",
			config: func() *Config {
				cfg := GetDefaultConfig()
				cfg.TLS.AutoCert = false // Disable to avoid validation requirements
				cfg.Backends.Routes = map[string]BackendRoute{
					"example": {
						URL:            "http://example.com",
						HealthPath:     "/health",
						HealthInterval: 30 * time.Second,
						Timeout:        30 * time.Second,
						MaxIdleConns:   100,
						MaxIdlePerHost: 10,
						DialTimeout:    10 * time.Second,
					},
				}
				return cfg
			}(),
			wantErr: false,
		},
		{
			name: "invalid server port - too low",
			config: &Config{
				Server: ServerConfig{
					Host: "localhost",
					Port: 0,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid server port - too high",
			config: &Config{
				Server: ServerConfig{
					Host: "localhost",
					Port: 70000,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid metrics port",
			config: &Config{
				Metrics: MetricsConfig{
					Port: -1,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid log level",
			config: &Config{
				Logging: LoggingConfig{
					Level: "invalid",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid log format",
			config: &Config{
				Logging: LoggingConfig{
					Format: "invalid",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := loader.Validate(tt.config)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfigLoader_Watch(t *testing.T) {
	// Create a temporary config file
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.yaml")

	configContent := `
server:
  host: "watch-host"
  port: 8080
tls:
  enabled: true
  auto_cert: false
backends:
  routes: {}
`

	err := os.WriteFile(configFile, []byte(configContent), 0644)
	require.NoError(t, err)

	// Change working directory to temp dir for this test
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		if err := os.Chdir(oldWd); err != nil {
			t.Errorf("Failed to restore working directory: %v", err)
		}
	}()

	err = os.Chdir(tempDir)
	require.NoError(t, err)

	loader := NewConfigLoader()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	configCh, err := loader.Watch(ctx)
	require.NoError(t, err)

	// Wait for initial config
	select {
	case config := <-configCh:
		assert.Equal(t, "watch-host", config.Server.Host)
		assert.Equal(t, 8080, config.Server.Port)
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for initial config")
	}

	// Test context cancellation
	cancel()

	// Channel should close when context is cancelled
	select {
	case _, ok := <-configCh:
		assert.False(t, ok, "Channel should be closed after context cancellation")
	case <-time.After(2 * time.Second):
		t.Fatal("Channel should have been closed")
	}
}

func TestGetDefaultBackendRoute(t *testing.T) {
	route := GetDefaultBackendRoute()

	assert.Equal(t, "/health", route.HealthPath)
	assert.Equal(t, 30*time.Second, route.HealthInterval)
	assert.Equal(t, 30*time.Second, route.Timeout)
	assert.Equal(t, 100, route.MaxIdleConns)
	assert.Equal(t, 10, route.MaxIdlePerHost)
	assert.Equal(t, 10*time.Second, route.DialTimeout)
	assert.Empty(t, route.URL) // Should be empty as it's required to be set
}

// Helper function to set environment variables for testing
func setEnvVars(_ *testing.T, vars map[string]string) func() {
	originalVars := make(map[string]string)

	for key, value := range vars {
		originalVars[key] = os.Getenv(key)
		os.Setenv(key, value)
	}

	return func() {
		for key, originalValue := range originalVars {
			if originalValue == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, originalValue)
			}
		}
	}
}

// Benchmark tests
func BenchmarkConfigLoader_Load(b *testing.B) {
	loader := NewConfigLoader()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		config, err := loader.Load()
		if err != nil {
			b.Fatal(err)
		}
		_ = config
	}
}

func BenchmarkConfigLoader_Validate(b *testing.B) {
	loader := NewConfigLoader()
	config := GetDefaultConfig()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := loader.Validate(config)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGetDefaultConfig(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = GetDefaultConfig()
	}
}

func BenchmarkGetDefaultBackendRoute(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = GetDefaultBackendRoute()
	}
}
