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

// ApkRequestType represents the type of APK repository request
type ApkRequestType int

const (
	// ApkRequestTypeUnknown is unknown request type
	ApkRequestTypeUnknown ApkRequestType = iota
	// ApkRequestTypeAPKINDEX is APKINDEX.tar.gz metadata request
	ApkRequestTypeAPKINDEX
	// ApkRequestTypePackage is .apk package request
	ApkRequestTypePackage
	// ApkRequestTypeSignature is .sig signature request
	ApkRequestTypeSignature
	// ApkRequestTypeDescription is .txt description file request
	ApkRequestTypeDescription
)

// APK content types
const (
	ApkContentTypeAPK     = "application/vnd.alpine.package"
	ApkContentTypeGzip    = "application/gzip"
	ApkContentTypeTarGz   = "application/gzip"
	ApkContentTypeSig     = "application/octet-stream"
	ApkContentTypeText    = "text/plain; charset=utf-8"
	ApkContentTypeDefault = "application/octet-stream"

	// Cache status for APK responses
	apkCacheStatusMiss = "MISS"
)

// ApkService handles Alpine Linux APK repository proxy requests
type ApkService struct {
	*BaseProxyService
	config     *config.ApkProxySettings
	httpClient *http.Client
}

// NewApkService creates a new APK proxy service
func NewApkService(
	cache CacheService,
	configService ConfigService,
	upstreamClient UpstreamClient,
) (*ApkService, error) {
	base := NewBaseProxyService("apk", cache, configService, upstreamClient)

	// Load APK-specific configuration
	configInterface, err := configService.GetProxyConfig(context.Background(), "apk")
	if err != nil {
		return nil, fmt.Errorf("failed to load apk config: %w", err)
	}

	apkConfig, ok := configInterface.(*config.ApkProxySettings)
	if !ok {
		return nil, fmt.Errorf("invalid apk config type")
	}

	return &ApkService{
		BaseProxyService: base,
		config:           apkConfig,
		httpClient:       &http.Client{},
	}, nil
}

// HandleRequest processes an APK proxy request
func (s *ApkService) HandleRequest(ctx context.Context, req ProxyRequest) (*ProxyResponse, error) {
	// Validate request
	if err := s.ValidateRequest(req); err != nil {
		return s.HandleError(err, http.StatusBadRequest), nil
	}

	// Determine request type
	reqType := s.classifyRequest(req.Path)

	s.Logger.Debug("APK request classified",
		logging.F("path", req.Path),
		logging.F("type", reqType))

	switch reqType {
	case ApkRequestTypeAPKINDEX:
		return s.handleAPKINDEX(ctx, req)
	case ApkRequestTypePackage:
		return s.handlePackage(ctx, req)
	case ApkRequestTypeSignature:
		return s.handleSignature(ctx, req)
	case ApkRequestTypeDescription:
		return s.handleDescription(ctx, req)
	default:
		return s.handleGenericRequest(ctx, req)
	}
}

// classifyRequest determines the type of APK request
func (s *ApkService) classifyRequest(requestPath string) ApkRequestType {
	// Normalize path
	filename := strings.ToLower(filepath.Base(requestPath))
	lowerPath := strings.ToLower(requestPath)

	// APKINDEX.tar.gz - repository index
	if filename == "apkindex.tar.gz" {
		return ApkRequestTypeAPKINDEX
	}

	// .apk package file
	if strings.HasSuffix(lowerPath, ".apk") {
		return ApkRequestTypePackage
	}

	// Signature files (.sig, .rsa.pub)
	if strings.HasSuffix(lowerPath, ".sig") || strings.HasSuffix(lowerPath, ".rsa.pub") {
		return ApkRequestTypeSignature
	}

	// Description/text files
	if strings.HasSuffix(lowerPath, ".txt") || filename == "description" {
		return ApkRequestTypeDescription
	}

	return ApkRequestTypeUnknown
}

// handleAPKINDEX handles APKINDEX.tar.gz metadata requests
func (s *ApkService) handleAPKINDEX(ctx context.Context, req ProxyRequest) (*ProxyResponse, error) {
	cacheKey := s.BuildCacheKey(req.Path)

	// Try cache first (short TTL for APKINDEX as it changes with updates)
	content, found, err := s.TryCache(ctx, cacheKey)
	if err != nil {
		s.Logger.Warn("Cache lookup failed", logging.F("error", err))
	}

	if found && content != nil {
		data, err := io.ReadAll(content)
		_ = content.Close()
		if err == nil {
			headers := map[string]string{
				"Content-Type":   ApkContentTypeTarGz,
				"X-Cache-Status": "HIT",
			}

			return &ProxyResponse{
				Body:        io.NopCloser(bytes.NewReader(data)),
				StatusCode:  http.StatusOK,
				Headers:     headers,
				ContentType: ApkContentTypeTarGz,
				Cached:      true,
			}, nil
		}
	}

	// Fetch from upstream
	return s.fetchFromUpstream(ctx, req, cacheKey)
}

// handlePackage handles .apk package download requests
func (s *ApkService) handlePackage(ctx context.Context, req ProxyRequest) (*ProxyResponse, error) {
	cacheKey := s.BuildCacheKey(req.Path)

	// Try cache first (packages have long TTL)
	content, found, err := s.TryCache(ctx, cacheKey)
	if err != nil {
		s.Logger.Warn("Cache lookup failed", logging.F("error", err))
	}

	if found && content != nil {
		headers := map[string]string{
			"Content-Type":   ApkContentTypeAPK,
			"X-Cache-Status": "HIT",
		}

		return &ProxyResponse{
			Body:        content,
			StatusCode:  http.StatusOK,
			Headers:     headers,
			ContentType: ApkContentTypeAPK,
			Cached:      true,
		}, nil
	}

	// Fetch from upstream
	return s.fetchFromUpstream(ctx, req, cacheKey)
}

// handleSignature handles signature file requests
func (s *ApkService) handleSignature(ctx context.Context, req ProxyRequest) (*ProxyResponse, error) {
	cacheKey := s.BuildCacheKey(req.Path)

	// Try cache first
	content, found, err := s.TryCache(ctx, cacheKey)
	if err != nil {
		s.Logger.Warn("Cache lookup failed", logging.F("error", err))
	}

	if found && content != nil {
		headers := map[string]string{
			"Content-Type":   ApkContentTypeSig,
			"X-Cache-Status": "HIT",
		}

		return &ProxyResponse{
			Body:        content,
			StatusCode:  http.StatusOK,
			Headers:     headers,
			ContentType: ApkContentTypeSig,
			Cached:      true,
		}, nil
	}

	// Fetch from upstream
	return s.fetchFromUpstream(ctx, req, cacheKey)
}

// handleDescription handles description/text file requests
func (s *ApkService) handleDescription(ctx context.Context, req ProxyRequest) (*ProxyResponse, error) {
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
				"Content-Type":   ApkContentTypeText,
				"X-Cache-Status": "HIT",
			}

			return &ProxyResponse{
				Body:        io.NopCloser(bytes.NewReader(data)),
				StatusCode:  http.StatusOK,
				Headers:     headers,
				ContentType: ApkContentTypeText,
				Cached:      true,
			}, nil
		}
	}

	// Fetch from upstream
	return s.fetchFromUpstream(ctx, req, cacheKey)
}

// handleGenericRequest handles unknown request types
func (s *ApkService) handleGenericRequest(ctx context.Context, req ProxyRequest) (*ProxyResponse, error) {
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

// fetchFromUpstream fetches content from upstream APK repository
func (s *ApkService) fetchFromUpstream(ctx context.Context, req ProxyRequest, cacheKey string) (*ProxyResponse, error) {
	if s.config == nil || len(s.config.Proxies) == 0 {
		return s.HandleError(fmt.Errorf("no apk proxies configured"), http.StatusServiceUnavailable), nil
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
			headers["X-Cache-Status"] = apkCacheStatusMiss

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

// buildUpstreamURL constructs the upstream APK repository URL
func (s *ApkService) buildUpstreamURL(serverURL, requestPath string) string {
	// Remove leading slash from path
	requestPath = strings.TrimPrefix(requestPath, "/")

	// Remove trailing slash from server URL
	serverURL = strings.TrimSuffix(serverURL, "/")

	return serverURL + "/" + requestPath
}

// copyRequestHeaders copies relevant headers from the original request
func (s *ApkService) copyRequestHeaders(req ProxyRequest, httpReq *http.Request) {
	if req.Headers != nil {
		for _, h := range []string{"Accept", "Accept-Encoding", "User-Agent", "If-Modified-Since", "If-None-Match"} {
			if v, ok := req.Headers[h]; ok && v != "" {
				httpReq.Header.Set(h, v)
			}
		}
	}

	// Set default User-Agent
	if httpReq.Header.Get("User-Agent") == "" {
		httpReq.Header.Set("User-Agent", "ProxyND/1.0 APK-Proxy")
	}

	// Accept compressed content
	if httpReq.Header.Get("Accept-Encoding") == "" {
		httpReq.Header.Set("Accept-Encoding", "gzip, deflate")
	}
}

// getContentType determines the content type based on file path
func (s *ApkService) getContentType(requestPath string) string {
	lowerPath := strings.ToLower(requestPath)
	filename := strings.ToLower(filepath.Base(requestPath))

	switch {
	case strings.HasSuffix(lowerPath, ".apk"):
		return ApkContentTypeAPK
	case filename == "apkindex.tar.gz":
		return ApkContentTypeTarGz
	case strings.HasSuffix(lowerPath, ".tar.gz") || strings.HasSuffix(lowerPath, ".tgz"):
		return ApkContentTypeTarGz
	case strings.HasSuffix(lowerPath, ".gz"):
		return ApkContentTypeGzip
	case strings.HasSuffix(lowerPath, ".sig") || strings.HasSuffix(lowerPath, ".rsa.pub"):
		return ApkContentTypeSig
	case strings.HasSuffix(lowerPath, ".txt"):
		return ApkContentTypeText
	default:
		return ApkContentTypeDefault
	}
}

// SupportsHead returns whether the service supports HEAD requests
func (s *ApkService) SupportsHead() bool {
	return true
}

// HandleHead handles HEAD requests for file existence checks
func (s *ApkService) HandleHead(ctx context.Context, req ProxyRequest) (*ProxyResponse, error) {
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
func (s *ApkService) GetRequestType(requestPath string) ApkRequestType {
	return s.classifyRequest(requestPath)
}

// GetConfig returns the APK configuration
func (s *ApkService) GetConfig() *config.ApkProxySettings {
	return s.config
}
