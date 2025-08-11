package proxy

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"proxynd/helpers"
	"proxynd/internal/config"
	"proxynd/internal/errors"
	"proxynd/internal/security"
	"proxynd/logging"
)

// MavenHandler V1과 V2의 기능을 통합한 Maven 핸들러
type MavenHandler struct {
	logger     logging.Logger
	Config     *config.MavenProxySettings
	storageDir string
}

// NewMavenHandler 새로운 Maven 핸들러 생성
func NewMavenHandler() *MavenHandler {
	return &MavenHandler{
		logger:     logging.GetLogger(),
		Config:     &config.MavenProxySettings{},
		storageDir: helpers.GetStorageDir(),
	}
}

// Type 프록시 타입 반환
func (h *MavenHandler) Type() string {
	return proxyTypeMaven
}

// IsEnabled 활성화 상태 확인
func (h *MavenHandler) IsEnabled() bool {
	if err := h.Config.ReadConfig(); err != nil {
		h.logger.Error("Failed to read Maven config", logging.F("error", err))
		return false
	}
	return len(h.Config.Proxies) > 0
}

// GenerateCacheKey 캐시 키 생성
func (h *MavenHandler) GenerateCacheKey(c *fiber.Ctx) string {
	artifactPath := c.Params("*")
	return fmt.Sprintf("maven:%s", strings.ReplaceAll(artifactPath, "/", "_"))
}

// Handle 파일 시스템 저장 및 다중 레포지토리 재시도를 포함한 핸들러
func (h *MavenHandler) Handle(c *fiber.Ctx) error {
	proxyType := h.Type()
	startTime := time.Now()

	// 요청 로깅
	h.logger.Info("Maven request started",
		logging.F("path", c.Path()),
		logging.F("method", c.Method()),
	)

	defer func() {
		h.logger.Info("Maven request completed",
			logging.F("duration_ms", time.Since(startTime).Milliseconds()),
		)
	}()

	// 1. 프록시 활성화 확인
	if !h.IsEnabled() {
		return errors.NewError("PROXY001", "프록시가 비활성화되어 있습니다").
			WithDomain(proxyType).
			Build()
	}

	// 2. 아티팩트 경로 파싱
	artifactPath := c.Params("*")

	// SNAPSHOT 버전 확인
	isSnapshot := strings.Contains(artifactPath, "-SNAPSHOT")

	// 3. 파일 시스템 캐시 확인 (SNAPSHOT이 아닌 경우만)
	if !isSnapshot {
		// 보안 경로 조합
		safeBasePath := filepath.Join(h.storageDir, "maven")
		safePath, err := security.SafeJoinPath(safeBasePath, artifactPath)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid path")
		}

		// 파일이 이미 존재하는지 확인
		if fileInfo, err := os.Stat(safePath); err == nil && !fileInfo.IsDir() {
			h.logger.Debug("Serving from cache", logging.F("path", safePath))

			// Content-Type 설정
			c.Set("Content-Type", getMavenContentType(artifactPath))

			// Content-Disposition 설정
			filename := filepath.Base(artifactPath)
			if shouldInlineMaven(artifactPath) {
				c.Set("Content-Disposition", "inline")
			} else {
				c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
			}

			c.Set("X-Cache-Status", "HIT")
			c.Set("X-Proxy-Type", proxyType)
			c.Set("X-Maven-Snapshot", "false")

			return c.SendFile(safePath)
		}
	}

	// 4. 업스트림에서 가져오기 (모든 레포지토리 시도)
	if err := h.Config.ReadConfig(); err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to load configuration")
	}

	if len(h.Config.Proxies) == 0 {
		return c.Status(fiber.StatusNotFound).SendString("No Maven repositories configured")
	}

	var lastErr error
	for i, repo := range h.Config.Proxies {
		if repo.URL == "" {
			continue
		}

		baseURL := strings.TrimRight(repo.URL, "/")
		cleanPath := strings.TrimLeft(artifactPath, "/")
		upstreamURL := fmt.Sprintf("%s/%s", baseURL, cleanPath)

		h.logger.Debug("Trying repository",
			logging.F("repo_index", i),
			logging.F("repo_name", repo.Name),
			logging.F("url", upstreamURL),
		)

		// Fiber Agent로 요청
		agent := fiber.Get(upstreamURL)

		// 헤더 설정
		agent.Set("User-Agent", "ProxyND/1.0 Maven-Proxy")
		agent.Set("X-Maven-Proxy", "ProxyND")

		// 레포지토리 인증이 있는 경우
		if repo.BasicAuth.Username != "" && repo.BasicAuth.Password != "" {
			agent.BasicAuth(repo.BasicAuth.Username, repo.BasicAuth.Password)
		}

		// 요청 실행
		statusCode, body, errs := agent.Bytes()
		if len(errs) > 0 {
			lastErr = errs[0]
			h.logger.Error("Failed to fetch from repository",
				logging.F("repo_index", i),
				logging.F("repo_name", repo.Name),
				logging.F("error", lastErr),
			)
			continue
		}

		if statusCode != fiber.StatusOK {
			lastErr = fmt.Errorf("upstream returned status %d", statusCode)
			h.logger.Error("Non-200 status from repository",
				logging.F("repo_index", i),
				logging.F("repo_name", repo.Name),
				logging.F("status_code", statusCode),
			)
			continue
		}

		// 체크섬 검증 (체크섬 파일인 경우 제외)
		if !isChecksumFile(artifactPath) && h.Config.UseCache {
			// 체크섬 파일 다운로드 시도
			checksumURL := upstreamURL + ".sha1"
			checksumAgent := fiber.Get(checksumURL)
			if repo.BasicAuth.Username != "" && repo.BasicAuth.Password != "" {
				checksumAgent.BasicAuth(repo.BasicAuth.Username, repo.BasicAuth.Password)
			}

			if checksumCode, checksumBody, checksumErrs := checksumAgent.Bytes(); len(checksumErrs) == 0 && checksumCode == fiber.StatusOK { //nolint:lll
				expectedChecksum := strings.TrimSpace(string(checksumBody))
				// SHA1 계산
				hash := sha1.New()
				hash.Write(body)
				actualChecksum := hex.EncodeToString(hash.Sum(nil))

				if expectedChecksum != actualChecksum {
					h.logger.Error("Checksum verification failed",
						logging.F("expected", expectedChecksum),
						logging.F("actual", actualChecksum),
					)
					lastErr = fmt.Errorf("checksum verification failed")
					continue
				}
			}
		}

		// 성공적으로 가져온 경우 파일로 저장 (SNAPSHOT이 아닌 경우만)
		if !isSnapshot {
			safeBasePath := filepath.Join(h.storageDir, "maven")
			safePath, err := security.SafeJoinPath(safeBasePath, artifactPath)
			if err == nil {
				dir := filepath.Dir(safePath)
				if err := os.MkdirAll(dir, 0o755); err != nil {
					h.logger.Error("Failed to create directory",
						logging.F("dir", dir),
						logging.F("error", err),
					)
				} else {
					// 임시 파일에 쓰고 원자적으로 이동
					tmpFile := safePath + ".tmp"
					if err := os.WriteFile(tmpFile, body, 0o644); err == nil {
						if err := os.Rename(tmpFile, safePath); err != nil {
							_ = os.Remove(tmpFile)
							h.logger.Error("Failed to rename temp file",
								logging.F("error", err),
							)
						}
					}
				}
			}
		}

		// 응답 헤더 설정
		c.Set("Content-Type", getMavenContentType(artifactPath))

		filename := filepath.Base(artifactPath)
		if shouldInlineMaven(artifactPath) {
			c.Set("Content-Disposition", "inline")
		} else {
			c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
		}

		c.Set("X-Cache-Status", "MISS")
		c.Set("X-Proxy-Type", proxyType)
		c.Set("X-Maven-Snapshot", fmt.Sprintf("%v", isSnapshot))
		c.Set("X-Maven-Repository", repo.Name)

		// 메트릭 기록
		h.RecordRequestMetrics(c, statusCode, time.Since(startTime))

		return c.Send(body)
	}

	// 모든 레포지토리 실패
	if lastErr != nil {
		return h.HandleError(lastErr, c)
	}

	return c.Status(fiber.StatusNotFound).SendString("Artifact not found in any repository")
}

// ShouldCache 캐시 정책 결정
func (h *MavenHandler) ShouldCache(c *fiber.Ctx, statusCode int) bool {
	if statusCode != 200 && statusCode != 304 {
		return false
	}

	path := c.Path()

	// SNAPSHOT 버전은 캐시하지 않음
	if strings.Contains(path, "-SNAPSHOT") || strings.Contains(path, "SNAPSHOT") {
		return false
	}

	// 메타데이터 파일은 짧은 TTL로 캐시
	if strings.HasSuffix(path, "maven-metadata.xml") {
		return true
	}

	// 아티팩트 파일들은 캐시
	if strings.HasSuffix(path, ".jar") || strings.HasSuffix(path, ".war") ||
		strings.HasSuffix(path, ".ear") || strings.HasSuffix(path, ".pom") {
		return true
	}

	// 체크섬 파일들도 캐시
	if isChecksumFile(path) {
		return true
	}

	return false
}

// GetCacheTTL 캐시 TTL 반환
func (h *MavenHandler) GetCacheTTL(c *fiber.Ctx) time.Duration {
	path := c.Path()

	// SNAPSHOT 버전은 캐시하지 않음
	if strings.Contains(path, "-SNAPSHOT") {
		return 0
	}

	// 메타데이터 파일은 짧은 TTL (30분)
	if strings.HasSuffix(path, "maven-metadata.xml") {
		return 30 * time.Minute
	}

	// 체크섬 파일은 중간 TTL (1일)
	if isChecksumFile(path) {
		return 24 * time.Hour
	}

	// 릴리즈 아티팩트는 긴 TTL (30일)
	if strings.HasSuffix(path, ".jar") || strings.HasSuffix(path, ".war") ||
		strings.HasSuffix(path, ".ear") || strings.HasSuffix(path, ".pom") {
		return 30 * 24 * time.Hour
	}

	// 기본 1일
	return 24 * time.Hour
}

// HandleError 에러 처리
func (h *MavenHandler) HandleError(err error, _ *fiber.Ctx) error {
	// Maven 도메인 에러로 변환
	if strings.Contains(err.Error(), "레포지토리가 설정되지 않았습니다") {
		return errors.WrapMavenError(err, "MAVEN002", "Maven 레포지토리 서버에 접근할 수 없습니다")
	}

	if strings.Contains(err.Error(), "설정 로드 실패") {
		return errors.WrapMavenError(err, "MAVEN006", "Maven 설정 파일을 읽을 수 없습니다")
	}

	if strings.Contains(err.Error(), "checksum") {
		return errors.WrapMavenError(err, "MAVEN003", "체크섬 검증에 실패했습니다")
	}

	if strings.Contains(err.Error(), "path") {
		return errors.WrapMavenError(err, "MAVEN005", "잘못된 아티팩트 경로입니다")
	}

	// 기본 Maven 에러
	return errors.WrapMavenError(err, "MAVEN001", "Maven 아티팩트를 찾을 수 없습니다")
}

// RecordRequestMetrics 요청 메트릭 기록
func (h *MavenHandler) RecordRequestMetrics(c *fiber.Ctx, statusCode int, duration time.Duration) {
	artifactPath := c.Params("*")
	h.logger.Info("Maven request metrics",
		logging.F("status_code", statusCode),
		logging.F("duration_ms", duration.Milliseconds()),
		logging.F("artifact_path", artifactPath),
		logging.F("method", c.Method()),
		logging.F("is_snapshot", strings.Contains(artifactPath, "-SNAPSHOT")),
		logging.F("is_checksum", isChecksumFile(artifactPath)),
	)
}

// getMavenContentType Maven 아티팩트 타입에 따른 Content-Type 반환
func getMavenContentType(path string) string {
	switch {
	case strings.HasSuffix(path, ".jar"):
		return mimeApplicationJavaArchive
	case strings.HasSuffix(path, ".war"):
		return mimeApplicationJavaArchive
	case strings.HasSuffix(path, ".ear"):
		return mimeApplicationJavaArchive
	case strings.HasSuffix(path, ".pom"):
		return mimeApplicationXML
	case strings.HasSuffix(path, ".xml"):
		return mimeApplicationXML
	case strings.HasSuffix(path, ".sha1"):
		return mimeTextPlain
	case strings.HasSuffix(path, ".sha256"):
		return mimeTextPlain
	case strings.HasSuffix(path, ".md5"):
		return mimeTextPlain
	case strings.HasSuffix(path, ".asc"):
		return mimeApplicationPGPSignature
	case strings.HasSuffix(path, ".zip"):
		return mimeApplicationZip
	case strings.HasSuffix(path, ".tar.gz"):
		return "application/gzip"
	default:
		return mimeApplicationOctetStream
	}
}

// shouldInlineMaven 인라인으로 표시할 파일인지 확인
func shouldInlineMaven(path string) bool {
	return strings.HasSuffix(path, ".pom") ||
		strings.HasSuffix(path, ".xml") ||
		strings.HasSuffix(path, ".sha1") ||
		strings.HasSuffix(path, ".sha256") ||
		strings.HasSuffix(path, ".md5")
}

// isChecksumFile 체크섬 파일인지 확인
func isChecksumFile(path string) bool {
	return strings.HasSuffix(path, ".sha1") ||
		strings.HasSuffix(path, ".sha256") ||
		strings.HasSuffix(path, ".md5")
}
