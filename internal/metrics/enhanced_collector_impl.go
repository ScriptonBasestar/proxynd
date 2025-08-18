package metrics

import (
	"strconv"
	"time"

	"proxynd/internal/logging"
)

// Enhanced collector 메서드 구현

// updatePopularPackages 인기 패키지 업데이트
func (ec *EnhancedMetricsCollector) updatePopularPackages() {
	ec.popularPackages.UpdateRankings()

	// 프로메테우스 메트릭 업데이트
	for registryType, rankings := range ec.popularPackages.rankings {
		for i, packageName := range rankings {
			rank := strconv.Itoa(i + 1)
			ec.metrics.PopularPackagesGauge.WithLabelValues(registryType, packageName, rank).Set(float64(i + 1))
		}
	}
}

// updateThroughputMetrics 처리량 메트릭 업데이트
func (ec *EnhancedMetricsCollector) updateThroughputMetrics() {
	registryTypes := []string{"npm", "maven", "apt", "docker", "pypi", "yum", "apk"}
	timeWindows := []string{"1m", "5m", "15m"}

	for _, registryType := range registryTypes {
		for _, window := range timeWindows {
			var throughput float64

			switch window {
			case "1m":
				throughput = ec.throughputTracker.CalculateThroughput(registryType)
			case "5m":
				// 5분 평균 계산
				throughput = ec.calculateThroughputForWindow(registryType, 5*time.Minute)
			case "15m":
				// 15분 평균 계산
				throughput = ec.calculateThroughputForWindow(registryType, 15*time.Minute)
			}

			ec.metrics.ThroughputRequestsPerSec.WithLabelValues(registryType, window).Set(throughput)
		}
	}
}

// calculateThroughputForWindow 특정 시간 윈도우의 처리량 계산
func (ec *EnhancedMetricsCollector) calculateThroughputForWindow(registryType string, window time.Duration) float64 {
	ec.throughputTracker.mu.RLock()
	defer ec.throughputTracker.mu.RUnlock()

	now := time.Now()
	cutoff := now.Add(-window)
	totalRequests := int64(0)

	for windowKey, timeWindow := range ec.throughputTracker.windows {
		if timeWindow.Start.After(cutoff) && timeWindow.Start.Before(now) {
			if len(windowKey) > len(registryType) && windowKey[:len(registryType)] == registryType {
				totalRequests += timeWindow.RequestCount
			}
		}
	}

	return float64(totalRequests) / window.Seconds()
}

// updateRetryRates 재시도율 업데이트
func (ec *EnhancedMetricsCollector) updateRetryRates() {
	ec.retryTracker.mu.RLock()
	defer ec.retryTracker.mu.RUnlock()

	for _, stats := range ec.retryTracker.retryStats {
		ec.metrics.RetrySuccessRate.WithLabelValues(stats.RegistryType, stats.RetryReason).Set(stats.SuccessRate)
	}
}

// updateUpstreamFailureRates 업스트림 실패율 업데이트
func (ec *EnhancedMetricsCollector) updateUpstreamFailureRates() {
	ec.upstreamTracker.mu.RLock()
	defer ec.upstreamTracker.mu.RUnlock()

	for _, stats := range ec.upstreamTracker.upstreamStats {
		ec.metrics.UpstreamFailureRate.
			WithLabelValues(
				stats.RegistryType,
				stats.Upstream,
				stats.FailureType,
			).
			Set(stats.FailureRate)
	}
}

// updateUniqueUsers 고유 사용자 수 업데이트
func (ec *EnhancedMetricsCollector) updateUniqueUsers() {
	registryTypes := []string{"npm", "maven", "apt", "docker", "pypi", "yum", "apk"}
	timeWindows := []string{"1h", "24h", "7d"}
	userTypes := []string{"authenticated", "anonymous", "bot"}

	for _, registryType := range registryTypes {
		for _, timeWindow := range timeWindows {
			for _, userType := range userTypes {
				count := ec.userTracker.GetActiveUserCount(registryType, timeWindow)
				ec.metrics.UniqueUsers.WithLabelValues(registryType, timeWindow, userType).Set(float64(count))
			}
		}
	}
}

// calculateLatencyPercentiles 지연시간 백분위수 계산
func (ec *EnhancedMetricsCollector) calculateLatencyPercentiles() {
	ec.latencyTracker.mu.Lock()
	defer ec.latencyTracker.mu.Unlock()

	for _, measurement := range ec.latencyTracker.measurements {
		if len(measurement.Samples) > 10 { // 최소 10개 샘플 필요
			ec.latencyTracker.CalculatePercentiles(measurement.RegistryType, measurement.Method)

			// 프로메테우스 메트릭 업데이트
			percentiles := map[string]float64{
				"p50": measurement.P50,
				"p90": measurement.P90,
				"p95": measurement.P95,
				"p99": measurement.P99,
			}

			for percentile, value := range percentiles {
				ec.metrics.LatencyPercentiles.WithLabelValues(
					measurement.RegistryType,
					percentile,
					measurement.Method,
				).Set(value)
			}
		}
	}
}

// performCleanup 오래된 데이터 정리
func (ec *EnhancedMetricsCollector) performCleanup() {
	cutoff := time.Now().Add(-ec.retentionPeriod)

	// 패키지 통계 정리
	ec.cleanupPackageStats(cutoff)

	// 사용자 에이전트 정리
	ec.cleanupUserAgentStats(cutoff)

	// 지리적 통계 정리
	ec.cleanupGeographicStats(cutoff)

	// 지연시간 측정값 정리
	ec.cleanupLatencyMeasurements(cutoff)

	// 처리량 윈도우 정리
	ec.cleanupThroughputWindows(cutoff)

	// 세션 정리
	ec.cleanupSessions(cutoff)

	ec.logger.Debug("Completed metrics cleanup",
		logging.F("cutoff", cutoff),
		logging.F("retention_hours", ec.retentionPeriod.Hours()))
}

// cleanupPackageStats 패키지 통계 정리
func (ec *EnhancedMetricsCollector) cleanupPackageStats(cutoff time.Time) {
	ec.mu.Lock()
	defer ec.mu.Unlock()

	for key, stats := range ec.packageStats {
		if stats.LastActivity.Before(cutoff) {
			delete(ec.packageStats, key)
		}
	}
}

// cleanupUserAgentStats 사용자 에이전트 통계 정리
func (ec *EnhancedMetricsCollector) cleanupUserAgentStats(cutoff time.Time) {
	ec.userAgentTracker.mu.Lock()
	defer ec.userAgentTracker.mu.Unlock()

	for agentKey, stats := range ec.userAgentTracker.agentStats {
		if stats.LastSeen.Before(cutoff) {
			delete(ec.userAgentTracker.agentStats, agentKey)
		}
	}
}

// cleanupGeographicStats 지리적 통계 정리
func (ec *EnhancedMetricsCollector) cleanupGeographicStats(cutoff time.Time) {
	ec.geoLocationTracker.mu.Lock()
	defer ec.geoLocationTracker.mu.Unlock()

	for locationKey, stats := range ec.geoLocationTracker.locationStats {
		if stats.LastSeen.Before(cutoff) {
			delete(ec.geoLocationTracker.locationStats, locationKey)
		}
	}
}

// cleanupLatencyMeasurements 지연시간 측정값 정리
func (ec *EnhancedMetricsCollector) cleanupLatencyMeasurements(cutoff time.Time) {
	ec.latencyTracker.mu.Lock()
	defer ec.latencyTracker.mu.Unlock()

	for measurementKey, measurement := range ec.latencyTracker.measurements {
		if measurement.LastUpdate.Before(cutoff) {
			delete(ec.latencyTracker.measurements, measurementKey)
		}
	}
}

// cleanupThroughputWindows 처리량 윈도우 정리
func (ec *EnhancedMetricsCollector) cleanupThroughputWindows(cutoff time.Time) {
	ec.throughputTracker.mu.Lock()
	defer ec.throughputTracker.mu.Unlock()

	for windowKey, window := range ec.throughputTracker.windows {
		if window.End.Before(cutoff) {
			delete(ec.throughputTracker.windows, windowKey)
		}
	}
}

// cleanupSessions 세션 정리
func (ec *EnhancedMetricsCollector) cleanupSessions(cutoff time.Time) {
	ec.sessionTracker.mu.Lock()
	defer ec.sessionTracker.mu.Unlock()

	for sessionKey, session := range ec.sessionTracker.sessions {
		if session.LastActivity.Before(cutoff) {
			delete(ec.sessionTracker.sessions, sessionKey)
		}
	}
}

// GetMetricsSnapshot 메트릭 스냅샷 조회
func (ec *EnhancedMetricsCollector) GetMetricsSnapshot() *EnhancedMetricsSnapshot {
	ec.mu.RLock()
	defer ec.mu.RUnlock()

	return &EnhancedMetricsSnapshot{
		Timestamp:         time.Now(),
		PackageStats:      ec.copyPackageStats(),
		PopularPackages:   ec.copyPopularPackages(),
		UserAgentStats:    ec.copyUserAgentStats(),
		GeographicStats:   ec.copyGeographicStats(),
		LatencyStats:      ec.copyLatencyStats(),
		ThroughputStats:   ec.copyThroughputStats(),
		ConnectionStats:   ec.copyConnectionStats(),
		ErrorStats:        ec.copyErrorStats(),
		RetryStats:        ec.copyRetryStats(),
		UpstreamStats:     ec.copyUpstreamStats(),
		UserActivityStats: ec.copyUserActivityStats(),
		SessionStats:      ec.copySessionStats(),
		BehaviorStats:     ec.copyBehaviorStats(),
	}
}

// EnhancedMetricsSnapshot 강화된 메트릭 스냅샷
type EnhancedMetricsSnapshot struct {
	Timestamp         time.Time                          `json:"timestamp"`
	PackageStats      map[string]*PackageStatistics      `json:"package_stats"`
	PopularPackages   map[string][]string                `json:"popular_packages"`
	UserAgentStats    map[string]*UserAgentStats         `json:"user_agent_stats"`
	GeographicStats   map[string]*GeographicStats        `json:"geographic_stats"`
	LatencyStats      map[string]*LatencyMeasurements    `json:"latency_stats"`
	ThroughputStats   map[string]*ThroughputMeasurements `json:"throughput_stats"`
	ConnectionStats   map[string]*ConnectionStats        `json:"connection_stats"`
	ErrorStats        map[string]*ErrorStats             `json:"error_stats"`
	RetryStats        map[string]*RetryStats             `json:"retry_stats"`
	UpstreamStats     map[string]*UpstreamStats          `json:"upstream_stats"`
	UserActivityStats map[string]*UserStats              `json:"user_activity_stats"`
	SessionStats      map[string]*SessionInfo            `json:"session_stats"`
	BehaviorStats     map[string]*BehaviorStats          `json:"behavior_stats"`
}

// Copy methods for thread-safe snapshot creation
func (ec *EnhancedMetricsCollector) copyPackageStats() map[string]*PackageStatistics {
	result := make(map[string]*PackageStatistics)
	for key, stats := range ec.packageStats {
		newStats := *stats
		if stats.Versions != nil {
			newStats.Versions = make(map[string]int64)
			for k, v := range stats.Versions {
				newStats.Versions[k] = v
			}
		}
		if stats.FileTypes != nil {
			newStats.FileTypes = make(map[string]int64)
			for k, v := range stats.FileTypes {
				newStats.FileTypes[k] = v
			}
		}
		result[key] = &newStats
	}
	return result
}

func (ec *EnhancedMetricsCollector) copyPopularPackages() map[string][]string {
	ec.popularPackages.mu.RLock()
	defer ec.popularPackages.mu.RUnlock()

	result := make(map[string][]string)
	for registryType, packages := range ec.popularPackages.rankings {
		result[registryType] = make([]string, len(packages))
		copy(result[registryType], packages)
	}
	return result
}

func (ec *EnhancedMetricsCollector) copyUserAgentStats() map[string]*UserAgentStats {
	ec.userAgentTracker.mu.RLock()
	defer ec.userAgentTracker.mu.RUnlock()

	result := make(map[string]*UserAgentStats)
	for key, stats := range ec.userAgentTracker.agentStats {
		newStats := *stats
		if stats.Registries != nil {
			newStats.Registries = make(map[string]int64)
			for k, v := range stats.Registries {
				newStats.Registries[k] = v
			}
		}
		result[key] = &newStats
	}
	return result
}

func (ec *EnhancedMetricsCollector) copyGeographicStats() map[string]*GeographicStats {
	ec.geoLocationTracker.mu.RLock()
	defer ec.geoLocationTracker.mu.RUnlock()

	result := make(map[string]*GeographicStats)
	for key, stats := range ec.geoLocationTracker.locationStats {
		newStats := *stats
		if stats.Registries != nil {
			newStats.Registries = make(map[string]int64)
			for k, v := range stats.Registries {
				newStats.Registries[k] = v
			}
		}
		result[key] = &newStats
	}
	return result
}

func (ec *EnhancedMetricsCollector) copyLatencyStats() map[string]*LatencyMeasurements {
	ec.latencyTracker.mu.RLock()
	defer ec.latencyTracker.mu.RUnlock()

	result := make(map[string]*LatencyMeasurements)
	for key, stats := range ec.latencyTracker.measurements {
		newStats := *stats
		newStats.Samples = make([]float64, len(stats.Samples))
		copy(newStats.Samples, stats.Samples)
		result[key] = &newStats
	}
	return result
}

func (ec *EnhancedMetricsCollector) copyThroughputStats() map[string]*ThroughputMeasurements {
	ec.throughputTracker.mu.RLock()
	defer ec.throughputTracker.mu.RUnlock()

	result := make(map[string]*ThroughputMeasurements)
	for key, measurement := range ec.throughputTracker.measurements {
		newMeasurement := *measurement
		if measurement.Windows != nil {
			newMeasurement.Windows = make(map[string]float64)
			for k, v := range measurement.Windows {
				newMeasurement.Windows[k] = v
			}
		}
		result[key] = &newMeasurement
	}
	return result
}

func (ec *EnhancedMetricsCollector) copyConnectionStats() map[string]*ConnectionStats {
	ec.connectionTracker.mu.RLock()
	defer ec.connectionTracker.mu.RUnlock()

	result := make(map[string]*ConnectionStats)
	for key, stats := range ec.connectionTracker.connections {
		newStats := *stats
		result[key] = &newStats
	}
	return result
}

func (ec *EnhancedMetricsCollector) copyErrorStats() map[string]*ErrorStats {
	ec.errorTracker.mu.RLock()
	defer ec.errorTracker.mu.RUnlock()

	result := make(map[string]*ErrorStats)
	for key, stats := range ec.errorTracker.errorStats {
		newStats := *stats
		result[key] = &newStats
	}
	return result
}

func (ec *EnhancedMetricsCollector) copyRetryStats() map[string]*RetryStats {
	ec.retryTracker.mu.RLock()
	defer ec.retryTracker.mu.RUnlock()

	result := make(map[string]*RetryStats)
	for key, stats := range ec.retryTracker.retryStats {
		newStats := *stats
		result[key] = &newStats
	}
	return result
}

func (ec *EnhancedMetricsCollector) copyUpstreamStats() map[string]*UpstreamStats {
	ec.upstreamTracker.mu.RLock()
	defer ec.upstreamTracker.mu.RUnlock()

	result := make(map[string]*UpstreamStats)
	for key, stats := range ec.upstreamTracker.upstreamStats {
		newStats := *stats
		result[key] = &newStats
	}
	return result
}

func (ec *EnhancedMetricsCollector) copyUserActivityStats() map[string]*UserStats {
	ec.userTracker.mu.RLock()
	defer ec.userTracker.mu.RUnlock()

	result := make(map[string]*UserStats)
	for key, stats := range ec.userTracker.userStats {
		newStats := *stats
		result[key] = &newStats
	}
	return result
}

func (ec *EnhancedMetricsCollector) copySessionStats() map[string]*SessionInfo {
	ec.sessionTracker.mu.RLock()
	defer ec.sessionTracker.mu.RUnlock()

	result := make(map[string]*SessionInfo)
	for key, stats := range ec.sessionTracker.sessions {
		newStats := *stats
		result[key] = &newStats
	}
	return result
}

func (ec *EnhancedMetricsCollector) copyBehaviorStats() map[string]*BehaviorStats {
	ec.behaviorTracker.mu.RLock()
	defer ec.behaviorTracker.mu.RUnlock()

	result := make(map[string]*BehaviorStats)
	for key, stats := range ec.behaviorTracker.behaviorStats {
		newStats := *stats
		result[key] = &newStats
	}
	return result
}

// Global enhanced collector instance
var globalEnhancedCollector *EnhancedMetricsCollector

// InitEnhancedMetricsCollector 강화된 메트릭 수집기 초기화
func InitEnhancedMetricsCollector(logger logging.Logger, config *EnhancedCollectorConfig) {
	if globalEnhancedCollector == nil {
		globalEnhancedCollector = NewEnhancedMetricsCollector(logger, config)
		// Start the collector
		if err := globalEnhancedCollector.Start(); err != nil {
			logger.Error("Failed to start enhanced metrics collector", logging.ErrorField(err))
		}
	}
}

// GetEnhancedMetricsCollector 전역 강화된 메트릭 수집기 반환
func GetEnhancedMetricsCollector() *EnhancedMetricsCollector {
	return globalEnhancedCollector
}

// StopEnhancedMetricsCollector 강화된 메트릭 수집기 중지
func StopEnhancedMetricsCollector() error {
	if globalEnhancedCollector != nil {
		return globalEnhancedCollector.Stop()
	}
	return nil
}
