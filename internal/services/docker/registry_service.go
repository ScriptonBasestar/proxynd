package docker

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"proxynd/internal/domain/docker"
	"proxynd/internal/logging"
)

// registryServiceImpl Docker 레지스트리 서비스 구현
type registryServiceImpl struct {
	config           docker.ProxyConfig
	manifestManager  docker.ManifestManager
	blobManager      docker.BlobManager
	authManager      docker.AuthenticationManager
	registryManager  docker.RegistryManager
	cacheManager     docker.CacheManager
	metricsCollector docker.MetricsCollector
	logger           logging.Logger
}

// NewRegistryService RegistryService 생성자
func NewRegistryService(
	config docker.ProxyConfig,
	manifestManager docker.ManifestManager,
	blobManager docker.BlobManager,
	authManager docker.AuthenticationManager,
	registryManager docker.RegistryManager,
	cacheManager docker.CacheManager,
	metricsCollector docker.MetricsCollector,
	logger logging.Logger,
) docker.RegistryHandler {
	return &registryServiceImpl{
		config:           config,
		manifestManager:  manifestManager,
		blobManager:      blobManager,
		authManager:      authManager,
		registryManager:  registryManager,
		cacheManager:     cacheManager,
		metricsCollector: metricsCollector,
		logger:           logger,
	}
}

// Handle Docker 레지스트리 요청을 처리하고 응답 반환
func (s *registryServiceImpl) Handle(ctx context.Context, request *docker.RegistryRequest) (*docker.ManifestResponse, error) { //nolint:lll
	startTime := time.Now()

	// 요청 유효성 검증
	if err := s.ValidateRequest(ctx, request); err != nil {
		s.logger.Error("Invalid Docker registry request", logging.F("error", err), logging.F("path", request.Path))
		// 잘못된 API 버전 등은 404로 응답
		return &docker.ManifestResponse{
			StatusCode: http.StatusNotFound,
			Headers:    make(map[string]string),
			Data:       []byte(`{"errors":[{"code":"UNSUPPORTED","message":"API version not supported"}]}`),
		}, nil
	}

	// 메트릭 기록용 기본 정보 설정
	metrics := &docker.RequestMetrics{
		Repository: request.Repository,
		Reference:  request.Reference,
		Operation:  request.Operation,
		Method:     request.Method,
		Timestamp:  startTime,
		ClientIP:   request.ClientIP,
		UserAgent:  request.Headers["User-Agent"],
	}

	var response *docker.ManifestResponse
	var err error

	// 작업 타입별 라우팅
	switch request.Operation {
	case dockerResourceManifest:
		response, err = s.handleManifestRequest(ctx, request)
	case dockerResourceBlob:
		response, err = s.handleBlobRequest(ctx, request)
	case "tags":
		response, err = s.handleTagsRequest(ctx, request)
	case "catalog":
		response, err = s.handleCatalogRequest(ctx, request)
	default:
		err = fmt.Errorf("unsupported operation: %s", request.Operation)
	}

	// 메트릭 완성 및 기록
	metrics.Duration = time.Since(startTime)
	if response != nil {
		metrics.StatusCode = response.StatusCode
		metrics.FromCache = response.FromCache
		metrics.RegistryUsed = response.RegistryUsed
		metrics.BytesServed = int64(len(response.Data))
		metrics.Digest = response.Digest
		metrics.ContentLength = int64(len(response.Data))
	}

	if err != nil {
		metrics.StatusCode = http.StatusInternalServerError
		_ = s.metricsCollector.RecordError(ctx, "registry_operation", request.Operation)
	}

	_ = s.metricsCollector.RecordRequest(ctx, metrics)
	_ = s.metricsCollector.RecordDuration(ctx, request.Operation, metrics.Duration.Milliseconds())

	return response, err
}

// HandleV2Base Docker Registry v2 베이스 엔드포인트 처리 (/v2/)
func (s *registryServiceImpl) HandleV2Base(ctx context.Context) (*docker.ManifestResponse, error) {
	s.logger.Debug("Handling Docker Registry v2 base endpoint")

	// Docker Registry v2 API 기본 응답 (빈 객체)
	data := []byte(`{}`)

	response := &docker.ManifestResponse{
		Data:        data,
		ContentType: "application/json",
		StatusCode:  http.StatusOK,
		Headers: map[string]string{
			"Docker-Distribution-Api-Version": "registry/2.0",
			"Content-Type":                    "application/json",
		},
		FromCache: false,
	}

	return response, nil
}

// ValidateRequest 요청 유효성 검증 (경로, 메서드, 헤더)
func (s *registryServiceImpl) ValidateRequest(ctx context.Context, request *docker.RegistryRequest) error {
	// HTTP 메서드 검증
	validMethods := map[string]bool{
		"GET": true, "HEAD": true, "PUT": true, "POST": true, "DELETE": true,
	}

	if !validMethods[request.Method] {
		return fmt.Errorf("unsupported HTTP method: %s", request.Method)
	}

	// Docker Registry v2 API 경로 검증
	if !strings.HasPrefix(request.Path, "/v2/") && !strings.HasPrefix(request.Path, "v2/") {
		return fmt.Errorf("invalid Docker registry path: %s", request.Path)
	}

	// 필수 헤더 검증 (매니페스트 요청의 경우)
	if request.Operation == dockerResourceManifest && request.Method == "GET" {
		if accept := request.Headers["Accept"]; accept == "" {
			s.logger.Warn("Missing Accept header in manifest request", logging.F("path", request.Path))
		}
	}

	return nil
}

// GetSupportedOperations 지원하는 작업 목록 반환
func (s *registryServiceImpl) GetSupportedOperations() []string {
	return []string{"manifest", "blob", "tags", "catalog"}
}

// handleManifestRequest 매니페스트 요청 처리
func (s *registryServiceImpl) handleManifestRequest(ctx context.Context, request *docker.RegistryRequest) (*docker.ManifestResponse, error) { //nolint:lll
	s.logger.Debug("Handling manifest request", logging.F("repository", request.Repository), logging.F("reference", request.Reference)) //nolint:lll

	return s.manifestManager.GetManifest(ctx, request.Repository, request.Reference)
}

// handleBlobRequest blob 요청 처리
func (s *registryServiceImpl) handleBlobRequest(ctx context.Context, request *docker.RegistryRequest) (*docker.ManifestResponse, error) { //nolint:lll
	s.logger.Debug("Handling blob request", logging.F("repository", request.Repository), logging.F("reference", request.Reference)) //nolint:lll

	return s.blobManager.GetBlob(ctx, request.Repository, request.Reference)
}

// handleTagsRequest 태그 목록 요청 처리
func (s *registryServiceImpl) handleTagsRequest(ctx context.Context, request *docker.RegistryRequest) (*docker.ManifestResponse, error) { //nolint:lll
	s.logger.Debug("Handling tags request", logging.F("repository", request.Repository))

	// 캐시 확인
	cacheKey := s.cacheManager.GenerateCacheKey(request.Repository, "", "tags")
	if entry, err := s.cacheManager.Get(ctx, cacheKey); err == nil {
		if s.cacheManager.ValidateCacheEntry(ctx, entry) {
			_ = s.metricsCollector.RecordCacheHit(ctx, true, "tags")

			return &docker.ManifestResponse{
				Data:        entry.Headers, // tags 데이터는 headers에 저장
				ContentType: entry.ContentType,
				StatusCode:  http.StatusOK,
				FromCache:   true,
			}, nil
		}
	}

	// 업스트림에서 태그 목록 가져오기
	registry, err := s.registryManager.SelectRegistry(ctx, request.Repository)
	if err != nil {
		return nil, fmt.Errorf("failed to select registry: %w", err)
	}

	url := s.registryManager.BuildUpstreamURL(registry, fmt.Sprintf("/v2/%s/tags/list", request.Repository))

	// HTTP 요청 생성 및 실행
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 인증 설정
	if auth, err := s.authManager.GetAuthToken(ctx, registry.URL, request.Repository); err == nil {
		if auth.Type == authTypeBearer && auth.Token != "" {
			_ = s.authManager.SetBearerAuth(req, auth.Token)
		} else if auth.Type == "basic" && auth.Username != "" {
			_ = s.authManager.SetBasicAuth(req, auth.Username, auth.Password)
		}
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		_ = s.registryManager.MarkRegistryFailed(ctx, registry.URL, err)
		return nil, fmt.Errorf("failed to fetch tags: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		// 에러 응답 본문 읽기 (Docker Registry v2 API 에러 형식 포함)
		errorData, _ := io.ReadAll(resp.Body)
		return &docker.ManifestResponse{
			StatusCode: resp.StatusCode,
			Headers:    make(map[string]string),
			Data:       errorData,
		}, nil
	}

	// 응답 데이터 읽기
	data := make([]byte, resp.ContentLength)
	if _, err := resp.Body.Read(data); err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// 캐시에 저장
	entry := &docker.CacheEntry{
		Key:         cacheKey,
		ContentType: resp.Header.Get("Content-Type"),
		Size:        int64(len(data)),
		CreatedAt:   time.Now(),
		TTL:         6 * time.Hour, // 태그는 비교적 짧은 TTL
		Repository:  request.Repository,
		Headers:     data, // 태그 데이터
	}

	_ = s.cacheManager.Set(ctx, entry)
	_ = s.metricsCollector.RecordCacheHit(ctx, false, "tags")

	return &docker.ManifestResponse{
		Data:         data,
		ContentType:  resp.Header.Get("Content-Type"),
		StatusCode:   http.StatusOK,
		Headers:      make(map[string]string),
		FromCache:    false,
		RegistryUsed: registry.Name,
	}, nil
}

// handleCatalogRequest 카탈로그 요청 처리
func (s *registryServiceImpl) handleCatalogRequest(ctx context.Context, request *docker.RegistryRequest) (*docker.ManifestResponse, error) { //nolint:lll
	s.logger.Debug("Handling catalog request")

	// 카탈로그는 일반적으로 캐시하지 않음 (동적 데이터)
	registry, err := s.registryManager.GetDefaultRegistry(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get default registry: %w", err)
	}

	url := s.registryManager.BuildUpstreamURL(registry, "/v2/_catalog")

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create catalog request: %w", err)
	}

	// 인증이 필요한 경우 설정
	if auth, err := s.authManager.GetAuthToken(ctx, registry.URL, ""); err == nil {
		if auth.Type == authTypeBearer && auth.Token != "" {
			_ = s.authManager.SetBearerAuth(req, auth.Token)
		}
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch catalog: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	data := make([]byte, resp.ContentLength)
	if _, err := resp.Body.Read(data); err != nil {
		return nil, fmt.Errorf("failed to read catalog response: %w", err)
	}

	return &docker.ManifestResponse{
		Data:         data,
		ContentType:  resp.Header.Get("Content-Type"),
		StatusCode:   resp.StatusCode,
		Headers:      make(map[string]string),
		FromCache:    false,
		RegistryUsed: registry.Name,
	}, nil
}
