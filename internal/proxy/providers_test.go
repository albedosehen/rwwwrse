package proxy

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/albedosehen/rwwwrse/internal/config"
	"github.com/albedosehen/rwwwrse/internal/observability"
)

func TestProvideRouter(t *testing.T) {
	tests := []struct {
		name           string
		backendsConfig config.BackendsConfig
		logger         observability.Logger
		metrics        observability.MetricsCollector
		expectPanic    bool
	}{
		{
			name: "ProvideRouter_ValidConfig_ReturnsRouter",
			backendsConfig: config.BackendsConfig{
				Routes: map[string]config.BackendRoute{
					"api.example.com": {
						URL:     "http://api.example.com:8080",
						Timeout: 30 * time.Second,
					},
				},
			},
			logger:      &mockProvidersLogger{},
			metrics:     &mockProvidersMetrics{},
			expectPanic: false,
		},
		{
			name: "ProvideRouter_EmptyConfig_ReturnsRouter",
			backendsConfig: config.BackendsConfig{
				Routes: map[string]config.BackendRoute{},
			},
			logger:      &mockProvidersLogger{},
			metrics:     &mockProvidersMetrics{},
			expectPanic: false,
		},
		{
			name: "ProvideRouter_NilLogger_Panics",
			backendsConfig: config.BackendsConfig{
				Routes: map[string]config.BackendRoute{},
			},
			logger:      nil,
			metrics:     &mockProvidersMetrics{},
			expectPanic: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			if tt.logger != nil {
				mockLogger := tt.logger.(*mockProvidersLogger)
				mockLogger.On("Info", mock.Anything, mock.Anything).Return()
			}

			// Act & Assert
			if tt.expectPanic {
				assert.Panics(t, func() {
					ProvideRouter(tt.backendsConfig, tt.logger, tt.metrics)
				})
			} else {
				router := ProvideRouter(tt.backendsConfig, tt.logger, tt.metrics)
				assert.NotNil(t, router)
				assert.Implements(t, (*Router)(nil), router)

				mockLogger := tt.logger.(*mockProvidersLogger)
				mockLogger.AssertExpectations(t)
			}
		})
	}
}

func TestProvideBackendManager(t *testing.T) {
	tests := []struct {
		name           string
		backendsConfig config.BackendsConfig
		logger         observability.Logger
		metrics        observability.MetricsCollector
		expectPanic    bool
	}{
		{
			name: "ProvideBackendManager_ValidConfig_ReturnsManager",
			backendsConfig: config.BackendsConfig{
				Routes: map[string]config.BackendRoute{
					"api.example.com": {
						URL:     "http://api.example.com:8080",
						Timeout: 30 * time.Second,
					},
				},
			},
			logger:      &mockProvidersLogger{},
			metrics:     &mockProvidersMetrics{},
			expectPanic: false,
		},
		{
			name: "ProvideBackendManager_EmptyConfig_ReturnsManager",
			backendsConfig: config.BackendsConfig{
				Routes: map[string]config.BackendRoute{},
			},
			logger:      &mockProvidersLogger{},
			metrics:     &mockProvidersMetrics{},
			expectPanic: false,
		},
		{
			name: "ProvideBackendManager_NilLogger_Panics",
			backendsConfig: config.BackendsConfig{
				Routes: map[string]config.BackendRoute{},
			},
			logger:      nil,
			metrics:     &mockProvidersMetrics{},
			expectPanic: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			if tt.logger != nil {
				mockLogger := tt.logger.(*mockProvidersLogger)
				// Mock creation logging for each backend and manager
				for range tt.backendsConfig.Routes {
					mockLogger.On("Info", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()
				}
				// Mock manager initialization logging
				mockLogger.On("Info", mock.Anything, mock.Anything, mock.Anything).Return()
			}

			// Act & Assert
			if tt.expectPanic {
				assert.Panics(t, func() {
					ProvideBackendManager(tt.backendsConfig, tt.logger, tt.metrics)
				})
			} else {
				manager := ProvideBackendManager(tt.backendsConfig, tt.logger, tt.metrics)
				assert.NotNil(t, manager)
				assert.Implements(t, (*BackendManager)(nil), manager)

				mockLogger := tt.logger.(*mockProvidersLogger)
				mockLogger.AssertExpectations(t)
			}
		})
	}
}

func TestProvideConnectionPool(t *testing.T) {
	tests := []struct {
		name           string
		backendsConfig config.BackendsConfig
		logger         observability.Logger
		expectPanic    bool
	}{
		{
			name: "ProvideConnectionPool_ValidConfig_ReturnsPool",
			backendsConfig: config.BackendsConfig{
				Routes: map[string]config.BackendRoute{
					"api.example.com": {
						URL:     "http://api.example.com:8080",
						Timeout: 30 * time.Second,
					},
				},
			},
			logger:      &mockProvidersLogger{},
			expectPanic: false,
		},
		{
			name: "ProvideConnectionPool_EmptyConfig_ReturnsPool",
			backendsConfig: config.BackendsConfig{
				Routes: map[string]config.BackendRoute{},
			},
			logger:      &mockProvidersLogger{},
			expectPanic: false,
		},
		{
			name: "ProvideConnectionPool_NilLogger_Panics",
			backendsConfig: config.BackendsConfig{
				Routes: map[string]config.BackendRoute{},
			},
			logger:      nil,
			expectPanic: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			if tt.logger != nil {
				mockLogger := tt.logger.(*mockProvidersLogger)
				mockLogger.On("Info", mock.Anything, mock.Anything).Return()
			}

			// Act & Assert
			mockMetrics := &mockProvidersMetrics{}
			if tt.expectPanic {
				assert.Panics(t, func() {
					ProvideConnectionPool(tt.backendsConfig, tt.logger, mockMetrics)
				})
			} else {
				pool := ProvideConnectionPool(tt.backendsConfig, tt.logger, mockMetrics)
				assert.NotNil(t, pool)
				assert.Implements(t, (*ConnectionPool)(nil), pool)

				mockLogger := tt.logger.(*mockProvidersLogger)
				mockLogger.AssertExpectations(t)
			}
		})
	}
}

func TestProvideProxyHandler(t *testing.T) {
	tests := []struct {
		name        string
		router      Router
		logger      observability.Logger
		metrics     observability.MetricsCollector
		expectPanic bool
	}{
		{
			name:        "ProvideProxyHandler_ValidInputs_ReturnsHandler",
			router:      &mockRouter{},
			logger:      &mockProvidersLogger{},
			metrics:     &mockProvidersMetrics{},
			expectPanic: false,
		},
		{
			name:        "ProvideProxyHandler_NilRouter_Panics",
			router:      nil,
			logger:      &mockProvidersLogger{},
			metrics:     &mockProvidersMetrics{},
			expectPanic: true,
		},
		{
			name:        "ProvideProxyHandler_NilLogger_Panics",
			router:      &mockRouter{},
			logger:      nil,
			metrics:     &mockProvidersMetrics{},
			expectPanic: true,
		},
		{
			name:        "ProvideProxyHandler_NilMetrics_ReturnsHandler",
			router:      &mockRouter{},
			logger:      &mockProvidersLogger{},
			metrics:     nil,
			expectPanic: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			if tt.logger != nil {
				mockLogger := tt.logger.(*mockProvidersLogger)
				mockLogger.On("Info", mock.Anything, mock.Anything).Return()
			}

			// Act & Assert
			if tt.expectPanic {
				assert.Panics(t, func() {
					ProvideProxyHandler(tt.router, tt.logger, tt.metrics)
				})
			} else {
				handler := ProvideProxyHandler(tt.router, tt.logger, tt.metrics)
				assert.NotNil(t, handler)
				assert.Implements(t, (*ProxyHandler)(nil), handler)

				if tt.logger != nil {
					mockLogger := tt.logger.(*mockProvidersLogger)
					mockLogger.AssertExpectations(t)
				}
			}
		})
	}
}

func TestProvideRouter_PanicMessage(t *testing.T) {
	t.Run("ProvideRouter_InvalidConfig_PanicsWithMessage", func(t *testing.T) {
		// Arrange
		backendsConfig := config.BackendsConfig{
			Routes: map[string]config.BackendRoute{
				"api.example.com": {
					URL: "invalid-url", // Invalid URL will cause error
				},
			},
		}
		mockLogger := &mockProvidersLogger{}

		// Act & Assert
		assert.PanicsWithValue(t, "failed to create router: configuration field url: parse \"invalid-url\": invalid URI for request", func() {
			ProvideRouter(backendsConfig, mockLogger, &mockProvidersMetrics{})
		})
	})
}

func TestProvideBackendManager_PanicMessage(t *testing.T) {
	t.Run("ProvideBackendManager_InvalidConfig_PanicsWithMessage", func(t *testing.T) {
		// Arrange
		backendsConfig := config.BackendsConfig{
			Routes: map[string]config.BackendRoute{
				"api.example.com": {
					URL: "invalid-url", // Invalid URL will cause error
				},
			},
		}
		mockLogger := &mockProvidersLogger{}

		// Act & Assert
		assert.PanicsWithValue(t, "failed to create backend manager: configuration field url: parse \"invalid-url\": invalid URI for request", func() {
			ProvideBackendManager(backendsConfig, mockLogger, &mockProvidersMetrics{})
		})
	})
}

func TestProvideConnectionPool_PanicMessage(t *testing.T) {
	t.Run("ProvideConnectionPool_InvalidConfig_PanicsWithMessage", func(t *testing.T) {
		// Act & Assert
		assert.PanicsWithValue(t, "failed to create connection pool: configuration field logger: <nil>", func() {
			ProvideConnectionPool(config.BackendsConfig{}, nil, &mockProvidersMetrics{})
		})
	})
}

func TestProvideProxyHandler_PanicMessage(t *testing.T) {
	t.Run("ProvideProxyHandler_InvalidConfig_PanicsWithMessage", func(t *testing.T) {
		// Act & Assert
		assert.PanicsWithValue(t, "failed to create proxy handler: configuration field router: <nil>", func() {
			ProvideProxyHandler(nil, &mockProvidersLogger{}, &mockProvidersMetrics{})
		})
	})
}

// Mock types for providers tests
type mockProvidersLogger struct {
	mock.Mock
}

func (m *mockProvidersLogger) Debug(ctx context.Context, msg string, fields ...observability.Field) {
	args := []interface{}{ctx, msg}
	for _, field := range fields {
		args = append(args, field)
	}
	m.Called(args...)
}

func (m *mockProvidersLogger) Info(ctx context.Context, msg string, fields ...observability.Field) {
	args := []interface{}{ctx, msg}
	for _, field := range fields {
		args = append(args, field)
	}
	m.Called(args...)
}

func (m *mockProvidersLogger) Warn(ctx context.Context, msg string, fields ...observability.Field) {
	args := []interface{}{ctx, msg}
	for _, field := range fields {
		args = append(args, field)
	}
	m.Called(args...)
}

func (m *mockProvidersLogger) Error(ctx context.Context, err error, msg string, fields ...observability.Field) {
	args := []interface{}{ctx, err, msg}
	for _, field := range fields {
		args = append(args, field)
	}
	m.Called(args...)
}

func (m *mockProvidersLogger) WithFields(fields ...observability.Field) observability.Logger {
	args := []interface{}{}
	for _, field := range fields {
		args = append(args, field)
	}
	ret := m.Called(args...)
	return ret.Get(0).(observability.Logger)
}

func (m *mockProvidersLogger) WithContext(ctx context.Context) observability.Logger {
	ret := m.Called(ctx)
	return ret.Get(0).(observability.Logger)
}

type mockProvidersMetrics struct {
	mock.Mock
}

func (m *mockProvidersMetrics) RecordRequest(method, host, status string, duration time.Duration) {
	m.Called(method, host, status, duration)
}

func (m *mockProvidersMetrics) RecordBackendRequest(backend, status string, duration time.Duration) {
	m.Called(backend, status, duration)
}

func (m *mockProvidersMetrics) IncActiveConnections() {
	m.Called()
}

func (m *mockProvidersMetrics) DecActiveConnections() {
	m.Called()
}

func (m *mockProvidersMetrics) RecordCertificateRenewal(domain string, success bool) {
	m.Called(domain, success)
}

func (m *mockProvidersMetrics) RecordRateLimitHit(key string) {
	m.Called(key)
}

func (m *mockProvidersMetrics) RecordHealthCheck(target string, success bool, duration time.Duration) {
	m.Called(target, success, duration)
}

// Benchmark tests for providers
func BenchmarkProvideRouter(b *testing.B) {
	// Setup
	backendsConfig := config.BackendsConfig{
		Routes: map[string]config.BackendRoute{
			"api.example.com": {
				URL:     "http://api.example.com:8080",
				Timeout: 30 * time.Second,
			},
		},
	}
	mockLogger := &mockProvidersLogger{}
	mockLogger.On("Info", mock.Anything, mock.Anything).Return()
	mockMetrics := &mockProvidersMetrics{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ProvideRouter(backendsConfig, mockLogger, mockMetrics)
	}
}

func BenchmarkProvideConnectionPool(b *testing.B) {
	// Setup
	backendsConfig := config.BackendsConfig{
		Routes: map[string]config.BackendRoute{},
	}
	mockLogger := &mockProvidersLogger{}
	mockLogger.On("Info", mock.Anything, mock.Anything).Return()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ProvideConnectionPool(backendsConfig, mockLogger, &mockProvidersMetrics{})
	}
}
