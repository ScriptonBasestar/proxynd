package http

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"proxynd/internal/config"
	"proxynd/internal/domain/docker"
	"proxynd/internal/helpers"
	"proxynd/internal/logging"
	dockerServices "proxynd/internal/services/docker"
)

const (
	dockerTagsEndpoint  = "tags"
	dockerListEndpoint  = "list"
	mimeApplicationJSON = "application/json"
)

// DockerHandlerAdapter Fiber HTTP 요청을 Docker 도메인 서비스로 연결하는 어댑터
type DockerHandlerAdapter struct {
	registryHandler docker.RegistryHandler
	logger          logging.Logger
	config          docker.ProxyConfig
	pathPattern     *regexp.Regexp
}

// NewDockerHandlerAdapter 새로운 Docker 핸들러 어댑터 생성
func NewDockerHandlerAdapter(config config.DockerProxySettings, logger logging.Logger) *DockerHandlerAdapter {
	// 설정을 도메인 인터페이스로 래핑
	storageDir := helpers.GetStorageDir()
	proxyConfig := docker.NewDefaultProxyConfig(&config, storageDir)

	// 설정된 레지스트리 수 로그
	registries := proxyConfig.GetRegistries()
	logger.Info("Docker handler adapter created",
		logging.F("registries_count", len(registries)),
		logging.F("use_cache", config.UseCache))

	// 서비스 팩토리를 통한 의존성 생성
	serviceFactory := dockerServices.NewServiceFactory(proxyConfig, logger)
	registryHandler := serviceFactory.CreateRegistryService()

	// Docker Registry v2 API 경로 패턴
	pathPattern := regexp.MustCompile(`^v2(/.*)?$`)

	return &DockerHandlerAdapter{
		registryHandler: registryHandler,
		logger:          logger,
		config:          proxyConfig,
		pathPattern:     pathPattern,
	}
}

// Handle Fiber HTTP 요청을 처리하는 메인 핸들러
func (a *DockerHandlerAdapter) Handle(c *fiber.Ctx) error {
	requestPath := c.Params("*")
	method := c.Method()

	a.logger.Debug("Docker proxy request",
		logging.F("method", method),
		logging.F("path", requestPath))

	// Docker Registry v2 API 기본 엔드포인트 처리
	if requestPath == "v2" || requestPath == "v2/" {
		return a.handleV2Base(c)
	}

	// 경로 파싱 및 요청 객체 생성
	request, err := a.parseDockerRequest(c, requestPath)
	if err != nil {
		a.logger.Error("Failed to parse Docker request",
			logging.F("error", err),
			logging.F("path", requestPath))
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("Invalid Docker request: %v", err),
		})
	}

	// 컨텍스트 생성 (타임아웃 설정)
	ctx, cancel := context.WithTimeout(c.Context(), 60*time.Second)
	defer cancel()

	// 서비스 레이어로 요청 전달
	response, err := a.registryHandler.Handle(ctx, request)
	if err != nil {
		a.logger.Error("Docker registry service error",
			logging.F("error", err),
			logging.F("error_msg", err.Error()),
			logging.F("error_type", fmt.Sprintf("%T", err)),
			logging.F("operation", request.Operation))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Internal server error: %v", err),
		})
	}

	// 응답 처리 및 반환
	return a.processResponse(c, response)
}

// handleV2Base Docker Registry v2 기본 엔드포인트 처리 (/v2/)
func (a *DockerHandlerAdapter) handleV2Base(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.Context(), 10*time.Second)
	defer cancel()

	response, err := a.registryHandler.HandleV2Base(ctx)
	if err != nil {
		a.logger.Error("Failed to handle v2 base endpoint", logging.F("error", err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to handle registry base endpoint",
		})
	}

	return a.processResponse(c, response)
}

// parseDockerRequest Fiber 요청을 Docker 도메인 요청으로 변환
func (a *DockerHandlerAdapter) parseDockerRequest(c *fiber.Ctx, requestPath string) (*docker.RegistryRequest, error) {
	// 경로에서 레포지토리, 참조, 작업 타입 추출
	repository, reference, operation := a.extractPathComponents(requestPath)

	// HTTP 헤더 변환
	headers := make(map[string]string)
	for key, value := range c.Request().Header.All() {
		headers[string(key)] = string(value)
	}

	// 클라이언트 IP 추출
	clientIP := c.IP()

	// 프록시 서버 기본 URL 구성
	baseURL := fmt.Sprintf("%s://%s", c.Protocol(), c.Hostname())
	if port := c.Port(); port != "" {
		baseURL += ":" + port
	}

	request := &docker.RegistryRequest{
		Path:       "/" + requestPath,
		Method:     c.Method(),
		Headers:    headers,
		Repository: repository,
		Reference:  reference,
		Operation:  operation,
		BaseURL:    baseURL,
		ClientIP:   clientIP,
	}

	return request, nil
}

// extractPathComponents 경로에서 레포지토리, 참조, 작업 정보 추출
func (a *DockerHandlerAdapter) extractPathComponents(requestPath string) (repository, reference, operation string) {
	// v2 prefix 제거
	path := strings.TrimPrefix(requestPath, "v2/")

	// 카탈로그 요청 확인
	if path == "_catalog" {
		return "", "", "catalog"
	}

	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		return "", "", "unknown"
	}

	// 매니페스트 요청: /v2/<name>/manifests/<reference>
	if len(parts) >= 3 && parts[len(parts)-2] == "manifests" {
		repository = strings.Join(parts[:len(parts)-2], "/")
		reference = parts[len(parts)-1]
		operation = "manifest"
		return repository, reference, operation
	}

	// Blob 요청: /v2/<name>/blobs/<digest>
	if len(parts) >= 3 && parts[len(parts)-2] == "blobs" {
		repository = strings.Join(parts[:len(parts)-2], "/")
		reference = parts[len(parts)-1]
		operation = "blob"
		return repository, reference, operation
	}

	// 태그 목록 요청: /v2/<name>/tags/list
	if len(parts) >= 3 && parts[len(parts)-2] == dockerTagsEndpoint && parts[len(parts)-1] == dockerListEndpoint {
		repository = strings.Join(parts[:len(parts)-2], "/")
		reference = ""
		operation = dockerTagsEndpoint
		return repository, reference, operation
	}

	// 기본값
	repository = strings.Join(parts, "/")
	reference = ""
	operation = "unknown"
	return repository, reference, operation
}

// processResponse Docker 서비스 응답을 Fiber 응답으로 변환
func (a *DockerHandlerAdapter) processResponse(c *fiber.Ctx, response *docker.ManifestResponse) error {
	// HTTP 상태 코드 설정
	c.Status(response.StatusCode)

	// 응답 헤더 설정
	for key, value := range response.Headers {
		c.Set(key, value)
	}

	// Content-Type 설정 (Docker 매니페스트/blob별 구분)
	if response.ContentType != "" {
		c.Set("Content-Type", response.ContentType)
	}

	// Docker 전용 헤더 설정
	if response.Digest != "" {
		c.Set("Docker-Content-Digest", response.Digest)
	}

	// Docker Distribution API 버전 헤더
	c.Set("Docker-Distribution-Api-Version", "registry/2.0")

	// 캐시 관련 헤더 (캐시에서 온 경우)
	if response.FromCache {
		c.Set("X-Cache", "HIT")
	} else {
		c.Set("X-Cache", "MISS")
	}

	// 사용된 레지스트리 정보 (디버깅용)
	if response.RegistryUsed != "" {
		c.Set("X-Registry-Used", response.RegistryUsed)
	}

	// 응답 본문 전송
	if len(response.Data) > 0 {
		return c.Send(response.Data)
	}

	return c.SendStatus(response.StatusCode)
}

// SupportedOperations 지원하는 Docker 작업 목록 반환
func (a *DockerHandlerAdapter) SupportedOperations() []string {
	return a.registryHandler.GetSupportedOperations()
}

// IsDockerRequest 요청이 Docker Registry 요청인지 확인
func (a *DockerHandlerAdapter) IsDockerRequest(requestPath string) bool {
	return a.pathPattern.MatchString(requestPath)
}

// GetDockerContentType 요청 타입에 따른 적절한 Content-Type 반환
func (a *DockerHandlerAdapter) GetDockerContentType(operation, reference string) string {
	switch operation {
	case "manifest":
		// 매니페스트 타입별 Content-Type 구분
		if strings.HasPrefix(reference, "sha256:") {
			return "application/vnd.docker.distribution.manifest.v2+json"
		}
		// 태그 기반 매니페스트는 다양한 타입 지원
		return "application/vnd.docker.distribution.manifest.v2+json"
	case "blob":
		return "application/octet-stream"
	case dockerTagsEndpoint:
		return mimeApplicationJSON
	case "catalog":
		return mimeApplicationJSON
	default:
		return mimeApplicationJSON
	}
}

// ValidateDockerPath Docker Registry 경로 유효성 검증
func (a *DockerHandlerAdapter) ValidateDockerPath(requestPath string) error {
	// v2 API 경로 검증
	if !strings.HasPrefix(requestPath, "v2") {
		return fmt.Errorf("invalid Docker registry path: must start with v2")
	}

	// 지원하는 엔드포인트 패턴 검증
	supportedPatterns := []string{
		`^v2/?$`,                            // /v2/
		`^v2/_catalog$`,                     // /v2/_catalog
		`^v2/.+/manifests/.+$`,              // /v2/<name>/manifests/<reference>
		`^v2/.+/blobs/sha256:[a-f0-9]{64}$`, // /v2/<name>/blobs/<digest>
		`^v2/.+/tags/list$`,                 // /v2/<name>/tags/list
	}

	for _, pattern := range supportedPatterns {
		matched, _ := regexp.MatchString(pattern, requestPath)
		if matched {
			return nil
		}
	}

	return fmt.Errorf("unsupported Docker registry endpoint: %s", requestPath)
}

// extractRepositoryFromPath 경로에서 레포지토리명 추출
//
//nolint:unused // Docker 레지스트리 v2 API 지원을 위해 유지
func (a *DockerHandlerAdapter) extractRepositoryFromPath(requestPath string) string {
	// v2 prefix 제거
	path := strings.TrimPrefix(requestPath, "v2/")

	// 특수 엔드포인트 처리
	if path == "_catalog" {
		return ""
	}

	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		return ""
	}

	// manifests, blobs, tags 앞까지가 레포지토리명
	for i, part := range parts {
		if part == "manifests" || part == "blobs" || part == dockerTagsEndpoint {
			return strings.Join(parts[:i], "/")
		}
	}

	return strings.Join(parts, "/")
}
