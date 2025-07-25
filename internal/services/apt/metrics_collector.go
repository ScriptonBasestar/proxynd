package apt

import (
	"context"
	"sync"

	"proxynd/internal/domain/apt"
	"proxynd/logging"
)

// metricsCollectorImpl 메트릭 수집 서비스 구현
type metricsCollectorImpl struct {
	logger     logging.Logger
	stats      map[string]*apt.RequestStats // osType -> stats
	statsMutex sync.RWMutex
}

// NewMetricsCollector MetricsCollector 생성자
func NewMetricsCollector(logger logging.Logger) apt.MetricsCollector {
	return &metricsCollectorImpl{
		logger: logger,
		stats:  make(map[string]*apt.RequestStats),
	}
}

// RecordRequest 요청 메트릭 기록
func (m *metricsCollectorImpl) RecordRequest(ctx context.Context, metrics *apt.RequestMetrics) error {
	m.statsMutex.Lock()
	defer m.statsMutex.Unlock()

	// OS 타입별 통계 초기화
	if _, exists := m.stats[metrics.OSType]; !exists {
		m.stats[metrics.OSType] = &apt.RequestStats{
			OSType:      metrics.OSType,
			TopPackages: make([]apt.PackageStats, 0),
			MirrorUsage: make([]apt.MirrorUsageStats, 0),
		}
	}

	stats := m.stats[metrics.OSType]

	// 기본 통계 업데이트
	stats.TotalRequests++
	stats.TotalBytesServed += metrics.BytesServed

	// 평균 응답 시간 계산 (단순 이동 평균)
	if stats.TotalRequests == 1 {
		stats.AvgResponseTime = float64(metrics.Duration.Milliseconds())
	} else {
		stats.AvgResponseTime = (stats.AvgResponseTime + float64(metrics.Duration.Milliseconds())) / 2
	}

	// 캐시 히트율 계산 (단순화)
	if metrics.FromCache {
		stats.CacheHitRate = (stats.CacheHitRate + 1.0) / 2
	} else {
		stats.CacheHitRate = stats.CacheHitRate / 2
	}

	m.logger.Debug("Recorded APT request metrics",
		logging.F("osType", metrics.OSType),
		logging.F("path", metrics.Path),
		logging.F("statusCode", metrics.StatusCode),
		logging.F("duration_ms", metrics.Duration.Milliseconds()),
		logging.F("fromCache", metrics.FromCache),
	)

	return nil
}

// GetRequestStats 요청 통계 조회
func (m *metricsCollectorImpl) GetRequestStats(ctx context.Context, osType string) (*apt.RequestStats, error) {
	m.statsMutex.RLock()
	defer m.statsMutex.RUnlock()

	if stats, exists := m.stats[osType]; exists {
		// 복사본 반환
		result := &apt.RequestStats{
			OSType:           stats.OSType,
			TotalRequests:    stats.TotalRequests,
			CacheHitRate:     stats.CacheHitRate,
			AvgResponseTime:  stats.AvgResponseTime,
			TotalBytesServed: stats.TotalBytesServed,
			TopPackages:      make([]apt.PackageStats, len(stats.TopPackages)),
			MirrorUsage:      make([]apt.MirrorUsageStats, len(stats.MirrorUsage)),
		}

		copy(result.TopPackages, stats.TopPackages)
		copy(result.MirrorUsage, stats.MirrorUsage)

		return result, nil
	}

	// 없으면 빈 통계 반환
	return &apt.RequestStats{
		OSType:      osType,
		TopPackages: make([]apt.PackageStats, 0),
		MirrorUsage: make([]apt.MirrorUsageStats, 0),
	}, nil
}
