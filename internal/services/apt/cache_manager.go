package apt

import (
	"context"
	"crypto/md5"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"proxynd/internal/domain/apt"
	"proxynd/internal/security"
	"proxynd/logging"
)

// cacheManagerImpl 캐시 관리 서비스 구현
type cacheManagerImpl struct {
	config      apt.ProxyConfig
	logger      logging.Logger
	stats       *apt.CacheStats
	statsMutex  sync.RWMutex
	lastCleanup time.Time
}

// NewCacheManager CacheManager 생성자
func NewCacheManager(config apt.ProxyConfig, logger logging.Logger) apt.CacheManager {
	manager := &cacheManagerImpl{
		config: config,
		logger: logger,
		stats: &apt.CacheStats{
			LastCleanup: time.Now().Format("2006-01-02 15:04:05"),
		},
		lastCleanup: time.Now(),
	}

	// 백그라운드 정리 작업 시작
	go manager.startCleanupWorker()

	return manager
}

// Get 캐시에서 패키지 조회
func (c *cacheManagerImpl) Get(ctx context.Context, key string) (*apt.CacheEntry, error) {
	cacheConfig := c.config.GetCacheConfig()
	if !cacheConfig.Enabled {
		return nil, fmt.Errorf("cache disabled")
	}

	// 캐시 파일 경로 생성
	cachePath := c.getCachePath(key)

	// 파일 존재 여부 확인
	fileInfo, err := os.Stat(cachePath)
	if err != nil {
		if os.IsNotExist(err) {
			c.updateStats(func(s *apt.CacheStats) {
				// Miss count는 별도 메트릭에서 관리
			})
			return nil, fmt.Errorf("cache miss")
		}
		return nil, fmt.Errorf("failed to stat cache file: %w", err)
	}

	// TTL 확인
	if time.Since(fileInfo.ModTime()) > cacheConfig.TTL {
		c.logger.Debug("Cache entry expired", logging.F("key", key))
		os.Remove(cachePath) // 만료된 파일 삭제
		return nil, fmt.Errorf("cache entry expired")
	}

	// 캐시 엔트리 생성
	entry := &apt.CacheEntry{
		Key:          key,
		Path:         cachePath,
		ContentType:  c.guessContentType(cachePath),
		Size:         fileInfo.Size(),
		CreatedAt:    fileInfo.ModTime(),
		LastAccessed: time.Now(),
		TTL:          cacheConfig.TTL,
	}

	c.logger.Debug("Cache hit", logging.F("key", key), logging.F("size", fileInfo.Size()))

	c.updateStats(func(s *apt.CacheStats) {
		// Hit rate는 별도 메트릭에서 계산
	})

	return entry, nil
}

// Set 패키지를 캐시에 저장
func (c *cacheManagerImpl) Set(ctx context.Context, key string, data []byte, contentType string) error {
	cacheConfig := c.config.GetCacheConfig()
	if !cacheConfig.Enabled {
		return nil // 캐시 비활성화 시 무시
	}

	cachePath := c.getCachePath(key)

	// 디렉토리 생성
	dir := filepath.Dir(cachePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	// 임시 파일에 쓰고 원자적으로 이동
	tmpPath := cachePath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	if err := os.Rename(tmpPath, cachePath); err != nil {
		os.Remove(tmpPath) // 실패 시 임시 파일 정리
		return fmt.Errorf("failed to rename cache file: %w", err)
	}

	c.logger.Debug("Cache entry saved",
		logging.F("key", key),
		logging.F("size", len(data)),
		logging.F("contentType", contentType),
	)

	c.updateStats(func(s *apt.CacheStats) {
		s.TotalEntries++
		s.TotalSizeMB += int64(len(data)) / (1024 * 1024)
	})

	return nil
}

// Delete 캐시에서 패키지 삭제
func (c *cacheManagerImpl) Delete(ctx context.Context, key string) error {
	cachePath := c.getCachePath(key)

	if err := os.Remove(cachePath); err != nil {
		if os.IsNotExist(err) {
			return nil // 이미 없음
		}
		return fmt.Errorf("failed to delete cache file: %w", err)
	}

	c.logger.Debug("Cache entry deleted", logging.F("key", key))

	c.updateStats(func(s *apt.CacheStats) {
		s.TotalEntries--
	})

	return nil
}

// GenerateKey 캐시 키 생성
func (c *cacheManagerImpl) GenerateKey(osType, packagePath string) string {
	// MD5 해시로 안전한 파일명 생성
	data := fmt.Sprintf("%s/%s", osType, packagePath)
	hash := fmt.Sprintf("%x", md5.Sum([]byte(data)))

	// 원본 경로도 포함하여 디버깅 용이성 확보
	safePath := strings.ReplaceAll(packagePath, "/", "_")
	safePath = strings.ReplaceAll(safePath, "..", "_")

	return fmt.Sprintf("%s_%s_%s", osType, safePath, hash[:8])
}

// Cleanup 오래된 캐시 파일 정리
func (c *cacheManagerImpl) Cleanup(ctx context.Context) error {
	cacheConfig := c.config.GetCacheConfig()
	if !cacheConfig.Enabled {
		return nil
	}

	startTime := time.Now()
	c.logger.Info("Starting cache cleanup")

	baseDir := cacheConfig.BaseDir
	var cleanedFiles int64
	var cleanedSize int64

	err := filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // 에러가 있는 파일은 건너뛰기
		}

		if info.IsDir() {
			return nil
		}

		// TTL 확인
		if time.Since(info.ModTime()) > cacheConfig.TTL {
			if err := os.Remove(path); err != nil {
				c.logger.Warn("Failed to remove expired cache file",
					logging.F("path", path),
					logging.F("error", err),
				)
			} else {
				cleanedFiles++
				cleanedSize += info.Size()
				c.logger.Debug("Removed expired cache file", logging.F("path", path))
			}
		}

		return nil
	})

	if err != nil {
		c.logger.Error("Cache cleanup walk failed", logging.F("error", err))
		return err
	}

	c.updateStats(func(s *apt.CacheStats) {
		s.CleanupCount++
		s.LastCleanup = time.Now().Format("2006-01-02 15:04:05")
		s.TotalEntries -= cleanedFiles
		s.TotalSizeMB -= cleanedSize / (1024 * 1024)
	})

	c.lastCleanup = time.Now()

	c.logger.Info("Cache cleanup completed",
		logging.F("cleanedFiles", cleanedFiles),
		logging.F("cleanedSizeMB", cleanedSize/(1024*1024)),
		logging.F("duration_ms", time.Since(startTime).Milliseconds()),
	)

	return nil
}

// GetStats 캐시 통계 조회
func (c *cacheManagerImpl) GetStats(ctx context.Context) (*apt.CacheStats, error) {
	c.statsMutex.RLock()
	defer c.statsMutex.RUnlock()

	// 현재 통계의 복사본 반환
	stats := &apt.CacheStats{
		TotalEntries: c.stats.TotalEntries,
		TotalSizeMB:  c.stats.TotalSizeMB,
		HitRate:      c.stats.HitRate,
		OldestEntry:  c.stats.OldestEntry,
		NewestEntry:  c.stats.NewestEntry,
		CleanupCount: c.stats.CleanupCount,
		LastCleanup:  c.stats.LastCleanup,
	}

	return stats, nil
}

// getCachePath 캐시 키에서 파일 경로 생성
func (c *cacheManagerImpl) getCachePath(key string) string {
	cacheConfig := c.config.GetCacheConfig()
	baseDir := filepath.Join(cacheConfig.BaseDir, "apt")

	// 보안을 위한 안전한 경로 조합
	safePath, err := security.SafeJoinPath(baseDir, key)
	if err != nil {
		// 에러 시 기본 경로 사용
		return filepath.Join(baseDir, "safe_"+key)
	}

	return safePath
}

// guessContentType 파일 경로에서 Content-Type 추측
func (c *cacheManagerImpl) guessContentType(filePath string) string {
	ext := filepath.Ext(filePath)
	switch ext {
	case ".deb":
		return "application/vnd.debian.binary-package"
	case ".gz":
		return "application/gzip"
	default:
		return "application/octet-stream"
	}
}

// updateStats 통계 업데이트 (동시성 안전)
func (c *cacheManagerImpl) updateStats(updateFunc func(*apt.CacheStats)) {
	c.statsMutex.Lock()
	defer c.statsMutex.Unlock()
	updateFunc(c.stats)
}

// startCleanupWorker 백그라운드 정리 작업 시작
func (c *cacheManagerImpl) startCleanupWorker() {
	cacheConfig := c.config.GetCacheConfig()
	if !cacheConfig.Enabled {
		return
	}

	interval := time.Duration(cacheConfig.CleanupHours) * time.Hour
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	c.logger.Info("Started cache cleanup worker", logging.F("interval_hours", cacheConfig.CleanupHours))

	for {
		select {
		case <-ticker.C:
			ctx := context.Background()
			if err := c.Cleanup(ctx); err != nil {
				c.logger.Error("Scheduled cache cleanup failed", logging.F("error", err))
			}
		}
	}
}
