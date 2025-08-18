package middlewares

import (
	"encoding/base64"
	"fmt"
	"net"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"

	"proxynd/internal/logging"
)

// AdvancedSecurityConfig 고급 보안 설정
type AdvancedSecurityConfig struct {
	// DDoS Protection
	EnableDDoSProtection  bool          `json:"enable_ddos_protection"`
	SuspiciousThreshold   int           `json:"suspicious_threshold"`    // 의심스러운 요청 임계값
	BlockDuration         time.Duration `json:"block_duration"`          // 차단 지속 시간
	MaxConcurrentRequests int           `json:"max_concurrent_requests"` // IP당 최대 동시 요청

	// Request Validation
	MaxRequestSize int64    `json:"max_request_size"` // 최대 요청 크기 (bytes)
	MaxHeaderSize  int      `json:"max_header_size"`  // 최대 헤더 크기
	MaxQueryParams int      `json:"max_query_params"` // 최대 쿼리 파라미터 수
	AllowedMethods []string `json:"allowed_methods"`  // 허용된 HTTP 메서드

	// Content Security
	EnableCSP     bool              `json:"enable_csp"`     // Content Security Policy
	CSPDirectives map[string]string `json:"csp_directives"` // CSP 지시문
	EnableHSTS    bool              `json:"enable_hsts"`    // HTTP Strict Transport Security
	HSTSMaxAge    int               `json:"hsts_max_age"`   // HSTS 최대 지속 시간

	// IP Filtering
	BlockedIPs       []string `json:"blocked_ips"`       // 차단된 IP 목록
	AllowedIPs       []string `json:"allowed_ips"`       // 허용된 IP 목록 (화이트리스트)
	BlockedCountries []string `json:"blocked_countries"` // 차단된 국가 코드

	// User Agent Filtering
	BlockedUserAgents []string `json:"blocked_user_agents"` // 차단된 User-Agent 패턴
	RequireUserAgent  bool     `json:"require_user_agent"`  // User-Agent 필수 여부

	// Path Security
	BlockedPaths []string `json:"blocked_paths"` // 차단된 경로 패턴
	HiddenPaths  []string `json:"hidden_paths"`  // 숨겨진 경로 (404 반환)

	// Logging and Monitoring
	EnableSecurityLogging bool `json:"enable_security_logging"` // 보안 로깅 활성화
	EnableMetrics         bool `json:"enable_metrics"`          // 메트릭 수집 활성화

	// Custom Headers
	CustomHeaders map[string]string `json:"custom_headers"` // 커스텀 응답 헤더
}

// DefaultAdvancedSecurityConfig 기본 고급 보안 설정
func DefaultAdvancedSecurityConfig() AdvancedSecurityConfig {
	return AdvancedSecurityConfig{
		// DDoS Protection
		EnableDDoSProtection:  true,
		SuspiciousThreshold:   100,
		BlockDuration:         15 * time.Minute,
		MaxConcurrentRequests: 10,

		// Request Validation
		MaxRequestSize: 100 * 1024 * 1024, // 100MB
		MaxHeaderSize:  32 * 1024,         // 32KB
		MaxQueryParams: 50,
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "HEAD", "OPTIONS"},

		// Content Security
		EnableCSP: true,
		CSPDirectives: map[string]string{
			"default-src": "'self'",
			"script-src":  "'self' 'unsafe-inline'",
			"style-src":   "'self' 'unsafe-inline'",
			"img-src":     "'self' data: https:",
		},
		EnableHSTS: true,
		HSTSMaxAge: 31536000, // 1년

		// Security Features
		RequireUserAgent:      false,
		EnableSecurityLogging: true,
		EnableMetrics:         true,

		// Custom Security Headers
		CustomHeaders: map[string]string{
			"X-Content-Type-Options": "nosniff",
			"X-Frame-Options":        "DENY",
			"X-XSS-Protection":       "1; mode=block",
			"Referrer-Policy":        "strict-origin-when-cross-origin",
		},

		// Default blocked patterns
		BlockedUserAgents: []string{
			"(?i)(bot|crawler|spider|scraper)",
			"(?i)(scanner|vulnerability|exploit)",
		},
		BlockedPaths: []string{
			"(?i)(\\.\\.|/\\.|\\\\)",          // Directory traversal
			"(?i)(admin|wp-admin|phpmyadmin)", // Common admin paths
			"(?i)(\\.php|\\.asp|\\.jsp)",      // Script file extensions
		},
	}
}

// SecurityMetrics 보안 메트릭
type SecurityMetrics struct {
	mu                 sync.RWMutex
	BlockedRequests    int64            `json:"blocked_requests"`
	SuspiciousRequests int64            `json:"suspicious_requests"`
	BlockedIPs         map[string]int64 `json:"blocked_ips"`
	BlockedUserAgents  map[string]int64 `json:"blocked_user_agents"`
	LastReset          time.Time        `json:"last_reset"`
}

// AdvancedSecurityMiddleware 고급 보안 미들웨어
type AdvancedSecurityMiddleware struct {
	config          AdvancedSecurityConfig
	blockedIPs      map[string]time.Time
	concurrentReqs  map[string]int
	ipMutex         sync.RWMutex
	concurrentMutex sync.RWMutex
	metrics         *SecurityMetrics
	logger          logging.Logger

	// Compiled regex patterns for performance
	blockedUserAgentRegex []*regexp.Regexp
	blockedPathRegex      []*regexp.Regexp
}

// NewAdvancedSecurityMiddleware 고급 보안 미들웨어 생성
func NewAdvancedSecurityMiddleware(config AdvancedSecurityConfig) *AdvancedSecurityMiddleware {
	middleware := &AdvancedSecurityMiddleware{
		config:         config,
		blockedIPs:     make(map[string]time.Time),
		concurrentReqs: make(map[string]int),
		metrics: &SecurityMetrics{
			BlockedIPs:        make(map[string]int64),
			BlockedUserAgents: make(map[string]int64),
			LastReset:         time.Now(),
		},
		logger: logging.GetLogger(),
	}

	// 정규식 패턴 컴파일
	middleware.compileRegexPatterns()

	// 주기적으로 차단된 IP 정리
	go middleware.cleanupBlockedIPs()

	return middleware
}

// compileRegexPatterns 정규식 패턴 컴파일
func (m *AdvancedSecurityMiddleware) compileRegexPatterns() {
	// User-Agent 패턴 컴파일
	for _, pattern := range m.config.BlockedUserAgents {
		if regex, err := regexp.Compile(pattern); err == nil {
			m.blockedUserAgentRegex = append(m.blockedUserAgentRegex, regex)
		} else {
			m.logger.Warn("Invalid user agent regex pattern",
				logging.F("pattern", pattern),
				logging.F("error", err))
		}
	}

	// Path 패턴 컴파일
	for _, pattern := range m.config.BlockedPaths {
		if regex, err := regexp.Compile(pattern); err == nil {
			m.blockedPathRegex = append(m.blockedPathRegex, regex)
		} else {
			m.logger.Warn("Invalid path regex pattern",
				logging.F("pattern", pattern),
				logging.F("error", err))
		}
	}
}

// Handler 미들웨어 핸들러 반환
func (m *AdvancedSecurityMiddleware) Handler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		clientIP := c.IP()

		// 1. IP 기반 검증
		if blocked := m.checkIPBlocking(clientIP); blocked {
			m.incrementMetric("blocked_requests")
			m.logSecurityEvent("IP_BLOCKED", clientIP, c.Path(), "IP is in blocklist or temporarily blocked")
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Access denied",
			})
		}

		// 2. 동시 요청 수 제한
		if exceeded := m.checkConcurrentRequests(clientIP); exceeded {
			m.incrementMetric("blocked_requests")
			m.logSecurityEvent("CONCURRENT_LIMIT", clientIP, c.Path(), "Too many concurrent requests")
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "Too many concurrent requests",
			})
		}

		// 동시 요청 카운터 증가
		m.incrementConcurrentRequests(clientIP)
		defer m.decrementConcurrentRequests(clientIP)

		// 3. HTTP 메서드 검증
		if !m.isMethodAllowed(c.Method()) {
			m.incrementMetric("blocked_requests")
			m.logSecurityEvent("METHOD_NOT_ALLOWED", clientIP, c.Path(),
				fmt.Sprintf("Method %s not allowed", c.Method()))
			return c.Status(fiber.StatusMethodNotAllowed).JSON(fiber.Map{
				"error": "Method not allowed",
			})
		}

		// 4. 요청 크기 검증
		if int64(c.Request().Header.ContentLength()) > m.config.MaxRequestSize {
			m.incrementMetric("blocked_requests")
			m.logSecurityEvent("REQUEST_TOO_LARGE", clientIP, c.Path(), "Request body too large")
			return c.Status(fiber.StatusRequestEntityTooLarge).JSON(fiber.Map{
				"error": "Request too large",
			})
		}

		// 5. User-Agent 검증
		if blocked := m.checkUserAgent(c.Get("User-Agent")); blocked {
			m.incrementMetric("blocked_requests")
			m.incrementBlockedUserAgent(c.Get("User-Agent"))
			m.logSecurityEvent("USER_AGENT_BLOCKED", clientIP, c.Path(), "Blocked user agent pattern")
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Access denied",
			})
		}

		// 6. 경로 패턴 검증
		if blocked := m.checkPathSecurity(c.Path()); blocked {
			m.incrementMetric("blocked_requests")
			m.logSecurityEvent("PATH_BLOCKED", clientIP, c.Path(), "Blocked path pattern")
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Not found",
			})
		}

		// 7. 쿼리 파라미터 수 제한
		if len(c.Request().URI().QueryArgs().String()) > 0 {
			queryCount := strings.Count(c.Request().URI().QueryArgs().String(), "&") + 1
			if queryCount > m.config.MaxQueryParams {
				m.incrementMetric("suspicious_requests")
				m.logSecurityEvent("TOO_MANY_PARAMS", clientIP, c.Path(),
					fmt.Sprintf("Too many query parameters: %d", queryCount))
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": "Too many query parameters",
				})
			}
		}

		// 8. 보안 헤더 추가
		m.addSecurityHeaders(c)

		// 요청 처리
		return c.Next()
	}
}

// checkIPBlocking IP 차단 검사
func (m *AdvancedSecurityMiddleware) checkIPBlocking(clientIP string) bool {
	// 허용된 IP 목록 확인 (화이트리스트)
	if len(m.config.AllowedIPs) > 0 {
		for _, allowedIP := range m.config.AllowedIPs {
			if m.matchIPPattern(clientIP, allowedIP) {
				return false // 허용된 IP
			}
		}
		return true // 화이트리스트에 없으면 차단
	}

	// 차단된 IP 목록 확인
	for _, blockedIP := range m.config.BlockedIPs {
		if m.matchIPPattern(clientIP, blockedIP) {
			return true
		}
	}

	// 임시 차단된 IP 확인
	m.ipMutex.RLock()
	defer m.ipMutex.RUnlock()

	if blockTime, exists := m.blockedIPs[clientIP]; exists {
		if time.Since(blockTime) < m.config.BlockDuration {
			return true
		}
		// 차단 시간이 지났으면 제거
		delete(m.blockedIPs, clientIP)
	}

	return false
}

// matchIPPattern IP 패턴 매칭
func (m *AdvancedSecurityMiddleware) matchIPPattern(clientIP, pattern string) bool {
	// CIDR 표기법 지원
	if strings.Contains(pattern, "/") {
		_, ipNet, err := net.ParseCIDR(pattern)
		if err != nil {
			return false
		}
		ip := net.ParseIP(clientIP)
		return ip != nil && ipNet.Contains(ip)
	}

	// 정확한 매칭
	return clientIP == pattern
}

// checkConcurrentRequests 동시 요청 수 검사
func (m *AdvancedSecurityMiddleware) checkConcurrentRequests(clientIP string) bool {
	m.concurrentMutex.RLock()
	current := m.concurrentReqs[clientIP]
	m.concurrentMutex.RUnlock()

	return current >= m.config.MaxConcurrentRequests
}

// incrementConcurrentRequests 동시 요청 카운터 증가
func (m *AdvancedSecurityMiddleware) incrementConcurrentRequests(clientIP string) {
	m.concurrentMutex.Lock()
	m.concurrentReqs[clientIP]++
	m.concurrentMutex.Unlock()
}

// decrementConcurrentRequests 동시 요청 카운터 감소
func (m *AdvancedSecurityMiddleware) decrementConcurrentRequests(clientIP string) {
	m.concurrentMutex.Lock()
	defer m.concurrentMutex.Unlock()

	if count := m.concurrentReqs[clientIP]; count > 0 {
		m.concurrentReqs[clientIP]--
		if m.concurrentReqs[clientIP] == 0 {
			delete(m.concurrentReqs, clientIP)
		}
	}
}

// isMethodAllowed HTTP 메서드 허용 검사
func (m *AdvancedSecurityMiddleware) isMethodAllowed(method string) bool {
	if len(m.config.AllowedMethods) == 0 {
		return true // 제한 없음
	}

	for _, allowedMethod := range m.config.AllowedMethods {
		if strings.EqualFold(method, allowedMethod) {
			return true
		}
	}
	return false
}

// checkUserAgent User-Agent 검사
func (m *AdvancedSecurityMiddleware) checkUserAgent(userAgent string) bool {
	// User-Agent 필수 검사
	if m.config.RequireUserAgent && userAgent == "" {
		return true // 차단
	}

	// 패턴 매칭
	for _, regex := range m.blockedUserAgentRegex {
		if regex.MatchString(userAgent) {
			return true // 차단
		}
	}

	return false
}

// checkPathSecurity 경로 보안 검사
func (m *AdvancedSecurityMiddleware) checkPathSecurity(path string) bool {
	// 차단된 경로 패턴 검사
	for _, regex := range m.blockedPathRegex {
		if regex.MatchString(path) {
			return true // 차단
		}
	}

	// 숨겨진 경로 검사
	for _, hiddenPath := range m.config.HiddenPaths {
		if strings.HasPrefix(path, hiddenPath) {
			return true // 차단 (404 반환)
		}
	}

	return false
}

// addSecurityHeaders 보안 헤더 추가
func (m *AdvancedSecurityMiddleware) addSecurityHeaders(c *fiber.Ctx) {
	// HSTS 헤더
	if m.config.EnableHSTS {
		c.Set("Strict-Transport-Security",
			fmt.Sprintf("max-age=%d; includeSubDomains", m.config.HSTSMaxAge))
	}

	// CSP 헤더
	if m.config.EnableCSP && len(m.config.CSPDirectives) > 0 {
		var cspParts []string
		for directive, value := range m.config.CSPDirectives {
			cspParts = append(cspParts, fmt.Sprintf("%s %s", directive, value))
		}
		c.Set("Content-Security-Policy", strings.Join(cspParts, "; "))
	}

	// 커스텀 헤더
	for header, value := range m.config.CustomHeaders {
		c.Set(header, value)
	}

	// ProxyND 식별 헤더
	c.Set("X-Powered-By", "ProxyND")
	c.Set("X-Security-Level", "Enhanced")
}

// cleanupBlockedIPs 차단된 IP 정리
func (m *AdvancedSecurityMiddleware) cleanupBlockedIPs() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		m.ipMutex.Lock()
		now := time.Now()
		for ip, blockTime := range m.blockedIPs {
			if now.Sub(blockTime) >= m.config.BlockDuration {
				delete(m.blockedIPs, ip)
			}
		}
		m.ipMutex.Unlock()
	}
}

// BlockIP IP 임시 차단
func (m *AdvancedSecurityMiddleware) BlockIP(ip, reason string) {
	m.ipMutex.Lock()
	m.blockedIPs[ip] = time.Now()
	m.ipMutex.Unlock()

	m.incrementBlockedIP(ip)
	m.logSecurityEvent("IP_TEMP_BLOCKED", ip, "", reason)
}

// incrementMetric 메트릭 증가
func (m *AdvancedSecurityMiddleware) incrementMetric(metric string) {
	if !m.config.EnableMetrics {
		return
	}

	m.metrics.mu.Lock()
	defer m.metrics.mu.Unlock()

	switch metric {
	case "blocked_requests":
		m.metrics.BlockedRequests++
	case "suspicious_requests":
		m.metrics.SuspiciousRequests++
	}
}

// incrementBlockedIP 차단된 IP 메트릭 증가
func (m *AdvancedSecurityMiddleware) incrementBlockedIP(ip string) {
	if !m.config.EnableMetrics {
		return
	}

	m.metrics.mu.Lock()
	m.metrics.BlockedIPs[ip]++
	m.metrics.mu.Unlock()
}

// incrementBlockedUserAgent 차단된 User-Agent 메트릭 증가
func (m *AdvancedSecurityMiddleware) incrementBlockedUserAgent(userAgent string) {
	if !m.config.EnableMetrics {
		return
	}

	m.metrics.mu.Lock()
	// User-Agent를 base64로 인코딩하여 저장 (특수문자 문제 방지)
	encoded := base64.StdEncoding.EncodeToString([]byte(userAgent))
	m.metrics.BlockedUserAgents[encoded]++
	m.metrics.mu.Unlock()
}

// logSecurityEvent 보안 이벤트 로깅
func (m *AdvancedSecurityMiddleware) logSecurityEvent(eventType, clientIP, path, details string) {
	if !m.config.EnableSecurityLogging {
		return
	}

	m.logger.Warn("Security event detected",
		logging.F("event_type", eventType),
		logging.F("client_ip", clientIP),
		logging.F("path", path),
		logging.F("details", details),
		logging.F("timestamp", time.Now().UTC().Format(time.RFC3339)))
}

// GetMetrics 보안 메트릭 반환
func (m *AdvancedSecurityMiddleware) GetMetrics() *SecurityMetrics {
	m.metrics.mu.RLock()
	defer m.metrics.mu.RUnlock()

	// 메트릭 복사 반환
	metrics := &SecurityMetrics{
		BlockedRequests:    m.metrics.BlockedRequests,
		SuspiciousRequests: m.metrics.SuspiciousRequests,
		BlockedIPs:         make(map[string]int64),
		BlockedUserAgents:  make(map[string]int64),
		LastReset:          m.metrics.LastReset,
	}

	for k, v := range m.metrics.BlockedIPs {
		metrics.BlockedIPs[k] = v
	}

	for k, v := range m.metrics.BlockedUserAgents {
		metrics.BlockedUserAgents[k] = v
	}

	return metrics
}

// ResetMetrics 메트릭 초기화
func (m *AdvancedSecurityMiddleware) ResetMetrics() {
	m.metrics.mu.Lock()
	defer m.metrics.mu.Unlock()

	m.metrics.BlockedRequests = 0
	m.metrics.SuspiciousRequests = 0
	m.metrics.BlockedIPs = make(map[string]int64)
	m.metrics.BlockedUserAgents = make(map[string]int64)
	m.metrics.LastReset = time.Now()
}
