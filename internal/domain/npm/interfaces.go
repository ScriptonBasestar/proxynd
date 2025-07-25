package npm

import (
	"context"
)

// PackageHandler NPM 패키지 요청 처리 인터페이스 (도메인 로직)
type PackageHandler interface {
	// Handle 패키지 요청 처리
	Handle(ctx context.Context, request *PackageRequest) (*PackageResponse, error)

	// GetPackageMetadata 패키지 메타데이터 조회
	GetPackageMetadata(ctx context.Context, packagePath string) (*PackageMetadata, error)
}

// ProxyManager 프록시 서버 관리 인터페이스
type ProxyManager interface {
	// GetNextProxy 다음 사용 가능한 프록시 서버 반환
	GetNextProxy() (*ProxyStatus, error)

	// CheckProxyHealth 프록시 서버 상태 확인
	CheckProxyHealth(ctx context.Context, proxy *ProxyStatus) error

	// GetProxyStats 프록시 통계 조회
	GetProxyStats() ([]*ProxyStatus, error)

	// MarkProxyFailed 프록시를 실패로 표시
	MarkProxyFailed(proxyURL string, err error)
}

// CacheManager 캐시 관리 인터페이스
type CacheManager interface {
	// Get 캐시에서 패키지 조회
	Get(ctx context.Context, key string) (*CacheEntry, error)

	// Set 패키지를 캐시에 저장
	Set(ctx context.Context, key string, data []byte, contentType string, isMetadata bool) error

	// Delete 캐시에서 패키지 삭제
	Delete(ctx context.Context, key string) error

	// GenerateKey 캐시 키 생성
	GenerateKey(packagePath string) string

	// Cleanup 오래된 캐시 파일 정리
	Cleanup(ctx context.Context) error

	// GetStats 캐시 통계 조회
	GetStats(ctx context.Context) (*CacheStats, error)
}

// MetadataProcessor NPM 메타데이터 처리 인터페이스
type MetadataProcessor interface {
	// IsMetadataRequest 메타데이터 요청인지 확인
	IsMetadataRequest(packagePath string) bool

	// RewriteMetadata 메타데이터의 URL을 프록시 URL로 재작성
	RewriteMetadata(data []byte, baseURL, packagePath string) []byte

	// GetContentType 패키지 경로에서 Content-Type 결정
	GetContentType(packagePath string) string

	// GetDisposition Content-Disposition 헤더 생성
	GetDisposition(packagePath string, isMetadata bool) string
}

// MetricsCollector 메트릭 수집 인터페이스
type MetricsCollector interface {
	// RecordRequest 요청 메트릭 기록
	RecordRequest(ctx context.Context, metrics *RequestMetrics) error

	// GetRequestStats 요청 통계 조회
	GetRequestStats(ctx context.Context) (*RequestStats, error)
}

// CacheStats 캐시 통계
type CacheStats struct {
	TotalEntries    int64   `json:"totalEntries"`
	TotalSizeMB     int64   `json:"totalSizeMB"`
	HitRate         float64 `json:"hitRate"`
	MetadataEntries int64   `json:"metadataEntries"` // 메타데이터 캐시 개수
	TarballEntries  int64   `json:"tarballEntries"`  // 패키지 파일 캐시 개수
	OldestEntry     string  `json:"oldestEntry"`
	NewestEntry     string  `json:"newestEntry"`
	CleanupCount    int64   `json:"cleanupCount"`
	LastCleanup     string  `json:"lastCleanup"`
}

// RequestStats 요청 통계
type RequestStats struct {
	TotalRequests    int64             `json:"totalRequests"`
	CacheHitRate     float64           `json:"cacheHitRate"`
	AvgResponseTime  float64           `json:"avgResponseTime"` // milliseconds
	TotalBytesServed int64             `json:"totalBytesServed"`
	MetadataRequests int64             `json:"metadataRequests"` // 메타데이터 요청 수
	TarballRequests  int64             `json:"tarballRequests"`  // 패키지 파일 요청 수
	TopPackages      []PackageStats    `json:"topPackages"`
	ProxyUsage       []ProxyUsageStats `json:"proxyUsage"`
}

// PackageStats 패키지별 통계
type PackageStats struct {
	PackageName   string `json:"packageName"`
	RequestCount  int64  `json:"requestCount"`
	TotalBytes    int64  `json:"totalBytes"`
	LastRequested string `json:"lastRequested"`
	IsScoped      bool   `json:"isScoped"` // @scope/package 형태인지
}

// ProxyUsageStats 프록시 사용 통계
type ProxyUsageStats struct {
	ProxyURL     string  `json:"proxyURL"`
	RequestCount int64   `json:"requestCount"`
	SuccessRate  float64 `json:"successRate"`
	AvgLatency   float64 `json:"avgLatency"` // milliseconds
}
