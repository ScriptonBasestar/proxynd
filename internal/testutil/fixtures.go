package testutil

import (
	"bytes"
	"io"
	"net/http"
	"time"

	"proxynd/configs"
	"proxynd/internal/services/proxy"
)

// Fixtures provides pre-configured test data
type Fixtures struct{}

// NewFixtures creates a new fixtures instance
func NewFixtures() *Fixtures {
	return &Fixtures{}
}

// ValidGlobalConfig returns a valid global configuration for testing
func (f *Fixtures) ValidGlobalConfig() *configs.GlobalConfig {
	return &configs.GlobalConfig{
		StorageDir:   "/tmp/test-storage",
		ConfigDir:    "/tmp/test-config",
		CacheDir:     "/tmp/test-cache",
		CacheTTL:     3600,
		MaxCacheSize: 1024 * 1024 * 1024, // 1GB
		Cache: configs.Cache{
			Type:      "filesystem",
			TTL:       3600,
			MaxSize:   1073741824,
			Directory: "/tmp/test-cache",
		},
	}
}

// ValidAPTConfig returns a valid APT proxy configuration
func (f *Fixtures) ValidAPTConfig() *configs.AptProxyConfig {
	return &configs.AptProxyConfig{
		Name:    "apt-proxy",
		Enabled: true,
		Servers: []configs.AptProxyServer{
			{
				Name:    "ubuntu-main",
				Enabled: true,
				Server: configs.UpstreamServer{
					ID:       "ubuntu",
					URL:      "http://archive.ubuntu.com/ubuntu",
					Priority: 1,
					Timeout:  30,
				},
			},
		},
	}
}

// ValidMavenConfig returns a valid Maven proxy configuration
func (f *Fixtures) ValidMavenConfig() *configs.MavenProxyConfig {
	return &configs.MavenProxyConfig{
		Name:    "maven-proxy",
		Enabled: true,
		Servers: []configs.MavenProxyServer{
			{
				Name:    "central",
				Enabled: true,
				Url:     "https://repo1.maven.org/maven2",
			},
		},
	}
}

// ValidNPMConfig returns a valid NPM proxy configuration
func (f *Fixtures) ValidNPMConfig() *configs.NpmProxyConfig {
	return &configs.NpmProxyConfig{
		Name:    "npm-proxy",
		Enabled: true,
		Servers: []configs.NpmProxyServer{
			{
				Name:    "npmjs",
				Enabled: true,
				Url:     "https://registry.npmjs.org",
			},
		},
	}
}

// ProxyRequest creates a test proxy request
func (f *Fixtures) ProxyRequest(method, path string) *proxy.ProxyRequest {
	return &proxy.ProxyRequest{
		Method:     method,
		Path:       path,
		ProxyType:  "test",
		Headers:    make(map[string]string),
		RemoteAddr: "127.0.0.1:12345",
	}
}

// ProxyRequestWithHeaders creates a test proxy request with headers
func (f *Fixtures) ProxyRequestWithHeaders(method, path string, headers map[string]string) *proxy.ProxyRequest {
	req := f.ProxyRequest(method, path)
	req.Headers = headers
	return req
}

// ProxyResponse creates a test proxy response
func (f *Fixtures) ProxyResponse(statusCode int, body string) *proxy.ProxyResponse {
	return &proxy.ProxyResponse{
		Body:        io.NopCloser(bytes.NewBufferString(body)),
		StatusCode:  statusCode,
		Headers:     make(map[string]string),
		ContentType: "text/plain",
		Cached:      false,
	}
}

// ProxyResponseWithHeaders creates a test proxy response with headers
func (f *Fixtures) ProxyResponseWithHeaders(statusCode int, body string, headers map[string]string) *proxy.ProxyResponse {
	resp := f.ProxyResponse(statusCode, body)
	resp.Headers = headers
	return resp
}

// HTTPRequest creates a test HTTP request
func (f *Fixtures) HTTPRequest(method, url string) *http.Request {
	req, _ := http.NewRequest(method, url, nil)
	return req
}

// HTTPRequestWithBody creates a test HTTP request with body
func (f *Fixtures) HTTPRequestWithBody(method, url string, body []byte) *http.Request {
	req, _ := http.NewRequest(method, url, bytes.NewReader(body))
	return req
}

// SampleAPTPackageMetadata returns sample APT package metadata
func (f *Fixtures) SampleAPTPackageMetadata() string {
	return `Package: test-package
Version: 1.0.0
Architecture: amd64
Maintainer: Test Maintainer <test@example.com>
Description: Test package for unit testing
`
}

// SampleMavenPOM returns sample Maven POM XML
func (f *Fixtures) SampleMavenPOM() string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0">
    <modelVersion>4.0.0</modelVersion>
    <groupId>com.example</groupId>
    <artifactId>test-artifact</artifactId>
    <version>1.0.0</version>
    <packaging>jar</packaging>
</project>`
}

// SampleNPMPackageJSON returns sample NPM package.json
func (f *Fixtures) SampleNPMPackageJSON() string {
	return `{
  "name": "test-package",
  "version": "1.0.0",
  "description": "Test package for unit testing",
  "main": "index.js",
  "dependencies": {
    "express": "^4.18.0"
  }
}`
}

// TimeFixture returns a fixed time for testing
func (f *Fixtures) TimeFixture() time.Time {
	return time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
}