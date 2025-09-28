package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig_IsProductionReady(t *testing.T) {
	tests := []struct {
		name   string
		config *Config
		want   bool
	}{
		{
			name: "production ready config",
			config: &Config{
				TLS: TLSConfig{
					AutoCert: true,
					Staging:  false,
					Email:    "admin@example.com",
					Domains:  []string{"example.com", "api.example.com"},
				},
			},
			want: true,
		},
		{
			name: "staging config not production ready",
			config: &Config{
				TLS: TLSConfig{
					AutoCert: true,
					Staging:  true,
					Email:    "admin@example.com",
					Domains:  []string{"example.com"},
				},
			},
			want: false,
		},
		{
			name: "no auto cert not production ready",
			config: &Config{
				TLS: TLSConfig{
					AutoCert: false,
					Staging:  false,
					Email:    "admin@example.com",
					Domains:  []string{"example.com"},
				},
			},
			want: false,
		},
		{
			name: "no domains not production ready",
			config: &Config{
				TLS: TLSConfig{
					AutoCert: true,
					Staging:  false,
					Email:    "admin@example.com",
					Domains:  []string{},
				},
			},
			want: false,
		},
		{
			name: "no email not production ready",
			config: &Config{
				TLS: TLSConfig{
					AutoCert: true,
					Staging:  false,
					Email:    "",
					Domains:  []string{"example.com"},
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.config.IsProductionReady())
		})
	}
}

func TestTLSConfig_GetCAEndpoint(t *testing.T) {
	tests := []struct {
		name    string
		staging bool
		want    string
	}{
		{
			name:    "production endpoint",
			staging: false,
			want:    "https://acme-v02.api.letsencrypt.org/directory",
		},
		{
			name:    "staging endpoint",
			staging: true,
			want:    "https://acme-staging-v02.api.letsencrypt.org/directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &TLSConfig{
				Staging: tt.staging,
			}
			assert.Equal(t, tt.want, config.GetCAEndpoint())
		})
	}
}

func TestServerConfig_GetServerAddress(t *testing.T) {
	tests := []struct {
		name   string
		config *ServerConfig
		want   string
	}{
		{
			name: "with host specified",
			config: &ServerConfig{
				Host: "localhost",
				Port: 8080,
			},
			want: "localhost:8080",
		},
		{
			name: "empty host defaults to 0.0.0.0",
			config: &ServerConfig{
				Host: "",
				Port: 9090,
			},
			want: "0.0.0.0:9090",
		},
		{
			name: "different port",
			config: &ServerConfig{
				Host: "127.0.0.1",
				Port: 3000,
			},
			want: "127.0.0.1:3000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.config.GetServerAddress())
		})
	}
}

func TestServerConfig_GetHTTPSAddress(t *testing.T) {
	tests := []struct {
		name   string
		config *ServerConfig
		want   string
	}{
		{
			name: "with host specified",
			config: &ServerConfig{
				Host:      "localhost",
				HTTPSPort: 8443,
			},
			want: "localhost:8443",
		},
		{
			name: "empty host defaults to 0.0.0.0",
			config: &ServerConfig{
				Host:      "",
				HTTPSPort: 9443,
			},
			want: "0.0.0.0:9443",
		},
		{
			name: "different HTTPS port",
			config: &ServerConfig{
				Host:      "127.0.0.1",
				HTTPSPort: 4443,
			},
			want: "127.0.0.1:4443",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.config.GetHTTPSAddress())
		})
	}
}

func TestGetDefaultConfig(t *testing.T) {
	config := GetDefaultConfig()

	require.NotNil(t, config)

	// Test server defaults
	assert.Equal(t, "0.0.0.0", config.Server.Host)
	assert.Equal(t, 8080, config.Server.Port)
	assert.Equal(t, 8443, config.Server.HTTPSPort)
	assert.Equal(t, 30*time.Second, config.Server.ReadTimeout)
	assert.Equal(t, 30*time.Second, config.Server.WriteTimeout)
	assert.Equal(t, 60*time.Second, config.Server.IdleTimeout)
	assert.Equal(t, 30*time.Second, config.Server.GracefulTimeout)

	// Test TLS defaults
	assert.True(t, config.TLS.Enabled)
	assert.True(t, config.TLS.AutoCert)
	assert.Equal(t, "/tmp/certs", config.TLS.CacheDir)
	assert.False(t, config.TLS.Staging)
	assert.Equal(t, 720*time.Hour, config.TLS.RenewBefore)

	// Test logging defaults
	assert.Equal(t, "info", config.Logging.Level)
	assert.Equal(t, "json", config.Logging.Format)
	assert.Equal(t, "stdout", config.Logging.Output)

	// Test metrics defaults
	assert.True(t, config.Metrics.Enabled)
	assert.Equal(t, "/metrics", config.Metrics.Path)
	assert.Equal(t, 9090, config.Metrics.Port)

	// Test health defaults
	assert.True(t, config.Health.Enabled)
	assert.Equal(t, "/health", config.Health.Path)
	assert.Equal(t, 5*time.Second, config.Health.Timeout)
	assert.Equal(t, 30*time.Second, config.Health.Interval)
	assert.Equal(t, 3, config.Health.UnhealthyThreshold)
	assert.Equal(t, 2, config.Health.HealthyThreshold)

	// Test rate limit defaults
	assert.Equal(t, 100.0, config.RateLimit.RequestsPerSecond)
	assert.Equal(t, 200, config.RateLimit.BurstSize)
	assert.Equal(t, 10*time.Minute, config.RateLimit.CleanupInterval)

	// Test security defaults
	assert.True(t, config.Security.RateLimitEnabled)
	assert.True(t, config.Security.CORSEnabled)
	assert.Contains(t, config.Security.CORSOrigins, "*")

	// Test security headers defaults
	assert.True(t, config.Security.Headers.ContentTypeNosniff)
	assert.Equal(t, "DENY", config.Security.Headers.FrameOptions)
	assert.Equal(t, "default-src 'self'", config.Security.Headers.ContentSecurityPolicy)
	assert.Equal(t, "max-age=31536000; includeSubDomains", config.Security.Headers.StrictTransportSecurity)
	assert.Equal(t, "strict-origin-when-cross-origin", config.Security.Headers.ReferrerPolicy)
}

func TestBackendRoute_Defaults(t *testing.T) {
	// Test that BackendRoute has sensible zero values
	route := BackendRoute{}

	assert.Empty(t, route.URL)
	assert.Empty(t, route.HealthPath)
	assert.Equal(t, time.Duration(0), route.HealthInterval)
	assert.Equal(t, time.Duration(0), route.Timeout)
	assert.Equal(t, 0, route.MaxIdleConns)
	assert.Equal(t, 0, route.MaxIdlePerHost)
	assert.Equal(t, time.Duration(0), route.DialTimeout)
}

// Benchmark tests
func BenchmarkConfig_IsProductionReady(b *testing.B) {
	config := &Config{
		TLS: TLSConfig{
			AutoCert: true,
			Staging:  false,
			Email:    "admin@example.com",
			Domains:  []string{"example.com", "api.example.com"},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = config.IsProductionReady()
	}
}

func BenchmarkTLSConfig_GetCAEndpoint(b *testing.B) {
	config := &TLSConfig{
		Staging: false,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = config.GetCAEndpoint()
	}
}

func BenchmarkServerConfig_GetServerAddress(b *testing.B) {
	config := &ServerConfig{
		Host: "localhost",
		Port: 8080,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = config.GetServerAddress()
	}
}
