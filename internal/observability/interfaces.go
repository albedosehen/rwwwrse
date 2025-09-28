package observability

import (
	"context"
	"log/slog"
	"time"
)

// Logger provides structured logging with context awareness.
// It wraps slog functionality with additional proxy-specific features.
type Logger interface {
	// Debug logs debug-level messages with optional fields.
	Debug(ctx context.Context, msg string, fields ...Field)

	// Info logs info-level messages with optional fields.
	Info(ctx context.Context, msg string, fields ...Field)

	// Warn logs warning-level messages with optional fields.
	Warn(ctx context.Context, msg string, fields ...Field)

	// Error logs error-level messages with error and optional fields.
	Error(ctx context.Context, err error, msg string, fields ...Field)

	// WithFields returns a new logger with the specified fields pre-set.
	WithFields(fields ...Field) Logger

	// WithContext returns a new logger with context-specific fields.
	WithContext(ctx context.Context) Logger
}

// MetricsCollector collects and exports application metrics.
// It provides methods for recording various proxy-related metrics.
type MetricsCollector interface {
	// RecordRequest records metrics for an HTTP request.
	RecordRequest(method, host, status string, duration time.Duration)

	// RecordBackendRequest records metrics for a backend request.
	RecordBackendRequest(backend, status string, duration time.Duration)

	// IncActiveConnections increments the active connections counter.
	IncActiveConnections()

	// DecActiveConnections decrements the active connections counter.
	DecActiveConnections()

	// RecordCertificateRenewal records certificate renewal metrics.
	RecordCertificateRenewal(domain string, success bool)

	// RecordRateLimitHit records rate limiting metrics.
	RecordRateLimitHit(key string)

	// RecordHealthCheck records health check metrics.
	RecordHealthCheck(target string, success bool, duration time.Duration)
}

// Tracer provides distributed tracing capabilities.
// It creates and manages trace spans for request tracking.
type Tracer interface {
	// StartSpan creates a new trace span with the given name.
	StartSpan(ctx context.Context, name string) (context.Context, Span)

	// InjectHeaders injects trace context into HTTP headers.
	InjectHeaders(ctx context.Context, headers map[string]string)

	// ExtractHeaders extracts trace context from HTTP headers.
	ExtractHeaders(headers map[string]string) context.Context
}

// Span represents a trace span with lifecycle management.
type Span interface {
	// SetTag sets a tag on the span.
	SetTag(key string, value interface{})

	// SetError marks the span as having an error.
	SetError(err error)

	// AddEvent adds an event to the span.
	AddEvent(name string, attributes map[string]interface{})

	// Finish completes the span.
	Finish()

	// Context returns the span's context.
	Context() context.Context
}

// Field represents a structured log field with key-value data.
type Field struct {
	Key   string
	Value interface{}
}

// ToSlogAttr converts a Field to a slog.Attr for compatibility.
func (f Field) ToSlogAttr() slog.Attr {
	return slog.Any(f.Key, f.Value)
}

// Common field creation functions

// String creates a string field.
func String(key, value string) Field {
	return Field{Key: key, Value: value}
}

// Int creates an integer field.
func Int(key string, value int) Field {
	return Field{Key: key, Value: value}
}

// Int64 creates an int64 field.
func Int64(key string, value int64) Field {
	return Field{Key: key, Value: value}
}

// Float64 creates a float64 field.
func Float64(key string, value float64) Field {
	return Field{Key: key, Value: value}
}

// Bool creates a boolean field.
func Bool(key string, value bool) Field {
	return Field{Key: key, Value: value}
}

// Duration creates a duration field.
func Duration(key string, value time.Duration) Field {
	return Field{Key: key, Value: value}
}

// Time creates a time field.
func Time(key string, value time.Time) Field {
	return Field{Key: key, Value: value}
}

// Error creates an error field.
func Error(err error) Field {
	return Field{Key: "error", Value: err.Error()}
}

// Any creates a field with an arbitrary value.
func Any(key string, value interface{}) Field {
	return Field{Key: key, Value: value}
}

// RequestID creates a request ID field.
func RequestID(id string) Field {
	return Field{Key: "request_id", Value: id}
}

// Host creates a host field.
func Host(host string) Field {
	return Field{Key: "host", Value: host}
}

// Backend creates a backend field.
func Backend(backend string) Field {
	return Field{Key: "backend", Value: backend}
}

// Method creates an HTTP method field.
func Method(method string) Field {
	return Field{Key: "method", Value: method}
}

// Status creates an HTTP status field.
func Status(status int) Field {
	return Field{Key: "status", Value: status}
}

// Path creates a URL path field.
func Path(path string) Field {
	return Field{Key: "path", Value: path}
}

// RemoteAddr creates a remote address field.
func RemoteAddr(addr string) Field {
	return Field{Key: "remote_addr", Value: addr}
}

// UserAgent creates a user agent field.
func UserAgent(ua string) Field {
	return Field{Key: "user_agent", Value: ua}
}

// Component creates a component field for identifying the source.
func Component(component string) Field {
	return Field{Key: "component", Value: component}
}

// MetricsExporter exports metrics to external systems.
type MetricsExporter interface {
	// Export exports metrics to the configured backend.
	Export(ctx context.Context) error

	// Start begins the metrics export process.
	Start(ctx context.Context) error

	// Stop halts the metrics export process.
	Stop(ctx context.Context) error
}

// LogLevel represents logging severity levels.
type LogLevel int

const (
	LevelDebug LogLevel = iota
	LevelInfo
	LevelWarn
	LevelError
)

// String returns the string representation of the log level.
func (l LogLevel) String() string {
	switch l {
	case LevelDebug:
		return "debug"
	case LevelInfo:
		return "info"
	case LevelWarn:
		return "warn"
	case LevelError:
		return "error"
	default:
		return "unknown"
	}
}

// ToSlogLevel converts LogLevel to slog.Level.
func (l LogLevel) ToSlogLevel() slog.Level {
	switch l {
	case LevelDebug:
		return slog.LevelDebug
	case LevelInfo:
		return slog.LevelInfo
	case LevelWarn:
		return slog.LevelWarn
	case LevelError:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// LogFormat represents different logging output formats.
type LogFormat int

const (
	FormatJSON LogFormat = iota
	FormatText
)

// String returns the string representation of the log format.
func (f LogFormat) String() string {
	switch f {
	case FormatJSON:
		return "json"
	case FormatText:
		return "text"
	default:
		return "json"
	}
}

// ObservabilityConfig configures observability components.
type ObservabilityConfig struct {
	Logging LoggingConfig `json:"logging"`
	Metrics MetricsConfig `json:"metrics"`
	Tracing TracingConfig `json:"tracing"`
}

// LoggingConfig configures the logging system.
type LoggingConfig struct {
	Level      LogLevel  `json:"level"`
	Format     LogFormat `json:"format"`
	Output     string    `json:"output"`
	AddSource  bool      `json:"add_source"`
	TimeFormat string    `json:"time_format"`
}

// MetricsConfig configures the metrics collection system.
type MetricsConfig struct {
	Enabled   bool   `json:"enabled"`
	Address   string `json:"address"`
	Path      string `json:"path"`
	Namespace string `json:"namespace"`
	Subsystem string `json:"subsystem"`
}

// TracingConfig configures the distributed tracing system.
type TracingConfig struct {
	Enabled     bool    `json:"enabled"`
	ServiceName string  `json:"service_name"`
	Endpoint    string  `json:"endpoint"`
	SampleRate  float64 `json:"sample_rate"`
}

// ObservabilityProvider creates and manages observability components.
type ObservabilityProvider interface {
	// Logger returns the configured logger instance.
	Logger() Logger

	// Metrics returns the configured metrics collector.
	Metrics() MetricsCollector

	// Tracer returns the configured tracer instance.
	Tracer() Tracer

	// Shutdown gracefully shuts down all observability components.
	Shutdown(ctx context.Context) error
}
