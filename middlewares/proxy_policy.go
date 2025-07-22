package middlewares

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/gofiber/fiber/v2"

	"proxynd/configs"
	"proxynd/helpers"
	"proxynd/internal/security"
)

// ProxyPolicyMiddleware 프록시 정책 처리 미들웨어
// 캐시 hit/miss 판단, 인증/허가 체크, 요청 허용/차단 정책 적용
func ProxyPolicyMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// 프록시 타입과 경로 추출
		proxyType := c.Params("type")
		requestPath := c.Params("*")

		log.Printf("ProxyPolicy middleware - type: %s, path: %s\n", proxyType, requestPath)

		// 1. 요청 허용/차단 정책 체크
		if !isProxyTypeAllowed(proxyType) {
			return c.Status(fiber.StatusForbidden).SendString("Proxy type not allowed: " + proxyType)
		}

		// 2. 인증/허가 체크 (현재는 기본 구현)
		if !isAuthenticated(c) {
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
	allowedTypes := []string{"apt", "maven", "npm", "pip", "docker"}
	for _, allowed := range allowedTypes {
		if proxyType == allowed {
			return true
		}
	}
	return false
}

// isAuthenticated 인증 여부 확인 (기본 구현)
func isAuthenticated(_ *fiber.Ctx) bool {
	// TODO: 실제 인증 로직 구현
	// 현재는 모든 요청을 허용
	return true
}

// checkCache 캐시 존재 여부 확인
func checkCache(proxyType, requestPath string) CacheInfo {
	storageDir := helpers.GetStorageDir()

	// 프록시 타입별 설정 읽기
	var cachePath string
	switch proxyType {
	case "maven":
		config := configs.MavenProxyConfig{}
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
		config := configs.AptProxyConfig{}
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
		config := configs.NpmProxyConfig{}
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
		config := configs.PipProxyConfig{}
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
		config := configs.DockerProxyConfig{}
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
