package apk

import (
	"context"
)

// PackageService APK 패키지 관리 서비스 인터페이스
type PackageService interface {
	// HandleRequest 패키지 요청 처리
	HandleRequest(ctx context.Context, request *PackageRequest) (*PackageResponse, error)
	
	// GetPackageInfo APK 패키지 정보 조회
	GetPackageInfo(ctx context.Context, packagePath string) (*PackageInfo, error)
	
	// ValidatePackagePath 패키지 경로 유효성 검증
	ValidatePackagePath(packagePath string) error
}

// RepositoryManager APK 리포지토리 관리 서비스 인터페이스
type RepositoryManager interface {
	// GetRepositoryInfo 리포지토리 정보 조회
	GetRepositoryInfo(ctx context.Context, architecture, branch, component string) (*RepositoryInfo, error)
	
	// UpdateRepositoryIndex 리포지토리 인덱스 업데이트
	UpdateRepositoryIndex(ctx context.Context, info *RepositoryInfo) error
	
	// ValidateRepository 리포지토리 유효성 검증
	ValidateRepository(architecture, branch, component string) error
	
	// GetAvailableArchitectures 사용 가능한 아키텍처 목록 조회
	GetAvailableArchitectures(ctx context.Context) ([]string, error)
	
	// GetAvailableBranches 사용 가능한 브랜치 목록 조회
	GetAvailableBranches(ctx context.Context) ([]string, error)
	
	// GetAvailableComponents 사용 가능한 컴포넌트 목록 조회
	GetAvailableComponents(ctx context.Context) ([]string, error)
}

// CacheManager APK 캐시 관리 서비스 인터페이스
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
	
	// GetCacheKeysByType 타입별 캐시 키 조회
	GetCacheKeysByType(ctx context.Context, fileType string) ([]string, error)
}

// SignatureVerifier APK 서명 검증 서비스 인터페이스
type SignatureVerifier interface {
	// VerifyPackage APK 패키지 서명 검증
	VerifyPackage(ctx context.Context, packagePath string) (*SignatureInfo, error)
	
	// LoadTrustedKeys 신뢰할 수 있는 키 로드
	LoadTrustedKeys(ctx context.Context, keyDirectory string) error
	
	// ValidateSignature 서명 유효성 검증
	ValidateSignature(ctx context.Context, signaturePath string) (*SignatureInfo, error)
	
	// GetTrustedKeys 신뢰할 수 있는 키 목록 조회
	GetTrustedKeys(ctx context.Context) ([]string, error)
	
	// IsPackageFile APK 패키지 파일인지 확인
	IsPackageFile(filename string) bool
	
	// IsSignatureFile 서명 파일인지 확인
	IsSignatureFile(filename string) bool
}

// MetricsCollector APK 메트릭 수집 서비스 인터페이스
type MetricsCollector interface {
	// RecordRequest 요청 메트릭 기록
	RecordRequest(ctx context.Context, metrics *RequestMetrics) error
	
	// GetRequestStats 요청 통계 조회
	GetRequestStats(ctx context.Context, period string) (map[string]interface{}, error)
	
	// GetPopularPackages 인기 패키지 조회
	GetPopularPackages(ctx context.Context, limit int) ([]*PackageInfo, error)
	
	// GetArchitectureStats 아키텍처별 통계 조회
	GetArchitectureStats(ctx context.Context) (map[string]interface{}, error)
	
	// GetBranchStats 브랜치별 통계 조회
	GetBranchStats(ctx context.Context) (map[string]interface{}, error)
	
	// GetComponentStats 컴포넌트별 통계 조회
	GetComponentStats(ctx context.Context) (map[string]interface{}, error)
	
	// RecordCacheHit 캐시 히트 기록
	RecordCacheHit(ctx context.Context, key string, hit bool) error
	
	// GetCacheMetrics 캐시 메트릭 조회
	GetCacheMetrics(ctx context.Context) (map[string]interface{}, error)
	
	// RecordSignatureVerification 서명 검증 결과 기록
	RecordSignatureVerification(ctx context.Context, packagePath string, result *SignatureInfo) error
}