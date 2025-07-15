package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"proxynd/logging"
)

// FileRepository implements cache storage using the file system
type FileRepository struct {
	basePath string
	logger   logging.Logger
	mu       sync.RWMutex
	metadata map[string]*CacheItem
	stats    *CacheStats
	maxSize  int64 // Maximum cache size in bytes
	maxAge   time.Duration
}

// NewFileRepository creates a new file-based cache repository
func NewFileRepository(basePath string, maxSize int64, maxAge time.Duration) (*FileRepository, error) {
	// Ensure base path exists
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	repo := &FileRepository{
		basePath: basePath,
		logger:   logging.GetLogger(),
		metadata: make(map[string]*CacheItem),
		stats: &CacheStats{
			HitCount:  0,
			MissCount: 0,
		},
		maxSize: maxSize,
		maxAge:  maxAge,
	}

	// Load existing metadata
	if err := repo.loadMetadata(); err != nil {
		repo.logger.Warn("Failed to load cache metadata", logging.F("error", err))
	}

	// Start cleanup goroutine
	go repo.cleanupRoutine()

	return repo, nil
}

// Get retrieves an item from the cache
func (r *FileRepository) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Check if item exists
	meta, exists := r.metadata[key]
	if !exists {
		r.stats.MissCount++
		return nil, fmt.Errorf("cache miss: key not found")
	}

	// Check if item has expired
	if r.isExpired(meta) {
		r.stats.MissCount++
		return nil, fmt.Errorf("cache miss: item expired")
	}

	// Build file path
	filePath := r.getFilePath(key)

	// Open file
	file, err := os.Open(filePath)
	if err != nil {
		r.stats.MissCount++
		return nil, fmt.Errorf("failed to open cached file: %w", err)
	}

	// Update access time
	meta.AccessedAt = time.Now()
	r.stats.HitCount++

	r.logger.Debug("Cache hit", logging.F("key", key))

	return file, nil
}

// Put stores an item in the cache
func (r *FileRepository) Put(ctx context.Context, key string, content io.Reader, ttl time.Duration) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Build file path
	filePath := r.getFilePath(key)

	// Ensure directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Create temporary file
	tmpFile, err := os.CreateTemp(dir, ".tmp-")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	// Copy content to temp file
	size, err := io.Copy(tmpFile, content)
	if err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("failed to write cache content: %w", err)
	}
	tmpFile.Close()

	// Move temp file to final location
	if err := os.Rename(tmpPath, filePath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to move cache file: %w", err)
	}

	// Update metadata
	now := time.Now()
	r.metadata[key] = &CacheItem{
		Key:        key,
		Size:       size,
		ModifiedAt: now,
		AccessedAt: now,
		TTL:        ttl,
	}

	// Save metadata
	r.saveMetadata()

	r.logger.Debug("Cached item", logging.F("key", key), logging.F("size", size))

	return nil
}

// Exists checks if a key exists in the cache
func (r *FileRepository) Exists(ctx context.Context, key string) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	meta, exists := r.metadata[key]
	if !exists {
		return false, nil
	}

	// Check if expired
	if r.isExpired(meta) {
		return false, nil
	}

	// Check if file actually exists
	filePath := r.getFilePath(key)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return false, nil
	}

	return true, nil
}

// Delete removes an item from the cache
func (r *FileRepository) Delete(ctx context.Context, key string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Remove file
	filePath := r.getFilePath(key)
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove cache file: %w", err)
	}

	// Remove metadata
	delete(r.metadata, key)

	// Save metadata
	r.saveMetadata()

	r.logger.Debug("Deleted cache item", logging.F("key", key))

	return nil
}

// List returns all cache keys matching a pattern
func (r *FileRepository) List(ctx context.Context, pattern string) ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var keys []string
	for key, meta := range r.metadata {
		// Skip expired items
		if r.isExpired(meta) {
			continue
		}

		// Match pattern
		if matched, _ := filepath.Match(pattern, key); matched {
			keys = append(keys, key)
		}
	}

	return keys, nil
}

// Size returns the size of a cached item in bytes
func (r *FileRepository) Size(ctx context.Context, key string) (int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	meta, exists := r.metadata[key]
	if !exists {
		return 0, fmt.Errorf("key not found")
	}

	return meta.Size, nil
}

// Clear removes all items from the cache
func (r *FileRepository) Clear(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Remove all files
	err := filepath.Walk(r.basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and metadata file
		if info.IsDir() || strings.HasSuffix(path, "metadata.json") {
			return nil
		}

		return os.Remove(path)
	})

	if err != nil {
		return fmt.Errorf("failed to clear cache: %w", err)
	}

	// Clear metadata
	r.metadata = make(map[string]*CacheItem)
	r.saveMetadata()

	r.logger.Info("Cache cleared")

	return nil
}

// Stats returns cache statistics
func (r *FileRepository) Stats(ctx context.Context) (*CacheStats, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	stats := &CacheStats{
		TotalItems:   int64(len(r.metadata)),
		HitCount:     r.stats.HitCount,
		MissCount:    r.stats.MissCount,
		EvictedCount: r.stats.EvictedCount,
	}

	var totalSize int64
	var oldest, newest time.Time

	for _, meta := range r.metadata {
		totalSize += meta.Size

		if oldest.IsZero() || meta.ModifiedAt.Before(oldest) {
			oldest = meta.ModifiedAt
		}

		if newest.IsZero() || meta.ModifiedAt.After(newest) {
			newest = meta.ModifiedAt
		}
	}

	stats.TotalSize = totalSize
	stats.OldestItem = oldest
	stats.NewestItem = newest

	return stats, nil
}

// Helper methods

func (r *FileRepository) getFilePath(key string) string {
	// Sanitize key to create safe file path
	safePath := strings.ReplaceAll(key, "..", "")
	return filepath.Join(r.basePath, safePath)
}

func (r *FileRepository) isExpired(item *CacheItem) bool {
	if item.TTL == 0 {
		// No TTL means it never expires
		return false
	}

	expiryTime := item.ModifiedAt.Add(item.TTL)
	return time.Now().After(expiryTime)
}

func (r *FileRepository) loadMetadata() error {
	metaPath := filepath.Join(r.basePath, "metadata.json")

	data, err := os.ReadFile(metaPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No metadata file yet
		}
		return err
	}

	return json.Unmarshal(data, &r.metadata)
}

func (r *FileRepository) saveMetadata() error {
	metaPath := filepath.Join(r.basePath, "metadata.json")

	data, err := json.MarshalIndent(r.metadata, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(metaPath, data, 0644)
}

func (r *FileRepository) cleanupRoutine() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		r.cleanup()
	}
}

func (r *FileRepository) cleanup() {
	r.mu.Lock()
	defer r.mu.Unlock()

	var keysToDelete []string

	// Find expired items
	for key, meta := range r.metadata {
		if r.isExpired(meta) {
			keysToDelete = append(keysToDelete, key)
		}
	}

	// Delete expired items
	for _, key := range keysToDelete {
		filePath := r.getFilePath(key)
		os.Remove(filePath)
		delete(r.metadata, key)
		r.stats.EvictedCount++
	}

	if len(keysToDelete) > 0 {
		r.saveMetadata()
		r.logger.Info("Cleaned up expired cache items", logging.F("count", len(keysToDelete)))
	}
}
