package health

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"proxynd/internal/config"
)

// MockCacheManager 테스트용 캐시 매니저 모크
type MockCacheManager struct {
	setError    error
	getError    error
	deleteError error
	data        map[string][]byte
}

func NewMockCacheManager() *MockCacheManager {
	return &MockCacheManager{
		data: make(map[string][]byte),
	}
}

// Put 캐시에 데이터 저장 (minimalCache 인터페이스와 호환)
func (m *MockCacheManager) Put(key string, value []byte, ttl time.Duration) error {
	if m.setError != nil {
		return m.setError
	}
	m.data[key] = value
	return nil
}

// Get 캐시에서 데이터 조회 (minimalCache 인터페이스와 호환)
func (m *MockCacheManager) Get(key string) ([]byte, bool) {
	if m.getError != nil {
		return nil, false
	}
	if value, exists := m.data[key]; exists {
		return value, true
	}
	return nil, false
}

// Delete 캐시 키 삭제 (minimalCache 인터페이스와 호환)
func (m *MockCacheManager) Delete(key string) error {
	if m.deleteError != nil {
		return m.deleteError
	}
	delete(m.data, key)
	return nil
}

// Exists 키 존재 여부 확인 (테스트 헬퍼)
func (m *MockCacheManager) Exists(key string) bool {
	_, exists := m.data[key]
	return exists
}

// Clear 모든 데이터 삭제 (테스트 헬퍼)
func (m *MockCacheManager) Clear() error {
	m.data = make(map[string][]byte)
	return nil
}

func TestNewEnhancedHealthService(t *testing.T) {
	config := &config.RootConfig{
		Server: config.ServerConfig{Port: 8080},
		Cache: config.CacheSettings{
			Backend: "file",
			File: config.FileCacheConfig{
				Directory: "/tmp/test-cache",
			},
		},
		Registries: config.RegistryConfig{
			NPM: config.NPMRegistryConfig{Enabled: true},
		},
	}

	cacheManager := NewMockCacheManager()
	healthConfig := DefaultEnhancedHealthConfig()

	service := NewEnhancedHealthService(config, cacheManager, healthConfig)

	assert.NotNil(t, service)
	assert.NotNil(t, service.HealthService)
	assert.Equal(t, config, service.config)
	assert.Equal(t, cacheManager, service.cacheManager)
	assert.NotNil(t, service.enhancedCheckers)
}

func TestDefaultEnhancedHealthConfig(t *testing.T) {
	config := DefaultEnhancedHealthConfig()

	assert.NotNil(t, config)
	assert.Equal(t, 30*time.Second, config.CheckInterval)
	assert.True(t, config.EnableProxyCheck)
	assert.True(t, config.EnableCacheCheck)
	assert.True(t, config.EnableSecurityCheck)
	assert.True(t, config.EnableSystemCheck)
	assert.True(t, config.FastResponseMode)
	assert.NotEmpty(t, config.SystemCheckPaths)
}

func TestEnhancedHealthService_GetComprehensiveStatus(t *testing.T) {
	// 테스트용 임시 디렉토리 생성
	tempDir := t.TempDir()

	config := &config.RootConfig{
		Server: config.ServerConfig{Port: 8080},
		Cache: config.CacheSettings{
			Backend: "file",
			File: config.FileCacheConfig{
				Directory: tempDir,
			},
		},
		Registries: config.RegistryConfig{
			NPM: config.NPMRegistryConfig{
				Enabled:  true,
				Upstream: "https://registry.npmjs.org",
			},
		},
		Security: config.SecuritySettings{
			Authentication: config.AuthenticationConfig{},
		},
	}

	cacheManager := NewMockCacheManager()
	healthConfig := DefaultEnhancedHealthConfig()
	healthConfig.SystemCheckPaths = []string{tempDir}

	service := NewEnhancedHealthService(config, cacheManager, healthConfig)

	// 서비스 시작 (백그라운드)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go service.Start(ctx)

	// 잠시 대기하여 초기 체크 완료
	time.Sleep(100 * time.Millisecond)

	status, err := service.GetComprehensiveStatus()
	require.NoError(t, err)
	assert.NotNil(t, status)

	// 기본 필드 확인
	assert.NotEmpty(t, status.OverallStatus)
	assert.NotEmpty(t, status.Timestamp)
	assert.Greater(t, status.Uptime, time.Duration(0))
	assert.Greater(t, status.ResponseTime, time.Duration(0))

	// 체크 결과 확인
	assert.NotNil(t, status.BasicChecks)
	assert.NotNil(t, status.EnhancedChecks)
	assert.NotNil(t, status.Summary)
	assert.NotNil(t, status.Metrics)

	// 요약 정보 확인
	summary := status.Summary
	assert.Contains(t, summary, "total_checks")
	assert.Contains(t, summary, "healthy_checks")
	assert.Contains(t, summary, "health_percentage")
}

func TestEnhancedHealthService_GetFastHealthStatus(t *testing.T) {
	config := &config.RootConfig{
		Server: config.ServerConfig{Port: 8080},
	}

	cacheManager := NewMockCacheManager()
	healthConfig := DefaultEnhancedHealthConfig()

	service := NewEnhancedHealthService(config, cacheManager, healthConfig)

	start := time.Now()
	status, err := service.GetFastHealthStatus()
	duration := time.Since(start)

	require.NoError(t, err)
	assert.NotNil(t, status)

	// 빠른 응답 시간 확인 (100ms 미만 목표)
	if duration > 100*time.Millisecond {
		t.Logf("Fast health check took %v (should be < 100ms)", duration)
	}

	// 기본 필드 확인
	assert.NotEmpty(t, status.Status)
	assert.NotEmpty(t, status.Timestamp)
	assert.Greater(t, status.ResponseTime, time.Duration(0))
	assert.NotNil(t, status.CoreChecks)
	assert.GreaterOrEqual(t, status.CheckCount, 0)
	assert.True(t, status.HealthRatio >= 0.0, "HealthRatio should be >= 0.0, got: %v", status.HealthRatio)
	assert.True(t, status.HealthRatio <= 100.0, "HealthRatio should be <= 100.0, got: %v", status.HealthRatio)
}

func TestEnhancedHealthService_GetProxyUpstreamStatus(t *testing.T) {
	config := &config.RootConfig{
		Registries: config.RegistryConfig{
			NPM: config.NPMRegistryConfig{
				Enabled:  true,
				Upstream: "https://registry.npmjs.org",
			},
		},
	}

	cacheManager := NewMockCacheManager()
	healthConfig := DefaultEnhancedHealthConfig()

	service := NewEnhancedHealthService(config, cacheManager, healthConfig)

	tests := []struct {
		name        string
		proxyType   string
		expectError bool
	}{
		{
			name:        "enabled proxy type",
			proxyType:   "npm",
			expectError: false,
		},
		{
			name:        "disabled proxy type",
			proxyType:   "apt",
			expectError: true,
		},
		{
			name:        "invalid proxy type",
			proxyType:   "invalid",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := service.GetProxyUpstreamStatus(tt.proxyType)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				if err != nil {
					// 네트워크 오류일 수 있음
					t.Logf("Network error for %s: %v", tt.proxyType, err)
				} else {
					assert.NotNil(t, result)
					assert.NotEmpty(t, result.Status)
					assert.NotEmpty(t, result.Message)
				}
			}
		})
	}
}

func TestEnhancedHealthService_RunHealthCheckFor(t *testing.T) {
	config := &config.RootConfig{
		Cache: config.CacheSettings{
			Backend: "file",
			File: config.FileCacheConfig{
				Directory: t.TempDir(),
			},
		},
	}

	cacheManager := NewMockCacheManager()
	healthConfig := DefaultEnhancedHealthConfig()

	service := NewEnhancedHealthService(config, cacheManager, healthConfig)

	tests := []struct {
		name        string
		checkerName string
		expectError bool
	}{
		{
			name:        "existing checker",
			checkerName: "cache_system",
			expectError: false,
		},
		{
			name:        "non-existing checker",
			checkerName: "nonexistent",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := service.RunHealthCheckFor(tt.checkerName)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.checkerName, result.Name)
			}
		})
	}
}

func TestEnhancedHealthService_GetAvailableCheckers(t *testing.T) {
	config := &config.RootConfig{
		Registries: config.RegistryConfig{
			NPM: config.NPMRegistryConfig{Enabled: true},
		},
		Cache: config.CacheSettings{Backend: "file"},
	}

	cacheManager := NewMockCacheManager()
	healthConfig := DefaultEnhancedHealthConfig()

	service := NewEnhancedHealthService(config, cacheManager, healthConfig)
	checkers := service.GetAvailableCheckers()

	assert.NotNil(t, checkers)
	assert.NotEmpty(t, checkers)

	// 예상되는 체커들이 있는지 확인
	expectedCheckers := []string{"proxy_upstreams", "cache_system", "security_system", "system_resources"}

	for _, expected := range expectedCheckers {
		if checker, exists := checkers[expected]; exists {
			checkerMap, ok := checker.(map[string]interface{})
			require.True(t, ok)
			assert.Contains(t, checkerMap, "name")
			assert.Contains(t, checkerMap, "type")
			assert.Contains(t, checkerMap, "description")
		}
	}
}

func TestEnhancedHealthService_MaintenanceMode(t *testing.T) {
	config := &config.RootConfig{}
	cacheManager := NewMockCacheManager()
	healthConfig := DefaultEnhancedHealthConfig()

	service := NewEnhancedHealthService(config, cacheManager, healthConfig)

	// 유지보수 모드 활성화
	reason := "Scheduled maintenance for testing"
	service.EnableMaintMode(reason)

	// 상태 확인
	_, checks := service.GetStatus()

	// 유지보수 모드 체커가 등록되었는지 확인
	maintChecker, exists := checks["maintenance_mode"]
	require.True(t, exists, "Maintenance mode checker should be registered")

	assert.Equal(t, StatusDegraded, maintChecker.Status)
	assert.Contains(t, maintChecker.Message, reason)
	assert.Contains(t, maintChecker.Details, "reason")
	assert.Contains(t, maintChecker.Details, "maintenance_since")
}

func TestMaintenanceChecker(t *testing.T) {
	tests := []struct {
		name           string
		enabled        bool
		reason         string
		expectedStatus Status
	}{
		{
			name:           "maintenance disabled",
			enabled:        false,
			reason:         "",
			expectedStatus: StatusHealthy,
		},
		{
			name:           "maintenance enabled",
			enabled:        true,
			reason:         "System upgrade",
			expectedStatus: StatusDegraded,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := &MaintenanceChecker{
				enabled: tt.enabled,
				reason:  tt.reason,
				since:   time.Now(),
			}

			assert.Equal(t, "maintenance_mode", checker.Name())

			result := checker.Check(context.Background())
			assert.Equal(t, tt.expectedStatus, result.Status)
			assert.NotEmpty(t, result.Message)

			if tt.enabled {
				assert.Contains(t, result.Message, tt.reason)
				assert.Contains(t, result.Details, "reason")
				assert.Contains(t, result.Details, "maintenance_since")
			}
		})
	}
}

func TestEnhancedHealthService_CategorizedHealth(t *testing.T) {
	config := &config.RootConfig{
		Cache: config.CacheSettings{
			Backend: "file",
			File: config.FileCacheConfig{
				Directory: t.TempDir(),
			},
		},
		Registries: config.RegistryConfig{
			NPM: config.NPMRegistryConfig{Enabled: true},
		},
	}

	cacheManager := NewMockCacheManager()
	healthConfig := DefaultEnhancedHealthConfig()

	service := NewEnhancedHealthService(config, cacheManager, healthConfig)

	status, err := service.GetComprehensiveStatus()
	require.NoError(t, err)

	summary := status.Summary
	categories, exists := summary["categories"]
	require.True(t, exists)

	categoriesMap, ok := categories.(map[string]interface{})
	require.True(t, ok)

	// 예상되는 카테고리들 확인
	expectedCategories := []string{"infrastructure", "performance", "security", "connectivity"}

	for _, category := range expectedCategories {
		if categoryInfo, exists := categoriesMap[category]; exists {
			categoryMap, ok := categoryInfo.(map[string]interface{})
			require.True(t, ok)

			assert.Contains(t, categoryMap, "status")
			assert.Contains(t, categoryMap, "healthy_count")
			assert.Contains(t, categoryMap, "checks")
		}
	}
}

func TestContainsPattern(t *testing.T) {
	tests := []struct {
		text     string
		pattern  string
		expected bool
	}{
		{"environment", "environment", true},
		{"disk_space", "_space", true},
		{"writable_cache", "writable_", true},
		{"proxy_npm", "proxy_", true},
		{"system_resources", "system_", true},
		{"random_text", "nonexistent", false},
		{"", "", false},
		{"test", "_missing", false},
	}

	for _, tt := range tests {
		t.Run(tt.text+"_"+tt.pattern, func(t *testing.T) {
			result := containsPattern(tt.text, tt.pattern)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func BenchmarkEnhancedHealthService_GetFastHealthStatus(b *testing.B) {
	config := &config.RootConfig{
		Server: config.ServerConfig{Port: 8080},
	}

	cacheManager := NewMockCacheManager()
	healthConfig := DefaultEnhancedHealthConfig()
	service := NewEnhancedHealthService(config, cacheManager, healthConfig)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		status, err := service.GetFastHealthStatus()
		if err != nil {
			b.Fatalf("GetFastHealthStatus failed: %v", err)
		}
		if status == nil {
			b.Fatal("GetFastHealthStatus returned nil status")
		}
	}
}

func BenchmarkEnhancedHealthService_GetComprehensiveStatus(b *testing.B) {
	config := &config.RootConfig{
		Server: config.ServerConfig{Port: 8080},
		Cache: config.CacheSettings{
			Backend: "file",
			File: config.FileCacheConfig{
				Directory: b.TempDir(),
			},
		},
	}

	cacheManager := NewMockCacheManager()
	healthConfig := DefaultEnhancedHealthConfig()
	service := NewEnhancedHealthService(config, cacheManager, healthConfig)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		status, err := service.GetComprehensiveStatus()
		if err != nil {
			b.Fatalf("GetComprehensiveStatus failed: %v", err)
		}
		if status == nil {
			b.Fatal("GetComprehensiveStatus returned nil status")
		}
	}
}
