package http

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"proxynd/internal/config"
	"proxynd/internal/domain/npm"
	"proxynd/internal/helpers"
	"proxynd/internal/logging"
	npmServices "proxynd/internal/services/npm"
)

// NPMHandlerAdapter Fiber HTTP 요청을 NPM 도메인 서비스로 연결하는 어댑터
type NPMHandlerAdapter struct {
	packageHandler npm.PackageHandler
	logger         logging.Logger
}

// NewNPMHandlerAdapter 새로운 NPM 핸들러 어댑터 생성
func NewNPMHandlerAdapter(config config.NpmProxySettings, logger logging.Logger) *NPMHandlerAdapter {
	// 설정을 도메인 인터페이스로 래핑
	storageDir := helpers.GetStorageDir()
	proxyConfig := npm.NewDefaultProxyConfig(&config, storageDir)

	// 서비스 의존성 생성
	proxyManager := npmServices.NewProxyManager(proxyConfig, logger)
	cacheManager := npmServices.NewCacheManager(proxyConfig, logger)
	metadataProcessor := npmServices.NewMetadataProcessor()
	metricsCollector := npmServices.NewMetricsCollector(logger)

	// 패키지 핸들러 생성
	packageHandler := npmServices.NewPackageService(
		proxyConfig,
		logger,
		proxyManager,
		cacheManager,
		metadataProcessor,
		metricsCollector,
	)

	return &NPMHandlerAdapter{
		packageHandler: packageHandler,
		logger:         logger,
	}
}

// Handle Fiber HTTP 요청을 처리하여 도메인 서비스로 전달
func (a *NPMHandlerAdapter) Handle(c *fiber.Ctx) error {
	startTime := time.Now()
	ctx := c.Context()

	// Fiber Context를 도메인 요청으로 변환
	request := a.fiberToDomainRequest(c)

	a.logger.Info("NPM request via adapter",
		logging.F("packagePath", request.PackagePath),
		logging.F("method", request.Method),
		logging.F("remoteIP", c.IP()),
	)

	// 도메인 서비스로 요청 전달
	response, err := a.packageHandler.Handle(ctx, request)
	if err != nil {
		a.logger.Error("Package handler failed",
			logging.F("error", err),
			logging.F("duration_ms", time.Since(startTime).Milliseconds()),
		)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to process NPM request",
		})
	}

	// 도메인 응답을 Fiber 응답으로 변환
	return a.domainToFiberResponse(c, response, request.PackagePath)
}

// Type 프록시 타입 반환
func (a *NPMHandlerAdapter) Type() string {
	return "npm"
}

// fiberToDomainRequest Fiber 요청을 도메인 요청으로 변환
func (a *NPMHandlerAdapter) fiberToDomainRequest(c *fiber.Ctx) *npm.PackageRequest {
	// 헤더 추출
	headers := make(map[string]string)
	for key, value := range c.Request().Header.All() {
		headers[string(key)] = string(value)
	}

	return &npm.PackageRequest{
		PackagePath: c.Params("*"),
		Headers:     headers,
		Method:      c.Method(),
		BaseURL:     c.BaseURL(),
	}
}

// domainToFiberResponse 도메인 응답을 Fiber 응답으로 변환
func (a *NPMHandlerAdapter) domainToFiberResponse(c *fiber.Ctx, response *npm.PackageResponse, packagePath string) error { //nolint:lll
	// 상태 코드 설정
	c.Status(response.StatusCode)

	// 404나 다른 에러 상태 코드인 경우 JSON 형식으로 응답
	if response.StatusCode >= 400 {
		return c.JSON(fiber.Map{
			"error": fmt.Sprintf("Package not found or error occurred (status: %d)", response.StatusCode),
		})
	}

	// Content-Type 설정
	c.Set("Content-Type", response.ContentType)

	// Content-Disposition 설정
	if response.IsMetadata {
		c.Set("Content-Disposition", "inline")
	} else {
		filename := getFilenameFromPath(packagePath)
		c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	}

	// 헤더 설정
	for key, value := range response.Headers {
		c.Set(key, value)
	}

	// 캐시된 파일인 경우 파일로 직접 응답
	if response.FromCache && len(response.Data) == 0 {
		return a.serveCachedFile(c, packagePath)
	}

	// 메모리의 데이터로 응답
	return c.Send(response.Data)
}

// serveCachedFile 캐시된 파일을 직접 전송
func (a *NPMHandlerAdapter) serveCachedFile(c *fiber.Ctx, packagePath string) error {
	// 캐시 매니저를 통해 캐시 파일 경로 생성 (간단화)
	storageDir := helpers.GetStorageDir()
	cacheKey := generateCacheKey(packagePath)
	cachePath := fmt.Sprintf("%s/npm/%s", storageDir, cacheKey)

	// 파일 존재 확인 및 전송
	if data, err := os.ReadFile(cachePath); err == nil {
		return c.Send(data)
	}

	// 파일이 없는 경우 404 반환
	return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
		"error": "Cached file not found",
	})
}

// IsEnabled 핸들러 활성화 상태 확인
func (a *NPMHandlerAdapter) IsEnabled() bool {
	return a.packageHandler != nil
}

// 호환성을 위한 레거시 메서드들

// GenerateCacheKey 캐시 키 생성 (호환성을 위한 메서드)
func (a *NPMHandlerAdapter) GenerateCacheKey(c *fiber.Ctx) string {
	packagePath := c.Params("*")
	return generateCacheKey(packagePath)
}

// BuildUpstreamURL 업스트림 URL 구성 (호환성을 위한 메서드)
func (a *NPMHandlerAdapter) BuildUpstreamURL(c *fiber.Ctx) (string, error) {
	packagePath := c.Params("*")
	return fmt.Sprintf("/%s", packagePath), nil
}

// ShouldCache 캐시 여부 결정 (호환성을 위한 메서드)
func (a *NPMHandlerAdapter) ShouldCache(c *fiber.Ctx, statusCode int) bool {
	return statusCode == fiber.StatusOK
}

// GetCacheTTL 캐시 TTL 반환 (호환성을 위한 메서드)
func (a *NPMHandlerAdapter) GetCacheTTL(c *fiber.Ctx) time.Duration {
	return 6 * time.Hour // NPM 메타데이터는 6시간 캐시
}

// HandleError 에러 처리 (호환성을 위한 메서드)
func (a *NPMHandlerAdapter) HandleError(err error, c *fiber.Ctx) error {
	a.logger.Error("NPM handler error", logging.F("error", err))
	return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
		"error": err.Error(),
	})
}

// RecordRequestMetrics 요청 메트릭 기록 (호환성을 위한 메서드)
func (a *NPMHandlerAdapter) RecordRequestMetrics(c *fiber.Ctx, statusCode int, duration time.Duration) {
	a.logger.Debug("NPM request completed",
		logging.F("statusCode", statusCode),
		logging.F("duration_ms", duration.Milliseconds()),
		logging.F("path", c.Path()),
	)
}

// 유틸리티 함수들

// generateCacheKey 패키지 경로에서 캐시 키 생성
func generateCacheKey(packagePath string) string {
	// 패키지 경로를 안전한 파일명으로 변환
	key := packagePath
	key = fmt.Sprintf("%s_%s", "npm", key)
	key = strings.ReplaceAll(key, "/", "_")
	key = strings.ReplaceAll(key, "@", "at_")
	key = strings.ReplaceAll(key, ":", "_")
	return key
}

// getFilenameFromPath 패키지 경로에서 파일명 추출
func getFilenameFromPath(packagePath string) string {
	if idx := strings.LastIndex(packagePath, "/"); idx != -1 {
		return packagePath[idx+1:]
	}
	return packagePath
}
