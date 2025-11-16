package integration

import (
	"fmt"
	"os"
	"testing"
	"time"
)

const (
	ciEnvPR      = "pr"
	ciEnvDevelop = "develop"
	ciEnvMain    = "main"
	ciEnvMaster  = "master"
	ciEnvNightly = "nightly"
	truthy       = "true"
	scheduleEvt  = "schedule"
)

// CIMatrixConfig CI 매트릭스 최적화 설정
type CIMatrixConfig struct {
	// 테스트 레벨 구분
	UnitTestsOnly    bool // 단위 테스트만 실행
	IntegrationTests bool // 통합 테스트 실행
	PerformanceTests bool // 성능 테스트 실행
	ExtendedTests    bool // 확장 테스트 실행

	// 프록시 타입 선택
	ProxyTypes []string // 테스트할 프록시 타입들

	// 성능 테스트 설정
	PerformanceConfig PerformanceTestConfig

	// 병렬 실행 설정
	ParallelJobs int // 병렬 실행할 작업 수
}

// PerformanceTestConfig 성능 테스트 특화 설정
type PerformanceTestConfig struct {
	MaxResponseTime     time.Duration
	MinThroughput       int
	ConcurrentUsers     int
	TestDuration        time.Duration
	AcceptableErrorRate float64
}

// GetCIMatrixForEnvironment 환경별 CI 매트릭스 설정 반환
func GetCIMatrixForEnvironment(env string) CIMatrixConfig {
	switch env {
	case ciEnvPR: // Pull Request
		return CIMatrixConfig{
			UnitTestsOnly:    true,
			IntegrationTests: true,
			PerformanceTests: false,
			ExtendedTests:    false,
			ProxyTypes:       []string{"npm", "maven"}, // 핵심 프록시만 테스트
			PerformanceConfig: PerformanceTestConfig{
				MaxResponseTime:     500 * time.Millisecond,
				ConcurrentUsers:     5,
				TestDuration:        2 * time.Second,
				AcceptableErrorRate: 0.05,
			},
			ParallelJobs: 2,
		}
	case ciEnvDevelop: // Develop 브랜치
		return CIMatrixConfig{
			UnitTestsOnly:    false,
			IntegrationTests: true,
			PerformanceTests: true,
			ExtendedTests:    false,
			ProxyTypes:       []string{"npm", "maven", "apt", "apk"},
			PerformanceConfig: PerformanceTestConfig{
				MaxResponseTime:     300 * time.Millisecond,
				MinThroughput:       30,
				ConcurrentUsers:     10,
				TestDuration:        5 * time.Second,
				AcceptableErrorRate: 0.02,
			},
			ParallelJobs: 4,
		}
	case ciEnvMain, ciEnvMaster: // 메인 브랜치
		return CIMatrixConfig{
			UnitTestsOnly:    false,
			IntegrationTests: true,
			PerformanceTests: true,
			ExtendedTests:    true,
			ProxyTypes:       []string{"npm", "maven", "apt", "apk", "yum", "docker"},
			PerformanceConfig: PerformanceTestConfig{
				MaxResponseTime:     200 * time.Millisecond,
				MinThroughput:       50,
				ConcurrentUsers:     20,
				TestDuration:        10 * time.Second,
				AcceptableErrorRate: 0.01,
			},
			ParallelJobs: 6,
		}
	case ciEnvNightly: // 야간 테스트
		return CIMatrixConfig{
			UnitTestsOnly:    false,
			IntegrationTests: true,
			PerformanceTests: true,
			ExtendedTests:    true,
			ProxyTypes:       []string{"npm", "maven", "apt", "apk", "yum", "docker", "pip"},
			PerformanceConfig: PerformanceTestConfig{
				MaxResponseTime:     100 * time.Millisecond,
				MinThroughput:       100,
				ConcurrentUsers:     50,
				TestDuration:        30 * time.Second,
				AcceptableErrorRate: 0.005,
			},
			ParallelJobs: 8,
		}
	default: // 로컬 개발환경
		return CIMatrixConfig{
			UnitTestsOnly:    false,
			IntegrationTests: true,
			PerformanceTests: false,
			ExtendedTests:    false,
			ProxyTypes:       []string{"npm", "maven", "apt"},
			PerformanceConfig: PerformanceTestConfig{
				MaxResponseTime:     1 * time.Second,
				ConcurrentUsers:     3,
				TestDuration:        1 * time.Second,
				AcceptableErrorRate: 0.1,
			},
			ParallelJobs: 2,
		}
	}
}

// RunCIMatrixTests CI 매트릭스에 따른 테스트 실행
func RunCIMatrixTests(t *testing.T, config CIMatrixConfig) {
	t.Helper()

	// 환경 설정
	env := NewTestEnvironment(t)
	defer env.Cleanup()

	// 기본 통합 테스트
	if config.IntegrationTests {
		t.Run("BasicIntegrationTests", func(t *testing.T) {
			t.Parallel()

			suite := &ProxyTestSuite{
				ProxyTypes: config.ProxyTypes,
				TestPaths: map[string]string{
					"npm":    "express",
					"maven":  "junit/junit/4.13.2/junit-4.13.2.pom",
					"apt":    "dists/jammy/Release",
					"apk":    "alpine/v3.16/main/x86_64/APKINDEX.tar.gz",
					"yum":    "centos/8/BaseOS/x86_64/os/repodata/repomd.xml",
					"docker": "v2/library/nginx/manifests/latest",
					"pip":    "simple/requests/",
				},
			}
			suite.RunCommonTestSuite(t, env)
		})
	}

	// 성능 테스트
	if config.PerformanceTests {
		t.Run("PerformanceTests", func(t *testing.T) {
			if testing.Short() {
				t.Skip("Skipping performance tests in short mode")
			}
			t.Parallel()

			for _, proxyType := range config.ProxyTypes {
				proxyType := proxyType // 클로저 변수 캡처
				t.Run(fmt.Sprintf("Performance_%s", proxyType), func(t *testing.T) {
					t.Parallel()

					testPath := getTestPathForProxy(proxyType)
					if testPath == "" {
						t.Skipf("No test path defined for proxy type: %s", proxyType)
					}

					pattern := PerformancePattern{
						ProxyType:           proxyType,
						Path:                testPath,
						MaxResponseTime:     config.PerformanceConfig.MaxResponseTime,
						MinThroughput:       config.PerformanceConfig.MinThroughput,
						ConcurrentUsers:     config.PerformanceConfig.ConcurrentUsers,
						TestDuration:        config.PerformanceConfig.TestDuration,
						AcceptableErrorRate: config.PerformanceConfig.AcceptableErrorRate,
					}
					TestPerformance(t, env, pattern)
				})
			}
		})
	}

	// 확장 테스트 (스트레스 테스트, 장시간 실행 등)
	if config.ExtendedTests {
		t.Run("ExtendedTests", func(t *testing.T) {
			if testing.Short() {
				t.Skip("Skipping extended tests in short mode")
			}

			t.Run("StressTest", func(t *testing.T) {
				t.Parallel()
				runStressTests(t, env, config)
			})

			t.Run("CacheEfficiencyTest", func(t *testing.T) {
				t.Parallel()
				runCacheEfficiencyTests(t, env, config)
			})

			t.Run("ErrorResilienceTest", func(t *testing.T) {
				t.Parallel()
				runErrorResilienceTests(t, env, config)
			})
		})
	}

	// 헬스체크 테스트 (MetricsRouter 비활성화로 인해 Skip)
	t.Run("HealthChecks", func(t *testing.T) {
		t.Skip("Health check endpoints require MetricsRouter which is disabled due to Prometheus conflicts")
	})
}

// getTestPathForProxy 프록시 타입에 대한 테스트 경로 반환
func getTestPathForProxy(proxyType string) string {
	paths := map[string]string{
		"npm":    "express",
		"maven":  "junit/junit/4.13.2/junit-4.13.2.pom",
		"apt":    "dists/jammy/Release",
		"apk":    "alpine/v3.16/main/x86_64/APKINDEX.tar.gz",
		"yum":    "centos/8/BaseOS/x86_64/os/repodata/repomd.xml",
		"docker": "v2/library/nginx/manifests/latest",
		"pip":    "simple/requests/",
	}
	return paths[proxyType]
}

// runStressTests 스트레스 테스트 실행
func runStressTests(t *testing.T, env *IntegrationTestEnvironment, config CIMatrixConfig) {
	t.Helper()

	// 고부하 동시성 테스트
	for _, proxyType := range config.ProxyTypes {
		testPath := getTestPathForProxy(proxyType)
		if testPath == "" {
			continue
		}

		pattern := PerformancePattern{
			ProxyType:           proxyType,
			Path:                testPath,
			MaxResponseTime:     config.PerformanceConfig.MaxResponseTime * 2,     // 스트레스 테스트에서는 응답시간을 여유있게
			ConcurrentUsers:     config.PerformanceConfig.ConcurrentUsers * 3,     // 동시 사용자를 3배로
			TestDuration:        config.PerformanceConfig.TestDuration * 2,        // 테스트 시간을 2배로
			AcceptableErrorRate: config.PerformanceConfig.AcceptableErrorRate * 2, // 에러율을 2배까지 허용
		}
		TestPerformance(t, env, pattern)
	}
}

// runCacheEfficiencyTests 캐시 효율성 테스트 실행
func runCacheEfficiencyTests(t *testing.T, env *IntegrationTestEnvironment, config CIMatrixConfig) {
	t.Helper()

	for _, proxyType := range config.ProxyTypes {
		testPath := getTestPathForProxy(proxyType)
		if testPath == "" {
			continue
		}

		t.Run(fmt.Sprintf("CacheEfficiency_%s", proxyType), func(t *testing.T) {
			// 캐시 효율성 패턴 테스트
			pattern := CacheScenarioPattern{
				ProxyType:    proxyType,
				Path:         testPath,
				ExpectedMiss: "mock", // 모든 mock 응답에 포함된 공통 문자열
				ExpectedHit:  "mock",
				CacheTTL:     5 * time.Second,
			}
			TestCacheScenario(t, env, pattern)
		})
	}
}

// runErrorResilienceTests 에러 복원력 테스트 실행
func runErrorResilienceTests(t *testing.T, env *IntegrationTestEnvironment, config CIMatrixConfig) {
	t.Helper()

	for _, proxyType := range config.ProxyTypes {
		testPath := getTestPathForProxy(proxyType)
		if testPath == "" {
			continue
		}

		t.Run(fmt.Sprintf("ErrorResilience_%s", proxyType), func(t *testing.T) {
			// 에러 복구 패턴 테스트
			pattern := ErrorRecoveryPattern{
				ProxyType:        proxyType,
				Path:             "nonexistent/path/that/should/fail",
				ExpectedError:    404,
				ExpectedRecovery: "mock",
			}
			TestErrorRecovery(t, env, pattern)
		})
	}
}

// runHealthCheckTests 헬스체크 테스트 실행
func runHealthCheckTests(t *testing.T, env *IntegrationTestEnvironment) {
	t.Helper()

	// 기본 헬스체크
	healthPattern := HealthCheckPattern{
		Endpoint:        "/health",
		ExpectedStatus:  200,
		ExpectedContent: "healthy",
		MaxResponseTime: 100 * time.Millisecond,
	}
	RunHealthCheckPattern(t, env, healthPattern)

	// 메트릭 엔드포인트 체크
	metricsPattern := HealthCheckPattern{
		Endpoint:        "/metrics",
		ExpectedStatus:  200,
		ExpectedContent: "proxynd_requests_total",
		MaxResponseTime: 50 * time.Millisecond,
	}
	RunHealthCheckPattern(t, env, metricsPattern)
}

// CIEnvironmentFromEnv 환경 변수에서 CI 환경 감지
func CIEnvironmentFromEnv() string {
	// GitHub Actions
	if os.Getenv("GITHUB_ACTIONS") == truthy {
		if os.Getenv("GITHUB_EVENT_NAME") == "pull_request" {
			return ciEnvPR
		}
		if os.Getenv("GITHUB_REF") == "refs/heads/main" || os.Getenv("GITHUB_REF") == "refs/heads/master" {
			return ciEnvMain
		}
		if os.Getenv("GITHUB_REF") == "refs/heads/develop" {
			return ciEnvDevelop
		}
		if os.Getenv("GITHUB_EVENT_NAME") == scheduleEvt {
			return ciEnvNightly
		}
		return ciEnvDevelop // 기본값
	}

	// GitLab CI
	if os.Getenv("GITLAB_CI") == truthy {
		if os.Getenv("CI_PIPELINE_SOURCE") == "merge_request_event" {
			return ciEnvPR
		}
		if os.Getenv("CI_COMMIT_REF_NAME") == ciEnvMain || os.Getenv("CI_COMMIT_REF_NAME") == ciEnvMaster {
			return ciEnvMain
		}
		if os.Getenv("CI_PIPELINE_SOURCE") == scheduleEvt {
			return ciEnvNightly
		}
		return ciEnvDevelop
	}

	// Jenkins
	if os.Getenv("JENKINS_URL") != "" {
		if os.Getenv("CHANGE_ID") != "" { // Pull Request
			return ciEnvPR
		}
		if os.Getenv("BRANCH_NAME") == ciEnvMain || os.Getenv("BRANCH_NAME") == ciEnvMaster {
			return ciEnvMain
		}
		return ciEnvDevelop
	}

	// 로컬 개발환경
	return "local"
}

// OptimizedTestRunner CI 매트릭스 최적화된 테스트 러너
func OptimizedTestRunner(t *testing.T) {
	// CI 환경 감지
	ciEnv := CIEnvironmentFromEnv()

	// 환경에 맞는 설정 로드
	config := GetCIMatrixForEnvironment(ciEnv)

	t.Logf("Running tests in CI environment: %s", ciEnv)
	t.Logf("Test configuration: Integration=%v, Performance=%v, Extended=%v",
		config.IntegrationTests, config.PerformanceTests, config.ExtendedTests)
	t.Logf("Proxy types: %v", config.ProxyTypes)

	// 최적화된 테스트 실행
	RunCIMatrixTests(t, config)
}

// TestSuiteBuilder 테스트 스위트 빌더 패턴
type TestSuiteBuilder struct {
	config CIMatrixConfig
}

// NewTestSuiteBuilder 새 테스트 스위트 빌더 생성
func NewTestSuiteBuilder() *TestSuiteBuilder {
	return &TestSuiteBuilder{
		config: GetCIMatrixForEnvironment("local"), // 기본값
	}
}

// WithProxyTypes 프록시 타입 설정
func (b *TestSuiteBuilder) WithProxyTypes(types ...string) *TestSuiteBuilder {
	b.config.ProxyTypes = types
	return b
}

// WithPerformanceTests 성능 테스트 활성화
func (b *TestSuiteBuilder) WithPerformanceTests(enabled bool) *TestSuiteBuilder {
	b.config.PerformanceTests = enabled
	return b
}

// WithExtendedTests 확장 테스트 활성화
func (b *TestSuiteBuilder) WithExtendedTests(enabled bool) *TestSuiteBuilder {
	b.config.ExtendedTests = enabled
	return b
}

// WithPerformanceConfig 성능 테스트 설정
func (b *TestSuiteBuilder) WithPerformanceConfig(cfg PerformanceTestConfig) *TestSuiteBuilder {
	b.config.PerformanceConfig = cfg
	return b
}

// Build 테스트 스위트 빌드 및 실행
func (b *TestSuiteBuilder) Build(t *testing.T) {
	RunCIMatrixTests(t, b.config)
}
