package yum

import (
	"time"

	"proxynd/internal/config"
)

// YUM metadata types
const (
	MetadataTypePrimary   = "primary"
	MetadataTypeFilelists = "filelists"
	MetadataTypeOther     = "other"
	MetadataTypeRepomd    = "repomd"
	MetadataTypeUpdateinfo = "updateinfo"
	MetadataTypeModules   = "modules"
)

// PackageRequest YUM 패키지 요청
type PackageRequest struct {
	PackagePath string            `json:"packagePath"` // 패키지 경로 (repository/package.rpm 등)
	Headers     map[string]string `json:"headers"`     // HTTP 헤더
	Method      string            `json:"method"`      // HTTP 메서드
	BaseURL     string            `json:"baseUrl"`     // 프록시 서버 기본 URL
}

// PackageResponse YUM 패키지 응답
type PackageResponse struct {
	Data        []byte            `json:"data"`
	ContentType string            `json:"contentType"`
	Headers     map[string]string `json:"headers"`
	StatusCode  int               `json:"statusCode"`
	FromCache   bool              `json:"fromCache"`
	ProxyUsed   string            `json:"proxyUsed,omitempty"`  // 사용된 프록시 서버
	IsRepoMeta  bool              `json:"isRepoMeta,omitempty"` // 리포지토리 메타데이터 여부
	IsRpmFile   bool              `json:"isRpmFile,omitempty"`  // RPM 파일 여부
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
	IsRepoMeta   bool          `json:"isRepoMeta"` // 리포지토리 메타데이터 파일 여부
	IsRpmFile    bool          `json:"isRpmFile"`  // RPM 파일 여부
}

// Package YUM 패키지 기본 구조체
type Package struct {
	Name         string `json:"name"`
	Version      string `json:"version"`
	Release      string `json:"release"`
	Architecture string `json:"architecture"`
	Repository   string `json:"repository"`
	Summary      string `json:"summary"`
	Size         int64  `json:"size"`
}

// SearchFilters YUM 패키지 검색 필터
type SearchFilters struct {
	Name         string   `json:"name,omitempty"`
	Architecture []string `json:"architecture,omitempty"`
	Repository   []string `json:"repository,omitempty"`
	Group        string   `json:"group,omitempty"`
	Summary      string   `json:"summary,omitempty"`
}

// RepositoryMetadata YUM 리포지토리 메타데이터 (별칭)
type RepositoryMetadata = RepoMetadata

// RepoMetadata YUM 리포지토리 메타데이터
type RepoMetadata struct {
	Repository   string    `json:"repository"`
	MetadataType string    `json:"metadataType"` // repomd, primary, filelists, other
	Path         string    `json:"path"`
	Size         int64     `json:"size"`
	Checksum     string    `json:"checksum"`
	LastModified time.Time `json:"lastModified"`
}

// PackageInfo RPM 패키지 정보
type PackageInfo struct {
	Name         string   `json:"name"`
	Version      string   `json:"version"`
	Release      string   `json:"release"`
	Architecture string   `json:"architecture"`
	Summary      string   `json:"summary"`
	Description  string   `json:"description"`
	Size         int64    `json:"size"`
	BuildTime    string   `json:"buildTime"`
	Vendor       string   `json:"vendor"`
	License      string   `json:"license"`
	Group        string   `json:"group"`
	URL          string   `json:"url"`
	Requires     []string `json:"requires,omitempty"`
	Provides     []string `json:"provides,omitempty"`
	Conflicts    []string `json:"conflicts,omitempty"`
	Obsoletes    []string `json:"obsoletes,omitempty"`
}

// RequestMetrics YUM 요청 메트릭
type RequestMetrics struct {
	Path         string    `json:"path"`
	Repository   string    `json:"repository"`
	Method       string    `json:"method"`
	StatusCode   int       `json:"statusCode"`
	ResponseTime int64     `json:"responseTime"` // 밀리초
	CacheHit     bool      `json:"cacheHit"`
	ProxyUsed    string    `json:"proxyUsed"`
	UserAgent    string    `json:"userAgent"`
	ClientIP     string    `json:"clientIP"`
	RequestTime  time.Time `json:"requestTime"`
	FileSize     int64     `json:"fileSize"`
	IsRepoMeta   bool      `json:"isRepoMeta"`
	IsRpmFile    bool      `json:"isRpmFile"`
}

// ProxyConfig YUM 프록시 설정을 추상화하는 인터페이스
type ProxyConfig interface {
	GetPath() string
	GetUseCache() bool
	GetProxies() []config.YumProxy
	GetCacheTTL() time.Duration
	GetMaxCacheSize() int64
	GetRepoMetadataTTL() time.Duration
	GetRpmFileTTL() time.Duration
}

// defaultProxyConfig config.YumProxyConfig를 ProxyConfig 인터페이스로 래핑
type defaultProxyConfig struct {
	config  *config.YumProxySettings
	baseDir string
}

// NewDefaultProxyConfig 기본 프록시 설정 어댑터 생성
func NewDefaultProxyConfig(config *config.YumProxySettings, baseDir string) ProxyConfig {
	return &defaultProxyConfig{
		config:  config,
		baseDir: baseDir,
	}
}

func (c *defaultProxyConfig) GetPath() string {
	return c.config.Path
}

func (c *defaultProxyConfig) GetUseCache() bool {
	return c.config.UseCache
}

func (c *defaultProxyConfig) GetProxies() []config.YumProxy {
	return c.config.Proxies
}

func (c *defaultProxyConfig) GetCacheTTL() time.Duration {
	return 3600 * time.Second // 기본 1시간
}

func (c *defaultProxyConfig) GetMaxCacheSize() int64 {
	return 10 * 1024 * 1024 * 1024 // 기본 10GB
}

func (c *defaultProxyConfig) GetRepoMetadataTTL() time.Duration {
	return 30 * time.Minute // 리포지토리 메타데이터는 30분
}

func (c *defaultProxyConfig) GetRpmFileTTL() time.Duration {
	return 24 * time.Hour // RPM 파일은 24시간
}
