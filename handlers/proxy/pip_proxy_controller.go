package proxy

import (
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

// PipProxy pip 패키지 매니저 프록시 핸들러
func PipProxy(c *fiber.Ctx) error {
	log.Printf("Access proxy pip\n")
	
	requestPath := c.Params("*")
	log.Printf("PIP request path: %s\n", requestPath)
	
	// 설정 읽기
	storageDir := helpers.GetStorageDir()
	globalConfig := configs.GlobalConfig{}
	globalConfig.ReadConfig()
	config := configs.PipProxyConfig{}
	config.ReadConfig()
	
	// 미들웨어에서 전달된 캐시 정보 확인
	cacheHit, _ := c.Locals("cache_hit").(bool)
	cachePath, _ := c.Locals("cache_path").(string)
	
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
		
		// PyPI API 요청 처리
		var responseContent []byte
		
		for i, server := range config.Proxies {
			log.Printf("Trying pip proxy server %d: %s\n", i, server.Name)
			
			// PyPI URL 구성
			url := buildPipURL(server.URL, requestPath)
			resp, err := http.Get(url)
			if err != nil {
				log.Printf("Error fetching from proxy %s: %v", server.Name, err)
				continue
			}
			
			if resp.StatusCode == http.StatusOK {
				bytes, _ := io.ReadAll(resp.Body)
				
				// 파일 저장
				err = os.WriteFile(filefullpath, bytes, 0766)
				if err != nil {
					log.Printf("Error writing file: %v", err)
					resp.Body.Close()
					return c.Status(fiber.StatusInternalServerError).SendString("Error writing file")
				}
				
				responseContent = bytes
				resp.Body.Close()
				log.Printf("Successfully fetched from %s\n", server.Name)
				break
			}
			
			resp.Body.Close()
		}
		
		if responseContent == nil {
			return c.Status(fiber.StatusNotFound).SendString("Package not found in any proxy")
		}
	}
	
	// 파일 읽기
	bytes, err := os.ReadFile(filefullpath)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Error reading cached file")
	}
	
	// Content-Type 설정
	contentType := getPipContentType(filename, requestPath)
	c.Set("Content-Type", contentType)
	
	// JSON API 응답은 inline, 패키지 파일은 attachment
	if strings.Contains(contentType, "json") || strings.Contains(contentType, "html") {
		c.Set("Content-Disposition", "inline; filename="+filename)
	} else {
		c.Set("Content-Disposition", "attachment; filename="+filename)
	}
	
	return c.Send(bytes)
}

// buildPipURL PyPI URL 구성
func buildPipURL(serverURL, requestPath string) string {
	// /simple/ 경로 처리
	if strings.HasPrefix(requestPath, "simple/") {
		return helpers.JoinURL(serverURL, requestPath)
	}
	
	// /packages/ 경로 처리
	if strings.HasPrefix(requestPath, "packages/") {
		return helpers.JoinURL(serverURL, requestPath)
	}
	
	// 기본적으로 simple API 사용
	return helpers.JoinURL(serverURL, "simple", requestPath)
}

// getPipContentType pip 파일 타입에 따른 Content-Type 반환
func getPipContentType(filename, path string) string {
	// API 응답
	if strings.Contains(path, "/simple/") && !strings.Contains(filename, ".") {
		return "text/html; charset=utf-8"
	}
	
	// JSON API
	if strings.Contains(path, "/json") {
		return "application/json"
	}
	
	// 패키지 파일
	switch {
	case strings.HasSuffix(filename, ".whl"):
		return "application/zip"
	case strings.HasSuffix(filename, ".tar.gz"):
		return "application/x-gzip"
	case strings.HasSuffix(filename, ".tar.bz2"):
		return "application/x-bzip2"
	case strings.HasSuffix(filename, ".zip"):
		return "application/zip"
	case strings.HasSuffix(filename, ".egg"):
		return "application/zip"
	default:
		return "application/octet-stream"
	}
}