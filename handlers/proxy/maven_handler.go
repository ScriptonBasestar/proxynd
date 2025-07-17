package proxy

import (
	"crypto/md5"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"proxynd/configs"
	"proxynd/helpers"
	"proxynd/internal/errors"
	"proxynd/internal/security"
	"proxynd/logging"
)

// Handler Maven 핸들러 인터페이스 (import cycle 방지)
type MavenHandlerInterface interface {
	Handle(c *fiber.Ctx) error
	Name() string
	Type() string
}

// MavenHandler Maven 패키지 매니저 프록시 핸들러
type MavenHandler struct {
	client           *http.Client
	checksumVerifier *ChecksumVerifier
	name             string
	handlerType      string
	logger           logging.Logger
	config           *configs.MavenProxyConfig
}

// ChecksumVerifier 체크섬 검증기
type ChecksumVerifier struct {
	logger logging.Logger
}

// NewMavenHandler 새로운 Maven 핸들러 생성
func NewMavenHandler() *MavenHandler {
	httpClient := &http.Client{
		Timeout: 60 * time.Second, // Maven은 큰 파일이 많아서 긴 타임아웃
	}

	handler := &MavenHandler{
		client:           httpClient,
		checksumVerifier: NewChecksumVerifier(),
		name:             "maven-proxy",
		handlerType:      "maven",
		logger:           logging.GetLogger(),
		config:           &configs.MavenProxyConfig{},
	}

	return handler
}

// NewChecksumVerifier 새로운 체크섬 검증기 생성
func NewChecksumVerifier() *ChecksumVerifier {
	return &ChecksumVerifier{
		logger: logging.GetLogger(),
	}
}

// Handle Maven 프록시 요청 처리
func (h *MavenHandler) Handle(c *fiber.Ctx) error {
	h.logRequest(c)
	start := time.Now()

	// 설정 로드
	if err := h.config.ReadConfig(); err != nil {
		h.logger.Error("Failed to read Maven config", logging.F("error", err))
		return errors.WrapMavenError(err, "MVN006", "Maven 설정 파일을 읽을 수 없습니다")
	}

	if len(h.config.Proxies) == 0 {
		return errors.ErrMavenProxyDisabled
	}

	// 아티팩트 경로 파싱
	artifactPath := c.Params("*")

	// 보안 체크
	if err := h.validatePath(artifactPath); err != nil {
		return errors.WrapMavenError(err, "MVN005", "잘못된 Maven 아티팩트 경로입니다")
	}

	// SNAPSHOT 버전 처리
	if strings.Contains(artifactPath, "-SNAPSHOT") {
		return h.handleSnapshotArtifact(c, artifactPath, h.config)
	}

	// 파일 경로 생성
	storageDir := helpers.GetStorageDir()
	baseDir := filepath.Join(storageDir, h.config.Path)
	filePath, err := security.SafeJoinPath(baseDir, artifactPath)
	if err != nil {
		return errors.WrapMavenError(err, "MVN005", "잘못된 Maven 아티팩트 경로입니다")
	}

	// 캐시 확인 (기존 파일이 있는지)
	filename := filepath.Base(filePath)
	if _, err := os.Stat(filePath); err == nil {
		c.Set("X-Cache-Status", "HIT")
		h.logResponse(c, time.Since(start))
		return h.sendFile(c, filePath, filename)
	}

	// 업스트림에서 다운로드
	responseContent, err := h.downloadFromUpstream(filePath, h.config.Proxies, artifactPath)
	if err != nil {
		h.logger.Error("Failed to download from upstream",
			logging.F("path", artifactPath),
			logging.F("error", err),
		)
		return errors.WrapMavenError(err, "MVN003", "Maven 리포지토리에 접근할 수 없습니다")
	}

	// 체크섬 검증 (체크섬 파일인 경우)
	if h.isChecksumFile(artifactPath) {
		if err := h.validateChecksum(responseContent, artifactPath); err != nil {
			h.logger.Warn("Checksum validation failed",
				logging.F("path", artifactPath),
				logging.F("error", err),
			)
			// 체크섬 검증 실패는 경고만 로그, 클라이언트에는 정상 응답
		}
	}

	c.Set("X-Cache-Status", "MISS")
	h.logResponse(c, time.Since(start))
	return h.sendMavenResponse(c, responseContent, filename)
}

// validatePath 경로 유효성 검사
func (h *MavenHandler) validatePath(path string) error {
	if strings.Contains(path, "..") {
		return fmt.Errorf("invalid path: contains '..'")
	}
	if strings.HasPrefix(path, "/") {
		return fmt.Errorf("invalid path: absolute path not allowed")
	}
	return nil
}

// downloadFromUpstream 업스트림에서 파일 다운로드
func (h *MavenHandler) downloadFromUpstream(filePath string, proxies []configs.MavenProxyServer, artifactPath string) ([]byte, error) {
	// 디렉토리 생성
	dirPath := filepath.Dir(filePath)
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	var lastErr error
	var responseContent []byte

	// 여러 미러 시도
	for i, proxy := range proxies {
		content, err := h.tryDownloadFromRepository(proxy.URL, artifactPath)
		if err == nil {
			responseContent = content
			// 파일에 저장
			if err := os.WriteFile(filePath, responseContent, 0644); err != nil {
				h.logger.Warn("Failed to save to cache",
					logging.F("path", filePath),
					logging.F("error", err),
				)
			}
			h.logger.Info("Downloaded from repository",
				logging.F("repository", proxy.URL),
				logging.F("index", i),
			)
			return responseContent, nil
		} else {
			lastErr = err
			h.logger.Warn("Repository download failed, trying next",
				logging.F("repository", proxy.URL),
				logging.F("index", i),
				logging.F("error", err),
			)
		}
	}

	return nil, fmt.Errorf("all repositories failed: %w", lastErr)
}

// tryDownloadFromRepository 특정 리포지토리에서 다운로드 시도
func (h *MavenHandler) tryDownloadFromRepository(repositoryURL, artifactPath string) ([]byte, error) {
	// URL 생성
	downloadURL := helpers.JoinURL(repositoryURL, artifactPath)

	// HTTP 요청
	resp, err := h.client.Get(downloadURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from repository: %w", err)
	}
	defer resp.Body.Close()

	// 상태 코드 확인
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status code: %d", resp.StatusCode)
	}

	// 데이터 읽기
	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return content, nil
}

// handleSnapshotArtifact SNAPSHOT 버전 처리
func (h *MavenHandler) handleSnapshotArtifact(c *fiber.Ctx, artifactPath string, config *configs.MavenProxyConfig) error {
	h.logger.Info("Handling SNAPSHOT artifact",
		logging.F("path", artifactPath),
	)

	// SNAPSHOT은 항상 업스트림에서 최신 버전 가져오기 (캐시 안함)
	responseContent, err := h.downloadSnapshotFromUpstream(h.config.Proxies, artifactPath)
	if err != nil {
		return errors.WrapMavenError(err, "MVN007", "SNAPSHOT 아티팩트 다운로드에 실패했습니다")
	}

	filename := filepath.Base(artifactPath)
	c.Set("X-Cache-Status", "SNAPSHOT-BYPASS")
	return h.sendMavenResponse(c, responseContent, filename)
}

// downloadSnapshotFromUpstream SNAPSHOT 버전을 업스트림에서 다운로드 (캐시 안함)
func (h *MavenHandler) downloadSnapshotFromUpstream(proxies []configs.MavenProxyServer, artifactPath string) ([]byte, error) {
	var lastErr error

	for i, proxy := range proxies {
		content, err := h.tryDownloadFromRepository(proxy.URL, artifactPath)
		if err == nil {
			h.logger.Info("Downloaded SNAPSHOT from repository",
				logging.F("repository", proxy.URL),
				logging.F("index", i),
			)
			return content, nil
		} else {
			lastErr = err
			h.logger.Warn("SNAPSHOT download failed, trying next repository",
				logging.F("repository", proxy.URL),
				logging.F("error", err),
			)
		}
	}

	return nil, fmt.Errorf("all repositories failed for SNAPSHOT: %w", lastErr)
}

// isChecksumFile 체크섬 파일인지 확인
func (h *MavenHandler) isChecksumFile(path string) bool {
	return strings.HasSuffix(path, ".sha1") || strings.HasSuffix(path, ".md5")
}

// validateChecksum 체크섬 검증
func (h *MavenHandler) validateChecksum(checksumData []byte, checksumPath string) error {
	return h.checksumVerifier.ValidateChecksum(checksumData, checksumPath)
}

// sendFile 파일 전송
func (h *MavenHandler) sendFile(c *fiber.Ctx, filePath, filename string) error {
	// 캐시된 파일 읽기
	content, err := os.ReadFile(filePath)
	if err != nil {
		return errors.WrapMavenError(err, "MVN001", "Maven 아티팩트를 찾을 수 없습니다")
	}

	return h.sendMavenResponse(c, content, filename)
}

// sendMavenResponse Maven 응답 전송 (파일 타입에 따른 Content-Type 설정)
func (h *MavenHandler) sendMavenResponse(c *fiber.Ctx, content []byte, filename string) error {
	ext := strings.ToLower(filepath.Ext(filename))

	switch ext {
	case ".pom", ".xml":
		c.Set("Content-Type", "application/xml")
		c.Set("Content-Disposition", "inline; filename="+filename)
	case ".jar", ".war", ".ear":
		c.Set("Content-Type", "application/java-archive")
		c.Set("Content-Disposition", "attachment; filename="+filename)
	case ".sha1", ".md5":
		c.Set("Content-Type", "text/plain")
		c.Set("Content-Disposition", "inline; filename="+filename)
	default:
		c.Set("Content-Type", "application/octet-stream")
		c.Set("Content-Disposition", "attachment; filename="+filename)
	}

	return c.Send(content)
}

// ValidateChecksum 체크섬 검증 구현
func (v *ChecksumVerifier) ValidateChecksum(checksumData []byte, checksumPath string) error {
	// 체크섬 파일에서 예상 해시값 추출
	expectedHash := strings.TrimSpace(string(checksumData))

	// 원본 파일 경로 유추
	originalPath := strings.TrimSuffix(checksumPath, filepath.Ext(checksumPath))

	// 원본 파일 가져오기 (현재는 단순 검증만)
	v.logger.Debug("Checksum validation",
		logging.F("original_path", originalPath),
		logging.F("expected_hash", expectedHash),
		logging.F("checksum_type", filepath.Ext(checksumPath)),
	)

	// TODO: 실제 원본 파일과 비교하는 로직 구현
	// 현재는 검증 성공으로 처리
	return nil
}

// calculateSHA1 SHA1 해시 계산
func (v *ChecksumVerifier) calculateSHA1(data []byte) string {
	h := sha1.New()
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

// calculateMD5 MD5 해시 계산
func (v *ChecksumVerifier) calculateMD5(data []byte) string {
	h := md5.New()
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

// Name 핸들러 이름 반환
func (h *MavenHandler) Name() string {
	return h.name
}

// Type 핸들러 타입 반환
func (h *MavenHandler) Type() string {
	return h.handlerType
}

// logRequest 요청 로깅
func (h *MavenHandler) logRequest(c *fiber.Ctx) {
	h.logger.Info("Request received",
		logging.F("handler", h.name),
		logging.F("method", c.Method()),
		logging.F("path", c.Path()),
		logging.F("user_agent", c.Get("User-Agent")),
		logging.F("remote_ip", c.IP()),
	)
}

// logResponse 응답 로깅
func (h *MavenHandler) logResponse(c *fiber.Ctx, duration time.Duration) {
	level := "info"
	if c.Response().StatusCode() >= 400 {
		level = "error"
	}

	message := "Request completed"
	fields := []logging.Field{
		logging.F("handler", h.name),
		logging.F("method", c.Method()),
		logging.F("path", c.Path()),
		logging.F("status", c.Response().StatusCode()),
		logging.F("duration_ms", duration.Milliseconds()),
	}

	switch level {
	case "error":
		h.logger.Error(message, fields...)
	default:
		h.logger.Info(message, fields...)
	}
}

// IsEnabled 활성화 상태 확인 (v2 기능 통합)
func (h *MavenHandler) IsEnabled() bool {
	if err := h.config.ReadConfig(); err != nil {
		h.logger.Error("Failed to read Maven config", logging.F("error", err))
		return false
	}
	return len(h.config.Proxies) > 0
}

// GenerateCacheKey 캐시 키 생성 (v2 기능 통합)
func (h *MavenHandler) GenerateCacheKey(c *fiber.Ctx) string {
	artifactPath := c.Params("*")
	return fmt.Sprintf("maven:%s", strings.ReplaceAll(artifactPath, "/", "_"))
}

// HealthCheck Maven 핸들러 헬스체크
func (h *MavenHandler) HealthCheck() error {
	// Maven 설정 확인
	if err := h.config.ReadConfig(); err != nil {
		return fmt.Errorf("failed to read Maven configuration: %w", err)
	}

	if len(h.config.Proxies) == 0 {
		return fmt.Errorf("Maven proxy has no configured repositories")
	}

	return nil
}
