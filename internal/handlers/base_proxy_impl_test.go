package handlers

import (
	"errors"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"proxynd/cache/mocks"
	"proxynd/internal/app"
	"proxynd/logging"
)

// MockBaseProxyHandler 테스트용 Mock 핸들러
type MockBaseProxyHandler struct {
	mock.Mock
}

func (m *MockBaseProxyHandler) Type() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockBaseProxyHandler) IsEnabled() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockBaseProxyHandler) GenerateCacheKey(c *fiber.Ctx) string {
	args := m.Called(c)
	return args.String(0)
}

func (m *MockBaseProxyHandler) BuildUpstreamURL(c *fiber.Ctx) (string, error) {
	args := m.Called(c)
	return args.String(0), args.Error(1)
}

func (m *MockBaseProxyHandler) TransformRequest(c *fiber.Ctx, upstreamReq *fiber.Request) error {
	args := m.Called(c, upstreamReq)
	return args.Error(0)
}

func (m *MockBaseProxyHandler) TransformResponse(resp []byte, c *fiber.Ctx) ([]byte, error) {
	args := m.Called(resp, c)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockBaseProxyHandler) ShouldCache(c *fiber.Ctx, statusCode int) bool {
	args := m.Called(c, statusCode)
	return args.Bool(0)
}

func (m *MockBaseProxyHandler) GetCacheTTL(c *fiber.Ctx) time.Duration {
	args := m.Called(c)
	return args.Get(0).(time.Duration)
}

func (m *MockBaseProxyHandler) HandleError(err error, c *fiber.Ctx) error {
	args := m.Called(err, c)
	return args.Error(0)
}

// MockContainer 테스트용 Mock 컨테이너
type MockContainer struct {
	mock.Mock
}

func (m *MockContainer) Cache() interface{} {
	args := m.Called()
	return args.Get(0)
}

func TestNewBaseProxyHandlerImpl(t *testing.T) {
	// Mock 설정
	mockHandler := &MockBaseProxyHandler{}
	mockContainer := &MockContainer{}
	mockCache := &mocks.MockCache{}

	mockContainer.On("Cache").Return(mockCache)

	// BaseProxyHandlerImpl 생성
	impl := NewBaseProxyHandlerImpl(mockContainer, mockHandler)

	// 검증
	assert.NotNil(t, impl)
	assert.Equal(t, mockHandler, impl.handler)
	assert.NotNil(t, impl.client)
	assert.NotNil(t, impl.logger)
}

func TestBaseProxyHandlerImpl_Handle_ProxyDisabled(t *testing.T) {
	// Mock 설정
	mockHandler := &MockBaseProxyHandler{}
	mockContainer := &MockContainer{}
	mockCache := &mocks.MockCache{}

	mockContainer.On("Cache").Return(mockCache)
	mockHandler.On("Type").Return("test")
	mockHandler.On("IsEnabled").Return(false)

	impl := NewBaseProxyHandlerImpl(mockContainer, mockHandler)

	// Fiber 앱 및 컨텍스트 생성
	app := fiber.New()
	c := app.AcquireCtx(&fiber.Ctx{})
	defer app.ReleaseCtx(c)

	// 테스트 실행
	err := impl.Handle(c)

	// 검증
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "프록시가 비활성화되어 있습니다")
	mockHandler.AssertExpectations(t)
}

func TestBaseProxyHandlerImpl_Handle_CacheHit(t *testing.T) {
	// Mock 설정
	mockHandler := &MockBaseProxyHandler{}
	mockContainer := &MockContainer{}
	mockCache := &mocks.MockCache{}

	cachedData := []byte("cached response")
	cacheKey := "test:cache:key"

	mockContainer.On("Cache").Return(mockCache)
	mockHandler.On("Type").Return("test")
	mockHandler.On("IsEnabled").Return(true)
	mockHandler.On("GenerateCacheKey", mock.AnythingOfType("*fiber.Ctx")).Return(cacheKey)
	mockCache.On("Get", cacheKey).Return(cachedData, nil)

	impl := NewBaseProxyHandlerImpl(mockContainer, mockHandler)

	// Fiber 앱 및 컨텍스트 생성
	app := fiber.New()
	c := app.AcquireCtx(&fiber.Ctx{})
	defer app.ReleaseCtx(c)

	// 테스트 실행
	err := impl.Handle(c)

	// 검증
	assert.NoError(t, err)
	assert.Equal(t, "HIT", c.Get("X-Cache-Status"))
	assert.Equal(t, "test", c.Get("X-Proxy-Type"))
	mockHandler.AssertExpectations(t)
	mockCache.AssertExpectations(t)
}

func TestBaseProxyHandlerImpl_Handle_BuildUpstreamURLError(t *testing.T) {
	// Mock 설정
	mockHandler := &MockBaseProxyHandler{}
	mockContainer := &MockContainer{}
	mockCache := &mocks.MockCache{}

	cacheKey := "test:cache:key"
	cacheError := errors.New("cache miss")
	urlError := errors.New("URL build failed")

	mockContainer.On("Cache").Return(mockCache)
	mockHandler.On("Type").Return("test")
	mockHandler.On("IsEnabled").Return(true)
	mockHandler.On("GenerateCacheKey", mock.AnythingOfType("*fiber.Ctx")).Return(cacheKey)
	mockCache.On("Get", cacheKey).Return(nil, cacheError)
	mockHandler.On("BuildUpstreamURL", mock.AnythingOfType("*fiber.Ctx")).Return("", urlError)
	mockHandler.On("HandleError", urlError, mock.AnythingOfType("*fiber.Ctx")).Return(nil)

	impl := NewBaseProxyHandlerImpl(mockContainer, mockHandler)

	// Fiber 앱 및 컨텍스트 생성
	app := fiber.New()
	c := app.AcquireCtx(&fiber.Ctx{})
	defer app.ReleaseCtx(c)

	// 테스트 실행
	err := impl.Handle(c)

	// 검증
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "업스트림 URL 구성 실패")
	mockHandler.AssertExpectations(t)
	mockCache.AssertExpectations(t)
}

func TestBaseProxyHandlerImpl_Name(t *testing.T) {
	// Mock 설정
	mockHandler := &MockBaseProxyHandler{}
	mockContainer := &MockContainer{}
	mockCache := &mocks.MockCache{}

	mockContainer.On("Cache").Return(mockCache)
	mockHandler.On("Type").Return("test")

	impl := NewBaseProxyHandlerImpl(mockContainer, mockHandler)

	// 테스트 실행
	name := impl.Name()

	// 검증
	assert.Equal(t, "base-proxy-test", name)
	mockHandler.AssertExpectations(t)
}

func TestBaseProxyHandlerImpl_Type(t *testing.T) {
	// Mock 설정
	mockHandler := &MockBaseProxyHandler{}
	mockContainer := &MockContainer{}
	mockCache := &mocks.MockCache{}

	mockContainer.On("Cache").Return(mockCache)
	mockHandler.On("Type").Return("test")

	impl := NewBaseProxyHandlerImpl(mockContainer, mockHandler)

	// 테스트 실행
	proxyType := impl.Type()

	// 검증
	assert.Equal(t, "test", proxyType)
	mockHandler.AssertExpectations(t)
}

func TestBaseProxyHandlerImpl_HealthCheck_Disabled(t *testing.T) {
	// Mock 설정
	mockHandler := &MockBaseProxyHandler{}
	mockContainer := &MockContainer{}
	mockCache := &mocks.MockCache{}

	mockContainer.On("Cache").Return(mockCache)
	mockHandler.On("Type").Return("test")
	mockHandler.On("IsEnabled").Return(false)

	impl := NewBaseProxyHandlerImpl(mockContainer, mockHandler)

	// 테스트 실행
	err := impl.HealthCheck()

	// 검증
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "프록시 'test'가 비활성화되어 있습니다")
	mockHandler.AssertExpectations(t)
}

func TestBaseProxyHandlerImpl_HealthCheck_Enabled(t *testing.T) {
	// Mock 설정
	mockHandler := &MockBaseProxyHandler{}
	mockContainer := &MockContainer{}
	mockCache := &mocks.MockCache{}

	mockContainer.On("Cache").Return(mockCache)
	mockHandler.On("Type").Return("test")
	mockHandler.On("IsEnabled").Return(true)

	impl := NewBaseProxyHandlerImpl(mockContainer, mockHandler)

	// 테스트 실행
	err := impl.HealthCheck()

	// 검증
	assert.NoError(t, err)
	mockHandler.AssertExpectations(t)
}

func TestBaseProxyHandlerImpl_isHopByHopHeader(t *testing.T) {
	// Mock 설정
	mockHandler := &MockBaseProxyHandler{}
	mockContainer := &MockContainer{}
	mockCache := &mocks.MockCache{}

	mockContainer.On("Cache").Return(mockCache)

	impl := NewBaseProxyHandlerImpl(mockContainer, mockHandler)

	// 테스트 케이스들
	testCases := []struct {
		header   string
		expected bool
	}{
		{"Connection", true},
		{"Keep-Alive", true},
		{"Transfer-Encoding", true},
		{"Content-Type", false},
		{"Authorization", false},
		{"User-Agent", false},
	}

	for _, tc := range testCases {
		t.Run(tc.header, func(t *testing.T) {
			result := impl.isHopByHopHeader(tc.header)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// 벤치마크 테스트
func BenchmarkBaseProxyHandlerImpl_Handle_CacheHit(b *testing.B) {
	// Mock 설정
	mockHandler := &MockBaseProxyHandler{}
	mockContainer := &MockContainer{}
	mockCache := &mocks.MockCache{}

	cachedData := []byte("cached response")
	cacheKey := "test:cache:key"

	mockContainer.On("Cache").Return(mockCache)
	mockHandler.On("Type").Return("test")
	mockHandler.On("IsEnabled").Return(true)
	mockHandler.On("GenerateCacheKey", mock.AnythingOfType("*fiber.Ctx")).Return(cacheKey)
	mockCache.On("Get", cacheKey).Return(cachedData, nil)

	impl := NewBaseProxyHandlerImpl(mockContainer, mockHandler)

	// Fiber 앱 및 컨텍스트 생성
	app := fiber.New()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c := app.AcquireCtx(&fiber.Ctx{})
		impl.Handle(c)
		app.ReleaseCtx(c)
	}
}