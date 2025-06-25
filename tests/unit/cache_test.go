package unit

import (
	"bytes"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"proxynd/cache"
)

// TestFileSystemBackend 파일시스템 캐시 백엔드 테스트
func TestFileSystemBackend(t *testing.T) {
	// 임시 디렉토리 생성
	tempDir, err := os.MkdirTemp("", "proxynd-cache-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// 백엔드 생성
	backend, err := cache.NewFileSystemBackendWithConfig(cache.FileSystemConfig{
		Path: tempDir,
	})
	require.NoError(t, err)

	t.Run("Put and Get", func(t *testing.T) {
		key := "test-key"
		data := []byte("test data")
		ttl := time.Hour

		// 데이터 저장
		err := backend.Put(key, bytes.NewReader(data), ttl)
		require.NoError(t, err)

		// 데이터 조회
		reader, err := backend.Get(key)
		require.NoError(t, err)
		defer reader.Close()

		// 데이터 비교
		buf := new(bytes.Buffer)
		_, err = buf.ReadFrom(reader)
		require.NoError(t, err)
		assert.Equal(t, data, buf.Bytes())
	})

	t.Run("Exists", func(t *testing.T) {
		key := "exists-test"
		data := []byte("exists data")

		// 존재하지 않는 키
		assert.False(t, backend.Exists("nonexistent"))

		// 데이터 저장 후 존재 확인
		err := backend.Put(key, bytes.NewReader(data), time.Hour)
		require.NoError(t, err)
		assert.True(t, backend.Exists(key))
	})

	t.Run("Delete", func(t *testing.T) {
		key := "delete-test"
		data := []byte("delete data")

		// 데이터 저장
		err := backend.Put(key, bytes.NewReader(data), time.Hour)
		require.NoError(t, err)
		assert.True(t, backend.Exists(key))

		// 데이터 삭제
		err = backend.Delete(key)
		require.NoError(t, err)
		assert.False(t, backend.Exists(key))
	})

	t.Run("TTL Expiration", func(t *testing.T) {
		key := "ttl-test"
		data := []byte("ttl data")
		shortTTL := 100 * time.Millisecond

		// 짧은 TTL로 데이터 저장
		err := backend.Put(key, bytes.NewReader(data), shortTTL)
		require.NoError(t, err)

		// 즉시 조회 (성공해야 함)
		assert.True(t, backend.Exists(key))

		// TTL 만료 대기
		time.Sleep(150 * time.Millisecond)

		// 만료 후 조회 (실패해야 함)
		assert.False(t, backend.Exists(key))
	})

	t.Run("Size", func(t *testing.T) {
		// 초기 크기
		initialSize, err := backend.Size()
		require.NoError(t, err)

		// 데이터 추가
		data := []byte("size test data")
		err = backend.Put("size-test", bytes.NewReader(data), time.Hour)
		require.NoError(t, err)

		// 크기 증가 확인
		newSize, err := backend.Size()
		require.NoError(t, err)
		assert.Greater(t, newSize, initialSize)
	})

	t.Run("Clear", func(t *testing.T) {
		// 여러 데이터 저장
		for i := 0; i < 5; i++ {
			key := fmt.Sprintf("clear-test-%d", i)
			data := []byte(fmt.Sprintf("clear data %d", i))
			err := backend.Put(key, bytes.NewReader(data), time.Hour)
			require.NoError(t, err)
		}

		// 전체 삭제
		err := backend.Clear()
		require.NoError(t, err)

		// 모든 데이터가 삭제되었는지 확인
		for i := 0; i < 5; i++ {
			key := fmt.Sprintf("clear-test-%d", i)
			assert.False(t, backend.Exists(key))
		}
	})
}

// TestCacheManager 캐시 매니저 테스트
func TestCacheManager(t *testing.T) {
	// 임시 디렉토리 생성
	tempDir, err := os.MkdirTemp("", "proxynd-manager-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// 백엔드 생성
	backend, err := cache.NewFileSystemBackendWithConfig(cache.FileSystemConfig{
		Path: tempDir,
	})
	require.NoError(t, err)

	// 매니저 생성
	options := cache.CacheOptions{
		DefaultTTL: time.Hour,
		MaxSize:    1024 * 1024, // 1MB
		BasePath:   "cache",
	}
	manager := cache.NewManager(backend, options)

	t.Run("Get and Put", func(t *testing.T) {
		key := "manager-test"
		data := []byte("manager test data")

		// 존재하지 않는 키 조회
		_, exists := manager.Get(key)
		assert.False(t, exists)

		// 데이터 저장
		err := manager.Put(key, data, 0) // 기본 TTL 사용
		require.NoError(t, err)

		// 데이터 조회
		retrieved, exists := manager.Get(key)
		assert.True(t, exists)
		assert.Equal(t, data, retrieved)
	})

	t.Run("Cache Stats", func(t *testing.T) {
		// 초기 통계
		stats := manager.GetStats()
		initialHits := stats.Hits
		initialMisses := stats.Misses

		key := "stats-test"
		data := []byte("stats data")

		// 존재하지 않는 키 조회 (미스)
		_, exists := manager.Get(key)
		assert.False(t, exists)

		// 미스 증가 확인
		stats = manager.GetStats()
		assert.Equal(t, initialMisses+1, stats.Misses)

		// 데이터 저장
		err := manager.Put(key, data, 0)
		require.NoError(t, err)

		// 존재하는 키 조회 (히트)
		_, exists = manager.Get(key)
		assert.True(t, exists)

		// 히트 증가 확인
		stats = manager.GetStats()
		assert.Equal(t, initialHits+1, stats.Hits)
	})

	t.Run("Cache Path Generation", func(t *testing.T) {
		proxyType := "npm"
		requestPath := "express/4.18.2"
		
		expectedPath := "cache/npm/express/4.18.2"
		actualPath := manager.GetCachePath(proxyType, requestPath)
		
		assert.Equal(t, expectedPath, actualPath)
	})

	t.Run("Delete", func(t *testing.T) {
		key := "delete-test"
		data := []byte("delete data")

		// 데이터 저장
		err := manager.Put(key, data, 0)
		require.NoError(t, err)

		// 존재 확인
		assert.True(t, manager.Exists(key))

		// 삭제
		err = manager.Delete(key)
		require.NoError(t, err)

		// 삭제 확인
		assert.False(t, manager.Exists(key))
	})

	t.Run("Clear", func(t *testing.T) {
		// 여러 데이터 저장
		for i := 0; i < 3; i++ {
			key := fmt.Sprintf("clear-test-%d", i)
			data := []byte(fmt.Sprintf("clear data %d", i))
			err := manager.Put(key, data, 0)
			require.NoError(t, err)
		}

		// 전체 삭제
		err := manager.Clear()
		require.NoError(t, err)

		// 통계 확인 (LastClear 시간이 업데이트되어야 함)
		stats := manager.GetStats()
		assert.True(t, time.Since(stats.LastClear) < time.Second)
	})
}

// TestCacheEviction 캐시 제거 정책 테스트
func TestCacheEviction(t *testing.T) {
	t.Run("LRU Policy", func(t *testing.T) {
		policy := &cache.LRUEvictionPolicy{}

		// TTL 만료 테스트
		now := time.Now()
		expiredMeta := &cache.CacheMetadata{
			Key:        "expired",
			CreatedAt:  now.Add(-2 * time.Hour),
			TTL:        time.Hour,
		}
		
		shouldEvict := policy.ShouldEvict(100, 200, expiredMeta)
		assert.True(t, shouldEvict, "TTL 만료된 항목은 제거되어야 함")

		// 크기 초과 테스트
		validMeta := &cache.CacheMetadata{
			Key:       "valid",
			CreatedAt: now,
			TTL:       time.Hour,
		}
		
		shouldEvict = policy.ShouldEvict(300, 200, validMeta)
		assert.True(t, shouldEvict, "크기 초과 시 제거되어야 함")

		// 정상 상태 테스트
		shouldEvict = policy.ShouldEvict(100, 200, validMeta)
		assert.False(t, shouldEvict, "정상 상태에서는 제거되지 않아야 함")
	})

	t.Run("Eviction Candidates Selection", func(t *testing.T) {
		policy := &cache.LRUEvictionPolicy{}
		now := time.Now()

		// 테스트 메타데이터 생성 (접근 시간 순서대로)
		items := []*cache.CacheMetadata{
			{
				Key:        "oldest",
				Size:       100,
				AccessedAt: now.Add(-3 * time.Hour),
			},
			{
				Key:        "middle",
				Size:       150,
				AccessedAt: now.Add(-2 * time.Hour),
			},
			{
				Key:        "newest",
				Size:       200,
				AccessedAt: now.Add(-1 * time.Hour),
			},
		}

		// 200 바이트 공간이 필요한 경우
		candidates := policy.SelectEvictionCandidates(items, 200)
		
		// 가장 오래된 항목부터 선택되어야 함
		assert.Contains(t, candidates, "oldest")
		
		// 필요한 공간이 확보될 때까지 선택
		totalFreed := int64(0)
		for _, candidate := range candidates {
			for _, item := range items {
				if item.Key == candidate {
					totalFreed += item.Size
					break
				}
			}
		}
		assert.GreaterOrEqual(t, totalFreed, int64(200))
	})
}