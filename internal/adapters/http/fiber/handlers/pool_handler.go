package handlers

import (
	"github.com/gofiber/fiber/v2"

	"proxynd/configs"
	"proxynd/internal/pool"
	"proxynd/logging"
)

// PoolHandler Connection Pool 관리 핸들러
type PoolHandler struct {
	logger        logging.Logger
	poolConfig    *configs.ConnectionPoolConfig
	clientFactory *pool.ProxyClientFactory
}

// NewPoolHandler 새로운 Pool 핸들러 생성
func NewPoolHandler() *PoolHandler {
	return &PoolHandler{
		logger:        logging.GetLogger(),
		poolConfig:    configs.NewConnectionPoolConfig(),
		clientFactory: pool.GetGlobalClientFactory(),
	}
}

// GetPoolStatus Connection Pool 상태 조회
func (h *PoolHandler) GetPoolStatus(c *fiber.Ctx) error {
	statistics := h.clientFactory.GetPoolStatistics()
	configStats := h.poolConfig.GetSettings().GetStatistics()

	response := fiber.Map{
		"pool_statistics": statistics,
		"configuration":   configStats,
		"timestamp":       c.Context().Time().Format("2006-01-02T15:04:05Z07:00"),
	}

	return c.JSON(response)
}

// GetPoolConfiguration Connection Pool 설정 조회
func (h *PoolHandler) GetPoolConfiguration(c *fiber.Ctx) error {
	settings := h.poolConfig.GetSettings()

	response := fiber.Map{
		"configuration": settings,
		"config_exists": h.poolConfig.ConfigExists(),
		"statistics":    settings.GetStatistics(),
	}

	return c.JSON(response)
}

// UpdatePoolConfiguration Connection Pool 설정 업데이트
func (h *PoolHandler) UpdatePoolConfiguration(c *fiber.Ctx) error {
	var newSettings configs.ConnectionPoolSettings

	if err := c.BodyParser(&newSettings); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "잘못된 요청 데이터",
			"details": err.Error(),
		})
	}

	// 설정 검증
	if err := newSettings.Validate(); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "설정 검증 실패",
			"details": err.Error(),
		})
	}

	// 설정 업데이트
	if err := h.poolConfig.UpdateSettings(&newSettings); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "설정 업데이트 실패",
			"details": err.Error(),
		})
	}

	// 실제 Connection Pool에 설정 적용
	poolConfig := newSettings.ToPoolConfig()
	globalPool := pool.GetGlobalPool()
	if err := globalPool.UpdateConfig(poolConfig); err != nil {
		h.logger.Error("Connection Pool 설정 적용 실패",
			logging.F("error", err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Connection Pool 설정 적용 실패",
			"details": err.Error(),
		})
	}

	// 프록시별 타임아웃 업데이트
	for proxyType := range newSettings.ProxyTimeouts {
		h.clientFactory.UpdateTimeout(proxyType, newSettings.GetProxyTimeout(proxyType))
	}

	h.logger.Info("Connection Pool 설정이 업데이트되었습니다",
		logging.F("max_total_connections", newSettings.MaxTotalConnections),
		logging.F("max_connections_per_host", newSettings.MaxConnectionsPerHost),
	)

	return c.JSON(fiber.Map{
		"message":       "Connection Pool 설정이 업데이트되었습니다",
		"configuration": newSettings,
	})
}

// UpdateProxyTimeout 특정 프록시 타입의 타임아웃 업데이트
func (h *PoolHandler) UpdateProxyTimeout(c *fiber.Ctx) error {
	proxyType := c.Params("type")
	if proxyType == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "프록시 타입이 필요합니다",
		})
	}

	var request struct {
		TimeoutSeconds int `json:"timeout_seconds"`
	}

	if err := c.BodyParser(&request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "잘못된 요청 데이터",
			"details": err.Error(),
		})
	}

	// 설정 업데이트
	settings := h.poolConfig.GetSettings()
	if err := settings.UpdateProxyTimeout(proxyType, request.TimeoutSeconds); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "타임아웃 업데이트 실패",
			"details": err.Error(),
		})
	}

	// 클라이언트 팩토리에 적용
	h.clientFactory.UpdateTimeout(proxyType, settings.GetProxyTimeout(proxyType))

	h.logger.Info("프록시 타임아웃이 업데이트되었습니다",
		logging.F("proxy_type", proxyType),
		logging.F("timeout_seconds", request.TimeoutSeconds),
	)

	return c.JSON(fiber.Map{
		"message":         "프록시 타임아웃이 업데이트되었습니다",
		"proxy_type":      proxyType,
		"timeout_seconds": request.TimeoutSeconds,
	})
}

// GetProxyTimeouts 모든 프록시 타입의 타임아웃 조회
func (h *PoolHandler) GetProxyTimeouts(c *fiber.Ctx) error {
	settings := h.poolConfig.GetSettings()
	supportedTypes := h.clientFactory.GetSupportedProxyTypes()

	timeouts := make(map[string]interface{})
	for _, proxyType := range supportedTypes {
		timeouts[proxyType] = map[string]interface{}{
			"timeout_seconds": settings.GetProxyTimeout(proxyType).Seconds(),
			"configured":      settings.ProxyTimeouts[proxyType] != 0,
		}
	}

	return c.JSON(fiber.Map{
		"proxy_timeouts":  timeouts,
		"supported_types": supportedTypes,
		"total_types":     len(supportedTypes),
	})
}

// ResetPoolConfiguration Connection Pool 설정을 기본값으로 리셋
func (h *PoolHandler) ResetPoolConfiguration(c *fiber.Ctx) error {
	defaultSettings := configs.DefaultConnectionPoolSettings()

	// 설정 업데이트
	if err := h.poolConfig.UpdateSettings(defaultSettings); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "기본 설정 복원 실패",
			"details": err.Error(),
		})
	}

	// Connection Pool에 기본 설정 적용
	poolConfig := defaultSettings.ToPoolConfig()
	globalPool := pool.GetGlobalPool()
	if err := globalPool.UpdateConfig(poolConfig); err != nil {
		h.logger.Error("Connection Pool 기본 설정 적용 실패",
			logging.F("error", err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Connection Pool 기본 설정 적용 실패",
			"details": err.Error(),
		})
	}

	h.logger.Info("Connection Pool 설정이 기본값으로 리셋되었습니다")

	return c.JSON(fiber.Map{
		"message":       "Connection Pool 설정이 기본값으로 리셋되었습니다",
		"configuration": defaultSettings,
	})
}

// GetPoolHealth Connection Pool 헬스체크
func (h *PoolHandler) GetPoolHealth(c *fiber.Ctx) error {
	statistics := h.clientFactory.GetPoolStatistics()
	settings := h.poolConfig.GetSettings()

	// 헬스 상태 판단
	healthy := true
	issues := make([]string, 0)

	// 활성 연결 수 체크
	if statistics.ActiveConnections > int64(float64(settings.MaxTotalConnections)*0.9) {
		healthy = false
		issues = append(issues, "활성 연결 수가 제한에 근접함")
	}

	// 요청 통계 체크
	if statistics.TotalRequests == 0 {
		issues = append(issues, "아직 요청이 없음")
	}

	status := fiber.StatusOK
	if !healthy {
		status = fiber.StatusServiceUnavailable
	}

	response := fiber.Map{
		"healthy":    healthy,
		"status":     map[bool]string{true: "healthy", false: "unhealthy"}[healthy],
		"statistics": statistics,
		"configuration_limits": map[string]interface{}{
			"max_total_connections":    settings.MaxTotalConnections,
			"max_connections_per_host": settings.MaxConnectionsPerHost,
		},
		"issues":    issues,
		"timestamp": c.Context().Time().Format("2006-01-02T15:04:05Z07:00"),
	}

	return c.Status(status).JSON(response)
}

// SavePoolConfiguration Connection Pool 설정을 파일로 저장
func (h *PoolHandler) SavePoolConfiguration(c *fiber.Ctx) error {
	if err := h.poolConfig.WriteConfig(); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "설정 파일 저장 실패",
			"details": err.Error(),
		})
	}

	h.logger.Info("Connection Pool 설정이 파일로 저장되었습니다")

	return c.JSON(fiber.Map{
		"message":     "Connection Pool 설정이 저장되었습니다",
		"config_path": "connection-pool.yaml",
	})
}
