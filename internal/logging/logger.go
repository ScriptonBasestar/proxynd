package logging

import (
	"context"
	"fmt"
	"io"
	"os"
	"runtime"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// Logger interface for structured logging
type Logger interface {
	Debug(msg string, fields ...Field)
	Info(msg string, fields ...Field)
	Warn(msg string, fields ...Field)
	Error(msg string, fields ...Field)
	Fatal(msg string, fields ...Field)

	WithField(key string, value interface{}) Logger
	WithFields(fields map[string]interface{}) Logger
	WithContext(ctx context.Context) Logger
	WithComponent(component string) Logger
}

// Field represents a log field
type Field struct {
	Key   string
	Value interface{}
}

// Config holds logging configuration
type Config struct {
	Level       string                 `yaml:"level" json:"level"`
	Format      string                 `yaml:"format" json:"format"` // json, text
	Output      []OutputConfig         `yaml:"output" json:"output"`
	Sampling    *SamplingConfig        `yaml:"sampling,omitempty" json:"sampling,omitempty"`
	Correlation bool                   `yaml:"correlation" json:"correlation"`
	Caller      bool                   `yaml:"caller" json:"caller"`
	Fields      map[string]interface{} `yaml:"fields,omitempty" json:"fields,omitempty"`
}

// OutputConfig defines where logs should be written
type OutputConfig struct {
	Type       string `yaml:"type" json:"type"` // file, stdout, stderr, syslog
	Path       string `yaml:"path,omitempty" json:"path,omitempty"`
	MaxSize    int    `yaml:"max_size,omitempty" json:"max_size,omitempty"` // MB
	MaxAge     int    `yaml:"max_age,omitempty" json:"max_age,omitempty"`   // days
	MaxBackups int    `yaml:"max_backups,omitempty" json:"max_backups,omitempty"`
	Compress   bool   `yaml:"compress,omitempty" json:"compress,omitempty"`
}

// SamplingConfig for log sampling to reduce volume
type SamplingConfig struct {
	Enabled    bool    `yaml:"enabled" json:"enabled"`
	Initial    int     `yaml:"initial" json:"initial"`               // Log first N messages
	Thereafter int     `yaml:"thereafter" json:"thereafter"`         // Then log every Nth message
	Rate       float64 `yaml:"rate,omitempty" json:"rate,omitempty"` // Alternative: sample rate 0.0-1.0
}

// StructuredLogger implements Logger interface
type StructuredLogger struct {
	entry       *logrus.Entry
	config      *Config
	component   string
	correlation bool
}

// contextKey for correlation ID
type contextKey string

const (
	// CorrelationIDKey is the context key for correlation ID
	CorrelationIDKey contextKey = "correlation_id"
	// RequestIDKey is the context key for request ID
	RequestIDKey contextKey = "request_id"
	// UserIDKey is the context key for user ID
	UserIDKey contextKey = "user_id"
	// SessionIDKey is the context key for session ID
	SessionIDKey contextKey = "session_id"
)

// NewLogger creates a new structured logger
func NewLogger(config *Config) (Logger, error) {
	logrusLogger := logrus.New()

	// Set log level
	level, err := logrus.ParseLevel(config.Level)
	if err != nil {
		return nil, fmt.Errorf("invalid log level: %w", err)
	}
	logrusLogger.SetLevel(level)

	// Set formatter
	switch config.Format {
	case "json":
		logrusLogger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: time.RFC3339Nano,
			FieldMap: logrus.FieldMap{
				logrus.FieldKeyTime:  "timestamp",
				logrus.FieldKeyLevel: "level",
				logrus.FieldKeyMsg:   "message",
				logrus.FieldKeyFunc:  "caller",
			},
		})
	case "text":
		logrusLogger.SetFormatter(&logrus.TextFormatter{
			TimestampFormat: time.RFC3339,
			FullTimestamp:   true,
		})
	default:
		return nil, fmt.Errorf("invalid log format: %s", config.Format)
	}

	// Set output
	if len(config.Output) > 0 {
		writers := make([]io.Writer, 0, len(config.Output))
		for _, output := range config.Output {
			writer, err := createWriter(output)
			if err != nil {
				return nil, fmt.Errorf("failed to create writer: %w", err)
			}
			writers = append(writers, writer)
		}
		logrusLogger.SetOutput(io.MultiWriter(writers...))
	}

	// Set caller reporting
	if config.Caller {
		logrusLogger.SetReportCaller(true)
	}

	// Create base entry with global fields
	entry := logrusLogger.WithFields(logrus.Fields{
		"service":  "proxynd",
		"version":  getVersion(),
		"hostname": getHostname(),
		"pid":      os.Getpid(),
	})

	// Add custom fields
	if config.Fields != nil {
		entry = entry.WithFields(logrus.Fields(config.Fields))
	}

	return &StructuredLogger{
		entry:       entry,
		config:      config,
		correlation: config.Correlation,
	}, nil
}

// createWriter creates appropriate writer based on output config
func createWriter(config OutputConfig) (io.Writer, error) {
	switch config.Type {
	case "stdout":
		return os.Stdout, nil
	case "stderr":
		return os.Stderr, nil
	case "file":
		if config.Path == "" {
			return nil, fmt.Errorf("file path is required for file output")
		}
		// Create file with proper permissions
		file, err := os.OpenFile(config.Path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file: %w", err)
		}
		return file, nil
	default:
		return nil, fmt.Errorf("unsupported output type: %s", config.Type)
	}
}

// Debug logs a debug message
func (l *StructuredLogger) Debug(msg string, fields ...Field) {
	l.entry.WithFields(l.fieldsToLogrus(fields...)).Debug(msg)
}

// Info logs an info message
func (l *StructuredLogger) Info(msg string, fields ...Field) {
	l.entry.WithFields(l.fieldsToLogrus(fields...)).Info(msg)
}

// Warn logs a warning message
func (l *StructuredLogger) Warn(msg string, fields ...Field) {
	l.entry.WithFields(l.fieldsToLogrus(fields...)).Warn(msg)
}

// Error logs an error message
func (l *StructuredLogger) Error(msg string, fields ...Field) {
	// Add caller information for errors
	if l.config.Caller {
		if pc, file, line, ok := runtime.Caller(1); ok {
			fn := runtime.FuncForPC(pc)
			fields = append(fields, Field{
				Key:   "caller",
				Value: fmt.Sprintf("%s:%d %s", file, line, fn.Name()),
			})
		}
	}
	l.entry.WithFields(l.fieldsToLogrus(fields...)).Error(msg)
}

// Fatal logs a fatal message and exits
func (l *StructuredLogger) Fatal(msg string, fields ...Field) {
	l.entry.WithFields(l.fieldsToLogrus(fields...)).Fatal(msg)
}

// WithField adds a field to the logger
func (l *StructuredLogger) WithField(key string, value interface{}) Logger {
	return &StructuredLogger{
		entry:       l.entry.WithField(key, value),
		config:      l.config,
		component:   l.component,
		correlation: l.correlation,
	}
}

// WithFields adds multiple fields to the logger
func (l *StructuredLogger) WithFields(fields map[string]interface{}) Logger {
	return &StructuredLogger{
		entry:       l.entry.WithFields(logrus.Fields(fields)),
		config:      l.config,
		component:   l.component,
		correlation: l.correlation,
	}
}

// WithContext adds context information to the logger
func (l *StructuredLogger) WithContext(ctx context.Context) Logger {
	newEntry := l.entry

	if l.correlation {
		// Add correlation ID from context
		if correlationID := GetCorrelationID(ctx); correlationID != "" {
			newEntry = newEntry.WithField("correlation_id", correlationID)
		}

		// Add request ID from context
		if requestID := GetRequestID(ctx); requestID != "" {
			newEntry = newEntry.WithField("request_id", requestID)
		}

		// Add user ID from context
		if userID := GetUserID(ctx); userID != "" {
			newEntry = newEntry.WithField("user_id", userID)
		}

		// Add session ID from context
		if sessionID := GetSessionID(ctx); sessionID != "" {
			newEntry = newEntry.WithField("session_id", sessionID)
		}
	}

	return &StructuredLogger{
		entry:       newEntry,
		config:      l.config,
		component:   l.component,
		correlation: l.correlation,
	}
}

// WithComponent adds component information to the logger
func (l *StructuredLogger) WithComponent(component string) Logger {
	return &StructuredLogger{
		entry:       l.entry.WithField("component", component),
		config:      l.config,
		component:   component,
		correlation: l.correlation,
	}
}

// fieldsToLogrus converts Field slice to logrus.Fields
func (l *StructuredLogger) fieldsToLogrus(fields ...Field) logrus.Fields {
	logrusFields := make(logrus.Fields, len(fields))
	for _, field := range fields {
		logrusFields[field.Key] = field.Value
	}
	return logrusFields
}

// GetCorrelationID returns the correlation ID from context
func GetCorrelationID(ctx context.Context) string {
	if id, ok := ctx.Value(CorrelationIDKey).(string); ok {
		return id
	}
	return ""
}

// GetRequestID returns the request ID from context
func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(RequestIDKey).(string); ok {
		return id
	}
	return ""
}

// GetUserID returns the user ID from context
func GetUserID(ctx context.Context) string {
	if id, ok := ctx.Value(UserIDKey).(string); ok {
		return id
	}
	return ""
}

// GetSessionID returns the session ID from context
func GetSessionID(ctx context.Context) string {
	if id, ok := ctx.Value(SessionIDKey).(string); ok {
		return id
	}
	return ""
}

// WithCorrelationID adds correlation ID to context
func WithCorrelationID(ctx context.Context, id string) context.Context {
	if id == "" {
		id = uuid.New().String()
	}
	return context.WithValue(ctx, CorrelationIDKey, id)
}

// WithRequestID adds request ID to context
func WithRequestID(ctx context.Context, id string) context.Context {
	if id == "" {
		id = uuid.New().String()
	}
	return context.WithValue(ctx, RequestIDKey, id)
}

// WithUserID adds user ID to context
func WithUserID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, UserIDKey, id)
}

// WithSessionID adds session ID to context
func WithSessionID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, SessionIDKey, id)
}

// Utility functions
func getVersion() string {
	// This would typically be set during build
	return "dev"
}

func getHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return hostname
}

// NewField creates a new log field
func NewField(key string, value interface{}) Field {
	return Field{Key: key, Value: value}
}

// String creates a string field
func String(key, value string) Field {
	return NewField(key, value)
}

// Int creates an integer field
func Int(key string, value int) Field {
	return NewField(key, value)
}

// Float64 creates a float64 field
func Float64(key string, value float64) Field {
	return NewField(key, value)
}

// Bool creates a boolean field
func Bool(key string, value bool) Field {
	return NewField(key, value)
}

// Duration creates a duration field
func Duration(key string, value time.Duration) Field {
	return NewField(key, value)
}

// Error creates an error field
func Error(err error) Field {
	return NewField("error", err.Error())
}

// StackTrace creates a stack trace field
func StackTrace(err error) Field {
	return NewField("stack_trace", fmt.Sprintf("%+v", err))
}
