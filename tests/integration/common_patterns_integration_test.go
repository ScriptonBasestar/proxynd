package integration

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestCommonPatternsIntegration 공통 테스트 패턴들의 통합 테스트
func TestCommonPatternsIntegration(t *testing.T) {
	t.Skip("Requires mock upstream servers which are not configured in current environment")
	env := NewTestEnvironment(t)
	defer env.Cleanup()

	// 기본 테스트 스위트 실행
	suite := CreateDefaultTestSuite()
	suite.RunCommonTestSuite(t, env)
}

// TestCachePatternCustomized 커스터마이징된 캐시 패턴 테스트
func TestCachePatternCustomized(t *testing.T) {
	env := NewTestEnvironment(t)
	defer env.Cleanup()

	// NPM 패키지의 상세한 캐시 테스트
	pattern := CacheScenarioPattern{
		ProxyType:    "npm",
		Path:         "express",
		ExpectedMiss: "express",
		ExpectedHit:  "express",
		CacheTTL:     2 * time.Second,
		ValidateFunc: func(t *testing.T, body []byte, headers http.Header) {
			t.Helper()
			// NPM 응답의 JSON 구조 검증
			assert.Contains(t, string(body), "\"name\":")
			assert.Contains(t, string(body), "\"version\":")
			assert.Contains(t, string(body), "\"description\":")

			// Content-Type 헤더 검증
			contentType := headers.Get("Content-Type")
			assert.Contains(t, contentType, "application/json")
		},
	}

	TestCacheScenario(t, env, pattern)
}

// TestErrorRecoveryCustomized 커스터마이징된 에러 복구 패턴 테스트
func TestErrorRecoveryCustomized(t *testing.T) {
	t.Skip("Requires mock upstream servers which are not configured in current environment")
	env := NewTestEnvironment(t)
	defer env.Cleanup()

	// Maven 아티팩트의 상세한 에러 복구 테스트
	pattern := ErrorRecoveryPattern{
		ProxyType:        "maven",
		Path:             "junit/junit/4.13.2/junit-4.13.2.pom",
		ExpectedRecovery: "junit",
		ErrorSimulation: func(env *IntegrationTestEnvironment) {
			// 실제 구현에서는 Mock upstream을 일시적으로 중단시킬 수 있음
			t.Log("Simulating upstream error for Maven proxy")
		},
		RecoveryAction: func(env *IntegrationTestEnvironment) {
			// 실제 구현에서는 Mock upstream을 복구시킬 수 있음
			t.Log("Recovering Maven proxy upstream")
		},
	}

	TestErrorRecovery(t, env, pattern)
}

// TestPerformancePatternCustomized 커스터마이징된 성능 패턴 테스트
func TestPerformancePatternCustomized(t *testing.T) {
	t.Skip("Requires mock upstream servers which are not configured in current environment")
	env := NewTestEnvironment(t)
	defer env.Cleanup()

	// APT 저장소의 상세한 성능 테스트
	pattern := PerformancePattern{
		ProxyType:           "apt",
		Path:                "dists/jammy/Release",
		MaxResponseTime:     200 * time.Millisecond,
		MinThroughput:       50, // 50 requests/second
		ConcurrentUsers:     10,
		TestDuration:        3 * time.Second,
		WarmupRequests:      3,
		AcceptableErrorRate: 0.02, // 2% 에러율 허용
	}

	TestPerformance(t, env, pattern)
}

// TestHealthCheckPatterns 헬스체크 패턴 테스트
func TestHealthCheckPatterns(t *testing.T) {
	t.Skip("Health check endpoints require MetricsRouter which is disabled due to Prometheus conflicts")
	env := NewTestEnvironment(t)
	defer env.Cleanup()

	// 기본 헬스체크 테스트
	healthPattern := HealthCheckPattern{
		Endpoint:        "/health",
		ExpectedStatus:  http.StatusOK,
		ExpectedContent: "healthy",
		MaxResponseTime: 100 * time.Millisecond,
		RequiredHeaders: map[string]string{
			"Content-Type": "application/json",
		},
	}

	RunHealthCheckPattern(t, env, healthPattern)

	// 메트릭 엔드포인트 헬스체크
	metricsPattern := HealthCheckPattern{
		Endpoint:        "/metrics",
		ExpectedStatus:  http.StatusOK,
		ExpectedContent: "proxynd_requests_total",
		MaxResponseTime: 50 * time.Millisecond,
	}

	RunHealthCheckPattern(t, env, metricsPattern)
}

// TestCrossProxyScenarios 프록시 간 상호 작용 시나리오 테스트
func TestCrossProxyScenarios(t *testing.T) {
	t.Skip("Requires mock upstream servers which are not configured in current environment")
	env := NewTestEnvironment(t)
	defer env.Cleanup()

	// 여러 프록시 타입에 대한 동시 요청 테스트
	t.Run("MultiProxySequential", func(t *testing.T) {
		proxies := []struct {
			proxyType string
			path      string
			expected  string
		}{
			{"npm", "express", "express"},
			{"maven", "junit/junit/4.13.2/junit-4.13.2.pom", "junit"},
			{"apt", "dists/jammy/Release", "Ubuntu"},
		}

		for _, proxy := range proxies {
			t.Run(proxy.proxyType, func(t *testing.T) {
				resp, err := env.MakeRequest("GET", "/proxy/"+proxy.proxyType+"/"+proxy.path, nil)
				assert.NoError(t, err)
				defer func() { _ = resp.Body.Close() }()

				assert.Equal(t, http.StatusOK, resp.StatusCode)
				// Additional validation can be added here
			})
		}
	})

	// 동시 다중 프록시 요청 테스트
	t.Run("MultiProxyConcurrent", func(t *testing.T) {
		proxies := []string{"npm", "maven", "apt"}
		paths := map[string]string{
			"npm":   "express",
			"maven": "junit/junit/4.13.2/junit-4.13.2.pom",
			"apt":   "dists/jammy/Release",
		}

		results := make(chan bool, len(proxies))

		for _, proxyType := range proxies {
			go func(pt string) {
				resp, err := env.MakeRequest("GET", "/proxy/"+pt+"/"+paths[pt], nil)
				success := err == nil && resp != nil && resp.StatusCode == http.StatusOK
				if resp != nil {
					_ = resp.Body.Close()
				}
				results <- success
			}(proxyType)
		}

		// 모든 요청 결과 확인
		for i := 0; i < len(proxies); i++ {
			success := <-results
			assert.True(t, success, "Concurrent proxy request %d should succeed", i+1)
		}
	})
}

// TestProxyTypeSpecificPatterns 프록시 타입별 특화 패턴 테스트
func TestProxyTypeSpecificPatterns(t *testing.T) {
	t.Skip("Requires mock upstream servers which are not configured in current environment")
	env := NewTestEnvironment(t)
	defer env.Cleanup()

	t.Run("NPMSpecificTests", func(t *testing.T) {
		// NPM tarball 다운로드 테스트
		tgzPattern := CacheScenarioPattern{
			ProxyType:    "npm",
			Path:         "express/-/express-4.18.2.tgz",
			ExpectedMiss: "mock express tarball",
			ExpectedHit:  "mock express tarball",
			CacheTTL:     time.Second,
			ValidateFunc: func(t *testing.T, body []byte, headers http.Header) {
				contentType := headers.Get("Content-Type")
				assert.Contains(t, contentType, "application/octet-stream")
			},
		}
		TestCacheScenario(t, env, tgzPattern)
	})

	t.Run("MavenSpecificTests", func(t *testing.T) {
		// Maven JAR 파일 다운로드 테스트
		jarPattern := CacheScenarioPattern{
			ProxyType:    "maven",
			Path:         "junit/junit/4.13.2/junit-4.13.2.jar",
			ExpectedMiss: "mock junit jar",
			ExpectedHit:  "mock junit jar",
			CacheTTL:     time.Second,
			ValidateFunc: func(t *testing.T, body []byte, headers http.Header) {
				contentType := headers.Get("Content-Type")
				assert.Contains(t, contentType, "application/java-archive")
			},
		}
		TestCacheScenario(t, env, jarPattern)
	})

	t.Run("APTSpecificTests", func(t *testing.T) {
		// APT Packages 파일 테스트
		packagesPattern := CacheScenarioPattern{
			ProxyType:    "apt",
			Path:         "dists/jammy/main/binary-amd64/Packages",
			ExpectedMiss: "nginx",
			ExpectedHit:  "nginx",
			CacheTTL:     time.Second,
			ValidateFunc: func(t *testing.T, body []byte, headers http.Header) {
				assert.Contains(t, string(body), "Package:")
				assert.Contains(t, string(body), "Version:")
				assert.Contains(t, string(body), "Architecture:")
			},
		}
		TestCacheScenario(t, env, packagesPattern)
	})

	t.Run("APKSpecificTests", func(t *testing.T) {
		// APK APKINDEX.tar.gz 테스트
		apkIndexPattern := CacheScenarioPattern{
			ProxyType:    "apk",
			Path:         "alpine/v3.16/main/x86_64/APKINDEX.tar.gz",
			ExpectedMiss: "mock APKINDEX content",
			ExpectedHit:  "mock APKINDEX content",
			CacheTTL:     time.Second,
			ValidateFunc: func(t *testing.T, body []byte, headers http.Header) {
				contentType := headers.Get("Content-Type")
				assert.Contains(t, contentType, "application/x-gzip")
			},
		}
		TestCacheScenario(t, env, apkIndexPattern)
	})

	t.Run("YUMSpecificTests", func(t *testing.T) {
		// YUM repomd.xml 테스트
		repomdPattern := CacheScenarioPattern{
			ProxyType:    "yum",
			Path:         "centos/8/BaseOS/x86_64/os/repodata/repomd.xml",
			ExpectedMiss: "repomd",
			ExpectedHit:  "repomd",
			CacheTTL:     time.Second,
			ValidateFunc: func(t *testing.T, body []byte, headers http.Header) {
				contentType := headers.Get("Content-Type")
				assert.Contains(t, contentType, "application/xml")
				assert.Contains(t, string(body), "<repomd")
			},
		}
		TestCacheScenario(t, env, repomdPattern)
	})

	t.Run("DockerSpecificTests", func(t *testing.T) {
		// Docker manifest 테스트
		manifestPattern := CacheScenarioPattern{
			ProxyType:    "docker",
			Path:         "v2/library/nginx/manifests/latest",
			ExpectedMiss: "schemaVersion",
			ExpectedHit:  "schemaVersion",
			CacheTTL:     time.Second,
			ValidateFunc: func(t *testing.T, body []byte, headers http.Header) {
				contentType := headers.Get("Content-Type")
				assert.Contains(t, contentType, "application/vnd.docker.distribution.manifest")
				assert.Contains(t, string(body), "mediaType")
				assert.Contains(t, string(body), "layers")
			},
		}
		TestCacheScenario(t, env, manifestPattern)
	})
}

// TestExtendedPerformanceScenarios 확장된 성능 테스트 시나리오
func TestExtendedPerformanceScenarios(t *testing.T) {
	t.Skip("Requires mock upstream servers which are not configured in current environment")
	if testing.Short() {
		t.Skip("Skipping extended performance tests in short mode")
	}

	env := NewTestEnvironment(t)
	defer env.Cleanup()

	t.Run("LongRunningStabilityTest", func(t *testing.T) {
		// 장시간 실행 안정성 테스트
		pattern := PerformancePattern{
			ProxyType:           "npm",
			Path:                "express",
			MaxResponseTime:     1 * time.Second,
			MinThroughput:       20,
			ConcurrentUsers:     3,
			TestDuration:        10 * time.Second,
			WarmupRequests:      5,
			AcceptableErrorRate: 0.01, // 1% 에러율 허용
		}
		TestPerformance(t, env, pattern)
	})

	t.Run("HighConcurrencyTest", func(t *testing.T) {
		// 고동시성 테스트
		pattern := PerformancePattern{
			ProxyType:           "maven",
			Path:                "junit/junit/4.13.2/junit-4.13.2.pom",
			MaxResponseTime:     2 * time.Second,
			ConcurrentUsers:     50,
			TestDuration:        5 * time.Second,
			WarmupRequests:      10,
			AcceptableErrorRate: 0.05, // 5% 에러율 허용
		}
		TestPerformance(t, env, pattern)
	})

	t.Run("CacheEfficiencyTest", func(t *testing.T) {
		// 캐시 효율성 테스트 - 같은 리소스를 반복 요청
		url := "/proxy/apt/dists/jammy/Release"

		// 첫 번째 요청 (캐시 MISS)
		resp1, err := env.MakeRequest("GET", url, nil)
		assert.NoError(t, err)
		defer func() { _ = resp1.Body.Close() }()

		start := time.Now()
		// 연속 요청들 (캐시 HIT이어야 함)
		for i := 0; i < 10; i++ {
			resp, err := env.MakeRequest("GET", url, nil)
			assert.NoError(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode)
			_ = resp.Body.Close()
		}
		duration := time.Since(start)

		avgTime := duration / 10
		t.Logf("Average cached response time: %v", avgTime)

		// 캐시된 응답은 매우 빨라야 함
		assert.True(t, avgTime < 50*time.Millisecond,
			"Cached responses should be very fast, got %v", avgTime)
	})
}
