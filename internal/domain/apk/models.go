package apk

import (
	"time"

	"proxynd/internal/config"
)

// PackageRequest APK 패키지 요청
type PackageRequest struct {
	PackagePath string            `json:"packagePath"` // 패키지 경로 (v3.18/main/x86_64/package.apk 등)
	Headers     map[string]string `json:"headers"`     // HTTP 헤더
	Method      string            `json:"method"`      // HTTP 메서드
	BaseURL     string            `json:"baseUrl"`     // 프록시 서버 기본 URL
}

// PackageResponse APK 패키지 응답
type PackageResponse struct {
	Data        []byte            `json:"data"`
	ContentType string            `json:"contentType"`
	Headers     map[string]string `json:"headers"`
	StatusCode  int               `json:"statusCode"`
	FromCache   bool              `json:"fromCache"`
	ProxyUsed   string            `json:"proxyUsed,omitempty"`   // 사용된 프록시 서버
	IsApkFile   bool              `json:"isApkFile,omitempty"`   // APK 파일 여부
	IsIndex     bool              `json:"isIndex,omitempty"`     // APKINDEX 파일 여부
	IsSignature bool              `json:"isSignature,omitempty"` // 서명 파일 여부
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
	IsApkFile    bool          `json:"isApkFile"`   // APK 파일 여부
	IsIndex      bool          `json:"isIndex"`     // APKINDEX 파일 여부
	IsSignature  bool          `json:"isSignature"` // 서명 파일 여부
}

// SearchFilters APK 패키지 검색 필터
type SearchFilters struct {
	Name         string   `json:"name,omitempty"`
	Architecture []string `json:"architecture,omitempty"`
	Branch       []string `json:"branch,omitempty"`
	Component    []string `json:"component,omitempty"`
	Description  string   `json:"description,omitempty"`
	Maintainer   string   `json:"maintainer,omitempty"`
}

// RepositoryIndex APK 리포지토리 인덱스
type RepositoryIndex struct {
	Architecture string        `json:"architecture"`
	Branch       string        `json:"branch"`
	Component    string        `json:"component"`
	Packages     []PackageInfo `json:"packages"`
	LastModified time.Time     `json:"lastModified"`
	IndexFile    string        `json:"indexFile"`
	Size         int64         `json:"size"`
}

// RepositoryInfo APK 리포지토리 정보
type RepositoryInfo struct {
	Architecture string    `json:"architecture"`
	Branch       string    `json:"branch"`    // v3.18, edge 등
	Component    string    `json:"component"` // main, community, testing
	IndexFile    string    `json:"indexFile"` // APKINDEX.tar.gz
	LastModified time.Time `json:"lastModified"`
	Size         int64     `json:"size"`
}

// Package APK 패키지 기본 구조체
type Package struct {
	Name         string `json:"name"`
	Version      string `json:"version"`
	Architecture string `json:"architecture"`
	Repository   string `json:"repository"`
	Branch       string `json:"branch"`
	Component    string `json:"component"`
}

// PackageInfo APK 패키지 정보
type PackageInfo struct {
	Name          string   `json:"name"`
	Version       string   `json:"version"`
	Architecture  string   `json:"architecture"`
	Description   string   `json:"description"`
	Homepage      string   `json:"homepage"`
	Size          int64    `json:"size"`
	InstalledSize int64    `json:"installedSize"`
	License       string   `json:"license"`
	Origin        string   `json:"origin"`
	Maintainer    string   `json:"maintainer"`
	BuildTime     string   `json:"buildTime"`
	CommitID      string   `json:"commitId"`
	PackagerID    string   `json:"packageId"`
	Dependencies  []string `json:"dependencies,omitempty"`
	Provides      []string `json:"provides,omitempty"`
	InstallIf     []string `json:"installIf,omitempty"`
}

// SignatureInfo APK 서명 정보
type SignatureInfo struct {
	IsValid        bool      `json:"isValid"`
	SignatureFile  string    `json:"signatureFile"`
	KeyFingerprint string    `json:"keyFingerprint"`
	KeyName        string    `json:"keyName"`
	Algorithm      string    `json:"algorithm"`
	ValidFrom      time.Time `json:"validFrom"`
	ValidTo        time.Time `json:"validTo"`
	Error          string    `json:"error,omitempty"`
}

// RequestMetrics APK 요청 메트릭
type RequestMetrics struct {
	Path         string    `json:"path"`
	Architecture string    `json:"architecture"`
	Branch       string    `json:"branch"`
	Component    string    `json:"component"`
	Method       string    `json:"method"`
	StatusCode   int       `json:"statusCode"`
	ResponseTime int64     `json:"responseTime"` // 밀리초
	CacheHit     bool      `json:"cacheHit"`
	ProxyUsed    string    `json:"proxyUsed"`
	UserAgent    string    `json:"userAgent"`
	ClientIP     string    `json:"clientIP"`
	RequestTime  time.Time `json:"requestTime"`
	FileSize     int64     `json:"fileSize"`
	IsApkFile    bool      `json:"isApkFile"`
	IsIndex      bool      `json:"isIndex"`
	IsSignature  bool      `json:"isSignature"`
}

// ProxyConfig APK 프록시 설정을 추상화하는 인터페이스
type ProxyConfig interface {
	GetPath() string
	GetUseCache() bool
	GetProxies() []config.ApkProxy
	GetCacheTTL() time.Duration
	GetMaxCacheSize() int64
	GetVerificationEnabled() bool
	GetVerificationKeyDirectory() string
	GetVerificationFailOnInvalid() bool
	GetMirrorSelectionEnabled() bool
	GetMirrorSelectionConfig() config.ApkMirrorSelectionConfig
	GetApkFileTTL() time.Duration
	GetIndexFileTTL() time.Duration
	GetSignatureFileTTL() time.Duration
}

// defaultProxyConfig config.ApkProxyConfig를 ProxyConfig 인터페이스로 래핑
type defaultProxyConfig struct {
	config  *config.ApkProxyConfig
	baseDir string
}

// NewDefaultProxyConfig 기본 프록시 설정 어댑터 생성
func NewDefaultProxyConfig(config *config.ApkProxyConfig, baseDir string) ProxyConfig {
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

func (c *defaultProxyConfig) GetProxies() []config.ApkProxy {
	return c.config.Proxies
}

func (c *defaultProxyConfig) GetCacheTTL() time.Duration {
	return 3600 * time.Second // 기본 1시간
}

func (c *defaultProxyConfig) GetMaxCacheSize() int64 {
	return 10 * 1024 * 1024 * 1024 // 기본 10GB
}

func (c *defaultProxyConfig) GetVerificationEnabled() bool {
	return c.config.Verification.Enabled
}

func (c *defaultProxyConfig) GetVerificationKeyDirectory() string {
	return c.config.Verification.KeyDirectory
}

func (c *defaultProxyConfig) GetVerificationFailOnInvalid() bool {
	return c.config.Verification.FailOnInvalid
}

func (c *defaultProxyConfig) GetMirrorSelectionEnabled() bool {
	return c.config.MirrorSelection.Enabled
}

func (c *defaultProxyConfig) GetMirrorSelectionConfig() config.ApkMirrorSelectionConfig {
	return c.config.MirrorSelection
}

func (c *defaultProxyConfig) GetApkFileTTL() time.Duration {
	return 24 * time.Hour // APK 파일은 24시간
}

func (c *defaultProxyConfig) GetIndexFileTTL() time.Duration {
	return 1 * time.Hour // APKINDEX 파일은 1시간
}

func (c *defaultProxyConfig) GetSignatureFileTTL() time.Duration {
	return 6 * time.Hour // 서명 파일은 6시간
}
