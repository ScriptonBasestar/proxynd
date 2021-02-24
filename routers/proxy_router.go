package routers

import (
	"dohoarding/handlers/proxy"
	"github.com/gin-gonic/gin"
)

func ProxyRouter(r *gin.Engine) {
	//mavenGroup := r.Group("/repo")
	//mavenGroup.GET("maven*", handlers.Proxy)
	//r.GET("/repo/maven/:path", handlers.Proxy)
	r.GET("/proxy/maven/*path", proxy.Maven)
}
