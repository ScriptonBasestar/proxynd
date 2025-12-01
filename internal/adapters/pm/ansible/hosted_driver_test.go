//go:build integration

package ansible_test

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"proxynd/internal/adapters/observability/zap"
	"proxynd/internal/adapters/pm/ansible"
	"proxynd/internal/adapters/storage/cas"
	"proxynd/internal/config"
	"proxynd/internal/ports"
	"proxynd/internal/repositories/metadata/sqlite"
)

// TestHostedDriver_StoreAndGetPackage tests the full upload and download cycle
func TestHostedDriver_StoreAndGetPackage(t *testing.T) {
	// Setup: Create temporary directories
	tmpDir := t.TempDir()
	casDir := filepath.Join(tmpDir, "cas")
	dbPath := filepath.Join(tmpDir, "metadata.db")

	// Setup: Create CAS
	casStorage, err := cas.NewLocalCAS(casDir)
	require.NoError(t, err, "Failed to create CAS")

	// Setup: Create SQLite metadata repository
	metadataRepo, err := sqlite.NewSQLiteMetadataRepository(dbPath)
	require.NoError(t, err, "Failed to create SQLite metadata repository")
	defer metadataRepo.Close()

	// Setup: Create logger
	logger, err := zap.NewLogger(zap.DevelopmentLoggerConfig())
	require.NoError(t, err, "Failed to create logger")

	// Setup: Create driver config
	cfg := &config.AnsibleRepositoryConfig{
		Name:              "test-repo",
		Type:              "hosted",
		Enabled:           true,
		MaxUploadSize:     10 * 1024 * 1024, // 10MB
		AllowedNamespaces: []string{"testns"},
		RequireAuth:       false,
	}

	// Setup: Create hosted driver
	driver, err := ansible.NewHostedDriver(casStorage, metadataRepo, cfg, logger)
	require.NoError(t, err, "Failed to create hosted driver")

	// Test data: Create a simple tarball with galaxy.yml
	tarballContent := createTestTarball(t, "testns", "testcol", "1.0.0")

	// Calculate actual SHA256
	hash := sha256.Sum256(tarballContent)
	sha256Hash := fmt.Sprintf("%x", hash)

	ctx := context.Background()

	// Test: Store package
	t.Run("StorePackage", func(t *testing.T) {
		req := &ports.StorePackageRequest{
			Content:    bytes.NewReader(tarballContent),
			Size:       int64(len(tarballContent)),
			SHA256:     sha256Hash,
			UploadedBy: "test-user",
		}

		resp, err := driver.StorePackage(ctx, req)
		require.NoError(t, err, "StorePackage should succeed")
		assert.NotNil(t, resp)
		assert.Equal(t, sha256Hash, resp.SHA256)
		assert.Equal(t, "testns-testcol-1.0.0.tar.gz", resp.PackagePath)
	})

	// Test: Get package
	t.Run("GetPackage", func(t *testing.T) {
		req := &ports.GetPackageRequest{
			Name:    "testns.testcol",
			Version: "1.0.0",
		}

		resp, err := driver.GetPackage(ctx, req)
		require.NoError(t, err, "GetPackage should succeed")
		assert.NotNil(t, resp)
		assert.Equal(t, sha256Hash, resp.SHA256)
		assert.Equal(t, "application/gzip", resp.ContentType)
		assert.NotNil(t, resp.Content)

		// Verify metadata
		assert.Equal(t, "testns", resp.Metadata.Namespace)
		assert.Equal(t, "testcol", resp.Metadata.Name)
		assert.Equal(t, "1.0.0", resp.Metadata.Version)
	})

	// Test: List packages
	t.Run("ListPackages", func(t *testing.T) {
		req := &ports.ListPackagesRequest{
			Page:     1,
			PageSize: 10,
		}

		resp, err := driver.ListPackages(ctx, req)
		require.NoError(t, err, "ListPackages should succeed")
		assert.NotNil(t, resp)
		assert.GreaterOrEqual(t, resp.Total, int64(1))
		assert.Len(t, resp.Packages, 1)
		assert.Equal(t, "testns", resp.Packages[0].Namespace)
		assert.Equal(t, "testcol", resp.Packages[0].Name)
	})

	// Test: Get metadata
	t.Run("GetMetadata", func(t *testing.T) {
		req := &ports.GetMetadataRequest{
			Name: "testns.testcol",
		}

		meta, err := driver.GetMetadata(ctx, req)
		require.NoError(t, err, "GetMetadata should succeed")
		assert.NotNil(t, meta)
		assert.Equal(t, "testns", meta.Namespace)
		assert.Equal(t, "testcol", meta.Name)
	})

	// Test: Delete package
	t.Run("DeletePackage", func(t *testing.T) {
		req := &ports.DeletePackageRequest{
			Name:    "testns.testcol",
			Version: "1.0.0",
		}

		err := driver.DeletePackage(ctx, req)
		require.NoError(t, err, "DeletePackage should succeed")

		// Verify deletion: GetPackage should fail
		getReq := &ports.GetPackageRequest{
			Name:    "testns.testcol",
			Version: "1.0.0",
		}
		_, err = driver.GetPackage(ctx, getReq)
		assert.Error(t, err, "GetPackage should fail after deletion")
	})
}

// TestHostedDriver_NamespaceValidation tests namespace permission checks
func TestHostedDriver_NamespaceValidation(t *testing.T) {
	// Setup (similar to above but simplified)
	tmpDir := t.TempDir()
	casDir := filepath.Join(tmpDir, "cas")
	dbPath := filepath.Join(tmpDir, "metadata.db")

	casStorage, err := cas.NewLocalCAS(casDir)
	require.NoError(t, err)

	metadataRepo, err := sqlite.NewSQLiteMetadataRepository(dbPath)
	require.NoError(t, err)
	defer metadataRepo.Close()

	logger, err := zap.NewLogger(zap.DevelopmentLoggerConfig())
	require.NoError(t, err)

	cfg := &config.AnsibleRepositoryConfig{
		Name:              "test-repo",
		Type:              "hosted",
		Enabled:           true,
		MaxUploadSize:     10 * 1024 * 1024,
		AllowedNamespaces: []string{"allowedns"}, // Only "allowedns" is allowed
		RequireAuth:       false,
	}

	driver, err := ansible.NewHostedDriver(casStorage, metadataRepo, cfg, logger)
	require.NoError(t, err)

	ctx := context.Background()

	// Test: Upload with forbidden namespace should fail
	t.Run("ForbiddenNamespace", func(t *testing.T) {
		tarballContent := createTestTarball(t, "forbiddenns", "testcol", "1.0.0")
		req := &ports.StorePackageRequest{
			Content:    bytes.NewReader(tarballContent),
			Size:       int64(len(tarballContent)),
			SHA256:     "test-sha256",
			UploadedBy: "test-user",
		}

		_, err := driver.StorePackage(ctx, req)
		assert.Error(t, err, "StorePackage should fail for forbidden namespace")
		assert.Contains(t, err.Error(), "not allowed", "Error should mention namespace restriction")
	})

	// Test: Upload with allowed namespace should succeed
	t.Run("AllowedNamespace", func(t *testing.T) {
		tarballContent := createTestTarball(t, "allowedns", "testcol", "1.0.0")
		req := &ports.StorePackageRequest{
			Content:    bytes.NewReader(tarballContent),
			Size:       int64(len(tarballContent)),
			SHA256:     "test-sha256-allowed",
			UploadedBy: "test-user",
		}

		resp, err := driver.StorePackage(ctx, req)
		require.NoError(t, err, "StorePackage should succeed for allowed namespace")
		assert.NotNil(t, resp)
	})
}

// createTestTarball creates a minimal valid Ansible collection tarball for testing
func createTestTarball(t *testing.T, namespace, name, version string) []byte {
	t.Helper()

	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gzw)

	baseDir := fmt.Sprintf("%s-%s-%s", namespace, name, version)

	// 1. Create galaxy.yml
	galaxyYML := fmt.Sprintf(`namespace: %s
name: %s
version: %s
authors:
  - Test Author <test@example.com>
description: Test collection for integration testing
license:
  - MIT
tags:
  - test
  - integration
dependencies: {}
repository: https://github.com/test/test-collection
documentation: https://docs.example.com
homepage: https://example.com
issues: https://github.com/test/test-collection/issues
requires_ansible: ">=2.9.0"
`, namespace, name, version)

	addFileToTar(t, tw, filepath.Join(baseDir, "galaxy.yml"), galaxyYML)

	// 2. Create MANIFEST.json
	manifestJSON := `{
  "collection_info": {
    "namespace": "` + namespace + `",
    "name": "` + name + `",
    "version": "` + version + `",
    "authors": ["Test Author <test@example.com>"],
    "description": "Test collection for integration testing",
    "license": ["MIT"],
    "tags": ["test", "integration"],
    "dependencies": {},
    "repository": "https://github.com/test/test-collection",
    "documentation": "https://docs.example.com",
    "homepage": "https://example.com",
    "issues": "https://github.com/test/test-collection/issues"
  },
  "file_manifest_file": {
    "name": "FILES.json",
    "ftype": "file",
    "chksum_type": "sha256",
    "chksum_sha256": "placeholder"
  },
  "format": 1
}`

	addFileToTar(t, tw, filepath.Join(baseDir, "MANIFEST.json"), manifestJSON)

	// 3. Create FILES.json
	filesJSON := `{
  "files": [
    {
      "name": ".",
      "ftype": "dir",
      "chksum_type": null,
      "chksum_sha256": null,
      "format": 1
    },
    {
      "name": "README.md",
      "ftype": "file",
      "chksum_type": "sha256",
      "chksum_sha256": "placeholder"
    }
  ],
  "format": 1
}`

	addFileToTar(t, tw, filepath.Join(baseDir, "FILES.json"), filesJSON)

	// 4. Create README.md
	readmeMD := fmt.Sprintf(`# %s.%s

Test collection for integration testing.

## Version

%s

## Installation

`+"```"+`bash
ansible-galaxy collection install %s.%s
`+"```"+`
`, namespace, name, version, namespace, name)

	addFileToTar(t, tw, filepath.Join(baseDir, "README.md"), readmeMD)

	// 5. Create plugins/modules directory with a sample module
	modulePy := `#!/usr/bin/python
# -*- coding: utf-8 -*-

DOCUMENTATION = r'''
---
module: test_module
short_description: Test module
description:
    - This is a test module for integration testing
version_added: "1.0.0"
author:
    - Test Author (@testauthor)
'''

EXAMPLES = r'''
- name: Test example
  test_module:
'''

RETURN = r'''
msg:
    description: Test message
    returned: always
    type: str
'''

from ansible.module_utils.basic import AnsibleModule

def main():
    module = AnsibleModule(
        argument_spec=dict()
    )
    module.exit_json(changed=False, msg='Test module executed')

if __name__ == '__main__':
    main()
`

	addFileToTar(t, tw, filepath.Join(baseDir, "plugins", "modules", "test_module.py"), modulePy)

	// 6. Create meta/runtime.yml
	runtimeYML := `requires_ansible: ">=2.9.0"
`

	addFileToTar(t, tw, filepath.Join(baseDir, "meta", "runtime.yml"), runtimeYML)

	// Close tar and gzip writers
	require.NoError(t, tw.Close(), "Failed to close tar writer")
	require.NoError(t, gzw.Close(), "Failed to close gzip writer")

	return buf.Bytes()
}

// addFileToTar adds a file to the tar archive
func addFileToTar(t *testing.T, tw *tar.Writer, name, content string) {
	t.Helper()

	hdr := &tar.Header{
		Name: name,
		Mode: 0o644,
		Size: int64(len(content)),
	}

	require.NoError(t, tw.WriteHeader(hdr), "Failed to write tar header for "+name)
	_, err := tw.Write([]byte(content))
	require.NoError(t, err, "Failed to write tar content for "+name)
}
