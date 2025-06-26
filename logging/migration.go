package logging

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

// StdLogger 표준 log.Logger 호환 래퍼
type StdLogger struct {
	logger Logger
}

// NewStdLogger 표준 로거 호환 래퍼 생성
func NewStdLogger(logger Logger) *log.Logger {
	std := &StdLogger{logger: logger}
	return log.New(std, "", 0)
}

// Write io.Writer 인터페이스 구현
func (s *StdLogger) Write(p []byte) (n int, err error) {
	// 줄바꿈 제거
	msg := string(p)
	if len(msg) > 0 && msg[len(msg)-1] == '\n' {
		msg = msg[:len(msg)-1]
	}
	
	// 로그 레벨 추측
	switch {
	case contains(msg, "ERROR", "error", "Error"):
		s.logger.Error(msg)
	case contains(msg, "WARN", "warn", "Warn", "WARNING", "warning"):
		s.logger.Warn(msg)
	case contains(msg, "DEBUG", "debug", "Debug"):
		s.logger.Debug(msg)
	case contains(msg, "FATAL", "fatal", "Fatal"):
		s.logger.Fatal(msg)
	case contains(msg, "PANIC", "panic", "Panic"):
		s.logger.Panic(msg)
	default:
		s.logger.Info(msg)
	}
	
	return len(p), nil
}

// contains 문자열에 키워드 포함 여부 확인
func contains(s string, keywords ...string) bool {
	for _, keyword := range keywords {
		if len(s) >= len(keyword) && s[:len(keyword)] == keyword {
			return true
		}
	}
	return false
}

// ReplaceStdLogger 표준 로거 교체
func ReplaceStdLogger(logger Logger) {
	std := &StdLogger{logger: logger}
	log.SetOutput(std)
	log.SetFlags(0)
}

// LoggerFunc 함수형 로거 어댑터
type LoggerFunc func(format string, v ...interface{})

// Printf 구현
func (f LoggerFunc) Printf(format string, v ...interface{}) {
	f(format, v...)
}

// CreateLoggerFunc 로거 함수 생성
func CreateLoggerFunc(logger Logger, level LogLevel) LoggerFunc {
	return func(format string, v ...interface{}) {
		msg := fmt.Sprintf(format, v...)
		switch level {
		case LevelDebug:
			logger.Debug(msg)
		case LevelInfo:
			logger.Info(msg)
		case LevelWarn:
			logger.Warn(msg)
		case LevelError:
			logger.Error(msg)
		default:
			logger.Info(msg)
		}
	}
}

// LegacyLogger 기존 코드와의 호환성을 위한 래퍼
type LegacyLogger struct {
	logger Logger
}

// NewLegacyLogger 레거시 로거 생성
func NewLegacyLogger(name string) *LegacyLogger {
	return &LegacyLogger{
		logger: NewLogger(name),
	}
}

// Printf log.Printf 호환
func (l *LegacyLogger) Printf(format string, v ...interface{}) {
	l.logger.Printf(format, v...)
}

// Println log.Println 호환
func (l *LegacyLogger) Println(v ...interface{}) {
	l.logger.Info(fmt.Sprint(v...))
}

// Print log.Print 호환
func (l *LegacyLogger) Print(v ...interface{}) {
	l.logger.Info(fmt.Sprint(v...))
}

// Fatal log.Fatal 호환
func (l *LegacyLogger) Fatal(v ...interface{}) {
	l.logger.Fatal(fmt.Sprint(v...))
}

// Fatalf log.Fatalf 호환
func (l *LegacyLogger) Fatalf(format string, v ...interface{}) {
	l.logger.Fatal(fmt.Sprintf(format, v...))
}

// Fatalln log.Fatalln 호환
func (l *LegacyLogger) Fatalln(v ...interface{}) {
	l.logger.Fatal(fmt.Sprint(v...))
}

// Panic log.Panic 호환
func (l *LegacyLogger) Panic(v ...interface{}) {
	l.logger.Panic(fmt.Sprint(v...))
}

// Panicf log.Panicf 호환
func (l *LegacyLogger) Panicf(format string, v ...interface{}) {
	l.logger.Panic(fmt.Sprintf(format, v...))
}

// Panicln log.Panicln 호환
func (l *LegacyLogger) Panicln(v ...interface{}) {
	l.logger.Panic(fmt.Sprint(v...))
}

// MigrateEnvConfig 환경 변수 기반 로그 설정 마이그레이션
func MigrateEnvConfig() LogConfig {
	config := LogConfig{
		Level:      LevelInfo,
		Format:     "json",
		Output:     "stdout",
		TimeFormat: time.RFC3339,
		File: FileConfig{
			MaxSize:    100,
			MaxBackups: 10,
			MaxAge:     30,
			Compress:   true,
		},
	}
	
	// 로그 레벨
	if level := os.Getenv("LOG_LEVEL"); level != "" {
		config.Level = LogLevel(strings.ToLower(level))
	}
	
	// 로그 포맷
	if format := os.Getenv("LOG_FORMAT"); format != "" {
		config.Format = strings.ToLower(format)
	}
	
	// 로그 출력
	if output := os.Getenv("LOG_OUTPUT"); output != "" {
		config.Output = strings.ToLower(output)
	}
	
	// 파일 로그 설정
	if logFile := os.Getenv("LOG_FILE"); logFile != "" {
		config.File.Path = logFile
		if config.Output == "stdout" {
			config.Output = "both" // 파일과 콘솔 모두 출력
		}
	}
	
	// 기본 필드 설정
	config.DefaultFields = map[string]interface{}{
		"service": "proxynd",
	}
	
	// 환경 정보 추가
	if env := os.Getenv("ENVIRONMENT"); env != "" {
		config.DefaultFields["environment"] = env
	}
	
	// 버전 정보 추가
	if version := os.Getenv("VERSION"); version != "" {
		config.DefaultFields["version"] = version
	}
	
	return config
}

// InitFromEnv 환경 변수로부터 로거 초기화
func InitFromEnv() error {
	config := MigrateEnvConfig()
	return InitLogger(config)
}

