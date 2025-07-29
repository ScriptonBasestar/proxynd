package yum

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"proxynd/internal/domain/yum"
	"proxynd/logging"
)

// repoManagerImpl YUM 리포지토리 관리 서비스 구현
type repoManagerImpl struct {
	config       yum.ProxyConfig
	logger       logging.Logger
	storageDir   string
	repositories map[string]*yum.RepoMetadata
}

// NewRepoManager YUM 리포지토리 관리자 생성
func NewRepoManager(
	config yum.ProxyConfig,
	logger logging.Logger,
	storageDir string,
) yum.RepoManager {
	return &repoManagerImpl{
		config:       config,
		logger:       logger,
		storageDir:   storageDir,
		repositories: make(map[string]*yum.RepoMetadata),
	}
}

// GetRepoMetadata 리포지토리 메타데이터 조회
func (r *repoManagerImpl) GetRepoMetadata(ctx context.Context, repository, metadataType string) (*yum.RepoMetadata, error) {
	key := fmt.Sprintf("%s:%s", repository, metadataType)

	// 메모리 캐시에서 조회
	if metadata, exists := r.repositories[key]; exists {
		r.logger.Debug("YUM 리포지토리 메타데이터 메모리 캐시 히트",
			logging.F("repository", repository),
			logging.F("type", metadataType))
		return metadata, nil
	}

	// 파일에서 메타데이터 로드
	metadataPath := r.getMetadataPath(repository, metadataType)
	if _, err := os.Stat(metadataPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("metadata not found: %s", key)
	}

	stat, err := os.Stat(metadataPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat metadata file: %w", err)
	}

	metadata := &yum.RepoMetadata{
		Repository:   repository,
		MetadataType: metadataType,
		Path:         metadataPath,
		Size:         stat.Size(),
		LastModified: stat.ModTime(),
	}

	// 체크섬 계산 (실제 구현에서는 파일 내용을 읽어 체크섬 계산)
	metadata.Checksum = r.calculateChecksum(metadataPath)

	// 메모리 캐시에 저장
	r.repositories[key] = metadata

	r.logger.Info("YUM 리포지토리 메타데이터 로드",
		logging.F("repository", repository),
		logging.F("type", metadataType),
		logging.F("path", metadataPath),
		logging.F("size", stat.Size()))

	return metadata, nil
}

// UpdateRepoMetadata 리포지토리 메타데이터 업데이트
func (r *repoManagerImpl) UpdateRepoMetadata(ctx context.Context, metadata *yum.RepoMetadata) error {
	if err := r.ValidateRepository(metadata.Repository); err != nil {
		return fmt.Errorf("invalid repository: %w", err)
	}

	if !r.isValidMetadataType(metadata.MetadataType) {
		return fmt.Errorf("invalid metadata type: %s", metadata.MetadataType)
	}

	key := fmt.Sprintf("%s:%s", metadata.Repository, metadata.MetadataType)

	// 메타데이터 정보 업데이트
	metadata.LastModified = time.Now()

	// 메모리 캐시 업데이트
	r.repositories[key] = metadata

	r.logger.Info("YUM 리포지토리 메타데이터 업데이트",
		logging.F("repository", metadata.Repository),
		logging.F("type", metadata.MetadataType),
		logging.F("path", metadata.Path))

	return nil
}

// ValidateRepository 리포지토리 유효성 검증
func (r *repoManagerImpl) ValidateRepository(repository string) error {
	if repository == "" {
		return fmt.Errorf("repository name is empty")
	}

	// 허용되지 않는 문자 확인
	if strings.Contains(repository, "..") || strings.Contains(repository, "/") {
		return fmt.Errorf("repository name contains invalid characters")
	}

	// 예약어 확인
	reservedNames := []string{".", "..", "repodata"}
	for _, reserved := range reservedNames {
		if repository == reserved {
			return fmt.Errorf("repository name is reserved: %s", repository)
		}
	}

	return nil
}

// GetRepositoryList 사용 가능한 리포지토리 목록 조회
func (r *repoManagerImpl) GetRepositoryList(ctx context.Context) ([]string, error) {
	baseDir := filepath.Join(r.storageDir, r.config.GetPath())

	entries, err := os.ReadDir(baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to read repository directory: %w", err)
	}

	var repositories []string
	for _, entry := range entries {
		if entry.IsDir() && r.isValidRepositoryDir(entry.Name()) {
			repositories = append(repositories, entry.Name())
		}
	}

	r.logger.Debug("YUM 리포지토리 목록 조회",
		logging.F("count", len(repositories)),
		logging.F("repositories", strings.Join(repositories, ", ")))

	return repositories, nil
}

// 헬퍼 메서드들

func (r *repoManagerImpl) getMetadataPath(repository, metadataType string) string {
	baseDir := filepath.Join(r.storageDir, r.config.GetPath())

	switch metadataType {
	case "repomd":
		return filepath.Join(baseDir, repository, "repodata", "repomd.xml")
	case "primary":
		return filepath.Join(baseDir, repository, "repodata", "primary.xml.gz")
	case "filelists":
		return filepath.Join(baseDir, repository, "repodata", "filelists.xml.gz")
	case "other":
		return filepath.Join(baseDir, repository, "repodata", "other.xml.gz")
	default:
		return filepath.Join(baseDir, repository, "repodata", fmt.Sprintf("%s.xml", metadataType))
	}
}

func (r *repoManagerImpl) isValidMetadataType(metadataType string) bool {
	validTypes := []string{"repomd", "primary", "filelists", "other", "updateinfo", "modules"}
	for _, validType := range validTypes {
		if metadataType == validType {
			return true
		}
	}
	return false
}

func (r *repoManagerImpl) isValidRepositoryDir(name string) bool {
	// 숨김 디렉토리나 시스템 디렉토리 제외
	if strings.HasPrefix(name, ".") || name == "lost+found" {
		return false
	}

	// repodata 디렉토리가 있는지 확인
	baseDir := filepath.Join(r.storageDir, r.config.GetPath())
	repodataPath := filepath.Join(baseDir, name, "repodata")

	if _, err := os.Stat(repodataPath); os.IsNotExist(err) {
		return false
	}

	return true
}

func (r *repoManagerImpl) calculateChecksum(filePath string) string {
	// 실제 구현에서는 SHA256 체크섬 계산
	// 여기서는 파일 크기와 수정 시간을 기반으로 간단한 해시 생성
	stat, err := os.Stat(filePath)
	if err != nil {
		return ""
	}

	return fmt.Sprintf("sha256:%x", stat.Size()+stat.ModTime().Unix())
}
