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

// NpmRequestType represents the type of NPM Registry request
type NpmRequestType int

const (
	// NpmRequestTypeUnknown is unknown request type
	NpmRequestTypeUnknown NpmRequestType = iota
	// NpmRequestTypeMetadata is package metadata request
	NpmRequestTypeMetadata
	// NpmRequestTypeTarball is tarball download request
	NpmRequestTypeTarball
	// NpmRequestTypeSearch is search request
	NpmRequestTypeSearch
	// NpmRequestTypeScopedMetadata is scoped package metadata
	NpmRequestTypeScopedMetadata
)

// NpmService handles NPM repository proxy requests
type NpmService struct {
	*BaseProxyService
	config     *config.NpmProxySettings
	httpClient *http.Client
}

// NewNpmService creates a new NPM proxy service
func NewNpmService(
	cache CacheService,
	configService ConfigService,
	upstreamClient UpstreamClient,
) (*NpmService, error) {
	base := NewBaseProxyService("npm", cache, configService, upstreamClient)

	// Load NPM-specific configuration
	configInterface, err := configService.GetProxyConfig(context.Background(), "npm")
	if err != nil {
		return nil, fmt.Errorf("failed to load npm config: %w", err)
	}

	npmConfig, ok := configInterface.(*config.NpmProxySettings)
	if !ok {
		return nil, fmt.Errorf("invalid npm config type")
	}

	return &NpmService{
		BaseProxyService: base,
		config:           npmConfig,
		httpClient:       &http.Client{},
	}, nil
}

// HandleRequest processes an NPM proxy request
func (s *NpmService) HandleRequest(ctx context.Context, req ProxyRequest) (*ProxyResponse, error) {
	// Validate request
	if err := s.ValidateRequest(req); err != nil {
		return s.HandleError(err, http.StatusBadRequest), nil
	}

	// Determine request type
	reqType := s.parseRequestType(req.Path)

	switch reqType {
	case NpmRequestTypeMetadata, NpmRequestTypeScopedMetadata:
		return s.handleMetadata(ctx, req)
	case NpmRequestTypeTarball:
		return s.handleTarball(ctx, req)
	case NpmRequestTypeSearch:
		return s.handleSearch(ctx, req)
	default:
		return s.HandleError(fmt.Errorf("unknown npm request type"), http.StatusBadRequest), nil
	}
}

// parseRequestType determines the NPM request type from path
func (s *NpmService) parseRequestType(path string) NpmRequestType {
	// Normalize path
	path = strings.TrimPrefix(path, "/")

	// Search request: /-/v1/search?text=query
	if strings.HasPrefix(path, "-/v1/search") || strings.HasPrefix(path, "-/all") {
		return NpmRequestTypeSearch
	}

	// Tarball request: <package>/-/<tarball>.tgz or @scope/<package>/-/<tarball>.tgz
	if strings.Contains(path, "/-/") && (strings.HasSuffix(path, ".tgz") || strings.HasSuffix(path, ".tar.gz")) {
		return NpmRequestTypeTarball
	}

	// Scoped package metadata: @scope/package
	if strings.HasPrefix(path, "@") && !strings.Contains(path, "/-/") {
		parts := strings.Split(path, "/")
		if len(parts) == 2 {
			return NpmRequestTypeScopedMetadata
		}
	}

	// Regular package metadata: package
	parts := strings.Split(path, "/")
	if len(parts) == 1 && parts[0] != "" && !strings.HasPrefix(parts[0], "-") {
		return NpmRequestTypeMetadata
	}

	return NpmRequestTypeUnknown
}

// handleMetadata handles package metadata requests
func (s *NpmService) handleMetadata(ctx context.Context, req ProxyRequest) (*ProxyResponse, error) {
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
			headers := map[string]string{
				"Content-Type": "application/json; charset=utf-8",
			}

			return &ProxyResponse{
				Body:        io.NopCloser(bytes.NewReader(data)),
				StatusCode:  http.StatusOK,
				Headers:     headers,
				ContentType: "application/json; charset=utf-8",
				Cached:      true,
			}, nil
		}
	}

	// Fetch from upstream
	return s.fetchFromUpstream(ctx, req, cacheKey, true)
}

// handleTarball handles tarball download requests
func (s *NpmService) handleTarball(ctx context.Context, req ProxyRequest) (*ProxyResponse, error) {
	cacheKey := s.BuildCacheKey(req.Path)

	// Try cache first
	content, found, err := s.TryCache(ctx, cacheKey)
	if err != nil {
		s.Logger.Warn("Cache lookup failed", logging.F("error", err))
	}

	if found && content != nil {
		headers := map[string]string{
			"Content-Type": "application/octet-stream",
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

// handleSearch handles search requests
func (s *NpmService) handleSearch(ctx context.Context, req ProxyRequest) (*ProxyResponse, error) {
	// Search results are dynamic, don't cache long
	return s.fetchFromUpstream(ctx, req, "", false)
}

// fetchFromUpstream fetches content from upstream NPM registry
func (s *NpmService) fetchFromUpstream(ctx context.Context, req ProxyRequest, cacheKey string, isMetadata bool) (*ProxyResponse, error) {
	if s.config == nil || len(s.config.Proxies) == 0 {
		return s.HandleError(fmt.Errorf("no npm proxies configured"), http.StatusServiceUnavailable), nil
	}

	// Get default proxies
	proxies, ok := s.config.Proxies["default"]
	if !ok || len(proxies) == 0 {
		return s.HandleError(fmt.Errorf("no default npm proxies configured"), http.StatusServiceUnavailable), nil
	}

	for _, proxy := range proxies {
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
		if proxy.BasicAuth.Username != "" && proxy.BasicAuth.Password != "" {
			httpReq.SetBasicAuth(proxy.BasicAuth.Username, proxy.BasicAuth.Password)
		}

		// Execute request
		resp, err := s.httpClient.Do(httpReq)
		if err != nil {
			s.Logger.Warn("Failed to fetch from upstream",
				logging.F("proxy", proxy.Name),
				logging.F("error", err))
			continue
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

			// Rewrite metadata URLs if needed
			if isMetadata {
				data = s.rewriteMetadataURLs(data, req)
			}

			// Cache the response if cacheKey is provided
			if cacheKey != "" {
				if err := s.CacheResponse(ctx, cacheKey, bytes.NewReader(data)); err != nil {
					s.Logger.Warn("Failed to cache response", logging.F("error", err))
				}
			}

			// Build response headers
			headers := make(map[string]string)

			// Copy relevant headers from upstream
			for _, h := range []string{"Content-Length", "ETag", "Last-Modified"} {
				if v := resp.Header.Get(h); v != "" {
					headers[h] = v
				}
			}

			// Determine content type
			contentType := "application/octet-stream"
			if isMetadata {
				contentType = "application/json; charset=utf-8"
			} else if strings.HasSuffix(req.Path, ".tgz") || strings.HasSuffix(req.Path, ".tar.gz") {
				contentType = "application/x-gzip"
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

	return s.HandleNotFound("package not found in any upstream registry"), nil
}

// buildUpstreamURL constructs the upstream NPM registry URL
func (s *NpmService) buildUpstreamURL(serverURL, requestPath string) string {
	// Remove leading slash from path
	requestPath = strings.TrimPrefix(requestPath, "/")

	// Remove trailing slash from server URL
	serverURL = strings.TrimSuffix(serverURL, "/")

	return serverURL + "/" + requestPath
}

// copyRequestHeaders copies relevant headers from the original request
func (s *NpmService) copyRequestHeaders(req ProxyRequest, httpReq *http.Request) {
	if req.Headers != nil {
		for _, h := range []string{"Accept", "Authorization", "User-Agent", "Accept-Encoding"} {
			if v, ok := req.Headers[h]; ok && v != "" {
				httpReq.Header.Set(h, v)
			}
		}
	}

	// Set default Accept for JSON responses
	if httpReq.Header.Get("Accept") == "" {
		httpReq.Header.Set("Accept", "application/json")
	}
}

// rewriteMetadataURLs rewrites tarball URLs in package metadata to point to proxy
func (s *NpmService) rewriteMetadataURLs(data []byte, req ProxyRequest) []byte {
	var metadata map[string]interface{}
	if err := json.Unmarshal(data, &metadata); err != nil {
		return data
	}

	// Get base URL from request if available
	baseURL := ""
	if req.Headers != nil {
		if host, ok := req.Headers["Host"]; ok && host != "" {
			scheme := "http"
			if forwarded, ok := req.Headers["X-Forwarded-Proto"]; ok && forwarded != "" {
				scheme = forwarded
			}
			baseURL = scheme + "://" + host
		}
	}

	if baseURL == "" {
		// No rewriting if we don't have base URL
		return data
	}

	// Rewrite tarball URLs in versions
	if versions, ok := metadata["versions"].(map[string]interface{}); ok {
		for _, versionData := range versions {
			if version, ok := versionData.(map[string]interface{}); ok {
				if dist, ok := version["dist"].(map[string]interface{}); ok {
					if tarball, ok := dist["tarball"].(string); ok {
						// Extract package path from tarball URL
						packagePath := s.extractPackagePath(tarball)
						if packagePath != "" {
							dist["tarball"] = baseURL + "/npm/" + packagePath
						}
					}
				}
			}
		}
	}

	rewritten, err := json.Marshal(metadata)
	if err != nil {
		return data
	}
	return rewritten
}

// extractPackagePath extracts the package path from a tarball URL
func (s *NpmService) extractPackagePath(tarballURL string) string {
	// Handle various registry URL formats
	// https://registry.npmjs.org/lodash/-/lodash-4.17.21.tgz
	// https://registry.yarnpkg.com/lodash/-/lodash-4.17.21.tgz

	registries := []string{
		"registry.npmjs.org/",
		"registry.yarnpkg.com/",
	}

	for _, registry := range registries {
		if idx := strings.Index(tarballURL, registry); idx != -1 {
			return tarballURL[idx+len(registry):]
		}
	}

	// Try generic pattern: extract everything after the last known registry pattern
	if idx := strings.LastIndex(tarballURL, "://"); idx != -1 {
		rest := tarballURL[idx+3:]
		if slashIdx := strings.Index(rest, "/"); slashIdx != -1 {
			return rest[slashIdx+1:]
		}
	}

	return ""
}

// SupportsHead returns whether the service supports HEAD requests
func (s *NpmService) SupportsHead() bool {
	return true
}

// HandleHead handles HEAD requests for package existence checks
func (s *NpmService) HandleHead(ctx context.Context, req ProxyRequest) (*ProxyResponse, error) {
	// Check cache first
	cacheKey := s.BuildCacheKey(req.Path)
	exists, err := s.Cache.Exists(ctx, cacheKey)
	if err != nil {
		s.Logger.Warn("Cache existence check failed", logging.F("error", err))
	}

	if exists {
		headers := map[string]string{
			"Content-Type": "application/json",
		}

		return &ProxyResponse{
			Body:        io.NopCloser(bytes.NewReader(nil)),
			StatusCode:  http.StatusOK,
			Headers:     headers,
			ContentType: "application/json",
			Cached:      true,
		}, nil
	}

	// Check upstream
	if s.config == nil || len(s.config.Proxies) == 0 {
		return s.HandleNotFound("no proxies configured"), nil
	}

	proxies, ok := s.config.Proxies["default"]
	if !ok || len(proxies) == 0 {
		return s.HandleNotFound("no default proxies configured"), nil
	}

	for _, proxy := range proxies {
		url := s.buildUpstreamURL(proxy.URL, req.Path)

		httpReq, err := http.NewRequestWithContext(ctx, "HEAD", url, nil)
		if err != nil {
			continue
		}

		s.copyRequestHeaders(req, httpReq)

		if proxy.BasicAuth.Username != "" && proxy.BasicAuth.Password != "" {
			httpReq.SetBasicAuth(proxy.BasicAuth.Username, proxy.BasicAuth.Password)
		}

		resp, err := s.httpClient.Do(httpReq)
		if err != nil {
			continue
		}
		_ = resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			headers := make(map[string]string)

			for _, h := range []string{"Content-Length", "Content-Type", "ETag", "Last-Modified"} {
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

	return s.HandleNotFound("package not found"), nil
}
