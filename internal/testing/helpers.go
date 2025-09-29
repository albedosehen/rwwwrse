package testing

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/albedosehen/rwwwrse/internal/config"
	"github.com/albedosehen/rwwwrse/internal/observability"
)

const TestTimeout = 10 * time.Second

type TestConfig struct {
	TempDir    string
	ConfigFile string
	cleanup    []func()
	mu         sync.Mutex
}

func NewTestConfig(t *testing.T) *TestConfig {
	t.Helper()

	tempDir := t.TempDir()

	return &TestConfig{
		TempDir:    tempDir,
		ConfigFile: "",
		cleanup:    make([]func(), 0),
	}
}

func (tc *TestConfig) AddCleanup(cleanup func()) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	tc.cleanup = append(tc.cleanup, cleanup)
}

func (tc *TestConfig) Cleanup() {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	for i := len(tc.cleanup) - 1; i >= 0; i-- {
		tc.cleanup[i]()
	}
	tc.cleanup = nil
}

func SetEnvVars(t *testing.T, vars map[string]string) func() {
	t.Helper()

	originalVars := make(map[string]string)

	for key, value := range vars {
		originalVars[key] = os.Getenv(key)
		err := os.Setenv(key, value)
		require.NoError(t, err, "failed to set environment variable %s", key)
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

func CreateTestBackend(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()

	if handler == nil {
		handler = func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("test backend response"))
		}
	}

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	return server
}

func CreateTestBackendWithStatus(t *testing.T, status int, body string) *httptest.Server {
	t.Helper()

	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		if body != "" {
			_, _ = w.Write([]byte(body))
		}
	}

	return CreateTestBackend(t, handler)
}

func CreateSlowTestBackend(t *testing.T, delay time.Duration) *httptest.Server {
	t.Helper()

	handler := func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(delay)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("slow response"))
	}

	return CreateTestBackend(t, handler)
}

func AssertEventuallyTrue(t *testing.T, condition func() bool, timeout time.Duration, msg string) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			assert.Fail(t, "condition was not met within timeout", msg)
			return
		case <-ticker.C:
			if condition() {
				return
			}
		}
	}
}

func AssertContextCanceled(t *testing.T, ctx context.Context, timeout time.Duration) {
	t.Helper()

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return
	case <-timer.C:
		assert.Fail(t, "context was not canceled within timeout")
	}
}

func GetTestConfig() *config.Config {
	cfg := config.GetDefaultConfig()

	// Disable AutoCert for testing
	cfg.TLS.AutoCert = false
	cfg.TLS.Enabled = false

	// Set test-friendly values
	cfg.Server.Port = 0 // Use random port
	cfg.Server.HTTPSPort = 0
	cfg.Metrics.Port = 0

	// Basic backend route for testing
	cfg.Backends.Routes = map[string]config.BackendRoute{
		"test.example.com": {
			URL:            "http://localhost:8080",
			HealthPath:     "/health",
			HealthInterval: time.Second,
			Timeout:        time.Second,
			MaxIdleConns:   10,
			MaxIdlePerHost: 2,
			DialTimeout:    time.Second,
		},
	}

	return cfg
}

func ParseURL(t *testing.T, rawURL string) *url.URL {
	t.Helper()

	u, err := url.Parse(rawURL)
	require.NoError(t, err, "failed to parse URL: %s", rawURL)

	return u
}

func MustParseURL(rawURL string) *url.URL {
	u, err := url.Parse(rawURL)
	if err != nil {
		panic("invalid URL: " + rawURL)
	}
	return u
}

func WaitForCondition(t *testing.T, condition func() bool, timeout, interval time.Duration) bool {
	t.Helper()

	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		if condition() {
			return true
		}
		time.Sleep(interval)
	}

	return false
}

func RunConcurrently(t *testing.T, n int, fn func(i int)) {
	t.Helper()

	var wg sync.WaitGroup
	wg.Add(n)

	for i := 0; i < n; i++ {
		go func(index int) {
			defer wg.Done()
			fn(index)
		}(i)
	}

	wg.Wait()
}

func TestContext(t *testing.T) (context.Context, context.CancelFunc) {
	t.Helper()

	return context.WithTimeout(context.Background(), TestTimeout)
}

func TestContextWithTimeout(t *testing.T, timeout time.Duration) (context.Context, context.CancelFunc) {
	t.Helper()

	return context.WithTimeout(context.Background(), timeout)
}

func AssertNoError(t *testing.T, err error, msgAndArgs ...interface{}) {
	t.Helper()
	require.NoError(t, err, msgAndArgs...)
}

func AssertError(t *testing.T, err error, msgAndArgs ...interface{}) {
	t.Helper()
	require.Error(t, err, msgAndArgs...)
}

func AssertEqual(t *testing.T, expected, actual interface{}, msgAndArgs ...interface{}) {
	t.Helper()
	assert.Equal(t, expected, actual, msgAndArgs...)
}

func AssertNotEqual(t *testing.T, expected, actual interface{}, msgAndArgs ...interface{}) {
	t.Helper()
	assert.NotEqual(t, expected, actual, msgAndArgs...)
}

func AssertNil(t *testing.T, object interface{}, msgAndArgs ...interface{}) {
	t.Helper()
	assert.Nil(t, object, msgAndArgs...)
}

func AssertNotNil(t *testing.T, object interface{}, msgAndArgs ...interface{}) {
	t.Helper()
	assert.NotNil(t, object, msgAndArgs...)
}

func AssertTrue(t *testing.T, value bool, msgAndArgs ...interface{}) {
	t.Helper()
	assert.True(t, value, msgAndArgs...)
}

func AssertFalse(t *testing.T, value bool, msgAndArgs ...interface{}) {
	t.Helper()
	assert.False(t, value, msgAndArgs...)
}

func AssertContains(t *testing.T, s, contains interface{}, msgAndArgs ...interface{}) {
	t.Helper()
	assert.Contains(t, s, contains, msgAndArgs...)
}

func AssertNotContains(t *testing.T, s, contains interface{}, msgAndArgs ...interface{}) {
	t.Helper()
	assert.NotContains(t, s, contains, msgAndArgs...)
}

func AssertLen(t *testing.T, object interface{}, length int, msgAndArgs ...interface{}) {
	t.Helper()
	assert.Len(t, object, length, msgAndArgs...)
}

func AssertEmpty(t *testing.T, object interface{}, msgAndArgs ...interface{}) {
	t.Helper()
	assert.Empty(t, object, msgAndArgs...)
}

func AssertNotEmpty(t *testing.T, object interface{}, msgAndArgs ...interface{}) {
	t.Helper()
	assert.NotEmpty(t, object, msgAndArgs...)
}

type CountingLogger struct {
	observability.Logger
	InfoCount  int
	ErrorCount int
	WarnCount  int
	DebugCount int
	mu         sync.Mutex
}

func NewCountingLogger() *CountingLogger {
	return &CountingLogger{}
}

func (cl *CountingLogger) Info(ctx context.Context, msg string, fields ...observability.Field) {
	cl.mu.Lock()
	defer cl.mu.Unlock()
	cl.InfoCount++
}

func (cl *CountingLogger) Error(ctx context.Context, msg string, fields ...observability.Field) {
	cl.mu.Lock()
	defer cl.mu.Unlock()
	cl.ErrorCount++
}

func (cl *CountingLogger) Warn(ctx context.Context, msg string, fields ...observability.Field) {
	cl.mu.Lock()
	defer cl.mu.Unlock()
	cl.WarnCount++
}

func (cl *CountingLogger) Debug(ctx context.Context, msg string, fields ...observability.Field) {
	cl.mu.Lock()
	defer cl.mu.Unlock()
	cl.DebugCount++
}

func (cl *CountingLogger) GetCounts() (info, error, warn, debug int) {
	cl.mu.Lock()
	defer cl.mu.Unlock()
	return cl.InfoCount, cl.ErrorCount, cl.WarnCount, cl.DebugCount
}

func (cl *CountingLogger) Reset() {
	cl.mu.Lock()
	defer cl.mu.Unlock()
	cl.InfoCount = 0
	cl.ErrorCount = 0
	cl.WarnCount = 0
	cl.DebugCount = 0
}
