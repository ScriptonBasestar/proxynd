package yum

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"proxynd/internal/domain/yum"
	"proxynd/internal/logging"
)

// metricsCollectorImpl YUM 메트릭 수집 서비스 구현
type metricsCollectorImpl struct {
	config     yum.ProxyConfig
	logger     logging.Logger
	storageDir string
	metrics    *metricsData
	mutex      sync.RWMutex
}

type metricsData struct {
	Requests        []*yum.RequestMetrics    `json:"requests"`
	PackageStats    map[string]*packageStats `json:"package_stats"`
	RepositoryStats map[string]*repoStats    `json:"repository_stats"`
	CacheMetrics    *cacheMetrics            `json:"cache_metrics"`
	LastUpdated     time.Time                `json:"last_updated"`
}

type packageStats struct {
	Name         string    `json:"name"`
	RequestCount int64     `json:"request_count"`
	TotalSize    int64     `json:"total_size"`
	LastAccessed time.Time `json:"last_accessed"`
	Repository   string    `json:"repository"`
}

type repoStats struct {
	Repository   string    `json:"repository"`
	RequestCount int64     `json:"request_count"`
	CacheHits    int64     `json:"cache_hits"`
	CacheMisses  int64     `json:"cache_misses"`
	TotalSize    int64     `json:"total_size"`
	LastAccessed time.Time `json:"last_accessed"`
}

type cacheMetrics struct {
	Hits     int64     `json:"hits"`
	Misses   int64     `json:"misses"`
	HitRatio float64   `json:"hit_ratio"`
	Size     int64     `json:"size"`
	Updated  time.Time `json:"updated"`
}

// NewMetricsCollector YUM 메트릭 수집기 생성
func NewMetricsCollector(
	config yum.ProxyConfig,
	logger logging.Logger,
	storageDir string,
) yum.MetricsCollector {
	collector := &metricsCollectorImpl{
		config:     config,
		logger:     logger,
		storageDir: storageDir,
		metrics: &metricsData{
			Requests:        make([]*yum.RequestMetrics, 0),
			PackageStats:    make(map[string]*packageStats),
			RepositoryStats: make(map[string]*repoStats),
			CacheMetrics:    &cacheMetrics{},
			LastUpdated:     time.Now(),
		},
	}

	// 기존 메트릭 데이터 로드
	collector.loadMetrics()

	return collector
}

// RecordRequest 요청 메트릭 기록
func (m *metricsCollectorImpl) RecordRequest(ctx context.Context, metrics *yum.RequestMetrics) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// 요청 메트릭 추가
	m.metrics.Requests = append(m.metrics.Requests, metrics)

	// 최근 1000개 요청만 유지
	if len(m.metrics.Requests) > 1000 {
		m.metrics.Requests = m.metrics.Requests[len(m.metrics.Requests)-1000:]
	}

	// 패키지 통계 업데이트
	m.updatePackageStats(metrics)

	// 리포지토리 통계 업데이트
	m.updateRepositoryStats(metrics)

	m.metrics.LastUpdated = time.Now()

	// 메트릭 데이터 저장
	m.saveMetrics()

	m.logger.Debug("YUM 요청 메트릭 기록",
		logging.F("path", metrics.Path),
		logging.F("repository", metrics.Repository),
		logging.F("status", metrics.StatusCode),
		logging.F("cache_hit", metrics.CacheHit))

	return nil
}

// GetRequestStats 요청 통계 조회
func (m *metricsCollectorImpl) GetRequestStats(ctx context.Context, period string) (map[string]interface{}, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	duration := m.parsePeriod(period)
	cutoff := time.Now().Add(-duration)

	var recentRequests []*yum.RequestMetrics
	for _, req := range m.metrics.Requests {
		if req.RequestTime.After(cutoff) {
			recentRequests = append(recentRequests, req)
		}
	}

	stats := map[string]interface{}{
		"period":           period,
		"total_requests":   len(recentRequests),
		"cache_hits":       m.countCacheHits(recentRequests),
		"cache_misses":     m.countCacheMisses(recentRequests),
		"average_response": m.calculateAverageResponse(recentRequests),
		"status_codes":     m.countStatusCodes(recentRequests),
		"file_types":       m.countFileTypes(recentRequests),
		"repositories":     m.countRepositories(recentRequests),
		"total_size":       m.calculateTotalSize(recentRequests),
	}

	return stats, nil
}

// GetPopularPackages 인기 패키지 조회
func (m *metricsCollectorImpl) GetPopularPackages(ctx context.Context, limit int) ([]*yum.PackageInfo, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	// 패키지 통계를 요청 수로 정렬
	type packageRank struct {
		name  string
		stats *packageStats
	}

	var packages []packageRank
	for name, stats := range m.metrics.PackageStats {
		packages = append(packages, packageRank{name: name, stats: stats})
	}

	sort.Slice(packages, func(i, j int) bool {
		return packages[i].stats.RequestCount > packages[j].stats.RequestCount
	})

	// 상위 패키지들을 PackageInfo로 변환
	var result []*yum.PackageInfo
	maxResults := limit
	if len(packages) < maxResults {
		maxResults = len(packages)
	}

	for i := 0; i < maxResults; i++ {
		pkg := packages[i]
		result = append(result, &yum.PackageInfo{
			Name: pkg.name,
			// 실제 구현에서는 패키지 정보를 더 자세히 채워야 함
		})
	}

	return result, nil
}

// GetRepositoryStats 리포지토리별 통계 조회
func (m *metricsCollectorImpl) GetRepositoryStats(ctx context.Context) (map[string]interface{}, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	stats := make(map[string]interface{})

	for repo, repoStats := range m.metrics.RepositoryStats {
		stats[repo] = map[string]interface{}{
			"request_count":   repoStats.RequestCount,
			"cache_hits":      repoStats.CacheHits,
			"cache_misses":    repoStats.CacheMisses,
			"total_size":      repoStats.TotalSize,
			"last_accessed":   repoStats.LastAccessed,
			"cache_hit_ratio": m.calculateCacheHitRatio(repoStats.CacheHits, repoStats.CacheMisses),
		}
	}

	return stats, nil
}

// RecordCacheHit 캐시 히트 기록
func (m *metricsCollectorImpl) RecordCacheHit(ctx context.Context, key string, hit bool) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if hit {
		m.metrics.CacheMetrics.Hits++
	} else {
		m.metrics.CacheMetrics.Misses++
	}

	total := m.metrics.CacheMetrics.Hits + m.metrics.CacheMetrics.Misses
	if total > 0 {
		m.metrics.CacheMetrics.HitRatio = float64(m.metrics.CacheMetrics.Hits) / float64(total)
	}

	m.metrics.CacheMetrics.Updated = time.Now()

	return nil
}

// GetCacheMetrics 캐시 메트릭 조회
func (m *metricsCollectorImpl) GetCacheMetrics(ctx context.Context) (map[string]interface{}, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	metrics := map[string]interface{}{
		"hits":      m.metrics.CacheMetrics.Hits,
		"misses":    m.metrics.CacheMetrics.Misses,
		"hit_ratio": m.metrics.CacheMetrics.HitRatio,
		"size":      m.metrics.CacheMetrics.Size,
		"updated":   m.metrics.CacheMetrics.Updated,
	}

	return metrics, nil
}

// 헬퍼 메서드들

func (m *metricsCollectorImpl) updatePackageStats(metrics *yum.RequestMetrics) {
	if !metrics.IsRpmFile {
		return
	}

	packageName := m.extractPackageName(metrics.Path)
	if packageName == "" {
		return
	}

	stats, exists := m.metrics.PackageStats[packageName]
	if !exists {
		stats = &packageStats{
			Name:       packageName,
			Repository: metrics.Repository,
		}
		m.metrics.PackageStats[packageName] = stats
	}

	stats.RequestCount++
	stats.TotalSize += metrics.FileSize
	stats.LastAccessed = metrics.RequestTime
}

func (m *metricsCollectorImpl) updateRepositoryStats(metrics *yum.RequestMetrics) {
	stats, exists := m.metrics.RepositoryStats[metrics.Repository]
	if !exists {
		stats = &repoStats{
			Repository: metrics.Repository,
		}
		m.metrics.RepositoryStats[metrics.Repository] = stats
	}

	stats.RequestCount++
	stats.TotalSize += metrics.FileSize
	stats.LastAccessed = metrics.RequestTime

	if metrics.CacheHit {
		stats.CacheHits++
	} else {
		stats.CacheMisses++
	}
}

func (m *metricsCollectorImpl) extractPackageName(path string) string {
	// path에서 패키지 이름 추출 (예: repo/package-1.0.0-1.el7.x86_64.rpm -> package)
	filename := filepath.Base(path)
	if !strings.HasSuffix(filename, ".rpm") {
		return ""
	}

	// RPM 파일명에서 패키지 이름 추출
	filename = filename[:len(filename)-4] // .rpm 제거

	// 버전 정보 제거 (첫 번째 하이픈 이전이 패키지 이름)
	parts := strings.Split(filename, "-")
	if len(parts) > 0 {
		return parts[0]
	}

	return filename
}

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
		return time.Hour // 기본값
	}
}

func (m *metricsCollectorImpl) countCacheHits(requests []*yum.RequestMetrics) int {
	count := 0
	for _, req := range requests {
		if req.CacheHit {
			count++
		}
	}
	return count
}

func (m *metricsCollectorImpl) countCacheMisses(requests []*yum.RequestMetrics) int {
	count := 0
	for _, req := range requests {
		if !req.CacheHit {
			count++
		}
	}
	return count
}

func (m *metricsCollectorImpl) calculateAverageResponse(requests []*yum.RequestMetrics) float64 {
	if len(requests) == 0 {
		return 0
	}

	total := int64(0)
	for _, req := range requests {
		total += req.ResponseTime
	}

	return float64(total) / float64(len(requests))
}

func (m *metricsCollectorImpl) countStatusCodes(requests []*yum.RequestMetrics) map[string]int {
	counts := make(map[string]int)
	for _, req := range requests {
		code := fmt.Sprintf("%d", req.StatusCode)
		counts[code]++
	}
	return counts
}

func (m *metricsCollectorImpl) countFileTypes(requests []*yum.RequestMetrics) map[string]int {
	counts := make(map[string]int)
	for _, req := range requests {
		if req.IsRpmFile {
			counts["rpm"]++
		} else if req.IsRepoMeta {
			counts["metadata"]++
		} else {
			counts["other"]++
		}
	}
	return counts
}

func (m *metricsCollectorImpl) countRepositories(requests []*yum.RequestMetrics) map[string]int {
	counts := make(map[string]int)
	for _, req := range requests {
		counts[req.Repository]++
	}
	return counts
}

func (m *metricsCollectorImpl) calculateTotalSize(requests []*yum.RequestMetrics) int64 {
	total := int64(0)
	for _, req := range requests {
		total += req.FileSize
	}
	return total
}

func (m *metricsCollectorImpl) calculateCacheHitRatio(hits, misses int64) float64 {
	total := hits + misses
	if total == 0 {
		return 0.0
	}
	return float64(hits) / float64(total)
}

func (m *metricsCollectorImpl) loadMetrics() {
	metricsPath := filepath.Join(m.storageDir, "metrics", "yum", "metrics.json")

	data, err := os.ReadFile(metricsPath)
	if err != nil {
		if !os.IsNotExist(err) {
			m.logger.Warn("YUM 메트릭 데이터 로드 실패",
				logging.F("error", err.Error()))
		}
		return
	}

	if err := json.Unmarshal(data, m.metrics); err != nil {
		m.logger.Warn("YUM 메트릭 데이터 파싱 실패",
			logging.F("error", err.Error()))
		return
	}

	m.logger.Info("YUM 메트릭 데이터 로드 완료",
		logging.F("requests", len(m.metrics.Requests)),
		logging.F("packages", len(m.metrics.PackageStats)),
		logging.F("repositories", len(m.metrics.RepositoryStats)))
}

func (m *metricsCollectorImpl) saveMetrics() {
	metricsPath := filepath.Join(m.storageDir, "metrics", "yum", "metrics.json")
	metricsDir := filepath.Dir(metricsPath)

	if err := os.MkdirAll(metricsDir, os.ModePerm); err != nil {
		m.logger.Warn("YUM 메트릭 디렉토리 생성 실패",
			logging.F("error", err.Error()))
		return
	}

	data, err := json.Marshal(m.metrics)
	if err != nil {
		m.logger.Warn("YUM 메트릭 데이터 직렬화 실패",
			logging.F("error", err.Error()))
		return
	}

	if err := os.WriteFile(metricsPath, data, 0o644); err != nil {
		m.logger.Warn("YUM 메트릭 데이터 저장 실패",
			logging.F("error", err.Error()))
	}
}
