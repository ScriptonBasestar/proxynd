package middlewares

import (
	"path/filepath"
	"time"

	"github.com/gofiber/fiber/v2"

	"proxynd/logging"
)

// StructuredAccessLogConfig 구조화된 액세스 로그 설정
type StructuredAccessLogConfig struct {
	// Logger 사용할 로거
	Logger logging.Logger

	// SkipPaths 로깅을 건너뛸 경로
	SkipPaths []string

	// LogLevel 액세스 로그 레벨
	LogLevel logging.LogLevel

	// IncludeRequestBody 요청 본문 포함 여부
	IncludeRequestBody bool

	// IncludeResponseBody 응답 본문 포함 여부
	IncludeResponseBody bool

	// MaxBodySize 로깅할 최대 본문 크기
	MaxBodySize int
}

// DefaultStructuredAccessLogConfig 기본 설정
var DefaultStructuredAccessLogConfig = StructuredAccessLogConfig{
	LogLevel: logging.LevelInfo,
	SkipPaths: []string{
		"/healthz",
		"/metrics",
		"/health/live",
		"/health/ready",
	},
	MaxBodySize: 1024 * 10, // 10KB
}

// StructuredAccessLog 구조화된 액세스 로그 미들웨어
func StructuredAccessLog(config ...StructuredAccessLogConfig) fiber.Handler {
	cfg := DefaultStructuredAccessLogConfig
	if len(config) > 0 {
		cfg = config[0]
	}

	if cfg.Logger == nil {
		cfg.Logger = logging.NewLogger("access")
	}

	return func(c *fiber.Ctx) error {
		// 경로 확인
		for _, skip := range cfg.SkipPaths {
			if c.Path() == skip {
				return c.Next()
			}
		}

		// 시작 시간
		start := time.Now()

		// 요청 정보 수집
		fields := []logging.Field{
			logging.F("method", c.Method()),
			logging.F("path", c.Path()),
			logging.F("ip", c.IP()),
			logging.F("user_agent", c.Get("User-Agent")),
			logging.F("referer", c.Get("Referer")),
		}

		// 요청 ID
		if reqID := c.Get("X-Request-ID"); reqID != "" {
			fields = append(fields, logging.F("request_id", reqID))
		}

		// 프록시 정보
		if proxyType := c.Params("type"); proxyType != "" {
			fields = append(fields, logging.F("proxy_type", proxyType))
		}

		// 패키지 경로
		if packagePath := c.Params("*"); packagePath != "" {
			fields = append(fields, logging.F("package_path", packagePath))
		}

		// 요청 크기
		if c.Request().Header.ContentLength() > 0 {
			fields = append(fields, logging.F("request_size", c.Request().Header.ContentLength()))
		}

		// 인증 정보
		if username := c.Locals("username"); username != nil {
			fields = append(fields, logging.F("username", username))
		}

		// 다음 핸들러 실행
		err := c.Next()

		// 응답 정보 수집
		duration := time.Since(start)
		statusCode := c.Response().StatusCode()

		fields = append(fields,
			logging.F("status", statusCode),
			logging.F("duration_ms", duration.Milliseconds()),
			logging.F("duration", duration.String()),
		)

		// 응답 크기
		if size := len(c.Response().Body()); size > 0 {
			fields = append(fields, logging.F("response_size", size))
		}

		// 캐시 상태
		if cacheHit := c.Locals("cache_hit"); cacheHit != nil {
			fields = append(fields, logging.F("cache_hit", cacheHit))

			// 캐시 히트인 경우 절약된 대역폭
			if hit, ok := cacheHit.(bool); ok && hit {
				if size := len(c.Response().Body()); size > 0 {
					fields = append(fields, logging.F("bandwidth_saved", size))
				}
			}
		}

		// 캐시 백엔드
		if cacheBackend := c.Locals("cache_backend"); cacheBackend != nil {
			fields = append(fields, logging.F("cache_backend", cacheBackend))
		}

		// 업스트림 정보
		if upstream := c.Locals("upstream"); upstream != nil {
			fields = append(fields, logging.F("upstream", upstream))
		}

		// 업스트림 응답 시간
		if upstreamDuration := c.Locals("upstream_duration"); upstreamDuration != nil {
			if d, ok := upstreamDuration.(time.Duration); ok {
				fields = append(fields, logging.F("upstream_duration_ms", d.Milliseconds()))
			}
		}

		// 해시 검증 상태
		if hashVerified := c.Locals("hashVerified"); hashVerified != nil {
			fields = append(fields, logging.F("hash_verified", hashVerified))
		}

		// 패키지 검증 상태
		if packageVerified := c.Locals("packageVerified"); packageVerified != nil {
			fields = append(fields, logging.F("package_verified", packageVerified))
		}

		// 오류 정보
		if err != nil {
			fields = append(fields, logging.F("error", err.Error()))
		}

		// 요청 본문 (설정된 경우)
		if cfg.IncludeRequestBody && c.Method() != "GET" && c.Method() != "HEAD" {
			body := c.Body()
			if len(body) > 0 && len(body) <= cfg.MaxBodySize {
				fields = append(fields, logging.F("request_body", string(body)))
			}
		}

		// 로그 메시지 생성
		message := c.Method() + " " + c.Path()

		// 로그 레벨에 따라 로깅
		logger := cfg.Logger.WithFields(fields...)

		switch {
		case err != nil || statusCode >= 500:
			logger.Error(message)
		case statusCode >= 400:
			logger.Warn(message)
		default:
			switch cfg.LogLevel {
			case logging.LevelDebug:
				logger.Debug(message)
			case logging.LevelInfo:
				logger.Info(message)
			default:
				logger.Info(message)
			}
		}

		return err
	}
}

// MigrateAccessLog 기존 액세스 로그 설정을 구조화된 로깅으로 마이그레이션
func MigrateAccessLog(config AccessLogConfig) fiber.Handler {
	// 구조화된 로거 생성
	logger := logging.NewLogger("access")

	// 파일 출력이 설정된 경우
	filePath := filepath.Join(config.LogDir, config.LogFile)
	if filePath != "" && !config.ConsoleOutput {
		// 로그 설정 생성
		logConfig := logging.LogConfig{
			Level:  logging.LevelInfo,
			Format: "json",
			Output: "file",
			File: logging.FileConfig{
				Path:       filePath,
				MaxSize:    100,
				MaxBackups: 10,
				MaxAge:     30,
				Compress:   true,
			},
		}

		if config.Format == "text" {
			logConfig.Format = "text"
		}

		// 새 로거 초기화
		if err := logging.InitLogger(logConfig); err != nil {
			// 로거 초기화 실패 시 기본 로거 사용
			logger = logging.GetLogger()
		}
		logger = logging.GetLogger()
	}

	// 구조화된 액세스 로그 설정
	structuredConfig := StructuredAccessLogConfig{
		Logger:    logger,
		SkipPaths: DefaultStructuredAccessLogConfig.SkipPaths,
		LogLevel:  logging.LevelInfo,
	}

	return StructuredAccessLog(structuredConfig)
}
