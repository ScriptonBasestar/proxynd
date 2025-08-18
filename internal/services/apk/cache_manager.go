package apk

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"proxynd/internal/domain/apk"
	"proxynd/internal/logging"
)

const (
	mimeApplicationGzip        = "application/gzip"
	mimeApplicationOctetStream = "application/octet-stream"
)

type cacheManagerImpl struct {
	config     apk.ProxyConfig
	logger     logging.Logger
	storageDir string
	cache      map[string]*apk.CacheEntry
	mutex      sync.RWMutex
}

func NewCacheManager(config apk.ProxyConfig, logger logging.Logger, storageDir string) apk.CacheManager {
	return &cacheManagerImpl{
		config:     config,
		logger:     logger,
		storageDir: storageDir,
		cache:      make(map[string]*apk.CacheEntry),
	}
}

func (c *cacheManagerImpl) Get(ctx context.Context, key string) (*apk.CacheEntry, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	entry, exists := c.cache[key]
	if !exists {
		return nil, fmt.Errorf("cache entry not found: %s", key)
	}

	if time.Since(entry.CreatedAt) > entry.TTL {
		delete(c.cache, key)
		return nil, fmt.Errorf("cache entry expired: %s", key)
	}

	if _, err := os.Stat(entry.Path); os.IsNotExist(err) {
		delete(c.cache, key)
		return nil, fmt.Errorf("cache file not found: %s", entry.Path)
	}

	entry.LastAccessed = time.Now()
	return entry, nil
}

func (c *cacheManagerImpl) Set(ctx context.Context, key string, data []byte, ttl int64) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	cachePath := c.getCachePath(key)
	cacheDir := filepath.Dir(cachePath)

	if err := os.MkdirAll(cacheDir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	if err := os.WriteFile(cachePath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	entry := &apk.CacheEntry{
		Key:          key,
		Path:         cachePath,
		ContentType:  c.getContentTypeFromKey(key),
		Size:         int64(len(data)),
		CreatedAt:    time.Now(),
		LastAccessed: time.Now(),
		TTL:          time.Duration(ttl),
		IsApkFile:    c.isApkFileKey(key),
		IsIndex:      c.isIndexFileKey(key),
		IsSignature:  c.isSignatureFileKey(key),
	}

	c.cache[key] = entry
	return nil
}

func (c *cacheManagerImpl) Delete(ctx context.Context, key string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	entry, exists := c.cache[key]
	if !exists {
		return nil
	}

	if err := os.Remove(entry.Path); err != nil && !os.IsNotExist(err) {
		c.logger.Warn("APK 캐시 파일 삭제 실패", logging.F("path", entry.Path), logging.F("error", err.Error()))
	}

	delete(c.cache, key)
	return nil
}

func (c *cacheManagerImpl) Clear(ctx context.Context) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	for key, entry := range c.cache {
		if err := os.Remove(entry.Path); err != nil && !os.IsNotExist(err) {
			c.logger.Warn("APK 캐시 파일 삭제 실패", logging.F("path", entry.Path), logging.F("error", err.Error()))
		}
		delete(c.cache, key)
	}

	return nil
}

func (c *cacheManagerImpl) GetStats(ctx context.Context) (map[string]interface{}, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	totalSize := int64(0)
	for _, entry := range c.cache {
		totalSize += entry.Size
	}

	return map[string]interface{}{
		"entries":    len(c.cache),
		"total_size": totalSize,
		"max_size":   c.config.GetMaxCacheSize(),
	}, nil
}

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
			c.logger.Warn("APK 만료된 캐시 파일 삭제 실패", logging.F("path", entry.Path), logging.F("error", err.Error()))
		}
		delete(c.cache, key)
	}

	return nil
}

func (c *cacheManagerImpl) GetCacheKeysByType(ctx context.Context, fileType string) ([]string, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	var keys []string
	for key, entry := range c.cache {
		switch fileType {
		case "apk":
			if entry.IsApkFile {
				keys = append(keys, key)
			}
		case "index":
			if entry.IsIndex {
				keys = append(keys, key)
			}
		case "signature":
			if entry.IsSignature {
				keys = append(keys, key)
			}
		}
	}

	return keys, nil
}

func (c *cacheManagerImpl) getCachePath(key string) string {
	keyPath := key[4:] // "apk:" 제거
	return filepath.Join(c.storageDir, "cache", "apk", keyPath)
}

func (c *cacheManagerImpl) getContentTypeFromKey(key string) string {
	if c.isApkFileKey(key) {
		return "application/vnd.alpine.apk"
	} else if c.isIndexFileKey(key) {
		return mimeApplicationGzip
	}
	return mimeApplicationOctetStream
}

func (c *cacheManagerImpl) isApkFileKey(key string) bool {
	return strings.HasSuffix(key, ".apk")
}

func (c *cacheManagerImpl) isIndexFileKey(key string) bool {
	return strings.Contains(key, "APKINDEX") || strings.Contains(key, "PACKAGES")
}

func (c *cacheManagerImpl) isSignatureFileKey(key string) bool {
	return strings.HasSuffix(key, ".asc") || strings.HasSuffix(key, ".sig")
}
