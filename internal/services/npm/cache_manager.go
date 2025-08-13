package npm

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"proxynd/internal/domain/npm"
	"proxynd/internal/security"
	"proxynd/logging"
)

const (
	mimeApplicationJSONCharsetUTF8 = "application/json; charset=utf-8"
	mimeApplicationXGzip           = "application/x-gzip"
)

// cacheManagerImpl 캐시 관리 서비스 구현
type cacheManagerImpl struct {
	config npm.ProxyConfig
	logger logging.Logger
}

// NewCacheManager CacheManager 생성자
func NewCacheManager(config npm.ProxyConfig, logger logging.Logger) npm.CacheManager {
	return &cacheManagerImpl{
		config: config,
		logger: logger,
	}
}

// Get 캐시에서 패키지 조회
func (c *cacheManagerImpl) Get(ctx context.Context, key string) (*npm.CacheEntry, error) {
	cacheConfig := c.config.GetCacheConfig()
	if !cacheConfig.Enabled {
		return nil, fmt.Errorf("cache disabled")
	}

	// 캐시 파일 경로 생성 (보안 검증)
	cachePath, err := c.getCachePath(key)
	if err != nil {
		return nil, fmt.Errorf("invalid cache key: %w", err)
	}

	// 파일 존재 확인
	stat, err := os.Stat(cachePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("cache miss")
		}
		return nil, fmt.Errorf("cache stat error: %w", err)
	}

	// TTL 확인
	if time.Since(stat.ModTime()) > cacheConfig.TTL {
		c.logger.Debug("Cache entry expired",
			logging.F("key", key),
			logging.F("path", cachePath),
			logging.F("age", time.Since(stat.ModTime())),
		)
		return nil, fmt.Errorf("cache expired")
	}

	// 메타데이터 여부 확인 (패키지 경로에서 추정)
	isMetadata := c.isMetadataPath(key)

	entry := &npm.CacheEntry{
		Key:          key,
		Path:         cachePath,
		ContentType:  c.getContentTypeFromPath(key),
		Size:         stat.Size(),
		CreatedAt:    stat.ModTime(),
		LastAccessed: time.Now(),
		TTL:          cacheConfig.TTL,
		IsMetadata:   isMetadata,
	}

	c.logger.Debug("Cache hit",
		logging.F("key", key),
		logging.F("path", cachePath),
		logging.F("size", stat.Size()),
	)

	return entry, nil
}

// Set 패키지를 캐시에 저장
func (c *cacheManagerImpl) Set(ctx context.Context, key string, data []byte, contentType string, isMetadata bool) error { //nolint:lll
	cacheConfig := c.config.GetCacheConfig()
	if !cacheConfig.Enabled {
		return nil // 캐시 비활성화 시 조용히 무시
	}

	// 캐시 파일 경로 생성 (보안 검증)
	cachePath, err := c.getCachePath(key)
	if err != nil {
		return fmt.Errorf("invalid cache key: %w", err)
	}

	// 디렉토리 생성
	dir := filepath.Dir(cachePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	// 원자적 파일 쓰기 (임시 파일 → 이동)
	tempPath := cachePath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	if err := os.Rename(tempPath, cachePath); err != nil {
		_ = os.Remove(tempPath) // 정리
		return fmt.Errorf("failed to commit cache file: %w", err)
	}

	c.logger.Debug("Cached package",
		logging.F("key", key),
		logging.F("path", cachePath),
		logging.F("size", len(data)),
		logging.F("contentType", contentType),
		logging.F("isMetadata", isMetadata),
	)

	return nil
}

// Delete 캐시에서 패키지 삭제
func (c *cacheManagerImpl) Delete(ctx context.Context, key string) error {
	cachePath, err := c.getCachePath(key)
	if err != nil {
		return fmt.Errorf("invalid cache key: %w", err)
	}

	if err := os.Remove(cachePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete cache file: %w", err)
	}

	c.logger.Debug("Deleted cache entry",
		logging.F("key", key),
		logging.F("path", cachePath),
	)

	return nil
}

// GenerateKey 캐시 키 생성
func (c *cacheManagerImpl) GenerateKey(packagePath string) string {
	// 패키지 경로를 안전한 파일명으로 변환
	key := strings.ReplaceAll(packagePath, "/", "_")
	key = strings.ReplaceAll(key, "@", "at_")
	key = strings.ReplaceAll(key, ":", "_")

	return key
}

// Cleanup 오래된 캐시 파일 정리
func (c *cacheManagerImpl) Cleanup(ctx context.Context) error {
	cacheConfig := c.config.GetCacheConfig()
	if !cacheConfig.Enabled {
		return nil
	}

	cacheDir := filepath.Join(cacheConfig.BaseDir, c.config.GetPath())

	c.logger.Info("Starting cache cleanup",
		logging.F("cacheDir", cacheDir),
		logging.F("ttl", cacheConfig.TTL),
	)

	cleanupCount := 0
	err := filepath.Walk(cacheDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // 개별 파일 에러는 무시
		}

		if info.IsDir() {
			return nil
		}

		// TTL 만료된 파일 삭제
		if time.Since(info.ModTime()) > cacheConfig.TTL {
			if err := os.Remove(path); err == nil {
				cleanupCount++
			}
		}

		return nil
	})

	c.logger.Info("Cache cleanup completed",
		logging.F("cleanedFiles", cleanupCount),
		logging.F("error", err),
	)

	return err
}

// GetStats 캐시 통계 조회
func (c *cacheManagerImpl) GetStats(ctx context.Context) (*npm.CacheStats, error) {
	cacheConfig := c.config.GetCacheConfig()
	cacheDir := filepath.Join(cacheConfig.BaseDir, c.config.GetPath())

	stats := &npm.CacheStats{}
	var totalSize int64
	var oldestTime, newestTime time.Time

	err := filepath.Walk(cacheDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		stats.TotalEntries++
		totalSize += info.Size()

		// 메타데이터 vs 패키지 파일 구분
		if c.isMetadataPath(filepath.Base(path)) {
			stats.MetadataEntries++
		} else {
			stats.TarballEntries++
		}

		// 가장 오래된/최신 파일 추적
		if oldestTime.IsZero() || info.ModTime().Before(oldestTime) {
			oldestTime = info.ModTime()
			stats.OldestEntry = path
		}
		if newestTime.IsZero() || info.ModTime().After(newestTime) {
			newestTime = info.ModTime()
			stats.NewestEntry = path
		}

		return nil
	})

	stats.TotalSizeMB = totalSize / (1024 * 1024)
	stats.LastCleanup = time.Now().Format(time.RFC3339)

	return stats, err
}

// getCachePath 캐시 키에서 안전한 파일 경로 생성
func (c *cacheManagerImpl) getCachePath(key string) (string, error) {
	cacheConfig := c.config.GetCacheConfig()
	baseDir := filepath.Join(cacheConfig.BaseDir, c.config.GetPath())

	// 보안 검증을 통한 경로 생성
	return security.SafeJoinPath(baseDir, key)
}

// isMetadataPath 메타데이터 파일인지 경로에서 추정
func (c *cacheManagerImpl) isMetadataPath(path string) bool {
	// .tgz가 없고 특수 문자가 포함되지 않은 경우 메타데이터로 추정
	return !strings.Contains(path, ".tgz") && !strings.Contains(path, "/-/")
}

// getContentTypeFromPath 파일 경로에서 Content-Type 추정
func (c *cacheManagerImpl) getContentTypeFromPath(path string) string {
	if c.isMetadataPath(path) {
		return mimeApplicationJSONCharsetUTF8
	} else if strings.Contains(path, ".tgz") {
		return mimeApplicationXGzip
	}
	return "application/octet-stream"
}
