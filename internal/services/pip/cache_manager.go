package pip

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"proxynd/internal/domain/pip"
	"proxynd/internal/security"
	"proxynd/logging"
)

// cacheManagerImpl PIP 캐시 관리 서비스 구현
type cacheManagerImpl struct {
	config      pip.ProxyConfig
	logger      logging.Logger
	cacheConfig pip.CacheConfig
	hitCount    int64
	missCount   int64
}

// NewCacheManager CacheManager 생성자
func NewCacheManager(config pip.ProxyConfig, logger logging.Logger) pip.CacheManager {
	return &cacheManagerImpl{
		config:      config,
		logger:      logger,
		cacheConfig: config.GetCacheConfig(),
	}
}

// Get 캐시에서 패키지 조회
func (c *cacheManagerImpl) Get(ctx context.Context, key string) (*pip.CacheEntry, error) {
	if !c.cacheConfig.Enabled {
		return nil, fmt.Errorf("cache is disabled")
	}

	filePath := c.getFilePath(key)

	// 파일 존재 확인
	fileInfo, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		c.missCount++
		return nil, fmt.Errorf("cache miss: %s", key)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to stat cache file: %w", err)
	}

	// TTL 확인
	if time.Since(fileInfo.ModTime()) > c.cacheConfig.TTL {
		c.missCount++
		c.logger.Debug("Cache entry expired",
			logging.F("key", key),
			logging.F("age", time.Since(fileInfo.ModTime())),
			logging.F("ttl", c.cacheConfig.TTL),
		)
		return nil, fmt.Errorf("cache entry expired: %s", key)
	}

	c.hitCount++

	// 캐시 엔트리 정보 생성
	entry := &pip.CacheEntry{
		Key:           key,
		Path:          filePath,
		Size:          fileInfo.Size(),
		CreatedAt:     fileInfo.ModTime(),
		LastAccessed:  time.Now(),
		TTL:           c.cacheConfig.TTL,
		IsSimpleAPI:   c.isSimpleAPIKey(key),
		IsPackageFile: c.isPackageFileKey(key),
	}

	// Content-Type 추정
	entry.ContentType = c.guessContentType(key)

	c.logger.Debug("Cache hit",
		logging.F("key", key),
		logging.F("size", entry.Size),
		logging.F("age", time.Since(entry.CreatedAt)),
	)

	return entry, nil
}

// Set 패키지를 캐시에 저장
func (c *cacheManagerImpl) Set(ctx context.Context, key string, data []byte, contentType string, metadata *pip.PackageMetadata) error {
	if !c.cacheConfig.Enabled {
		return nil // 캐시 비활성화 시 조용히 무시
	}

	filePath := c.getFilePath(key)

	// 디렉토리 생성
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	// 파일 쓰기
	if err := os.WriteFile(filePath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	// 체크섬 검증 (필요한 경우)
	if c.cacheConfig.ChecksumVerify && metadata != nil && metadata.ChecksumSHA256 != "" {
		if !c.VerifyChecksum(ctx, filePath, metadata.ChecksumSHA256) {
			// 체크섬 불일치 시 파일 삭제
			_ = os.Remove(filePath)
			return fmt.Errorf("checksum verification failed for %s", key)
		}
	}

	c.logger.Debug("Cache stored",
		logging.F("key", key),
		logging.F("size", len(data)),
		logging.F("contentType", contentType),
	)

	return nil
}

// Delete 캐시에서 패키지 삭제
func (c *cacheManagerImpl) Delete(ctx context.Context, key string) error {
	filePath := c.getFilePath(key)

	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete cache file: %w", err)
	}

	c.logger.Debug("Cache entry deleted", logging.F("key", key))
	return nil
}

// GenerateKey 캐시 키 생성
func (c *cacheManagerImpl) GenerateKey(packagePath string) string {
	// 안전한 파일명으로 변환
	safeKey := strings.ReplaceAll(packagePath, "/", "_")
	safeKey = strings.ReplaceAll(safeKey, ":", "_")
	safeKey = strings.ReplaceAll(safeKey, "?", "_")

	return fmt.Sprintf("pip_%s", safeKey)
}

// Cleanup 오래된 캐시 파일 정리
func (c *cacheManagerImpl) Cleanup(ctx context.Context) error {
	if !c.cacheConfig.Enabled {
		return nil
	}

	cacheDir := c.cacheConfig.BaseDir
	cleanupTime := time.Now().Add(-c.cacheConfig.TTL)
	cleanedCount := 0

	err := filepath.Walk(cacheDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 디렉토리는 스킵
		if info.IsDir() {
			return nil
		}

		// TTL 초과한 파일 삭제
		if info.ModTime().Before(cleanupTime) {
			if err := os.Remove(path); err != nil {
				c.logger.Warn("Failed to remove expired cache file",
					logging.F("path", path),
					logging.F("error", err),
				)
			} else {
				cleanedCount++
			}
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("cache cleanup failed: %w", err)
	}

	c.logger.Info("Cache cleanup completed",
		logging.F("cleanedFiles", cleanedCount),
		logging.F("cleanupTime", cleanupTime),
	)

	return nil
}

// GetStats 캐시 통계 조회
func (c *cacheManagerImpl) GetStats(ctx context.Context) (*pip.CacheStats, error) {
	if !c.cacheConfig.Enabled {
		return &pip.CacheStats{}, nil
	}

	stats := &pip.CacheStats{
		TotalEntries: 0,
		TotalSizeMB:  0,
	}

	// 캐시 디렉토리 순회
	cacheDir := c.cacheConfig.BaseDir
	err := filepath.Walk(cacheDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			stats.TotalEntries++
			stats.TotalSizeMB += info.Size() / (1024 * 1024)

			// 파일 타입별 분류
			if strings.Contains(path, "simple_") {
				stats.SimpleAPIEntries++
			} else if strings.HasSuffix(path, ".whl") {
				stats.WheelEntries++
				stats.PackageFileEntries++
			} else if strings.HasSuffix(path, ".tar.gz") || strings.HasSuffix(path, ".zip") {
				stats.SourceEntries++
				stats.PackageFileEntries++
			}
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to collect cache stats: %w", err)
	}

	// Hit rate 계산
	totalRequests := c.hitCount + c.missCount
	if totalRequests > 0 {
		stats.HitRate = float64(c.hitCount) / float64(totalRequests)
	}

	return stats, nil
}

// VerifyChecksum 캐시된 파일의 체크섬 검증
func (c *cacheManagerImpl) VerifyChecksum(ctx context.Context, filePath, expectedChecksum string) bool {
	data, err := os.ReadFile(filePath)
	if err != nil {
		c.logger.Warn("Failed to read file for checksum verification",
			logging.F("filePath", filePath),
			logging.F("error", err),
		)
		return false
	}

	// SHA256 해시 계산
	hash := sha256.Sum256(data)
	actualChecksum := hex.EncodeToString(hash[:])

	return actualChecksum == expectedChecksum
}

// getFilePath 캐시 키로부터 파일 경로 생성
func (c *cacheManagerImpl) getFilePath(key string) string {
	// 보안 검증
	safeDir := c.cacheConfig.BaseDir
	safePath, err := security.SafeJoinPath(safeDir, key)
	if err != nil {
		// fallback: 기본 경로 사용
		return filepath.Join(safeDir, key)
	}
	return safePath
}

// isSimpleAPIKey Simple API 캐시 키인지 확인
func (c *cacheManagerImpl) isSimpleAPIKey(key string) bool {
	return strings.Contains(key, "simple_")
}

// isPackageFileKey 패키지 파일 캐시 키인지 확인
func (c *cacheManagerImpl) isPackageFileKey(key string) bool {
	return strings.Contains(key, ".whl") || strings.Contains(key, ".tar.gz") || strings.Contains(key, ".zip")
}

// guessContentType 키에서 Content-Type 추정
func (c *cacheManagerImpl) guessContentType(key string) string {
	if c.isSimpleAPIKey(key) {
		return "text/html; charset=utf-8"
	}

	if strings.Contains(key, ".whl") || strings.Contains(key, ".zip") {
		return "application/zip"
	}

	if strings.Contains(key, ".tar.gz") {
		return "application/x-gzip"
	}

	if strings.Contains(key, ".tar.bz2") {
		return "application/x-bzip2"
	}

	return "application/octet-stream"
}
