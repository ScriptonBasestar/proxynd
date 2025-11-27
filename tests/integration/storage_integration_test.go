//go:build integration

package integration

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"proxynd/internal/adapters/storage/cas"
	"proxynd/internal/ports"
	"proxynd/internal/repositories/metadata/sqlite"
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

// TestEndToEnd_AnsibleCollectionStorage tests the complete workflow:
// 1. Store artifact blob in CAS
// 2. Store metadata in database
// 3. Verify artifact deduplication
// 4. Retrieve and verify data
func TestEndToEnd_AnsibleCollectionStorage(t *testing.T) {
	// Setup
	tmpDir := t.TempDir()
	casDir := filepath.Join(tmpDir, "cas")
	dbPath := filepath.Join(tmpDir, "metadata.db")

	// Create CAS and metadata repository
	storage, err := cas.NewLocalCAS(casDir, &mockLogger{})
	require.NoError(t, err)

	metaRepo, err := sqlite.NewRepository(dbPath, &mockLogger{})
	require.NoError(t, err)
	defer metaRepo.Close()

	ctx := context.Background()
	ansible := metaRepo.AnsibleMetadata()

	// Test data: Ansible collection tarball
	collectionData := []byte("fake-ansible-collection-tarball-content")

	// Step 1: Store artifact blob
	putResp, err := storage.PutBlob(ctx, &ports.PutBlobRequest{
		Content: bytes.NewReader(collectionData),
	})
	require.NoError(t, err)
	assert.False(t, putResp.AlreadyExists, "First upload should not exist")
	sha256 := putResp.SHA256

	// Step 2: Store artifact metadata
	artifact := &ports.ArtifactMetadata{
		SHA256:      sha256,
		Size:        int64(len(collectionData)),
		ContentType: "application/gzip",
		RefCount:    1,
		CreatedAt:   time.Now(),
		AccessedAt:  time.Now(),
	}
	err = metaRepo.StoreArtifact(ctx, artifact)
	require.NoError(t, err)

	// Step 3: Store collection version metadata
	versionReq := &ports.CreateVersionRequest{
		Namespace:      "community",
		Name:           "general",
		Version:        "6.0.0",
		ArtifactSHA256: sha256,
		License:        "GPL-3.0-or-later",
		Tags:           []string{"system", "cloud", "networking"},
		Authors:        []string{"Ansible Community"},
		Dependencies: map[string]string{
			"ansible.posix":    ">=1.4.0",
			"ansible.utils":    ">=2.0.0",
			"ansible.windows":  ">=1.0.0",
			"community.crypto": ">=2.0.0",
		},
		RepositoryURL:    "https://github.com/ansible-collections/community.general",
		DocumentationURL: "https://docs.ansible.com/ansible/latest/collections/community/general/",
		HomepageURL:      "https://github.com/ansible-collections/community.general",
		IssuesURL:        "https://github.com/ansible-collections/community.general/issues",
		RequiresAnsible:  ">=2.14.0",
		UploadedBy:       "admin",
	}

	version, err := ansible.CreateVersion(ctx, versionReq)
	require.NoError(t, err)
	assert.Equal(t, "6.0.0", version.Version)
	assert.Equal(t, sha256, version.ArtifactSHA256)

	// Step 4: Test deduplication - upload same artifact again
	putResp2, err := storage.PutBlob(ctx, &ports.PutBlobRequest{
		Content: bytes.NewReader(collectionData),
	})
	require.NoError(t, err)
	assert.True(t, putResp2.AlreadyExists, "Second upload should be deduplicated")
	assert.Equal(t, sha256, putResp2.SHA256, "SHA256 should match")

	// Step 5: Retrieve and verify collection version
	retrieved, err := ansible.GetVersion(ctx, "community", "general", "6.0.0")
	require.NoError(t, err)
	assert.Equal(t, version.ID, retrieved.ID)
	assert.Equal(t, version.Version, retrieved.Version)
	assert.Equal(t, version.ArtifactSHA256, retrieved.ArtifactSHA256)
	assert.Equal(t, len(versionReq.Tags), len(retrieved.Tags))
	assert.Equal(t, len(versionReq.Dependencies), len(retrieved.Dependencies))

	// Step 6: Retrieve blob content and verify
	reader, err := storage.GetBlob(ctx, sha256)
	require.NoError(t, err)
	defer reader.Close()

	retrievedData := make([]byte, len(collectionData))
	_, err = reader.Read(retrievedData)
	require.NoError(t, err)
	assert.Equal(t, collectionData, retrievedData)

	// Step 7: Check artifact metadata
	artifactMeta, err := metaRepo.GetArtifact(ctx, sha256)
	require.NoError(t, err)
	assert.Equal(t, sha256, artifactMeta.SHA256)
	assert.Equal(t, int64(len(collectionData)), artifactMeta.Size)
	assert.Equal(t, 1, artifactMeta.RefCount)

	// Step 8: Test reference counting
	err = metaRepo.IncrementRefCount(ctx, sha256)
	require.NoError(t, err)

	artifactMeta, err = metaRepo.GetArtifact(ctx, sha256)
	require.NoError(t, err)
	assert.Equal(t, 2, artifactMeta.RefCount)

	// Step 9: Delete version
	err = ansible.DeleteVersion(ctx, "community", "general", "6.0.0")
	require.NoError(t, err)

	// Step 10: Decrement ref count
	err = metaRepo.DecrementRefCount(ctx, sha256)
	require.NoError(t, err)

	artifactMeta, err = metaRepo.GetArtifact(ctx, sha256)
	require.NoError(t, err)
	assert.Equal(t, 1, artifactMeta.RefCount)
}

// TestEndToEnd_PythonPackageStorage tests the complete workflow for Python packages
func TestEndToEnd_PythonPackageStorage(t *testing.T) {
	// Setup
	tmpDir := t.TempDir()
	casDir := filepath.Join(tmpDir, "cas")
	dbPath := filepath.Join(tmpDir, "metadata.db")

	storage, err := cas.NewLocalCAS(casDir, &mockLogger{})
	require.NoError(t, err)

	metaRepo, err := sqlite.NewRepository(dbPath, &mockLogger{})
	require.NoError(t, err)
	defer metaRepo.Close()

	ctx := context.Background()
	python := metaRepo.PythonMetadata()

	// Test data: Python wheel file
	wheelData := []byte("fake-python-wheel-content")

	// Step 1: Store artifact blob
	putResp, err := storage.PutBlob(ctx, &ports.PutBlobRequest{
		Content: bytes.NewReader(wheelData),
	})
	require.NoError(t, err)
	sha256 := putResp.SHA256

	// Step 2: Store artifact metadata
	artifact := &ports.ArtifactMetadata{
		SHA256:      sha256,
		Size:        int64(len(wheelData)),
		ContentType: "application/octet-stream",
		RefCount:    1,
		CreatedAt:   time.Now(),
		AccessedAt:  time.Now(),
	}
	err = metaRepo.StoreArtifact(ctx, artifact)
	require.NoError(t, err)

	// Step 3: Store package version metadata
	versionReq := &ports.CreatePythonVersionRequest{
		PackageName:    "requests",
		Version:        "2.31.0",
		ArtifactSHA256: sha256,
		Filename:       "requests-2.31.0-py3-none-any.whl",
		PythonVersion:  "py3",
		RequiresPython: ">=3.7",
		PackageType:    "bdist_wheel",
		Summary:        "Python HTTP for Humans.",
		Keywords:       "http,requests,client",
		Classifiers: []string{
			"Development Status :: 5 - Production/Stable",
			"Intended Audience :: Developers",
			"License :: OSI Approved :: Apache Software License",
			"Programming Language :: Python :: 3",
		},
		Dependencies: []string{
			"charset-normalizer>=2,<4",
			"idna>=2.5,<4",
			"urllib3>=1.21.1,<3",
			"certifi>=2017.4.17",
		},
		UploadedBy: "admin",
	}

	version, err := python.CreateVersion(ctx, versionReq)
	require.NoError(t, err)
	assert.Equal(t, "2.31.0", version.Version)
	assert.Equal(t, sha256, version.ArtifactSHA256)

	// Step 4: Test normalized name lookup
	pkg, err := python.GetPackage(ctx, "requests")
	require.NoError(t, err)
	assert.Equal(t, "requests", pkg.Name)
	assert.Equal(t, "requests", pkg.NormalizedName)

	// Also works with different formatting
	pkg2, err := python.GetPackage(ctx, "Requests")
	require.NoError(t, err)
	assert.Equal(t, pkg.ID, pkg2.ID)

	// Step 5: List versions
	versionsResp, err := python.ListVersions(ctx, &ports.ListPythonVersionsRequest{
		PackageName: "requests",
		Page:        1,
		PageSize:    10,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, versionsResp.Total)
	assert.Len(t, versionsResp.Versions, 1)
	assert.Equal(t, "2.31.0", versionsResp.Versions[0].Version)

	// Step 6: Retrieve blob and verify
	reader, err := storage.GetBlob(ctx, sha256)
	require.NoError(t, err)
	defer reader.Close()

	// Step 7: Delete version
	err = python.DeleteVersion(ctx, "requests", "2.31.0")
	require.NoError(t, err)

	_, err = python.GetVersion(ctx, "requests", "2.31.0")
	assert.Error(t, err)
}

// TestCASandMetadataConsistency tests data consistency between CAS and metadata
func TestCASandMetadataConsistency(t *testing.T) {
	tmpDir := t.TempDir()
	casDir := filepath.Join(tmpDir, "cas")
	dbPath := filepath.Join(tmpDir, "metadata.db")

	storage, err := cas.NewLocalCAS(casDir, &mockLogger{})
	require.NoError(t, err)

	metaRepo, err := sqlite.NewRepository(dbPath, &mockLogger{})
	require.NoError(t, err)
	defer metaRepo.Close()

	ctx := context.Background()

	// Create multiple artifacts
	testData := []struct {
		content []byte
		refCnt  int
	}{
		{[]byte("artifact-1-content"), 1},
		{[]byte("artifact-2-content"), 3},
		{[]byte("artifact-3-content"), 5},
	}

	var sha256s []string

	// Store artifacts
	for _, td := range testData {
		putResp, err := storage.PutBlob(ctx, &ports.PutBlobRequest{
			Content: bytes.NewReader(td.content),
		})
		require.NoError(t, err)

		artifact := &ports.ArtifactMetadata{
			SHA256:     putResp.SHA256,
			Size:       int64(len(td.content)),
			RefCount:   td.refCnt,
			CreatedAt:  time.Now(),
			AccessedAt: time.Now(),
		}
		err = metaRepo.StoreArtifact(ctx, artifact)
		require.NoError(t, err)

		sha256s = append(sha256s, putResp.SHA256)
	}

	// Verify all artifacts exist in both CAS and metadata
	for i, sha256 := range sha256s {
		// Check CAS
		exists, err := storage.ExistsBlob(ctx, sha256)
		require.NoError(t, err)
		assert.True(t, exists, "Blob should exist in CAS")

		blobInfo, err := storage.GetBlobInfo(ctx, sha256)
		require.NoError(t, err)
		assert.Equal(t, int64(len(testData[i].content)), blobInfo.Size)

		// Check metadata
		artifactMeta, err := metaRepo.GetArtifact(ctx, sha256)
		require.NoError(t, err)
		assert.Equal(t, sha256, artifactMeta.SHA256)
		assert.Equal(t, testData[i].refCnt, artifactMeta.RefCount)
	}

	// Get storage stats
	stats, err := storage.GetStats(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(3), stats.TotalBlobs)
	assert.Greater(t, stats.TotalSize, int64(0))

	// Delete artifacts with refCount = 1
	for i, sha256 := range sha256s {
		if testData[i].refCnt == 1 {
			err := storage.DeleteBlob(ctx, sha256)
			require.NoError(t, err)

			exists, err := storage.ExistsBlob(ctx, sha256)
			require.NoError(t, err)
			assert.False(t, exists, "Blob should be deleted from CAS")
		}
	}
}

func TestCleanup(t *testing.T) {
	// Cleanup is automatic via t.TempDir()
	// This test just verifies no resource leaks
	tmpDir := t.TempDir()

	storage, err := cas.NewLocalCAS(tmpDir, &mockLogger{})
	require.NoError(t, err)

	ctx := context.Background()

	// Create and delete many blobs
	for i := 0; i < 100; i++ {
		data := []byte(string(rune(i)))
		putResp, err := storage.PutBlob(ctx, &ports.PutBlobRequest{
			Content: bytes.NewReader(data),
		})
		require.NoError(t, err)

		if i%2 == 0 {
			err = storage.DeleteBlob(ctx, putResp.SHA256)
			require.NoError(t, err)
		}
	}

	stats, err := storage.GetStats(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(50), stats.TotalBlobs, "Should have 50 blobs remaining")

	// Cleanup happens automatically when tmpDir is removed
}
