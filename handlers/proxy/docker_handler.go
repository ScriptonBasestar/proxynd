package proxy

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gofiber/fiber/v2"

	"proxynd/configs"
	"proxynd/helpers"
	"proxynd/internal/security"
)

// DockerProxy Docker 레지스트리 프록시 핸들러
func DockerProxy(c *fiber.Ctx) error {
	log.Printf("Access proxy docker\n")

	requestPath := c.Params("*")
	log.Printf("Docker request path: %s\n", requestPath)

	// Docker Registry v2 API 라우팅
	if requestPath == "v2" || requestPath == "v2/" {
		return handleDockerV2Base(c)
	}

	// 설정 읽기
	storageDir := helpers.GetStorageDir()
	config := configs.DockerProxyConfig{}
	if err := config.ReadConfig(); err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to read Docker config")
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

	// 캐시 경로가 있으면 사용, 없으면 기본 경로 생성 (보안 검증)
	if cachePath != "" {
		filefullpath = cachePath
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
	}

	// 매니페스트 요청인지 확인
	isManifest := strings.Contains(requestPath, "/manifests/")
	isBlob := strings.Contains(requestPath, "/blobs/")

	// 캐시가 히트하지 않았을 때만 다운로드
	if !cacheHit {
		dirpath := filepath.Dir(filefullpath)
		if err := os.MkdirAll(dirpath, 0766); err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString("Failed to create directory")
		}

		// Docker 레지스트리에서 데이터 가져오기
		var responseContent []byte
		var headers http.Header

		for i, server := range config.Proxies {
			log.Printf("Trying docker proxy server %d: %s\n", i, server.Name)

			// Docker Registry URL 구성
			url := buildDockerURL(server.URL, requestPath)

			// HTTP 요청 생성
			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				log.Printf("Error creating request: %v", err)
				continue
			}

			// Docker 레지스트리 헤더 복사
			copyDockerHeaders(c, req)

			// 인증 처리
			if server.Auth.Username != "" && server.Auth.Password != "" {
				req.SetBasicAuth(server.Auth.Username, server.Auth.Password)
			}

			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				log.Printf("Error fetching from proxy %s: %v", server.Name, err)
				continue
			}

			if resp.StatusCode == http.StatusOK {
				bytes, err := io.ReadAll(resp.Body)
				if err != nil {
					_ = resp.Body.Close()
					continue
				}
				headers = resp.Header

				// 파일 저장
				err = os.WriteFile(filefullpath, bytes, 0766)
				if err != nil {
					log.Printf("Error writing file: %v", err)
					_ = resp.Body.Close()
					return c.Status(fiber.StatusInternalServerError).SendString("Error writing file")
				}

				// 헤더 정보도 저장 (매니페스트의 경우)
				if isManifest {
					if err := saveDockerHeaders(filefullpath+".headers", headers); err != nil {
						log.Printf("Warning: Failed to save Docker headers: %v", err)
					}
				}

				responseContent = bytes
				_ = resp.Body.Close()
				log.Printf("Successfully fetched from %s\n", server.Name)
				break
			} else if resp.StatusCode == http.StatusUnauthorized {
				// 인증 챌린지 반환
				copyResponseHeaders(resp.Header, c)
				_ = resp.Body.Close()
				return c.Status(resp.StatusCode).Send(nil)
			}

			_ = resp.Body.Close()
		}

		if responseContent == nil {
			return c.Status(fiber.StatusNotFound).SendString("Resource not found in any proxy")
		}
	}

	// 파일 읽기
	bytes, err := os.ReadFile(filefullpath)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Error reading cached file")
	}

	// 매니페스트의 경우 저장된 헤더도 읽기
	if isManifest {
		if headers, err := loadDockerHeaders(filefullpath + ".headers"); err == nil {
			copyResponseHeaders(headers, c)
		}
	}

	// Content-Type 설정
	if isManifest {
		// 매니페스트 타입 자동 감지
		contentType := detectManifestType(bytes)
		c.Set("Content-Type", contentType)
	} else if isBlob {
		c.Set("Content-Type", "application/octet-stream")
	}

	// Docker-Content-Digest 헤더 설정
	if digest := extractDigest(requestPath); digest != "" {
		c.Set("Docker-Content-Digest", digest)
	}

	return c.Send(bytes)
}

// handleDockerV2Base Docker Registry v2 베이스 엔드포인트 처리
func handleDockerV2Base(c *fiber.Ctx) error {
	// Docker Registry v2 API 응답
	response := map[string]interface{}{
		"errors": []interface{}{},
	}

	c.Set("Docker-Distribution-Api-Version", "registry/2.0")
	return c.JSON(response)
}

// buildDockerURL Docker Registry URL 구성
func buildDockerURL(serverURL, requestPath string) string {
	// v2 prefix 추가
	if !strings.HasPrefix(requestPath, "v2/") {
		requestPath = "v2/" + requestPath
	}
	return helpers.JoinURL(serverURL, requestPath)
}

// copyDockerHeaders 클라이언트 요청 헤더를 프록시 요청으로 복사
func copyDockerHeaders(c *fiber.Ctx, req *http.Request) {
	// Accept 헤더
	if accept := c.Get("Accept"); accept != "" {
		req.Header.Set("Accept", accept)
	}

	// Docker 관련 헤더들
	dockerHeaders := []string{
		"Docker-Distribution-Api-Version",
		"Authorization",
		"User-Agent",
	}

	for _, header := range dockerHeaders {
		if value := c.Get(header); value != "" {
			req.Header.Set(header, value)
		}
	}
}

// copyResponseHeaders 응답 헤더 복사
func copyResponseHeaders(from http.Header, to *fiber.Ctx) {
	for key, values := range from {
		if len(values) > 0 {
			to.Set(key, values[0])
		}
	}
}

// detectManifestType 매니페스트 타입 감지
func detectManifestType(data []byte) string {
	var manifest map[string]interface{}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return "application/vnd.docker.distribution.manifest.v2+json"
	}

	if schemaVersion, ok := manifest["schemaVersion"].(float64); ok {
		if schemaVersion == 1 {
			return "application/vnd.docker.distribution.manifest.v1+json"
		}

		// v2 매니페스트 타입 구분
		if _, ok := manifest["manifests"]; ok {
			return "application/vnd.docker.distribution.manifest.list.v2+json"
		}
	}

	return "application/vnd.docker.distribution.manifest.v2+json"
}

// extractDigest 경로에서 다이제스트 추출
func extractDigest(path string) string {
	parts := strings.Split(path, "/")
	for _, part := range parts {
		if strings.HasPrefix(part, "sha256:") {
			return part
		}
	}
	return ""
}

// saveDockerHeaders 헤더 정보를 파일로 저장
func saveDockerHeaders(filename string, headers http.Header) error {
	headerMap := make(map[string][]string)
	for key, values := range headers {
		headerMap[key] = values
	}

	data, err := json.Marshal(headerMap)
	if err != nil {
		return err
	}

	return os.WriteFile(filename, data, 0666)
}

// loadDockerHeaders 저장된 헤더 정보 로드
func loadDockerHeaders(filename string) (http.Header, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var headerMap map[string][]string
	if err := json.Unmarshal(data, &headerMap); err != nil {
		return nil, err
	}

	headers := make(http.Header)
	for key, values := range headerMap {
		for _, value := range values {
			headers.Add(key, value)
		}
	}

	return headers, nil
}
