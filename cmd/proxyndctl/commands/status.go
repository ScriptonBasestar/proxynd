package commands

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
)

// ServerStatusResponse API 응답 구조체
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

// NewStatusCmd 상태 확인 명령어 생성
func NewStatusCmd() *cobra.Command {
	var detailed bool

	cmd := &cobra.Command{
		Use:   "status",
		Short: "서버 상태 확인",
		Long:  "ProxyND 서버의 전반적인 상태를 확인합니다.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runServerStatus(detailed)
		},
	}

	cmd.Flags().BoolVarP(&detailed, "detailed", "d", false, "상세 정보 표시")

	return cmd
}

// NewHealthCmd 헬스체크 명령어 생성
func NewHealthCmd() *cobra.Command {
	var showDependencies bool

	cmd := &cobra.Command{
		Use:   "health",
		Short: "헬스체크 수행",
		Long:  "ProxyND 서버의 헬스체크를 수행하고 업스트림 연결 상태를 확인합니다.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runHealthCheck(showDependencies)
		},
	}

	cmd.Flags().BoolVarP(&showDependencies, "dependencies", "d", false, "의존성 상태 표시")

	return cmd
}

// NewMetricsCmd 메트릭 조회 명령어 생성
func NewMetricsCmd() *cobra.Command {
	var (
		category string
		detailed bool
	)

	cmd := &cobra.Command{
		Use:   "metrics",
		Short: "메트릭 조회",
		Long:  "ProxyND 서버의 Prometheus 메트릭을 사용자 친화적 형태로 조회합니다.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMetrics(category, detailed)
		},
	}

	cmd.Flags().StringVarP(&category, "category", "c", "", "메트릭 카테고리 (system, cache, proxy, registry, health)")
	cmd.Flags().BoolVarP(&detailed, "detailed", "d", false, "상세 메트릭 표시")

	return cmd
}

// runServerStatus 서버 상태 조회 실행
func runServerStatus(detailed bool) error {
	serverURL := getServerURL()
	apiURL := fmt.Sprintf("%s/api/status/", serverURL)

	// API 호출
	resp, err := http.Get(apiURL)
	if err != nil {
		return fmt.Errorf("서버 연결 실패: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API 요청 실패: HTTP %d", resp.StatusCode)
	}

	// 응답 파싱
	var result ServerStatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("응답 파싱 실패: %v", err)
	}

	// 결과 출력
	outputFormat := getOutputFormat()
	switch outputFormat {
	case "json":
		return outputJSON(result)
	case "yaml":
		return outputYAML(result)
	default:
		return outputServerStatusTable(result, detailed)
	}
}

// runHealthCheck 헬스체크 실행
func runHealthCheck(showDependencies bool) error {
	serverURL := getServerURL()
	apiURL := fmt.Sprintf("%s/api/status/health", serverURL)

	// API 호출
	resp, err := http.Get(apiURL)
	if err != nil {
		return fmt.Errorf("서버 연결 실패: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API 요청 실패: HTTP %d", resp.StatusCode)
	}

	// 응답 파싱
	var result HealthCheckResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("응답 파싱 실패: %v", err)
	}

	// 결과 출력
	outputFormat := getOutputFormat()
	switch outputFormat {
	case "json":
		return outputJSON(result)
	case "yaml":
		return outputYAML(result)
	default:
		return outputHealthCheckTable(result, showDependencies)
	}
}

// runMetrics 메트릭 조회 실행
func runMetrics(category string, detailed bool) error {
	serverURL := getServerURL()
	apiURL := fmt.Sprintf("%s/api/status/metrics", serverURL)

	// API 호출
	resp, err := http.Get(apiURL)
	if err != nil {
		return fmt.Errorf("서버 연결 실패: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API 요청 실패: HTTP %d", resp.StatusCode)
	}

	// 응답 파싱
	var result MetricsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("응답 파싱 실패: %v", err)
	}

	// 결과 출력
	outputFormat := getOutputFormat()
	switch outputFormat {
	case "json":
		return outputJSON(result)
	case "yaml":
		return outputYAML(result)
	default:
		return outputMetricsTable(result, category, detailed)
	}
}

// outputServerStatusTable 서버 상태를 테이블 형태로 출력
func outputServerStatusTable(result ServerStatusResponse, detailed bool) error {
	// 상태 아이콘
	statusIcon := "✅"
	if result.Status != "healthy" {
		statusIcon = "❌"
	}

	fmt.Printf("🖥️ ProxyND 서버 상태: %s %s\n\n", statusIcon, result.Status)

	// 기본 정보
	fmt.Printf("📊 기본 정보:\n")
	fmt.Printf("  업타임: %s\n", result.Uptime)
	if result.Version != "" {
		fmt.Printf("  버전: %s\n", result.Version)
	}
	if result.Environment != "" {
		fmt.Printf("  환경: %s\n", result.Environment)
	}
	fmt.Printf("  타임스탬프: %s\n\n", result.Timestamp.Format("2006-01-02 15:04:05"))

	// 서버 정보
	fmt.Printf("🌐 서버 정보:\n")
	fmt.Printf("  주소: %s:%s\n", result.Server.Host, result.Server.Port)
	fmt.Printf("  TLS: %s\n", boolToStatus(result.Server.TLSEnabled))
	fmt.Printf("  핸들러 수: %d개\n", result.Server.Handlers)
	fmt.Printf("  PID: %d\n\n", result.Server.PID)

	// 시스템 정보
	fmt.Printf("💻 시스템 정보:\n")
	fmt.Printf("  OS/아키텍처: %s/%s\n", result.System.OS, result.System.Arch)
	fmt.Printf("  CPU: %d코어\n", result.System.CPUs)
	fmt.Printf("  고루틴: %d개\n", result.System.Goroutines)
	fmt.Printf("  메모리 사용량: %.1f MB\n", result.System.MemoryUsage)
	fmt.Printf("  GC 사이클: %d회\n", result.System.GCCycles)
	fmt.Printf("  GC CPU 사용률: %.2f%%\n\n", result.System.GCCPUPercent)

	// 연결 정보
	fmt.Printf("🔗 연결 정보:\n")
	fmt.Printf("  활성 연결: %d개\n", result.Connections.Active)
	fmt.Printf("  전체 연결: %d개\n", result.Connections.Total)
	fmt.Printf("  유휴 연결: %d개\n", result.Connections.Idle)
	fmt.Printf("  대기 연결: %d개\n\n", result.Connections.Waiting)

	// 통계 정보
	fmt.Printf("📈 통계 정보:\n")
	fmt.Printf("  총 요청: %d건\n", result.Statistics.TotalRequests)
	fmt.Printf("  총 오류: %d건\n", result.Statistics.TotalErrors)
	fmt.Printf("  성공률: %.1f%%\n", result.Statistics.SuccessRate)
	fmt.Printf("  평균 지연시간: %.1f ms\n", result.Statistics.AverageLatency)
	fmt.Printf("  초당 요청: %.1f req/s\n", result.Statistics.RequestsPerSecond)
	fmt.Printf("  전송 바이트: %s\n", formatBytes(result.Statistics.BytesTransferred))

	// 헬스체크 정보 (상세 모드)
	if detailed && len(result.Checks) > 0 {
		fmt.Printf("\n🔍 헬스체크 상태:\n")
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "이름\t상태\t메시지")
		fmt.Fprintln(w, "----\t----\t------")

		for name, check := range result.Checks {
			if checkMap, ok := check.(map[string]interface{}); ok {
				status := "❓"
				if statusStr, exists := checkMap["status"].(string); exists {
					switch statusStr {
					case "healthy":
						status = "✅"
					case "unhealthy":
						status = "❌"
					case "degraded":
						status = "⚠️"
					}
				}

				message := ""
				if msg, exists := checkMap["message"].(string); exists {
					message = msg
				}

				fmt.Fprintf(w, "%s\t%s\t%s\n", name, status, message)
			}
		}
		w.Flush()
	}

	return nil
}

// outputHealthCheckTable 헬스체크 결과를 테이블 형태로 출력
func outputHealthCheckTable(result HealthCheckResponse, showDependencies bool) error {
	// 전체 상태
	statusIcon := "✅"
	if result.Status != "healthy" {
		statusIcon = "❌"
	}

	fmt.Printf("🏥 헬스체크 결과: %s %s\n\n", statusIcon, result.Status)

	// 요약 정보
	summary := result.Summary
	fmt.Printf("📊 요약:\n")
	fmt.Printf("  전체 체크: %d개\n", summary.TotalChecks)
	fmt.Printf("  정상: %d개\n", summary.HealthyChecks)
	fmt.Printf("  비정상: %d개\n", summary.UnhealthyChecks)
	fmt.Printf("  성능저하: %d개\n", summary.DegradedChecks)
	fmt.Printf("  업타임: %s\n", result.Uptime)
	fmt.Printf("  체크 시간: %s\n\n", result.Timestamp.Format("2006-01-02 15:04:05"))

	// 개별 체크 결과
	if len(result.Checks) > 0 {
		fmt.Println("🔍 개별 체크 결과:")
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "이름\t상태\t마지막체크\t소요시간\t메시지")
		fmt.Fprintln(w, "----\t----\t--------\t--------\t------")

		for name, check := range result.Checks {
			status := "❓"
			switch check.Status {
			case "healthy":
				status = "✅"
			case "unhealthy":
				status = "❌"
			case "degraded":
				status = "⚠️"
			}

			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
				name,
				status,
				check.LastChecked.Format("15:04:05"),
				check.Duration,
				check.Message,
			)
		}
		w.Flush()
		fmt.Println()
	}

	// 의존성 상태 (요청된 경우)
	if showDependencies && len(result.Dependencies) > 0 {
		fmt.Println("🔗 의존성 상태:")
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "이름\t타입\t상태\t지연시간\t오류")
		fmt.Fprintln(w, "----\t----\t----\t--------\t----")

		for _, dep := range result.Dependencies {
			status := "❓"
			switch dep.Status {
			case "healthy":
				status = "✅"
			case "unhealthy":
				status = "❌"
			case "degraded":
				status = "⚠️"
			}

			errorMsg := dep.Error
			if errorMsg == "" {
				errorMsg = "-"
			}

			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
				dep.Name,
				dep.Type,
				status,
				dep.Latency,
				errorMsg,
			)
		}
		w.Flush()
	}

	return nil
}

// outputMetricsTable 메트릭을 테이블 형태로 출력
func outputMetricsTable(result MetricsResponse, category string, detailed bool) error {
	fmt.Printf("📊 메트릭 정보 (%s)\n\n", result.Timestamp.Format("2006-01-02 15:04:05"))

	// 카테고리별 출력 또는 전체 출력
	if category == "" || category == "system" {
		fmt.Println("💻 시스템 메트릭:")
		fmt.Printf("  CPU 사용률: %.1f%%\n", result.System.CPUUsage)
		fmt.Printf("  메모리 사용률: %.1f%%\n", result.System.MemoryUsage)
		fmt.Printf("  디스크 사용률: %.1f%%\n", result.System.DiskUsage)
		fmt.Printf("  로드 애버리지: %.2f, %.2f, %.2f\n",
			result.System.LoadAverage1m,
			result.System.LoadAverage5m,
			result.System.LoadAverage15m)
		fmt.Printf("  업타임: %.0f초\n\n", result.System.UptimeSeconds)
	}

	if category == "" || category == "cache" {
		fmt.Println("🗄️ 캐시 메트릭:")
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "타입\t히트\t미스\t히트률\t크기\t항목수")
		fmt.Fprintln(w, "----\t----\t----\t------\t----\t------")

		for cacheType, metrics := range result.Cache {
			fmt.Fprintf(w, "%s\t%d\t%d\t%.1f%%\t%s\t%d\n",
				cacheType,
				metrics.Hits,
				metrics.Misses,
				metrics.HitRate,
				formatBytes(metrics.SizeBytes),
				metrics.ItemsCount,
			)
		}
		w.Flush()
		fmt.Println()
	}

	if category == "" || category == "proxy" {
		fmt.Println("🔀 프록시 메트릭:")
		fmt.Printf("  총 요청: %d건\n", result.Proxy.TotalRequests)
		fmt.Printf("  총 오류: %d건\n", result.Proxy.TotalErrors)
		fmt.Printf("  업로드: %s\n", formatBytes(result.Proxy.BytesUploaded))
		fmt.Printf("  다운로드: %s\n", formatBytes(result.Proxy.BytesDownloaded))
		fmt.Printf("  평균 지연시간: %.1f ms\n\n", result.Proxy.AverageLatencyMs)
	}

	if category == "" || category == "registry" {
		fmt.Println("📦 레지스트리 메트릭:")
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "타입\t활성\t요청\t오류\t성공률\t지연시간")
		fmt.Fprintln(w, "----\t----\t----\t----\t------\t--------")

		for regType, metrics := range result.Registry {
			enabled := "❌"
			if metrics.Enabled {
				enabled = "✅"
			}

			fmt.Fprintf(w, "%s\t%s\t%d\t%d\t%.1f%%\t%.1f ms\n",
				regType,
				enabled,
				metrics.Requests,
				metrics.Errors,
				metrics.SuccessRate,
				metrics.AverageLatency,
			)
		}
		w.Flush()
		fmt.Println()
	}

	if category == "" || category == "health" {
		fmt.Println("🏥 헬스 메트릭:")
		fmt.Printf("  전체 상태: %s\n", result.Health.OverallStatus)
		fmt.Printf("  정상 서비스: %d/%d개\n", result.Health.HealthyServices, result.Health.TotalServices)
		fmt.Printf("  헬스 스코어: %.1f%%\n", result.Health.HealthScore)
	}

	return nil
}

// boolToStatus 불린 값을 상태 문자열로 변환
func boolToStatus(value bool) string {
	if value {
		return "활성화"
	}
	return "비활성화"
}
