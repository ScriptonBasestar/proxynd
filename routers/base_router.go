package routers

import (
	"deproxy/configs"
	"github.com/gin-gonic/gin"
)

// SetupRouter function will perform all route operations
func SetupRouter() *gin.Engine {
	r := gin.Default()

	//Giving access to storage folder
	r.Static("/storage", "storage")

	//Giving access to template folder
	//r.Static("/templates", "templates")
	r.LoadHTMLGlob("templates/*")

	r.GET("/", func(c *gin.Context) {
		mvnSite := configs.MavenConfig{}
		mvnSite.ReadConfig("conf/proxy-maven.yaml")
		aptSite := configs.AptConfig{}
		aptSite.ReadConfig("conf/proxy-apt.yaml")
		c.HTML(200, "dashboard.html", gin.H{
			"maven": mvnSite.Servers,
			"apt":   aptSite.Servers,
		})
	})

	//r.Use(func(c *gin.Context) {
	//	// add header Access-Control-Allow-Origin
	//	c.Writer.Header().Set("Content-Type", "application/json")
	//	c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
	//	c.Writer.Header().Set("Access-Control-Max-Age", "86400")
	//	c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, GET, PUT, DELETE, UPDATE")
	//	c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, X-Max")
	//	c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
	//
	//	if c.Request.Method == "OPTIONS" {
	//		c.AbortWithStatus(200)
	//	} else {
	//		c.Next()
	//	}
	//})

	return r
}
