package plugins

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"proxynd/internal/logging"
)

// PluginMiddleware 플러그인 시스템 미들웨어
type PluginMiddleware struct {
	manager *PluginManager
	logger  logging.Logger
}

// PluginMiddlewareConfig 플러그인 미들웨어 설정
type PluginMiddlewareConfig struct {
	BasePath      string        `json:"base_path" yaml:"base_path"`
	DefaultMode   OperationMode `json:"default_mode" yaml:"default_mode"`
	Timeout       time.Duration `json:"timeout" yaml:"timeout"`
	EnableMetrics bool          `json:"enable_metrics" yaml:"enable_metrics"`
	EnableLogging bool          `json:"enable_logging" yaml:"enable_logging"`
}

// NewPluginMiddleware 새로운 플러그인 미들웨어 생성
func NewPluginMiddleware(manager *PluginManager, config PluginMiddlewareConfig) *PluginMiddleware {
	if config.BasePath == "" {
		config.BasePath = "/proxy"
	}
	if config.DefaultMode == "" {
		config.DefaultMode = ProxyMode
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	return &PluginMiddleware{
		manager: manager,
		logger:  logging.GetLogger(),
	}
}

// Handler 플러그인 라우팅 핸들러
func (pm *PluginMiddleware) Handler(config PluginMiddlewareConfig) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		// 요청 경로 파싱
		path := c.Path()
		if !strings.HasPrefix(path, config.BasePath) {
			return c.Next()
		}

		// 패키지 타입 추출
		packageType, mode, err := pm.parseRequest(c, config)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}

		// 타임아웃 설정
		ctx, cancel := context.WithTimeout(c.Context(), config.Timeout)
		defer cancel()
		c.SetUserContext(ctx)

		// 로깅
		if config.EnableLogging {
			pm.logger.Info("Plugin request started",
				logging.F("package_type", packageType),
				logging.F("mode", string(mode)),
				logging.F("path", c.Path()),
				logging.F("method", c.Method()),
				logging.F("client_ip", c.IP()))
		}

		// 요청 처리
		err = pm.manager.HandleRequest(c, packageType, mode)

		// 메트릭 수집
		if config.EnableMetrics {
			duration := time.Since(start)
			pm.recordMetrics(packageType, string(mode), c.Response().StatusCode(), duration)
		}

		// 응답 로깅
		if config.EnableLogging {
			pm.logger.Info("Plugin request completed",
				logging.F("package_type", packageType),
				logging.F("mode", string(mode)),
				logging.F("status_code", c.Response().StatusCode()),
				logging.F("duration", time.Since(start)),
				logging.F("response_size", len(c.Response().Body())))
		}

		return err
	}
}

// HealthCheckHandler 플러그인 헬스체크 핸들러
func (pm *PluginMiddleware) HealthCheckHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(c.Context(), 10*time.Second)
		defer cancel()

		results := pm.manager.HealthCheck(ctx)

		healthStatus := make(map[string]interface{})
		allHealthy := true

		for packageType, err := range results {
			if err != nil {
				healthStatus[packageType] = map[string]interface{}{
					"status": "unhealthy",
					"error":  err.Error(),
				}
				allHealthy = false
			} else {
				healthStatus[packageType] = map[string]interface{}{
					"status": "healthy",
				}
			}
		}

		response := map[string]interface{}{
			"status":    "healthy",
			"handlers":  healthStatus,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}

		if !allHealthy {
			response["status"] = "degraded"
			return c.Status(fiber.StatusServiceUnavailable).JSON(response)
		}

		return c.JSON(response)
	}
}

// StatisticsHandler 플러그인 통계 핸들러
func (pm *PluginMiddleware) StatisticsHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		stats := pm.manager.GetStatistics()

		response := map[string]interface{}{
			"statistics": stats,
			"timestamp":  time.Now().UTC().Format(time.RFC3339),
		}

		return c.JSON(response)
	}
}

// ListHandlersHandler 핸들러 목록 조회 핸들러
func (pm *PluginMiddleware) ListHandlersHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		handlers := pm.manager.ListHandlers()

		handlerDetails := make(map[string]interface{})
		for _, packageType := range handlers {
			if metadata, exists := pm.manager.GetPluginMetadata(packageType); exists {
				handlerDetails[packageType] = map[string]interface{}{
					"name":        metadata.Name,
					"version":     metadata.Version,
					"description": metadata.Description,
					"author":      metadata.Author,
					"tags":        metadata.Tags,
				}

				// 핸들러 지원 모드 확인
				if handler, exists := pm.manager.GetHandler(packageType); exists {
					supportedModes := []string{}
					if handler.SupportsMode(ProxyMode) {
						supportedModes = append(supportedModes, string(ProxyMode))
					}
					if handler.SupportsMode(MirrorMode) {
						supportedModes = append(supportedModes, string(MirrorMode))
					}
					if details, ok := handlerDetails[packageType].(map[string]interface{}); ok {
						details["supported_modes"] = supportedModes
					}
				}
			}
		}

		response := map[string]interface{}{
			"handlers":  handlerDetails,
			"count":     len(handlers),
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}

		return c.JSON(response)
	}
}

// parseRequest 요청 파싱하여 패키지 타입과 모드 추출
func (pm *PluginMiddleware) parseRequest(c *fiber.Ctx, config PluginMiddlewareConfig) (string, OperationMode, error) {
	path := c.Path()

	// 기본 경로 제거
	relativePath := strings.TrimPrefix(path, config.BasePath)
	relativePath = strings.TrimPrefix(relativePath, "/")

	// 경로 구조: /proxy/{mode}/{type}/{path...} 또는 /proxy/{type}/{path...}
	parts := strings.Split(relativePath, "/")
	if len(parts) == 0 {
		return "", "", fmt.Errorf("invalid request path")
	}

	var packageType string
	var mode OperationMode

	// 모드가 명시적으로 지정된 경우
	if len(parts) >= 2 && (parts[0] == string(ProxyMode) || parts[0] == string(MirrorMode)) {
		mode = OperationMode(parts[0])
		packageType = parts[1]
	} else {
		// 모드가 명시되지 않은 경우 기본 모드 사용
		mode = config.DefaultMode
		packageType = parts[0]
	}

	// 패키지 타입 검증
	if packageType == "" {
		return "", "", fmt.Errorf("package type not specified")
	}

	// 쿼리 파라미터에서 모드 오버라이드 확인
	if queryMode := c.Query("mode"); queryMode != "" {
		if queryMode == string(ProxyMode) || queryMode == string(MirrorMode) {
			mode = OperationMode(queryMode)
		}
	}

	return packageType, mode, nil
}

// recordMetrics 메트릭 기록
func (pm *PluginMiddleware) recordMetrics(packageType, mode string, statusCode int, duration time.Duration) {
	pm.logger.Debug("Recording plugin metrics",
		logging.F("package_type", packageType),
		logging.F("mode", mode),
		logging.F("status_code", statusCode),
		logging.F("duration_ms", duration.Milliseconds()))

	// TODO: 실제 메트릭 시스템과 연동 (Prometheus 등)
}

// CacheMiddleware 캐시 미들웨어
func (pm *PluginMiddleware) CacheMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// 캐시 키 생성
		cacheKey := pm.buildCacheKey(c)

		pm.logger.Debug("Cache middleware",
			logging.F("cache_key", cacheKey),
			logging.F("method", c.Method()))

		// GET 요청에 대해서만 캐시 적용
		if c.Method() != fiber.MethodGet {
			return c.Next()
		}

		// TODO: 캐시에서 조회
		// 캐시 히트시 응답 반환
		// 캐시 미스시 다음 핸들러 호출 후 결과 캐시

		// 현재는 단순히 다음 핸들러 호출
		return c.Next()
	}
}

// buildCacheKey 캐시 키 생성
func (pm *PluginMiddleware) buildCacheKey(c *fiber.Ctx) string {
	return fmt.Sprintf("plugin:%s:%s", c.Method(), c.Path())
}

// ErrorHandler 플러그인 에러 핸들러
func (pm *PluginMiddleware) ErrorHandler() fiber.ErrorHandler {
	return func(c *fiber.Ctx, err error) error {
		code := fiber.StatusInternalServerError
		message := "Internal Server Error"

		// Fiber 에러인 경우
		if e, ok := err.(*fiber.Error); ok {
			code = e.Code
			message = e.Message
		}

		pm.logger.Error("Plugin request error",
			logging.F("path", c.Path()),
			logging.F("method", c.Method()),
			logging.F("status_code", code),
			logging.F("error", err))

		// 에러 응답
		return c.Status(code).JSON(map[string]interface{}{
			"error": map[string]interface{}{
				"code":      code,
				"message":   message,
				"path":      c.Path(),
				"method":    c.Method(),
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			},
		})
	}
}

// ValidationMiddleware 요청 검증 미들웨어
func (pm *PluginMiddleware) ValidationMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// 기본 검증
		if c.Path() == "" {
			return fiber.NewError(fiber.StatusBadRequest, "Empty path not allowed")
		}

		// 패키지 타입별 추가 검증은 각 핸들러에서 수행
		return c.Next()
	}
}

// RateLimitMiddleware 레이트 리미팅 미들웨어 (플러그인별)
func (pm *PluginMiddleware) RateLimitMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// TODO: 플러그인별 레이트 리미팅 구현
		// 패키지 타입과 클라이언트 IP 기반으로 제한

		return c.Next()
	}
}
