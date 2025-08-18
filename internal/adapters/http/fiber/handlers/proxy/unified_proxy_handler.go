package proxy

import (
	"log"

	"github.com/gofiber/fiber/v2"

	"proxynd/internal/adapters/http"
	"proxynd/internal/config"
	"proxynd/internal/factory"
	"proxynd/internal/helpers"
	"proxynd/internal/logging"
	"proxynd/internal/usecase"
)

// 전역 팩토리 인스턴스
var globalAdapterFactory *factory.HandlerAdapterFactory

// TODO: HEXAGONAL_MIGRATION - Replace with dependency injection
// Global proxy service for migration - should be injected via DI container
var globalProxyService *usecase.ProxyService

// InitializeGlobalFactory 전역 팩토리 초기화
func InitializeGlobalFactory() {
	if globalAdapterFactory == nil {
		globalAdapterFactory = factory.NewHandlerAdapterFactory()
		log.Printf("Global handler adapter factory initialized")
	}
}

// UnifiedProxyHandlerWithFactory 팩토리 기반 통합 프록시 핸들러
func UnifiedProxyHandlerWithFactory(c *fiber.Ctx) error {
	// TODO: HEXAGONAL_MIGRATION - Replace factory logic with usecase calls
	// Current implementation uses factory pattern for adapter selection
	// Should be migrated to use usecase.ProxyService.HandleProxyRequest()

	proxyType := c.Params("type")
	path := c.Params("*")

	log.Printf("Access unified proxy (factory) - type: %s, path: %s\n", proxyType, path)

	// TODO: HEXAGONAL_MIGRATION - Remove factory dependency, use injected ProxyService
	if globalProxyService != nil {
		// New architecture path
		req := &usecase.ProxyRequest{
			PackageType: proxyType,
			Path:        path,
			Method:      c.Method(),
			Headers:     make(map[string]string),
			QueryParams: make(map[string]string),
			Body:        c.Body(),
		}

		// Copy headers
		c.GetReqHeaders()
		for key, values := range c.GetReqHeaders() {
			if len(values) > 0 {
				req.Headers[key] = values[0]
			}
		}

		// Copy query params
		c.Context().QueryArgs().VisitAll(func(key, value []byte) {
			req.QueryParams[string(key)] = string(value)
		})

		resp, err := globalProxyService.HandleProxyRequest(c.Context(), req)
		if err != nil {
			log.Printf("ProxyService error: %v", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		// Set response headers
		for key, value := range resp.Headers {
			c.Set(key, value)
		}

		return c.Status(resp.StatusCode).Send(resp.Content)
	}

	// Legacy factory path (fallback during migration)
	if globalAdapterFactory == nil {
		InitializeGlobalFactory()
	}

	adapter, err := globalAdapterFactory.GetAdapter(proxyType)
	if err != nil {
		log.Printf("Failed to get adapter for type %s: %v", proxyType, err)
		return c.Status(fiber.StatusBadRequest).SendString("Unknown proxy type: " + proxyType)
	}

	return adapter.Handle(c)
}

// UnifiedProxyHandler 통합 프록시 핸들러 - 모든 프록시 타입을 처리 (레거시 호환용)
func UnifiedProxyHandler(c *fiber.Ctx) error {
	proxyType := c.Params("type")
	path := c.Params("*")

	log.Printf("Access unified proxy - type: %s, path: %s\n", proxyType, path)

	// 프록시 타입별로 적절한 핸들러로 라우팅
	switch proxyType {
	case "maven":
		// Maven 설정 확인
		mavenConfig := config.MavenProxySettings{}
		if !mavenConfig.ConfigExists() {
			return renderConfigAlert(c, "maven-proxy.yaml")
		}

		// 설정 로드
		if err := mavenConfig.ReadConfig(); err != nil {
			log.Printf("Warning: Failed to read Maven config: %v", err)
			return c.Status(500).SendString("Maven 설정을 읽을 수 없습니다")
		}

		// 브라우저 요청 vs 파일 다운로드 요청 구분
		if IsBrowserRequest(c) && IsDirectoryPath(path) {
			// 브라우저 인터페이스 요청 - 새로운 아키텍처 사용
			logger := logging.GetLogger()
			adapter := http.NewMavenBrowserAdapter(mavenConfig, logger)
			return adapter.Handle(c)
		} else {
			// 파일 다운로드 요청 - 향상된 핸들러 사용
			handler := NewMavenHandler()
			return handler.Handle(c)
		}

	case "apt":
		// APT 설정 확인
		aptConfig := config.AptProxyConfig{}
		if !aptConfig.ConfigExists() {
			return renderConfigAlert(c, "apt-proxy.yaml")
		}

		// 설정 로드
		if err := aptConfig.ReadConfig(); err != nil {
			log.Printf("Warning: Failed to read APT config: %v", err)
			return c.Status(500).SendString("APT 설정을 읽을 수 없습니다")
		}

		// 새로운 아키텍처 사용 - APT Handler Adapter
		logger := logging.GetLogger()
		adapter := http.NewAPTHandlerAdapter(aptConfig, logger)
		return adapter.Handle(c)

	case "npm":
		// NPM 설정 확인
		npmConfig := config.NpmProxySettings{}
		if !npmConfig.ConfigExists() {
			return renderConfigAlert(c, "npm-proxy.yaml")
		}

		// 설정 로드
		if err := npmConfig.ReadConfig(); err != nil {
			log.Printf("Warning: Failed to read NPM config: %v", err)
			return c.Status(500).SendString("NPM 설정을 읽을 수 없습니다")
		}

		// 새로운 아키텍처 사용 - NPM Handler Adapter
		logger := logging.GetLogger()
		adapter := http.NewNPMHandlerAdapter(npmConfig, logger)
		return adapter.Handle(c)

	case "pip":
		// PIP 설정 확인
		pipConfig := config.PipProxySettings{}
		if !pipConfig.ConfigExists() {
			return renderConfigAlert(c, "pip-proxy.yaml")
		}

		// 설정 로드
		if err := pipConfig.ReadConfig(); err != nil {
			log.Printf("Warning: Failed to read PIP config: %v", err)
			return c.Status(500).SendString("PIP 설정을 읽을 수 없습니다")
		}

		// 새로운 아키텍처 사용 - PIP Handler Adapter
		logger := logging.GetLogger()
		adapter := http.NewPIPHandlerAdapter(pipConfig, logger)
		return adapter.Handle(c)

	case "docker":
		// Docker 설정 확인
		dockerConfig := config.DockerProxySettings{}
		if !dockerConfig.ConfigExists() {
			return renderConfigAlert(c, "docker-proxy.yaml")
		}

		// 설정 로드
		if err := dockerConfig.ReadConfig(); err != nil {
			log.Printf("Warning: Failed to read Docker config: %v", err)
			return c.Status(500).SendString("Docker 설정을 읽을 수 없습니다")
		}

		// 새로운 아키텍처 사용 - Docker Handler Adapter
		logger := logging.GetLogger()
		adapter := http.NewDockerHandlerAdapter(dockerConfig, logger)
		return adapter.Handle(c)

	case "yum":
		// YUM 설정 확인
		yumConfig := config.YumProxySettings{}
		if !yumConfig.ConfigExists() {
			return renderConfigAlert(c, "yum-proxy.yaml")
		}

		// 설정 로드
		if err := yumConfig.ReadConfig(); err != nil {
			log.Printf("Warning: Failed to read YUM config: %v", err)
			return c.Status(500).SendString("YUM 설정을 읽을 수 없습니다")
		}

		// 새로운 아키텍처 사용 - YUM Handler Adapter
		logger := logging.GetLogger()
		adapter := http.NewYumHandlerAdapter(yumConfig, logger)
		return adapter.Handle(c)

	case "apk":
		// APK 설정 확인
		apkConfig := config.ApkProxySettings{}
		if !apkConfig.ConfigExists() {
			return renderConfigAlert(c, "apk-proxy.yaml")
		}

		// 설정 로드
		if err := apkConfig.ReadConfig(); err != nil {
			log.Printf("Warning: Failed to read APK config: %v", err)
			return c.Status(500).SendString("APK 설정을 읽을 수 없습니다")
		}

		// 새로운 아키텍처 사용 - APK Handler Adapter
		logger := logging.GetLogger()
		adapter := http.NewApkHandlerAdapter(apkConfig, logger)
		return adapter.Handle(c)

	default:
		return c.Status(fiber.StatusBadRequest).SendString("Unknown proxy type: " + proxyType)
	}
}

// renderConfigAlert 설정 파일이 없을 때 알림 페이지 렌더링
func renderConfigAlert(c *fiber.Ctx, configFileName string) error {
	storageDir := helpers.GetStorageDir()
	return c.Render("alert", fiber.Map{
		"ConfigFileDir":  storageDir,
		"ConfigFileName": configFileName,
	})
}
