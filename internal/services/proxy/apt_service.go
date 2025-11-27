// Package proxy provides proxy service implementations
package proxy

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"

	"proxynd/internal/config"
	"proxynd/internal/logging"
)

// AptRequestType represents the type of APT repository request
type AptRequestType int

const (
	// AptRequestTypeUnknown is unknown request type
	AptRequestTypeUnknown AptRequestType = iota
	// AptRequestTypeInRelease is InRelease file request
	AptRequestTypeInRelease
	// AptRequestTypeRelease is Release file request
	AptRequestTypeRelease
	// AptRequestTypeReleaseGPG is Release.gpg signature request
	AptRequestTypeReleaseGPG
	// AptRequestTypePackages is Packages index request
	AptRequestTypePackages
	// AptRequestTypeSources is Sources index request
	AptRequestTypeSources
	// AptRequestTypeDebPackage is .deb package request
	AptRequestTypeDebPackage
	// AptRequestTypeContents is Contents file request
	AptRequestTypeContents
)

// APT content types
const (
	AptContentTypeDebian   = "application/vnd.debian.binary-package"
	AptContentTypeGzip     = "application/gzip"
	AptContentTypeXz       = "application/x-xz"
	AptContentTypeBz2      = "application/x-bzip2"
	AptContentTypePlain    = "text/plain; charset=utf-8"
	AptContentTypeGPG      = "application/pgp-signature"
	AptContentTypeDefault  = "application/octet-stream"
)

// AptService handles APT repository proxy requests
type AptService struct {
	*BaseProxyService
	config     *config.AptProxyConfig
	httpClient *http.Client
}

// NewAptService creates a new APT proxy service
func NewAptService(
	cache CacheService,
	configService ConfigService,
	upstreamClient UpstreamClient,
) (*AptService, error) {
	base := NewBaseProxyService("apt", cache, configService, upstreamClient)

	// Load APT-specific configuration
	configInterface, err := configService.GetProxyConfig(context.Background(), "apt")
	if err != nil {
		return nil, fmt.Errorf("failed to load apt config: %w", err)
	}

	aptConfig, ok := configInterface.(*config.AptProxyConfig)
	if !ok {
		return nil, fmt.Errorf("invalid apt config type")
	}

	return &AptService{
		BaseProxyService: base,
		config:           aptConfig,
		httpClient:       &http.Client{},
	}, nil
}

// HandleRequest processes an APT proxy request
func (s *AptService) HandleRequest(ctx context.Context, req ProxyRequest) (*ProxyResponse, error) {
	// Validate request
	if err := s.ValidateRequest(req); err != nil {
		return s.HandleError(err, http.StatusBadRequest), nil
	}

	// Determine request type
	reqType := s.classifyRequest(req.Path)

	s.Logger.Debug("APT request classified",
		logging.F("path", req.Path),
		logging.F("type", reqType))

	switch reqType {
	case AptRequestTypeInRelease, AptRequestTypeRelease, AptRequestTypeReleaseGPG:
		return s.handleReleaseRequest(ctx, req)
	case AptRequestTypePackages, AptRequestTypeSources, AptRequestTypeContents:
		return s.handleIndexRequest(ctx, req)
	case AptRequestTypeDebPackage:
		return s.handleDebPackageRequest(ctx, req)
	default:
		return s.handleGenericRequest(ctx, req)
	}
}

// classifyRequest determines the type of APT request
func (s *AptService) classifyRequest(requestPath string) AptRequestType {
	filename := path.Base(requestPath)
	lowerPath := strings.ToLower(requestPath)

	switch {
	case filename == "InRelease":
		return AptRequestTypeInRelease
	case filename == "Release":
		return AptRequestTypeRelease
	case filename == "Release.gpg":
		return AptRequestTypeReleaseGPG
	case strings.HasPrefix(filename, "Packages"):
		return AptRequestTypePackages
	case strings.HasPrefix(filename, "Sources"):
		return AptRequestTypeSources
	case strings.HasSuffix(lowerPath, ".deb") || strings.HasSuffix(lowerPath, ".udeb"):
		return AptRequestTypeDebPackage
	case strings.HasPrefix(filename, "Contents"):
		return AptRequestTypeContents
	default:
		return AptRequestTypeUnknown
	}
}

// handleReleaseRequest handles InRelease, Release, Release.gpg requests
func (s *AptService) handleReleaseRequest(ctx context.Context, req ProxyRequest) (*ProxyResponse, error) {
	cacheKey := s.BuildCacheKey(req.Path)

	// Try cache first
	content, found, err := s.TryCache(ctx, cacheKey)
	if err != nil {
		s.Logger.Warn("Cache lookup failed", logging.F("error", err))
	}

	if found && content != nil {
		data, err := io.ReadAll(content)
		_ = content.Close()
		if err == nil {
			contentType := s.getContentType(req.Path)
			headers := map[string]string{
				"Content-Type":   contentType,
				"X-Cache-Status": "HIT",
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
	return s.fetchFromUpstream(ctx, req, cacheKey)
}

// handleIndexRequest handles Packages, Sources, Contents index requests
func (s *AptService) handleIndexRequest(ctx context.Context, req ProxyRequest) (*ProxyResponse, error) {
	cacheKey := s.BuildCacheKey(req.Path)

	// Try cache first
	content, found, err := s.TryCache(ctx, cacheKey)
	if err != nil {
		s.Logger.Warn("Cache lookup failed", logging.F("error", err))
	}

	if found && content != nil {
		data, err := io.ReadAll(content)
		_ = content.Close()
		if err == nil {
			contentType := s.getContentType(req.Path)
			headers := map[string]string{
				"Content-Type":   contentType,
				"X-Cache-Status": "HIT",
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
	return s.fetchFromUpstream(ctx, req, cacheKey)
}

// handleDebPackageRequest handles .deb package download requests
func (s *AptService) handleDebPackageRequest(ctx context.Context, req ProxyRequest) (*ProxyResponse, error) {
	cacheKey := s.BuildCacheKey(req.Path)

	// Try cache first (deb packages have long TTL)
	content, found, err := s.TryCache(ctx, cacheKey)
	if err != nil {
		s.Logger.Warn("Cache lookup failed", logging.F("error", err))
	}

	if found && content != nil {
		headers := map[string]string{
			"Content-Type":   AptContentTypeDebian,
			"X-Cache-Status": "HIT",
		}

		return &ProxyResponse{
			Body:        content,
			StatusCode:  http.StatusOK,
			Headers:     headers,
			ContentType: AptContentTypeDebian,
			Cached:      true,
		}, nil
	}

	// Fetch from upstream
	return s.fetchFromUpstream(ctx, req, cacheKey)
}

// handleGenericRequest handles unknown request types
func (s *AptService) handleGenericRequest(ctx context.Context, req ProxyRequest) (*ProxyResponse, error) {
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

// fetchFromUpstream fetches content from upstream APT repository
func (s *AptService) fetchFromUpstream(ctx context.Context, req ProxyRequest, cacheKey string) (*ProxyResponse, error) {
	if s.config == nil || len(s.config.Proxies) == 0 {
		return s.HandleError(fmt.Errorf("no apt proxies configured"), http.StatusServiceUnavailable), nil
	}

	// Extract OS type from path (e.g., ubuntu, debian)
	osType := s.extractOSType(req.Path)

	// Get proxies for OS type, fallback to default
	proxies, ok := s.config.Proxies[osType]
	if !ok || len(proxies) == 0 {
		proxies, ok = s.config.Proxies["default"]
		if !ok || len(proxies) == 0 {
			// Try first available
			for _, p := range s.config.Proxies {
				proxies = p
				break
			}
		}
	}

	if len(proxies) == 0 {
		return s.HandleError(fmt.Errorf("no apt proxies configured for %s", osType), http.StatusServiceUnavailable), nil
	}

	// Extract the path portion after OS type
	requestPath := s.extractRequestPath(req.Path, osType)

	for _, proxy := range proxies {
		// Build upstream URL
		url := s.buildUpstreamURL(proxy.URL, requestPath)

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

				// Parse Packages.gz for metadata extraction
				if strings.HasSuffix(req.Path, "Packages.gz") {
					s.parsePackagesIndex(data)
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
			headers["X-Cache-Status"] = cacheStatusMiss

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

	return s.HandleNotFound("resource not found in any upstream repository"), nil
}

// buildUpstreamURL constructs the upstream APT repository URL
func (s *AptService) buildUpstreamURL(serverURL, requestPath string) string {
	// Remove leading slash from path
	requestPath = strings.TrimPrefix(requestPath, "/")

	// Remove trailing slash from server URL
	serverURL = strings.TrimSuffix(serverURL, "/")

	return serverURL + "/" + requestPath
}

// copyRequestHeaders copies relevant headers from the original request
func (s *AptService) copyRequestHeaders(req ProxyRequest, httpReq *http.Request) {
	if req.Headers != nil {
		for _, h := range []string{"Accept", "Accept-Encoding", "User-Agent", "If-Modified-Since", "If-None-Match"} {
			if v, ok := req.Headers[h]; ok && v != "" {
				httpReq.Header.Set(h, v)
			}
		}
	}

	// Set default User-Agent
	if httpReq.Header.Get("User-Agent") == "" {
		httpReq.Header.Set("User-Agent", "ProxyND/1.0 APT-Proxy")
	}

	// Accept compressed content
	if httpReq.Header.Get("Accept-Encoding") == "" {
		httpReq.Header.Set("Accept-Encoding", "gzip, deflate")
	}
}

// extractOSType extracts the OS type from the request path
func (s *AptService) extractOSType(requestPath string) string {
	// Normalize path
	requestPath = strings.TrimPrefix(requestPath, "/")
	parts := strings.Split(requestPath, "/")

	if len(parts) > 0 {
		// Check if first part is a known OS type
		osType := strings.ToLower(parts[0])
		switch osType {
		case "ubuntu", "debian", "raspbian", "linuxmint":
			return osType
		}
	}

	return "default"
}

// extractRequestPath extracts the request path after the OS type
func (s *AptService) extractRequestPath(requestPath, osType string) string {
	requestPath = strings.TrimPrefix(requestPath, "/")

	// If path starts with OS type, remove it
	if osType != "default" && strings.HasPrefix(strings.ToLower(requestPath), osType+"/") {
		requestPath = requestPath[len(osType)+1:]
	}

	return requestPath
}

// getContentType determines the content type based on file path
func (s *AptService) getContentType(requestPath string) string {
	lowerPath := strings.ToLower(requestPath)

	switch {
	case strings.HasSuffix(lowerPath, ".deb") || strings.HasSuffix(lowerPath, ".udeb"):
		return AptContentTypeDebian
	case strings.HasSuffix(lowerPath, ".gz"):
		return AptContentTypeGzip
	case strings.HasSuffix(lowerPath, ".xz"):
		return AptContentTypeXz
	case strings.HasSuffix(lowerPath, ".bz2"):
		return AptContentTypeBz2
	case strings.HasSuffix(lowerPath, ".gpg") || strings.HasSuffix(lowerPath, ".sig"):
		return AptContentTypeGPG
	case strings.Contains(lowerPath, "release") ||
		strings.Contains(lowerPath, "packages") ||
		strings.Contains(lowerPath, "sources") ||
		strings.Contains(lowerPath, "contents"):
		return AptContentTypePlain
	default:
		return AptContentTypeDefault
	}
}

// parsePackagesIndex parses a gzipped Packages index for metadata extraction
func (s *AptService) parsePackagesIndex(data []byte) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		s.Logger.Debug("Failed to decompress Packages.gz", logging.F("error", err))
		return
	}
	defer func() { _ = reader.Close() }()

	content, err := io.ReadAll(reader)
	if err != nil {
		s.Logger.Debug("Failed to read Packages content", logging.F("error", err))
		return
	}

	// Count packages (simple parsing)
	packageCount := strings.Count(string(content), "\nPackage: ")
	s.Logger.Debug("Parsed Packages index", logging.F("package_count", packageCount))
}

// SupportsHead returns whether the service supports HEAD requests
func (s *AptService) SupportsHead() bool {
	return true
}

// HandleHead handles HEAD requests for file existence checks
func (s *AptService) HandleHead(ctx context.Context, req ProxyRequest) (*ProxyResponse, error) {
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

	osType := s.extractOSType(req.Path)
	proxies, ok := s.config.Proxies[osType]
	if !ok || len(proxies) == 0 {
		proxies, ok = s.config.Proxies["default"]
		if !ok || len(proxies) == 0 {
			return s.HandleNotFound("no proxies configured"), nil
		}
	}

	requestPath := s.extractRequestPath(req.Path, osType)

	for _, proxy := range proxies {
		url := s.buildUpstreamURL(proxy.URL, requestPath)

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

	return s.HandleNotFound("file not found"), nil
}

// GetRequestType returns the request type for a given path (for external use)
func (s *AptService) GetRequestType(requestPath string) AptRequestType {
	return s.classifyRequest(requestPath)
}

// GetConfig returns the APT configuration
func (s *AptService) GetConfig() *config.AptProxyConfig {
	return s.config
}
