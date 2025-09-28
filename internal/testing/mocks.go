package testing

import (
	"context"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/stretchr/testify/mock"

	"github.com/albedosehen/rwwwrse/internal/config"
	"github.com/albedosehen/rwwwrse/internal/observability"
	"github.com/albedosehen/rwwwrse/internal/proxy"
)

type MockLogger struct {
	mock.Mock
	CallCount int64
}

func NewMockLogger() *MockLogger {
	return &MockLogger{}
}

func (m *MockLogger) Info(ctx context.Context, msg string, fields ...observability.Field) {
	atomic.AddInt64(&m.CallCount, 1)
	args := make([]interface{}, 0, len(fields)+2)
	args = append(args, ctx, msg)
	for _, field := range fields {
		args = append(args, field)
	}
	m.Called(args...)
}

func (m *MockLogger) Error(ctx context.Context, err error, msg string, fields ...observability.Field) {
	atomic.AddInt64(&m.CallCount, 1)
	args := make([]interface{}, 0, len(fields)+3)
	args = append(args, ctx, err, msg)
	for _, field := range fields {
		args = append(args, field)
	}
	m.Called(args...)
}

func (m *MockLogger) Warn(ctx context.Context, msg string, fields ...observability.Field) {
	atomic.AddInt64(&m.CallCount, 1)
	args := make([]interface{}, 0, len(fields)+2)
	args = append(args, ctx, msg)
	for _, field := range fields {
		args = append(args, field)
	}
	m.Called(args...)
}

func (m *MockLogger) Debug(ctx context.Context, msg string, fields ...observability.Field) {
	atomic.AddInt64(&m.CallCount, 1)
	args := make([]interface{}, 0, len(fields)+2)
	args = append(args, ctx, msg)
	for _, field := range fields {
		args = append(args, field)
	}
	m.Called(args...)
}

func (m *MockLogger) WithFields(fields ...observability.Field) observability.Logger {
	args := make([]interface{}, len(fields))
	for i, field := range fields {
		args[i] = field
	}
	m.Called(args...)
	return m
}

func (m *MockLogger) WithContext(ctx context.Context) observability.Logger {
	m.Called(ctx)
	return m
}

func (m *MockLogger) GetCallCount() int64 {
	return atomic.LoadInt64(&m.CallCount)
}

type MockMetricsCollector struct {
	mock.Mock
	Counters   map[string]float64
	Gauges     map[string]float64
	Histograms map[string][]float64
	mu         sync.RWMutex
}

func NewMockMetricsCollector() *MockMetricsCollector {
	return &MockMetricsCollector{
		Counters:   make(map[string]float64),
		Gauges:     make(map[string]float64),
		Histograms: make(map[string][]float64),
	}
}

func (m *MockMetricsCollector) RecordCounter(name string, value float64, labels map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := name
	if len(labels) > 0 {
		key += "_with_labels"
	}
	m.Counters[key] += value

	m.Called(name, value, labels)
}

func (m *MockMetricsCollector) RecordGauge(name string, value float64, labels map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := name
	if len(labels) > 0 {
		key += "_with_labels"
	}
	m.Gauges[key] = value

	m.Called(name, value, labels)
}

func (m *MockMetricsCollector) RecordHistogram(name string, value float64, labels map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := name
	if len(labels) > 0 {
		key += "_with_labels"
	}
	m.Histograms[key] = append(m.Histograms[key], value)

	m.Called(name, value, labels)
}

func (m *MockMetricsCollector) GetCounterValue(name string) float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.Counters[name]
}

func (m *MockMetricsCollector) GetGaugeValue(name string) float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.Gauges[name]
}

func (m *MockMetricsCollector) GetHistogramValues(name string) []float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	values := make([]float64, len(m.Histograms[name]))
	copy(values, m.Histograms[name])
	return values
}

func (m *MockMetricsCollector) RecordRequest(method, host, status string, duration time.Duration) {
	m.Called(method, host, status, duration)
}

func (m *MockMetricsCollector) RecordBackendRequest(backend, status string, duration time.Duration) {
	m.Called(backend, status, duration)
}

func (m *MockMetricsCollector) IncActiveConnections() {
	m.Called()
}

func (m *MockMetricsCollector) DecActiveConnections() {
	m.Called()
}

func (m *MockMetricsCollector) RecordCertificateRenewal(domain string, success bool) {
	m.Called(domain, success)
}

func (m *MockMetricsCollector) RecordRateLimitHit(key string) {
	m.Called(key)
}

func (m *MockMetricsCollector) RecordHealthCheck(target string, success bool, duration time.Duration) {
	m.Called(target, success, duration)
}

func (m *MockMetricsCollector) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.Counters = make(map[string]float64)
	m.Gauges = make(map[string]float64)
	m.Histograms = make(map[string][]float64)
}

type MockBackend struct {
	mock.Mock
	url       *url.URL
	transport http.RoundTripper
	healthy   bool
	name      string
	mu        sync.RWMutex
}

func NewMockBackend(name, urlStr string) *MockBackend {
	u, _ := url.Parse(urlStr)
	return &MockBackend{
		url:       u,
		transport: http.DefaultTransport,
		healthy:   true,
		name:      name,
	}
}

func (m *MockBackend) URL() *url.URL {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.url
}

func (m *MockBackend) Transport() http.RoundTripper {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.transport
}

func (m *MockBackend) IsHealthy(ctx context.Context) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	args := m.Called(ctx)
	if len(args) > 0 {
		return args.Bool(0)
	}
	return m.healthy
}

func (m *MockBackend) Name() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.name
}

func (m *MockBackend) SetHealthy(healthy bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.healthy = healthy
}

func (m *MockBackend) SetURL(urlStr string) error {
	u, err := url.Parse(urlStr)
	if err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.url = u
	return nil
}

func (m *MockBackend) SetTransport(transport http.RoundTripper) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.transport = transport
}

type MockRouter struct {
	mock.Mock
	backends map[string]proxy.Backend
	mu       sync.RWMutex
}

func NewMockRouter() *MockRouter {
	return &MockRouter{
		backends: make(map[string]proxy.Backend),
	}
}

func (m *MockRouter) Route(ctx context.Context, host string) (proxy.Backend, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	args := m.Called(ctx, host)

	if backend, ok := args.Get(0).(proxy.Backend); ok {
		return backend, args.Error(1)
	}

	// Fallback to internal backends map
	if backend, exists := m.backends[host]; exists {
		return backend, nil
	}

	return nil, args.Error(1)
}

func (m *MockRouter) Register(host string, backend proxy.Backend) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	args := m.Called(host, backend)

	if args.Error(0) == nil {
		m.backends[host] = backend
	}

	return args.Error(0)
}

func (m *MockRouter) Unregister(host string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	args := m.Called(host)

	if args.Error(0) == nil {
		delete(m.backends, host)
	}

	return args.Error(0)
}

func (m *MockRouter) Backends() map[string]proxy.Backend {
	m.mu.RLock()
	defer m.mu.RUnlock()

	args := m.Called()

	if backends, ok := args.Get(0).(map[string]proxy.Backend); ok {
		return backends
	}

	result := make(map[string]proxy.Backend)
	for k, v := range m.backends {
		result[k] = v
	}
	return result
}

func (m *MockRouter) AddBackend(host string, backend proxy.Backend) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.backends[host] = backend
}

type MockBackendManager struct {
	mock.Mock
	backends map[string]proxy.Backend
	mu       sync.RWMutex
}

func NewMockBackendManager() *MockBackendManager {
	return &MockBackendManager{
		backends: make(map[string]proxy.Backend),
	}
}

func (m *MockBackendManager) AddBackend(ctx context.Context, host string, backend proxy.Backend) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	args := m.Called(ctx, host, backend)

	if args.Error(0) == nil {
		m.backends[host] = backend
	}

	return args.Error(0)
}

func (m *MockBackendManager) RemoveBackend(ctx context.Context, host string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	args := m.Called(ctx, host)

	if args.Error(0) == nil {
		delete(m.backends, host)
	}

	return args.Error(0)
}

func (m *MockBackendManager) UpdateBackend(ctx context.Context, host string, backend proxy.Backend) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	args := m.Called(ctx, host, backend)

	if args.Error(0) == nil {
		m.backends[host] = backend
	}

	return args.Error(0)
}

func (m *MockBackendManager) GetBackend(ctx context.Context, host string) (proxy.Backend, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	args := m.Called(ctx, host)

	if backend, ok := args.Get(0).(proxy.Backend); ok {
		return backend, args.Error(1)
	}

	// Fallback to internal backends map
	if backend, exists := m.backends[host]; exists {
		return backend, nil
	}

	return nil, args.Error(1)
}

func (m *MockBackendManager) ListBackends(ctx context.Context) (map[string]proxy.Backend, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	args := m.Called(ctx)

	if backends, ok := args.Get(0).(map[string]proxy.Backend); ok {
		return backends, args.Error(1)
	}

	// Return copy of internal backends
	result := make(map[string]proxy.Backend)
	for k, v := range m.backends {
		result[k] = v
	}
	return result, args.Error(1)
}

type MockProxyHandler struct {
	mock.Mock
	requestCount int64
}

func NewMockProxyHandler() *MockProxyHandler {
	return &MockProxyHandler{}
}

func (m *MockProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	atomic.AddInt64(&m.requestCount, 1)
	m.Called(w, r)
}

func (m *MockProxyHandler) Shutdown(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockProxyHandler) GetRequestCount() int64 {
	return atomic.LoadInt64(&m.requestCount)
}

func (m *MockProxyHandler) ResetRequestCount() {
	atomic.StoreInt64(&m.requestCount, 0)
}

type MockRoundTripper struct {
	mock.Mock
	responses map[string]*http.Response
	delay     time.Duration
	mu        sync.RWMutex
}

func NewMockRoundTripper() *MockRoundTripper {
	return &MockRoundTripper{
		responses: make(map[string]*http.Response),
	}
}

func (m *MockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if m.delay > 0 {
		time.Sleep(m.delay)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	args := m.Called(req)

	if resp, ok := args.Get(0).(*http.Response); ok {
		return resp, args.Error(1)
	}

	// Fallback to predefined responses
	key := req.URL.String()
	if resp, exists := m.responses[key]; exists {
		return resp, nil
	}

	return nil, args.Error(1)
}

func (m *MockRoundTripper) SetResponse(url string, resp *http.Response) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.responses[url] = resp
}

func (m *MockRoundTripper) SetDelay(delay time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.delay = delay
}

type MockConfigLoader struct {
	mock.Mock
	config *config.Config
	mu     sync.RWMutex
}

func NewMockConfigLoader() *MockConfigLoader {
	return &MockConfigLoader{
		config: GetTestConfig(),
	}
}

func (m *MockConfigLoader) Load() (*config.Config, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	args := m.Called()

	if cfg, ok := args.Get(0).(*config.Config); ok {
		return cfg, args.Error(1)
	}

	return m.config, args.Error(1)
}

func (m *MockConfigLoader) Validate(cfg *config.Config) error {
	args := m.Called(cfg)
	return args.Error(0)
}

func (m *MockConfigLoader) Watch(ctx context.Context) (<-chan *config.Config, error) {
	args := m.Called(ctx)

	if ch, ok := args.Get(0).(<-chan *config.Config); ok {
		return ch, args.Error(1)
	}

	ch := make(chan *config.Config, 1)
	ch <- m.config
	close(ch)

	return ch, args.Error(1)
}

func (m *MockConfigLoader) SetConfig(cfg *config.Config) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.config = cfg
}
