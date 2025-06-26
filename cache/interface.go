package cache

import (
	"io"
	"time"
)

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

// CacheOptions 캐시 옵션
type CacheOptions struct {
	DefaultTTL time.Duration
	MaxSize    int64  // 최대 캐시 크기 (바이트)
	BasePath   string // 기본 경로
}

// CacheStats 캐시 통계
type CacheStats struct {
	Hits      int64
	Misses    int64
	Size      int64
	ItemCount int64
	LastClear time.Time
}
