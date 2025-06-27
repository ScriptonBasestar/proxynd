package commands

import (
	"testing"
	"time"
)

func TestBoolToStatus(t *testing.T) {
	tests := []struct {
		input    bool
		expected string
	}{
		{true, "활성화"},
		{false, "비활성화"},
	}

	for _, test := range tests {
		result := boolToStatus(test.input)
		if result != test.expected {
			t.Errorf("boolToStatus(%t) = %s; expected %s", test.input, result, test.expected)
		}
	}
}

func TestServerStatusResponse(t *testing.T) {
	// 구조체 초기화 테스트
	response := ServerStatusResponse{
		Status:    "healthy",
		Timestamp: time.Now(),
		Uptime:    "1h 30m 45s",
		Version:   "1.0.0",
		Server: ServerInfo{
			Host:       "localhost",
			Port:       "8080",
			TLSEnabled: false,
			Handlers:   52,
			PID:        12345,
		},
		System: SystemInfo{
			OS:           "linux",
			Arch:         "amd64",
			CPUs:         8,
			Goroutines:   25,
			MemoryUsage:  128.5,
			GCCycles:     10,
			GCCPUPercent: 2.5,
		},
	}

	if response.Status != "healthy" {
		t.Errorf("Expected status to be 'healthy', got %s", response.Status)
	}

	if response.Server.Port != "8080" {
		t.Errorf("Expected port to be '8080', got %s", response.Server.Port)
	}

	if response.System.OS != "linux" {
		t.Errorf("Expected OS to be 'linux', got %s", response.System.OS)
	}
}

func TestHealthCheckResponse(t *testing.T) {
	response := HealthCheckResponse{
		Status:    "healthy",
		Timestamp: time.Now(),
		Uptime:    "2h 15m 30s",
		Checks: map[string]HealthCheckItem{
			"database": {
				Status:      "healthy",
				Message:     "Connection OK",
				LastChecked: time.Now(),
				Duration:    "5ms",
			},
		},
		Summary: HealthCheckSummary{
			TotalChecks:     5,
			HealthyChecks:   4,
			UnhealthyChecks: 1,
			DegradedChecks:  0,
		},
	}

	if response.Summary.TotalChecks != 5 {
		t.Errorf("Expected TotalChecks to be 5, got %d", response.Summary.TotalChecks)
	}

	if response.Summary.HealthyChecks != 4 {
		t.Errorf("Expected HealthyChecks to be 4, got %d", response.Summary.HealthyChecks)
	}

	if len(response.Checks) != 1 {
		t.Errorf("Expected 1 check, got %d", len(response.Checks))
	}
}

func TestMetricsResponse(t *testing.T) {
	response := MetricsResponse{
		Timestamp: time.Now(),
		System: SystemMetrics{
			CPUUsage:       25.5,
			MemoryUsage:    75.2,
			DiskUsage:      45.8,
			LoadAverage1m:  0.85,
			LoadAverage5m:  0.92,
			LoadAverage15m: 1.05,
			UptimeSeconds:  7200,
		},
		Cache: map[string]CacheMetrics{
			"npm": {
				Hits:                1250,
				Misses:              350,
				HitRate:             78.1,
				SizeBytes:           1024 * 1024 * 512, // 512MB
				ItemsCount:          1600,
				BandwidthSavedBytes: 1024 * 1024 * 1024 * 2, // 2GB
			},
		},
		Health: HealthMetrics{
			OverallStatus:   "healthy",
			HealthyServices: 8,
			TotalServices:   10,
			HealthScore:     80.0,
		},
	}

	if response.System.CPUUsage != 25.5 {
		t.Errorf("Expected CPUUsage to be 25.5, got %f", response.System.CPUUsage)
	}

	npmCache, exists := response.Cache["npm"]
	if !exists {
		t.Error("Expected npm cache metrics to exist")
	}

	if npmCache.HitRate != 78.1 {
		t.Errorf("Expected npm cache hit rate to be 78.1, got %f", npmCache.HitRate)
	}

	if response.Health.HealthScore != 80.0 {
		t.Errorf("Expected health score to be 80.0, got %f", response.Health.HealthScore)
	}
}
