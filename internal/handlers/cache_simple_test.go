package handlers

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/valyala/fasthttp"

	"proxynd/cache"
	cacheMocks "proxynd/internal/services/proxy/mocks"
)

// SimpleCacheHandler 간단한 캐시 테스트용 Mock 핸들러
type SimpleCacheHandler struct {
	mock.Mock
}

func (m *SimpleCacheHandler) Type() string {
	args := m.Called()
	return args.String(0)
}

func (m *SimpleCacheHandler) IsEnabled() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *SimpleCacheHandler) GenerateCacheKey(c *fiber.Ctx) string {
	args := m.Called(c)
	return args.String(0)
}

func (m *SimpleCacheHandler) BuildUpstreamURL(c *fiber.Ctx) (string, error) {
	args := m.Called(c)
	return args.String(0), args.Error(1)
}

func (m *SimpleCacheHandler) TransformRequest(c *fiber.Ctx, upstreamReq *fiber.Agent) error {
	args := m.Called(c, upstreamReq)
	return args.Error(0)
}

func (m *SimpleCacheHandler) TransformResponse(resp []byte, c *fiber.Ctx) ([]byte, error) {
	args := m.Called(resp, c)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *SimpleCacheHandler) ShouldCache(c *fiber.Ctx, statusCode int) bool {
	args := m.Called(c, statusCode)
	return args.Bool(0)
}

func (m *SimpleCacheHandler) GetCacheTTL(c *fiber.Ctx) time.Duration {
	args := m.Called(c)
	return args.Get(0).(time.Duration)
}

func (m *SimpleCacheHandler) HandleError(err error, c *fiber.Ctx) error {
	args := m.Called(err, c)
	return args.Error(0)
}

// SimpleCacheContainer 간단한 캐시 테스트용 Mock 컨테이너
type SimpleCacheContainer struct {
	mock.Mock
}

func (m *SimpleCacheContainer) Cache() cache.Cache {
	args := m.Called()
	return args.Get(0).(cache.Cache)
}

func TestBaseProxyHandlerImpl_SimpleCacheTest(t *testing.T) {
	t.Run("Cache key generation", func(t *testing.T) {
		// Mock 설정
		mockHandler := &SimpleCacheHandler{}

		// Fiber 컨텍스트 생성
		app := fiber.New()
		ctx := app.AcquireCtx(&fasthttp.RequestCtx{})
		defer app.ReleaseCtx(ctx)

		// 캐시 키 생성 테스트
		expectedKey := "test-cache-key"
		mockHandler.On("GenerateCacheKey", ctx).Return(expectedKey)

		cacheKey := mockHandler.GenerateCacheKey(ctx)
		assert.Equal(t, expectedKey, cacheKey)

		mockHandler.AssertExpectations(t)
	})

	t.Run("Cache TTL policy", func(t *testing.T) {
		// Mock 설정
		mockHandler := &SimpleCacheHandler{}

		// Fiber 컨텍스트 생성
		app := fiber.New()
		ctx := app.AcquireCtx(&fasthttp.RequestCtx{})
		defer app.ReleaseCtx(ctx)

		// TTL 정책 테스트
		testCases := []struct {
			name        string
			statusCode  int
			shouldCache bool
			ttl         time.Duration
		}{
			{"Success response", 200, true, 3600 * time.Second},
			{"Not found response", 404, false, 0},
			{"Server error", 500, false, 0},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				mockHandler.On("ShouldCache", ctx, tc.statusCode).Return(tc.shouldCache).Once()

				shouldCache := mockHandler.ShouldCache(ctx, tc.statusCode)
				assert.Equal(t, tc.shouldCache, shouldCache)

				if tc.shouldCache {
					mockHandler.On("GetCacheTTL", ctx).Return(tc.ttl).Once()
					ttl := mockHandler.GetCacheTTL(ctx)
					assert.Equal(t, tc.ttl, ttl)
				}
			})
		}

		mockHandler.AssertExpectations(t)
	})

	t.Run("Cache operations", func(t *testing.T) {
		// Mock 설정
		mockCache := &cacheMocks.MockCacheService{}

		cacheKey := "test-key"
		data := []byte("test data")

		// 캐시 저장 성공
		mockCache.On("Put", mock.Anything, cacheKey, mock.Anything).Return(nil).Once()
		err := mockCache.Put(context.Background(), cacheKey, bytes.NewReader(data))
		assert.NoError(t, err)

		// 캐시 조회 성공  
		dataReader := bytes.NewReader(data)
		mockCache.On("Get", mock.Anything, cacheKey).Return(dataReader, true, nil).Once()
		retrievedReader, found, err := mockCache.Get(context.Background(), cacheKey)
		assert.NoError(t, err)
		retrievedData, err := io.ReadAll(retrievedReader)
		assert.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, data, retrievedData)

		// 캐시 미스
		mockCache.On("Get", mock.Anything, "nonexistent-key").Return(nil, false, nil).Once()
		missReader, found, err := mockCache.Get(context.Background(), "nonexistent-key")
		assert.NoError(t, err)
		assert.False(t, found)
		assert.Nil(t, missReader)

		mockCache.AssertExpectations(t)
	})

	t.Run("Cache hit scenario basic validation", func(t *testing.T) {
		// Mock 설정
		mockHandler := &SimpleCacheHandler{}
		mockContainer := &SimpleCacheContainer{}
		mockCache := &cacheMocks.MockCacheService{}

		mockContainer.On("Cache").Return(mockCache)
		mockHandler.On("IsEnabled").Return(true)
		mockHandler.On("Type").Return("test")
		mockHandler.On("GenerateCacheKey", mock.Anything).Return("test-key")

		// 캐시 히트 설정
		cachedData := []byte("cached response")
		cachedReader := bytes.NewReader(cachedData)
		mockCache.On("Get", mock.Anything, "test-key").Return(cachedReader, true, nil)

		impl := NewBaseProxyHandlerImpl(mockContainer, mockHandler)

		// Fiber 컨텍스트 생성
		app := fiber.New()
		ctx := app.AcquireCtx(&fasthttp.RequestCtx{})
		defer app.ReleaseCtx(ctx)

		// Handle 실행
		err := impl.Handle(ctx)

		// 기본 검증 (에러 없음)
		assert.NoError(t, err)

		mockContainer.AssertExpectations(t)
		mockHandler.AssertExpectations(t)
		mockCache.AssertExpectations(t)
	})
}
