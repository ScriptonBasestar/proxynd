package handlers

import (
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"proxynd/cache/mocks"
)

func TestNewProxyHandlerFactory(t *testing.T) {
	// Mock 컨테이너 설정
	mockContainer := &MockContainer{}
	mockCache := &mocks.MockCache{}
	mockContainer.On("Cache").Return(mockCache)

	// 팩토리 생성
	factory := NewProxyHandlerFactory(mockContainer)

	// 검증
	assert.NotNil(t, factory)
	assert.NotNil(t, factory.handlers)
	assert.NotNil(t, factory.instances)
	assert.NotNil(t, factory.logger)
}

func TestProxyHandlerFactory_Register(t *testing.T) {
	// Mock 컨테이너 설정
	mockContainer := &MockContainer{}
	mockCache := &mocks.MockCache{}
	mockContainer.On("Cache").Return(mockCache)

	factory := NewProxyHandlerFactory(mockContainer)

	// 핸들러 등록
	mockCreator := func() BaseProxyHandler {
		return &MockBaseProxyHandler{}
	}

	factory.Register("test", mockCreator)

	// 검증
	assert.True(t, factory.IsSupported("test"))
	assert.Contains(t, factory.GetSupportedTypes(), "test")
}

func TestProxyHandlerFactory_Unregister(t *testing.T) {
	// Mock 컨테이너 설정
	mockContainer := &MockContainer{}
	mockCache := &mocks.MockCache{}
	mockContainer.On("Cache").Return(mockCache)

	factory := NewProxyHandlerFactory(mockContainer)

	// 핸들러 등록 후 해제
	mockCreator := func() BaseProxyHandler {
		return &MockBaseProxyHandler{}
	}

	factory.Register("test", mockCreator)
	assert.True(t, factory.IsSupported("test"))

	factory.Unregister("test")
	assert.False(t, factory.IsSupported("test"))
	assert.NotContains(t, factory.GetSupportedTypes(), "test")
}

func TestProxyHandlerFactory_Create(t *testing.T) {
	// Mock 컨테이너 설정
	mockContainer := &MockContainer{}
	mockCache := &mocks.MockCache{}
	mockContainer.On("Cache").Return(mockCache)

	factory := NewProxyHandlerFactory(mockContainer)

	// Mock 핸들러 등록
	mockHandler := &MockBaseProxyHandler{}
	mockHandler.On("Type").Return("test")

	mockCreator := func() BaseProxyHandler {
		return mockHandler
	}

	factory.Register("test", mockCreator)

	// 핸들러 생성
	impl, err := factory.Create("test")

	// 검증
	assert.NoError(t, err)
	assert.NotNil(t, impl)
	assert.Equal(t, "test", impl.Type())
}

func TestProxyHandlerFactory_Create_UnsupportedType(t *testing.T) {
	// Mock 컨테이너 설정
	mockContainer := &MockContainer{}
	mockCache := &mocks.MockCache{}
	mockContainer.On("Cache").Return(mockCache)

	factory := NewProxyHandlerFactory(mockContainer)

	// 지원하지 않는 타입으로 생성 시도
	impl, err := factory.Create("unsupported")

	// 검증
	assert.Error(t, err)
	assert.Nil(t, impl)
	assert.Contains(t, err.Error(), "지원하지 않는 프록시 타입")
}

func TestProxyHandlerFactory_CreateSingleton(t *testing.T) {
	// Mock 컨테이너 설정
	mockContainer := &MockContainer{}
	mockCache := &mocks.MockCache{}
	mockContainer.On("Cache").Return(mockCache)

	factory := NewProxyHandlerFactory(mockContainer)

	// Mock 핸들러 등록
	mockHandler := &MockBaseProxyHandler{}
	mockHandler.On("Type").Return("test")

	mockCreator := func() BaseProxyHandler {
		return mockHandler
	}

	factory.Register("test", mockCreator)

	// 첫 번째 싱글톤 생성
	impl1, err1 := factory.CreateSingleton("test")
	assert.NoError(t, err1)
	assert.NotNil(t, impl1)

	// 두 번째 싱글톤 생성 (같은 인스턴스여야 함)
	impl2, err2 := factory.CreateSingleton("test")
	assert.NoError(t, err2)
	assert.NotNil(t, impl2)

	// 같은 인스턴스인지 확인
	assert.Equal(t, impl1, impl2)
}

func TestProxyHandlerFactory_GetHandlerInfo(t *testing.T) {
	// Mock 컨테이너 설정
	mockContainer := &MockContainer{}
	mockCache := &mocks.MockCache{}
	mockContainer.On("Cache").Return(mockCache)

	factory := NewProxyHandlerFactory(mockContainer)

	// Mock 핸들러 등록
	mockHandler := &MockBaseProxyHandler{}
	mockHandler.On("Type").Return("test")
	mockHandler.On("IsEnabled").Return(true)

	mockCreator := func() BaseProxyHandler {
		return mockHandler
	}

	factory.Register("test", mockCreator)

	// 핸들러 정보 조회
	info, err := factory.GetHandlerInfo("test")

	// 검증
	assert.NoError(t, err)
	assert.NotNil(t, info)
	assert.Equal(t, "test", info["type"])
	assert.Equal(t, true, info["enabled"])
	assert.Equal(t, false, info["supports_cache"]) // MockBaseProxyHandler는 CacheableProxyHandler가 아님
	assert.Equal(t, false, info["supports_auth"])
	assert.Equal(t, false, info["supports_metrics"])
	assert.Equal(t, false, info["supports_health_check"])
}

func TestProxyHandlerFactory_GetHandlerInfo_UnsupportedType(t *testing.T) {
	// Mock 컨테이너 설정
	mockContainer := &MockContainer{}
	mockCache := &mocks.MockCache{}
	mockContainer.On("Cache").Return(mockCache)

	factory := NewProxyHandlerFactory(mockContainer)

	// 지원하지 않는 타입으로 정보 조회
	info, err := factory.GetHandlerInfo("unsupported")

	// 검증
	assert.Error(t, err)
	assert.Nil(t, info)
	assert.Contains(t, err.Error(), "지원하지 않는 프록시 타입")
}

func TestProxyHandlerFactory_HealthCheck(t *testing.T) {
	// Mock 컨테이너 설정
	mockContainer := &MockContainer{}
	mockCache := &mocks.MockCache{}
	mockContainer.On("Cache").Return(mockCache)

	factory := NewProxyHandlerFactory(mockContainer)

	// Mock 핸들러 등록 (활성화된 핸들러)
	mockHandler1 := &MockBaseProxyHandler{}
	mockHandler1.On("Type").Return("test1")
	mockHandler1.On("IsEnabled").Return(true)

	// Mock 핸들러 등록 (비활성화된 핸들러)
	mockHandler2 := &MockBaseProxyHandler{}
	mockHandler2.On("Type").Return("test2")
	mockHandler2.On("IsEnabled").Return(false)

	factory.Register("test1", func() BaseProxyHandler { return mockHandler1 })
	factory.Register("test2", func() BaseProxyHandler { return mockHandler2 })

	// 헬스체크 실행
	results := factory.HealthCheck()

	// 검증
	assert.NotNil(t, results)
	// test1은 활성화되어 있지만 Healthable 인터페이스를 구현하지 않음
	// test2는 비활성화되어 있어서 헬스체크 대상에서 제외
	assert.Empty(t, results) // MockBaseProxyHandler는 Healthable을 구현하지 않음
}

func TestProxyHandlerFactory_GetStatistics(t *testing.T) {
	// Mock 컨테이너 설정
	mockContainer := &MockContainer{}
	mockCache := &mocks.MockCache{}
	mockContainer.On("Cache").Return(mockCache)

	factory := NewProxyHandlerFactory(mockContainer)

	// Mock 핸들러 등록
	mockHandler := &MockBaseProxyHandler{}
	mockHandler.On("Type").Return("test")
	mockHandler.On("IsEnabled").Return(true)

	factory.Register("test", func() BaseProxyHandler { return mockHandler })

	// 통계 정보 조회
	stats := factory.GetStatistics()

	// 검증
	assert.NotNil(t, stats)
	assert.Equal(t, 1, stats["total_registered_types"])
	assert.Equal(t, 0, stats["active_instances"]) // 아직 싱글톤 인스턴스가 생성되지 않음
	assert.Contains(t, stats["supported_types"], "test")

	typeDetails := stats["type_details"].(map[string]interface{})
	testDetails := typeDetails["test"].(map[string]interface{})
	assert.Equal(t, true, testDetails["enabled"])
	assert.Equal(t, false, testDetails["has_instance"])
}

func TestProxyHandlerFactory_Shutdown(t *testing.T) {
	// Mock 컨테이너 설정
	mockContainer := &MockContainer{}
	mockCache := &mocks.MockCache{}
	mockContainer.On("Cache").Return(mockCache)

	factory := NewProxyHandlerFactory(mockContainer)

	// Mock 핸들러 등록 및 싱글톤 생성
	mockHandler := &MockBaseProxyHandler{}
	mockHandler.On("Type").Return("test")

	factory.Register("test", func() BaseProxyHandler { return mockHandler })
	_, err := factory.CreateSingleton("test")
	assert.NoError(t, err)

	// 인스턴스가 생성되었는지 확인
	stats := factory.GetStatistics()
	assert.Equal(t, 1, stats["active_instances"])

	// 팩토리 종료
	factory.Shutdown()

	// 인스턴스가 정리되었는지 확인
	statsAfterShutdown := factory.GetStatistics()
	assert.Equal(t, 0, statsAfterShutdown["active_instances"])
}

// 확장된 Mock 핸들러 (Healthable 인터페이스 구현)
type MockHealthableProxyHandler struct {
	MockBaseProxyHandler
}

func (m *MockHealthableProxyHandler) HealthCheck() error {
	args := m.Called()
	return args.Error(0)
}

func TestProxyHandlerFactory_HealthCheck_WithHealthableHandler(t *testing.T) {
	// Mock 컨테이너 설정
	mockContainer := &MockContainer{}
	mockCache := &mocks.MockCache{}
	mockContainer.On("Cache").Return(mockCache)

	factory := NewProxyHandlerFactory(mockContainer)

	// Healthable Mock 핸들러 등록
	mockHandler := &MockHealthableProxyHandler{}
	mockHandler.On("Type").Return("healthable")
	mockHandler.On("IsEnabled").Return(true)
	mockHandler.On("HealthCheck").Return(nil)

	factory.Register("healthable", func() BaseProxyHandler { return mockHandler })

	// 헬스체크 실행
	results := factory.HealthCheck()

	// 검증
	assert.NotNil(t, results)
	assert.Contains(t, results, "healthable")
	assert.NoError(t, results["healthable"])
	mockHandler.AssertExpectations(t)
}

// 벤치마크 테스트
func BenchmarkProxyHandlerFactory_Create(b *testing.B) {
	// Mock 컨테이너 설정
	mockContainer := &MockContainer{}
	mockCache := &mocks.MockCache{}
	mockContainer.On("Cache").Return(mockCache)

	factory := NewProxyHandlerFactory(mockContainer)

	// Mock 핸들러 등록
	mockHandler := &MockBaseProxyHandler{}
	mockHandler.On("Type").Return("test")

	factory.Register("test", func() BaseProxyHandler { return mockHandler })

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := factory.Create("test")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkProxyHandlerFactory_CreateSingleton(b *testing.B) {
	// Mock 컨테이너 설정
	mockContainer := &MockContainer{}
	mockCache := &mocks.MockCache{}
	mockContainer.On("Cache").Return(mockCache)

	factory := NewProxyHandlerFactory(mockContainer)

	// Mock 핸들러 등록
	mockHandler := &MockBaseProxyHandler{}
	mockHandler.On("Type").Return("test")

	factory.Register("test", func() BaseProxyHandler { return mockHandler })

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := factory.CreateSingleton("test")
		if err != nil {
			b.Fatal(err)
		}
	}
}