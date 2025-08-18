package middlewares

import (
	"fmt"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"

	"proxynd/internal/logging"
)

// RateLimiter 요청 속도 제한기
type RateLimiter struct {
	requests map[string][]time.Time
	mutex    sync.RWMutex
	limit    int
	window   time.Duration
	logger   logging.Logger
}

// RateLimitConfig Rate Limiting 설정
type RateLimitConfig struct {
	RequestsPerWindow int           `json:"requests_per_window"` // 윈도우당 허용 요청 수
	WindowSize        time.Duration `json:"window_size"`         // 시간 윈도우 크기
	SkipSuccessful    bool          `json:"skip_successful"`     // 성공한 요청은 카운트에서 제외
	EnableLogging     bool          `json:"enable_logging"`      // 로깅 활성화
	TrustedProxies    []string      `json:"trusted_proxies"`     // 신뢰할 수 있는 프록시 IP 목록
}

// DefaultRateLimitConfig 기본 Rate Limit 설정
func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		RequestsPerWindow: 60, // 분당 60개 요청
		WindowSize:        time.Minute,
		SkipSuccessful:    false,
		EnableLogging:     true,
		TrustedProxies:    []string{"127.0.0.1", "::1"}, // localhost만 기본 신뢰
	}
}

// StrictRateLimitConfig 엄격한 Rate Limit 설정
func StrictRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		RequestsPerWindow: 30, // 분당 30개 요청
		WindowSize:        time.Minute,
		SkipSuccessful:    false,
		EnableLogging:     true,
		TrustedProxies:    []string{"127.0.0.1", "::1"},
	}
}

// NewRateLimiter Rate Limiter 생성
func NewRateLimiter(config RateLimitConfig) *RateLimiter {
	rl := &RateLimiter{
		requests: make(map[string][]time.Time),
		limit:    config.RequestsPerWindow,
		window:   config.WindowSize,
		logger:   logging.GetLogger(),
	}

	// 주기적으로 오래된 요청 기록 정리
	go rl.cleanup()

	return rl
}

// cleanup 오래된 요청 기록 정리
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(rl.window / 2) // 윈도우 크기의 절반마다 정리
	defer ticker.Stop()

	for range ticker.C {
		rl.mutex.Lock()
		now := time.Now()
		for ip, times := range rl.requests {
			// 윈도우 밖의 요청들 제거
			var validTimes []time.Time
			for _, t := range times {
				if now.Sub(t) < rl.window {
					validTimes = append(validTimes, t)
				}
			}

			if len(validTimes) == 0 {
				delete(rl.requests, ip)
			} else {
				rl.requests[ip] = validTimes
			}
		}

		// 메모리 사용량 로깅 (디버그용)
		if len(rl.requests) > 1000 {
			rl.logger.Warn("Rate limiter memory usage high",
				logging.F("tracked_ips", len(rl.requests)))
		}

		rl.mutex.Unlock()
	}
}

// isAllowed 요청이 허용되는지 확인
func (rl *RateLimiter) isAllowed(ip string) (bool, int, time.Time) {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	now := time.Now()
	times := rl.requests[ip]

	// 윈도우 내의 요청만 카운트
	var validTimes []time.Time
	for _, t := range times {
		if now.Sub(t) < rl.window {
			validTimes = append(validTimes, t)
		}
	}

	// 제한 확인
	if len(validTimes) >= rl.limit {
		// 가장 오래된 요청이 만료되는 시간 계산
		oldestRequest := validTimes[0]
		retryAfter := oldestRequest.Add(rl.window)
		return false, len(validTimes), retryAfter
	}

	// 새 요청 기록
	validTimes = append(validTimes, now)
	rl.requests[ip] = validTimes

	return true, len(validTimes), time.Time{}
}

// getClientIP 실제 클라이언트 IP 추출
func getClientIP(c *fiber.Ctx, trustedProxies []string) string {
	// X-Forwarded-For 헤더 확인 (신뢰할 수 있는 프록시에서만)
	if forwardedFor := c.Get("X-Forwarded-For"); forwardedFor != "" {
		clientIP := c.IP()
		// 현재 연결이 신뢰할 수 있는 프록시인지 확인
		for _, trustedIP := range trustedProxies {
			if clientIP == trustedIP {
				// 첫 번째 IP 사용 (체인의 원본 클라이언트)
				if firstIP := forwardedFor; firstIP != "" {
					if commaIndex := len(firstIP); commaIndex > 0 {
						return firstIP[:commaIndex]
					}
					return firstIP
				}
				break
			}
		}
	}

	// X-Real-IP 헤더 확인
	if realIP := c.Get("X-Real-IP"); realIP != "" {
		return realIP
	}

	// 기본적으로 직접 연결 IP 사용
	return c.IP()
}

// RateLimit Rate Limiting 미들웨어
func RateLimit(config ...RateLimitConfig) fiber.Handler {
	cfg := DefaultRateLimitConfig()
	if len(config) > 0 {
		cfg = config[0]
	}

	limiter := NewRateLimiter(cfg)
	logger := logging.GetLogger()

	return func(c *fiber.Ctx) error {
		// 클라이언트 IP 추출
		clientIP := getClientIP(c, cfg.TrustedProxies)

		// Rate limit 검사
		allowed, currentCount, retryAfter := limiter.isAllowed(clientIP)

		if !allowed {
			// Rate limit 초과
			retryAfterSeconds := int(time.Until(retryAfter).Seconds())
			if retryAfterSeconds <= 0 {
				retryAfterSeconds = int(cfg.WindowSize.Seconds())
			}

			if cfg.EnableLogging {
				logger.Warn("Rate limit exceeded",
					logging.F("client_ip", clientIP),
					logging.F("current_count", currentCount),
					logging.F("limit", cfg.RequestsPerWindow),
					logging.F("window_seconds", int(cfg.WindowSize.Seconds())),
					logging.F("path", c.Path()),
					logging.F("user_agent", c.Get("User-Agent")))
			}

			// Rate limit 관련 헤더 추가
			c.Set("X-RateLimit-Limit", fmt.Sprintf("%d", cfg.RequestsPerWindow))
			c.Set("X-RateLimit-Remaining", "0")
			c.Set("X-RateLimit-Reset", fmt.Sprintf("%d", retryAfter.Unix()))
			c.Set("Retry-After", fmt.Sprintf("%d", retryAfterSeconds))

			return c.Status(429).JSON(fiber.Map{
				"error":       "Rate limit exceeded",
				"retry_after": retryAfterSeconds,
				"limit":       cfg.RequestsPerWindow,
				"window":      int(cfg.WindowSize.Seconds()),
			})
		}

		// Rate limit 정보 헤더 추가 (성공 시)
		remaining := cfg.RequestsPerWindow - currentCount
		c.Set("X-RateLimit-Limit", fmt.Sprintf("%d", cfg.RequestsPerWindow))
		c.Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
		c.Set("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(cfg.WindowSize).Unix()))

		// 요청 처리
		err := c.Next()

		// 성공한 요청은 카운트에서 제외하는 옵션
		if cfg.SkipSuccessful && err == nil && c.Response().StatusCode() < 400 {
			limiter.mutex.Lock()
			if times := limiter.requests[clientIP]; len(times) > 0 {
				// 마지막 요청 (방금 기록한 것) 제거
				limiter.requests[clientIP] = times[:len(times)-1]
			}
			limiter.mutex.Unlock()
		}

		return err
	}
}

// BurstRateLimit 버스트 요청을 허용하는 Rate Limiter
func BurstRateLimit(normalLimit, burstLimit int, windowSize time.Duration) fiber.Handler {
	normalLimiter := NewRateLimiter(RateLimitConfig{
		RequestsPerWindow: normalLimit,
		WindowSize:        windowSize,
		EnableLogging:     true,
	})

	burstLimiter := NewRateLimiter(RateLimitConfig{
		RequestsPerWindow: burstLimit,
		WindowSize:        windowSize / 10, // 더 짧은 윈도우로 버스트 제한
		EnableLogging:     true,
	})

	logger := logging.GetLogger()

	return func(c *fiber.Ctx) error {
		clientIP := getClientIP(c, []string{"127.0.0.1", "::1"})

		// 먼저 버스트 제한 확인
		burstAllowed, burstCount, burstRetryAfter := burstLimiter.isAllowed(clientIP)
		if !burstAllowed {
			logger.Warn("Burst rate limit exceeded",
				logging.F("client_ip", clientIP),
				logging.F("burst_count", burstCount),
				logging.F("burst_limit", burstLimit))

			return c.Status(429).JSON(fiber.Map{
				"error":       "Burst rate limit exceeded",
				"retry_after": int(time.Until(burstRetryAfter).Seconds()),
				"type":        "burst",
			})
		}

		// 그 다음 일반 제한 확인
		normalAllowed, normalCount, normalRetryAfter := normalLimiter.isAllowed(clientIP)
		if !normalAllowed {
			logger.Warn("Normal rate limit exceeded",
				logging.F("client_ip", clientIP),
				logging.F("normal_count", normalCount),
				logging.F("normal_limit", normalLimit))

			return c.Status(429).JSON(fiber.Map{
				"error":       "Rate limit exceeded",
				"retry_after": int(time.Until(normalRetryAfter).Seconds()),
				"type":        "normal",
			})
		}

		return c.Next()
	}
}
