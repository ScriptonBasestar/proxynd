package proxy

import (
	"context"
	"encoding/json"
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

// NpmProxy npm 패키지 매니저 프록시 핸들러
func NpmProxy(c *fiber.Ctx) error {
	// 요청 컨텍스트 생성 (45초 타임아웃)
	ctx, cancel := context.WithTimeout(c.Context(), 45*time.Second)
	defer cancel()

	log.Printf("Access proxy npm\n")

	requestPath := c.Params("*")
	log.Printf("NPM request path: %s\n", requestPath)

	// 설정 읽기
	storageDir := helpers.GetStorageDir()
	globalConfig := config.GlobalConfig{}
	if err := globalConfig.ReadConfig(); err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to read global config")
	}
	config := config.NpmProxySettings{}
	if err := config.ReadConfig(); err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to read NPM config")
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

	// 캐시 경로가 있으면 사용, 없으면 기본 경로 생성 (보안 검증)
	if cachePath != "" {
		filefullpath = cachePath
		filename = filepath.Base(filefullpath)
	} else {
		baseDir := filepath.Join(storageDir, config.Path)
		var err error
		filefullpath, err = security.SafeJoinPath(baseDir, requestPath)
		if err != nil {
			log.Printf("Invalid path detected: %v", err)
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid path",
			})
		}
		filename = filepath.Base(filefullpath)
	}

	// NPM 레지스트리 메타데이터 요청인지 확인
	isMetadata := isNpmMetadataRequest(requestPath)

	// 캐시가 히트하지 않았을 때만 다운로드
	if !cacheHit {
		dirpath := filepath.Dir(filefullpath)
		if err := os.MkdirAll(dirpath, 0o766); err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString("Failed to create directory")
		}

		// HTTP 클라이언트 생성 (프록시 최적화 설정)
		proxyClient := httpclient.NewProxyClient()

		// NPM 레지스트리에서 데이터 가져오기
		var responseContent []byte
		var contentType string

		// 설정된 프록시 서버들을 순회하며 시도
		proxies := config.Proxies["default"]
		if proxies == nil {
			return c.Status(fiber.StatusInternalServerError).SendString("No NPM proxy servers configured")
		}

		for i, server := range proxies {
			log.Printf("Trying npm proxy server %d: %s\n", i, server.Name)

			fullURL := helpers.JoinURL(server.URL, requestPath)

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
				contentType = resp.Header.Get("Content-Type")

				// NPM 메타데이터는 URL 재작성이 필요할 수 있음
				if isMetadata && strings.Contains(contentType, "json") {
					bytes = rewriteNpmMetadata(bytes, c.BaseURL(), requestPath)
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
	contentType := getNpmContentType(filename, requestPath)
	c.Set("Content-Type", contentType)

	// JSON 응답은 inline, 패키지 파일은 attachment
	if strings.Contains(contentType, "json") {
		c.Set("Content-Disposition", "inline")
	} else {
		c.Set("Content-Disposition", "attachment; filename="+filename)
	}

	return c.Send(bytes)
}

// isNpmMetadataRequest NPM 메타데이터 요청인지 확인
func isNpmMetadataRequest(path string) bool {
	// 패키지 메타데이터는 패키지명만 있거나 @scope/package 형태
	parts := strings.Split(path, "/")

	// .tgz, .tar.gz 등 패키지 파일이 아닌 경우
	if strings.Contains(path, ".tgz") || strings.Contains(path, ".tar.gz") {
		return false
	}

	// /-/ 를 포함하는 특수 경로가 아닌 경우
	if strings.Contains(path, "/-/") {
		return false
	}

	// 일반 패키지명 또는 스코프 패키지명
	return len(parts) == 1 || (len(parts) == 2 && strings.HasPrefix(parts[0], "@"))
}

// rewriteNpmMetadata NPM 메타데이터의 URL을 프록시 서버로 재작성
func rewriteNpmMetadata(data []byte, baseURL, _ string) []byte {
	var metadata map[string]interface{}
	if err := json.Unmarshal(data, &metadata); err != nil {
		return data
	}

	// tarball URL 재작성
	if versions, ok := metadata["versions"].(map[string]interface{}); ok {
		for _, versionData := range versions {
			if version, ok := versionData.(map[string]interface{}); ok {
				if dist, ok := version["dist"].(map[string]interface{}); ok {
					if tarball, ok := dist["tarball"].(string); ok {
						// 원본 tarball URL을 프록시 URL로 변경
						proxyURL := baseURL + "/proxy/npm/" + extractPackagePath(tarball)
						dist["tarball"] = proxyURL
					}
				}
			}
		}
	}

	rewritten, err := json.Marshal(metadata)
	if err != nil {
		// 에러 발생 시 원본 데이터 반환
		return data
	}
	return rewritten
}

// extractPackagePath tarball URL에서 패키지 경로 추출
func extractPackagePath(tarballURL string) string {
	// https://registry.npmjs.org/package/-/package-1.0.0.tgz 형태에서 패키지 경로 추출
	parts := strings.Split(tarballURL, "registry.npmjs.org/")
	if len(parts) > 1 {
		return parts[1]
	}
	return tarballURL
}

// getNpmContentType npm 파일 타입에 따른 Content-Type 반환
func getNpmContentType(filename, path string) string {
	// 메타데이터
	if isNpmMetadataRequest(path) {
		return "application/json; charset=utf-8"
	}

	// 패키지 파일
	switch {
	case strings.HasSuffix(filename, ".tgz"):
		return MimeApplicationXGzip
	case strings.HasSuffix(filename, ".tar.gz"):
		return MimeApplicationXGzip
	case strings.HasSuffix(filename, ".json"):
		return MimeApplicationJSON
	default:
		return MimeApplicationOctetStream
	}
}
