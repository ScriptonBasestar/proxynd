package proxy

import (
	"log"

	"github.com/gofiber/fiber/v2"

	"proxynd/configs"
	"proxynd/helpers"
	"proxynd/internal/adapters/http"
	"proxynd/logging"
)

// UnifiedProxyHandler 통합 프록시 핸들러 - 모든 프록시 타입을 처리
func UnifiedProxyHandler(c *fiber.Ctx) error {
	proxyType := c.Params("type")
	path := c.Params("*")

	log.Printf("Access unified proxy - type: %s, path: %s\n", proxyType, path)

	// 프록시 타입별로 적절한 핸들러로 라우팅
	switch proxyType {
	case "maven":
		// Maven 설정 확인
		mavenConfig := configs.MavenProxyConfig{}
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
		aptConfig := configs.AptProxyConfig{}
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
		npmConfig := configs.NpmProxyConfig{}
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
		pipConfig := configs.PipProxyConfig{}
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
		dockerConfig := configs.DockerProxyConfig{}
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
		yumConfig := configs.YumProxyConfig{}
		if !yumConfig.ConfigExists() {
			return renderConfigAlert(c, "yum-proxy.yaml")
		}
		return YumProxyHandler(c)

	case "apk":
		// APK 설정 확인
		apkConfig := configs.ApkProxyConfig{}
		if !apkConfig.ConfigExists() {
			return renderConfigAlert(c, "apk-proxy.yaml")
		}
		return ApkProxyHandler(c)

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
