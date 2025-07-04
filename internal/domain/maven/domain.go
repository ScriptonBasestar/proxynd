package maven

import (
	"context"
	"fmt"
	"path"
	"regexp"
	"strings"

	"proxynd/internal/domain/common"
)

// Domain implements Maven-specific proxy domain logic
type Domain struct {
	artifactPattern *regexp.Regexp
	metadataPattern *regexp.Regexp
}

// NewDomain creates a new Maven domain instance
func NewDomain() *Domain {
	return &Domain{
		// Pattern for Maven artifacts: groupId/artifactId/version/artifactId-version[-classifier].extension
		artifactPattern: regexp.MustCompile(`^(.+)/([^/]+)/([^/]+)/([^/]+)$`),
		// Pattern for Maven metadata files
		metadataPattern: regexp.MustCompile(`^(.+)/maven-metadata\.xml(\.sha1|\.md5)?$`),
	}
}

// ParseRequest parses and validates a Maven proxy request
func (d *Domain) ParseRequest(ctx context.Context, req *common.ProxyRequest) error {
	if req.Method != "GET" && req.Method != "HEAD" && req.Method != "PUT" {
		return fmt.Errorf("unsupported method for Maven proxy: %s", req.Method)
	}
	
	// Validate path
	if err := d.ValidatePath(req.Path); err != nil {
		return err
	}
	
	return nil
}

// BuildUpstreamURL constructs the upstream URL for a Maven request
func (d *Domain) BuildUpstreamURL(repo common.Repository, path string) string {
	// Ensure base URL doesn't end with slash
	baseURL := strings.TrimRight(repo.URL, "/")
	
	// Ensure path starts with slash
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	
	return baseURL + path
}

// ValidatePath validates if the requested path is valid for Maven
func (d *Domain) ValidatePath(requestPath string) error {
	// Remove leading slash for pattern matching
	cleanPath := strings.TrimPrefix(requestPath, "/")
	
	// Check if it's a valid artifact path
	if d.artifactPattern.MatchString(cleanPath) {
		return nil
	}
	
	// Check if it's a metadata file
	if d.metadataPattern.MatchString(cleanPath) {
		return nil
	}
	
	// Check for other valid Maven files
	if strings.HasSuffix(cleanPath, ".pom") ||
		strings.HasSuffix(cleanPath, ".jar") ||
		strings.HasSuffix(cleanPath, ".war") ||
		strings.HasSuffix(cleanPath, ".ear") ||
		strings.HasSuffix(cleanPath, ".sha1") ||
		strings.HasSuffix(cleanPath, ".md5") ||
		strings.HasSuffix(cleanPath, ".asc") {
		return nil
	}
	
	return fmt.Errorf("invalid Maven path: %s", requestPath)
}

// ExtractMetadata extracts package metadata from the Maven request path
func (d *Domain) ExtractMetadata(requestPath string) (*common.PackageMetadata, error) {
	cleanPath := strings.TrimPrefix(requestPath, "/")
	
	// Try to match artifact pattern
	matches := d.artifactPattern.FindStringSubmatch(cleanPath)
	if len(matches) != 5 {
		return nil, fmt.Errorf("cannot extract metadata from path: %s", requestPath)
	}
	
	groupPath := matches[1]
	artifactId := matches[2]
	version := matches[3]
	filename := matches[4]
	
	// Convert group path to groupId (replace / with .)
	groupId := strings.ReplaceAll(groupPath, "/", ".")
	
	return &common.PackageMetadata{
		Name:    fmt.Sprintf("%s:%s", groupId, artifactId),
		Version: version,
		// Additional metadata would be extracted from POM file
	}, nil
}

// TransformResponse transforms the upstream response if needed
func (d *Domain) TransformResponse(ctx context.Context, resp *common.ProxyResponse) error {
	// Maven responses typically don't need transformation
	// This could be extended to modify repository URLs in metadata files
	return nil
}

// GetContentType returns the content type for a given Maven file
func (d *Domain) GetContentType(filename string) string {
	ext := strings.ToLower(path.Ext(filename))
	
	switch ext {
	case ".pom", ".xml":
		return "application/xml"
	case ".jar":
		return "application/java-archive"
	case ".war":
		return "application/java-archive"
	case ".ear":
		return "application/java-archive"
	case ".sha1", ".md5":
		return "text/plain"
	case ".asc":
		return "text/plain"
	default:
		return "application/octet-stream"
	}
}

// ShouldCache determines if a Maven response should be cached
func (d *Domain) ShouldCache(requestPath string, resp *common.ProxyResponse) bool {
	// Don't cache error responses
	if resp.StatusCode >= 400 {
		return false
	}
	
	// Don't cache SNAPSHOT versions (they can change)
	if strings.Contains(requestPath, "-SNAPSHOT") {
		return false
	}
	
	// Don't cache metadata files (they can change)
	if strings.Contains(requestPath, "maven-metadata.xml") {
		return false
	}
	
	// Cache everything else
	return true
}

// ParseArtifactPath parses a Maven artifact path into its components
func (d *Domain) ParseArtifactPath(path string) (*ArtifactInfo, error) {
	cleanPath := strings.TrimPrefix(path, "/")
	
	matches := d.artifactPattern.FindStringSubmatch(cleanPath)
	if len(matches) != 5 {
		return nil, fmt.Errorf("invalid artifact path: %s", path)
	}
	
	groupPath := matches[1]
	artifactId := matches[2]
	version := matches[3]
	filename := matches[4]
	
	// Convert group path to groupId
	groupId := strings.ReplaceAll(groupPath, "/", ".")
	
	// Extract classifier and extension from filename
	classifier, extension := d.parseFilename(filename, artifactId, version)
	
	return &ArtifactInfo{
		GroupId:    groupId,
		ArtifactId: artifactId,
		Version:    version,
		Classifier: classifier,
		Extension:  extension,
		Filename:   filename,
	}, nil
}

// parseFilename extracts classifier and extension from a Maven filename
func (d *Domain) parseFilename(filename, artifactId, version string) (classifier, extension string) {
	// Expected format: artifactId-version[-classifier].extension
	prefix := fmt.Sprintf("%s-%s", artifactId, version)
	
	if !strings.HasPrefix(filename, prefix) {
		// Unexpected format, just return extension
		ext := path.Ext(filename)
		if ext != "" {
			extension = ext[1:] // Remove leading dot
		}
		return "", extension
	}
	
	remainder := filename[len(prefix):]
	
	// If remainder starts with -, we have a classifier
	if strings.HasPrefix(remainder, "-") {
		// Find the last dot for extension
		lastDot := strings.LastIndex(remainder, ".")
		if lastDot > 0 {
			classifier = remainder[1:lastDot] // Skip leading dash
			extension = remainder[lastDot+1:]
		} else {
			classifier = remainder[1:] // No extension
		}
	} else if strings.HasPrefix(remainder, ".") {
		// No classifier, just extension
		extension = remainder[1:]
	}
	
	return classifier, extension
}

// ArtifactInfo represents parsed Maven artifact information
type ArtifactInfo struct {
	GroupId    string
	ArtifactId string
	Version    string
	Classifier string
	Extension  string
	Filename   string
}

// Ensure Domain implements the ProxyDomain interface
var _ common.ProxyDomain = (*Domain)(nil)