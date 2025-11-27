package cas

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"proxynd/internal/ports"
)

// LocalCAS implements ContentAddressableStorage using local filesystem
type LocalCAS struct {
	basePath string
	logger   ports.Logger
}

// NewLocalCAS creates a new local CAS instance
func NewLocalCAS(basePath string, logger ports.Logger) (*LocalCAS, error) {
	// Create base directory
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create CAS directory: %w", err)
	}

	return &LocalCAS{
		basePath: basePath,
		logger:   logger,
	}, nil
}

// PutBlob stores a blob
func (c *LocalCAS) PutBlob(ctx context.Context, req *ports.PutBlobRequest) (*ports.PutBlobResponse, error) {
	// 1. Calculate SHA256 if not provided
	var calculatedSHA256 string
	var tempFile *os.File
	var err error

	if req.SHA256 == "" {
		// Create temp file for calculating hash
		tempFile, err = os.CreateTemp("", "cas-upload-*")
		if err != nil {
			return nil, fmt.Errorf("failed to create temp file: %w", err)
		}
		defer func() { _ = os.Remove(tempFile.Name()) }()
		defer func() { _ = tempFile.Close() }()

		// Calculate SHA256 while copying to temp file
		hash := sha256.New()
		multiWriter := io.MultiWriter(tempFile, hash)
		_, err = io.Copy(multiWriter, req.Content)
		if err != nil {
			return nil, fmt.Errorf("failed to calculate SHA256: %w", err)
		}
		calculatedSHA256 = hex.EncodeToString(hash.Sum(nil))

		// Rewind temp file for later use
		if _, err := tempFile.Seek(0, 0); err != nil {
			return nil, fmt.Errorf("failed to rewind temp file: %w", err)
		}
	} else {
		calculatedSHA256 = req.SHA256
	}

	// 2. Check if blob already exists (deduplication)
	blobPath := c.getBlobPath(calculatedSHA256)
	if _, err := os.Stat(blobPath); err == nil {
		// Blob already exists (deduplication)
		info, _ := os.Stat(blobPath)
		return &ports.PutBlobResponse{
			SHA256:        calculatedSHA256,
			Size:          info.Size(),
			StoredAt:      info.ModTime(),
			AlreadyExists: true,
		}, nil
	}

	// 3. Create directory structure
	dir := filepath.Dir(blobPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	// 4. Write to temporary file first (atomic operation)
	tmpPath := blobPath + ".tmp"
	tmpOutput, err := os.Create(tmpPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer func() { _ = os.Remove(tmpPath) }() // Cleanup on error

	// 5. Copy content
	var written int64
	if tempFile != nil {
		// Use already calculated temp file
		written, err = io.Copy(tmpOutput, tempFile)
	} else {
		// Copy from original reader
		written, err = io.Copy(tmpOutput, req.Content)
	}

	if err != nil {
		_ = tmpOutput.Close()
		return nil, fmt.Errorf("failed to write blob: %w", err)
	}
	_ = tmpOutput.Close()

	// 6. Verify size if provided
	if req.Size > 0 && written != req.Size {
		return nil, fmt.Errorf("size mismatch: expected %d, got %d", req.Size, written)
	}

	// 7. Atomic rename
	if err := os.Rename(tmpPath, blobPath); err != nil {
		return nil, fmt.Errorf("failed to rename temp file: %w", err)
	}

	// Blob stored successfully

	return &ports.PutBlobResponse{
		SHA256:        calculatedSHA256,
		Size:          written,
		StoredAt:      time.Now(),
		AlreadyExists: false,
	}, nil
}

// GetBlob retrieves a blob
func (c *LocalCAS) GetBlob(ctx context.Context, sha256 string) (io.ReadCloser, error) {
	blobPath := c.getBlobPath(sha256)

	file, err := os.Open(blobPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("blob not found: %s", sha256)
		}
		return nil, fmt.Errorf("failed to open blob: %w", err)
	}

	return file, nil
}

// GetBlobInfo retrieves blob metadata
func (c *LocalCAS) GetBlobInfo(ctx context.Context, sha256 string) (*ports.BlobInfo, error) {
	blobPath := c.getBlobPath(sha256)

	info, err := os.Stat(blobPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("blob not found: %s", sha256)
		}
		return nil, err
	}

	return &ports.BlobInfo{
		SHA256:     sha256,
		Size:       info.Size(),
		CreatedAt:  info.ModTime(),
		AccessedAt: info.ModTime(), // TODO: Track actual access time
		RefCount:   1,              // TODO: Implement reference counting
	}, nil
}

// ExistsBlob checks if a blob exists
func (c *LocalCAS) ExistsBlob(ctx context.Context, sha256 string) (bool, error) {
	blobPath := c.getBlobPath(sha256)
	_, err := os.Stat(blobPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// DeleteBlob removes a blob
func (c *LocalCAS) DeleteBlob(ctx context.Context, sha256 string) error {
	// TODO: Implement reference counting
	// Only delete if RefCount == 0

	blobPath := c.getBlobPath(sha256)

	if err := os.Remove(blobPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("blob not found: %s", sha256)
		}
		return fmt.Errorf("failed to delete blob: %w", err)
	}

	// Blob deleted successfully
	return nil
}

// ListBlobs lists all blobs (paginated)
func (c *LocalCAS) ListBlobs(ctx context.Context, req *ports.ListBlobsRequest) (*ports.ListBlobsResponse, error) {
	var blobs []*ports.BlobInfo

	err := filepath.Walk(c.basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Ext(path) == ".blob" {
			sha256 := filepath.Base(path)
			sha256 = sha256[:len(sha256)-5] // Remove ".blob"

			if req.Prefix == "" || strings.HasPrefix(sha256, req.Prefix) {
				blobs = append(blobs, &ports.BlobInfo{
					SHA256:    sha256,
					Size:      info.Size(),
					CreatedAt: info.ModTime(),
				})
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	// Pagination
	total := len(blobs)
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 100
	}

	page := req.Page
	if page < 1 {
		page = 1
	}

	start := (page - 1) * pageSize
	end := start + pageSize

	if start >= total {
		return &ports.ListBlobsResponse{
			Blobs:      []*ports.BlobInfo{},
			Total:      total,
			Page:       page,
			TotalPages: (total + pageSize - 1) / pageSize,
		}, nil
	}

	if end > total {
		end = total
	}

	return &ports.ListBlobsResponse{
		Blobs:      blobs[start:end],
		Total:      total,
		Page:       page,
		TotalPages: (total + pageSize - 1) / pageSize,
	}, nil
}

// GetStats returns storage statistics
func (c *LocalCAS) GetStats(ctx context.Context) (*ports.StorageStats, error) {
	var totalBlobs int64
	var totalSize int64

	// Walk through all blobs
	err := filepath.Walk(c.basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Ext(path) == ".blob" {
			totalBlobs++
			totalSize += info.Size()
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return &ports.StorageStats{
		TotalBlobs:  totalBlobs,
		TotalSize:   totalSize,
		TotalSizeMB: float64(totalSize) / 1024 / 1024,
		LastUpdated: time.Now(),
	}, nil
}

// getBlobPath generates filesystem path for a SHA256 hash
func (c *LocalCAS) getBlobPath(sha256 string) string {
	// Example: sha256 = "abcd1234..."
	// Result: /var/lib/proxynd/artifacts/ab/cd/abcd1234....blob
	return filepath.Join(
		c.basePath,
		sha256[0:2],
		sha256[2:4],
		sha256+".blob",
	)
}
