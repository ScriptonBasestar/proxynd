package handlers

import (
	"net/http"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"proxynd/internal/app"
)

// TestHandler 테스트용 핸들러 구현체
type TestHandler struct {
	*BaseHandler
	testError error
}

func NewTestHandler(container *app.Container, name, proxyType string) *TestHandler {
	return &TestHandler{
		BaseHandler: NewBaseHandler(container, name, proxyType),
	}
}

func (h *TestHandler) Handle(c *fiber.Ctx) error {
	if h.testError != nil {
		return h.testError
	}
	return c.JSON(fiber.Map{
		"handler": h.Name(),
		"type":    h.Type(),
		"path":    c.Path(),
		"method":  c.Method(),
	})
}

func (h *TestHandler) SetTestError(err error) {
	h.testError = err
}

func TestHandlerInterfaces(t *testing.T) {
	// 임시 컨테이너 생성 (실제 구현에서는 실제 Container 사용)
	cfg := &app.Config{
		ConfigDir: "/tmp/test-config",
		Port:      "8080",
	}
	container := app.NewContainer(cfg)
	defer container.Close()

	// BaseHandler가 모든 인터페이스를 구현하는지 확인
	t.Run("BaseHandler implements all interfaces", func(t *testing.T) {
		handler := NewBaseHandler(container, "test-handler", "test")

		// 기본 Handler 인터페이스
		assert.Implements(t, (*Handler)(nil), handler)
		assert.Equal(t, "test-handler", handler.Name())
		assert.Equal(t, "test", handler.Type())

		// 추가 인터페이스들
		assert.Implements(t, (*Healthable)(nil), handler)
		assert.Implements(t, (*Cacheable)(nil), handler)
		assert.Implements(t, (*Authenticable)(nil), handler)
		assert.Implements(t, (*Loggable)(nil), handler)
		assert.Implements(t, (*Metrics)(nil), handler)
		assert.Implements(t, (*Testable)(nil), handler)
		assert.Implements(t, (*Configurable)(nil), handler)
	})

	// TestHandler 기본 동작 테스트
	t.Run("TestHandler basic functionality", func(t *testing.T) {
		handler := NewTestHandler(container, "test-handler", "test")

		// Fiber 앱 생성
		app := fiber.New()
		app.Get("/test", handler.Handle)

		// 테스트 요청 (HTTP 패키지 사용)
		req, err := http.NewRequest("GET", "/test", nil)
		require.NoError(t, err)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
	})

	// HealthCheck 테스트
	t.Run("HealthCheck functionality", func(t *testing.T) {
		handler := NewBaseHandler(container, "test-handler", "test")

		err := handler.HealthCheck()
		assert.NoError(t, err)
	})

	// 캐시 기능 테스트
	t.Run("Cache functionality", func(t *testing.T) {
		handler := NewBaseHandler(container, "test-handler", "test")

		// Fiber 앱 생성 및 핸들러 등록
		app := fiber.New()

		// GET 요청 테스트
		app.Get("/test", func(c *fiber.Ctx) error {
			assert.True(t, handler.IsCacheable(c))
			cacheKey := handler.GetCacheKey(c)
			assert.Contains(t, cacheKey, "test")
			return c.SendString("OK")
		})

		// POST 요청 테스트
		app.Post("/test", func(c *fiber.Ctx) error {
			assert.False(t, handler.IsCacheable(c))
			return c.SendString("OK")
		})

		// GET 요청 테스트 실행
		req, err := http.NewRequest("GET", "/test", nil)
		require.NoError(t, err)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)

		// POST 요청 테스트 실행
		req, err = http.NewRequest("POST", "/test", nil)
		require.NoError(t, err)

		resp, err = app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
	})

	// 인증 기능 테스트
	t.Run("Authentication functionality", func(t *testing.T) {
		handler := NewBaseHandler(container, "test-handler", "test")

		// Fiber 앱 생성 및 테스트 컨텍스트 생성
		app := fiber.New()
		app.Get("/test", func(c *fiber.Ctx) error {
			// 기본적으로 인증 불필요
			assert.False(t, handler.RequiresAuth(c))

			// 인증 통과
			err := handler.Authenticate(c)
			assert.NoError(t, err)
			return c.SendString("OK")
		})

		// 테스트 요청
		req, err := http.NewRequest("GET", "/test", nil)
		require.NoError(t, err)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
	})

	// 로깅 기능 테스트
	t.Run("Logging functionality", func(t *testing.T) {
		handler := NewBaseHandler(container, "test-handler", "test")

		// Fiber 앱 생성 및 테스트 컨텍스트 생성
		app := fiber.New()
		app.Get("/test", func(c *fiber.Ctx) error {
			// 기본적으로 로깅 활성화
			assert.True(t, handler.ShouldLog(c))

			// 성공 응답의 로그 레벨 (기본 200)
			assert.Equal(t, "info", handler.GetLogLevel(c))

			return c.SendString("OK")
		})

		// 에러 응답 테스트
		app.Get("/error", func(c *fiber.Ctx) error {
			// 상태 코드 먼저 설정
			c.Status(500)
			// 에러 응답의 로그 레벨
			assert.Equal(t, "error", handler.GetLogLevel(c))
			return c.SendString("Error")
		})

		// 성공 요청 테스트
		req, err := http.NewRequest("GET", "/test", nil)
		require.NoError(t, err)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)

		// 에러 요청 테스트
		req, err = http.NewRequest("GET", "/error", nil)
		require.NoError(t, err)

		resp, err = app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 500, resp.StatusCode)
	})

	// 메트릭 기능 테스트
	t.Run("Metrics functionality", func(t *testing.T) {
		handler := NewBaseHandler(container, "test-handler", "test")

		// Fiber 앱 생성 및 테스트 컨텍스트 생성
		app := fiber.New()
		app.Get("/test", func(c *fiber.Ctx) error {
			// 메트릭 기록 (패닉 없이 완료되어야 함)
			assert.NotPanics(t, func() {
				handler.RecordMetrics(c, int64(time.Millisecond*100), 200)
			})
			return c.SendString("OK")
		})

		// 테스트 요청
		req, err := http.NewRequest("GET", "/test", nil)
		require.NoError(t, err)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
	})

	// 테스트 기능 테스트
	t.Run("Test functionality", func(t *testing.T) {
		handler := NewBaseHandler(container, "test-handler", "test")

		err := handler.Test()
		assert.NoError(t, err)
	})

	// 설정 기능 테스트
	t.Run("Configuration functionality", func(t *testing.T) {
		handler := NewBaseHandler(container, "test-handler", "test")

		// 기본 설정 적용
		err := handler.Configure(map[string]interface{}{
			"test_key": "test_value",
		})
		assert.NoError(t, err)

		// 설정 가져오기
		config := handler.GetConfig()
		assert.NotNil(t, config)
	})
}

func TestHandlerCompatibility(t *testing.T) {
	// 컨테이너 생성
	cfg := &app.Config{
		ConfigDir: "/tmp/test-config",
		Port:      "8080",
	}
	container := app.NewContainer(cfg)
	defer container.Close()

	// BaseHandler가 ComposeHandler 인터페이스를 구현하는지 확인
	t.Run("BaseHandler implements ComposeHandler", func(t *testing.T) {
		handler := NewBaseHandler(container, "test-handler", "test")

		// ComposeHandler 인터페이스 확인
		assert.Implements(t, (*ComposeHandler)(nil), handler)

		// 모든 메서드가 호출 가능한지 확인
		assert.Equal(t, "test-handler", handler.Name())
		assert.Equal(t, "test", handler.Type())
		assert.NoError(t, handler.HealthCheck())
		assert.NoError(t, handler.Test())
		assert.NoError(t, handler.Configure(nil))

		// Fiber 컨텍스트 필요한 메서드들
		app := fiber.New()
		app.Get("/test", func(c *fiber.Ctx) error {
			assert.NotPanics(t, func() {
				handler.IsCacheable(c)
				handler.GetCacheKey(c)
				handler.RequiresAuth(c)
				handler.Authenticate(c)
				handler.ShouldLog(c)
				handler.GetLogLevel(c)
				handler.RecordMetrics(c, 100, 200)
			})
			return c.SendString("OK")
		})

		// 테스트 요청
		req, err := http.NewRequest("GET", "/test", nil)
		require.NoError(t, err)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
	})

	// 커스텀 핸들러가 인터페이스를 올바르게 구현하는지 확인
	t.Run("Custom handler implements interfaces", func(t *testing.T) {
		handler := NewTestHandler(container, "custom-handler", "custom")

		// 기본 인터페이스
		assert.Implements(t, (*Handler)(nil), handler)
		assert.Equal(t, "custom-handler", handler.Name())
		assert.Equal(t, "custom", handler.Type())

		// 추가 인터페이스들
		assert.Implements(t, (*Healthable)(nil), handler)
		assert.Implements(t, (*Cacheable)(nil), handler)
		assert.Implements(t, (*Authenticable)(nil), handler)
		assert.Implements(t, (*Loggable)(nil), handler)
		assert.Implements(t, (*Metrics)(nil), handler)
		assert.Implements(t, (*Testable)(nil), handler)
		assert.Implements(t, (*Configurable)(nil), handler)
		assert.Implements(t, (*ComposeHandler)(nil), handler)
	})
}

func TestInterfaceConformance(t *testing.T) {
	// 각 인터페이스가 올바르게 정의되었는지 확인
	t.Run("Interface definitions", func(t *testing.T) {
		// 컴파일 타임 인터페이스 확인
		var _ Handler = (*BaseHandler)(nil)
		var _ Healthable = (*BaseHandler)(nil)
		var _ Cacheable = (*BaseHandler)(nil)
		var _ Authenticable = (*BaseHandler)(nil)
		var _ Loggable = (*BaseHandler)(nil)
		var _ Metrics = (*BaseHandler)(nil)
		var _ Testable = (*BaseHandler)(nil)
		var _ Configurable = (*BaseHandler)(nil)
		var _ ComposeHandler = (*BaseHandler)(nil)

		// 커스텀 핸들러도 확인
		var _ Handler = (*TestHandler)(nil)
		var _ ComposeHandler = (*TestHandler)(nil)

		// 인터페이스 확인 완료
		assert.True(t, true)
	})
}
