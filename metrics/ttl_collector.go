package metrics

import (
	"math"
	"sort"
	"sync"
	"time"
)

// TTLStats TTL 통계 정보
type TTLStats struct {
	TotalCalculations   int64     `json:"total_calculations"`
	AverageTTL          float64   `json:"average_ttl"`
	MinTTL              int       `json:"min_ttl"`
	MaxTTL              int       `json:"max_ttl"`
	MedianTTL           float64   `json:"median_ttl"`
	P90TTL              float64   `json:"p90_ttl"`
	P95TTL              float64   `json:"p95_ttl"`
	P99TTL              float64   `json:"p99_ttl"`
	EarlyExpirationRate float64   `json:"early_expiration_rate"`
	HeaderBasedRate     float64   `json:"header_based_rate"`
	PatternBasedRate    float64   `json:"pattern_based_rate"`
	LastUpdated         time.Time `json:"last_updated"`
}

// TTLEntry TTL 계산 기록
type TTLEntry struct {
	PackageName   string    `json:"package_name"`
	PackageType   string    `json:"package_type"`
	CalculatedTTL int       `json:"calculated_ttl"`
	Source        string    `json:"source"`
	Timestamp     time.Time `json:"timestamp"`
	IsExpired     bool      `json:"is_expired"`
	ExpiryReason  string    `json:"expiry_reason,omitempty"`
}

// TTLCollector TTL 통계 수집기
type TTLCollector struct {
	mu             sync.RWMutex
	entries        []TTLEntry
	maxEntries     int
	registryStats  map[string]*TTLStats
	globalStats    *TTLStats
	lastCalculated time.Time
}

// NewTTLCollector 새 TTL 수집기 생성
func NewTTLCollector(maxEntries int) *TTLCollector {
	if maxEntries <= 0 {
		maxEntries = 10000 // 기본값
	}

	return &TTLCollector{
		entries:       make([]TTLEntry, 0),
		maxEntries:    maxEntries,
		registryStats: make(map[string]*TTLStats),
		globalStats:   &TTLStats{},
	}
}

// RecordTTLCalculation TTL 계산 기록
func (c *TTLCollector) RecordTTLCalculation(packageName, packageType string, ttl int, source string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry := TTLEntry{
		PackageName:   packageName,
		PackageType:   packageType,
		CalculatedTTL: ttl,
		Source:        source,
		Timestamp:     time.Now(),
	}

	// 메트릭 기록
	metrics := GetMetrics()
	if metrics != nil {
		metrics.TTLCalculationsTotal.WithLabelValues(packageType, packageName, source).Inc()
		metrics.TTLSourceTotal.WithLabelValues(source).Inc()
		metrics.CacheTTLGauge.WithLabelValues(packageType, packageName, source).Set(float64(ttl))
	}

	// 엔트리 추가
	c.entries = append(c.entries, entry)

	// 최대 크기 초과 시 오래된 항목 제거
	if len(c.entries) > c.maxEntries {
		c.entries = c.entries[len(c.entries)-c.maxEntries:]
	}

	// 통계 갱신 (일정 간격으로만)
	if time.Since(c.lastCalculated) > 30*time.Second {
		c.updateStatistics()
		c.lastCalculated = time.Now()
	}
}

// RecordTTLExpiration TTL 만료 기록
func (c *TTLCollector) RecordTTLExpiration(packageType, expirationType string) {
	metrics := GetMetrics()
	if metrics != nil {
		metrics.TTLExpirationTotal.WithLabelValues(packageType, expirationType).Inc()
	}

	// 조기 만료 추적을 위한 엔트리 업데이트
	c.mu.Lock()
	defer c.mu.Unlock()

	for i := len(c.entries) - 1; i >= 0; i-- {
		entry := &c.entries[i]
		if entry.PackageType == packageType && !entry.IsExpired {
			// 최근 항목 중 해당하는 것 찾아서 만료 표시
			if expirationType == "early" {
				entry.IsExpired = true
				entry.ExpiryReason = "early_expiration"
				break
			}
		}
	}
}

// GetStats 통계 조회
func (c *TTLCollector) GetStats(registryType string) *TTLStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if registryType == "" {
		return c.globalStats
	}

	if stats, exists := c.registryStats[registryType]; exists {
		return stats
	}

	return &TTLStats{}
}

// GetAllStats 모든 레지스트리 통계 조회
func (c *TTLCollector) GetAllStats() map[string]*TTLStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make(map[string]*TTLStats)
	result["global"] = c.globalStats

	for registryType, stats := range c.registryStats {
		result[registryType] = stats
	}

	return result
}

// GetRecentEntries 최근 엔트리 조회
func (c *TTLCollector) GetRecentEntries(limit int) []TTLEntry {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if limit <= 0 || limit > len(c.entries) {
		limit = len(c.entries)
	}

	// 최근 항목부터 반환
	start := len(c.entries) - limit
	if start < 0 {
		start = 0
	}

	return c.entries[start:]
}

// updateStatistics 통계 업데이트 (내부 함수, 락 획득 후 호출)
func (c *TTLCollector) updateStatistics() {
	// 글로벌 통계 계산
	c.globalStats = c.calculateStats("")

	// 레지스트리별 통계 계산
	registryTypes := make(map[string]bool)
	for _, entry := range c.entries {
		registryTypes[entry.PackageType] = true
	}

	for registryType := range registryTypes {
		c.registryStats[registryType] = c.calculateStats(registryType)
	}

	// Prometheus 메트릭 업데이트
	c.updatePrometheusMetrics()
}

// calculateStats 통계 계산
func (c *TTLCollector) calculateStats(registryType string) *TTLStats {
	var filteredEntries []TTLEntry
	for _, entry := range c.entries {
		if registryType == "" || entry.PackageType == registryType {
			filteredEntries = append(filteredEntries, entry)
		}
	}

	if len(filteredEntries) == 0 {
		return &TTLStats{LastUpdated: time.Now()}
	}

	// TTL 값들 추출 및 정렬
	ttlValues := make([]int, len(filteredEntries))
	var sum int64
	sourceCounts := make(map[string]int)
	earlyExpirations := 0

	for i, entry := range filteredEntries {
		ttlValues[i] = entry.CalculatedTTL
		sum += int64(entry.CalculatedTTL)
		sourceCounts[entry.Source]++

		if entry.IsExpired && entry.ExpiryReason == "early_expiration" {
			earlyExpirations++
		}
	}

	sort.Ints(ttlValues)

	// 통계 계산
	total := int64(len(filteredEntries))
	average := float64(sum) / float64(total)
	min := ttlValues[0]
	max := ttlValues[len(ttlValues)-1]

	median := calculatePercentile(ttlValues, 50)
	p90 := calculatePercentile(ttlValues, 90)
	p95 := calculatePercentile(ttlValues, 95)
	p99 := calculatePercentile(ttlValues, 99)

	// 비율 계산
	earlyExpirationRate := float64(earlyExpirations) / float64(total)
	headerBasedRate := float64(sourceCounts["cache_header"]) / float64(total)
	patternBasedRate := float64(sourceCounts["pattern_match"]) / float64(total)

	return &TTLStats{
		TotalCalculations:   total,
		AverageTTL:          average,
		MinTTL:              min,
		MaxTTL:              max,
		MedianTTL:           median,
		P90TTL:              p90,
		P95TTL:              p95,
		P99TTL:              p99,
		EarlyExpirationRate: earlyExpirationRate,
		HeaderBasedRate:     headerBasedRate,
		PatternBasedRate:    patternBasedRate,
		LastUpdated:         time.Now(),
	}
}

// calculatePercentile 백분위수 계산
func calculatePercentile(sortedValues []int, percentile float64) float64 {
	if len(sortedValues) == 0 {
		return 0
	}

	if percentile <= 0 {
		return float64(sortedValues[0])
	}
	if percentile >= 100 {
		return float64(sortedValues[len(sortedValues)-1])
	}

	index := (percentile / 100.0) * float64(len(sortedValues)-1)
	lower := int(math.Floor(index))
	upper := int(math.Ceil(index))

	if lower == upper {
		return float64(sortedValues[lower])
	}

	// 선형 보간
	weight := index - float64(lower)
	return float64(sortedValues[lower])*(1-weight) + float64(sortedValues[upper])*weight
}

// updatePrometheusMetrics Prometheus 메트릭 업데이트
func (c *TTLCollector) updatePrometheusMetrics() {
	metrics := GetMetrics()
	if metrics == nil {
		return
	}

	// 글로벌 통계 업데이트
	stats := c.globalStats
	metrics.TTLStatistics.WithLabelValues("global", "avg").Set(stats.AverageTTL)
	metrics.TTLStatistics.WithLabelValues("global", "min").Set(float64(stats.MinTTL))
	metrics.TTLStatistics.WithLabelValues("global", "max").Set(float64(stats.MaxTTL))
	metrics.TTLStatistics.WithLabelValues("global", "p50").Set(stats.MedianTTL)
	metrics.TTLStatistics.WithLabelValues("global", "p90").Set(stats.P90TTL)
	metrics.TTLStatistics.WithLabelValues("global", "p95").Set(stats.P95TTL)
	metrics.TTLStatistics.WithLabelValues("global", "p99").Set(stats.P99TTL)

	// 레지스트리별 통계 업데이트
	for registryType, stats := range c.registryStats {
		metrics.TTLStatistics.WithLabelValues(registryType, "avg").Set(stats.AverageTTL)
		metrics.TTLStatistics.WithLabelValues(registryType, "min").Set(float64(stats.MinTTL))
		metrics.TTLStatistics.WithLabelValues(registryType, "max").Set(float64(stats.MaxTTL))
		metrics.TTLStatistics.WithLabelValues(registryType, "p50").Set(stats.MedianTTL)
		metrics.TTLStatistics.WithLabelValues(registryType, "p90").Set(stats.P90TTL)
		metrics.TTLStatistics.WithLabelValues(registryType, "p95").Set(stats.P95TTL)
		metrics.TTLStatistics.WithLabelValues(registryType, "p99").Set(stats.P99TTL)
	}
}

// 전역 TTL 수집기 인스턴스
var globalTTLCollector *TTLCollector

// InitTTLCollector TTL 수집기 초기화
func InitTTLCollector(maxEntries int) {
	globalTTLCollector = NewTTLCollector(maxEntries)
}

// GetTTLCollector 전역 TTL 수집기 반환
func GetTTLCollector() *TTLCollector {
	if globalTTLCollector == nil {
		InitTTLCollector(10000) // 기본값으로 초기화
	}
	return globalTTLCollector
}

// RecordTTLCalculation 편의 함수
func RecordTTLCalculation(packageName, packageType string, ttl int, source string) {
	collector := GetTTLCollector()
	collector.RecordTTLCalculation(packageName, packageType, ttl, source)
}

// RecordTTLExpiration 편의 함수
func RecordTTLExpiration(packageType, expirationType string) {
	collector := GetTTLCollector()
	collector.RecordTTLExpiration(packageType, expirationType)
}
