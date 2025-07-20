package benchmark

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"proxynd/internal/repositories/cache"
)

// BenchmarkCacheSimple 간단한 캐시 성능 벤치마크
func BenchmarkCacheSimple(b *testing.B) {
	storageDir := b.TempDir()
	maxSize := int64(1024 * 1024 * 1024) // 1GB
	ttl := time.Hour

	repo, err := cache.NewFileRepository(storageDir, maxSize, ttl)
	require.NoError(b, err)

	ctx := context.Background()
	testData := make([]byte, 1024) // 1KB 테스트 데이터
	for i := range testData {
		testData[i] = byte(i % 256)
	}

	b.Run("Put-1KB", func(b *testing.B) {
		b.SetBytes(1024)
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("put-test-%d", i)
			reader := bytes.NewReader(testData)
			err := repo.Put(ctx, key, reader, ttl)
			require.NoError(b, err)
		}
	})

	// 데이터 준비
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("get-test-%d", i)
		reader := bytes.NewReader(testData)
		_ = repo.Put(ctx, key, reader, ttl)
	}

	b.Run("Get-1KB", func(b *testing.B) {
		b.SetBytes(1024)
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("get-test-%d", i%1000)
			reader, err := repo.Get(ctx, key)
			require.NoError(b, err)
			if reader != nil {
				_, _ = io.Copy(io.Discard, reader)
				_ = reader.Close()
			}
		}
	})

	b.Run("Exists", func(b *testing.B) {
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("get-test-%d", i%1000)
			exists, err := repo.Exists(ctx, key)
			require.NoError(b, err)
			_ = exists
		}
	})
}

// BenchmarkCacheConcurrent 동시 캐시 액세스 벤치마크
func BenchmarkCacheConcurrent(b *testing.B) {
	storageDir := b.TempDir()
	maxSize := int64(2 * 1024 * 1024 * 1024) // 2GB
	ttl := time.Hour

	repo, err := cache.NewFileRepository(storageDir, maxSize, ttl)
	require.NoError(b, err)

	ctx := context.Background()
	testData := make([]byte, 1024) // 1KB
	for i := range testData {
		testData[i] = byte(i % 256)
	}

	// 데이터 준비
	for i := 0; i < 10000; i++ {
		key := fmt.Sprintf("concurrent-test-%d", i)
		reader := bytes.NewReader(testData)
		_ = repo.Put(ctx, key, reader, ttl)
	}

	b.Run("ConcurrentGet", func(b *testing.B) {
		b.SetBytes(1024)
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				key := fmt.Sprintf("concurrent-test-%d", i%10000)
				reader, err := repo.Get(ctx, key)
				require.NoError(b, err)
				if reader != nil {
					_, _ = io.Copy(io.Discard, reader)
					_ = reader.Close()
				}
				i++
			}
		})
	})

	b.Run("ConcurrentPut", func(b *testing.B) {
		b.SetBytes(1024)
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				key := fmt.Sprintf("concurrent-put-%d", i)
				reader := bytes.NewReader(testData)
				err := repo.Put(ctx, key, reader, ttl)
				require.NoError(b, err)
				i++
			}
		})
	})
}

// BenchmarkCacheSizes 다양한 크기 데이터 캐시 성능
func BenchmarkCacheSizes(b *testing.B) {
	storageDir := b.TempDir()
	maxSize := int64(5 * 1024 * 1024 * 1024) // 5GB
	ttl := time.Hour

	repo, err := cache.NewFileRepository(storageDir, maxSize, ttl)
	require.NoError(b, err)

	ctx := context.Background()

	sizes := []struct {
		name string
		size int
	}{
		{"1KB", 1024},
		{"10KB", 10 * 1024},
		{"100KB", 100 * 1024},
		{"1MB", 1024 * 1024},
	}

	for _, size := range sizes {
		testData := make([]byte, size.size)
		for i := range testData {
			testData[i] = byte(i % 256)
		}

		b.Run(fmt.Sprintf("Put-%s", size.name), func(b *testing.B) {
			b.SetBytes(int64(size.size))
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				key := fmt.Sprintf("%s-put-%d", size.name, i)
				reader := bytes.NewReader(testData)
				err := repo.Put(ctx, key, reader, ttl)
				require.NoError(b, err)
			}
		})

		// 데이터 준비
		keys := make([]string, 100)
		for i := 0; i < 100; i++ {
			key := fmt.Sprintf("%s-get-%d", size.name, i)
			keys[i] = key
			reader := bytes.NewReader(testData)
			_ = repo.Put(ctx, key, reader, ttl)
		}

		b.Run(fmt.Sprintf("Get-%s", size.name), func(b *testing.B) {
			b.SetBytes(int64(size.size))
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				key := keys[i%100]
				reader, err := repo.Get(ctx, key)
				require.NoError(b, err)
				if reader != nil {
					n, err := io.Copy(io.Discard, reader)
					require.NoError(b, err)
					require.Equal(b, int64(size.size), n)
					_ = reader.Close()
				}
			}
		})
	}
}

// BenchmarkCacheMemory 캐시 메모리 사용량 벤치마크
func BenchmarkCacheMemory(b *testing.B) {
	storageDir := b.TempDir()
	maxSize := int64(1024 * 1024 * 1024) // 1GB
	ttl := time.Hour

	repo, err := cache.NewFileRepository(storageDir, maxSize, ttl)
	require.NoError(b, err)

	ctx := context.Background()
	testData := make([]byte, 1024) // 1KB
	for i := range testData {
		testData[i] = byte(i % 256)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("memory-test-%d", i)
		reader := bytes.NewReader(testData)
		err := repo.Put(ctx, key, reader, ttl)
		require.NoError(b, err)

		// 즉시 읽기
		readCloser, err := repo.Get(ctx, key)
		require.NoError(b, err)
		if readCloser != nil {
			_, _ = io.Copy(io.Discard, readCloser)
			_ = readCloser.Close()
		}
	}
}
