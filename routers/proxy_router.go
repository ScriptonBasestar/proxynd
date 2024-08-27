package routers

import (
	"deproxy/configs"
	dproxy "deproxy/handlers/proxy"
	"github.com/gin-gonic/gin"
)

func ProxyRouter(r *gin.Engine) {
	// fixme di??
	globalConfig := configs.GlobalConfig{}
	globalConfig.ReadConfig()

	mavenProxyConfig := configs.MavenProxyConfig{}
	mavenProxyConfig.ReadConfig()

	if mavenProxyConfig.ConfigExists() {
		mavenGroup := r.Group("/proxy/maven")
		mavenGroup.GET("*path", dproxy.MavenProxy)
	} else {
		r.GET("/proxy/maven/*path", func(c *gin.Context) {
			c.HTML(200, "alert.html", gin.H{
				"ConfigFileDir":  globalConfig.BaseDir,
				"ConfigFileName": "maven-proxy.yaml",
			})
		})
	}

	aptProxyConfig := configs.AptProxyConfig{}
	aptProxyConfig.ReadConfig()

	if aptProxyConfig.ConfigExists() {
		aptGroup := r.Group("/proxy/apt")
		aptGroup.GET(":osType/*requestPath", dproxy.AptProxy)
	} else {
		r.GET("/proxy/maven/*path", func(c *gin.Context) {
			c.HTML(200, "alert.html", gin.H{
				"ConfigFileDir":  globalConfig.BaseDir,
				"ConfigFileName": "apt-proxy.yaml",
			})
		})
	}
}
