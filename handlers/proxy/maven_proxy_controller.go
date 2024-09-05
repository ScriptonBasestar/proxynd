package proxy

import (
	"deproxy/configs"
	"deproxy/helpers"
	"fmt"
	"github.com/gin-gonic/gin"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

func responseHandler(c *gin.Context, responseContent []byte, filename string) {
	ext := filename[strings.LastIndex(filename, ".")+1:]
	// text or octetstream
	if ext == "pom" || ext == "xml" {
		c.Header("Content-Type", "application/xml")
		//c.Header("Last-Modified", "date")
		c.Header("Content-Type", "application/xml")
		c.Header("Content-Disposition", "inline; filename="+filename)
		//c.XML(http.StatusOK, xmlContent)
		c.Data(http.StatusOK, "application/xml", responseContent)
	} else {
		c.Header("Content-Description", "File Transfer")
		c.Header("Content-Transfer-Encoding", "binary")
		c.Header("Content-Disposition", "attachment; filename="+filename)
		c.Header("Content-Type", "application/octet-stream")
		//c.File(filefullpath)
		c.Data(http.StatusOK, "application/octet-stream", responseContent)
	}
}

func MavenProxy(c *gin.Context) {
	log.Printf("Access proxy maven\n")

	requestPath := c.Param("path")

	// fixme di
	storageDir := helpers.GetStorageDir()
	globalConfig := configs.GlobalConfig{}
	globalConfig.ReadConfig()
	config := configs.MavenProxyConfig{}
	config.ReadConfig()

	//basedir := helpers.GetEnv("STORAGE_DIR", "./tmp")
	//if basedir == "" {
	//	var err error
	//	basedir, err = homedir.Dir()
	//	if err != nil {
	//		log.Fatal(err)
	//	}
	//}

	var responseContent []byte
	// Create the file
	filefullpath := path.Join(storageDir, config.Path, requestPath)
	filename := filepath.Base(filefullpath)
	if _, err := os.Stat(filefullpath); os.IsNotExist(err) {
		dirpath := filepath.Dir(filefullpath)
		os.MkdirAll(dirpath, 0766)

		// Get the data
		fmt.Println(len(config.Proxies))

		for s, server := range config.Proxies {
			fmt.Printf("for moon %d\n", s)
			resp, err := http.Get(helpers.JoinURL(server.Url, requestPath))
			if err != nil {
				log.Fatal(err)
				return
			}
			//fmt.Println(resp.Header)
			fmt.Println(resp.StatusCode)
			// Writer the body to file
			//_, err = io.Copy(out, resp.Body)
			bytes, _ := io.ReadAll(resp.Body)
			err = os.WriteFile(filefullpath, bytes, 0766)
			if err != nil {
				log.Fatal(err)
				return
			}
			resp.Body.Close()
			responseContent = bytes
			break
		}
	} else {
		bytes, _ := os.ReadFile(filefullpath)
		responseContent = bytes
	}

	responseHandler(c, responseContent, filename)
}
