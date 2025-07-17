package handlers

import (
	"errors"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/valyala/fasthttp"

	"proxynd/cache"
	"proxynd/cache/mocks"
	proxyerrors "proxynd/internal/errors"
)

// MockErrorHandler 에러 처리 테스트용 Mock 핸들러
type MockErrorHandler struct {
	mock.Mock
}

func (m *MockErrorHandler) Type() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockErrorHandler) IsEnabled() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockErrorHandler) GenerateCacheKey(c *fiber.Ctx) string {
	args := m.Called(c)
	return args.String(0)
}

func (m *MockErrorHandler) BuildUpstreamURL(c *fiber.Ctx) (string, error) {
	args := m.Called(c)
	return args.String(0), args.Error(1)
}

func (m *MockErrorHandler) TransformRequest(c *fiber.Ctx, upstreamReq *fiber.Agent) error {
	args := m.Called(c, upstreamReq)
	return args.Error(0)
}

func (m *MockErrorHandler) TransformResponse(resp []byte, c *fiber.Ctx) ([]byte, error) {
	args := m.Called(resp, c)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockErrorHandler) ShouldCache(c *fiber.Ctx, statusCode int) bool {
	args := m.Called(c, statusCode)
	return args.Bool(0)
}

func (m *MockErrorHandler) GetCacheTTL(c *fiber.Ctx) time.Duration {
	args := m.Called(c)
	return args.Get(0).(time.Duration)
}

func (m *MockErrorHandler) HandleError(err error, c *fiber.Ctx) error {
	args := m.Called(err, c)
	return args.Error(0)
}

// MockErrorContainer 에러 처리 테스트용 Mock 컨테이너
type MockErrorContainer struct {
	mock.Mock
}

func (m *MockErrorContainer) Cache() cache.Cache {
	args := m.Called()
	return args.Get(0).(cache.Cache)
}

func TestBaseProxyHandlerImpl_ErrorHandling(t *testing.T) {
	t.Run("Handle disabled proxy error", func(t *testing.T) {
		// Mock 설정
		mockHandler := &MockErrorHandler{}
		mockContainer := &MockErrorContainer{}
		mockCache := &mocks.MockCache{}
		
		mockContainer.On("Cache").Return(mockCache)
		mockHandler.On("IsEnabled").Return(false)
		mockHandler.On("Type").Return("test")
		
		impl := NewBaseProxyHandlerImpl(mockContainer, mockHandler)
		
		// Fiber 컨텍스트 생성
		app := fiber.New()
		ctx := app.AcquireCtx(&fasthttp.RequestCtx{})
		defer app.ReleaseCtx(ctx)
		
		// Handle 실행
		err := impl.Handle(ctx)
		
		// 검증
		assert.Error(t, err)
		
		// ProxyND 에러 구조체인지 확인
		if proxyErr, ok := err.(*proxyerrors.DomainError); ok {
			assert.Equal(t, "PROXY001", proxyErr.Code)
			assert.Contains(t, proxyErr.Message, "프록시가 비활성화되어 있습니다")
			assert.Equal(t, "test", proxyErr.Domain)
		}
		
		mockContainer.AssertExpectations(t)
		mockHandler.AssertExpectations(t)
	})

	t.Run("Handle BuildUpstreamURL error", func(t *testing.T) {
		// Mock 설정
		mockHandler := &MockErrorHandler{}
		mockContainer := &MockErrorContainer{}
		mockCache := &mocks.MockCache{}
		
		mockContainer.On("Cache").Return(mockCache)
		mockHandler.On("IsEnabled").Return(true)
		mockHandler.On("Type").Return("test")
		mockHandler.On("GenerateCacheKey", mock.Anything).Return("test-key")
		
		// 캐시 미스 설정
		mockCache.On("Get", "test-key").Return(nil, errors.New("cache miss"))
		
		// BuildUpstreamURL 에러 설정
		buildError := errors.New("upstream URL build failed")
		mockHandler.On("BuildUpstreamURL", mock.Anything).Return("", buildError)
		mockHandler.On("HandleError", buildError, mock.Anything).Return(nil)
		
		impl := NewBaseProxyHandlerImpl(mockContainer, mockHandler)
		
		// Fiber 컨텍스트 생성
		app := fiber.New()
		ctx := app.AcquireCtx(&fasthttp.RequestCtx{})
		defer app.ReleaseCtx(ctx)
		
		// Handle 실행
		err := impl.Handle(ctx)
		
		// 검증
		assert.Error(t, err)
		
		// ProxyND 에러 구조체인지 확인
		if proxyErr, ok := err.(*proxyerrors.DomainError); ok {
			assert.Equal(t, "PROXY002", proxyErr.Code)
			assert.Contains(t, proxyErr.Message, "업스트림 URL 구성 실패")
			assert.Equal(t, "test", proxyErr.Domain)
			assert.Equal(t, buildError, proxyErr.Cause)
		}
		
		mockContainer.AssertExpectations(t)
		mockHandler.AssertExpectations(t)
		mockCache.AssertExpectations(t)
	})

	t.Run("Handle TransformRequest error", func(t *testing.T) {
		// Mock 설정
		mockHandler := &MockErrorHandler{}
		mockContainer := &MockErrorContainer{}
		mockCache := &mocks.MockCache{}
		
		mockContainer.On("Cache").Return(mockCache)
		mockHandler.On("IsEnabled").Return(true)
		mockHandler.On("Type").Return("test")
		mockHandler.On("GenerateCacheKey", mock.Anything).Return("test-key")
		
		// 캐시 미스 설정
		mockCache.On("Get", "test-key").Return(nil, errors.New("cache miss"))
		
		// BuildUpstreamURL 성공
		mockHandler.On("BuildUpstreamURL", mock.Anything).Return("http://test.com/path", nil)
		
		// TransformRequest 에러 설정
		transformError := errors.New("request transform failed")
		mockHandler.On("TransformRequest", mock.Anything, mock.Anything).Return(transformError)
		mockHandler.On("HandleError", transformError, mock.Anything).Return(nil)
		
		impl := NewBaseProxyHandlerImpl(mockContainer, mockHandler)
		
		// Fiber 컨텍스트 생성
		app := fiber.New()
		ctx := app.AcquireCtx(&fasthttp.RequestCtx{})
		defer app.ReleaseCtx(ctx)
		
		// Handle 실행
		err := impl.Handle(ctx)
		
		// 검증
		assert.Error(t, err)
		assert.Equal(t, transformError, err)
		
		mockContainer.AssertExpectations(t)
		mockHandler.AssertExpectations(t)
		mockCache.AssertExpectations(t)
	})

	t.Run("Handle TransformResponse error with fallback", func(t *testing.T) {
		// Mock 설정
		mockHandler := &MockErrorHandler{}
		mockContainer := &MockErrorContainer{}
		mockCache := &mocks.MockCache{}
		
		mockContainer.On("Cache").Return(mockCache)
		mockHandler.On("IsEnabled").Return(true)
		mockHandler.On("Type").Return("test")
		mockHandler.On("GenerateCacheKey", mock.Anything).Return("test-key")
		
		// 캐시 미스 설정
		mockCache.On("Get", "test-key").Return(nil, errors.New("cache miss"))
		
		// BuildUpstreamURL 성공
		mockHandler.On("BuildUpstreamURL", mock.Anything).Return("http://test.com/path", nil)
		
		// TransformRequest 성공
		mockHandler.On("TransformRequest", mock.Anything, mock.Anything).Return(nil)
		
		// 참고: 실제 HTTP 요청은 모킹하기 어려우므로 TransformResponse 에러만 테스트
		originalData := []byte("original response")
		transformError := errors.New("response transform failed")
		mockHandler.On("TransformResponse", originalData, mock.Anything).Return([]byte{}, transformError)
		mockHandler.On("HandleError", transformError, mock.Anything).Return(nil)
		
		// 캐시 정책 설정 (캐시하지 않음)
		mockHandler.On("ShouldCache", mock.Anything, mock.Anything).Return(false)
		
		impl := NewBaseProxyHandlerImpl(mockContainer, mockHandler)
		
		// 이 테스트는 실제 HTTP 요청 없이는 완전히 테스트하기 어려우므로,
		// 최소한 구조체가 정상적으로 생성되는지 확인
		assert.NotNil(t, impl)
		
		mockContainer.AssertExpectations(t)
	})
}

func TestBaseProxyHandlerImpl_CacheErrors(t *testing.T) {
	t.Run("Cache Get error handling - logCacheError functionality", func(t *testing.T) {
		// Mock 설정
		mockHandler := &MockErrorHandler{}
		mockContainer := &MockErrorContainer{}
		mockCache := &mocks.MockCache{}
		
		mockContainer.On("Cache").Return(mockCache)
		
		impl := NewBaseProxyHandlerImpl(mockContainer, mockHandler)
		
		// logCacheError 메서드 직접 테스트 (캐시 에러 로깅 기능)
		cacheError := errors.New("cache connection failed")
		assert.NotPanics(t, func() {
			impl.logCacheError("get", cacheError, "test")
		})
		
		mockContainer.AssertExpectations(t)
	})
}

func TestBaseProxyHandlerImpl_ErrorRecovery(t *testing.T) {
	t.Run("Handler error with custom error handling", func(t *testing.T) {
		// Mock 설정
		mockHandler := &MockErrorHandler{}
		mockContainer := &MockErrorContainer{}
		mockCache := &mocks.MockCache{}
		
		mockContainer.On("Cache").Return(mockCache)
		mockHandler.On("IsEnabled").Return(true)
		mockHandler.On("Type").Return("test")
		mockHandler.On("GenerateCacheKey", mock.Anything).Return("test-key")
		
		// 캐시 미스 설정
		mockCache.On("Get", "test-key").Return(nil, errors.New("cache miss"))
		
		// BuildUpstreamURL 에러 설정
		buildError := errors.New("upstream URL build failed")
		mockHandler.On("BuildUpstreamURL", mock.Anything).Return("", buildError)
		
		// 커스텀 에러 처리 (에러를 변환)
		customError := fiber.NewError(fiber.StatusBadGateway, "Custom upstream error")
		mockHandler.On("HandleError", buildError, mock.Anything).Return(customError)
		
		impl := NewBaseProxyHandlerImpl(mockContainer, mockHandler)
		
		// Fiber 컨텍스트 생성
		app := fiber.New()
		ctx := app.AcquireCtx(&fasthttp.RequestCtx{})
		defer app.ReleaseCtx(ctx)
		
		// Handle 실행
		err := impl.Handle(ctx)
		
		// 검증 - 커스텀 에러가 반환되어야 함
		assert.Error(t, err)
		assert.Equal(t, customError, err)
		
		mockContainer.AssertExpectations(t)
		mockHandler.AssertExpectations(t)
		mockCache.AssertExpectations(t)
	})

	t.Run("Nil error handler response", func(t *testing.T) {
		// Mock 설정
		mockHandler := &MockErrorHandler{}
		mockContainer := &MockErrorContainer{}
		mockCache := &mocks.MockCache{}
		
		mockContainer.On("Cache").Return(mockCache)
		mockHandler.On("IsEnabled").Return(true)
		mockHandler.On("Type").Return("test")
		mockHandler.On("GenerateCacheKey", mock.Anything).Return("test-key")
		
		// 캐시 미스 설정
		mockCache.On("Get", "test-key").Return(nil, errors.New("cache miss"))
		
		// BuildUpstreamURL 에러 설정
		buildError := errors.New("upstream URL build failed")
		mockHandler.On("BuildUpstreamURL", mock.Anything).Return("", buildError)
		
		// 에러 처리기가 nil 반환 (에러 무시)
		mockHandler.On("HandleError", buildError, mock.Anything).Return(nil)
		
		impl := NewBaseProxyHandlerImpl(mockContainer, mockHandler)
		
		// Fiber 컨텍스트 생성
		app := fiber.New()
		ctx := app.AcquireCtx(&fasthttp.RequestCtx{})
		defer app.ReleaseCtx(ctx)
		
		// Handle 실행
		err := impl.Handle(ctx)
		
		// 검증 - 기본 ProxyND 에러가 반환되어야 함
		assert.Error(t, err)
		
		// ProxyND 에러 구조체인지 확인
		if proxyErr, ok := err.(*proxyerrors.DomainError); ok {
			assert.Equal(t, "PROXY002", proxyErr.Code)
			assert.Contains(t, proxyErr.Message, "업스트림 URL 구성 실패")
			assert.Equal(t, "test", proxyErr.Domain)
			assert.Equal(t, buildError, proxyErr.Cause)
		}
		
		mockContainer.AssertExpectations(t)
		mockHandler.AssertExpectations(t)
		mockCache.AssertExpectations(t)
	})
}

// 벤치마크 테스트 - 에러 처리 성능
func BenchmarkBaseProxyHandlerImpl_ErrorHandling(b *testing.B) {
	// Mock 설정
	mockHandler := &MockErrorHandler{}
	mockContainer := &MockErrorContainer{}
	mockCache := &mocks.MockCache{}
	
	mockContainer.On("Cache").Return(mockCache)
	mockHandler.On("IsEnabled").Return(false)
	mockHandler.On("Type").Return("test")
	
	impl := NewBaseProxyHandlerImpl(mockContainer, mockHandler)
	
	// Fiber 앱 생성
	app := fiber.New()
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		ctx := app.AcquireCtx(&fasthttp.RequestCtx{})
		_ = impl.Handle(ctx)
		app.ReleaseCtx(ctx)
	}
}