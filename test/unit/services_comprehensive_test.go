package unit

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"proxynd/cache/mocks"
	cacheMocks "proxynd/cache/mocks"
	"proxynd/configs"
	"proxynd/internal/domain/docker"
	"proxynd/internal/domain/pip"
	"proxynd/internal/repositories/cache"
	dockerServices "proxynd/internal/services/docker"
	pipServices "proxynd/internal/services/pip"
	proxyServices "proxynd/internal/services/proxy"
)

// ServiceTestSuite 서비스 테스트 스위트
type ServiceTestSuite struct {
	ctx          context.Context
	mockCache    *cacheMocks.MockCache
	mockEviction *mocks.MockEvictionPolicy
	cacheRepo    cache.Repository
	testTempDir  string
}

// SetupServiceTestSuite 서비스 테스트 스위트 설정
func SetupServiceTestSuite(t *testing.T) *ServiceTestSuite {
	suite := &ServiceTestSuite{
		ctx:         context.Background(),
		testTempDir: t.TempDir(),
	}

	// Mock 객체들 생성
	suite.mockCache = cacheMocks.NewMockCache(t)
	suite.mockEviction = mocks.NewMockEvictionPolicy(t)

	return suite
}

// TestPIPServices PIP 서비스 계층 테스트
func TestPIPServices(t *testing.T) {
	suite := SetupServiceTestSuite(t)

	t.Run("PackageService", func(t *testing.T) {
		// 설정 생성
		config := &configs.PipProxyConfig{
			Enabled: true,
			Mirrors: []configs.PipMirror{
				{Name: "pypi", URL: "https://pypi.org", Timeout: "30s"},
			},
			Cache: configs.CacheConfig{
				Enabled: true,
				TTL:     "1h",
			},
		}

		// PackageService 생성
		packageService := pipServices.NewPackageService(config, suite.mockCache)

		t.Run("GetPackageMetadata Success", func(t *testing.T) {
			// 캐시에서 메타데이터를 찾지 못하는 경우
			suite.mockCache.EXPECT().
				Get(mock.Anything, mock.AnythingOfType("string")).
				Return(nil, fmt.Errorf("cache miss")).
				Once()

			// 캐시에 메타데이터 저장
			suite.mockCache.EXPECT().
				Put(mock.Anything, mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("time.Duration")).
				Return(nil).
				Once()

			metadata, err := packageService.GetPackageMetadata(suite.ctx, "requests", "2.28.1")

			assert.NoError(t, err)
			assert.NotNil(t, metadata)
			assert.Equal(t, "requests", metadata.Name)
			assert.Equal(t, "2.28.1", metadata.Version)
		})

		t.Run("GetPackageMetadata Cache Hit", func(t *testing.T) {
			// 캐시된 데이터 준비
			cachedData := `{"name":"requests","version":"2.28.1","summary":"HTTP library"}`

			suite.mockCache.EXPECT().
				Get(mock.Anything, mock.AnythingOfType("string")).
				Return(io.NopCloser(strings.NewReader(cachedData)), nil).
				Once()

			metadata, err := packageService.GetPackageMetadata(suite.ctx, "requests", "2.28.1")

			assert.NoError(t, err)
			assert.NotNil(t, metadata)
			assert.Equal(t, "requests", metadata.Name)
		})

		t.Run("SearchPackages", func(t *testing.T) {
			filters := pip.SearchFilters{
				Classifier: "Development Status :: 4 - Beta",
				MaxResults: 10,
			}

			packages, err := packageService.SearchPackages(suite.ctx, "http", filters)

			assert.NoError(t, err)
			assert.NotNil(t, packages)
			// 실제 검색 결과는 mock upstream 설정에 따라 달라짐
		})
	})

	t.Run("IndexManager", func(t *testing.T) {
		config := &configs.PipProxyConfig{
			Enabled: true,
			Mirrors: []configs.PipMirror{
				{Name: "pypi", URL: "https://pypi.org"},
			},
		}

		indexManager := pipServices.NewIndexManager(config, suite.mockCache)

		t.Run("GetSimpleIndex", func(t *testing.T) {
			// 캐시 미스 시뮬레이션
			suite.mockCache.EXPECT().
				Get(mock.Anything, mock.AnythingOfType("string")).
				Return(nil, fmt.Errorf("cache miss")).
				Once()

			// 캐시 저장
			suite.mockCache.EXPECT().
				Put(mock.Anything, mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("time.Duration")).
				Return(nil).
				Once()

			indexHTML, err := indexManager.GetSimpleIndex(suite.ctx, "requests")

			assert.NoError(t, err)
			assert.NotEmpty(t, indexHTML)
			assert.Contains(t, indexHTML, "requests")
		})

		t.Run("UpdateIndex", func(t *testing.T) {
			packageList := []string{"requests", "numpy", "django"}

			err := indexManager.UpdateIndex(suite.ctx, packageList)

			assert.NoError(t, err)
		})
	})

	t.Run("CacheManager", func(t *testing.T) {
		config := &configs.PipProxyConfig{
			Cache: configs.CacheConfig{
				Enabled:   true,
				TTL:       "1h",
				MaxSize:   "1GB",
				Directory: suite.testTempDir,
			},
		}

		cacheManager := pipServices.NewCacheManager(config, suite.mockCache)

		t.Run("CachePackageFile", func(t *testing.T) {
			fileData := []byte("mock package data")
			reader := strings.NewReader(string(fileData))

			suite.mockCache.EXPECT().
				Put(mock.Anything, mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("time.Duration")).
				Return(nil).
				Once()

			err := cacheManager.CachePackageFile(suite.ctx, "requests-2.28.1.tar.gz", reader, time.Hour)

			assert.NoError(t, err)
		})

		t.Run("GetCachedFile", func(t *testing.T) {
			cachedData := "cached package data"

			suite.mockCache.EXPECT().
				Get(mock.Anything, "requests-2.28.1.tar.gz").
				Return(io.NopCloser(strings.NewReader(cachedData)), nil).
				Once()

			reader, err := cacheManager.GetCachedFile(suite.ctx, "requests-2.28.1.tar.gz")

			assert.NoError(t, err)
			assert.NotNil(t, reader)

			// 데이터 검증
			data, err := io.ReadAll(reader)
			assert.NoError(t, err)
			assert.Equal(t, cachedData, string(data))
		})

		t.Run("InvalidateCache", func(t *testing.T) {
			err := cacheManager.InvalidateCache(suite.ctx, "requests*")

			assert.NoError(t, err)
		})
	})

	t.Run("MetricsCollector", func(t *testing.T) {
		metricsCollector := pipServices.NewMetricsCollector()

		t.Run("RecordRequest", func(t *testing.T) {
			// 다양한 요청 기록
			metricsCollector.RecordRequest("simple", 200, 100*time.Millisecond)
			metricsCollector.RecordRequest("pypi", 200, 150*time.Millisecond)
			metricsCollector.RecordRequest("simple", 404, 50*time.Millisecond)

			// 메트릭 검증
			metrics := metricsCollector.GetMetrics()
			assert.NotNil(t, metrics)
			assert.Greater(t, metrics.TotalRequests, int64(0))
			assert.Greater(t, metrics.SuccessfulRequests, int64(0))
		})

		t.Run("RecordCacheOperation", func(t *testing.T) {
			metricsCollector.RecordCacheHit("requests")
			metricsCollector.RecordCacheMiss("numpy")
			metricsCollector.RecordCacheHit("django")

			metrics := metricsCollector.GetMetrics()
			assert.Greater(t, metrics.CacheHits, int64(0))
			assert.Greater(t, metrics.CacheMisses, int64(0))
		})

		t.Run("GetStatistics", func(t *testing.T) {
			stats := metricsCollector.GetStatistics()
			assert.NotNil(t, stats)

			// 통계 필드 검증
			assert.GreaterOrEqual(t, stats.RequestsPerSecond, float64(0))
			assert.GreaterOrEqual(t, stats.AverageResponseTime, time.Duration(0))
			assert.GreaterOrEqual(t, stats.CacheHitRatio, float64(0))
			assert.LessOrEqual(t, stats.CacheHitRatio, float64(1))
		})
	})
}

// TestDockerServices Docker 서비스 계층 테스트
func TestDockerServices(t *testing.T) {
	suite := SetupServiceTestSuite(t)

	t.Run("RegistryService", func(t *testing.T) {
		config := &configs.DockerProxyConfig{
			Enabled: true,
			Registries: []configs.DockerRegistry{
				{Name: "dockerhub", URL: "https://registry-1.docker.io", Timeout: "30s"},
			},
			Cache: configs.CacheConfig{
				Enabled: true,
				TTL:     "1h",
			},
		}

		registryService := dockerServices.NewRegistryService(config, suite.mockCache)

		t.Run("GetManifest", func(t *testing.T) {
			// 캐시 미스
			suite.mockCache.EXPECT().
				Get(mock.Anything, mock.AnythingOfType("string")).
				Return(nil, fmt.Errorf("cache miss")).
				Once()

			// 캐시 저장
			suite.mockCache.EXPECT().
				Put(mock.Anything, mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("time.Duration")).
				Return(nil).
				Once()

			manifest, err := registryService.GetManifest(suite.ctx, "library/nginx", "latest")

			assert.NoError(t, err)
			assert.NotNil(t, manifest)
			assert.NotEmpty(t, manifest.MediaType)
		})

		t.Run("ListTags", func(t *testing.T) {
			tags, err := registryService.ListTags(suite.ctx, "library/nginx")

			assert.NoError(t, err)
			assert.NotNil(t, tags)
		})
	})

	t.Run("BlobManager", func(t *testing.T) {
		config := &configs.DockerProxyConfig{
			Cache: configs.CacheConfig{
				Enabled:   true,
				Directory: suite.testTempDir,
			},
		}

		blobManager := dockerServices.NewBlobManager(config, suite.mockCache)

		t.Run("GetBlob", func(t *testing.T) {
			digest := "sha256:abcd1234"
			blobData := "mock blob data"

			suite.mockCache.EXPECT().
				Get(mock.Anything, digest).
				Return(io.NopCloser(strings.NewReader(blobData)), nil).
				Once()

			reader, err := blobManager.GetBlob(suite.ctx, digest)

			assert.NoError(t, err)
			assert.NotNil(t, reader)

			data, err := io.ReadAll(reader)
			assert.NoError(t, err)
			assert.Equal(t, blobData, string(data))
		})

		t.Run("StoreBlob", func(t *testing.T) {
			blobData := []byte("blob content")
			reader := strings.NewReader(string(blobData))

			suite.mockCache.EXPECT().
				Put(mock.Anything, mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("time.Duration")).
				Return(nil).
				Once()

			digest, err := blobManager.StoreBlob(suite.ctx, reader)

			assert.NoError(t, err)
			assert.NotEmpty(t, digest)
			assert.Contains(t, digest, "sha256:")
		})
	})

	t.Run("AuthManager", func(t *testing.T) {
		config := &configs.DockerProxyConfig{
			Auth: configs.DockerAuth{
				Enabled:  true,
				Username: "testuser",
				Password: "testpass",
			},
		}

		authManager := dockerServices.NewAuthManager(config)

		t.Run("GetAuthToken", func(t *testing.T) {
			token, err := authManager.GetAuthToken(suite.ctx, "library/nginx", []string{"pull"})

			assert.NoError(t, err)
			assert.NotEmpty(t, token)
		})

		t.Run("ValidateToken", func(t *testing.T) {
			// 유효한 토큰 테스트
			validToken := "Bearer valid-token-here"
			isValid := authManager.ValidateToken(suite.ctx, validToken)
			assert.True(t, isValid)

			// 무효한 토큰 테스트
			invalidToken := "Bearer invalid-token"
			isValid = authManager.ValidateToken(suite.ctx, invalidToken)
			assert.False(t, isValid)
		})
	})
}

// TestProxyServices 프록시 서비스 테스트
func TestProxyServices(t *testing.T) {
	suite := SetupServiceTestSuite(t)

	t.Run("BaseService", func(t *testing.T) {
		config := &configs.GlobalConfig{
			Cache: configs.CacheConfig{
				Enabled: true,
				TTL:     "1h",
			},
		}

		baseService := proxyServices.NewBaseService(config, suite.mockCache)

		t.Run("ValidateRequest", func(t *testing.T) {
			// 유효한 요청
			err := baseService.ValidateRequest(suite.ctx, "/valid/path", "GET")
			assert.NoError(t, err)

			// 무효한 요청 (보안상 문제가 있는 경로)
			err = baseService.ValidateRequest(suite.ctx, "/../../../etc/passwd", "GET")
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "invalid path")
		})

		t.Run("NormalizeURL", func(t *testing.T) {
			testCases := []struct {
				input    string
				expected string
			}{
				{"http://example.com//path//to//resource", "http://example.com/path/to/resource"},
				{"https://api.com/v1/../v2/data", "https://api.com/v2/data"},
				{"/simple//package//", "/simple/package/"},
			}

			for _, tc := range testCases {
				result := baseService.NormalizeURL(tc.input)
				assert.Equal(t, tc.expected, result)
			}
		})

		t.Run("CacheOperations", func(t *testing.T) {
			cacheKey := "test:key"
			cacheData := "test data"

			// Put 작업
			suite.mockCache.EXPECT().
				Put(mock.Anything, cacheKey, mock.Anything, mock.AnythingOfType("time.Duration")).
				Return(nil).
				Once()

			err := baseService.CacheData(suite.ctx, cacheKey, []byte(cacheData), time.Hour)
			assert.NoError(t, err)

			// Get 작업
			suite.mockCache.EXPECT().
				Get(mock.Anything, cacheKey).
				Return(io.NopCloser(strings.NewReader(cacheData)), nil).
				Once()

			reader, err := baseService.GetCachedData(suite.ctx, cacheKey)
			assert.NoError(t, err)
			assert.NotNil(t, reader)

			data, err := io.ReadAll(reader)
			assert.NoError(t, err)
			assert.Equal(t, cacheData, string(data))
		})
	})

	t.Run("ServiceFactory", func(t *testing.T) {
		factory := proxyServices.NewServiceFactory()

		t.Run("CreateService", func(t *testing.T) {
			// NPM 서비스 생성
			npmConfig := &configs.NpmProxyConfig{Enabled: true}
			npmService, err := factory.CreateService("npm", npmConfig, suite.mockCache)
			assert.NoError(t, err)
			assert.NotNil(t, npmService)

			// Maven 서비스 생성
			mavenConfig := &configs.MavenProxyConfig{Enabled: true}
			mavenService, err := factory.CreateService("maven", mavenConfig, suite.mockCache)
			assert.NoError(t, err)
			assert.NotNil(t, mavenService)

			// 지원되지 않는 타입
			_, err = factory.CreateService("unsupported", nil, suite.mockCache)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "unsupported service type")
		})

		t.Run("RegisterService", func(t *testing.T) {
			// 커스텀 서비스 등록
			customService := &proxyServices.BaseService{}
			err := factory.RegisterService("custom", func(config interface{}, cache interface{}) (interface{}, error) {
				return customService, nil
			})
			assert.NoError(t, err)

			// 등록된 서비스 생성
			service, err := factory.CreateService("custom", nil, suite.mockCache)
			assert.NoError(t, err)
			assert.Equal(t, customService, service)
		})
	})
}

// TestCacheIntegration 캐시 통합 테스트
func TestCacheIntegration(t *testing.T) {
	suite := SetupServiceTestSuite(t)

	t.Run("Cache Lifecycle", func(t *testing.T) {
		key := "integration:test:key"
		data := "integration test data"
		ttl := time.Minute

		// 1. 초기 상태 - 캐시 미스
		suite.mockCache.EXPECT().
			Get(mock.Anything, key).
			Return(nil, fmt.Errorf("cache miss")).
			Once()

		_, err := suite.mockCache.Get(suite.ctx, key)
		assert.Error(t, err)

		// 2. 데이터 저장
		suite.mockCache.EXPECT().
			Put(mock.Anything, key, mock.Anything, ttl).
			Return(nil).
			Once()

		err = suite.mockCache.Put(suite.ctx, key, strings.NewReader(data), ttl)
		assert.NoError(t, err)

		// 3. 캐시 히트
		suite.mockCache.EXPECT().
			Get(mock.Anything, key).
			Return(io.NopCloser(strings.NewReader(data)), nil).
			Once()

		reader, err := suite.mockCache.Get(suite.ctx, key)
		assert.NoError(t, err)
		assert.NotNil(t, reader)

		retrievedData, err := io.ReadAll(reader)
		assert.NoError(t, err)
		assert.Equal(t, data, string(retrievedData))

		// 4. 만료 후 캐시 미스
		suite.mockCache.EXPECT().
			Get(mock.Anything, key).
			Return(nil, fmt.Errorf("cache expired")).
			Once()

		_, err = suite.mockCache.Get(suite.ctx, key)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cache expired")
	})

	t.Run("Cache Performance", func(t *testing.T) {
		const numOperations = 100

		// 대량 쓰기 작업
		suite.mockCache.EXPECT().
			Put(mock.Anything, mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("time.Duration")).
			Return(nil).
			Times(numOperations)

		start := time.Now()
		for i := 0; i < numOperations; i++ {
			key := fmt.Sprintf("perf:test:%d", i)
			data := fmt.Sprintf("data_%d", i)
			err := suite.mockCache.Put(suite.ctx, key, strings.NewReader(data), time.Minute)
			assert.NoError(t, err)
		}
		writeDuration := time.Since(start)

		// 성능 검증 (실제 구현에서는 더 정확한 벤치마크 필요)
		assert.Less(t, writeDuration, time.Second, "Bulk write operations should complete within 1 second")

		// 대량 읽기 작업
		suite.mockCache.EXPECT().
			Get(mock.Anything, mock.AnythingOfType("string")).
			Return(io.NopCloser(strings.NewReader("cached_data")), nil).
			Times(numOperations)

		start = time.Now()
		for i := 0; i < numOperations; i++ {
			key := fmt.Sprintf("perf:test:%d", i)
			_, err := suite.mockCache.Get(suite.ctx, key)
			assert.NoError(t, err)
		}
		readDuration := time.Since(start)

		assert.Less(t, readDuration, time.Second, "Bulk read operations should complete within 1 second")
	})
}

// TestErrorHandling 에러 처리 테스트
func TestErrorHandling(t *testing.T) {
	suite := SetupServiceTestSuite(t)

	t.Run("Service Error Scenarios", func(t *testing.T) {
		config := &configs.PipProxyConfig{
			Enabled: true,
			Mirrors: []configs.PipMirror{
				{Name: "pypi", URL: "https://pypi.org"},
			},
		}

		packageService := pipServices.NewPackageService(config, suite.mockCache)

		t.Run("Network Error", func(t *testing.T) {
			// 캐시 미스
			suite.mockCache.EXPECT().
				Get(mock.Anything, mock.AnythingOfType("string")).
				Return(nil, fmt.Errorf("cache miss")).
				Once()

			// 네트워크 에러를 시뮬레이션하기 위해 잘못된 패키지명 사용
			_, err := packageService.GetPackageMetadata(suite.ctx, "", "")
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "invalid package name")
		})

		t.Run("Cache Error", func(t *testing.T) {
			// 캐시 에러 시뮬레이션
			suite.mockCache.EXPECT().
				Get(mock.Anything, mock.AnythingOfType("string")).
				Return(nil, fmt.Errorf("cache service unavailable")).
				Once()

			_, err := packageService.GetPackageMetadata(suite.ctx, "requests", "2.28.1")
			// 캐시 에러는 무시하고 upstream에서 가져와야 함
			assert.NoError(t, err)
		})

		t.Run("Context Cancellation", func(t *testing.T) {
			// 취소된 컨텍스트
			canceledCtx, cancel := context.WithCancel(suite.ctx)
			cancel()

			_, err := packageService.GetPackageMetadata(canceledCtx, "requests", "2.28.1")
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "context canceled")
		})

		t.Run("Timeout", func(t *testing.T) {
			// 짧은 타임아웃 컨텍스트
			timeoutCtx, cancel := context.WithTimeout(suite.ctx, 1*time.Millisecond)
			defer cancel()

			time.Sleep(2 * time.Millisecond) // 타임아웃을 확실히 발생시킴

			_, err := packageService.GetPackageMetadata(timeoutCtx, "requests", "2.28.1")
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "context deadline exceeded")
		})
	})
}
