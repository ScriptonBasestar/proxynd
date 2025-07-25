package cache

import (
	"context"
	"fmt"
	"io"
	"log"
	"runtime"
	"sync"
	"time"

	"proxynd/logging"
	"proxynd/metrics"
)

// Manager 캐시 매니저
type Manager struct {
	backend CacheBackend
	options CacheOptions
	stats   CacheStats
	evictor *CacheEvictor
	mu      sync.RWMutex
}

// NewManager 새 캐시 매니저 생성
func NewManager(backend CacheBackend, options CacheOptions) *Manager {
	evictor := NewCacheEvictor(backend, &LRUEvictionPolicy{}, options)
	return &Manager{
		backend: backend,
		options: options,
		stats:   CacheStats{},
		evictor: evictor,
	}
}

// Get 캐시에서 데이터 가져오기
func (m *Manager) Get(key string) ([]byte, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	reader, err := m.backend.Get(key)
	if err != nil {
		m.stats.Misses++
		return nil, false
	}
	defer func() { _ = reader.Close() }()

	// 읽기 성공
	m.stats.Hits++

	// io.ReadAll을 사용하여 데이터 읽기
	data := make([]byte, 0)
	buf := make([]byte, 1024)
	for {
		n, err := reader.Read(buf)
		if n > 0 {
			data = append(data, buf[:n]...)
		}
		if err != nil {
			break
		}
	}

	return data, true
}

// Put 캐시에 데이터 저장
func (m *Manager) Put(key string, data []byte, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 기본 TTL 사용
	if ttl == 0 {
		ttl = m.options.DefaultTTL
	}

	// 캐시 크기 체크 및 정리
	currentSize, err := m.backend.Size()
	if err != nil {
		// 크기 조회 실패 시 경고 로그 후 계속 진행
		log.Printf("Warning: Failed to get cache size: %v", err)
		currentSize = 0
	}
	if m.options.MaxSize > 0 && currentSize+int64(len(data)) > m.options.MaxSize {
		// 필요한 공간 계산
		requiredSpace := currentSize + int64(len(data)) - m.options.MaxSize
		if err := m.evictor.evictByPolicy(requiredSpace); err != nil {
			// 정리 실패 시 로그 출력하고 계속 진행
			// 로그는 단순화
			_ = err // 에러를 명시적으로 무시
		}
	}

	// 데이터 저장
	reader := &bytesReader{data: data}
	return m.backend.Put(key, reader, ttl)
}

// Exists 캐시 존재 여부 확인
func (m *Manager) Exists(key string) bool {
	return m.backend.Exists(key)
}

// Delete 캐시 항목 삭제
func (m *Manager) Delete(key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.backend.Delete(key)
}

// Clear 전체 캐시 삭제
func (m *Manager) Clear() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.stats.LastClear = time.Now()
	return m.backend.Clear()
}

// GetStats 캐시 통계 조회
func (m *Manager) GetStats() CacheStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := m.stats
	size, err := m.backend.Size()
	if err != nil {
		log.Printf("Warning: Failed to get cache size for stats: %v", err)
		size = 0
	}
	stats.Size = size
	return stats
}

// GetCachePath 캐시 경로 생성
func (m *Manager) GetCachePath(proxyType, requestPath string) string {
	return fmt.Sprintf("%s/%s/%s", m.options.BasePath, proxyType, requestPath)
}

// StartEviction 캐시 정리 프로세스 시작
func (m *Manager) StartEviction(ctx context.Context, interval time.Duration) {
	m.evictor.Start(ctx, interval)
}

// StopEviction 캐시 정리 프로세스 중지
func (m *Manager) StopEviction() {
	m.evictor.Stop()
}

// ForceEviction 강제 캐시 정리 수행
func (m *Manager) ForceEviction() error {
	return m.evictor.performEviction()
}

// bytesReader []byte를 io.Reader로 변환
type bytesReader struct {
	data []byte
	pos  int
}

// Read reads data into p from the internal byte slice
func (r *bytesReader) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n = copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}
