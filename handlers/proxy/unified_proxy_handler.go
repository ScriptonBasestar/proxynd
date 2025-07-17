package proxy

import (
	"log"

	"github.com/gofiber/fiber/v2"

	"proxynd/configs"
	"proxynd/helpers"
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
		// 새로운 DI 기반 Maven 핸들러 사용
		handler := NewMavenHandler()
		return handler.Handle(c)

	case "apt":
		// APT 설정 확인
		aptConfig := configs.AptProxyConfig{}
		if !aptConfig.ConfigExists() {
			return renderConfigAlert(c, "apt-proxy.yaml")
		}
		// 새로운 DI 기반 핸들러 사용
		handler := NewAPTHandler()
		return handler.Handle(c)

	case "npm":
		// NPM 설정 확인
		npmConfig := configs.NpmProxyConfig{}
		if !npmConfig.ConfigExists() {
			return renderConfigAlert(c, "npm-proxy.yaml")
		}
		return NpmProxy(c)

	case "pip":
		// PIP 설정 확인
		pipConfig := configs.PipProxyConfig{}
		if !pipConfig.ConfigExists() {
			return renderConfigAlert(c, "pip-proxy.yaml")
		}
		return PipProxy(c)

	case "docker":
		// Docker 설정 확인
		dockerConfig := configs.DockerProxyConfig{}
		if !dockerConfig.ConfigExists() {
			return renderConfigAlert(c, "docker-proxy.yaml")
		}
		return DockerProxy(c)

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
