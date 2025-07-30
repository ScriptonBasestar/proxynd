package proxy

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"proxynd/internal/config"
	"proxynd/internal/errors"
	"proxynd/internal/pool"
	"proxynd/logging"
)

// MavenHandlerV3 Base Handler 패턴을 사용하는 새로운 Maven 핸들러
type MavenHandlerV3 struct {
	*BaseProxyHandler
	Config        *config.MavenProxySettings
	clientFactory *pool.ProxyClientFactory
}

// NewMavenHandlerV3 새로운 Maven 핸들러 생성
func NewMavenHandlerV3() *MavenHandlerV3 {
	return &MavenHandlerV3{
		BaseProxyHandler: NewBaseProxyHandler(),
		Config:           &config.MavenProxySettings{},
		clientFactory:    pool.GetGlobalClientFactory(),
	}
}

// Type 프록시 타입 반환
func (h *MavenHandlerV3) Type() string {
	return "maven"
}

// IsEnabled 활성화 상태 확인
func (h *MavenHandlerV3) IsEnabled() bool {
	if err := h.LoadConfig(); err != nil {
		h.GetLogger().Error("Failed to read Maven config", logging.F("error", err))
		return false
	}
	return len(h.Config.Proxies) > 0
}

// LoadConfig 설정 로드
func (h *MavenHandlerV3) LoadConfig() error {
	return h.Config.ReadConfig()
}

// GenerateCacheKey 캐시 키 생성
func (h *MavenHandlerV3) GenerateCacheKey(c *fiber.Ctx) string {
	artifactPath := c.Params("*")
	return fmt.Sprintf("maven:%s", strings.ReplaceAll(artifactPath, "/", "_"))
}

// BuildUpstreamURL 업스트림 URL 생성 (첫 번째 레포지토리 사용)
func (h *MavenHandlerV3) BuildUpstreamURL(c *fiber.Ctx) (string, error) {
	artifactPath := c.Params("*")

	if len(h.Config.Proxies) == 0 {
		return "", fmt.Errorf("Maven 레포지토리가 설정되지 않았습니다")
	}

	repo := h.Config.Proxies[0] // 첫 번째 레포지토리 사용
	if repo.URL == "" {
		return "", fmt.Errorf("Maven 레포지토리 URL이 설정되지 않았습니다")
	}

	baseURL := strings.TrimRight(repo.URL, "/")
	cleanPath := strings.TrimLeft(artifactPath, "/")

	return fmt.Sprintf("%s/%s", baseURL, cleanPath), nil
}

// FetchFromUpstream 업스트림에서 데이터 가져오기 (모든 레포지토리 시도) - Connection Pool 사용
func (h *MavenHandlerV3) FetchFromUpstream(c *fiber.Ctx, _ string) ([]byte, int, error) {
	artifactPath := c.Params("*")
	var lastErr error
	ctx := context.Background()

	// Maven 전용 HTTP 클라이언트는 ExecuteProxyRequest에서 내부적으로 사용됨

	for i, repo := range h.Config.Proxies {
		if repo.URL == "" {
			continue
		}

		baseURL := strings.TrimRight(repo.URL, "/")
		cleanPath := strings.TrimLeft(artifactPath, "/")
		upstreamURL := fmt.Sprintf("%s/%s", baseURL, cleanPath)

		h.GetLogger().Debug("Trying repository with Connection Pool",
			logging.F("repo_index", i),
			logging.F("repo_name", repo.Name),
			logging.F("url", upstreamURL),
		)

		// HTTP 요청 생성
		req, err := http.NewRequestWithContext(ctx, "GET", upstreamURL, nil)
		if err != nil {
			lastErr = fmt.Errorf("요청 생성 실패: %w", err)
			continue
		}

		// 헤더 설정
		req.Header.Set("User-Agent", "ProxyND/1.0 Maven-Proxy")
		req.Header.Set("X-Maven-Proxy", "ProxyND")

		// 레포지토리 인증이 있는 경우
		if repo.BasicAuth.Username != "" && repo.BasicAuth.Password != "" {
			req.SetBasicAuth(repo.BasicAuth.Username, repo.BasicAuth.Password)
		}

		// Connection Pool을 통한 요청 실행
		resp, err := h.clientFactory.ExecuteProxyRequest("maven", req)
		if err != nil {
			lastErr = err
			h.GetLogger().Error("Failed to fetch from repository",
				logging.F("repo_index", i),
				logging.F("repo_name", repo.Name),
				logging.F("error", lastErr),
			)
			continue
		}

		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("upstream returned status %d", resp.StatusCode)
			h.GetLogger().Error("Non-200 status from repository",
				logging.F("repo_index", i),
				logging.F("repo_name", repo.Name),
				logging.F("status_code", resp.StatusCode),
			)
			continue
		}

		// 응답 바디 읽기
		body := make([]byte, 0, 1024*1024) // 1MB 버퍼
		buffer := make([]byte, 8192)
		for {
			n, err := resp.Body.Read(buffer)
			if n > 0 {
				body = append(body, buffer[:n]...)
			}
			if err != nil {
				break
			}
		}

		// 체크섬 검증 (체크섬 파일인 경우 제외)
		if h.Config.UseCache && !h.isChecksumFile(artifactPath) {
			if err := h.validateChecksumWithPool(repo, upstreamURL, body); err != nil {
				h.GetLogger().Error("Checksum validation failed",
					logging.F("repo_name", repo.Name),
					logging.F("error", err),
				)
				lastErr = err
				continue
			}
		}

		h.GetLogger().Info("Successfully fetched from repository",
			logging.F("repo_name", repo.Name),
			logging.F("size_bytes", len(body)),
		)

		return body, resp.StatusCode, nil
	}

	if lastErr != nil {
		return nil, 0, lastErr
	}

	return nil, 0, fmt.Errorf("artifact not found in any repository")
}

// ProcessResponse 응답 처리 (Maven은 일반적으로 원본 그대로 반환)
func (h *MavenHandlerV3) ProcessResponse(c *fiber.Ctx, body []byte, statusCode int) ([]byte, error) {
	// Maven은 특별한 응답 처리가 필요하지 않음
	return body, nil
}

// ShouldCache 캐시 정책 결정
func (h *MavenHandlerV3) ShouldCache(c *fiber.Ctx, statusCode int) bool {
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
	if h.isChecksumFile(path) {
		return true
	}

	return false
}

// GetCacheTTL 캐시 TTL 반환
func (h *MavenHandlerV3) GetCacheTTL(c *fiber.Ctx) time.Duration {
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
	if h.isChecksumFile(path) {
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

// GetContentType Content-Type 결정
func (h *MavenHandlerV3) GetContentType(path string) string {
	switch {
	case strings.HasSuffix(path, ".jar"):
		return "application/java-archive"
	case strings.HasSuffix(path, ".war"):
		return "application/java-archive"
	case strings.HasSuffix(path, ".ear"):
		return "application/java-archive"
	case strings.HasSuffix(path, ".pom"):
		return "application/xml"
	case strings.HasSuffix(path, ".xml"):
		return "application/xml"
	case strings.HasSuffix(path, ".sha1"):
		return "text/plain"
	case strings.HasSuffix(path, ".sha256"):
		return "text/plain"
	case strings.HasSuffix(path, ".md5"):
		return "text/plain"
	case strings.HasSuffix(path, ".asc"):
		return "application/pgp-signature"
	case strings.HasSuffix(path, ".zip"):
		return "application/zip"
	case strings.HasSuffix(path, ".tar.gz"):
		return "application/gzip"
	default:
		return "application/octet-stream"
	}
}

// ShouldInline 인라인 표시 여부 결정
func (h *MavenHandlerV3) ShouldInline(path string) bool {
	return strings.HasSuffix(path, ".pom") ||
		strings.HasSuffix(path, ".xml") ||
		strings.HasSuffix(path, ".sha1") ||
		strings.HasSuffix(path, ".sha256") ||
		strings.HasSuffix(path, ".md5")
}

// HandleError 에러 처리
func (h *MavenHandlerV3) HandleError(err error, c *fiber.Ctx) error {
	// 이미 DomainError인 경우 그대로 전송
	if _, ok := err.(*errors.DomainError); ok {
		return errors.SendProxyError(c, err)
	}

	// 에러 메시지 기반 도메인 에러 변환
	var domainErr *errors.DomainError
	errorMsg := err.Error()

	switch {
	case strings.Contains(errorMsg, "레포지토리가 설정되지 않았습니다") || strings.Contains(errorMsg, "repository"):
		domainErr = errors.WrapMavenError(err, "MAVEN002", "Maven 레포지토리 서버에 접근할 수 없습니다")
	case strings.Contains(errorMsg, "checksum") || strings.Contains(errorMsg, "체크섬"):
		domainErr = errors.WrapMavenError(err, "MAVEN003", "체크섬 검증에 실패했습니다")
	case strings.Contains(errorMsg, "비활성화") || strings.Contains(errorMsg, "disabled"):
		domainErr = errors.WrapMavenError(err, "MAVEN004", "Maven 프록시가 비활성화되어 있습니다")
	case strings.Contains(errorMsg, "path") || strings.Contains(errorMsg, "Invalid path"):
		domainErr = errors.WrapMavenError(err, "MAVEN005", "잘못된 아티팩트 경로입니다")
	case strings.Contains(errorMsg, "설정") || strings.Contains(errorMsg, "config"):
		domainErr = errors.WrapMavenError(err, "MAVEN006", "Maven 설정 파일을 읽을 수 없습니다")
	default:
		// 기본 Maven 에러
		domainErr = errors.WrapMavenError(err, "MAVEN001", "Maven 아티팩트를 찾을 수 없습니다")
	}

	return errors.SendProxyError(c, domainErr)
}

// Handle 메인 핸들러 - Base Handler의 Template Method 사용
func (h *MavenHandlerV3) Handle(c *fiber.Ctx) error {
	return h.BaseProxyHandler.Handle(h, c)
}

// 헬퍼 메서드들

// isChecksumFile 체크섬 파일인지 확인
func (h *MavenHandlerV3) isChecksumFile(path string) bool {
	return strings.HasSuffix(path, ".sha1") ||
		strings.HasSuffix(path, ".sha256") ||
		strings.HasSuffix(path, ".md5")
}

// validateChecksumWithPool Connection Pool을 사용한 체크섬 검증
func (h *MavenHandlerV3) validateChecksumWithPool(repo config.MavenProxyServer, upstreamURL string, body []byte) error {
	// 체크섬 파일 다운로드 시도
	checksumURL := upstreamURL + ".sha1"

	req, err := http.NewRequestWithContext(context.Background(), "GET", checksumURL, nil)
	if err != nil {
		h.GetLogger().Warn("Failed to create checksum request",
			logging.F("checksum_url", checksumURL),
			logging.F("error", err))
		return nil
	}

	// 인증 설정
	if repo.BasicAuth.Username != "" && repo.BasicAuth.Password != "" {
		req.SetBasicAuth(repo.BasicAuth.Username, repo.BasicAuth.Password)
	}

	// Connection Pool을 통한 체크섬 파일 요청
	client := h.clientFactory.GetClientForProxy("maven")
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		// 체크섬 파일이 없는 경우는 경고만 출력하고 계속 진행
		h.GetLogger().Warn("Checksum file not available",
			logging.F("checksum_url", checksumURL),
			logging.F("status_code", func() int {
				if resp != nil {
					return resp.StatusCode
				}
				return 0
			}()),
			logging.F("error", err))
		if resp != nil {
			resp.Body.Close()
		}
		return nil
	}
	defer resp.Body.Close()

	// 체크섬 파일 내용 읽기
	checksumBody := make([]byte, 0, 64) // SHA1은 40자 + 여유
	buffer := make([]byte, 64)
	for {
		n, err := resp.Body.Read(buffer)
		if n > 0 {
			checksumBody = append(checksumBody, buffer[:n]...)
		}
		if err != nil {
			break
		}
	}

	expectedChecksum := strings.TrimSpace(string(checksumBody))

	// SHA1 계산
	hash := sha1.New()
	hash.Write(body)
	actualChecksum := hex.EncodeToString(hash.Sum(nil))

	if expectedChecksum != actualChecksum {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedChecksum, actualChecksum)
	}

	h.GetLogger().Debug("Checksum validation successful",
		logging.F("expected", expectedChecksum),
		logging.F("actual", actualChecksum))

	return nil
}

// extractRequestPath Base Handler에서 사용하는 메서드 (Maven은 기본 구현 사용)
func (h *MavenHandlerV3) extractRequestPath(c *fiber.Ctx) string {
	return c.Params("*")
}
