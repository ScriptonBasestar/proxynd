package proxy

import (
	"github.com/gofiber/fiber/v2"

	"proxynd/logging"
)

// UnifiedProxyHandlerV3 V3 팩토리를 사용한 새로운 통합 프록시 핸들러
func UnifiedProxyHandlerV3(c *fiber.Ctx) error {
	proxyType := c.Params("type")
	path := c.Params("*")

	logger := logging.GetLogger()
	logger.Debug("통합 프록시 요청 (V3)",
		logging.F("proxy_type", proxyType),
		logging.F("path", path),
		logging.F("method", c.Method()),
		logging.F("ip", c.IP()),
	)

	// V3 팩토리를 사용하여 요청 처리
	return HandleProxyRequest(c)
}

// GetProxyStatus 모든 프록시의 상태 정보 반환 (API 엔드포인트용)
func GetProxyStatus(c *fiber.Ctx) error {
	factory := GetGlobalFactoryV3()

	statistics := factory.GetStatistics()
	healthStatus := factory.HealthCheck()

	// 각 핸들러의 상세 정보 수집
	handlerDetails := make(map[string]interface{})
	for _, proxyType := range factory.GetSupportedTypes() {
		if info, err := factory.GetHandlerInfo(proxyType); err == nil {
			handlerDetails[proxyType] = info
		}
	}

	response := fiber.Map{
		"factory_info":    statistics,
		"health_status":   healthStatus,
		"handler_details": handlerDetails,
		"timestamp": fiber.Map{
			"unix": c.Context().Time().Unix(),
			"iso":  c.Context().Time().Format("2006-01-02T15:04:05Z07:00"),
		},
	}

	return c.JSON(response)
}

// ValidateProxyType 프록시 타입 유효성 검사 (미들웨어용)
func ValidateProxyType(c *fiber.Ctx) error {
	proxyType := c.Params("type")

	factory := GetGlobalFactoryV3()
	if !factory.IsSupported(proxyType) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":           "지원하지 않는 프록시 타입",
			"type":            proxyType,
			"supported_types": factory.GetSupportedTypes(),
		})
	}

	return c.Next()
}

// GetProxyTypes 지원하는 프록시 타입 목록 반환 (API 엔드포인트용)
func GetProxyTypes(c *fiber.Ctx) error {
	factory := GetGlobalFactoryV3()

	supportedTypes := factory.GetSupportedTypes()
	enabledTypes := factory.GetEnabledTypes()

	response := fiber.Map{
		"supported_types": supportedTypes,
		"enabled_types":   enabledTypes,
		"total_supported": len(supportedTypes),
		"total_enabled":   len(enabledTypes),
	}

	return c.JSON(response)
}
