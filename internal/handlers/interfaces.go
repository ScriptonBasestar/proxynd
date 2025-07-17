package handlers

import (
	"time"

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

// BaseProxyHandler Template Method 패턴을 위한 프록시 핸들러 인터페이스
type BaseProxyHandler interface {
	// 프록시 타입 식별자 반환
	Type() string

	// 설정 검증 및 활성화 상태 확인
	IsEnabled() bool

	// 캐시 키 생성 (프록시별 고유 로직)
	GenerateCacheKey(c *fiber.Ctx) string

	// 업스트림 URL 구성 (미러 선택, 경로 변환 등)
	BuildUpstreamURL(c *fiber.Ctx) (string, error)

	// 요청 변환 (헤더 추가, 인증 정보 등)
	TransformRequest(c *fiber.Ctx, upstreamReq *fiber.Agent) error

	// 응답 변환 (압축 해제, 포맷 변경 등)
	TransformResponse(resp []byte, c *fiber.Ctx) ([]byte, error)

	// 캐시 정책 결정
	ShouldCache(c *fiber.Ctx, statusCode int) bool
	GetCacheTTL(c *fiber.Ctx) time.Duration

	// 에러 처리 (프록시별 특화 에러)
	HandleError(err error, c *fiber.Ctx) error
}

// CacheableProxyHandler 캐시 지원 프록시 핸들러 인터페이스
type CacheableProxyHandler interface {
	BaseProxyHandler

	// 캐시 무효화 조건
	ShouldInvalidateCache(c *fiber.Ctx) bool

	// 캐시 우선순위 (용량 제한 시 사용)
	GetCachePriority(c *fiber.Ctx) int
}

// AuthenticatedProxyHandler 인증이 필요한 프록시 핸들러 인터페이스
type AuthenticatedProxyHandler interface {
	BaseProxyHandler

	// 업스트림 인증 정보 구성
	GetUpstreamAuth(c *fiber.Ctx) (username, password string, err error)

	// 클라이언트 인증 검증
	ValidateClientAuth(c *fiber.Ctx) error
}

// MetricsAwareProxyHandler 메트릭 수집이 가능한 프록시 핸들러 인터페이스
type MetricsAwareProxyHandler interface {
	BaseProxyHandler

	// 요청 메트릭 기록
	RecordRequestMetrics(c *fiber.Ctx, statusCode int, duration time.Duration)

	// 캐시 메트릭 기록
	RecordCacheMetrics(cacheKey string, hit bool, size int)
}
