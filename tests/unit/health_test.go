package unit

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-playground/assert/v2"
	"github.com/gofiber/fiber/v2"

	"proxynd/health"
	"proxynd/routers"
)

func TestHealthChecker_Environment(t *testing.T) {
	// 환경 변수 설정
	_ = os.Setenv("TEST_VAR1", "value1")
	_ = os.Setenv("TEST_VAR2", "value2")
	defer func() {
		_ = os.Unsetenv("TEST_VAR1")
		_ = os.Unsetenv("TEST_VAR2")
		_ = os.Unsetenv("TEST_VAR3")
	}()

	// 환경 변수 체커 생성
	checker := health.NewEnvironmentChecker([]string{"TEST_VAR1", "TEST_VAR2", "TEST_VAR3"})

	// 체크 실행
	ctx := context.Background()
	result := checker.Check(ctx)

	// 결과 확인
	assert.Equal(t, "environment", checker.Name())
	assert.Equal(t, health.StatusUnhealthy, result.Status) // TEST_VAR3가 없으므로 unhealthy
	assert.Equal(t, true, result.Details["TEST_VAR1"].(bool))
	assert.Equal(t, true, result.Details["TEST_VAR2"].(bool))
	assert.Equal(t, false, result.Details["TEST_VAR3"].(bool))
}

func TestHealthChecker_Writable(t *testing.T) {
	// 임시 디렉토리 생성
	tempDir := t.TempDir()

	// 쓰기 가능 체커 생성
	checker := health.NewWritableChecker(tempDir)

	// 체크 실행
	ctx := context.Background()
	result := checker.Check(ctx)

	// 결과 확인
	assert.Equal(t, health.StatusHealthy, result.Status)
	assert.Equal(t, "Directory is writable", result.Message)
	assert.Equal(t, tempDir, result.Details["path"])
}

func TestHealthChecker_DiskSpace(t *testing.T) {
	// 디스크 공간 체커 생성
	// 매우 작은 최소 공간 요구사항 설정 (테스트 통과를 위해)
	checker := health.NewDiskSpaceChecker("/tmp", 1024, 0.1) // 1KB, 0.1%

	// 체크 실행
	ctx := context.Background()
	result := checker.Check(ctx)

	// 결과 확인
	assert.Equal(t, "disk_space", checker.Name())
	// 대부분의 시스템에서는 healthy 상태여야 함
	assert.NotEqual(t, health.StatusUnhealthy, result.Status)
	assert.NotEqual(t, nil, result.Details["total_bytes"])
	assert.NotEqual(t, nil, result.Details["free_bytes"])
	assert.NotEqual(t, nil, result.Details["free_percent"])
}

func TestHealthService(t *testing.T) {
	// 건강 상태 서비스 생성
	service := health.NewHealthService(100 * time.Millisecond)

	// 테스트 체커 등록
	_ = os.Setenv("TEST_ENV", "test")
	defer func() { _ = os.Unsetenv("TEST_ENV") }()

	service.RegisterChecker(health.NewEnvironmentChecker([]string{"TEST_ENV"}))

	// 서비스 시작
	ctx, cancel := context.WithCancel(context.Background())
	go service.Start(ctx)

	// 체크가 실행될 때까지 대기
	time.Sleep(200 * time.Millisecond)

	// 상태 확인
	status, checks := service.GetStatus()
	assert.Equal(t, health.StatusHealthy, status)
	assert.Equal(t, 1, len(checks))
	assert.NotEqual(t, nil, checks["environment"])
	assert.Equal(t, health.StatusHealthy, checks["environment"].Status)

	// 가동 시간 확인
	uptime := service.GetUptime()
	assert.Equal(t, true, uptime > 0)

	// 서비스 중지
	cancel()
}

func TestHealthEndpoint(t *testing.T) {
	// Fiber 앱 생성
	app := fiber.New()
	routers.HealthRouter(app)

	tests := []struct {
		name           string
		path           string
		expectedStatus int
		checkResponse  func(t *testing.T, body []byte)
	}{
		{
			name:           "Basic health check",
			path:           "/healthz",
			expectedStatus: 200,
			checkResponse: func(t *testing.T, body []byte) {
				// JSON 응답 확인
				assert.Equal(t, true, len(body) > 0)
			},
		},
		{
			name:           "Simple format",
			path:           "/healthz?format=simple",
			expectedStatus: 200,
			checkResponse: func(t *testing.T, body []byte) {
				// 간단한 형식 확인
				assert.Equal(t, true, len(body) > 0)
			},
		},
		{
			name:           "Liveness probe",
			path:           "/health/live",
			expectedStatus: 200,
			checkResponse: func(t *testing.T, body []byte) {
				assert.Equal(t, true, string(body) != "")
			},
		},
		{
			name:           "Readiness probe",
			path:           "/health/ready",
			expectedStatus: 503, // 필수 체크가 없으므로 not ready
			checkResponse: func(t *testing.T, body []byte) {
				assert.Equal(t, true, string(body) != "")
			},
		},
		{
			name:           "Individual check - not found",
			path:           "/health/check/nonexistent",
			expectedStatus: 404,
			checkResponse: func(t *testing.T, body []byte) {
				assert.Equal(t, true, string(body) != "")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			resp, err := app.Test(req, -1)

			assert.Equal(t, nil, err)
			defer func() { _ = resp.Body.Close() }()
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			body, _ := io.ReadAll(resp.Body)
			if tt.checkResponse != nil {
				tt.checkResponse(t, body)
			}
		})
	}
}

func TestHealthEndpoint_Debug(t *testing.T) {
	// Fiber 앱 생성
	app := fiber.New()
	routers.HealthRouter(app)

	// Production 환경에서는 접근 불가
	_ = os.Setenv("ENVIRONMENT", "production")
	req := httptest.NewRequest("GET", "/health/debug", nil)
	resp, err := app.Test(req)

	assert.Equal(t, nil, err)
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, 403, resp.StatusCode)

	// Development 환경에서는 접근 가능
	_ = os.Setenv("ENVIRONMENT", "development")
	req = httptest.NewRequest("GET", "/health/debug", nil)
	resp, err = app.Test(req)

	assert.Equal(t, nil, err)
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, 200, resp.StatusCode)

	_ = os.Unsetenv("ENVIRONMENT")
}

func TestHTTPChecker(t *testing.T) {
	// 테스트 서버 생성
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("OK"))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer testServer.Close()

	// HTTP 체커 생성
	checker := health.NewHTTPChecker("test_endpoint", testServer.URL+"/health", 5*time.Second)

	// 체크 실행
	ctx := context.Background()
	result := checker.Check(ctx)

	// 결과 확인
	assert.Equal(t, "test_endpoint", checker.Name())
	assert.Equal(t, health.StatusHealthy, result.Status)
	assert.Equal(t, 200, result.Details["status_code"])

	// 실패하는 엔드포인트 테스트
	checker2 := health.NewHTTPChecker("test_404", testServer.URL+"/notfound", 5*time.Second)
	result2 := checker2.Check(ctx)

	assert.Equal(t, health.StatusUnhealthy, result2.Status)
	assert.Equal(t, 404, result2.Details["status_code"])
}

func TestCacheBackendChecker(t *testing.T) {
	// 성공하는 체크 함수
	successCheck := func(_ context.Context) error {
		return nil
	}

	// 실패하는 체크 함수
	failCheck := func(_ context.Context) error {
		return fmt.Errorf("connection failed")
	}

	// 성공 체커 테스트
	checker1 := health.NewCacheBackendChecker("memory", successCheck)
	result1 := checker1.Check(context.Background())

	assert.Equal(t, "cache_memory", checker1.Name())
	assert.Equal(t, health.StatusHealthy, result1.Status)
	assert.Equal(t, "Cache backend is healthy", result1.Message)

	// 실패 체커 테스트
	checker2 := health.NewCacheBackendChecker("redis", failCheck)
	result2 := checker2.Check(context.Background())

	assert.Equal(t, "cache_redis", checker2.Name())
	assert.Equal(t, health.StatusUnhealthy, result2.Status)
	assert.Equal(t, true, strings.Contains(result2.Message, "connection failed"))
}
