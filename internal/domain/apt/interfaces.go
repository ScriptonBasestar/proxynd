package apt

import (
	"context"
)

// PackageHandler APT 패키지 요청 처리 인터페이스 (도메인 로직)
type PackageHandler interface {
	// Handle 패키지 요청 처리
	Handle(ctx context.Context, request *PackageRequest) (*PackageResponse, error)

	// GetPackageMetadata 패키지 메타데이터 조회
	GetPackageMetadata(ctx context.Context, osType, packagePath string) (*PackageMetadata, error)
}

// MirrorManager 미러 관리 인터페이스
type MirrorManager interface {
	// GetNextMirror 라운드로빈으로 다음 미러 선택
	GetNextMirror(osType string) (*MirrorStatus, error)

	// CheckMirrorHealth 미러 상태 확인
	CheckMirrorHealth(ctx context.Context, mirror *MirrorStatus) error

	// GetMirrorStats 미러 통계 조회
	GetMirrorStats(osType string) ([]*MirrorStatus, error)

	// MarkMirrorFailed 미러를 실패로 표시
	MarkMirrorFailed(osType, mirrorURL string, err error)
}

// CacheManager 캐시 관리 인터페이스
type CacheManager interface {
	// Get 캐시에서 패키지 조회
	Get(ctx context.Context, key string) (*CacheEntry, error)

	// Set 패키지를 캐시에 저장
	Set(ctx context.Context, key string, data []byte, contentType string) error

	// Delete 캐시에서 패키지 삭제
	Delete(ctx context.Context, key string) error

	// GenerateKey 캐시 키 생성
	GenerateKey(osType, packagePath string) string

	// Cleanup 오래된 캐시 파일 정리
	Cleanup(ctx context.Context) error

	// GetStats 캐시 통계 조회
	GetStats(ctx context.Context) (*CacheStats, error)
}

// ContentTypeResolver 콘텐츠 타입 결정 인터페이스
type ContentTypeResolver interface {
	// GetContentType 패키지 경로에서 Content-Type 결정
	GetContentType(packagePath string) string

	// ShouldInline 인라인으로 표시할지 결정
	ShouldInline(packagePath string) bool

	// GetDisposition Content-Disposition 헤더 생성
	GetDisposition(packagePath string) string
}

// MetricsCollector 메트릭 수집 인터페이스
type MetricsCollector interface {
	// RecordRequest 요청 메트릭 기록
	RecordRequest(ctx context.Context, metrics *RequestMetrics) error

	// GetRequestStats 요청 통계 조회
	GetRequestStats(ctx context.Context, osType string) (*RequestStats, error)
}

// CacheStats 캐시 통계
type CacheStats struct {
	TotalEntries int64   `json:"totalEntries"`
	TotalSizeMB  int64   `json:"totalSizeMB"`
	HitRate      float64 `json:"hitRate"`
	OldestEntry  string  `json:"oldestEntry"`
	NewestEntry  string  `json:"newestEntry"`
	CleanupCount int64   `json:"cleanupCount"`
	LastCleanup  string  `json:"lastCleanup"`
}

// RequestStats 요청 통계
type RequestStats struct {
	OSType           string             `json:"osType"`
	TotalRequests    int64              `json:"totalRequests"`
	CacheHitRate     float64            `json:"cacheHitRate"`
	AvgResponseTime  float64            `json:"avgResponseTime"` // milliseconds
	TotalBytesServed int64              `json:"totalBytesServed"`
	TopPackages      []PackageStats     `json:"topPackages"`
	MirrorUsage      []MirrorUsageStats `json:"mirrorUsage"`
}

// PackageStats 패키지별 통계
type PackageStats struct {
	PackagePath   string `json:"packagePath"`
	RequestCount  int64  `json:"requestCount"`
	TotalBytes    int64  `json:"totalBytes"`
	LastRequested string `json:"lastRequested"`
}

// MirrorUsageStats 미러 사용 통계
type MirrorUsageStats struct {
	MirrorURL    string  `json:"mirrorURL"`
	RequestCount int64   `json:"requestCount"`
	SuccessRate  float64 `json:"successRate"`
	AvgLatency   float64 `json:"avgLatency"` // milliseconds
}
