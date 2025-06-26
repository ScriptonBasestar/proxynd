package logging

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// LoadConfigFromEnv 환경 변수에서 로그 설정 로드
func LoadConfigFromEnv() LogConfig {
	config := LogConfig{
		Level:      LevelInfo,
		Format:     "json",
		Output:     "stdout",
		TimeFormat: "2006-01-02T15:04:05.000Z07:00",
		File: FileConfig{
			MaxSize:    100, // 100MB
			MaxBackups: 10,
			MaxAge:     30, // 30 days
			Compress:   true,
		},
		DefaultFields: make(map[string]interface{}),
	}

	// 로그 레벨
	if level := os.Getenv("LOG_LEVEL"); level != "" {
		switch strings.ToLower(level) {
		case "debug":
			config.Level = LevelDebug
		case "info":
			config.Level = LevelInfo
		case "warn", "warning":
			config.Level = LevelWarn
		case "error":
			config.Level = LevelError
		case "fatal":
			config.Level = LevelFatal
		case "panic":
			config.Level = LevelPanic
		}
	}

	// 로그 포맷
	if format := os.Getenv("LOG_FORMAT"); format != "" {
		config.Format = strings.ToLower(format)
	}

	// 로그 출력
	if output := os.Getenv("LOG_OUTPUT"); output != "" {
		config.Output = strings.ToLower(output)
	}

	// 로그 파일
	if logFile := os.Getenv("LOG_FILE"); logFile != "" {
		config.File.Path = logFile
		if config.Output == "stdout" {
			config.Output = "both"
		}
	}

	// 액세스 로그 파일
	if accessLogPath := os.Getenv("ACCESS_LOG_PATH"); accessLogPath != "" {
		// 액세스 로그는 별도 처리
	}

	// 시간 포맷
	if timeFormat := os.Getenv("LOG_TIME_FORMAT"); timeFormat != "" {
		config.TimeFormat = timeFormat
	}

	// 기본 필드 설정
	config.DefaultFields["service"] = "proxynd"

	// 환경 정보
	if env := os.Getenv("ENVIRONMENT"); env != "" {
		config.DefaultFields["environment"] = env
	}

	// 버전 정보
	if version := os.Getenv("VERSION"); version != "" {
		config.DefaultFields["version"] = version
	}

	// 호스트 정보
	if hostname, err := os.Hostname(); err == nil {
		config.DefaultFields["hostname"] = hostname
	}

	// 인스턴스 ID (컨테이너 환경)
	if instanceID := os.Getenv("INSTANCE_ID"); instanceID != "" {
		config.DefaultFields["instance_id"] = instanceID
	}

	// 샘플링 설정
	if sampling := os.Getenv("LOG_SAMPLING"); sampling == "true" {
		config.Sampling.Enabled = true
		config.Sampling.Initial = 100
		config.Sampling.Thereafter = 100
	}

	return config
}

// LoadConfigFromFile 파일에서 로그 설정 로드
func LoadConfigFromFile(configPath string) (LogConfig, error) {
	config := LogConfig{
		Level:         LevelInfo,
		Format:        "json",
		Output:        "stdout",
		DefaultFields: make(map[string]interface{}),
	}

	// YAML 파일 읽기
	data, err := os.ReadFile(configPath)
	if err != nil {
		return config, err
	}

	// YAML 파싱
	if err := yaml.Unmarshal(data, &config); err != nil {
		return config, err
	}

	return config, nil
}

// LoadConfigFromUnified 통합 설정에서 로그 설정 추출
func LoadConfigFromUnified(unifiedConfig interface{}) LogConfig {
	// TODO: 통합 설정 구조체에서 로그 설정 추출
	// 현재는 기본값 반환
	return LoadConfigFromEnv()
}

// ValidateConfig 로그 설정 검증
func ValidateConfig(config LogConfig) error {
	// 로그 레벨 검증
	validLevels := map[LogLevel]bool{
		LevelDebug: true,
		LevelInfo:  true,
		LevelWarn:  true,
		LevelError: true,
		LevelFatal: true,
		LevelPanic: true,
	}

	if !validLevels[config.Level] {
		return fmt.Errorf("invalid log level: %s", config.Level)
	}

	// 포맷 검증
	validFormats := map[string]bool{
		"json":    true,
		"text":    true,
		"console": true,
	}

	if !validFormats[config.Format] {
		return fmt.Errorf("invalid log format: %s", config.Format)
	}

	// 출력 검증
	validOutputs := map[string]bool{
		"stdout": true,
		"stderr": true,
		"file":   true,
		"both":   true,
	}

	if !validOutputs[config.Output] {
		return fmt.Errorf("invalid log output: %s", config.Output)
	}

	// 파일 출력 설정 검증
	if config.Output == "file" || config.Output == "both" {
		if config.File.Path == "" {
			return fmt.Errorf("log file path is required for file output")
		}

		// 디렉토리 존재 확인
		dir := filepath.Dir(config.File.Path)
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			// 디렉토리 생성 시도
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("failed to create log directory: %w", err)
			}
		}
	}

	return nil
}

// SetupLogging 로깅 시스템 설정
func SetupLogging() error {
	// 환경 변수에서 설정 로드
	config := LoadConfigFromEnv()

	// 설정 검증
	if err := ValidateConfig(config); err != nil {
		return fmt.Errorf("invalid log config: %w", err)
	}

	// 로거 초기화
	if err := InitLogger(config); err != nil {
		return fmt.Errorf("failed to init logger: %w", err)
	}

	// 표준 로거 교체
	ReplaceStdLogger(GetLogger())

	// 초기화 로그
	Info("Logging system initialized",
		F("level", config.Level),
		F("format", config.Format),
		F("output", config.Output),
	)

	return nil
}
