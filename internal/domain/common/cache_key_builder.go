package common

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
)

// StandardCacheKeyBuilder implements a standard cache key builder
type StandardCacheKeyBuilder struct {
	separator string
}

// NewStandardCacheKeyBuilder creates a new standard cache key builder
func NewStandardCacheKeyBuilder() *StandardCacheKeyBuilder {
	return &StandardCacheKeyBuilder{
		separator: ":",
	}
}

// BuildKey builds a cache key from request parameters
func (b *StandardCacheKeyBuilder) BuildKey(proxyType, path string, params map[string]string) string {
	// Start with proxy type and path
	parts := []string{proxyType, b.sanitizePath(path)}

	// Add sorted query parameters if any
	if len(params) > 0 {
		paramStr := b.buildParamString(params)
		if paramStr != "" {
			parts = append(parts, paramStr)
		}
	}

	return strings.Join(parts, b.separator)
}

// ParseKey parses a cache key back to its components
func (b *StandardCacheKeyBuilder) ParseKey(key string) (proxyType, path string, params map[string]string, err error) {
	parts := strings.Split(key, b.separator)

	if len(parts) < 2 {
		return "", "", nil, fmt.Errorf("invalid cache key format: %s", key)
	}

	proxyType = parts[0]
	path = parts[1]

	// Parse parameters if present
	params = make(map[string]string)
	if len(parts) > 2 {
		paramStr := parts[2]
		params, err = b.parseParamString(paramStr)
		if err != nil {
			return "", "", nil, fmt.Errorf("failed to parse parameters: %w", err)
		}
	}

	return proxyType, path, params, nil
}

// sanitizePath cleans up a path for use in cache key
func (b *StandardCacheKeyBuilder) sanitizePath(path string) string {
	// Remove leading/trailing slashes
	path = strings.Trim(path, "/")

	// Replace multiple slashes with single slash
	path = strings.ReplaceAll(path, "//", "/")

	// URL encode special characters
	path = url.QueryEscape(path)

	return path
}

// buildParamString creates a consistent string representation of parameters
func (b *StandardCacheKeyBuilder) buildParamString(params map[string]string) string {
	if len(params) == 0 {
		return ""
	}

	// Sort keys for consistent ordering
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Build parameter pairs
	pairs := make([]string, 0, len(keys))
	for _, k := range keys {
		v := params[k]
		// URL encode both key and value
		pair := fmt.Sprintf("%s=%s", url.QueryEscape(k), url.QueryEscape(v))
		pairs = append(pairs, pair)
	}

	return strings.Join(pairs, "&")
}

// parseParamString parses a parameter string back to a map
func (b *StandardCacheKeyBuilder) parseParamString(paramStr string) (map[string]string, error) {
	params := make(map[string]string)

	if paramStr == "" {
		return params, nil
	}

	pairs := strings.Split(paramStr, "&")
	for _, pair := range pairs {
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key, err := url.QueryUnescape(parts[0])
		if err != nil {
			return nil, fmt.Errorf("failed to decode key %s: %w", parts[0], err)
		}

		value, err := url.QueryUnescape(parts[1])
		if err != nil {
			return nil, fmt.Errorf("failed to decode value %s: %w", parts[1], err)
		}

		params[key] = value
	}

	return params, nil
}

// Ensure StandardCacheKeyBuilder implements the CacheKeyBuilder interface
var _ CacheKeyBuilder = (*StandardCacheKeyBuilder)(nil)
