package routers

import (
	"github.com/gofiber/fiber/v2"
	"proxynd/configs"
	proxynd "proxynd/handlers/proxy"
	"proxynd/helpers"
)

func ProxyRouter(app *fiber.App) {
	// fixme di??
	storageDir := helpers.GetStorageDir()
	globalConfig := configs.GlobalConfig{}
	globalConfig.ReadConfig()

	mavenProxyConfig := configs.MavenProxyConfig{}
	//mavenProxyConfig.ReadConfig()

	if mavenProxyConfig.ConfigExists() {
		mavenGroup := app.Group("/proxy/maven")
		mavenGroup.Get("/*", proxynd.MavenProxy)
	} else {
		app.Get("/proxy/maven/*", func(c *fiber.Ctx) error {
			return c.Render("alert", fiber.Map{
				"ConfigFileDir":  storageDir,
				"ConfigFileName": "maven-proxy.yaml",
			})
		})
	}

	aptProxyConfig := configs.AptProxyConfig{}
	//aptProxyConfig.ReadConfig()

	if aptProxyConfig.ConfigExists() {
		aptGroup := app.Group("/proxy/apt")
		aptGroup.Get("/:osType/*", proxynd.AptProxy)
	} else {
		app.Get("/proxy/apt/*", func(c *fiber.Ctx) error {
			return c.Render("alert", fiber.Map{
				"ConfigFileDir":  storageDir,
				"ConfigFileName": "apt-proxy.yaml",
			})
		})
	}
}
