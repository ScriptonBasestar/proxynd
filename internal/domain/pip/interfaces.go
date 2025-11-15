package pip

import (
	"context"
)

// PackageHandler PIP 패키지 요청 처리 인터페이스 (도메인 로직)
type PackageHandler interface {
	// Handle 패키지 요청 처리
	Handle(ctx context.Context, request *PackageRequest) (*PackageResponse, error)

	// GetPackageMetadata 패키지 메타데이터 조회
	GetPackageMetadata(ctx context.Context, packagePath string) (*PackageMetadata, error)

	// ValidatePackage 패키지 무결성 검증
	ValidatePackage(ctx context.Context, data []byte, expectedChecksum string) error
}

// IndexManager PyPI 인덱스 서버 관리 인터페이스
type IndexManager interface {
	// GetNextIndex 다음 사용 가능한 인덱스 서버 반환
	GetNextIndex() (*IndexStatus, error)

	// CheckIndexHealth 인덱스 서버 상태 확인
	CheckIndexHealth(ctx context.Context, index *IndexStatus) error

	// GetIndexStats 인덱스 통계 조회
	GetIndexStats() ([]*IndexStatus, error)

	// MarkIndexFailed 인덱스를 실패로 표시
	MarkIndexFailed(indexURL string, err error)

	// BuildPackageURL 패키지 요청 URL 구성
	BuildPackageURL(baseURL, packagePath string) string
}

// CacheManager 캐시 관리 인터페이스
type CacheManager interface {
	// Get 캐시에서 패키지 조회
	Get(ctx context.Context, key string) (*CacheEntry, error)

	// GetData 캐시에서 패키지 데이터 조회
	GetData(ctx context.Context, key string) ([]byte, error)

	// Set 패키지를 캐시에 저장
	Set(ctx context.Context, key string, data []byte, contentType string, metadata *PackageMetadata) error

	// Delete 캐시에서 패키지 삭제
	Delete(ctx context.Context, key string) error

	// GenerateKey 캐시 키 생성
	GenerateKey(packagePath string) string

	// Cleanup 오래된 캐시 파일 정리
	Cleanup(ctx context.Context) error

	// GetStats 캐시 통계 조회
	GetStats(ctx context.Context) (*CacheStats, error)

	// VerifyChecksum 캐시된 파일의 체크섬 검증
	VerifyChecksum(ctx context.Context, filePath, expectedChecksum string) bool
}

// MetadataProcessor PIP 메타데이터 처리 인터페이스
type MetadataProcessor interface {
	// IsSimpleAPIRequest Simple API 요청인지 확인
	IsSimpleAPIRequest(packagePath string) bool

	// IsPackageFileRequest 패키지 파일 요청인지 확인
	IsPackageFileRequest(packagePath string) bool

	// RewriteSimpleAPI Simple API HTML의 URL을 프록시 URL로 재작성
	RewriteSimpleAPI(data []byte, baseURL, packagePath string) []byte

	// GetContentType 패키지 경로에서 Content-Type 결정
	GetContentType(packagePath, fileName string) string

	// GetDisposition Content-Disposition 헤더 생성
	GetDisposition(packagePath, fileName string, isSimpleAPI bool) string

	// ExtractChecksumFromURL URL에서 체크섬 정보 추출
	ExtractChecksumFromURL(packageURL string) string

	// GetFileExtension 파일 경로에서 확장자 추출
	GetFileExtension(fileName string) string
}

// MetricsCollector 메트릭 수집 인터페이스
type MetricsCollector interface {
	// RecordRequest 요청 메트릭 기록
	RecordRequest(ctx context.Context, metrics *RequestMetrics) error

	// GetRequestStats 요청 통계 조회
	GetRequestStats(ctx context.Context) (*RequestStats, error)

	// RecordCacheHit 캐시 히트 기록
	RecordCacheHit(ctx context.Context, packagePath string, size int64)

	// RecordCacheMiss 캐시 미스 기록
	RecordCacheMiss(ctx context.Context, packagePath string)
}

// CacheStats 캐시 통계
type CacheStats struct {
	TotalEntries       int64   `json:"totalEntries"`
	TotalSizeMB        int64   `json:"totalSizeMB"`
	HitRate            float64 `json:"hitRate"`
	SimpleAPIEntries   int64   `json:"simpleAPIEntries"`   // Simple API 캐시 개수
	PackageFileEntries int64   `json:"packageFileEntries"` // 패키지 파일 캐시 개수
	WheelEntries       int64   `json:"wheelEntries"`       // .whl 파일 개수
	SourceEntries      int64   `json:"sourceEntries"`      // .tar.gz 등 소스 파일 개수
	OldestEntry        string  `json:"oldestEntry"`
	NewestEntry        string  `json:"newestEntry"`
	CleanupCount       int64   `json:"cleanupCount"`
	LastCleanup        string  `json:"lastCleanup"`
	ChecksumVerified   int64   `json:"checksumVerified"` // 체크섬 검증된 파일 수
	ChecksumFailed     int64   `json:"checksumFailed"`   // 체크섬 검증 실패 파일 수
}

// RequestStats 요청 통계
type RequestStats struct {
	TotalRequests       int64             `json:"totalRequests"`
	CacheHitRate        float64           `json:"cacheHitRate"`
	AvgResponseTime     float64           `json:"avgResponseTime"` // milliseconds
	TotalBytesServed    int64             `json:"totalBytesServed"`
	SimpleAPIRequests   int64             `json:"simpleAPIRequests"`   // Simple API 요청 수
	PackageFileRequests int64             `json:"packageFileRequests"` // 패키지 파일 요청 수
	WheelDownloads      int64             `json:"wheelDownloads"`      // .whl 다운로드 수
	SourceDownloads     int64             `json:"sourceDownloads"`     // 소스 패키지 다운로드 수
	TopPackages         []PackageStats    `json:"topPackages"`
	IndexUsage          []IndexUsageStats `json:"indexUsage"`
	UserAgents          []UserAgentStats  `json:"userAgents"`
}

// PackageStats 패키지별 통계
type PackageStats struct {
	PackageName   string           `json:"packageName"`
	RequestCount  int64            `json:"requestCount"`
	TotalBytes    int64            `json:"totalBytes"`
	LastRequested string           `json:"lastRequested"`
	FileTypes     map[string]int64 `json:"fileTypes"` // "wheel", "sdist" 별 다운로드 수
}

// IndexUsageStats 인덱스 사용 통계
type IndexUsageStats struct {
	IndexURL     string  `json:"indexURL"`
	RequestCount int64   `json:"requestCount"`
	SuccessRate  float64 `json:"successRate"`
	AvgLatency   float64 `json:"avgLatency"` // milliseconds
}

// UserAgentStats 사용자 에이전트 통계
type UserAgentStats struct {
	UserAgent    string `json:"userAgent"`
	RequestCount int64  `json:"requestCount"`
	PipVersion   string `json:"pipVersion,omitempty"` // pip 버전 정보 (파싱 가능한 경우)
}
