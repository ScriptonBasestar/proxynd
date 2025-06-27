package proxy

import (
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"proxynd/configs"
	"proxynd/helpers"
	"proxynd/logging"
	"proxynd/verification/apk"
	"strings"
	"sync"

	"github.com/gofiber/fiber/v2"
)

var (
	apkVerifier     *apk.SignatureVerifier
	apkVerifierOnce sync.Once
)

// getApkVerifier APK 서명 검증기 싱글톤 인스턴스 반환
func getApkVerifier() *apk.SignatureVerifier {
	apkVerifierOnce.Do(func() {
		apkVerifier = apk.NewSignatureVerifier()
	})
	return apkVerifier
}

// ApkProxyHandler APK 프록시 요청 처리 핸들러
func ApkProxyHandler(c *fiber.Ctx) error {
	requestPath := c.Params("*")
	log.Printf("Access proxy apk: %s\n", requestPath)

	// 설정 읽기
	storageDir := helpers.GetStorageDir()
	globalConfig := configs.GlobalConfig{}
	globalConfig.ReadConfig()
	apkConfig := configs.ApkProxyConfig{}
	apkConfig.ReadConfig()

	// 파일 경로 생성
	filefullpath := path.Join(storageDir, apkConfig.Path, requestPath)
	filename := filepath.Base(filefullpath)

	// 서명 검증기 초기화 (필요시)
	if apkConfig.Verification.Enabled {
		verifier := getApkVerifier()
		if err := verifier.LoadTrustedKeys(apkConfig.Verification.KeyDirectory); err != nil {
			logger := logging.GetLogger()
			logger.Error("APK 신뢰 키 로드 실패", logging.F("error", err.Error()))
		}
	}

	// 캐시 확인
	if apkConfig.UseCache && helpers.FileExists(filefullpath) {
		log.Printf("Serving from cache: %s\n", filefullpath)

		// APK 파일인 경우 서명 검증 수행
		if apkConfig.Verification.Enabled && isApkFile(requestPath) {
			if !verifyApkFileSignature(filefullpath, apkConfig, c) {
				return nil // 검증 실패 시 응답은 이미 처리됨
			}
		}

		// Content-Type 설정
		contentType := getApkContentType(requestPath)
		c.Set("Content-Type", contentType)

		return c.SendFile(filefullpath)
	}

	// 캐시에 없으면 업스트림에서 가져오기
	if _, err := os.Stat(filefullpath); os.IsNotExist(err) {
		dirpath := filepath.Dir(filefullpath)
		os.MkdirAll(dirpath, os.ModePerm)
		out, err := os.Create(filefullpath)
		if err != nil {
			log.Printf("Error creating file: %v", err)
			return c.Status(fiber.StatusInternalServerError).SendString("Error creating file")
		}
		defer out.Close()

		// 업스트림 서버에서 파일 가져오기
		for _, proxy := range apkConfig.Proxies {
			fullURL := helpers.JoinURL(proxy.Url, requestPath)
			log.Printf("Fetching from upstream %s: %s\n", proxy.Name, fullURL)

			resp, err := http.Get(fullURL)
			if err != nil {
				log.Printf("Error fetching from proxy %s: %v\n", proxy.Name, err)
				continue
			}

			if resp.StatusCode == http.StatusOK {
				// 파일 저장
				_, err = io.Copy(out, resp.Body)
				resp.Body.Close()
				if err != nil {
					log.Printf("Error copying file: %v", err)
					return c.Status(fiber.StatusInternalServerError).SendString("Error copying file")
				}
				break
			}
			resp.Body.Close()
			log.Printf("Upstream %s returned status: %d\n", proxy.Name, resp.StatusCode)
		}
	}

	// 새로 다운로드한 APK 파일인 경우 서명 검증 수행
	if apkConfig.Verification.Enabled && isApkFile(requestPath) {
		if !verifyApkFileSignature(filefullpath, apkConfig, c) {
			return nil // 검증 실패 시 응답은 이미 처리됨
		}
	}

	// Content-Type 설정
	contentType := getApkContentType(requestPath)
	c.Set("Content-Type", contentType)

	// Content-Disposition 설정
	if isApkInlineFile(filename) {
		c.Set("Content-Disposition", "inline; filename="+filename)
	} else {
		c.Set("Content-Disposition", "attachment; filename="+filename)
	}

	return c.SendFile(filefullpath)
}

// getApkContentType 파일 경로에 따른 Content-Type 반환
func getApkContentType(filename string) string {
	switch {
	case strings.HasSuffix(filename, ".apk"):
		return "application/vnd.alpine.apk"
	case strings.HasSuffix(filename, "APKINDEX.tar.gz"):
		return "application/gzip"
	case strings.HasSuffix(filename, "APKINDEX"):
		return "text/plain"
	case strings.HasSuffix(filename, ".asc"):
		return "application/pgp-signature"
	case strings.HasSuffix(filename, ".rsa"):
		return "application/octet-stream"
	case strings.HasSuffix(filename, ".tar.gz"):
		return "application/gzip"
	case strings.HasSuffix(filename, ".gz"):
		return "application/gzip"
	default:
		return "application/octet-stream"
	}
}

// isApkInlineFile 인라인으로 표시할 파일 확인
func isApkInlineFile(filename string) bool {
	inlineExtensions := []string{"APKINDEX", ".asc", ".txt"}
	for _, ext := range inlineExtensions {
		if strings.Contains(filename, ext) {
			return true
		}
	}
	return false
}

// isApkFile APK 패키지 파일인지 확인
func isApkFile(filename string) bool {
	return strings.HasSuffix(filename, ".apk")
}

// verifyApkFileSignature APK 파일의 서명 검증 수행
func verifyApkFileSignature(filePath string, config configs.ApkProxyConfig, c *fiber.Ctx) bool {
	logger := logging.GetLogger()
	verifier := getApkVerifier()

	// 서명 검증 수행
	result := verifier.VerifyApkSignature(filePath)

	logger.Debug("APK 서명 검증 결과",
		logging.F("file", filePath),
		logging.F("valid", result.IsValid),
		logging.F("signature_file", result.SignatureFile),
		logging.F("key_fingerprint", result.KeyFingerprint))

	// 검증 성공 시
	if result.IsValid {
		logger.Info("APK 서명 검증 성공",
			logging.F("file", filePath),
			logging.F("key", result.KeyFingerprint))
		return true
	}

	// 검증 실패 시 처리
	logger.Warn("APK 서명 검증 실패",
		logging.F("file", filePath),
		logging.F("error", result.Error))

	// 검증 실패 시 요청 차단 설정이 활성화된 경우
	if config.Verification.FailOnInvalid {
		logger.Error("APK 서명 검증 실패로 요청 차단",
			logging.F("file", filePath))

		c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error":   "APK signature verification failed",
			"message": "The requested APK file failed signature verification",
			"details": result.Error,
		})
		return false
	}

	// 경고만 하고 계속 진행
	logger.Warn("APK 서명 검증 실패하지만 계속 진행",
		logging.F("file", filePath),
		logging.F("reason", "fail_on_invalid is disabled"))

	return true
}
