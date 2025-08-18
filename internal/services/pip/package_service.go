package pip

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"proxynd/internal/domain/pip"
	"proxynd/internal/logging"
	"proxynd/pkg/httpclient"
)

// packageServiceImpl PIP 패키지 처리 서비스 구현
type packageServiceImpl struct {
	config            pip.ProxyConfig
	logger            logging.Logger
	indexManager      pip.IndexManager
	cacheManager      pip.CacheManager
	metadataProcessor pip.MetadataProcessor
	metricsCollector  pip.MetricsCollector
}

// NewPackageService PackageService 생성자
func NewPackageService(
	config pip.ProxyConfig,
	logger logging.Logger,
	indexManager pip.IndexManager,
	cacheManager pip.CacheManager,
	metadataProcessor pip.MetadataProcessor,
	metricsCollector pip.MetricsCollector,
) pip.PackageHandler {
	return &packageServiceImpl{
		config:            config,
		logger:            logger,
		indexManager:      indexManager,
		cacheManager:      cacheManager,
		metadataProcessor: metadataProcessor,
		metricsCollector:  metricsCollector,
	}
}

// Handle PIP 패키지 요청 처리
func (s *packageServiceImpl) Handle(ctx context.Context, request *pip.PackageRequest) (*pip.PackageResponse, error) {
	startTime := time.Now()

	s.logger.Info("Processing PIP package request",
		logging.F("packagePath", request.PackagePath),
		logging.F("method", request.Method),
	)

	// 캐시 키 생성
	cacheKey := s.cacheManager.GenerateKey(request.PackagePath)

	// 캐시 확인 (캐시가 활성화된 경우만)
	if s.config.IsCacheEnabled() {
		cacheEntry, err := s.cacheManager.Get(ctx, cacheKey)
		if err == nil && cacheEntry != nil {
			s.logger.Debug("Cache hit for PIP package",
				logging.F("packagePath", request.PackagePath),
				logging.F("cacheKey", cacheKey),
			)

			// 메트릭 기록
			s.recordMetrics(ctx, request, http.StatusOK, time.Since(startTime), true, "", cacheEntry.Size)

			return &pip.PackageResponse{
				Data:          nil, // 캐시에서는 파일 경로만 반환
				ContentType:   cacheEntry.ContentType,
				Headers:       make(map[string]string),
				StatusCode:    http.StatusOK,
				FromCache:     true,
				IsSimpleAPI:   cacheEntry.IsSimpleAPI,
				IsPackageFile: cacheEntry.IsPackageFile,
			}, nil
		}
	}

	// 캐시 미스 - 인덱스에서 데이터 가져오기
	response, err := s.fetchFromIndex(ctx, request)
	if err != nil {
		s.logger.Error("Failed to fetch from index",
			logging.F("error", err),
			logging.F("packagePath", request.PackagePath),
		)

		// 메트릭 기록 (실패)
		s.recordMetrics(ctx, request, http.StatusInternalServerError, time.Since(startTime), false, "", 0)
		return nil, fmt.Errorf("failed to fetch package: %w", err)
	}

	// 성공한 응답을 캐시에 저장 (캐시가 활성화된 경우)
	if response.StatusCode == http.StatusOK && s.config.IsCacheEnabled() {
		metadata := &pip.PackageMetadata{
			Name:          s.extractPackageName(request.PackagePath),
			IsSimpleAPI:   response.IsSimpleAPI,
			IsPackageFile: response.IsPackageFile,
			ModTime:       time.Now(),
		}

		err = s.cacheManager.Set(ctx, cacheKey, response.Data, response.ContentType, metadata)
		if err != nil {
			s.logger.Warn("Failed to cache package",
				logging.F("error", err),
				logging.F("packagePath", request.PackagePath),
			)
		}
	}

	// 메트릭 기록
	s.recordMetrics(ctx, request, response.StatusCode, time.Since(startTime), false, response.ProxyUsed, int64(len(response.Data))) //nolint:lll

	return response, nil
}

// fetchFromIndex 인덱스에서 패키지 데이터 가져오기
func (s *packageServiceImpl) fetchFromIndex(ctx context.Context, request *pip.PackageRequest) (*pip.PackageResponse, error) { //nolint:lll
	index, err := s.indexManager.GetNextIndex()
	if err != nil {
		return nil, fmt.Errorf("no available index: %w", err)
	}

	// HTTP 클라이언트 생성
	proxyClient := httpclient.NewProxyClient()

	// URL 구성
	fullURL := s.indexManager.BuildPackageURL(index.URL, request.PackagePath)

	s.logger.Debug("Fetching from PyPI index",
		logging.F("indexURL", index.URL),
		logging.F("fullURL", fullURL),
	)

	// 요청 실행 (재시도 포함)
	resp, err := proxyClient.GetWithRetry(ctx, fullURL, 2)
	if err != nil {
		s.indexManager.MarkIndexFailed(index.URL, err)
		return nil, fmt.Errorf("index request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// 응답 데이터 읽기
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Content-Type 결정
	fileName := s.extractFileName(request.PackagePath)
	contentType := s.metadataProcessor.GetContentType(request.PackagePath, fileName)

	// Simple API 요청인 경우 URL 재작성
	isSimpleAPI := s.metadataProcessor.IsSimpleAPIRequest(request.PackagePath)
	isPackageFile := s.metadataProcessor.IsPackageFileRequest(request.PackagePath)

	if isSimpleAPI && resp.StatusCode == http.StatusOK {
		data = s.metadataProcessor.RewriteSimpleAPI(data, request.BaseURL, request.PackagePath)
	}

	return &pip.PackageResponse{
		Data:          data,
		ContentType:   contentType,
		Headers:       make(map[string]string),
		StatusCode:    resp.StatusCode,
		FromCache:     false,
		ProxyUsed:     index.Name,
		IsSimpleAPI:   isSimpleAPI,
		IsPackageFile: isPackageFile,
	}, nil
}

// GetPackageMetadata 패키지 메타데이터 조회
func (s *packageServiceImpl) GetPackageMetadata(ctx context.Context, packagePath string) (*pip.PackageMetadata, error) {
	isSimpleAPI := s.metadataProcessor.IsSimpleAPIRequest(packagePath)
	isPackageFile := s.metadataProcessor.IsPackageFileRequest(packagePath)

	fileName := s.extractFileName(packagePath)
	extension := s.metadataProcessor.GetFileExtension(fileName)

	// 패키지 경로 분석
	metadata := &pip.PackageMetadata{
		Name:          s.extractPackageName(packagePath),
		FileType:      s.determineFileType(packagePath, extension),
		Extension:     extension,
		ModTime:       time.Now(),
		IsSimpleAPI:   isSimpleAPI,
		IsPackageFile: isPackageFile,
	}

	return metadata, nil
}

// ValidatePackage 패키지 무결성 검증
func (s *packageServiceImpl) ValidatePackage(ctx context.Context, data []byte, expectedChecksum string) error {
	if expectedChecksum == "" {
		return nil // 체크섬이 없으면 검증 생략
	}

	// TODO: SHA256 체크섬 검증 구현
	s.logger.Debug("Package checksum validation",
		logging.F("expectedChecksum", expectedChecksum),
		logging.F("dataSize", len(data)),
	)

	return nil
}

// recordMetrics 요청 메트릭 기록
func (s *packageServiceImpl) recordMetrics(ctx context.Context, request *pip.PackageRequest, statusCode int, duration time.Duration, fromCache bool, proxyUsed string, bytesServed int64) { //nolint:lll
	metrics := &pip.RequestMetrics{
		PackagePath:   request.PackagePath,
		Method:        request.Method,
		StatusCode:    statusCode,
		Duration:      duration,
		FromCache:     fromCache,
		ProxyUsed:     proxyUsed,
		BytesServed:   bytesServed,
		IsSimpleAPI:   s.metadataProcessor.IsSimpleAPIRequest(request.PackagePath),
		IsPackageFile: s.metadataProcessor.IsPackageFileRequest(request.PackagePath),
		Timestamp:     time.Now(),
		UserAgent:     request.Headers["User-Agent"],
	}

	if err := s.metricsCollector.RecordRequest(ctx, metrics); err != nil {
		s.logger.Warn("Failed to record metrics", logging.F("error", err))
	}
}

// extractPackageName 패키지 경로에서 패키지명 추출
func (s *packageServiceImpl) extractPackageName(packagePath string) string {
	// simple/package-name 형태에서 package-name 추출
	if strings.HasPrefix(packagePath, "simple/") {
		return strings.TrimPrefix(packagePath, "simple/")
	}

	// packages/source/p/package/package-1.0.tar.gz 형태에서 package 추출
	parts := strings.Split(packagePath, "/")
	if len(parts) > 1 {
		return parts[len(parts)-2] // 파일명 바로 앞 디렉토리가 패키지명
	}

	return packagePath
}

// extractFileName 패키지 경로에서 파일명 추출
func (s *packageServiceImpl) extractFileName(packagePath string) string {
	parts := strings.Split(packagePath, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return packagePath
}

// determineFileType 파일 타입 결정
func (s *packageServiceImpl) determineFileType(packagePath, extension string) string {
	if s.metadataProcessor.IsSimpleAPIRequest(packagePath) {
		return "simple"
	}

	switch extension {
	case ".whl":
		return "wheel"
	case ".tar.gz", ".tar.bz2", ".zip":
		return "sdist"
	default:
		return "other"
	}
}
