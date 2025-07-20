// Package proxy provides unified proxy handling functionality
package proxy

import (
	"sync"

	"github.com/gofiber/fiber/v2"

	"proxynd/internal/errors"
	"proxynd/internal/handlers"
	"proxynd/logging"
)

// UnifiedProxyHandler Template Method 패턴을 사용하는 통합 프록시 핸들러
type UnifiedProxyHandler struct {
	factory *handlers.ProxyHandlerFactory
	logger  logging.Logger
}

var (
	unifiedHandlerInstance *UnifiedProxyHandler
	unifiedHandlerOnce     sync.Once
)

// NewUnifiedProxyHandler 새로운 통합 프록시 핸들러 생성 (싱글톤)
func NewUnifiedProxyHandler(factory *handlers.ProxyHandlerFactory) *UnifiedProxyHandler {
	unifiedHandlerOnce.Do(func() {
		unifiedHandlerInstance = &UnifiedProxyHandler{
			factory: factory,
			logger:  logging.GetLogger(),
		}
	})
	return unifiedHandlerInstance
}

// Handle 통합 프록시 요청 처리
func (h *UnifiedProxyHandler) Handle(c *fiber.Ctx) error {
	proxyType := c.Params("type")
	path := c.Params("*")

	h.logger.Info("Unified proxy request received",
		logging.F("proxy_type", proxyType),
		logging.F("path", path),
		logging.F("method", c.Method()),
		logging.F("remote_ip", c.IP()),
	)

	// 프록시 타입 지원 여부 확인
	if !h.factory.IsSupported(proxyType) {
		return errors.NewError("PROXY006", "지원하지 않는 프록시 타입입니다").
			WithDomain("unified").
			WithDetails(map[string]interface{}{
				"proxy_type":      proxyType,
				"supported_types": h.factory.GetSupportedTypes(),
			}).
			Build()
	}

	// 프록시 핸들러 생성 (싱글톤으로 관리)
	handler, err := h.factory.CreateSingleton(proxyType)
	if err != nil {
		return errors.NewError("PROXY007", "프록시 핸들러 생성 실패").
			WithDomain("unified").
			WithCause(err).
			WithDetails(map[string]interface{}{
				"proxy_type": proxyType,
			}).
			Build()
	}

	// 핸들러로 요청 위임
	return handler.Handle(c)
}

// GetSupportedTypes 지원하는 프록시 타입 목록 반환
func (h *UnifiedProxyHandler) GetSupportedTypes() []string {
	return h.factory.GetSupportedTypes()
}

// IsSupported 프록시 타입 지원 여부 확인
func (h *UnifiedProxyHandler) IsSupported(proxyType string) bool {
	return h.factory.IsSupported(proxyType)
}

// GetHandlerInfo 핸들러 정보 반환
func (h *UnifiedProxyHandler) GetHandlerInfo(proxyType string) (map[string]interface{}, error) {
	return h.factory.GetHandlerInfo(proxyType)
}

// HealthCheck 모든 활성화된 핸들러의 헬스체크 수행
func (h *UnifiedProxyHandler) HealthCheck() map[string]error {
	return h.factory.HealthCheck()
}

// GetStatistics 통합 핸들러 통계 정보 반환
func (h *UnifiedProxyHandler) GetStatistics() map[string]interface{} {
	stats := h.factory.GetStatistics()

	// 추가 통계 정보
	stats["handler_version"] = "v1"
	stats["pattern_type"] = "template_method"

	return stats
}

// RegisterProxyType 새로운 프록시 타입 등록
func (h *UnifiedProxyHandler) RegisterProxyType(proxyType string, creator func() handlers.BaseProxyHandler) {
	h.factory.Register(proxyType, creator)

	h.logger.Info("New proxy type registered",
		logging.F("proxy_type", proxyType),
	)
}

// UnregisterProxyType 프록시 타입 등록 해제
func (h *UnifiedProxyHandler) UnregisterProxyType(proxyType string) {
	h.factory.Unregister(proxyType)

	h.logger.Info("Proxy type unregistered",
		logging.F("proxy_type", proxyType),
	)
}

// Shutdown 통합 핸들러 종료
func (h *UnifiedProxyHandler) Shutdown() {
	h.factory.Shutdown()
	h.logger.Info("Unified proxy handler shutdown completed")
}

// Name 핸들러 이름 반환
func (h *UnifiedProxyHandler) Name() string {
	return "unified-proxy"
}

// Type 핸들러 타입 반환
func (h *UnifiedProxyHandler) Type() string {
	return "unified"
}

// 편의 함수들

// GetUnifiedHandlerInstance 전역 통합 핸들러 인스턴스 반환
func GetUnifiedHandlerInstance() *UnifiedProxyHandler {
	return unifiedHandlerInstance
}
