package proxy

import (
	"fmt"
	"github.com/gofiber/fiber/v2"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"proxynd/configs"
	"proxynd/helpers"
	"strings"
)

func responseHandler(c *fiber.Ctx, responseContent []byte, filename string) error {
	ext := filename[strings.LastIndex(filename, ".")+1:]
	// text or octetstream
	if ext == "pom" || ext == "xml" {
		c.Set("Content-Type", "application/xml")
		//c.Set("Last-Modified", "date")
		c.Set("Content-Disposition", "inline; filename="+filename)
		return c.Status(fiber.StatusOK).Send(responseContent)
	} else {
		c.Set("Content-Description", "File Transfer")
		c.Set("Content-Transfer-Encoding", "binary")
		c.Set("Content-Disposition", "attachment; filename="+filename)
		c.Set("Content-Type", "application/octet-stream")
		return c.Status(fiber.StatusOK).Send(responseContent)
	}
}

func MavenProxy(c *fiber.Ctx) error {
	log.Printf("Access proxy maven\n")

	requestPath := c.Params("*")

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

	// 미들웨어에서 전달된 캐시 정보 확인
	cacheHit, _ := c.Locals("cache_hit").(bool)
	cachePath, _ := c.Locals("cache_path").(string)
	
	var responseContent []byte
	var filefullpath string
	var filename string
	
	// 캐시 경로가 있으면 사용, 없으면 기본 경로 생성
	if cachePath != "" {
		filefullpath = cachePath
		filename = filepath.Base(filefullpath)
	} else {
		filefullpath = path.Join(storageDir, config.Path, requestPath)
		filename = filepath.Base(filefullpath)
	}
	
	// 캐시가 히트하지 않았을 때만 다운로드
	if !cacheHit {
		dirpath := filepath.Dir(filefullpath)
		os.MkdirAll(dirpath, 0766)

		// Get the data
		fmt.Println(len(config.Proxies))

		for s, server := range config.Proxies {
			fmt.Printf("for moon %d\n", s)
			resp, err := http.Get(helpers.JoinURL(server.Url, requestPath))
			if err != nil {
				log.Printf("Error fetching from proxy: %v", err)
				continue
			}
			//fmt.Println(resp.Header)
			fmt.Println(resp.StatusCode)
			// Writer the body to file
			//_, err = io.Copy(out, resp.Body)
			bytes, _ := io.ReadAll(resp.Body)
			err = os.WriteFile(filefullpath, bytes, 0766)
			if err != nil {
				log.Printf("Error writing file: %v", err)
				return c.Status(fiber.StatusInternalServerError).SendString("Error writing file")
			}
			resp.Body.Close()
			responseContent = bytes
			break
		}
	} else {
		bytes, _ := os.ReadFile(filefullpath)
		responseContent = bytes
	}

	return responseHandler(c, responseContent, filename)
}
