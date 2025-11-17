package enterprise

import (
	"time"
)

// DashboardOverview represents high-level metrics for the dashboard
type DashboardOverview struct {
	TotalRequests    int64   `json:"total_requests"`
	CacheHitRate     float64 `json:"cache_hit_rate"`
	AvgResponseTime  float64 `json:"avg_response_time_ms"`
	BandwidthUsed    int64   `json:"bandwidth_used_bytes"`
	ActiveUsers      int     `json:"active_users"`
	TopPackages      []PackageUsage `json:"top_packages"`
	RequestsByPM     map[string]int64 `json:"requests_by_pm"`
	ErrorRate        float64 `json:"error_rate"`
	Timestamp        time.Time `json:"timestamp"`
}

// PackageUsage represents usage stats for a package
type PackageUsage struct {
	PackageManager string `json:"package_manager"`
	PackageName    string `json:"package_name"`
	Version        string `json:"version,omitempty"`
	RequestCount   int64  `json:"request_count"`
	BytesServed    int64  `json:"bytes_served"`
}

// UsageStats represents detailed usage statistics
type UsageStats struct {
	TimeRange        string              `json:"time_range"`
	TotalRequests    int64               `json:"total_requests"`
	UniquePackages   int                 `json:"unique_packages"`
	UniqueUsers      int                 `json:"unique_users"`
	ByPackageManager map[string]PMStats  `json:"by_package_manager"`
	ByUser           []UserStats         `json:"by_user"`
	ByTimeOfDay      map[string]int64    `json:"by_time_of_day"`
	Trends           []TrendDataPoint    `json:"trends"`
}

// PMStats represents stats for a package manager
type PMStats struct {
	Name         string  `json:"name"`
	Requests     int64   `json:"requests"`
	CacheHits    int64   `json:"cache_hits"`
	CacheMisses  int64   `json:"cache_misses"`
	HitRate      float64 `json:"hit_rate"`
	BytesServed  int64   `json:"bytes_served"`
}

// UserStats represents stats for a user
type UserStats struct {
	UserID      string `json:"user_id"`
	UserEmail   string `json:"user_email,omitempty"`
	Requests    int64  `json:"requests"`
	BytesUsed   int64  `json:"bytes_used"`
	TopPackages []string `json:"top_packages"`
}

// TrendDataPoint represents a time-series data point
type TrendDataPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
	Label     string    `json:"label"`
}

// PerformanceMetrics represents performance analysis
type PerformanceMetrics struct {
	TimeRange       string             `json:"time_range"`
	AvgLatency      float64            `json:"avg_latency_ms"`
	P50Latency      float64            `json:"p50_latency_ms"`
	P95Latency      float64            `json:"p95_latency_ms"`
	P99Latency      float64            `json:"p99_latency_ms"`
	Throughput      float64            `json:"throughput_rps"`
	ErrorRate       float64            `json:"error_rate"`
	ByEndpoint      map[string]EndpointMetrics `json:"by_endpoint"`
	LatencyTrends   []TrendDataPoint   `json:"latency_trends"`
}

// EndpointMetrics represents metrics for a specific endpoint
type EndpointMetrics struct {
	Endpoint    string  `json:"endpoint"`
	Requests    int64   `json:"requests"`
	AvgLatency  float64 `json:"avg_latency_ms"`
	P95Latency  float64 `json:"p95_latency_ms"`
	ErrorCount  int64   `json:"error_count"`
	ErrorRate   float64 `json:"error_rate"`
}

// CacheEfficiency represents cache performance analysis
type CacheEfficiency struct {
	TimeRange        string             `json:"time_range"`
	OverallHitRate   float64            `json:"overall_hit_rate"`
	TotalHits        int64              `json:"total_hits"`
	TotalMisses      int64              `json:"total_misses"`
	CacheSizeBytes   int64              `json:"cache_size_bytes"`
	EvictionCount    int64              `json:"eviction_count"`
	ByPackageManager map[string]float64 `json:"by_package_manager"`
	HitRateTrends    []TrendDataPoint   `json:"hit_rate_trends"`
	TopCachedPackages []PackageUsage    `json:"top_cached_packages"`
}

// Report represents a saved custom report
type Report struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Type        string                 `json:"type"` // usage, performance, cost, security
	Schedule    string                 `json:"schedule,omitempty"` // cron expression
	Filters     map[string]interface{} `json:"filters"`
	CreatedBy   string                 `json:"created_by"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// CostAnalysis represents cost breakdown
type CostAnalysis struct {
	TimeRange        string              `json:"time_range"`
	TotalCost        float64             `json:"total_cost"`
	BandwidthCost    float64             `json:"bandwidth_cost"`
	StorageCost      float64             `json:"storage_cost"`
	ComputeCost      float64             `json:"compute_cost"`
	ByPackageManager map[string]float64  `json:"by_package_manager"`
	ByUser           map[string]float64  `json:"by_user"`
	CostTrends       []TrendDataPoint    `json:"cost_trends"`
}

// Analytics errors
var (
	ErrInvalidTimeRange = &DomainError{Code: "INVALID_TIME_RANGE", Message: "Invalid time range specified"}
	ErrReportNotFound   = &DomainError{Code: "REPORT_NOT_FOUND", Message: "Report not found"}
	ErrInvalidReportType = &DomainError{Code: "INVALID_REPORT_TYPE", Message: "Invalid report type"}
)
