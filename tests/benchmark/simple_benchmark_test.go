package benchmark

import (
	"bytes"
	"path/filepath"
	"testing"
	"time"

	"proxynd/cache"
)

// BenchmarkSimpleCache 간단한 캐시 성능 벤치마크
func BenchmarkSimpleCache(b *testing.B) {
	tempDir := b.TempDir()
	cacheBackend, err := cache.NewFileSystemBackendWithConfig(cache.FileSystemConfig{
		Path: filepath.Join(tempDir, "cache"),
	})
	if err != nil {
		b.Fatalf("Failed to create cache backend: %v", err)
	}

	cacheOptions := cache.CacheOptions{
		MaxSize:    1024 * 1024 * 100,  // 100MB
		DefaultTTL: 3600 * time.Second, // 1 hour
	}
	cacheManager := cache.NewManager(cacheBackend, cacheOptions)

	testData := make([]byte, 1024) // 1KB 테스트 데이터

	b.Run("Put", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			key := "test-key"
			err := cacheManager.Put(key, testData, 3600*time.Second)
			if err != nil {
				b.Fatalf("Cache put failed: %v", err)
			}
		}
	})

	// 캐시에 데이터 준비
	cacheManager.Put("bench-key", testData, 3600*time.Second)

	b.Run("Get", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, exists := cacheManager.Get("bench-key")
			if !exists {
				b.Fatalf("Cache miss for key: bench-key")
			}
		}
	})

	b.Run("Exists", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			exists := cacheManager.Exists("bench-key")
			if !exists {
				b.Fatalf("Key should exist: bench-key")
			}
		}
	})
}

// BenchmarkDataSizes 데이터 크기별 성능 벤치마크
func BenchmarkDataSizes(b *testing.B) {
	tempDir := b.TempDir()
	cacheBackend, err := cache.NewFileSystemBackendWithConfig(cache.FileSystemConfig{
		Path: filepath.Join(tempDir, "cache"),
	})
	if err != nil {
		b.Fatalf("Failed to create cache backend: %v", err)
	}

	cacheOptions := cache.CacheOptions{
		MaxSize:    1024 * 1024 * 500,  // 500MB
		DefaultTTL: 3600 * time.Second, // 1 hour
	}
	cacheManager := cache.NewManager(cacheBackend, cacheOptions)

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

		b.Run("Put_"+size.name, func(b *testing.B) {
			b.SetBytes(int64(size.size))
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				key := "test-key"
				err := cacheManager.Put(key, testData, 3600*time.Second)
				if err != nil {
					b.Fatalf("Cache put failed: %v", err)
				}
			}
		})

		// 캐시에 데이터 준비
		cacheManager.Put("bench-key-"+size.name, testData, 3600*time.Second)

		b.Run("Get_"+size.name, func(b *testing.B) {
			b.SetBytes(int64(size.size))
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				data, exists := cacheManager.Get("bench-key-" + size.name)
				if !exists {
					b.Fatalf("Cache miss for key: bench-key-%s", size.name)
				}
				if len(data) != size.size {
					b.Fatalf("Expected %d bytes, got %d", size.size, len(data))
				}
			}
		})
	}
}

// BenchmarkConcurrentAccess 동시 접근 성능 벤치마크
func BenchmarkConcurrentAccess(b *testing.B) {
	tempDir := b.TempDir()
	cacheBackend, err := cache.NewFileSystemBackendWithConfig(cache.FileSystemConfig{
		Path: filepath.Join(tempDir, "cache"),
	})
	if err != nil {
		b.Fatalf("Failed to create cache backend: %v", err)
	}

	cacheOptions := cache.CacheOptions{
		MaxSize:    1024 * 1024 * 100,  // 100MB
		DefaultTTL: 3600 * time.Second, // 1 hour
	}
	cacheManager := cache.NewManager(cacheBackend, cacheOptions)

	testData := make([]byte, 1024) // 1KB 테스트 데이터

	// 초기 데이터 준비
	for i := 0; i < 100; i++ {
		key := "concurrent-key"
		cacheManager.Put(key, testData, 3600*time.Second)
	}

	b.Run("ConcurrentReads", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_, exists := cacheManager.Get("concurrent-key")
				if !exists {
					b.Errorf("Cache miss for concurrent key")
				}
			}
		})
	})

	b.Run("ConcurrentWrites", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				key := "concurrent-write"
				err := cacheManager.Put(key, testData, 3600*time.Second)
				if err != nil {
					b.Errorf("Cache put failed: %v", err)
				}
				i++
			}
		})
	})
}

// BenchmarkMemoryAllocation 메모리 할당 성능 벤치마크
func BenchmarkMemoryAllocation(b *testing.B) {
	data1KB := make([]byte, 1024)
	data10KB := make([]byte, 10*1024)
	data100KB := make([]byte, 100*1024)

	b.Run("ByteSlice_1KB", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_ = append([]byte(nil), data1KB...)
		}
	})

	b.Run("ByteSlice_10KB", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_ = append([]byte(nil), data10KB...)
		}
	})

	b.Run("ByteSlice_100KB", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_ = append([]byte(nil), data100KB...)
		}
	})

	b.Run("ByteBuffer_1KB", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			var buf bytes.Buffer
			buf.Write(data1KB)
			_ = buf.Bytes()
		}
	})
}

// BenchmarkStringOperations 문자열 작업 성능 벤치마크
func BenchmarkStringOperations(b *testing.B) {
	testKeys := []string{
		"simple-key",
		"proxy/apt/ubuntu/dists/focal/Release",
		"proxy/maven/org/springframework/spring-core/5.3.21/spring-core-5.3.21.jar",
		"proxy/npm/@types/node/-/node-18.0.0.tgz",
	}

	b.Run("KeyGeneration", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			for _, key := range testKeys {
				_ = "cache:" + key
			}
		}
	})

	b.Run("KeyValidation", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			for _, key := range testKeys {
				// 간단한 키 검증 로직
				if len(key) > 0 && len(key) < 1000 {
					_ = true
				}
			}
		}
	})
}
