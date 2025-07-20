// Package verification provides package verification functionality
package verification

import (
	"context"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"hash"
	"strings"

	"github.com/gofiber/fiber/v2"

	"proxynd/alerts"
)

// HashType 해시 타입
type HashType string

// HashTypeSHA1 is exported
// HashTypeSHA1 represents a type identifier
const (
	HashTypeSHA1 HashType = "sha1"
	// HashTypeSHA256 is a const that hash type s h a256
	// HashTypeSHA512 is a const that hash type s h a512
	// HashTypeMD5 is a const that hash type m d5
	HashTypeSHA256 HashType = "sha256"
	HashTypeSHA512 HashType = "sha512"
	HashTypeMD5    HashType = "md5" // 보안상 권장하지 않음
)

// Result is an alias for VerificationResult to avoid package stuttering
type Result = VerificationResult

// VerificationResult 검증 결과
type VerificationResult struct {
	Valid        bool              `json:"valid"`
	HashType     HashType          `json:"hash_type"`
	ExpectedHash string            `json:"expected_hash"`
	ActualHash   string            `json:"actual_hash"`
	Message      string            `json:"message"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// PackageVerifier 패키지 검증기
type PackageVerifier struct {
	alertManager alerts.AlertManager
	strictMode   bool // 검증 실패 시 차단 여부
	config       *VerifierConfig
}

// VerifierConfig 검증기 설정
type VerifierConfig struct {
	StrictMode         bool                     `yaml:"strict_mode" json:"strict_mode"`
	BlockOnFailure     bool                     `yaml:"block_on_failure" json:"block_on_failure"`
	AlertOnFailure     bool                     `yaml:"alert_on_failure" json:"alert_on_failure"`
	PackageTypeConfigs map[string]PackageConfig `yaml:"package_types" json:"package_types"`
}

// PackageConfig 패키지 타입별 설정
type PackageConfig struct {
	Enabled          bool     `yaml:"enabled" json:"enabled"`
	RequiredHashes   []string `yaml:"required_hashes" json:"required_hashes"`
	TrustedSources   []string `yaml:"trusted_sources" json:"trusted_sources"`
	SkipVerification []string `yaml:"skip_verification" json:"skip_verification"` // 검증 제외 패턴
}

// NewPackageVerifier 새 패키지 검증기 생성
func NewPackageVerifier(config *VerifierConfig, alertManager alerts.AlertManager) *PackageVerifier {
	return &PackageVerifier{
		alertManager: alertManager,
		strictMode:   config.StrictMode,
		config:       config,
	}
}

// VerifyPackage 패키지 검증
func (pv *PackageVerifier) VerifyPackage(
	ctx context.Context,
	packageType string,
	packagePath string,
	data []byte,
	metadata map[string]string,
) (*VerificationResult, error) {
	// 패키지 타입별 설정 확인
	pkgConfig, exists := pv.config.PackageTypeConfigs[packageType]
	if !exists || !pkgConfig.Enabled {
		return &VerificationResult{
			Valid:   true,
			Message: "Package type verification not enabled",
		}, nil
	}

	// 검증 제외 패턴 확인
	if pv.shouldSkipVerification(packagePath, pkgConfig.SkipVerification) {
		return &VerificationResult{
			Valid:   true,
			Message: "Package verification skipped by configuration",
		}, nil
	}

	// 패키지 타입별 검증 수행
	var result *VerificationResult
	var err error

	switch packageType {
	case "npm":
		result, err = pv.verifyNpmPackage(ctx, packagePath, data, metadata)
	case "pip":
		result, err = pv.verifyPipPackage(ctx, packagePath, data, metadata)
	case "apt":
		result, err = pv.verifyAptPackage(ctx, packagePath, data, metadata)
	case "docker":
		result, err = pv.verifyDockerPackage(ctx, packagePath, data, metadata)
	case "maven":
		result, err = pv.verifyMavenPackage(ctx, packagePath, data, metadata)
	default:
		result = &VerificationResult{
			Valid:   true,
			Message: fmt.Sprintf("Unknown package type: %s", packageType),
		}
	}

	// 검증 실패 시 알림 전송
	if err != nil || (result != nil && !result.Valid) {
		if pv.config.AlertOnFailure && pv.alertManager != nil {
			pv.sendVerificationAlert(ctx, packageType, packagePath, result, metadata)
		}
	}

	return result, err
}

// verifyNpmPackage NPM 패키지 검증
func (pv *PackageVerifier) verifyNpmPackage(
	_ context.Context,
	_ string,
	data []byte,
	metadata map[string]string,
) (*VerificationResult, error) {
	// NPM integrity 필드 확인 (SHA512)
	integrity := metadata["npm-integrity"]
	if integrity == "" {
		return &VerificationResult{
			Valid:   true,
			Message: "No integrity field provided",
		}, nil
	}

	// integrity 형식: "sha512-..."
	if strings.HasPrefix(integrity, "sha512-") {
		expectedHash := strings.TrimPrefix(integrity, "sha512-")
		actualHash := pv.calculateHash(data, HashTypeSHA512)

		if actualHash != expectedHash {
			return &VerificationResult{
				Valid:        false,
				HashType:     HashTypeSHA512,
				ExpectedHash: expectedHash,
				ActualHash:   actualHash,
				Message:      "NPM package integrity check failed",
			}, nil
		}
	}

	return &VerificationResult{
		Valid:   true,
		Message: "NPM package verified successfully",
	}, nil
}

// verifyPipPackage PyPI 패키지 검증
func (pv *PackageVerifier) verifyPipPackage(
	_ context.Context,
	packagePath string,
	data []byte,
	metadata map[string]string,
) (*VerificationResult, error) {
	// PyPI는 주로 SHA256 사용
	expectedHash := metadata["sha256"]
	if expectedHash == "" {
		// URL에서 해시 추출 시도
		if strings.Contains(packagePath, "#sha256=") {
			parts := strings.Split(packagePath, "#sha256=")
			if len(parts) > 1 {
				expectedHash = parts[1]
			}
		}
	}

	if expectedHash == "" {
		return &VerificationResult{
			Valid:   true,
			Message: "No SHA256 hash provided",
		}, nil
	}

	actualHash := pv.calculateHash(data, HashTypeSHA256)
	if actualHash != expectedHash {
		return &VerificationResult{
			Valid:        false,
			HashType:     HashTypeSHA256,
			ExpectedHash: expectedHash,
			ActualHash:   actualHash,
			Message:      "PyPI package hash verification failed",
		}, nil
	}

	return &VerificationResult{
		Valid:    true,
		HashType: HashTypeSHA256,
		Message:  "PyPI package verified successfully",
	}, nil
}

// verifyAptPackage APT 패키지 검증
func (pv *PackageVerifier) verifyAptPackage(
	_ context.Context,
	_ string,
	data []byte,
	metadata map[string]string,
) (*VerificationResult, error) {
	// APT는 Release 파일의 SHA256 사용
	// NOTE: GPG 서명 검증 구현 필요

	expectedHash := metadata["sha256sum"]
	if expectedHash == "" {
		return &VerificationResult{
			Valid:   true,
			Message: "No SHA256 sum provided for APT package",
		}, nil
	}

	actualHash := pv.calculateHash(data, HashTypeSHA256)
	if actualHash != expectedHash {
		return &VerificationResult{
			Valid:        false,
			HashType:     HashTypeSHA256,
			ExpectedHash: expectedHash,
			ActualHash:   actualHash,
			Message:      "APT package checksum verification failed",
		}, nil
	}

	return &VerificationResult{
		Valid:    true,
		HashType: HashTypeSHA256,
		Message:  "APT package verified successfully",
	}, nil
}

// verifyDockerPackage Docker 이미지/레이어 검증
func (pv *PackageVerifier) verifyDockerPackage(
	_ context.Context,
	packagePath string,
	data []byte,
	metadata map[string]string,
) (*VerificationResult, error) {
	// Docker는 manifest의 digest 사용 (SHA256)
	digest := metadata["docker-content-digest"]
	if digest == "" {
		// 경로에서 digest 추출
		if strings.Contains(packagePath, "@sha256:") {
			parts := strings.Split(packagePath, "@sha256:")
			if len(parts) > 1 {
				digest = "sha256:" + parts[1]
			}
		}
	}

	if digest == "" {
		return &VerificationResult{
			Valid:   true,
			Message: "No content digest provided",
		}, nil
	}

	// "sha256:" 프리픽스 제거
	expectedHash := strings.TrimPrefix(digest, "sha256:")
	actualHash := pv.calculateHash(data, HashTypeSHA256)

	if actualHash != expectedHash {
		return &VerificationResult{
			Valid:        false,
			HashType:     HashTypeSHA256,
			ExpectedHash: expectedHash,
			ActualHash:   actualHash,
			Message:      "Docker content digest verification failed",
		}, nil
	}

	return &VerificationResult{
		Valid:    true,
		HashType: HashTypeSHA256,
		Message:  "Docker content verified successfully",
	}, nil
}

// verifyMavenPackage Maven 패키지 검증
func (pv *PackageVerifier) verifyMavenPackage(
	_ context.Context,
	_ string,
	data []byte,
	metadata map[string]string,
) (*VerificationResult, error) {
	// Maven은 SHA1이 기본, SHA256도 지원
	sha1Expected := metadata["sha1"]
	sha256Expected := metadata["sha256"]

	// SHA256 우선
	if sha256Expected != "" {
		actualHash := pv.calculateHash(data, HashTypeSHA256)
		if actualHash != sha256Expected {
			return &VerificationResult{
				Valid:        false,
				HashType:     HashTypeSHA256,
				ExpectedHash: sha256Expected,
				ActualHash:   actualHash,
				Message:      "Maven artifact SHA256 verification failed",
			}, nil
		}
		return &VerificationResult{
			Valid:    true,
			HashType: HashTypeSHA256,
			Message:  "Maven artifact verified successfully",
		}, nil
	}

	// SHA1 검증
	if sha1Expected != "" {
		actualHash := pv.calculateHash(data, HashTypeSHA1)
		if actualHash != sha1Expected {
			return &VerificationResult{
				Valid:        false,
				HashType:     HashTypeSHA1,
				ExpectedHash: sha1Expected,
				ActualHash:   actualHash,
				Message:      "Maven artifact SHA1 verification failed",
			}, nil
		}
		return &VerificationResult{
			Valid:    true,
			HashType: HashTypeSHA1,
			Message:  "Maven artifact verified successfully",
		}, nil
	}

	return &VerificationResult{
		Valid:   true,
		Message: "No checksum provided for Maven artifact",
	}, nil
}

// calculateHash 해시 계산
func (pv *PackageVerifier) calculateHash(data []byte, hashType HashType) string {
	var h hash.Hash

	switch hashType {
	case HashTypeSHA1:
		h = sha1.New()
	case HashTypeSHA256:
		h = sha256.New()
	case HashTypeSHA512:
		h = sha512.New()
	default:
		return ""
	}

	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

// shouldSkipVerification 검증 제외 여부 확인
func (pv *PackageVerifier) shouldSkipVerification(path string, skipPatterns []string) bool {
	for _, pattern := range skipPatterns {
		if strings.Contains(path, pattern) {
			return true
		}
	}
	return false
}

// sendVerificationAlert 검증 실패 알림 전송
func (pv *PackageVerifier) sendVerificationAlert(
	ctx context.Context,
	packageType string,
	packagePath string,
	result *VerificationResult,
	metadata map[string]string,
) {
	// 패키지 정보 추출
	packageInfo := &alerts.PackageInfo{
		Type:    packageType,
		Path:    packagePath,
		Name:    extractPackageName(packagePath, packageType),
		Version: metadata["version"],
	}

	if result != nil {
		packageInfo.ExpectedHash = result.ExpectedHash
		packageInfo.ActualHash = result.ActualHash
	}

	// 알림 이벤트 생성
	event := &alerts.AlertEvent{
		Level:       alerts.AlertLevelCritical,
		Type:        "package_verification_failed",
		Title:       "Package Verification Failed",
		Message:     fmt.Sprintf("Verification failed for %s package: %s", packageType, packagePath),
		Source:      "package_verifier",
		PackageInfo: packageInfo,
		Metadata: map[string]interface{}{
			"strict_mode":         pv.strictMode,
			"verification_result": result,
		},
	}

	// 알림 전송
	if err := pv.alertManager.Send(ctx, event); err != nil {
		// 알림 전송 실패는 로그만 남기고 계속 진행
		fmt.Printf("Failed to send verification alert: %v\n", err)
	}
}

// extractPackageName 패키지 경로에서 이름 추출
func extractPackageName(path string, packageType string) string {
	switch packageType {
	case "npm":
		// @scope/package 형식 처리
		parts := strings.Split(path, "/")
		if len(parts) >= 2 {
			if strings.HasPrefix(parts[0], "@") {
				return parts[0] + "/" + parts[1]
			}
			return parts[0]
		}
	case "pip":
		// package-name-version.whl 형식
		parts := strings.Split(path, "/")
		if len(parts) > 0 {
			filename := parts[len(parts)-1]
			dashParts := strings.Split(filename, "-")
			if len(dashParts) > 0 {
				return dashParts[0]
			}
		}
	case "maven":
		// groupId/artifactId/version/file 형식
		parts := strings.Split(path, "/")
		if len(parts) >= 3 {
			return parts[len(parts)-3]
		}
	}

	// 기본: 마지막 경로 요소
	parts := strings.Split(path, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return path
}

// Middleware is an alias for VerificationMiddleware to avoid package stuttering
var Middleware = VerificationMiddleware

// VerificationMiddleware Fiber 미들웨어로 패키지 검증 통합
func VerificationMiddleware(verifier *PackageVerifier) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// 프록시 요청인지 확인
		if !strings.HasPrefix(c.Path(), "/proxy/") {
			return c.Next()
		}

		// 패키지 타입과 경로 추출
		proxyType := c.Params("type")
		packagePath := c.Params("*")

		if proxyType == "" || packagePath == "" {
			return c.Next()
		}

		// 다운로드 응답인 경우 검증 수행
		if c.Method() == "GET" && c.Response().StatusCode() == fiber.StatusOK {
			// 응답 본문 읽기
			body := c.Response().Body()
			if len(body) > 0 {
				// 메타데이터 수집
				metadata := make(map[string]string)
				metadata["content-type"] = string(c.Response().Header.ContentType())
				metadata["etag"] = string(c.Response().Header.Peek("ETag"))
				metadata["docker-content-digest"] = string(c.Response().Header.Peek("Docker-Content-Digest"))

				// 검증 수행
				ctx := c.Context()
				result, err := verifier.VerifyPackage(ctx, proxyType, packagePath, body, metadata)

				if err != nil || (result != nil && !result.Valid) {
					// 엄격 모드인 경우 차단
					if verifier.config.BlockOnFailure {
						return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
							"error":   "Package verification failed",
							"details": result,
						})
					}
				}

				// 검증 결과를 헤더에 추가
				if result != nil {
					c.Response().Header.Set("X-Package-Verified", fmt.Sprintf("%v", result.Valid))
					if result.HashType != "" {
						c.Response().Header.Set("X-Verified-Hash-Type", string(result.HashType))
					}
				}
			}
		}

		return c.Next()
	}
}
