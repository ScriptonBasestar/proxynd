package ansible

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"path/filepath"
	"strings"
)

// TarballValidator validates Ansible collection tarballs
type TarballValidator struct {
	MaxSize int64 // Maximum tarball size in bytes
}

// NewTarballValidator creates a new tarball validator
func NewTarballValidator(maxSize int64) *TarballValidator {
	return &TarballValidator{
		MaxSize: maxSize,
	}
}

// Forbidden files and directories
var (
	forbiddenPatterns = []string{
		".git",
		".svn",
		".hg",
		"__pycache__",
		"*.pyc",
		"*.pyo",
		".DS_Store",
		"Thumbs.db",
		".idea",
		".vscode",
	}
)

// ValidateSize validates the tarball size
func (v *TarballValidator) ValidateSize(size int64) error {
	if size > v.MaxSize {
		return fmt.Errorf("%w: %d bytes (max: %d)", ErrTarballTooLarge, size, v.MaxSize)
	}
	return nil
}

// ValidateStructure validates the tarball structure
func (v *TarballValidator) ValidateStructure(tarballReader io.Reader) error {
	// Create gzip reader
	gzr, err := gzip.NewReader(tarballReader)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrTarballCorrupted, err)
	}
	defer gzr.Close()

	// Create tar reader
	tr := tar.NewReader(gzr)

	hasGalaxyYml := false
	hasManifestJson := false
	totalSize := int64(0)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("%w: %v", ErrTarballCorrupted, err)
		}

		// Accumulate total size
		totalSize += header.Size

		// Check for size limit during extraction
		if totalSize > v.MaxSize*2 { // Allow 2x compression ratio
			return fmt.Errorf("%w: uncompressed size exceeds limit", ErrTarballTooLarge)
		}

		// Check for dangerous paths
		if err := validatePath(header.Name); err != nil {
			return err
		}

		// Check for forbidden files
		if isForbiddenFile(header.Name) {
			return fmt.Errorf("%w: %s", ErrForbiddenFile, header.Name)
		}

		// Check for required files
		filename := filepath.Base(header.Name)
		if filename == "galaxy.yml" || filename == "galaxy.yaml" {
			hasGalaxyYml = true
		}
		if filename == "MANIFEST.json" {
			hasManifestJson = true
		}

		// Validate file type
		if header.Typeflag != tar.TypeDir && header.Typeflag != tar.TypeReg {
			return fmt.Errorf("%w: unsupported file type in %s", ErrTarballCorrupted, header.Name)
		}
	}

	// galaxy.yml is required
	if !hasGalaxyYml {
		return ErrManifestNotFound
	}

	// MANIFEST.json is optional but recommended
	_ = hasManifestJson

	return nil
}

// validatePath checks for directory traversal and absolute paths
func validatePath(path string) error {
	// Clean the path
	cleaned := filepath.Clean(path)

	// Check for absolute paths
	if filepath.IsAbs(cleaned) {
		return fmt.Errorf("%w: absolute path not allowed: %s", ErrInvalidTarballPath, path)
	}

	// Check for directory traversal
	if strings.Contains(cleaned, "..") {
		return fmt.Errorf("%w: directory traversal not allowed: %s", ErrInvalidTarballPath, path)
	}

	// Check for paths starting with /
	if strings.HasPrefix(path, "/") {
		return fmt.Errorf("%w: absolute path not allowed: %s", ErrInvalidTarballPath, path)
	}

	return nil
}

// isForbiddenFile checks if a file matches forbidden patterns
func isForbiddenFile(path string) bool {
	// Get base name and directory components
	parts := strings.Split(filepath.Clean(path), string(filepath.Separator))

	for _, part := range parts {
		for _, pattern := range forbiddenPatterns {
			// Simple pattern matching
			if pattern == part {
				return true
			}

			// Wildcard matching for extensions
			if strings.HasPrefix(pattern, "*.") {
				ext := pattern[1:] // Remove *
				if strings.HasSuffix(part, ext) {
					return true
				}
			}
		}
	}

	return false
}

// ValidateCollectionFilename validates the collection tarball filename
// Expected format: namespace-name-version.tar.gz
func ValidateCollectionFilename(filename string) (namespace, name, version string, err error) {
	// Remove .tar.gz extension
	if !strings.HasSuffix(filename, ".tar.gz") {
		return "", "", "", fmt.Errorf("invalid filename: must end with .tar.gz")
	}

	baseName := strings.TrimSuffix(filename, ".tar.gz")

	// Split by hyphen
	parts := strings.Split(baseName, "-")
	if len(parts) < 3 {
		return "", "", "", fmt.Errorf("invalid filename format: expected namespace-name-version.tar.gz")
	}

	namespace = parts[0]
	name = parts[1]
	version = strings.Join(parts[2:], "-") // Version might contain hyphens (e.g., 1.0.0-beta1)

	// Validate namespace
	if err := ValidateNamespace(namespace); err != nil {
		return "", "", "", fmt.Errorf("invalid namespace in filename: %w", err)
	}

	// Validate name
	if err := ValidateName(name); err != nil {
		return "", "", "", fmt.Errorf("invalid name in filename: %w", err)
	}

	// Validate version
	if err := ValidateVersion(version); err != nil {
		return "", "", "", fmt.Errorf("invalid version in filename: %w", err)
	}

	return namespace, name, version, nil
}
