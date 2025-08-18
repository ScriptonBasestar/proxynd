package proxy

import (
	"context"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"

	"proxynd/helpers"
	"proxynd/internal/config"
	"proxynd/internal/logging"
	"proxynd/internal/mirror"
	"proxynd/internal/security"
	"proxynd/pkg/httpclient"
	"proxynd/verification/apk"
)

var (
	apkVerifier        *apk.SignatureVerifier
	apkVerifierOnce    sync.Once
	mirrorSelector     *mirror.AlpineMirrorSelector
	mirrorSelectorOnce sync.Once
	selectorStarted    bool
	selectorMutex      sync.Mutex
)

// getApkVerifier APK 서명 검증기 싱글톤 인스턴스 반환
func getApkVerifier() *apk.SignatureVerifier {
	apkVerifierOnce.Do(func() {
		apkVerifier = apk.NewSignatureVerifier()
	})
	return apkVerifier
}

// getMirrorSelector 미러 선택기 싱글톤 인스턴스 반환
func getMirrorSelector() *mirror.AlpineMirrorSelector {
	mirrorSelectorOnce.Do(func() {
		mirrorSelector = mirror.NewAlpineMirrorSelector()
	})
	return mirrorSelector
}

// initializeMirrorSelector 미러 선택기 초기화 (필요시)
func initializeMirrorSelector(apkConfig config.ApkProxySettings) {
	if !apkConfig.MirrorSelection.Enabled {
		return
	}

	selectorMutex.Lock()
	defer selectorMutex.Unlock()

	if selectorStarted {
		return
	}

	selector := getMirrorSelector()
	config := convertToMirrorConfig(apkConfig.MirrorSelection)
	selector.Start(config, apkConfig.Proxies)
	selectorStarted = true
}

// convertToMirrorConfig 설정 변환
func convertToMirrorConfig(config config.ApkMirrorSelectionConfig) mirror.AlpineMirrorConfig {
	// 문자열을 time.Duration으로 변환
	healthCheckInterval, err := time.ParseDuration(config.HealthCheckInterval)
	if err != nil || healthCheckInterval == 0 {
		healthCheckInterval = 5 * time.Minute
	}

	healthCheckTimeout, err := time.ParseDuration(config.HealthCheckTimeout)
	if err != nil || healthCheckTimeout == 0 {
		healthCheckTimeout = 10 * time.Second
	}

	maxErrorCount := config.MaxErrorCount
	if maxErrorCount == 0 {
		maxErrorCount = 3
	}

	return mirror.AlpineMirrorConfig{
		HealthCheckInterval: healthCheckInterval,
		HealthCheckTimeout:  healthCheckTimeout,
		PreferredRegions:    config.PreferredRegions,
		FallbackToGlobal:    config.FallbackToGlobal,
		MaxErrorCount:       maxErrorCount,
		RegionDetectionMode: config.RegionDetectionMode,
	}
}

// ApkProxyHandler APK 프록시 요청 처리 핸들러
func ApkProxyHandler(c *fiber.Ctx) error {
	// 요청 컨텍스트 생성 (45초 타임아웃)
	ctx, cancel := context.WithTimeout(c.Context(), 45*time.Second)
	defer cancel()

	requestPath := c.Params("*")
	log.Printf("Access proxy apk: %s\n", requestPath)

	// 설정 읽기
	storageDir := helpers.GetStorageDir()
	globalConfig := config.GlobalConfig{}
	if err := globalConfig.ReadConfig(); err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to read global config")
	}
	apkConfig := config.ApkProxySettings{}
	if err := apkConfig.ReadConfig(); err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to read APK config")
	}

	// 파일 경로 생성
	baseDir := filepath.Join(storageDir, apkConfig.Path)
	filefullpath, err := security.SafeJoinPath(baseDir, requestPath)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid path")
	}
	filename := filepath.Base(filefullpath)

	// 서명 검증기 초기화 (필요시)
	if apkConfig.Verification.Enabled {
		verifier := getApkVerifier()
		if err := verifier.LoadTrustedKeys(apkConfig.Verification.KeyDirectory); err != nil {
			logger := logging.GetLogger()
			logger.Error("APK 신뢰 키 로드 실패", logging.F("error", err.Error()))
		}
	}

	// 미러 선택기 초기화 (필요시)
	initializeMirrorSelector(apkConfig)

	// 캐시 확인
	if apkConfig.UseCache && helpers.FileExists(filefullpath) {
		log.Printf("Serving from cache: %s\n", filefullpath)

		// APK 파일인 경우 서명 검증 수행
		if apkConfig.Verification.Enabled && isApkFile(requestPath) {
			if !verifyApkFileSignature(filefullpath, apkConfig, c) {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
					"error":   "APK signature verification failed",
					"message": "The requested APK file failed signature verification",
				})
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
		if err := os.MkdirAll(dirpath, os.ModePerm); err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString("Failed to create directory")
		}
		out, err := os.Create(filefullpath)
		if err != nil {
			log.Printf("Error creating file: %v", err)
			return c.Status(fiber.StatusInternalServerError).SendString("Error creating file")
		}
		defer func() { _ = out.Close() }()

		// HTTP 클라이언트 생성 (프록시 최적화 설정)
		proxyClient := httpclient.NewProxyClient()

		// 미러 선택을 사용하여 업스트림 서버에서 파일 가져오기
		proxies := apkConfig.Proxies
		if apkConfig.MirrorSelection.Enabled {
			selector := getMirrorSelector()
			proxies = selector.SelectBestMirror(requestPath, apkConfig.Proxies)
			log.Printf("미러 선택 결과: %d개 미러 선택됨\n", len(proxies))
		}

		for _, proxy := range proxies {
			fullURL := helpers.JoinURL(proxy.URL, requestPath)
			log.Printf("Fetching from upstream %s: %s\n", proxy.Name, fullURL)

			// 컨텍스트 기반 요청 (재시도 포함)
			resp, err := proxyClient.GetWithRetry(ctx, fullURL, 2)
			if err != nil {
				log.Printf("Error fetching from proxy %s: %v\n", proxy.Name, err)
				continue
			}
			defer func() { _ = resp.Body.Close() }()

			if resp.StatusCode == http.StatusOK {
				// 파일 저장
				_, err = io.Copy(out, resp.Body)
				if err != nil {
					log.Printf("Error copying file: %v", err)
					return c.Status(fiber.StatusInternalServerError).SendString("Error copying file")
				}
				log.Printf("Successfully fetched from %s\n", proxy.Name)
				break
			}
			log.Printf("Upstream %s returned status: %d\n", proxy.Name, resp.StatusCode)
		}
	}

	// 새로 다운로드한 APK 파일인 경우 서명 검증 수행
	if apkConfig.Verification.Enabled && isApkFile(requestPath) {
		if !verifyApkFileSignature(filefullpath, apkConfig, c) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error":   "APK signature verification failed",
				"message": "The requested APK file failed signature verification",
			})
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
		return MimeApplicationGzip
	case strings.HasSuffix(filename, "APKINDEX"):
		return MimeTextPlain
	case strings.HasSuffix(filename, ".asc"):
		return MimeApplicationPGPSignature
	case strings.HasSuffix(filename, ".rsa"):
		return MimeApplicationOctetStream
	case strings.HasSuffix(filename, ".tar.gz"):
		return MimeApplicationGzip
	case strings.HasSuffix(filename, ".gz"):
		return MimeApplicationGzip
	default:
		return MimeApplicationOctetStream
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
func verifyApkFileSignature(filePath string, config config.ApkProxySettings, c *fiber.Ctx) bool {
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
		return false
	}

	// 경고만 하고 계속 진행
	logger.Warn("APK 서명 검증 실패하지만 계속 진행",
		logging.F("file", filePath),
		logging.F("reason", "fail_on_invalid is disabled"))

	return true
}
