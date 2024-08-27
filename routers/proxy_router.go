package routers

import (
	dproxy "deproxy/handlers/proxy"
	"github.com/gin-gonic/gin"
)

func ProxyRouter(r *gin.Engine) {
	mavenGroup := r.Group("/proxy/repo")
	mavenGroup.GET("*path", dproxy.Maven)
	//r.GET("/repo/maven/:path", handlers.Proxy)
	//r.GET("/proxy/maven/*path", dproxy.Maven)
	aptGroup := r.Group("/proxy/apt")
	aptGroup.GET("*path", dproxy.Apt)
	//r.GET("/proxy/apt/*path", dproxy.Apt)
	//r.GET("/proxy/vagrant/*path", func(context *gin.Context) {
}
