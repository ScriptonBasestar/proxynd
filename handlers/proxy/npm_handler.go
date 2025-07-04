package proxy

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/gofiber/fiber/v2"

	"proxynd/configs"
	"proxynd/helpers"
)

// NpmProxy npm 패키지 매니저 프록시 핸들러
func NpmProxy(c *fiber.Ctx) error {
	log.Printf("Access proxy npm\n")

	requestPath := c.Params("*")
	log.Printf("NPM request path: %s\n", requestPath)

	// 설정 읽기
	storageDir := helpers.GetStorageDir()
	globalConfig := configs.GlobalConfig{}
	globalConfig.ReadConfig()
	config := configs.NpmProxyConfig{}
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

	// NPM 레지스트리 메타데이터 요청인지 확인
	isMetadata := isNpmMetadataRequest(requestPath)

	// 캐시가 히트하지 않았을 때만 다운로드
	if !cacheHit {
		dirpath := filepath.Dir(filefullpath)
		os.MkdirAll(dirpath, 0766)

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

			url := helpers.JoinURL(server.URL, requestPath)
			resp, err := http.Get(url)
			if err != nil {
				log.Printf("Error fetching from proxy %s: %v", server.Name, err)
				continue
			}

			if resp.StatusCode == http.StatusOK {
				bytes, _ := io.ReadAll(resp.Body)
				contentType = resp.Header.Get("Content-Type")

				// NPM 메타데이터는 URL 재작성이 필요할 수 있음
				if isMetadata && strings.Contains(contentType, "json") {
					bytes = rewriteNpmMetadata(bytes, c.BaseURL(), requestPath)
				}

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
func rewriteNpmMetadata(data []byte, baseURL, requestPath string) []byte {
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

	rewritten, _ := json.Marshal(metadata)
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
		return "application/x-gzip"
	case strings.HasSuffix(filename, ".tar.gz"):
		return "application/x-gzip"
	case strings.HasSuffix(filename, ".json"):
		return "application/json"
	default:
		return "application/octet-stream"
	}
}
