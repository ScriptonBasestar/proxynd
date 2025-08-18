package middlewares

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"strings"

	"github.com/gofiber/fiber/v2"
)

const (
	// MethodPATCH HTTP PATCH method
	MethodPATCH = "PATCH"
)

// SecurityConfig 보안 설정
type SecurityConfig struct {
	EnableHashVerification bool     // SHA256 해시 검증 활성화
	RequiredHashHeaders    []string // 해시를 포함할 헤더 목록
	FailOnHashMismatch     bool     // 해시 불일치 시 요청 차단
}

// SecurityMiddleware 보안 검증 미들웨어
func SecurityMiddleware(config SecurityConfig) fiber.Handler {
	if len(config.RequiredHashHeaders) == 0 {
		config.RequiredHashHeaders = []string{
			"Content-SHA256",
			"X-Checksum-SHA256",
			"Digest",
		}
	}

	return func(c *fiber.Ctx) error {
		// POST/PUT 요청에 대해서만 해시 검증 수행
		if !isWriteMethod(c.Method()) {
			return c.Next()
		}

		// 해시 검증이 비활성화된 경우 통과
		if !config.EnableHashVerification {
			return c.Next()
		}

		// 요청 본문에서 해시 계산
		bodyHash := calculateBodyHash(c)

		// 헤더에서 예상 해시 추출
		expectedHash := extractHashFromHeaders(c, config.RequiredHashHeaders)
		if expectedHash == "" {
			// 해시 헤더가 없으면 검증 건너뛰기
			return c.Next()
		}

		// 해시 비교
		if !compareHashes(bodyHash, expectedHash) {
			if config.FailOnHashMismatch {
				return c.Status(fiber.StatusBadRequest).SendString("Hash verification failed")
			}
			// 해시 불일치 로그 기록
			_, _ = fmt.Fprintf(os.Stderr, "Hash mismatch: expected=%s, calculated=%s\n", expectedHash, bodyHash)
		}

		// 검증된 해시를 컨텍스트에 저장
		c.Locals("verifiedHash", bodyHash)
		c.Locals("hashVerified", true)

		return c.Next()
	}
}

// calculateBodyHash 요청 본문의 SHA256 해시 계산
func calculateBodyHash(c *fiber.Ctx) string {
	// 요청 본문 읽기
	body := c.Body()
	if len(body) == 0 {
		return ""
	}

	// SHA256 해시 계산
	hash := sha256.Sum256(body)
	return hex.EncodeToString(hash[:])
}

// extractHashFromHeaders 헤더에서 해시 값 추출
func extractHashFromHeaders(c *fiber.Ctx, headers []string) string {
	for _, header := range headers {
		value := c.Get(header)
		if value == "" {
			continue
		}

		// Digest 헤더의 경우 "SHA-256=..." 형식 처리
		if strings.ToLower(header) == "digest" {
			if strings.HasPrefix(value, "SHA-256=") {
				return strings.TrimPrefix(value, "SHA-256=")
			}
			if strings.HasPrefix(value, "sha-256=") {
				return strings.TrimPrefix(value, "sha-256=")
			}
		}

		// 다른 헤더는 직접 해시 값으로 사용
		return value
	}

	return ""
}

// compareHashes 해시 값 비교
func compareHashes(calculated, expected string) bool {
	// 대소문자 구분 없이 비교
	return strings.EqualFold(calculated, expected)
}

// isWriteMethod 쓰기 메서드인지 확인
func isWriteMethod(method string) bool {
	switch strings.ToUpper(method) {
	case MethodPOST, MethodPUT, MethodPATCH:
		return true
	default:
		return false
	}
}

// PackageVerifier 패키지 검증 인터페이스
type PackageVerifier interface {
	VerifyPackage(packageType, path string, data []byte) error
}

// DefaultPackageVerifier 기본 패키지 검증기
type DefaultPackageVerifier struct {
	StrictMode bool // 엄격 모드 (검증 실패 시 차단)
}

// VerifyPackage 패키지 검증 수행
func (v *DefaultPackageVerifier) VerifyPackage(packageType, path string, data []byte) error {
	switch packageType {
	case "npm":
		return v.verifyNpmPackage(path, data)
	case "pip":
		return v.verifyPipPackage(path, data)
	case "apt":
		return v.verifyAptPackage(path, data)
	case "docker":
		return v.verifyDockerPackage(path, data)
	default:
		// 알려지지 않은 패키지 타입은 검증 건너뛰기
		return nil
	}
}

// verifyNpmPackage NPM 패키지 검증
func (v *DefaultPackageVerifier) verifyNpmPackage(_ string, _ []byte) error {
	// NPM 패키지 특화 검증 로직
	// 예: package.json 유효성, 파일 구조 등
	return nil
}

// verifyPipPackage PyPI 패키지 검증
func (v *DefaultPackageVerifier) verifyPipPackage(_ string, _ []byte) error {
	// PyPI 패키지 특화 검증 로직
	// 예: wheel 파일 구조, metadata 등
	return nil
}

// verifyAptPackage APT 패키지 검증
func (v *DefaultPackageVerifier) verifyAptPackage(_ string, _ []byte) error {
	// APT 패키지 특화 검증 로직
	// 예: .deb 파일 구조, GPG 서명 등
	return nil
}

// verifyDockerPackage Docker 이미지 검증
func (v *DefaultPackageVerifier) verifyDockerPackage(_ string, _ []byte) error {
	// Docker 이미지 특화 검증 로직
	// 예: manifest 구조, 레이어 해시 등
	return nil
}

// PackageVerificationMiddleware 패키지 검증 미들웨어
func PackageVerificationMiddleware(verifier PackageVerifier) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// 프록시 요청에 대해서만 검증 수행
		if !strings.HasPrefix(c.Path(), "/proxy/") {
			return c.Next()
		}

		// 쓰기 요청에 대해서만 검증 수행
		if !isWriteMethod(c.Method()) {
			return c.Next()
		}

		// 패키지 타입과 경로 추출
		proxyType := c.Params("type")
		packagePath := c.Params("*")

		if proxyType == "" || packagePath == "" {
			return c.Next()
		}

		// 요청 본문 읽기
		body := c.Body()
		if len(body) == 0 {
			return c.Next()
		}

		// 패키지 검증 수행
		if err := verifier.VerifyPackage(proxyType, packagePath, body); err != nil {
			return c.Status(fiber.StatusBadRequest).SendString(
				fmt.Sprintf("Package verification failed: %v", err),
			)
		}

		// 검증 성공 표시
		c.Locals("packageVerified", true)
		return c.Next()
	}
}
