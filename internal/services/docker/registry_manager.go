package docker

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"proxynd/internal/domain/docker"
	"proxynd/logging"
)

// registryManagerImpl Docker 레지스트리 서버 관리 구현
type registryManagerImpl struct {
	config          docker.ProxyConfig
	logger          logging.Logger
	registryStatus  map[string]*docker.RegistryStatus
	roundRobinIndex int
	statusMutex     sync.RWMutex
}

// NewRegistryManager RegistryManager 생성자
func NewRegistryManager(
	config docker.ProxyConfig,
	logger logging.Logger,
) docker.RegistryManager {
	return &registryManagerImpl{
		config:          config,
		logger:          logger,
		registryStatus:  make(map[string]*docker.RegistryStatus),
		roundRobinIndex: 0,
		statusMutex:     sync.RWMutex{},
	}
}

// SelectRegistry 요청에 적합한 레지스트리 선택 (라운드로빈, 가용성 기반)
func (r *registryManagerImpl) SelectRegistry(ctx context.Context, repository string) (*docker.RegistryStatus, error) {
	r.logger.Debug("Selecting registry", logging.F("repository", repository))

	registries := r.config.GetRegistries()
	if len(registries) == 0 {
		return nil, fmt.Errorf("no registries configured")
	}

	// 가용한 레지스트리 필터링
	availableRegistries := make([]*docker.RegistryStatus, 0)

	for _, registry := range registries {
		status, err := r.GetRegistryStatus(ctx, registry.URL)
		if err != nil {
			r.logger.Warn("Failed to get registry status", logging.F("url", registry.URL), logging.F("error", err))
			continue
		}

		if status.Available {
			availableRegistries = append(availableRegistries, status)
		}
	}

	if len(availableRegistries) == 0 {
		return nil, fmt.Errorf("no available registries for repository: %s", repository)
	}

	// 라운드로빈 선택
	r.statusMutex.Lock()
	selectedIndex := r.roundRobinIndex % len(availableRegistries)
	r.roundRobinIndex++
	r.statusMutex.Unlock()

	selected := availableRegistries[selectedIndex]
	r.logger.Debug("Registry selected", logging.F("name", selected.Name), logging.F("url", selected.URL))

	return selected, nil
}

// CheckRegistryHealth 레지스트리 서버 상태 확인
func (r *registryManagerImpl) CheckRegistryHealth(ctx context.Context, registryURL string) (*docker.RegistryStatus, error) {
	r.logger.Debug("Checking registry health", logging.F("url", registryURL))

	startTime := time.Now()

	// Docker Registry v2 헬스체크 (/v2/ 엔드포인트)
	healthURL := fmt.Sprintf("%s/v2/", strings.TrimRight(registryURL, "/"))

	req, err := http.NewRequestWithContext(ctx, "GET", healthURL, nil)
	if err != nil {
		return r.createFailedStatus(registryURL, err), nil
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(req)
	responseTime := float64(time.Since(startTime).Nanoseconds()) / 1e6 // milliseconds

	if err != nil {
		return r.createFailedStatus(registryURL, err), nil
	}
	defer resp.Body.Close()

	// 상태 정보 생성
	status := &docker.RegistryStatus{
		Name:         r.extractRegistryName(registryURL),
		URL:          registryURL,
		Available:    resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusUnauthorized,
		LastCheck:    time.Now(),
		ResponseTime: responseTime,
		APIVersion:   r.detectAPIVersion(resp.Header),
		Features:     r.detectFeatures(resp.Header),
	}

	if !status.Available {
		status.Error = fmt.Sprintf("HTTP %d", resp.StatusCode)
	}

	r.logger.Debug("Registry health check completed",
		logging.F("url", registryURL),
		logging.F("available", status.Available),
		logging.F("responseTime", responseTime))

	return status, nil
}

// GetRegistryStatus 레지스트리 상태 정보 반환
func (r *registryManagerImpl) GetRegistryStatus(ctx context.Context, registryURL string) (*docker.RegistryStatus, error) {
	r.statusMutex.RLock()
	if status, exists := r.registryStatus[registryURL]; exists {
		// 캐시된 상태가 최근 것인지 확인 (5분 이내)
		if time.Since(status.LastCheck) < 5*time.Minute {
			r.statusMutex.RUnlock()
			return status, nil
		}
	}
	r.statusMutex.RUnlock()

	// 새로운 헬스체크 수행
	status, err := r.CheckRegistryHealth(ctx, registryURL)
	if err != nil {
		return nil, err
	}

	// 상태 캐시 업데이트
	r.statusMutex.Lock()
	r.registryStatus[registryURL] = status
	r.statusMutex.Unlock()

	return status, nil
}

// MarkRegistryFailed 레지스트리 실패 마킹 (일시적 제외)
func (r *registryManagerImpl) MarkRegistryFailed(ctx context.Context, registryURL string, err error) error {
	r.logger.Warn("Marking registry as failed", logging.F("url", registryURL), logging.F("error", err))

	r.statusMutex.Lock()
	defer r.statusMutex.Unlock()

	status, exists := r.registryStatus[registryURL]
	if !exists {
		status = &docker.RegistryStatus{
			Name: r.extractRegistryName(registryURL),
			URL:  registryURL,
		}
		r.registryStatus[registryURL] = status
	}

	status.Available = false
	status.LastCheck = time.Now()
	status.Error = err.Error()
	status.ResponseTime = 0

	return nil
}

// BuildUpstreamURL 업스트림 레지스트리 URL 구성
func (r *registryManagerImpl) BuildUpstreamURL(registry *docker.RegistryStatus, requestPath string) string {
	baseURL := strings.TrimRight(registry.URL, "/")

	// 경로가 /v2/로 시작하지 않으면 추가
	if !strings.HasPrefix(requestPath, "/v2/") && !strings.HasPrefix(requestPath, "v2/") {
		if strings.HasPrefix(requestPath, "/") {
			requestPath = "/v2" + requestPath
		} else {
			requestPath = "/v2/" + requestPath
		}
	}

	// 경로가 /로 시작하지 않으면 추가
	if !strings.HasPrefix(requestPath, "/") {
		requestPath = "/" + requestPath
	}

	return baseURL + requestPath
}

// CopyRequestHeaders 요청 헤더 복사 (Accept, Authorization 등)
func (r *registryManagerImpl) CopyRequestHeaders(source http.Header, target *http.Request) error {
	// Docker Registry에서 중요한 헤더들
	importantHeaders := []string{
		"Accept",
		"Authorization",
		"User-Agent",
		"Docker-Distribution-Api-Version",
		"If-None-Match",
		"If-Modified-Since",
	}

	for _, header := range importantHeaders {
		if values := source.Values(header); len(values) > 0 {
			for _, value := range values {
				target.Header.Add(header, value)
			}
		}
	}

	return nil
}

// GetDefaultRegistry 기본 레지스트리 반환
func (r *registryManagerImpl) GetDefaultRegistry(ctx context.Context) (*docker.RegistryStatus, error) {
	defaultRegistry := r.config.GetDefaultRegistry()
	if defaultRegistry == nil {
		return nil, fmt.Errorf("no default registry configured")
	}

	return r.GetRegistryStatus(ctx, defaultRegistry.URL)
}

// createFailedStatus 실패한 레지스트리 상태 생성
func (r *registryManagerImpl) createFailedStatus(registryURL string, err error) *docker.RegistryStatus {
	return &docker.RegistryStatus{
		Name:         r.extractRegistryName(registryURL),
		URL:          registryURL,
		Available:    false,
		LastCheck:    time.Now(),
		Error:        err.Error(),
		ResponseTime: 0,
		APIVersion:   "unknown",
		Features:     []string{},
	}
}

// extractRegistryName URL에서 레지스트리 이름 추출
func (r *registryManagerImpl) extractRegistryName(registryURL string) string {
	// URL에서 호스트명 추출
	if strings.HasPrefix(registryURL, "http://") {
		registryURL = strings.TrimPrefix(registryURL, "http://")
	} else if strings.HasPrefix(registryURL, "https://") {
		registryURL = strings.TrimPrefix(registryURL, "https://")
	}

	// 포트 번호 제거
	if idx := strings.Index(registryURL, ":"); idx != -1 {
		registryURL = registryURL[:idx]
	}

	// 경로 제거
	if idx := strings.Index(registryURL, "/"); idx != -1 {
		registryURL = registryURL[:idx]
	}

	return registryURL
}

// detectAPIVersion 응답 헤더에서 API 버전 감지
func (r *registryManagerImpl) detectAPIVersion(headers http.Header) string {
	if version := headers.Get("Docker-Distribution-Api-Version"); version != "" {
		return version
	}

	// 기본값
	return "registry/2.0"
}

// detectFeatures 응답 헤더에서 지원 기능 감지
func (r *registryManagerImpl) detectFeatures(headers http.Header) []string {
	features := make([]string, 0)

	// Docker Registry v2 기본 기능
	features = append(features, "manifests", "blobs")

	// 추가 기능 감지
	if headers.Get("Docker-Distribution-Api-Version") != "" {
		features = append(features, "distribution-api")
	}

	// OCI 지원 확인
	if accept := headers.Get("Accept"); strings.Contains(accept, "oci") {
		features = append(features, "oci")
	}

	return features
}

// performPeriodicHealthCheck 주기적 헬스체크 (백그라운드 작업)
func (r *registryManagerImpl) performPeriodicHealthCheck(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.checkAllRegistries(ctx)
		}
	}
}

// checkAllRegistries 모든 레지스트리 상태 확인
func (r *registryManagerImpl) checkAllRegistries(ctx context.Context) {
	registries := r.config.GetRegistries()

	for _, registry := range registries {
		if _, err := r.CheckRegistryHealth(ctx, registry.URL); err != nil {
			r.logger.Warn("Periodic health check failed", logging.F("url", registry.URL), logging.F("error", err))
		}
	}

	r.logger.Debug("Periodic health check completed", logging.F("registries", len(registries)))
}
