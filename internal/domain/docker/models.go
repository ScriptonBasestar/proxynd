package docker

import (
	"time"

	"proxynd/configs"
)

// RegistryRequest Docker 레지스트리 요청
type RegistryRequest struct {
	Path       string            `json:"path"`       // 요청 경로 (v2/library/ubuntu/manifests/latest)
	Method     string            `json:"method"`     // HTTP 메서드 (GET, HEAD, PUT, POST, DELETE)
	Headers    map[string]string `json:"headers"`    // HTTP 헤더
	Repository string            `json:"repository"` // 이미지 레포지토리 (library/ubuntu)
	Reference  string            `json:"reference"`  // 태그 또는 다이제스트 (latest, sha256:abc123)
	Operation  string            `json:"operation"`  // 작업 타입 (manifest, blob, tags, catalog)
	BaseURL    string            `json:"baseUrl"`    // 프록시 서버 기본 URL
	ClientIP   string            `json:"clientIP"`   // 클라이언트 IP
}

// ManifestResponse Docker 매니페스트 응답
type ManifestResponse struct {
	Data            []byte            `json:"data"`
	ContentType     string            `json:"contentType"`
	Digest          string            `json:"digest"` // 매니페스트 SHA256 다이제스트
	Headers         map[string]string `json:"headers"`
	StatusCode      int               `json:"statusCode"`
	FromCache       bool              `json:"fromCache"`
	RegistryUsed    string            `json:"registryUsed,omitempty"`    // 사용된 레지스트리
	SchemaVersion   int               `json:"schemaVersion,omitempty"`   // 매니페스트 스키마 버전 (1, 2)
	MediaType       string            `json:"mediaType,omitempty"`       // 매니페스트 미디어 타입
	IsMultiPlatform bool              `json:"isMultiPlatform,omitempty"` // 멀티 플랫폼 매니페스트 여부
}

// BlobReference Docker blob 참조
type BlobReference struct {
	Digest      string `json:"digest"`      // SHA256 다이제스트
	Size        int64  `json:"size"`        // 파일 크기
	MediaType   string `json:"mediaType"`   // 미디어 타입
	ContentType string `json:"contentType"` // HTTP Content-Type
	Repository  string `json:"repository"`  // 소속 레포지토리
	IsConfig    bool   `json:"isConfig"`    // 이미지 설정 blob 여부
	IsLayer     bool   `json:"isLayer"`     // 이미지 레이어 blob 여부
}

// LayerInfo Docker 레이어 정보
type LayerInfo struct {
	Digest       string    `json:"digest"`
	Size         int64     `json:"size"`
	MediaType    string    `json:"mediaType"`
	CreatedAt    time.Time `json:"createdAt"`
	Command      string    `json:"command,omitempty"`  // 레이어 생성 명령
	IsCompressed bool      `json:"isCompressed"`       // 압축 여부
	ParentID     string    `json:"parentID,omitempty"` // 부모 레이어 ID
	DiffID       string    `json:"diffID,omitempty"`   // 비압축 다이제스트
}

// RepositoryInfo Docker 레포지토리 정보
type RepositoryInfo struct {
	Name         string            `json:"name"`         // 레포지토리명 (library/ubuntu)
	Namespace    string            `json:"namespace"`    // 네임스페이스 (library)
	ImageName    string            `json:"imageName"`    // 이미지명 (ubuntu)
	Tags         []string          `json:"tags"`         // 사용 가능한 태그 목록
	LastModified time.Time         `json:"lastModified"` // 마지막 수정 시간
	Size         int64             `json:"size"`         // 전체 크기
	LayerCount   int               `json:"layerCount"`   // 레이어 수
	Manifests    map[string]string `json:"manifests"`    // 태그 또는 다이제스트 매핑
	IsOfficial   bool              `json:"isOfficial"`   // 공식 이미지 여부
}

// RegistryAuth Docker 레지스트리 인증 정보
type RegistryAuth struct {
	Type         string    `json:"type"`         // 인증 타입 (basic, bearer, none)
	Username     string    `json:"username"`     // 사용자명
	Password     string    `json:"password"`     // 비밀번호
	Token        string    `json:"token"`        // Bearer 토큰
	RefreshToken string    `json:"refreshToken"` // 리프레시 토큰
	ExpiresAt    time.Time `json:"expiresAt"`    // 토큰 만료 시간
	Realm        string    `json:"realm"`        // 인증 영역
	Service      string    `json:"service"`      // 서비스명
	Scope        string    `json:"scope"`        // 인증 범위
}

// RegistryStatus 레지스트리 서버 상태 정보
type RegistryStatus struct {
	Name         string    `json:"name"`
	URL          string    `json:"url"`
	Available    bool      `json:"available"`
	LastCheck    time.Time `json:"lastCheck"`
	Error        string    `json:"error,omitempty"`
	ResponseTime float64   `json:"responseTime"` // milliseconds
	APIVersion   string    `json:"apiVersion"`   // v1, v2
	Features     []string  `json:"features"`     // 지원 기능 목록
}

// CacheEntry Docker 캐시 엔트리
type CacheEntry struct {
	Key          string        `json:"key"`
	Path         string        `json:"path"`
	ContentType  string        `json:"contentType"`
	Size         int64         `json:"size"`
	CreatedAt    time.Time     `json:"createdAt"`
	LastAccessed time.Time     `json:"lastAccessed"`
	TTL          time.Duration `json:"ttl"`
	IsManifest   bool          `json:"isManifest"` // 매니페스트 캐시 여부
	IsBlob       bool          `json:"isBlob"`     // blob 캐시 여부
	Digest       string        `json:"digest"`     // SHA256 다이제스트
	Repository   string        `json:"repository"` // 소속 레포지토리
	Reference    string        `json:"reference"`  // 태그 또는 다이제스트
	Headers      []byte        `json:"headers"`    // 저장된 헤더 (매니페스트용)
}

// RequestMetrics Docker 요청 메트릭
type RequestMetrics struct {
	Repository    string        `json:"repository"`
	Reference     string        `json:"reference"`
	Operation     string        `json:"operation"` // manifest, blob, tags, catalog
	Method        string        `json:"method"`
	StatusCode    int           `json:"statusCode"`
	Duration      time.Duration `json:"duration"`
	FromCache     bool          `json:"fromCache"`
	RegistryUsed  string        `json:"registryUsed,omitempty"`
	BytesServed   int64         `json:"bytesServed"`
	Timestamp     time.Time     `json:"timestamp"`
	ClientIP      string        `json:"clientIP"`
	UserAgent     string        `json:"userAgent,omitempty"`
	Digest        string        `json:"digest,omitempty"`        // 매니페스트/blob 다이제스트
	ContentLength int64         `json:"contentLength,omitempty"` // 전송된 컨텐츠 크기
}

// ProxyConfig Docker 프록시 설정 인터페이스 (의존성 추상화)
type ProxyConfig interface {
	// GetRegistries 레지스트리 목록 반환
	GetRegistries() []configs.DockerProxyServer

	// GetRegistryConfig 특정 레지스트리 설정 반환
	GetRegistryConfig(registryName string) *configs.DockerProxyRegistryConfig

	// GetCacheConfig 캐시 설정 반환
	GetCacheConfig() CacheConfig

	// IsEnabled 프록시 활성화 여부 확인
	IsEnabled() bool

	// GetPath 프록시 경로 반환
	GetPath() string

	// IsCacheEnabled 캐시 사용 여부 확인
	IsCacheEnabled() bool

	// GetDefaultRegistry 기본 레지스트리 반환
	GetDefaultRegistry() *configs.DockerProxyServer
}

// CacheConfig Docker 캐시 설정
type CacheConfig struct {
	Enabled         bool          `json:"enabled"`
	BaseDir         string        `json:"baseDir"`
	ManifestTTL     time.Duration `json:"manifestTTL"` // 매니페스트 TTL (짧음)
	BlobTTL         time.Duration `json:"blobTTL"`     // blob TTL (김)
	MaxSizeMB       int64         `json:"maxSizeMB"`
	CleanupHours    int           `json:"cleanupHours"`
	LayerDedup      bool          `json:"layerDedup"`      // 레이어 중복 제거
	CompressionType string        `json:"compressionType"` // 캐시 압축 타입
}

// DefaultProxyConfig configs.DockerProxyConfig의 어댑터 (기본 구현)
type DefaultProxyConfig struct {
	config  *configs.DockerProxyConfig
	baseDir string
}

// NewDefaultProxyConfig DefaultProxyConfig 생성자
func NewDefaultProxyConfig(config *configs.DockerProxyConfig, baseDir string) ProxyConfig {
	return &DefaultProxyConfig{
		config:  config,
		baseDir: baseDir,
	}
}

// GetRegistries 레지스트리 목록 반환
func (c *DefaultProxyConfig) GetRegistries() []configs.DockerProxyServer {
	if c.config == nil || c.config.Proxies == nil {
		return []configs.DockerProxyServer{}
	}

	return c.config.Proxies
}

// GetRegistryConfig 특정 레지스트리 설정 반환
func (c *DefaultProxyConfig) GetRegistryConfig(registryName string) *configs.DockerProxyRegistryConfig {
	if c.config == nil || c.config.Registries == nil {
		return nil
	}

	if registryConfig, exists := c.config.Registries[registryName]; exists {
		return &registryConfig
	}

	return nil
}

// GetCacheConfig 캐시 설정 반환
func (c *DefaultProxyConfig) GetCacheConfig() CacheConfig {
	return CacheConfig{
		Enabled:         true,
		BaseDir:         c.baseDir,
		ManifestTTL:     6 * time.Hour,      // 매니페스트는 비교적 짧은 TTL
		BlobTTL:         7 * 24 * time.Hour, // blob은 불변이므로 긴 TTL
		MaxSizeMB:       50000,              // 50GB 기본 제한 (이미지는 클 수 있음)
		CleanupHours:    6,                  // 6시간마다 정리
		LayerDedup:      true,               // 레이어 중복 제거 활성화
		CompressionType: "gzip",             // gzip 압축 사용
	}
}

// IsEnabled 프록시 활성화 여부 확인
func (c *DefaultProxyConfig) IsEnabled() bool {
	if c.config == nil || c.config.Proxies == nil {
		return false
	}

	// 레지스트리가 하나 이상 설정되어 있어야 함
	return len(c.config.Proxies) > 0
}

// GetPath 프록시 경로 반환
func (c *DefaultProxyConfig) GetPath() string {
	if c.config == nil {
		return "docker"
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

// GetDefaultRegistry 기본 레지스트리 반환
func (c *DefaultProxyConfig) GetDefaultRegistry() *configs.DockerProxyServer {
	registries := c.GetRegistries()
	if len(registries) > 0 {
		return &registries[0] // 첫 번째 레지스트리를 기본으로 사용
	}
	return nil
}
