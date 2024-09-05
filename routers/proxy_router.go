package routers

import (
	"github.com/gin-gonic/gin"
	"proxynd/configs"
	proxynd "proxynd/handlers/proxy"
	"proxynd/helpers"
)

func ProxyRouter(r *gin.Engine) {
	// fixme di??
	storageDir := helpers.GetStorageDir()
	globalConfig := configs.GlobalConfig{}
	globalConfig.ReadConfig()

	mavenProxyConfig := configs.MavenProxyConfig{}
	//mavenProxyConfig.ReadConfig()

	if mavenProxyConfig.ConfigExists() {
		mavenGroup := r.Group("/proxy/maven")
		mavenGroup.GET("*path", proxynd.MavenProxy)
	} else {
		r.GET("/proxy/maven/*path", func(c *gin.Context) {
			c.HTML(200, "alert.html", gin.H{
				"ConfigFileDir":  storageDir,
				"ConfigFileName": "maven-proxy.yaml",
			})
		})
	}

	aptProxyConfig := configs.AptProxyConfig{}
	//aptProxyConfig.ReadConfig()

	if aptProxyConfig.ConfigExists() {
		aptGroup := r.Group("/proxy/apt")
		aptGroup.GET(":osType/*requestPath", proxynd.AptProxy)
	} else {
		r.GET("/proxy/maven/*path", func(c *gin.Context) {
			c.HTML(200, "alert.html", gin.H{
				"ConfigFileDir":  storageDir,
				"ConfigFileName": "apt-proxy.yaml",
			})
		})
	}
}
