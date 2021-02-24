package routers

import (
	"dohoarding/handlers/mirror"
	"github.com/gin-gonic/gin"
)

func MirrorRouter(r *gin.Engine) {
	mirrorGroup := r.Group("/mirror")
	mirrorGroup.Use()
	{
		mirrorGroup.GET("maven/*path", mirror.MavenDownload)

		// list repository
		mirrorGroup.GET("maven", mirror.MavenDownload)
		// add repository
		mirrorGroup.POST("maven", mirror.MavenDownload)
		// modify repository
		mirrorGroup.PUT("maven", mirror.MavenDownload)
		// remove repository
		mirrorGroup.DELETE("maven", mirror.MavenDownload)
	}
}
