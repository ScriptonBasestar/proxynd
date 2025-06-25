package cache

import (
	"fmt"
	"sync"
	"time"
)

// Manager 캐시 매니저
type Manager struct {
	backend CacheBackend
	options CacheOptions
	stats   CacheStats
	mu      sync.RWMutex
}

// NewManager 새 캐시 매니저 생성
func NewManager(backend CacheBackend, options CacheOptions) *Manager {
	return &Manager{
		backend: backend,
		options: options,
		stats:   CacheStats{},
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
	defer reader.Close()
	
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
	
	// 캐시 크기 체크
	currentSize, _ := m.backend.Size()
	if m.options.MaxSize > 0 && currentSize+int64(len(data)) > m.options.MaxSize {
		// LRU 정책으로 오래된 항목 삭제 (간단한 구현)
		m.evictOldItems(int64(len(data)))
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
	stats.Size, _ = m.backend.Size()
	return stats
}

// GetCachePath 캐시 경로 생성
func (m *Manager) GetCachePath(proxyType, requestPath string) string {
	return fmt.Sprintf("%s/%s/%s", m.options.BasePath, proxyType, requestPath)
}

// evictOldItems 오래된 항목 삭제 (간단한 LRU 구현)
func (m *Manager) evictOldItems(requiredSpace int64) {
	// TODO: 실제 LRU 구현
	// 현재는 전체 캐시의 10%를 삭제하는 간단한 방식
	currentSize, _ := m.backend.Size()
	if currentSize > 0 {
		// 구현 단순화를 위해 일부만 삭제
		// 실제로는 메타데이터를 사용하여 LRU 구현 필요
	}
}

// bytesReader []byte를 io.Reader로 변환
type bytesReader struct {
	data []byte
	pos  int
}

func (r *bytesReader) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n = copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

// io.EOF 정의
var io struct {
	EOF error
}

func init() {
	io.EOF = fmt.Errorf("EOF")
}