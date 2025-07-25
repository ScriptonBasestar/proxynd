package maven

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"time"

	"proxynd/internal/domain/maven"
	"proxynd/logging"
)

// cacheManagerImpl CacheManager 인터페이스 구현
type cacheManagerImpl struct {
	config       maven.ProxyConfig
	logger       logging.Logger
	cache        map[string]*cacheItem
	cacheMutex   sync.RWMutex
	stats        *cacheStats
	statsMutex   sync.RWMutex
	popularPaths []string
}

// cacheItem 캐시 항목
type cacheItem struct {
	Key         string      `json:"key"`
	Data        interface{} `json:"data"`
	ExpiresAt   time.Time   `json:"expiresAt"`
	CreatedAt   time.Time   `json:"createdAt"`
	AccessedAt  time.Time   `json:"accessedAt"`
	AccessCount int64       `json:"accessCount"`
}

// cacheStats 캐시 통계
type cacheStats struct {
	TotalEntries int64 `json:"totalEntries"`
	HitCount     int64 `json:"hitCount"`
	MissCount    int64 `json:"missCount"`
	MemoryUsage  int64 `json:"memoryUsage"`
}

// NewCacheManager CacheManager 생성자
func NewCacheManager(config maven.ProxyConfig, logger logging.Logger) maven.CacheManager {
	manager := &cacheManagerImpl{
		config: config,
		logger: logger,
		cache:  make(map[string]*cacheItem),
		stats:  &cacheStats{},
		popularPaths: []string{
			// 인기 Maven 경로들
			"org/springframework",
			"org/apache/commons",
			"com/google/guava",
			"org/slf4j",
			"ch/qos/logback",
			"junit/junit",
			"org/mockito",
			"com/fasterxml/jackson/core",
			"org/hibernate",
			"mysql/mysql-connector-java",
		},
	}

	// 정기적인 캐시 정리 작업 시작
	go manager.startCleanupRoutine()

	return manager
}

// Get 캐시에서 데이터 조회
func (c *cacheManagerImpl) Get(ctx context.Context, key string) (*maven.CacheEntry, bool) {
	c.cacheMutex.RLock()
	item, exists := c.cache[key]
	c.cacheMutex.RUnlock()

	if !exists {
		c.recordMiss()
		return nil, false
	}

	// 만료 확인
	if time.Now().After(item.ExpiresAt) {
		c.cacheMutex.Lock()
		delete(c.cache, key)
		c.cacheMutex.Unlock()
		c.recordMiss()
		return nil, false
	}

	// 액세스 정보 업데이트
	c.cacheMutex.Lock()
	item.AccessedAt = time.Now()
	item.AccessCount++
	c.cacheMutex.Unlock()

	c.recordHit()

	entry := &maven.CacheEntry{
		Key:        item.Key,
		Data:       item.Data,
		ExpiresAt:  item.ExpiresAt,
		CreatedAt:  item.CreatedAt,
		AccessedAt: item.AccessedAt,
	}

	c.logger.Debug("Cache hit",
		logging.F("key", key),
		logging.F("accessCount", item.AccessCount),
	)

	return entry, true
}

// Set 캐시에 데이터 저장
func (c *cacheManagerImpl) Set(ctx context.Context, key string, data interface{}, ttl time.Duration) error {
	now := time.Now()
	expiresAt := now.Add(ttl)

	item := &cacheItem{
		Key:         key,
		Data:        data,
		ExpiresAt:   expiresAt,
		CreatedAt:   now,
		AccessedAt:  now,
		AccessCount: 0,
	}

	c.cacheMutex.Lock()
	c.cache[key] = item
	c.cacheMutex.Unlock()

	c.logger.Debug("Cache set",
		logging.F("key", key),
		logging.F("ttl", ttl),
		logging.F("expiresAt", expiresAt),
	)

	// 캐시 크기 제한 확인
	c.evictIfNeeded()

	return nil
}

// Delete 캐시에서 데이터 삭제
func (c *cacheManagerImpl) Delete(ctx context.Context, key string) error {
	c.cacheMutex.Lock()
	delete(c.cache, key)
	c.cacheMutex.Unlock()

	c.logger.Debug("Cache delete", logging.F("key", key))

	return nil
}

// PreloadPopularPaths 인기 경로 사전 캐싱
func (c *cacheManagerImpl) PreloadPopularPaths(ctx context.Context) error {
	c.logger.Info("Starting popular paths preloading",
		logging.F("pathCount", len(c.popularPaths)),
	)

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 5) // 동시 실행 제한

	for _, path := range c.popularPaths {
		wg.Add(1)
		go func(p string) {
			defer wg.Done()
			semaphore <- struct{}{}        // 세마포어 획득
			defer func() { <-semaphore }() // 세마포어 해제

			if err := c.preloadPath(ctx, p); err != nil {
				c.logger.Warn("Failed to preload path",
					logging.F("path", p),
					logging.F("error", err),
				)
			}
		}(path)
	}

	wg.Wait()

	c.logger.Info("Popular paths preloading completed")
	return nil
}

// GetStats 캐시 통계 조회
func (c *cacheManagerImpl) GetStats(ctx context.Context) (*maven.CacheStats, error) {
	c.statsMutex.RLock()
	c.cacheMutex.RLock()

	totalEntries := int64(len(c.cache))
	hitCount := c.stats.HitCount
	missCount := c.stats.MissCount

	c.cacheMutex.RUnlock()
	c.statsMutex.RUnlock()

	// 히트율 계산
	totalRequests := hitCount + missCount
	var hitRate, missRate float64
	if totalRequests > 0 {
		hitRate = float64(hitCount) / float64(totalRequests)
		missRate = float64(missCount) / float64(totalRequests)
	}

	// 메모리 사용량 추정 (간단한 계산)
	memoryUsage := c.estimateMemoryUsage()

	stats := &maven.CacheStats{
		TotalEntries: int(totalEntries),
		HitRate:      hitRate,
		MissRate:     missRate,
		MemoryUsage:  memoryUsage,
	}

	return stats, nil
}

// preloadPath 특정 경로를 사전 캐싱
func (c *cacheManagerImpl) preloadPath(ctx context.Context, path string) error {
	// 이미 캐시된 경우 스킵
	key := fmt.Sprintf("directory:%s", path)
	if _, exists := c.Get(ctx, key); exists {
		return nil
	}

	c.logger.Debug("Preloading path", logging.F("path", path))

	// 실제 구현에서는 DirectoryCollector를 사용하여 데이터 수집
	// 여기서는 placeholder 데이터 사용
	placeholderData := map[string]interface{}{
		"path":        path,
		"preloaded":   true,
		"preloadTime": time.Now(),
	}

	// 1시간 TTL로 캐싱
	return c.Set(ctx, key, placeholderData, time.Hour)
}

// recordHit 캐시 히트 기록
func (c *cacheManagerImpl) recordHit() {
	c.statsMutex.Lock()
	c.stats.HitCount++
	c.statsMutex.Unlock()
}

// recordMiss 캐시 미스 기록
func (c *cacheManagerImpl) recordMiss() {
	c.statsMutex.Lock()
	c.stats.MissCount++
	c.statsMutex.Unlock()
}

// estimateMemoryUsage 메모리 사용량 추정
func (c *cacheManagerImpl) estimateMemoryUsage() int64 {
	c.cacheMutex.RLock()
	defer c.cacheMutex.RUnlock()

	var totalSize int64
	for _, item := range c.cache {
		// JSON 직렬화로 대략적인 크기 추정
		if data, err := json.Marshal(item); err == nil {
			totalSize += int64(len(data))
		}
	}

	return totalSize
}

// evictIfNeeded 필요시 캐시 제거
func (c *cacheManagerImpl) evictIfNeeded() {
	c.cacheMutex.RLock()
	cacheSize := len(c.cache)
	c.cacheMutex.RUnlock()

	// 최대 캐시 크기 제한 (기본값: 1000)
	maxCacheSize := 1000

	if cacheSize <= maxCacheSize {
		return
	}

	c.logger.Debug("Cache eviction needed",
		logging.F("currentSize", cacheSize),
		logging.F("maxSize", maxCacheSize),
	)

	// LRU 제거 정책 구현
	c.evictLRU(cacheSize - maxCacheSize + 100) // 여유분 확보
}

// evictLRU LRU 정책으로 캐시 항목 제거
func (c *cacheManagerImpl) evictLRU(evictCount int) {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()

	// 액세스 시간 순으로 정렬하기 위한 슬라이스
	type keyTime struct {
		key        string
		accessedAt time.Time
	}

	var items []keyTime
	for key, item := range c.cache {
		items = append(items, keyTime{
			key:        key,
			accessedAt: item.AccessedAt,
		})
	}

	// 가장 오래된 항목부터 정렬
	sort.Slice(items, func(i, j int) bool {
		return items[i].accessedAt.Before(items[j].accessedAt)
	})

	// 지정된 수만큼 제거
	evicted := 0
	for _, item := range items {
		if evicted >= evictCount {
			break
		}
		delete(c.cache, item.key)
		evicted++
	}

	c.logger.Info("Cache LRU eviction completed",
		logging.F("evictedCount", evicted),
	)
}

// startCleanupRoutine 정기적인 캐시 정리 작업 시작
func (c *cacheManagerImpl) startCleanupRoutine() {
	ticker := time.NewTicker(5 * time.Minute) // 5분마다 실행
	defer ticker.Stop()

	for range ticker.C {
		c.cleanupExpiredEntries()
	}
}

// cleanupExpiredEntries 만료된 캐시 항목 정리
func (c *cacheManagerImpl) cleanupExpiredEntries() {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()

	now := time.Now()
	expiredKeys := make([]string, 0, len(c.cache)/10) // 예상 만료 항목 수

	for key, item := range c.cache {
		if now.After(item.ExpiresAt) {
			expiredKeys = append(expiredKeys, key)
		}
	}

	// 만료된 항목 삭제
	for _, key := range expiredKeys {
		delete(c.cache, key)
	}

	if len(expiredKeys) > 0 {
		c.logger.Debug("Cache cleanup completed",
			logging.F("expiredCount", len(expiredKeys)),
		)
	}
}
