package pip

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"proxynd/internal/domain/pip"
	"proxynd/logging"
)

// metricsCollectorImpl PIP 메트릭 수집 서비스 구현
type metricsCollectorImpl struct {
	config    pip.ProxyConfig
	logger    logging.Logger
	requests  []*pip.RequestMetrics
	mu        sync.RWMutex
	cacheHits int64
	cacheMiss int64
}

// NewMetricsCollector MetricsCollector 생성자
func NewMetricsCollector(config pip.ProxyConfig, logger logging.Logger) pip.MetricsCollector {
	return &metricsCollectorImpl{
		config:   config,
		logger:   logger,
		requests: make([]*pip.RequestMetrics, 0),
	}
}

// RecordRequest 요청 메트릭 기록
func (m *metricsCollectorImpl) RecordRequest(ctx context.Context, metrics *pip.RequestMetrics) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 메트릭 저장 (최근 1000개만 유지)
	m.requests = append(m.requests, metrics)
	if len(m.requests) > 1000 {
		m.requests = m.requests[1:] // 가장 오래된 것 제거
	}

	// 캐시 상태 업데이트
	if metrics.FromCache {
		m.cacheHits++
	} else {
		m.cacheMiss++
	}

	m.logger.Debug("Request metrics recorded",
		logging.F("packagePath", metrics.PackagePath),
		logging.F("statusCode", metrics.StatusCode),
		logging.F("duration", metrics.Duration),
		logging.F("fromCache", metrics.FromCache),
		logging.F("bytesServed", metrics.BytesServed),
	)

	return nil
}

// GetRequestStats 요청 통계 조회
func (m *metricsCollectorImpl) GetRequestStats(ctx context.Context) (*pip.RequestStats, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.requests) == 0 {
		return &pip.RequestStats{}, nil
	}

	stats := &pip.RequestStats{
		TotalRequests: int64(len(m.requests)),
	}

	// 기본 통계 계산
	var totalDuration time.Duration
	var totalBytes int64
	packageStats := make(map[string]*pip.PackageStats)
	indexStats := make(map[string]*pip.IndexUsageStats)
	userAgentStats := make(map[string]*pip.UserAgentStats)

	for _, req := range m.requests {
		// 전체 통계
		totalDuration += req.Duration
		totalBytes += req.BytesServed

		// 요청 타입별 계산
		if req.IsSimpleAPI {
			stats.SimpleAPIRequests++
		} else if req.IsPackageFile {
			stats.PackageFileRequests++

			// 파일 타입별 분류
			if strings.Contains(req.PackagePath, ".whl") {
				stats.WheelDownloads++
			} else {
				stats.SourceDownloads++
			}
		}

		// 패키지별 통계
		packageName := m.extractPackageName(req.PackagePath)
		if packageName != "" {
			if ps, exists := packageStats[packageName]; exists {
				ps.RequestCount++
				ps.TotalBytes += req.BytesServed
				ps.LastRequested = req.Timestamp.Format(time.RFC3339)

				// 파일 타입 계산
				fileType := m.determineFileType(req.PackagePath)
				if ps.FileTypes == nil {
					ps.FileTypes = make(map[string]int64)
				}
				ps.FileTypes[fileType]++
			} else {
				fileTypes := make(map[string]int64)
				fileType := m.determineFileType(req.PackagePath)
				fileTypes[fileType] = 1

				packageStats[packageName] = &pip.PackageStats{
					PackageName:   packageName,
					RequestCount:  1,
					TotalBytes:    req.BytesServed,
					LastRequested: req.Timestamp.Format(time.RFC3339),
					FileTypes:     fileTypes,
				}
			}
		}

		// 인덱스 사용 통계
		if req.ProxyUsed != "" {
			if is, exists := indexStats[req.ProxyUsed]; exists {
				is.RequestCount++
				if req.StatusCode >= 200 && req.StatusCode < 300 {
					is.SuccessRate = float64(is.RequestCount) / float64(is.RequestCount)
				}
				is.AvgLatency = (is.AvgLatency + float64(req.Duration.Milliseconds())) / 2
			} else {
				successRate := 0.0
				if req.StatusCode >= 200 && req.StatusCode < 300 {
					successRate = 1.0
				}

				indexStats[req.ProxyUsed] = &pip.IndexUsageStats{
					IndexURL:     req.ProxyUsed,
					RequestCount: 1,
					SuccessRate:  successRate,
					AvgLatency:   float64(req.Duration.Milliseconds()),
				}
			}
		}

		// User Agent 통계
		if req.UserAgent != "" {
			if uas, exists := userAgentStats[req.UserAgent]; exists {
				uas.RequestCount++
			} else {
				userAgentStats[req.UserAgent] = &pip.UserAgentStats{
					UserAgent:    req.UserAgent,
					RequestCount: 1,
					PipVersion:   m.extractPipVersion(req.UserAgent),
				}
			}
		}
	}

	// 평균 응답 시간 계산
	if stats.TotalRequests > 0 {
		stats.AvgResponseTime = float64(totalDuration.Milliseconds()) / float64(stats.TotalRequests)
		stats.TotalBytesServed = totalBytes
	}

	// 상위 패키지 정렬 (TOP 10)
	stats.TopPackages = m.getTopPackages(packageStats, 10)

	// 인덱스 사용 통계 변환
	stats.IndexUsage = make([]pip.IndexUsageStats, 0, len(indexStats))
	for _, is := range indexStats {
		stats.IndexUsage = append(stats.IndexUsage, *is)
	}

	// User Agent 통계 변환 (TOP 5)
	stats.UserAgents = m.getTopUserAgents(userAgentStats, 5)

	// 캐시 Hit Rate 계산
	totalCacheRequests := m.cacheHits + m.cacheMiss
	if totalCacheRequests > 0 {
		stats.CacheHitRate = float64(m.cacheHits) / float64(totalCacheRequests)
	}

	return stats, nil
}

// RecordCacheHit 캐시 히트 기록
func (m *metricsCollectorImpl) RecordCacheHit(ctx context.Context, packagePath string, size int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.cacheHits++

	m.logger.Debug("Cache hit recorded",
		logging.F("packagePath", packagePath),
		logging.F("size", size),
	)
}

// RecordCacheMiss 캐시 미스 기록
func (m *metricsCollectorImpl) RecordCacheMiss(ctx context.Context, packagePath string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.cacheMiss++

	m.logger.Debug("Cache miss recorded",
		logging.F("packagePath", packagePath),
	)
}

// extractPackageName 패키지 경로에서 패키지명 추출
func (m *metricsCollectorImpl) extractPackageName(packagePath string) string {
	// simple/package-name 형태에서 package-name 추출
	if strings.HasPrefix(packagePath, "simple/") {
		packageName := strings.TrimPrefix(packagePath, "simple/")
		packageName = strings.TrimSuffix(packageName, "/")
		return packageName
	}

	// packages/source/p/package/package-1.0.tar.gz 형태에서 package 추출
	parts := strings.Split(packagePath, "/")
	if len(parts) >= 2 {
		fileName := parts[len(parts)-1]
		// 파일명에서 패키지명 추출 (버전 제거)
		if strings.Contains(fileName, "-") {
			nameParts := strings.Split(fileName, "-")
			if len(nameParts) > 0 {
				return nameParts[0]
			}
		}
	}

	return ""
}

// determineFileType 파일 타입 결정
func (m *metricsCollectorImpl) determineFileType(packagePath string) string {
	switch {
	case strings.Contains(packagePath, "simple/"):
		return "simple"
	case strings.HasSuffix(packagePath, ".whl"):
		return "wheel"
	case strings.HasSuffix(packagePath, ".tar.gz"), strings.HasSuffix(packagePath, ".zip"):
		return "sdist"
	default:
		return "other"
	}
}

// extractPipVersion User Agent에서 pip 버전 추출
func (m *metricsCollectorImpl) extractPipVersion(userAgent string) string {
	// 예: "pip/21.3.1 {"implementation": "CPython", "version": "3.9.7"}"
	if strings.HasPrefix(userAgent, "pip/") {
		parts := strings.Split(userAgent, " ")
		if len(parts) > 0 {
			return strings.TrimPrefix(parts[0], "pip/")
		}
	}
	return ""
}

// getTopPackages 상위 패키지 목록 반환
func (m *metricsCollectorImpl) getTopPackages(packageStats map[string]*pip.PackageStats, limit int) []pip.PackageStats {
	// 슬라이스로 변환
	packages := make([]pip.PackageStats, 0, len(packageStats))
	for _, ps := range packageStats {
		packages = append(packages, *ps)
	}

	// 요청 수로 정렬
	sort.Slice(packages, func(i, j int) bool {
		return packages[i].RequestCount > packages[j].RequestCount
	})

	// 상위 N개만 반환
	if len(packages) > limit {
		packages = packages[:limit]
	}

	return packages
}

// getTopUserAgents 상위 User Agent 목록 반환
func (m *metricsCollectorImpl) getTopUserAgents(userAgentStats map[string]*pip.UserAgentStats, limit int) []pip.UserAgentStats {
	// 슬라이스로 변환
	userAgents := make([]pip.UserAgentStats, 0, len(userAgentStats))
	for _, uas := range userAgentStats {
		userAgents = append(userAgents, *uas)
	}

	// 요청 수로 정렬
	sort.Slice(userAgents, func(i, j int) bool {
		return userAgents[i].RequestCount > userAgents[j].RequestCount
	})

	// 상위 N개만 반환
	if len(userAgents) > limit {
		userAgents = userAgents[:limit]
	}

	return userAgents
}
