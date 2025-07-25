package npm

import (
	"time"

	"proxynd/configs"
)

// PackageRequest NPM 패키지 요청
type PackageRequest struct {
	PackagePath string            `json:"packagePath"` // 패키지 경로 (@scope/package, package/version 등)
	Headers     map[string]string `json:"headers"`     // HTTP 헤더
	Method      string            `json:"method"`      // HTTP 메서드
	BaseURL     string            `json:"baseUrl"`     // 프록시 서버 기본 URL
}

// PackageResponse NPM 패키지 응답
type PackageResponse struct {
	Data        []byte            `json:"data"`
	ContentType string            `json:"contentType"`
	Headers     map[string]string `json:"headers"`
	StatusCode  int               `json:"statusCode"`
	FromCache   bool              `json:"fromCache"`
	ProxyUsed   string            `json:"proxyUsed,omitempty"`  // 사용된 프록시 서버
	IsMetadata  bool              `json:"isMetadata,omitempty"` // 메타데이터 요청 여부
}

// ProxyStatus 프록시 서버 상태 정보
type ProxyStatus struct {
	Name      string    `json:"name"`
	URL       string    `json:"url"`
	Available bool      `json:"available"`
	LastCheck time.Time `json:"lastCheck"`
	Error     string    `json:"error,omitempty"`
}

// CacheEntry 캐시 엔트리
type CacheEntry struct {
	Key          string        `json:"key"`
	Path         string        `json:"path"`
	ContentType  string        `json:"contentType"`
	Size         int64         `json:"size"`
	CreatedAt    time.Time     `json:"createdAt"`
	LastAccessed time.Time     `json:"lastAccessed"`
	TTL          time.Duration `json:"ttl"`
	IsMetadata   bool          `json:"isMetadata"` // 메타데이터 파일 여부
}

// PackageMetadata NPM 패키지 메타데이터
type PackageMetadata struct {
	Name       string    `json:"name"`
	Version    string    `json:"version,omitempty"`
	IsScoped   bool      `json:"isScoped"`   // @scope/package 형태인지
	IsMetadata bool      `json:"isMetadata"` // 메타데이터 요청인지
	IsTarball  bool      `json:"isTarball"`  // .tgz 파일인지
	FileType   string    `json:"fileType"`   // "metadata", "tarball", "other"
	Size       int64     `json:"size"`
	ModTime    time.Time `json:"modTime"`
}

// RequestMetrics 요청 메트릭
type RequestMetrics struct {
	PackagePath string        `json:"packagePath"`
	Method      string        `json:"method"`
	StatusCode  int           `json:"statusCode"`
	Duration    time.Duration `json:"duration"`
	FromCache   bool          `json:"fromCache"`
	ProxyUsed   string        `json:"proxyUsed,omitempty"`
	BytesServed int64         `json:"bytesServed"`
	IsMetadata  bool          `json:"isMetadata"`
	Timestamp   time.Time     `json:"timestamp"`
}

// ProxyConfig NPM 프록시 설정 인터페이스 (의존성 추상화)
type ProxyConfig interface {
	// GetProxies 기본 프록시 목록 반환
	GetProxies() []configs.NpmProxyServer

	// GetCacheConfig 캐시 설정 반환
	GetCacheConfig() CacheConfig

	// IsEnabled 프록시 활성화 여부 확인
	IsEnabled() bool

	// GetPath 프록시 경로 반환
	GetPath() string
}

// CacheConfig 캐시 설정
type CacheConfig struct {
	Enabled      bool          `json:"enabled"`
	BaseDir      string        `json:"baseDir"`
	TTL          time.Duration `json:"ttl"`
	MaxSizeMB    int64         `json:"maxSizeMB"`
	CleanupHours int           `json:"cleanupHours"`
}

// DefaultProxyConfig configs.NpmProxyConfig의 어댑터 (기본 구현)
type DefaultProxyConfig struct {
	config  *configs.NpmProxyConfig
	baseDir string
}

// NewDefaultProxyConfig DefaultProxyConfig 생성자
func NewDefaultProxyConfig(config *configs.NpmProxyConfig, baseDir string) ProxyConfig {
	return &DefaultProxyConfig{
		config:  config,
		baseDir: baseDir,
	}
}

// GetProxies 기본 프록시 목록 반환
func (c *DefaultProxyConfig) GetProxies() []configs.NpmProxyServer {
	if c.config == nil || c.config.Proxies == nil {
		return []configs.NpmProxyServer{}
	}

	if proxies, exists := c.config.Proxies["default"]; exists {
		return proxies
	}

	return []configs.NpmProxyServer{}
}

// GetCacheConfig 캐시 설정 반환
func (c *DefaultProxyConfig) GetCacheConfig() CacheConfig {
	return CacheConfig{
		Enabled:      true,
		BaseDir:      c.baseDir,
		TTL:          6 * time.Hour, // 6시간 캐시 (NPM 메타데이터는 자주 변경됨)
		MaxSizeMB:    3000,          // 3GB 기본 제한
		CleanupHours: 4,             // 4시간마다 정리
	}
}

// IsEnabled 프록시 활성화 여부 확인
func (c *DefaultProxyConfig) IsEnabled() bool {
	if c.config == nil || c.config.Proxies == nil {
		return false
	}

	// 기본 프록시가 설정되어 있어야 함
	if proxies, exists := c.config.Proxies["default"]; exists && len(proxies) > 0 {
		return true
	}

	return false
}

// GetPath 프록시 경로 반환
func (c *DefaultProxyConfig) GetPath() string {
	if c.config == nil {
		return "npm"
	}
	return c.config.Path
}
