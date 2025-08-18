package middlewares

import (
	"strings"

	"github.com/gofiber/fiber/v2"
)

// Permission 권한 타입
type Permission int

const (
	// PermissionRead 읽기 권한
	PermissionRead Permission = 1 << iota
	// PermissionWrite 쓰기 권한
	PermissionWrite
	// PermissionDelete 삭제 권한
	PermissionDelete
)

// UserPermissions 사용자별 권한 설정
type UserPermissions struct {
	Username    string
	Permissions Permission
}

// PermissionConfig 권한 설정
type PermissionConfig struct {
	UserPermissions   map[string]Permission // username -> permissions 매핑
	DefaultPermission Permission            // 기본 권한 (인증되지 않은 사용자)
}

// LegacyPermissionMiddleware 사용자별 권한 체크 미들웨어 (legacy)
func LegacyPermissionMiddleware(config PermissionConfig) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// HTTP 메서드에 따른 필요 권한 결정
		requiredPermission := getRequiredPermission(c.Method())

		// 인증된 사용자명 확인
		username, ok := c.Locals("username").(string)
		if !ok || username == "" {
			// 인증되지 않은 사용자 - 기본 권한 사용
			if !hasPermission(config.DefaultPermission, requiredPermission) {
				return c.Status(fiber.StatusForbidden).SendString("Access denied: Authentication required")
			}
			return c.Next()
		}

		// 사용자별 권한 확인
		userPermission, exists := config.UserPermissions[username]
		if !exists {
			// 사용자별 권한이 없으면 기본 권한 사용
			userPermission = config.DefaultPermission
		}

		// 권한 체크
		if !hasPermission(userPermission, requiredPermission) {
			return c.Status(fiber.StatusForbidden).SendString("Access denied: Insufficient permissions")
		}

		// 권한 정보를 컨텍스트에 저장
		c.Locals("userPermissions", userPermission)
		return c.Next()
	}
}

// getRequiredPermission HTTP 메서드에 따른 필요 권한 반환
func getRequiredPermission(method string) Permission {
	switch strings.ToUpper(method) {
	case HTTPMethodGET, HTTPMethodHEAD:
		return PermissionRead
	case HTTPMethodPOST, HTTPMethodPUT, HTTPMethodPATCH:
		return PermissionWrite
	case HTTPMethodDELETE:
		return PermissionDelete
	default:
		return PermissionRead // 기본값
	}
}

// hasPermission 권한 보유 여부 확인
func hasPermission(userPermission, requiredPermission Permission) bool {
	return userPermission&requiredPermission != 0
}

// PackageFilter 패키지 필터 설정
type PackageFilter struct {
	AllowedPackages map[string][]string // proxyType -> package patterns
	DenyAll         bool                // 기본적으로 모든 패키지 차단
}

// PackageFilterMiddleware 허용된 패키지 목록 기반 필터 미들웨어
func PackageFilterMiddleware(config PackageFilter) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// 프록시 타입 및 패키지 경로 추출
		proxyType := c.Params("type")
		packagePath := c.Params("*")

		if proxyType == "" || packagePath == "" {
			if config.DenyAll {
				return c.Status(fiber.StatusForbidden).SendString("Access denied: Invalid package request")
			}
			return c.Next()
		}

		// 허용된 패키지 패턴 확인
		allowedPatterns, exists := config.AllowedPackages[proxyType]
		if !exists {
			if config.DenyAll {
				return c.Status(fiber.StatusForbidden).SendString("Access denied: Package type not allowed")
			}
			return c.Next()
		}

		// 패키지 경로가 허용된 패턴과 일치하는지 확인
		for _, pattern := range allowedPatterns {
			if matchesPattern(packagePath, pattern) {
				return c.Next()
			}
		}

		// 일치하는 패턴이 없으면 차단
		if config.DenyAll {
			return c.Status(fiber.StatusForbidden).SendString("Access denied: Package not allowed")
		}

		return c.Next()
	}
}

// matchesPattern 패키지 경로가 패턴과 일치하는지 확인
func matchesPattern(path, pattern string) bool {
	// 간단한 패턴 매칭 (와일드카드 지원)
	if pattern == "*" {
		return true
	}

	if strings.Contains(pattern, "*") {
		// 와일드카드 패턴 처리
		parts := strings.Split(pattern, "*")
		if len(parts) == 2 {
			prefix, suffix := parts[0], parts[1]
			return strings.HasPrefix(path, prefix) && strings.HasSuffix(path, suffix)
		}
	}

	// 정확한 일치 또는 프리픽스 일치
	return path == pattern || strings.HasPrefix(path, pattern+"/")
}
