package apt

import (
	"time"

	"proxynd/internal/config"
)

// PackageRequest APT 패키지 요청
type PackageRequest struct {
	OSType      string            `json:"osType"`      // ubuntu, debian, etc.
	PackagePath string            `json:"packagePath"` // 패키지 경로
	Headers     map[string]string `json:"headers"`     // HTTP 헤더
	Method      string            `json:"method"`      // HTTP 메서드
}

// PackageResponse APT 패키지 응답
type PackageResponse struct {
	Data        []byte            `json:"data"`
	ContentType string            `json:"contentType"`
	Headers     map[string]string `json:"headers"`
	StatusCode  int               `json:"statusCode"`
	FromCache   bool              `json:"fromCache"`
	MirrorUsed  string            `json:"mirrorUsed,omitempty"`
}

// MirrorStatus 미러 상태 정보
type MirrorStatus struct {
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
}

// PackageMetadata 패키지 메타데이터
type PackageMetadata struct {
	OSType       string    `json:"osType"`
	PackageName  string    `json:"packageName"`
	Version      string    `json:"version,omitempty"`
	Architecture string    `json:"architecture,omitempty"`
	FileType     string    `json:"fileType"` // "deb", "Release", "Packages", etc.
	Size         int64     `json:"size"`
	ModTime      time.Time `json:"modTime"`
}

// RequestMetrics 요청 메트릭
type RequestMetrics struct {
	OSType      string        `json:"osType"`
	Path        string        `json:"path"`
	Method      string        `json:"method"`
	StatusCode  int           `json:"statusCode"`
	Duration    time.Duration `json:"duration"`
	FromCache   bool          `json:"fromCache"`
	MirrorUsed  string        `json:"mirrorUsed,omitempty"`
	BytesServed int64         `json:"bytesServed"`
	Timestamp   time.Time     `json:"timestamp"`
}

// ProxyConfig APT 프록시 설정 인터페이스 (의존성 추상화)
type ProxyConfig interface {
	// GetMirrors 특정 OS 타입의 미러 목록 반환
	GetMirrors(osType string) []config.AptProxy

	// GetCacheConfig 캐시 설정 반환
	GetCacheConfig() CacheConfig

	// GetAllOSTypes 지원하는 모든 OS 타입 반환
	GetAllOSTypes() []string

	// IsEnabled 프록시 활성화 여부 확인
	IsEnabled() bool
}

// CacheConfig 캐시 설정
type CacheConfig struct {
	Enabled      bool          `json:"enabled"`
	BaseDir      string        `json:"baseDir"`
	TTL          time.Duration `json:"ttl"`
	MaxSizeMB    int64         `json:"maxSizeMB"`
	CleanupHours int           `json:"cleanupHours"`
}

// DefaultProxyConfig config.AptProxyConfig의 어댑터 (기본 구현)
type DefaultProxyConfig struct {
	config  *config.AptProxyConfig
	baseDir string
}

// NewDefaultProxyConfig DefaultProxyConfig 생성자
func NewDefaultProxyConfig(config *config.AptProxyConfig, baseDir string) ProxyConfig {
	return &DefaultProxyConfig{
		config:  config,
		baseDir: baseDir,
	}
}

// GetMirrors 특정 OS 타입의 미러 목록 반환
func (c *DefaultProxyConfig) GetMirrors(osType string) []config.AptProxy {
	if c.config == nil || c.config.Proxies == nil {
		return []config.AptProxy{}
	}

	if mirrors, exists := c.config.Proxies[osType]; exists {
		return mirrors
	}

	return []config.AptProxy{}
}

// GetCacheConfig 캐시 설정 반환
func (c *DefaultProxyConfig) GetCacheConfig() CacheConfig {
	return CacheConfig{
		Enabled:      true,
		BaseDir:      c.baseDir,
		TTL:          24 * time.Hour, // 24시간 캐시
		MaxSizeMB:    5000,           // 5GB 기본 제한
		CleanupHours: 6,              // 6시간마다 정리
	}
}

// GetAllOSTypes 지원하는 모든 OS 타입 반환
func (c *DefaultProxyConfig) GetAllOSTypes() []string {
	if c.config == nil || c.config.Proxies == nil {
		return []string{}
	}

	var osTypes []string
	for osType := range c.config.Proxies {
		osTypes = append(osTypes, osType)
	}

	return osTypes
}

// IsEnabled 프록시 활성화 여부 확인
func (c *DefaultProxyConfig) IsEnabled() bool {
	if c.config == nil || c.config.Proxies == nil {
		return false
	}

	// 최소 하나의 OS 타입에 미러가 설정되어 있어야 함
	for _, mirrors := range c.config.Proxies {
		if len(mirrors) > 0 {
			return true
		}
	}

	return false
}
