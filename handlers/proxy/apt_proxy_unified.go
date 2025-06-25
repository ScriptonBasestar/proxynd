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

// AptProxyUnified 통합 라우터용 APT 프록시 핸들러
func AptProxyUnified(c *fiber.Ctx) error {
	log.Printf("Access proxy apt (unified)\n")

	// 전체 경로에서 osType과 실제 요청 경로 분리
	fullPath := c.Params("*")
	pathParts := strings.SplitN(fullPath, "/", 2)
	
	if len(pathParts) < 2 {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid APT proxy path format. Expected: /proxy/apt/{osType}/{path}")
	}
	
	osType := pathParts[0]
	requestPath := pathParts[1]
	
	log.Printf("APT proxy - osType: %s, path: %s\n", osType, requestPath)

	// fixme di
	storageDir := helpers.GetStorageDir()
	globalConfig := configs.GlobalConfig{}
	globalConfig.ReadConfig()
	config := configs.AptProxyConfig{}
	config.ReadConfig()

	// Create the file
	filefullpath := path.Join(storageDir, config.Path, requestPath)
	filename := filepath.Base(filefullpath)
	if _, err := os.Stat(filefullpath); os.IsNotExist(err) {
		dirpath := filepath.Dir(filefullpath)
		os.MkdirAll(dirpath, os.ModePerm)
		out, err := os.Create(filefullpath)
		if err != nil {
			log.Printf("Error creating file: %v", err)
			return c.Status(fiber.StatusInternalServerError).SendString("Error creating file")
		}
		defer out.Close()

		// Get the data
		proxy := config.Proxies[osType]
		if proxy == nil {
			return c.Status(fiber.StatusNotFound).SendString(fmt.Sprintf("No proxy configuration found for OS type: %s", osType))
		}
		
		for s, server := range proxy {
			fmt.Printf("for moon %d\n", s)
			resp, err := http.Get(helpers.JoinURL(server.URL, requestPath))
			if err != nil {
				log.Printf("Error fetching from proxy: %v", err)
				continue
			}
			//fmt.Println(resp.Header)
			fmt.Println(resp.StatusCode)
			// Writer the body to file
			_, err = io.Copy(out, resp.Body)
			if err != nil {
				log.Printf("Error copying file: %v", err)
				return c.Status(fiber.StatusInternalServerError).SendString("Error copying file")
			}
			resp.Body.Close()
			break
		}
	}

	c.Set("Content-Description", "File Transfer")
	c.Set("Content-Transfer-Encoding", "binary")
	c.Set("Content-Disposition", "attachment; filename="+filename)
	c.Set("Content-Type", "application/octet-stream")
	return c.SendFile(filefullpath)
}