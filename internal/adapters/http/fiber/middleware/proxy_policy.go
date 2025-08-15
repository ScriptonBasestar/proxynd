package middlewares

import (
	"encoding/base64"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/gofiber/fiber/v2"

	"proxynd/helpers"
	"proxynd/internal/config"
	"proxynd/internal/security"
)

// ProxyPolicyMiddleware 프록시 정책 처리 미들웨어
// 캐시 hit/miss 판단, 인증/허가 체크, 요청 허용/차단 정책 적용
func ProxyPolicyMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// 프록시 타입과 경로 추출
		proxyType := c.Params("type")
		requestPath := c.Params("*")

		log.Printf("ProxyPolicy middleware - original_url: %s, type: %s, path: %s, route: %s\n",
			c.OriginalURL(), proxyType, requestPath, c.Route().Path)

		// 1. 요청 허용/차단 정책 체크
		if !isProxyTypeAllowed(proxyType) {
			log.Printf("BLOCKED - Proxy type not allowed: '%s' (length: %d)\n", proxyType, len(proxyType))
			return c.Status(fiber.StatusForbidden).SendString("Proxy type not allowed: '" + proxyType + "'")
		}

		// 2. 인증/허가 체크 (현재는 기본 구현)
		if !isBasicAuthenticated(c) {
			return c.Status(fiber.StatusUnauthorized).SendString("Authentication required")
		}

		// 3. 캐시 hit/miss 판단
		cacheInfo := checkCache(proxyType, requestPath)
		c.Locals("cache_hit", cacheInfo.Hit)
		c.Locals("cache_path", cacheInfo.Path)

		if cacheInfo.Hit {
			log.Printf("Cache HIT for %s/%s\n", proxyType, requestPath)
		} else {
			log.Printf("Cache MISS for %s/%s\n", proxyType, requestPath)
		}

		// 다음 핸들러로 진행
		return c.Next()
	}
}

// CacheInfo 캐시 정보 구조체
type CacheInfo struct {
	Hit  bool
	Path string
}

// isProxyTypeAllowed 프록시 타입이 허용되는지 확인
func isProxyTypeAllowed(proxyType string) bool {
	allowedTypes := []string{"apt", "maven", "npm", "pip", "docker", "apk", "yum"}
	for _, allowed := range allowedTypes {
		if proxyType == allowed {
			return true
		}
	}
	return false
}

// isBasicAuthenticated 인증 여부 확인 (기본 구현)
func isBasicAuthenticated(c *fiber.Ctx) bool {
	// 1. API 키 인증 확인 (X-API-Key 헤더)
	if apiKey := c.Get("X-API-Key"); apiKey != "" {
		return validateAPIKey(apiKey)
	}

	// 2. Bearer 토큰 인증 확인 (Authorization 헤더)
	if authHeader := c.Get("Authorization"); authHeader != "" {
		if strings.HasPrefix(authHeader, "Bearer ") {
			token := strings.TrimPrefix(authHeader, "Bearer ")
			return validateBearerToken(token)
		}
	}

	// 3. Basic 인증 확인
	if authHeader := c.Get("Authorization"); authHeader != "" {
		if strings.HasPrefix(authHeader, "Basic ") {
			return validateBasicAuth(authHeader)
		}
	}

	// 4. 공개 접근 허용 정책 (설정에 따라)
	// 개발 환경이나 특정 엔드포인트에서는 인증 없이 허용 가능
	if isPublicAccessAllowed(c) {
		return true
	}

	// 기본적으로 인증되지 않은 요청은 차단
	log.Printf("Authentication failed for request: %s %s from %s",
		c.Method(), c.Path(), c.IP())
	return false
}

// validateAPIKey API 키 검증
func validateAPIKey(apiKey string) bool {
	// API 키 형식 기본 검증 (최소 길이, 문자 제한 등)
	if len(apiKey) < 16 || len(apiKey) > 128 {
		log.Printf("Invalid API key length: %d", len(apiKey))
		return false
	}

	// TODO: API 키 매니저를 통한 실제 검증 로직 구현
	// 현재는 기본 검증만 수행
	if strings.HasPrefix(apiKey, "px_") || strings.HasPrefix(apiKey, "proxynd_") {
		log.Printf("API key validation passed for key: %s...", apiKey[:8])
		return true
	}

	log.Printf("Invalid API key format")
	return false
}

// validateBearerToken Bearer 토큰 검증
func validateBearerToken(token string) bool {
	// JWT 토큰 형식 기본 검증
	if len(token) < 20 {
		log.Printf("Token too short")
		return false
	}

	// JWT 형식 확인 (3개 파트가 점으로 구분)
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		log.Printf("Invalid JWT format")
		return false
	}

	// TODO: JWT 서비스를 통한 실제 토큰 검증 구현
	// 현재는 기본 형식 검증만 수행
	log.Printf("Bearer token validation passed")
	return true
}

// validateBasicAuth Basic 인증 검증
func validateBasicAuth(authHeader string) bool {
	// Basic 인증 헤더에서 사용자 정보 추출
	encodedCredentials := strings.TrimPrefix(authHeader, "Basic ")

	// Base64 디코딩
	credentials, err := base64.StdEncoding.DecodeString(encodedCredentials)
	if err != nil {
		log.Printf("Failed to decode basic auth: %v", err)
		return false
	}

	// username:password 형식으로 분리
	parts := strings.SplitN(string(credentials), ":", 2)
	if len(parts) != 2 {
		log.Printf("Invalid basic auth format")
		return false
	}

	username, password := parts[0], parts[1]

	// TODO: 사용자 저장소를 통한 실제 인증 로직 구현
	// 현재는 기본 검증만 수행
	if len(username) > 0 && len(password) >= 8 {
		log.Printf("Basic auth validation passed for user: %s", username)
		return true
	}

	log.Printf("Basic auth validation failed")
	return false
}

// isPublicAccessAllowed 공개 접근 허용 여부 확인
func isPublicAccessAllowed(c *fiber.Ctx) bool {
	// 헬스체크 엔드포인트는 항상 공개
	if strings.HasPrefix(c.Path(), "/healthz") || strings.HasPrefix(c.Path(), "/health") {
		return true
	}

	// 메트릭 엔드포인트는 공개 (Prometheus 등)
	if strings.HasPrefix(c.Path(), "/metrics") {
		return true
	}

	// 개발 환경에서는 더 관대한 정책 적용
	if isDevelopmentMode() {
		log.Printf("Development mode: allowing public access to %s", c.Path())
		return true
	}

	return false
}

// isDevelopmentMode 개발 모드 확인
func isDevelopmentMode() bool {
	// 환경 변수나 설정을 통해 개발 모드 확인
	env := strings.ToLower(os.Getenv("PROXYND_ENV"))
	return env == "development" || env == "dev" || env == "local"
}

// checkCache 캐시 존재 여부 확인
func checkCache(proxyType, requestPath string) CacheInfo {
	storageDir := helpers.GetStorageDir()

	// 프록시 타입별 설정 읽기
	var cachePath string
	switch proxyType {
	case "maven":
		config := config.MavenProxySettings{}
		if err := config.ReadConfig(); err != nil {
			log.Printf("Warning: Failed to read Maven config, using defaults: %v", err)
		}
		baseDir := filepath.Join(storageDir, config.Path)
		var err error
		cachePath, err = security.SafeJoinPath(baseDir, requestPath)
		if err != nil {
			log.Printf("Invalid path in maven cache check: %v", err)
			return CacheInfo{Hit: false, Path: ""}
		}

	case "apt":
		config := config.AptProxyConfig{}
		if err := config.ReadConfig(); err != nil {
			log.Printf("Warning: Failed to read APT config, using defaults: %v", err)
		}
		baseDir := filepath.Join(storageDir, config.Path)
		// APT는 osType을 포함하므로 경로 처리가 다름
		pathParts := strings.SplitN(requestPath, "/", 2)
		var targetPath string
		if len(pathParts) >= 2 {
			targetPath = pathParts[1]
		} else {
			targetPath = requestPath
		}
		var err error
		cachePath, err = security.SafeJoinPath(baseDir, targetPath)
		if err != nil {
			log.Printf("Invalid path in apt cache check: %v", err)
			return CacheInfo{Hit: false, Path: ""}
		}

	case "npm":
		config := config.NpmProxySettings{}
		if err := config.ReadConfig(); err != nil {
			log.Printf("Warning: Failed to read NPM config, using defaults: %v", err)
		}
		baseDir := filepath.Join(storageDir, config.Path)
		var err error
		cachePath, err = security.SafeJoinPath(baseDir, requestPath)
		if err != nil {
			log.Printf("Invalid path in npm cache check: %v", err)
			return CacheInfo{Hit: false, Path: ""}
		}

	case "pip":
		config := config.PipProxySettings{}
		if err := config.ReadConfig(); err != nil {
			log.Printf("Warning: Failed to read PIP config, using defaults: %v", err)
		}
		baseDir := filepath.Join(storageDir, config.Path)
		var err error
		cachePath, err = security.SafeJoinPath(baseDir, requestPath)
		if err != nil {
			log.Printf("Invalid path in pip cache check: %v", err)
			return CacheInfo{Hit: false, Path: ""}
		}

	case "docker":
		config := config.DockerProxySettings{}
		if err := config.ReadConfig(); err != nil {
			log.Printf("Warning: Failed to read Docker config, using defaults: %v", err)
		}
		baseDir := filepath.Join(storageDir, config.Path)
		var err error
		cachePath, err = security.SafeJoinPath(baseDir, requestPath)
		if err != nil {
			log.Printf("Invalid path in docker cache check: %v", err)
			return CacheInfo{Hit: false, Path: ""}
		}

	case "apk":
		config := config.ApkProxySettings{}
		if err := config.ReadConfig(); err != nil {
			log.Printf("Warning: Failed to read APK config, using defaults: %v", err)
		}
		baseDir := filepath.Join(storageDir, config.Path)
		var err error
		cachePath, err = security.SafeJoinPath(baseDir, requestPath)
		if err != nil {
			log.Printf("Invalid path in apk cache check: %v", err)
			return CacheInfo{Hit: false, Path: ""}
		}

	case "yum":
		config := config.YumProxySettings{}
		if err := config.ReadConfig(); err != nil {
			log.Printf("Warning: Failed to read YUM config, using defaults: %v", err)
		}
		baseDir := filepath.Join(storageDir, config.Path)
		var err error
		cachePath, err = security.SafeJoinPath(baseDir, requestPath)
		if err != nil {
			log.Printf("Invalid path in yum cache check: %v", err)
			return CacheInfo{Hit: false, Path: ""}
		}

	default:
		baseDir := filepath.Join(storageDir, "proxy", proxyType)
		var err error
		cachePath, err = security.SafeJoinPath(baseDir, requestPath)
		if err != nil {
			log.Printf("Invalid path in default cache check: %v", err)
			return CacheInfo{Hit: false, Path: ""}
		}
	}

	// 파일 존재 여부 확인
	if _, err := os.Stat(cachePath); err == nil {
		return CacheInfo{
			Hit:  true,
			Path: cachePath,
		}
	}

	return CacheInfo{
		Hit:  false,
		Path: cachePath,
	}
}
