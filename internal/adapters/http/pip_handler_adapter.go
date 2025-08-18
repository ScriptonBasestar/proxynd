package http

import (
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"proxynd/helpers"
	"proxynd/internal/config"
	"proxynd/internal/domain/pip"
	"proxynd/internal/logging"
	pipServices "proxynd/internal/services/pip"
)

// PIPHandlerAdapter Fiber HTTP 요청을 PIP 도메인 서비스로 연결하는 어댑터
type PIPHandlerAdapter struct {
	packageHandler pip.PackageHandler
	logger         logging.Logger
	config         pip.ProxyConfig
}

// NewPIPHandlerAdapter 새로운 PIP 핸들러 어댑터 생성
func NewPIPHandlerAdapter(config config.PipProxySettings, logger logging.Logger) *PIPHandlerAdapter {
	// 설정을 도메인 인터페이스로 래핑
	storageDir := helpers.GetStorageDir()
	proxyConfig := pip.NewDefaultProxyConfig(&config, storageDir)

	// 서비스 의존성 생성
	indexManager := pipServices.NewIndexManager(proxyConfig, logger)
	cacheManager := pipServices.NewCacheManager(proxyConfig, logger)
	metadataProcessor := pipServices.NewMetadataProcessor(proxyConfig, logger)
	metricsCollector := pipServices.NewMetricsCollector(proxyConfig, logger)

	// 패키지 핸들러 생성
	packageHandler := pipServices.NewPackageService(
		proxyConfig,
		logger,
		indexManager,
		cacheManager,
		metadataProcessor,
		metricsCollector,
	)

	return &PIPHandlerAdapter{
		packageHandler: packageHandler,
		logger:         logger,
		config:         proxyConfig,
	}
}

// Handle Fiber HTTP 요청을 처리하여 도메인 서비스로 전달
func (a *PIPHandlerAdapter) Handle(c *fiber.Ctx) error {
	startTime := time.Now()
	ctx := c.Context()

	// Fiber Context를 도메인 요청으로 변환
	request := a.fiberToDomainRequest(c)

	a.logger.Info("PIP request via adapter",
		logging.F("packagePath", request.PackagePath),
		logging.F("method", request.Method),
		logging.F("remoteIP", c.IP()),
		logging.F("userAgent", c.Get("User-Agent")),
	)

	// 도메인 서비스로 요청 전달
	response, err := a.packageHandler.Handle(ctx, request)
	if err != nil {
		a.logger.Error("Package handler failed",
			logging.F("error", err),
			logging.F("duration_ms", time.Since(startTime).Milliseconds()),
		)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to process PIP request",
		})
	}

	// 도메인 응답을 Fiber 응답으로 변환
	return a.domainToFiberResponse(c, response)
}

// Type 프록시 타입 반환
func (a *PIPHandlerAdapter) Type() string {
	return "pip"
}

// fiberToDomainRequest Fiber 요청을 도메인 요청으로 변환
func (a *PIPHandlerAdapter) fiberToDomainRequest(c *fiber.Ctx) *pip.PackageRequest {
	// 헤더 추출
	headers := make(map[string]string)
	c.Request().Header.VisitAll(func(key, value []byte) {
		headers[string(key)] = string(value)
	})

	// 쿼리 파라미터 추출
	queryParams := make(map[string]string)
	c.Request().URI().QueryArgs().VisitAll(func(key, value []byte) {
		queryParams[string(key)] = string(value)
	})

	return &pip.PackageRequest{
		PackagePath: c.Params("*"),
		Headers:     headers,
		Method:      c.Method(),
		BaseURL:     fmt.Sprintf("%s://%s", c.Protocol(), c.Hostname()),
		QueryParams: queryParams,
	}
}

// domainToFiberResponse 도메인 응답을 Fiber 응답으로 변환
func (a *PIPHandlerAdapter) domainToFiberResponse(c *fiber.Ctx, response *pip.PackageResponse) error {
	// 상태 코드 설정
	c.Status(response.StatusCode)

	// 헤더 설정
	for key, value := range response.Headers {
		c.Set(key, value)
	}

	// Content-Type 설정
	if response.ContentType != "" {
		c.Set("Content-Type", response.ContentType)
	}

	// Content-Disposition 설정
	fileName := a.extractFileName(response)
	disposition := a.getContentDisposition(response, fileName)
	if disposition != "" {
		c.Set("Content-Disposition", disposition)
	}

	// 캐시된 파일인 경우 파일로 직접 응답
	if response.FromCache && len(response.Data) == 0 {
		// 캐시된 파일 경로에서 파일 전송
		return a.serveCachedFile(c, response)
	}

	// 메모리의 데이터로 응답
	return c.Send(response.Data)
}

// serveCachedFile 캐시된 파일을 직접 전송
func (a *PIPHandlerAdapter) serveCachedFile(c *fiber.Ctx, response *pip.PackageResponse) error {
	// 실제 구현에서는 캐시 매니저로부터 파일 경로를 받아와야 함
	// 현재는 응답 데이터가 있다고 가정하고 처리
	if len(response.Data) > 0 {
		return c.Send(response.Data)
	}

	// 파일이 없는 경우 404 반환
	return c.Status(fiber.StatusNotFound).SendString("Cached file not found")
}

// extractFileName 응답에서 파일명 추출
func (a *PIPHandlerAdapter) extractFileName(response *pip.PackageResponse) string {
	// 헤더에서 파일명 찾기
	if contentDisp, exists := response.Headers["Content-Disposition"]; exists {
		if strings.Contains(contentDisp, "filename=") {
			parts := strings.Split(contentDisp, "filename=")
			if len(parts) > 1 {
				return strings.Trim(parts[1], "\"")
			}
		}
	}

	// 기본 파일명
	if response.IsPackageFile {
		return "package"
	} else if response.IsSimpleAPI {
		return "index"
	}

	return "file"
}

// getContentDisposition Content-Disposition 헤더 생성
func (a *PIPHandlerAdapter) getContentDisposition(response *pip.PackageResponse, fileName string) string {
	// Simple API나 메타데이터 응답은 inline
	if response.IsSimpleAPI || strings.Contains(response.ContentType, "json") || strings.Contains(response.ContentType, "html") { //nolint:lll
		return fmt.Sprintf("inline; filename=%s", fileName)
	}

	// 패키지 파일은 attachment
	if response.IsPackageFile {
		return fmt.Sprintf("attachment; filename=%s", fileName)
	}

	// 기본값
	return fmt.Sprintf("inline; filename=%s", fileName)
}

// IsEnabled 핸들러 활성화 상태 확인
func (a *PIPHandlerAdapter) IsEnabled() bool {
	// 도메인 서비스를 통해 확인
	return a.packageHandler != nil && a.config.IsEnabled()
}

// GenerateCacheKey 캐시 키 생성 (호환성을 위한 메서드)
func (a *PIPHandlerAdapter) GenerateCacheKey(c *fiber.Ctx) string {
	packagePath := c.Params("*")

	// 도메인 서비스의 키 생성 로직을 사용해야 하지만,
	// 현재 구조에서는 직접 생성
	return fmt.Sprintf("pip_%s", strings.ReplaceAll(packagePath, "/", "_"))
}

// BuildUpstreamURL 업스트림 URL 구성 (호환성을 위한 메서드)
func (a *PIPHandlerAdapter) BuildUpstreamURL(c *fiber.Ctx) (string, error) {
	// 이 메서드는 레거시 호환성을 위해 유지하지만,
	// 실제 로직은 도메인 서비스 내부에서 처리됨
	packagePath := c.Params("*")

	// PyPI URL 패턴 처리
	if strings.HasPrefix(packagePath, "simple/") {
		return fmt.Sprintf("/simple/%s", strings.TrimPrefix(packagePath, "simple/")), nil
	}

	if strings.HasPrefix(packagePath, "packages/") {
		return fmt.Sprintf("/%s", packagePath), nil
	}

	// 기본적으로 simple API 사용
	return fmt.Sprintf("/simple/%s", packagePath), nil
}

// ShouldCache 캐시 여부 결정 (호환성을 위한 메서드)
func (a *PIPHandlerAdapter) ShouldCache(c *fiber.Ctx, statusCode int) bool {
	// 성공적인 응답만 캐시
	return statusCode == fiber.StatusOK && a.config.IsCacheEnabled()
}

// GetCacheTTL 캐시 TTL 반환 (호환성을 위한 메서드)
func (a *PIPHandlerAdapter) GetCacheTTL(c *fiber.Ctx) time.Duration {
	// 도메인 설정에서 TTL 가져오기
	cacheConfig := a.config.GetCacheConfig()
	return cacheConfig.TTL
}

// HandleError 에러 처리 (호환성을 위한 메서드)
func (a *PIPHandlerAdapter) HandleError(err error, c *fiber.Ctx) error {
	a.logger.Error("PIP handler error", logging.F("error", err))
	return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
		"error": err.Error(),
	})
}

// RecordRequestMetrics 요청 메트릭 기록 (호환성을 위한 메서드)
func (a *PIPHandlerAdapter) RecordRequestMetrics(c *fiber.Ctx, statusCode int, duration time.Duration) {
	// 메트릭은 도메인 서비스 내부에서 처리됨
	a.logger.Debug("PIP request completed",
		logging.F("statusCode", statusCode),
		logging.F("duration_ms", duration.Milliseconds()),
		logging.F("path", c.Path()),
		logging.F("userAgent", c.Get("User-Agent")),
	)
}

// 추가 헬퍼 메서드들
