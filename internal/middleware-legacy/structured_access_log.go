package middlewares

import (
	"path/filepath"
	"time"

	"github.com/gofiber/fiber/v2"

	"proxynd/internal/logging"
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
		if shouldSkipPath(c.Path(), cfg.SkipPaths) {
			return c.Next()
		}

		// 시작 시간
		start := time.Now()

		// 요청 정보 수집
		fields := collectRequestFields(c)

		// 다음 핸들러 실행
		err := c.Next()

		// 응답 정보 추가
		duration := time.Since(start)
		fields = addResponseFields(fields, c, duration)

		// 추가 정보 수집
		fields = addLocalFields(fields, c)

		// 오류 정보
		if err != nil {
			fields = append(fields, logging.F("error", err.Error()))
		}

		// 요청 본문 추가
		if shouldLogRequestBody(cfg, c) {
			fields = addRequestBody(fields, c, cfg.MaxBodySize)
		}

		// 로그 기록
		logAccessEntry(cfg, c, fields, err)

		return err
	}
}

// shouldSkipPath 경로를 건너뛸지 확인
func shouldSkipPath(path string, skipPaths []string) bool {
	for _, skip := range skipPaths {
		if path == skip {
			return true
		}
	}
	return false
}

// collectRequestFields 요청 관련 필드 수집
func collectRequestFields(c *fiber.Ctx) []logging.Field {
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

	return fields
}

// addResponseFields 응답 관련 필드 추가
func addResponseFields(fields []logging.Field, c *fiber.Ctx, duration time.Duration) []logging.Field {
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

	return fields
}

// addLocalFields 로컬 변수에서 필드 추가
func addLocalFields(fields []logging.Field, c *fiber.Ctx) []logging.Field {
	// 캐시 관련 필드
	fields = addCacheFields(fields, c)

	// 업스트림 관련 필드
	fields = addUpstreamFields(fields, c)

	// 검증 관련 필드
	fields = addVerificationFields(fields, c)

	return fields
}

// addCacheFields 캐시 관련 필드 추가
func addCacheFields(fields []logging.Field, c *fiber.Ctx) []logging.Field {
	if cacheHit := c.Locals("cache_hit"); cacheHit != nil {
		fields = append(fields, logging.F("cache_hit", cacheHit))

		// 캐시 히트인 경우 절약된 대역폭
		if hit, ok := cacheHit.(bool); ok && hit {
			if size := len(c.Response().Body()); size > 0 {
				fields = append(fields, logging.F("bandwidth_saved", size))
			}
		}
	}

	if cacheBackend := c.Locals("cache_backend"); cacheBackend != nil {
		fields = append(fields, logging.F("cache_backend", cacheBackend))
	}

	return fields
}

// addUpstreamFields 업스트림 관련 필드 추가
func addUpstreamFields(fields []logging.Field, c *fiber.Ctx) []logging.Field {
	if upstream := c.Locals("upstream"); upstream != nil {
		fields = append(fields, logging.F("upstream", upstream))
	}

	if upstreamDuration := c.Locals("upstream_duration"); upstreamDuration != nil {
		if d, ok := upstreamDuration.(time.Duration); ok {
			fields = append(fields, logging.F("upstream_duration_ms", d.Milliseconds()))
		}
	}

	return fields
}

// addVerificationFields 검증 관련 필드 추가
func addVerificationFields(fields []logging.Field, c *fiber.Ctx) []logging.Field {
	if hashVerified := c.Locals("hashVerified"); hashVerified != nil {
		fields = append(fields, logging.F("hash_verified", hashVerified))
	}

	if packageVerified := c.Locals("packageVerified"); packageVerified != nil {
		fields = append(fields, logging.F("package_verified", packageVerified))
	}

	return fields
}

// shouldLogRequestBody 요청 본문을 로깅할지 확인
func shouldLogRequestBody(cfg StructuredAccessLogConfig, c *fiber.Ctx) bool {
	return cfg.IncludeRequestBody && c.Method() != "GET" && c.Method() != "HEAD"
}

// addRequestBody 요청 본문 추가
func addRequestBody(fields []logging.Field, c *fiber.Ctx, maxSize int) []logging.Field {
	body := c.Body()
	if len(body) > 0 && len(body) <= maxSize {
		fields = append(fields, logging.F("request_body", string(body)))
	}
	return fields
}

// logAccessEntry 액세스 로그 기록
func logAccessEntry(cfg StructuredAccessLogConfig, c *fiber.Ctx, fields []logging.Field, err error) {
	message := c.Method() + " " + c.Path()
	logger := cfg.Logger.WithFields(fields...)
	statusCode := c.Response().StatusCode()

	// 로그 레벨 결정
	logLevel := determineLogLevel(statusCode, err != nil, cfg.LogLevel)

	switch logLevel {
	case logging.LevelError:
		logger.Error(message)
	case logging.LevelWarn:
		logger.Warn(message)
	case logging.LevelDebug:
		logger.Debug(message)
	default:
		logger.Info(message)
	}
}

// determineLogLevel 로그 레벨 결정
func determineLogLevel(statusCode int, hasError bool, configLevel logging.LogLevel) logging.LogLevel {
	if hasError || statusCode >= 500 {
		return logging.LevelError
	}
	if statusCode >= 400 {
		return logging.LevelWarn
	}
	return configLevel
}

// LegacyAccessLogConfig 기존 액세스 로그 설정 (마이그레이션용)
type LegacyAccessLogConfig struct {
	LogDir        string
	LogFile       string
	ConsoleOutput bool
	Format        string
}

// MigrateAccessLog 기존 액세스 로그 설정을 구조화된 로깅으로 마이그레이션
func MigrateAccessLog(config LegacyAccessLogConfig) fiber.Handler {
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
			// 로거 초기화 실패 시 기본 로거 사용, 에러 로깅
			logger = logging.GetLogger()
			logger.Error("Failed to initialize logger", logging.F("error", err.Error()))
		} else {
			logger = logging.GetLogger()
		}
	}

	// 구조화된 액세스 로그 설정
	structuredConfig := StructuredAccessLogConfig{
		Logger:    logger,
		SkipPaths: DefaultStructuredAccessLogConfig.SkipPaths,
		LogLevel:  logging.LevelInfo,
	}

	return StructuredAccessLog(structuredConfig)
}
