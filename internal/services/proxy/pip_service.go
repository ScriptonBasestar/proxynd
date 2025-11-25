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

// PipRequestType represents the type of PyPI repository request
type PipRequestType int

const (
	// PipRequestTypeUnknown is unknown request type
	PipRequestTypeUnknown PipRequestType = iota
	// PipRequestTypeSimpleIndex is simple index request (/)
	PipRequestTypeSimpleIndex
	// PipRequestTypePackageIndex is package index request (/simple/<package>/)
	PipRequestTypePackageIndex
	// PipRequestTypePackageFile is package file request (.whl, .tar.gz, .egg)
	PipRequestTypePackageFile
	// PipRequestTypeJSON is JSON API request (/pypi/<package>/json)
	PipRequestTypeJSON
)

// PyPI content types
const (
	PipContentTypeHTML    = "text/html; charset=utf-8"
	PipContentTypeJSON    = "application/json"
	PipContentTypeWheel   = "application/octet-stream"
	PipContentTypeTarGz   = "application/gzip"
	PipContentTypeZip     = "application/zip"
	PipContentTypeEgg     = "application/zip"
	PipContentTypeDefault = "application/octet-stream"
)

// PipService handles PyPI repository proxy requests
type PipService struct {
	*BaseProxyService
	config     *config.PipProxySettings
	httpClient *http.Client
}

// NewPipService creates a new PyPI proxy service
func NewPipService(
	cache CacheService,
	configService ConfigService,
	upstreamClient UpstreamClient,
) (*PipService, error) {
	base := NewBaseProxyService("pip", cache, configService, upstreamClient)

	// Load PyPI-specific configuration
	configInterface, err := configService.GetProxyConfig(context.Background(), "pip")
	if err != nil {
		return nil, fmt.Errorf("failed to load pip config: %w", err)
	}

	pipConfig, ok := configInterface.(*config.PipProxySettings)
	if !ok {
		return nil, fmt.Errorf("invalid pip config type")
	}

	return &PipService{
		BaseProxyService: base,
		config:           pipConfig,
		httpClient:       &http.Client{},
	}, nil
}

// HandleRequest processes a PyPI proxy request
func (s *PipService) HandleRequest(ctx context.Context, req ProxyRequest) (*ProxyResponse, error) {
	// Validate request
	if err := s.ValidateRequest(req); err != nil {
		return s.HandleError(err, http.StatusBadRequest), nil
	}

	// Determine request type
	reqType := s.classifyRequest(req.Path)

	s.Logger.Debug("PyPI request classified",
		logging.F("path", req.Path),
		logging.F("type", reqType))

	switch reqType {
	case PipRequestTypeSimpleIndex:
		return s.handleSimpleIndex(ctx, req)
	case PipRequestTypePackageIndex:
		return s.handlePackageIndex(ctx, req)
	case PipRequestTypePackageFile:
		return s.handlePackageFile(ctx, req)
	case PipRequestTypeJSON:
		return s.handleJSONAPI(ctx, req)
	default:
		return s.handleGenericRequest(ctx, req)
	}
}

// classifyRequest determines the type of PyPI request
func (s *PipService) classifyRequest(requestPath string) PipRequestType {
	// Normalize path
	path := strings.TrimPrefix(requestPath, "/")
	path = strings.TrimSuffix(path, "/")

	// Root simple index
	if path == "" || path == "simple" {
		return PipRequestTypeSimpleIndex
	}

	// JSON API: /pypi/<package>/json or /pypi/<package>/<version>/json
	if strings.HasPrefix(path, "pypi/") && strings.HasSuffix(path, "/json") {
		return PipRequestTypeJSON
	}

	// Package file downloads (.whl, .tar.gz, .zip, .egg)
	lowerPath := strings.ToLower(path)
	if strings.HasSuffix(lowerPath, ".whl") ||
		strings.HasSuffix(lowerPath, ".tar.gz") ||
		strings.HasSuffix(lowerPath, ".zip") ||
		strings.HasSuffix(lowerPath, ".egg") ||
		strings.HasSuffix(lowerPath, ".tar.bz2") ||
		strings.HasSuffix(lowerPath, ".tgz") {
		return PipRequestTypePackageFile
	}

	// Simple package index: /simple/<package>/
	if strings.HasPrefix(path, "simple/") {
		return PipRequestTypePackageIndex
	}

	return PipRequestTypeUnknown
}

// handleSimpleIndex handles the root simple index request
func (s *PipService) handleSimpleIndex(ctx context.Context, req ProxyRequest) (*ProxyResponse, error) {
	cacheKey := s.BuildCacheKey("simple/index")

	// Try cache first
	content, found, err := s.TryCache(ctx, cacheKey)
	if err != nil {
		s.Logger.Warn("Cache lookup failed", logging.F("error", err))
	}

	if found && content != nil {
		data, err := io.ReadAll(content)
		_ = content.Close()
		if err == nil {
			headers := map[string]string{
				"Content-Type":   PipContentTypeHTML,
				"X-Cache-Status": "HIT",
			}

			return &ProxyResponse{
				Body:        io.NopCloser(bytes.NewReader(data)),
				StatusCode:  http.StatusOK,
				Headers:     headers,
				ContentType: PipContentTypeHTML,
				Cached:      true,
			}, nil
		}
	}

	// Fetch from upstream
	return s.fetchFromUpstream(ctx, req, cacheKey)
}

// handlePackageIndex handles package-specific simple index requests
func (s *PipService) handlePackageIndex(ctx context.Context, req ProxyRequest) (*ProxyResponse, error) {
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
			headers := map[string]string{
				"Content-Type":   PipContentTypeHTML,
				"X-Cache-Status": "HIT",
			}

			return &ProxyResponse{
				Body:        io.NopCloser(bytes.NewReader(data)),
				StatusCode:  http.StatusOK,
				Headers:     headers,
				ContentType: PipContentTypeHTML,
				Cached:      true,
			}, nil
		}
	}

	// Fetch from upstream
	return s.fetchFromUpstream(ctx, req, cacheKey)
}

// handlePackageFile handles package file downloads (.whl, .tar.gz, etc.)
func (s *PipService) handlePackageFile(ctx context.Context, req ProxyRequest) (*ProxyResponse, error) {
	cacheKey := s.BuildCacheKey(req.Path)

	// Try cache first (package files have long TTL)
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

// handleJSONAPI handles JSON API requests (/pypi/<package>/json)
func (s *PipService) handleJSONAPI(ctx context.Context, req ProxyRequest) (*ProxyResponse, error) {
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
			headers := map[string]string{
				"Content-Type":   PipContentTypeJSON,
				"X-Cache-Status": "HIT",
			}

			return &ProxyResponse{
				Body:        io.NopCloser(bytes.NewReader(data)),
				StatusCode:  http.StatusOK,
				Headers:     headers,
				ContentType: PipContentTypeJSON,
				Cached:      true,
			}, nil
		}
	}

	// Fetch from upstream
	return s.fetchFromUpstream(ctx, req, cacheKey)
}

// handleGenericRequest handles unknown request types
func (s *PipService) handleGenericRequest(ctx context.Context, req ProxyRequest) (*ProxyResponse, error) {
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

// fetchFromUpstream fetches content from upstream PyPI repository
func (s *PipService) fetchFromUpstream(ctx context.Context, req ProxyRequest, cacheKey string) (*ProxyResponse, error) {
	if s.config == nil || len(s.config.Proxies) == 0 {
		return s.HandleError(fmt.Errorf("no pip proxies configured"), http.StatusServiceUnavailable), nil
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

// buildUpstreamURL constructs the upstream PyPI repository URL
func (s *PipService) buildUpstreamURL(serverURL, requestPath string) string {
	// Remove leading slash from path
	requestPath = strings.TrimPrefix(requestPath, "/")

	// Remove trailing slash from server URL
	serverURL = strings.TrimSuffix(serverURL, "/")

	return serverURL + "/" + requestPath
}

// copyRequestHeaders copies relevant headers from the original request
func (s *PipService) copyRequestHeaders(req ProxyRequest, httpReq *http.Request) {
	if req.Headers != nil {
		for _, h := range []string{"Accept", "Accept-Encoding", "User-Agent"} {
			if v, ok := req.Headers[h]; ok && v != "" {
				httpReq.Header.Set(h, v)
			}
		}
	}

	// Set default Accept for simple API
	if httpReq.Header.Get("Accept") == "" {
		httpReq.Header.Set("Accept", "text/html, application/json")
	}

	// Set default User-Agent
	if httpReq.Header.Get("User-Agent") == "" {
		httpReq.Header.Set("User-Agent", "ProxyND/1.0 PyPI-Proxy")
	}
}

// getContentType determines the content type based on file path
func (s *PipService) getContentType(requestPath string) string {
	lowerPath := strings.ToLower(requestPath)
	ext := strings.ToLower(filepath.Ext(requestPath))

	switch {
	case strings.HasSuffix(lowerPath, ".whl"):
		return PipContentTypeWheel
	case strings.HasSuffix(lowerPath, ".tar.gz") || strings.HasSuffix(lowerPath, ".tgz"):
		return PipContentTypeTarGz
	case strings.HasSuffix(lowerPath, ".zip"):
		return PipContentTypeZip
	case strings.HasSuffix(lowerPath, ".egg"):
		return PipContentTypeEgg
	case strings.HasSuffix(lowerPath, ".tar.bz2"):
		return "application/x-bzip2"
	case ext == ".json" || strings.HasSuffix(lowerPath, "/json"):
		return PipContentTypeJSON
	case strings.Contains(lowerPath, "/simple/") || lowerPath == "simple":
		return PipContentTypeHTML
	default:
		return PipContentTypeDefault
	}
}

// SupportsHead returns whether the service supports HEAD requests
func (s *PipService) SupportsHead() bool {
	return true
}

// HandleHead handles HEAD requests for file existence checks
func (s *PipService) HandleHead(ctx context.Context, req ProxyRequest) (*ProxyResponse, error) {
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
func (s *PipService) GetRequestType(requestPath string) PipRequestType {
	return s.classifyRequest(requestPath)
}

// GetConfig returns the PyPI configuration
func (s *PipService) GetConfig() *config.PipProxySettings {
	return s.config
}
