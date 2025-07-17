package proxy

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"proxynd/internal/security"
	"proxynd/logging"
)

// BaseProxyService provides common functionality for all proxy services
type BaseProxyService struct {
	Logger         logging.Logger
	Cache          CacheService
	Config         ConfigService
	UpstreamClient UpstreamClient
	ProxyType      string
}

// NewBaseProxyService creates a new base proxy service
func NewBaseProxyService(
	proxyType string,
	cache CacheService,
	config ConfigService,
	upstreamClient UpstreamClient,
) *BaseProxyService {
	return &BaseProxyService{
		Logger:         logging.GetLogger(),
		Cache:          cache,
		Config:         config,
		UpstreamClient: upstreamClient,
		ProxyType:      proxyType,
	}
}

// GetProxyType returns the type of proxy
func (s *BaseProxyService) GetProxyType() string {
	return s.ProxyType
}

// ValidateRequest provides basic request validation
func (s *BaseProxyService) ValidateRequest(req ProxyRequest) error {
	if req.Path == "" {
		return fmt.Errorf("request path cannot be empty")
	}

	// Enhanced security validation using security package
	if security.ContainsTraversalPattern(req.Path) {
		return fmt.Errorf("invalid path: directory traversal detected")
	}

	return nil
}

// TryCache attempts to retrieve content from cache
func (s *BaseProxyService) TryCache(ctx context.Context, cacheKey string) (io.ReadCloser, bool, error) {
	// Check if content exists in cache
	exists, err := s.Cache.Exists(ctx, cacheKey)
	if err != nil {
		s.Logger.Warn("Failed to check cache existence",
			logging.F("key", cacheKey),
			logging.F("error", err))
		return nil, false, nil
	}

	if !exists {
		return nil, false, nil
	}

	// Retrieve from cache
	content, found, err := s.Cache.Get(ctx, cacheKey)
	if err != nil {
		s.Logger.Error("Failed to retrieve from cache",
			logging.F("key", cacheKey),
			logging.F("error", err))
		return nil, false, err
	}

	if found {
		s.Logger.Info("Cache hit",
			logging.F("proxy_type", s.ProxyType),
			logging.F("key", cacheKey))
	}

	return content, found, nil
}

// CacheResponse stores response in cache
func (s *BaseProxyService) CacheResponse(ctx context.Context, cacheKey string, content io.Reader) error {
	err := s.Cache.Put(ctx, cacheKey, content)
	if err != nil {
		s.Logger.Error("Failed to cache response",
			logging.F("key", cacheKey),
			logging.F("error", err))
		return err
	}

	s.Logger.Info("Response cached",
		logging.F("proxy_type", s.ProxyType),
		logging.F("key", cacheKey))

	return nil
}

// BuildCacheKey creates a cache key from proxy type and request path
func (s *BaseProxyService) BuildCacheKey(requestPath string) string {
	// Use SafeJoinPath for security
	cacheKey, err := security.SafeJoinPath(s.ProxyType, requestPath)
	if err != nil {
		// Fallback to safe key if error
		return filepath.Join(s.ProxyType, "invalid-path")
	}
	return cacheKey
}

// DetermineContentType determines content type based on file extension
func (s *BaseProxyService) DetermineContentType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))

	switch ext {
	case ".xml", ".pom":
		return "application/xml"
	case ".json":
		return "application/json"
	case ".jar":
		return "application/java-archive"
	case ".tar":
		return "application/x-tar"
	case ".gz", ".tgz":
		return "application/gzip"
	case ".zip":
		return "application/zip"
	case ".deb":
		return "application/vnd.debian.binary-package"
	case ".rpm":
		return "application/x-rpm"
	default:
		return "application/octet-stream"
	}
}

// BuildProxyResponse creates a standard proxy response
func (s *BaseProxyService) BuildProxyResponse(
	body io.ReadCloser,
	statusCode int,
	contentType string,
	filename string,
	cached bool,
) *ProxyResponse {
	headers := make(map[string]string)

	if filename != "" {
		if contentType == "application/xml" || strings.HasSuffix(filename, ".xml") {
			headers["Content-Disposition"] = fmt.Sprintf("inline; filename=%s", filename)
		} else {
			headers["Content-Disposition"] = fmt.Sprintf("attachment; filename=%s", filename)
		}
	}

	if cached {
		headers["X-Cache"] = "HIT"
	} else {
		headers["X-Cache"] = "MISS"
	}

	headers["X-Proxy-Type"] = s.ProxyType

	return &ProxyResponse{
		Body:        body,
		StatusCode:  statusCode,
		Headers:     headers,
		ContentType: contentType,
		FileName:    filename,
		Cached:      cached,
	}
}

// HandleNotFound returns a standard 404 response
func (s *BaseProxyService) HandleNotFound(message string) *ProxyResponse {
	return &ProxyResponse{
		Body:        io.NopCloser(strings.NewReader(message)),
		StatusCode:  http.StatusNotFound,
		Headers:     map[string]string{"Content-Type": "text/plain"},
		ContentType: "text/plain",
		Cached:      false,
	}
}

// HandleError returns a standard error response
func (s *BaseProxyService) HandleError(err error, statusCode int) *ProxyResponse {
	errorMessage := "unknown error"
	if err != nil {
		errorMessage = err.Error()
	}

	return &ProxyResponse{
		Body:        io.NopCloser(strings.NewReader(errorMessage)),
		StatusCode:  statusCode,
		Headers:     map[string]string{"Content-Type": "text/plain"},
		ContentType: "text/plain",
		Cached:      false,
	}
}

// HandleRequest provides a default implementation that returns 501 Not Implemented
func (s *BaseProxyService) HandleRequest(ctx context.Context, req ProxyRequest) (*ProxyResponse, error) {
	return &ProxyResponse{
		Body:        io.NopCloser(strings.NewReader("service not implemented")),
		StatusCode:  http.StatusNotImplemented,
		Headers:     map[string]string{"Content-Type": "text/plain"},
		ContentType: "text/plain",
		Cached:      false,
	}, nil
}
