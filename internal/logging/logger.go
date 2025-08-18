// Package logging provides structured logging capabilities for ProxyND.
// It uses zerolog under the hood and supports JSON/text output formats,
// log rotation, context-aware logging, and various log levels.
package logging

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/natefinch/lumberjack.v2"
)

// LogLevel represents the severity level of a log message.
type LogLevel string

const (
	// LevelDebug is for debug-level messages
	LevelDebug LogLevel = "debug"
	// LevelInfo is for informational messages
	LevelInfo LogLevel = "info"
	// LevelWarn is for warning messages
	LevelWarn LogLevel = "warn"
	// LevelError is for error messages
	LevelError LogLevel = "error"
	// LevelFatal is for fatal errors (causes program exit)
	LevelFatal LogLevel = "fatal"
	// LevelPanic is for panic situations
	LevelPanic LogLevel = "panic"
)

// Logger is the interface for structured logging in ProxyND.
// It provides methods for different log levels and context/field management.
type Logger interface {
	Debug(msg string, fields ...Field)
	Info(msg string, fields ...Field)
	Warn(msg string, fields ...Field)
	Error(msg string, fields ...Field)
	Fatal(msg string, fields ...Field)
	Panic(msg string, fields ...Field)

	WithContext(ctx context.Context) Logger
	WithFields(fields ...Field) Logger
	WithField(key string, value interface{}) Logger

	// 기존 log.Logger와의 호환성
	Printf(format string, v ...interface{})
}

// Field represents a key-value pair for structured logging.
type Field struct {
	Key   string
	Value interface{}
}

// F is a helper function to create a Field for structured logging.
func F(key string, value interface{}) Field {
	return Field{Key: key, Value: value}
}

// ZeroLogger is the zerolog-based implementation of the Logger interface.
type ZeroLogger struct {
	logger zerolog.Logger
	fields map[string]interface{}
	mu     sync.RWMutex
}

// 전역 로거 인스턴스
var (
	globalLogger Logger
	once         sync.Once
)

// LogConfig contains all configuration options for the logging system.
type LogConfig struct {
	Level      LogLevel `json:"level" yaml:"level"`
	Format     string   `json:"format" yaml:"format"` // json, text
	Output     string   `json:"output" yaml:"output"` // stdout, stderr, file
	TimeFormat string   `json:"time_format" yaml:"time_format"`

	// 파일 출력 설정
	File FileConfig `json:"file" yaml:"file"`

	// 컨텍스트 필드
	DefaultFields map[string]interface{} `json:"default_fields" yaml:"default_fields"`

	// 샘플링 설정
	Sampling SamplingConfig `json:"sampling" yaml:"sampling"`
}

// FileConfig contains configuration for file-based logging with rotation.
type FileConfig struct {
	Path       string `json:"path" yaml:"path"`
	MaxSize    int    `json:"max_size" yaml:"max_size"` // MB
	MaxBackups int    `json:"max_backups" yaml:"max_backups"`
	MaxAge     int    `json:"max_age" yaml:"max_age"` // days
	Compress   bool   `json:"compress" yaml:"compress"`
}

// SamplingConfig controls log sampling to reduce volume in high-throughput scenarios.
type SamplingConfig struct {
	Enabled    bool `json:"enabled" yaml:"enabled"`
	Initial    int  `json:"initial" yaml:"initial"`
	Thereafter int  `json:"thereafter" yaml:"thereafter"`
}

// InitLogger initializes the global logger with the provided configuration.
// It sets up output destinations, formatting, log levels, and optional features like sampling.
// Returns an error if initialization fails.
func InitLogger(config LogConfig) error {
	var logger zerolog.Logger

	// 로그 레벨 설정
	level := parseLevel(config.Level)
	zerolog.SetGlobalLevel(level)

	// 시간 포맷 설정
	if config.TimeFormat != "" {
		zerolog.TimeFieldFormat = config.TimeFormat
	} else {
		zerolog.TimeFieldFormat = time.RFC3339
	}

	// 출력 설정
	var writers []io.Writer

	switch config.Output {
	case "stderr":
		writers = append(writers, os.Stderr)
	case fieldFile:
		if config.File.Path == "" {
			return fmt.Errorf("file path is required for file output")
		}
		fileWriter := createFileWriter(config.File)
		writers = append(writers, fileWriter)
	case "both":
		writers = append(writers, os.Stdout)
		if config.File.Path != "" {
			fileWriter := createFileWriter(config.File)
			writers = append(writers, fileWriter)
		}
	default: // stdout
		writers = append(writers, os.Stdout)
	}

	var output io.Writer
	if len(writers) == 1 {
		output = writers[0]
	} else {
		output = io.MultiWriter(writers...)
	}

	// 포맷 설정
	if config.Format == "text" || config.Format == "console" {
		output = zerolog.ConsoleWriter{
			Out:        output,
			TimeFormat: time.Kitchen,
			NoColor:    false,
		}
	}

	// 기본 컨텍스트 설정
	ctx := zerolog.New(output).With().Timestamp()

	// 기본 필드 추가
	if config.DefaultFields != nil {
		for k, v := range config.DefaultFields {
			ctx = ctx.Interface(k, v)
		}
	}

	// 호스트 정보 추가
	if hostname, err := os.Hostname(); err == nil {
		ctx = ctx.Str("hostname", hostname)
	}

	// 프로세스 정보 추가
	ctx = ctx.Int("pid", os.Getpid())

	logger = ctx.Logger()

	// 샘플링 설정
	if config.Sampling.Enabled {
		logger = logger.Sample(&zerolog.BasicSampler{
			N: uint32(config.Sampling.Initial),
		})
	}

	// 전역 로거 설정
	log.Logger = logger
	globalLogger = &ZeroLogger{
		logger: logger,
		fields: make(map[string]interface{}),
	}

	return nil
}

// GetLogger returns the global logger instance.
// If not initialized, it creates a default logger with INFO level and JSON format.
func GetLogger() Logger {
	once.Do(func() {
		// 기본 설정으로 초기화
		if err := InitLogger(LogConfig{
			Level:  LevelInfo,
			Format: "json",
			Output: "stdout",
		}); err != nil {
			// 기본 로거 초기화 실패 시에도 전역 로거는 사용 가능하도록 유지
			// 실패한 경우에도 globalLogger는 이미 유효한 상태이므로 계속 진행
			if globalLogger != nil {
				globalLogger.Error("Failed to initialize default logger", Field{Key: "error", Value: err})
			}
		}
	})
	return globalLogger
}

// NewLogger creates a new logger with the specified component name.
// The returned logger includes a "component" field with the given name.
func NewLogger(name string) Logger {
	baseLogger := GetLogger()
	return baseLogger.WithField("component", name)
}

// parseLevel converts LogLevel to zerolog.Level.
func parseLevel(level LogLevel) zerolog.Level {
	switch level {
	case LevelDebug:
		return zerolog.DebugLevel
	case LevelInfo:
		return zerolog.InfoLevel
	case LevelWarn:
		return zerolog.WarnLevel
	case LevelError:
		return zerolog.ErrorLevel
	case LevelFatal:
		return zerolog.FatalLevel
	case LevelPanic:
		return zerolog.PanicLevel
	default:
		return zerolog.InfoLevel
	}
}

// createFileWriter creates a file writer with log rotation support.
// It uses lumberjack for automatic log rotation based on size, age, and backup count.
func createFileWriter(config FileConfig) io.Writer {
	// 디렉토리 생성
	dir := filepath.Dir(config.Path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		log.Error().Err(err).Str(fieldPath, dir).Msg("Failed to create log directory")
	}

	// lumberjack을 사용한 로그 로테이션
	return &lumberjack.Logger{
		Filename:   config.Path,
		MaxSize:    config.MaxSize, // megabytes
		MaxBackups: config.MaxBackups,
		MaxAge:     config.MaxAge, // days
		Compress:   config.Compress,
		LocalTime:  true,
	}
}

// ZeroLogger 메서드 구현

// Debug performs debug operation
func (zl *ZeroLogger) Debug(msg string, fields ...Field) {
	event := zl.logger.Debug()
	zl.applyFields(event, fields...)
	event.Msg(msg)
}

// Info logs an informational message
func (zl *ZeroLogger) Info(msg string, fields ...Field) {
	event := zl.logger.Info()
	zl.applyFields(event, fields...)
	event.Msg(msg)
}

// Warn logs a warning message
func (zl *ZeroLogger) Warn(msg string, fields ...Field) {
	event := zl.logger.Warn()
	zl.applyFields(event, fields...)
	event.Msg(msg)
}

// Error logs an error message
func (zl *ZeroLogger) Error(msg string, fields ...Field) {
	event := zl.logger.Error()
	zl.applyFields(event, fields...)
	event.Msg(msg)
}

// Fatal logs a fatal message and exits
func (zl *ZeroLogger) Fatal(msg string, fields ...Field) {
	event := zl.logger.Fatal()
	zl.applyFields(event, fields...)
	event.Msg(msg)
}

// Panic logs a panic message
func (zl *ZeroLogger) Panic(msg string, fields ...Field) {
	event := zl.logger.Panic()
	zl.applyFields(event, fields...)
	event.Msg(msg)
}

// WithContext creates a logger with context information
func (zl *ZeroLogger) WithContext(ctx context.Context) Logger {
	// 컨텍스트에서 값 추출
	newLogger := zl.logger.With().Logger()

	// Request ID 추출
	if reqID := ctx.Value(fieldRequestID); reqID != nil {
		newLogger = newLogger.With().Str(fieldRequestID, fmt.Sprintf("%v", reqID)).Logger()
	}

	// User ID 추출
	if userID := ctx.Value(fieldUserID); userID != nil {
		newLogger = newLogger.With().Str(fieldUserID, fmt.Sprintf("%v", userID)).Logger()
	}

	return &ZeroLogger{
		logger: newLogger,
		fields: zl.copyFields(),
	}
}

// WithFields creates a logger with multiple fields
func (zl *ZeroLogger) WithFields(fields ...Field) Logger {
	newLogger := zl.logger.With().Logger()
	newFields := zl.copyFields()

	for _, field := range fields {
		newFields[field.Key] = field.Value
		newLogger = newLogger.With().Interface(field.Key, field.Value).Logger()
	}

	return &ZeroLogger{
		logger: newLogger,
		fields: newFields,
	}
}

// WithField creates a logger with a single field
func (zl *ZeroLogger) WithField(key string, value interface{}) Logger {
	return zl.WithFields(Field{Key: key, Value: value})
}

// Printf implements the Printf method for compatibility
func (zl *ZeroLogger) Printf(format string, v ...interface{}) {
	zl.logger.Info().Msgf(format, v...)
}

// applyFields applies both stored and new fields to a log event.
func (zl *ZeroLogger) applyFields(event *zerolog.Event, fields ...Field) {
	// 기존 필드 적용
	zl.mu.RLock()
	for k, v := range zl.fields {
		event = event.Interface(k, v)
	}
	zl.mu.RUnlock()

	// 새 필드 적용
	for _, field := range fields {
		event = event.Interface(field.Key, field.Value)
	}
}

// copyFields creates a copy of the logger's fields map.
func (zl *ZeroLogger) copyFields() map[string]interface{} {
	zl.mu.RLock()
	defer zl.mu.RUnlock()

	copied := make(map[string]interface{})
	for k, v := range zl.fields {
		copied[k] = v
	}
	return copied
}

// 편의 함수들

// String creates a string field
func String(key, value string) Field {
	return F(key, value)
}

// Int creates an int field
func Int(key string, value int) Field {
	return F(key, value)
}

// Int64 creates an int64 field
func Int64(key string, value int64) Field {
	return F(key, value)
}

// Uint64 creates a uint64 field
func Uint64(key string, value uint64) Field {
	return F(key, value)
}

// Float64 creates a float64 field
func Float64(key string, value float64) Field {
	return F(key, value)
}

// Bool creates a bool field
func Bool(key string, value bool) Field {
	return F(key, value)
}

// Time creates a time field
func Time(key string, value time.Time) Field {
	return F(key, value)
}

// Duration creates a duration field
func Duration(key string, value time.Duration) Field {
	return F(key, value.String())
}

// ErrorField creates an error field
func ErrorField(err error) Field {
	if err == nil {
		return F(fieldError, nil)
	}
	return F(fieldError, err.Error())
}

// Uint32 creates a uint32 field
func Uint32(key string, value uint32) Field {
	return F(key, value)
}

// Strings creates a string slice field
func Strings(key string, value []string) Field {
	return F(key, value)
}

// Any creates a field with any value
func Any(key string, value interface{}) Field {
	return F(key, value)
}

// Debug logs a message at debug level using the global logger.
func Debug(msg string, fields ...Field) {
	GetLogger().Debug(msg, fields...)
}

// Info logs a message at info level using the global logger.
func Info(msg string, fields ...Field) {
	GetLogger().Info(msg, fields...)
}

// Warn logs a message at warning level using the global logger.
func Warn(msg string, fields ...Field) {
	GetLogger().Warn(msg, fields...)
}

// Error logs a message at error level using the global logger.
func Error(msg string, fields ...Field) {
	GetLogger().Error(msg, fields...)
}

// Fatal logs a message at fatal level using the global logger and exits the program.
func Fatal(msg string, fields ...Field) {
	GetLogger().Fatal(msg, fields...)
}

// Panic logs a message at panic level using the global logger and panics.
func Panic(msg string, fields ...Field) {
	GetLogger().Panic(msg, fields...)
}

// WithContext returns a logger that extracts values from the context.
// It looks for "request_id" and "user_id" in the context.
func WithContext(ctx context.Context) Logger {
	return GetLogger().WithContext(ctx)
}

// WithFields returns a logger with additional fields.
func WithFields(fields ...Field) Logger {
	return GetLogger().WithFields(fields...)
}

// WithField returns a logger with a single additional field.
func WithField(key string, value interface{}) Logger {
	return GetLogger().WithField(key, value)
}
