package ports

import (
	"context"
	"io"
	"time"
)

// ContentAddressableStorage provides content-addressable blob storage
// All artifacts are stored by their SHA256 hash
type ContentAddressableStorage interface {
	// PutBlob stores a blob with given SHA256 hash
	// If SHA256 is empty, it will be calculated from content
	PutBlob(ctx context.Context, req *PutBlobRequest) (*PutBlobResponse, error)

	// GetBlob retrieves a blob by its SHA256 hash
	GetBlob(ctx context.Context, sha256 string) (io.ReadCloser, error)

	// GetBlobInfo retrieves blob metadata without content
	GetBlobInfo(ctx context.Context, sha256 string) (*BlobInfo, error)

	// ExistsBlob checks if a blob exists
	ExistsBlob(ctx context.Context, sha256 string) (bool, error)

	// DeleteBlob removes a blob (with reference counting)
	DeleteBlob(ctx context.Context, sha256 string) error

	// ListBlobs lists all blobs (paginated)
	ListBlobs(ctx context.Context, req *ListBlobsRequest) (*ListBlobsResponse, error)

	// GetStats returns storage statistics
	GetStats(ctx context.Context) (*StorageStats, error)
}

// PutBlobRequest represents a blob storage request
type PutBlobRequest struct {
	SHA256      string    `json:"sha256"` // Optional: will be calculated if empty
	Content     io.Reader `json:"-"`
	Size        int64     `json:"size"`         // Optional: for validation
	ContentType string    `json:"content_type"` // Optional: metadata
}

// PutBlobResponse represents a blob storage response
type PutBlobResponse struct {
	SHA256        string    `json:"sha256"`
	Size          int64     `json:"size"`
	StoredAt      time.Time `json:"stored_at"`
	AlreadyExists bool      `json:"already_exists"` // Deduplication
}

// BlobInfo represents blob metadata
type BlobInfo struct {
	SHA256      string    `json:"sha256"`
	Size        int64     `json:"size"`
	ContentType string    `json:"content_type"`
	RefCount    int       `json:"ref_count"` // Reference counting for GC
	CreatedAt   time.Time `json:"created_at"`
	AccessedAt  time.Time `json:"accessed_at"`
}

// ListBlobsRequest represents blob listing request
type ListBlobsRequest struct {
	Prefix   string `json:"prefix"` // SHA256 prefix filter
	Page     int    `json:"page"`
	PageSize int    `json:"page_size"`
}

// ListBlobsResponse represents blob listing response
type ListBlobsResponse struct {
	Blobs      []*BlobInfo `json:"blobs"`
	Total      int         `json:"total"`
	Page       int         `json:"page"`
	TotalPages int         `json:"total_pages"`
}

// StorageStats represents storage statistics
type StorageStats struct {
	TotalBlobs  int64     `json:"total_blobs"`
	TotalSize   int64     `json:"total_size"`
	TotalSizeMB float64   `json:"total_size_mb"`
	LastUpdated time.Time `json:"last_updated"`
}
