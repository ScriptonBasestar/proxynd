package proxy

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"proxynd/configs"
	"proxynd/helpers"
	"proxynd/pkg/httpclient"
)

func responseHandler(c *fiber.Ctx, responseContent []byte, filename string) error {
	ext := filename[strings.LastIndex(filename, ".")+1:]
	// text or octetstream
	if ext == "pom" || ext == "xml" {
		c.Set("Content-Type", "application/xml")
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
	// 요청 컨텍스트 생성 (60초 타임아웃 - Maven은 큰 파일이 많음)
	ctx, cancel := context.WithTimeout(c.Context(), 60*time.Second)
	defer cancel()

	log.Printf("Access proxy maven\n")

	requestPath := c.Params("*")

	// fixme di
	storageDir := helpers.GetStorageDir()
	globalConfig := configs.GlobalConfig{}
	globalConfig.ReadConfig()
	config := configs.MavenProxyConfig{}
	config.ReadConfig()

	// Check cache information from middleware
	cacheHit, _ := c.Locals("cache_hit").(bool)
	cachePath, _ := c.Locals("cache_path").(string)

	var responseContent []byte
	var filefullpath string
	var filename string

	// Use cache path if available, otherwise create default path
	if cachePath != "" {
		filefullpath = cachePath
		filename = filepath.Base(filefullpath)
	} else {
		filefullpath = path.Join(storageDir, config.Path, requestPath)
		filename = filepath.Base(filefullpath)
	}

	// Download only when cache miss
	if !cacheHit {
		dirpath := filepath.Dir(filefullpath)
		os.MkdirAll(dirpath, 0766)

		// HTTP 클라이언트 생성 (프록시 최적화 설정)
		proxyClient := httpclient.NewProxyClient()

		// Get the data
		fmt.Println(len(config.Proxies))

		for s, server := range config.Proxies {
			fmt.Printf("for moon %d\n", s)
			fullURL := helpers.JoinURL(server.Url, requestPath)

			// 컨텍스트 기반 요청 (재시도 포함)
			resp, err := proxyClient.GetWithRetry(ctx, fullURL, 2)
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

			// Read the body
			bytes, err := io.ReadAll(resp.Body)
			if err != nil {
				log.Printf("Error reading response body: %v", err)
				continue
			}

			// Write to file
			err = os.WriteFile(filefullpath, bytes, 0766)
			if err != nil {
				log.Printf("Error writing file: %v", err)
				return c.Status(fiber.StatusInternalServerError).SendString("Error writing file")
			}
			responseContent = bytes
			break
		}
	} else {
		bytes, _ := os.ReadFile(filefullpath)
		responseContent = bytes
	}

	return responseHandler(c, responseContent, filename)
}
