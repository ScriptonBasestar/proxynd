package common

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"proxynd/internal/ports"
)

// PackageNormalizer implements ports.PackageNormalizer for all package managers
type PackageNormalizer struct{}

// NewPackageNormalizer creates a new package normalizer
func NewPackageNormalizer() ports.PackageNormalizer {
	return &PackageNormalizer{}
}

// NormalizePath normalizes package path for the specific package manager
func (n *PackageNormalizer) NormalizePath(pmType, path string) (string, error) {
	switch pmType {
	case "maven":
		return n.normalizeMavenPath(path)
	case "npm":
		return n.normalizeNpmPath(path)
	case "apt":
		return n.normalizeAptPath(path)
	case "pypi":
		return n.normalizePypiPath(path)
	case "yum":
		return n.normalizeYumPath(path)
	case "apk":
		return n.normalizeApkPath(path)
	case "registry":
		return n.normalizeRegistryPath(path)
	default:
		return n.normalizeGenericPath(path)
	}
}

// ValidatePath validates if the path is valid for the package manager
func (n *PackageNormalizer) ValidatePath(pmType, path string) error {
	normalized, err := n.NormalizePath(pmType, path)
	if err != nil {
		return err
	}
	
	switch pmType {
	case "maven":
		return n.validateMavenPath(normalized)
	case "npm":
		return n.validateNpmPath(normalized)
	case "apt":
		return n.validateAptPath(normalized)
	case "pypi":
		return n.validatePypiPath(normalized)
	case "yum":
		return n.validateYumPath(normalized)
	case "apk":
		return n.validateApkPath(normalized)
	case "registry":
		return n.validateRegistryPath(normalized)
	default:
		return nil
	}
}

// ExtractComponents extracts package components from path
func (n *PackageNormalizer) ExtractComponents(pmType, path string) (*ports.PathComponents, error) {
	normalized, err := n.NormalizePath(pmType, path)
	if err != nil {
		return nil, err
	}
	
	switch pmType {
	case "maven":
		return n.extractMavenComponents(normalized)
	case "npm":
		return n.extractNpmComponents(normalized)
	case "apt":
		return n.extractAptComponents(normalized)
	case "pypi":
		return n.extractPypiComponents(normalized)
	case "yum":
		return n.extractYumComponents(normalized)
	case "apk":
		return n.extractApkComponents(normalized)
	case "registry":
		return n.extractRegistryComponents(normalized)
	default:
		return &ports.PathComponents{}, nil
	}
}

// Maven-specific normalization
func (n *PackageNormalizer) normalizeMavenPath(path string) (string, error) {
	// Clean the path
	normalized := filepath.Clean(path)
	if !strings.HasPrefix(normalized, "/") {
		normalized = "/" + normalized
	}
	
	// Remove duplicate slashes
	normalized = regexp.MustCompile(`/+`).ReplaceAllString(normalized, "/")
	
	return normalized, nil
}

func (n *PackageNormalizer) validateMavenPath(path string) error {
	// Maven paths should follow: /groupId/artifactId/version/filename
	if !regexp.MustCompile(`^/[^/]+(/[^/]+)*/[^/]+\.[^/]+$`).MatchString(path) {
		return fmt.Errorf("invalid maven path format: %s", path)
	}
	return nil
}

func (n *PackageNormalizer) extractMavenComponents(path string) (*ports.PathComponents, error) {
	// TODO: Implement Maven GAV (GroupId:ArtifactId:Version) extraction
	// This should parse paths like /org/springframework/spring-core/5.3.21/spring-core-5.3.21.jar
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid maven path: %s", path)
	}
	
	// Extract version and filename
	version := parts[len(parts)-2]
	filename := parts[len(parts)-1]
	
	// Extract artifact ID from filename
	artifactId := parts[len(parts)-3]
	
	// Group ID is everything before artifact ID
	groupId := strings.Join(parts[:len(parts)-3], ".")
	
	return &ports.PathComponents{
		PackageName: fmt.Sprintf("%s:%s", groupId, artifactId),
		Version:     version,
		Extension:   filepath.Ext(filename),
		Attributes: map[string]string{
			"groupId":    groupId,
			"artifactId": artifactId,
			"filename":   filename,
		},
	}, nil
}

// NPM-specific normalization
func (n *PackageNormalizer) normalizeNpmPath(path string) (string, error) {
	normalized := strings.TrimSpace(path)
	if !strings.HasPrefix(normalized, "/") {
		normalized = "/" + normalized
	}
	
	// Handle double encoding of scoped packages
	normalized = strings.ReplaceAll(normalized, "%40", "@")
	normalized = strings.ReplaceAll(normalized, "%2F", "/")
	
	// Convert package names to lowercase
	if strings.HasPrefix(normalized, "/@") {
		// Scoped package
		parts := strings.Split(normalized, "/")
		if len(parts) >= 3 {
			parts[1] = strings.ToLower(parts[1]) // @scope
			parts[2] = strings.ToLower(parts[2]) // package-name
			normalized = strings.Join(parts, "/")
		}
	} else if strings.HasPrefix(normalized, "/") && !strings.HasPrefix(normalized, "/@") {
		// Regular package
		parts := strings.Split(normalized, "/")
		if len(parts) >= 2 {
			parts[1] = strings.ToLower(parts[1])
			normalized = strings.Join(parts, "/")
		}
	}
	
	return normalized, nil
}

func (n *PackageNormalizer) validateNpmPath(path string) error {
	patterns := []string{
		`^/@[^/]+/[^/]+$`,                     // scoped package
		`^/[^@][^/]*$`,                        // regular package
		`^/[^/]+/-/[^/]+-[^/]+\.tgz$`,        // tarball
		`^/@[^/]+/[^/]+/-/[^/]+-[^/]+\.tgz$`, // scoped tarball
	}
	
	for _, pattern := range patterns {
		if matched, _ := regexp.MatchString(pattern, path); matched {
			return nil
		}
	}
	
	return fmt.Errorf("invalid npm path format: %s", path)
}

func (n *PackageNormalizer) extractNpmComponents(path string) (*ports.PathComponents, error) {
	if strings.HasPrefix(path, "/@") {
		// Scoped package: /@scope/package-name
		parts := strings.Split(path, "/")
		if len(parts) >= 3 {
			scope := parts[1]
			packageName := parts[2]
			return &ports.PathComponents{
				PackageName: fmt.Sprintf("%s/%s", scope, packageName),
				Scope:       scope,
				Attributes: map[string]string{
					"scope": scope,
					"name":  packageName,
				},
			}, nil
		}
	} else {
		// Regular package
		parts := strings.Split(strings.Trim(path, "/"), "/")
		if len(parts) >= 1 {
			return &ports.PathComponents{
				PackageName: parts[0],
				Attributes: map[string]string{
					"name": parts[0],
				},
			}, nil
		}
	}
	
	return nil, fmt.Errorf("invalid npm path: %s", path)
}

// Generic normalization for other package managers
func (n *PackageNormalizer) normalizeAptPath(path string) (string, error) {
	return n.normalizeGenericPath(path)
}

func (n *PackageNormalizer) validateAptPath(path string) error {
	// APT paths are quite flexible, basic validation
	if !strings.HasPrefix(path, "/") {
		return fmt.Errorf("path must start with /")
	}
	return nil
}

func (n *PackageNormalizer) extractAptComponents(path string) (*ports.PathComponents, error) {
	// TODO: Implement APT-specific component extraction
	return &ports.PathComponents{}, nil
}

func (n *PackageNormalizer) normalizePypiPath(path string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(path))
	if !strings.HasPrefix(normalized, "/") {
		normalized = "/" + normalized
	}
	
	// PyPI package names normalize _ to - and are case-insensitive
	if strings.HasPrefix(normalized, "/simple/") {
		parts := strings.Split(normalized, "/")
		if len(parts) >= 3 {
			packageName := parts[2]
			packageName = strings.ReplaceAll(packageName, "_", "-")
			packageName = strings.ReplaceAll(packageName, ".", "-")
			parts[2] = packageName
			normalized = strings.Join(parts, "/")
		}
	}
	
	return normalized, nil
}

func (n *PackageNormalizer) validatePypiPath(path string) error {
	patterns := []string{
		`^/simple/[^/]+/?$`,
		`^/pypi/[^/]+/?$`,
		`^/packages/.*\.(tar\.gz|zip|whl|egg)$`,
	}
	
	for _, pattern := range patterns {
		if matched, _ := regexp.MatchString(pattern, path); matched {
			return nil
		}
	}
	
	return fmt.Errorf("invalid pypi path format: %s", path)
}

func (n *PackageNormalizer) extractPypiComponents(path string) (*ports.PathComponents, error) {
	// TODO: Implement PyPI-specific component extraction
	return &ports.PathComponents{}, nil
}

func (n *PackageNormalizer) normalizeYumPath(path string) (string, error) {
	return n.normalizeGenericPath(path)
}

func (n *PackageNormalizer) validateYumPath(path string) error {
	return nil // YUM paths are flexible
}

func (n *PackageNormalizer) extractYumComponents(path string) (*ports.PathComponents, error) {
	// TODO: Implement YUM-specific component extraction
	return &ports.PathComponents{}, nil
}

func (n *PackageNormalizer) normalizeApkPath(path string) (string, error) {
	return n.normalizeGenericPath(path)
}

func (n *PackageNormalizer) validateApkPath(path string) error {
	return nil // APK paths are flexible
}

func (n *PackageNormalizer) extractApkComponents(path string) (*ports.PathComponents, error) {
	// TODO: Implement APK-specific component extraction
	return &ports.PathComponents{}, nil
}

func (n *PackageNormalizer) normalizeRegistryPath(path string) (string, error) {
	normalized := strings.TrimSpace(path)
	if !strings.HasPrefix(normalized, "/") {
		normalized = "/" + normalized
	}
	
	// Ensure v2 API prefix
	if normalized == "/" {
		normalized = "/v2/"
	}
	
	return normalized, nil
}

func (n *PackageNormalizer) validateRegistryPath(path string) error {
	patterns := []string{
		`^/v2/?$`,
		`^/v2/[^/]+/[^/]+/manifests/[^/]+$`,
		`^/v2/[^/]+/[^/]+/blobs/sha256:[a-f0-9]+$`,
		`^/v2/[^/]+/[^/]+/tags/list$`,
		`^/v2/_catalog$`,
	}
	
	for _, pattern := range patterns {
		if matched, _ := regexp.MatchString(pattern, path); matched {
			return nil
		}
	}
	
	return fmt.Errorf("invalid registry path format: %s", path)
}

func (n *PackageNormalizer) extractRegistryComponents(path string) (*ports.PathComponents, error) {
	// TODO: Implement Docker registry-specific component extraction
	return &ports.PathComponents{}, nil
}

// Generic path normalization
func (n *PackageNormalizer) normalizeGenericPath(path string) (string, error) {
	normalized := filepath.Clean(path)
	if !strings.HasPrefix(normalized, "/") {
		normalized = "/" + normalized
	}
	
	// Remove duplicate slashes
	normalized = regexp.MustCompile(`/+`).ReplaceAllString(normalized, "/")
	
	return normalized, nil
}