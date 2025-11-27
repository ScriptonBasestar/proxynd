package cas_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"proxynd/internal/adapters/storage/cas"
	"proxynd/internal/ports"
)

// mockLogger implements ports.Logger for testing
type mockLogger struct{}

func (m *mockLogger) Debug(ctx context.Context, msg string, fields ...ports.Field) {}
func (m *mockLogger) Info(ctx context.Context, msg string, fields ...ports.Field)  {}
func (m *mockLogger) Warn(ctx context.Context, msg string, fields ...ports.Field)  {}
func (m *mockLogger) Error(ctx context.Context, msg string, fields ...ports.Field) {}
func (m *mockLogger) Fatal(ctx context.Context, msg string, fields ...ports.Field) {}
func (m *mockLogger) With(fields ...ports.Field) ports.Logger                      { return m }
func (m *mockLogger) WithContext(ctx context.Context) ports.Logger                 { return m }
func (m *mockLogger) WithError(err error) ports.Logger                             { return m }
func (m *mockLogger) WithComponent(component string) ports.Logger                  { return m }
func (m *mockLogger) WithRequestID(requestID string) ports.Logger                  { return m }
func (m *mockLogger) SetLevel(level string) error                                  { return nil }
func (m *mockLogger) GetLevel() string                                             { return "info" }
func (m *mockLogger) IsDebugEnabled() bool                                         { return false }
func (m *mockLogger) Sync() error                                                  { return nil }

func TestLocalCAS_PutBlob(t *testing.T) {
	// Setup
	ctx := context.Background()
	tmpDir := t.TempDir()

	storage, err := cas.NewLocalCAS(tmpDir, &mockLogger{})
	require.NoError(t, err)

	// Test data
	content := []byte("test content for CAS storage")
	expectedSHA256 := calculateSHA256(content)

	t.Run("store new blob without SHA256", func(t *testing.T) {
		resp, err := storage.PutBlob(ctx, &ports.PutBlobRequest{
			Content: bytes.NewReader(content),
			Size:    int64(len(content)),
		})

		require.NoError(t, err)
		assert.Equal(t, expectedSHA256, resp.SHA256)
		assert.Equal(t, int64(len(content)), resp.Size)
		assert.False(t, resp.AlreadyExists)
	})

	t.Run("store same blob again (deduplication)", func(t *testing.T) {
		resp, err := storage.PutBlob(ctx, &ports.PutBlobRequest{
			Content: bytes.NewReader(content),
			Size:    int64(len(content)),
		})

		require.NoError(t, err)
		assert.Equal(t, expectedSHA256, resp.SHA256)
		assert.True(t, resp.AlreadyExists)
	})

	t.Run("store blob with pre-calculated SHA256", func(t *testing.T) {
		newContent := []byte("different content")
		newSHA256 := calculateSHA256(newContent)

		resp, err := storage.PutBlob(ctx, &ports.PutBlobRequest{
			SHA256:  newSHA256,
			Content: bytes.NewReader(newContent),
			Size:    int64(len(newContent)),
		})

		require.NoError(t, err)
		assert.Equal(t, newSHA256, resp.SHA256)
		assert.False(t, resp.AlreadyExists)
	})

	t.Run("verify directory structure", func(t *testing.T) {
		// Blob should be stored at: ab/cd/abcd....blob
		expectedPath := filepath.Join(
			tmpDir,
			expectedSHA256[0:2],
			expectedSHA256[2:4],
			expectedSHA256+".blob",
		)

		_, err := os.Stat(expectedPath)
		assert.NoError(t, err, "blob file should exist at expected path")
	})
}

func TestLocalCAS_GetBlob(t *testing.T) {
	// Setup
	ctx := context.Background()
	tmpDir := t.TempDir()

	storage, err := cas.NewLocalCAS(tmpDir, &mockLogger{})
	require.NoError(t, err)

	// Store a blob first
	content := []byte("test content for retrieval")
	storeResp, err := storage.PutBlob(ctx, &ports.PutBlobRequest{
		Content: bytes.NewReader(content),
	})
	require.NoError(t, err)

	t.Run("get existing blob", func(t *testing.T) {
		reader, err := storage.GetBlob(ctx, storeResp.SHA256)
		require.NoError(t, err)
		defer func() { _ = reader.Close() }()

		retrieved, err := io.ReadAll(reader)
		require.NoError(t, err)

		assert.Equal(t, content, retrieved)
	})

	t.Run("get non-existent blob", func(t *testing.T) {
		nonExistentSHA256 := "0000000000000000000000000000000000000000000000000000000000000000"
		_, err := storage.GetBlob(ctx, nonExistentSHA256)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "blob not found")
	})
}

func TestLocalCAS_ExistsBlob(t *testing.T) {
	// Setup
	ctx := context.Background()
	tmpDir := t.TempDir()

	storage, err := cas.NewLocalCAS(tmpDir, &mockLogger{})
	require.NoError(t, err)

	// Store a blob
	content := []byte("test content")
	storeResp, err := storage.PutBlob(ctx, &ports.PutBlobRequest{
		Content: bytes.NewReader(content),
	})
	require.NoError(t, err)

	t.Run("check existing blob", func(t *testing.T) {
		exists, err := storage.ExistsBlob(ctx, storeResp.SHA256)
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("check non-existent blob", func(t *testing.T) {
		nonExistentSHA256 := "0000000000000000000000000000000000000000000000000000000000000000"
		exists, err := storage.ExistsBlob(ctx, nonExistentSHA256)
		require.NoError(t, err)
		assert.False(t, exists)
	})
}

func TestLocalCAS_GetBlobInfo(t *testing.T) {
	// Setup
	ctx := context.Background()
	tmpDir := t.TempDir()

	storage, err := cas.NewLocalCAS(tmpDir, &mockLogger{})
	require.NoError(t, err)

	// Store a blob
	content := []byte("test content for info")
	storeResp, err := storage.PutBlob(ctx, &ports.PutBlobRequest{
		Content: bytes.NewReader(content),
	})
	require.NoError(t, err)

	t.Run("get blob info", func(t *testing.T) {
		info, err := storage.GetBlobInfo(ctx, storeResp.SHA256)
		require.NoError(t, err)

		assert.Equal(t, storeResp.SHA256, info.SHA256)
		assert.Equal(t, int64(len(content)), info.Size)
		assert.NotZero(t, info.CreatedAt)
	})
}

func TestLocalCAS_DeleteBlob(t *testing.T) {
	// Setup
	ctx := context.Background()
	tmpDir := t.TempDir()

	storage, err := cas.NewLocalCAS(tmpDir, &mockLogger{})
	require.NoError(t, err)

	// Store a blob
	content := []byte("test content to delete")
	storeResp, err := storage.PutBlob(ctx, &ports.PutBlobRequest{
		Content: bytes.NewReader(content),
	})
	require.NoError(t, err)

	t.Run("delete existing blob", func(t *testing.T) {
		err := storage.DeleteBlob(ctx, storeResp.SHA256)
		require.NoError(t, err)

		// Verify blob no longer exists
		exists, err := storage.ExistsBlob(ctx, storeResp.SHA256)
		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("delete non-existent blob", func(t *testing.T) {
		nonExistentSHA256 := "0000000000000000000000000000000000000000000000000000000000000000"
		err := storage.DeleteBlob(ctx, nonExistentSHA256)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "blob not found")
	})
}

func TestLocalCAS_GetStats(t *testing.T) {
	// Setup
	ctx := context.Background()
	tmpDir := t.TempDir()

	storage, err := cas.NewLocalCAS(tmpDir, &mockLogger{})
	require.NoError(t, err)

	// Store multiple blobs
	blobs := []string{
		"content 1",
		"content 2 with more data",
		"content 3",
	}

	var totalSize int64
	for _, content := range blobs {
		_, err := storage.PutBlob(ctx, &ports.PutBlobRequest{
			Content: bytes.NewReader([]byte(content)),
		})
		require.NoError(t, err)
		totalSize += int64(len(content))
	}

	t.Run("get storage stats", func(t *testing.T) {
		stats, err := storage.GetStats(ctx)
		require.NoError(t, err)

		assert.Equal(t, int64(len(blobs)), stats.TotalBlobs)
		assert.Equal(t, totalSize, stats.TotalSize)
		assert.Greater(t, stats.TotalSizeMB, float64(0))
	})
}

func TestLocalCAS_ListBlobs(t *testing.T) {
	// Setup
	ctx := context.Background()
	tmpDir := t.TempDir()

	storage, err := cas.NewLocalCAS(tmpDir, &mockLogger{})
	require.NoError(t, err)

	// Store multiple blobs
	numBlobs := 5
	for i := 0; i < numBlobs; i++ {
		content := []byte(fmt.Sprintf("content %d", i))
		_, err := storage.PutBlob(ctx, &ports.PutBlobRequest{
			Content: bytes.NewReader(content),
		})
		require.NoError(t, err)
	}

	t.Run("list all blobs", func(t *testing.T) {
		resp, err := storage.ListBlobs(ctx, &ports.ListBlobsRequest{
			Page:     1,
			PageSize: 10,
		})
		require.NoError(t, err)

		assert.Equal(t, numBlobs, resp.Total)
		assert.Len(t, resp.Blobs, numBlobs)
	})

	t.Run("list blobs with pagination", func(t *testing.T) {
		resp, err := storage.ListBlobs(ctx, &ports.ListBlobsRequest{
			Page:     1,
			PageSize: 2,
		})
		require.NoError(t, err)

		assert.Equal(t, numBlobs, resp.Total)
		assert.Len(t, resp.Blobs, 2)
		assert.Equal(t, 3, resp.TotalPages)
	})
}

// Helper function
func calculateSHA256(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}
