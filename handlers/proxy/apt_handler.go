// Package proxy provides HTTP handlers for various package manager proxies.
// It includes handlers for APT, Maven, NPM, Docker, and other package managers.
package proxy

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"

	"github.com/gofiber/fiber/v2"

	"proxynd/configs"
	"proxynd/helpers"
)

// AptProxy handles APT package manager proxy requests.
// It caches Debian/Ubuntu packages and metadata locally to improve download speeds.
// The handler supports multiple upstream mirrors with failover capability.
func AptProxy(c *fiber.Ctx) error {
	log.Printf("Access proxy apt\n")

	pathOs := c.Params("osType")
	requestPath := c.Params("*")

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
		proxy := config.Proxies[pathOs]
		for s, server := range proxy {
			fmt.Printf("for moon %d\n", s)
			resp, err := http.Get(helpers.JoinURL(server.URL, requestPath))
			if err != nil {
				log.Printf("Error fetching from proxy: %v", err)
				continue
			}
			// Ensure response body is always closed
			defer resp.Body.Close()

			//fmt.Println(resp.Header)
			fmt.Println(resp.StatusCode)

			// Check status code before processing
			if resp.StatusCode != http.StatusOK {
				log.Printf("Error response from proxy: %d", resp.StatusCode)
				continue
			}

			// Writer the body to file
			_, err = io.Copy(out, resp.Body)
			if err != nil {
				log.Printf("Error copying file: %v", err)
				return c.Status(fiber.StatusInternalServerError).SendString("Error copying file")
			}
			break
		}
	}

	// APT 파일 타입에 따른 적절한 Content-Type 설정
	contentType := getAptContentType(filename)
	c.Set("Content-Type", contentType)

	// 특정 파일은 inline으로 전송
	if isInlineFile(filename) {
		c.Set("Content-Disposition", "inline; filename="+filename)
	} else {
		c.Set("Content-Disposition", "attachment; filename="+filename)
	}

	return c.SendFile(filefullpath)
}
