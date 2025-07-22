package cache

import (
	"io"
	"time"
)

// Backend is an alias for CacheBackend to avoid stuttering
type Backend = CacheBackend

// Cache 캐시 인터페이스 (copylocks 문제 방지용)
type Cache interface {
	// Get 캐시에서 데이터 가져오기
	Get(key string) ([]byte, bool)

	// Put 캐시에 데이터 저장
	Put(key string, data []byte, ttl time.Duration) error

	// Exists 캐시 존재 여부 확인
	Exists(key string) bool

	// Delete 캐시 항목 삭제
	Delete(key string) error

	// Clear 전체 캐시 삭제
	Clear() error

	// GetStats 캐시 통계 조회
	GetStats() CacheStats

	// GetCachePath 캐시 경로 생성
	GetCachePath(proxyType, requestPath string) string

	// StartEviction 캐시 정리 프로세스 시작
	StartEviction(ctx interface{}, interval time.Duration)

	// StopEviction 캐시 정리 프로세스 중지
	StopEviction()

	// ForceEviction 강제 캐시 정리 수행
	ForceEviction() error
}

// CacheBackend 캐시 백엔드 인터페이스
type CacheBackend interface {
	// Get 캐시에서 데이터 읽기
	Get(key string) (io.ReadCloser, error)

	// Put 캐시에 데이터 저장
	Put(key string, data io.Reader, ttl time.Duration) error

	// Exists 캐시 키 존재 여부 확인
	Exists(key string) bool

	// Delete 캐시 항목 삭제
	Delete(key string) error

	// GetMetadata 캐시 메타데이터 조회
	GetMetadata(key string) (*CacheMetadata, error)

	// Clear 전체 캐시 삭제
	Clear() error

	// Size 캐시 크기 조회 (바이트)
	Size() (int64, error)
}

// Metadata is an alias for CacheMetadata to avoid stuttering
type Metadata = CacheMetadata

// CacheMetadata 캐시 메타데이터
type CacheMetadata struct {
	Key         string
	Size        int64
	CreatedAt   time.Time
	AccessedAt  time.Time
	TTL         time.Duration
	ContentType string
	ETag        string
}

// Options is an alias for CacheOptions to avoid stuttering
type Options = CacheOptions

// CacheOptions 캐시 옵션
type CacheOptions struct {
	DefaultTTL time.Duration
	MaxSize    int64  // 최대 캐시 크기 (바이트)
	BasePath   string // 기본 경로
}

// Stats is an alias for CacheStats to avoid stuttering
type Stats = CacheStats

// CacheStats 캐시 통계
type CacheStats struct {
	Hits      int64
	Misses    int64
	Size      int64
	ItemCount int64
	LastClear time.Time
}
