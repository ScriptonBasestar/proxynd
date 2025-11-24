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
	path := c.Params("*")

	log.Printf("Access unified proxy - type: %s, path: %s\n", proxyType, path)

	// 프록시 타입별로 적절한 핸들러로 라우팅
	switch proxyType {
	case "maven":
		// Maven 설정 초기화
		mavenConfig := config.MavenProxySettings{}

		// Maven 설정 확인 (개발/테스트 환경에서는 스킵)
		if !isDevelopmentMode() {
			if !mavenConfig.ConfigExists() {
				return renderConfigAlert(c, "maven-proxy.yaml")
			}

			// 설정 로드
			if err := mavenConfig.ReadConfig(); err != nil {
				log.Printf("Warning: Failed to read Maven config: %v", err)
				return c.Status(500).SendString("Maven 설정을 읽을 수 없습니다")
			}
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
		// APT 설정 초기화
		aptConfig := config.AptProxyConfig{}

		// APT 설정 확인 (개발/테스트 환경에서는 스킵)
		if !isDevelopmentMode() {
			if !aptConfig.ConfigExists() {
				return renderConfigAlert(c, "apt-proxy.yaml")
			}

			// 설정 로드
			if err := aptConfig.ReadConfig(); err != nil {
				log.Printf("Warning: Failed to read APT config: %v", err)
				return c.Status(500).SendString("APT 설정을 읽을 수 없습니다")
			}
		}

		// 새로운 아키텍처 사용 - APT Handler Adapter
		logger := logging.GetLogger()
		adapter := http.NewAPTHandlerAdapter(aptConfig, logger)
		return adapter.Handle(c)

	case "npm":
		// NPM 설정 초기화
		npmConfig := config.NpmProxySettings{}

		// 설정 파일 존재 확인
		if !npmConfig.ConfigExists() {
			if !isDevelopmentMode() {
				return renderConfigAlert(c, "npm-proxy.yaml")
			}
			// 개발/테스트 환경에서 설정 파일이 없으면 경고만 출력
			log.Printf("Warning: npm-proxy.yaml not found, using empty config")
		} else {
			// 설정 파일이 있으면 로드 (개발/테스트 환경 포함)
			if err := npmConfig.ReadConfig(); err != nil {
				log.Printf("Warning: Failed to read NPM config: %v", err)
				if !isDevelopmentMode() {
					return c.Status(500).SendString("NPM 설정을 읽을 수 없습니다")
				}
			}
		}

		// 새로운 아키텍처 사용 - NPM Handler Adapter
		logger := logging.GetLogger()
		adapter := http.NewNPMHandlerAdapter(npmConfig, logger)
		return adapter.Handle(c)

	case "pip":
		// PIP 설정 초기화
		pipConfig := config.PipProxySettings{}

		// 설정 파일 존재 확인
		if !pipConfig.ConfigExists() {
			if !isDevelopmentMode() {
				return renderConfigAlert(c, "pip-proxy.yaml")
			}
			log.Printf("Warning: pip-proxy.yaml not found, using empty config")
		} else {
			// 설정 파일이 있으면 로드 (개발/테스트 환경 포함)
			if err := pipConfig.ReadConfig(); err != nil {
				log.Printf("Warning: Failed to read PIP config: %v", err)
				if !isDevelopmentMode() {
					return c.Status(500).SendString("PIP 설정을 읽을 수 없습니다")
				}
			}
		}

		// 새로운 아키텍처 사용 - PIP Handler Adapter
		logger := logging.GetLogger()
		adapter := http.NewPIPHandlerAdapter(pipConfig, logger)
		return adapter.Handle(c)

	case "docker":
		// Docker 설정 초기화
		dockerConfig := config.DockerProxySettings{}

		// 설정 파일 존재 확인
		if !dockerConfig.ConfigExists() {
			if !isDevelopmentMode() {
				return renderConfigAlert(c, "docker-proxy.yaml")
			}
			log.Printf("Warning: docker-proxy.yaml not found, using empty config")
		} else {
			// 설정 파일이 있으면 로드 (개발/테스트 환경 포함)
			if err := dockerConfig.ReadConfig(); err != nil {
				log.Printf("Warning: Failed to read Docker config: %v", err)
				if !isDevelopmentMode() {
					return c.Status(500).SendString("Docker 설정을 읽을 수 없습니다")
				}
			}
		}

		// 새로운 아키텍처 사용 - Docker Handler Adapter
		logger := logging.GetLogger()
		adapter := http.NewDockerHandlerAdapter(dockerConfig, logger)
		return adapter.Handle(c)

	case "yum":
		// YUM 설정 초기화
		yumConfig := config.YumProxySettings{}

		// 설정 파일 존재 확인
		if !yumConfig.ConfigExists() {
			if !isDevelopmentMode() {
				return renderConfigAlert(c, "yum-proxy.yaml")
			}
			log.Printf("Warning: yum-proxy.yaml not found, using empty config")
		} else {
			// 설정 파일이 있으면 로드 (개발/테스트 환경 포함)
			if err := yumConfig.ReadConfig(); err != nil {
				log.Printf("Warning: Failed to read YUM config: %v", err)
				if !isDevelopmentMode() {
					return c.Status(500).SendString("YUM 설정을 읽을 수 없습니다")
				}
			}
		}

		// 새로운 아키텍처 사용 - YUM Handler Adapter
		logger := logging.GetLogger()
		adapter := http.NewYumHandlerAdapter(yumConfig, logger)
		return adapter.Handle(c)

	case "apk":
		// APK 설정 초기화
		apkConfig := config.ApkProxySettings{}

		// 설정 파일 존재 확인
		if !apkConfig.ConfigExists() {
			if !isDevelopmentMode() {
				return renderConfigAlert(c, "apk-proxy.yaml")
			}
			log.Printf("Warning: apk-proxy.yaml not found, using empty config")
		} else {
			// 설정 파일이 있으면 로드 (개발/테스트 환경 포함)
			if err := apkConfig.ReadConfig(); err != nil {
				log.Printf("Warning: Failed to read APK config: %v", err)
				if !isDevelopmentMode() {
					return c.Status(500).SendString("APK 설정을 읽을 수 없습니다")
				}
			}
		}

		// 새로운 아키텍처 사용 - APK Handler Adapter
		logger := logging.GetLogger()
		adapter := http.NewApkHandlerAdapter(apkConfig, logger)
		return adapter.Handle(c)

	default:
		return c.Status(fiber.StatusBadRequest).SendString("Unknown proxy type: " + proxyType)
	}
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
