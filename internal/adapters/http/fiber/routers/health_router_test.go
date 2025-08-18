package routers

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"proxynd/internal/factory"
	"proxynd/internal/logging"
)

// mockAdapterHealthChecker는 테스트용 어댑터 헬스체크 모의 객체
type mockAdapterHealthChecker struct {
	healthStatus map[string]factory.HealthStatus
}

func (m *mockAdapterHealthChecker) HealthCheck() map[string]factory.HealthStatus {
	if m.healthStatus == nil {
		// 기본 상태: 모든 어댑터가 건강함
		m.healthStatus = map[string]factory.HealthStatus{
			"maven": {
				IsHealthy:   true,
				Initialized: true,
				LastCheck:   time.Now().Unix(),
				ErrorCount:  0,
			},
			"npm": {
				IsHealthy:   true,
				Initialized: true,
				LastCheck:   time.Now().Unix(),
				ErrorCount:  0,
			},
			"apt": {
				IsHealthy:   false, // 하나는 비정상으로 설정
				Initialized: true,
				LastCheck:   time.Now().Unix(),
				ErrorCount:  1,
				LastError:   "mock error for testing",
			},
		}
	}
	return m.healthStatus
}

// TestHealthAdaptersEndpoint 어댑터 헬스체크 엔드포인트 테스트
func TestHealthAdaptersEndpoint(t *testing.T) {
	// Fiber 앱 생성
	app := fiber.New()

	// 모의 어댑터 헬스체커 설정
	mockChecker := &mockAdapterHealthChecker{}
	InitHandlerAdapterFactory(mockChecker)

	// 헬스 라우터 초기화
	HealthRouter(app)

	tests := []struct {
		name               string
		endpoint           string
		expectedStatusCode int
		healthyAdapters    bool
	}{
		{
			name:               "기본 헬스체크",
			endpoint:           "/healthz",
			expectedStatusCode: http.StatusOK,
			healthyAdapters:    true,
		},
		{
			name:               "어댑터 헬스체크 - 일부 비정상",
			endpoint:           "/health/adapters",
			expectedStatusCode: http.StatusServiceUnavailable, // apt가 비정상이므로
			healthyAdapters:    false,
		},
		{
			name:               "개별 어댑터 헬스체크 - 정상 (maven)",
			endpoint:           "/health/adapters/maven",
			expectedStatusCode: http.StatusOK,
			healthyAdapters:    true,
		},
		{
			name:               "개별 어댑터 헬스체크 - 비정상 (apt)",
			endpoint:           "/health/adapters/apt",
			expectedStatusCode: http.StatusServiceUnavailable,
			healthyAdapters:    false,
		},
		{
			name:               "개별 어댑터 헬스체크 - 존재하지 않는 어댑터",
			endpoint:           "/health/adapters/nonexistent",
			expectedStatusCode: http.StatusNotFound,
			healthyAdapters:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// HTTP 요청 생성
			req := httptest.NewRequest(http.MethodGet, tt.endpoint, nil)
			resp, err := app.Test(req, 5000) // 5초 타임아웃

			// 기본 검증
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }() // 응답 바디 닫기
			assert.Equal(t, tt.expectedStatusCode, resp.StatusCode)

			// 응답 헤더 확인 (Fiber의 기본 JSON Content-Type)
			assert.Contains(t, resp.Header.Get("Content-Type"), "application/json")
		})
	}
}

// TestHealthAdaptersWithoutFactory 어댑터 팩토리가 초기화되지 않은 경우 테스트
func TestHealthAdaptersWithoutFactory(t *testing.T) {
	// 전역 어댑터 팩토리 초기화 해제
	handlerAdapterFactory = nil

	// Fiber 앱 생성
	app := fiber.New()

	// 헬스 라우터 초기화 (어댑터 팩토리 없이)
	HealthRouter(app)

	// 어댑터 헬스체크 요청
	req := httptest.NewRequest(http.MethodGet, "/health/adapters", nil)
	resp, err := app.Test(req, 5000)

	// 검증: 서비스 사용 불가 상태 반환
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }() // 응답 바디 닫기
	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
}

// TestGetHealthyAdapterCount 건강한 어댑터 개수 계산 함수 테스트
func TestGetHealthyAdapterCount(t *testing.T) {
	adapterStatus := map[string]factory.HealthStatus{
		"maven": {IsHealthy: true, Initialized: true},
		"npm":   {IsHealthy: true, Initialized: true},
		"apt":   {IsHealthy: false, Initialized: true},
		"pip":   {IsHealthy: true, Initialized: false}, // 초기화되지 않음
	}

	healthyCount := getHealthyAdapterCount(adapterStatus)
	assert.Equal(t, 2, healthyCount) // maven과 npm만 건강하고 초기화됨
}

// TestGetOverallAdapterStatus 전체 어댑터 상태 문자열 테스트
func TestGetOverallAdapterStatus(t *testing.T) {
	assert.Equal(t, "healthy", getOverallAdapterStatus(true))
	assert.Equal(t, "unhealthy", getOverallAdapterStatus(false))
}

// TestInit 초기화 함수 테스트
func TestInit(t *testing.T) {
	// 로깅 시스템이 초기화되어 있는지 확인
	logger := logging.GetLogger()
	assert.NotNil(t, logger)
}
