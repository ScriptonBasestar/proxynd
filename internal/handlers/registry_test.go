package handlers

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"proxynd/internal/app"
)

func TestHandlerRegistry(t *testing.T) {
	// 테스트용 컨테이너 생성
	cfg := &app.Config{
		ConfigDir: "/tmp/test-config",
		Port:      "8080",
	}
	container := app.NewContainer(cfg)
	defer func() { _ = container.Close() }()

	t.Run("Registry creation", func(t *testing.T) {
		registry := NewHandlerRegistry(container)

		assert.NotNil(t, registry)
		assert.NotNil(t, registry.handlers)
		assert.NotNil(t, registry.factory)
		assert.NotNil(t, registry.logger)
		assert.NotNil(t, registry.stats)

		// 기본 핸들러들이 등록되어 있는지 확인
		supportedTypes := registry.GetSupportedTypes()
		assert.Greater(t, len(supportedTypes), 0)
		assert.Contains(t, supportedTypes, "apt")
		assert.Contains(t, supportedTypes, "maven")
		assert.Contains(t, supportedTypes, "npm")
	})

	t.Run("Handler registration", func(t *testing.T) {
		registry := NewHandlerRegistry(container)

		// 커스텀 핸들러 등록
		registry.Register("custom", func(container *app.Container) Handler {
			return NewTestHandler(container, "custom-handler", "custom")
		})

		// 지원되는 타입 확인
		assert.True(t, registry.IsSupported("custom"))
		assert.Contains(t, registry.GetSupportedTypes(), "custom")
	})

	t.Run("Handler retrieval and caching", func(t *testing.T) {
		registry := NewHandlerRegistry(container)

		// 핸들러 등록
		registry.Register("test", func(container *app.Container) Handler {
			return NewTestHandler(container, "test-handler", "test")
		})

		// 첫 번째 호출 - 새 핸들러 생성
		handler1, err := registry.Get("test")
		require.NoError(t, err)
		assert.NotNil(t, handler1)
		assert.Equal(t, "test-handler", handler1.Name())

		// 두 번째 호출 - 캐시된 핸들러 반환
		handler2, err := registry.Get("test")
		require.NoError(t, err)
		assert.Same(t, handler1, handler2) // 같은 인스턴스여야 함

		// 통계 확인
		stats := registry.GetStats()
		assert.Equal(t, int64(1), stats["created_handlers"])
		assert.Equal(t, int64(1), stats["cache_hits"])
		assert.Equal(t, int64(1), stats["cache_misses"])
		assert.Equal(t, int64(2), stats["total_requests"])
		assert.Equal(t, 0.5, stats["cache_hit_rate"])
	})

	t.Run("Handler GetOrCreate", func(t *testing.T) {
		registry := NewHandlerRegistry(container)

		// 핸들러 등록
		registry.Register("test", func(container *app.Container) Handler {
			return NewTestHandler(container, "test-handler", "test")
		})

		// GetOrCreate 호출
		handler, err := registry.GetOrCreate("test")
		require.NoError(t, err)
		assert.NotNil(t, handler)
		assert.Equal(t, "test-handler", handler.Name())
	})

	t.Run("Handler GetCached", func(t *testing.T) {
		registry := NewHandlerRegistry(container)

		// 핸들러 등록
		registry.Register("test", func(container *app.Container) Handler {
			return NewTestHandler(container, "test-handler", "test")
		})

		// 캐시된 핸들러 없음
		handler := registry.GetCached("test")
		assert.Nil(t, handler)

		// 핸들러 생성 후 캐시된 핸들러 확인
		_, err := registry.Get("test")
		require.NoError(t, err)

		cachedHandler := registry.GetCached("test")
		assert.NotNil(t, cachedHandler)
		assert.Equal(t, "test-handler", cachedHandler.Name())
	})

	t.Run("Handler removal", func(t *testing.T) {
		registry := NewHandlerRegistry(container)

		// 핸들러 등록 및 생성
		registry.Register("test", func(container *app.Container) Handler {
			return NewTestHandler(container, "test-handler", "test")
		})

		handler, err := registry.Get("test")
		require.NoError(t, err)
		assert.NotNil(t, handler)

		// 핸들러 제거
		removed := registry.Remove("test")
		assert.True(t, removed)

		// 캐시된 핸들러 없음
		cachedHandler := registry.GetCached("test")
		assert.Nil(t, cachedHandler)

		// 존재하지 않는 핸들러 제거 시도
		removed = registry.Remove("nonexistent")
		assert.False(t, removed)
	})

	t.Run("Registry clear", func(t *testing.T) {
		registry := NewHandlerRegistry(container)

		// 여러 핸들러 등록 및 생성
		registry.Register("test1", func(container *app.Container) Handler {
			return NewTestHandler(container, "test1-handler", "test1")
		})
		registry.Register("test2", func(container *app.Container) Handler {
			return NewTestHandler(container, "test2-handler", "test2")
		})

		_, err := registry.Get("test1")
		require.NoError(t, err)
		_, err = registry.Get("test2")
		require.NoError(t, err)

		stats := registry.GetStats()
		assert.Equal(t, 2, stats["cached_handlers"])

		// 모든 캐시된 핸들러 제거
		registry.Clear()

		stats = registry.GetStats()
		assert.Equal(t, 0, stats["cached_handlers"])
	})

	t.Run("Registry stats", func(t *testing.T) {
		registry := NewHandlerRegistry(container)

		// 핸들러 등록
		registry.Register("test", func(container *app.Container) Handler {
			return NewTestHandler(container, "test-handler", "test")
		})

		// 여러 번 호출
		_, err := registry.Get("test")
		require.NoError(t, err)
		_, err = registry.Get("test")
		require.NoError(t, err)
		_, err = registry.Get("test")
		require.NoError(t, err)

		stats := registry.GetStats()
		assert.Equal(t, int64(1), stats["created_handlers"])
		assert.Equal(t, int64(2), stats["cache_hits"])
		assert.Equal(t, int64(1), stats["cache_misses"])
		assert.Equal(t, int64(3), stats["total_requests"])
		assert.Equal(t, 2.0/3.0, stats["cache_hit_rate"])

		// 통계 초기화
		registry.ResetStats()
		stats = registry.GetStats()
		assert.Equal(t, int64(0), stats["created_handlers"])
		assert.Equal(t, int64(0), stats["cache_hits"])
		assert.Equal(t, int64(0), stats["cache_misses"])
		assert.Equal(t, int64(0), stats["total_requests"])
		assert.Equal(t, 0.0, stats["cache_hit_rate"])
	})

	t.Run("Registry health check", func(t *testing.T) {
		registry := NewHandlerRegistry(container)

		// 정상 상태
		err := registry.Health()
		assert.NoError(t, err)

		// 핸들러 등록 후 생성
		registry.Register("test", func(container *app.Container) Handler {
			return NewTestHandler(container, "test-handler", "test")
		})

		_, err = registry.Get("test")
		require.NoError(t, err)

		// 헬스체크 (캐시된 핸들러 포함)
		err = registry.Health()
		assert.NoError(t, err)
	})

	t.Run("Handler preload", func(t *testing.T) {
		registry := NewHandlerRegistry(container)

		// 핸들러 등록
		registry.Register("test1", func(container *app.Container) Handler {
			return NewTestHandler(container, "test1-handler", "test1")
		})
		registry.Register("test2", func(container *app.Container) Handler {
			return NewTestHandler(container, "test2-handler", "test2")
		})

		// 프리로드
		err := registry.Preload("test1", "test2")
		assert.NoError(t, err)

		// 캐시된 핸들러 확인
		handler1 := registry.GetCached("test1")
		assert.NotNil(t, handler1)
		handler2 := registry.GetCached("test2")
		assert.NotNil(t, handler2)

		// 지원하지 않는 핸들러 프리로드 시도
		err = registry.Preload("nonexistent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported handler type")
	})

	t.Run("Registry warm up", func(t *testing.T) {
		registry := NewHandlerRegistry(container)

		// 워밍업 실행
		err := registry.WarmUp()
		assert.NoError(t, err)

		// 기본 핸들러들이 모두 캐시되었는지 확인
		stats := registry.GetStats()
		supportedTypes := registry.GetSupportedTypes()

		assert.Equal(t, len(supportedTypes), stats["cached_handlers"])

		// 각 핸들러가 캐시되었는지 확인
		for _, handlerType := range supportedTypes {
			handler := registry.GetCached(handlerType)
			assert.NotNil(t, handler, "Handler %s should be cached", handlerType)
		}
	})

	t.Run("Concurrent access", func(t *testing.T) {
		registry := NewHandlerRegistry(container)

		// 핸들러 등록
		registry.Register("test", func(container *app.Container) Handler {
			return NewTestHandler(container, "test-handler", "test")
		})

		// 동시에 핸들러 요청
		var wg sync.WaitGroup
		handlers := make([]Handler, 100)

		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				handler, err := registry.Get("test")
				assert.NoError(t, err)
				handlers[index] = handler
			}(i)
		}

		wg.Wait()

		// 모든 핸들러가 같은 인스턴스인지 확인
		for i := 1; i < len(handlers); i++ {
			assert.Same(t, handlers[0], handlers[i])
		}

		// 통계 확인
		stats := registry.GetStats()
		assert.Equal(t, int64(1), stats["created_handlers"])
		assert.Equal(t, int64(100), stats["total_requests"])
		assert.Equal(t, int64(99), stats["cache_hits"])
		assert.Equal(t, int64(1), stats["cache_misses"])
	})

	t.Run("Default registry", func(t *testing.T) {
		registry := GetDefaultRegistry(container)

		assert.NotNil(t, registry)
		assert.Same(t, registry, GetDefaultRegistry(container)) // 싱글톤 확인

		// 기본 핸들러들이 등록되어 있는지 확인
		supportedTypes := registry.GetSupportedTypes()
		assert.Contains(t, supportedTypes, "apt")
		assert.Contains(t, supportedTypes, "maven")
		assert.Contains(t, supportedTypes, "npm")
	})

	t.Run("Error handling", func(t *testing.T) {
		registry := NewHandlerRegistry(container)

		// 존재하지 않는 핸들러 요청
		_, err := registry.Get("nonexistent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unknown handler type")

		// 에러 발생하는 핸들러 등록
		registry.Register("error", func(_ *app.Container) Handler {
			return nil // nil 반환
		})

		// 통계에 오류가 기록되는지 확인
		stats := registry.GetStats()
		initialRequests := stats["total_requests"].(int64)

		handler, err := registry.Get("error")
		assert.NoError(t, err) // 팩토리에서는 에러 없음
		assert.Nil(t, handler) // 하지만 핸들러는 nil

		// 요청 수는 증가해야 함
		stats = registry.GetStats()
		assert.Equal(t, initialRequests+1, stats["total_requests"])
	})

	t.Run("Performance test", func(t *testing.T) {
		registry := NewHandlerRegistry(container)

		// 핸들러 등록
		registry.Register("perf", func(container *app.Container) Handler {
			return NewTestHandler(container, "perf-handler", "perf")
		})

		// 성능 테스트
		start := time.Now()
		iterations := 10000

		for i := 0; i < iterations; i++ {
			_, err := registry.Get("perf")
			assert.NoError(t, err)
		}

		duration := time.Since(start)

		// 통계 확인
		stats := registry.GetStats()
		assert.Equal(t, int64(1), stats["created_handlers"])
		assert.Equal(t, int64(iterations), stats["total_requests"])
		assert.Equal(t, int64(iterations-1), stats["cache_hits"])
		assert.Equal(t, int64(1), stats["cache_misses"])

		// 성능 확인 (평균 요청 처리 시간이 1μs 미만이어야 함)
		avgDuration := duration.Nanoseconds() / int64(iterations)
		assert.Less(t, avgDuration, int64(1000)) // 1μs = 1000ns

		t.Logf("Performance: %d requests in %v (avg: %dns per request)",
			iterations, duration, avgDuration)
	})
}
