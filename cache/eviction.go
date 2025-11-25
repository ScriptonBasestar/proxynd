package cache

import (
	"container/heap"
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// EvictionPolicy 캐시 제거 정책
type EvictionPolicy interface {
	// ShouldEvict 제거가 필요한지 확인
	ShouldEvict(currentSize, maxSize int64, item *CacheMetadata) bool

	// SelectEvictionCandidates 제거할 항목 선택
	SelectEvictionCandidates(items []*CacheMetadata, requiredSpace int64) []string
}

// LRUEvictionPolicy LRU (Least Recently Used) 정책
type LRUEvictionPolicy struct{}

// ShouldEvict determines if a cache item should be evicted based on TTL and size constraints.
func (p *LRUEvictionPolicy) ShouldEvict(currentSize, maxSize int64, item *CacheMetadata) bool {
	// TTL 만료 확인
	if item.TTL > 0 && time.Since(item.CreatedAt) > item.TTL {
		return true
	}

	// 크기 초과 확인
	return currentSize > maxSize
}

// SelectEvictionCandidates selects cache items for eviction to free the required space.
func (p *LRUEvictionPolicy) SelectEvictionCandidates(items []*CacheMetadata, requiredSpace int64) []string {
	// 접근 시간 기준으로 정렬
	h := &metadataHeap{items: items}
	heap.Init(h)

	var candidates []string
	var freedSpace int64

	for h.Len() > 0 && freedSpace < requiredSpace {
		itemInterface := heap.Pop(h)
		if item, ok := itemInterface.(*CacheMetadata); ok {
			candidates = append(candidates, item.Key)
			freedSpace += item.Size
		}
	}

	return candidates
}

// Evictor is an alias for CacheEvictor to avoid stuttering
type Evictor = CacheEvictor

// CacheEvictor manages cache eviction policies and cleanup operations.
// Note: Named CacheEvictor for clarity despite package name repetition.
type CacheEvictor struct {
	backend CacheBackend
	policy  EvictionPolicy
	options CacheOptions
	mu      sync.Mutex
	stopCh  chan struct{}
	wg      sync.WaitGroup
}

// NewCacheEvictor 새 캐시 제거 관리자 생성
func NewCacheEvictor(backend CacheBackend, policy EvictionPolicy, options CacheOptions) *CacheEvictor {
	return &CacheEvictor{
		backend: backend,
		policy:  policy,
		options: options,
		stopCh:  make(chan struct{}),
	}
}

// Start 캐시 제거 프로세스 시작
func (e *CacheEvictor) Start(ctx context.Context, interval time.Duration) {
	e.wg.Add(1)
	go e.evictionLoop(ctx, interval)
}

// Stop 캐시 제거 프로세스 중지
func (e *CacheEvictor) Stop() {
	close(e.stopCh)
	e.wg.Wait()
}

// evictionLoop 주기적 캐시 정리
func (e *CacheEvictor) evictionLoop(ctx context.Context, interval time.Duration) {
	defer e.wg.Done()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-e.stopCh:
			return
		case <-ticker.C:
			if err := e.performEviction(); err != nil {
				log.Printf("Cache eviction error: %v", err)
			}
		}
	}
}

// performEviction 캐시 정리 수행
func (e *CacheEvictor) performEviction() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// 현재 캐시 크기 확인
	currentSize, err := e.backend.Size()
	if err != nil {
		return fmt.Errorf("failed to get cache size: %w", err)
	}

	// 크기 제한 확인
	if e.options.MaxSize > 0 && currentSize <= e.options.MaxSize {
		// TTL 만료 항목만 확인
		return e.evictExpiredItems()
	}

	// 크기 초과 시 LRU 정책 적용
	requiredSpace := currentSize - int64(float64(e.options.MaxSize)*0.9) // 90%로 줄이기
	return e.evictByPolicy(requiredSpace)
}

// evictExpiredItems TTL 만료 항목 제거
func (e *CacheEvictor) evictExpiredItems() error {
	// FileSystemBackend인 경우 직접 스캔
	if fsBackend, ok := e.backend.(*FileSystemBackend); ok {
		return e.evictExpiredFromFileSystem(fsBackend)
	}

	// S3Backend인 경우 메타데이터 스캔
	if s3Backend, ok := e.backend.(*S3Backend); ok {
		return e.evictExpiredFromS3(s3Backend)
	}

	return nil
}

// evictExpiredFromFileSystem 파일 시스템에서 만료 항목 제거
func (e *CacheEvictor) evictExpiredFromFileSystem(backend *FileSystemBackend) error {
	// 디렉토리 스캔으로 캐시 키 목록 가져오기
	keys, err := e.scanCacheKeys(backend.basePath)
	if err != nil {
		log.Printf("Failed to scan cache keys: %v", err)
		return err
	}

	evictedCount := 0
	for _, key := range keys {
		meta, err := backend.GetMetadata(key)
		if err != nil {
			continue
		}

		if e.policy.ShouldEvict(0, 0, meta) {
			if err := backend.Delete(key); err != nil {
				log.Printf("Failed to delete expired key %s: %v", key, err)
			} else {
				evictedCount++
			}
		}
	}

	if evictedCount > 0 {
		log.Printf("Evicted %d expired cache entries", evictedCount)
	}

	return nil
}

// scanCacheKeys 디렉토리 스캔으로 캐시 키 목록 수집
func (e *CacheEvictor) scanCacheKeys(basePath string) ([]string, error) {
	var keys []string

	err := filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// 접근 오류는 무시하고 계속 진행
			return nil
		}

		// 디렉토리는 건너뛰기
		if info.IsDir() {
			return nil
		}

		// 메타데이터 파일(.meta)은 건너뛰기 - 실제 캐시 파일만 처리
		if strings.HasSuffix(path, ".meta") {
			return nil
		}

		// 임시 파일(.tmp, .partial)은 건너뛰기
		if strings.HasSuffix(path, ".tmp") || strings.HasSuffix(path, ".partial") {
			return nil
		}

		// basePath를 기준으로 상대 경로를 키로 사용
		relPath, err := filepath.Rel(basePath, path)
		if err != nil {
			return nil
		}

		// 경로 구분자를 슬래시로 정규화 (플랫폼 독립성)
		key := filepath.ToSlash(relPath)
		keys = append(keys, key)

		return nil
	})

	return keys, err
}

// evictExpiredFromS3 S3에서 만료 항목 제거
func (e *CacheEvictor) evictExpiredFromS3(_ *S3Backend) error {
	// S3 객체 목록 스캔 필요
	// 구현 간소화를 위해 생략
	return nil
}

// evictByPolicy 정책에 따른 캐시 제거 (외부에서 호출 가능)
func (e *CacheEvictor) evictByPolicy(requiredSpace int64) error {
	// 모든 캐시 메타데이터 수집
	allMetadata, err := e.collectAllMetadata()
	if err != nil {
		log.Printf("Failed to collect cache metadata: %v", err)
		return err
	}

	// 제거 후보 선택
	candidates := e.policy.SelectEvictionCandidates(allMetadata, requiredSpace)

	// 캐시 항목 제거
	evictedCount := 0
	for _, key := range candidates {
		if err := e.backend.Delete(key); err != nil {
			log.Printf("Failed to evict key %s: %v", key, err)
		} else {
			evictedCount++
		}
	}

	if evictedCount > 0 {
		log.Printf("Evicted %d cache entries by policy", evictedCount)
	}

	return nil
}

// collectAllMetadata 모든 캐시 메타데이터 수집
func (e *CacheEvictor) collectAllMetadata() ([]*CacheMetadata, error) {
	var allMetadata []*CacheMetadata

	// FileSystemBackend인 경우
	if fsBackend, ok := e.backend.(*FileSystemBackend); ok {
		keys, err := e.scanCacheKeys(fsBackend.basePath)
		if err != nil {
			return nil, err
		}

		for _, key := range keys {
			meta, err := fsBackend.GetMetadata(key)
			if err != nil {
				continue // 메타데이터 없는 항목은 건너뛰기
			}
			allMetadata = append(allMetadata, meta)
		}
	}

	return allMetadata, nil
}

// EvictKey 특정 키 강제 제거
func (e *CacheEvictor) EvictKey(key string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	return e.backend.Delete(key)
}

// metadataHeap 메타데이터 힙 (LRU 정렬용)
type metadataHeap struct {
	items []*CacheMetadata
}

// Len returns the number of items in the heap
func (h metadataHeap) Len() int { return len(h.items) }

// Less reports whether the element with index i should sort before the element with index j
func (h metadataHeap) Less(i, j int) bool {
	// 접근 시간이 오래된 것이 우선
	return h.items[i].AccessedAt.Before(h.items[j].AccessedAt)
}

// Swap swaps the elements with indexes i and j
func (h metadataHeap) Swap(i, j int) {
	h.items[i], h.items[j] = h.items[j], h.items[i]
}

// Push pushes the element x onto the heap
func (h *metadataHeap) Push(x interface{}) {
	if item, ok := x.(*CacheMetadata); ok {
		h.items = append(h.items, item)
	}
}

// Pop removes and returns the minimum element from the heap
func (h *metadataHeap) Pop() interface{} {
	old := h.items
	n := len(old)
	item := old[n-1]
	h.items = old[0 : n-1]
	return item
}
