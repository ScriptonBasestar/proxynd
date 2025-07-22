package handlers

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"

	"proxynd/cache"
	"proxynd/internal/errors"
	"proxynd/logging"
)

// BaseProxyHandlerImpl Template Method 패턴 구현체
type BaseProxyHandlerImpl struct {
	container interface{}
	handler   BaseProxyHandler
	cache     cache.Cache
	logger    logging.Logger
	client    *fiber.Client
}

// NewBaseProxyHandlerImpl 새로운 BaseProxyHandlerImpl 생성
func NewBaseProxyHandlerImpl(container interface{}, handler BaseProxyHandler) *BaseProxyHandlerImpl {
	// 간단한 인터페이스로 캐시 접근
	type CacheProvider interface {
		Cache() cache.Cache
	}

	var cacheInstance cache.Cache
	if cp, ok := container.(CacheProvider); ok {
		cacheInstance = cp.Cache()
	}

	return &BaseProxyHandlerImpl{
		container: container,
		handler:   handler,
		cache:     cacheInstance,
		logger:    logging.GetLogger(),
		client:    fiber.AcquireClient(),
	}
}

// Handle Template Method 패턴 메인 플로우
func (b *BaseProxyHandlerImpl) Handle(c *fiber.Ctx) error {
	proxyType := b.handler.Type()
	startTime := time.Now()

	// 요청 로깅
	b.logRequest(c, proxyType)
	defer func() {
		b.logResponse(c, proxyType, time.Since(startTime))
	}()

	// 1. 프록시 활성화 확인
	if !b.handler.IsEnabled() {
		return errors.NewError("PROXY001", "프록시가 비활성화되어 있습니다").
			WithDomain(proxyType).
			Build()
	}

	// 2. 캐시 키 생성
	cacheKey := b.handler.GenerateCacheKey(c)

	// 3. 캐시 조회
	if cachedData, found := b.cache.Get(cacheKey); found && cachedData != nil {
		// 캐시 히트
		c.Set("X-Cache-Status", "HIT")
		c.Set("X-Proxy-Type", proxyType)

		// 캐시 메트릭 기록
		if metricsHandler, ok := b.handler.(MetricsAwareProxyHandler); ok {
			metricsHandler.RecordCacheMetrics(cacheKey, true, len(cachedData))
		}

		return c.Send(cachedData)
	}

	// 캐시 미스
	c.Set("X-Cache-Status", "MISS")
	c.Set("X-Proxy-Type", proxyType)

	// 4. 업스트림 URL 구성
	upstreamURL, err := b.handler.BuildUpstreamURL(c)
	if err != nil {
		handlerErr := b.handler.HandleError(err, c)
		if handlerErr != nil {
			return handlerErr
		}
		return errors.NewError("PROXY002", "업스트림 URL 구성 실패").
			WithDomain(proxyType).
			WithCause(err).
			Build()
	}

	// 5. Fiber Agent 생성 및 요청 설정
	agent := b.client.Get(upstreamURL)

	// 기본 헤더 복사 (hop-by-hop 헤더 제외)
	for key, values := range c.GetReqHeaders() {
		if !b.isHopByHopHeader(key) {
			// 복수 값이 있는 경우 첫 번째 값 사용
			if len(values) > 0 {
				agent.Set(key, values[0])
			}
		}
	}

	// 프록시별 요청 변환
	if err := b.handler.TransformRequest(c, agent); err != nil {
		handlerErr := b.handler.HandleError(err, c)
		if handlerErr != nil {
			return handlerErr
		}
		return err
	}

	// 6. 업스트림 요청 실행
	statusCode, body, errs := agent.Bytes()
	if len(errs) > 0 {
		// 에러 로깅
		b.logger.Error("업스트림 요청 실패",
			logging.F("proxy_type", proxyType),
			logging.F("upstream_url", upstreamURL),
			logging.F("error", errs[0]),
		)

		// 에러 처리
		handlerErr := b.handler.HandleError(errs[0], c)
		if handlerErr != nil {
			return handlerErr
		}
		return errors.NewError("PROXY003", "업스트림 서버 연결 실패").
			WithDomain(proxyType).
			WithCause(errs[0]).
			WithDetails(map[string]string{
				"upstream_url": upstreamURL,
			}).
			Build()
	}

	// 7. 응답 변환 (프록시별 커스터마이징)
	transformed, err := b.handler.TransformResponse(body, c)
	if err != nil {
		handlerErr := b.handler.HandleError(err, c)
		if handlerErr != nil {
			return handlerErr
		}
		// 변환 실패 시 원본 응답 사용
		transformed = body
	}

	// 8. 캐시 저장 (비동기)
	if b.handler.ShouldCache(c, statusCode) {
		ttl := b.handler.GetCacheTTL(c)
		go func() {
			if err := b.cache.Put(cacheKey, transformed, ttl); err != nil {
				b.logCacheError("save", err, proxyType)
			} else {
				// 캐시 메트릭 기록
				if metricsHandler, ok := b.handler.(MetricsAwareProxyHandler); ok {
					metricsHandler.RecordCacheMetrics(cacheKey, false, len(transformed))
				}
			}
		}()
	}

	// 9. 응답 헤더 설정
	// Note: Fiber Agent doesn't expose response headers directly
	// Headers are set via the fiber context during agent execution

	// 10. 응답 전송
	c.Status(statusCode)
	return c.Send(transformed)
}

// hop-by-hop 헤더 확인
func (b *BaseProxyHandlerImpl) isHopByHopHeader(header string) bool {
	hopByHopHeaders := map[string]bool{
		"Connection":          true,
		"Keep-Alive":          true,
		"Proxy-Authenticate":  true,
		"Proxy-Authorization": true,
		"TE":                  true,
		"Trailers":            true,
		"Transfer-Encoding":   true,
		"Upgrade":             true,
	}
	return hopByHopHeaders[header]
}

// 요청 로깅
func (b *BaseProxyHandlerImpl) logRequest(c *fiber.Ctx, proxyType string) {
	b.logger.Info("Proxy request received",
		logging.F("proxy_type", proxyType),
		logging.F("method", c.Method()),
		logging.F("path", c.Path()),
		logging.F("user_agent", c.Get("User-Agent")),
		logging.F("remote_ip", c.IP()),
	)
}

// 응답 로깅
func (b *BaseProxyHandlerImpl) logResponse(c *fiber.Ctx, proxyType string, duration time.Duration) {
	level := "info"
	statusCode := c.Response().StatusCode()
	if statusCode >= 400 {
		level = "error"
	}

	message := "Proxy request completed"
	fields := []logging.Field{
		logging.F("proxy_type", proxyType),
		logging.F("method", c.Method()),
		logging.F("path", c.Path()),
		logging.F("status", statusCode),
		logging.F("duration_ms", duration.Milliseconds()),
		logging.F("cache_status", c.Get("X-Cache-Status")),
	}

	// 메트릭 기록 (MetricsAwareProxyHandler인 경우)
	if metricsHandler, ok := b.handler.(MetricsAwareProxyHandler); ok {
		metricsHandler.RecordRequestMetrics(c, statusCode, duration)
	}

	switch level {
	case "error":
		b.logger.Error(message, fields...)
	default:
		b.logger.Info(message, fields...)
	}
}

// 캐시 에러 로깅
func (b *BaseProxyHandlerImpl) logCacheError(operation string, err error, proxyType string) {
	b.logger.Warn("Cache operation failed",
		logging.F("proxy_type", proxyType),
		logging.F("operation", operation),
		logging.F("error", err.Error()),
	)
}

// Name 핸들러 이름 반환
func (b *BaseProxyHandlerImpl) Name() string {
	return fmt.Sprintf("base-proxy-%s", b.handler.Type())
}

// Type 핸들러 타입 반환
func (b *BaseProxyHandlerImpl) Type() string {
	return b.handler.Type()
}

// HealthCheck 헬스체크
func (b *BaseProxyHandlerImpl) HealthCheck() error {
	if !b.handler.IsEnabled() {
		return fmt.Errorf("프록시 '%s'가 비활성화되어 있습니다", b.handler.Type())
	}

	// 핸들러별 추가 헬스체크 (Healthable 인터페이스 구현 시)
	if healthableHandler, ok := b.handler.(Healthable); ok {
		return healthableHandler.HealthCheck()
	}

	return nil
}
