package docker

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"proxynd/internal/domain/docker"
	"proxynd/internal/logging"
	"proxynd/internal/security"
)

const (
	dockerResourceManifest = "manifest"
	dockerResourceBlob     = "blob"
)

// cacheManagerImpl Docker 캐시 관리 구현
type cacheManagerImpl struct {
	config     docker.ProxyConfig
	logger     logging.Logger
	stats      *docker.CacheStats
	statsMutex sync.RWMutex
}

// NewCacheManager CacheManager 생성자
func NewCacheManager(
	config docker.ProxyConfig,
	logger logging.Logger,
) docker.CacheManager {
	return &cacheManagerImpl{
		config: config,
		logger: logger,
		stats: &docker.CacheStats{
			TotalEntries:    0,
			ManifestEntries: 0,
			BlobEntries:     0,
			TotalSizeBytes:  0,
			HitRate:         0.0,
			MissRate:        0.0,
			EvictionCount:   0,
		},
		statsMutex: sync.RWMutex{},
	}
}

// Get 캐시에서 데이터 조회
func (c *cacheManagerImpl) Get(ctx context.Context, key string) (*docker.CacheEntry, error) {
	c.logger.Debug("Getting cache entry", logging.F("key", key))

	// 캐시 파일 경로 생성
	filePath, err := c.buildCachePath(key)
	if err != nil {
		return nil, fmt.Errorf("failed to build cache path: %w", err)
	}

	// 메타데이터 파일 확인
	metaPath := filePath + ".meta"
	if _, err := os.Stat(metaPath); os.IsNotExist(err) {
		c.updateMissRate()
		return nil, fmt.Errorf("cache entry not found: %s", key)
	}

	// 메타데이터 로드
	entry, err := c.loadCacheEntry(metaPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load cache entry: %w", err)
	}

	// 유효성 검증
	if !c.ValidateCacheEntry(ctx, entry) {
		_ = c.Delete(ctx, key)
		c.updateMissRate()
		return nil, fmt.Errorf("cache entry expired or invalid: %s", key)
	}

	// 실제 데이터 로드
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		_ = c.Delete(ctx, key)
		return nil, fmt.Errorf("cache data file not found: %s", key)
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read cache data: %w", err)
	}

	// 데이터 할당 및 접근 시간 업데이트
	entry.Headers = data
	entry.LastAccessed = time.Now()

	// 메타데이터 업데이트 (접근 시간)
	_ = c.saveCacheEntry(metaPath, entry)

	c.updateHitRate()
	c.logger.Debug("Cache hit", logging.F("key", key), logging.F("size", len(data)))

	return entry, nil
}

// Set 캐시에 데이터 저장 (TTL, 타입별 분리)
func (c *cacheManagerImpl) Set(ctx context.Context, entry *docker.CacheEntry) error {
	c.logger.Debug("Setting cache entry", logging.F("key", entry.Key), logging.F("size", entry.Size))

	// 캐시가 비활성화된 경우
	if !c.config.IsCacheEnabled() {
		return nil
	}

	// 캐시 파일 경로 생성
	filePath, err := c.buildCachePath(entry.Key)
	if err != nil {
		return fmt.Errorf("failed to build cache path: %w", err)
	}

	// 디렉토리 생성
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	// 캐시 크기 제한 확인
	cacheConfig := c.config.GetCacheConfig()
	if c.shouldEvict(entry.Size, cacheConfig.MaxSizeMB) {
		if err := c.evictOldEntries(ctx, entry.Size); err != nil {
			c.logger.Warn("Failed to evict old entries", logging.F("error", err))
		}
	}

	// 데이터 파일 저장
	if err := os.WriteFile(filePath, entry.Headers, 0o644); err != nil {
		return fmt.Errorf("failed to write cache data: %w", err)
	}

	// 메타데이터 저장
	metaPath := filePath + ".meta"
	entry.Path = filePath
	entry.CreatedAt = time.Now()
	entry.LastAccessed = time.Now()

	if err := c.saveCacheEntry(metaPath, entry); err != nil {
		// 메타데이터 저장 실패 시 데이터 파일도 삭제
		_ = os.Remove(filePath)
		return fmt.Errorf("failed to save cache metadata: %w", err)
	}

	// 통계 업데이트
	c.updateCacheStats(entry, true)

	c.logger.Debug("Cache entry saved", logging.F("key", entry.Key), logging.F("path", filePath))
	return nil
}

// Delete 캐시 엔트리 삭제
func (c *cacheManagerImpl) Delete(ctx context.Context, key string) error {
	c.logger.Debug("Deleting cache entry", logging.F("key", key))

	filePath, err := c.buildCachePath(key)
	if err != nil {
		return fmt.Errorf("failed to build cache path: %w", err)
	}

	metaPath := filePath + ".meta"

	// 통계 업데이트용 정보 로드
	if entry, err := c.loadCacheEntry(metaPath); err == nil {
		c.updateCacheStats(entry, false)
	}

	// 파일 삭제
	_ = os.Remove(filePath)
	_ = os.Remove(metaPath)
	_ = os.Remove(filePath + ".headers") // 헤더 파일도 삭제

	return nil
}

// GenerateCacheKey 캐시 키 생성 (레포지토리, 참조, 타입별)
func (c *cacheManagerImpl) GenerateCacheKey(repository, reference, operation string) string {
	// SHA256 해시로 키 생성 (긴 경로 방지)
	input := fmt.Sprintf("%s:%s:%s", repository, reference, operation)
	hash := sha256.Sum256([]byte(input))
	hashStr := fmt.Sprintf("%x", hash)

	// 타입별 접두사 추가
	var prefix string
	switch operation {
	case dockerResourceManifest:
		prefix = "m"
	case dockerResourceBlob:
		prefix = "b"
	case "tags":
		prefix = "t"
	case "catalog":
		prefix = "c"
	default:
		prefix = "o"
	}

	return fmt.Sprintf("%s_%s", prefix, hashStr[:16])
}

// ValidateCacheEntry 캐시 엔트리 유효성 검증 (TTL, 무결성)
func (c *cacheManagerImpl) ValidateCacheEntry(ctx context.Context, entry *docker.CacheEntry) bool {
	if entry == nil {
		return false
	}

	// TTL 확인
	if time.Since(entry.CreatedAt) > entry.TTL {
		c.logger.Debug("Cache entry expired", logging.F("key", entry.Key), logging.F("age", time.Since(entry.CreatedAt)))
		return false
	}

	// 파일 존재 확인
	if entry.Path != "" {
		if _, err := os.Stat(entry.Path); os.IsNotExist(err) {
			c.logger.Debug("Cache file not found", logging.F("path", entry.Path))
			return false
		}
	}

	// 다이제스트 검증 (blob의 경우)
	if entry.IsBlob && entry.Digest != "" {
		if len(entry.Headers) > 0 {
			hash := sha256.Sum256(entry.Headers)
			actualDigest := fmt.Sprintf("sha256:%x", hash)
			if actualDigest != entry.Digest {
				c.logger.Warn("Cache entry digest mismatch", logging.F("key", entry.Key))
				return false
			}
		}
	}

	return true
}

// CleanupExpired 만료된 캐시 엔트리 정리
func (c *cacheManagerImpl) CleanupExpired(ctx context.Context) error {
	c.logger.Debug("Starting cache cleanup")

	cacheDir := c.config.GetCacheConfig().BaseDir
	if cacheDir == "" {
		return nil
	}

	dockerCacheDir := filepath.Join(cacheDir, c.config.GetPath())

	var removedCount int64
	var reclaimedBytes int64

	err := filepath.Walk(dockerCacheDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // 계속 진행
		}

		// 메타데이터 파일만 처리
		if !strings.HasSuffix(path, ".meta") {
			return nil
		}

		// 메타데이터 로드
		entry, err := c.loadCacheEntry(path)
		if err != nil {
			// 손상된 메타데이터 파일 삭제
			_ = os.Remove(path)
			return nil
		}

		// 만료 확인
		if !c.ValidateCacheEntry(ctx, entry) {
			_ = c.Delete(ctx, entry.Key)
			removedCount++
			reclaimedBytes += entry.Size
		}

		return nil
	})
	if err != nil {
		c.logger.Warn("Cache cleanup completed with errors", logging.F("error", err))
	}

	c.logger.Info("Cache cleanup completed",
		logging.F("removed_count", removedCount),
		logging.F("reclaimed_bytes", reclaimedBytes))

	// 통계 업데이트
	c.statsMutex.Lock()
	c.stats.EvictionCount += removedCount
	c.statsMutex.Unlock()

	return nil
}

// GetCacheStats 캐시 통계 정보 반환
func (c *cacheManagerImpl) GetCacheStats(ctx context.Context) (*docker.CacheStats, error) {
	c.statsMutex.RLock()
	defer c.statsMutex.RUnlock()

	// 통계 복사본 반환
	return &docker.CacheStats{
		TotalEntries:     c.stats.TotalEntries,
		ManifestEntries:  c.stats.ManifestEntries,
		BlobEntries:      c.stats.BlobEntries,
		TotalSizeBytes:   c.stats.TotalSizeBytes,
		HitRate:          c.stats.HitRate,
		MissRate:         c.stats.MissRate,
		EvictionCount:    c.stats.EvictionCount,
		OldestEntryAge:   c.stats.OldestEntryAge,
		AverageEntrySize: c.calculateAverageEntrySize(),
	}, nil
}

// SaveHeaders 매니페스트 헤더 정보 별도 저장
func (c *cacheManagerImpl) SaveHeaders(ctx context.Context, key string, headers http.Header) error {
	filePath, err := c.buildCachePath(key)
	if err != nil {
		return err
	}

	headerPath := filePath + ".headers"

	// 헤더를 JSON으로 직렬화
	headerMap := make(map[string][]string)
	for k, v := range headers {
		headerMap[k] = v
	}

	data, err := json.Marshal(headerMap)
	if err != nil {
		return fmt.Errorf("failed to marshal headers: %w", err)
	}

	return os.WriteFile(headerPath, data, 0o644)
}

// LoadHeaders 저장된 헤더 정보 로드
func (c *cacheManagerImpl) LoadHeaders(ctx context.Context, key string) (http.Header, error) {
	filePath, err := c.buildCachePath(key)
	if err != nil {
		return nil, err
	}

	headerPath := filePath + ".headers"

	data, err := os.ReadFile(headerPath)
	if err != nil {
		return nil, err
	}

	var headerMap map[string][]string
	if err := json.Unmarshal(data, &headerMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal headers: %w", err)
	}

	headers := make(http.Header)
	for k, v := range headerMap {
		for _, value := range v {
			headers.Add(k, value)
		}
	}

	return headers, nil
}

// buildCachePath 캐시 파일 경로 생성
func (c *cacheManagerImpl) buildCachePath(key string) (string, error) {
	cacheConfig := c.config.GetCacheConfig()
	baseDir := filepath.Join(cacheConfig.BaseDir, c.config.GetPath())

	// 보안 검증된 경로 결합
	return security.SafeJoinPath(baseDir, key)
}

// loadCacheEntry 캐시 엔트리 메타데이터 로드
func (c *cacheManagerImpl) loadCacheEntry(metaPath string) (*docker.CacheEntry, error) {
	data, err := os.ReadFile(metaPath)
	if err != nil {
		return nil, err
	}

	var entry docker.CacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, err
	}

	return &entry, nil
}

// saveCacheEntry 캐시 엔트리 메타데이터 저장
func (c *cacheManagerImpl) saveCacheEntry(metaPath string, entry *docker.CacheEntry) error {
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	return os.WriteFile(metaPath, data, 0o644)
}

// shouldEvict 캐시 제거 필요 여부 확인
func (c *cacheManagerImpl) shouldEvict(newEntrySize, maxSizeMB int64) bool {
	c.statsMutex.RLock()
	defer c.statsMutex.RUnlock()

	maxSizeBytes := maxSizeMB * 1024 * 1024
	return c.stats.TotalSizeBytes+newEntrySize > maxSizeBytes
}

// evictOldEntries 오래된 엔트리 제거
func (c *cacheManagerImpl) evictOldEntries(ctx context.Context, requiredSize int64) error {
	// LRU 기반 제거 구현 (단순화)
	// TODO: 더 정교한 제거 정책 구현
	return c.CleanupExpired(ctx)
}

// updateCacheStats 캐시 통계 업데이트
func (c *cacheManagerImpl) updateCacheStats(entry *docker.CacheEntry, isAdd bool) {
	c.statsMutex.Lock()
	defer c.statsMutex.Unlock()

	if isAdd {
		c.stats.TotalEntries++
		c.stats.TotalSizeBytes += entry.Size

		if entry.IsManifest {
			c.stats.ManifestEntries++
		} else if entry.IsBlob {
			c.stats.BlobEntries++
		}
	} else {
		c.stats.TotalEntries--
		c.stats.TotalSizeBytes -= entry.Size

		if entry.IsManifest {
			c.stats.ManifestEntries--
		} else if entry.IsBlob {
			c.stats.BlobEntries--
		}
	}
}

// updateHitRate 캐시 히트율 업데이트
func (c *cacheManagerImpl) updateHitRate() {
	// 간단한 구현 (실제로는 더 정교한 추적 필요)
	c.statsMutex.Lock()
	defer c.statsMutex.Unlock()
	c.stats.HitRate += 0.01 // 임시 구현
}

// updateMissRate 캐시 미스율 업데이트
func (c *cacheManagerImpl) updateMissRate() {
	c.statsMutex.Lock()
	defer c.statsMutex.Unlock()
	c.stats.MissRate += 0.01 // 임시 구현
}

// calculateAverageEntrySize 평균 엔트리 크기 계산
func (c *cacheManagerImpl) calculateAverageEntrySize() int64 {
	if c.stats.TotalEntries == 0 {
		return 0
	}
	return c.stats.TotalSizeBytes / c.stats.TotalEntries
}
