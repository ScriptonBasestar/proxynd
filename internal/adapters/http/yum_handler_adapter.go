package http

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"

	"proxynd/helpers"
	"proxynd/internal/config"
	yumDomain "proxynd/internal/domain/yum"
	yumServices "proxynd/internal/services/yum"
	"proxynd/logging"
)

// YumHandlerAdapter YUM HTTP 어댑터
type YumHandlerAdapter struct {
	packageService yumDomain.PackageService
	logger         logging.Logger
	config         yumDomain.ProxyConfig
}

// NewYumHandlerAdapter YUM HTTP 어댑터 생성
func NewYumHandlerAdapter(config config.YumProxySettings, logger logging.Logger) *YumHandlerAdapter {
	storageDir := helpers.GetStorageDir()
	proxyConfig := yumDomain.NewDefaultProxyConfig(&config, storageDir)

	// 서비스 팩토리 패턴으로 의존성 생성
	repoManager := yumServices.NewRepoManager(proxyConfig, logger, storageDir)
	cacheManager := yumServices.NewCacheManager(proxyConfig, logger, storageDir)
	metadataProcessor := yumServices.NewMetadataProcessor(proxyConfig, logger)
	metricsCollector := yumServices.NewMetricsCollector(proxyConfig, logger, storageDir)

	packageService := yumServices.NewPackageService(
		proxyConfig,
		repoManager,
		cacheManager,
		metadataProcessor,
		metricsCollector,
		logger,
		storageDir,
	)

	return &YumHandlerAdapter{
		packageService: packageService,
		logger:         logger,
		config:         proxyConfig,
	}
}

// Handle YUM 프록시 요청 처리
func (a *YumHandlerAdapter) Handle(c *fiber.Ctx) error {
	// 요청 컨텍스트 생성 (60초 타임아웃 - YUM은 큰 파일이 많음)
	ctx, cancel := context.WithTimeout(c.Context(), 60*time.Second)
	defer cancel()

	// 요청 경로 추출
	requestPath := c.Params("*")
	if requestPath == "" {
		a.logger.Warn("YUM 요청 경로가 비어있음")
		return c.Status(fiber.StatusBadRequest).SendString("Request path is required")
	}

	a.logger.Info("YUM 프록시 요청 처리 시작",
		logging.F("path", requestPath),
		logging.F("method", c.Method()),
		logging.F("ip", c.IP()))

	// 도메인 요청 객체 생성
	request := &yumDomain.PackageRequest{
		PackagePath: requestPath,
		Headers:     a.extractHeaders(c),
		Method:      c.Method(),
		BaseURL:     c.BaseURL(),
	}

	// 패키지 서비스로 요청 처리
	response, err := a.packageService.HandleRequest(ctx, request)
	if err != nil {
		a.logger.Error("YUM 패키지 서비스 요청 처리 실패",
			logging.F("path", requestPath),
			logging.F("error", err.Error()))

		return c.Status(fiber.StatusInternalServerError).SendString("Failed to process request")
	}

	// HTTP 응답 설정
	a.setResponseHeaders(c, response)

	// 응답 데이터 전송
	c.Status(response.StatusCode)
	return c.Send(response.Data)
}

// 헬퍼 메서드들

// extractHeaders Fiber 컨텍스트에서 헤더 추출
func (a *YumHandlerAdapter) extractHeaders(c *fiber.Ctx) map[string]string {
	headers := make(map[string]string)

	// 중요한 헤더들만 추출
	importantHeaders := []string{
		"User-Agent",
		"Accept",
		"Accept-Encoding",
		"Accept-Language",
		"Cache-Control",
		"If-Modified-Since",
		"If-None-Match",
	}

	for _, headerName := range importantHeaders {
		if value := c.Get(headerName); value != "" {
			headers[headerName] = value
		}
	}

	return headers
}

// setResponseHeaders 응답 헤더 설정
func (a *YumHandlerAdapter) setResponseHeaders(c *fiber.Ctx, response *yumDomain.PackageResponse) {
	// Content-Type 설정
	if response.ContentType != "" {
		c.Set("Content-Type", response.ContentType)
	}

	// 커스텀 헤더 설정
	for key, value := range response.Headers {
		c.Set(key, value)
	}

	// 캐시 정보 헤더
	if response.FromCache {
		c.Set("X-Cache", "HIT")
		c.Set("X-Cache-Source", "proxynd-yum")
	} else {
		c.Set("X-Cache", "MISS")
		if response.ProxyUsed != "" {
			c.Set("X-Upstream-Server", response.ProxyUsed)
		}
	}

	// 파일 타입별 추가 헤더
	if response.IsRpmFile {
		c.Set("X-Content-Type", "rpm-package")
	} else if response.IsRepoMeta {
		c.Set("X-Content-Type", "repository-metadata")
	}

	// 보안 헤더
	c.Set("X-Content-Type-Options", "nosniff")
	c.Set("X-Frame-Options", "DENY")

	// 캐시 제어 헤더
	if response.IsRepoMeta {
		// 리포지토리 메타데이터는 짧은 캐시 시간
		c.Set("Cache-Control", "public, max-age=1800") // 30분
	} else if response.IsRpmFile {
		// RPM 파일은 긴 캐시 시간
		c.Set("Cache-Control", "public, max-age=86400") // 24시간
	} else {
		// 기타 파일은 기본 캐시 시간
		c.Set("Cache-Control", "public, max-age=3600") // 1시간
	}
}
