package yum

import (
	"context"
)

// PackageService YUM 패키지 관리 서비스 인터페이스
type PackageService interface {
	// HandleRequest 패키지 요청 처리
	HandleRequest(ctx context.Context, request *PackageRequest) (*PackageResponse, error)

	// GetPackageInfo RPM 패키지 정보 조회
	GetPackageInfo(ctx context.Context, packagePath string) (*PackageInfo, error)

	// ValidatePackagePath 패키지 경로 유효성 검증
	ValidatePackagePath(packagePath string) error
}

// RepoManager YUM 리포지토리 관리 서비스 인터페이스
type RepoManager interface {
	// GetRepoMetadata 리포지토리 메타데이터 조회
	GetRepoMetadata(ctx context.Context, repository, metadataType string) (*RepoMetadata, error)

	// UpdateRepoMetadata 리포지토리 메타데이터 업데이트
	UpdateRepoMetadata(ctx context.Context, metadata *RepoMetadata) error

	// ValidateRepository 리포지토리 유효성 검증
	ValidateRepository(repository string) error

	// GetRepositoryList 사용 가능한 리포지토리 목록 조회
	GetRepositoryList(ctx context.Context) ([]string, error)
}

// CacheManager YUM 캐시 관리 서비스 인터페이스
type CacheManager interface {
	// Get 캐시에서 데이터 조회
	Get(ctx context.Context, key string) (*CacheEntry, error)

	// Set 캐시에 데이터 저장
	Set(ctx context.Context, key string, data []byte, ttl int64) error

	// Delete 캐시에서 데이터 삭제
	Delete(ctx context.Context, key string) error

	// Clear 전체 캐시 삭제
	Clear(ctx context.Context) error

	// GetStats 캐시 통계 조회
	GetStats(ctx context.Context) (map[string]interface{}, error)

	// Cleanup 만료된 캐시 정리
	Cleanup(ctx context.Context) error
}

// MetadataProcessor YUM 메타데이터 처리 서비스 인터페이스
type MetadataProcessor interface {
	// ProcessRepomd repomd.xml 처리
	ProcessRepomd(ctx context.Context, data []byte) (*RepoMetadata, error)

	// ProcessPrimaryXML primary.xml 처리
	ProcessPrimaryXML(ctx context.Context, data []byte) ([]*PackageInfo, error)

	// ProcessFilelistsXML filelists.xml 처리
	ProcessFilelistsXML(ctx context.Context, data []byte) (map[string][]string, error)

	// ProcessOtherXML other.xml 처리
	ProcessOtherXML(ctx context.Context, data []byte) (map[string]interface{}, error)

	// ValidateMetadata 메타데이터 유효성 검증
	ValidateMetadata(ctx context.Context, metadataType string, data []byte) error
}

// MetricsCollector YUM 메트릭 수집 서비스 인터페이스
type MetricsCollector interface {
	// RecordRequest 요청 메트릭 기록
	RecordRequest(ctx context.Context, metrics *RequestMetrics) error

	// GetRequestStats 요청 통계 조회
	GetRequestStats(ctx context.Context, period string) (map[string]interface{}, error)

	// GetPopularPackages 인기 패키지 조회
	GetPopularPackages(ctx context.Context, limit int) ([]*PackageInfo, error)

	// GetRepositoryStats 리포지토리별 통계 조회
	GetRepositoryStats(ctx context.Context) (map[string]interface{}, error)

	// RecordCacheHit 캐시 히트 기록
	RecordCacheHit(ctx context.Context, key string, hit bool) error

	// GetCacheMetrics 캐시 메트릭 조회
	GetCacheMetrics(ctx context.Context) (map[string]interface{}, error)
}
