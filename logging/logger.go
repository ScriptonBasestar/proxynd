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

// LogLevel 로그 레벨
type LogLevel string

const (
	LevelDebug LogLevel = "debug"
	LevelInfo  LogLevel = "info"
	LevelWarn  LogLevel = "warn"
	LevelError LogLevel = "error"
	LevelFatal LogLevel = "fatal"
	LevelPanic LogLevel = "panic"
)

// Logger 구조화된 로거 인터페이스
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

// Field 로그 필드
type Field struct {
	Key   string
	Value interface{}
}

// F 필드 생성 헬퍼
func F(key string, value interface{}) Field {
	return Field{Key: key, Value: value}
}

// ZeroLogger zerolog 기반 로거 구현
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

// LogConfig 로그 설정
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

// FileConfig 파일 로그 설정
type FileConfig struct {
	Path       string `json:"path" yaml:"path"`
	MaxSize    int    `json:"max_size" yaml:"max_size"` // MB
	MaxBackups int    `json:"max_backups" yaml:"max_backups"`
	MaxAge     int    `json:"max_age" yaml:"max_age"` // days
	Compress   bool   `json:"compress" yaml:"compress"`
}

// SamplingConfig 로그 샘플링 설정
type SamplingConfig struct {
	Enabled    bool `json:"enabled" yaml:"enabled"`
	Initial    int  `json:"initial" yaml:"initial"`
	Thereafter int  `json:"thereafter" yaml:"thereafter"`
}

// InitLogger 로거 초기화
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
	case "file":
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

// GetLogger 전역 로거 반환
func GetLogger() Logger {
	once.Do(func() {
		// 기본 설정으로 초기화
		if err := InitLogger(LogConfig{
			Level:  LevelInfo,
			Format: "json",
			Output: "stdout",
		}); err != nil {
			// 기본 로거 초기화 실패 시에도 전역 로거는 사용 가능하도록 유지
		}
	})
	return globalLogger
}

// NewLogger 새 로거 생성
func NewLogger(name string) Logger {
	baseLogger := GetLogger()
	return baseLogger.WithField("component", name)
}

// parseLevel 로그 레벨 파싱
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

// createFileWriter 파일 라이터 생성
func createFileWriter(config FileConfig) io.Writer {
	// 디렉토리 생성
	dir := filepath.Dir(config.Path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Error().Err(err).Str("path", dir).Msg("Failed to create log directory")
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

func (zl *ZeroLogger) Debug(msg string, fields ...Field) {
	event := zl.logger.Debug()
	zl.applyFields(event, fields...)
	event.Msg(msg)
}

func (zl *ZeroLogger) Info(msg string, fields ...Field) {
	event := zl.logger.Info()
	zl.applyFields(event, fields...)
	event.Msg(msg)
}

func (zl *ZeroLogger) Warn(msg string, fields ...Field) {
	event := zl.logger.Warn()
	zl.applyFields(event, fields...)
	event.Msg(msg)
}

func (zl *ZeroLogger) Error(msg string, fields ...Field) {
	event := zl.logger.Error()
	zl.applyFields(event, fields...)
	event.Msg(msg)
}

func (zl *ZeroLogger) Fatal(msg string, fields ...Field) {
	event := zl.logger.Fatal()
	zl.applyFields(event, fields...)
	event.Msg(msg)
}

func (zl *ZeroLogger) Panic(msg string, fields ...Field) {
	event := zl.logger.Panic()
	zl.applyFields(event, fields...)
	event.Msg(msg)
}

func (zl *ZeroLogger) WithContext(ctx context.Context) Logger {
	// 컨텍스트에서 값 추출
	newLogger := zl.logger.With().Logger()

	// Request ID 추출
	if reqID := ctx.Value("request_id"); reqID != nil {
		newLogger = newLogger.With().Str("request_id", fmt.Sprintf("%v", reqID)).Logger()
	}

	// User ID 추출
	if userID := ctx.Value("user_id"); userID != nil {
		newLogger = newLogger.With().Str("user_id", fmt.Sprintf("%v", userID)).Logger()
	}

	return &ZeroLogger{
		logger: newLogger,
		fields: zl.copyFields(),
	}
}

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

func (zl *ZeroLogger) WithField(key string, value interface{}) Logger {
	return zl.WithFields(Field{Key: key, Value: value})
}

func (zl *ZeroLogger) Printf(format string, v ...interface{}) {
	zl.logger.Info().Msgf(format, v...)
}

// applyFields 이벤트에 필드 적용
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

// copyFields 필드 복사
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

// Debug 디버그 로그
func Debug(msg string, fields ...Field) {
	GetLogger().Debug(msg, fields...)
}

// Info 정보 로그
func Info(msg string, fields ...Field) {
	GetLogger().Info(msg, fields...)
}

// Warn 경고 로그
func Warn(msg string, fields ...Field) {
	GetLogger().Warn(msg, fields...)
}

// Error 오류 로그
func Error(msg string, fields ...Field) {
	GetLogger().Error(msg, fields...)
}

// Fatal 치명적 오류 로그 (프로그램 종료)
func Fatal(msg string, fields ...Field) {
	GetLogger().Fatal(msg, fields...)
}

// Panic 패닉 로그
func Panic(msg string, fields ...Field) {
	GetLogger().Panic(msg, fields...)
}

// WithContext 컨텍스트 포함 로거
func WithContext(ctx context.Context) Logger {
	return GetLogger().WithContext(ctx)
}

// WithFields 필드 포함 로거
func WithFields(fields ...Field) Logger {
	return GetLogger().WithFields(fields...)
}

// WithField 단일 필드 포함 로거
func WithField(key string, value interface{}) Logger {
	return GetLogger().WithField(key, value)
}
