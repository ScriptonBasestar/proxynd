package middlewares

import (
	"fmt"
	"net"
	"regexp"
	"strings"

	"github.com/gofiber/fiber/v2"

	"proxynd/logging"
)

const (
	// MethodGET HTTP GET method
	MethodGET = "GET"
	// MethodPOST HTTP POST method
	MethodPOST = "POST"
	// MethodPUT HTTP PUT method
	MethodPUT = "PUT"
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
		if c.Method() == MethodPOST || c.Method() == MethodPUT {
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
	for key, value := range c.Context().QueryArgs().All() {
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
			return fmt.Errorf("query parameter too long (max %d characters)", cfg.MaxQueryLength)
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
				return fmt.Errorf("dangerous pattern detected in query parameter")
			}
		}
	}

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

// StrictValidationConfig 더 엄격한 검증을 위한 설정
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

// EnhancedInputValidation 향상된 입력 검증 미들웨어 (SQL 인젝션, XSS 등 추가 보안)
func EnhancedInputValidation(config ...ValidationConfig) fiber.Handler {
	cfg := DefaultValidationConfig()
	if len(config) > 0 {
		cfg = config[0]
	}

	logger := logging.GetLogger()

	// 추가 위험 패턴 (SQL 인젝션, NoSQL 인젝션 등)
	sqlPatterns := []string{
		"union", "select", "insert", "update", "delete", "drop", "create", "alter",
		"exec", "execute", "sp_", "xp_", "0x", "@@", "char(", "nchar(",
		"$where", "$regex", "$ne", "$gt", "$lt", "$in", "$nin",
	}

	xssPatterns := []string{
		"<iframe", "<object", "<embed", "<applet", "<meta",
		"vbscript:", "livescript:", "mocha:", "charset=",
		"&#", "%3c", "%3e", "%22", "%27",
	}

	return func(c *fiber.Ctx) error {
		// 기존 검증 먼저 실행
		baseValidation := InputValidation(cfg)
		if err := baseValidation(c); err != nil {
			return err
		}

		// 향상된 검증 실행
		if err := validateAdvancedSecurity(c, cfg, logger, sqlPatterns, xssPatterns); err != nil {
			return c.Status(400).JSON(fiber.Map{
				"error":   "Security validation failed",
				"details": err.Error(),
			})
		}

		return c.Next()
	}
}

// validateAdvancedSecurity 고급 보안 검증
func validateAdvancedSecurity(
	c *fiber.Ctx,
	cfg ValidationConfig,
	logger logging.Logger,
	sqlPatterns, xssPatterns []string,
) error {
	// 1. Request Body 검증 (JSON/XML 파싱 없이)
	if c.Method() == MethodPOST || c.Method() == MethodPUT {
		bodyBytes := c.Body()
		if len(bodyBytes) > 0 {
			bodyStr := strings.ToLower(string(bodyBytes))

			// SQL 인젝션 패턴 검사
			for _, pattern := range sqlPatterns {
				if strings.Contains(bodyStr, pattern) {
					if cfg.EnableLogging {
						logger.Warn("SQL injection pattern detected in body",
							logging.F("client_ip", c.IP()),
							logging.F("pattern", pattern),
							logging.F("path", c.Path()),
							logging.F("method", c.Method()))
					}
					return fmt.Errorf("invalid request content")
				}
			}

			// XSS 패턴 검사
			for _, pattern := range xssPatterns {
				if strings.Contains(bodyStr, pattern) {
					if cfg.EnableLogging {
						logger.Warn("XSS pattern detected in body",
							logging.F("client_ip", c.IP()),
							logging.F("pattern", pattern),
							logging.F("path", c.Path()),
							logging.F("method", c.Method()))
					}
					return fmt.Errorf("invalid request content")
				}
			}
		}
	}

	// 2. HTTP 헤더 검증
	if err := validateHTTPHeaders(c, cfg, logger, xssPatterns); err != nil {
		return err
	}

	// 3. Content-Type 검증
	if err := validateContentType(c, cfg, logger); err != nil {
		return err
	}

	return nil
}

// validateHTTPHeaders HTTP 헤더 검증
func validateHTTPHeaders(c *fiber.Ctx, cfg ValidationConfig, logger logging.Logger, xssPatterns []string) error {
	// 중요한 헤더들 검증
	headers := []string{"Referer", "X-Forwarded-For", "X-Real-IP", "Authorization"}

	for _, headerName := range headers {
		headerValue := strings.ToLower(c.Get(headerName))
		if headerValue == "" {
			continue
		}

		// 헤더 값 길이 제한
		if len(headerValue) > 2000 {
			if cfg.EnableLogging {
				logger.Warn("HTTP header too long",
					logging.F("client_ip", c.IP()),
					logging.F("header", headerName),
					logging.F("length", len(headerValue)),
					logging.F("path", c.Path()))
			}
			return fmt.Errorf("header %s too long", headerName)
		}

		// XSS 패턴 검사
		for _, pattern := range xssPatterns {
			if strings.Contains(headerValue, pattern) {
				if cfg.EnableLogging {
					logger.Warn("XSS pattern detected in header",
						logging.F("client_ip", c.IP()),
						logging.F("header", headerName),
						logging.F("pattern", pattern),
						logging.F("path", c.Path()))
				}
				return fmt.Errorf("invalid header format")
			}
		}
	}

	return nil
}

// validateContentType Content-Type 검증
func validateContentType(c *fiber.Ctx, cfg ValidationConfig, logger logging.Logger) error {
	contentType := strings.ToLower(c.Get("Content-Type"))

	// POST/PUT 요청에 대한 Content-Type 검증
	if (c.Method() == MethodPOST || c.Method() == MethodPUT) && len(c.Body()) > 0 {
		allowedContentTypes := []string{
			"application/json",
			"application/xml",
			"text/xml",
			"application/x-www-form-urlencoded",
			"multipart/form-data",
			"application/octet-stream",
			"text/plain",
		}

		isValid := false
		for _, allowed := range allowedContentTypes {
			if strings.HasPrefix(contentType, allowed) {
				isValid = true
				break
			}
		}

		if !isValid && contentType != "" {
			if cfg.EnableLogging {
				logger.Warn("Invalid Content-Type",
					logging.F("client_ip", c.IP()),
					logging.F("content_type", contentType),
					logging.F("path", c.Path()),
					logging.F("method", c.Method()))
			}
			return fmt.Errorf("invalid content type: %s", contentType)
		}
	}

	return nil
}

// IPValidation IP 주소 기반 검증
func IPValidation(blockedCIDRs, allowedCIDRs []string) fiber.Handler {
	logger := logging.GetLogger()

	return func(c *fiber.Ctx) error {
		clientIP := c.IP()

		// 차단된 CIDR 확인
		for _, cidr := range blockedCIDRs {
			if isIPInCIDR(clientIP, cidr) {
				logger.Warn("Request from blocked IP range",
					logging.F("client_ip", clientIP),
					logging.F("blocked_cidr", cidr),
					logging.F("path", c.Path()))
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
					"error": "Access denied",
				})
			}
		}

		// 허용된 CIDR 확인 (설정된 경우)
		if len(allowedCIDRs) > 0 {
			allowed := false
			for _, cidr := range allowedCIDRs {
				if isIPInCIDR(clientIP, cidr) {
					allowed = true
					break
				}
			}

			if !allowed {
				logger.Warn("Request from non-allowed IP range",
					logging.F("client_ip", clientIP),
					logging.F("path", c.Path()))
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
					"error": "Access denied",
				})
			}
		}

		return c.Next()
	}
}

// isIPInCIDR IP가 CIDR 범위에 있는지 확인
func isIPInCIDR(ip, cidr string) bool {
	if !strings.Contains(cidr, "/") {
		return ip == cidr
	}

	clientIP := net.ParseIP(ip)
	if clientIP == nil {
		return false
	}

	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return false
	}

	return ipNet.Contains(clientIP)
}
