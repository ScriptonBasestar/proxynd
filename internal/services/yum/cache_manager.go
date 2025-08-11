package yum

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"proxynd/internal/domain/yum"
	"proxynd/logging"
)

// cacheManagerImpl YUM 캐시 관리 서비스 구현
type cacheManagerImpl struct {
	config     yum.ProxyConfig
	logger     logging.Logger
	storageDir string
	cache      map[string]*yum.CacheEntry
	mutex      sync.RWMutex
	stats      *cacheStats
}

type cacheStats struct {
	Hits    int64 `json:"hits"`
	Misses  int64 `json:"misses"`
	Sets    int64 `json:"sets"`
	Deletes int64 `json:"deletes"`
	Size    int64 `json:"size"`
}

// NewCacheManager YUM 캐시 관리자 생성
func NewCacheManager(
	config yum.ProxyConfig,
	logger logging.Logger,
	storageDir string,
) yum.CacheManager {
	manager := &cacheManagerImpl{
		config:     config,
		logger:     logger,
		storageDir: storageDir,
		cache:      make(map[string]*yum.CacheEntry),
		stats:      &cacheStats{},
	}

	// 기존 캐시 엔트리 로드
	manager.loadCacheIndex()

	return manager
}

// Get 캐시에서 데이터 조회
func (c *cacheManagerImpl) Get(ctx context.Context, key string) (*yum.CacheEntry, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	entry, exists := c.cache[key]
	if !exists {
		c.stats.Misses++
		return nil, fmt.Errorf("cache entry not found: %s", key)
	}

	// TTL 확인
	if time.Since(entry.CreatedAt) > entry.TTL {
		c.mutex.RUnlock()
		c.mutex.Lock()
		delete(c.cache, key)
		c.mutex.Unlock()
		c.mutex.RLock()

		c.stats.Misses++
		return nil, fmt.Errorf("cache entry expired: %s", key)
	}

	// 파일 존재 확인
	if _, err := os.Stat(entry.Path); os.IsNotExist(err) {
		c.mutex.RUnlock()
		c.mutex.Lock()
		delete(c.cache, key)
		c.mutex.Unlock()
		c.mutex.RLock()

		c.stats.Misses++
		return nil, fmt.Errorf("cache file not found: %s", entry.Path)
	}

	// 접근 시간 업데이트
	entry.LastAccessed = time.Now()
	c.stats.Hits++

	c.logger.Debug("YUM 캐시 히트",
		logging.F("key", key),
		logging.F("path", entry.Path))

	return entry, nil
}

// Set 캐시에 데이터 저장
func (c *cacheManagerImpl) Set(ctx context.Context, key string, data []byte, ttl int64) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// 캐시 크기 확인
	if c.getTotalCacheSize()+int64(len(data)) > c.config.GetMaxCacheSize() {
		if err := c.evictLRU(); err != nil {
			c.logger.Warn("YUM 캐시 LRU 제거 실패", logging.F("error", err.Error()))
		}
	}

	// 파일 경로 생성
	cachePath := c.getCachePath(key)
	cacheDir := filepath.Dir(cachePath)

	if err := os.MkdirAll(cacheDir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	// 파일 저장
	if err := os.WriteFile(cachePath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	// 캐시 엔트리 생성
	entry := &yum.CacheEntry{
		Key:          key,
		Path:         cachePath,
		ContentType:  c.getContentTypeFromKey(key),
		Size:         int64(len(data)),
		CreatedAt:    time.Now(),
		LastAccessed: time.Now(),
		TTL:          time.Duration(ttl),
		IsRepoMeta:   c.isRepoMetadataKey(key),
		IsRpmFile:    c.isRpmFileKey(key),
	}

	c.cache[key] = entry
	c.stats.Sets++
	c.stats.Size += int64(len(data))

	c.logger.Debug("YUM 캐시 저장",
		logging.F("key", key),
		logging.F("path", cachePath),
		logging.F("size", len(data)))

	// 캐시 인덱스 저장
	c.saveCacheIndex()

	return nil
}

// Delete 캐시에서 데이터 삭제
func (c *cacheManagerImpl) Delete(ctx context.Context, key string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	entry, exists := c.cache[key]
	if !exists {
		return nil // 이미 삭제됨
	}

	// 파일 삭제
	if err := os.Remove(entry.Path); err != nil && !os.IsNotExist(err) {
		c.logger.Warn("YUM 캐시 파일 삭제 실패",
			logging.F("path", entry.Path),
			logging.F("error", err.Error()))
	}

	// 캐시에서 제거
	c.stats.Size -= entry.Size
	delete(c.cache, key)
	c.stats.Deletes++

	c.logger.Debug("YUM 캐시 삭제", logging.F("key", key))

	return nil
}

// Clear 전체 캐시 삭제
func (c *cacheManagerImpl) Clear(ctx context.Context) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	for key, entry := range c.cache {
		if err := os.Remove(entry.Path); err != nil && !os.IsNotExist(err) {
			c.logger.Warn("YUM 캐시 파일 삭제 실패",
				logging.F("path", entry.Path),
				logging.F("error", err.Error()))
		}
		delete(c.cache, key)
	}

	c.stats = &cacheStats{}

	c.logger.Info("YUM 전체 캐시 삭제 완료")

	return nil
}

// GetStats 캐시 통계 조회
func (c *cacheManagerImpl) GetStats(ctx context.Context) (map[string]interface{}, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	stats := map[string]interface{}{
		"hits":        c.stats.Hits,
		"misses":      c.stats.Misses,
		"sets":        c.stats.Sets,
		"deletes":     c.stats.Deletes,
		"size":        c.stats.Size,
		"entries":     len(c.cache),
		"hit_ratio":   c.calculateHitRatio(),
		"max_size":    c.config.GetMaxCacheSize(),
		"repo_meta":   c.countByType(true, false),
		"rpm_files":   c.countByType(false, true),
		"other_files": c.countByType(false, false),
	}

	return stats, nil
}

// Cleanup 만료된 캐시 정리
func (c *cacheManagerImpl) Cleanup(ctx context.Context) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	var expiredKeys []string
	now := time.Now()

	for key, entry := range c.cache {
		if now.Sub(entry.CreatedAt) > entry.TTL {
			expiredKeys = append(expiredKeys, key)
		}
	}

	for _, key := range expiredKeys {
		entry := c.cache[key]
		if err := os.Remove(entry.Path); err != nil && !os.IsNotExist(err) {
			c.logger.Warn("YUM 만료된 캐시 파일 삭제 실패",
				logging.F("path", entry.Path),
				logging.F("error", err.Error()))
		}

		c.stats.Size -= entry.Size
		delete(c.cache, key)
	}

	c.logger.Info("YUM 만료된 캐시 정리 완료",
		logging.F("expired_count", len(expiredKeys)))

	return nil
}

// 헬퍼 메서드들

func (c *cacheManagerImpl) getCachePath(key string) string {
	// yum:path/to/file.rpm -> cache/yum/path/to/file.rpm
	keyPath := key[4:] // "yum:" 제거
	return filepath.Join(c.storageDir, "cache", "yum", keyPath)
}

func (c *cacheManagerImpl) getContentTypeFromKey(key string) string {
	if c.isRpmFileKey(key) {
		return "application/x-rpm"
	} else if c.isRepoMetadataKey(key) {
		return "application/xml"
	}
	return "application/octet-stream"
}

func (c *cacheManagerImpl) isRepoMetadataKey(key string) bool {
	return strings.Contains(key, "repomd.xml") ||
		strings.Contains(key, "repodata/") ||
		strings.HasSuffix(key, ".xml") ||
		strings.HasSuffix(key, ".xml.gz")
}

func (c *cacheManagerImpl) isRpmFileKey(key string) bool {
	return strings.HasSuffix(key, ".rpm")
}

func (c *cacheManagerImpl) getTotalCacheSize() int64 {
	var total int64
	for _, entry := range c.cache {
		total += entry.Size
	}
	return total
}

func (c *cacheManagerImpl) evictLRU() error {
	var oldestKey string
	oldestTime := time.Now()

	for key, entry := range c.cache {
		if entry.LastAccessed.Before(oldestTime) {
			oldestTime = entry.LastAccessed
			oldestKey = key
		}
	}

	if oldestKey != "" {
		return c.Delete(context.Background(), oldestKey)
	}

	return nil
}

func (c *cacheManagerImpl) calculateHitRatio() float64 {
	total := c.stats.Hits + c.stats.Misses
	if total == 0 {
		return 0.0
	}
	return float64(c.stats.Hits) / float64(total)
}

func (c *cacheManagerImpl) countByType(repoMeta, rpmFile bool) int {
	count := 0
	for _, entry := range c.cache {
		if repoMeta && entry.IsRepoMeta {
			count++
		} else if rpmFile && entry.IsRpmFile {
			count++
		} else if !repoMeta && !rpmFile && !entry.IsRepoMeta && !entry.IsRpmFile {
			count++
		}
	}
	return count
}

func (c *cacheManagerImpl) loadCacheIndex() {
	indexPath := filepath.Join(c.storageDir, "cache", "yum", "index.json")

	data, err := os.ReadFile(indexPath)
	if err != nil {
		if !os.IsNotExist(err) {
			c.logger.Warn("YUM 캐시 인덱스 로드 실패",
				logging.F("error", err.Error()))
		}
		return
	}

	var entries map[string]*yum.CacheEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		c.logger.Warn("YUM 캐시 인덱스 파싱 실패",
			logging.F("error", err.Error()))
		return
	}

	c.cache = entries
	c.logger.Info("YUM 캐시 인덱스 로드 완료",
		logging.F("entries", len(entries)))
}

func (c *cacheManagerImpl) saveCacheIndex() {
	indexPath := filepath.Join(c.storageDir, "cache", "yum", "index.json")
	indexDir := filepath.Dir(indexPath)

	if err := os.MkdirAll(indexDir, os.ModePerm); err != nil {
		c.logger.Warn("YUM 캐시 인덱스 디렉토리 생성 실패",
			logging.F("error", err.Error()))
		return
	}

	data, err := json.Marshal(c.cache)
	if err != nil {
		c.logger.Warn("YUM 캐시 인덱스 직렬화 실패",
			logging.F("error", err.Error()))
		return
	}

	if err := os.WriteFile(indexPath, data, 0o644); err != nil {
		c.logger.Warn("YUM 캐시 인덱스 저장 실패",
			logging.F("error", err.Error()))
	}
}
