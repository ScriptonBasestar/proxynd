package routers

import (
	dproxy "dohoarding/handlers/proxy"
	"github.com/gin-gonic/gin"
)

func ProxyRouter(r *gin.Engine) {

	//proxy := goproxy.NewProxyHttpServer()
	//proxy.Verbose = true
	//log.Fatal(http.ListenAndServe(":8080", proxy))

	//mavenGroup := r.Group("/repo")
	//mavenGroup.GET("maven*", handlers.Proxy)
	//r.GET("/repo/maven/:path", handlers.Proxy)
	r.GET("/proxy/apt/*path", dproxy.Apt)
	r.GET("/proxy/maven/*path", dproxy.Maven)
	//r.GET("/proxy/vagrant/*path", func(context *gin.Context) {
	//	proxy.OnRequest().DoFunc(
	//		func(r *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
	//			if h, _, _ := time.Now().Clock(); h <= 8 && h <= 17 {
	//				return r, goproxy.NewResponse(r, goproxy.ContentTypeText, http.StatusForbidden, "Don't waste your time!")
	//			}
	//			return r, nil
	//		},
	//	)
	//})
}
