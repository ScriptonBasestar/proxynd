package handlers

import (
	"github.com/gofiber/fiber/v2"

	"proxynd/internal/app"
)

// Handler 기본 핸들러 인터페이스
type Handler interface {
	Handle(c *fiber.Ctx) error
	Name() string
	Type() string
}

// HandlerFactory 핸들러 팩토리 인터페이스
type HandlerFactory interface {
	Create(container *app.Container) Handler
}

// Configurable 설정 가능한 핸들러 인터페이스
type Configurable interface {
	Configure(config interface{}) error
}

// Testable 테스트 가능한 핸들러 인터페이스
type Testable interface {
	Test() error
}

// Healthable 헬스체크 가능한 핸들러 인터페이스
type Healthable interface {
	HealthCheck() error
}

// Cacheable 캐시 가능한 핸들러 인터페이스
type Cacheable interface {
	IsCacheable(c *fiber.Ctx) bool
	GetCacheKey(c *fiber.Ctx) string
}

// Authenticable 인증 가능한 핸들러 인터페이스
type Authenticable interface {
	RequiresAuth(c *fiber.Ctx) bool
	Authenticate(c *fiber.Ctx) error
}

// Loggable 로깅 가능한 핸들러 인터페이스
type Loggable interface {
	ShouldLog(c *fiber.Ctx) bool
	GetLogLevel(c *fiber.Ctx) string
}

// Metrics 메트릭 수집 가능한 핸들러 인터페이스
type Metrics interface {
	RecordMetrics(c *fiber.Ctx, duration int64, statusCode int)
}

// ProxyHandler 프록시 핸들러 특화 인터페이스
type ProxyHandler interface {
	Handler
	Healthable
	Cacheable
	GetUpstreamURL(c *fiber.Ctx) (string, error)
	ModifyRequest(c *fiber.Ctx) error
	ModifyResponse(c *fiber.Ctx) error
}

// ComposeHandler 복합 핸들러 인터페이스 (모든 기능 포함)
type ComposeHandler interface {
	Handler
	Configurable
	Testable
	Healthable
	Cacheable
	Authenticable
	Loggable
	Metrics
}
