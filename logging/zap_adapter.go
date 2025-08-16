package logging

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// ZapAdapter implements the Logger interface using zap instead of zerolog
type ZapAdapter struct {
	logger    *zap.Logger
	sugar     *zap.SugaredLogger
	fields    []zap.Field
	component string
}

// ZapMigrationConfig contains configuration for zap migration
type ZapMigrationConfig struct {
	Level       string                 `json:"level" yaml:"level"`
	Format      string                 `json:"format" yaml:"format"` // json, console
	Output      []OutputConfig         `json:"output" yaml:"output"`
	Fields      map[string]interface{} `json:"fields" yaml:"fields"`
	Sampling    *SamplingConfig        `json:"sampling" yaml:"sampling"`
	Correlation bool                   `json:"correlation" yaml:"correlation"`
	Caller      bool                   `json:"caller" yaml:"caller"`
}

// OutputConfig defines output configuration for zap
type OutputConfig struct {
	Type       string `json:"type" yaml:"type"` // stdout, stderr, file, syslog
	Path       string `json:"path,omitempty" yaml:"path,omitempty"`
	MaxSize    int    `json:"max_size,omitempty" yaml:"max_size,omitempty"`
	MaxAge     int    `json:"max_age,omitempty" yaml:"max_age,omitempty"`
	MaxBackups int    `json:"max_backups,omitempty" yaml:"max_backups,omitempty"`
	Compress   bool   `json:"compress,omitempty" yaml:"compress,omitempty"`
}

// StandardizedFields defines standard field names for ProxyND observability
var StandardizedFields = struct {
	RequestID   string
	Principal   string
	Manager     string
	Path        string
	CacheHit    string
	LatencyMs   string
	Component   string
	UserID      string
	Error       string
	Method      string
	StatusCode  string
	BytesIn     string
	BytesOut    string
	Upstream    string
	PackageName string
	Version     string
	TraceID     string
	SpanID      string
}{
	RequestID:   "request_id",
	Principal:   "principal",
	Manager:     "manager",
	Path:        "path",
	CacheHit:    "cache_hit",
	LatencyMs:   "latency_ms",
	Component:   "component",
	UserID:      "user_id",
	Error:       "error",
	Method:      "method",
	StatusCode:  "status_code",
	BytesIn:     "bytes_in",
	BytesOut:    "bytes_out",
	Upstream:    "upstream",
	PackageName: "package_name",
	Version:     "version",
	TraceID:     "trace_id",
	SpanID:      "span_id",
}

// NewZapAdapter creates a new ZapAdapter that implements the Logger interface
func NewZapAdapter(config ZapMigrationConfig) (Logger, error) {
	// Parse log level
	level := parseZapLevel(config.Level)

	// Create encoder config with standardized fields
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "message",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.RFC3339TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// Create encoder
	var encoder zapcore.Encoder
	if config.Format == "console" || config.Format == "text" {
		encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	}

	// Create writer syncer
	writeSyncer, err := createZapWriteSyncerFromConfig(config.Output)
	if err != nil {
		return nil, fmt.Errorf("failed to create write syncer: %w", err)
	}

	// Create core
	core := zapcore.NewCore(encoder, writeSyncer, level)

	// Add sampling if enabled
	if config.Sampling != nil && config.Sampling.Enabled {
		core = zapcore.NewSamplerWithOptions(
			core,
			time.Second,
			config.Sampling.Initial,
			config.Sampling.Thereafter,
		)
	}

	// Create logger options
	opts := []zap.Option{}
	if config.Caller {
		opts = append(opts, zap.AddCaller(), zap.AddCallerSkip(1))
	}
	opts = append(opts, zap.AddStacktrace(zap.ErrorLevel))

	// Create logger
	zapLogger := zap.New(core, opts...)

	// Add default fields
	var fields []zap.Field
	if config.Fields != nil {
		for k, v := range config.Fields {
			fields = append(fields, zap.Any(k, v))
		}
	}

	// Add hostname
	if hostname, err := os.Hostname(); err == nil {
		fields = append(fields, zap.String("hostname", hostname))
	}

	// Add PID
	fields = append(fields, zap.Int("pid", os.Getpid()))

	// Apply default fields
	if len(fields) > 0 {
		zapLogger = zapLogger.With(fields...)
	}

	return &ZapAdapter{
		logger: zapLogger,
		sugar:  zapLogger.Sugar(),
		fields: make([]zap.Field, 0),
	}, nil
}

// Debug logs a debug message
func (z *ZapAdapter) Debug(msg string, fields ...Field) {
	zapFields := z.convertFields(fields...)
	z.logger.Debug(msg, append(z.fields, zapFields...)...)
}

// Info logs an info message
func (z *ZapAdapter) Info(msg string, fields ...Field) {
	zapFields := z.convertFields(fields...)
	z.logger.Info(msg, append(z.fields, zapFields...)...)
}

// Warn logs a warning message
func (z *ZapAdapter) Warn(msg string, fields ...Field) {
	zapFields := z.convertFields(fields...)
	z.logger.Warn(msg, append(z.fields, zapFields...)...)
}

// Error logs an error message
func (z *ZapAdapter) Error(msg string, fields ...Field) {
	zapFields := z.convertFields(fields...)
	z.logger.Error(msg, append(z.fields, zapFields...)...)
}

// Fatal logs a fatal message and exits
func (z *ZapAdapter) Fatal(msg string, fields ...Field) {
	zapFields := z.convertFields(fields...)
	z.logger.Fatal(msg, append(z.fields, zapFields...)...)
}

// Panic logs a panic message and panics
func (z *ZapAdapter) Panic(msg string, fields ...Field) {
	zapFields := z.convertFields(fields...)
	z.logger.Panic(msg, append(z.fields, zapFields...)...)
}

// WithContext creates a logger with context information
func (z *ZapAdapter) WithContext(ctx context.Context) Logger {
	fields := make([]zap.Field, 0, 4)

	// Extract standardized context values
	if reqID := ctx.Value(StandardizedFields.RequestID); reqID != nil {
		fields = append(fields, zap.String(StandardizedFields.RequestID, fmt.Sprintf("%v", reqID)))
	}

	if userID := ctx.Value(StandardizedFields.UserID); userID != nil {
		fields = append(fields, zap.String(StandardizedFields.UserID, fmt.Sprintf("%v", userID)))
	}

	if principal := ctx.Value(StandardizedFields.Principal); principal != nil {
		fields = append(fields, zap.String(StandardizedFields.Principal, fmt.Sprintf("%v", principal)))
	}

	if traceID := ctx.Value(StandardizedFields.TraceID); traceID != nil {
		fields = append(fields, zap.String(StandardizedFields.TraceID, fmt.Sprintf("%v", traceID)))
	}

	return &ZapAdapter{
		logger:    z.logger,
		sugar:     z.sugar,
		fields:    append(z.fields, fields...),
		component: z.component,
	}
}

// WithFields creates a logger with multiple fields
func (z *ZapAdapter) WithFields(fields ...Field) Logger {
	zapFields := z.convertFields(fields...)
	return &ZapAdapter{
		logger:    z.logger,
		sugar:     z.sugar,
		fields:    append(z.fields, zapFields...),
		component: z.component,
	}
}

// WithField creates a logger with a single field
func (z *ZapAdapter) WithField(key string, value interface{}) Logger {
	field := zap.Any(key, value)
	return &ZapAdapter{
		logger:    z.logger,
		sugar:     z.sugar,
		fields:    append(z.fields, field),
		component: z.component,
	}
}

// Printf provides printf-style logging for compatibility
func (z *ZapAdapter) Printf(format string, v ...interface{}) {
	z.sugar.Infof(format, v...)
}

// convertFields converts Field slice to zap.Field slice
func (z *ZapAdapter) convertFields(fields ...Field) []zap.Field {
	zapFields := make([]zap.Field, len(fields))
	for i, field := range fields {
		zapFields[i] = z.convertField(field)
	}
	return zapFields
}

// convertField converts a single Field to zap.Field
func (z *ZapAdapter) convertField(field Field) zap.Field {
	switch v := field.Value.(type) {
	case string:
		return zap.String(field.Key, v)
	case int:
		return zap.Int(field.Key, v)
	case int32:
		return zap.Int32(field.Key, v)
	case int64:
		return zap.Int64(field.Key, v)
	case uint:
		return zap.Uint(field.Key, v)
	case uint32:
		return zap.Uint32(field.Key, v)
	case uint64:
		return zap.Uint64(field.Key, v)
	case float32:
		return zap.Float32(field.Key, v)
	case float64:
		return zap.Float64(field.Key, v)
	case bool:
		return zap.Bool(field.Key, v)
	case time.Duration:
		return zap.Duration(field.Key, v)
	case time.Time:
		return zap.Time(field.Key, v)
	case error:
		return zap.NamedError(field.Key, v)
	case []string:
		return zap.Strings(field.Key, v)
	default:
		return zap.Any(field.Key, v)
	}
}

// parseZapLevel converts string level to zapcore.Level
func parseZapLevel(level string) zapcore.Level {
	switch level {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn", "warning":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	case "fatal":
		return zapcore.FatalLevel
	case "panic":
		return zapcore.PanicLevel
	default:
		return zapcore.InfoLevel
	}
}

// createZapWriteSyncerFromConfig creates a zapcore.WriteSyncer from output config
func createZapWriteSyncerFromConfig(outputs []OutputConfig) (zapcore.WriteSyncer, error) {
	if len(outputs) == 0 {
		return zapcore.AddSync(os.Stdout), nil
	}

	var writers []zapcore.WriteSyncer

	for _, output := range outputs {
		switch output.Type {
		case "stdout":
			writers = append(writers, zapcore.AddSync(os.Stdout))
		case "stderr":
			writers = append(writers, zapcore.AddSync(os.Stderr))
		case "file":
			if output.Path == "" {
				return nil, fmt.Errorf("file path is required for file output")
			}

			lumberjackLogger := &lumberjack.Logger{
				Filename:   output.Path,
				MaxSize:    output.MaxSize,
				MaxBackups: output.MaxBackups,
				MaxAge:     output.MaxAge,
				Compress:   output.Compress,
				LocalTime:  true,
			}

			writers = append(writers, zapcore.AddSync(lumberjackLogger))
		default:
			return nil, fmt.Errorf("unsupported output type: %s", output.Type)
		}
	}

	if len(writers) == 1 {
		return writers[0], nil
	}

	return zapcore.NewMultiWriteSyncer(writers...), nil
}

// StandardizedZapLogger provides convenience methods for standardized logging
type StandardizedZapLogger struct {
	*ZapAdapter
}

// NewStandardizedZapLogger creates a logger with standardized field names
func NewStandardizedZapLogger(config ZapMigrationConfig) (*StandardizedZapLogger, error) {
	zapAdapter, err := NewZapAdapter(config)
	if err != nil {
		return nil, err
	}

	return &StandardizedZapLogger{
		ZapAdapter: zapAdapter.(*ZapAdapter),
	}, nil
}

// LogHTTPRequest logs an HTTP request with standardized fields
func (s *StandardizedZapLogger) LogHTTPRequest(ctx context.Context, method, path string, statusCode int, latencyMs int64, cacheHit bool) {
	fields := []Field{
		{Key: StandardizedFields.Method, Value: method},
		{Key: StandardizedFields.Path, Value: path},
		{Key: StandardizedFields.StatusCode, Value: statusCode},
		{Key: StandardizedFields.LatencyMs, Value: latencyMs},
		{Key: StandardizedFields.CacheHit, Value: cacheHit},
	}

	logger := s.WithContext(ctx)
	if statusCode >= 400 {
		logger.Error("HTTP request completed with error", fields...)
	} else {
		logger.Info("HTTP request completed", fields...)
	}
}

// LogProxyOperation logs a proxy operation with standardized fields
func (s *StandardizedZapLogger) LogProxyOperation(ctx context.Context, manager, upstream, packageName, version string, bytesIn, bytesOut int64, err error) {
	fields := []Field{
		{Key: StandardizedFields.Manager, Value: manager},
		{Key: StandardizedFields.Upstream, Value: upstream},
		{Key: StandardizedFields.PackageName, Value: packageName},
		{Key: StandardizedFields.Version, Value: version},
		{Key: StandardizedFields.BytesIn, Value: bytesIn},
		{Key: StandardizedFields.BytesOut, Value: bytesOut},
	}

	if err != nil {
		fields = append(fields, Field{Key: StandardizedFields.Error, Value: err.Error()})
	}

	logger := s.WithContext(ctx)
	if err != nil {
		logger.Error("Proxy operation failed", fields...)
	} else {
		logger.Info("Proxy operation completed", fields...)
	}
}

// LogCacheOperation logs a cache operation with standardized fields
func (s *StandardizedZapLogger) LogCacheOperation(ctx context.Context, operation string, cacheHit bool, manager, key string) {
	fields := []Field{
		{Key: "operation", Value: operation},
		{Key: StandardizedFields.CacheHit, Value: cacheHit},
		{Key: StandardizedFields.Manager, Value: manager},
		{Key: "cache_key", Value: key},
	}

	logger := s.WithContext(ctx)
	logger.Debug("Cache operation", fields...)
}

// WithComponent creates a logger with component field
func (s *StandardizedZapLogger) WithComponent(component string) *StandardizedZapLogger {
	newAdapter := s.WithField(StandardizedFields.Component, component).(*ZapAdapter)
	return &StandardizedZapLogger{
		ZapAdapter: newAdapter,
	}
}

// Sync flushes any buffered log entries
func (z *ZapAdapter) Sync() error {
	return z.logger.Sync()
}

// Close closes the logger
func (z *ZapAdapter) Close() error {
	return z.logger.Sync()
}

// MigrateToZap creates a global zap logger to replace the existing zerolog implementation
func MigrateToZap(config ZapMigrationConfig) error {
	zapLogger, err := NewZapAdapter(config)
	if err != nil {
		return fmt.Errorf("failed to create zap adapter: %w", err)
	}

	// Replace global logger
	globalLogger = zapLogger

	return nil
}

// GetStandardizedLogger returns a standardized logger instance
func GetStandardizedLogger() *StandardizedZapLogger {
	if adapter, ok := globalLogger.(*ZapAdapter); ok {
		return &StandardizedZapLogger{ZapAdapter: adapter}
	}

	// Fallback to creating a new standardized logger
	config := ZapMigrationConfig{
		Level:  "info",
		Format: "json",
		Output: []OutputConfig{{Type: "stdout"}},
	}

	logger, err := NewStandardizedZapLogger(config)
	if err != nil {
		// Return a basic adapter if creation fails
		return &StandardizedZapLogger{
			ZapAdapter: &ZapAdapter{},
		}
	}

	return logger
}
