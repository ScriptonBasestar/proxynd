package handlers

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"proxynd/cache"
	"proxynd/internal/app"
	"proxynd/internal/errors"
	"proxynd/logging"
)

// BaseProxyHandlerImpl Template Method 패턴을 구현하는 기본 프록시 핸들러
type BaseProxyHandlerImpl struct {
	container *app.Container
	handler   BaseProxyHandler
	cache     cache.Cache
	client    *http.Client
	logger    logging.Logger
}

// NewBaseProxyHandlerImpl 새로운 BaseProxyHandler 생성
func NewBaseProxyHandlerImpl(container *app.Container, handler BaseProxyHandler) *BaseProxyHandlerImpl {
	return &BaseProxyHandlerImpl{
		container: container,
		handler:   handler,
		cache:     container.Cache(),
		client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		logger: logging.GetLogger(),
	}
}

// Handle 공통 처리 로직 (Template Method 패턴)
func (b *BaseProxyHandlerImpl) Handle(c *fiber.Ctx) error {
	start := time.Now()
	proxyType := b.handler.Type()
	
	b.logRequest(c, proxyType)

	// 1. 설정 확인
	if !b.handler.IsEnabled() {
		return errors.NewError("PROXY001", "프록시가 비활성화되어 있습니다").
			WithDomain(proxyType).
			Build()
	}

	// 2. 캐시 조회
	cacheKey := b.handler.GenerateCacheKey(c)
	if cached, err := b.getFromCache(cacheKey); err == nil {
		c.Set("X-Cache-Status", "HIT")
		c.Set("X-Proxy-Type", proxyType)
		b.logResponse(c, proxyType, time.Since(start))
		return c.Send(cached)
	}

	// 3. 업스트림 URL 구성
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

	// 4. 업스트림 요청 생성
	req, err := b.createUpstreamRequest(c, upstreamURL)
	if err != nil {
		return errors.NewError("PROXY003", "업스트림 요청 생성 실패").
			WithDomain(proxyType).
			WithCause(err).
			Build()
	}

	// 5. 요청 변환 (프록시별 커스터마이징)
	fiberReq := fiber.AcquireRequest()
	defer fiber.ReleaseRequest(fiberReq)
	
	// HTTP 요청을 Fiber 요청으로 변환
	b.convertHTTPToFiberRequest(req, fiberReq)
	
	if err := b.handler.TransformRequest(c, fiberReq); err != nil {
		handlerErr := b.handler.HandleError(err, c)
		if handlerErr != nil {
			return handlerErr
		}
		return err
	}

	// 변경된 Fiber 요청을 다시 HTTP 요청에 적용
	b.applyFiberRequestToHTTP(fiberReq, req)

	// 6. 업스트림 요청 실행
	resp, err := b.client.Do(req)
	if err != nil {
		handlerErr := b.handler.HandleError(err, c)
		if handlerErr != nil {
			return handlerErr
		}
		return errors.NewError("PROXY004", "업스트림 서버 연결 실패").
			WithDomain(proxyType).
			WithCause(err).
			WithDetails(map[string]string{
				"upstream_url": upstreamURL,
			}).
			Build()
	}
	defer resp.Body.Close()

	// 7. 응답 읽기
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return errors.NewError("PROXY005", "응답 읽기 실패").
			WithDomain(proxyType).
			WithCause(err).
			Build()
	}

	// 8. 응답 변환 (프록시별 커스터마이징)
	transformed, err := b.handler.TransformResponse(body, c)
	if err != nil {
		handlerErr := b.handler.HandleError(err, c)
		if handlerErr != nil {
			return handlerErr
		}
		return err
	}

	// 9. 캐시 저장 (조건부)
	if b.handler.ShouldCache(c, resp.StatusCode) {
		ttl := b.handler.GetCacheTTL(c)
		if err := b.saveToCache(cacheKey, transformed, ttl); err != nil {
			// 캐시 저장 실패는 로그만, 계속 진행
			b.logCacheError("save", err, proxyType)
		}
	}

	// 10. 헤더 전달
	b.forwardHeaders(c, resp)
	c.Set("X-Cache-Status", "MISS")
	c.Set("X-Proxy-Type", proxyType)

	// 11. 응답 전송
	b.logResponse(c, proxyType, time.Since(start))
	return c.Status(resp.StatusCode).Send(transformed)
}

// 캐시 조회
func (b *BaseProxyHandlerImpl) getFromCache(key string) ([]byte, error) {
	data, err := b.cache.Get(key)
	if err != nil {
		return nil, err
	}

	// 메트릭 기록 (MetricsAwareProxyHandler인 경우)
	if metricsHandler, ok := b.handler.(MetricsAwareProxyHandler); ok {
		metricsHandler.RecordCacheMetrics(key, true, len(data))
	}
	
	return data, nil
}

// 캐시 저장
func (b *BaseProxyHandlerImpl) saveToCache(key string, data []byte, ttl time.Duration) error {
	err := b.cache.SetWithTTL(key, data, ttl)
	
	// 메트릭 기록 (MetricsAwareProxyHandler인 경우)
	if metricsHandler, ok := b.handler.(MetricsAwareProxyHandler); ok {
		metricsHandler.RecordCacheMetrics(key, false, len(data))
	}
	
	return err
}

// 업스트림 요청 생성
func (b *BaseProxyHandlerImpl) createUpstreamRequest(c *fiber.Ctx, url string) (*http.Request, error) {
	req, err := http.NewRequest(c.Method(), url, bytes.NewReader(c.Body()))
	if err != nil {
		return nil, err
	}

	// 기본 헤더 복사 (홉바이홉 헤더 제외)
	c.Request().Header.VisitAll(func(key, value []byte) {
		k := string(key)
		if !b.isHopByHopHeader(k) {
			req.Header.Set(k, string(value))
		}
	})

	return req, nil
}

// 응답 헤더 전달
func (b *BaseProxyHandlerImpl) forwardHeaders(c *fiber.Ctx, resp *http.Response) {
	for key, values := range resp.Header {
		if !b.isHopByHopHeader(key) && len(values) > 0 {
			c.Set(key, values[0])
		}
	}
}

// 홉바이홉 헤더 검사
func (b *BaseProxyHandlerImpl) isHopByHopHeader(header string) bool {
	hopByHopHeaders := []string{
		"Connection",
		"Keep-Alive",
		"Proxy-Authenticate",
		"Proxy-Authorization",
		"Te",
		"Trailers",
		"Transfer-Encoding",
		"Upgrade",
	}

	for _, h := range hopByHopHeaders {
		if strings.EqualFold(header, h) {
			return true
		}
	}
	return false
}

// HTTP 요청을 Fiber 요청으로 변환
func (b *BaseProxyHandlerImpl) convertHTTPToFiberRequest(httpReq *http.Request, fiberReq *fiber.Request) {
	fiberReq.SetRequestURI(httpReq.URL.String())
	
	for key, values := range httpReq.Header {
		if len(values) > 0 {
			fiberReq.Header.Set(key, values[0])
		}
	}
}

// Fiber 요청을 HTTP 요청에 적용
func (b *BaseProxyHandlerImpl) applyFiberRequestToHTTP(fiberReq *fiber.Request, httpReq *http.Request) {
	// 헤더 적용
	fiberReq.Header.VisitAll(func(key, value []byte) {
		httpReq.Header.Set(string(key), string(value))
	})
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