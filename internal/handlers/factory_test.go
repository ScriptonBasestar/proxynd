package handlers

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"proxynd/internal/app"
)

func TestHandlerFactory(t *testing.T) {
	// 테스트용 컨테이너 생성
	cfg := &app.Config{
		ConfigDir: "/tmp/test-config",
		Port:      "8080",
	}
	container := app.NewContainer(cfg)
	defer container.Close()

	t.Run("Factory creation", func(t *testing.T) {
		factory := NewHandlerFactory(container)

		assert.NotNil(t, factory)
		assert.NotNil(t, factory.container)
		assert.NotNil(t, factory.creators)
		assert.NotNil(t, factory.logger)
		assert.Equal(t, 0, len(factory.GetSupportedTypes()))
	})

	t.Run("Handler registration", func(t *testing.T) {
		factory := NewHandlerFactory(container)

		// 핸들러 등록
		factory.Register("test", func(container *app.Container) Handler {
			return NewTestHandler(container, "test-handler", "test")
		})

		// 지원되는 타입 확인
		supportedTypes := factory.GetSupportedTypes()
		assert.Equal(t, 1, len(supportedTypes))
		assert.Contains(t, supportedTypes, "test")
		assert.True(t, factory.IsSupported("test"))
		assert.False(t, factory.IsSupported("nonexistent"))
	})

	t.Run("Handler creation", func(t *testing.T) {
		factory := NewHandlerFactory(container)

		// 핸들러 등록
		factory.Register("test", func(container *app.Container) Handler {
			return NewTestHandler(container, "test-handler", "test")
		})

		// 핸들러 생성
		handler, err := factory.Create("test")
		require.NoError(t, err)
		assert.NotNil(t, handler)
		assert.Equal(t, "test-handler", handler.Name())
		assert.Equal(t, "test", handler.Type())

		// 존재하지 않는 타입 생성 시도
		_, err = factory.Create("nonexistent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unknown handler type")
	})

	t.Run("Handler unregistration", func(t *testing.T) {
		factory := NewHandlerFactory(container)

		// 핸들러 등록
		factory.Register("test", func(container *app.Container) Handler {
			return NewTestHandler(container, "test-handler", "test")
		})

		assert.True(t, factory.IsSupported("test"))

		// 핸들러 등록 해제
		result := factory.Unregister("test")
		assert.True(t, result)
		assert.False(t, factory.IsSupported("test"))

		// 이미 등록 해제된 핸들러 다시 해제 시도
		result = factory.Unregister("test")
		assert.False(t, result)
	})

	t.Run("Factory clear", func(t *testing.T) {
		factory := NewHandlerFactory(container)

		// 여러 핸들러 등록
		factory.Register("test1", func(container *app.Container) Handler {
			return NewTestHandler(container, "test1-handler", "test1")
		})
		factory.Register("test2", func(container *app.Container) Handler {
			return NewTestHandler(container, "test2-handler", "test2")
		})

		assert.Equal(t, 2, len(factory.GetSupportedTypes()))

		// 모든 핸들러 제거
		factory.Clear()
		assert.Equal(t, 0, len(factory.GetSupportedTypes()))
	})

	t.Run("Factory stats", func(t *testing.T) {
		factory := NewHandlerFactory(container)

		// 핸들러 등록
		factory.Register("test", func(container *app.Container) Handler {
			return NewTestHandler(container, "test-handler", "test")
		})

		// 통계 확인
		stats := factory.GetStats()
		assert.Equal(t, 1, stats["total_types"])
		assert.Equal(t, []string{"test"}, stats["supported_types"])
		assert.Equal(t, true, stats["container_status"])
	})

	t.Run("Default factory", func(t *testing.T) {
		factory := GetDefaultFactory(container)

		assert.NotNil(t, factory)
		assert.Same(t, factory, GetDefaultFactory(container)) // 싱글톤 확인
	})

	t.Run("Register default handlers", func(t *testing.T) {
		factory := NewHandlerFactory(container)

		// 기본 핸들러들 등록
		RegisterDefaultHandlers(factory)

		// 지원되는 타입 확인
		supportedTypes := factory.GetSupportedTypes()
		expectedTypes := []string{"apt", "maven", "npm", "docker", "pip", "yum", "apk"}

		assert.Equal(t, len(expectedTypes), len(supportedTypes))
		for _, expectedType := range expectedTypes {
			assert.Contains(t, supportedTypes, expectedType)
			assert.True(t, factory.IsSupported(expectedType))
		}

		// 각 핸들러 생성 테스트
		for _, handlerType := range expectedTypes {
			handler, err := factory.Create(handlerType)
			require.NoError(t, err)
			assert.NotNil(t, handler)
			assert.Equal(t, handlerType, handler.Type())
		}
	})

	t.Run("Concurrent access", func(t *testing.T) {
		factory := NewHandlerFactory(container)

		// 동시에 핸들러 등록
		done := make(chan bool, 10)

		for i := 0; i < 10; i++ {
			go func(index int) {
				factory.Register(fmt.Sprintf("test%d", index), func(container *app.Container) Handler {
					return NewTestHandler(container, fmt.Sprintf("test%d-handler", index), fmt.Sprintf("test%d", index))
				})
				done <- true
			}(i)
		}

		// 모든 고루틴 완료 대기
		for i := 0; i < 10; i++ {
			<-done
		}

		// 결과 확인
		supportedTypes := factory.GetSupportedTypes()
		assert.Equal(t, 10, len(supportedTypes))

		// 동시에 핸들러 생성
		for i := 0; i < 10; i++ {
			go func(index int) {
				handler, err := factory.Create(fmt.Sprintf("test%d", index))
				assert.NoError(t, err)
				assert.NotNil(t, handler)
				done <- true
			}(i)
		}

		// 모든 고루틴 완료 대기
		for i := 0; i < 10; i++ {
			<-done
		}
	})
}
