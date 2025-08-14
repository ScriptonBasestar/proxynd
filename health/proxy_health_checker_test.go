package health

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"proxynd/internal/config"
)

func TestNewProxyHealthChecker(t *testing.T) {
	config := &config.RootConfig{
		Registries: config.RegistryConfig{
			NPM: config.NPMRegistryConfig{
				Enabled:  true,
				Upstream: "https://registry.npmjs.org",
			},
		},
	}

	checker := NewProxyHealthChecker(config)
	assert.NotNil(t, checker)
	assert.Equal(t, "proxy_upstreams", checker.Name())
	assert.NotNil(t, checker.client)
	assert.NotNil(t, checker.circuitBreaker)
}

func TestProxyHealthChecker_Check(t *testing.T) {
	// Mock HTTP 서버 생성
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/-/ping":
			// NPM ping 엔드포인트
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("{}"))
		case "/simple/":
			// PyPI simple 엔드포인트
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("<html><head><title>Simple Index</title></head></html>"))
		default:
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer mockServer.Close()

	config := &config.RootConfig{
		Registries: config.RegistryConfig{
			NPM: config.NPMRegistryConfig{
				Enabled:  true,
				Upstream: mockServer.URL,
			},
			PyPI: config.PyPIRegistryConfig{
				Enabled: true,
				Simple:  mockServer.URL + "/simple",
			},
		},
	}

	checker := NewProxyHealthChecker(config)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result := checker.Check(ctx)

	assert.NotNil(t, result)
	assert.Equal(t, "proxy_upstreams", result.Name)
	assert.NotEmpty(t, result.Details)

	// 세부 내용 확인
	details, ok := result.Details["npm"].(map[string]interface{})
	require.True(t, ok, "NPM details should exist")
	assert.Equal(t, "healthy", details["status"])

	details, ok = result.Details["pypi"].(map[string]interface{})
	require.True(t, ok, "PyPI details should exist")
	assert.Equal(t, "healthy", details["status"])

	summary, ok := result.Details["summary"].(map[string]interface{})
	require.True(t, ok, "Summary should exist")
	assert.Greater(t, summary["healthy_proxies"], 0)
}

func TestProxyHealthChecker_checkNPMUpstream(t *testing.T) {
	tests := []struct {
		name           string
		serverResponse func(w http.ResponseWriter, r *http.Request)
		expectedStatus Status
	}{
		{
			name: "healthy npm registry",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("{}"))
			},
			expectedStatus: StatusHealthy,
		},
		{
			name: "npm registry error",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			expectedStatus: StatusUnhealthy,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockServer := httptest.NewServer(http.HandlerFunc(tt.serverResponse))
			defer mockServer.Close()

			config := &config.RootConfig{
				Registries: config.RegistryConfig{
					NPM: config.NPMRegistryConfig{
						Enabled:  true,
						Upstream: mockServer.URL,
					},
				},
			}

			checker := NewProxyHealthChecker(config)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			result := checker.checkNPMUpstream(ctx)
			assert.Equal(t, tt.expectedStatus, result.Status)
			assert.NotEmpty(t, result.Message)
			assert.Contains(t, result.UpstreamURL, mockServer.URL)
		})
	}
}

func TestProxyHealthChecker_checkPyPIUpstream(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/simple/" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("<html><head><title>Simple Index</title></head></html>"))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer mockServer.Close()

	config := &config.RootConfig{
		Registries: config.RegistryConfig{
			PyPI: config.PyPIRegistryConfig{
				Enabled: true,
				Simple:  mockServer.URL + "/simple",
			},
		},
	}

	checker := NewProxyHealthChecker(config)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result := checker.checkPyPIUpstream(ctx)
	assert.Equal(t, StatusHealthy, result.Status)
	assert.Contains(t, result.UpstreamURL, mockServer.URL)
	assert.NotEmpty(t, result.Message)
}

func TestProxyHealthChecker_checkAPTUpstream(t *testing.T) {
	tests := []struct {
		name           string
		config         *config.RootConfig
		expectedStatus Status
		expectError    bool
	}{
		{
			name: "no mirrors configured",
			config: &config.RootConfig{
				Registries: config.RegistryConfig{
					APT: config.APTRegistryConfig{
						Enabled: true,
						Mirrors: map[string][]config.APTMirror{},
					},
				},
			},
			expectedStatus: StatusUnhealthy,
			expectError:    false,
		},
		{
			name: "mirrors configured",
			config: &config.RootConfig{
				Registries: config.RegistryConfig{
					APT: config.APTRegistryConfig{
						Enabled: true,
						Mirrors: map[string][]config.APTMirror{
							"ubuntu": {
								{
									Name: "Ubuntu Main",
									URL:  "http://archive.ubuntu.com/ubuntu/",
								},
							},
						},
					},
				},
			},
			expectedStatus: StatusUnhealthy, // 실제 연결은 실패할 것으로 예상
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := NewProxyHealthChecker(tt.config)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			result := checker.checkAPTUpstream(ctx)
			assert.Equal(t, tt.expectedStatus, result.Status)
			assert.NotEmpty(t, result.Message)
		})
	}
}

func TestProxyHealthChecker_GetUpstreamStatus(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("{}"))
	}))
	defer mockServer.Close()

	config := &config.RootConfig{
		Registries: config.RegistryConfig{
			NPM: config.NPMRegistryConfig{
				Enabled:  true,
				Upstream: mockServer.URL,
			},
		},
	}

	checker := NewProxyHealthChecker(config)

	tests := []struct {
		name        string
		proxyType   string
		expectError bool
	}{
		{
			name:        "valid proxy type",
			proxyType:   "npm",
			expectError: false,
		},
		{
			name:        "invalid proxy type",
			proxyType:   "invalid",
			expectError: true,
		},
		{
			name:        "disabled proxy type",
			proxyType:   "apt",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := checker.GetUpstreamStatus(tt.proxyType)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, StatusHealthy, result.Status)
			}
		})
	}
}

func TestProxyHealthChecker_CircuitBreaker(t *testing.T) {
	// 항상 실패하는 서버
	failingServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer failingServer.Close()

	config := &config.RootConfig{
		Registries: config.RegistryConfig{
			NPM: config.NPMRegistryConfig{
				Enabled:  true,
				Upstream: failingServer.URL,
			},
		},
	}

	checker := NewProxyHealthChecker(config)

	// 여러 번 실패시켜서 서킷 브레이커 동작 확인
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		result := checker.checkNPMUpstream(ctx)
		assert.Equal(t, StatusUnhealthy, result.Status)
	}

	// 서킷 브레이커 상태 확인
	cbStatus := checker.GetCircuitBreakerStatus()
	assert.NotEmpty(t, cbStatus)

	npmCB, exists := cbStatus["npm"]
	assert.True(t, exists)
	assert.NotNil(t, npmCB)
}

func TestProxyHealthChecker_PerformanceCheck(t *testing.T) {
	// 느린 응답 서버 (100ms 지연)
	slowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("{}"))
	}))
	defer slowServer.Close()

	config := &config.RootConfig{
		Registries: config.RegistryConfig{
			NPM: config.NPMRegistryConfig{
				Enabled:  true,
				Upstream: slowServer.URL,
			},
		},
	}

	checker := NewProxyHealthChecker(config)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result := checker.checkNPMUpstream(ctx)

	// 느린 응답이지만 성공해야 함
	assert.Equal(t, StatusHealthy, result.Status) // 100ms는 성능 저하 임계값(2초)보다 빠름
	assert.Greater(t, result.ResponseTime, 50*time.Millisecond)
	assert.Contains(t, result.UpstreamURL, slowServer.URL)
}

func TestProxyHealthChecker_TimeoutHandling(t *testing.T) {
	// 응답하지 않는 서버
	hangingServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Second) // 요청보다 오래 대기
	}))
	defer hangingServer.Close()

	config := &config.RootConfig{
		Registries: config.RegistryConfig{
			NPM: config.NPMRegistryConfig{
				Enabled:  true,
				Upstream: hangingServer.URL,
			},
		},
	}

	checker := NewProxyHealthChecker(config)

	// 짧은 타임아웃으로 컨텍스트 생성
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	start := time.Now()
	result := checker.checkNPMUpstream(ctx)
	duration := time.Since(start)

	// 타임아웃으로 인해 실패해야 함
	assert.Equal(t, StatusUnhealthy, result.Status)
	assert.Less(t, duration, 2*time.Second)  // 타임아웃에 의해 빠르게 실패
	assert.Contains(t, result.Message, "실패") // 한국어 에러 메시지
}

func BenchmarkProxyHealthChecker_Check(b *testing.B) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("{}"))
	}))
	defer mockServer.Close()

	config := &config.RootConfig{
		Registries: config.RegistryConfig{
			NPM: config.NPMRegistryConfig{
				Enabled:  true,
				Upstream: mockServer.URL,
			},
		},
	}

	checker := NewProxyHealthChecker(config)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := checker.Check(ctx)
		if result.Status == StatusUnhealthy {
			b.Fatalf("Unexpected unhealthy status: %s", result.Message)
		}
	}
}

func TestProxyHealthChecker_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// 실제 공개 레지스트리에 대한 통합 테스트
	config := &config.RootConfig{
		Registries: config.RegistryConfig{
			NPM: config.NPMRegistryConfig{
				Enabled:  true,
				Upstream: "https://registry.npmjs.org",
			},
			PyPI: config.PyPIRegistryConfig{
				Enabled: true,
				Simple:  "https://pypi.org/simple",
			},
		},
	}

	checker := NewProxyHealthChecker(config)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result := checker.Check(ctx)

	assert.NotNil(t, result)
	assert.Equal(t, "proxy_upstreams", result.Name)

	// 실제 레지스트리는 정상이어야 함
	if result.Status == StatusUnhealthy {
		t.Logf("Integration test failed - this might be due to network issues: %s", result.Message)
	}

	// 응답에 필요한 필드들이 있는지 확인
	assert.NotEmpty(t, result.Details)
	assert.NotEmpty(t, result.Message)
	assert.Greater(t, result.Duration, time.Duration(0))
}
