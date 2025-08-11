package backup

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"proxynd/logging"
)

// BackupMetadata 백업 메타데이터
type BackupMetadata struct {
	LastBackupTime time.Time         `json:"last_backup_time"`
	FileChecksums  map[string]string `json:"file_checksums"`
	BackupID       string            `json:"backup_id"`
	TotalFiles     int               `json:"total_files"`
	TotalSize      int64             `json:"total_size"`
}

// IncrementalBackup Maven 증분백업 구조체
type IncrementalBackup struct {
	sourceDir      string
	backupDir      string
	metadataFile   string
	logger         logging.Logger
	excludePattern []string
	mu             sync.Mutex
}

// NewIncrementalBackup 새로운 증분백업 인스턴스 생성
func NewIncrementalBackup(sourceDir, backupDir string) *IncrementalBackup {
	return &IncrementalBackup{
		sourceDir:    sourceDir,
		backupDir:    backupDir,
		metadataFile: filepath.Join(backupDir, ".backup-metadata.json"),
		logger:       logging.GetLogger(),
		excludePattern: []string{
			"*.tmp",
			"*.lock",
			"*.part",
			"_remote.repositories",
			"_maven.repositories",
		},
	}
}

// loadMetadata 이전 백업 메타데이터 로드
func (b *IncrementalBackup) loadMetadata() (*BackupMetadata, error) {
	data, err := os.ReadFile(b.metadataFile)
	if err != nil {
		if os.IsNotExist(err) {
			// 첫 백업인 경우
			return &BackupMetadata{
				FileChecksums: make(map[string]string),
			}, nil
		}
		return nil, err
	}

	var metadata BackupMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, err
	}

	if metadata.FileChecksums == nil {
		metadata.FileChecksums = make(map[string]string)
	}

	return &metadata, nil
}

// saveMetadata 백업 메타데이터 저장
func (b *IncrementalBackup) saveMetadata(metadata *BackupMetadata) error {
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return err
	}

	// 백업 디렉토리 생성
	if err := os.MkdirAll(b.backupDir, 0o755); err != nil {
		return err
	}

	return os.WriteFile(b.metadataFile, data, 0o644)
}

// calculateChecksum 파일 체크섬 계산
func (b *IncrementalBackup) calculateChecksum(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer func() { _ = file.Close() }()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// shouldExclude 제외 패턴 확인
func (b *IncrementalBackup) shouldExclude(path string) bool {
	base := filepath.Base(path)
	for _, pattern := range b.excludePattern {
		if matched, _ := filepath.Match(pattern, base); matched {
			return true
		}
	}
	return false
}

// BackupOptions 백업 옵션
type BackupOptions struct {
	// 체크섬 기반 변경 감지 (느리지만 정확)
	UseChecksum bool
	// 병렬 처리 워커 수
	Workers int
	// 진행 상황 콜백
	ProgressCallback func(current, total int, file string)
}

// PerformBackup 증분백업 수행
func (b *IncrementalBackup) PerformBackup(options *BackupOptions) (*BackupMetadata, error) {
	if options == nil {
		options = &BackupOptions{
			UseChecksum: true,
			Workers:     4,
		}
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	// 이전 메타데이터 로드
	oldMetadata, err := b.loadMetadata()
	if err != nil {
		return nil, fmt.Errorf("failed to load metadata: %w", err)
	}

	// 새 메타데이터 준비
	newMetadata := &BackupMetadata{
		LastBackupTime: time.Now(),
		FileChecksums:  make(map[string]string),
		BackupID:       fmt.Sprintf("backup-%d", time.Now().Unix()),
	}

	// 백업할 파일 수집
	filesToBackup := make([]string, 0)
	var totalSize int64

	err = filepath.Walk(b.sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // 에러 무시하고 계속
		}

		// 디렉토리는 스킵
		if info.IsDir() {
			return nil
		}

		// 제외 패턴 확인
		if b.shouldExclude(path) {
			return nil
		}

		relPath, err := filepath.Rel(b.sourceDir, path)
		if err != nil {
			return nil
		}

		// 변경 여부 확인
		needsBackup := false

		if options.UseChecksum {
			// 체크섬 기반 확인
			checksum, err := b.calculateChecksum(path)
			if err != nil {
				b.logger.Warn("Failed to calculate checksum",
					logging.F("file", path),
					logging.F("error", err))
				needsBackup = true
			} else {
				newMetadata.FileChecksums[relPath] = checksum
				if oldChecksum, exists := oldMetadata.FileChecksums[relPath]; !exists || oldChecksum != checksum {
					needsBackup = true
				}
			}
		} else {
			// 타임스탬프 기반 확인
			if info.ModTime().After(oldMetadata.LastBackupTime) {
				needsBackup = true
			}
		}

		if needsBackup {
			filesToBackup = append(filesToBackup, path)
			totalSize += info.Size()
		}

		newMetadata.TotalFiles++
		newMetadata.TotalSize += info.Size()

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to walk source directory: %w", err)
	}

	b.logger.Info("Incremental backup started",
		logging.F("total_files", newMetadata.TotalFiles),
		logging.F("files_to_backup", len(filesToBackup)),
		logging.F("size_to_backup", totalSize))

	// 파일 백업 수행
	if len(filesToBackup) > 0 {
		if err := b.backupFiles(filesToBackup, options); err != nil {
			return nil, err
		}
	}

	// 메타데이터 저장
	if err := b.saveMetadata(newMetadata); err != nil {
		return nil, fmt.Errorf("failed to save metadata: %w", err)
	}

	return newMetadata, nil
}

// backupFiles 실제 파일 백업 수행
func (b *IncrementalBackup) backupFiles(files []string, options *BackupOptions) error {
	// 워커 풀 생성
	workerCount := options.Workers
	if workerCount <= 0 {
		workerCount = 4
	}

	jobs := make(chan string, len(files))
	errors := make(chan error, len(files))
	var wg sync.WaitGroup

	// 워커 시작
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for srcPath := range jobs {
				if err := b.backupSingleFile(srcPath); err != nil {
					errors <- fmt.Errorf("failed to backup %s: %w", srcPath, err)
				}

				// 진행 상황 콜백
				if options.ProgressCallback != nil {
					relPath, _ := filepath.Rel(b.sourceDir, srcPath)
					options.ProgressCallback(len(files)-len(jobs), len(files), relPath)
				}
			}
		}()
	}

	// 작업 추가
	for _, file := range files {
		jobs <- file
	}
	close(jobs)

	// 완료 대기
	wg.Wait()
	close(errors)

	// 에러 수집
	var errs []error
	for err := range errors {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return fmt.Errorf("backup completed with %d errors: %v", len(errs), errs[0])
	}

	return nil
}

// backupSingleFile 단일 파일 백업
func (b *IncrementalBackup) backupSingleFile(srcPath string) error {
	relPath, err := filepath.Rel(b.sourceDir, srcPath)
	if err != nil {
		return err
	}

	dstPath := filepath.Join(b.backupDir, relPath)

	// 대상 디렉토리 생성
	dstDir := filepath.Dir(dstPath)
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		return err
	}

	// 파일 복사
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer func() { _ = srcFile.Close() }()

	dstFile, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer func() { _ = dstFile.Close() }()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		_ = os.Remove(dstPath)
		return err
	}

	// 파일 권한 복사
	if info, err := os.Stat(srcPath); err == nil {
		_ = os.Chmod(dstPath, info.Mode())
	}

	return nil
}

// RestoreOptions 복원 옵션
type RestoreOptions struct {
	// 기존 파일 덮어쓰기
	Overwrite bool
	// 복원 전 검증
	Verify bool
	// 진행 상황 콜백
	ProgressCallback func(current, total int, file string)
}

// Restore 백업에서 복원
func (b *IncrementalBackup) Restore(targetDir string, options *RestoreOptions) error {
	if options == nil {
		options = &RestoreOptions{
			Overwrite: false,
			Verify:    true,
		}
	}

	// 백업 메타데이터 로드
	metadata, err := b.loadMetadata()
	if err != nil {
		return fmt.Errorf("failed to load backup metadata: %w", err)
	}

	b.logger.Info("Starting restore",
		logging.F("backup_id", metadata.BackupID),
		logging.F("total_files", metadata.TotalFiles))

	// 백업된 파일 복원
	restored := 0
	err = filepath.Walk(b.backupDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if info.IsDir() || strings.HasSuffix(path, ".backup-metadata.json") {
			return nil
		}

		relPath, err := filepath.Rel(b.backupDir, path)
		if err != nil {
			return nil
		}

		targetPath := filepath.Join(targetDir, relPath)

		// 대상 파일 존재 확인
		if _, err := os.Stat(targetPath); err == nil && !options.Overwrite {
			b.logger.Warn("Target file exists, skipping",
				logging.F("file", targetPath))
			return nil
		}

		// 검증
		if options.Verify && metadata.FileChecksums != nil {
			if expectedChecksum, exists := metadata.FileChecksums[relPath]; exists {
				actualChecksum, err := b.calculateChecksum(path)
				if err != nil {
					return fmt.Errorf("failed to verify %s: %w", path, err)
				}
				if actualChecksum != expectedChecksum {
					return fmt.Errorf("checksum mismatch for %s", path)
				}
			}
		}

		// 복원
		if err := b.backupSingleFile(path); err != nil {
			return fmt.Errorf("failed to restore %s: %w", path, err)
		}

		restored++
		if options.ProgressCallback != nil {
			options.ProgressCallback(restored, metadata.TotalFiles, relPath)
		}

		return nil
	})
	if err != nil {
		return err
	}

	b.logger.Info("Restore completed",
		logging.F("restored_files", restored))

	return nil
}

// GetBackupInfo 백업 정보 조회
func (b *IncrementalBackup) GetBackupInfo() (*BackupMetadata, error) {
	return b.loadMetadata()
}

// CleanOldBackups 오래된 백업 정리
func (b *IncrementalBackup) CleanOldBackups(keepDays int) error {
	// 증분백업에서는 전체 백업을 유지하므로
	// 별도의 정리 정책이 필요합니다
	// 예: 스냅샷 기반 백업으로 전환
	return nil
}
