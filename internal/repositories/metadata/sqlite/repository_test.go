//go:build integration

package sqlite_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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

func setupTestDB(t *testing.T) (*sqlite.Repository, func()) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	repo, err := sqlite.NewRepository(dbPath, &mockLogger{})
	require.NoError(t, err)

	cleanup := func() {
		repo.Close()
		os.RemoveAll(tmpDir)
	}

	return repo, cleanup
}

func TestRepository_ArtifactOperations(t *testing.T) {
	repo, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("store and retrieve artifact", func(t *testing.T) {
		artifact := &ports.ArtifactMetadata{
			SHA256:      "abcd1234567890abcd1234567890abcd1234567890abcd1234567890abcd1234",
			Size:        1024,
			ContentType: "application/gzip",
			RefCount:    1,
			CreatedAt:   time.Now(),
			AccessedAt:  time.Now(),
		}

		err := repo.StoreArtifact(ctx, artifact)
		require.NoError(t, err)

		retrieved, err := repo.GetArtifact(ctx, artifact.SHA256)
		require.NoError(t, err)

		assert.Equal(t, artifact.SHA256, retrieved.SHA256)
		assert.Equal(t, artifact.Size, retrieved.Size)
		assert.Equal(t, artifact.ContentType, retrieved.ContentType)
		assert.Equal(t, artifact.RefCount, retrieved.RefCount)
	})

	t.Run("increment ref count", func(t *testing.T) {
		sha256 := "1111222233334444555566667777888899990000aaaabbbbccccddddeeeeffff"
		artifact := &ports.ArtifactMetadata{
			SHA256:     sha256,
			Size:       2048,
			RefCount:   1,
			CreatedAt:  time.Now(),
			AccessedAt: time.Now(),
		}

		err := repo.StoreArtifact(ctx, artifact)
		require.NoError(t, err)

		err = repo.IncrementRefCount(ctx, sha256)
		require.NoError(t, err)

		retrieved, err := repo.GetArtifact(ctx, sha256)
		require.NoError(t, err)
		assert.Equal(t, 2, retrieved.RefCount)
	})

	t.Run("decrement ref count", func(t *testing.T) {
		sha256 := "9999888877776666555544443333222211110000ffffeeeedddcccbbbaaa999"
		artifact := &ports.ArtifactMetadata{
			SHA256:     sha256,
			Size:       4096,
			RefCount:   3,
			CreatedAt:  time.Now(),
			AccessedAt: time.Now(),
		}

		err := repo.StoreArtifact(ctx, artifact)
		require.NoError(t, err)

		err = repo.DecrementRefCount(ctx, sha256)
		require.NoError(t, err)

		retrieved, err := repo.GetArtifact(ctx, sha256)
		require.NoError(t, err)
		assert.Equal(t, 2, retrieved.RefCount)
	})

	t.Run("get non-existent artifact", func(t *testing.T) {
		_, err := repo.GetArtifact(ctx, "nonexistent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "artifact not found")
	})
}

func TestRepository_AnsibleOperations(t *testing.T) {
	repo, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	ansible := repo.AnsibleMetadata()

	t.Run("create and get namespace", func(t *testing.T) {
		req := &ports.CreateNamespaceRequest{
			Name:        "community",
			Description: "Community collections",
			Company:     "Ansible Community",
		}

		ns, err := ansible.CreateNamespace(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, req.Name, ns.Name)
		assert.Equal(t, req.Description, ns.Description)

		retrieved, err := ansible.GetNamespace(ctx, "community")
		require.NoError(t, err)
		assert.Equal(t, ns.ID, retrieved.ID)
		assert.Equal(t, ns.Name, retrieved.Name)
	})

	t.Run("create and get collection", func(t *testing.T) {
		req := &ports.CreateCollectionRequest{
			Namespace:   "ansible",
			Name:        "posix",
			Description: "POSIX utilities collection",
		}

		coll, err := ansible.CreateCollection(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, req.Namespace, coll.Namespace)
		assert.Equal(t, req.Name, coll.Name)

		retrieved, err := ansible.GetCollection(ctx, "ansible", "posix")
		require.NoError(t, err)
		assert.Equal(t, coll.ID, retrieved.ID)
		assert.Equal(t, coll.Name, retrieved.Name)
	})

	t.Run("create and get version", func(t *testing.T) {
		// First store artifact
		artifact := &ports.ArtifactMetadata{
			SHA256:     "aabbccdd11223344556677889900aabbccdd11223344556677889900aabbccdd",
			Size:       10240,
			RefCount:   1,
			CreatedAt:  time.Now(),
			AccessedAt: time.Now(),
		}
		err := repo.StoreArtifact(ctx, artifact)
		require.NoError(t, err)

		req := &ports.CreateVersionRequest{
			Namespace:      "community",
			Name:           "general",
			Version:        "1.2.3",
			ArtifactSHA256: artifact.SHA256,
			License:        "GPL-3.0-or-later",
			Tags:           []string{"system", "utilities"},
			Authors:        []string{"Ansible Community"},
			Dependencies: map[string]string{
				"ansible.posix": ">=1.0.0",
			},
			RequiresAnsible: ">=2.9",
			UploadedBy:      "testuser",
		}

		ver, err := ansible.CreateVersion(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, req.Version, ver.Version)
		assert.Equal(t, req.ArtifactSHA256, ver.ArtifactSHA256)

		retrieved, err := ansible.GetVersion(ctx, "community", "general", "1.2.3")
		require.NoError(t, err)
		assert.Equal(t, ver.ID, retrieved.ID)
		assert.Equal(t, ver.Version, retrieved.Version)
		assert.Equal(t, len(req.Tags), len(retrieved.Tags))
		assert.Equal(t, len(req.Dependencies), len(retrieved.Dependencies))
	})

	t.Run("list versions", func(t *testing.T) {
		resp, err := ansible.ListVersions(ctx, &ports.ListVersionsRequest{
			Namespace: "community",
			Name:      "general",
			Page:      1,
			PageSize:  10,
		})
		require.NoError(t, err)
		assert.Greater(t, len(resp.Versions), 0)
		assert.Greater(t, resp.Total, 0)
	})

	t.Run("delete version", func(t *testing.T) {
		err := ansible.DeleteVersion(ctx, "community", "general", "1.2.3")
		require.NoError(t, err)

		_, err = ansible.GetVersion(ctx, "community", "general", "1.2.3")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "version not found")
	})
}

func TestRepository_PythonOperations(t *testing.T) {
	repo, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	python := repo.PythonMetadata()

	t.Run("create and get package", func(t *testing.T) {
		req := &ports.CreatePythonPackageRequest{
			Name:        "requests",
			Description: "Python HTTP library",
			HomePage:    "https://requests.readthedocs.io",
			Author:      "Kenneth Reitz",
			License:     "Apache-2.0",
		}

		pkg, err := python.CreatePackage(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, req.Name, pkg.Name)
		assert.Equal(t, "requests", pkg.NormalizedName)

		retrieved, err := python.GetPackage(ctx, "requests")
		require.NoError(t, err)
		assert.Equal(t, pkg.ID, retrieved.ID)
		assert.Equal(t, pkg.Name, retrieved.Name)
	})

	t.Run("normalized name lookup", func(t *testing.T) {
		// Create with underscores
		req := &ports.CreatePythonPackageRequest{
			Name: "django_rest_framework",
		}
		pkg, err := python.CreatePackage(ctx, req)
		require.NoError(t, err)

		// Lookup with hyphens should work
		retrieved, err := python.GetPackage(ctx, "django-rest-framework")
		require.NoError(t, err)
		assert.Equal(t, pkg.ID, retrieved.ID)
	})

	t.Run("create and get version", func(t *testing.T) {
		// First store artifact
		artifact := &ports.ArtifactMetadata{
			SHA256:     "ddccbbaa99887766554433221100ffeeddccbbaa99887766554433221100ffee",
			Size:       51200,
			RefCount:   1,
			CreatedAt:  time.Now(),
			AccessedAt: time.Now(),
		}
		err := repo.StoreArtifact(ctx, artifact)
		require.NoError(t, err)

		req := &ports.CreatePythonVersionRequest{
			PackageName:    "requests",
			Version:        "2.28.1",
			ArtifactSHA256: artifact.SHA256,
			Filename:       "requests-2.28.1-py3-none-any.whl",
			PythonVersion:  "py3",
			RequiresPython: ">=3.7",
			PackageType:    "bdist_wheel",
			Summary:        "HTTP library",
			Classifiers:    []string{"Development Status :: 5 - Production/Stable"},
			Dependencies:   []string{"urllib3>=1.21.1"},
			UploadedBy:     "testuser",
		}

		ver, err := python.CreateVersion(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, req.Version, ver.Version)
		assert.Equal(t, req.Filename, ver.Filename)

		retrieved, err := python.GetVersion(ctx, "requests", "2.28.1")
		require.NoError(t, err)
		assert.Equal(t, ver.ID, retrieved.ID)
		assert.Equal(t, ver.Version, retrieved.Version)
		assert.Equal(t, len(req.Classifiers), len(retrieved.Classifiers))
	})

	t.Run("list versions", func(t *testing.T) {
		resp, err := python.ListVersions(ctx, &ports.ListPythonVersionsRequest{
			PackageName: "requests",
			Page:        1,
			PageSize:    10,
		})
		require.NoError(t, err)
		assert.Greater(t, len(resp.Versions), 0)
		assert.Greater(t, resp.Total, 0)
	})

	t.Run("delete version", func(t *testing.T) {
		err := python.DeleteVersion(ctx, "requests", "2.28.1")
		require.NoError(t, err)

		_, err = python.GetVersion(ctx, "requests", "2.28.1")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "version not found")
	})
}
