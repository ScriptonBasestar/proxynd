package unit

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"

	// cacheMocks "proxynd/cache/mocks"
	"proxynd/internal/app"
	"proxynd/internal/config"
	apkMocks "proxynd/internal/services/apk/mocks"
	dockerMocks "proxynd/internal/services/docker/mocks"
	pipMocks "proxynd/internal/services/pip/mocks"
	yumMocks "proxynd/internal/services/yum/mocks"
)

// TestUtilities 테스트 유틸리티 모음
type TestUtilities struct {
	t *testing.T
}

// NewTestUtilities 새 테스트 유틸리티 생성
func NewTestUtilities(t *testing.T) *TestUtilities {
	return &TestUtilities{t: t}
}

// MockFactory Mock 객체 팩토리
type MockFactory struct {
	t *testing.T
}

// NewMockFactory Mock 팩토리 생성
func NewMockFactory(t *testing.T) *MockFactory {
	return &MockFactory{t: t}
}

// CreateMockCache 캐시 Mock 생성
// func (mf *MockFactory) CreateMockCache() *cacheMocks.MockCache {
// 	return cacheMocks.NewMockCache(mf.t)
// }

// CreateMockPIPService PIP 서비스 Mock 생성
func (mf *MockFactory) CreateMockPIPService() *pipMocks.MockPackageService {
	return pipMocks.NewMockPackageService(mf.t)
}

// CreateMockDockerRegistry Docker 레지스트리 Mock 생성
func (mf *MockFactory) CreateMockDockerRegistry() *dockerMocks.MockRegistryService {
	return dockerMocks.NewMockRegistryService(mf.t)
}

// CreateMockDockerBlob Docker Blob Mock 생성
func (mf *MockFactory) CreateMockDockerBlob() *dockerMocks.MockBlobManager {
	return dockerMocks.NewMockBlobManager(mf.t)
}

// CreateMockYUMService YUM 서비스 Mock 생성
func (mf *MockFactory) CreateMockYUMService() *yumMocks.MockRepositoryService {
	return yumMocks.NewMockRepositoryService(mf.t)
}

// CreateMockAPKService APK 서비스 Mock 생성
func (mf *MockFactory) CreateMockAPKService() *apkMocks.MockPackageService {
	return apkMocks.NewMockPackageService(mf.t)
}

// HTTPTestHelper HTTP 테스트 헬퍼
type HTTPTestHelper struct {
	t   *testing.T
	app *fiber.App
}

// NewHTTPTestHelper HTTP 테스트 헬퍼 생성
func NewHTTPTestHelper(t *testing.T) *HTTPTestHelper {
	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return c.Status(500).JSON(fiber.Map{
				"error": err.Error(),
			})
		},
	})

	return &HTTPTestHelper{
		t:   t,
		app: app,
	}
}

// CreateFiberContext Fiber 컨텍스트 생성
func (h *HTTPTestHelper) CreateFiberContext(method, path string, body io.Reader) *fiber.Ctx {
	var req *http.Request
	if body != nil {
		req = httptest.NewRequest(method, path, body)
	} else {
		req = httptest.NewRequest(method, path, nil)
	}

	ctx := h.app.AcquireCtx(&fiber.DefaultCtx{})
	ctx.Request().SetRequestURI(path)
	ctx.Request().Header.SetMethod(method)

	if body != nil {
		if bodyBytes, err := io.ReadAll(body); err == nil {
			ctx.Request().SetBody(bodyBytes)
		}
	}

	return ctx
}

// ReleaseFiberContext Fiber 컨텍스트 해제
func (h *HTTPTestHelper) ReleaseFiberContext(ctx *fiber.Ctx) {
	h.app.ReleaseCtx(ctx)
}

// AssertHTTPResponse HTTP 응답 검증
func (h *HTTPTestHelper) AssertHTTPResponse(ctx *fiber.Ctx, expectedStatus int, expectedBodyContains ...string) {
	assert.Equal(h.t, expectedStatus, ctx.Response().StatusCode())

	body := string(ctx.Response().Body())
	for _, expected := range expectedBodyContains {
		assert.Contains(h.t, body, expected)
	}
}

// CacheTestHelper 캐시 테스트 헬퍼
type CacheTestHelper struct {
	t *testing.T
	// mockCache *cacheMocks.MockCache
}

// NewCacheTestHelper 캐시 테스트 헬퍼 생성
// func NewCacheTestHelper(t *testing.T, mockCache *cacheMocks.MockCache) *CacheTestHelper {
// 	return &CacheTestHelper{
// 		t:         t,
// 		mockCache: mockCache,
// 	}
// }

// ExpectCacheHit 캐시 히트 설정
// func (ch *CacheTestHelper) ExpectCacheHit(key string, data string) {
// 	ch.mockCache.EXPECT().
// 		Get(mock.Anything, key).
// 		Return(io.NopCloser(strings.NewReader(data)), nil).
// 		Once()
// }

// ExpectCacheMiss 캐시 미스 설정
// func (ch *CacheTestHelper) ExpectCacheMiss(key string) {
// 	ch.mockCache.EXPECT().
// 		Get(mock.Anything, key).
// 		Return(nil, fmt.Errorf("cache miss")).
// 		Once()
// }

// ExpectCachePut 캐시 저장 설정
// func (ch *CacheTestHelper) ExpectCachePut(key string) {
// 	ch.mockCache.EXPECT().
// 		Put(mock.Anything, key, mock.Anything, mock.AnythingOfType("time.Duration")).
// 		Return(nil).
// 		Once()
// }

// ExpectCacheMultipleOps 여러 캐시 작업 설정
// func (ch *CacheTestHelper) ExpectCacheMultipleOps(ops []CacheOperation) {
// 	for _, op := range ops {
// 		switch op.Type {
// 		case "get_hit":
// 			ch.ExpectCacheHit(op.Key, op.Data)
// 		case "get_miss":
// 			ch.ExpectCacheMiss(op.Key)
// 		case "put":
// 			ch.ExpectCachePut(op.Key)
// 		}
// 	}
// }

// CacheOperation 캐시 작업 정의
// type CacheOperation struct {
// 	Type string // "get_hit", "get_miss", "put"
// 	Key  string
// 	Data string
// }

// ConfigTestHelper 설정 테스트 헬퍼
type ConfigTestHelper struct {
	t *testing.T
}

// NewConfigTestHelper 설정 테스트 헬퍼 생성
func NewConfigTestHelper(t *testing.T) *ConfigTestHelper {
	return &ConfigTestHelper{t: t}
}

// CreatePIPConfig PIP 설정 생성
func (ch *ConfigTestHelper) CreatePIPConfig() *config.PipProxyConfig {
	return &config.PipProxyConfig{
		Enabled: true,
		Mirrors: []config.PipMirror{
			{
				Name:    "pypi",
				URL:     "https://pypi.org",
				Timeout: "30s",
			},
		},
		Cache: config.CacheConfig{
			Enabled: true,
			TTL:     "1h",
		},
	}
}

// CreateDockerConfig Docker 설정 생성
func (ch *ConfigTestHelper) CreateDockerConfig() *config.DockerProxyConfig {
	return &config.DockerProxyConfig{
		Enabled: true,
		Registries: []config.DockerRegistry{
			{
				Name:    "dockerhub",
				URL:     "https://registry-1.docker.io",
				Timeout: "30s",
			},
		},
		Cache: config.CacheConfig{
			Enabled: true,
			TTL:     "1h",
		},
	}
}

// CreateYUMConfig YUM 설정 생성
func (ch *ConfigTestHelper) CreateYUMConfig() *config.YumProxyConfig {
	return &config.YumProxyConfig{
		Enabled: true,
		Repositories: []config.YumRepository{
			{
				Name:    "centos",
				BaseURL: "http://mirror.centos.org/centos",
				Timeout: "30s",
			},
		},
		Cache: config.CacheConfig{
			Enabled: true,
			TTL:     "1h",
		},
	}
}

// CreateAPKConfig APK 설정 생성
func (ch *ConfigTestHelper) CreateAPKConfig() *config.ApkProxyConfig {
	return &config.ApkProxyConfig{
		Enabled: true,
		Repositories: []config.ApkRepository{
			{
				Name:    "alpine",
				BaseURL: "http://dl-cdn.alpinelinux.org/alpine",
				Timeout: "30s",
			},
		},
		Cache: config.CacheConfig{
			Enabled: true,
			TTL:     "1h",
		},
	}
}

// CreateAppConfig 앱 설정 생성
func (ch *ConfigTestHelper) CreateAppConfig(tempDir string) *app.Config {
	return &app.Config{
		Port:        "8080",
		Version:     "test",
		StorageDir:  tempDir,
		ConfigDir:   tempDir,
		CacheMaxAge: time.Hour,
	}
}

// PerformanceTestHelper 성능 테스트 헬퍼
type PerformanceTestHelper struct {
	t *testing.T
}

// NewPerformanceTestHelper 성능 테스트 헬퍼 생성
func NewPerformanceTestHelper(t *testing.T) *PerformanceTestHelper {
	return &PerformanceTestHelper{t: t}
}

// BenchmarkFunction 함수 벤치마크
func (ph *PerformanceTestHelper) BenchmarkFunction(name string, iterations int, fn func()) time.Duration {
	start := time.Now()

	for i := 0; i < iterations; i++ {
		fn()
	}

	duration := time.Since(start)
	avgDuration := duration / time.Duration(iterations)

	ph.t.Logf("Benchmark %s: %d iterations, total: %v, avg: %v",
		name, iterations, duration, avgDuration)

	return avgDuration
}

// AssertPerformance 성능 검증
func (ph *PerformanceTestHelper) AssertPerformance(name string, actual time.Duration, maxExpected time.Duration) {
	assert.LessOrEqual(ph.t, actual, maxExpected,
		"Performance test %s failed: actual %v > expected %v", name, actual, maxExpected)
}

// ConcurrentTestHelper 동시성 테스트 헬퍼
type ConcurrentTestHelper struct {
	t *testing.T
}

// NewConcurrentTestHelper 동시성 테스트 헬퍼 생성
func NewConcurrentTestHelper(t *testing.T) *ConcurrentTestHelper {
	return &ConcurrentTestHelper{t: t}
}

// RunConcurrent 동시 실행
func (ch *ConcurrentTestHelper) RunConcurrent(numGoroutines int, fn func(int) error) []error {
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(index int) {
			errors <- fn(index)
		}(i)
	}

	var results []error
	for i := 0; i < numGoroutines; i++ {
		if err := <-errors; err != nil {
			results = append(results, err)
		}
	}

	return results
}

// AssertNoConcurrentErrors 동시성 에러 검증
func (ch *ConcurrentTestHelper) AssertNoConcurrentErrors(errors []error) {
	for _, err := range errors {
		assert.NoError(ch.t, err)
	}
}

// PropertyTestHelper 속성 기반 테스트 헬퍼
type PropertyTestHelper struct {
	t *testing.T
}

// NewPropertyTestHelper 속성 기반 테스트 헬퍼 생성
func NewPropertyTestHelper(t *testing.T) *PropertyTestHelper {
	return &PropertyTestHelper{t: t}
}

// TestProperty 속성 테스트
func (ph *PropertyTestHelper) TestProperty(name string, testCases []PropertyTestCase) {
	for i, tc := range testCases {
		ph.t.Run(fmt.Sprintf("%s_case_%d", name, i), func(t *testing.T) {
			result := tc.Function(tc.Input)
			tc.Assertion(t, tc.Input, result)
		})
	}
}

// PropertyTestCase 속성 테스트 케이스
type PropertyTestCase struct {
	Input     interface{}
	Function  func(interface{}) interface{}
	Assertion func(*testing.T, interface{}, interface{})
}

// CommonAssertions 공통 검증 함수들
type CommonAssertions struct {
	t *testing.T
}

// NewCommonAssertions 공통 검증 생성
func NewCommonAssertions(t *testing.T) *CommonAssertions {
	return &CommonAssertions{t: t}
}

// AssertValidCacheKey 유효한 캐시 키 검증
func (ca *CommonAssertions) AssertValidCacheKey(key string) {
	assert.NotEmpty(ca.t, key, "Cache key should not be empty")
	assert.NotContains(ca.t, key, " ", "Cache key should not contain spaces")
	assert.NotContains(ca.t, key, "\n", "Cache key should not contain newlines")
}

// AssertValidURL 유효한 URL 검증
func (ca *CommonAssertions) AssertValidURL(url string) {
	assert.NotEmpty(ca.t, url, "URL should not be empty")
	assert.True(ca.t, strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://"),
		"URL should start with http:// or https://")
}

// AssertJSONResponse JSON 응답 검증
func (ca *CommonAssertions) AssertJSONResponse(body []byte) {
	assert.True(ca.t, len(body) > 0, "Response body should not be empty")
	assert.True(ca.t, strings.HasPrefix(string(body), "{") || strings.HasPrefix(string(body), "["),
		"Response should be valid JSON")
}

// AssertHTMLResponse HTML 응답 검증
func (ca *CommonAssertions) AssertHTMLResponse(body []byte) {
	bodyStr := string(body)
	assert.True(ca.t, len(body) > 0, "Response body should not be empty")
	assert.True(ca.t, strings.Contains(bodyStr, "<html>") || strings.Contains(bodyStr, "<!DOCTYPE"),
		"Response should be valid HTML")
}

// TestDataGenerator 테스트 데이터 생성기
type TestDataGenerator struct {
	t *testing.T
}

// NewTestDataGenerator 테스트 데이터 생성기 생성
func NewTestDataGenerator(t *testing.T) *TestDataGenerator {
	return &TestDataGenerator{t: t}
}

// GeneratePackageNames 패키지명 생성
func (tdg *TestDataGenerator) GeneratePackageNames(count int) []string {
	names := make([]string, count)
	for i := 0; i < count; i++ {
		names[i] = fmt.Sprintf("test-package-%d", i)
	}
	return names
}

// GenerateVersions 버전 생성
func (tdg *TestDataGenerator) GenerateVersions(count int) []string {
	versions := make([]string, count)
	for i := 0; i < count; i++ {
		versions[i] = fmt.Sprintf("1.%d.0", i)
	}
	return versions
}

// GenerateCacheKeys 캐시 키 생성
func (tdg *TestDataGenerator) GenerateCacheKeys(prefix string, count int) []string {
	keys := make([]string, count)
	for i := 0; i < count; i++ {
		keys[i] = fmt.Sprintf("%s:key:%d", prefix, i)
	}
	return keys
}

// GenerateTestData 테스트 데이터 생성
func (tdg *TestDataGenerator) GenerateTestData(size int) []byte {
	data := make([]byte, size)
	for i := 0; i < size; i++ {
		data[i] = byte(i % 256)
	}
	return data
}

// ErrorTestHelper 에러 테스트 헬퍼
type ErrorTestHelper struct {
	t *testing.T
}

// NewErrorTestHelper 에러 테스트 헬퍼 생성
func NewErrorTestHelper(t *testing.T) *ErrorTestHelper {
	return &ErrorTestHelper{t: t}
}

// AssertErrorContains 에러 메시지 포함 검증
func (eth *ErrorTestHelper) AssertErrorContains(err error, expectedMessages ...string) {
	assert.Error(eth.t, err, "Expected an error")
	for _, msg := range expectedMessages {
		assert.Contains(eth.t, err.Error(), msg)
	}
}

// AssertErrorType 에러 타입 검증
func (eth *ErrorTestHelper) AssertErrorType(err error, expectedType interface{}) {
	assert.Error(eth.t, err, "Expected an error")
	assert.IsType(eth.t, expectedType, err)
}

// TestTimeouts 타임아웃 테스트
func (eth *ErrorTestHelper) TestTimeouts(ctx context.Context, timeout time.Duration, fn func(context.Context) error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	err := fn(timeoutCtx)
	eth.AssertErrorContains(err, "context deadline exceeded")
}

// CleanupHelper 정리 헬퍼
type CleanupHelper struct {
	t       *testing.T
	cleanup []func()
}

// NewCleanupHelper 정리 헬퍼 생성
func NewCleanupHelper(t *testing.T) *CleanupHelper {
	helper := &CleanupHelper{t: t}
	t.Cleanup(helper.ExecuteCleanup)
	return helper
}

// AddCleanup 정리 함수 추가
func (ch *CleanupHelper) AddCleanup(fn func()) {
	ch.cleanup = append(ch.cleanup, fn)
}

// ExecuteCleanup 정리 실행
func (ch *CleanupHelper) ExecuteCleanup() {
	for i := len(ch.cleanup) - 1; i >= 0; i-- {
		ch.cleanup[i]()
	}
}
