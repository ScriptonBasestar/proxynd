package proxy

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"proxynd/internal/config"
	"proxynd/logging"
)

// MavenService handles Maven repository proxy requests
type MavenService struct {
	*BaseProxyService
	config *config.MavenProxyConfig
}

// NewMavenService creates a new Maven proxy service
func NewMavenService(
	cache CacheService,
	configService ConfigService,
	upstreamClient UpstreamClient,
) (*MavenService, error) {
	base := NewBaseProxyService("maven", cache, configService, upstreamClient)

	// Load Maven-specific configuration
	configInterface, err := configService.GetProxyConfig(context.Background(), "maven")
	if err != nil {
		return nil, fmt.Errorf("failed to load maven config: %w", err)
	}

	mavenConfig, ok := configInterface.(*config.MavenProxyConfig)
	if !ok {
		return nil, fmt.Errorf("invalid maven config type")
	}

	return &MavenService{
		BaseProxyService: base,
		config:           mavenConfig,
	}, nil
}

// HandleRequest processes a Maven proxy request
func (s *MavenService) HandleRequest(ctx context.Context, req ProxyRequest) (*ProxyResponse, error) {
	// Validate request
	if err := s.ValidateRequest(req); err != nil {
		return s.HandleError(err, http.StatusBadRequest), nil
	}

	// Build cache key
	cacheKey := s.BuildCacheKey(req.Path)

	// Try to get from cache first
	cachedContent, found, err := s.TryCache(ctx, cacheKey)
	if err != nil {
		s.Logger.Warn("Cache error, continuing with upstream",
			logging.F("error", err))
	}

	if found && cachedContent != nil {
		// Return cached content
		filename := filepath.Base(req.Path)
		contentType := s.DetermineContentType(filename)

		return s.BuildProxyResponse(
			cachedContent,
			http.StatusOK,
			contentType,
			filename,
			true,
		), nil
	}

	// Not in cache, fetch from upstream
	response, err := s.fetchFromUpstream(ctx, req)
	if err != nil {
		s.Logger.Error("Failed to fetch from upstream",
			logging.F("path", req.Path),
			logging.F("error", err))
		return s.HandleError(err, http.StatusBadGateway), nil
	}

	return response, nil
}

// ValidateRequest validates Maven-specific request requirements
func (s *MavenService) ValidateRequest(req ProxyRequest) error {
	// Call base validation first
	if err := s.BaseProxyService.ValidateRequest(req); err != nil {
		return err
	}

	// Maven-specific validation
	if req.Method != "GET" && req.Method != "HEAD" {
		return fmt.Errorf("unsupported method for Maven proxy: %s", req.Method)
	}

	// Check if path looks like a valid Maven artifact path
	if !s.isValidMavenPath(req.Path) {
		return fmt.Errorf("invalid Maven artifact path: %s", req.Path)
	}

	return nil
}

// fetchFromUpstream fetches content from upstream Maven repositories
func (s *MavenService) fetchFromUpstream(ctx context.Context, req ProxyRequest) (*ProxyResponse, error) {
	// Try each configured proxy in order
	for _, proxy := range s.config.Proxies {
		if !proxy.Enabled {
			continue
		}

		url := s.buildUpstreamURL(proxy.URL, req.Path)

		s.Logger.Debug("Trying upstream",
			logging.F("proxy", proxy.Name),
			logging.F("url", url))

		// Build headers
		headers := make(map[string]string)
		if proxy.BasicAuth.Username != "" && proxy.BasicAuth.Password != "" {
			// Add basic auth if configured
			headers["Authorization"] = fmt.Sprintf("Basic %s",
				s.encodeBasicAuth(proxy.BasicAuth.Username, proxy.BasicAuth.Password))
		}

		// Fetch from upstream
		response, err := s.UpstreamClient.Fetch(ctx, url, headers)
		if err != nil {
			s.Logger.Warn("Failed to fetch from upstream",
				logging.F("proxy", proxy.Name),
				logging.F("url", url),
				logging.F("error", err))
			continue
		}

		if response.StatusCode == http.StatusOK {
			// Success! Cache the response if enabled
			if s.config.Cache.Enabled {
				// We need to read the content to cache it
				// This is typically done by the cache service implementation
				cacheKey := s.BuildCacheKey(req.Path)
				if err := s.CacheResponse(ctx, cacheKey, response.Body); err != nil {
					s.Logger.Warn("Failed to cache response",
						logging.F("error", err))
				}
			}

			return response, nil
		}

		// Close the body if not successful
		if response.Body != nil {
			_ = response.Body.Close()
		}
	}

	return nil, fmt.Errorf("all upstream proxies failed")
}

// isValidMavenPath checks if the path looks like a valid Maven artifact path
func (s *MavenService) isValidMavenPath(path string) bool {
	// Basic validation - Maven paths typically contain group/artifact/version structure
	// This is a simplified check
	parts := strings.Split(strings.Trim(path, "/"), "/")

	// At minimum, we need group/artifact/version/file
	if len(parts) < 4 {
		return false
	}

	// Check for common Maven file extensions
	filename := parts[len(parts)-1]
	validExtensions := []string{".pom", ".jar", ".war", ".ear", ".xml", ".sha1", ".md5", ".asc"}

	hasValidExtension := false
	for _, ext := range validExtensions {
		if strings.HasSuffix(filename, ext) {
			hasValidExtension = true
			break
		}
	}

	return hasValidExtension
}

// buildUpstreamURL constructs the full upstream URL
func (s *MavenService) buildUpstreamURL(baseURL, requestPath string) string {
	// Ensure base URL doesn't end with slash
	baseURL = strings.TrimRight(baseURL, "/")
	// Ensure request path starts with slash
	if !strings.HasPrefix(requestPath, "/") {
		requestPath = "/" + requestPath
	}
	return baseURL + requestPath
}

// encodeBasicAuth encodes username and password for basic authentication
func (s *MavenService) encodeBasicAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}
