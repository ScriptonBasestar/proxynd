package proxy

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"proxynd/configs"
	"proxynd/helpers"
	"proxynd/internal/security"
	"proxynd/pkg/httpclient"
)

// AptProxyUnified 통합 라우터용 APT 프록시 핸들러
func AptProxyUnified(c *fiber.Ctx) error {
	// 요청 컨텍스트 생성 (30초 타임아웃)
	ctx, cancel := context.WithTimeout(c.Context(), 30*time.Second)
	defer cancel()

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
	filefullpath := security.SafeJoinPath(storageDir, config.Path, requestPath)
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

		// HTTP 클라이언트 생성 (프록시 최적화 설정)
		proxyClient := httpclient.NewProxyClient()

		for s, server := range proxy {
			fmt.Printf("for moon %d\n", s)
			fullURL := helpers.JoinURL(server.URL, requestPath)

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
