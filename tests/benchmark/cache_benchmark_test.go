package benchmark

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"proxynd/internal/repositories/cache"
)

// BenchmarkCacheBackend 캐시 백엔드 성능 벤치마크
func BenchmarkCacheBackend(b *testing.B) {
	storageDir := b.TempDir()
	maxSize := int64(1024 * 1024 * 1024) // 1GB
	ttl := time.Hour

	repo, err := cache.NewFileRepository(storageDir, maxSize, ttl)
	require.NoError(b, err)

	testData := make([]byte, 1024) // 1KB 테스트 데이터
	for i := range testData {
		testData[i] = byte(i % 256)
	}

	ctx := context.Background()

	b.Run("Put", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("test-key-%d", i)
			reader := bytes.NewReader(testData)
			err := repo.Put(ctx, key, reader, ttl)
			require.NoError(b, err)
		}
	})

	// Put으로 데이터 준비
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("benchmark-key-%d", i)
		reader := bytes.NewReader(testData)
		_ = repo.Put(ctx, key, reader, ttl)
	}

	b.Run("Get", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("benchmark-key-%d", i%1000)
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
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("benchmark-key-%d", i%1000)
			exists, err := repo.Exists(ctx, key)
			require.NoError(b, err)
			_ = exists
		}
	})

	b.Run("Delete", func(b *testing.B) {
		// 삭제용 데이터 준비
		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("delete-key-%d", i)
			reader := bytes.NewReader(testData)
			_ = repo.Put(ctx, key, reader, ttl)
		}

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("delete-key-%d", i)
			err := repo.Delete(ctx, key)
			require.NoError(b, err)
		}
	})
}

// BenchmarkCacheDataSizes 다양한 데이터 크기별 캐시 성능
func BenchmarkCacheDataSizes(b *testing.B) {
	storageDir := b.TempDir()
	maxSize := int64(10 * 1024 * 1024 * 1024) // 10GB
	ttl := time.Hour

	repo, err := cache.NewFileRepository(storageDir, maxSize, ttl)
	require.NoError(b, err)

	ctx := context.Background()

	dataSizes := []struct {
		name string
		size int
	}{
		{"1KB", 1024},
		{"10KB", 10 * 1024},
		{"100KB", 100 * 1024},
		{"1MB", 1024 * 1024},
		{"10MB", 10 * 1024 * 1024},
		{"100MB", 100 * 1024 * 1024},
	}

	for _, ds := range dataSizes {
		b.Run(ds.name, func(b *testing.B) {
			testData := make([]byte, ds.size)
			for i := range testData {
				testData[i] = byte(i % 256)
			}

			b.Run("Put", func(b *testing.B) {
				b.SetBytes(int64(ds.size))
				b.ResetTimer()

				for i := 0; i < b.N; i++ {
					key := fmt.Sprintf("%s-put-%d", ds.name, i)
					reader := bytes.NewReader(testData)
					err := repo.Put(ctx, key, reader, ttl)
					require.NoError(b, err)
				}
			})

			// 데이터 준비
			keys := make([]string, 100)
			for i := 0; i < 100; i++ {
				key := fmt.Sprintf("%s-get-%d", ds.name, i)
				keys[i] = key
				reader := bytes.NewReader(testData)
				_ = repo.Put(ctx, key, reader, ttl)
			}

			b.Run("Get", func(b *testing.B) {
				b.SetBytes(int64(ds.size))
				b.ResetTimer()

				for i := 0; i < b.N; i++ {
					key := keys[i%100]
					reader, err := repo.Get(ctx, key)
					require.NoError(b, err)
					if reader != nil {
						n, err := io.Copy(io.Discard, reader)
						require.NoError(b, err)
						require.Equal(b, int64(ds.size), n)
						_ = reader.Close()
					}
				}
			})
		})
	}
}

// BenchmarkCacheConcurrency 동시성 캐시 벤치마크
func BenchmarkCacheConcurrency(b *testing.B) {
	storageDir := b.TempDir()
	maxSize := int64(5 * 1024 * 1024 * 1024) // 5GB
	ttl := time.Hour

	repo, err := cache.NewFileRepository(storageDir, maxSize, ttl)
	require.NoError(b, err)

	ctx := context.Background()
	testData := make([]byte, 10*1024) // 10KB
	for i := range testData {
		testData[i] = byte(i % 256)
	}

	// 동시 쓰기 성능
	b.Run("ConcurrentPut", func(b *testing.B) {
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

	// 동시 읽기를 위한 데이터 준비
	for i := 0; i < 10000; i++ {
		key := fmt.Sprintf("concurrent-get-%d", i)
		reader := bytes.NewReader(testData)
		_ = repo.Put(ctx, key, reader, ttl)
	}

	// 동시 읽기 성능
	b.Run("ConcurrentGet", func(b *testing.B) {
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				key := fmt.Sprintf("concurrent-get-%d", i%10000)
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

	// 혼합 동시 작업 (읽기/쓰기/존재 확인)
	b.Run("ConcurrentMixed", func(b *testing.B) {
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				switch i % 3 {
				case 0: // 쓰기
					key := fmt.Sprintf("mixed-put-%d", i)
					reader := bytes.NewReader(testData)
					_ = repo.Put(ctx, key, reader, ttl)
				case 1: // 읽기
					key := fmt.Sprintf("concurrent-get-%d", i%10000)
					reader, err := repo.Get(ctx, key)
					if err == nil && reader != nil {
						_, _ = io.Copy(io.Discard, reader)
						_ = reader.Close()
					}
				case 2: // 존재 확인
					key := fmt.Sprintf("concurrent-get-%d", i%10000)
					_, _ = repo.Exists(ctx, key)
				}
				i++
			}
		})
	})
}

// BenchmarkCacheTTL TTL 관련 캐시 성능
func BenchmarkCacheTTL(b *testing.B) {
	storageDir := b.TempDir()
	maxSize := int64(1024 * 1024 * 1024) // 1GB
	shortTTL := 100 * time.Millisecond
	longTTL := time.Hour

	repo, err := cache.NewFileRepository(storageDir, maxSize, longTTL)
	require.NoError(b, err)

	ctx := context.Background()
	testData := make([]byte, 1024) // 1KB
	for i := range testData {
		testData[i] = byte(i % 256)
	}

	b.Run("ShortTTL", func(b *testing.B) {
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("short-ttl-%d", i)
			reader := bytes.NewReader(testData)
			err := repo.Put(ctx, key, reader, shortTTL)
			require.NoError(b, err)
		}
	})

	b.Run("LongTTL", func(b *testing.B) {
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("long-ttl-%d", i)
			reader := bytes.NewReader(testData)
			err := repo.Put(ctx, key, reader, longTTL)
			require.NoError(b, err)
		}
	})

	// TTL 만료된 데이터 읽기 성능
	b.Run("ExpiredRead", func(b *testing.B) {
		// 짧은 TTL로 데이터 저장 후 만료 대기
		for i := 0; i < 1000; i++ {
			key := fmt.Sprintf("expired-%d", i)
			reader := bytes.NewReader(testData)
			_ = repo.Put(ctx, key, reader, 50*time.Millisecond)
		}

		time.Sleep(100 * time.Millisecond) // TTL 만료 대기

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("expired-%d", i%1000)
			reader, err := repo.Get(ctx, key)
			if err == nil && reader != nil {
				_, _ = io.Copy(io.Discard, reader)
				_ = reader.Close()
			}
		}
	})
}

// BenchmarkCacheEviction 캐시 정리 성능
func BenchmarkCacheEviction(b *testing.B) {
	storageDir := b.TempDir()
	maxSize := int64(50 * 1024 * 1024) // 50MB 작은 캐시
	ttl := time.Hour

	repo, err := cache.NewFileRepository(storageDir, maxSize, ttl)
	require.NoError(b, err)

	ctx := context.Background()
	// 큰 데이터로 캐시 포화
	largeData := make([]byte, 1024*1024) // 1MB
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	b.Run("EvictionPressure", func(b *testing.B) {
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("eviction-%d", i)
			reader := bytes.NewReader(largeData)
			err := repo.Put(ctx, key, reader, ttl)
			// 캐시가 가득 찬 경우 에러가 발생할 수 있음
			_ = err
		}
	})
}

// BenchmarkCacheMetadata 캐시 메타데이터 성능
func BenchmarkCacheMetadata(b *testing.B) {
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

	// 메타데이터 조회를 위한 데이터 준비
	for i := 0; i < 10000; i++ {
		key := fmt.Sprintf("metadata-%d", i)
		reader := bytes.NewReader(testData)
		_ = repo.Put(ctx, key, reader, ttl)
	}

	b.Run("GetSize", func(b *testing.B) {
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("metadata-%d", i%10000)
			size, err := repo.Size(ctx, key)
			require.NoError(b, err)
			_ = size
		}
	})

	b.Run("Stats", func(b *testing.B) {
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			stats, err := repo.Stats(ctx)
			require.NoError(b, err)
			_ = stats
		}
	})

	b.Run("ListKeys", func(b *testing.B) {
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			prefix := fmt.Sprintf("metadata-%d*", i%100)
			keys, err := repo.List(ctx, prefix)
			require.NoError(b, err)
			_ = keys
		}
	})
}

// BenchmarkCachePatterns 실제 사용 패턴 시뮬레이션
func BenchmarkCachePatterns(b *testing.B) {
	storageDir := b.TempDir()
	maxSize := int64(2 * 1024 * 1024 * 1024) // 2GB
	ttl := time.Hour

	repo, err := cache.NewFileRepository(storageDir, maxSize, ttl)
	require.NoError(b, err)

	ctx := context.Background()
	// 다양한 크기의 패키지 시뮬레이션
	packageSizes := map[string]int{
		"small":  1024,             // 1KB - 메타데이터
		"medium": 100 * 1024,       // 100KB - 작은 패키지
		"large":  10 * 1024 * 1024, // 10MB - 큰 패키지
	}

	// 실제 패키지 이름 패턴
	packageNames := []string{
		"npm/express", "npm/react", "npm/lodash", "npm/axios", "npm/moment",
		"maven/junit/junit", "maven/spring/core", "maven/hibernate/core",
		"apt/ubuntu/nginx", "apt/ubuntu/apache2", "apt/ubuntu/mysql-server",
	}

	b.Run("RealWorldPattern", func(b *testing.B) {
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			// 90% 읽기, 10% 쓰기 패턴
			if i%10 == 0 {
				// 쓰기 작업
				packageName := packageNames[i%len(packageNames)]
				sizeType := []string{"small", "medium", "large"}[i%3]
				size := packageSizes[sizeType]

				data := make([]byte, size)
				for j := range data {
					data[j] = byte(j % 256)
				}

				key := fmt.Sprintf("%s/%s-%d", packageName, sizeType, i)
				reader := bytes.NewReader(data)
				_ = repo.Put(ctx, key, reader, ttl)
			} else {
				// 읽기 작업 (기존 데이터 중 랜덤 선택)
				packageName := packageNames[i%len(packageNames)]
				sizeType := []string{"small", "medium", "large"}[i%3]
				key := fmt.Sprintf("%s/%s-%d", packageName, sizeType, (i/10)*10)

				reader, err := repo.Get(ctx, key)
				if err == nil && reader != nil {
					_, _ = io.Copy(io.Discard, reader)
					_ = reader.Close()
				}
			}
		}
	})

	// 캐시 히트율 시뮬레이션
	b.Run("HighHitRate", func(b *testing.B) {
		// 인기 패키지 데이터 준비 (자주 요청되는 패키지)
		popularPackages := []string{
			"npm/express/metadata", "npm/react/metadata", "npm/lodash/metadata",
		}

		for _, pkg := range popularPackages {
			data := []byte(strings.Repeat("metadata", 100))
			reader := bytes.NewReader(data)
			_ = repo.Put(ctx, pkg, reader, ttl)
		}

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			// 80% 인기 패키지 요청, 20% 신규 패키지 요청
			if i%5 == 0 {
				// 신규 패키지 (캐시 미스)
				key := fmt.Sprintf("new-package-%d", i)
				data := []byte(strings.Repeat("new-data", 50))
				reader := bytes.NewReader(data)
				_ = repo.Put(ctx, key, reader, ttl)
			} else {
				// 인기 패키지 (캐시 히트)
				pkg := popularPackages[i%len(popularPackages)]
				reader, err := repo.Get(ctx, pkg)
				if err == nil && reader != nil {
					_, _ = io.Copy(io.Discard, reader)
					_ = reader.Close()
				}
			}
		}
	})
}
