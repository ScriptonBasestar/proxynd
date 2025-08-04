package http

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"

	"proxynd/helpers"
	"proxynd/internal/config"
	"proxynd/internal/domain/apt"
	aptServices "proxynd/internal/services/apt"
	"proxynd/logging"
)

// APTHandlerAdapter Fiber HTTP 요청을 APT 도메인 서비스로 연결하는 어댑터
type APTHandlerAdapter struct {
	packageHandler apt.PackageHandler
	logger         logging.Logger
}

// NewAPTHandlerAdapter 새로운 APT 핸들러 어댑터 생성
func NewAPTHandlerAdapter(config config.AptProxyConfig, logger logging.Logger) *APTHandlerAdapter {
	// 설정을 도메인 인터페이스로 래핑
	storageDir := helpers.GetStorageDir()
	proxyConfig := apt.NewDefaultProxyConfig(&config, storageDir)

	// 서비스 의존성 생성
	mirrorManager := aptServices.NewMirrorManager(proxyConfig, logger)
	cacheManager := aptServices.NewCacheManager(proxyConfig, logger)
	contentResolver := aptServices.NewContentResolver()
	metricsCollector := aptServices.NewMetricsCollector(logger) // 간단한 메트릭 수집기

	// 패키지 핸들러 생성
	packageHandler := aptServices.NewPackageService(
		proxyConfig,
		logger,
		mirrorManager,
		cacheManager,
		contentResolver,
		metricsCollector,
	)

	return &APTHandlerAdapter{
		packageHandler: packageHandler,
		logger:         logger,
	}
}

// Handle Fiber HTTP 요청을 처리하여 도메인 서비스로 전달
func (a *APTHandlerAdapter) Handle(c *fiber.Ctx) error {
	startTime := time.Now()
	ctx := c.Context()

	// Fiber Context를 도메인 요청으로 변환
	request := a.fiberToDomainRequest(c)

	a.logger.Info("APT request via adapter",
		logging.F("osType", request.OSType),
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
			"error": "Failed to process APT request",
		})
	}

	// 도메인 응답을 Fiber 응답으로 변환
	return a.domainToFiberResponse(c, response)
}

// Type 프록시 타입 반환
func (a *APTHandlerAdapter) Type() string {
	return "apt"
}

// fiberToDomainRequest Fiber 요청을 도메인 요청으로 변환
func (a *APTHandlerAdapter) fiberToDomainRequest(c *fiber.Ctx) *apt.PackageRequest {
	// 헤더 추출
	headers := make(map[string]string)
	c.Request().Header.VisitAll(func(key, value []byte) {
		headers[string(key)] = string(value)
	})

	return &apt.PackageRequest{
		OSType:      c.Params("osType", "ubuntu"),
		PackagePath: c.Params("*"),
		Headers:     headers,
		Method:      c.Method(),
	}
}

// domainToFiberResponse 도메인 응답을 Fiber 응답으로 변환
func (a *APTHandlerAdapter) domainToFiberResponse(c *fiber.Ctx, response *apt.PackageResponse) error {
	// 상태 코드 설정
	c.Status(response.StatusCode)

	// 헤더 설정
	for key, value := range response.Headers {
		c.Set(key, value)
	}

	// 캐시된 파일인 경우 파일로 직접 응답
	if response.FromCache && len(response.Data) == 0 {
		// 캐시된 파일 경로에서 파일 전송
		// 실제 구현에서는 캐시 매니저에서 파일 경로를 받아와야 함
		return a.serveCachedFile(c, response)
	}

	// 메모리의 데이터로 응답
	return c.Send(response.Data)
}

// serveCachedFile 캐시된 파일을 직접 전송
func (a *APTHandlerAdapter) serveCachedFile(c *fiber.Ctx, response *apt.PackageResponse) error {
	// 실제 구현에서는 캐시 매니저로부터 파일 경로를 받아와야 함
	// 현재는 응답 데이터가 있다고 가정하고 처리
	if len(response.Data) > 0 {
		return c.Send(response.Data)
	}

	// 파일이 없는 경우 404 반환
	return c.Status(fiber.StatusNotFound).SendString("Cached file not found")
}

// IsEnabled 핸들러 활성화 상태 확인
func (a *APTHandlerAdapter) IsEnabled() bool {
	// 도메인 서비스를 통해 확인하거나, 설정을 직접 확인
	// 현재는 패키지 핸들러가 있으면 활성화된 것으로 간주
	return a.packageHandler != nil
}

// GenerateCacheKey 캐시 키 생성 (호환성을 위한 메서드)
func (a *APTHandlerAdapter) GenerateCacheKey(c *fiber.Ctx) string {
	osType := c.Params("osType", "ubuntu")
	packagePath := c.Params("*")

	// 도메인 서비스의 키 생성 로직을 사용해야 하지만,
	// 현재 구조에서는 직접 생성
	return fmt.Sprintf("%s_%s", osType, packagePath)
}

// BuildUpstreamURL 업스트림 URL 구성 (호환성을 위한 메서드)
func (a *APTHandlerAdapter) BuildUpstreamURL(c *fiber.Ctx) (string, error) {
	// 이 메서드는 레거시 호환성을 위해 유지하지만,
	// 실제 로직은 도메인 서비스 내부에서 처리됨
	osType := c.Params("osType", "ubuntu")
	packagePath := c.Params("*")

	return fmt.Sprintf("/%s/%s", osType, packagePath), nil
}

// ShouldCache 캐시 여부 결정 (호환성을 위한 메서드)
func (a *APTHandlerAdapter) ShouldCache(c *fiber.Ctx, statusCode int) bool {
	// 성공적인 응답만 캐시
	return statusCode == fiber.StatusOK
}

// GetCacheTTL 캐시 TTL 반환 (호환성을 위한 메서드)
func (a *APTHandlerAdapter) GetCacheTTL(c *fiber.Ctx) time.Duration {
	// 기본 24시간 캐시
	return 24 * time.Hour
}

// HandleError 에러 처리 (호환성을 위한 메서드)
func (a *APTHandlerAdapter) HandleError(err error, c *fiber.Ctx) error {
	a.logger.Error("APT handler error", logging.F("error", err))
	return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
		"error": err.Error(),
	})
}

// RecordRequestMetrics 요청 메트릭 기록 (호환성을 위한 메서드)
func (a *APTHandlerAdapter) RecordRequestMetrics(c *fiber.Ctx, statusCode int, duration time.Duration) {
	// 메트릭은 도메인 서비스 내부에서 처리됨
	a.logger.Debug("APT request completed",
		logging.F("statusCode", statusCode),
		logging.F("duration_ms", duration.Milliseconds()),
		logging.F("path", c.Path()),
	)
}

// 추가 헬퍼 메서드들

// getContentType APT 패키지 경로에서 Content-Type 결정 (호환성용)
func getAptContentType(packagePath string) string {
	resolver := aptServices.NewContentResolver()
	return resolver.GetContentType(packagePath)
}

// shouldInline 인라인 표시 여부 결정 (호환성용)
func shouldInline(packagePath string) bool {
	resolver := aptServices.NewContentResolver()
	return resolver.ShouldInline(packagePath)
}

// copyToFile 스트림을 파일로 복사 (유틸리티 함수)
func copyToFile(dst string, src io.Reader) error {
	file, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()

	_, err = io.Copy(file, src)
	return err
}
