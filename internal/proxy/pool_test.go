package proxy

import (
	"context"
	"net/http"
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

func TestNewConnectionPoolImpl(t *testing.T) {
	tests := []struct {
		name           string
		backendsConfig config.BackendsConfig
		logger         observability.Logger
		metrics        observability.MetricsCollector
		expectErr      bool
		errType        string
	}{
		{
			name: "NewConnectionPoolImpl_ValidConfig_Success",
			backendsConfig: config.BackendsConfig{
				Routes: map[string]config.BackendRoute{
					"api.example.com": {
						URL:     "http://api.example.com:8080",
						Timeout: 30 * time.Second,
					},
				},
			},
			logger:    &mockPoolLogger{},
			metrics:   &mockMetricsCollector{},
			expectErr: false,
		},
		{
			name: "NewConnectionPoolImpl_EmptyConfig_Success",
			backendsConfig: config.BackendsConfig{
				Routes: map[string]config.BackendRoute{},
			},
			logger:    &mockPoolLogger{},
			metrics:   &mockMetricsCollector{},
			expectErr: false,
		},
		{
			name: "NewConnectionPoolImpl_NilLogger_ReturnsError",
			backendsConfig: config.BackendsConfig{
				Routes: map[string]config.BackendRoute{},
			},
			logger:    nil,
			metrics:   &mockMetricsCollector{},
			expectErr: true,
			errType:   "*errors.ProxyError",
		},
		{
			name: "NewConnectionPoolImpl_NilMetrics_ReturnsError",
			backendsConfig: config.BackendsConfig{
				Routes: map[string]config.BackendRoute{},
			},
			logger:    &mockPoolLogger{},
			metrics:   nil,
			expectErr: true,
			errType:   "*errors.ProxyError",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			if tt.logger != nil {
				mockLogger := tt.logger.(*mockPoolLogger)
				mockLogger.On("Info", mock.Anything, mock.Anything, mock.Anything).Return()
			}

			// Act
			pool, err := NewConnectionPoolImpl(tt.backendsConfig, tt.logger, tt.metrics)

			// Assert
			if tt.expectErr {
				assert.Error(t, err)
				assert.Nil(t, pool)
				if tt.errType != "" {
					assert.IsType(t, &proxyerrors.ProxyError{}, err)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, pool)
				assert.Implements(t, (*ConnectionPool)(nil), pool)
			}

			if tt.logger != nil {
				mockLogger := tt.logger.(*mockPoolLogger)
				mockLogger.AssertExpectations(t)
			}
		})
	}
}

func TestConnectionPool_GetConnection(t *testing.T) {
	tests := []struct {
		name      string
		backend   Backend
		expectErr bool
		errType   string
	}{
		{
			name:      "GetConnection_ValidBackend_ReturnsTransport",
			backend:   newMockBackend("api", "http://api.example.com"),
			expectErr: false,
		},
		{
			name:      "GetConnection_NilBackend_ReturnsError",
			backend:   nil,
			expectErr: true,
			errType:   "*errors.ProxyError",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockLogger := &mockPoolLogger{}
			mockLogger.On("Info", mock.Anything, mock.Anything, mock.Anything).Return()

			mockMetrics := &mockMetricsCollector{}
			if tt.backend != nil {
				backend := tt.backend.(*mockBackend)
				backend.On("Name").Return("api")
				backend.On("Transport").Return(http.DefaultTransport)
				backend.On("URL").Return(&url.URL{Scheme: "http", Host: "api.example.com"})
				mockLogger.On("Debug", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()
				mockLogger.On("Error", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()
				mockMetrics.On("IncActiveConnections").Return()
			} else {
				mockLogger.On("Error", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()
			}

			pool, err := NewConnectionPoolImpl(config.BackendsConfig{Routes: map[string]config.BackendRoute{}}, mockLogger, &mockMetricsCollector{})
			require.NoError(t, err)

			ctx := context.Background()

			// Act
			transport, err := pool.GetConnection(ctx, tt.backend)

			// Assert
			if tt.expectErr {
				assert.Error(t, err)
				assert.Nil(t, transport)
				if tt.errType != "" {
					assert.IsType(t, &proxyerrors.ProxyError{}, err)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, transport)
				assert.Implements(t, (*http.RoundTripper)(nil), transport)
			}

			mockLogger.AssertExpectations(t)
			if tt.backend != nil {
				tt.backend.(*mockBackend).AssertExpectations(t)
			}
		})
	}
}

func TestConnectionPool_GetConnection_Caching(t *testing.T) {
	t.Run("GetConnection_SameBackend_ReturnsCachedTransport", func(t *testing.T) {
		// Arrange
		mockLogger := &mockPoolLogger{}
		mockLogger.On("Info", mock.Anything, mock.Anything, mock.Anything).Return()
		mockLogger.On("Debug", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()
		mockMetrics := &mockMetricsCollector{}
		mockMetrics.On("IncActiveConnections").Return()

		backend := newMockBackend("api", "http://api.example.com")
		backend.On("Name").Return("api")
		backend.On("Transport").Return(http.DefaultTransport)
		backend.On("URL").Return(&url.URL{Scheme: "http", Host: "api.example.com"})

		pool, err := NewConnectionPoolImpl(config.BackendsConfig{Routes: map[string]config.BackendRoute{}}, mockLogger, &mockMetricsCollector{})
		require.NoError(t, err)

		ctx := context.Background()

		// Act - First call
		transport1, err1 := pool.GetConnection(ctx, backend)
		require.NoError(t, err1)

		// Act - Second call (should return cached)
		transport2, err2 := pool.GetConnection(ctx, backend)
		require.NoError(t, err2)

		// Assert
		assert.Equal(t, transport1, transport2)

		mockLogger.AssertExpectations(t)
		backend.AssertExpectations(t)
	})
}

func TestConnectionPool_ReleaseConnection(t *testing.T) {
	tests := []struct {
		name      string
		backend   Backend
		transport http.RoundTripper
		expectErr bool
		errType   string
	}{
		{
			name:      "ReleaseConnection_ValidBackend_Success",
			backend:   newMockBackend("api", "http://api.example.com"),
			transport: http.DefaultTransport,
			expectErr: false,
		},
		{
			name:      "ReleaseConnection_NilBackend_ReturnsError",
			backend:   nil,
			transport: http.DefaultTransport,
			expectErr: true,
			errType:   "*errors.ProxyError",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockLogger := &mockPoolLogger{}
			mockLogger.On("Info", mock.Anything, mock.Anything, mock.Anything).Return()

			pool, err := NewConnectionPoolImpl(config.BackendsConfig{Routes: map[string]config.BackendRoute{}}, mockLogger, &mockMetricsCollector{})
			require.NoError(t, err)

			// Act
			err = pool.ReleaseConnection(tt.backend, tt.transport)

			// Assert
			if tt.expectErr {
				assert.Error(t, err)
				if tt.errType != "" {
					assert.IsType(t, &proxyerrors.ProxyError{}, err)
				}
			} else {
				assert.NoError(t, err)
			}

			mockLogger.AssertExpectations(t)
		})
	}
}

func TestConnectionPool_ReleaseConnection_DecrementsActiveConnections(t *testing.T) {
	t.Run("ReleaseConnection_AfterGetConnection_DecrementsActiveConnections", func(t *testing.T) {
		// Arrange
		mockLogger := &mockPoolLogger{}
		mockLogger.On("Info", mock.Anything, mock.Anything, mock.Anything).Return()
		mockLogger.On("Debug", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()

		backend := newMockBackend("api", "http://api.example.com")
		backend.On("Name").Return("api")
		backend.On("Transport").Return(&http.Transport{})
		backend.On("URL").Return(&url.URL{Scheme: "http", Host: "api.example.com"})

		pool, err := NewConnectionPoolImpl(config.BackendsConfig{Routes: map[string]config.BackendRoute{}}, mockLogger, &mockMetricsCollector{})
		require.NoError(t, err)

		ctx := context.Background()

		// Act - Get a connection (should increment active connections)
		transport, err := pool.GetConnection(ctx, backend)
		require.NoError(t, err)
		require.NotNil(t, transport)

		// Verify active connections increased
		stats := pool.Stats()
		assert.Equal(t, 1, stats.ActiveConnections)

		// Act - Release the connection (should decrement active connections)
		err = pool.ReleaseConnection(backend, transport)
		require.NoError(t, err)

		// Assert - Verify active connections decreased
		stats = pool.Stats()
		assert.Equal(t, 0, stats.ActiveConnections)

		mockLogger.AssertExpectations(t)
		backend.AssertExpectations(t)
	})

	t.Run("ReleaseConnection_MultipleConnections_DecrementsCorrectly", func(t *testing.T) {
		// Arrange
		mockLogger := &mockPoolLogger{}
		mockLogger.On("Info", mock.Anything, mock.Anything, mock.Anything).Return()
		mockLogger.On("Debug", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()

		backend1 := newMockBackend("api1", "http://api1.example.com")
		backend1.On("Name").Return("api1")
		backend1.On("Transport").Return(&http.Transport{})
		backend1.On("URL").Return(&url.URL{Scheme: "http", Host: "api1.example.com"})

		backend2 := newMockBackend("api2", "http://api2.example.com")
		backend2.On("Name").Return("api2")
		backend2.On("Transport").Return(&http.Transport{})
		backend2.On("URL").Return(&url.URL{Scheme: "http", Host: "api2.example.com"})

		pool, err := NewConnectionPoolImpl(config.BackendsConfig{Routes: map[string]config.BackendRoute{}}, mockLogger, &mockMetricsCollector{})
		require.NoError(t, err)

		ctx := context.Background()

		// Act - Get multiple connections
		transport1, err := pool.GetConnection(ctx, backend1)
		require.NoError(t, err)
		transport2, err := pool.GetConnection(ctx, backend2)
		require.NoError(t, err)

		// Verify active connections
		stats := pool.Stats()
		assert.Equal(t, 2, stats.ActiveConnections)

		// Act - Release one connection
		err = pool.ReleaseConnection(backend1, transport1)
		require.NoError(t, err)

		// Assert - Verify active connections decreased by 1
		stats = pool.Stats()
		assert.Equal(t, 1, stats.ActiveConnections)

		// Act - Release second connection
		err = pool.ReleaseConnection(backend2, transport2)
		require.NoError(t, err)

		// Assert - Verify all active connections released
		stats = pool.Stats()
		assert.Equal(t, 0, stats.ActiveConnections)

		mockLogger.AssertExpectations(t)
		backend1.AssertExpectations(t)
		backend2.AssertExpectations(t)
	})
}

func TestConnectionPool_Close(t *testing.T) {
	t.Run("Close_WithConnections_Success", func(t *testing.T) {
		// Arrange
		mockLogger := &mockPoolLogger{}
		mockLogger.On("Info", mock.Anything, mock.Anything, mock.Anything).Return()
		mockLogger.On("Debug", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()
		mockLogger.On("Info", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()

		backend := newMockBackend("api", "http://api.example.com")
		backend.On("Name").Return("api")
		backend.On("Transport").Return(&http.Transport{})
		backend.On("URL").Return(&url.URL{Scheme: "http", Host: "api.example.com"})

		pool, err := NewConnectionPoolImpl(config.BackendsConfig{Routes: map[string]config.BackendRoute{}}, mockLogger, &mockMetricsCollector{})
		require.NoError(t, err)

		// Add a connection to the pool
		ctx := context.Background()
		_, err = pool.GetConnection(ctx, backend)
		require.NoError(t, err)

		// Act
		err = pool.Close()

		// Assert
		assert.NoError(t, err)

		// Verify pool is empty after close
		stats := pool.Stats()
		assert.Equal(t, 0, stats.TotalConnections)

		mockLogger.AssertExpectations(t)
		backend.AssertExpectations(t)
	})
}

func TestConnectionPool_Stats(t *testing.T) {
	t.Run("Stats_EmptyPool_ReturnsZeroStats", func(t *testing.T) {
		// Arrange
		mockLogger := &mockPoolLogger{}
		mockLogger.On("Info", mock.Anything, mock.Anything, mock.Anything).Return()

		pool, err := NewConnectionPoolImpl(config.BackendsConfig{Routes: map[string]config.BackendRoute{}}, mockLogger, &mockMetricsCollector{})
		require.NoError(t, err)

		// Act
		stats := pool.Stats()

		// Assert
		assert.Equal(t, 0, stats.TotalConnections)

		mockLogger.AssertExpectations(t)
	})

	t.Run("Stats_WithConnections_ReturnsCorrectCount", func(t *testing.T) {
		// Arrange
		mockLogger := &mockPoolLogger{}
		mockLogger.On("Info", mock.Anything, mock.Anything, mock.Anything).Return()
		mockLogger.On("Debug", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()
		mockMetrics := &mockMetricsCollector{}
		mockMetrics.On("IncActiveConnections").Return()

		backend1 := newMockBackend("api1", "http://api1.example.com")
		backend1.On("Name").Return("api1")
		backend1.On("Transport").Return(http.DefaultTransport)
		backend1.On("URL").Return(&url.URL{Scheme: "http", Host: "api1.example.com"})

		backend2 := newMockBackend("api2", "http://api2.example.com")
		backend2.On("Name").Return("api2")
		backend2.On("Transport").Return(http.DefaultTransport)
		backend2.On("URL").Return(&url.URL{Scheme: "http", Host: "api2.example.com"})

		pool, err := NewConnectionPoolImpl(config.BackendsConfig{Routes: map[string]config.BackendRoute{}}, mockLogger, &mockMetricsCollector{})
		require.NoError(t, err)

		ctx := context.Background()

		// Add connections to the pool
		_, err = pool.GetConnection(ctx, backend1)
		require.NoError(t, err)
		_, err = pool.GetConnection(ctx, backend2)
		require.NoError(t, err)

		// Act
		stats := pool.Stats()

		// Assert
		assert.Equal(t, 2, stats.TotalConnections)

		mockLogger.AssertExpectations(t)
		backend1.AssertExpectations(t)
		backend2.AssertExpectations(t)
	})
}

// Mock types for pool tests
type mockPoolLogger struct {
	mock.Mock
}

func (m *mockPoolLogger) Debug(ctx context.Context, msg string, fields ...observability.Field) {
	args := []interface{}{ctx, msg}
	for _, field := range fields {
		args = append(args, field)
	}
	m.Called(args...)
}

func (m *mockPoolLogger) Info(ctx context.Context, msg string, fields ...observability.Field) {
	args := []interface{}{ctx, msg}
	for _, field := range fields {
		args = append(args, field)
	}
	m.Called(args...)
}

func (m *mockPoolLogger) Warn(ctx context.Context, msg string, fields ...observability.Field) {
	args := []interface{}{ctx, msg}
	for _, field := range fields {
		args = append(args, field)
	}
	m.Called(args...)
}

func (m *mockPoolLogger) Error(ctx context.Context, err error, msg string, fields ...observability.Field) {
	args := []interface{}{ctx, err, msg}
	for _, field := range fields {
		args = append(args, field)
	}
	m.Called(args...)
}

func (m *mockPoolLogger) WithFields(fields ...observability.Field) observability.Logger {
	args := []interface{}{}
	for _, field := range fields {
		args = append(args, field)
	}
	ret := m.Called(args...)
	return ret.Get(0).(observability.Logger)
}

func (m *mockPoolLogger) WithContext(ctx context.Context) observability.Logger {
	ret := m.Called(ctx)
	return ret.Get(0).(observability.Logger)
}

// mockMetricsCollector implements observability.MetricsCollector for testing
type mockMetricsCollector struct {
	mock.Mock
}

func (m *mockMetricsCollector) RecordRequest(method, host, status string, duration time.Duration) {
	m.Called(method, host, status, duration)
}

func (m *mockMetricsCollector) RecordBackendRequest(backend, status string, duration time.Duration) {
	m.Called(backend, status, duration)
}

func (m *mockMetricsCollector) IncActiveConnections() {
	m.Called()
}

func (m *mockMetricsCollector) DecActiveConnections() {
	m.Called()
}

func (m *mockMetricsCollector) RecordCertificateRenewal(domain string, success bool) {
	m.Called(domain, success)
}

func (m *mockMetricsCollector) RecordRateLimitHit(key string) {
	m.Called(key)
}

func (m *mockMetricsCollector) RecordHealthCheck(target string, success bool, duration time.Duration) {
	m.Called(target, success, duration)
}

// Benchmark tests for connection pool operations
func BenchmarkConnectionPool_GetConnection(b *testing.B) {
	// Setup
	mockLogger := &mockPoolLogger{}
	mockLogger.On("Info", mock.Anything, mock.Anything, mock.Anything).Return()
	mockLogger.On("Debug", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()
	mockMetrics := &mockMetricsCollector{}
	mockMetrics.On("IncActiveConnections").Return()

	backend := newMockBackend("api", "http://api.example.com")
	backend.On("Name").Return("api")
	backend.On("Transport").Return(http.DefaultTransport)
	backend.On("URL").Return(&url.URL{Scheme: "http", Host: "api.example.com"})

	pool, err := NewConnectionPoolImpl(config.BackendsConfig{Routes: map[string]config.BackendRoute{}}, mockLogger, &mockMetricsCollector{})
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = pool.GetConnection(ctx, backend)
	}
}

func BenchmarkConnectionPool_Stats(b *testing.B) {
	// Setup
	mockLogger := &mockPoolLogger{}
	mockLogger.On("Info", mock.Anything, mock.Anything, mock.Anything).Return()

	pool, err := NewConnectionPoolImpl(config.BackendsConfig{Routes: map[string]config.BackendRoute{}}, mockLogger, &mockMetricsCollector{})
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = pool.Stats()
	}
}
