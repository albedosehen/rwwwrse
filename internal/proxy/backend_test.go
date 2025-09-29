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

func TestNewBackendImpl(t *testing.T) {
	tests := []struct {
		name        string
		backendName string
		route       config.BackendRoute
		logger      observability.Logger
		metrics     observability.MetricsCollector
		expectErr   bool
		errType     string
	}{
		{
			name:        "NewBackendImpl_ValidConfig_Success",
			backendName: "api-backend",
			route: config.BackendRoute{
				URL:            "http://api.example.com:8080",
				Timeout:        30 * time.Second,
				DialTimeout:    5 * time.Second,
				MaxIdleConns:   10,
				MaxIdlePerHost: 5,
			},
			logger:    &mockRouterLogger{},
			metrics:   &mockRouterMetrics{},
			expectErr: false,
		},
		{
			name:        "NewBackendImpl_EmptyName_ReturnsError",
			backendName: "",
			route: config.BackendRoute{
				URL: "http://api.example.com:8080",
			},
			logger:    &mockRouterLogger{},
			metrics:   &mockRouterMetrics{},
			expectErr: true,
			errType:   "*errors.ProxyError",
		},
		{
			name:        "NewBackendImpl_EmptyURL_ReturnsError",
			backendName: "api-backend",
			route: config.BackendRoute{
				URL: "",
			},
			logger:    &mockRouterLogger{},
			metrics:   &mockRouterMetrics{},
			expectErr: true,
			errType:   "*errors.ProxyError",
		},
		{
			name:        "NewBackendImpl_InvalidURL_ReturnsError",
			backendName: "api-backend",
			route: config.BackendRoute{
				URL: "invalid-url",
			},
			logger:    &mockRouterLogger{},
			metrics:   &mockRouterMetrics{},
			expectErr: true,
			errType:   "*errors.ProxyError",
		},
		{
			name:        "NewBackendImpl_NilLogger_ReturnsError",
			backendName: "api-backend",
			route: config.BackendRoute{
				URL: "http://api.example.com:8080",
			},
			logger:    nil,
			metrics:   &mockRouterMetrics{},
			expectErr: true,
			errType:   "*errors.ProxyError",
		},
		{
			name:        "NewBackendImpl_HTTPSURL_Success",
			backendName: "secure-backend",
			route: config.BackendRoute{
				URL:            "https://secure.example.com:8443",
				Timeout:        30 * time.Second,
				DialTimeout:    5 * time.Second,
				MaxIdleConns:   20,
				MaxIdlePerHost: 10,
			},
			logger:    &mockRouterLogger{},
			metrics:   &mockRouterMetrics{},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			if tt.logger != nil && !tt.expectErr {
				// Only expect logging for successful backend creation
				mockLogger := tt.logger.(*mockRouterLogger)
				mockLogger.On("Info", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()
			}

			// Act
			backend, err := NewBackendImpl(tt.backendName, tt.route, tt.logger, tt.metrics)

			// Assert
			if tt.expectErr {
				assert.Error(t, err)
				assert.Nil(t, backend)
				if tt.errType != "" {
					assert.IsType(t, &proxyerrors.ProxyError{}, err)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, backend)
				assert.Implements(t, (*Backend)(nil), backend)

				// Verify backend properties
				assert.Equal(t, tt.backendName, backend.Name())
				assert.NotNil(t, backend.URL())
				assert.NotNil(t, backend.Transport())
				assert.True(t, backend.IsHealthy(context.Background()))
			}

			if tt.logger != nil && !tt.expectErr {
				mockLogger := tt.logger.(*mockRouterLogger)
				mockLogger.AssertExpectations(t)
			}
		})
	}
}

func TestBackend_URL(t *testing.T) {
	tests := []struct {
		name        string
		backendURL  string
		expectedURL string
	}{
		{
			name:        "URL_HTTPBackend_ReturnsCorrectURL",
			backendURL:  "http://api.example.com:8080",
			expectedURL: "http://api.example.com:8080",
		},
		{
			name:        "URL_HTTPSBackend_ReturnsCorrectURL",
			backendURL:  "https://secure.example.com:8443",
			expectedURL: "https://secure.example.com:8443",
		},
		{
			name:        "URL_LocalhostBackend_ReturnsCorrectURL",
			backendURL:  "http://localhost:3000",
			expectedURL: "http://localhost:3000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockLogger := &mockRouterLogger{}
			mockLogger.On("Info", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()

			route := config.BackendRoute{
				URL:     tt.backendURL,
				Timeout: 30 * time.Second,
			}

			backend, err := NewBackendImpl("test", route, mockLogger, &mockRouterMetrics{})
			require.NoError(t, err)

			// Act
			actualURL := backend.URL()

			// Assert
			assert.Equal(t, tt.expectedURL, actualURL.String())
			mockLogger.AssertExpectations(t)
		})
	}
}

func TestBackend_Transport(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "Transport_ReturnsHTTPTransport",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockLogger := &mockRouterLogger{}
			mockLogger.On("Info", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()

			route := config.BackendRoute{
				URL:     "http://api.example.com:8080",
				Timeout: 30 * time.Second,
			}

			backend, err := NewBackendImpl("test", route, mockLogger, &mockRouterMetrics{})
			require.NoError(t, err)

			// Act
			transport := backend.Transport()

			// Assert
			assert.NotNil(t, transport)
			assert.Implements(t, (*http.RoundTripper)(nil), transport)

			// Verify it's an HTTP transport
			httpTransport, ok := transport.(*http.Transport)
			assert.True(t, ok)
			assert.NotNil(t, httpTransport)

			mockLogger.AssertExpectations(t)
		})
	}
}

func TestBackend_IsHealthy(t *testing.T) {
	tests := []struct {
		name           string
		expectedHealth bool
	}{
		{
			name:           "IsHealthy_InitiallyHealthy_ReturnsTrue",
			expectedHealth: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockLogger := &mockRouterLogger{}
			mockLogger.On("Info", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()

			route := config.BackendRoute{
				URL:     "http://api.example.com:8080",
				Timeout: 30 * time.Second,
			}

			backend, err := NewBackendImpl("test", route, mockLogger, &mockRouterMetrics{})
			require.NoError(t, err)

			ctx := context.Background()

			// Act
			isHealthy := backend.IsHealthy(ctx)

			// Assert
			assert.Equal(t, tt.expectedHealth, isHealthy)
			mockLogger.AssertExpectations(t)
		})
	}
}

func TestBackend_Name(t *testing.T) {
	tests := []struct {
		name         string
		backendName  string
		expectedName string
	}{
		{
			name:         "Name_APIBackend_ReturnsCorrectName",
			backendName:  "api-backend",
			expectedName: "api-backend",
		},
		{
			name:         "Name_WebBackend_ReturnsCorrectName",
			backendName:  "web-backend",
			expectedName: "web-backend",
		},
		{
			name:         "Name_ServiceBackend_ReturnsCorrectName",
			backendName:  "user-service",
			expectedName: "user-service",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockLogger := &mockRouterLogger{}
			mockLogger.On("Info", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()

			route := config.BackendRoute{
				URL:     "http://api.example.com:8080",
				Timeout: 30 * time.Second,
			}

			backend, err := NewBackendImpl(tt.backendName, route, mockLogger, &mockRouterMetrics{})
			require.NoError(t, err)

			// Act
			name := backend.Name()

			// Assert
			assert.Equal(t, tt.expectedName, name)
			mockLogger.AssertExpectations(t)
		})
	}
}

func TestNewBackendManagerImpl(t *testing.T) {
	tests := []struct {
		name           string
		backendsConfig config.BackendsConfig
		logger         observability.Logger
		metrics        observability.MetricsCollector
		expectErr      bool
		expectedCount  int
	}{
		{
			name: "NewBackendManagerImpl_ValidConfig_Success",
			backendsConfig: config.BackendsConfig{
				Routes: map[string]config.BackendRoute{
					"api.example.com": {
						URL:            "http://api.example.com:8080",
						Timeout:        30 * time.Second,
						DialTimeout:    5 * time.Second,
						MaxIdleConns:   10,
						MaxIdlePerHost: 5,
					},
					"web.example.com": {
						URL:            "http://web.example.com:8080",
						Timeout:        30 * time.Second,
						DialTimeout:    5 * time.Second,
						MaxIdleConns:   10,
						MaxIdlePerHost: 5,
					},
				},
			},
			logger:        &mockRouterLogger{},
			metrics:       &mockRouterMetrics{},
			expectErr:     false,
			expectedCount: 2,
		},
		{
			name: "NewBackendManagerImpl_EmptyConfig_Success",
			backendsConfig: config.BackendsConfig{
				Routes: map[string]config.BackendRoute{},
			},
			logger:        &mockRouterLogger{},
			metrics:       &mockRouterMetrics{},
			expectErr:     false,
			expectedCount: 0,
		},
		{
			name: "NewBackendManagerImpl_NilLogger_ReturnsError",
			backendsConfig: config.BackendsConfig{
				Routes: map[string]config.BackendRoute{},
			},
			logger:    nil,
			metrics:   &mockRouterMetrics{},
			expectErr: true,
		},
		{
			name: "NewBackendManagerImpl_InvalidBackendURL_ReturnsError",
			backendsConfig: config.BackendsConfig{
				Routes: map[string]config.BackendRoute{
					"api.example.com": {
						URL: "invalid-url",
					},
				},
			},
			logger:    &mockRouterLogger{},
			metrics:   &mockRouterMetrics{},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			if tt.logger != nil {
				mockLogger := tt.logger.(*mockRouterLogger)
				// Mock creation logging for each backend and manager
				for range tt.backendsConfig.Routes {
					mockLogger.On("Info", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()
				}
				// Mock manager initialization logging
				mockLogger.On("Info", mock.Anything, mock.Anything, mock.Anything).Return()
			}

			// Act
			manager, err := NewBackendManagerImpl(tt.backendsConfig, tt.logger, tt.metrics)

			// Assert
			if tt.expectErr {
				assert.Error(t, err)
				assert.Nil(t, manager)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, manager)
				assert.Implements(t, (*BackendManager)(nil), manager)

				// Verify backend count
				ctx := context.Background()
				backends, err := manager.ListBackends(ctx)
				assert.NoError(t, err)
				assert.Len(t, backends, tt.expectedCount)
			}

			if tt.logger != nil {
				mockLogger := tt.logger.(*mockRouterLogger)
				mockLogger.AssertExpectations(t)
			}
		})
	}
}

func TestBackendManager_AddBackend(t *testing.T) {
	tests := []struct {
		name      string
		host      string
		backend   Backend
		expectErr bool
		errType   string
	}{
		{
			name:      "AddBackend_ValidHostAndBackend_Success",
			host:      "new.example.com",
			backend:   newMockBackend("new", "http://new.example.com"),
			expectErr: false,
		},
		{
			name:      "AddBackend_EmptyHost_ReturnsError",
			host:      "",
			backend:   newMockBackend("new", "http://new.example.com"),
			expectErr: true,
			errType:   "*errors.ProxyError",
		},
		{
			name:      "AddBackend_NilBackend_ReturnsError",
			host:      "new.example.com",
			backend:   nil,
			expectErr: true,
			errType:   "*errors.ProxyError",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockLogger := &mockRouterLogger{}
			mockLogger.On("Info", mock.Anything, mock.Anything, mock.Anything).Return()

			if !tt.expectErr {
				mockLogger.On("Info", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()
				if tt.backend != nil {
					backend := tt.backend.(*mockBackend)
					backend.On("Name").Return("new")
					backend.On("URL").Return(&url.URL{Scheme: "http", Host: "new.example.com"})
				}
			}

			manager, err := NewBackendManagerImpl(config.BackendsConfig{Routes: map[string]config.BackendRoute{}}, mockLogger, &mockRouterMetrics{})
			require.NoError(t, err)

			ctx := context.Background()

			// Act
			err = manager.AddBackend(ctx, tt.host, tt.backend)

			// Assert
			if tt.expectErr {
				assert.Error(t, err)
				if tt.errType != "" {
					assert.IsType(t, &proxyerrors.ProxyError{}, err)
				}
			} else {
				assert.NoError(t, err)

				// Verify backend was added
				backend, err := manager.GetBackend(ctx, tt.host)
				assert.NoError(t, err)
				assert.Equal(t, tt.backend, backend)
			}

			mockLogger.AssertExpectations(t)
			if tt.backend != nil {
				tt.backend.(*mockBackend).AssertExpectations(t)
			}
		})
	}
}

func TestBackendManager_RemoveBackend(t *testing.T) {
	tests := []struct {
		name      string
		host      string
		expectErr bool
		errType   string
	}{
		{
			name:      "RemoveBackend_ExistingHost_Success",
			host:      "api.example.com",
			expectErr: false,
		},
		{
			name:      "RemoveBackend_EmptyHost_ReturnsError",
			host:      "",
			expectErr: true,
			errType:   "*errors.ProxyError",
		},
		{
			name:      "RemoveBackend_NonexistentHost_ReturnsError",
			host:      "nonexistent.example.com",
			expectErr: true,
			errType:   "*errors.ProxyError",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockLogger := &mockRouterLogger{}

			// Setup initial backend if needed
			backendsConfig := config.BackendsConfig{Routes: map[string]config.BackendRoute{}}
			if tt.host == "api.example.com" {
				backendsConfig.Routes[tt.host] = config.BackendRoute{
					URL:     "http://api.example.com:8080",
					Timeout: 30 * time.Second,
				}
				mockLogger.On("Info", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()
			}

			mockLogger.On("Info", mock.Anything, mock.Anything, mock.Anything).Return()

			if !tt.expectErr && tt.host != "" {
				mockLogger.On("Info", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()
			}

			manager, err := NewBackendManagerImpl(backendsConfig, mockLogger, &mockRouterMetrics{})
			require.NoError(t, err)

			ctx := context.Background()

			// Act
			err = manager.RemoveBackend(ctx, tt.host)

			// Assert
			if tt.expectErr {
				assert.Error(t, err)
				if tt.errType != "" {
					assert.IsType(t, &proxyerrors.ProxyError{}, err)
				}
			} else {
				assert.NoError(t, err)

				// Verify backend was removed
				_, err := manager.GetBackend(ctx, tt.host)
				assert.Error(t, err)
			}

			mockLogger.AssertExpectations(t)
		})
	}
}

func TestBackendManager_GetBackend(t *testing.T) {
	tests := []struct {
		name      string
		host      string
		expectErr bool
		errType   string
	}{
		{
			name:      "GetBackend_ExistingHost_ReturnsBackend",
			host:      "api.example.com",
			expectErr: false,
		},
		{
			name:      "GetBackend_EmptyHost_ReturnsError",
			host:      "",
			expectErr: true,
			errType:   "*errors.ProxyError",
		},
		{
			name:      "GetBackend_NonexistentHost_ReturnsError",
			host:      "nonexistent.example.com",
			expectErr: true,
			errType:   "*errors.ProxyError",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockLogger := &mockRouterLogger{}

			// Setup initial backend if needed
			backendsConfig := config.BackendsConfig{Routes: map[string]config.BackendRoute{}}
			if tt.host == "api.example.com" {
				backendsConfig.Routes[tt.host] = config.BackendRoute{
					URL:     "http://api.example.com:8080",
					Timeout: 30 * time.Second,
				}
				mockLogger.On("Info", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()
			}

			mockLogger.On("Info", mock.Anything, mock.Anything, mock.Anything).Return()

			manager, err := NewBackendManagerImpl(backendsConfig, mockLogger, &mockRouterMetrics{})
			require.NoError(t, err)

			ctx := context.Background()

			// Act
			backend, err := manager.GetBackend(ctx, tt.host)

			// Assert
			if tt.expectErr {
				assert.Error(t, err)
				assert.Nil(t, backend)
				if tt.errType != "" {
					assert.IsType(t, &proxyerrors.ProxyError{}, err)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, backend)
				assert.Equal(t, tt.host, backend.Name())
			}

			mockLogger.AssertExpectations(t)
		})
	}
}

// Benchmark tests for backend operations
func BenchmarkBackendImpl_IsHealthy(b *testing.B) {
	// Setup
	mockLogger := &mockRouterLogger{}
	mockLogger.On("Info", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()

	route := config.BackendRoute{
		URL:     "http://api.example.com:8080",
		Timeout: 30 * time.Second,
	}

	backend, err := NewBackendImpl("test", route, mockLogger, &mockRouterMetrics{})
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = backend.IsHealthy(ctx)
	}
}

func BenchmarkBackendManager_GetBackend(b *testing.B) {
	// Setup
	mockLogger := &mockRouterLogger{}
	mockLogger.On("Info", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()
	mockLogger.On("Info", mock.Anything, mock.Anything, mock.Anything).Return()

	backendsConfig := config.BackendsConfig{
		Routes: map[string]config.BackendRoute{
			"api.example.com": {
				URL:     "http://api.example.com:8080",
				Timeout: 30 * time.Second,
			},
		},
	}

	manager, err := NewBackendManagerImpl(backendsConfig, mockLogger, &mockRouterMetrics{})
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.Background()
	host := "api.example.com"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = manager.GetBackend(ctx, host)
	}
}
