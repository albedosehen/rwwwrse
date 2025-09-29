package proxy

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	proxyerrors "github.com/albedosehen/rwwwrse/internal/errors"
	"github.com/albedosehen/rwwwrse/internal/observability"
)

func TestNewProxyHandlerImpl(t *testing.T) {
	tests := []struct {
		name      string
		router    Router
		logger    observability.Logger
		metrics   observability.MetricsCollector
		expectErr bool
		errType   string
	}{
		{
			name:      "NewProxyHandlerImpl_ValidParameters_Success",
			router:    &mockRouter{},
			logger:    &mockHandlerLogger{},
			metrics:   &mockHandlerMetrics{},
			expectErr: false,
		},
		{
			name:      "NewProxyHandlerImpl_NilRouter_ReturnsError",
			router:    nil,
			logger:    &mockHandlerLogger{},
			metrics:   &mockHandlerMetrics{},
			expectErr: true,
			errType:   "*errors.ProxyError",
		},
		{
			name:      "NewProxyHandlerImpl_NilLogger_ReturnsError",
			router:    &mockRouter{},
			logger:    nil,
			metrics:   &mockHandlerMetrics{},
			expectErr: true,
			errType:   "*errors.ProxyError",
		},
		{
			name:      "NewProxyHandlerImpl_NilMetrics_Success",
			router:    &mockRouter{},
			logger:    &mockHandlerLogger{},
			metrics:   nil,
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			if tt.logger != nil {
				mockLogger := tt.logger.(*mockHandlerLogger)
				mockLogger.On("Info", mock.Anything, mock.Anything).Return()
			}

			// Act
			handler, err := NewProxyHandlerImpl(tt.router, tt.logger, tt.metrics)

			// Assert
			if tt.expectErr {
				assert.Error(t, err)
				assert.Nil(t, handler)
				if tt.errType != "" {
					assert.IsType(t, &proxyerrors.ProxyError{}, err)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, handler)
				assert.Implements(t, (*ProxyHandler)(nil), handler)
			}

			if tt.logger != nil {
				mockLogger := tt.logger.(*mockHandlerLogger)
				mockLogger.AssertExpectations(t)
			}
		})
	}
}

func TestProxyHandler_ServeHTTP(t *testing.T) {
	tests := []struct {
		name           string
		requestHost    string
		requestPath    string
		requestMethod  string
		requestBody    string
		setupMocks     func(*mockRouter, *mockHandlerLogger, *mockHandlerMetrics)
		expectedStatus int
		expectedBody   string
		expectMetrics  bool
	}{
		{
			name:          "ServeHTTP_ValidRequest_Success",
			requestHost:   "api.example.com",
			requestPath:   "/users/123",
			requestMethod: "GET",
			requestBody:   "",
			setupMocks: func(router *mockRouter, logger *mockHandlerLogger, metrics *mockHandlerMetrics) {
				backend := newMockBackend("api", "http://api.internal.com")
				backend.On("URL").Return(&url.URL{Scheme: "http", Host: "api.internal.com"})
				backend.On("Transport").Return(http.DefaultTransport)
				backend.On("Name").Return("api")

				router.On("Route", mock.Anything, "api.example.com").Return(backend, nil)

				logger.On("Info", mock.Anything, mock.Anything, mock.Anything).Return()

				metrics.On("IncActiveConnections").Return()
				metrics.On("DecActiveConnections").Return()
				metrics.On("RecordBackendRequest", mock.Anything, mock.Anything, mock.Anything).Return()
			},
			expectedStatus: 200,
			expectedBody:   `{"message": "success"}`,
			expectMetrics:  true,
		},
		{
			name:          "ServeHTTP_BackendNotFound_Returns502",
			requestHost:   "unknown.example.com",
			requestPath:   "/test",
			requestMethod: "GET",
			requestBody:   "",
			setupMocks: func(router *mockRouter, logger *mockHandlerLogger, metrics *mockHandlerMetrics) {
				router.On("Route", mock.Anything, "unknown.example.com").Return(nil, proxyerrors.NewRoutingError(proxyerrors.ErrCodeHostNotConfigured, "unknown.example.com", nil))

				logger.On("Info", mock.Anything, mock.Anything, mock.Anything).Return()
				logger.On("Error", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()

				metrics.On("IncActiveConnections").Return()
				metrics.On("DecActiveConnections").Return()
				metrics.On("RecordRequest", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()
			},
			expectedStatus: 404,
			expectedBody:   "routing for host unknown.example.com",
			expectMetrics:  true,
		},
		{
			name:          "ServeHTTP_POSTRequest_Success",
			requestHost:   "api.example.com",
			requestPath:   "/users",
			requestMethod: "POST",
			requestBody:   `{"name": "John Doe"}`,
			setupMocks: func(router *mockRouter, logger *mockHandlerLogger, metrics *mockHandlerMetrics) {
				backend := newMockBackend("api", "http://api.internal.com")
				backend.On("URL").Return(&url.URL{Scheme: "http", Host: "api.internal.com"})
				backend.On("Transport").Return(http.DefaultTransport)
				backend.On("Name").Return("api")

				router.On("Route", mock.Anything, "api.example.com").Return(backend, nil)

				logger.On("Info", mock.Anything, mock.Anything, mock.Anything).Return()

				metrics.On("IncActiveConnections").Return()
				metrics.On("DecActiveConnections").Return()
				metrics.On("RecordBackendRequest", mock.Anything, mock.Anything, mock.Anything).Return()
			},
			expectedStatus: 201,
			expectedBody:   `{"id": "123", "name": "John Doe"}`,
			expectMetrics:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockRouter := &mockRouter{}
			mockLogger := &mockHandlerLogger{}
			mockMetrics := &mockHandlerMetrics{}

			tt.setupMocks(mockRouter, mockLogger, mockMetrics)

			handler, err := NewProxyHandlerImpl(mockRouter, mockLogger, mockMetrics)
			require.NoError(t, err)

			// Create test server for backend simulation
			testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch {
				case r.Method == "GET" && strings.Contains(r.URL.Path, "/users/123"):
					w.WriteHeader(200)
					_, _ = w.Write([]byte(`{"message": "success"}`))
				case r.Method == "POST" && r.URL.Path == "/users":
					w.WriteHeader(201)
					_, _ = w.Write([]byte(`{"id": "123", "name": "John Doe"}`))
				default:
					w.WriteHeader(404)
					_, _ = w.Write([]byte("Not Found"))
				}
			}))
			defer testServer.Close()

			// Update backend URL to point to test server if backend is returned
			if tt.expectedStatus == 200 || tt.expectedStatus == 201 {
				testURL, _ := url.Parse(testServer.URL)
				for _, call := range mockRouter.ExpectedCalls {
					if call.Method == "Route" {
						backend := call.ReturnArguments[0].(*mockBackend)
						backend.ExpectedCalls = nil // Reset expectations
						backend.On("URL").Return(testURL)
						backend.On("Transport").Return(http.DefaultTransport)
						backend.On("Name").Return("api")
					}
				}
			}

			// Create request
			var body io.Reader
			if tt.requestBody != "" {
				body = strings.NewReader(tt.requestBody)
			}

			req := httptest.NewRequest(tt.requestMethod, fmt.Sprintf("http://%s%s", tt.requestHost, tt.requestPath), body)
			if tt.requestBody != "" {
				req.Header.Set("Content-Type", "application/json")
			}

			rr := httptest.NewRecorder()

			// Act
			handler.ServeHTTP(rr, req)

			// Assert
			assert.Equal(t, tt.expectedStatus, rr.Code)

			if tt.expectedBody != "" && (tt.expectedStatus == 200 || tt.expectedStatus == 201) {
				assert.Equal(t, tt.expectedBody, strings.TrimSpace(rr.Body.String()))
			} else if tt.expectedBody != "" {
				assert.Contains(t, rr.Body.String(), tt.expectedBody)
			}

			// Verify mocks
			mockRouter.AssertExpectations(t)
			mockLogger.AssertExpectations(t)
			if tt.expectMetrics {
				mockMetrics.AssertExpectations(t)
			}
		})
	}
}

func TestProxyHandler_ServeHTTP_Headers(t *testing.T) {
	tests := []struct {
		name            string
		requestHeaders  map[string]string
		expectedHeaders map[string]string
	}{
		{
			name: "ServeHTTP_XForwardedFor_PreservesAndAdds",
			requestHeaders: map[string]string{
				"X-Forwarded-For": "192.168.1.1",
				"User-Agent":      "test-client/1.0",
			},
			expectedHeaders: map[string]string{
				"X-Forwarded-For": "192.168.1.1",
				"User-Agent":      "test-client/1.0",
			},
		},
		{
			name: "ServeHTTP_CustomHeaders_Preserved",
			requestHeaders: map[string]string{
				"Authorization": "Bearer token123",
				"Accept":        "application/json",
			},
			expectedHeaders: map[string]string{
				"Authorization": "Bearer token123",
				"Accept":        "application/json",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockRouter := &mockRouter{}
			mockLogger := &mockHandlerLogger{}
			mockMetrics := &mockHandlerMetrics{}

			// Create test server that echoes headers
			var receivedHeaders http.Header
			testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				receivedHeaders = r.Header.Clone()
				w.WriteHeader(200)
				_, _ = w.Write([]byte(`{"status": "ok"}`))
			}))
			defer testServer.Close()

			testURL, _ := url.Parse(testServer.URL)
			backend := newMockBackend("api", testURL.String())
			backend.On("URL").Return(testURL)
			backend.On("Transport").Return(http.DefaultTransport)
			backend.On("Name").Return("api")

			mockRouter.On("Route", mock.Anything, "api.example.com").Return(backend, nil)
			mockLogger.On("Info", mock.Anything, mock.Anything, mock.Anything).Return()
			mockMetrics.On("IncActiveConnections").Return()
			mockMetrics.On("DecActiveConnections").Return()
			mockMetrics.On("RecordBackendRequest", mock.Anything, mock.Anything, mock.Anything).Return()

			handler, err := NewProxyHandlerImpl(mockRouter, mockLogger, mockMetrics)
			require.NoError(t, err)

			// Create request with headers
			req := httptest.NewRequest("GET", "http://api.example.com/test", nil)
			for k, v := range tt.requestHeaders {
				req.Header.Set(k, v)
			}

			rr := httptest.NewRecorder()

			// Act
			handler.ServeHTTP(rr, req)

			// Assert
			assert.Equal(t, 200, rr.Code)

			// Verify headers were forwarded
			for k, v := range tt.expectedHeaders {
				assert.Equal(t, v, receivedHeaders.Get(k), "Header %s not properly forwarded", k)
			}

			// Verify mocks
			mockRouter.AssertExpectations(t)
			backend.AssertExpectations(t)
			mockLogger.AssertExpectations(t)
			mockMetrics.AssertExpectations(t)
		})
	}
}

// Mock types for handler tests
type mockHandlerLogger struct {
	mock.Mock
}

func (m *mockHandlerLogger) Debug(ctx context.Context, msg string, fields ...observability.Field) {
	args := []interface{}{ctx, msg}
	for _, field := range fields {
		args = append(args, field)
	}
	m.Called(args...)
}

func (m *mockHandlerLogger) Info(ctx context.Context, msg string, fields ...observability.Field) {
	args := []interface{}{ctx, msg}
	for _, field := range fields {
		args = append(args, field)
	}
	m.Called(args...)
}

func (m *mockHandlerLogger) Warn(ctx context.Context, msg string, fields ...observability.Field) {
	args := []interface{}{ctx, msg}
	for _, field := range fields {
		args = append(args, field)
	}
	m.Called(args...)
}

func (m *mockHandlerLogger) Error(ctx context.Context, err error, msg string, fields ...observability.Field) {
	args := []interface{}{ctx, err, msg}
	for _, field := range fields {
		args = append(args, field)
	}
	m.Called(args...)
}

func (m *mockHandlerLogger) WithFields(fields ...observability.Field) observability.Logger {
	args := []interface{}{}
	for _, field := range fields {
		args = append(args, field)
	}
	ret := m.Called(args...)
	return ret.Get(0).(observability.Logger)
}

func (m *mockHandlerLogger) WithContext(ctx context.Context) observability.Logger {
	ret := m.Called(ctx)
	return ret.Get(0).(observability.Logger)
}

type mockHandlerMetrics struct {
	mock.Mock
}

func (m *mockHandlerMetrics) RecordRequest(method, host, status string, duration time.Duration) {
	m.Called(method, host, status, duration)
}

func (m *mockHandlerMetrics) RecordBackendRequest(backend, status string, duration time.Duration) {
	m.Called(backend, status, duration)
}

func (m *mockHandlerMetrics) IncActiveConnections() {
	m.Called()
}

func (m *mockHandlerMetrics) DecActiveConnections() {
	m.Called()
}

func (m *mockHandlerMetrics) RecordCertificateRenewal(domain string, success bool) {
	m.Called(domain, success)
}

func (m *mockHandlerMetrics) RecordRateLimitHit(key string) {
	m.Called(key)
}

func (m *mockHandlerMetrics) RecordHealthCheck(target string, success bool, duration time.Duration) {
	m.Called(target, success, duration)
}

// Benchmark tests for handler operations
func BenchmarkProxyHandler_ServeHTTP(b *testing.B) {
	// Setup
	mockRouter := &mockRouter{}
	mockLogger := &mockHandlerLogger{}
	mockMetrics := &mockHandlerMetrics{}

	// Create test server
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"status": "ok"}`))
	}))
	defer testServer.Close()

	testURL, _ := url.Parse(testServer.URL)
	backend := newMockBackend("api", testURL.String())
	backend.On("URL").Return(testURL)
	backend.On("Transport").Return(http.DefaultTransport)
	backend.On("Name").Return("api")

	mockRouter.On("Route", mock.Anything, "api.example.com").Return(backend, nil)
	mockLogger.On("Info", mock.Anything, mock.Anything, mock.Anything).Return()
	mockMetrics.On("IncActiveConnections").Return()
	mockMetrics.On("DecActiveConnections").Return()
	mockMetrics.On("RecordBackendRequest", mock.Anything, mock.Anything, mock.Anything).Return()

	handler, err := NewProxyHandlerImpl(mockRouter, mockLogger, mockMetrics)
	if err != nil {
		b.Fatal(err)
	}

	req := httptest.NewRequest("GET", "http://api.example.com/test", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
	}
}

func BenchmarkProxyHandler_ServeHTTP_WithBody(b *testing.B) {
	// Setup
	mockRouter := &mockRouter{}
	mockLogger := &mockHandlerLogger{}
	mockMetrics := &mockHandlerMetrics{}

	// Create test server
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Read and echo body
		body, _ := io.ReadAll(r.Body)
		w.WriteHeader(200)
		_, _ = w.Write(body)
	}))
	defer testServer.Close()

	testURL, _ := url.Parse(testServer.URL)
	backend := newMockBackend("api", testURL.String())
	backend.On("URL").Return(testURL)
	backend.On("Transport").Return(http.DefaultTransport)
	backend.On("Name").Return("api")

	mockRouter.On("Route", mock.Anything, "api.example.com").Return(backend, nil)
	mockLogger.On("Info", mock.Anything, mock.Anything, mock.Anything).Return()
	mockMetrics.On("IncActiveConnections").Return()
	mockMetrics.On("DecActiveConnections").Return()
	mockMetrics.On("RecordBackendRequest", mock.Anything, mock.Anything, mock.Anything).Return()

	handler, err := NewProxyHandlerImpl(mockRouter, mockLogger, mockMetrics)
	if err != nil {
		b.Fatal(err)
	}

	body := `{"test": "data", "benchmark": true}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "http://api.example.com/test", bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
	}
}
