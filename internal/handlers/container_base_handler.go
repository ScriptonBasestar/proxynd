package handlers

import (
	"time"

	"github.com/gofiber/fiber/v2"

	"proxynd/internal/container"
	"proxynd/logging"
)

// ContainerBaseHandler Container 기반 핸들러의 기본 구현체
type ContainerBaseHandler struct {
	containerProvider container.ContainerProvider
	logger            logging.Logger
	name              string
	proxyType         string
}

// NewContainerBaseHandler 새로운 Container 기반 핸들러 생성
func NewContainerBaseHandler(name, proxyType string) *ContainerBaseHandler {
	return &ContainerBaseHandler{
		logger:    logging.GetLogger(),
		name:      name,
		proxyType: proxyType,
	}
}

// Handler 인터페이스 구현
func (h *ContainerBaseHandler) Handle(c *fiber.Ctx) error {
	// 기본 구현: 자식 클래스에서 오버라이드
	return fiber.NewError(fiber.StatusNotImplemented, "Handler not implemented")
}

func (h *ContainerBaseHandler) Name() string {
	return h.name
}

func (h *ContainerBaseHandler) Type() string {
	return h.proxyType
}

// ContainerAwareHandler 인터페이스 구현
func (h *ContainerBaseHandler) SetContainer(provider container.ContainerProvider) {
	h.containerProvider = provider
}

func (h *ContainerBaseHandler) GetContainer() container.ContainerProvider {
	return h.containerProvider
}

func (h *ContainerBaseHandler) LoadConfig() error {
	// 기본 구현: 자식 클래스에서 오버라이드
	return nil
}

func (h *ContainerBaseHandler) ReloadConfig() error {
	// 기본 구현: 자식 클래스에서 오버라이드
	return nil
}

// BaseProxyHandler 인터페이스 기본 구현
func (h *ContainerBaseHandler) IsEnabled() bool {
	// 기본 구현: 자식 클래스에서 오버라이드
	return true
}

func (h *ContainerBaseHandler) GenerateCacheKey(c *fiber.Ctx) string {
	// 기본 캐시 키 생성: 프록시타입_메소드_경로
	return h.proxyType + "_" + c.Method() + "_" + c.Path()
}

func (h *ContainerBaseHandler) BuildUpstreamURL(c *fiber.Ctx) (string, error) {
	// 기본 구현: 자식 클래스에서 오버라이드
	return "", fiber.NewError(fiber.StatusNotImplemented, "BuildUpstreamURL not implemented")
}

func (h *ContainerBaseHandler) TransformRequest(c *fiber.Ctx, upstreamReq *fiber.Agent) error {
	// 기본 요청 변환: 헤더 복사
	//nolint:staticcheck // SA1019: VisitAll provides better performance than alternatives
	c.Request().Header.VisitAll(func(key, value []byte) {
		upstreamReq.Set(string(key), string(value))
	})
	return nil
}

func (h *ContainerBaseHandler) TransformResponse(resp []byte, c *fiber.Ctx) ([]byte, error) {
	// 기본 응답 변환: 원본 그대로 반환
	return resp, nil
}

func (h *ContainerBaseHandler) ShouldCache(c *fiber.Ctx, statusCode int) bool {
	// 기본 캐싱 정책: 200 응답만 캐시
	return statusCode == fiber.StatusOK
}

func (h *ContainerBaseHandler) GetCacheTTL(c *fiber.Ctx) time.Duration {
	// 기본 TTL: 1시간
	return time.Hour
}

func (h *ContainerBaseHandler) HandleError(err error, c *fiber.Ctx) error {
	h.logger.Error("Proxy handler error",
		logging.F("error", err),
		logging.F("type", h.proxyType),
		logging.F("path", c.Path()),
	)
	return err
}

// Healthable 인터페이스 구현
func (h *ContainerBaseHandler) HealthCheck() error {
	if h.containerProvider == nil {
		return fiber.NewError(fiber.StatusServiceUnavailable, "Container not initialized")
	}
	return nil
}

// Cacheable 인터페이스 구현
func (h *ContainerBaseHandler) IsCacheable(c *fiber.Ctx) bool {
	// GET 요청만 캐시 가능
	return c.Method() == fiber.MethodGet
}

func (h *ContainerBaseHandler) GetCacheKey(c *fiber.Ctx) string {
	return h.GenerateCacheKey(c)
}

// GetUpstreamURL ProxyHandler 인터페이스 구현
func (h *ContainerBaseHandler) GetUpstreamURL(c *fiber.Ctx) (string, error) {
	return h.BuildUpstreamURL(c)
}

func (h *ContainerBaseHandler) ModifyRequest(c *fiber.Ctx) error {
	// 기본 구현: 아무것도 하지 않음
	return nil
}

func (h *ContainerBaseHandler) ModifyResponse(c *fiber.Ctx) error {
	// 기본 구현: 아무것도 하지 않음
	return nil
}

// 유틸리티 메서드들
func (h *ContainerBaseHandler) GetLogger() logging.Logger {
	return h.logger
}

func (h *ContainerBaseHandler) IsContainerReady() bool {
	return h.containerProvider != nil
}
