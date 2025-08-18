package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"

	"proxynd/internal/container"
	"proxynd/internal/handlers"
	"proxynd/internal/metrics"
)

// ContainerMetricsMiddleware Container 기반 핸들러용 메트릭 미들웨어
func ContainerMetricsMiddleware() fiber.Handler {
	containerMetrics := metrics.GetContainerMetrics()

	return func(c *fiber.Ctx) error {
		start := time.Now()

		// 요청 처리
		err := c.Next()

		// 메트릭 수집
		duration := time.Since(start).Seconds()
		method := c.Method()
		statusCode := c.Response().StatusCode()
		path := c.Path()

		// Container 핸들러 타입 추출
		handlerType := extractHandlerTypeFromPath(path)

		if handlerType != "" {
			// Container 핸들러 요청 메트릭 기록
			containerMetrics.RecordHandlerRequest(handlerType, method, statusCode)
			containerMetrics.RecordHandlerDuration(handlerType, method, duration)

			// 오류 상태 코드인 경우 오류 메트릭 기록
			if statusCode >= 400 {
				errorType := getErrorType(statusCode)
				containerMetrics.RecordHandlerError(handlerType, errorType)
			}
		}

		return err
	}
}

// extractHandlerTypeFromPath 경로에서 핸들러 타입 추출
func extractHandlerTypeFromPath(path string) string {
	// Container 프록시 경로 패턴: /api/v1/proxy/:type/* 또는 /proxy/:type/*
	if len(path) < 8 {
		return ""
	}

	// 새로운 API 패턴 (/api/v1/proxy/:type/*)
	if len(path) > 16 && path[:16] == "/api/v1/proxy/" {
		pathParts := extractPathSegments(path[16:])
		if len(pathParts) > 0 {
			return pathParts[0]
		}
	}

	// 레거시 API 패턴 (/proxy/:type/*)
	if len(path) > 7 && path[:7] == "/proxy/" {
		pathParts := extractPathSegments(path[7:])
		if len(pathParts) > 0 {
			return pathParts[0]
		}
	}

	return ""
}

// extractPathSegments 경로를 세그먼트로 분할
func extractPathSegments(path string) []string {
	if path == "" {
		return nil
	}

	segments := make([]string, 0, 3)
	current := ""

	for i, r := range path {
		if r == '/' {
			if current != "" {
				segments = append(segments, current)
				current = ""
			}
		} else {
			current += string(r)
		}

		// 첫 번째 세그먼트만 필요하므로 조기 종료
		if len(segments) > 0 && i > 0 && r == '/' {
			break
		}
	}

	// 마지막 세그먼트 추가
	if current != "" {
		segments = append(segments, current)
	}

	return segments
}

// getErrorType 상태 코드에서 오류 타입 추출
func getErrorType(statusCode int) string {
	switch {
	case statusCode >= 400 && statusCode < 500:
		return "client_error"
	case statusCode >= 500:
		return "server_error"
	default:
		return "unknown_error"
	}
}

// ContainerHandlerMetricsWrapper Container 핸들러에 메트릭 기능을 추가하는 래퍼
type ContainerHandlerMetricsWrapper struct {
	handler          handlers.ContainerProxyHandler
	containerMetrics *metrics.ContainerMetrics
	handlerType      string
}

// NewContainerHandlerMetricsWrapper 새로운 메트릭 래퍼 생성
func NewContainerHandlerMetricsWrapper(handler handlers.ContainerProxyHandler) *ContainerHandlerMetricsWrapper {
	return &ContainerHandlerMetricsWrapper{
		handler:          handler,
		containerMetrics: metrics.GetContainerMetrics(),
		handlerType:      handler.Type(),
	}
}

// Handler 인터페이스 구현

// Name 핸들러 이름 반환
func (w *ContainerHandlerMetricsWrapper) Name() string {
	return w.handler.Name()
}

// Type 핸들러 타입 반환
func (w *ContainerHandlerMetricsWrapper) Type() string {
	return w.handler.Type()
}

// IsEnabled 활성화 상태 반환
func (w *ContainerHandlerMetricsWrapper) IsEnabled() bool {
	return w.handler.IsEnabled()
}

// HealthCheck 헬스체크 수행 및 메트릭 기록 (BaseProxyHandler 인터페이스)
func (w *ContainerHandlerMetricsWrapper) HealthCheck() error {
	// ContainerProxyHandler는 BaseProxyHandler를 임베드하므로 해당 메서드가 있어야 함
	if healthChecker, ok := w.handler.(interface{ HealthCheck() error }); ok {
		err := healthChecker.HealthCheck()
		w.containerMetrics.RecordHealthCheck(w.handlerType, err == nil)
		return err
	}
	// 헬스체크 메서드가 없는 경우 성공으로 간주
	w.containerMetrics.RecordHealthCheck(w.handlerType, true)
	return nil
}

// LoadConfig 설정 로드 및 메트릭 기록
func (w *ContainerHandlerMetricsWrapper) LoadConfig() error {
	start := time.Now()
	err := w.handler.LoadConfig()
	duration := time.Since(start).Seconds()

	w.containerMetrics.RecordConfigLoad(w.handlerType, "load", err == nil, duration)
	return err
}

// ReloadConfig 설정 리로드 및 메트릭 기록
func (w *ContainerHandlerMetricsWrapper) ReloadConfig() error {
	start := time.Now()
	err := w.handler.ReloadConfig()
	duration := time.Since(start).Seconds()

	w.containerMetrics.RecordConfigLoad(w.handlerType, "reload", err == nil, duration)
	w.containerMetrics.RecordConfigReload(w.handlerType, err == nil)
	return err
}

// GenerateCacheKey 캐시 키 생성 및 메트릭 기록 (BaseProxyHandler 인터페이스)
func (w *ContainerHandlerMetricsWrapper) GenerateCacheKey(c *fiber.Ctx) string {
	start := time.Now()
	key := w.handler.GenerateCacheKey(c)
	duration := time.Since(start).Seconds()

	w.containerMetrics.RecordCacheKeyGeneration(w.handlerType, duration)
	return key
}

// IsCacheable 캐시 가능 여부 확인 - 기본 구현
func (w *ContainerHandlerMetricsWrapper) IsCacheable(c *fiber.Ctx) bool {
	// GET 요청만 캐시 가능하다고 가정
	return c.Method() == "GET"
}

// ShouldCache 캐시 저장 여부 확인
func (w *ContainerHandlerMetricsWrapper) ShouldCache(c *fiber.Ctx, statusCode int) bool {
	return w.handler.ShouldCache(c, statusCode)
}

// BuildUpstreamURL 업스트림 URL 빌드 및 메트릭 기록
func (w *ContainerHandlerMetricsWrapper) BuildUpstreamURL(c *fiber.Ctx) (string, error) {
	url, err := w.handler.BuildUpstreamURL(c)
	w.containerMetrics.RecordUpstreamBuild(w.handlerType, err == nil)
	return url, err
}

// Handle 요청 처리 및 메트릭 기록 (Handler 인터페이스)
func (w *ContainerHandlerMetricsWrapper) Handle(c *fiber.Ctx) error {
	start := time.Now()

	err := w.handler.Handle(c)

	duration := time.Since(start).Seconds()
	method := c.Method()
	statusCode := c.Response().StatusCode()

	// 메트릭 기록
	w.containerMetrics.RecordHandlerRequest(w.handlerType, method, statusCode)
	w.containerMetrics.RecordHandlerDuration(w.handlerType, method, duration)

	if statusCode >= 400 {
		errorType := getErrorType(statusCode)
		w.containerMetrics.RecordHandlerError(w.handlerType, errorType)
	}

	return err
}

// BaseProxyHandler 인터페이스 메서드들

// TransformRequest 요청 변환
func (w *ContainerHandlerMetricsWrapper) TransformRequest(c *fiber.Ctx, upstreamReq *fiber.Agent) error {
	return w.handler.TransformRequest(c, upstreamReq)
}

// TransformResponse 응답 변환
func (w *ContainerHandlerMetricsWrapper) TransformResponse(resp []byte, c *fiber.Ctx) ([]byte, error) {
	return w.handler.TransformResponse(resp, c)
}

// GetCacheTTL 캐시 TTL 반환
func (w *ContainerHandlerMetricsWrapper) GetCacheTTL(c *fiber.Ctx) time.Duration {
	return w.handler.GetCacheTTL(c)
}

// HandleError 에러 처리
func (w *ContainerHandlerMetricsWrapper) HandleError(err error, c *fiber.Ctx) error {
	return w.handler.HandleError(err, c)
}

// ContainerAwareHandler 인터페이스 메서드들

// GetContainer Container Provider 반환
func (w *ContainerHandlerMetricsWrapper) GetContainer() container.ContainerProvider {
	return w.handler.GetContainer()
}

// SetContainer Container Provider 설정
func (w *ContainerHandlerMetricsWrapper) SetContainer(provider container.ContainerProvider) {
	w.handler.SetContainer(provider)
}

// MetricsCollector 메트릭 수집 인터페이스
type MetricsCollector interface {
	// CollectHandlerMetrics 핸들러별 메트릭 수집
	CollectHandlerMetrics() map[string]interface{}
}

// CollectHandlerMetrics 현재 핸들러의 메트릭 정보 수집
func (w *ContainerHandlerMetricsWrapper) CollectHandlerMetrics() map[string]interface{} {
	metrics := make(map[string]interface{})

	metrics["handler_name"] = w.Name()
	metrics["handler_type"] = w.Type()
	metrics["is_enabled"] = w.IsEnabled()

	// 헬스체크 상태 - 래퍼를 통해 호출
	healthErr := w.HealthCheck()
	metrics["health_status"] = healthErr == nil
	if healthErr != nil {
		metrics["health_error"] = healthErr.Error()
	}

	return metrics
}

// WrapContainerHandler Container 핸들러를 메트릭 래퍼로 감쌈
func WrapContainerHandler(handler handlers.ContainerProxyHandler) handlers.ContainerProxyHandler {
	wrapper := NewContainerHandlerMetricsWrapper(handler)

	// 핸들러 초기화 메트릭 기록
	containerMetrics := metrics.GetContainerMetrics()
	containerMetrics.RecordHandlerInitialization(handler.Type(), true)
	containerMetrics.SetActiveHandlerInstances(handler.Type(), 1)

	return wrapper
}
