package proxy

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/gofiber/fiber/v2"
	"proxynd/alerts"
	"proxynd/configs"
	"proxynd/verification"
)

// VerificationHandler 검증 핸들러
type VerificationHandler struct {
	verifier     *verification.PackageVerifier
	alertManager alerts.AlertManager
	config       *configs.GlobalConfig
}

// NewVerificationHandler 새 검증 핸들러 생성
func NewVerificationHandler(
	globalConfig *configs.GlobalConfig,
	alertManager alerts.AlertManager,
) *VerificationHandler {
	// 검증 설정 로드
	verifierConfig := loadVerifierConfig(globalConfig)

	// 패키지 검증기 생성
	verifier := verification.NewPackageVerifier(verifierConfig, alertManager)

	return &VerificationHandler{
		verifier:     verifier,
		alertManager: alertManager,
		config:       globalConfig,
	}
}

// VerifyDownloadedPackage 다운로드된 패키지 검증
func (vh *VerificationHandler) VerifyDownloadedPackage(
	c *fiber.Ctx,
	packageType string,
	packagePath string,
	content []byte,
	headers map[string]string,
) error {
	ctx := context.Background()

	// 메타데이터 수집
	metadata := make(map[string]string)
	for k, v := range headers {
		metadata[strings.ToLower(k)] = v
	}

	// 추가 메타데이터
	metadata["client_ip"] = c.IP()
	metadata["user_agent"] = c.Get("User-Agent")

	// 패키지 검증 수행
	result, err := vh.verifier.VerifyPackage(ctx, packageType, packagePath, content, metadata)
	if err != nil {
		log.Printf("Package verification error: %v", err)

		// 오류 알림 전송
		vh.sendErrorAlert(ctx, packageType, packagePath, err, metadata)

		// 엄격 모드에서는 차단
		if vh.isStrictMode() {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error":   "Package verification error",
				"message": err.Error(),
			})
		}

		return nil
	}

	// 검증 실패 처리
	if result != nil && !result.Valid {
		log.Printf("Package verification failed: %s", result.Message)

		// 차단 모드 확인
		if vh.shouldBlockOnFailure() {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error":   "Package verification failed",
				"details": result,
			})
		}

		// 검증 실패를 헤더로 표시
		c.Set("X-Package-Verification", "failed")
		c.Set("X-Verification-Message", result.Message)
	} else if result != nil && result.Valid {
		// 검증 성공 표시
		c.Set("X-Package-Verification", "success")
		if result.HashType != "" {
			c.Set("X-Verified-Hash-Type", string(result.HashType))
		}
	}

	return nil
}

// VerifyUploadedPackage 업로드된 패키지 검증
func (vh *VerificationHandler) VerifyUploadedPackage(
	c *fiber.Ctx,
	packageType string,
	packagePath string,
) error {
	// 요청 본문 읽기
	body := c.Body()
	if len(body) == 0 {
		return nil
	}

	// 헤더에서 해시 정보 추출
	metadata := make(map[string]string)
	metadata["content-sha256"] = c.Get("Content-SHA256")
	metadata["x-checksum-sha256"] = c.Get("X-Checksum-SHA256")
	metadata["digest"] = c.Get("Digest")
	metadata["client_ip"] = c.IP()
	metadata["user_agent"] = c.Get("User-Agent")

	// 사용자 정보 추가
	if username := c.Locals("username"); username != nil {
		metadata["username"] = username.(string)
	}

	ctx := context.Background()

	// 패키지 검증 수행
	result, err := vh.verifier.VerifyPackage(ctx, packageType, packagePath, body, metadata)
	if err != nil {
		log.Printf("Upload verification error: %v", err)

		// 오류 알림 전송
		vh.sendErrorAlert(ctx, packageType, packagePath, err, metadata)

		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Package verification error",
			"message": err.Error(),
		})
	}

	// 검증 실패 처리
	if result != nil && !result.Valid {
		log.Printf("Upload verification failed: %s", result.Message)

		// 업로드는 항상 검증 실패 시 차단
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Package verification failed",
			"details": result,
		})
	}

	// 검증 성공 정보를 컨텍스트에 저장
	c.Locals("packageVerified", true)
	c.Locals("verificationResult", result)

	return nil
}

// sendErrorAlert 오류 알림 전송
func (vh *VerificationHandler) sendErrorAlert(
	ctx context.Context,
	packageType string,
	packagePath string,
	err error,
	metadata map[string]string,
) {
	if vh.alertManager == nil {
		return
	}

	event := &alerts.AlertEvent{
		Level:   alerts.AlertLevelError,
		Type:    "package_verification_error",
		Title:   "Package Verification Error",
		Message: fmt.Sprintf("Error verifying %s package: %v", packageType, err),
		Source:  "verification_handler",
		PackageInfo: &alerts.PackageInfo{
			Type: packageType,
			Path: packagePath,
		},
		Metadata: map[string]interface{}{
			"error":            err.Error(),
			"request_metadata": metadata,
		},
	}

	if err := vh.alertManager.Send(ctx, event); err != nil {
		log.Printf("Failed to send error alert: %v", err)
	}
}

// isStrictMode 엄격 모드 확인
func (vh *VerificationHandler) isStrictMode() bool {
	// 설정에서 확인
	// TODO: 실제 설정 구조에 맞게 수정
	return true
}

// shouldBlockOnFailure 검증 실패 시 차단 여부
func (vh *VerificationHandler) shouldBlockOnFailure() bool {
	// 설정에서 확인
	// TODO: 실제 설정 구조에 맞게 수정
	return true
}

// loadVerifierConfig 검증 설정 로드
func loadVerifierConfig(globalConfig *configs.GlobalConfig) *verification.VerifierConfig {
	// 기본 설정
	config := &verification.VerifierConfig{
		StrictMode:     true,
		BlockOnFailure: true,
		AlertOnFailure: true,
		PackageTypeConfigs: map[string]verification.PackageConfig{
			"npm": {
				Enabled:        true,
				RequiredHashes: []string{"sha512"},
				TrustedSources: []string{"https://registry.npmjs.org"},
			},
			"pip": {
				Enabled:        true,
				RequiredHashes: []string{"sha256"},
				TrustedSources: []string{"https://pypi.org", "https://files.pythonhosted.org"},
			},
			"apt": {
				Enabled:        true,
				RequiredHashes: []string{"sha256"},
				TrustedSources: []string{"http://archive.ubuntu.com", "http://security.ubuntu.com"},
			},
			"docker": {
				Enabled:        true,
				RequiredHashes: []string{"sha256"},
				TrustedSources: []string{"https://registry-1.docker.io"},
			},
			"maven": {
				Enabled:        true,
				RequiredHashes: []string{"sha256", "sha1"},
				TrustedSources: []string{"https://repo1.maven.org"},
			},
		},
	}

	// TODO: 실제 설정 파일에서 로드

	return config
}

// VerificationMiddleware 검증 미들웨어
func (vh *VerificationHandler) VerificationMiddleware() fiber.Handler {
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

		// 업로드 요청 검증 (POST, PUT)
		if c.Method() == "POST" || c.Method() == "PUT" {
			if err := vh.VerifyUploadedPackage(c, proxyType, packagePath); err != nil {
				return err
			}
		}

		// 다음 핸들러 실행
		if err := c.Next(); err != nil {
			return err
		}

		// 다운로드 응답 검증 (GET)
		if c.Method() == "GET" && c.Response().StatusCode() == fiber.StatusOK {
			body := c.Response().Body()
			if len(body) > 0 {
				// 응답 헤더 수집
				headers := make(map[string]string)
				c.Response().Header.VisitAll(func(key, value []byte) {
					headers[string(key)] = string(value)
				})

				// 검증 수행
				if err := vh.VerifyDownloadedPackage(c, proxyType, packagePath, body, headers); err != nil {
					return err
				}
			}
		}

		return nil
	}
}
