package middlewares

import (
	"fmt"
	"net"
	"strings"
	"time"
	
	"github.com/gofiber/fiber/v2"
	"github.com/ulule/limiter/v3"
	"github.com/ulule/limiter/v3/drivers/store/memory"
	
	"proxynd/logging"
)

// EnhancedRateLimitConfig 개선된 Rate Limit 설정
type EnhancedRateLimitConfig struct {
	// 기본 설정
	Rate             string        `json:"rate"`              // "100-M" (100 requests per minute)
	Burst            int           `json:"burst"`             // 버스트 허용량
	KeyGenerator     func(*fiber.Ctx) string `json:"-"`       // 키 생성 함수
	SkipHandler      func(*fiber.Ctx) bool   `json:"-"`       // 스킵 조건
	ErrorHandler     fiber.ErrorHandler      `json:"-"`       // 에러 핸들러
	
	// 고급 설정
	TrustedProxies   []string      `json:"trusted_proxies"`   // 신뢰할 수 있는 프록시
	WhitelistIPs     []string      `json:"whitelist_ips"`     // 화이트리스트 IP
	BlacklistIPs     []string      `json:"blacklist_ips"`     // 블랙리스트 IP
	EnableLogging    bool          `json:"enable_logging"`    // 로깅 활성화
	
	// 경로별 설정
	PathConfigs      map[string]PathRateConfig `json:"path_configs"` // 경로별 개별 설정
}

// PathRateConfig 경로별 Rate Limit 설정
type PathRateConfig struct {
	Rate  string `json:"rate"`  // 해당 경로의 Rate Limit
	Burst int    `json:"burst"` // 해당 경로의 Burst
}

// DefaultEnhancedRateLimitConfig 기본 개선된 Rate Limit 설정
func DefaultEnhancedRateLimitConfig() EnhancedRateLimitConfig {
	return EnhancedRateLimitConfig{
		Rate:  "1000-H", // 시간당 1000개 요청
		Burst: 100,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP() // IP 기반 제한
		},
		SkipHandler: func(c *fiber.Ctx) bool {
			// 헬스체크 및 메트릭 엔드포인트는 스킵
			return strings.HasPrefix(c.Path(), "/health") || 
				   strings.HasPrefix(c.Path(), "/metrics")
		},
		ErrorHandler: func(c *fiber.Ctx, _ error) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error":       "Too many requests",
				"retry_after": time.Now().Add(time.Minute).Unix(),
				"limit":       "API rate limit exceeded",
			})
		},
		TrustedProxies: []string{"127.0.0.1", "::1"},
		WhitelistIPs:   []string{},
		BlacklistIPs:   []string{},
		EnableLogging:  true,
		PathConfigs: map[string]PathRateConfig{
			"/auth":   {Rate: "5-M", Burst: 2},    // 인증: 분당 5개 요청
			"/api":    {Rate: "100-M", Burst: 20}, // API: 분당 100개 요청  
			"/proxy":  {Rate: "500-M", Burst: 50}, // 프록시: 분당 500개 요청
		},
	}
}

// NewEnhancedRateLimiter 개선된 Rate Limiter 생성
func NewEnhancedRateLimiter(config ...EnhancedRateLimitConfig) fiber.Handler {
	cfg := DefaultEnhancedRateLimitConfig()
	if len(config) > 0 {
		cfg = config[0]
		// ErrorHandler가 nil인 경우 기본값 설정
		if cfg.ErrorHandler == nil {
			cfg.ErrorHandler = DefaultEnhancedRateLimitConfig().ErrorHandler
		}
		// KeyGenerator가 nil인 경우 기본값 설정
		if cfg.KeyGenerator == nil {
			cfg.KeyGenerator = DefaultEnhancedRateLimitConfig().KeyGenerator
		}
	}
	
	logger := logging.GetLogger()
	
	// 메모리 스토어 생성
	store := memory.NewStore()
	
	// 기본 Rate 파싱
	defaultRate, err := limiter.NewRateFromFormatted(cfg.Rate)
	if err != nil {
		logger.Error("Invalid default rate format, using fallback",
			logging.F("error", err),
			logging.F("rate", cfg.Rate))
		// 기본값으로 fallback (1000 requests per hour)
		defaultRate = limiter.Rate{
			Period: time.Hour,
			Limit:  1000,
		}
	}
	
	// 기본 리미터 생성
	defaultLimiter := limiter.New(store, defaultRate)
	
	// 경로별 리미터 생성
	pathLimiters := make(map[string]*limiter.Limiter)
	for path, pathConfig := range cfg.PathConfigs {
		rate, err := limiter.NewRateFromFormatted(pathConfig.Rate)
		if err != nil {
			logger.Error("Invalid path rate format",
				logging.F("path", path),
				logging.F("rate", pathConfig.Rate),
				logging.F("error", err))
			continue
		}
		pathLimiters[path] = limiter.New(store, rate)
	}
	
	return func(c *fiber.Ctx) error {
		// 스킵 조건 확인
		if cfg.SkipHandler != nil && cfg.SkipHandler(c) {
			return c.Next()
		}
		
		// IP 주소 가져오기
		ip := getRealIP(c, cfg.TrustedProxies)
		
		// 블랙리스트 확인
		if isIPInList(ip, cfg.BlacklistIPs) {
			if cfg.EnableLogging {
				logger.Warn("Blocked request from blacklisted IP",
					logging.F("ip", ip),
					logging.F("path", c.Path()),
					logging.F("user_agent", c.Get("User-Agent")))
			}
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Access denied",
			})
		}
		
		// 화이트리스트 확인
		if len(cfg.WhitelistIPs) > 0 && !isIPInList(ip, cfg.WhitelistIPs) {
			if cfg.EnableLogging {
				logger.Warn("Blocked request from non-whitelisted IP",
					logging.F("ip", ip),
					logging.F("path", c.Path()))
			}
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Access denied",
			})
		}
		
		// 키 생성
		key := cfg.KeyGenerator(c)
		
		// 경로별 리미터 선택
		selectedLimiter := defaultLimiter
		for pathPrefix, pathLimiter := range pathLimiters {
			if strings.HasPrefix(c.Path(), pathPrefix) {
				selectedLimiter = pathLimiter
				break
			}
		}
		
		// Rate Limit 확인
		context, err := selectedLimiter.Get(c.Context(), key)
		if err != nil {
			logger.Error("Rate limiter error",
				logging.F("key", key),
				logging.F("error", err))
			return fiber.NewError(fiber.StatusInternalServerError, "Rate limiter error")
		}
		
		// Rate Limit 헤더 설정
		c.Set("X-RateLimit-Limit", fmt.Sprintf("%d", context.Limit))
		c.Set("X-RateLimit-Remaining", fmt.Sprintf("%d", context.Remaining))
		c.Set("X-RateLimit-Reset", fmt.Sprintf("%d", context.Reset))
		
		// Rate Limit 도달 확인
		if context.Reached {
			if cfg.EnableLogging {
				logger.Warn("Rate limit exceeded",
					logging.F("ip", ip),
					logging.F("key", key),
					logging.F("path", c.Path()),
					logging.F("limit", context.Limit),
					logging.F("reset", context.Reset))
			}
			
			return cfg.ErrorHandler(c, fiber.ErrTooManyRequests)
		}
		
		// 성공 로깅 (디버그 레벨)
		if cfg.EnableLogging {
			logger.Debug("Rate limit check passed",
				logging.F("ip", ip),
				logging.F("key", key),
				logging.F("path", c.Path()),
				logging.F("remaining", context.Remaining))
		}
		
		return c.Next()
	}
}

// getRealIP 실제 IP 주소 가져오기 (프록시 고려)
func getRealIP(c *fiber.Ctx, trustedProxies []string) string {
	// 직접 연결인 경우
	if len(trustedProxies) == 0 {
		return c.IP()
	}
	
	// 신뢰할 수 있는 프록시인지 확인
	clientIP := c.IP()
	if !isIPInList(clientIP, trustedProxies) {
		return clientIP
	}
	
	// X-Forwarded-For 헤더 확인
	forwarded := c.Get("X-Forwarded-For")
	if forwarded != "" {
		// 첫 번째 IP 사용 (클라이언트 IP)
		ips := strings.Split(forwarded, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}
	
	// X-Real-IP 헤더 확인
	realIP := c.Get("X-Real-IP")
	if realIP != "" {
		return realIP
	}
	
	return clientIP
}

// isIPInList IP가 리스트에 있는지 확인
func isIPInList(ip string, list []string) bool {
	clientIP := net.ParseIP(ip)
	if clientIP == nil {
		return false
	}
	
	for _, allowedIP := range list {
		// CIDR 형식 확인
		if strings.Contains(allowedIP, "/") {
			_, ipNet, err := net.ParseCIDR(allowedIP)
			if err == nil && ipNet.Contains(clientIP) {
				return true
			}
		} else if allowedIP == ip {
			// 단일 IP 확인
			return true
		}
	}
	
	return false
}

// AuthRateLimiter 인증 전용 Rate Limiter (매우 엄격)
func AuthRateLimiter() fiber.Handler {
	return NewEnhancedRateLimiter(EnhancedRateLimitConfig{
		Rate:  "5-M", // 분당 5개 요청만 허용
		Burst: 2,     // 버스트 2개
		KeyGenerator: func(c *fiber.Ctx) string {
			// IP + User-Agent 조합으로 더 정확한 제한
			return c.IP() + ":" + c.Get("User-Agent")
		},
		ErrorHandler: func(c *fiber.Ctx, _ error) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error":       "Authentication rate limit exceeded",
				"message":     "Too many authentication attempts. Please try again later.",
				"retry_after": time.Now().Add(time.Minute).Unix(),
			})
		},
		EnableLogging: true,
	})
}

// ProxyRateLimiter 프록시 전용 Rate Limiter (관대함)
func ProxyRateLimiter() fiber.Handler {
	return NewEnhancedRateLimiter(EnhancedRateLimitConfig{
		Rate:  "500-M", // 분당 500개 요청 허용
		Burst: 50,      // 버스트 50개
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
		SkipHandler: func(c *fiber.Ctx) bool {
			// 특정 패키지 타입은 더 관대하게
			return strings.Contains(c.Path(), "/proxy/apt") ||
				   strings.Contains(c.Path(), "/proxy/maven")
		},
		EnableLogging: true,
	})
}