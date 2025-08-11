package apt

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/gofiber/fiber/v2"

	"proxynd/internal/domain/apt"
	"proxynd/logging"
)

// packageServiceImpl APT 패키지 서비스 구현
type packageServiceImpl struct {
	config           apt.ProxyConfig
	logger           logging.Logger
	mirrorManager    apt.MirrorManager
	cacheManager     apt.CacheManager
	contentResolver  apt.ContentTypeResolver
	metricsCollector apt.MetricsCollector
}

// NewPackageService PackageService 생성자
func NewPackageService(
	config apt.ProxyConfig,
	logger logging.Logger,
	mirrorManager apt.MirrorManager,
	cacheManager apt.CacheManager,
	contentResolver apt.ContentTypeResolver,
	metricsCollector apt.MetricsCollector,
) apt.PackageHandler {
	return &packageServiceImpl{
		config:           config,
		logger:           logger,
		mirrorManager:    mirrorManager,
		cacheManager:     cacheManager,
		contentResolver:  contentResolver,
		metricsCollector: metricsCollector,
	}
}

// Handle 패키지 요청 처리
func (s *packageServiceImpl) Handle(ctx context.Context, request *apt.PackageRequest) (*apt.PackageResponse, error) {
	startTime := time.Now()

	s.logger.Info("APT package request started",
		logging.F("osType", request.OSType),
		logging.F("packagePath", request.PackagePath),
		logging.F("method", request.Method),
	)

	// 1. 캐시에서 먼저 확인
	cacheKey := s.cacheManager.GenerateKey(request.OSType, request.PackagePath)
	if cacheEntry, err := s.cacheManager.Get(ctx, cacheKey); err == nil {
		s.logger.Debug("Serving from cache", logging.F("cacheKey", cacheKey))

		response := &apt.PackageResponse{
			Data:        nil, // 캐시된 파일은 파일 경로로만 전달
			ContentType: cacheEntry.ContentType,
			Headers: map[string]string{
				"Content-Type":        cacheEntry.ContentType,
				"Content-Disposition": s.contentResolver.GetDisposition(request.PackagePath),
				"X-Cache-Status":      "HIT",
				"X-Proxy-Type":        "apt",
			},
			StatusCode: 200,
			FromCache:  true,
		}

		// 메트릭 기록
		s.recordMetrics(ctx, request, response, time.Since(startTime), "")

		return response, nil
	}

	// 2. 업스트림 미러에서 가져오기
	mirrors := s.config.GetMirrors(request.OSType)
	if len(mirrors) == 0 {
		return nil, fmt.Errorf("no mirrors configured for OS type: %s", request.OSType)
	}

	var lastErr error
	for _, mirror := range mirrors {
		if mirror.URL == "" {
			continue
		}

		s.logger.Debug("Trying mirror",
			logging.F("mirrorURL", mirror.URL),
			logging.F("osType", request.OSType),
		)

		response, err := s.fetchFromMirror(ctx, mirror.URL, request)
		if err != nil {
			s.mirrorManager.MarkMirrorFailed(request.OSType, mirror.URL, err)
			lastErr = err
			continue
		}

		// 3. 성공한 경우 캐시에 저장
		if err := s.cacheManager.Set(ctx, cacheKey, response.Data, response.ContentType); err != nil {
			s.logger.Warn("Failed to cache response", logging.F("error", err))
		}

		// 응답 헤더 설정
		response.Headers = map[string]string{
			"Content-Type":        response.ContentType,
			"Content-Disposition": s.contentResolver.GetDisposition(request.PackagePath),
			"X-Cache-Status":      "MISS",
			"X-Proxy-Type":        "apt",
		}

		// 메트릭 기록
		s.recordMetrics(ctx, request, response, time.Since(startTime), mirror.URL)

		s.logger.Info("APT package request completed",
			logging.F("duration_ms", time.Since(startTime).Milliseconds()),
			logging.F("fromCache", response.FromCache),
			logging.F("mirrorUsed", mirror.URL),
		)

		return response, nil
	}

	// 모든 미러 실패
	if lastErr != nil {
		return nil, fmt.Errorf("all mirrors failed, last error: %w", lastErr)
	}

	return nil, fmt.Errorf("no available mirrors for OS type: %s", request.OSType)
}

// GetPackageMetadata 패키지 메타데이터 조회
func (s *packageServiceImpl) GetPackageMetadata(ctx context.Context, osType, packagePath string) (*apt.PackageMetadata, error) { //nolint:lll
	// 패키지 경로에서 메타데이터 추출
	metadata := &apt.PackageMetadata{
		OSType:      osType,
		PackageName: s.extractPackageName(packagePath),
		FileType:    s.extractFileType(packagePath),
	}

	// 캐시에서 파일 정보 확인
	cacheKey := s.cacheManager.GenerateKey(osType, packagePath)
	if cacheEntry, err := s.cacheManager.Get(ctx, cacheKey); err == nil {
		metadata.Size = cacheEntry.Size
		metadata.ModTime = cacheEntry.CreatedAt
	}

	return metadata, nil
}

// fetchFromMirror 특정 미러에서 패키지 가져오기
func (s *packageServiceImpl) fetchFromMirror(ctx context.Context, mirrorURL string, request *apt.PackageRequest) (*apt.PackageResponse, error) { //nolint:lll
	// URL 구성
	baseURL := filepath.Join(mirrorURL, request.PackagePath)

	s.logger.Debug("Fetching from mirror",
		logging.F("url", baseURL),
		logging.F("packagePath", request.PackagePath),
	)

	// Fiber Agent로 요청
	agent := fiber.Get(baseURL)

	// 헤더 설정
	agent.Set("User-Agent", "ProxyND/1.0 APT-Proxy")
	agent.Set("X-APT-Proxy", "ProxyND")

	// 압축 지원
	if acceptEncoding, exists := request.Headers["Accept-Encoding"]; exists {
		agent.Set("Accept-Encoding", acceptEncoding)
	} else {
		agent.Set("Accept-Encoding", "gzip, deflate")
	}

	// 요청 실행
	statusCode, body, errs := agent.Bytes()
	if len(errs) > 0 {
		return nil, fmt.Errorf("http request failed: %w", errs[0])
	}

	if statusCode != fiber.StatusOK {
		return nil, fmt.Errorf("upstream returned status %d", statusCode)
	}

	response := &apt.PackageResponse{
		Data:        body,
		ContentType: s.contentResolver.GetContentType(request.PackagePath),
		StatusCode:  statusCode,
		FromCache:   false,
		MirrorUsed:  mirrorURL,
	}

	return response, nil
}

// recordMetrics 요청 메트릭 기록
func (s *packageServiceImpl) recordMetrics(ctx context.Context, request *apt.PackageRequest, response *apt.PackageResponse, duration time.Duration, mirrorUsed string) { //nolint:lll
	metrics := &apt.RequestMetrics{
		OSType:      request.OSType,
		Path:        request.PackagePath,
		Method:      request.Method,
		StatusCode:  response.StatusCode,
		Duration:    duration,
		FromCache:   response.FromCache,
		MirrorUsed:  mirrorUsed,
		BytesServed: int64(len(response.Data)),
		Timestamp:   time.Now(),
	}

	if err := s.metricsCollector.RecordRequest(ctx, metrics); err != nil {
		s.logger.Warn("Failed to record metrics", logging.F("error", err))
	}
}

// extractPackageName 패키지 경로에서 패키지 이름 추출
func (s *packageServiceImpl) extractPackageName(packagePath string) string {
	// 예: pool/main/a/apache2/apache2_2.4.41-4ubuntu3_amd64.deb
	// -> apache2
	parts := filepath.Dir(packagePath)
	if parts != "." {
		return filepath.Base(parts)
	}
	return filepath.Base(packagePath)
}

// extractFileType 패키지 경로에서 파일 타입 추출
func (s *packageServiceImpl) extractFileType(packagePath string) string {
	ext := filepath.Ext(packagePath)
	switch ext {
	case ".deb":
		return "deb"
	case ".gz":
		if filepath.Base(packagePath) == "Packages.gz" || filepath.Base(packagePath) == "Release.gz" {
			return "metadata"
		}
		return "archive"
	default:
		base := filepath.Base(packagePath)
		if base == "Release" || base == "Packages" || base == "InRelease" {
			return "metadata"
		}
		return "unknown"
	}
}
