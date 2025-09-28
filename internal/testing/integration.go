//go:build integration

package testing

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/albedosehen/rwwwrse/internal/config"
	"github.com/albedosehen/rwwwrse/internal/di"
	"github.com/albedosehen/rwwwrse/internal/observability"
	"github.com/albedosehen/rwwwrse/internal/proxy"
)

// IntegrationTestSuite provides integration testing utilities
type IntegrationTestSuite struct {
	t           *testing.T
	backends    []*httptest.Server
	proxyServer *httptest.Server
	app         *di.Application
	config      *config.Config
	cleanup     []func()
}

// NewIntegrationTestSuite creates a new integration test suite
func NewIntegrationTestSuite(t *testing.T) *IntegrationTestSuite {
	t.Helper()

	return &IntegrationTestSuite{
		t:       t,
		cleanup: make([]func(), 0),
	}
}

// SetupBackends creates test backend servers
func (s *IntegrationTestSuite) SetupBackends(backends map[string]http.HandlerFunc) {
	s.t.Helper()

	for name, handler := range backends {
		server := httptest.NewServer(handler)
		s.backends = append(s.backends, server)

		s.addCleanup(server.Close)

		s.t.Logf("Created backend %s at %s", name, server.URL)
	}
}

// SetupProxyWithConfig creates a proxy server with the given configuration
func (s *IntegrationTestSuite) SetupProxyWithConfig(cfg *config.Config) error {
	s.t.Helper()

	s.config = cfg

	// Create application with dependency injection
	app, err := di.InitializeApplication(cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize application: %w", err)
	}

	s.app = app

	// Create test server with the proxy handler
	s.proxyServer = httptest.NewServer(app.Handler)
	s.addCleanup(s.proxyServer.Close)

	s.t.Logf("Created proxy server at %s", s.proxyServer.URL)

	return nil
}

// SetupProxy creates a proxy server with default test configuration
func (s *IntegrationTestSuite) SetupProxy() error {
	s.t.Helper()

	cfg := GetTestConfig()

	// Update backend routes with actual backend URLs
	if len(s.backends) > 0 {
		cfg.Backends.Routes = make(map[string]config.BackendRoute)
		for i, backend := range s.backends {
			host := fmt.Sprintf("backend%d.test.com", i+1)
			cfg.Backends.Routes[host] = config.BackendRoute{
				URL:            backend.URL,
				HealthPath:     "/health",
				HealthInterval: time.Second,
				Timeout:        time.Second,
				MaxIdleConns:   10,
				MaxIdlePerHost: 2,
				DialTimeout:    time.Second,
			}
		}
	}

	return s.SetupProxyWithConfig(cfg)
}

// MakeRequest makes an HTTP request to the proxy server
func (s *IntegrationTestSuite) MakeRequest(method, path string, headers map[string]string) (*http.Response, error) {
	s.t.Helper()

	if s.proxyServer == nil {
		s.t.Fatal("proxy server not set up, call SetupProxy first")
	}

	url := s.proxyServer.URL + path
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	return client.Do(req)
}

// MakeProxyRequest makes a request through the proxy to a specific backend
func (s *IntegrationTestSuite) MakeProxyRequest(host, method, path string) (*http.Response, error) {
	s.t.Helper()

	headers := map[string]string{
		"Host": host,
	}

	return s.MakeRequest(method, path, headers)
}

// AssertProxyResponse asserts the proxy response matches expectations
func (s *IntegrationTestSuite) AssertProxyResponse(resp *http.Response, expectedStatus int, expectedBody string) {
	s.t.Helper()

	assert.Equal(s.t, expectedStatus, resp.StatusCode, "unexpected status code")

	if expectedBody != "" {
		body := make([]byte, len(expectedBody))
		_, err := resp.Body.Read(body)
		assert.NoError(s.t, err, "failed to read response body")
		assert.Equal(s.t, expectedBody, string(body), "unexpected response body")
	}
}

// TestHealthEndpoint tests the health endpoint
func (s *IntegrationTestSuite) TestHealthEndpoint() {
	s.t.Helper()

	resp, err := s.MakeRequest("GET", "/health", nil)
	require.NoError(s.t, err, "health request failed")
	defer resp.Body.Close()

	assert.Equal(s.t, http.StatusOK, resp.StatusCode, "health endpoint should return 200")
}

// TestMetricsEndpoint tests the metrics endpoint
func (s *IntegrationTestSuite) TestMetricsEndpoint() {
	s.t.Helper()

	resp, err := s.MakeRequest("GET", "/metrics", nil)
	require.NoError(s.t, err, "metrics request failed")
	defer resp.Body.Close()

	assert.Equal(s.t, http.StatusOK, resp.StatusCode, "metrics endpoint should return 200")
}

// TestConcurrentRequests tests concurrent request handling
func (s *IntegrationTestSuite) TestConcurrentRequests(numRequests int, host, path string) {
	s.t.Helper()

	RunConcurrently(s.t, numRequests, func(i int) {
		resp, err := s.MakeProxyRequest(host, "GET", path)
		assert.NoError(s.t, err, "request %d failed", i)
		if resp != nil {
			resp.Body.Close()
		}
	})
}

// WaitForBackendHealth waits for backend to become healthy
func (s *IntegrationTestSuite) WaitForBackendHealth(host string, timeout time.Duration) bool {
	s.t.Helper()

	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		resp, err := s.MakeProxyRequest(host, "GET", "/health")
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			return true
		}
		if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(100 * time.Millisecond)
	}

	return false
}

// GetBackendURL returns the URL of the backend at the given index
func (s *IntegrationTestSuite) GetBackendURL(index int) string {
	s.t.Helper()

	if index >= len(s.backends) {
		s.t.Fatalf("backend index %d out of range (have %d backends)", index, len(s.backends))
	}

	return s.backends[index].URL
}

// GetProxyURL returns the URL of the proxy server
func (s *IntegrationTestSuite) GetProxyURL() string {
	s.t.Helper()

	if s.proxyServer == nil {
		s.t.Fatal("proxy server not set up")
	}

	return s.proxyServer.URL
}

// GetApplication returns the application instance
func (s *IntegrationTestSuite) GetApplication() *di.Application {
	return s.app
}

// GetConfig returns the configuration
func (s *IntegrationTestSuite) GetConfig() *config.Config {
	return s.config
}

// addCleanup adds a cleanup function
func (s *IntegrationTestSuite) addCleanup(fn func()) {
	s.cleanup = append(s.cleanup, fn)
}

// Cleanup cleans up all resources
func (s *IntegrationTestSuite) Cleanup() {
	for i := len(s.cleanup) - 1; i >= 0; i-- {
		s.cleanup[i]()
	}
	s.cleanup = nil
}

// LoadTestConfig represents load test configuration
type LoadTestConfig struct {
	NumClients     int
	RequestsPerSec int
	Duration       time.Duration
	RequestPath    string
	RequestHost    string
	RequestMethod  string
	ExpectedStatus int
}

// DefaultLoadTestConfig returns default load test configuration
func DefaultLoadTestConfig() LoadTestConfig {
	return LoadTestConfig{
		NumClients:     10,
		RequestsPerSec: 100,
		Duration:       5 * time.Second,
		RequestPath:    "/",
		RequestHost:    "test.example.com",
		RequestMethod:  "GET",
		ExpectedStatus: http.StatusOK,
	}
}

// LoadTestResult represents load test results
type LoadTestResult struct {
	TotalRequests  int
	SuccessfulReqs int
	FailedRequests int
	AverageLatency time.Duration
	MinLatency     time.Duration
	MaxLatency     time.Duration
	RequestsPerSec float64
	Errors         []error
}

// RunLoadTest performs a load test against the proxy
func (s *IntegrationTestSuite) RunLoadTest(config LoadTestConfig) LoadTestResult {
	s.t.Helper()

	if s.proxyServer == nil {
		s.t.Fatal("proxy server not set up")
	}

	result := LoadTestResult{
		MinLatency: time.Hour, // Initialize with large value
		Errors:     make([]error, 0),
	}

	ctx, cancel := context.WithTimeout(context.Background(), config.Duration)
	defer cancel()

	// Channel to collect results
	resultCh := make(chan struct {
		success bool
		latency time.Duration
		err     error
	}, config.NumClients*100)

	// Start workers
	for i := 0; i < config.NumClients; i++ {
		go func() {
			client := &http.Client{
				Timeout: 5 * time.Second,
				Transport: &http.Transport{
					MaxIdleConnsPerHost: 10,
				},
			}

			ticker := time.NewTicker(time.Second / time.Duration(config.RequestsPerSec/config.NumClients))
			defer ticker.Stop()

			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					start := time.Now()

					url := s.proxyServer.URL + config.RequestPath
					req, err := http.NewRequestWithContext(ctx, config.RequestMethod, url, nil)
					if err != nil {
						resultCh <- struct {
							success bool
							latency time.Duration
							err     error
						}{false, 0, err}
						continue
					}

					req.Header.Set("Host", config.RequestHost)

					resp, err := client.Do(req)
					latency := time.Since(start)

					success := err == nil && resp != nil && resp.StatusCode == config.ExpectedStatus
					if resp != nil {
						resp.Body.Close()
					}

					resultCh <- struct {
						success bool
						latency time.Duration
						err     error
					}{success, latency, err}
				}
			}
		}()
	}

	// Collect results
	var totalLatency time.Duration

	for {
		select {
		case <-ctx.Done():
			goto done
		case res := <-resultCh:
			result.TotalRequests++

			if res.success {
				result.SuccessfulReqs++
			} else {
				result.FailedRequests++
				if res.err != nil {
					result.Errors = append(result.Errors, res.err)
				}
			}

			if res.latency > 0 {
				totalLatency += res.latency
				if res.latency < result.MinLatency {
					result.MinLatency = res.latency
				}
				if res.latency > result.MaxLatency {
					result.MaxLatency = res.latency
				}
			}
		}
	}

done:
	// Calculate final metrics
	if result.TotalRequests > 0 {
		result.AverageLatency = totalLatency / time.Duration(result.TotalRequests)
		result.RequestsPerSec = float64(result.TotalRequests) / config.Duration.Seconds()
	}

	return result
}

// TLSTestServer creates a test server with TLS
type TLSTestServer struct {
	*httptest.Server
	Certificate tls.Certificate
}

// NewTLSTestServer creates a new TLS test server
func NewTLSTestServer(handler http.Handler) *TLSTestServer {
	server := httptest.NewTLSServer(handler)

	return &TLSTestServer{
		Server:      server,
		Certificate: server.TLS.Certificates[0],
	}
}

// GetTLSClient returns an HTTP client configured for this TLS server
func (s *TLSTestServer) GetTLSClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
		Timeout: 5 * time.Second,
	}
}

// PortScanner helps find available ports for testing
type PortScanner struct{}

// FindAvailablePort finds an available port
func (ps *PortScanner) FindAvailablePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()

	return l.Addr().(*net.TCPAddr).Port, nil
}

// FindAvailablePorts finds multiple available ports
func (ps *PortScanner) FindAvailablePorts(count int) ([]int, error) {
	ports := make([]int, 0, count)

	for i := 0; i < count; i++ {
		port, err := ps.FindAvailablePort()
		if err != nil {
			return nil, err
		}
		ports = append(ports, port)
	}

	return ports, nil
}

// NetworkTestHelper provides network testing utilities
type NetworkTestHelper struct {
	t *testing.T
}

// NewNetworkTestHelper creates a new network test helper
func NewNetworkTestHelper(t *testing.T) *NetworkTestHelper {
	return &NetworkTestHelper{t: t}
}

// WaitForPortOpen waits for a port to become available
func (nh *NetworkTestHelper) WaitForPortOpen(address string, timeout time.Duration) bool {
	nh.t.Helper()

	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", address, 100*time.Millisecond)
		if err == nil {
			conn.Close()
			return true
		}
		time.Sleep(50 * time.Millisecond)
	}

	return false
}

// WaitForPortClosed waits for a port to become unavailable
func (nh *NetworkTestHelper) WaitForPortClosed(address string, timeout time.Duration) bool {
	nh.t.Helper()

	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", address, 100*time.Millisecond)
		if err != nil {
			return true
		}
		conn.Close()
		time.Sleep(50 * time.Millisecond)
	}

	return false
}

// AssertPortOpen asserts that a port is open
func (nh *NetworkTestHelper) AssertPortOpen(address string) {
	nh.t.Helper()

	conn, err := net.DialTimeout("tcp", address, time.Second)
	require.NoError(nh.t, err, "port %s should be open", address)
	if conn != nil {
		conn.Close()
	}
}

// AssertPortClosed asserts that a port is closed
func (nh *NetworkTestHelper) AssertPortClosed(address string) {
	nh.t.Helper()

	conn, err := net.DialTimeout("tcp", address, 100*time.Millisecond)
	if conn != nil {
		conn.Close()
	}
	assert.Error(nh.t, err, "port %s should be closed", address)
}
