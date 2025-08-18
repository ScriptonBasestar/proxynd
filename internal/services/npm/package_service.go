package npm

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"proxynd/internal/domain/npm"
	"proxynd/internal/logging"
	"proxynd/pkg/httpclient"
)

// packageServiceImpl NPM 패키지 처리 서비스 구현
type packageServiceImpl struct {
	config            npm.ProxyConfig
	logger            logging.Logger
	proxyManager      npm.ProxyManager
	cacheManager      npm.CacheManager
	metadataProcessor npm.MetadataProcessor
	metricsCollector  npm.MetricsCollector
}

// NewPackageService PackageService 생성자
func NewPackageService(
	config npm.ProxyConfig,
	logger logging.Logger,
	proxyManager npm.ProxyManager,
	cacheManager npm.CacheManager,
	metadataProcessor npm.MetadataProcessor,
	metricsCollector npm.MetricsCollector,
) npm.PackageHandler {
	return &packageServiceImpl{
		config:            config,
		logger:            logger,
		proxyManager:      proxyManager,
		cacheManager:      cacheManager,
		metadataProcessor: metadataProcessor,
		metricsCollector:  metricsCollector,
	}
}

// Handle NPM 패키지 요청 처리
func (s *packageServiceImpl) Handle(ctx context.Context, request *npm.PackageRequest) (*npm.PackageResponse, error) {
	startTime := time.Now()

	s.logger.Info("Processing NPM package request",
		logging.F("packagePath", request.PackagePath),
		logging.F("method", request.Method),
	)

	// 캐시 키 생성
	cacheKey := s.cacheManager.GenerateKey(request.PackagePath)

	// 캐시에서 먼저 조회
	cacheEntry, err := s.cacheManager.Get(ctx, cacheKey)
	if err == nil && cacheEntry != nil {
		s.logger.Debug("Cache hit for NPM package",
			logging.F("packagePath", request.PackagePath),
			logging.F("cacheKey", cacheKey),
		)

		// 메트릭 기록
		s.recordMetrics(ctx, request, http.StatusOK, time.Since(startTime), true, "", cacheEntry.Size)

		return &npm.PackageResponse{
			Data:        nil, // 캐시에서는 파일 경로만 반환
			ContentType: cacheEntry.ContentType,
			Headers:     make(map[string]string),
			StatusCode:  http.StatusOK,
			FromCache:   true,
			IsMetadata:  cacheEntry.IsMetadata,
		}, nil
	}

	// 캐시 미스 - 프록시에서 데이터 가져오기
	response, err := s.fetchFromProxy(ctx, request)
	if err != nil {
		s.logger.Error("Failed to fetch from proxy",
			logging.F("error", err),
			logging.F("packagePath", request.PackagePath),
		)

		// 메트릭 기록 (실패)
		s.recordMetrics(ctx, request, http.StatusInternalServerError, time.Since(startTime), false, "", 0)
		return nil, fmt.Errorf("failed to fetch package: %w", err)
	}

	// 성공한 응답을 캐시에 저장
	isMetadata := s.metadataProcessor.IsMetadataRequest(request.PackagePath)
	if response.StatusCode == http.StatusOK {
		err = s.cacheManager.Set(ctx, cacheKey, response.Data, response.ContentType, isMetadata)
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

// fetchFromProxy 프록시에서 패키지 데이터 가져오기
func (s *packageServiceImpl) fetchFromProxy(ctx context.Context, request *npm.PackageRequest) (*npm.PackageResponse, error) { //nolint:lll
	proxy, err := s.proxyManager.GetNextProxy()
	if err != nil {
		return nil, fmt.Errorf("no available proxy: %w", err)
	}

	// HTTP 클라이언트 생성 (프록시 최적화 설정)
	proxyClient := httpclient.NewProxyClient()

	// URL 구성
	fullURL := fmt.Sprintf("%s/%s", proxy.URL, request.PackagePath)

	s.logger.Debug("Fetching from NPM proxy",
		logging.F("proxyURL", proxy.URL),
		logging.F("fullURL", fullURL),
	)

	// 요청 실행 (재시도 포함)
	resp, err := proxyClient.GetWithRetry(ctx, fullURL, 2)
	if err != nil {
		s.proxyManager.MarkProxyFailed(proxy.URL, err)
		return nil, fmt.Errorf("proxy request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// 응답 데이터 읽기
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Content-Type 결정
	contentType := s.metadataProcessor.GetContentType(request.PackagePath)

	// 메타데이터 요청인 경우 URL 재작성
	isMetadata := s.metadataProcessor.IsMetadataRequest(request.PackagePath)
	if isMetadata && resp.StatusCode == http.StatusOK {
		data = s.metadataProcessor.RewriteMetadata(data, request.BaseURL, request.PackagePath)
	}

	return &npm.PackageResponse{
		Data:        data,
		ContentType: contentType,
		Headers:     make(map[string]string),
		StatusCode:  resp.StatusCode,
		FromCache:   false,
		ProxyUsed:   proxy.Name,
		IsMetadata:  isMetadata,
	}, nil
}

// GetPackageMetadata 패키지 메타데이터 조회
func (s *packageServiceImpl) GetPackageMetadata(ctx context.Context, packagePath string) (*npm.PackageMetadata, error) {
	isMetadata := s.metadataProcessor.IsMetadataRequest(packagePath)

	// 패키지 경로 분석
	metadata := &npm.PackageMetadata{
		Name:       packagePath,
		IsMetadata: isMetadata,
		FileType:   "other",
		ModTime:    time.Now(),
	}

	// 패키지명 추출 및 분석
	if isMetadata {
		metadata.FileType = "metadata"
		metadata.IsScoped = len(packagePath) > 0 && packagePath[0] == '@'
	} else if s.isTarballRequest(packagePath) {
		metadata.FileType = "tarball"
		metadata.IsTarball = true
	}

	return metadata, nil
}

// recordMetrics 요청 메트릭 기록
func (s *packageServiceImpl) recordMetrics(ctx context.Context, request *npm.PackageRequest, statusCode int, duration time.Duration, fromCache bool, proxyUsed string, bytesServed int64) { //nolint:lll
	metrics := &npm.RequestMetrics{
		PackagePath: request.PackagePath,
		Method:      request.Method,
		StatusCode:  statusCode,
		Duration:    duration,
		FromCache:   fromCache,
		ProxyUsed:   proxyUsed,
		BytesServed: bytesServed,
		IsMetadata:  s.metadataProcessor.IsMetadataRequest(request.PackagePath),
		Timestamp:   time.Now(),
	}

	if err := s.metricsCollector.RecordRequest(ctx, metrics); err != nil {
		s.logger.Warn("Failed to record metrics", logging.F("error", err))
	}
}

// isTarballRequest tarball 요청인지 확인
func (s *packageServiceImpl) isTarballRequest(packagePath string) bool {
	return len(packagePath) > 4 && packagePath[len(packagePath)-4:] == ".tgz"
}
