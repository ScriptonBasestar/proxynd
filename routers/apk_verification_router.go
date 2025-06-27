package routers

import (
	"os"
	"path/filepath"
	"proxynd/configs"
	"proxynd/helpers"
	"proxynd/logging"
	"proxynd/verification/apk"

	"github.com/gofiber/fiber/v2"
)

// APKVerificationRouter APK 서명 검증 관련 API 라우터
func APKVerificationRouter(app *fiber.App) {
	api := app.Group("/api/apk/verification")

	// APK 서명 검증 상태 조회
	api.Get("/status", getApkVerificationStatus)

	// 특정 APK 파일의 서명 검증
	api.Post("/verify", verifySpecificApkFile)

	// 신뢰할 수 있는 키 목록 조회
	api.Get("/keys", getTrustedKeys)

	// 서명 검증 설정 조회
	api.Get("/config", getApkVerificationConfig)
}

// getApkVerificationStatus APK 서명 검증 상태 조회
func getApkVerificationStatus(c *fiber.Ctx) error {
	logger := logging.GetLogger()

	// APK 설정 읽기
	apkConfig := configs.ApkProxyConfig{}
	apkConfig.ReadConfig()

	// 서명 검증기 생성
	verifier := apk.NewSignatureVerifier()

	var keyCount int
	var keyLoadError string

	// 키 로드 시도
	if apkConfig.Verification.KeyDirectory != "" {
		if err := verifier.LoadTrustedKeys(apkConfig.Verification.KeyDirectory); err != nil {
			keyLoadError = err.Error()
			logger.Error("키 로드 실패", logging.F("error", err.Error()))
		}
		keyCount = verifier.GetTrustedKeyCount()
	}

	// 키 디렉토리 상태 확인
	var keyDirExists bool
	if apkConfig.Verification.KeyDirectory != "" {
		if _, err := os.Stat(apkConfig.Verification.KeyDirectory); err == nil {
			keyDirExists = true
		}
	}

	return c.JSON(fiber.Map{
		"enabled":           apkConfig.Verification.Enabled,
		"key_directory":     apkConfig.Verification.KeyDirectory,
		"key_dir_exists":    keyDirExists,
		"trusted_key_count": keyCount,
		"fail_on_invalid":   apkConfig.Verification.FailOnInvalid,
		"cache_validated":   apkConfig.Verification.CacheValidated,
		"key_load_error":    keyLoadError,
		"status": func() string {
			if !apkConfig.Verification.Enabled {
				return "disabled"
			}
			if keyCount == 0 {
				return "no_keys"
			}
			return "active"
		}(),
	})
}

// VerifyApkRequest APK 검증 요청 구조체
type VerifyApkRequest struct {
	FilePath string `json:"file_path" validate:"required"`
}

// verifySpecificApkFile 특정 APK 파일의 서명 검증
func verifySpecificApkFile(c *fiber.Ctx) error {
	logger := logging.GetLogger()

	var req VerifyApkRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request format",
		})
	}

	// 파일 경로 검증
	if req.FilePath == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "file_path is required",
		})
	}

	// 보안상 스토리지 디렉토리 내부 파일만 허용
	storageDir := helpers.GetStorageDir()
	fullPath := filepath.Join(storageDir, req.FilePath)

	// 경로 정규화 및 검증
	cleanPath := filepath.Clean(fullPath)
	if !filepath.HasPrefix(cleanPath, storageDir) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid file path: outside storage directory",
		})
	}

	// 파일 존재 확인
	if _, err := os.Stat(cleanPath); os.IsNotExist(err) {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "File not found",
		})
	}

	// APK 설정 읽기
	apkConfig := configs.ApkProxyConfig{}
	apkConfig.ReadConfig()

	// 서명 검증기 생성 및 키 로드
	verifier := apk.NewSignatureVerifier()
	if apkConfig.Verification.KeyDirectory != "" {
		if err := verifier.LoadTrustedKeys(apkConfig.Verification.KeyDirectory); err != nil {
			logger.Error("키 로드 실패", logging.F("error", err.Error()))
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error":   "Failed to load trusted keys",
				"details": err.Error(),
			})
		}
	}

	// 서명 검증 수행
	result := verifier.VerifyApkSignature(cleanPath)

	logger.Info("APK 서명 검증 API 호출",
		logging.F("file", req.FilePath),
		logging.F("valid", result.IsValid))

	return c.JSON(fiber.Map{
		"file_path":    req.FilePath,
		"verification": result,
		"trusted_keys": verifier.GetTrustedKeyCount(),
	})
}

// getTrustedKeys 신뢰할 수 있는 키 목록 조회
func getTrustedKeys(c *fiber.Ctx) error {
	logger := logging.GetLogger()

	// APK 설정 읽기
	apkConfig := configs.ApkProxyConfig{}
	apkConfig.ReadConfig()

	if apkConfig.Verification.KeyDirectory == "" {
		return c.JSON(fiber.Map{
			"keys":    []string{},
			"message": "Key directory not configured",
		})
	}

	// 서명 검증기 생성 및 키 로드
	verifier := apk.NewSignatureVerifier()
	if err := verifier.LoadTrustedKeys(apkConfig.Verification.KeyDirectory); err != nil {
		logger.Error("키 로드 실패", logging.F("error", err.Error()))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Failed to load trusted keys",
			"details": err.Error(),
		})
	}

	fingerprints := verifier.GetTrustedKeyFingerprints()

	return c.JSON(fiber.Map{
		"key_directory": apkConfig.Verification.KeyDirectory,
		"key_count":     len(fingerprints),
		"keys":          fingerprints,
	})
}

// getApkVerificationConfig APK 서명 검증 설정 조회
func getApkVerificationConfig(c *fiber.Ctx) error {
	// APK 설정 읽기
	apkConfig := configs.ApkProxyConfig{}
	apkConfig.ReadConfig()

	return c.JSON(fiber.Map{
		"verification": apkConfig.Verification,
	})
}
