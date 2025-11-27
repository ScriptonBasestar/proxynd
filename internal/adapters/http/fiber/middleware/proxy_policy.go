package middlewares

import (
	"encoding/base64"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"

	"proxynd/internal/auth"
	"proxynd/internal/auth/jwt"
	"proxynd/internal/config"
	"proxynd/internal/helpers"
	"proxynd/internal/security"
)

// authCacheEntry 인증 결과 캐시 엔트리
type authCacheEntry struct {
	valid     bool
	expiresAt time.Time
	userInfo  map[string]interface{}
}

var (
	authCacheMu   sync.RWMutex
	authCacheData = make(map[string]*authCacheEntry)

	// 서비스 싱글톤
	apiKeyManagerOnce sync.Once
	apiKeyManagerInst *auth.APIKeyManager

	jwtServiceOnce sync.Once
	jwtServiceInst *jwt.JWTService
)

// getAPIKeyManagerInstance API Key Manager 싱글톤
func getAPIKeyManagerInstance() *auth.APIKeyManager {
	apiKeyManagerOnce.Do(func() {
		var err error
		apiKeyManagerInst, err = auth.NewAPIKeyManager(nil)
		if err != nil {
			log.Printf("Failed to initialize API key manager: %v", err)
		}
	})
	return apiKeyManagerInst
}

// getJWTServiceInstance JWT Service 싱글톤
func getJWTServiceInstance() *jwt.JWTService {
	jwtServiceOnce.Do(func() {
		oauth2Config := &config.OAuth2Config{}
		if err := oauth2Config.ReadConfig(); err != nil {
			log.Printf("Failed to load OAuth2 config for JWT service: %v", err)
			return
		}
		jwtServiceInst = jwt.NewJWTService(oauth2Config)
	})
	return jwtServiceInst
}

// getCachedAuth 캐시된 인증 결과 조회
func getCachedAuth(key string) *authCacheEntry {
	authCacheMu.RLock()
	defer authCacheMu.RUnlock()

	if entry, exists := authCacheData[key]; exists {
		if time.Now().Before(entry.expiresAt) {
			return entry
		}
	}
	return nil
}

// setCachedAuth 인증 결과 캐시 저장
func setCachedAuth(key string, valid bool, userInfo map[string]interface{}, ttl time.Duration) {
	authCacheMu.Lock()
	defer authCacheMu.Unlock()

	authCacheData[key] = &authCacheEntry{
		valid:     valid,
		expiresAt: time.Now().Add(ttl),
		userInfo:  userInfo,
	}

	// 캐시 크기 제한 (1000개 초과 시 오래된 항목 정리)
	if len(authCacheData) > 1000 {
		now := time.Now()
		for k, v := range authCacheData {
			if now.After(v.expiresAt) {
				delete(authCacheData, k)
			}
		}
	}
}

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

// validateAPIKey API 키 검증 - API Key Manager 통합
func validateAPIKey(apiKey string) bool {
	// 캐시 키 생성 (처음 8자만 사용)
	cacheKeyLen := len(apiKey)
	if cacheKeyLen > 8 {
		cacheKeyLen = 8
	}
	cacheKey := "apikey:" + apiKey[:cacheKeyLen]

	// 캐시 확인
	if cached := getCachedAuth(cacheKey); cached != nil {
		return cached.valid
	}

	// API Key Manager를 통한 실제 검증
	manager := getAPIKeyManagerInstance()
	if manager != nil {
		validatedKey, err := manager.ValidateAPIKey(apiKey)
		if err == nil && validatedKey != nil {
			log.Printf("API key validated via manager for user: %s", validatedKey.UserID)
			setCachedAuth(cacheKey, true, map[string]interface{}{
				"user_id":     validatedKey.UserID,
				"permissions": validatedKey.Permissions,
			}, 5*time.Minute)
			return true
		}
		if err != nil {
			log.Printf("API key validation failed via manager: %v", err)
		}
	}

	// 폴백: 레거시 형식 검증
	if validateAPIKeyFormat(apiKey) {
		log.Printf("API key passed format validation (legacy): %s...", apiKey[:cacheKeyLen])
		setCachedAuth(cacheKey, true, nil, 5*time.Minute)
		return true
	}

	setCachedAuth(cacheKey, false, nil, 1*time.Minute)
	return false
}

// validateAPIKeyFormat API 키 형식 검증 (레거시 호환)
func validateAPIKeyFormat(apiKey string) bool {
	// API 키 형식 기본 검증 (최소 길이, 문자 제한 등)
	if len(apiKey) < 16 || len(apiKey) > 128 {
		log.Printf("Invalid API key length: %d", len(apiKey))
		return false
	}

	// API 키 형식 검증: px_ 또는 proxynd_ 접두사 필수
	if !strings.HasPrefix(apiKey, "px_") && !strings.HasPrefix(apiKey, "proxynd_") {
		log.Printf("Invalid API key format: missing valid prefix")
		return false
	}

	// 접두사 이후의 키 부분 검증
	var keyPart string
	if strings.HasPrefix(apiKey, "px_") {
		keyPart = strings.TrimPrefix(apiKey, "px_")
	} else {
		keyPart = strings.TrimPrefix(apiKey, "proxynd_")
	}

	// 키 부분은 최소 12자 이상, 영숫자와 -, _ 만 허용
	if len(keyPart) < 12 {
		log.Printf("API key body too short: %d chars", len(keyPart))
		return false
	}

	for _, char := range keyPart {
		if !isValidAPIKeyChar(char) {
			log.Printf("Invalid character in API key")
			return false
		}
	}

	return true
}

// isValidAPIKeyChar API 키에 허용되는 문자인지 확인
func isValidAPIKeyChar(c rune) bool {
	return (c >= 'a' && c <= 'z') ||
		(c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') ||
		c == '-' || c == '_'
}

// validateBearerToken Bearer 토큰 검증 - JWT 서비스 통합
func validateBearerToken(token string) bool {
	// 캐시 키 생성 (처음 8자 + 마지막 8자)
	tokenLen := len(token)
	var cacheKey string
	if tokenLen > 16 {
		cacheKey = "bearer:" + token[:8] + token[tokenLen-8:]
	} else {
		cacheKey = "bearer:" + token
	}

	// 캐시 확인
	if cached := getCachedAuth(cacheKey); cached != nil {
		return cached.valid
	}

	// JWT 서비스를 통한 서명 검증
	svc := getJWTServiceInstance()
	if svc != nil {
		claims, err := svc.ValidateAccessToken(token)
		if err == nil && claims != nil {
			log.Printf("JWT token validated for user: %s", claims.UserID)
			// 캐시 TTL: 토큰 만료까지 또는 최대 5분
			ttl := 5 * time.Minute
			if claims.ExpiresAt != nil {
				remaining := time.Until(claims.ExpiresAt.Time)
				if remaining > 0 && remaining < ttl {
					ttl = remaining
				}
			}
			setCachedAuth(cacheKey, true, map[string]interface{}{
				"user_id":       claims.UserID,
				"email":         claims.Email,
				"role":          claims.Role,
				"organizations": claims.Organizations,
			}, ttl)
			return true
		}
		if err != nil {
			log.Printf("JWT validation failed via service: %v", err)
		}
	}

	// 폴백: JWT 형식 검증만 (서명 검증 없음)
	if validateJWTFormat(token) {
		log.Printf("JWT token passed format validation only")
		setCachedAuth(cacheKey, true, nil, 1*time.Minute)
		return true
	}

	setCachedAuth(cacheKey, false, nil, 1*time.Minute)
	return false
}

// validateJWTFormat JWT 형식 검증 (서명 검증 없음)
func validateJWTFormat(token string) bool {
	// JWT 토큰 형식 기본 검증
	if len(token) < 20 {
		log.Printf("Token too short: %d chars", len(token))
		return false
	}

	// JWT 형식 확인 (3개 파트가 점으로 구분)
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		log.Printf("Invalid JWT format: expected 3 parts, got %d", len(parts))
		return false
	}

	// 각 파트가 유효한 base64url 인코딩인지 확인
	for i, part := range parts {
		if len(part) == 0 {
			log.Printf("Invalid JWT: empty part at position %d", i)
			return false
		}

		// base64url 디코딩 시도 (유효성 검증)
		decoded, err := base64.RawURLEncoding.DecodeString(part)
		if err != nil {
			log.Printf("Invalid JWT: part %d is not valid base64url", i)
			return false
		}

		// header와 payload는 JSON 형식이어야 함
		if i < 2 { // header (0) 또는 payload (1)
			if len(decoded) < 2 || decoded[0] != '{' {
				log.Printf("Invalid JWT: part %d is not valid JSON", i)
				return false
			}
		}
	}

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

	// 사용자명과 비밀번호 기본 검증
	if len(username) == 0 {
		log.Printf("Basic auth failed: empty username")
		return false
	}

	if len(password) < 8 {
		log.Printf("Basic auth failed: password too short for user %s", username)
		return false
	}

	// config에서 허용된 사용자 목록 로드 및 검증
	globalConfig := config.GlobalConfig{}
	if err := globalConfig.ReadConfig(); err == nil {
		// config 로드 성공 시 실제 사용자 검증
		if globalConfig.Authentication != nil &&
			globalConfig.Authentication.BasicAuth != nil &&
			globalConfig.Authentication.BasicAuth.Users != nil {
			expectedPassword, exists := globalConfig.Authentication.BasicAuth.Users[username]
			if exists && expectedPassword == password {
				log.Printf("Basic auth validation passed for user: %s (config)", username)
				return true
			}

			log.Printf("Basic auth failed: invalid credentials for user %s", username)
			return false
		}
	}

	// config 로드 실패 또는 BasicAuth 설정 없음 - 개발 모드에서만 허용
	if isDevelopmentMode() {
		log.Printf("Basic auth validation passed for user: %s (dev mode)", username)
		return true
	}

	log.Printf("Basic auth validation failed: no valid configuration")
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
