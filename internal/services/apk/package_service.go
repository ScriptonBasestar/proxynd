package apk

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"proxynd/helpers"
	"proxynd/internal/domain/apk"
	"proxynd/internal/security"
	"proxynd/logging"
	"proxynd/pkg/httpclient"
)

// packageServiceImpl APK 패키지 서비스 구현
type packageServiceImpl struct {
	config            apk.ProxyConfig
	repositoryManager apk.RepositoryManager
	cacheManager      apk.CacheManager
	signatureVerifier apk.SignatureVerifier
	metricsCollector  apk.MetricsCollector
	logger            logging.Logger
	storageDir        string
}

// NewPackageService APK 패키지 서비스 생성
func NewPackageService(
	config apk.ProxyConfig,
	repositoryManager apk.RepositoryManager,
	cacheManager apk.CacheManager,
	signatureVerifier apk.SignatureVerifier,
	metricsCollector apk.MetricsCollector,
	logger logging.Logger,
	storageDir string,
) apk.PackageService {
	return &packageServiceImpl{
		config:            config,
		repositoryManager: repositoryManager,
		cacheManager:      cacheManager,
		signatureVerifier: signatureVerifier,
		metricsCollector:  metricsCollector,
		logger:            logger,
		storageDir:        storageDir,
	}
}

// HandleRequest 패키지 요청 처리
func (s *packageServiceImpl) HandleRequest(ctx context.Context, request *apk.PackageRequest) (*apk.PackageResponse, error) { //nolint:lll
	startTime := time.Now()

	// 요청 경로 유효성 검증
	if err := s.ValidatePackagePath(request.PackagePath); err != nil {
		s.logger.Warn("APK 패키지 경로 유효성 검증 실패",
			logging.F("path", request.PackagePath),
			logging.F("error", err.Error()))
		return nil, fmt.Errorf("invalid package path: %w", err)
	}

	// 파일 경로 생성
	baseDir := filepath.Join(s.storageDir, s.config.GetPath())
	filePath, err := security.SafeJoinPath(baseDir, request.PackagePath)
	if err != nil {
		return nil, fmt.Errorf("unsafe path: %w", err)
	}

	// 캐시 확인
	cacheKey := s.generateCacheKey(request.PackagePath)
	if s.config.GetUseCache() {
		if cachedEntry, err := s.cacheManager.Get(ctx, cacheKey); err == nil {
			s.logger.Debug("APK 캐시에서 파일 제공",
				logging.F("path", request.PackagePath),
				logging.F("cache_key", cacheKey))

			// APK 파일인 경우 서명 검증
			if s.config.GetVerificationEnabled() && s.signatureVerifier.IsPackageFile(request.PackagePath) {
				if signatureInfo, err := s.signatureVerifier.VerifyPackage(ctx, cachedEntry.Path); err != nil || !signatureInfo.IsValid { //nolint:lll
					if s.config.GetVerificationFailOnInvalid() {
						s.logger.Error("APK 캐시된 파일 서명 검증 실패",
							logging.F("path", request.PackagePath),
							logging.F("error", err))
						return nil, fmt.Errorf("signature verification failed")
					}
				}
			}

			data, err := os.ReadFile(cachedEntry.Path)
			if err == nil {
				// 메트릭 기록
				s.recordRequestMetrics(ctx, request, http.StatusOK, time.Since(startTime), true, "", int64(len(data)))

				return &apk.PackageResponse{
					Data:        data,
					ContentType: s.getContentType(request.PackagePath),
					Headers:     s.generateHeaders(request.PackagePath),
					StatusCode:  http.StatusOK,
					FromCache:   true,
					IsApkFile:   s.signatureVerifier.IsPackageFile(request.PackagePath),
					IsIndex:     s.isIndexFile(request.PackagePath),
					IsSignature: s.signatureVerifier.IsSignatureFile(request.PackagePath),
				}, nil
			}
		}
	}

	// 업스트림에서 파일 다운로드
	data, proxyUsed, err := s.downloadFromUpstream(ctx, request.PackagePath, filePath)
	if err != nil {
		s.logger.Error("APK 업스트림 다운로드 실패",
			logging.F("path", request.PackagePath),
			logging.F("error", err.Error()))

		// 메트릭 기록
		s.recordRequestMetrics(ctx, request, http.StatusInternalServerError, time.Since(startTime), false, proxyUsed, 0)
		return nil, fmt.Errorf("download failed: %w", err)
	}

	// APK 파일 서명 검증
	if s.config.GetVerificationEnabled() && s.signatureVerifier.IsPackageFile(request.PackagePath) {
		if signatureInfo, err := s.signatureVerifier.VerifyPackage(ctx, filePath); err != nil || !signatureInfo.IsValid {
			if s.config.GetVerificationFailOnInvalid() {
				s.logger.Error("APK 새 파일 서명 검증 실패",
					logging.F("path", request.PackagePath),
					logging.F("error", err))
				return nil, fmt.Errorf("signature verification failed")
			}
		}
	}

	// 캐시에 저장
	if s.config.GetUseCache() {
		ttl := s.calculateTTL(request.PackagePath)
		if err := s.cacheManager.Set(ctx, cacheKey, data, ttl.Nanoseconds()); err != nil {
			s.logger.Warn("APK 캐시 저장 실패",
				logging.F("cache_key", cacheKey),
				logging.F("error", err.Error()))
		}
	}

	// 메트릭 기록
	s.recordRequestMetrics(ctx, request, http.StatusOK, time.Since(startTime), false, proxyUsed, int64(len(data)))

	return &apk.PackageResponse{
		Data:        data,
		ContentType: s.getContentType(request.PackagePath),
		Headers:     s.generateHeaders(request.PackagePath),
		StatusCode:  http.StatusOK,
		FromCache:   false,
		ProxyUsed:   proxyUsed,
		IsApkFile:   s.signatureVerifier.IsPackageFile(request.PackagePath),
		IsIndex:     s.isIndexFile(request.PackagePath),
		IsSignature: s.signatureVerifier.IsSignatureFile(request.PackagePath),
	}, nil
}

// GetPackageInfo APK 패키지 정보 조회
func (s *packageServiceImpl) GetPackageInfo(ctx context.Context, packagePath string) (*apk.PackageInfo, error) {
	if !s.signatureVerifier.IsPackageFile(packagePath) {
		return nil, fmt.Errorf("not an APK package file: %s", packagePath)
	}

	// 패키지 경로 파싱 (예: v3.18/main/x86_64/package-1.0.0-r0.apk)
	pathParts := strings.Split(packagePath, "/")
	if len(pathParts) < 4 {
		return nil, fmt.Errorf("invalid APK package path format: %s", packagePath)
	}

	architecture := pathParts[2] // x86_64, aarch64, etc.
	filename := filepath.Base(packagePath)

	// APK 파일명 파싱 (예: package-1.0.0-r0.apk)
	name, version := s.parseApkFilename(filename)

	return &apk.PackageInfo{
		Name:         name,
		Version:      version,
		Architecture: architecture,
		// 추가 정보는 실제 APK 파일을 파싱해야 함
	}, nil
}

// ValidatePackagePath 패키지 경로 유효성 검증
func (s *packageServiceImpl) ValidatePackagePath(packagePath string) error {
	if packagePath == "" {
		return fmt.Errorf("package path is empty")
	}

	// 경로 보안 검증
	if strings.Contains(packagePath, "..") {
		return fmt.Errorf("path contains invalid characters")
	}

	// 허용된 파일 확장자 검증
	allowedExtensions := []string{".apk", ".tar.gz", ".asc", ".rsa", ".gz"}
	hasValidExtension := false
	for _, ext := range allowedExtensions {
		if strings.HasSuffix(packagePath, ext) {
			hasValidExtension = true
			break
		}
	}

	// APKINDEX 파일도 허용
	if strings.Contains(packagePath, "APKINDEX") {
		hasValidExtension = true
	}

	if !hasValidExtension {
		return fmt.Errorf("unsupported file extension")
	}

	return nil
}

// 헬퍼 메서드들

func (s *packageServiceImpl) generateCacheKey(packagePath string) string {
	return fmt.Sprintf("apk:%s", packagePath)
}

func (s *packageServiceImpl) downloadFromUpstream(ctx context.Context, packagePath, filePath string) ([]byte, string, error) { //nolint:lll
	// 디렉토리 생성
	if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
		return nil, "", fmt.Errorf("failed to create directory: %w", err)
	}

	client := httpclient.NewProxyClient()
	proxies := s.config.GetProxies()

	// 미러 선택 로직 (간단화된 버전)
	if s.config.GetMirrorSelectionEnabled() {
		// 실제 구현에서는 미러 선택 로직 적용
		s.logger.Debug("APK 미러 선택 활성화됨", logging.F("path", packagePath))
	}

	for _, proxy := range proxies {
		fullURL := helpers.JoinURL(proxy.URL, packagePath)
		s.logger.Debug("APK 업스트림에서 다운로드 시도",
			logging.F("proxy", proxy.Name),
			logging.F("url", fullURL))

		resp, err := client.GetWithRetry(ctx, fullURL, 2)
		if err != nil {
			s.logger.Warn("APK 프록시 연결 실패",
				logging.F("proxy", proxy.Name),
				logging.F("error", err.Error()))
			continue
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode == http.StatusOK {
			// 파일로 저장
			file, err := os.Create(filePath)
			if err != nil {
				return nil, proxy.Name, fmt.Errorf("failed to create file: %w", err)
			}
			defer func() { _ = file.Close() }()

			data, err := io.ReadAll(resp.Body)
			if err != nil {
				return nil, proxy.Name, fmt.Errorf("failed to read response: %w", err)
			}

			if _, err := file.Write(data); err != nil {
				return nil, proxy.Name, fmt.Errorf("failed to write file: %w", err)
			}

			s.logger.Info("APK 파일 다운로드 성공",
				logging.F("proxy", proxy.Name),
				logging.F("path", packagePath),
				logging.F("size", len(data)))

			return data, proxy.Name, nil
		}

		s.logger.Warn("APK 업스트림 오류 응답",
			logging.F("proxy", proxy.Name),
			logging.F("status", resp.StatusCode))
	}

	return nil, "", fmt.Errorf("all upstream servers failed")
}

func (s *packageServiceImpl) getContentType(packagePath string) string {
	switch {
	case strings.HasSuffix(packagePath, ".apk"):
		return "application/vnd.alpine.apk"
	case strings.HasSuffix(packagePath, "APKINDEX.tar.gz"):
		return "application/gzip"
	case strings.Contains(packagePath, "APKINDEX"):
		return "text/plain"
	case strings.HasSuffix(packagePath, ".asc"):
		return "application/pgp-signature"
	case strings.HasSuffix(packagePath, ".rsa"):
		return "application/octet-stream"
	case strings.HasSuffix(packagePath, ".tar.gz"):
		return "application/gzip"
	case strings.HasSuffix(packagePath, ".gz"):
		return "application/gzip"
	default:
		return "application/octet-stream"
	}
}

func (s *packageServiceImpl) generateHeaders(packagePath string) map[string]string {
	headers := make(map[string]string)
	filename := filepath.Base(packagePath)

	if s.isInlineFile(filename) {
		headers["Content-Disposition"] = fmt.Sprintf("inline; filename=%s", filename)
	} else {
		headers["Content-Disposition"] = fmt.Sprintf("attachment; filename=%s", filename)
	}

	return headers
}

func (s *packageServiceImpl) isInlineFile(filename string) bool {
	inlineFiles := []string{"APKINDEX", ".asc", ".txt"}
	for _, pattern := range inlineFiles {
		if strings.Contains(filename, pattern) {
			return true
		}
	}
	return false
}

func (s *packageServiceImpl) isIndexFile(packagePath string) bool {
	return strings.Contains(packagePath, "APKINDEX")
}

func (s *packageServiceImpl) calculateTTL(packagePath string) time.Duration {
	if s.signatureVerifier.IsPackageFile(packagePath) {
		return s.config.GetApkFileTTL()
	}
	if s.isIndexFile(packagePath) {
		return s.config.GetIndexFileTTL()
	}
	if s.signatureVerifier.IsSignatureFile(packagePath) {
		return s.config.GetSignatureFileTTL()
	}
	return s.config.GetCacheTTL()
}

func (s *packageServiceImpl) parseApkFilename(filename string) (name, version string) {
	// APK 파일명 파싱 (예: package-1.0.0-r0.apk)
	filename = strings.TrimSuffix(filename, ".apk")

	// 마지막 - 이후가 릴리스 번호 (r0, r1 등)
	lastDash := strings.LastIndex(filename, "-r")
	if lastDash > 0 {
		filename = filename[:lastDash]
	}

	// 다음 - 이후가 버전
	lastDash = strings.LastIndex(filename, "-")
	if lastDash > 0 {
		name = filename[:lastDash]
		version = filename[lastDash+1:]
	} else {
		name = filename
	}

	return
}

func (s *packageServiceImpl) recordRequestMetrics(ctx context.Context, request *apk.PackageRequest, statusCode int, responseTime time.Duration, cacheHit bool, proxyUsed string, fileSize int64) { //nolint:lll
	arch, branch, component := s.parsePackagePath(request.PackagePath)

	metrics := &apk.RequestMetrics{
		Path:         request.PackagePath,
		Architecture: arch,
		Branch:       branch,
		Component:    component,
		Method:       request.Method,
		StatusCode:   statusCode,
		ResponseTime: responseTime.Milliseconds(),
		CacheHit:     cacheHit,
		ProxyUsed:    proxyUsed,
		RequestTime:  time.Now(),
		FileSize:     fileSize,
		IsApkFile:    s.signatureVerifier.IsPackageFile(request.PackagePath),
		IsIndex:      s.isIndexFile(request.PackagePath),
		IsSignature:  s.signatureVerifier.IsSignatureFile(request.PackagePath),
	}

	if err := s.metricsCollector.RecordRequest(ctx, metrics); err != nil {
		s.logger.Warn("APK 메트릭 기록 실패", logging.F("error", err.Error()))
	}
}

func (s *packageServiceImpl) parsePackagePath(packagePath string) (arch, branch, component string) {
	parts := strings.Split(packagePath, "/")
	if len(parts) >= 3 {
		branch = parts[0]    // v3.18
		component = parts[1] // main
		arch = parts[2]      // x86_64
	}
	return
}
