package observability

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type prometheusCollector struct {
	// HTTP request metrics
	requestsTotal     *prometheus.CounterVec
	requestDuration   *prometheus.HistogramVec
	activeConnections prometheus.Gauge

	// Backend metrics
	backendRequestsTotal   *prometheus.CounterVec
	backendRequestDuration *prometheus.HistogramVec
	backendHealthStatus    *prometheus.GaugeVec

	// TLS metrics
	certificateRenewalsTotal *prometheus.CounterVec
	certificateExpiry        *prometheus.GaugeVec

	// Rate limiting metrics
	rateLimitHitsTotal *prometheus.CounterVec

	// Health check metrics
	healthChecksTotal   *prometheus.CounterVec
	healthCheckDuration *prometheus.HistogramVec

	// System metrics
	startTime prometheus.Gauge

	registry *prometheus.Registry
	server   *http.Server
	mutex    sync.RWMutex
}

func NewPrometheusCollector(namespace, subsystem string) MetricsCollector {
	registry := prometheus.NewRegistry()

	// Create metrics with consistent labeling
	collector := &prometheusCollector{
		registry: registry,

		// HTTP request metrics
		requestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "requests_total",
				Help:      "Total number of HTTP requests processed",
			},
			[]string{"method", "host", "status_code"},
		),

		requestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "request_duration_seconds",
				Help:      "Time spent processing HTTP requests",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"method", "host", "status_code"},
		),

		activeConnections: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "active_connections",
				Help:      "Current number of active connections",
			},
		),

		// Backend metrics
		backendRequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "backend_requests_total",
				Help:      "Total number of backend requests",
			},
			[]string{"backend", "status_code"},
		),

		backendRequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "backend_request_duration_seconds",
				Help:      "Time spent on backend requests",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"backend", "status_code"},
		),

		backendHealthStatus: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "backend_health_status",
				Help:      "Backend health status (1=healthy, 0=unhealthy)",
			},
			[]string{"backend"},
		),

		// TLS metrics
		certificateRenewalsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "certificate_renewals_total",
				Help:      "Total number of certificate renewal attempts",
			},
			[]string{"domain", "status"},
		),

		certificateExpiry: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "certificate_expiry_timestamp",
				Help:      "Certificate expiry time as Unix timestamp",
			},
			[]string{"domain"},
		),

		// Rate limiting metrics
		rateLimitHitsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "rate_limit_hits_total",
				Help:      "Total number of rate limit hits",
			},
			[]string{"key"},
		),

		// Health check metrics
		healthChecksTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "health_checks_total",
				Help:      "Total number of health checks performed",
			},
			[]string{"target", "status"},
		),

		healthCheckDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "health_check_duration_seconds",
				Help:      "Time spent on health checks",
				Buckets:   []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
			},
			[]string{"target", "status"},
		),

		// System metrics
		startTime: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "start_time_timestamp",
				Help:      "Start time of the application as Unix timestamp",
			},
		),
	}

	// Register all metrics
	collector.registerMetrics()

	// Set start time
	collector.startTime.SetToCurrentTime()

	return collector
}

func (p *prometheusCollector) registerMetrics() {
	p.registry.MustRegister(
		p.requestsTotal,
		p.requestDuration,
		p.activeConnections,
		p.backendRequestsTotal,
		p.backendRequestDuration,
		p.backendHealthStatus,
		p.certificateRenewalsTotal,
		p.certificateExpiry,
		p.rateLimitHitsTotal,
		p.healthChecksTotal,
		p.healthCheckDuration,
		p.startTime,
	)
}

func (p *prometheusCollector) RecordRequest(method, host, status string, duration time.Duration) {
	labels := prometheus.Labels{
		"method":      method,
		"host":        host,
		"status_code": status,
	}

	p.requestsTotal.With(labels).Inc()
	p.requestDuration.With(labels).Observe(duration.Seconds())
}

func (p *prometheusCollector) RecordBackendRequest(backend, status string, duration time.Duration) {
	labels := prometheus.Labels{
		"backend":     backend,
		"status_code": status,
	}

	p.backendRequestsTotal.With(labels).Inc()
	p.backendRequestDuration.With(labels).Observe(duration.Seconds())
}

func (p *prometheusCollector) IncActiveConnections() {
	p.activeConnections.Inc()
}

func (p *prometheusCollector) DecActiveConnections() {
	p.activeConnections.Dec()
}

func (p *prometheusCollector) RecordCertificateRenewal(domain string, success bool) {
	status := "failure"
	if success {
		status = "success"
	}

	p.certificateRenewalsTotal.With(prometheus.Labels{
		"domain": domain,
		"status": status,
	}).Inc()
}

func (p *prometheusCollector) RecordRateLimitHit(key string) {
	p.rateLimitHitsTotal.With(prometheus.Labels{
		"key": key,
	}).Inc()
}

func (p *prometheusCollector) RecordHealthCheck(target string, success bool, duration time.Duration) {
	status := "failure"
	if success {
		status = "success"
	}

	labels := prometheus.Labels{
		"target": target,
		"status": status,
	}

	p.healthChecksTotal.With(labels).Inc()
	p.healthCheckDuration.With(labels).Observe(duration.Seconds())

	// Update backend health status
	healthValue := float64(0)
	if success {
		healthValue = 1
	}
	p.backendHealthStatus.With(prometheus.Labels{"backend": target}).Set(healthValue)
}

func (p *prometheusCollector) SetCertificateExpiry(domain string, expiry time.Time) {
	p.certificateExpiry.With(prometheus.Labels{
		"domain": domain,
	}).Set(float64(expiry.Unix()))
}

func (p *prometheusCollector) GetRegistry() *prometheus.Registry {
	return p.registry
}

func (p *prometheusCollector) StartMetricsServer(ctx context.Context, address string) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.server != nil {
		return fmt.Errorf("metrics server already running")
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(p.registry, promhttp.HandlerOpts{
		EnableOpenMetrics: true,
	}))

	// Add health endpoint for the metrics server itself
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	p.server = &http.Server{
		Addr:    address,
		Handler: mux,
	}

	go func() {
		if err := p.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			// Log error but don't fail the application
			// This should be logged via the logger when integrated
		}
	}()

	return nil
}

func (p *prometheusCollector) StopMetricsServer(ctx context.Context) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.server == nil {
		return nil
	}

	err := p.server.Shutdown(ctx)
	p.server = nil
	return err
}

type metricsExporter struct {
	collector *prometheusCollector
	config    MetricsConfig
}

// NewMetricsExporter creates a new metrics exporter.
// TODO: Implement OpenTelemetry, Datadog, etc.
func NewMetricsExporter(collector MetricsCollector, config MetricsConfig) MetricsExporter {
	promCollector, ok := collector.(*prometheusCollector)
	if !ok {
		// Return a no-op exporter if not using Prometheus
		return &noopExporter{}
	}

	return &metricsExporter{
		collector: promCollector,
		config:    config,
	}
}

// TODO: Implement other Export if necessary.
func (e *metricsExporter) Export(ctx context.Context) error {
	// For Prometheus, metrics are pulled via HTTP, so this is a no-op
	return nil
}

func (e *metricsExporter) Start(ctx context.Context) error {
	if !e.config.Enabled {
		return nil
	}

	address := e.config.Address
	if address == "" {
		address = ":9090"
	}

	return e.collector.StartMetricsServer(ctx, address)
}

func (e *metricsExporter) Stop(ctx context.Context) error {
	return e.collector.StopMetricsServer(ctx)
}

type noopExporter struct{}

func (e *noopExporter) Export(ctx context.Context) error { return nil }
func (e *noopExporter) Start(ctx context.Context) error  { return nil }
func (e *noopExporter) Stop(ctx context.Context) error   { return nil }

func MetricsMiddleware(collector MetricsCollector) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			collector.IncActiveConnections()
			defer collector.DecActiveConnections()

			// Wrap response writer to capture status code
			wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			next.ServeHTTP(wrapped, r)

			duration := time.Since(start)
			collector.RecordRequest(
				r.Method,
				r.Host,
				fmt.Sprintf("%d", wrapped.statusCode),
				duration,
			)
		})
	}
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
