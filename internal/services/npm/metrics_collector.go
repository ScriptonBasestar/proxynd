package npm

import (
	"context"
	"strings"
	"sync"
	"time"

	"proxynd/internal/domain/npm"
	"proxynd/logging"
)

// metricsCollectorImpl 메트릭 수집 서비스 구현
type metricsCollectorImpl struct {
	logger     logging.Logger
	stats      *npm.RequestStats
	statsMutex sync.RWMutex
}

// NewMetricsCollector MetricsCollector 생성자
func NewMetricsCollector(logger logging.Logger) npm.MetricsCollector {
	return &metricsCollectorImpl{
		logger: logger,
		stats: &npm.RequestStats{
			TopPackages: make([]npm.PackageStats, 0),
			ProxyUsage:  make([]npm.ProxyUsageStats, 0),
		},
	}
}

// RecordRequest 요청 메트릭 기록
func (m *metricsCollectorImpl) RecordRequest(ctx context.Context, metrics *npm.RequestMetrics) error {
	m.statsMutex.Lock()
	defer m.statsMutex.Unlock()

	// 기본 통계 업데이트
	m.stats.TotalRequests++
	m.stats.TotalBytesServed += metrics.BytesServed

	// 메타데이터 vs 패키지 파일 구분
	if metrics.IsMetadata {
		m.stats.MetadataRequests++
	} else {
		m.stats.TarballRequests++
	}

	// 평균 응답 시간 계산 (단순 이동 평균)
	if m.stats.TotalRequests == 1 {
		m.stats.AvgResponseTime = float64(metrics.Duration.Milliseconds())
	} else {
		m.stats.AvgResponseTime = (m.stats.AvgResponseTime + float64(metrics.Duration.Milliseconds())) / 2
	}

	// 캐시 히트율 계산 (단순화)
	if metrics.FromCache {
		m.stats.CacheHitRate = (m.stats.CacheHitRate + 1.0) / 2
	} else {
		m.stats.CacheHitRate = m.stats.CacheHitRate / 2
	}

	// 패키지별 통계 업데이트 (간단화)
	m.updatePackageStats(metrics)

	// 프록시별 통계 업데이트 (간단화)
	if metrics.ProxyUsed != "" {
		m.updateProxyStats(metrics)
	}

	m.logger.Debug("Recorded NPM request metrics",
		logging.F("packagePath", metrics.PackagePath),
		logging.F("statusCode", metrics.StatusCode),
		logging.F("duration_ms", metrics.Duration.Milliseconds()),
		logging.F("fromCache", metrics.FromCache),
		logging.F("isMetadata", metrics.IsMetadata),
	)

	return nil
}

// GetRequestStats 요청 통계 조회
func (m *metricsCollectorImpl) GetRequestStats(ctx context.Context) (*npm.RequestStats, error) {
	m.statsMutex.RLock()
	defer m.statsMutex.RUnlock()

	// 복사본 반환
	result := &npm.RequestStats{
		TotalRequests:    m.stats.TotalRequests,
		CacheHitRate:     m.stats.CacheHitRate,
		AvgResponseTime:  m.stats.AvgResponseTime,
		TotalBytesServed: m.stats.TotalBytesServed,
		MetadataRequests: m.stats.MetadataRequests,
		TarballRequests:  m.stats.TarballRequests,
		TopPackages:      make([]npm.PackageStats, len(m.stats.TopPackages)),
		ProxyUsage:       make([]npm.ProxyUsageStats, len(m.stats.ProxyUsage)),
	}

	copy(result.TopPackages, m.stats.TopPackages)
	copy(result.ProxyUsage, m.stats.ProxyUsage)

	return result, nil
}

// updatePackageStats 패키지별 통계 업데이트 (간단화된 버전)
func (m *metricsCollectorImpl) updatePackageStats(metrics *npm.RequestMetrics) {
	// 패키지명 추출 (간단화)
	packageName := m.extractPackageName(metrics.PackagePath)

	// 기존 패키지 통계 찾기
	for i := range m.stats.TopPackages {
		if m.stats.TopPackages[i].PackageName == packageName {
			m.stats.TopPackages[i].RequestCount++
			m.stats.TopPackages[i].TotalBytes += metrics.BytesServed
			m.stats.TopPackages[i].LastRequested = metrics.Timestamp.Format(time.RFC3339)
			return
		}
	}

	// 새 패키지 추가 (최대 10개까지)
	if len(m.stats.TopPackages) < 10 {
		m.stats.TopPackages = append(m.stats.TopPackages, npm.PackageStats{
			PackageName:   packageName,
			RequestCount:  1,
			TotalBytes:    metrics.BytesServed,
			LastRequested: metrics.Timestamp.Format(time.RFC3339),
			IsScoped:      len(packageName) > 0 && packageName[0] == '@',
		})
	}
}

// updateProxyStats 프록시별 통계 업데이트 (간단화된 버전)
func (m *metricsCollectorImpl) updateProxyStats(metrics *npm.RequestMetrics) {
	// 기존 프록시 통계 찾기
	for i := range m.stats.ProxyUsage {
		if m.stats.ProxyUsage[i].ProxyURL == metrics.ProxyUsed {
			m.stats.ProxyUsage[i].RequestCount++

			// 성공률 계산 (간단화)
			if metrics.StatusCode >= 200 && metrics.StatusCode < 300 {
				m.stats.ProxyUsage[i].SuccessRate = (m.stats.ProxyUsage[i].SuccessRate + 1.0) / 2
			} else {
				m.stats.ProxyUsage[i].SuccessRate = m.stats.ProxyUsage[i].SuccessRate / 2
			}

			// 평균 지연시간 계산
			m.stats.ProxyUsage[i].AvgLatency = (m.stats.ProxyUsage[i].AvgLatency + float64(metrics.Duration.Milliseconds())) / 2
			return
		}
	}

	// 새 프록시 추가 (최대 5개까지)
	if len(m.stats.ProxyUsage) < 5 {
		successRate := 1.0
		if metrics.StatusCode < 200 || metrics.StatusCode >= 300 {
			successRate = 0.0
		}

		m.stats.ProxyUsage = append(m.stats.ProxyUsage, npm.ProxyUsageStats{
			ProxyURL:     metrics.ProxyUsed,
			RequestCount: 1,
			SuccessRate:  successRate,
			AvgLatency:   float64(metrics.Duration.Milliseconds()),
		})
	}
}

// extractPackageName 패키지 경로에서 패키지명 추출
func (m *metricsCollectorImpl) extractPackageName(packagePath string) string {
	// @scope/package/version 또는 package/version에서 패키지명만 추출
	parts := strings.Split(packagePath, "/")

	if len(parts) >= 2 && strings.HasPrefix(parts[0], "@") {
		// 스코프 패키지: @scope/package
		return parts[0] + "/" + parts[1]
	} else if len(parts) >= 1 {
		// 일반 패키지: package
		return parts[0]
	}

	return packagePath
}
