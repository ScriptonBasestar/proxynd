package docker

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"proxynd/internal/domain/docker"
	"proxynd/logging"
)

// metricsCollectorImpl Docker 프록시 메트릭 수집 구현
type metricsCollectorImpl struct {
	config          docker.ProxyConfig
	logger          logging.Logger
	requestStats    *docker.RequestStats
	errorStats      *docker.ErrorStats
	imageStats      []docker.ImageStat
	repositoryMap   map[string]*docker.RepositoryStat
	counterMetrics  map[string]int64
	durationMetrics map[string][]int64
	mutex           sync.RWMutex
}

// NewMetricsCollector MetricsCollector 생성자
func NewMetricsCollector(
	config docker.ProxyConfig,
	logger logging.Logger,
) docker.MetricsCollector {
	return &metricsCollectorImpl{
		config: config,
		logger: logger,
		requestStats: &docker.RequestStats{
			RepositoryStats: make(map[string]*docker.RepositoryStat),
			RegistryUsage:   make(map[string]int64),
		},
		errorStats: &docker.ErrorStats{
			ErrorsByType:      make(map[string]int64),
			ErrorsByOperation: make(map[string]int64),
			ErrorsByRegistry:  make(map[string]int64),
			RecentErrors:      make([]string, 0),
		},
		imageStats:      make([]docker.ImageStat, 0),
		repositoryMap:   make(map[string]*docker.RepositoryStat),
		counterMetrics:  make(map[string]int64),
		durationMetrics: make(map[string][]int64),
		mutex:           sync.RWMutex{},
	}
}

// RecordRequest 요청 메트릭 기록
func (m *metricsCollectorImpl) RecordRequest(ctx context.Context, metrics *docker.RequestMetrics) error {
	m.logger.Debug("Recording request metrics",
		logging.F("repository", metrics.Repository),
		logging.F("operation", metrics.Operation),
		logging.F("status", metrics.StatusCode))

	m.mutex.Lock()
	defer m.mutex.Unlock()

	// 전체 요청 통계 업데이트
	m.requestStats.TotalRequests++

	// 작업별 요청 통계
	switch metrics.Operation {
	case dockerResourceManifest:
		m.requestStats.ManifestRequests++
	case dockerResourceBlob:
		m.requestStats.BlobRequests++
	}

	// 캐시 히트율 계산
	if metrics.FromCache {
		m.updateCacheHitRate(true)
	} else {
		m.updateCacheHitRate(false)
	}

	// 응답 시간 기록
	m.recordResponseTime(metrics.Duration.Milliseconds())

	// 레지스트리 사용량 기록
	if metrics.RegistryUsed != "" {
		m.requestStats.RegistryUsage[metrics.RegistryUsed] += metrics.BytesServed
	}

	// 전송된 바이트 수 누적
	m.requestStats.BytesServed += metrics.BytesServed

	// 레포지토리별 통계 업데이트
	m.updateRepositoryStats(metrics)

	// 이미지별 통계 업데이트 (매니페스트 요청의 경우)
	if metrics.Operation == "manifest" && metrics.StatusCode == 200 { //nolint:goconst
		m.updateImageStats(metrics)
	}

	// 에러 통계 (4xx, 5xx)
	if metrics.StatusCode >= 400 {
		m.requestStats.ErrorCount++
		m.recordErrorByStatus(metrics.StatusCode, metrics.Operation)
	}

	return nil
}

// IncrementCounter 카운터 메트릭 증가
func (m *metricsCollectorImpl) IncrementCounter(ctx context.Context, name string, tags map[string]string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// 태그를 포함한 메트릭 키 생성
	key := m.buildMetricKey(name, tags)
	m.counterMetrics[key]++

	m.logger.Debug("Counter incremented",
		logging.F("name", name),
		logging.F("key", key),
		logging.F("value", m.counterMetrics[key]))
	return nil
}

// RecordDuration 응답 시간 기록
func (m *metricsCollectorImpl) RecordDuration(ctx context.Context, operation string, duration int64) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	key := fmt.Sprintf("duration_%s", operation)
	if _, exists := m.durationMetrics[key]; !exists {
		m.durationMetrics[key] = make([]int64, 0)
	}

	m.durationMetrics[key] = append(m.durationMetrics[key], duration)

	// 최근 100개 기록만 유지 (메모리 관리)
	if len(m.durationMetrics[key]) > 100 {
		m.durationMetrics[key] = m.durationMetrics[key][1:]
	}

	return nil
}

// RecordCacheHit 캐시 히트/미스 기록
func (m *metricsCollectorImpl) RecordCacheHit(ctx context.Context, hit bool, operation string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	var counterName string
	if hit {
		counterName = fmt.Sprintf("cache_hit_%s", operation)
	} else {
		counterName = fmt.Sprintf("cache_miss_%s", operation)
	}

	m.counterMetrics[counterName]++

	m.logger.Debug("Cache metrics recorded", logging.F("operation", operation), logging.F("hit", hit))
	return nil
}

// RecordRegistryUsage 레지스트리 사용량 기록
func (m *metricsCollectorImpl) RecordRegistryUsage(ctx context.Context, registryURL string, bytes int64) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.requestStats.RegistryUsage[registryURL] += bytes

	return nil
}

// GetRequestStats 요청 통계 반환 (레포지토리별, 작업별)
func (m *metricsCollectorImpl) GetRequestStats(ctx context.Context, timeRange string) (*docker.RequestStats, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	// 통계 복사본 생성
	stats := &docker.RequestStats{
		TotalRequests:     m.requestStats.TotalRequests,
		ManifestRequests:  m.requestStats.ManifestRequests,
		BlobRequests:      m.requestStats.BlobRequests,
		CacheHitRate:      m.requestStats.CacheHitRate,
		AverageResponseMs: m.calculateAverageResponseTime(),
		RepositoryStats:   make(map[string]*docker.RepositoryStat),
		RegistryUsage:     make(map[string]int64),
		ErrorCount:        m.requestStats.ErrorCount,
		BytesServed:       m.requestStats.BytesServed,
		PopularOperations: m.getPopularOperations(),
	}

	// 레포지토리 통계 복사
	for k, v := range m.repositoryMap {
		stats.RepositoryStats[k] = &docker.RepositoryStat{
			Name:           v.Name,
			RequestCount:   v.RequestCount,
			BytesServed:    v.BytesServed,
			CacheHitRate:   v.CacheHitRate,
			LastAccessTime: v.LastAccessTime,
			PopularTags:    make([]string, len(v.PopularTags)),
		}
		copy(stats.RepositoryStats[k].PopularTags, v.PopularTags)
	}

	// 레지스트리 사용량 복사
	for k, v := range m.requestStats.RegistryUsage {
		stats.RegistryUsage[k] = v
	}

	return stats, nil
}

// GetPopularImages 인기 이미지 통계 반환
func (m *metricsCollectorImpl) GetPopularImages(ctx context.Context, limit int) ([]docker.ImageStat, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	// 인기도순으로 정렬
	sortedImages := make([]docker.ImageStat, len(m.imageStats))
	copy(sortedImages, m.imageStats)

	sort.Slice(sortedImages, func(i, j int) bool {
		return sortedImages[i].PopularityScore > sortedImages[j].PopularityScore
	})

	// 제한된 수만 반환
	if limit > 0 && limit < len(sortedImages) {
		sortedImages = sortedImages[:limit]
	}

	return sortedImages, nil
}

// RecordError 에러 메트릭 기록
func (m *metricsCollectorImpl) RecordError(ctx context.Context, errorType, operation string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.errorStats.TotalErrors++
	m.errorStats.ErrorsByType[errorType]++
	m.errorStats.ErrorsByOperation[operation]++

	// 최근 에러 기록 (최대 50개)
	errorMsg := fmt.Sprintf("%s:%s at %s", errorType, operation, time.Now().Format(time.RFC3339))
	m.errorStats.RecentErrors = append(m.errorStats.RecentErrors, errorMsg)

	if len(m.errorStats.RecentErrors) > 50 {
		m.errorStats.RecentErrors = m.errorStats.RecentErrors[1:]
	}

	// 에러율 계산
	if m.requestStats.TotalRequests > 0 {
		m.errorStats.ErrorRate = float64(m.errorStats.TotalErrors) / float64(m.requestStats.TotalRequests)
	}

	m.logger.Warn("Error recorded", logging.F("type", errorType), logging.F("operation", operation))
	return nil
}

// GetErrorStats 에러 통계 반환
func (m *metricsCollectorImpl) GetErrorStats(ctx context.Context, timeRange string) (*docker.ErrorStats, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	// 에러 통계 복사본 생성
	stats := &docker.ErrorStats{
		TotalErrors:       m.errorStats.TotalErrors,
		ErrorsByType:      make(map[string]int64),
		ErrorsByOperation: make(map[string]int64),
		ErrorsByRegistry:  make(map[string]int64),
		RecentErrors:      make([]string, len(m.errorStats.RecentErrors)),
		ErrorRate:         m.errorStats.ErrorRate,
	}

	// 맵 복사
	for k, v := range m.errorStats.ErrorsByType {
		stats.ErrorsByType[k] = v
	}
	for k, v := range m.errorStats.ErrorsByOperation {
		stats.ErrorsByOperation[k] = v
	}
	for k, v := range m.errorStats.ErrorsByRegistry {
		stats.ErrorsByRegistry[k] = v
	}

	// 슬라이스 복사
	copy(stats.RecentErrors, m.errorStats.RecentErrors)

	return stats, nil
}

// updateRepositoryStats 레포지토리별 통계 업데이트
func (m *metricsCollectorImpl) updateRepositoryStats(metrics *docker.RequestMetrics) {
	if metrics.Repository == "" {
		return
	}

	if _, exists := m.repositoryMap[metrics.Repository]; !exists {
		m.repositoryMap[metrics.Repository] = &docker.RepositoryStat{
			Name:        metrics.Repository,
			PopularTags: make([]string, 0),
		}
	}

	repo := m.repositoryMap[metrics.Repository]
	repo.RequestCount++
	repo.BytesServed += metrics.BytesServed
	repo.LastAccessTime = metrics.Timestamp.Unix()

	// 캐시 히트율 업데이트 (간단한 이동 평균)
	if metrics.FromCache {
		repo.CacheHitRate = (repo.CacheHitRate * 0.9) + (1.0 * 0.1)
	} else {
		repo.CacheHitRate = (repo.CacheHitRate * 0.9) + (0.0 * 0.1)
	}

	// 인기 태그 업데이트
	if metrics.Reference != "" && metrics.Operation == "manifest" {
		m.updatePopularTags(repo, metrics.Reference)
	}
}

// updateImageStats 이미지별 통계 업데이트
func (m *metricsCollectorImpl) updateImageStats(metrics *docker.RequestMetrics) {
	if metrics.Repository == "" || metrics.Reference == "" {
		return
	}

	// 기존 이미지 통계 찾기
	var imageStat *docker.ImageStat
	for i := range m.imageStats {
		if m.imageStats[i].Repository == metrics.Repository && m.imageStats[i].Tag == metrics.Reference {
			imageStat = &m.imageStats[i]
			break
		}
	}

	// 새로운 이미지 통계 생성
	if imageStat == nil {
		m.imageStats = append(m.imageStats, docker.ImageStat{
			Repository:      metrics.Repository,
			Tag:             metrics.Reference,
			PullCount:       0,
			LastPulled:      0,
			SizeBytes:       0,
			LayerCount:      0,
			PopularityScore: 0.0,
		})
		imageStat = &m.imageStats[len(m.imageStats)-1]
	}

	// 통계 업데이트
	imageStat.PullCount++
	imageStat.LastPulled = metrics.Timestamp.Unix()
	imageStat.SizeBytes += metrics.BytesServed

	// 인기도 점수 계산 (풀 횟수 + 최근성)
	recency := float64(time.Since(time.Unix(imageStat.LastPulled, 0)).Hours()) / 24.0 // 일 단위
	imageStat.PopularityScore = float64(imageStat.PullCount) * (1.0 / (1.0 + recency))
}

// updatePopularTags 인기 태그 업데이트
func (m *metricsCollectorImpl) updatePopularTags(repo *docker.RepositoryStat, tag string) {
	// 태그가 이미 있는지 확인
	for _, existingTag := range repo.PopularTags {
		if existingTag == tag {
			return
		}
	}

	// 새 태그 추가 (최대 10개까지)
	repo.PopularTags = append(repo.PopularTags, tag)
	if len(repo.PopularTags) > 10 {
		repo.PopularTags = repo.PopularTags[1:]
	}
}

// updateCacheHitRate 캐시 히트율 업데이트
func (m *metricsCollectorImpl) updateCacheHitRate(hit bool) {
	// 이동 평균 계산
	if hit {
		m.requestStats.CacheHitRate = (m.requestStats.CacheHitRate * 0.95) + (1.0 * 0.05)
	} else {
		m.requestStats.CacheHitRate = (m.requestStats.CacheHitRate * 0.95) + (0.0 * 0.05)
	}
}

// recordResponseTime 응답 시간 기록
func (m *metricsCollectorImpl) recordResponseTime(duration int64) {
	// 이동 평균 계산
	if m.requestStats.AverageResponseMs == 0 {
		m.requestStats.AverageResponseMs = float64(duration)
	} else {
		m.requestStats.AverageResponseMs = (m.requestStats.AverageResponseMs * 0.9) + (float64(duration) * 0.1)
	}
}

// calculateAverageResponseTime 평균 응답 시간 계산
func (m *metricsCollectorImpl) calculateAverageResponseTime() float64 {
	return m.requestStats.AverageResponseMs
}

// getPopularOperations 인기 작업 목록 반환
func (m *metricsCollectorImpl) getPopularOperations() []string {
	operations := make([]string, 0)

	if m.requestStats.ManifestRequests > 0 {
		operations = append(operations, "manifest")
	}
	if m.requestStats.BlobRequests > 0 {
		operations = append(operations, "blob")
	}

	// 요청 수에 따라 정렬
	sort.Slice(operations, func(i, j int) bool {
		var countI, countJ int64
		switch operations[i] {
		case "manifest":
			countI = m.requestStats.ManifestRequests
		case "blob": //nolint:goconst
			countJ = m.requestStats.BlobRequests
		}
		switch operations[j] {
		case "manifest":
			countJ = m.requestStats.ManifestRequests
		case "blob": //nolint:goconst
			countJ = m.requestStats.BlobRequests
		}
		return countI > countJ
	})

	return operations
}

// recordErrorByStatus HTTP 상태 코드별 에러 기록
func (m *metricsCollectorImpl) recordErrorByStatus(statusCode int, operation string) {
	var errorType string
	switch {
	case statusCode >= 400 && statusCode < 500:
		errorType = "client_error"
	case statusCode >= 500:
		errorType = "server_error"
	default:
		errorType = "unknown_error"
	}

	m.errorStats.ErrorsByType[errorType]++
	m.errorStats.ErrorsByOperation[operation]++
}

// buildMetricKey 메트릭 키 생성 (태그 포함)
func (m *metricsCollectorImpl) buildMetricKey(name string, tags map[string]string) string {
	if len(tags) == 0 {
		return name
	}

	key := name
	for k, v := range tags {
		key += fmt.Sprintf("_%s:%s", k, v)
	}

	return key
}
