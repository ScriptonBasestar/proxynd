package testutil

import (
	"proxynd/configs"
)

// ConfigBuilder 설정 빌더
type ConfigBuilder struct {
	config interface{}
}

// NewGlobalConfigBuilder 글로벌 설정 빌더 생성
func NewGlobalConfigBuilder() *ConfigBuilder {
	return &ConfigBuilder{
		config: &configs.GlobalConfig{
			StorageDir:   "./storage",
			ConfigDir:    "./config",
			CacheDir:     "./cache",
			CacheTTL:     3600,
			MaxCacheSize: 1024 * 1024 * 1024, // 1GB
			Cache: configs.Cache{
				TTL:                  3600,
				UseCacheHeaders:      false,
				MaxCacheHeaderTTL:    86400,
				MinCacheHeaderTTL:    300,
				StaleWhileRevalidate: false,
				StaleMaxAge:          3600,
			},
		},
	}
}

// WithStorageDir 스토리지 디렉토리 설정
func (b *ConfigBuilder) WithStorageDir(dir string) *ConfigBuilder {
	if gc, ok := b.config.(*configs.GlobalConfig); ok {
		gc.StorageDir = dir
	}
	return b
}

// WithCacheDir 캐시 디렉토리 설정
func (b *ConfigBuilder) WithCacheDir(dir string) *ConfigBuilder {
	if gc, ok := b.config.(*configs.GlobalConfig); ok {
		gc.CacheDir = dir
	}
	return b
}

// WithCacheTTL 캐시 TTL 설정
func (b *ConfigBuilder) WithCacheTTL(ttl int) *ConfigBuilder {
	if gc, ok := b.config.(*configs.GlobalConfig); ok {
		gc.CacheTTL = ttl
		gc.Cache.TTL = ttl
	}
	return b
}

// WithMaxCacheSize 최대 캐시 크기 설정
func (b *ConfigBuilder) WithMaxCacheSize(size int64) *ConfigBuilder {
	if gc, ok := b.config.(*configs.GlobalConfig); ok {
		gc.MaxCacheSize = size
	}
	return b
}

// Build 설정 빌드
func (b *ConfigBuilder) Build() interface{} {
	return b.config
}

// APTConfigBuilder APT 설정 빌더
type APTConfigBuilder struct {
	config *configs.AptProxyConfig
}

// NewAPTConfigBuilder APT 설정 빌더 생성
func NewAPTConfigBuilder() *APTConfigBuilder {
	return &APTConfigBuilder{
		config: &configs.AptProxyConfig{
			Path:     "/proxy/apt",
			UseCache: true,
			Proxies: map[string][]configs.AptProxy{
				"default": {
					{
						Name: "Ubuntu Archive",
						URL:  "http://archive.ubuntu.com/ubuntu",
					},
				},
			},
		},
	}
}

// WithPath 프록시 경로 설정
func (b *APTConfigBuilder) WithPath(path string) *APTConfigBuilder {
	b.config.Path = path
	return b
}

// WithCacheEnabled 캐시 활성화 설정
func (b *APTConfigBuilder) WithCacheEnabled(enabled bool) *APTConfigBuilder {
	b.config.UseCache = enabled
	return b
}

// WithProxy 프록시 설정 추가
func (b *APTConfigBuilder) WithProxy(name, proxiesKey, url string) *APTConfigBuilder {
	if b.config.Proxies == nil {
		b.config.Proxies = make(map[string][]configs.AptProxy)
	}
	b.config.Proxies[proxiesKey] = append(b.config.Proxies[proxiesKey], configs.AptProxy{
		Name: name,
		URL:  url,
	})
	return b
}

// Build 설정 빌드
func (b *APTConfigBuilder) Build() *configs.AptProxyConfig {
	return b.config
}

// MavenConfigBuilder Maven 설정 빌더
type MavenConfigBuilder struct {
	config *configs.MavenProxyConfig
}

// NewMavenConfigBuilder Maven 설정 빌드 생성
func NewMavenConfigBuilder() *MavenConfigBuilder {
	return &MavenConfigBuilder{
		config: &configs.MavenProxyConfig{
			Path:     "/proxy/maven",
			UseCache: true,
			Proxies: []configs.MavenProxyServer{
				{
					ID:          "central",
					Name:        "Central Repository",
					URL:         "https://repo1.maven.org/maven2",
					Description: "Maven Central Repository",
					Enabled:     true,
				},
			},
			Cache: configs.MavenProxyCacheConfig{
				Enabled: true,
			},
		},
	}
}

// WithPath 프록시 경로 설정
func (b *MavenConfigBuilder) WithPath(path string) *MavenConfigBuilder {
	b.config.Path = path
	return b
}

// WithProxy 프록시 서버 추가
func (b *MavenConfigBuilder) WithProxy(id, name, url, description string, enabled bool) *MavenConfigBuilder {
	b.config.Proxies = append(b.config.Proxies, configs.MavenProxyServer{
		ID:          id,
		Name:        name,
		URL:         url,
		Description: description,
		Enabled:     enabled,
	})
	return b
}

// WithCacheEnabled 캐시 활성화 설정
func (b *MavenConfigBuilder) WithCacheEnabled(enabled bool) *MavenConfigBuilder {
	b.config.UseCache = enabled
	b.config.Cache.Enabled = enabled
	return b
}

// Build 설정 빌드
func (b *MavenConfigBuilder) Build() *configs.MavenProxyConfig {
	return b.config
}

// NPMConfigBuilder NPM 설정 빌더
type NPMConfigBuilder struct {
	config *configs.NpmProxyConfig
}

// NewNPMConfigBuilder NPM 설정 빌더 생성
func NewNPMConfigBuilder() *NPMConfigBuilder {
	return &NPMConfigBuilder{
		config: &configs.NpmProxyConfig{
			Path:      "/proxy/npm",
			UseCache:  true,
			UserCache: false,
			Proxies: map[string][]configs.NpmProxyServer{
				"default": {
					{
						Name: "NPM Registry",
						URL:  "https://registry.npmjs.org",
					},
				},
			},
		},
	}
}

// WithPath 프록시 경로 설정
func (b *NPMConfigBuilder) WithPath(path string) *NPMConfigBuilder {
	b.config.Path = path
	return b
}

// WithCacheEnabled 캐시 활성화 설정
func (b *NPMConfigBuilder) WithCacheEnabled(enabled bool) *NPMConfigBuilder {
	b.config.UseCache = enabled
	return b
}

// WithUserCache 사용자 캐시 설정
func (b *NPMConfigBuilder) WithUserCache(enabled bool) *NPMConfigBuilder {
	b.config.UserCache = enabled
	return b
}

// WithProxy 프록시 서버 추가
func (b *NPMConfigBuilder) WithProxy(registryName, serverName, url string) *NPMConfigBuilder {
	if b.config.Proxies == nil {
		b.config.Proxies = make(map[string][]configs.NpmProxyServer)
	}
	b.config.Proxies[registryName] = append(b.config.Proxies[registryName], configs.NpmProxyServer{
		Name: serverName,
		URL:  url,
	})
	return b
}

// Build 설정 빌드
func (b *NPMConfigBuilder) Build() *configs.NpmProxyConfig {
	return b.config
}

// ProxyRequestBuilder 프록시 요청 빌더
type ProxyRequestBuilder struct {
	method  string
	path    string
	headers map[string]string
	body    []byte
}

// NewProxyRequestBuilder 프록시 요청 빌더 생성
func NewProxyRequestBuilder() *ProxyRequestBuilder {
	return &ProxyRequestBuilder{
		method:  "GET",
		headers: make(map[string]string),
	}
}

// WithMethod HTTP 메서드 설정
func (b *ProxyRequestBuilder) WithMethod(method string) *ProxyRequestBuilder {
	b.method = method
	return b
}

// WithPath 경로 설정
func (b *ProxyRequestBuilder) WithPath(path string) *ProxyRequestBuilder {
	b.path = path
	return b
}

// WithHeader 헤더 추가
func (b *ProxyRequestBuilder) WithHeader(key, value string) *ProxyRequestBuilder {
	b.headers[key] = value
	return b
}

// WithBody 바디 설정
func (b *ProxyRequestBuilder) WithBody(body []byte) *ProxyRequestBuilder {
	b.body = body
	return b
}

// Build 요청 빌드
func (b *ProxyRequestBuilder) Build() map[string]interface{} {
	return map[string]interface{}{
		"method":  b.method,
		"path":    b.path,
		"headers": b.headers,
		"body":    b.body,
	}
}
