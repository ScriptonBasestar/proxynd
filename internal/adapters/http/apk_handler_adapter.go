package http

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"

	"proxynd/configs"
	"proxynd/helpers"
	apkDomain "proxynd/internal/domain/apk"
	apkServices "proxynd/internal/services/apk"
	"proxynd/logging"
)

// ApkHandlerAdapter APK HTTP 어댑터
type ApkHandlerAdapter struct {
	packageService apkDomain.PackageService
	logger         logging.Logger
	config         apkDomain.ProxyConfig
}

// NewApkHandlerAdapter APK HTTP 어댑터 생성
func NewApkHandlerAdapter(config configs.ApkProxyConfig, logger logging.Logger) *ApkHandlerAdapter {
	storageDir := helpers.GetStorageDir()
	proxyConfig := apkDomain.NewDefaultProxyConfig(&config, storageDir)
	
	// 서비스 팩토리 패턴으로 의존성 생성
	repositoryManager := apkServices.NewRepositoryManager(proxyConfig, logger, storageDir)
	cacheManager := apkServices.NewCacheManager(proxyConfig, logger, storageDir)
	signatureVerifier := apkServices.NewSignatureVerifier(proxyConfig, logger)
	metricsCollector := apkServices.NewMetricsCollector(proxyConfig, logger, storageDir)
	
	// 서명 검증기 키 로드
	if proxyConfig.GetVerificationEnabled() {
		if err := signatureVerifier.LoadTrustedKeys(context.Background(), proxyConfig.GetVerificationKeyDirectory()); err != nil {
			logger.Warn("APK 신뢰 키 로드 실패", logging.F("error", err.Error()))
		}
	}
	
	packageService := apkServices.NewPackageService(
		proxyConfig,
		repositoryManager,
		cacheManager,
		signatureVerifier,
		metricsCollector,
		logger,
		storageDir,
	)
	
	return &ApkHandlerAdapter{
		packageService: packageService,
		logger:         logger,
		config:         proxyConfig,
	}
}

// Handle APK 프록시 요청 처리
func (a *ApkHandlerAdapter) Handle(c *fiber.Ctx) error {
	// 요청 컨텍스트 생성 (45초 타임아웃)
	ctx, cancel := context.WithTimeout(c.Context(), 45*time.Second)
	defer cancel()
	
	// 요청 경로 추출
	requestPath := c.Params("*")
	if requestPath == "" {
		a.logger.Warn("APK 요청 경로가 비어있음")
		return c.Status(fiber.StatusBadRequest).SendString("Request path is required")
	}
	
	a.logger.Info("APK 프록시 요청 처리 시작",
		logging.F("path", requestPath),
		logging.F("method", c.Method()),
		logging.F("ip", c.IP()))
	
	// 도메인 요청 객체 생성
	request := &apkDomain.PackageRequest{
		PackagePath: requestPath,
		Headers:     a.extractHeaders(c),
		Method:      c.Method(),
		BaseURL:     c.BaseURL(),
	}
	
	// 패키지 서비스로 요청 처리
	response, err := a.packageService.HandleRequest(ctx, request)
	if err != nil {
		a.logger.Error("APK 패키지 서비스 요청 처리 실패",
			logging.F("path", requestPath),
			logging.F("error", err.Error()))
		
		// 서명 검증 실패인 경우 특별한 처리
		if err.Error() == "signature verification failed" {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error":   "APK signature verification failed",
				"message": "The requested APK file failed signature verification",
			})
		}
		
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
func (a *ApkHandlerAdapter) extractHeaders(c *fiber.Ctx) map[string]string {
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
func (a *ApkHandlerAdapter) setResponseHeaders(c *fiber.Ctx, response *apkDomain.PackageResponse) {
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
		c.Set("X-Cache-Source", "proxynd-apk")
	} else {
		c.Set("X-Cache", "MISS")
		if response.ProxyUsed != "" {
			c.Set("X-Upstream-Server", response.ProxyUsed)
		}
	}
	
	// 파일 타입별 추가 헤더
	if response.IsApkFile {
		c.Set("X-Content-Type", "apk-package")
	} else if response.IsIndex {
		c.Set("X-Content-Type", "package-index")
	} else if response.IsSignature {
		c.Set("X-Content-Type", "signature-file")
	}
	
	// 보안 헤더
	c.Set("X-Content-Type-Options", "nosniff")
	c.Set("X-Frame-Options", "DENY")
	
	// 캐시 제어 헤더
	if response.IsIndex {
		// 인덱스 파일은 짧은 캐시 시간
		c.Set("Cache-Control", "public, max-age=3600") // 1시간
	} else if response.IsApkFile {
		// APK 파일은 긴 캐시 시간
		c.Set("Cache-Control", "public, max-age=86400") // 24시간
	} else if response.IsSignature {
		// 서명 파일은 중간 캐시 시간
		c.Set("Cache-Control", "public, max-age=21600") // 6시간
	} else {
		// 기타 파일은 기본 캐시 시간
		c.Set("Cache-Control", "public, max-age=3600") // 1시간
	}
}