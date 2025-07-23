package proxy

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"proxynd/logging"
)

// IndexStorage 인덱스 저장소 인터페이스
type IndexStorage interface {
	Save(entries []SearchIndexEntry) error
	Load() ([]SearchIndexEntry, error)
	GetLastModified() (time.Time, error)
}

// FileIndexStorage 파일 기반 인덱스 저장소
type FileIndexStorage struct {
	filePath string
	logger   logging.Logger
}

// NewFileIndexStorage 새로운 파일 기반 인덱스 저장소 생성
func NewFileIndexStorage(storageDir string) *FileIndexStorage {
	return &FileIndexStorage{
		filePath: filepath.Join(storageDir, "maven-search-index.json"),
		logger:   logging.GetLogger(),
	}
}

// Save 인덱스를 파일에 저장
func (s *FileIndexStorage) Save(entries []SearchIndexEntry) error {
	// 디렉토리 생성
	dir := filepath.Dir(s.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// JSON으로 인코딩
	data, err := json.Marshal(entries)
	if err != nil {
		return err
	}

	// 파일에 저장
	if err := os.WriteFile(s.filePath, data, 0644); err != nil {
		return err
	}

	s.logger.Info("Index saved to file",
		logging.F("path", s.filePath),
		logging.F("entries", len(entries)),
		logging.F("size", len(data)))

	return nil
}

// Load 파일에서 인덱스 로드
func (s *FileIndexStorage) Load() ([]SearchIndexEntry, error) {
	// 파일 읽기
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // 파일이 없으면 빈 슬라이스 반환
		}
		return nil, err
	}

	// JSON 디코딩
	var entries []SearchIndexEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, err
	}

	s.logger.Info("Index loaded from file",
		logging.F("path", s.filePath),
		logging.F("entries", len(entries)))

	return entries, nil
}

// GetLastModified 인덱스 파일의 마지막 수정 시간 반환
func (s *FileIndexStorage) GetLastModified() (time.Time, error) {
	info, err := os.Stat(s.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return time.Time{}, nil
		}
		return time.Time{}, err
	}
	return info.ModTime(), nil
}
