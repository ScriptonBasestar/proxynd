// Package proxy provides proxy service implementations
package proxy

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"proxynd/internal/config"
	"proxynd/internal/logging"
)

// YumRequestType represents the type of YUM repository request
type YumRequestType int

const (
	// YumRequestTypeUnknown is unknown request type
	YumRequestTypeUnknown YumRequestType = iota
	// YumRequestTypeRepoMD is repomd.xml metadata request
	YumRequestTypeRepoMD
	// YumRequestTypePrimary is primary.xml metadata request
	YumRequestTypePrimary
	// YumRequestTypeFilelists is filelists.xml metadata request
	YumRequestTypeFilelists
	// YumRequestTypeOther is other.xml metadata request
	YumRequestTypeOther
	// YumRequestTypeComps is comps.xml (group) metadata request
	YumRequestTypeComps
	// YumRequestTypeRPM is .rpm package request
	YumRequestTypeRPM
	// YumRequestTypeDRPM is .drpm delta rpm request
	YumRequestTypeDRPM
	// YumRequestTypeModules is modules.yaml metadata request
	YumRequestTypeModules
)

// YUM content types
const (
	YumContentTypeXML     = "application/xml"
	YumContentTypeGzip    = "application/gzip"
	YumContentTypeXz      = "application/x-xz"
	YumContentTypeBz2     = "application/x-bzip2"
	YumContentTypeRPM     = "application/x-rpm"
	YumContentTypeDefault = "application/octet-stream"
)

// YumService handles YUM repository proxy requests
type YumService struct {
	*BaseProxyService
	config     *config.YumProxySettings
	httpClient *http.Client
}

// NewYumService creates a new YUM proxy service
func NewYumService(
	cache CacheService,
	configService ConfigService,
	upstreamClient UpstreamClient,
) (*YumService, error) {
	base := NewBaseProxyService("yum", cache, configService, upstreamClient)

	// Load YUM-specific configuration
	configInterface, err := configService.GetProxyConfig(context.Background(), "yum")
	if err != nil {
		return nil, fmt.Errorf("failed to load yum config: %w", err)
	}

	yumConfig, ok := configInterface.(*config.YumProxySettings)
	if !ok {
		return nil, fmt.Errorf("invalid yum config type")
	}

	return &YumService{
		BaseProxyService: base,
		config:           yumConfig,
		httpClient:       &http.Client{},
	}, nil
}

// HandleRequest processes a YUM proxy request
func (s *YumService) HandleRequest(ctx context.Context, req ProxyRequest) (*ProxyResponse, error) {
	// Validate request
	if err := s.ValidateRequest(req); err != nil {
		return s.HandleError(err, http.StatusBadRequest), nil
	}

	// Determine request type
	reqType := s.classifyRequest(req.Path)

	s.Logger.Debug("YUM request classified",
		logging.F("path", req.Path),
		logging.F("type", reqType))

	switch reqType {
	case YumRequestTypeRepoMD:
		return s.handleRepoMD(ctx, req)
	case YumRequestTypePrimary, YumRequestTypeFilelists, YumRequestTypeOther, YumRequestTypeComps, YumRequestTypeModules:
		return s.handleMetadata(ctx, req)
	case YumRequestTypeRPM, YumRequestTypeDRPM:
		return s.handlePackage(ctx, req)
	default:
		return s.handleGenericRequest(ctx, req)
	}
}

// classifyRequest determines the type of YUM request
func (s *YumService) classifyRequest(requestPath string) YumRequestType {
	// Normalize path
	path := strings.TrimPrefix(requestPath, "/")
	filename := strings.ToLower(filepath.Base(path))
	lowerPath := strings.ToLower(path)

	// repomd.xml - repository metadata descriptor
	if filename == "repomd.xml" {
		return YumRequestTypeRepoMD
	}

	// Primary metadata (package info)
	if strings.HasPrefix(filename, "primary") {
		return YumRequestTypePrimary
	}

	// Filelists metadata
	if strings.HasPrefix(filename, "filelists") {
		return YumRequestTypeFilelists
	}

	// Other metadata (changelog)
	if strings.HasPrefix(filename, "other") {
		return YumRequestTypeOther
	}

	// Comps/group metadata
	if strings.Contains(filename, "comps") || strings.Contains(filename, "group") {
		return YumRequestTypeComps
	}

	// Modules metadata (DNF modularity)
	if strings.HasPrefix(filename, "modules") {
		return YumRequestTypeModules
	}

	// RPM package
	if strings.HasSuffix(lowerPath, ".rpm") {
		return YumRequestTypeRPM
	}

	// Delta RPM
	if strings.HasSuffix(lowerPath, ".drpm") {
		return YumRequestTypeDRPM
	}

	return YumRequestTypeUnknown
}

// handleRepoMD handles repomd.xml metadata requests
func (s *YumService) handleRepoMD(ctx context.Context, req ProxyRequest) (*ProxyResponse, error) {
	cacheKey := s.BuildCacheKey(req.Path)

	// Try cache first (short TTL for repomd.xml as it changes frequently)
	content, found, err := s.TryCache(ctx, cacheKey)
	if err != nil {
		s.Logger.Warn("Cache lookup failed", logging.F("error", err))
	}

	if found && content != nil {
		data, err := io.ReadAll(content)
		_ = content.Close()
		if err == nil {
			headers := map[string]string{
				"Content-Type":   YumContentTypeXML,
				"X-Cache-Status": "HIT",
			}

			return &ProxyResponse{
				Body:        io.NopCloser(bytes.NewReader(data)),
				StatusCode:  http.StatusOK,
				Headers:     headers,
				ContentType: YumContentTypeXML,
				Cached:      true,
			}, nil
		}
	}

	// Fetch from upstream
	return s.fetchFromUpstream(ctx, req, cacheKey)
}

// handleMetadata handles compressed metadata requests (primary, filelists, other, comps, modules)
func (s *YumService) handleMetadata(ctx context.Context, req ProxyRequest) (*ProxyResponse, error) {
	cacheKey := s.BuildCacheKey(req.Path)

	// Try cache first
	content, found, err := s.TryCache(ctx, cacheKey)
	if err != nil {
		s.Logger.Warn("Cache lookup failed", logging.F("error", err))
	}

	if found && content != nil {
		contentType := s.getContentType(req.Path)
		headers := map[string]string{
			"Content-Type":   contentType,
			"X-Cache-Status": "HIT",
		}

		return &ProxyResponse{
			Body:        content,
			StatusCode:  http.StatusOK,
			Headers:     headers,
			ContentType: contentType,
			Cached:      true,
		}, nil
	}

	// Fetch from upstream
	return s.fetchFromUpstream(ctx, req, cacheKey)
}

// handlePackage handles RPM/DRPM package download requests
func (s *YumService) handlePackage(ctx context.Context, req ProxyRequest) (*ProxyResponse, error) {
	cacheKey := s.BuildCacheKey(req.Path)

	// Try cache first (packages have long TTL)
	content, found, err := s.TryCache(ctx, cacheKey)
	if err != nil {
		s.Logger.Warn("Cache lookup failed", logging.F("error", err))
	}

	if found && content != nil {
		headers := map[string]string{
			"Content-Type":   YumContentTypeRPM,
			"X-Cache-Status": "HIT",
		}

		return &ProxyResponse{
			Body:        content,
			StatusCode:  http.StatusOK,
			Headers:     headers,
			ContentType: YumContentTypeRPM,
			Cached:      true,
		}, nil
	}

	// Fetch from upstream
	return s.fetchFromUpstream(ctx, req, cacheKey)
}

// handleGenericRequest handles unknown request types
func (s *YumService) handleGenericRequest(ctx context.Context, req ProxyRequest) (*ProxyResponse, error) {
	cacheKey := s.BuildCacheKey(req.Path)

	// Try cache first
	content, found, err := s.TryCache(ctx, cacheKey)
	if err != nil {
		s.Logger.Warn("Cache lookup failed", logging.F("error", err))
	}

	if found && content != nil {
		contentType := s.getContentType(req.Path)
		headers := map[string]string{
			"Content-Type":   contentType,
			"X-Cache-Status": "HIT",
		}

		return &ProxyResponse{
			Body:        content,
			StatusCode:  http.StatusOK,
			Headers:     headers,
			ContentType: contentType,
			Cached:      true,
		}, nil
	}

	// Fetch from upstream
	return s.fetchFromUpstream(ctx, req, cacheKey)
}

// fetchFromUpstream fetches content from upstream YUM repository
func (s *YumService) fetchFromUpstream(ctx context.Context, req ProxyRequest, cacheKey string) (*ProxyResponse, error) {
	if s.config == nil || len(s.config.Proxies) == 0 {
		return s.HandleError(fmt.Errorf("no yum proxies configured"), http.StatusServiceUnavailable), nil
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
			contentType := s.getContentType(req.Path)
			headers["Content-Type"] = contentType
			headers["X-Cache-Status"] = "MISS"

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

	return s.HandleNotFound("package not found in any upstream repository"), nil
}

// buildUpstreamURL constructs the upstream YUM repository URL
func (s *YumService) buildUpstreamURL(serverURL, requestPath string) string {
	// Remove leading slash from path
	requestPath = strings.TrimPrefix(requestPath, "/")

	// Remove trailing slash from server URL
	serverURL = strings.TrimSuffix(serverURL, "/")

	return serverURL + "/" + requestPath
}

// copyRequestHeaders copies relevant headers from the original request
func (s *YumService) copyRequestHeaders(req ProxyRequest, httpReq *http.Request) {
	if req.Headers != nil {
		for _, h := range []string{"Accept", "Accept-Encoding", "User-Agent", "If-Modified-Since", "If-None-Match"} {
			if v, ok := req.Headers[h]; ok && v != "" {
				httpReq.Header.Set(h, v)
			}
		}
	}

	// Set default User-Agent
	if httpReq.Header.Get("User-Agent") == "" {
		httpReq.Header.Set("User-Agent", "ProxyND/1.0 YUM-Proxy")
	}

	// Accept compressed content
	if httpReq.Header.Get("Accept-Encoding") == "" {
		httpReq.Header.Set("Accept-Encoding", "gzip, deflate")
	}
}

// getContentType determines the content type based on file path
func (s *YumService) getContentType(requestPath string) string {
	lowerPath := strings.ToLower(requestPath)
	ext := strings.ToLower(filepath.Ext(requestPath))

	switch {
	case strings.HasSuffix(lowerPath, ".rpm"):
		return YumContentTypeRPM
	case strings.HasSuffix(lowerPath, ".drpm"):
		return YumContentTypeRPM
	case strings.HasSuffix(lowerPath, ".gz"):
		return YumContentTypeGzip
	case strings.HasSuffix(lowerPath, ".xz"):
		return YumContentTypeXz
	case strings.HasSuffix(lowerPath, ".bz2"):
		return YumContentTypeBz2
	case ext == ".xml" || strings.HasSuffix(lowerPath, "repomd.xml"):
		return YumContentTypeXML
	default:
		return YumContentTypeDefault
	}
}

// SupportsHead returns whether the service supports HEAD requests
func (s *YumService) SupportsHead() bool {
	return true
}

// HandleHead handles HEAD requests for file existence checks
func (s *YumService) HandleHead(ctx context.Context, req ProxyRequest) (*ProxyResponse, error) {
	// Check cache first
	cacheKey := s.BuildCacheKey(req.Path)
	exists, err := s.Cache.Exists(ctx, cacheKey)
	if err != nil {
		s.Logger.Warn("Cache existence check failed", logging.F("error", err))
	}

	if exists {
		contentType := s.getContentType(req.Path)
		headers := map[string]string{
			"Content-Type": contentType,
		}

		return &ProxyResponse{
			Body:        io.NopCloser(bytes.NewReader(nil)),
			StatusCode:  http.StatusOK,
			Headers:     headers,
			ContentType: contentType,
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

// GetRequestType returns the request type for a given path (for external use)
func (s *YumService) GetRequestType(requestPath string) YumRequestType {
	return s.classifyRequest(requestPath)
}

// GetConfig returns the YUM configuration
func (s *YumService) GetConfig() *config.YumProxySettings {
	return s.config
}
