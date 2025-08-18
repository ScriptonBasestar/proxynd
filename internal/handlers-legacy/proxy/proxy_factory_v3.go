package proxy

import (
	"fmt"
	"sync"

	"github.com/gofiber/fiber/v2"

	"proxynd/internal/logging"
)

// ProxyHandlerV3Interface V3 프록시 핸들러 공통 인터페이스
type ProxyHandlerV3Interface interface {
	// 기본 정보
	Type() string
	Handle(c *fiber.Ctx) error

	// 활성화 상태 확인
	IsEnabled() bool
}

// ProxyFactoryV3 간소화된 V3 프록시 팩토리
type ProxyFactoryV3 struct {
	handlers map[string]ProxyHandlerV3Interface
	mutex    sync.RWMutex
	logger   logging.Logger
}

// NewProxyFactoryV3 새로운 V3 프록시 팩토리 생성
func NewProxyFactoryV3() *ProxyFactoryV3 {
	factory := &ProxyFactoryV3{
		handlers: make(map[string]ProxyHandlerV3Interface),
		logger:   logging.GetLogger(),
	}

	// 기본 V3 핸들러들 등록
	factory.registerDefaultHandlers()

	return factory
}

// registerDefaultHandlers 기본 V3 핸들러들 등록
func (f *ProxyFactoryV3) registerDefaultHandlers() {
	f.logger.Info("ProxyFactoryV3 - 기본 핸들러들 등록 시작")

	// 모든 V3 핸들러 등록
	handlers := map[string]ProxyHandlerV3Interface{
		"apt":    NewAPTHandlerV3(),
		"maven":  NewMavenHandlerV3(),
		"npm":    NewNPMHandlerV3(),
		"docker": NewDockerHandlerV3(),
		"pip":    NewPipHandlerV3(),
		"yum":    NewYumHandlerV3(),
		"apk":    NewApkHandlerV3(),
	}

	f.mutex.Lock()
	defer f.mutex.Unlock()

	for proxyType, handler := range handlers {
		f.handlers[proxyType] = handler
		f.logger.Debug("프록시 핸들러 등록됨",
			logging.F("proxy_type", proxyType),
			logging.F("handler_type", fmt.Sprintf("%T", handler)),
		)
	}

	f.logger.Info("ProxyFactoryV3 - 기본 핸들러 등록 완료",
		logging.F("total_handlers", len(f.handlers)),
	)
}

// GetHandler 프록시 타입에 해당하는 핸들러 반환
func (f *ProxyFactoryV3) GetHandler(proxyType string) (ProxyHandlerV3Interface, error) {
	f.mutex.RLock()
	defer f.mutex.RUnlock()

	handler, exists := f.handlers[proxyType]
	if !exists {
		return nil, fmt.Errorf("지원하지 않는 프록시 타입: %s", proxyType)
	}

	return handler, nil
}

// IsSupported 프록시 타입 지원 여부 확인
func (f *ProxyFactoryV3) IsSupported(proxyType string) bool {
	f.mutex.RLock()
	defer f.mutex.RUnlock()

	_, exists := f.handlers[proxyType]
	return exists
}

// GetSupportedTypes 지원하는 모든 프록시 타입 반환
func (f *ProxyFactoryV3) GetSupportedTypes() []string {
	f.mutex.RLock()
	defer f.mutex.RUnlock()

	types := make([]string, 0, len(f.handlers))
	for proxyType := range f.handlers {
		types = append(types, proxyType)
	}

	return types
}

// GetEnabledTypes 활성화된 프록시 타입들만 반환
func (f *ProxyFactoryV3) GetEnabledTypes() []string {
	f.mutex.RLock()
	defer f.mutex.RUnlock()

	var enabledTypes []string
	for proxyType, handler := range f.handlers {
		if handler.IsEnabled() {
			enabledTypes = append(enabledTypes, proxyType)
		}
	}

	return enabledTypes
}

// HealthCheck 모든 활성화된 핸들러의 상태 확인
func (f *ProxyFactoryV3) HealthCheck() map[string]bool {
	f.mutex.RLock()
	defer f.mutex.RUnlock()

	healthStatus := make(map[string]bool)
	for proxyType, handler := range f.handlers {
		healthStatus[proxyType] = handler.IsEnabled()
	}

	return healthStatus
}

// GetStatistics 팩토리 통계 정보 반환
func (f *ProxyFactoryV3) GetStatistics() map[string]interface{} {
	f.mutex.RLock()
	defer f.mutex.RUnlock()

	enabledCount := 0
	for _, handler := range f.handlers {
		if handler.IsEnabled() {
			enabledCount++
		}
	}

	return map[string]interface{}{
		"total_handlers":   len(f.handlers),
		"enabled_handlers": enabledCount,
		"supported_types":  f.GetSupportedTypes(),
		"enabled_types":    f.GetEnabledTypes(),
	}
}

// 전역 팩토리 인스턴스 (싱글톤)
var (
	globalFactoryV3     *ProxyFactoryV3
	globalFactoryV3Once sync.Once
)

// GetGlobalFactoryV3 전역 V3 팩토리 인스턴스 반환
func GetGlobalFactoryV3() *ProxyFactoryV3 {
	globalFactoryV3Once.Do(func() {
		globalFactoryV3 = NewProxyFactoryV3()
	})
	return globalFactoryV3
}

// HandleProxyRequest V3 팩토리를 사용한 통합 프록시 요청 처리
func HandleProxyRequest(c *fiber.Ctx) error {
	proxyType := c.Params("type")

	factory := GetGlobalFactoryV3()

	// 핸들러 가져오기
	handler, err := factory.GetHandler(proxyType)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":           "지원하지 않는 프록시 타입",
			"type":            proxyType,
			"supported_types": factory.GetSupportedTypes(),
		})
	}

	// 핸들러 활성화 상태 확인
	if !handler.IsEnabled() {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": fmt.Sprintf("%s 프록시가 비활성화되어 있습니다", proxyType),
			"type":  proxyType,
		})
	}

	// 로깅
	logging.GetLogger().Debug("프록시 요청 처리 시작",
		logging.F("proxy_type", proxyType),
		logging.F("path", c.Path()),
		logging.F("method", c.Method()),
		logging.F("ip", c.IP()),
	)

	// 핸들러로 요청 위임
	return handler.Handle(c)
}

// GetHandlerInfo 특정 핸들러의 상세 정보 반환 (디버깅/모니터링용)
func (f *ProxyFactoryV3) GetHandlerInfo(proxyType string) (map[string]interface{}, error) {
	f.mutex.RLock()
	defer f.mutex.RUnlock()

	handler, exists := f.handlers[proxyType]
	if !exists {
		return nil, fmt.Errorf("지원하지 않는 프록시 타입: %s", proxyType)
	}

	info := map[string]interface{}{
		"type":         handler.Type(),
		"enabled":      handler.IsEnabled(),
		"handler_impl": fmt.Sprintf("%T", handler),
	}

	return info, nil
}
