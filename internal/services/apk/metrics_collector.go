package apk

import (
	"context"
	"sort"
	"sync"
	"time"

	"proxynd/internal/domain/apk"
	"proxynd/logging"
)

type metricsCollectorImpl struct {
	config   apk.ProxyConfig
	logger   logging.Logger
	requests []*apk.RequestMetrics
	stats    map[string]interface{}
	mutex    sync.RWMutex
}

func NewMetricsCollector(config apk.ProxyConfig, logger logging.Logger, storageDir string) apk.MetricsCollector {
	return &metricsCollectorImpl{
		config:   config,
		logger:   logger,
		requests: make([]*apk.RequestMetrics, 0),
		stats:    make(map[string]interface{}),
	}
}

func (m *metricsCollectorImpl) RecordRequest(ctx context.Context, metrics *apk.RequestMetrics) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	m.requests = append(m.requests, metrics)
	
	// 최근 1000개 요청만 유지
	if len(m.requests) > 1000 {
		m.requests = m.requests[len(m.requests)-1000:]
	}
	
	return nil
}

func (m *metricsCollectorImpl) GetRequestStats(ctx context.Context, period string) (map[string]interface{}, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	duration := m.parsePeriod(period)
	cutoff := time.Now().Add(-duration)
	
	var recentRequests []*apk.RequestMetrics
	for _, req := range m.requests {
		if req.RequestTime.After(cutoff) {
			recentRequests = append(recentRequests, req)
		}
	}
	
	return map[string]interface{}{
		"period":         period,
		"total_requests": len(recentRequests),
		"cache_hits":     m.countCacheHits(recentRequests),
		"cache_misses":   m.countCacheMisses(recentRequests),
		"architectures":  m.countArchitectures(recentRequests),
		"branches":       m.countBranches(recentRequests),
		"components":     m.countComponents(recentRequests),
	}, nil
}

func (m *metricsCollectorImpl) GetPopularPackages(ctx context.Context, limit int) ([]*apk.PackageInfo, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	packageCounts := make(map[string]int)
	for _, req := range m.requests {
		if req.IsApkFile {
			packageCounts[req.Path]++
		}
	}
	
	type packageRank struct {
		path  string
		count int
	}
	
	var packages []packageRank
	for path, count := range packageCounts {
		packages = append(packages, packageRank{path: path, count: count})
	}
	
	sort.Slice(packages, func(i, j int) bool {
		return packages[i].count > packages[j].count
	})
	
	var result []*apk.PackageInfo
	maxResults := limit
	if len(packages) < maxResults {
		maxResults = len(packages)
	}
	
	for i := 0; i < maxResults; i++ {
		result = append(result, &apk.PackageInfo{
			Name: packages[i].path,
		})
	}
	
	return result, nil
}

func (m *metricsCollectorImpl) GetArchitectureStats(ctx context.Context) (map[string]interface{}, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	return m.countArchitectures(m.requests), nil
}

func (m *metricsCollectorImpl) GetBranchStats(ctx context.Context) (map[string]interface{}, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	return m.countBranches(m.requests), nil
}

func (m *metricsCollectorImpl) GetComponentStats(ctx context.Context) (map[string]interface{}, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	return m.countComponents(m.requests), nil
}

func (m *metricsCollectorImpl) RecordCacheHit(ctx context.Context, key string, hit bool) error {
	// 캐시 히트 기록은 캐시 관리자에서 처리
	return nil
}

func (m *metricsCollectorImpl) GetCacheMetrics(ctx context.Context) (map[string]interface{}, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	totalRequests := len(m.requests)
	cacheHits := m.countCacheHits(m.requests)
	
	hitRatio := float64(0)
	if totalRequests > 0 {
		hitRatio = float64(cacheHits) / float64(totalRequests)
	}
	
	return map[string]interface{}{
		"total_requests": totalRequests,
		"cache_hits":     cacheHits,
		"cache_misses":   totalRequests - cacheHits,
		"hit_ratio":      hitRatio,
	}, nil
}

func (m *metricsCollectorImpl) RecordSignatureVerification(ctx context.Context, packagePath string, result *apk.SignatureInfo) error {
	m.logger.Debug("APK 서명 검증 메트릭 기록",
		logging.F("package", packagePath),
		logging.F("valid", result.IsValid),
		logging.F("key", result.KeyFingerprint))
	
	return nil
}

// 헬퍼 메서드들

func (m *metricsCollectorImpl) parsePeriod(period string) time.Duration {
	switch period {
	case "1h", "hour":
		return time.Hour
	case "24h", "day":
		return 24 * time.Hour
	case "7d", "week":
		return 7 * 24 * time.Hour
	case "30d", "month":
		return 30 * 24 * time.Hour
	default:
		return time.Hour
	}
}

func (m *metricsCollectorImpl) countCacheHits(requests []*apk.RequestMetrics) int {
	count := 0
	for _, req := range requests {
		if req.CacheHit {
			count++
		}
	}
	return count
}

func (m *metricsCollectorImpl) countCacheMisses(requests []*apk.RequestMetrics) int {
	count := 0
	for _, req := range requests {
		if !req.CacheHit {
			count++
		}
	}
	return count
}

func (m *metricsCollectorImpl) countArchitectures(requests []*apk.RequestMetrics) map[string]interface{} {
	counts := make(map[string]int)
	for _, req := range requests {
		if req.Architecture != "" {
			counts[req.Architecture]++
		}
	}
	
	result := make(map[string]interface{})
	for arch, count := range counts {
		result[arch] = count
	}
	return result
}

func (m *metricsCollectorImpl) countBranches(requests []*apk.RequestMetrics) map[string]interface{} {
	counts := make(map[string]int)
	for _, req := range requests {
		if req.Branch != "" {
			counts[req.Branch]++
		}
	}
	
	result := make(map[string]interface{})
	for branch, count := range counts {
		result[branch] = count
	}
	return result
}

func (m *metricsCollectorImpl) countComponents(requests []*apk.RequestMetrics) map[string]interface{} {
	counts := make(map[string]int)
	for _, req := range requests {
		if req.Component != "" {
			counts[req.Component]++
		}
	}
	
	result := make(map[string]interface{})
	for component, count := range counts {
		result[component] = count
	}
	return result
}