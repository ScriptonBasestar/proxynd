package routers

import (
	"github.com/gin-gonic/gin"
	"proxynd/configs"
)

// BaseRouter function will perform all route operations
func BaseRouter() *gin.Engine {
	r := gin.Default()

	//Giving access to storage folder
	r.Static("/storage", "storage")

	//Giving access to template folder
	//r.Static("/templates", "templates")
	r.LoadHTMLGlob("templates/*")

	r.GET("/", func(c *gin.Context) {
		mvnSite := configs.MavenProxyConfig{}
		mvnSite.ReadConfig()
		aptSite := configs.AptProxyConfig{}
		aptSite.ReadConfig()
		c.HTML(200, "dashboard.html", gin.H{
			"mavenProxy": mvnSite.Proxies,
			"aptProxy":   aptSite.Proxies,
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
