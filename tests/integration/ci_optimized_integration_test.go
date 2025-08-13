package integration

import (
	"os"
	"testing"
	"time"
)

// TestCIOptimizedIntegration CI 최적화된 통합 테스트
// 환경별로 다른 테스트 매트릭스를 실행하여 CI 시간을 최적화
func TestCIOptimizedIntegration(t *testing.T) {
	OptimizedTestRunner(t)
}

// TestCustomCIMatrix 커스텀 CI 매트릭스 테스트 예제
func TestCustomCIMatrix(t *testing.T) {
	// 빌더 패턴을 사용한 커스텀 테스트 스위트
	NewTestSuiteBuilder().
		WithProxyTypes("npm", "maven"). // 핵심 프록시만 테스트
		WithPerformanceTests(true).
		WithExtendedTests(false).
		WithPerformanceConfig(PerformanceTestConfig{
			MaxResponseTime:     300 * time.Millisecond,
			MinThroughput:       25,
			ConcurrentUsers:     5,
			TestDuration:        3 * time.Second,
			AcceptableErrorRate: 0.03,
		}).
		Build(t)
}

// TestPullRequestMatrix Pull Request용 경량 테스트 매트릭스
func TestPullRequestMatrix(t *testing.T) {
	if os.Getenv("CI_ENVIRONMENT") != "pr" && os.Getenv("GITHUB_EVENT_NAME") != "pull_request" {
		t.Skip("Skipping PR tests outside of pull request context")
	}

	config := CIMatrixConfig{
		IntegrationTests: true,
		PerformanceTests: false, // PR에서는 성능 테스트 제외
		ExtendedTests:    false, // PR에서는 확장 테스트 제외
		ProxyTypes:       []string{"npm", "maven"}, // 핵심 프록시만 테스트
		PerformanceConfig: PerformanceTestConfig{
			MaxResponseTime:     1 * time.Second, // 관대한 설정
			ConcurrentUsers:     3,
			TestDuration:        1 * time.Second,
			AcceptableErrorRate: 0.1,
		},
		ParallelJobs: 2,
	}

	RunCIMatrixTests(t, config)
}

// TestDevelopBranchMatrix Develop 브랜치용 중간 수준 테스트 매트릭스
func TestDevelopBranchMatrix(t *testing.T) {
	if os.Getenv("GITHUB_REF") != "refs/heads/develop" && os.Getenv("CI_COMMIT_REF_NAME") != "develop" {
		t.Skip("Skipping develop tests outside of develop branch")
	}

	config := CIMatrixConfig{
		IntegrationTests: true,
		PerformanceTests: true,  // Develop에서는 성능 테스트 포함
		ExtendedTests:    false, // 확장 테스트는 여전히 제외
		ProxyTypes:       []string{"npm", "maven", "apt", "apk"},
		PerformanceConfig: PerformanceTestConfig{
			MaxResponseTime:     400 * time.Millisecond,
			MinThroughput:       20,
			ConcurrentUsers:     8,
			TestDuration:        4 * time.Second,
			AcceptableErrorRate: 0.03,
		},
		ParallelJobs: 4,
	}

	RunCIMatrixTests(t, config)
}

// TestMainBranchMatrix 메인 브랜치용 전체 테스트 매트릭스
func TestMainBranchMatrix(t *testing.T) {
	ref := os.Getenv("GITHUB_REF")
	commitRef := os.Getenv("CI_COMMIT_REF_NAME")
	
	if ref != "refs/heads/main" && ref != "refs/heads/master" && 
	   commitRef != "main" && commitRef != "master" {
		t.Skip("Skipping main branch tests outside of main/master branch")
	}

	config := CIMatrixConfig{
		IntegrationTests: true,
		PerformanceTests: true,
		ExtendedTests:    true, // 메인 브랜치에서는 모든 테스트 실행
		ProxyTypes:       []string{"npm", "maven", "apt", "apk", "yum", "docker"},
		PerformanceConfig: PerformanceTestConfig{
			MaxResponseTime:     200 * time.Millisecond, // 엄격한 성능 기준
			MinThroughput:       50,
			ConcurrentUsers:     15,
			TestDuration:        8 * time.Second,
			AcceptableErrorRate: 0.01, // 낮은 에러율 요구
		},
		ParallelJobs: 6,
	}

	RunCIMatrixTests(t, config)
}

// TestNightlyMatrix 야간 테스트용 완전한 테스트 매트릭스
func TestNightlyMatrix(t *testing.T) {
	if os.Getenv("GITHUB_EVENT_NAME") != "schedule" && os.Getenv("CI_PIPELINE_SOURCE") != "schedule" {
		t.Skip("Skipping nightly tests outside of scheduled context")
	}

	config := CIMatrixConfig{
		IntegrationTests: true,
		PerformanceTests: true,
		ExtendedTests:    true,
		ProxyTypes:       []string{"npm", "maven", "apt", "apk", "yum", "docker", "pip"},
		PerformanceConfig: PerformanceTestConfig{
			MaxResponseTime:     100 * time.Millisecond, // 매우 엄격한 성능 기준
			MinThroughput:       100,
			ConcurrentUsers:     30,
			TestDuration:        20 * time.Second,
			AcceptableErrorRate: 0.005, // 매우 낮은 에러율 요구
		},
		ParallelJobs: 8,
	}

	RunCIMatrixTests(t, config)
}

// TestLocalDevelopment 로컬 개발용 빠른 테스트
func TestLocalDevelopment(t *testing.T) {
	// CI 환경이 아닌 경우에만 실행
	if os.Getenv("CI") == "true" || os.Getenv("GITHUB_ACTIONS") == "true" || os.Getenv("GITLAB_CI") == "true" {
		t.Skip("Skipping local development tests in CI environment")
	}

	config := CIMatrixConfig{
		IntegrationTests: true,
		PerformanceTests: false, // 로컬에서는 성능 테스트 제외
		ExtendedTests:    false,
		ProxyTypes:       []string{"npm", "maven"}, // 최소한의 프록시만 테스트
		PerformanceConfig: PerformanceTestConfig{
			MaxResponseTime:     2 * time.Second, // 매우 관대한 설정
			ConcurrentUsers:     2,
			TestDuration:        1 * time.Second,
			AcceptableErrorRate: 0.2,
		},
		ParallelJobs: 2,
	}

	RunCIMatrixTests(t, config)
}

// TestSmokeTesting 스모크 테스트 - 기본 기능만 빠르게 검증
func TestSmokeTesting(t *testing.T) {
	env := NewTestEnvironment(t)
	defer env.Cleanup()

	// 각 프록시 타입의 기본 엔드포인트만 테스트
	smokeTests := []struct {
		proxyType string
		path      string
		expected  string
	}{
		{"npm", "express", "express"},
		{"maven", "junit/junit/4.13.2/junit-4.13.2.pom", "junit"},
		{"apt", "dists/jammy/Release", "Ubuntu"},
		{"apk", "alpine/v3.16/main/x86_64/APKINDEX.tar.gz", "mock APKINDEX"},
	}

	for _, test := range smokeTests {
		t.Run(test.proxyType, func(t *testing.T) {
			t.Parallel()
			
			resp, err := env.MakeRequest("GET", "/proxy/"+test.proxyType+"/"+test.path, nil)
			if err != nil {
				t.Fatalf("Smoke test failed for %s: %v", test.proxyType, err)
			}
			defer func() { _ = resp.Body.Close() }()

			if resp.StatusCode != 200 {
				t.Fatalf("Smoke test failed for %s: expected 200, got %d", test.proxyType, resp.StatusCode)
			}
			
			t.Logf("✓ Smoke test passed for %s proxy", test.proxyType)
		})
	}
}

// TestCriticalPathOnly 임계 경로만 테스트 - 최소 시간으로 핵심 기능 검증
func TestCriticalPathOnly(t *testing.T) {
	if !testing.Short() {
		t.Skip("Critical path tests only run in short mode (-test.short)")
	}

	env := NewTestEnvironment(t)
	defer env.Cleanup()

	// 가장 중요한 기능들만 테스트
	criticalTests := []struct {
		name     string
		endpoint string
		timeout  time.Duration
	}{
		{"Health Check", "/health", 100 * time.Millisecond},
		{"NPM Express", "/proxy/npm/express", 500 * time.Millisecond},
		{"Maven JUnit", "/proxy/maven/junit/junit/4.13.2/junit-4.13.2.pom", 500 * time.Millisecond},
	}

	for _, test := range criticalTests {
		t.Run(test.name, func(t *testing.T) {
			start := time.Now()
			resp, err := env.MakeRequest("GET", test.endpoint, nil)
			duration := time.Since(start)
			
			if err != nil {
				t.Fatalf("Critical path test failed for %s: %v", test.name, err)
			}
			defer func() { _ = resp.Body.Close() }()

			if resp.StatusCode != 200 {
				t.Fatalf("Critical path test failed for %s: expected 200, got %d", test.name, resp.StatusCode)
			}

			if duration > test.timeout {
				t.Fatalf("Critical path test too slow for %s: %v > %v", test.name, duration, test.timeout)
			}
			
			t.Logf("✓ Critical path test passed for %s in %v", test.name, duration)
		})
	}
}

// TestRegressionSuite 회귀 테스트 스위트
func TestRegressionSuite(t *testing.T) {
	// 환경 변수로 회귀 테스트 활성화 여부 결정
	if os.Getenv("RUN_REGRESSION_TESTS") != "true" {
		t.Skip("Regression tests disabled (set RUN_REGRESSION_TESTS=true to enable)")
	}

	env := NewTestEnvironment(t)
	defer env.Cleanup()

	// 과거에 발생했던 버그들에 대한 회귀 테스트
	t.Run("RegressionTests", func(t *testing.T) {
		// 예: 특정 NPM 패키지 요청 시 타임아웃 이슈 (회귀 방지)
		t.Run("NPMTimeoutRegression", func(t *testing.T) {
			pattern := PerformancePattern{
				ProxyType:       "npm",
				Path:            "express",
				MaxResponseTime: 1 * time.Second, // 타임아웃 회귀 방지
				ConcurrentUsers: 1,
				TestDuration:    1 * time.Second,
			}
			TestPerformance(t, env, pattern)
		})

		// 예: 캐시 일관성 문제 회귀 방지
		t.Run("CacheConsistencyRegression", func(t *testing.T) {
			pattern := CacheScenarioPattern{
				ProxyType:    "maven",
				Path:         "junit/junit/4.13.2/junit-4.13.2.pom",
				ExpectedMiss: "junit",
				ExpectedHit:  "junit",
				CacheTTL:     time.Second,
			}
			TestCacheScenario(t, env, pattern)
		})

		// 예: 동시성 처리 오류 회귀 방지
		t.Run("ConcurrencyRegression", func(t *testing.T) {
			pattern := PerformancePattern{
				ProxyType:           "apt",
				Path:                "dists/jammy/Release",
				ConcurrentUsers:     10,
				TestDuration:        2 * time.Second,
				AcceptableErrorRate: 0.0, // 동시성 오류는 허용 안됨
			}
			TestPerformance(t, env, pattern)
		})
	})
}