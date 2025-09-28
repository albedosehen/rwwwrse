package proxy

import (
	"context"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/albedosehen/rwwwrse/internal/config"
	proxyerrors "github.com/albedosehen/rwwwrse/internal/errors"
	"github.com/albedosehen/rwwwrse/internal/observability"
)

// Mock logger for router tests
type mockRouterLogger struct {
	mock.Mock
}

func (m *mockRouterLogger) Debug(ctx context.Context, msg string, fields ...observability.Field) {
	args := []interface{}{ctx, msg}
	for _, field := range fields {
		args = append(args, field)
	}
	m.Called(args...)
}

func (m *mockRouterLogger) Info(ctx context.Context, msg string, fields ...observability.Field) {
	args := []interface{}{ctx, msg}
	for _, field := range fields {
		args = append(args, field)
	}
	m.Called(args...)
}

func (m *mockRouterLogger) Warn(ctx context.Context, msg string, fields ...observability.Field) {
	args := []interface{}{ctx, msg}
	for _, field := range fields {
		args = append(args, field)
	}
	m.Called(args...)
}

func (m *mockRouterLogger) Error(ctx context.Context, err error, msg string, fields ...observability.Field) {
	args := []interface{}{ctx, err, msg}
	for _, field := range fields {
		args = append(args, field)
	}
	m.Called(args...)
}

func (m *mockRouterLogger) WithFields(fields ...observability.Field) observability.Logger {
	args := []interface{}{}
	for _, field := range fields {
		args = append(args, field)
	}
	result := m.Called(args...)
	return result.Get(0).(observability.Logger)
}

func (m *mockRouterLogger) WithContext(ctx context.Context) observability.Logger {
	args := m.Called(ctx)
	return args.Get(0).(observability.Logger)
}

// Mock metrics collector for router tests
type mockRouterMetrics struct {
	mock.Mock
}

func (m *mockRouterMetrics) RecordRequest(method, host, status string, duration time.Duration) {
	m.Called(method, host, status, duration)
}

func (m *mockRouterMetrics) RecordBackendRequest(backend, status string, duration time.Duration) {
	m.Called(backend, status, duration)
}

func (m *mockRouterMetrics) RecordHealthCheck(target string, healthy bool, duration time.Duration) {
	m.Called(target, healthy, duration)
}

func (m *mockRouterMetrics) IncActiveConnections() {
	m.Called()
}

func (m *mockRouterMetrics) DecActiveConnections() {
	m.Called()
}

func (m *mockRouterMetrics) RecordCertificateRenewal(domain string, success bool) {
	m.Called(domain, success)
}

func (m *mockRouterMetrics) RecordRateLimitHit(key string) {
	m.Called(key)
}

func TestNewRouterImpl(t *testing.T) {
	tests := []struct {
		name           string
		backendsConfig config.BackendsConfig
		logger         observability.Logger
		metrics        observability.MetricsCollector
		expectErr      bool
		errType        string
	}{
		{
			name: "NewRouterImpl_ValidConfig_Success",
			backendsConfig: config.BackendsConfig{
				Routes: map[string]config.BackendRoute{
					"api.example.com": {
						URL:            "http://localhost:8080",
						Timeout:        30 * time.Second,
						DialTimeout:    5 * time.Second,
						MaxIdleConns:   10,
						MaxIdlePerHost: 5,
					},
				},
			},
			logger:    &mockRouterLogger{},
			metrics:   &mockRouterMetrics{},
			expectErr: false,
		},
		{
			name: "NewRouterImpl_NilLogger_ReturnsError",
			backendsConfig: config.BackendsConfig{
				Routes: map[string]config.BackendRoute{},
			},
			logger:    nil,
			metrics:   &mockRouterMetrics{},
			expectErr: true,
			errType:   "*errors.ProxyError",
		},
		{
			name: "NewRouterImpl_InvalidBackendURL_ReturnsError",
			backendsConfig: config.BackendsConfig{
				Routes: map[string]config.BackendRoute{
					"api.example.com": {
						URL:     "invalid-url",
						Timeout: 30 * time.Second,
					},
				},
			},
			logger:    &mockRouterLogger{},
			metrics:   &mockRouterMetrics{},
			expectErr: true,
			errType:   "*errors.ProxyError",
		},
		{
			name: "NewRouterImpl_EmptyConfig_Success",
			backendsConfig: config.BackendsConfig{
				Routes: map[string]config.BackendRoute{},
			},
			logger:    &mockRouterLogger{},
			metrics:   &mockRouterMetrics{},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			if tt.logger != nil {
				mockLogger := tt.logger.(*mockRouterLogger)
				mockLogger.On("Info", mock.Anything, mock.Anything, mock.Anything).Return()
			}

			// Act
			router, err := NewRouterImpl(tt.backendsConfig, tt.logger, tt.metrics)

			// Assert
			if tt.expectErr {
				assert.Error(t, err)
				assert.Nil(t, router)
				if tt.errType != "" {
					assert.IsType(t, &proxyerrors.ProxyError{}, err)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, router)
				assert.Implements(t, (*Router)(nil), router)
			}

			if tt.logger != nil {
				mockLogger := tt.logger.(*mockRouterLogger)
				mockLogger.AssertExpectations(t)
			}
		})
	}
}

func TestRouter_Route(t *testing.T) {
	tests := []struct {
		name          string
		host          string
		setupRouter   func() *router
		expectErr     bool
		expectBackend bool
		errCode       proxyerrors.ProxyErrorCode
	}{
		{
			name: "Route_ValidHost_ReturnsBackend",
			host: "api.example.com",
			setupRouter: func() *router {
				mockLogger := &mockRouterLogger{}
				mockMetrics := &mockRouterMetrics{}
				mockLogger.On("Debug", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()

				backend := newMockBackend("api", "http://api.example.com")
				backend.On("IsHealthy", mock.Anything).Return(true)
				backend.On("Name").Return("api")
				backend.On("URL").Return(&url.URL{Scheme: "http", Host: "api.example.com"})

				r := &router{
					backends: map[string]Backend{
						"api.example.com": backend,
					},
					logger:  mockLogger,
					metrics: mockMetrics,
				}
				return r
			},
			expectErr:     false,
			expectBackend: true,
		},
		{
			name: "Route_EmptyHost_ReturnsError",
			host: "",
			setupRouter: func() *router {
				mockLogger := &mockRouterLogger{}
				mockMetrics := &mockRouterMetrics{}
				return &router{
					backends: make(map[string]Backend),
					logger:   mockLogger,
					metrics:  mockMetrics,
				}
			},
			expectErr:     true,
			expectBackend: false,
			errCode:       proxyerrors.ErrCodeInvalidHost,
		},
		{
			name: "Route_UnknownHost_ReturnsError",
			host: "unknown.example.com",
			setupRouter: func() *router {
				mockLogger := &mockRouterLogger{}
				mockMetrics := &mockRouterMetrics{}
				mockLogger.On("Warn", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()

				return &router{
					backends: make(map[string]Backend),
					logger:   mockLogger,
					metrics:  mockMetrics,
				}
			},
			expectErr:     true,
			expectBackend: false,
			errCode:       proxyerrors.ErrCodeHostNotConfigured,
		},
		{
			name: "Route_UnhealthyBackend_ReturnsError",
			host: "api.example.com",
			setupRouter: func() *router {
				mockLogger := &mockRouterLogger{}
				mockMetrics := &mockRouterMetrics{}
				mockLogger.On("Warn", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()

				backend := newMockBackend("api", "http://api.example.com")
				backend.On("IsHealthy", mock.Anything).Return(false)
				backend.On("Name").Return("api")

				r := &router{
					backends: map[string]Backend{
						"api.example.com": backend,
					},
					logger:  mockLogger,
					metrics: mockMetrics,
				}
				return r
			},
			expectErr:     true,
			expectBackend: false,
			errCode:       proxyerrors.ErrCodeBackendUnavailable,
		},
		{
			name: "Route_HostWithPort_NormalizesAndRoutes",
			host: "api.example.com:8080",
			setupRouter: func() *router {
				mockLogger := &mockRouterLogger{}
				mockMetrics := &mockRouterMetrics{}
				mockLogger.On("Debug", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()

				backend := newMockBackend("api", "http://api.example.com")
				backend.On("IsHealthy", mock.Anything).Return(true)
				backend.On("Name").Return("api")
				backend.On("URL").Return(&url.URL{Scheme: "http", Host: "api.example.com"})

				r := &router{
					backends: map[string]Backend{
						"api.example.com": backend,
					},
					logger:  mockLogger,
					metrics: mockMetrics,
				}
				return r
			},
			expectErr:     false,
			expectBackend: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			ctx := context.Background()
			router := tt.setupRouter()

			// Act
			backend, err := router.Route(ctx, tt.host)

			// Assert
			if tt.expectErr {
				assert.Error(t, err)
				if tt.errCode != 0 {
					var proxyErr *proxyerrors.ProxyError
					require.ErrorAs(t, err, &proxyErr)
					assert.Equal(t, tt.errCode, proxyErr.Code)
				}
			} else {
				assert.NoError(t, err)
			}

			if tt.expectBackend {
				assert.NotNil(t, backend)
			} else {
				assert.Nil(t, backend)
			}

			// Verify mock expectations
			for _, backend := range router.backends {
				if mockBackend, ok := backend.(*mockBackend); ok {
					mockBackend.AssertExpectations(t)
				}
			}
			router.logger.(*mockRouterLogger).AssertExpectations(t)
		})
	}
}

func TestRouter_Register(t *testing.T) {
	tests := []struct {
		name      string
		host      string
		backend   Backend
		setupMock func(*mockRouterLogger)
		expectErr bool
		errCode   proxyerrors.ProxyErrorCode
	}{
		{
			name:    "Register_ValidHostAndBackend_Success",
			host:    "api.example.com",
			backend: newMockBackend("api", "http://api.example.com"),
			setupMock: func(logger *mockRouterLogger) {
				logger.On("Info", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()
			},
			expectErr: false,
		},
		{
			name:    "Register_EmptyHost_ReturnsError",
			host:    "",
			backend: newMockBackend("api", "http://api.example.com"),
			setupMock: func(logger *mockRouterLogger) {
				// No logging expected for validation errors
			},
			expectErr: true,
			errCode:   proxyerrors.ErrCodeConfigInvalid,
		},
		{
			name:    "Register_NilBackend_ReturnsError",
			host:    "api.example.com",
			backend: nil,
			setupMock: func(logger *mockRouterLogger) {
				// No logging expected for validation errors
			},
			expectErr: true,
			errCode:   proxyerrors.ErrCodeConfigInvalid,
		},
		{
			name:    "Register_HostWithPort_NormalizesHost",
			host:    "api.example.com:8080",
			backend: newMockBackend("api", "http://api.example.com"),
			setupMock: func(logger *mockRouterLogger) {
				logger.On("Info", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockLogger := &mockRouterLogger{}
			mockMetrics := &mockRouterMetrics{}
			tt.setupMock(mockLogger)

			if tt.backend != nil {
				backend := tt.backend.(*mockBackend)
				backend.On("Name").Return("api")
				backend.On("URL").Return(&url.URL{Scheme: "http", Host: "api.example.com"})
			}

			router := &router{
				backends: make(map[string]Backend),
				logger:   mockLogger,
				metrics:  mockMetrics,
			}

			// Act
			err := router.Register(tt.host, tt.backend)

			// Assert
			if tt.expectErr {
				assert.Error(t, err)
				if tt.errCode != 0 {
					var proxyErr *proxyerrors.ProxyError
					require.ErrorAs(t, err, &proxyErr)
					assert.Equal(t, tt.errCode, proxyErr.Code)
				}
			} else {
				assert.NoError(t, err)
				// Verify backend was registered
				normalizedHost := router.normalizeHost(tt.host)
				assert.Contains(t, router.backends, normalizedHost)
				assert.Equal(t, tt.backend, router.backends[normalizedHost])
			}

			mockLogger.AssertExpectations(t)
			if tt.backend != nil {
				tt.backend.(*mockBackend).AssertExpectations(t)
			}
		})
	}
}

func TestRouter_Unregister(t *testing.T) {
	tests := []struct {
		name         string
		host         string
		setupRouter  func() *router
		expectErr    bool
		errCode      proxyerrors.ProxyErrorCode
		expectExists bool
	}{
		{
			name: "Unregister_ExistingHost_Success",
			host: "api.example.com",
			setupRouter: func() *router {
				mockLogger := &mockRouterLogger{}
				mockLogger.On("Info", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()

				backend := newMockBackend("api", "http://api.example.com")
				backend.On("Name").Return("api")

				return &router{
					backends: map[string]Backend{
						"api.example.com": backend,
					},
					logger:  mockLogger,
					metrics: &mockRouterMetrics{},
				}
			},
			expectErr:    false,
			expectExists: false,
		},
		{
			name: "Unregister_EmptyHost_ReturnsError",
			host: "",
			setupRouter: func() *router {
				return &router{
					backends: make(map[string]Backend),
					logger:   &mockRouterLogger{},
					metrics:  &mockRouterMetrics{},
				}
			},
			expectErr: true,
			errCode:   proxyerrors.ErrCodeConfigInvalid,
		},
		{
			name: "Unregister_NonexistentHost_ReturnsError",
			host: "nonexistent.example.com",
			setupRouter: func() *router {
				return &router{
					backends: make(map[string]Backend),
					logger:   &mockRouterLogger{},
					metrics:  &mockRouterMetrics{},
				}
			},
			expectErr: true,
			errCode:   proxyerrors.ErrCodeHostNotConfigured,
		},
		{
			name: "Unregister_HostWithPort_NormalizesAndUnregisters",
			host: "api.example.com:8080",
			setupRouter: func() *router {
				mockLogger := &mockRouterLogger{}
				mockLogger.On("Info", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()

				backend := newMockBackend("api", "http://api.example.com")
				backend.On("Name").Return("api")

				return &router{
					backends: map[string]Backend{
						"api.example.com": backend,
					},
					logger:  mockLogger,
					metrics: &mockRouterMetrics{},
				}
			},
			expectErr:    false,
			expectExists: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			router := tt.setupRouter()
			normalizedHost := router.normalizeHost(tt.host)

			// Act
			err := router.Unregister(tt.host)

			// Assert
			if tt.expectErr {
				assert.Error(t, err)
				if tt.errCode != 0 {
					var proxyErr *proxyerrors.ProxyError
					require.ErrorAs(t, err, &proxyErr)
					assert.Equal(t, tt.errCode, proxyErr.Code)
				}
			} else {
				assert.NoError(t, err)
			}

			// Check if backend exists after unregister
			_, exists := router.backends[normalizedHost]
			assert.Equal(t, tt.expectExists, exists)

			// Verify mock expectations
			router.logger.(*mockRouterLogger).AssertExpectations(t)
			for _, backend := range router.backends {
				if mockBackend, ok := backend.(*mockBackend); ok {
					mockBackend.AssertExpectations(t)
				}
			}
		})
	}
}

func TestRouter_Backends(t *testing.T) {
	tests := []struct {
		name          string
		setupRouter   func() *router
		expectedCount int
		expectedHosts []string
	}{
		{
			name: "Backends_EmptyRouter_ReturnsEmptyMap",
			setupRouter: func() *router {
				return &router{
					backends: make(map[string]Backend),
					logger:   &mockRouterLogger{},
					metrics:  &mockRouterMetrics{},
				}
			},
			expectedCount: 0,
			expectedHosts: []string{},
		},
		{
			name: "Backends_MultipleBackends_ReturnsAllBackends",
			setupRouter: func() *router {
				backend1 := newMockBackend("api", "http://api.example.com")
				backend2 := newMockBackend("web", "http://web.example.com")

				return &router{
					backends: map[string]Backend{
						"api.example.com": backend1,
						"web.example.com": backend2,
					},
					logger:  &mockRouterLogger{},
					metrics: &mockRouterMetrics{},
				}
			},
			expectedCount: 2,
			expectedHosts: []string{"api.example.com", "web.example.com"},
		},
		{
			name: "Backends_SingleBackend_ReturnsSingleBackend",
			setupRouter: func() *router {
				backend := newMockBackend("api", "http://api.example.com")

				return &router{
					backends: map[string]Backend{
						"api.example.com": backend,
					},
					logger:  &mockRouterLogger{},
					metrics: &mockRouterMetrics{},
				}
			},
			expectedCount: 1,
			expectedHosts: []string{"api.example.com"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			router := tt.setupRouter()

			// Act
			backends := router.Backends()

			// Assert
			assert.Len(t, backends, tt.expectedCount)

			for _, host := range tt.expectedHosts {
				assert.Contains(t, backends, host)
				assert.NotNil(t, backends[host])
			}

			// Verify it's a copy (modifying returned map shouldn't affect original)
			if len(backends) > 0 {
				for host := range backends {
					delete(backends, host)
					break
				}
				// Original should still have all backends
				assert.Len(t, router.backends, tt.expectedCount)
			}
		})
	}
}

func TestRouter_NormalizeHost(t *testing.T) {
	tests := []struct {
		name           string
		host           string
		expectedResult string
	}{
		{
			name:           "NormalizeHost_PlainHost_ReturnsLowercase",
			host:           "Example.Com",
			expectedResult: "example.com",
		},
		{
			name:           "NormalizeHost_HostWithPort_RemovesPort",
			host:           "example.com:8080",
			expectedResult: "example.com",
		},
		{
			name:           "NormalizeHost_HostWithHTTPSPort_RemovesPort",
			host:           "example.com:443",
			expectedResult: "example.com",
		},
		{
			name:           "NormalizeHost_EmptyHost_ReturnsEmpty",
			host:           "",
			expectedResult: "",
		},
		{
			name:           "NormalizeHost_IPv4WithPort_RemovesPort",
			host:           "192.168.1.1:8080",
			expectedResult: "192.168.1.1",
		},
		{
			name:           "NormalizeHost_IPv6WithBrackets_RemovesPort",
			host:           "[::1]:8080",
			expectedResult: "[::1]",
		},
		{
			name:           "NormalizeHost_IPv6WithoutBrackets_KeepsIntact",
			host:           "2001:db8::1",
			expectedResult: "2001:db8::1",
		},
		{
			name:           "NormalizeHost_Localhost_ReturnsLowercase",
			host:           "LocalHost",
			expectedResult: "localhost",
		},
		{
			name:           "NormalizeHost_HostWithCustomPort_RemovesPort",
			host:           "api.example.com:9000",
			expectedResult: "api.example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			router := &router{
				backends: make(map[string]Backend),
				logger:   &mockRouterLogger{},
				metrics:  &mockRouterMetrics{},
			}

			// Act
			result := router.normalizeHost(tt.host)

			// Assert
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

// Benchmark tests for router operations
func BenchmarkRouterRoute(b *testing.B) {
	// Setup
	mockLogger := &mockRouterLogger{}
	mockLogger.On("Debug", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()

	backend := newMockBackend("api", "http://api.example.com")
	backend.On("IsHealthy", mock.Anything).Return(true)
	backend.On("Name").Return("api")
	backend.On("URL").Return(&url.URL{Scheme: "http", Host: "api.example.com"})

	router := &router{
		backends: map[string]Backend{
			"api.example.com": backend,
		},
		logger:  mockLogger,
		metrics: &mockRouterMetrics{},
	}

	ctx := context.Background()
	host := "api.example.com"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = router.Route(ctx, host)
	}
}

func BenchmarkRouterRegister(b *testing.B) {
	// Setup
	mockLogger := &mockRouterLogger{}
	mockLogger.On("Info", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()

	router := &router{
		backends: make(map[string]Backend),
		logger:   mockLogger,
		metrics:  &mockRouterMetrics{},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		backend := newMockBackend("api", "http://api.example.com")
		backend.On("Name").Return("api")
		backend.On("URL").Return(&url.URL{Scheme: "http", Host: "api.example.com"})

		_ = router.Register("api.example.com", backend)
	}
}

func BenchmarkRouterNormalizeHost(b *testing.B) {
	router := &router{
		backends: make(map[string]Backend),
		logger:   &mockRouterLogger{},
		metrics:  &mockRouterMetrics{},
	}

	host := "API.EXAMPLE.COM:8080"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = router.normalizeHost(host)
	}
}
