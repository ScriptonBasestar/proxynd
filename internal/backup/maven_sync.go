package backup

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"proxynd/logging"
)

// SyncMode 동기화 모드
type SyncMode int

const (
	// SyncModeMirror 미러 모드 - 소스에 없는 파일은 대상에서 삭제
	SyncModeMirror SyncMode = iota
	// SyncModeBackup 백업 모드 - 대상의 파일은 삭제하지 않음
	SyncModeBackup
	// SyncModeIncremental 증분 모드 - 변경된 파일만 복사
	SyncModeIncremental
)

// MavenSync Maven 리포지토리 동기화 도구
type MavenSync struct {
	logger logging.Logger
	mu     sync.Mutex
}

// NewMavenSync 새로운 동기화 도구 생성
func NewMavenSync() *MavenSync {
	return &MavenSync{
		logger: logging.GetLogger(),
	}
}

// SyncOptions 동기화 옵션
type SyncOptions struct {
	// 동기화 모드
	Mode SyncMode
	// 체크섬 검증
	VerifyChecksum bool
	// 삭제 전 확인
	ConfirmDelete bool
	// 제외 패턴
	ExcludePatterns []string
	// 병렬 워커 수
	Workers int
	// 진행 상황 콜백
	ProgressCallback func(current, total int64, file string)
	// 드라이런 모드
	DryRun bool
	// 대역폭 제한 (bytes/sec, 0 = 무제한)
	BandwidthLimit int64
}

// SyncResult 동기화 결과
type SyncResult struct {
	FilesCopied  int
	FilesDeleted int
	FilesSkipped int
	BytesCopied  int64
	Duration     time.Duration
	Errors       []error
}

// Sync 디렉토리 동기화 수행
func (s *MavenSync) Sync(source, target string, options *SyncOptions) (*SyncResult, error) {
	if options == nil {
		options = &SyncOptions{
			Mode:    SyncModeBackup,
			Workers: 4,
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	startTime := time.Now()
	result := &SyncResult{
		Errors: make([]error, 0),
	}

	// 소스 디렉토리 확인
	if _, err := os.Stat(source); err != nil {
		return nil, fmt.Errorf("source directory error: %w", err)
	}

	// 대상 디렉토리 생성
	if !options.DryRun {
		if err := os.MkdirAll(target, 0o755); err != nil {
			return nil, fmt.Errorf("failed to create target directory: %w", err)
		}
	}

	s.logger.Info("Starting sync",
		logging.F("source", source),
		logging.F("target", target),
		logging.F("mode", options.Mode),
		logging.F("dry_run", options.DryRun))

	// 1단계: 소스 파일 수집 및 동기화
	sourceFiles := make(map[string]os.FileInfo)
	err := filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // 에러 무시하고 계속
		}

		relPath, err := filepath.Rel(source, path)
		if err != nil {
			return nil
		}

		// 제외 패턴 확인
		if s.shouldExclude(relPath, options.ExcludePatterns) {
			return nil
		}

		if !info.IsDir() {
			sourceFiles[relPath] = info
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to walk source: %w", err)
	}

	// 2단계: 파일 동기화
	if err := s.syncFiles(source, target, sourceFiles, options, result); err != nil {
		return nil, err
	}

	// 3단계: 미러 모드인 경우 대상에만 있는 파일 삭제
	if options.Mode == SyncModeMirror {
		if err := s.cleanupTarget(target, sourceFiles, options, result); err != nil {
			return nil, err
		}
	}

	result.Duration = time.Since(startTime)

	s.logger.Info("Sync completed",
		logging.F("files_copied", result.FilesCopied),
		logging.F("files_deleted", result.FilesDeleted),
		logging.F("files_skipped", result.FilesSkipped),
		logging.F("bytes_copied", result.BytesCopied),
		logging.F("duration", result.Duration))

	return result, nil
}

// syncFiles 파일 동기화 수행
func (s *MavenSync) syncFiles(source, target string, sourceFiles map[string]os.FileInfo,
	options *SyncOptions, result *SyncResult,
) error {
	// 워커 풀 설정
	type syncJob struct {
		relPath string
		info    os.FileInfo
	}

	jobs := make(chan syncJob, len(sourceFiles))
	var wg sync.WaitGroup

	// 대역폭 제한기
	var bandwidthLimiter *rateLimiter
	if options.BandwidthLimit > 0 {
		bandwidthLimiter = newRateLimiter(options.BandwidthLimit)
	}

	// 워커 시작
	for i := 0; i < options.Workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				srcPath := filepath.Join(source, job.relPath)
				dstPath := filepath.Join(target, job.relPath)

				needsSync, err := s.needsSync(srcPath, dstPath, job.info, options)
				if err != nil {
					result.Errors = append(result.Errors, err)
					continue
				}

				if !needsSync {
					result.FilesSkipped++
					continue
				}

				if options.DryRun {
					s.logger.Info("Would sync",
						logging.F("file", job.relPath))
					result.FilesCopied++
					continue
				}

				// 파일 복사
				copied, err := s.copyFileWithLimit(srcPath, dstPath, bandwidthLimiter)
				if err != nil {
					result.Errors = append(result.Errors,
						fmt.Errorf("failed to copy %s: %w", job.relPath, err))
					continue
				}

				result.FilesCopied++
				result.BytesCopied += copied

				if options.ProgressCallback != nil {
					options.ProgressCallback(result.BytesCopied, 0, job.relPath)
				}
			}
		}()
	}

	// 작업 추가
	for relPath, info := range sourceFiles {
		jobs <- syncJob{relPath: relPath, info: info}
	}
	close(jobs)

	wg.Wait()
	return nil
}

// needsSync 동기화 필요 여부 확인
func (s *MavenSync) needsSync(srcPath, dstPath string, srcInfo os.FileInfo, options *SyncOptions) (bool, error) {
	dstInfo, err := os.Stat(dstPath)
	if err != nil {
		if os.IsNotExist(err) {
			return true, nil // 대상 파일이 없으면 동기화 필요
		}
		return false, err
	}

	// 크기가 다르면 동기화 필요
	if srcInfo.Size() != dstInfo.Size() {
		return true, nil
	}

	// 증분 모드에서는 수정 시간 확인
	if options.Mode == SyncModeIncremental {
		if srcInfo.ModTime().After(dstInfo.ModTime()) {
			return true, nil
		}
	}

	// 체크섬 검증이 활성화된 경우
	if options.VerifyChecksum {
		srcChecksum, err := s.calculateFileChecksum(srcPath)
		if err != nil {
			return false, err
		}

		dstChecksum, err := s.calculateFileChecksum(dstPath)
		if err != nil {
			return false, err
		}

		if srcChecksum != dstChecksum {
			return true, nil
		}
	}

	return false, nil
}

// cleanupTarget 대상 디렉토리 정리 (미러 모드)
func (s *MavenSync) cleanupTarget(target string, sourceFiles map[string]os.FileInfo,
	options *SyncOptions, result *SyncResult,
) error {
	return filepath.Walk(target, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(target, path)
		if err != nil {
			return nil
		}

		// 소스에 없는 파일인 경우
		if _, exists := sourceFiles[relPath]; !exists {
			if s.shouldExclude(relPath, options.ExcludePatterns) {
				return nil
			}

			if options.DryRun {
				s.logger.Info("Would delete",
					logging.F("file", relPath))
				result.FilesDeleted++
				return nil
			}

			if options.ConfirmDelete {
				// 실제 구현에서는 사용자 확인 로직 추가
				s.logger.Warn("Skipping delete (confirmation required)",
					logging.F("file", relPath))
				return nil
			}

			if err := os.Remove(path); err != nil {
				result.Errors = append(result.Errors,
					fmt.Errorf("failed to delete %s: %w", relPath, err))
			} else {
				result.FilesDeleted++
			}
		}

		return nil
	})
}

// copyFileWithLimit 대역폭 제한이 있는 파일 복사
func (s *MavenSync) copyFileWithLimit(src, dst string, limiter *rateLimiter) (int64, error) {
	// 대상 디렉토리 생성
	dstDir := filepath.Dir(dst)
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		return 0, err
	}

	srcFile, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer func() { _ = srcFile.Close() }()

	dstFile, err := os.Create(dst)
	if err != nil {
		return 0, err
	}
	defer func() { _ = dstFile.Close() }()

	var copied int64
	if limiter != nil {
		// 대역폭 제한 적용
		copied, err = limiter.CopyWithLimit(dstFile, srcFile)
	} else {
		copied, err = io.Copy(dstFile, srcFile)
	}

	if err != nil {
		_ = os.Remove(dst)
		return 0, err
	}

	// 파일 권한 복사
	if info, err := os.Stat(src); err == nil {
		_ = os.Chmod(dst, info.Mode())
		// Maven 아티팩트는 수정 시간 유지가 중요
		_ = os.Chtimes(dst, info.ModTime(), info.ModTime())
	}

	return copied, nil
}

// calculateFileChecksum 파일 체크섬 계산
func (s *MavenSync) calculateFileChecksum(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = file.Close() }()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// shouldExclude 제외 패턴 확인
func (s *MavenSync) shouldExclude(path string, patterns []string) bool {
	// 기본 제외 패턴
	defaultExcludes := []string{
		"*.tmp",
		"*.lock",
		"*.part",
		"_remote.repositories",
		"_maven.repositories",
		"resolver-status.properties",
		"*.lastUpdated",
	}

	allPatterns := append(defaultExcludes, patterns...)

	for _, pattern := range allPatterns {
		if matched, _ := filepath.Match(pattern, filepath.Base(path)); matched {
			return true
		}
		// 경로 전체 매칭
		if strings.Contains(path, pattern) {
			return true
		}
	}

	return false
}

// rateLimiter 대역폭 제한기
type rateLimiter struct {
	bytesPerSecond int64
	lastTime       time.Time
	mu             sync.Mutex
}

func newRateLimiter(bytesPerSecond int64) *rateLimiter {
	return &rateLimiter{
		bytesPerSecond: bytesPerSecond,
		lastTime:       time.Now(),
	}
}

func (r *rateLimiter) CopyWithLimit(dst io.Writer, src io.Reader) (int64, error) {
	buf := make([]byte, 32*1024) // 32KB 버퍼
	var written int64

	for {
		nr, err := src.Read(buf)
		if nr > 0 {
			// 대역폭 제한 적용
			r.waitForQuota(int64(nr))

			nw, err := dst.Write(buf[0:nr])
			if nw > 0 {
				written += int64(nw)
			}
			if err != nil {
				return written, err
			}
			if nr != nw {
				return written, io.ErrShortWrite
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return written, err
		}
	}

	return written, nil
}

func (r *rateLimiter) waitForQuota(bytes int64) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(r.lastTime)

	// 전송 가능한 바이트 계산
	allowedBytes := int64(elapsed.Seconds() * float64(r.bytesPerSecond))

	if bytes > allowedBytes {
		// 대기 시간 계산
		waitTime := time.Duration(float64(bytes-allowedBytes) / float64(r.bytesPerSecond) * float64(time.Second))
		time.Sleep(waitTime)
		r.lastTime = time.Now()
	} else {
		r.lastTime = now
	}
}
