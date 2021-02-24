package proxy

import (
	u "dohoarding/apiHelpers"
	config2 "dohoarding/config"
	"dohoarding/helpers"
	"fmt"
	"github.com/gin-gonic/gin"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
)

func Maven(c *gin.Context) {
	log.Printf("Access proxy maven\n")
	var downloadService config2.DownloadService

	requestPath := c.Param("path")
	config, err := downloadService.DownloadProxy(requestPath)
	if err != nil {
		u.Respond(c.Writer, u.Message(1, "Invalid request"))
		return
	}

	// Create the file
	dirpath := filepath.Dir(path.Join("d:/tmp/cachedir" + requestPath))
	filename := filepath.Base(requestPath)
	os.MkdirAll(dirpath, os.ModePerm)
	out, err := os.Create(path.Join("d:/tmp/cachedir" + requestPath))
	//out, err := os.Create("d:/tmp/cachedir/aaa/bbb")
	if err != nil {
		panic(err)
		return
	}
	defer out.Close()

	// Get the data
	fmt.Println(len(config.Server))
	for s, server := range config.Server {
		fmt.Printf("for moon %s\n", s)
		resp, err := http.Get(helpers.JoinURL(server.Url, requestPath))
		if err != nil {
			log.Fatal(err)
			return
		}
		//fmt.Println(resp.Header)
		fmt.Println(resp.StatusCode)
		// Writer the body to file
		_, err = io.Copy(out, resp.Body)
		if err != nil {
			log.Fatal(err)
			return
		}
		resp.Body.Close()
		break
	}

	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Transfer-Encoding", "binary")
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Header("Content-Type", "application/octet-stream")
	c.File("d:/tmp/cachedir" + requestPath)
}
