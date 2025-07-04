package testutil

import (
	"proxynd/configs"
	"time"
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
				Type:      "filesystem",
				TTL:       3600,
				MaxSize:   1073741824,
				Directory: "./cache",
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
		gc.Cache.Directory = dir
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
		gc.Cache.MaxSize = size
	}
	return b
}

// Build 설정 빌드
func (b *ConfigBuilder) Build() interface{} {
	return b.config
}

// APTConfigBuilder APT 설정 빌더
type APTConfigBuilder struct {
	config *configs.APTConfig
}

// NewAPTConfigBuilder APT 설정 빌더 생성
func NewAPTConfigBuilder() *APTConfigBuilder {
	return &APTConfigBuilder{
		config: &configs.APTConfig{
			UpstreamURLs: []string{"http://archive.ubuntu.com/ubuntu"},
			CacheEnabled: true,
			AllowedArchitectures: []string{"amd64", "arm64"},
			AllowedDistributions: []string{"jammy", "focal"},
		},
	}
}

// WithUpstreamURLs 업스트림 URL 설정
func (b *APTConfigBuilder) WithUpstreamURLs(urls ...string) *APTConfigBuilder {
	b.config.UpstreamURLs = urls
	return b
}

// WithCacheEnabled 캐시 활성화 설정
func (b *APTConfigBuilder) WithCacheEnabled(enabled bool) *APTConfigBuilder {
	b.config.CacheEnabled = enabled
	return b
}

// WithAllowedArchitectures 허용 아키텍처 설정
func (b *APTConfigBuilder) WithAllowedArchitectures(archs ...string) *APTConfigBuilder {
	b.config.AllowedArchitectures = archs
	return b
}

// Build 설정 빌드
func (b *APTConfigBuilder) Build() *configs.APTConfig {
	return b.config
}

// MavenConfigBuilder Maven 설정 빌더
type MavenConfigBuilder struct {
	config *configs.MavenConfig
}

// NewMavenConfigBuilder Maven 설정 빌더 생성
func NewMavenConfigBuilder() *MavenConfigBuilder {
	return &MavenConfigBuilder{
		config: &configs.MavenConfig{
			UpstreamURLs: []configs.MavenUpstream{
				{
					ID:       "central",
					URL:      "https://repo1.maven.org/maven2",
					Priority: 1,
				},
			},
			CacheEnabled:       true,
			ChecksumValidation: true,
		},
	}
}

// WithUpstream 업스트림 추가
func (b *MavenConfigBuilder) WithUpstream(id, url string, priority int) *MavenConfigBuilder {
	b.config.UpstreamURLs = append(b.config.UpstreamURLs, configs.MavenUpstream{
		ID:       id,
		URL:      url,
		Priority: priority,
	})
	return b
}

// WithChecksumValidation 체크섬 검증 설정
func (b *MavenConfigBuilder) WithChecksumValidation(enabled bool) *MavenConfigBuilder {
	b.config.ChecksumValidation = enabled
	return b
}

// Build 설정 빌드
func (b *MavenConfigBuilder) Build() *configs.MavenConfig {
	return b.config
}

// NPMConfigBuilder NPM 설정 빌더
type NPMConfigBuilder struct {
	config *configs.NPMConfig
}

// NewNPMConfigBuilder NPM 설정 빌더 생성
func NewNPMConfigBuilder() *NPMConfigBuilder {
	return &NPMConfigBuilder{
		config: &configs.NPMConfig{
			UpstreamURLs:         []string{"https://registry.npmjs.org"},
			CacheEnabled:         true,
			ScopedPackagesAllowed: true,
		},
	}
}

// WithUpstreamURLs 업스트림 URL 설정
func (b *NPMConfigBuilder) WithUpstreamURLs(urls ...string) *NPMConfigBuilder {
	b.config.UpstreamURLs = urls
	return b
}

// WithScopedPackagesAllowed Scoped 패키지 허용 설정
func (b *NPMConfigBuilder) WithScopedPackagesAllowed(allowed bool) *NPMConfigBuilder {
	b.config.ScopedPackagesAllowed = allowed
	return b
}

// Build 설정 빌드
func (b *NPMConfigBuilder) Build() *configs.NPMConfig {
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