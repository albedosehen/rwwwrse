package observability

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
)

type slogLogger struct {
	logger *slog.Logger
	fields []Field
}

func NewLogger(config LoggingConfig) Logger {
	var handler slog.Handler

	// Create output writer
	var writer io.Writer
	switch strings.ToLower(config.Output) {
	case "stderr":
		writer = os.Stderr
	case "stdout", "":
		writer = os.Stdout
	default:
		// For file outputs, we'd need to open the file
		// For now, default to stdout
		writer = os.Stdout
	}

	// Configure handler options
	opts := &slog.HandlerOptions{
		Level:     config.Level.ToSlogLevel(),
		AddSource: config.AddSource,
	}

	// Create appropriate handler based on format
	switch config.Format {
	case FormatJSON:
		handler = slog.NewJSONHandler(writer, opts)
	case FormatText:
		handler = slog.NewTextHandler(writer, opts)
	default:
		handler = slog.NewJSONHandler(writer, opts)
	}

	return &slogLogger{
		logger: slog.New(handler),
		fields: make([]Field, 0),
	}
}

// Debug logs debug-level messages with optional fields.
func (l *slogLogger) Debug(ctx context.Context, msg string, fields ...Field) {
	if !l.logger.Enabled(ctx, slog.LevelDebug) {
		return
	}

	attrs := l.buildAttrs(ctx, fields...)
	l.logger.LogAttrs(ctx, slog.LevelDebug, msg, attrs...)
}

// Info logs info-level messages with optional fields.
func (l *slogLogger) Info(ctx context.Context, msg string, fields ...Field) {
	if !l.logger.Enabled(ctx, slog.LevelInfo) {
		return
	}

	attrs := l.buildAttrs(ctx, fields...)
	l.logger.LogAttrs(ctx, slog.LevelInfo, msg, attrs...)
}

// Warn logs warning-level messages with optional fields.
func (l *slogLogger) Warn(ctx context.Context, msg string, fields ...Field) {
	if !l.logger.Enabled(ctx, slog.LevelWarn) {
		return
	}

	attrs := l.buildAttrs(ctx, fields...)
	l.logger.LogAttrs(ctx, slog.LevelWarn, msg, attrs...)
}

// Error logs error-level messages with error and optional fields.
func (l *slogLogger) Error(ctx context.Context, err error, msg string, fields ...Field) {
	if !l.logger.Enabled(ctx, slog.LevelError) {
		return
	}

	// Add error to fields
	allFields := make([]Field, 0, len(fields)+1)
	if err != nil {
		allFields = append(allFields, Error(err))
	}
	allFields = append(allFields, fields...)

	attrs := l.buildAttrs(ctx, allFields...)
	l.logger.LogAttrs(ctx, slog.LevelError, msg, attrs...)
}

// WithFields returns a new logger with the specified fields pre-set.
func (l *slogLogger) WithFields(fields ...Field) Logger {
	newFields := make([]Field, 0, len(l.fields)+len(fields))
	newFields = append(newFields, l.fields...)
	newFields = append(newFields, fields...)

	return &slogLogger{
		logger: l.logger,
		fields: newFields,
	}
}

// WithContext returns a new logger with context-specific fields.
func (l *slogLogger) WithContext(ctx context.Context) Logger {
	contextFields := extractContextFields(ctx)
	return l.WithFields(contextFields...)
}

// buildAttrs combines pre-set fields, context fields, and provided fields into slog attributes.
func (l *slogLogger) buildAttrs(ctx context.Context, fields ...Field) []slog.Attr {
	// Calculate total capacity
	totalFields := len(l.fields) + len(fields)

	// Extract context fields
	contextFields := extractContextFields(ctx)
	totalFields += len(contextFields)

	attrs := make([]slog.Attr, 0, totalFields)

	// Add pre-set fields
	for _, field := range l.fields {
		attrs = append(attrs, field.ToSlogAttr())
	}

	// Add context fields
	for _, field := range contextFields {
		attrs = append(attrs, field.ToSlogAttr())
	}

	// Add provided fields
	for _, field := range fields {
		attrs = append(attrs, field.ToSlogAttr())
	}

	return attrs
}

type contextKey string

const (
	requestIDKey     contextKey = "request_id"
	correlationIDKey contextKey = "correlation_id"
	userIDKey        contextKey = "user_id"
	sessionIDKey     contextKey = "session_id"
	traceIDKey       contextKey = "trace_id"
	spanIDKey        contextKey = "span_id"
	componentKey     contextKey = "component"
)

func extractContextFields(ctx context.Context) []Field {
	var fields []Field

	if requestID, ok := ctx.Value(requestIDKey).(string); ok && requestID != "" {
		fields = append(fields, RequestID(requestID))
	}

	if correlationID, ok := ctx.Value(correlationIDKey).(string); ok && correlationID != "" {
		fields = append(fields, String("correlation_id", correlationID))
	}

	if userID, ok := ctx.Value(userIDKey).(string); ok && userID != "" {
		fields = append(fields, String("user_id", userID))
	}

	if sessionID, ok := ctx.Value(sessionIDKey).(string); ok && sessionID != "" {
		fields = append(fields, String("session_id", sessionID))
	}

	if traceID, ok := ctx.Value(traceIDKey).(string); ok && traceID != "" {
		fields = append(fields, String("trace_id", traceID))
	}

	if spanID, ok := ctx.Value(spanIDKey).(string); ok && spanID != "" {
		fields = append(fields, String("span_id", spanID))
	}

	if component, ok := ctx.Value(componentKey).(string); ok && component != "" {
		fields = append(fields, Component(component))
	}

	return fields
}

// WithRequestID adds a request ID to the context for logging.
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

// WithCorrelationID adds a correlation ID to the context for logging.
func WithCorrelationID(ctx context.Context, correlationID string) context.Context {
	return context.WithValue(ctx, correlationIDKey, correlationID)
}

// WithUserID adds a user ID to the context for logging.
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// WithSessionID adds a session ID to the context for logging.
func WithSessionID(ctx context.Context, sessionID string) context.Context {
	return context.WithValue(ctx, sessionIDKey, sessionID)
}

// WithTraceID adds a trace ID to the context for logging.
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceIDKey, traceID)
}

// WithSpanID adds a span ID to the context for logging.
func WithSpanID(ctx context.Context, spanID string) context.Context {
	return context.WithValue(ctx, spanIDKey, spanID)
}

// WithComponent adds a component identifier to the context for logging.
func WithComponent(ctx context.Context, component string) context.Context {
	return context.WithValue(ctx, componentKey, component)
}

func GetRequestID(ctx context.Context) string {
	if requestID, ok := ctx.Value(requestIDKey).(string); ok {
		return requestID
	}
	return ""
}

func GetCorrelationID(ctx context.Context) string {
	if correlationID, ok := ctx.Value(correlationIDKey).(string); ok {
		return correlationID
	}
	return ""
}

func GetTraceID(ctx context.Context) string {
	if traceID, ok := ctx.Value(traceIDKey).(string); ok {
		return traceID
	}
	return ""
}

var defaultLoggerConfig = LoggingConfig{
	Level:      LevelInfo,
	Format:     FormatJSON,
	Output:     "stdout",
	AddSource:  false,
	TimeFormat: "",
}

// Default returns a logger with default configuration.
func Default() Logger {
	return NewLogger(defaultLoggerConfig)
}

func ParseLogLevel(level string) LogLevel {
	switch strings.ToLower(level) {
	case "debug":
		return LevelDebug
	case "info":
		return LevelInfo
	case "warn", "warning":
		return LevelWarn
	case "error":
		return LevelError
	default:
		return LevelInfo
	}
}

func ParseLogFormat(format string) LogFormat {
	switch strings.ToLower(format) {
	case "json":
		return FormatJSON
	case "text", "console":
		return FormatText
	default:
		return FormatJSON
	}
}
