package pip

import (
	"time"

	"proxynd/internal/config"
)

// PackageRequest PIP 패키지 요청
type PackageRequest struct {
	PackagePath string            `json:"packagePath"`           // 패키지 경로 (simple/package, packages/source/package 등)
	Headers     map[string]string `json:"headers"`               // HTTP 헤더
	Method      string            `json:"method"`                // HTTP 메서드 (대부분 GET)
	BaseURL     string            `json:"baseUrl"`               // 프록시 서버 기본 URL
	QueryParams map[string]string `json:"queryParams,omitempty"` // PyPI API 쿼리 파라미터
}

// PackageResponse PIP 패키지 응답
type PackageResponse struct {
	Data          []byte            `json:"data"`
	ContentType   string            `json:"contentType"`
	Headers       map[string]string `json:"headers"`
	StatusCode    int               `json:"statusCode"`
	FromCache     bool              `json:"fromCache"`
	ProxyUsed     string            `json:"proxyUsed,omitempty"`     // 사용된 프록시 서버
	IsSimpleAPI   bool              `json:"isSimpleAPI,omitempty"`   // Simple API 요청 여부
	IsPackageFile bool              `json:"isPackageFile,omitempty"` // 패키지 파일 (.whl, .tar.gz) 여부
}

// IndexStatus 인덱스 서버 상태 정보
type IndexStatus struct {
	Name         string    `json:"name"`
	URL          string    `json:"url"`
	Available    bool      `json:"available"`
	LastCheck    time.Time `json:"lastCheck"`
	Error        string    `json:"error,omitempty"`
	ResponseTime float64   `json:"responseTime"` // milliseconds
}

// CacheEntry 캐시 엔트리
type CacheEntry struct {
	Key            string        `json:"key"`
	Path           string        `json:"path"`
	ContentType    string        `json:"contentType"`
	Size           int64         `json:"size"`
	CreatedAt      time.Time     `json:"createdAt"`
	LastAccessed   time.Time     `json:"lastAccessed"`
	TTL            time.Duration `json:"ttl"`
	IsSimpleAPI    bool          `json:"isSimpleAPI"`              // Simple API 캐시 여부
	IsPackageFile  bool          `json:"isPackageFile"`            // 패키지 파일 캐시 여부
	ChecksumSHA256 string        `json:"checksumSHA256,omitempty"` // 패키지 체크섬
}

// PackageMetadata PIP 패키지 메타데이터
type PackageMetadata struct {
	Name           string    `json:"name"`
	Version        string    `json:"version,omitempty"`
	FileType       string    `json:"fileType"`  // "wheel", "sdist", "simple", "metadata"
	Extension      string    `json:"extension"` // ".whl", ".tar.gz", ".zip" 등
	Size           int64     `json:"size"`
	ModTime        time.Time `json:"modTime"`
	IsSimpleAPI    bool      `json:"isSimpleAPI"`              // Simple API HTML 응답인지
	IsPackageFile  bool      `json:"isPackageFile"`            // 실제 패키지 파일인지
	ChecksumSHA256 string    `json:"checksumSHA256,omitempty"` // 패키지 체크섬
}

// RequestMetrics 요청 메트릭
type RequestMetrics struct {
	PackagePath   string        `json:"packagePath"`
	Method        string        `json:"method"`
	StatusCode    int           `json:"statusCode"`
	Duration      time.Duration `json:"duration"`
	FromCache     bool          `json:"fromCache"`
	ProxyUsed     string        `json:"proxyUsed,omitempty"`
	BytesServed   int64         `json:"bytesServed"`
	IsSimpleAPI   bool          `json:"isSimpleAPI"`
	IsPackageFile bool          `json:"isPackageFile"`
	Timestamp     time.Time     `json:"timestamp"`
	UserAgent     string        `json:"userAgent,omitempty"` // pip 클라이언트 정보
}

// ProxyConfig PIP 프록시 설정 인터페이스 (의존성 추상화)
type ProxyConfig interface {
	// GetProxies 프록시 목록 반환
	GetProxies() []config.PipProxyServer

	// GetCacheConfig 캐시 설정 반환
	GetCacheConfig() CacheConfig

	// IsEnabled 프록시 활성화 여부 확인
	IsEnabled() bool

	// GetPath 프록시 경로 반환
	GetPath() string

	// IsCacheEnabled 캐시 사용 여부 확인
	IsCacheEnabled() bool
}

// CacheConfig 캐시 설정
type CacheConfig struct {
	Enabled        bool          `json:"enabled"`
	BaseDir        string        `json:"baseDir"`
	TTL            time.Duration `json:"ttl"`
	MaxSizeMB      int64         `json:"maxSizeMB"`
	CleanupHours   int           `json:"cleanupHours"`
	ChecksumVerify bool          `json:"checksumVerify"` // 패키지 체크섬 검증 여부
}

// DefaultProxyConfig config.PipProxyConfig의 어댑터 (기본 구현)
type DefaultProxyConfig struct {
	config  *config.PipProxyConfig
	baseDir string
}

// NewDefaultProxyConfig DefaultProxyConfig 생성자
func NewDefaultProxyConfig(config *config.PipProxyConfig, baseDir string) ProxyConfig {
	return &DefaultProxyConfig{
		config:  config,
		baseDir: baseDir,
	}
}

// GetProxies 프록시 목록 반환
func (c *DefaultProxyConfig) GetProxies() []config.PipProxyServer {
	if c.config == nil || c.config.Proxies == nil {
		return []config.PipProxyServer{}
	}

	return c.config.Proxies
}

// GetCacheConfig 캐시 설정 반환
func (c *DefaultProxyConfig) GetCacheConfig() CacheConfig {
	return CacheConfig{
		Enabled:        true,
		BaseDir:        c.baseDir,
		TTL:            24 * time.Hour, // 24시간 캐시 (패키지 파일은 불변)
		MaxSizeMB:      10000,          // 10GB 기본 제한 (패키지 파일이 클 수 있음)
		CleanupHours:   12,             // 12시간마다 정리
		ChecksumVerify: true,           // 패키지 무결성 검증 활성화
	}
}

// IsEnabled 프록시 활성화 여부 확인
func (c *DefaultProxyConfig) IsEnabled() bool {
	if c.config == nil || c.config.Proxies == nil {
		return false
	}

	// 프록시가 하나 이상 설정되어 있어야 함
	return len(c.config.Proxies) > 0
}

// GetPath 프록시 경로 반환
func (c *DefaultProxyConfig) GetPath() string {
	if c.config == nil {
		return "pip"
	}
	return c.config.Path
}

// IsCacheEnabled 캐시 사용 여부 확인
func (c *DefaultProxyConfig) IsCacheEnabled() bool {
	if c.config == nil {
		return true // 기본값은 캐시 사용
	}
	return c.config.UseCache
}
