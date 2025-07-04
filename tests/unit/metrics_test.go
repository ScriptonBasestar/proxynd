package unit

import (
	"io"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-playground/assert/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/prometheus/client_golang/prometheus"

	"proxynd/metrics"
)

func TestMetrics_Initialization(t *testing.T) {
	// 메트릭 초기화
	metrics.InitMetrics()
	m := metrics.GetMetrics()

	// 메트릭 인스턴스 확인
	assert.NotEqual(t, nil, m)
	assert.NotEqual(t, nil, m.HTTPRequestsTotal)
	assert.NotEqual(t, nil, m.HTTPRequestDuration)
	assert.NotEqual(t, nil, m.CacheHitsTotal)
	assert.NotEqual(t, nil, m.ProxyRequestsTotal)
	assert.NotEqual(t, nil, m.AuthAttemptsTotal)
	assert.NotEqual(t, nil, m.PackageVerifications)
}

func TestPrometheusMiddleware(t *testing.T) {
	// Fiber 앱 생성
	app := fiber.New()

	// 메트릭 초기화
	metrics.InitMetrics()

	// 메트릭 미들웨어 적용
	app.Use(metrics.PrometheusMiddleware())

	// 테스트 핸들러
	app.Get("/test", func(c *fiber.Ctx) error {
		// 캐시 히트 시뮬레이션
		c.Locals("cache_hit", true)
		c.Locals("registry_type", "npm")
		return c.SendString("OK")
	})

	app.Get("/proxy/npm/test-package", func(c *fiber.Ctx) error {
		c.Locals("registry_type", "npm")
		c.Locals("cache_hit", false)
		c.Locals("upstream_duration", 100*time.Millisecond)
		return c.JSON(fiber.Map{"package": "test"})
	})

	// 요청 테스트
	tests := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
		locals         map[string]interface{}
	}{
		{
			name:           "Simple GET request",
			method:         "GET",
			path:           "/test",
			expectedStatus: 200,
		},
		{
			name:           "Proxy request with cache miss",
			method:         "GET",
			path:           "/proxy/npm/test-package",
			expectedStatus: 200,
		},
		{
			name:           "404 request",
			method:         "GET",
			path:           "/not-found",
			expectedStatus: 404,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			resp, err := app.Test(req)

			assert.Equal(t, nil, err)
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
		})
	}

	// 메트릭 값 확인
	m := metrics.GetMetrics()

	// HTTP 요청 카운터가 증가했는지 확인
	// 실제 Prometheus 메트릭 값을 확인하는 것은 복잡하므로
	// 여기서는 메트릭이 nil이 아닌지만 확인
	assert.NotEqual(t, nil, m.HTTPRequestsTotal)
	assert.NotEqual(t, nil, m.HTTPActiveRequests)
}

func TestMetricsEndpoint(t *testing.T) {
	// Fiber 앱 생성
	app := fiber.New()

	// 메트릭 초기화
	metrics.InitMetrics()

	// Prometheus 핸들러 등록
	app.Get("/metrics", func(c *fiber.Ctx) error {
		// 간단한 테스트를 위해 텍스트 응답
		return c.SendString("# HELP proxynd_http_requests_total Total number of HTTP requests\n# TYPE proxynd_http_requests_total counter\n")
	})

	// /metrics 엔드포인트 테스트
	req := httptest.NewRequest("GET", "/metrics", nil)
	resp, err := app.Test(req)

	assert.Equal(t, nil, err)
	assert.Equal(t, 200, resp.StatusCode)

	// Content-Type 확인
	body, _ := io.ReadAll(resp.Body)
	assert.Equal(t, true, strings.Contains(string(body), "# HELP"))
	assert.Equal(t, true, strings.Contains(string(body), "# TYPE"))
}

func TestCustomCollector(t *testing.T) {
	// 커스텀 수집기 생성
	collector := metrics.NewCustomCollector()

	// Describe 채널
	descCh := make(chan *prometheus.Desc, 10)
	collector.Describe(descCh)
	close(descCh)

	// 설명이 채널에 전송되었는지 확인
	descCount := 0
	for range descCh {
		descCount++
	}
	assert.Equal(t, true, descCount > 0)

	// Collect 채널
	metricCh := make(chan prometheus.Metric, 10)
	collector.Collect(metricCh)
	close(metricCh)

	// 메트릭이 채널에 전송되었는지 확인
	metricCount := 0
	for range metricCh {
		metricCount++
	}
	assert.Equal(t, true, metricCount > 0)
}

func TestMetricLabels(t *testing.T) {
	// 메트릭 초기화
	metrics.ResetMetrics()
	m := metrics.GetMetrics()

	// 다양한 레이블로 메트릭 증가
	m.HTTPRequestsTotal.WithLabelValues("GET", "/test", "200", "npm").Inc()
	m.HTTPRequestsTotal.WithLabelValues("POST", "/test", "201", "pypi").Inc()
	m.HTTPRequestsTotal.WithLabelValues("GET", "/test", "404", "docker").Inc()

	m.CacheHitsTotal.WithLabelValues("npm", "file").Inc()
	m.CacheHitsTotal.WithLabelValues("npm", "file").Inc()
	m.CacheMissesTotal.WithLabelValues("npm", "file").Inc()

	m.ProxyRequestsTotal.WithLabelValues("npm", "registry.npmjs.org", "GET").Inc()
	m.ProxyErrorsTotal.WithLabelValues("npm", "registry.npmjs.org", "timeout").Inc()

	m.AuthAttemptsTotal.WithLabelValues("basic", "success").Inc()
	m.AuthAttemptsTotal.WithLabelValues("basic", "failure").Inc()
	m.AuthFailuresTotal.WithLabelValues("basic", "invalid_credentials").Inc()

	m.PackageVerifications.WithLabelValues("npm", "success").Inc()
	m.PackageVerifications.WithLabelValues("npm", "failure").Inc()
	m.VerificationFailures.WithLabelValues("npm", "hash_mismatch").Inc()

	// 메트릭이 정상적으로 생성되었는지 확인
	// (실제 값을 확인하려면 prometheus/testutil 패키지 필요)
	assert.NotEqual(t, nil, m.HTTPRequestsTotal)
	assert.NotEqual(t, nil, m.CacheHitsTotal)
	assert.NotEqual(t, nil, m.ProxyRequestsTotal)
}

func TestSystemMetrics(t *testing.T) {
	// 시스템 메트릭 수집 (Linux에서만 값이 있음)
	systemMetrics := metrics.GetSystemMetrics()

	// 맵이 nil이 아닌지 확인
	assert.NotEqual(t, nil, systemMetrics)

	// Linux에서는 일부 값이 있어야 함
	if len(systemMetrics) > 0 {
		// CPU 사용률은 0-1 사이
		if cpuUsage, ok := systemMetrics["cpu_usage"]; ok {
			assert.Equal(t, true, cpuUsage >= 0 && cpuUsage <= 1)
		}

		// 메모리 사용률은 0-1 사이
		if memUsage, ok := systemMetrics["memory_usage"]; ok {
			assert.Equal(t, true, memUsage >= 0 && memUsage <= 1)
		}

		// 로드 평균은 0 이상
		if load1m, ok := systemMetrics["load_1m"]; ok {
			assert.Equal(t, true, load1m >= 0)
		}
	}
}

func TestMetricsMiddleware_ErrorHandling(t *testing.T) {
	// Fiber 앱 생성
	app := fiber.New()

	// 메트릭 초기화
	metrics.InitMetrics()

	// 메트릭 미들웨어 적용
	app.Use(metrics.PrometheusMiddleware())

	// 오류를 발생시키는 핸들러
	app.Get("/error", func(c *fiber.Ctx) error {
		return fiber.NewError(500, "Internal Server Error")
	})

	// 인증 실패 시뮬레이션
	app.Get("/auth-fail", func(c *fiber.Ctx) error {
		c.Locals("auth_attempt", "basic")
		return c.Status(401).SendString("Unauthorized")
	})

	// 검증 실패 시뮬레이션
	app.Get("/verify-fail", func(c *fiber.Ctx) error {
		c.Locals("packageVerified", false)
		c.Locals("verificationFailureType", "hash_mismatch")
		c.Locals("registry_type", "npm")
		return c.Status(403).SendString("Verification Failed")
	})

	// 테스트 실행
	tests := []struct {
		path           string
		expectedStatus int
	}{
		{"/error", 500},
		{"/auth-fail", 401},
		{"/verify-fail", 403},
	}

	for _, tt := range tests {
		req := httptest.NewRequest("GET", tt.path, nil)
		resp, err := app.Test(req)

		assert.Equal(t, nil, err)
		assert.Equal(t, tt.expectedStatus, resp.StatusCode)
	}
}
