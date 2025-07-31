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

// PipProxy pip 패키지 매니저 프록시 핸들러
func PipProxy(c *fiber.Ctx) error {
	// 요청 컨텍스트 생성 (45초 타임아웃)
	ctx, cancel := context.WithTimeout(c.Context(), 45*time.Second)
	defer cancel()

	log.Printf("Access proxy pip\n")

	requestPath := c.Params("*")
	log.Printf("PIP request path: %s\n", requestPath)

	// 설정 읽기
	storageDir := helpers.GetStorageDir()
	globalConfig := config.GlobalConfig{}
	if err := globalConfig.ReadConfig(); err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to read global config")
	}
	config := config.PipProxySettings{}
	if err := config.ReadConfig(); err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to read PIP config")
	}

	// 미들웨어에서 전달된 캐시 정보 확인
	cacheHit, ok := c.Locals("cache_hit").(bool)
	if !ok {
		cacheHit = false
	}
	cachePath, ok := c.Locals("cache_path").(string)
	if !ok {
		cachePath = ""
	}

	var filefullpath string
	var filename string

	// 캐시 경로가 있으면 사용, 없으면 기본 경로 생성
	if cachePath != "" {
		filefullpath = cachePath
		filename = filepath.Base(filefullpath)
	} else {
		baseDir := filepath.Join(storageDir, config.Path)
		var err error
		filefullpath, err = security.SafeJoinPath(baseDir, requestPath)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid path")
		}
		filename = filepath.Base(filefullpath)
	}

	// 캐시가 히트하지 않았을 때만 다운로드
	if !cacheHit {
		dirpath := filepath.Dir(filefullpath)
		if err := os.MkdirAll(dirpath, 0o766); err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString("Failed to create directory")
		}

		// HTTP 클라이언트 생성 (프록시 최적화 설정)
		proxyClient := httpclient.NewProxyClient()

		// PyPI API 요청 처리
		var responseContent []byte

		for i, server := range config.Proxies {
			log.Printf("Trying pip proxy server %d: %s\n", i, server.Name)

			// PyPI URL 구성
			fullURL := buildPipURL(server.URL, requestPath)

			// 컨텍스트 기반 요청 (재시도 포함)
			resp, err := proxyClient.GetWithRetry(ctx, fullURL, 2)
			if err != nil {
				log.Printf("Error fetching from proxy %s: %v", server.Name, err)
				continue
			}
			defer func() { _ = resp.Body.Close() }()

			if resp.StatusCode == http.StatusOK {
				bytes, err := io.ReadAll(resp.Body)
				if err != nil {
					log.Printf("Error reading response body: %v", err)
					continue
				}

				// 파일 저장
				err = os.WriteFile(filefullpath, bytes, 0o766)
				if err != nil {
					log.Printf("Error writing file: %v", err)
					return c.Status(fiber.StatusInternalServerError).SendString("Error writing file")
				}

				responseContent = bytes
				log.Printf("Successfully fetched from %s\n", server.Name)
				break
			}
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
		return mimeApplicationZip
	case strings.HasSuffix(filename, ".tar.gz"):
		return "application/x-gzip"
	case strings.HasSuffix(filename, ".tar.bz2"):
		return "application/x-bzip2"
	case strings.HasSuffix(filename, ".zip"):
		return mimeApplicationZip
	case strings.HasSuffix(filename, ".egg"):
		return mimeApplicationZip
	default:
		return mimeApplicationOctetStream
	}
}
