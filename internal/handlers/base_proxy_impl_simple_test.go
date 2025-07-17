package handlers

import (
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"proxynd/cache"
	"proxynd/cache/mocks"
)

// MockSimpleHandler 간단한 Mock 핸들러
type MockSimpleHandler struct {
	mock.Mock
}

func (m *MockSimpleHandler) Type() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockSimpleHandler) IsEnabled() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockSimpleHandler) GenerateCacheKey(c *fiber.Ctx) string {
	args := m.Called(c)
	return args.String(0)
}

func (m *MockSimpleHandler) BuildUpstreamURL(c *fiber.Ctx) (string, error) {
	args := m.Called(c)
	return args.String(0), args.Error(1)
}

func (m *MockSimpleHandler) TransformRequest(c *fiber.Ctx, upstreamReq *fiber.Agent) error {
	args := m.Called(c, upstreamReq)
	return args.Error(0)
}

func (m *MockSimpleHandler) TransformResponse(resp []byte, c *fiber.Ctx) ([]byte, error) {
	args := m.Called(resp, c)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockSimpleHandler) ShouldCache(c *fiber.Ctx, statusCode int) bool {
	args := m.Called(c, statusCode)
	return args.Bool(0)
}

func (m *MockSimpleHandler) GetCacheTTL(c *fiber.Ctx) time.Duration {
	args := m.Called(c)
	return args.Get(0).(time.Duration)
}

func (m *MockSimpleHandler) HandleError(err error, c *fiber.Ctx) error {
	args := m.Called(err, c)
	return args.Error(0)
}

// MockSimpleContainer 간단한 Mock 컨테이너
type MockSimpleContainer struct {
	mock.Mock
}

func (m *MockSimpleContainer) Cache() cache.Cache {
	args := m.Called()
	return args.Get(0).(cache.Cache)
}

func TestBaseProxyHandlerImpl_Constructor(t *testing.T) {
	t.Run("NewBaseProxyHandlerImpl with CacheProvider", func(t *testing.T) {
		// Mock 설정
		mockHandler := &MockSimpleHandler{}
		mockContainer := &MockSimpleContainer{}
		mockCache := &mocks.MockCache{}
		
		mockContainer.On("Cache").Return(mockCache)
		
		// BaseProxyHandlerImpl 생성
		impl := NewBaseProxyHandlerImpl(mockContainer, mockHandler)
		
		// 검증
		assert.NotNil(t, impl)
		assert.Equal(t, mockContainer, impl.container)
		assert.Equal(t, mockHandler, impl.handler)
		assert.Equal(t, mockCache, impl.cache)
		assert.NotNil(t, impl.logger)
		assert.NotNil(t, impl.client)
		
		mockContainer.AssertExpectations(t)
	})

	t.Run("NewBaseProxyHandlerImpl without CacheProvider", func(t *testing.T) {
		// Cache를 제공하지 않는 간단한 컨테이너
		mockHandler := &MockSimpleHandler{}
		simpleContainer := struct{}{}
		
		// BaseProxyHandlerImpl 생성 - 캐시 없이
		impl := NewBaseProxyHandlerImpl(simpleContainer, mockHandler)
		
		// 검증
		assert.NotNil(t, impl)
		assert.Equal(t, simpleContainer, impl.container)
		assert.Equal(t, mockHandler, impl.handler)
		assert.Nil(t, impl.cache) // Cache가 없는 경우
		assert.NotNil(t, impl.logger)
		assert.NotNil(t, impl.client)
	})
}

func TestBaseProxyHandlerImpl_Name(t *testing.T) {
	t.Run("Name returns correct format", func(t *testing.T) {
		// Mock 설정
		mockHandler := &MockSimpleHandler{}
		mockHandler.On("Type").Return("test")
		
		impl := &BaseProxyHandlerImpl{
			handler: mockHandler,
		}
		
		// Name 메서드 테스트
		name := impl.Name()
		assert.Equal(t, "base-proxy-test", name)
		
		mockHandler.AssertExpectations(t)
	})
}

func TestBaseProxyHandlerImpl_Type(t *testing.T) {
	t.Run("Type returns handler type", func(t *testing.T) {
		// Mock 설정
		mockHandler := &MockSimpleHandler{}
		mockHandler.On("Type").Return("apt")
		
		impl := &BaseProxyHandlerImpl{
			handler: mockHandler,
		}
		
		// Type 메서드 테스트
		proxyType := impl.Type()
		assert.Equal(t, "apt", proxyType)
		
		mockHandler.AssertExpectations(t)
	})
}

func TestBaseProxyHandlerImpl_HealthCheck(t *testing.T) {
	t.Run("HealthCheck with enabled handler", func(t *testing.T) {
		// Mock 설정
		mockHandler := &MockSimpleHandler{}
		mockHandler.On("IsEnabled").Return(true)
		
		impl := &BaseProxyHandlerImpl{
			handler: mockHandler,
		}
		
		// HealthCheck 테스트
		err := impl.HealthCheck()
		assert.NoError(t, err)
		
		mockHandler.AssertExpectations(t)
	})

	t.Run("HealthCheck with disabled handler", func(t *testing.T) {
		// Mock 설정
		mockHandler := &MockSimpleHandler{}
		mockHandler.On("IsEnabled").Return(false)
		mockHandler.On("Type").Return("test")
		
		impl := &BaseProxyHandlerImpl{
			handler: mockHandler,
		}
		
		// HealthCheck 테스트
		err := impl.HealthCheck()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "비활성화되어 있습니다")
		
		mockHandler.AssertExpectations(t)
	})
}

func TestBaseProxyHandlerImpl_isHopByHopHeader(t *testing.T) {
	t.Run("isHopByHopHeader detection", func(t *testing.T) {
		impl := &BaseProxyHandlerImpl{}
		
		// hop-by-hop 헤더들
		hopByHopHeaders := []string{
			"Connection",
			"Keep-Alive", 
			"Proxy-Authenticate",
			"Proxy-Authorization",
			"TE",
			"Trailers",
			"Transfer-Encoding",
			"Upgrade",
		}
		
		for _, header := range hopByHopHeaders {
			assert.True(t, impl.isHopByHopHeader(header), "Expected %s to be hop-by-hop header", header)
		}
		
		// 일반 헤더들
		normalHeaders := []string{
			"Content-Type",
			"Authorization",
			"User-Agent",
			"Content-Length",
			"Accept",
			"Host",
		}
		
		for _, header := range normalHeaders {
			assert.False(t, impl.isHopByHopHeader(header), "Expected %s to NOT be hop-by-hop header", header)
		}
	})

	t.Run("isHopByHopHeader case sensitivity", func(t *testing.T) {
		impl := &BaseProxyHandlerImpl{}
		
		// 대소문자 구분 확인
		assert.True(t, impl.isHopByHopHeader("Connection"))
		assert.False(t, impl.isHopByHopHeader("connection")) // 소문자는 false
		assert.False(t, impl.isHopByHopHeader("CONNECTION")) // 대문자는 false
	})
}

func TestBaseProxyHandlerImpl_Logging(t *testing.T) {
	t.Run("logCacheError", func(t *testing.T) {
		// Mock 설정
		mockHandler := &MockSimpleHandler{}
		mockContainer := &MockSimpleContainer{}
		mockCache := &mocks.MockCache{}
		
		mockContainer.On("Cache").Return(mockCache)
		
		// NewBaseProxyHandlerImpl을 통해 올바르게 초기화된 인스턴스 생성
		impl := NewBaseProxyHandlerImpl(mockContainer, mockHandler)
		
		// logCacheError는 void 함수이므로 에러가 발생하지 않는지만 확인
		assert.NotPanics(t, func() {
			impl.logCacheError("get", assert.AnError, "test")
		})
		
		mockContainer.AssertExpectations(t)
	})
}

// 벤치마크 테스트
func BenchmarkBaseProxyHandlerImpl_isHopByHopHeader(b *testing.B) {
	impl := &BaseProxyHandlerImpl{}
	
	b.Run("hop-by-hop header", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = impl.isHopByHopHeader("Connection")
		}
	})
	
	b.Run("normal header", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = impl.isHopByHopHeader("Content-Type")
		}
	})
}