package maven

import "proxynd/internal/config"

// ProxyConfig Maven 프록시 설정 인터페이스 (의존성 추상화)
type ProxyConfig interface {
	// GetProxies 설정된 프록시 서버 목록 반환
	GetProxies() []config.MavenProxyServer

	// GetCacheConfig 캐시 설정 반환
	GetCacheConfig() CacheConfig

	// GetSearchConfig 검색 설정 반환
	GetSearchConfig() SearchConfig
}

// CacheConfig 캐시 설정
type CacheConfig struct {
	Enabled             bool `json:"enabled"`
	TTLMinutes          int  `json:"ttlMinutes"`
	MaxSizeMB           int  `json:"maxSizeMB"`
	PreloadPopularPaths bool `json:"preloadPopularPaths"`
}

// SearchConfig 검색 설정
type SearchConfig struct {
	Enabled           bool `json:"enabled"`
	IndexRebuildHours int  `json:"indexRebuildHours"`
	MaxResults        int  `json:"maxResults"`
}

// DefaultProxyConfig config.MavenProxyConfig의 어댑터 (기본 구현)
type DefaultProxyConfig struct {
	config *config.MavenProxySettings
}

// NewDefaultProxyConfig DefaultProxyConfig 생성자
func NewDefaultProxyConfig(config *config.MavenProxySettings) ProxyConfig {
	return &DefaultProxyConfig{
		config: config,
	}
}

// GetProxies 프록시 서버 목록 반환
func (c *DefaultProxyConfig) GetProxies() []config.MavenProxyServer {
	if c.config == nil {
		return []config.MavenProxyServer{}
	}
	return c.config.Proxies
}

// GetCacheConfig 캐시 설정 반환
func (c *DefaultProxyConfig) GetCacheConfig() CacheConfig {
	// 기본값 설정
	cacheConfig := CacheConfig{
		Enabled:             true,
		TTLMinutes:          60,
		MaxSizeMB:           500,
		PreloadPopularPaths: true,
	}

	// config.MavenProxyConfig에서 캐시 관련 설정이 있다면 매핑
	// 현재는 기본값 사용
	return cacheConfig
}

// GetSearchConfig 검색 설정 반환
func (c *DefaultProxyConfig) GetSearchConfig() SearchConfig {
	// 기본값 설정
	searchConfig := SearchConfig{
		Enabled:           true,
		IndexRebuildHours: 24,
		MaxResults:        100,
	}

	// config.MavenProxyConfig에서 검색 관련 설정이 있다면 매핑
	// 현재는 기본값 사용
	return searchConfig
}
