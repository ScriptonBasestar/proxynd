package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"proxynd/internal/config"
	"proxynd/internal/logging"
)

// DockerRequestType represents the type of Docker Registry request
type DockerRequestType int

const (
	// DockerRequestTypeUnknown is unknown request type
	DockerRequestTypeUnknown DockerRequestType = iota
	// DockerRequestTypeAPIVersion is v2 API version check
	DockerRequestTypeAPIVersion
	// DockerRequestTypeManifest is manifest request
	DockerRequestTypeManifest
	// DockerRequestTypeBlob is blob request
	DockerRequestTypeBlob
	// DockerRequestTypeTags is tags list request
	DockerRequestTypeTags
	// DockerRequestTypeCatalog is catalog request
	DockerRequestTypeCatalog
)

// Docker manifest media types
const (
	MediaTypeDockerManifestV1     = "application/vnd.docker.distribution.manifest.v1+json"
	MediaTypeDockerManifestV2     = "application/vnd.docker.distribution.manifest.v2+json"
	MediaTypeDockerManifestListV2 = "application/vnd.docker.distribution.manifest.list.v2+json"
	MediaTypeOCIManifestV1        = "application/vnd.oci.image.manifest.v1+json"
	MediaTypeOCIIndexV1           = "application/vnd.oci.image.index.v1+json"
)

// DockerService handles Docker registry proxy requests
type DockerService struct {
	*BaseProxyService
	config     *config.DockerProxySettings
	httpClient *http.Client
}

// NewDockerService creates a new Docker proxy service
func NewDockerService(
	cache CacheService,
	configService ConfigService,
	upstreamClient UpstreamClient,
) (*DockerService, error) {
	base := NewBaseProxyService("docker", cache, configService, upstreamClient)

	// Load Docker-specific configuration
	configInterface, err := configService.GetProxyConfig(context.Background(), "docker")
	if err != nil {
		return nil, fmt.Errorf("failed to load docker config: %w", err)
	}

	dockerConfig, ok := configInterface.(*config.DockerProxySettings)
	if !ok {
		return nil, fmt.Errorf("invalid docker config type")
	}

	return &DockerService{
		BaseProxyService: base,
		config:           dockerConfig,
		httpClient:       &http.Client{},
	}, nil
}

// HandleRequest processes a Docker proxy request
func (s *DockerService) HandleRequest(ctx context.Context, req ProxyRequest) (*ProxyResponse, error) {
	// Validate request
	if err := s.ValidateRequest(req); err != nil {
		return s.HandleError(err, http.StatusBadRequest), nil
	}

	// Determine request type
	reqType := s.parseRequestType(req.Path)

	switch reqType {
	case DockerRequestTypeAPIVersion:
		return s.handleAPIVersion()
	case DockerRequestTypeManifest:
		return s.handleManifest(ctx, req)
	case DockerRequestTypeBlob:
		return s.handleBlob(ctx, req)
	case DockerRequestTypeTags:
		return s.handleTags(ctx, req)
	case DockerRequestTypeCatalog:
		return s.handleCatalog(ctx, req)
	default:
		return s.HandleError(fmt.Errorf("unknown docker request type"), http.StatusBadRequest), nil
	}
}

// parseRequestType determines the Docker Registry v2 request type from path
func (s *DockerService) parseRequestType(path string) DockerRequestType {
	// Normalize path
	path = strings.TrimPrefix(path, "/")

	// v2 API version check
	if path == "v2" || path == "v2/" {
		return DockerRequestTypeAPIVersion
	}

	// Manifests: /v2/<name>/manifests/<reference>
	if strings.Contains(path, "/manifests/") {
		return DockerRequestTypeManifest
	}

	// Blobs: /v2/<name>/blobs/<digest>
	if strings.Contains(path, "/blobs/") {
		return DockerRequestTypeBlob
	}

	// Tags list: /v2/<name>/tags/list
	if strings.HasSuffix(path, "/tags/list") {
		return DockerRequestTypeTags
	}

	// Catalog: /v2/_catalog
	if strings.Contains(path, "_catalog") {
		return DockerRequestTypeCatalog
	}

	return DockerRequestTypeUnknown
}

// handleAPIVersion handles the v2 API version check endpoint
func (s *DockerService) handleAPIVersion() (*ProxyResponse, error) {
	response := map[string]interface{}{
		"errors": []interface{}{},
	}

	body, err := json.Marshal(response)
	if err != nil {
		return s.HandleError(err, http.StatusInternalServerError), nil
	}

	headers := map[string]string{
		"Docker-Distribution-Api-Version": "registry/2.0",
		"Content-Type":                    "application/json",
	}

	return &ProxyResponse{
		Body:        io.NopCloser(bytes.NewReader(body)),
		StatusCode:  http.StatusOK,
		Headers:     headers,
		ContentType: "application/json",
		Cached:      false,
	}, nil
}

// handleManifest handles manifest requests
func (s *DockerService) handleManifest(ctx context.Context, req ProxyRequest) (*ProxyResponse, error) {
	cacheKey := s.BuildCacheKey(req.Path)

	// Try cache first
	content, found, err := s.TryCache(ctx, cacheKey)
	if err != nil {
		s.Logger.Warn("Cache lookup failed", logging.F("error", err))
	}

	if found && content != nil {
		// Read cached content
		data, err := io.ReadAll(content)
		_ = content.Close()
		if err == nil {
			contentType := s.detectManifestMediaType(data)
			headers := map[string]string{
				"Docker-Distribution-Api-Version": "registry/2.0",
				"Content-Type":                    contentType,
			}

			// Add Docker-Content-Digest if available
			if digest := s.extractDigestFromPath(req.Path); digest != "" {
				headers["Docker-Content-Digest"] = digest
			}

			return &ProxyResponse{
				Body:        io.NopCloser(bytes.NewReader(data)),
				StatusCode:  http.StatusOK,
				Headers:     headers,
				ContentType: contentType,
				Cached:      true,
			}, nil
		}
	}

	// Fetch from upstream
	return s.fetchFromUpstream(ctx, req, cacheKey, true)
}

// handleBlob handles blob requests
func (s *DockerService) handleBlob(ctx context.Context, req ProxyRequest) (*ProxyResponse, error) {
	cacheKey := s.BuildCacheKey(req.Path)

	// Try cache first
	content, found, err := s.TryCache(ctx, cacheKey)
	if err != nil {
		s.Logger.Warn("Cache lookup failed", logging.F("error", err))
	}

	if found && content != nil {
		headers := map[string]string{
			"Docker-Distribution-Api-Version": "registry/2.0",
			"Content-Type":                    "application/octet-stream",
		}

		// Add Docker-Content-Digest
		if digest := s.extractDigestFromPath(req.Path); digest != "" {
			headers["Docker-Content-Digest"] = digest
		}

		return &ProxyResponse{
			Body:        content,
			StatusCode:  http.StatusOK,
			Headers:     headers,
			ContentType: "application/octet-stream",
			Cached:      true,
		}, nil
	}

	// Fetch from upstream
	return s.fetchFromUpstream(ctx, req, cacheKey, false)
}

// handleTags handles tags list requests
func (s *DockerService) handleTags(ctx context.Context, req ProxyRequest) (*ProxyResponse, error) {
	// Tags are dynamic, don't cache long
	return s.fetchFromUpstream(ctx, req, "", false)
}

// handleCatalog handles catalog requests
func (s *DockerService) handleCatalog(ctx context.Context, req ProxyRequest) (*ProxyResponse, error) {
	// Catalog is dynamic, don't cache
	return s.fetchFromUpstream(ctx, req, "", false)
}

// fetchFromUpstream fetches content from upstream Docker registry
func (s *DockerService) fetchFromUpstream(ctx context.Context, req ProxyRequest, cacheKey string, isManifest bool) (*ProxyResponse, error) {
	if s.config == nil || len(s.config.Proxies) == 0 {
		return s.HandleError(fmt.Errorf("no docker proxies configured"), http.StatusServiceUnavailable), nil
	}

	for _, proxy := range s.config.Proxies {
		// Build upstream URL
		url := s.buildUpstreamURL(proxy.URL, req.Path)

		// Create request
		httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			s.Logger.Warn("Failed to create request",
				logging.F("proxy", proxy.Name),
				logging.F("error", err))
			continue
		}

		// Copy headers from original request
		s.copyRequestHeaders(req, httpReq)

		// Set authentication if configured
		if proxy.Auth.Username != "" && proxy.Auth.Password != "" {
			httpReq.SetBasicAuth(proxy.Auth.Username, proxy.Auth.Password)
		}

		// Execute request
		resp, err := s.httpClient.Do(httpReq)
		if err != nil {
			s.Logger.Warn("Failed to fetch from upstream",
				logging.F("proxy", proxy.Name),
				logging.F("error", err))
			continue
		}

		// Handle 401 Unauthorized - return auth challenge
		if resp.StatusCode == http.StatusUnauthorized {
			headers := make(map[string]string)
			for k, v := range resp.Header {
				if len(v) > 0 {
					headers[k] = v[0]
				}
			}
			_ = resp.Body.Close()
			return &ProxyResponse{
				Body:        io.NopCloser(bytes.NewReader([]byte{})),
				StatusCode:  http.StatusUnauthorized,
				Headers:     headers,
				ContentType: "application/json",
				Cached:      false,
			}, nil
		}

		// Handle success
		if resp.StatusCode == http.StatusOK {
			// Read response body
			data, err := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			if err != nil {
				s.Logger.Warn("Failed to read response body",
					logging.F("proxy", proxy.Name),
					logging.F("error", err))
				continue
			}

			// Cache the response if cacheKey is provided
			if cacheKey != "" {
				if err := s.CacheResponse(ctx, cacheKey, bytes.NewReader(data)); err != nil {
					s.Logger.Warn("Failed to cache response", logging.F("error", err))
				}
			}

			// Build response headers
			headers := map[string]string{
				"Docker-Distribution-Api-Version": "registry/2.0",
			}

			// Copy relevant headers from upstream
			for _, h := range []string{"Docker-Content-Digest", "Content-Length", "ETag"} {
				if v := resp.Header.Get(h); v != "" {
					headers[h] = v
				}
			}

			// Determine content type
			contentType := "application/octet-stream"
			if isManifest {
				contentType = s.detectManifestMediaType(data)
			}
			headers["Content-Type"] = contentType

			s.Logger.Info("Successfully fetched from upstream",
				logging.F("proxy", proxy.Name),
				logging.F("path", req.Path))

			return &ProxyResponse{
				Body:        io.NopCloser(bytes.NewReader(data)),
				StatusCode:  http.StatusOK,
				Headers:     headers,
				ContentType: contentType,
				Cached:      false,
			}, nil
		}

		// Not found or other error, try next proxy
		_ = resp.Body.Close()
		s.Logger.Debug("Upstream returned non-OK status",
			logging.F("proxy", proxy.Name),
			logging.F("status", resp.StatusCode))
	}

	return s.HandleNotFound("resource not found in any upstream registry"), nil
}

// buildUpstreamURL constructs the upstream Docker registry URL
func (s *DockerService) buildUpstreamURL(serverURL, requestPath string) string {
	// Ensure v2 prefix
	if !strings.HasPrefix(requestPath, "v2/") && !strings.HasPrefix(requestPath, "/v2/") {
		requestPath = "v2/" + requestPath
	}

	// Remove leading slash from path
	requestPath = strings.TrimPrefix(requestPath, "/")

	// Remove trailing slash from server URL
	serverURL = strings.TrimSuffix(serverURL, "/")

	return serverURL + "/" + requestPath
}

// copyRequestHeaders copies relevant headers from the original request
func (s *DockerService) copyRequestHeaders(req ProxyRequest, httpReq *http.Request) {
	// Copy Accept header for manifest requests
	if req.Headers != nil {
		for _, h := range []string{"Accept", "Authorization", "User-Agent", "Docker-Distribution-Api-Version"} {
			if v, ok := req.Headers[h]; ok && v != "" {
				httpReq.Header.Set(h, v)
			}
		}
	}

	// Set default Accept for manifest requests
	if httpReq.Header.Get("Accept") == "" && strings.Contains(req.Path, "/manifests/") {
		httpReq.Header.Set("Accept", strings.Join([]string{
			MediaTypeDockerManifestV2,
			MediaTypeDockerManifestListV2,
			MediaTypeOCIManifestV1,
			MediaTypeOCIIndexV1,
			MediaTypeDockerManifestV1,
		}, ", "))
	}
}

// detectManifestMediaType detects the media type of a manifest
func (s *DockerService) detectManifestMediaType(data []byte) string {
	var manifest map[string]interface{}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return MediaTypeDockerManifestV2
	}

	// Check schema version
	if schemaVersion, ok := manifest["schemaVersion"].(float64); ok {
		if schemaVersion == 1 {
			return MediaTypeDockerManifestV1
		}
	}

	// Check for manifest list
	if _, ok := manifest["manifests"]; ok {
		return MediaTypeDockerManifestListV2
	}

	// Check for OCI media type
	if mediaType, ok := manifest["mediaType"].(string); ok {
		switch mediaType {
		case MediaTypeOCIManifestV1, MediaTypeOCIIndexV1:
			return mediaType
		}
	}

	return MediaTypeDockerManifestV2
}

// extractDigestFromPath extracts the digest from a path containing sha256:
func (s *DockerService) extractDigestFromPath(path string) string {
	parts := strings.Split(path, "/")
	for _, part := range parts {
		if strings.HasPrefix(part, "sha256:") {
			return part
		}
	}
	return ""
}

// SupportsHead returns whether the service supports HEAD requests
func (s *DockerService) SupportsHead() bool {
	return true
}

// HandleHead handles HEAD requests for blob existence checks
func (s *DockerService) HandleHead(ctx context.Context, req ProxyRequest) (*ProxyResponse, error) {
	// Check cache first
	cacheKey := s.BuildCacheKey(req.Path)
	exists, err := s.Cache.Exists(ctx, cacheKey)
	if err != nil {
		s.Logger.Warn("Cache existence check failed", logging.F("error", err))
	}

	if exists {
		headers := map[string]string{
			"Docker-Distribution-Api-Version": "registry/2.0",
		}

		if digest := s.extractDigestFromPath(req.Path); digest != "" {
			headers["Docker-Content-Digest"] = digest
		}

		return &ProxyResponse{
			Body:        io.NopCloser(bytes.NewReader(nil)),
			StatusCode:  http.StatusOK,
			Headers:     headers,
			ContentType: "application/octet-stream",
			Cached:      true,
		}, nil
	}

	// Check upstream
	if s.config == nil || len(s.config.Proxies) == 0 {
		return s.HandleNotFound("no proxies configured"), nil
	}

	for _, proxy := range s.config.Proxies {
		url := s.buildUpstreamURL(proxy.URL, req.Path)

		httpReq, err := http.NewRequestWithContext(ctx, "HEAD", url, nil)
		if err != nil {
			continue
		}

		s.copyRequestHeaders(req, httpReq)

		if proxy.Auth.Username != "" && proxy.Auth.Password != "" {
			httpReq.SetBasicAuth(proxy.Auth.Username, proxy.Auth.Password)
		}

		resp, err := s.httpClient.Do(httpReq)
		if err != nil {
			continue
		}
		_ = resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			headers := map[string]string{
				"Docker-Distribution-Api-Version": "registry/2.0",
			}

			for _, h := range []string{"Docker-Content-Digest", "Content-Length", "Content-Type"} {
				if v := resp.Header.Get(h); v != "" {
					headers[h] = v
				}
			}

			return &ProxyResponse{
				Body:        io.NopCloser(bytes.NewReader(nil)),
				StatusCode:  http.StatusOK,
				Headers:     headers,
				ContentType: resp.Header.Get("Content-Type"),
				Cached:      false,
			}, nil
		}
	}

	return s.HandleNotFound("blob not found"), nil
}
