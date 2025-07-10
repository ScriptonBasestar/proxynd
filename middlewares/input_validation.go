package middlewares

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/gofiber/fiber/v2"

	"proxynd/logging"
)

var (
	// 안전한 파일명 패턴 (알파벳, 숫자, 하이픈, 언더스코어, 점)
	safeFilenameRegex = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)

	// 패키지 이름 패턴 (슬래시 포함으로 네임스페이스 지원)
	packageNameRegex = regexp.MustCompile(`^[a-zA-Z0-9._/@-]+$`)

	// 버전 패턴 (SNAPSHOT, RELEASE 등 포함)
	versionRegex = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)

	// 위험한 패턴들
	dangerousPatterns = []string{
		"<script", "</script", "javascript:", "data:",
		"../", "..", "%2e%2e", "%2f", "%5c",
		"<", ">", "'", "\"", "&",
		"eval(", "alert(", "document.", "window.",
	}
)

// ValidationConfig 입력 검증 설정
type ValidationConfig struct {
	MaxFileSize     int64    `json:"max_file_size"`     // bytes
	MaxQueryLength  int      `json:"max_query_length"`  // characters
	MaxParamLength  int      `json:"max_param_length"`  // characters
	AllowedFileExts []string `json:"allowed_file_exts"` // [".jar", ".tgz", ".deb"]
	EnableLogging   bool     `json:"enable_logging"`    // 로깅 활성화
}

// DefaultValidationConfig 기본 검증 설정
func DefaultValidationConfig() ValidationConfig {
	return ValidationConfig{
		MaxFileSize:    100 * 1024 * 1024, // 100MB
		MaxQueryLength: 1000,
		MaxParamLength: 255,
		AllowedFileExts: []string{
			".jar", ".war", ".pom", ".xml", ".md5", ".sha1", ".sha256",
			".tgz", ".tar.gz", ".tar.bz2", ".tar.xz",
			".deb", ".rpm", ".apk",
			".whl", ".egg", ".tar.gz",
			".gem",
		},
		EnableLogging: true,
	}
}

// InputValidation 입력 검증 미들웨어
func InputValidation(config ...ValidationConfig) fiber.Handler {
	cfg := DefaultValidationConfig()
	if len(config) > 0 {
		cfg = config[0]
	}

	logger := logging.GetLogger()

	return func(c *fiber.Ctx) error {
		// 1. Content-Length 검증
		if int64(c.Request().Header.ContentLength()) > cfg.MaxFileSize {
			if cfg.EnableLogging {
				logger.Warn("Request rejected: too large",
					logging.F("client_ip", c.IP()),
					logging.F("content_length", c.Request().Header.ContentLength()),
					logging.F("max_allowed", cfg.MaxFileSize),
					logging.F("path", c.Path()))
			}
			return c.Status(413).JSON(fiber.Map{
				"error":    "Request too large",
				"max_size": cfg.MaxFileSize,
			})
		}

		// 2. Query Parameters 검증
		if err := validateQueryParams(c, cfg, logger); err != nil {
			return c.Status(400).JSON(fiber.Map{
				"error":   "Invalid query parameter",
				"details": err.Error(),
			})
		}

		// 3. Path Parameters 검증
		if err := validatePathParams(c, cfg, logger); err != nil {
			return c.Status(400).JSON(fiber.Map{
				"error":   "Invalid path parameter",
				"details": err.Error(),
			})
		}

		// 4. 파일 확장자 검증 (파일 업로드 시)
		if c.Method() == "POST" || c.Method() == "PUT" {
			if err := validateFileExtension(c, cfg, logger); err != nil {
				return c.Status(400).JSON(fiber.Map{
					"error":   "Invalid file type",
					"details": err.Error(),
				})
			}
		}

		// 5. User-Agent 헤더 검증
		if err := validateUserAgent(c, logger); err != nil {
			return c.Status(400).JSON(fiber.Map{
				"error":   "Invalid user agent",
				"details": err.Error(),
			})
		}

		return c.Next()
	}
}

// validateQueryParams 쿼리 파라미터 검증
func validateQueryParams(c *fiber.Ctx, cfg ValidationConfig, logger logging.Logger) error {
	queryString := c.Request().URI().QueryString()

	// 전체 쿼리 스트링 길이 검증
	if len(queryString) > cfg.MaxQueryLength {
		if cfg.EnableLogging {
			logger.Warn("Query string too long",
				logging.F("client_ip", c.IP()),
				logging.F("query_length", len(queryString)),
				logging.F("max_allowed", cfg.MaxQueryLength),
				logging.F("path", c.Path()))
		}
		return fmt.Errorf("query string too long (max %d characters)", cfg.MaxQueryLength)
	}

	// 각 쿼리 파라미터 검증
	c.Context().QueryArgs().VisitAll(func(key, value []byte) {
		keyStr := string(key)
		valueStr := string(value)

		// 개별 파라미터 길이 검증
		if len(valueStr) > cfg.MaxQueryLength {
			if cfg.EnableLogging {
				logger.Warn("Query parameter too long",
					logging.F("client_ip", c.IP()),
					logging.F("parameter", keyStr),
					logging.F("value_length", len(valueStr)),
					logging.F("path", c.Path()))
			}
			return
		}

		// 위험한 패턴 검증
		valueStrLower := strings.ToLower(valueStr)
		for _, pattern := range dangerousPatterns {
			if strings.Contains(valueStrLower, strings.ToLower(pattern)) {
				if cfg.EnableLogging {
					logger.Warn("Dangerous pattern detected in query",
						logging.F("client_ip", c.IP()),
						logging.F("parameter", keyStr),
						logging.F("pattern", pattern),
						logging.F("path", c.Path()))
				}
				return
			}
		}
	})

	return nil
}

// validatePathParams 경로 파라미터 검증
func validatePathParams(c *fiber.Ctx, cfg ValidationConfig, logger logging.Logger) error {
	// 와일드카드 파라미터 검증 (*로 캐치되는 경로)
	if wildcardPath := c.Params("*"); wildcardPath != "" {
		if len(wildcardPath) > cfg.MaxParamLength*2 { // 와일드카드는 좀 더 길게 허용
			if cfg.EnableLogging {
				logger.Warn("Wildcard path too long",
					logging.F("client_ip", c.IP()),
					logging.F("path_length", len(wildcardPath)),
					logging.F("path", c.Path()))
			}
			return fmt.Errorf("path too long")
		}

		// 위험한 패턴 검증
		pathLower := strings.ToLower(wildcardPath)
		for _, pattern := range dangerousPatterns {
			if strings.Contains(pathLower, strings.ToLower(pattern)) {
				if cfg.EnableLogging {
					logger.Warn("Dangerous pattern detected in path",
						logging.F("client_ip", c.IP()),
						logging.F("pattern", pattern),
						logging.F("wildcard_path", wildcardPath),
						logging.F("path", c.Path()))
				}
				return fmt.Errorf("invalid path format")
			}
		}
	}

	// 패키지명 검증
	if packageName := c.Params("package"); packageName != "" {
		if len(packageName) > cfg.MaxParamLength {
			if cfg.EnableLogging {
				logger.Warn("Package name too long",
					logging.F("client_ip", c.IP()),
					logging.F("package", packageName),
					logging.F("path", c.Path()))
			}
			return fmt.Errorf("package name too long")
		}
		if !packageNameRegex.MatchString(packageName) {
			if cfg.EnableLogging {
				logger.Warn("Invalid package name format",
					logging.F("client_ip", c.IP()),
					logging.F("package", packageName),
					logging.F("path", c.Path()))
			}
			return fmt.Errorf("invalid package name format")
		}
	}

	// 버전 검증
	if version := c.Params("version"); version != "" {
		if len(version) > cfg.MaxParamLength {
			if cfg.EnableLogging {
				logger.Warn("Version string too long",
					logging.F("client_ip", c.IP()),
					logging.F("version", version),
					logging.F("path", c.Path()))
			}
			return fmt.Errorf("version string too long")
		}
		if !versionRegex.MatchString(version) {
			if cfg.EnableLogging {
				logger.Warn("Invalid version format",
					logging.F("client_ip", c.IP()),
					logging.F("version", version),
					logging.F("path", c.Path()))
			}
			return fmt.Errorf("invalid version format")
		}
	}

	// 파일명 검증
	if filename := c.Params("filename"); filename != "" {
		if len(filename) > cfg.MaxParamLength {
			if cfg.EnableLogging {
				logger.Warn("Filename too long",
					logging.F("client_ip", c.IP()),
					logging.F("filename", filename),
					logging.F("path", c.Path()))
			}
			return fmt.Errorf("filename too long")
		}
		if !safeFilenameRegex.MatchString(filename) {
			if cfg.EnableLogging {
				logger.Warn("Invalid filename format",
					logging.F("client_ip", c.IP()),
					logging.F("filename", filename),
					logging.F("path", c.Path()))
			}
			return fmt.Errorf("invalid filename format")
		}
	}

	// 프록시 타입 검증
	if proxyType := c.Params("type"); proxyType != "" {
		allowedTypes := []string{"apt", "npm", "maven", "pip", "docker", "yum", "gem", "apk"}
		isValid := false
		for _, validType := range allowedTypes {
			if proxyType == validType {
				isValid = true
				break
			}
		}
		if !isValid {
			if cfg.EnableLogging {
				logger.Warn("Invalid proxy type",
					logging.F("client_ip", c.IP()),
					logging.F("proxy_type", proxyType),
					logging.F("path", c.Path()))
			}
			return fmt.Errorf("invalid proxy type: %s", proxyType)
		}
	}

	return nil
}

// validateFileExtension 파일 확장자 검증
func validateFileExtension(c *fiber.Ctx, cfg ValidationConfig, logger logging.Logger) error {
	// Content-Type 또는 파일명으로 확장자 검증
	filename := c.Get("X-Filename")
	if filename == "" {
		// URL path에서 파일명 추출 시도
		path := c.Path()
		if lastSlash := strings.LastIndex(path, "/"); lastSlash != -1 {
			filename = path[lastSlash+1:]
		}
	}

	if filename != "" {
		filenameLower := strings.ToLower(filename)
		isAllowed := false

		for _, ext := range cfg.AllowedFileExts {
			if strings.HasSuffix(filenameLower, strings.ToLower(ext)) {
				isAllowed = true
				break
			}
		}

		if !isAllowed {
			if cfg.EnableLogging {
				logger.Warn("File extension not allowed",
					logging.F("client_ip", c.IP()),
					logging.F("filename", filename),
					logging.F("path", c.Path()))
			}
			return fmt.Errorf("file extension not allowed: %s", filename)
		}
	}

	return nil
}

// validateUserAgent User-Agent 헤더 검증
func validateUserAgent(c *fiber.Ctx, logger logging.Logger) error {
	userAgent := c.Get("User-Agent")

	// User-Agent가 없거나 너무 짧으면 의심스러운 요청
	if len(userAgent) < 3 {
		logger.Warn("Suspicious request: missing or short User-Agent",
			logging.F("client_ip", c.IP()),
			logging.F("user_agent", userAgent),
			logging.F("path", c.Path()))
		return fmt.Errorf("invalid user agent")
	}

	// 너무 긴 User-Agent도 의심스러움
	if len(userAgent) > 1000 {
		logger.Warn("Suspicious request: User-Agent too long",
			logging.F("client_ip", c.IP()),
			logging.F("user_agent_length", len(userAgent)),
			logging.F("path", c.Path()))
		return fmt.Errorf("user agent too long")
	}

	// 위험한 패턴 확인
	userAgentLower := strings.ToLower(userAgent)
	for _, pattern := range dangerousPatterns {
		if strings.Contains(userAgentLower, strings.ToLower(pattern)) {
			logger.Warn("Dangerous pattern detected in User-Agent",
				logging.F("client_ip", c.IP()),
				logging.F("pattern", pattern),
				logging.F("user_agent", userAgent),
				logging.F("path", c.Path()))
			return fmt.Errorf("invalid user agent format")
		}
	}

	return nil
}

// StrictValidation 더 엄격한 검증을 위한 설정
func StrictValidationConfig() ValidationConfig {
	return ValidationConfig{
		MaxFileSize:    10 * 1024 * 1024, // 10MB
		MaxQueryLength: 500,
		MaxParamLength: 100,
		AllowedFileExts: []string{
			".jar", ".pom", ".xml",
			".tgz", ".tar.gz",
			".deb", ".rpm",
			".whl",
			".gem",
		},
		EnableLogging: true,
	}
}