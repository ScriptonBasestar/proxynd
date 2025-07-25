package docker

import (
	"context"
	"net/http"
)

// RegistryHandler Docker 레지스트리 요청 처리 인터페이스
type RegistryHandler interface {
	// Handle Docker 레지스트리 요청을 처리하고 응답 반환
	Handle(ctx context.Context, request *RegistryRequest) (*ManifestResponse, error)

	// HandleV2Base Docker Registry v2 베이스 엔드포인트 처리 (/v2/)
	HandleV2Base(ctx context.Context) (*ManifestResponse, error)

	// ValidateRequest 요청 유효성 검증 (경로, 메서드, 헤더)
	ValidateRequest(ctx context.Context, request *RegistryRequest) error

	// GetSupportedOperations 지원하는 작업 목록 반환 (manifest, blob, tags, catalog)
	GetSupportedOperations() []string
}

// ManifestManager Docker 매니페스트 관리 인터페이스
type ManifestManager interface {
	// GetManifest 매니페스트 조회 (캐시 우선, 실패 시 레지스트리에서 다운로드)
	GetManifest(ctx context.Context, repository, reference string) (*ManifestResponse, error)

	// ValidateManifest 매니페스트 스키마 및 다이제스트 검증
	ValidateManifest(ctx context.Context, data []byte, expectedDigest string) error

	// DetectManifestType 매니페스트 타입 감지 (v1, v2, list)
	DetectManifestType(data []byte) string

	// ProcessManifestHeaders 매니페스트 응답 헤더 처리 및 설정
	ProcessManifestHeaders(manifest *ManifestResponse, headers http.Header) error

	// ShouldCacheManifest 매니페스트 캐시 여부 결정 (latest 태그 제외)
	ShouldCacheManifest(repository, reference string) bool

	// ExtractDigest 매니페스트에서 SHA256 다이제스트 추출
	ExtractDigest(data []byte) (string, error)

	// IsMultiPlatform 멀티 플랫폼 매니페스트 여부 확인
	IsMultiPlatform(data []byte) bool
}

// BlobManager Docker blob 관리 인터페이스
type BlobManager interface {
	// GetBlob blob 데이터 조회 (레이어, 설정 파일 등)
	GetBlob(ctx context.Context, repository, digest string) (*ManifestResponse, error)

	// ValidateBlobDigest blob 다이제스트 검증
	ValidateBlobDigest(ctx context.Context, data []byte, expectedDigest string) error

	// GetBlobInfo blob 메타데이터 정보 반환
	GetBlobInfo(ctx context.Context, repository, digest string) (*BlobReference, error)

	// ShouldCacheBlob blob 캐시 여부 결정 (일반적으로 항상 캐시)
	ShouldCacheBlob(repository, digest string) bool

	// ProcessBlobHeaders blob 응답 헤더 처리
	ProcessBlobHeaders(response *ManifestResponse, headers http.Header) error

	// CalculateBlobSize blob 크기 계산
	CalculateBlobSize(data []byte) int64

	// IsCompressed blob 압축 여부 확인
	IsCompressed(mediaType string) bool
}

// AuthenticationManager Docker 레지스트리 인증 관리 인터페이스
type AuthenticationManager interface {
	// GetAuthToken 레지스트리별 인증 토큰 조회 및 갱신
	GetAuthToken(ctx context.Context, registryURL, repository string) (*RegistryAuth, error)

	// RefreshToken 만료된 토큰 갱신
	RefreshToken(ctx context.Context, auth *RegistryAuth) error

	// ValidateToken 토큰 유효성 검증
	ValidateToken(ctx context.Context, auth *RegistryAuth) bool

	// ProcessAuthChallenge 인증 챌린지 처리 (401 응답)
	ProcessAuthChallenge(ctx context.Context, headers http.Header) (*RegistryAuth, error)

	// SetBasicAuth Basic 인증 설정
	SetBasicAuth(request *http.Request, username, password string) error

	// SetBearerAuth Bearer 토큰 인증 설정
	SetBearerAuth(request *http.Request, token string) error

	// ClearExpiredTokens 만료된 토큰 정리
	ClearExpiredTokens(ctx context.Context) error
}

// RegistryManager Docker 레지스트리 서버 관리 인터페이스
type RegistryManager interface {
	// SelectRegistry 요청에 적합한 레지스트리 선택 (라운드로빈, 가용성 기반)
	SelectRegistry(ctx context.Context, repository string) (*RegistryStatus, error)

	// CheckRegistryHealth 레지스트리 서버 상태 확인
	CheckRegistryHealth(ctx context.Context, registryURL string) (*RegistryStatus, error)

	// GetRegistryStatus 레지스트리 상태 정보 반환
	GetRegistryStatus(ctx context.Context, registryURL string) (*RegistryStatus, error)

	// MarkRegistryFailed 레지스트리 실패 마킹 (일시적 제외)
	MarkRegistryFailed(ctx context.Context, registryURL string, err error) error

	// BuildUpstreamURL 업스트림 레지스트리 URL 구성
	BuildUpstreamURL(registry *RegistryStatus, requestPath string) string

	// CopyRequestHeaders 요청 헤더 복사 (Accept, Authorization 등)
	CopyRequestHeaders(source http.Header, target *http.Request) error

	// GetDefaultRegistry 기본 레지스트리 반환
	GetDefaultRegistry(ctx context.Context) (*RegistryStatus, error)
}

// CacheManager Docker 캐시 관리 인터페이스
type CacheManager interface {
	// Get 캐시에서 데이터 조회
	Get(ctx context.Context, key string) (*CacheEntry, error)

	// Set 캐시에 데이터 저장 (TTL, 타입별 분리)
	Set(ctx context.Context, entry *CacheEntry) error

	// Delete 캐시 엔트리 삭제
	Delete(ctx context.Context, key string) error

	// GenerateCacheKey 캐시 키 생성 (레포지토리, 참조, 타입별)
	GenerateCacheKey(repository, reference, operation string) string

	// ValidateCacheEntry 캐시 엔트리 유효성 검증 (TTL, 무결성)
	ValidateCacheEntry(ctx context.Context, entry *CacheEntry) bool

	// CleanupExpired 만료된 캐시 엔트리 정리
	CleanupExpired(ctx context.Context) error

	// GetCacheStats 캐시 통계 정보 반환
	GetCacheStats(ctx context.Context) (*CacheStats, error)

	// SaveHeaders 매니페스트 헤더 정보 별도 저장
	SaveHeaders(ctx context.Context, key string, headers http.Header) error

	// LoadHeaders 저장된 헤더 정보 로드
	LoadHeaders(ctx context.Context, key string) (http.Header, error)
}

// MetricsCollector Docker 프록시 메트릭 수집 인터페이스
type MetricsCollector interface {
	// RecordRequest 요청 메트릭 기록
	RecordRequest(ctx context.Context, metrics *RequestMetrics) error

	// IncrementCounter 카운터 메트릭 증가
	IncrementCounter(ctx context.Context, name string, tags map[string]string) error

	// RecordDuration 응답 시간 기록
	RecordDuration(ctx context.Context, operation string, duration int64) error

	// RecordCacheHit 캐시 히트/미스 기록
	RecordCacheHit(ctx context.Context, hit bool, operation string) error

	// RecordRegistryUsage 레지스트리 사용량 기록
	RecordRegistryUsage(ctx context.Context, registryURL string, bytes int64) error

	// GetRequestStats 요청 통계 반환 (레포지토리별, 작업별)
	GetRequestStats(ctx context.Context, timeRange string) (*RequestStats, error)

	// GetPopularImages 인기 이미지 통계 반환
	GetPopularImages(ctx context.Context, limit int) ([]ImageStat, error)

	// RecordError 에러 메트릭 기록
	RecordError(ctx context.Context, errorType, operation string) error

	// GetErrorStats 에러 통계 반환
	GetErrorStats(ctx context.Context, timeRange string) (*ErrorStats, error)
}

// CacheStats 캐시 통계 정보
type CacheStats struct {
	TotalEntries     int64   `json:"totalEntries"`
	ManifestEntries  int64   `json:"manifestEntries"`
	BlobEntries      int64   `json:"blobEntries"`
	TotalSizeBytes   int64   `json:"totalSizeBytes"`
	HitRate          float64 `json:"hitRate"`
	MissRate         float64 `json:"missRate"`
	EvictionCount    int64   `json:"evictionCount"`
	OldestEntryAge   int64   `json:"oldestEntryAge"`   // seconds
	AverageEntrySize int64   `json:"averageEntrySize"` // bytes
}

// RequestStats 요청 통계 정보
type RequestStats struct {
	TotalRequests     int64                      `json:"totalRequests"`
	ManifestRequests  int64                      `json:"manifestRequests"`
	BlobRequests      int64                      `json:"blobRequests"`
	CacheHitRate      float64                    `json:"cacheHitRate"`
	AverageResponseMs float64                    `json:"averageResponseMs"`
	RepositoryStats   map[string]*RepositoryStat `json:"repositoryStats"`
	RegistryUsage     map[string]int64           `json:"registryUsage"`
	ErrorCount        int64                      `json:"errorCount"`
	BytesServed       int64                      `json:"bytesServed"`
	PopularOperations []string                   `json:"popularOperations"`
}

// RepositoryStat 레포지토리별 통계
type RepositoryStat struct {
	Name           string   `json:"name"`
	RequestCount   int64    `json:"requestCount"`
	BytesServed    int64    `json:"bytesServed"`
	CacheHitRate   float64  `json:"cacheHitRate"`
	LastAccessTime int64    `json:"lastAccessTime"` // unix timestamp
	PopularTags    []string `json:"popularTags"`
}

// ImageStat 이미지 통계
type ImageStat struct {
	Repository      string  `json:"repository"`
	Tag             string  `json:"tag"`
	PullCount       int64   `json:"pullCount"`
	LastPulled      int64   `json:"lastPulled"` // unix timestamp
	SizeBytes       int64   `json:"sizeBytes"`
	LayerCount      int     `json:"layerCount"`
	PopularityScore float64 `json:"popularityScore"`
}

// ErrorStats 에러 통계
type ErrorStats struct {
	TotalErrors       int64            `json:"totalErrors"`
	ErrorsByType      map[string]int64 `json:"errorsByType"`
	ErrorsByOperation map[string]int64 `json:"errorsByOperation"`
	ErrorsByRegistry  map[string]int64 `json:"errorsByRegistry"`
	RecentErrors      []string         `json:"recentErrors"`
	ErrorRate         float64          `json:"errorRate"`
}
