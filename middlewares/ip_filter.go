package middlewares

import (
	"net"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// IPFilterConfig IP 필터 설정
type IPFilterConfig struct {
	AllowedIPs   []string // 허용된 IP 주소 목록
	AllowedCIDRs []string // 허용된 CIDR 범위 목록
	DenyAll      bool     // 기본적으로 모든 IP 차단 (화이트리스트 모드)
}

// IPFilterMiddleware IP 기반 접근 제어 미들웨어
func IPFilterMiddleware(config IPFilterConfig) fiber.Handler {
	// CIDR 파싱
	var allowedNetworks []*net.IPNet
	for _, cidr := range config.AllowedCIDRs {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			continue // 잘못된 CIDR 무시
		}
		allowedNetworks = append(allowedNetworks, network)
	}

	return func(c *fiber.Ctx) error {
		// 클라이언트 IP 추출
		clientIP := getClientIP(c)
		if clientIP == "" {
			if config.DenyAll {
				return c.Status(fiber.StatusForbidden).SendString("Access denied: IP not found")
			}
			return c.Next()
		}

		// IP 주소 파싱
		ip := net.ParseIP(clientIP)
		if ip == nil {
			if config.DenyAll {
				return c.Status(fiber.StatusForbidden).SendString("Access denied: Invalid IP")
			}
			return c.Next()
		}

		// 허용된 IP 목록 확인
		for _, allowedIP := range config.AllowedIPs {
			if clientIP == allowedIP {
				return c.Next()
			}
		}

		// CIDR 범위 확인
		for _, network := range allowedNetworks {
			if network.Contains(ip) {
				return c.Next()
			}
		}

		// DenyAll이 true이면 기본적으로 차단
		if config.DenyAll {
			return c.Status(fiber.StatusForbidden).SendString("Access denied: IP not allowed")
		}

		// DenyAll이 false이면 기본적으로 허용
		return c.Next()
	}
}

// getClientIP 클라이언트 IP 주소 추출
func getClientIP(c *fiber.Ctx) string {
	// X-Forwarded-For 헤더 확인 (프록시/로드밸런서 환경)
	if xff := c.Get("X-Forwarded-For"); xff != "" {
		// 첫 번째 IP 사용 (클라이언트 IP)
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// X-Real-IP 헤더 확인 (Nginx 등)
	if xri := c.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}

	// RemoteAddr 사용 (직접 연결)
	return c.IP()
}
