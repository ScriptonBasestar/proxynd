package proxy

import (
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"proxynd/configs"
	"proxynd/helpers"

	"github.com/gofiber/fiber/v2"
)

// YumProxyHandler yum 프록시 요청 처리 핸들러
func YumProxyHandler(c *fiber.Ctx) error {
	requestPath := c.Params("*")
	log.Printf("Access proxy yum: %s\n", requestPath)

	// 설정 읽기
	storageDir := helpers.GetStorageDir()
	globalConfig := configs.GlobalConfig{}
	globalConfig.ReadConfig()
	yumConfig := configs.YumProxyConfig{}
	yumConfig.ReadConfig()

	// 파일 경로 생성
	filefullpath := path.Join(storageDir, yumConfig.Path, requestPath)
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
		os.MkdirAll(dirpath, os.ModePerm)
		out, err := os.Create(filefullpath)
		if err != nil {
			log.Printf("Error creating file: %v", err)
			return c.Status(fiber.StatusInternalServerError).SendString("Error creating file")
		}
		defer out.Close()

		// 업스트림 서버에서 파일 가져오기
		for _, proxy := range yumConfig.Proxies {
			fullURL := helpers.JoinURL(proxy.Url, requestPath)
			log.Printf("Fetching from upstream %s: %s\n", proxy.Name, fullURL)

			resp, err := http.Get(fullURL)
			if err != nil {
				log.Printf("Error fetching from proxy %s: %v\n", proxy.Name, err)
				continue
			}

			if resp.StatusCode == http.StatusOK {
				// 파일 저장
				_, err = io.Copy(out, resp.Body)
				resp.Body.Close()
				if err != nil {
					log.Printf("Error copying file: %v", err)
					return c.Status(fiber.StatusInternalServerError).SendString("Error copying file")
				}
				break
			}
			resp.Body.Close()
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
		return "application/xml"
	case strings.HasSuffix(filename, ".xml.bz2") || strings.HasSuffix(filename, ".xml.xz"):
		return "application/xml"
	case strings.HasSuffix(filename, ".sqlite") || strings.HasSuffix(filename, ".sqlite.bz2"):
		return "application/octet-stream"
	case strings.HasSuffix(filename, ".sqlite.gz") || strings.HasSuffix(filename, ".sqlite.xz"):
		return "application/octet-stream"
	case strings.HasSuffix(filename, ".asc") || strings.HasSuffix(filename, ".gpg"):
		return "application/pgp-signature"
	case strings.Contains(filename, "repomd.xml"):
		return "text/xml"
	default:
		return "application/octet-stream"
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
