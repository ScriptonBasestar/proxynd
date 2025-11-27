package proxy

import (
	"log"
	"os"

	"github.com/gofiber/fiber/v2"

	"proxynd/internal/adapters/http"
	"proxynd/internal/config"
	"proxynd/internal/factory"
	"proxynd/internal/helpers"
	"proxynd/internal/logging"
)

// isDevelopmentMode 개발/테스트 환경 확인
func isDevelopmentMode() bool {
	env := os.Getenv("PROXYND_ENV")
	return env == "development" || env == "test"
}

// 전역 팩토리 인스턴스
var globalAdapterFactory *factory.HandlerAdapterFactory

// InitializeGlobalFactory 전역 팩토리 초기화
func InitializeGlobalFactory() {
	if globalAdapterFactory == nil {
		globalAdapterFactory = factory.NewHandlerAdapterFactory()
		log.Printf("Global handler adapter factory initialized")
	}
}

// UnifiedProxyHandlerWithFactory 팩토리 기반 통합 프록시 핸들러
func UnifiedProxyHandlerWithFactory(c *fiber.Ctx) error {
	proxyType := c.Params("type")
	path := c.Params("*")

	log.Printf("Access unified proxy (factory) - type: %s, path: %s\n", proxyType, path)

	// 전역 팩토리 확인
	if globalAdapterFactory == nil {
		InitializeGlobalFactory()
	}

	// 프록시 타입에 맞는 어댑터 가져오기
	adapter, err := globalAdapterFactory.GetAdapter(proxyType)
	if err != nil {
		log.Printf("Failed to get adapter for type %s: %v", proxyType, err)
		return c.Status(fiber.StatusBadRequest).SendString("Unknown proxy type: " + proxyType)
	}

	// 어댑터로 요청 처리
	return adapter.Handle(c)
}

// UnifiedProxyHandler 통합 프록시 핸들러 - 모든 프록시 타입을 처리 (레거시 호환용)
func UnifiedProxyHandler(c *fiber.Ctx) error {
	proxyType := c.Params("type")
	log.Printf("Access unified proxy - type: %s, path: %s\n", proxyType, c.Params("*"))

	// 프록시 타입별로 적절한 핸들러로 라우팅
	switch proxyType {
	case "maven":
		return handleMavenProxy(c)
	case "apt":
		return handleAptProxy(c)
	case "npm":
		return handleNpmProxy(c)
	case "pip":
		return handlePipProxy(c)
	case "docker":
		return handleDockerProxy(c)
	case "yum":
		return handleYumProxy(c)
	case "apk":
		return handleApkProxy(c)
	default:
		return c.Status(fiber.StatusBadRequest).SendString("Unknown proxy type: " + proxyType)
	}
}

// handleMavenProxy 처리 Maven 프록시 요청
func handleMavenProxy(c *fiber.Ctx) error {
	mavenConfig := config.MavenProxySettings{}

	if !isDevelopmentMode() {
		if !mavenConfig.ConfigExists() {
			return renderConfigAlert(c, "maven-proxy.yaml")
		}
		if err := mavenConfig.ReadConfig(); err != nil {
			log.Printf("Warning: Failed to read Maven config: %v", err)
			return c.Status(500).SendString("Maven 설정을 읽을 수 없습니다")
		}
	}

	path := c.Params("*")
	if IsBrowserRequest(c) && IsDirectoryPath(path) {
		logger := logging.GetLogger()
		adapter := http.NewMavenBrowserAdapter(mavenConfig, logger)
		return adapter.Handle(c)
	}
	handler := NewMavenHandler()
	return handler.Handle(c)
}

// handleAptProxy 처리 APT 프록시 요청
func handleAptProxy(c *fiber.Ctx) error {
	aptConfig := config.AptProxyConfig{}

	if !isDevelopmentMode() {
		if !aptConfig.ConfigExists() {
			return renderConfigAlert(c, "apt-proxy.yaml")
		}
		if err := aptConfig.ReadConfig(); err != nil {
			log.Printf("Warning: Failed to read APT config: %v", err)
			return c.Status(500).SendString("APT 설정을 읽을 수 없습니다")
		}
	}

	logger := logging.GetLogger()
	adapter := http.NewAPTHandlerAdapter(aptConfig, logger)
	return adapter.Handle(c)
}

// handleNpmProxy 처리 NPM 프록시 요청
func handleNpmProxy(c *fiber.Ctx) error {
	npmConfig := config.NpmProxySettings{}

	if !npmConfig.ConfigExists() {
		if !isDevelopmentMode() {
			return renderConfigAlert(c, "npm-proxy.yaml")
		}
		log.Printf("Warning: npm-proxy.yaml not found, using empty config")
	} else {
		if err := npmConfig.ReadConfig(); err != nil {
			log.Printf("Warning: Failed to read NPM config: %v", err)
			if !isDevelopmentMode() {
				return c.Status(500).SendString("NPM 설정을 읽을 수 없습니다")
			}
		}
	}

	logger := logging.GetLogger()
	adapter := http.NewNPMHandlerAdapter(npmConfig, logger)
	return adapter.Handle(c)
}

// handlePipProxy 처리 PIP 프록시 요청
func handlePipProxy(c *fiber.Ctx) error {
	pipConfig := config.PipProxySettings{}

	if !pipConfig.ConfigExists() {
		if !isDevelopmentMode() {
			return renderConfigAlert(c, "pip-proxy.yaml")
		}
		log.Printf("Warning: pip-proxy.yaml not found, using empty config")
	} else {
		if err := pipConfig.ReadConfig(); err != nil {
			log.Printf("Warning: Failed to read PIP config: %v", err)
			if !isDevelopmentMode() {
				return c.Status(500).SendString("PIP 설정을 읽을 수 없습니다")
			}
		}
	}

	logger := logging.GetLogger()
	adapter := http.NewPIPHandlerAdapter(pipConfig, logger)
	return adapter.Handle(c)
}

// handleDockerProxy 처리 Docker 프록시 요청
func handleDockerProxy(c *fiber.Ctx) error {
	dockerConfig := config.DockerProxySettings{}

	if !dockerConfig.ConfigExists() {
		if !isDevelopmentMode() {
			return renderConfigAlert(c, "docker-proxy.yaml")
		}
		log.Printf("Warning: docker-proxy.yaml not found, using empty config")
	} else {
		if err := dockerConfig.ReadConfig(); err != nil {
			log.Printf("Warning: Failed to read Docker config: %v", err)
			if !isDevelopmentMode() {
				return c.Status(500).SendString("Docker 설정을 읽을 수 없습니다")
			}
		}
	}

	logger := logging.GetLogger()
	adapter := http.NewDockerHandlerAdapter(dockerConfig, logger)
	return adapter.Handle(c)
}

// handleYumProxy 처리 YUM 프록시 요청
func handleYumProxy(c *fiber.Ctx) error {
	yumConfig := config.YumProxySettings{}

	if !yumConfig.ConfigExists() {
		if !isDevelopmentMode() {
			return renderConfigAlert(c, "yum-proxy.yaml")
		}
		log.Printf("Warning: yum-proxy.yaml not found, using empty config")
	} else {
		if err := yumConfig.ReadConfig(); err != nil {
			log.Printf("Warning: Failed to read YUM config: %v", err)
			if !isDevelopmentMode() {
				return c.Status(500).SendString("YUM 설정을 읽을 수 없습니다")
			}
		}
	}

	logger := logging.GetLogger()
	adapter := http.NewYumHandlerAdapter(yumConfig, logger)
	return adapter.Handle(c)
}

// handleApkProxy 처리 APK 프록시 요청
func handleApkProxy(c *fiber.Ctx) error {
	apkConfig := config.ApkProxySettings{}

	if !apkConfig.ConfigExists() {
		if !isDevelopmentMode() {
			return renderConfigAlert(c, "apk-proxy.yaml")
		}
		log.Printf("Warning: apk-proxy.yaml not found, using empty config")
	} else {
		if err := apkConfig.ReadConfig(); err != nil {
			log.Printf("Warning: Failed to read APK config: %v", err)
			if !isDevelopmentMode() {
				return c.Status(500).SendString("APK 설정을 읽을 수 없습니다")
			}
		}
	}

	logger := logging.GetLogger()
	adapter := http.NewApkHandlerAdapter(apkConfig, logger)
	return adapter.Handle(c)
}

// renderConfigAlert 설정 파일이 없을 때 알림 응답
func renderConfigAlert(c *fiber.Ctx, configFileName string) error {
	storageDir := helpers.GetStorageDir()

	// JSON 응답으로 변경 (템플릿 의존성 제거)
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":         "warning",
		"message":        "Configuration file not found, using defaults",
		"configFileDir":  storageDir,
		"configFileName": configFileName,
	})
}
