package cache

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// FileSystemBackend 파일 시스템 기반 캐시 백엔드
type FileSystemBackend struct {
	basePath string
	mu       sync.RWMutex
}

// NewFileSystemBackend 새 파일 시스템 백엔드 생성
func NewFileSystemBackend(basePath string) (*FileSystemBackend, error) {
	// 기본 경로 생성
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}
	
	return &FileSystemBackend{
		basePath: basePath,
	}, nil
}

// Get 캐시에서 데이터 읽기
func (fs *FileSystemBackend) Get(key string) (io.ReadCloser, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	
	filePath := fs.getFilePath(key)
	
	// 메타데이터 확인
	meta, err := fs.loadMetadata(key)
	if err != nil {
		return nil, err
	}
	
	// TTL 확인
	if meta.TTL > 0 && time.Since(meta.CreatedAt) > meta.TTL {
		// 만료된 캐시 삭제
		fs.mu.RUnlock()
		fs.mu.Lock()
		os.Remove(filePath)
		os.Remove(fs.getMetadataPath(key))
		fs.mu.Unlock()
		fs.mu.RLock()
		return nil, fmt.Errorf("cache expired")
	}
	
	// 파일 열기
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	
	// 접근 시간 업데이트
	go fs.updateAccessTime(key)
	
	return file, nil
}

// Put 캐시에 데이터 저장
func (fs *FileSystemBackend) Put(key string, data io.Reader, ttl time.Duration) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	
	filePath := fs.getFilePath(key)
	
	// 디렉토리 생성
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	
	// 임시 파일에 쓰기
	tempFile := filePath + ".tmp"
	file, err := os.Create(tempFile)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	
	// 데이터 복사
	size, err := io.Copy(file, data)
	file.Close()
	if err != nil {
		os.Remove(tempFile)
		return fmt.Errorf("failed to write data: %w", err)
	}
	
	// 원본 파일로 이동
	if err := os.Rename(tempFile, filePath); err != nil {
		os.Remove(tempFile)
		return fmt.Errorf("failed to rename temp file: %w", err)
	}
	
	// 메타데이터 저장
	metadata := &CacheMetadata{
		Key:        key,
		Size:       size,
		CreatedAt:  time.Now(),
		AccessedAt: time.Now(),
		TTL:        ttl,
	}
	
	return fs.saveMetadata(key, metadata)
}

// Exists 캐시 키 존재 여부 확인
func (fs *FileSystemBackend) Exists(key string) bool {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	
	filePath := fs.getFilePath(key)
	
	// 파일 존재 확인
	if _, err := os.Stat(filePath); err != nil {
		return false
	}
	
	// 메타데이터 확인 및 TTL 체크
	meta, err := fs.loadMetadata(key)
	if err != nil {
		return false
	}
	
	// TTL 확인
	if meta.TTL > 0 && time.Since(meta.CreatedAt) > meta.TTL {
		// 만료된 캐시
		return false
	}
	
	return true
}

// Delete 캐시 항목 삭제
func (fs *FileSystemBackend) Delete(key string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	
	filePath := fs.getFilePath(key)
	metaPath := fs.getMetadataPath(key)
	
	// 파일 삭제
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete cache file: %w", err)
	}
	
	// 메타데이터 삭제
	if err := os.Remove(metaPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete metadata: %w", err)
	}
	
	return nil
}

// GetMetadata 캐시 메타데이터 조회
func (fs *FileSystemBackend) GetMetadata(key string) (*CacheMetadata, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	
	return fs.loadMetadata(key)
}

// Clear 전체 캐시 삭제
func (fs *FileSystemBackend) Clear() error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	
	// 캐시 디렉토리 삭제 후 재생성
	if err := os.RemoveAll(fs.basePath); err != nil {
		return fmt.Errorf("failed to clear cache: %w", err)
	}
	
	return os.MkdirAll(fs.basePath, 0755)
}

// Size 캐시 크기 조회
func (fs *FileSystemBackend) Size() (int64, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	
	var totalSize int64
	
	err := filepath.Walk(fs.basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		// 메타데이터 파일 제외
		if !strings.HasSuffix(path, ".meta") && !info.IsDir() {
			totalSize += info.Size()
		}
		
		return nil
	})
	
	return totalSize, err
}

// getFilePath 캐시 파일 경로 생성
func (fs *FileSystemBackend) getFilePath(key string) string {
	// 키를 안전한 파일 경로로 변환
	safePath := filepath.Join(fs.basePath, key)
	return safePath
}

// getMetadataPath 메타데이터 파일 경로 생성
func (fs *FileSystemBackend) getMetadataPath(key string) string {
	return fs.getFilePath(key) + ".meta"
}

// saveMetadata 메타데이터 저장
func (fs *FileSystemBackend) saveMetadata(key string, meta *CacheMetadata) error {
	metaPath := fs.getMetadataPath(key)
	
	// 디렉토리 생성
	dir := filepath.Dir(metaPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	
	// JSON으로 직렬화
	data, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	
	return os.WriteFile(metaPath, data, 0644)
}

// loadMetadata 메타데이터 로드
func (fs *FileSystemBackend) loadMetadata(key string) (*CacheMetadata, error) {
	metaPath := fs.getMetadataPath(key)
	
	data, err := os.ReadFile(metaPath)
	if err != nil {
		return nil, err
	}
	
	var meta CacheMetadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, err
	}
	
	return &meta, nil
}

// updateAccessTime 접근 시간 업데이트
func (fs *FileSystemBackend) updateAccessTime(key string) {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	
	meta, err := fs.loadMetadata(key)
	if err != nil {
		return
	}
	
	meta.AccessedAt = time.Now()
	fs.saveMetadata(key, meta)
}