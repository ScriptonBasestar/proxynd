package proxy

import (
	"context"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"proxynd/helpers"
	"proxynd/internal/config"
	"proxynd/internal/security"
	"proxynd/pkg/httpclient"
)

// YumProxyHandler yum 프록시 요청 처리 핸들러
func YumProxyHandler(c *fiber.Ctx) error {
	// 요청 컨텍스트 생성 (60초 타임아웃 - YUM은 큰 파일이 많음)
	ctx, cancel := context.WithTimeout(c.Context(), 60*time.Second)
	defer cancel()

	requestPath := c.Params("*")
	log.Printf("Access proxy yum: %s\n", requestPath)

	// 설정 읽기
	storageDir := helpers.GetStorageDir()
	globalConfig := config.GlobalConfig{}
	if err := globalConfig.ReadConfig(); err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to read global config")
	}
	yumConfig := config.YumProxySettings{}
	if err := yumConfig.ReadConfig(); err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to read YUM config")
	}

	// 파일 경로 생성
	baseDir := filepath.Join(storageDir, yumConfig.Path)
	filefullpath, err := security.SafeJoinPath(baseDir, requestPath)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid path")
	}
	filename := filepath.Base(filefullpath)

	// 캐시 확인
	if yumConfig.UseCache && helpers.FileExists(filefullpath) {
		log.Printf("Serving from cache: %s\n", filefullpath)

		// Content-Type 설정
		contentType := getYumContentType(requestPath)
		c.Set("Content-Type", contentType)

		return c.SendFile(filefullpath)
	}

	// 캐시에 없으면 업스트림에서 가져오기
	if _, err := os.Stat(filefullpath); os.IsNotExist(err) {
		dirpath := filepath.Dir(filefullpath)
		if err := os.MkdirAll(dirpath, os.ModePerm); err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString("Failed to create directory")
		}
		out, err := os.Create(filefullpath)
		if err != nil {
			log.Printf("Error creating file: %v", err)
			return c.Status(fiber.StatusInternalServerError).SendString("Error creating file")
		}
		defer func() { _ = out.Close() }()

		// HTTP 클라이언트 생성 (프록시 최적화 설정)
		proxyClient := httpclient.NewProxyClient()

		// 업스트림 서버에서 파일 가져오기
		for _, proxy := range yumConfig.Proxies {
			fullURL := helpers.JoinURL(proxy.URL, requestPath)
			log.Printf("Fetching from upstream %s: %s\n", proxy.Name, fullURL)

			// 컨텍스트 기반 요청 (재시도 포함)
			resp, err := proxyClient.GetWithRetry(ctx, fullURL, 2)
			if err != nil {
				log.Printf("Error fetching from proxy %s: %v\n", proxy.Name, err)
				continue
			}
			defer func() { _ = resp.Body.Close() }()

			if resp.StatusCode == http.StatusOK {
				// 파일 저장
				_, err = io.Copy(out, resp.Body)
				if err != nil {
					log.Printf("Error copying file: %v", err)
					return c.Status(fiber.StatusInternalServerError).SendString("Error copying file")
				}
				break
			}
			log.Printf("Upstream %s returned status: %d\n", proxy.Name, resp.StatusCode)
		}
	}

	// Content-Type 설정
	contentType := getYumContentType(requestPath)
	c.Set("Content-Type", contentType)

	// Content-Disposition 설정
	if isYumInlineFile(filename) {
		c.Set("Content-Disposition", "inline; filename="+filename)
	} else {
		c.Set("Content-Disposition", "attachment; filename="+filename)
	}

	return c.SendFile(filefullpath)
}

// getYumContentType 파일 경로에 따른 Content-Type 반환
func getYumContentType(filename string) string {
	switch {
	case strings.HasSuffix(filename, ".rpm"):
		return "application/x-rpm"
	case strings.HasSuffix(filename, ".xml") || strings.HasSuffix(filename, ".xml.gz"):
		return MimeApplicationXML
	case strings.HasSuffix(filename, ".xml.bz2") || strings.HasSuffix(filename, ".xml.xz"):
		return MimeApplicationXML
	case strings.HasSuffix(filename, ".sqlite") || strings.HasSuffix(filename, ".sqlite.bz2"):
		return MimeApplicationOctetStream
	case strings.HasSuffix(filename, ".sqlite.gz") || strings.HasSuffix(filename, ".sqlite.xz"):
		return MimeApplicationOctetStream
	case strings.HasSuffix(filename, ".asc") || strings.HasSuffix(filename, ".gpg"):
		return "application/pgp-signature"
	case strings.Contains(filename, "repomd.xml"):
		return "text/xml"
	default:
		return MimeApplicationOctetStream
	}
}

// isYumInlineFile 인라인으로 표시할 파일 확인
func isYumInlineFile(filename string) bool {
	inlineExtensions := []string{".xml", ".txt", ".asc", ".gpg"}
	for _, ext := range inlineExtensions {
		if strings.HasSuffix(filename, ext) {
			return true
		}
	}
	return false
}
