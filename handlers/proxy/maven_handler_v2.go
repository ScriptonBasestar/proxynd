package proxy

import (
	"encoding/base64"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"proxynd/configs"
	"proxynd/internal/errors"
	"proxynd/logging"
)

// MavenHandlerV2 Template Method 패턴을 사용하는 Maven 핸들러
type MavenHandlerV2 struct {
	logger logging.Logger
	Config *configs.MavenProxyConfig
}

// NewMavenHandlerV2 새로운 Maven 핸들러 v2 생성
func NewMavenHandlerV2() *MavenHandlerV2 {
	return &MavenHandlerV2{
		logger: logging.GetLogger(),
		Config: &configs.MavenProxyConfig{},
	}
}

// BaseProxyHandler 인터페이스 구현

// Type 프록시 타입 반환
func (h *MavenHandlerV2) Type() string {
	return "maven"
}

// IsEnabled 활성화 상태 확인
func (h *MavenHandlerV2) IsEnabled() bool {
	if err := h.Config.ReadConfig(); err != nil {
		h.logger.Error("Failed to read Maven config", logging.F("error", err))
		return false
	}
	return len(h.Config.Proxies) > 0
}

// GenerateCacheKey 캐시 키 생성
func (h *MavenHandlerV2) GenerateCacheKey(c *fiber.Ctx) string {
	artifactPath := c.Params("*")
	return fmt.Sprintf("maven:%s", strings.ReplaceAll(artifactPath, "/", "_"))
}

// BuildUpstreamURL 업스트림 URL 구성
func (h *MavenHandlerV2) BuildUpstreamURL(c *fiber.Ctx) (string, error) {
	if err := h.Config.ReadConfig(); err != nil {
		return "", fmt.Errorf("maven 설정 로드 실패: %w", err)
	}

	if len(h.Config.Proxies) == 0 {
		return "", fmt.Errorf("maven 리포지토리가 설정되지 않았습니다")
	}

	artifactPath := c.Params("*")
	if artifactPath == "" {
		return "", fmt.Errorf("아티팩트 경로가 비어있습니다")
	}

	// 경로 검증
	if err := h.validateArtifactPath(artifactPath); err != nil {
		return "", err
	}

	// 첫 번째 리포지토리 사용 (추후 로드밸런싱 구현)
	repository := h.Config.Proxies[0]
	if repository.URL == "" {
		return "", fmt.Errorf("maven 리포지토리 URL이 설정되지 않았습니다")
	}

	// URL 구성
	baseURL := strings.TrimRight(repository.URL, "/")
	cleanPath := strings.TrimLeft(artifactPath, "/")

	return fmt.Sprintf("%s/%s", baseURL, cleanPath), nil
}

// TransformRequest 요청 변환
func (h *MavenHandlerV2) TransformRequest(c *fiber.Ctx, upstreamReq *fiber.Agent) error {
	// Maven 특화 헤더 추가
	upstreamReq.Set("X-Maven-Proxy", "ProxyND")
	upstreamReq.Set("User-Agent", "ProxyND/1.0 Maven-Proxy")

	// Maven 클라이언트 정보
	upstreamReq.Set("X-Maven-Client", "ProxyND")

	// 압축 지원
	if c.Get("Accept-Encoding") == "" {
		upstreamReq.Set("Accept-Encoding", "gzip, deflate")
	}

	// SNAPSHOT 아티팩트의 경우 캐시 비활성화
	artifactPath := c.Params("*")
	if h.isSnapshotArtifact(artifactPath) {
		upstreamReq.Set("Cache-Control", "no-cache")
		upstreamReq.Set("Pragma", "no-cache")
	}

	// Basic Auth 설정 (필요한 경우)
	if username, password, err := h.GetUpstreamAuth(c); err == nil && username != "" {
		auth := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", username, password)))
		upstreamReq.Set("Authorization", fmt.Sprintf("Basic %s", auth))
	}

	h.logger.Debug("Maven request transformed",
		logging.F("artifact_path", artifactPath),
		logging.F("method", c.Method()),
		logging.F("is_snapshot", h.isSnapshotArtifact(artifactPath)),
	)

	return nil
}

// TransformResponse 응답 변환
func (h *MavenHandlerV2) TransformResponse(resp []byte, c *fiber.Ctx) ([]byte, error) {
	artifactPath := c.Params("*")

	// POM 파일 처리
	if strings.HasSuffix(artifactPath, ".pom") {
		h.logger.Debug("Processing POM file", logging.F("artifact_path", artifactPath))
		// POM 파일 후처리 로직 (필요 시)
	}

	// 체크섬 파일 처리
	if h.isChecksumFile(artifactPath) {
		h.logger.Debug("Processing checksum file", logging.F("artifact_path", artifactPath))
		// 체크섬 검증 로직 (필요 시)
		if err := h.validateChecksum(resp, artifactPath); err != nil {
			h.logger.Warn("Checksum validation failed",
				logging.F("artifact_path", artifactPath),
				logging.F("error", err),
			)
			// 체크섬 검증 실패는 경고만 로그, 응답은 그대로 전달
		}
	}

	// 메타데이터 파일 처리
	if strings.Contains(artifactPath, "maven-metadata.xml") {
		h.logger.Debug("Processing Maven metadata", logging.F("artifact_path", artifactPath))
		// 메타데이터 후처리 로직 (필요 시)
	}

	return resp, nil
}

// ShouldCache 캐시 정책 결정
func (h *MavenHandlerV2) ShouldCache(c *fiber.Ctx, statusCode int) bool {
	if statusCode != 200 && statusCode != 304 {
		return false
	}

	artifactPath := c.Params("*")

	// SNAPSHOT 아티팩트는 캐시하지 않음
	if h.isSnapshotArtifact(artifactPath) {
		return false
	}

	// JAR 파일은 캐시
	if strings.HasSuffix(artifactPath, ".jar") {
		return true
	}

	// WAR, EAR 파일도 캐시
	if strings.HasSuffix(artifactPath, ".war") || strings.HasSuffix(artifactPath, ".ear") {
		return true
	}

	// POM 파일은 캐시
	if strings.HasSuffix(artifactPath, ".pom") {
		return true
	}

	// 체크섬 파일은 캐시
	if h.isChecksumFile(artifactPath) {
		return true
	}

	// 메타데이터 파일은 짧은 시간만 캐시
	if strings.Contains(artifactPath, "maven-metadata.xml") {
		return true
	}

	return false
}

// GetCacheTTL 캐시 TTL 반환
func (h *MavenHandlerV2) GetCacheTTL(c *fiber.Ctx) time.Duration {
	artifactPath := c.Params("*")

	// 메타데이터 파일은 짧은 TTL (5분)
	if strings.Contains(artifactPath, "maven-metadata.xml") {
		return 5 * time.Minute
	}

	// 체크섬 파일은 아티팩트와 같은 TTL
	if h.isChecksumFile(artifactPath) {
		return 30 * 24 * time.Hour // 30일
	}

	// JAR, WAR, EAR 파일은 긴 TTL (30일)
	if strings.HasSuffix(artifactPath, ".jar") ||
		strings.HasSuffix(artifactPath, ".war") ||
		strings.HasSuffix(artifactPath, ".ear") {
		return 30 * 24 * time.Hour
	}

	// POM 파일은 중간 TTL (7일)
	if strings.HasSuffix(artifactPath, ".pom") {
		return 7 * 24 * time.Hour
	}

	// 기본 1일
	return 24 * time.Hour
}

// HandleError 에러 처리
func (h *MavenHandlerV2) HandleError(err error, _ *fiber.Ctx) error {
	// Maven 도메인 에러로 변환
	if strings.Contains(err.Error(), "리포지토리가 설정되지 않았습니다") {
		return errors.WrapMavenError(err, "MVN004", "Maven 프록시가 비활성화되어 있습니다")
	}

	if strings.Contains(err.Error(), "설정 로드 실패") {
		return errors.WrapMavenError(err, "MVN006", "Maven 설정 파일을 읽을 수 없습니다")
	}

	if strings.Contains(err.Error(), "아티팩트 경로") || strings.Contains(err.Error(), "path") {
		return errors.WrapMavenError(err, "MVN005", "잘못된 Maven 아티팩트 경로입니다")
	}

	if strings.Contains(err.Error(), "SNAPSHOT") {
		return errors.WrapMavenError(err, "MVN007", "SNAPSHOT 아티팩트 다운로드에 실패했습니다")
	}

	// 기본 Maven 에러
	return errors.WrapMavenError(err, "MVN001", "Maven 아티팩트를 찾을 수 없습니다")
}

// 헬프 메서드들

// validateArtifactPath 아티팩트 경로 검증
func (h *MavenHandlerV2) validateArtifactPath(path string) error {
	if strings.Contains(path, "..") {
		return fmt.Errorf("잘못된 아티팩트 경로: '..' 포함")
	}
	if strings.HasPrefix(path, "/") {
		return fmt.Errorf("잘못된 아티팩트 경로: 절대 경로 사용 불가")
	}
	return nil
}

// isSnapshotArtifact SNAPSHOT 아티팩트인지 확인
func (h *MavenHandlerV2) isSnapshotArtifact(path string) bool {
	return strings.Contains(path, "-SNAPSHOT")
}

// isChecksumFile 체크섬 파일인지 확인
func (h *MavenHandlerV2) isChecksumFile(path string) bool {
	return strings.HasSuffix(path, ".sha1") ||
		strings.HasSuffix(path, ".md5") ||
		strings.HasSuffix(path, ".sha256") ||
		strings.HasSuffix(path, ".sha512")
}

// validateChecksum 체크섬 검증 (기본 구현)
func (h *MavenHandlerV2) validateChecksum(checksumData []byte, checksumPath string) error {
	// 기본적인 체크섬 형식 검증
	checksum := strings.TrimSpace(string(checksumData))

	ext := filepath.Ext(checksumPath)
	switch ext {
	case ".sha1":
		if len(checksum) != 40 {
			return fmt.Errorf("잘못된 SHA1 체크섬 길이: %d", len(checksum))
		}
	case ".md5":
		if len(checksum) != 32 {
			return fmt.Errorf("잘못된 MD5 체크섬 길이: %d", len(checksum))
		}
	case ".sha256":
		if len(checksum) != 64 {
			return fmt.Errorf("잘못된 SHA256 체크섬 길이: %d", len(checksum))
		}
	case ".sha512":
		if len(checksum) != 128 {
			return fmt.Errorf("잘못된 SHA512 체크섬 길이: %d", len(checksum))
		}
	}

	h.logger.Debug("Checksum validation passed",
		logging.F("checksum_path", checksumPath),
		logging.F("checksum_length", len(checksum)),
	)

	return nil
}

// AuthenticatedProxyHandler 인터페이스 구현 (선택적)

// GetUpstreamAuth 업스트림 인증 정보 반환
func (h *MavenHandlerV2) GetUpstreamAuth(_ *fiber.Ctx) (string, string, error) {
	if err := h.Config.ReadConfig(); err != nil {
		return "", "", err
	}

	if len(h.Config.Proxies) == 0 {
		return "", "", nil
	}

	repository := h.Config.Proxies[0]
	if repository.BasicAuth.Username != "" {
		return repository.BasicAuth.Username, repository.BasicAuth.Password, nil
	}

	return "", "", nil
}

// ValidateClientAuth 클라이언트 인증 검증
func (h *MavenHandlerV2) ValidateClientAuth(_ *fiber.Ctx) error {
	// 현재는 클라이언트 인증 없음
	return nil
}

// MetricsAwareProxyHandler 인터페이스 구현 (선택적)

// RecordRequestMetrics 요청 메트릭 기록
func (h *MavenHandlerV2) RecordRequestMetrics(c *fiber.Ctx, statusCode int, duration time.Duration) {
	artifactPath := c.Params("*")

	h.logger.Info("Maven request metrics",
		logging.F("status_code", statusCode),
		logging.F("duration_ms", duration.Milliseconds()),
		logging.F("artifact_path", artifactPath),
		logging.F("method", c.Method()),
		logging.F("is_snapshot", h.isSnapshotArtifact(artifactPath)),
		logging.F("is_checksum", h.isChecksumFile(artifactPath)),
	)
}

// RecordCacheMetrics 캐시 메트릭 기록
func (h *MavenHandlerV2) RecordCacheMetrics(cacheKey string, hit bool, size int) {
	status := "miss"
	if hit {
		status = "hit"
	}

	h.logger.Info("Maven cache metrics",
		logging.F("cache_key", cacheKey),
		logging.F("cache_status", status),
		logging.F("size_bytes", size),
	)
}

// Healthable 인터페이스 구현

// HealthCheck Maven 핸들러 헬스체크
func (h *MavenHandlerV2) HealthCheck() error {
	if err := h.Config.ReadConfig(); err != nil {
		return fmt.Errorf("maven 설정 파일 읽기 실패: %w", err)
	}

	if len(h.Config.Proxies) == 0 {
		return fmt.Errorf("maven 리포지토리가 설정되지 않았습니다")
	}

	// 최소 하나의 유효한 리포지토리가 있는지 확인
	hasValidRepository := false
	for _, repository := range h.Config.Proxies {
		if repository.URL != "" {
			h.logger.Debug("Found valid Maven repository",
				logging.F("repository_name", repository.Name),
				logging.F("repository_url", repository.URL),
			)
			hasValidRepository = true
			break
		}
	}

	if !hasValidRepository {
		return fmt.Errorf("유효한 Maven 리포지토리가 설정되지 않았습니다")
	}

	return nil
}
