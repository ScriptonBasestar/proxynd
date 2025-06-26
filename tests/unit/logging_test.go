package unit

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
	
	"github.com/go-playground/assert/v2"
	"github.com/gofiber/fiber/v2"
	"proxynd/logging"
	"proxynd/middlewares"
)

func TestLogger_Basic(t *testing.T) {
	// 버퍼로 출력 캡처
	var buf bytes.Buffer
	
	// 테스트용 로그 설정
	config := logging.LogConfig{
		Level:  logging.LevelDebug,
		Format: "json",
		Output: "stdout",
	}
	
	// 로거 초기화 (실제로는 버퍼로 리다이렉트해야 함)
	err := logging.InitLogger(config)
	assert.Equal(t, nil, err)
	
	// 로거 가져오기
	logger := logging.GetLogger()
	
	// 각 레벨 테스트
	logger.Debug("Debug message", logging.F("key", "value"))
	logger.Info("Info message", logging.F("number", 42))
	logger.Warn("Warn message", logging.F("flag", true))
	logger.Error("Error message", logging.F("error", "test error"))
	
	// 필드 포함 로거
	loggerWithFields := logger.WithFields(
		logging.F("component", "test"),
		logging.F("version", "1.0.0"),
	)
	
	loggerWithFields.Info("Message with fields")
	
	// 단일 필드 추가
	loggerWithField := logger.WithField("user_id", "12345")
	loggerWithField.Info("Message with single field")
}

func TestLogger_Context(t *testing.T) {
	// 컨텍스트 생성
	ctx := context.Background()
	ctx = context.WithValue(ctx, "request_id", "test-request-123")
	ctx = context.WithValue(ctx, "user_id", "user-456")
	
	// 컨텍스트 로거
	logger := logging.GetLogger().WithContext(ctx)
	logger.Info("Message with context")
}

func TestLogger_Printf(t *testing.T) {
	logger := logging.GetLogger()
	
	// Printf 호환성 테스트
	logger.Printf("Test message: %s, number: %d", "hello", 42)
}

func TestLogger_Migration(t *testing.T) {
	// 레거시 로거 생성
	legacy := logging.NewLegacyLogger("test-component")
	
	// 기존 log 패키지 호환 메서드 테스트
	legacy.Printf("Printf test: %s", "value")
	legacy.Println("Println test")
	legacy.Print("Print test")
}

func TestLogger_EnvConfig(t *testing.T) {
	// 환경 변수 설정
	os.Setenv("LOG_LEVEL", "debug")
	os.Setenv("LOG_FORMAT", "json")
	os.Setenv("LOG_OUTPUT", "stdout")
	os.Setenv("ENVIRONMENT", "test")
	os.Setenv("VERSION", "1.0.0")
	
	defer func() {
		os.Unsetenv("LOG_LEVEL")
		os.Unsetenv("LOG_FORMAT")
		os.Unsetenv("LOG_OUTPUT")
		os.Unsetenv("ENVIRONMENT")
		os.Unsetenv("VERSION")
	}()
	
	// 환경 변수에서 설정 로드
	config := logging.LoadConfigFromEnv()
	
	assert.Equal(t, logging.LevelDebug, config.Level)
	assert.Equal(t, "json", config.Format)
	assert.Equal(t, "stdout", config.Output)
	assert.Equal(t, "test", config.DefaultFields["environment"])
	assert.Equal(t, "1.0.0", config.DefaultFields["version"])
}

func TestLoggingMiddleware(t *testing.T) {
	// Fiber 앱 생성
	app := fiber.New()
	
	// 로깅 미들웨어 적용
	app.Use(logging.RequestLogger())
	app.Use(logging.New())
	app.Use(logging.ErrorLogger())
	
	// 테스트 핸들러
	app.Get("/test", func(c *fiber.Ctx) error {
		logger := logging.GetRequestLogger(c)
		logger.Info("Handler executed")
		
		// 컨텍스트에 값 추가
		c.Locals("cache_hit", true)
		c.Locals("username", "testuser")
		
		return c.JSON(fiber.Map{
			"message": "success",
		})
	})
	
	app.Get("/error", func(c *fiber.Ctx) error {
		return fiber.NewError(500, "Test error")
	})
	
	// 테스트 실행
	tests := []struct {
		name           string
		path           string
		expectedStatus int
	}{
		{"Success request", "/test", 200},
		{"Error request", "/error", 500},
		{"Not found", "/notfound", 404},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			req.Header.Set("X-Request-ID", "test-req-123")
			
			resp, err := app.Test(req)
			assert.Equal(t, nil, err)
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
		})
	}
}

func TestStructuredAccessLog(t *testing.T) {
	// Fiber 앱 생성
	app := fiber.New()
	
	// 구조화된 액세스 로그 미들웨어
	app.Use(logging.RequestLogger())
	app.Use(middlewares.StructuredAccessLog())
	
	// 프록시 핸들러 시뮬레이션
	app.Get("/proxy/:type/*", func(c *fiber.Ctx) error {
		// 프록시 정보 설정
		c.Locals("cache_hit", true)
		c.Locals("cache_backend", "file")
		c.Locals("upstream", "registry.npmjs.org")
		c.Locals("upstream_duration", 150*time.Millisecond)
		
		return c.JSON(fiber.Map{
			"package": "test-package",
			"version": "1.0.0",
		})
	})
	
	// 테스트 요청
	req := httptest.NewRequest("GET", "/proxy/npm/test-package", nil)
	req.Header.Set("User-Agent", "npm/8.0.0")
	
	resp, err := app.Test(req)
	assert.Equal(t, nil, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestRecoveryLogger(t *testing.T) {
	// Fiber 앱 생성
	app := fiber.New()
	
	// 복구 로거 미들웨어
	app.Use(logging.RequestLogger())
	app.Use(logging.RecoveryLogger())
	
	// 패닉을 발생시키는 핸들러
	app.Get("/panic", func(c *fiber.Ctx) error {
		panic("test panic")
	})
	
	// 테스트 요청
	req := httptest.NewRequest("GET", "/panic", nil)
	resp, err := app.Test(req, -1)
	
	assert.Equal(t, nil, err)
	assert.Equal(t, 500, resp.StatusCode)
	
	// 응답 본문 확인
	body, _ := io.ReadAll(resp.Body)
	assert.Equal(t, true, strings.Contains(string(body), "Internal Server Error"))
}

func TestLoggerHelpers(t *testing.T) {
	// 편의 함수 테스트
	logging.Debug("Debug helper", logging.F("test", true))
	logging.Info("Info helper", logging.F("count", 10))
	logging.Warn("Warn helper", logging.F("warning", "test"))
	logging.Error("Error helper", logging.F("error", "test error"))
	
	// WithFields 테스트
	logger := logging.WithFields(
		logging.F("component", "test"),
		logging.F("method", "TestLoggerHelpers"),
	)
	logger.Info("Message with fields")
	
	// WithField 테스트
	logger2 := logging.WithField("single", "value")
	logger2.Info("Message with single field")
}

func TestLogConfig_Validation(t *testing.T) {
	tests := []struct {
		name    string
		config  logging.LogConfig
		wantErr bool
	}{
		{
			name: "Valid config",
			config: logging.LogConfig{
				Level:  logging.LevelInfo,
				Format: "json",
				Output: "stdout",
			},
			wantErr: false,
		},
		{
			name: "Invalid level",
			config: logging.LogConfig{
				Level:  "invalid",
				Format: "json",
				Output: "stdout",
			},
			wantErr: true,
		},
		{
			name: "Invalid format",
			config: logging.LogConfig{
				Level:  logging.LevelInfo,
				Format: "invalid",
				Output: "stdout",
			},
			wantErr: true,
		},
		{
			name: "File output without path",
			config: logging.LogConfig{
				Level:  logging.LevelInfo,
				Format: "json",
				Output: "file",
				File:   logging.FileConfig{},
			},
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := logging.ValidateConfig(tt.config)
			if tt.wantErr {
				assert.NotEqual(t, nil, err)
			} else {
				assert.Equal(t, nil, err)
			}
		})
	}
}

