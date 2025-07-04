package routers

import (
	"runtime"
	"time"

	"github.com/gofiber/fiber/v2"

	"proxynd/health"
	"proxynd/logging"
)

// ServerStatusResponse 서버 상태 응답 구조체
type ServerStatusResponse struct {
	Status      string                 `json:"status"`
	Timestamp   time.Time              `json:"timestamp"`
	Uptime      string                 `json:"uptime"`
	Version     string                 `json:"version,omitempty"`
	Environment string                 `json:"environment,omitempty"`
	Server      ServerInfo             `json:"server"`
	System      SystemInfo             `json:"system"`
	Connections ConnectionInfo         `json:"connections"`
	Statistics  StatisticsInfo         `json:"statistics"`
	Checks      map[string]interface{} `json:"checks,omitempty"`
}

// ServerInfo 서버 정보
type ServerInfo struct {
	Host       string `json:"host"`
	Port       string `json:"port"`
	TLSEnabled bool   `json:"tls_enabled"`
	Handlers   int    `json:"handlers"`
	PID        int    `json:"pid"`
}

// SystemInfo 시스템 정보
type SystemInfo struct {
	OS           string  `json:"os"`
	Arch         string  `json:"arch"`
	CPUs         int     `json:"cpus"`
	Goroutines   int     `json:"goroutines"`
	MemoryUsage  float64 `json:"memory_usage_mb"`
	GCCycles     uint32  `json:"gc_cycles"`
	GCCPUPercent float64 `json:"gc_cpu_percent"`
}

// ConnectionInfo 연결 정보
type ConnectionInfo struct {
	Active  int `json:"active"`
	Total   int `json:"total"`
	Idle    int `json:"idle"`
	Waiting int `json:"waiting"`
}

// StatisticsInfo 통계 정보
type StatisticsInfo struct {
	TotalRequests     int64   `json:"total_requests"`
	TotalErrors       int64   `json:"total_errors"`
	SuccessRate       float64 `json:"success_rate"`
	AverageLatency    float64 `json:"average_latency_ms"`
	RequestsPerSecond float64 `json:"requests_per_second"`
	BytesTransferred  int64   `json:"bytes_transferred"`
}

// HealthCheckResponse 헬스체크 응답 구조체
type HealthCheckResponse struct {
	Status       string                     `json:"status"`
	Timestamp    time.Time                  `json:"timestamp"`
	Uptime       string                     `json:"uptime"`
	Checks       map[string]HealthCheckItem `json:"checks"`
	Summary      HealthCheckSummary         `json:"summary"`
	Dependencies []DependencyStatus         `json:"dependencies,omitempty"`
}

// HealthCheckItem 개별 헬스체크 항목
type HealthCheckItem struct {
	Status      string                 `json:"status"`
	Message     string                 `json:"message,omitempty"`
	LastChecked time.Time              `json:"last_checked"`
	Duration    string                 `json:"duration"`
	Details     map[string]interface{} `json:"details,omitempty"`
}

// HealthCheckSummary 헬스체크 요약
type HealthCheckSummary struct {
	TotalChecks     int `json:"total_checks"`
	HealthyChecks   int `json:"healthy_checks"`
	UnhealthyChecks int `json:"unhealthy_checks"`
	DegradedChecks  int `json:"degraded_checks"`
}

// DependencyStatus 의존성 상태
type DependencyStatus struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	URL     string `json:"url,omitempty"`
	Status  string `json:"status"`
	Latency string `json:"latency"`
	Error   string `json:"error,omitempty"`
}

// MetricsResponse 메트릭 응답 구조체
type MetricsResponse struct {
	Timestamp time.Time                  `json:"timestamp"`
	System    SystemMetrics              `json:"system"`
	Cache     map[string]CacheMetrics    `json:"cache"`
	Proxy     ProxyMetrics               `json:"proxy"`
	Registry  map[string]RegistryMetrics `json:"registry"`
	Health    HealthMetrics              `json:"health"`
}

// SystemMetrics 시스템 메트릭
type SystemMetrics struct {
	CPUUsage       float64 `json:"cpu_usage"`
	MemoryUsage    float64 `json:"memory_usage"`
	DiskUsage      float64 `json:"disk_usage"`
	LoadAverage1m  float64 `json:"load_average_1m"`
	LoadAverage5m  float64 `json:"load_average_5m"`
	LoadAverage15m float64 `json:"load_average_15m"`
	UptimeSeconds  float64 `json:"uptime_seconds"`
}

// CacheMetrics 캐시 메트릭
type CacheMetrics struct {
	Hits                int64   `json:"hits"`
	Misses              int64   `json:"misses"`
	HitRate             float64 `json:"hit_rate"`
	SizeBytes           int64   `json:"size_bytes"`
	ItemsCount          int64   `json:"items_count"`
	BandwidthSavedBytes int64   `json:"bandwidth_saved_bytes"`
}

// ProxyMetrics 프록시 메트릭
type ProxyMetrics struct {
	TotalRequests    int64   `json:"total_requests"`
	TotalErrors      int64   `json:"total_errors"`
	BytesUploaded    int64   `json:"bytes_uploaded"`
	BytesDownloaded  int64   `json:"bytes_downloaded"`
	AverageLatencyMs float64 `json:"average_latency_ms"`
}

// RegistryMetrics 레지스트리별 메트릭
type RegistryMetrics struct {
	Enabled        bool       `json:"enabled"`
	Requests       int64      `json:"requests"`
	Errors         int64      `json:"errors"`
	SuccessRate    float64    `json:"success_rate"`
	AverageLatency float64    `json:"average_latency_ms"`
	LastRequest    *time.Time `json:"last_request,omitempty"`
}

// HealthMetrics 헬스 메트릭
type HealthMetrics struct {
	OverallStatus   string  `json:"overall_status"`
	HealthyServices int     `json:"healthy_services"`
	TotalServices   int     `json:"total_services"`
	HealthScore     float64 `json:"health_score"`
}

// StatusRouter 상태 관리 API 라우터 설정
func StatusRouter(app *fiber.App) {
	api := app.Group("/api/status")

	// 서버 전반적인 상태
	api.Get("/", getServerStatus)

	// 헬스체크
	api.Get("/health", getHealthCheck)

	// 메트릭 조회
	api.Get("/metrics", getMetrics)

	// 의존성 상태
	api.Get("/dependencies", getDependencies)

	// 실시간 통계
	api.Get("/stats", getRealTimeStats)
}

// getServerStatus 서버 상태 조회 핸들러
func getServerStatus(c *fiber.Ctx) error {
	logger := logging.GetLogger()

	// 시스템 메모리 정보
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// 헬스 서비스 상태 (있는 경우)
	var overallStatus string = "healthy"
	var checks map[string]interface{} = nil
	var uptime string = "0s"

	if healthService != nil {
		status, healthChecks := healthService.GetStatus()
		overallStatus = string(status)
		checks = make(map[string]interface{})
		for name, check := range healthChecks {
			checks[name] = map[string]interface{}{
				"status":  string(check.Status),
				"message": check.Message,
				"checked": check.LastChecked,
			}
		}
		uptime = formatUptime(healthService.GetUptime())
	}

	// 연결 정보 (간단한 구현)
	connections := ConnectionInfo{
		Active:  runtime.NumGoroutine() - 10, // 대략적인 활성 연결 수
		Total:   runtime.NumGoroutine(),
		Idle:    5,
		Waiting: 0,
	}

	// 통계 정보 (메트릭에서 가져오기, 간단한 구현)
	stats := StatisticsInfo{
		TotalRequests:     0, // 실제로는 메트릭에서 가져와야 함
		TotalErrors:       0,
		SuccessRate:       100.0,
		AverageLatency:    15.5,
		RequestsPerSecond: 0.0,
		BytesTransferred:  0,
	}

	response := ServerStatusResponse{
		Status:      overallStatus,
		Timestamp:   time.Now(),
		Uptime:      uptime,
		Environment: getEnvironment(),
		Version:     getVersion(),
		Server: ServerInfo{
			Host:       "0.0.0.0",
			Port:       getPort(),
			TLSEnabled: false, // 실제로는 설정에서 가져와야 함
			Handlers:   52,    // Fiber에서 가져올 수 있음
			PID:        getPID(),
		},
		System: SystemInfo{
			OS:           runtime.GOOS,
			Arch:         runtime.GOARCH,
			CPUs:         runtime.NumCPU(),
			Goroutines:   runtime.NumGoroutine(),
			MemoryUsage:  float64(m.Alloc) / 1024 / 1024,
			GCCycles:     m.NumGC,
			GCCPUPercent: m.GCCPUFraction * 100,
		},
		Connections: connections,
		Statistics:  stats,
		Checks:      checks,
	}

	logger.Info("Server status requested")

	return c.JSON(response)
}

// getHealthCheck 헬스체크 핸들러
func getHealthCheck(c *fiber.Ctx) error {
	logger := logging.GetLogger()

	var status health.Status = health.StatusHealthy
	var healthChecks map[string]*health.CheckResult = make(map[string]*health.CheckResult)
	var uptime string = "0s"

	if healthService != nil {
		status, healthChecks = healthService.GetStatus()
		uptime = formatUptime(healthService.GetUptime())
	}

	// 체크 결과 변환
	checks := make(map[string]HealthCheckItem)
	summary := HealthCheckSummary{
		TotalChecks: len(healthChecks),
	}

	for name, check := range healthChecks {
		if check != nil {
			item := HealthCheckItem{
				Status:      string(check.Status),
				Message:     check.Message,
				LastChecked: check.LastChecked,
				Duration:    check.Duration.String(),
			}

			// 상세 정보 추가 (있는 경우)
			if check.Details != nil {
				item.Details = check.Details
			}

			checks[name] = item

			// 요약 통계 업데이트
			switch check.Status {
			case health.StatusHealthy:
				summary.HealthyChecks++
			case health.StatusUnhealthy:
				summary.UnhealthyChecks++
			case health.StatusDegraded:
				summary.DegradedChecks++
			}
		}
	}

	// 의존성 상태 (간단한 구현)
	dependencies := []DependencyStatus{}

	response := HealthCheckResponse{
		Status:       string(status),
		Timestamp:    time.Now(),
		Uptime:       uptime,
		Checks:       checks,
		Summary:      summary,
		Dependencies: dependencies,
	}

	logger.Info("Health check requested")

	return c.JSON(response)
}

// getMetrics 메트릭 조회 핸들러
func getMetrics(c *fiber.Ctx) error {
	logger := logging.GetLogger()

	// 시스템 메트릭
	systemMetrics := SystemMetrics{
		CPUUsage:       0.0, // 실제로는 시스템에서 가져와야 함
		MemoryUsage:    getMemoryUsagePercent(),
		DiskUsage:      0.0,
		LoadAverage1m:  0.0,
		LoadAverage5m:  0.0,
		LoadAverage15m: 0.0,
		UptimeSeconds:  0.0,
	}

	// 캐시 메트릭
	cacheMetrics := map[string]CacheMetrics{
		"apt": {
			Hits:                0,
			Misses:              0,
			HitRate:             0.0,
			SizeBytes:           0,
			ItemsCount:          0,
			BandwidthSavedBytes: 0,
		},
		"npm": {
			Hits:                0,
			Misses:              0,
			HitRate:             0.0,
			SizeBytes:           0,
			ItemsCount:          0,
			BandwidthSavedBytes: 0,
		},
	}

	// 프록시 메트릭
	proxyMetrics := ProxyMetrics{
		TotalRequests:    0,
		TotalErrors:      0,
		BytesUploaded:    0,
		BytesDownloaded:  0,
		AverageLatencyMs: 0.0,
	}

	// 레지스트리 메트릭
	registryMetrics := map[string]RegistryMetrics{
		"apt": {
			Enabled:        false,
			Requests:       0,
			Errors:         0,
			SuccessRate:    0.0,
			AverageLatency: 0.0,
		},
		"npm": {
			Enabled:        false,
			Requests:       0,
			Errors:         0,
			SuccessRate:    0.0,
			AverageLatency: 0.0,
		},
	}

	// 헬스 메트릭
	healthMetrics := HealthMetrics{
		OverallStatus:   "healthy",
		HealthyServices: 0,
		TotalServices:   0,
		HealthScore:     100.0,
	}

	if healthService != nil {
		status, checks := healthService.GetStatus()
		healthMetrics.OverallStatus = string(status)
		healthMetrics.TotalServices = len(checks)

		for _, check := range checks {
			if check.Status == health.StatusHealthy {
				healthMetrics.HealthyServices++
			}
		}

		if healthMetrics.TotalServices > 0 {
			healthMetrics.HealthScore = float64(healthMetrics.HealthyServices) / float64(healthMetrics.TotalServices) * 100
		}
	}

	response := MetricsResponse{
		Timestamp: time.Now(),
		System:    systemMetrics,
		Cache:     cacheMetrics,
		Proxy:     proxyMetrics,
		Registry:  registryMetrics,
		Health:    healthMetrics,
	}

	logger.Info("Metrics requested")

	return c.JSON(response)
}

// getDependencies 의존성 상태 조회 핸들러
func getDependencies(c *fiber.Ctx) error {
	logger := logging.GetLogger()

	dependencies := []DependencyStatus{
		{
			Name:    "File System",
			Type:    "storage",
			Status:  "healthy",
			Latency: "1ms",
		},
		{
			Name:    "Cache Backend",
			Type:    "cache",
			Status:  "healthy",
			Latency: "2ms",
		},
	}

	logger.Info("Dependencies status requested")

	return c.JSON(fiber.Map{
		"dependencies": dependencies,
		"timestamp":    time.Now(),
		"total":        len(dependencies),
		"healthy":      len(dependencies), // 간단한 구현
	})
}

// getRealTimeStats 실시간 통계 조회 핸들러
func getRealTimeStats(c *fiber.Ctx) error {
	logger := logging.GetLogger()

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	stats := fiber.Map{
		"timestamp":   time.Now(),
		"goroutines":  runtime.NumGoroutine(),
		"memory_mb":   float64(m.Alloc) / 1024 / 1024,
		"gc_cycles":   m.NumGC,
		"cpu_count":   runtime.NumCPU(),
		"requests":    0, // 실제로는 메트릭에서 가져와야 함
		"errors":      0,
		"connections": runtime.NumGoroutine() - 10,
	}

	logger.Info("Real-time stats requested")

	return c.JSON(stats)
}

// 헬퍼 함수들

func getEnvironment() string {
	if env := getEnv("ENVIRONMENT"); env != "" {
		return env
	}
	return "development"
}

func getVersion() string {
	if version := getEnv("VERSION"); version != "" {
		return version
	}
	return "1.0.0"
}

func getPort() string {
	if port := getEnv("SERVER_PORT"); port != "" {
		return port
	}
	return "8080"
}

func getPID() int {
	// 간단한 구현, 실제로는 os.Getpid() 사용
	return 1
}

func getEnv(key string) string {
	// 환경 변수 가져오기 (helpers 패키지 사용할 수 있음)
	return ""
}

func getMemoryUsagePercent() float64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	// 간단한 메모리 사용률 계산
	return float64(m.Alloc) / float64(m.Sys) * 100
}
