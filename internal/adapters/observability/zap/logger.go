package zap

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"proxynd/internal/ports"
)

// Logger implements ports.Logger using Zap
type Logger struct {
	logger *zap.Logger
	sugar  *zap.SugaredLogger
}

// Field implements ports.Field using Zap
type Field struct {
	key   string
	value interface{}
}

// Key returns the field key
func (f *Field) Key() string {
	return f.key
}

// Value returns the field value
func (f *Field) Value() interface{} {
	return f.value
}

// NewField creates a new field
func NewField(key string, value interface{}) ports.Field {
	return &Field{
		key:   key,
		value: value,
	}
}

// NewLogger creates a new Zap logger adapter
func NewLogger(config *LoggerConfig) (ports.Logger, error) {
	zapConfig := zap.NewProductionConfig()
	
	if config != nil {
		// Apply configuration
		if config.Level != "" {
			level, err := zapcore.ParseLevel(config.Level)
			if err != nil {
				return nil, fmt.Errorf("invalid log level: %w", err)
			}
			zapConfig.Level = zap.NewAtomicLevelAt(level)
		}
		
		if config.Format == "console" {
			zapConfig.Encoding = "console"
		} else {
			zapConfig.Encoding = "json"
		}
		
		if len(config.OutputPaths) > 0 {
			zapConfig.OutputPaths = config.OutputPaths
		}
		
		if len(config.ErrorOutputPaths) > 0 {
			zapConfig.ErrorOutputPaths = config.ErrorOutputPaths
		}
		
		// Add caller info if enabled
		if config.EnableCaller {
			zapConfig.Development = true
		}
	}
	
	logger, err := zapConfig.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build zap logger: %w", err)
	}
	
	return &Logger{
		logger: logger,
		sugar:  logger.Sugar(),
	}, nil
}

// Debug logs debug level message
func (l *Logger) Debug(ctx context.Context, msg string, fields ...ports.Field) {
	zapFields := l.convertFields(ctx, fields...)
	l.logger.Debug(msg, zapFields...)
}

// Info logs info level message
func (l *Logger) Info(ctx context.Context, msg string, fields ...ports.Field) {
	zapFields := l.convertFields(ctx, fields...)
	l.logger.Info(msg, zapFields...)
}

// Warn logs warning level message
func (l *Logger) Warn(ctx context.Context, msg string, fields ...ports.Field) {
	zapFields := l.convertFields(ctx, fields...)
	l.logger.Warn(msg, zapFields...)
}

// Error logs error level message
func (l *Logger) Error(ctx context.Context, msg string, fields ...ports.Field) {
	zapFields := l.convertFields(ctx, fields...)
	l.logger.Error(msg, zapFields...)
}

// Fatal logs fatal level message and exits
func (l *Logger) Fatal(ctx context.Context, msg string, fields ...ports.Field) {
	zapFields := l.convertFields(ctx, fields...)
	l.logger.Fatal(msg, zapFields...)
}

// With creates logger with additional fields
func (l *Logger) With(fields ...ports.Field) ports.Logger {
	zapFields := make([]zap.Field, len(fields))
	for i, field := range fields {
		zapFields[i] = l.convertField(field)
	}
	
	return &Logger{
		logger: l.logger.With(zapFields...),
		sugar:  l.logger.With(zapFields...).Sugar(),
	}
}

// WithContext creates logger with context
func (l *Logger) WithContext(ctx context.Context) ports.Logger {
	fields := l.extractContextFields(ctx)
	if len(fields) == 0 {
		return l
	}
	return l.With(fields...)
}

// convertFields converts ports.Field slice to zap.Field slice
func (l *Logger) convertFields(ctx context.Context, fields ...ports.Field) []zap.Field {
	// Extract context fields first
	contextFields := l.extractContextFields(ctx)
	
	// Combine context fields with provided fields
	allFields := make([]ports.Field, len(contextFields)+len(fields))
	copy(allFields, contextFields)
	copy(allFields[len(contextFields):], fields)
	
	zapFields := make([]zap.Field, len(allFields))
	for i, field := range allFields {
		zapFields[i] = l.convertField(field)
	}
	
	return zapFields
}

// convertField converts a single ports.Field to zap.Field
func (l *Logger) convertField(field ports.Field) zap.Field {
	switch v := field.Value().(type) {
	case string:
		return zap.String(field.Key(), v)
	case int:
		return zap.Int(field.Key(), v)
	case int64:
		return zap.Int64(field.Key(), v)
	case float64:
		return zap.Float64(field.Key(), v)
	case bool:
		return zap.Bool(field.Key(), v)
	case time.Duration:
		return zap.Duration(field.Key(), v)
	case time.Time:
		return zap.Time(field.Key(), v)
	case error:
		return zap.Error(v)
	default:
		return zap.Any(field.Key(), v)
	}
}

// extractContextFields extracts logging fields from context
func (l *Logger) extractContextFields(ctx context.Context) []ports.Field {
	var fields []ports.Field
	
	// Extract trace ID
	if traceID := l.getTraceIDFromContext(ctx); traceID != "" {
		fields = append(fields, NewField("trace_id", traceID))
	}
	
	// Extract request ID
	if requestID := l.getRequestIDFromContext(ctx); requestID != "" {
		fields = append(fields, NewField("request_id", requestID))
	}
	
	// Extract user ID
	if userID := l.getUserIDFromContext(ctx); userID != "" {
		fields = append(fields, NewField("user_id", userID))
	}
	
	// Extract session ID
	if sessionID := l.getSessionIDFromContext(ctx); sessionID != "" {
		fields = append(fields, NewField("session_id", sessionID))
	}
	
	return fields
}

// Context value key constants
type contextKey string

const (
	traceIDKey   contextKey = "trace_id"
	requestIDKey contextKey = "request_id"
	userIDKey    contextKey = "user_id"
	sessionIDKey contextKey = "session_id"
)

// getTraceIDFromContext extracts trace ID from context
func (l *Logger) getTraceIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(traceIDKey).(string); ok {
		return id
	}
	return ""
}

// getRequestIDFromContext extracts request ID from context
func (l *Logger) getRequestIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey).(string); ok {
		return id
	}
	return ""
}

// getUserIDFromContext extracts user ID from context
func (l *Logger) getUserIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(userIDKey).(string); ok {
		return id
	}
	return ""
}

// getSessionIDFromContext extracts session ID from context
func (l *Logger) getSessionIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(sessionIDKey).(string); ok {
		return id
	}
	return ""
}

// LoggerConfig defines Zap logger configuration
type LoggerConfig struct {
	Level            string   `json:"level" yaml:"level" default:"info"`
	Format           string   `json:"format" yaml:"format" default:"json"` // json, console
	OutputPaths      []string `json:"output_paths" yaml:"output_paths"`
	ErrorOutputPaths []string `json:"error_output_paths" yaml:"error_output_paths"`
	EnableCaller     bool     `json:"enable_caller" yaml:"enable_caller" default:"true"`
	EnableStackTrace bool     `json:"enable_stack_trace" yaml:"enable_stack_trace" default:"false"`
}

// DefaultLoggerConfig returns default logger configuration
func DefaultLoggerConfig() *LoggerConfig {
	return &LoggerConfig{
		Level:            "info",
		Format:           "json",
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
		EnableCaller:     true,
		EnableStackTrace: false,
	}
}

// DevelopmentLoggerConfig returns development logger configuration
func DevelopmentLoggerConfig() *LoggerConfig {
	return &LoggerConfig{
		Level:            "debug",
		Format:           "console",
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
		EnableCaller:     true,
		EnableStackTrace: true,
	}
}

// Sync flushes any buffered log entries
func (l *Logger) Sync() error {
	return l.logger.Sync()
}

// Close closes the logger
func (l *Logger) Close() error {
	return l.logger.Sync()
}