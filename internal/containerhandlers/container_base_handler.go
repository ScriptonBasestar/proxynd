package containerhandlers

import (
	"time"

	"github.com/gofiber/fiber/v2"

	"proxynd/internal/container"
	"proxynd/metrics"
)

// BaseContainerHandler Container 기반 핸들러의 공통 기능을 제공하는 베이스 구조체
type BaseContainerHandler struct {
	containerProvider container.ContainerProvider
	containerMetrics  *metrics.ContainerMetrics
	handlerType       string
	handlerName       string
}

// NewBaseContainerHandler 새로운 베이스 Container 핸들러 생성
func NewBaseContainerHandler(
	provider container.ContainerProvider,
	handlerType string,
	handlerName string,
) *BaseContainerHandler {
	base := &BaseContainerHandler{
		containerProvider: provider,
		containerMetrics:  metrics.GetContainerMetrics(),
		handlerType:       handlerType,
		handlerName:       handlerName,
	}

	// 핸들러 초기화 메트릭 기록
	base.containerMetrics.RecordHandlerInitialization(handlerType, true)
	base.containerMetrics.SetActiveHandlerInstances(handlerType, 1)

	return base
}

// Name 핸들러 이름 반환
func (h *BaseContainerHandler) Name() string {
	return h.handlerName
}

// Type 핸들러 타입 반환
func (h *BaseContainerHandler) Type() string {
	return h.handlerType
}

// GetContainer Container Provider 반환
func (h *BaseContainerHandler) GetContainer() container.ContainerProvider {
	return h.containerProvider
}

// SetContainer Container Provider 설정
func (h *BaseContainerHandler) SetContainer(provider container.ContainerProvider) {
	h.containerProvider = provider
}

// RecordConfigLoad 설정 로딩 메트릭 기록
func (h *BaseContainerHandler) RecordConfigLoad(operation string, success bool, duration time.Duration) {
	h.containerMetrics.RecordConfigLoad(h.handlerType, operation, success, duration.Seconds())
}

// RecordHealthCheck 헬스체크 메트릭 기록
func (h *BaseContainerHandler) RecordHealthCheck(success bool) {
	h.containerMetrics.RecordHealthCheck(h.handlerType, success)
}

// RecordConfigReload 설정 리로드 메트릭 기록
func (h *BaseContainerHandler) RecordConfigReload(success bool) {
	h.containerMetrics.RecordConfigReload(h.handlerType, success)
}

// RecordUpstreamBuild 업스트림 URL 빌드 메트릭 기록
func (h *BaseContainerHandler) RecordUpstreamBuild(success bool) {
	h.containerMetrics.RecordUpstreamBuild(h.handlerType, success)
}

// RecordCacheKeyGeneration 캐시 키 생성 메트릭 기록
func (h *BaseContainerHandler) RecordCacheKeyGeneration(duration time.Duration) {
	h.containerMetrics.RecordCacheKeyGeneration(h.handlerType, duration.Seconds())
}

// RecordHandlerRequest 핸들러 요청 메트릭 기록
func (h *BaseContainerHandler) RecordHandlerRequest(method string, statusCode int) {
	h.containerMetrics.RecordHandlerRequest(h.handlerType, method, statusCode)
}

// RecordHandlerDuration 핸들러 처리 시간 메트릭 기록
func (h *BaseContainerHandler) RecordHandlerDuration(method string, duration time.Duration) {
	h.containerMetrics.RecordHandlerDuration(h.handlerType, method, duration.Seconds())
}

// RecordHandlerError 핸들러 오류 메트릭 기록
func (h *BaseContainerHandler) RecordHandlerError(errorType string) {
	h.containerMetrics.RecordHandlerError(h.handlerType, errorType)
}

// RecordContainerProviderCall Container Provider 호출 메트릭 기록
func (h *BaseContainerHandler) RecordContainerProviderCall(method string, success bool) {
	h.containerMetrics.RecordContainerProviderCall(method, h.handlerType, success)
}

// RecordConfigCacheHit 설정 캐시 히트 메트릭 기록
func (h *BaseContainerHandler) RecordConfigCacheHit(configType string) {
	h.containerMetrics.RecordConfigCacheHit(h.handlerType, configType)
}

// RecordConfigCacheMiss 설정 캐시 미스 메트릭 기록
func (h *BaseContainerHandler) RecordConfigCacheMiss(configType string) {
	h.containerMetrics.RecordConfigCacheMiss(h.handlerType, configType)
}

// WithMetrics 요청 처리를 메트릭과 함께 실행하는 헬퍼 메서드
func (h *BaseContainerHandler) WithMetrics(c *fiber.Ctx, handler func() error) error {
	start := time.Now()
	method := c.Method()

	err := handler()

	duration := time.Since(start)
	statusCode := c.Response().StatusCode()

	// 메트릭 기록
	h.RecordHandlerRequest(method, statusCode)
	h.RecordHandlerDuration(method, duration)

	if statusCode >= 400 {
		errorType := getErrorTypeFromStatusCode(statusCode)
		h.RecordHandlerError(errorType)
	}

	return err
}

// getErrorTypeFromStatusCode 상태 코드에서 오류 타입 추출
func getErrorTypeFromStatusCode(statusCode int) string {
	switch {
	case statusCode >= 400 && statusCode < 500:
		return "client_error"
	case statusCode >= 500:
		return "server_error"
	default:
		return "unknown_error"
	}
}

// ContainerHandlerMetricsCollector Container 핸들러 메트릭 수집 인터페이스
type ContainerHandlerMetricsCollector interface {
	// CollectMetrics 핸들러별 메트릭 수집
	CollectMetrics() map[string]interface{}
}

// CollectMetrics 현재 핸들러의 메트릭 정보 수집
func (h *BaseContainerHandler) CollectMetrics() map[string]interface{} {
	metrics := make(map[string]interface{})

	metrics["handler_name"] = h.Name()
	metrics["handler_type"] = h.Type()
	metrics["active_instances"] = 1

	// Container Provider 상태
	if h.containerProvider != nil {
		metrics["container_provider_available"] = true
		metrics["storage_dir"] = h.containerProvider.GetStorageDir()
		metrics["config_dir"] = h.containerProvider.GetConfigDir()
	} else {
		metrics["container_provider_available"] = false
	}

	return metrics
}
