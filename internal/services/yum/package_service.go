package yum

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"proxynd/internal/domain/yum"
	"proxynd/internal/helpers"
	"proxynd/internal/logging"
	"proxynd/internal/security"
	"proxynd/pkg/httpclient"
)

// packageServiceImpl YUM 패키지 서비스 구현
type packageServiceImpl struct {
	config            yum.ProxyConfig
	repoManager       yum.RepoManager
	cacheManager      yum.CacheManager
	metadataProcessor yum.MetadataProcessor
	metricsCollector  yum.MetricsCollector
	logger            logging.Logger
	storageDir        string
}

// NewPackageService YUM 패키지 서비스 생성
func NewPackageService(
	config yum.ProxyConfig,
	repoManager yum.RepoManager,
	cacheManager yum.CacheManager,
	metadataProcessor yum.MetadataProcessor,
	metricsCollector yum.MetricsCollector,
	logger logging.Logger,
	storageDir string,
) yum.PackageService {
	return &packageServiceImpl{
		config:            config,
		repoManager:       repoManager,
		cacheManager:      cacheManager,
		metadataProcessor: metadataProcessor,
		metricsCollector:  metricsCollector,
		logger:            logger,
		storageDir:        storageDir,
	}
}

// HandleRequest 패키지 요청 처리
func (s *packageServiceImpl) HandleRequest(ctx context.Context, request *yum.PackageRequest) (*yum.PackageResponse, error) { //nolint:lll
	startTime := time.Now()

	// 요청 경로 유효성 검증
	if err := s.ValidatePackagePath(request.PackagePath); err != nil {
		s.logger.Warn("YUM 패키지 경로 유효성 검증 실패",
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
			s.logger.Debug("YUM 캐시에서 파일 제공",
				logging.F("path", request.PackagePath),
				logging.F("cache_key", cacheKey))

			data, err := os.ReadFile(cachedEntry.Path)
			if err == nil {
				// 메트릭 기록
				s.recordRequestMetrics(ctx, request, http.StatusOK, time.Since(startTime), true, "", int64(len(data)))

				return &yum.PackageResponse{
					Data:        data,
					ContentType: s.getContentType(request.PackagePath),
					Headers:     s.generateHeaders(request.PackagePath),
					StatusCode:  http.StatusOK,
					FromCache:   true,
					IsRepoMeta:  s.isRepoMetadata(request.PackagePath),
					IsRpmFile:   s.isRpmFile(request.PackagePath),
				}, nil
			}
		}
	}

	// 업스트림에서 파일 다운로드
	data, proxyUsed, err := s.downloadFromUpstream(ctx, request.PackagePath, filePath)
	if err != nil {
		s.logger.Error("YUM 업스트림 다운로드 실패",
			logging.F("path", request.PackagePath),
			logging.F("error", err.Error()))

		// 메트릭 기록
		s.recordRequestMetrics(ctx, request, http.StatusInternalServerError, time.Since(startTime), false, proxyUsed, 0)
		return nil, fmt.Errorf("download failed: %w", err)
	}

	// 캐시에 저장
	if s.config.GetUseCache() {
		ttl := s.calculateTTL(request.PackagePath)
		if err := s.cacheManager.Set(ctx, cacheKey, data, ttl.Nanoseconds()); err != nil {
			s.logger.Warn("YUM 캐시 저장 실패",
				logging.F("cache_key", cacheKey),
				logging.F("error", err.Error()))
		}
	}

	// 메트릭 기록
	s.recordRequestMetrics(ctx, request, http.StatusOK, time.Since(startTime), false, proxyUsed, int64(len(data)))

	return &yum.PackageResponse{
		Data:        data,
		ContentType: s.getContentType(request.PackagePath),
		Headers:     s.generateHeaders(request.PackagePath),
		StatusCode:  http.StatusOK,
		FromCache:   false,
		ProxyUsed:   proxyUsed,
		IsRepoMeta:  s.isRepoMetadata(request.PackagePath),
		IsRpmFile:   s.isRpmFile(request.PackagePath),
	}, nil
}

// GetPackageInfo RPM 패키지 정보 조회
func (s *packageServiceImpl) GetPackageInfo(ctx context.Context, packagePath string) (*yum.PackageInfo, error) {
	if !s.isRpmFile(packagePath) {
		return nil, fmt.Errorf("not an RPM package file: %s", packagePath)
	}

	// 패키지 경로에서 정보 추출
	parts := strings.Split(packagePath, "/")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid package path format: %s", packagePath)
	}

	filename := filepath.Base(packagePath)
	// RPM 파일명 파싱 (예: package-1.0.0-1.el7.x86_64.rpm)
	name, version, release, arch := s.parseRpmFilename(filename)

	return &yum.PackageInfo{
		Name:         name,
		Version:      version,
		Release:      release,
		Architecture: arch,
		URL:          packagePath,
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
	allowedExtensions := []string{ //nolint:lll
		".rpm", ".xml", ".xml.gz", ".xml.bz2", ".xml.xz",
		".sqlite", ".sqlite.gz", ".sqlite.bz2", ".sqlite.xz",
		".asc", ".gpg",
	}
	hasValidExtension := false
	for _, ext := range allowedExtensions {
		if strings.HasSuffix(packagePath, ext) || strings.Contains(packagePath, "repomd.xml") {
			hasValidExtension = true
			break
		}
	}

	if !hasValidExtension {
		return fmt.Errorf("unsupported file extension")
	}

	return nil
}

// 헬퍼 메서드들

func (s *packageServiceImpl) generateCacheKey(packagePath string) string {
	return fmt.Sprintf("yum:%s", packagePath)
}

func (s *packageServiceImpl) downloadFromUpstream(ctx context.Context, packagePath, filePath string) ([]byte, string, error) { //nolint:lll
	// 디렉토리 생성
	if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
		return nil, "", fmt.Errorf("failed to create directory: %w", err)
	}

	client := httpclient.NewProxyClient()
	proxies := s.config.GetProxies()

	for _, proxy := range proxies {
		fullURL := helpers.JoinURL(proxy.URL, packagePath)
		s.logger.Debug("YUM 업스트림에서 다운로드 시도",
			logging.F("proxy", proxy.Name),
			logging.F("url", fullURL))

		resp, err := client.GetWithRetry(ctx, fullURL, 2)
		if err != nil {
			s.logger.Warn("YUM 프록시 연결 실패",
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

			s.logger.Info("YUM 파일 다운로드 성공",
				logging.F("proxy", proxy.Name),
				logging.F("path", packagePath),
				logging.F("size", len(data)))

			return data, proxy.Name, nil
		}

		s.logger.Warn("YUM 업스트림 오류 응답",
			logging.F("proxy", proxy.Name),
			logging.F("status", resp.StatusCode))
	}

	return nil, "", fmt.Errorf("all upstream servers failed")
}

func (s *packageServiceImpl) getContentType(packagePath string) string {
	switch {
	case strings.HasSuffix(packagePath, ".rpm"):
		return "application/x-rpm"
	case strings.HasSuffix(packagePath, ".xml") || strings.HasSuffix(packagePath, ".xml.gz"):
		return "application/xml"
	case strings.HasSuffix(packagePath, ".xml.bz2") || strings.HasSuffix(packagePath, ".xml.xz"):
		return "application/xml"
	case strings.HasSuffix(packagePath, ".sqlite") || strings.HasSuffix(packagePath, ".sqlite.bz2"):
		return "application/octet-stream" //nolint:goconst
	case strings.HasSuffix(packagePath, ".sqlite.gz") || strings.HasSuffix(packagePath, ".sqlite.xz"):
		return "application/octet-stream"
	case strings.HasSuffix(packagePath, ".asc") || strings.HasSuffix(packagePath, ".gpg"):
		return "application/pgp-signature"
	case strings.Contains(packagePath, "repomd.xml"):
		return "text/xml"
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
	inlineExtensions := []string{".xml", ".txt", ".asc", ".gpg"}
	for _, ext := range inlineExtensions {
		if strings.HasSuffix(filename, ext) {
			return true
		}
	}
	return false
}

func (s *packageServiceImpl) isRepoMetadata(packagePath string) bool {
	return strings.Contains(packagePath, "repomd.xml") ||
		strings.Contains(packagePath, "primary.xml") ||
		strings.Contains(packagePath, "filelists.xml") ||
		strings.Contains(packagePath, "other.xml") ||
		strings.HasSuffix(packagePath, ".sqlite")
}

func (s *packageServiceImpl) isRpmFile(packagePath string) bool {
	return strings.HasSuffix(packagePath, ".rpm")
}

func (s *packageServiceImpl) calculateTTL(packagePath string) time.Duration {
	if s.isRepoMetadata(packagePath) {
		return s.config.GetRepoMetadataTTL()
	}
	if s.isRpmFile(packagePath) {
		return s.config.GetRpmFileTTL()
	}
	return s.config.GetCacheTTL()
}

func (s *packageServiceImpl) parseRpmFilename(filename string) (name, version, release, arch string) {
	// RPM 파일명 파싱 (예: package-1.0.0-1.el7.x86_64.rpm)
	filename = strings.TrimSuffix(filename, ".rpm")

	// 아키텍처 추출
	lastDot := strings.LastIndex(filename, ".")
	if lastDot > 0 {
		arch = filename[lastDot+1:]
		filename = filename[:lastDot]
	}

	// 릴리스 추출
	lastDash := strings.LastIndex(filename, "-")
	if lastDash > 0 {
		release = filename[lastDash+1:]
		filename = filename[:lastDash]
	}

	// 버전 추출
	lastDash = strings.LastIndex(filename, "-")
	if lastDash > 0 {
		version = filename[lastDash+1:]
		name = filename[:lastDash]
	} else {
		name = filename
	}

	return
}

func (s *packageServiceImpl) recordRequestMetrics(ctx context.Context, request *yum.PackageRequest, statusCode int, responseTime time.Duration, cacheHit bool, proxyUsed string, fileSize int64) { //nolint:lll
	metrics := &yum.RequestMetrics{
		Path:         request.PackagePath,
		Repository:   s.extractRepository(request.PackagePath),
		Method:       request.Method,
		StatusCode:   statusCode,
		ResponseTime: responseTime.Milliseconds(),
		CacheHit:     cacheHit,
		ProxyUsed:    proxyUsed,
		RequestTime:  time.Now(),
		FileSize:     fileSize,
		IsRepoMeta:   s.isRepoMetadata(request.PackagePath),
		IsRpmFile:    s.isRpmFile(request.PackagePath),
	}

	if err := s.metricsCollector.RecordRequest(ctx, metrics); err != nil {
		s.logger.Warn("YUM 메트릭 기록 실패",
			logging.F("error", err.Error()))
	}
}

func (s *packageServiceImpl) extractRepository(packagePath string) string {
	parts := strings.Split(packagePath, "/")
	if len(parts) > 0 {
		return parts[0]
	}
	return "unknown"
}
