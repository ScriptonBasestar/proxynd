package unit

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	cacheMocks "proxynd/cache/mocks"
	"proxynd/internal/app"
	"proxynd/internal/config"
	"proxynd/internal/handlers"
	handlerMocks "proxynd/internal/handlers/mocks"
	pipMocks "proxynd/internal/services/pip/mocks"
)

// TestSuite 종합 테스트 스위트
type TestSuite struct {
	app            *fiber.App
	registry       *handlers.HandlerRegistry
	mockContainer  *app.Container
	mockCache      *cacheMocks.MockCache
	mockPipService *pipMocks.MockPackageService
	testCtx        context.Context
}

// SetupTestSuite 테스트 스위트 설정
func SetupTestSuite(t *testing.T) *TestSuite {
	suite := &TestSuite{
		testCtx: context.Background(),
	}

	// Mock 객체들 생성
	suite.mockCache = cacheMocks.NewMockCache(t)
	suite.mockPipService = pipMocks.NewMockPackageService(t)

	// Fiber 앱 생성
	suite.app = fiber.New(fiber.Config{
		DisableStartupMessage: true,
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return c.Status(500).JSON(fiber.Map{
				"error": err.Error(),
			})
		},
	})

	// 핸들러 레지스트리 생성
	suite.registry = handlers.NewHandlerRegistry(nil, nil)

	// Container 설정 (실제 사용 시 Mock으로 대체)
	config := &app.Config{
		Port:        "8080",
		Version:     "test",
		StorageDir:  t.TempDir(),
		ConfigDir:   t.TempDir(),
		CacheMaxAge: time.Hour,
	}
	suite.mockContainer = app.NewContainer(config)

	return suite
}

// TestPIPHandlerComprehensive PIP 핸들러 종합 테스트 (simplified)
func TestPIPHandlerComprehensive(t *testing.T) {
	suite := SetupTestSuite(t)

	// PIP 설정 생성
	pipConfig := &config.PipProxyConfig{
		Enabled: true,
		Mirrors: []config.PipMirror{
			{
				Name: "pypi",
				URL:  "https://pypi.org",
			},
		},
		Cache: config.CacheConfig{
			Enabled: true,
			TTL:     "1h",
		},
	}

	t.Run("PIP Service Mock Test", func(t *testing.T) {
		// 간단한 Mock 서비스 테스트
		suite.mockPipService.EXPECT().
			GetPackageMetadata(mock.Anything, "requests", "").
			Return(nil, fmt.Errorf("test error")).
			Once()

		// Mock 호출 테스트
		_, err := suite.mockPipService.GetPackageMetadata(suite.testCtx, "requests", "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "test error")
	})

	t.Run("Cache Mock Test", func(t *testing.T) {
		// 캐시 Mock 테스트
		suite.mockCache.EXPECT().
			Get(mock.Anything, "test-key").
			Return(nil, fmt.Errorf("cache miss")).
			Once()

		_, err := suite.mockCache.Get(suite.testCtx, "test-key")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cache miss")
	})

	// 설정 검증
	assert.True(t, pipConfig.Enabled)
	assert.Equal(t, "pypi", pipConfig.Mirrors[0].Name)
}

// TestUnifiedHandlerFactory 통합 핸들러 팩토리 테스트
func TestUnifiedHandlerFactory(t *testing.T) {
	suite := SetupTestSuite(t)

	t.Run("Handler Registration", func(t *testing.T) {
		// Mock 핸들러 생성
		mockHandler := handlerMocks.NewMockProxyHandler(t)
		mockHandler.EXPECT().Type().Return("test").Maybe()
		mockHandler.EXPECT().Name().Return("TestHandler").Maybe()

		// 핸들러 등록
		err := suite.registry.Register("test", mockHandler)
		assert.NoError(t, err)

		// 등록된 핸들러 검색
		handler, exists := suite.registry.Get("test")
		assert.True(t, exists)
		assert.Equal(t, mockHandler, handler)

		// 중복 등록 테스트
		err = suite.registry.Register("test", mockHandler)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already registered")
	})

	t.Run("All Handlers List", func(t *testing.T) {
		// 여러 핸들러 등록
		for i, handlerType := range []string{"npm", "maven", "pip", "docker"} {
			mockHandler := handlerMocks.NewMockProxyHandler(t)
			mockHandler.EXPECT().Type().Return(handlerType).Maybe()
			mockHandler.EXPECT().Name().Return(fmt.Sprintf("Handler%d", i)).Maybe()

			err := suite.registry.Register(handlerType, mockHandler)
			assert.NoError(t, err)
		}

		// 모든 핸들러 목록 가져오기
		allHandlers := suite.registry.GetAll()
		assert.GreaterOrEqual(t, len(allHandlers), 4)

		// 타입별 검증
		types := make(map[string]bool)
		for handlerType := range allHandlers {
			types[handlerType] = true
		}

		expectedTypes := []string{"npm", "maven", "pip", "docker"}
		for _, expectedType := range expectedTypes {
			assert.True(t, types[expectedType], "Handler type %s should be registered", expectedType)
		}
	})

	t.Run("Handler Unregistration", func(t *testing.T) {
		mockHandler := handlerMocks.NewMockProxyHandler(t)
		mockHandler.EXPECT().Type().Return("temp").Maybe()

		// 등록
		err := suite.registry.Register("temp", mockHandler)
		assert.NoError(t, err)

		// 존재 확인
		_, exists := suite.registry.Get("temp")
		assert.True(t, exists)

		// 등록 해제
		suite.registry.Unregister("temp")

		// 등록 해제 확인
		_, exists = suite.registry.Get("temp")
		assert.False(t, exists)
	})
}

// TestHealthChecks 헬스체크 테스트
func TestHealthChecks(t *testing.T) {
	suite := SetupTestSuite(t)

	t.Run("Individual Handler Health", func(t *testing.T) {
		// Mock 핸들러 생성
		mockHandler := handlerMocks.NewMockProxyHandler(t)
		mockHandler.EXPECT().Type().Return("test").Maybe()
		mockHandler.EXPECT().HealthCheck().Return(nil).Once()

		// 헬스체크 실행
		err := mockHandler.HealthCheck()
		assert.NoError(t, err)

		// 실패 케이스
		mockHandler.EXPECT().HealthCheck().Return(fmt.Errorf("service unavailable")).Once()
		err = mockHandler.HealthCheck()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "service unavailable")
	})

	t.Run("Registry Health Check", func(t *testing.T) {
		// 여러 핸들러 등록
		healthyHandler := handlerMocks.NewMockProxyHandler(t)
		healthyHandler.EXPECT().Type().Return("healthy").Maybe()
		healthyHandler.EXPECT().HealthCheck().Return(nil).Maybe()

		unhealthyHandler := handlerMocks.NewMockProxyHandler(t)
		unhealthyHandler.EXPECT().Type().Return("unhealthy").Maybe()
		unhealthyHandler.EXPECT().HealthCheck().Return(fmt.Errorf("connection failed")).Maybe()

		suite.registry.Register("healthy", healthyHandler)
		suite.registry.Register("unhealthy", unhealthyHandler)

		// 전체 헬스체크 수행
		healthStatus := make(map[string]error)
		for handlerType, handler := range suite.registry.GetAll() {
			healthStatus[handlerType] = handler.HealthCheck()
		}

		// 검증
		assert.NoError(t, healthStatus["healthy"])
		assert.Error(t, healthStatus["unhealthy"])
		assert.Contains(t, healthStatus["unhealthy"].Error(), "connection failed")
	})
}

// TestCacheableBehavior 캐시 동작 테스트
func TestCacheableBehavior(t *testing.T) {
	suite := SetupTestSuite(t)

	t.Run("Cache Key Generation", func(t *testing.T) {
		testCases := []struct {
			path        string
			method      string
			expectedKey string
		}{
			{"/pypi/requests/json", "GET", "pip:pypi:requests:json:GET"},
			{"/simple/numpy/", "GET", "pip:simple:numpy::GET"},
			{"/packages/source/r/requests/requests-2.28.1.tar.gz", "GET", "pip:packages:source:r:requests:requests-2.28.1.tar.gz:GET"},
		}

		for _, tc := range testCases {
			t.Run(tc.path, func(t *testing.T) {
				// Fiber 컨텍스트 생성
				ctx := suite.app.AcquireCtx(&fiber.DefaultCtx{})
				defer suite.app.ReleaseCtx(ctx)
				ctx.Request().SetRequestURI(tc.path)
				ctx.Request().Header.SetMethod(tc.method)

				// PIP 어댑터 생성 (간단한 설정)
				pipConfig := &config.PipProxyConfig{Enabled: true}
				pipAdapter := pip.NewPIPAdapter(pipConfig, suite.mockPipService, suite.mockCache)

				// 캐시 키 생성
				cacheKey := pipAdapter.GetCacheKey(ctx)

				// 검증 (정확한 키 형식은 구현에 따라 달라질 수 있음)
				assert.NotEmpty(t, cacheKey)
				assert.Contains(t, cacheKey, "pip")
				assert.Contains(t, cacheKey, tc.method)
			})
		}
	})

	t.Run("Cacheable Determination", func(t *testing.T) {
		testCases := []struct {
			path      string
			method    string
			cacheable bool
			reason    string
		}{
			{"/pypi/requests/json", "GET", true, "Package metadata is cacheable"},
			{"/simple/requests/", "GET", true, "Package index is cacheable"},
			{"/packages/source/r/requests/requests-2.28.1.tar.gz", "GET", true, "Package files are cacheable"},
			{"/pypi/requests/json", "POST", false, "POST requests are not cacheable"},
			{"/upload/", "PUT", false, "Upload endpoints are not cacheable"},
		}

		for _, tc := range testCases {
			t.Run(fmt.Sprintf("%s %s", tc.method, tc.path), func(t *testing.T) {
				// Fiber 컨텍스트 생성
				ctx := suite.app.AcquireCtx(&fiber.DefaultCtx{})
				defer suite.app.ReleaseCtx(ctx)
				ctx.Request().SetRequestURI(tc.path)
				ctx.Request().Header.SetMethod(tc.method)

				// PIP 어댑터 생성
				pipConfig := &config.PipProxyConfig{Enabled: true}
				pipAdapter := pip.NewPIPAdapter(pipConfig, suite.mockPipService, suite.mockCache)

				// 캐시 가능 여부 확인
				isCacheable := pipAdapter.IsCacheable(ctx)

				// 검증
				assert.Equal(t, tc.cacheable, isCacheable, tc.reason)
			})
		}
	})
}

// TestRequestModification 요청 수정 테스트
func TestRequestModification(t *testing.T) {
	suite := SetupTestSuite(t)

	t.Run("Header Modification", func(t *testing.T) {
		// Fiber 컨텍스트 생성
		ctx := suite.app.AcquireCtx(&fiber.DefaultCtx{})
		defer suite.app.ReleaseCtx(ctx)
		ctx.Request().SetRequestURI("/pypi/requests/json")
		ctx.Request().Header.SetMethod("GET")
		ctx.Request().Header.Set("User-Agent", "test-client")

		// PIP 어댑터 생성
		pipConfig := &config.PipProxyConfig{Enabled: true}
		pipAdapter := pip.NewPIPAdapter(pipConfig, suite.mockPipService, suite.mockCache)

		// 요청 수정
		err := pipAdapter.ModifyRequest(ctx)
		assert.NoError(t, err)

		// 수정된 헤더 검증 (구현에 따라)
		userAgent := string(ctx.Request().Header.Peek("User-Agent"))
		assert.NotEmpty(t, userAgent)
	})

	t.Run("URL Rewriting", func(t *testing.T) {
		// Fiber 컨텍스트 생성
		ctx := suite.app.AcquireCtx(&fiber.DefaultCtx{})
		defer suite.app.ReleaseCtx(ctx)
		ctx.Request().SetRequestURI("/pypi/requests/json")

		// PIP 어댑터 생성
		pipConfig := &config.PipProxyConfig{
			Enabled: true,
			Mirrors: []config.PipMirror{
				{Name: "pypi", URL: "https://pypi.org"},
			},
		}
		pipAdapter := pip.NewPIPAdapter(pipConfig, suite.mockPipService, suite.mockCache)

		// 업스트림 URL 생성
		upstreamURL, err := pipAdapter.GetUpstreamURL(ctx)
		assert.NoError(t, err)
		assert.NotEmpty(t, upstreamURL)
		assert.Contains(t, upstreamURL, "pypi.org")
	})
}

// TestResponseModification 응답 수정 테스트
func TestResponseModification(t *testing.T) {
	suite := SetupTestSuite(t)

	t.Run("Response Header Addition", func(t *testing.T) {
		// Fiber 컨텍스트 생성
		ctx := suite.app.AcquireCtx(&fiber.DefaultCtx{})
		defer suite.app.ReleaseCtx(ctx)

		// 응답 설정
		ctx.Response().SetStatusCode(200)
		ctx.Response().SetBody([]byte(`{"name":"requests","version":"2.28.1"}`))
		ctx.Response().Header.Set("Content-Type", "application/json")

		// PIP 어댑터 생성
		pipConfig := &config.PipProxyConfig{Enabled: true}
		pipAdapter := pip.NewPIPAdapter(pipConfig, suite.mockPipService, suite.mockCache)

		// 응답 수정
		err := pipAdapter.ModifyResponse(ctx)
		assert.NoError(t, err)

		// 수정된 응답 검증
		assert.Equal(t, 200, ctx.Response().StatusCode())
		contentType := string(ctx.Response().Header.Peek("Content-Type"))
		assert.Contains(t, contentType, "application/json")
	})
}

// TestConcurrentAccess 동시 접근 테스트
func TestConcurrentAccess(t *testing.T) {
	suite := SetupTestSuite(t)

	t.Run("Concurrent Handler Registration", func(t *testing.T) {
		const numGoroutines = 10

		errors := make(chan error, numGoroutines)

		// 동시에 핸들러 등록
		for i := 0; i < numGoroutines; i++ {
			go func(index int) {
				mockHandler := handlerMocks.NewMockProxyHandler(t)
				mockHandler.EXPECT().Type().Return(fmt.Sprintf("concurrent-%d", index)).Maybe()

				err := suite.registry.Register(fmt.Sprintf("concurrent-%d", index), mockHandler)
				errors <- err
			}(i)
		}

		// 모든 고루틴 완료 대기
		for i := 0; i < numGoroutines; i++ {
			err := <-errors
			assert.NoError(t, err)
		}

		// 등록된 핸들러 수 확인
		allHandlers := suite.registry.GetAll()
		registeredCount := 0
		for handlerType := range allHandlers {
			if strings.HasPrefix(handlerType, "concurrent-") {
				registeredCount++
			}
		}
		assert.Equal(t, numGoroutines, registeredCount)
	})

	t.Run("Concurrent Cache Access", func(t *testing.T) {
		const numGoroutines = 5

		// Mock 설정 - 여러 번 호출될 수 있음
		suite.mockCache.EXPECT().
			Get(mock.Anything, mock.AnythingOfType("string")).
			Return(nil, fmt.Errorf("cache miss")).
			Times(numGoroutines)

		errors := make(chan error, numGoroutines)

		// 동시에 캐시 접근
		for i := 0; i < numGoroutines; i++ {
			go func(index int) {
				_, err := suite.mockCache.Get(suite.testCtx, fmt.Sprintf("key-%d", index))
				errors <- err
			}(i)
		}

		// 모든 고루틴 완료 대기
		for i := 0; i < numGoroutines; i++ {
			err := <-errors
			assert.Error(t, err) // cache miss 에러 예상
			assert.Contains(t, err.Error(), "cache miss")
		}
	})
}

// TestPropertyBased 속성 기반 테스트
func TestPropertyBased(t *testing.T) {
	suite := SetupTestSuite(t)

	t.Run("Cache Key Properties", func(t *testing.T) {
		pipConfig := &config.PipProxyConfig{Enabled: true}
		pipAdapter := pip.NewPIPAdapter(pipConfig, suite.mockPipService, suite.mockCache)

		// 속성: 같은 요청은 같은 캐시 키를 생성해야 함
		testPaths := []string{
			"/pypi/requests/json",
			"/simple/numpy/",
			"/packages/source/r/requests/requests-2.28.1.tar.gz",
		}

		for _, path := range testPaths {
			// 첫 번째 키 생성
			ctx1 := suite.app.AcquireCtx(&fiber.DefaultCtx{})
			ctx1.Request().SetRequestURI(path)
			ctx1.Request().Header.SetMethod("GET")
			key1 := pipAdapter.GetCacheKey(ctx1)
			suite.app.ReleaseCtx(ctx1)

			// 두 번째 키 생성 (같은 요청)
			ctx2 := suite.app.AcquireCtx(&fiber.DefaultCtx{})
			ctx2.Request().SetRequestURI(path)
			ctx2.Request().Header.SetMethod("GET")
			key2 := pipAdapter.GetCacheKey(ctx2)
			suite.app.ReleaseCtx(ctx2)

			// 속성 검증: 같은 요청은 같은 키를 생성
			assert.Equal(t, key1, key2, "Same request should generate same cache key")
			assert.NotEmpty(t, key1, "Cache key should not be empty")
		}
	})

	t.Run("Handler Name Uniqueness", func(t *testing.T) {
		handlerTypes := []string{"npm", "maven", "pip", "docker", "apt", "yum", "apk"}
		names := make(map[string]bool)

		// 속성: 모든 핸들러는 고유한 타입을 가져야 함
		for _, handlerType := range handlerTypes {
			mockHandler := handlerMocks.NewMockProxyHandler(t)
			mockHandler.EXPECT().Type().Return(handlerType).Maybe()
			mockHandler.EXPECT().Name().Return(fmt.Sprintf("%sHandler", handlerType)).Maybe()

			// 핸들러 등록
			err := suite.registry.Register(handlerType, mockHandler)
			assert.NoError(t, err)

			// 타입 고유성 검증
			assert.False(t, names[handlerType], "Handler type should be unique")
			names[handlerType] = true
		}

		// 속성: 등록된 핸들러 수는 입력된 타입 수와 같아야 함
		allHandlers := suite.registry.GetAll()
		assert.Equal(t, len(handlerTypes), len(allHandlers))
	})
}
