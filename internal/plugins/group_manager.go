package plugins

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"proxynd/internal/logging"
)

// DefaultGroupManager 기본 그룹 매니저 구현
type DefaultGroupManager struct {
	client    *http.Client
	timeout   time.Duration
	maxRetry  int
	logger    logging.Logger
	userAgent string
}

// GroupManagerConfig 그룹 매니저 설정
type GroupManagerConfig struct {
	Timeout   time.Duration `json:"timeout" yaml:"timeout"`
	MaxRetry  int           `json:"max_retry" yaml:"max_retry"`
	UserAgent string        `json:"user_agent" yaml:"user_agent"`
}

// NewGroupManager 새로운 그룹 매니저 생성
func NewGroupManager(config GroupManagerConfig) GroupManager {
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	if config.MaxRetry == 0 {
		config.MaxRetry = 3
	}
	if config.UserAgent == "" {
		config.UserAgent = "ProxyND/1.0 GroupManager"
	}

	return &DefaultGroupManager{
		client: &http.Client{
			Timeout: config.Timeout,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		timeout:   config.Timeout,
		maxRetry:  config.MaxRetry,
		logger:    logging.GetLogger(),
		userAgent: config.UserAgent,
	}
}

// FetchOnDemand 프록시 모드: 요청 시점에 그룹에서 데이터 가져오기
func (m *DefaultGroupManager) FetchOnDemand(
	ctx context.Context,
	path string,
	upstreams []UpstreamConfig,
) ([]GroupResult, error) {
	return m.fetchFromGroup(ctx, path, upstreams, false)
}

// SyncFromMirrors 미러 모드: 미리 그룹에서 모든 데이터 동기화
func (m *DefaultGroupManager) SyncFromMirrors(ctx context.Context, mirrors []MirrorConfig) error {
	var wg sync.WaitGroup
	errChan := make(chan error, len(mirrors))

	// 미러별로 병렬 동기화
	for _, mirror := range mirrors {
		wg.Add(1)
		go func(mirror MirrorConfig) {
			defer wg.Done()
			if err := m.SyncFromSingleMirror(ctx, mirror); err != nil {
				m.logger.Error("Failed to sync from mirror",
					logging.F("mirror_url", mirror.URL),
					logging.F("error", err))
				errChan <- err
			}
		}(mirror)
	}

	wg.Wait()
	close(errChan)

	// 에러 수집
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	if len(errors) == len(mirrors) {
		return fmt.Errorf("all mirrors failed to sync")
	}

	if len(errors) > 0 {
		m.logger.Warn("Some mirrors failed to sync",
			logging.F("failed_count", len(errors)),
			logging.F("total_count", len(mirrors)))
	}

	return nil
}

// SyncFromSingleMirror 단일 미러 동기화
func (m *DefaultGroupManager) SyncFromSingleMirror(ctx context.Context, mirror MirrorConfig) error {
	m.logger.Info("Starting mirror sync",
		logging.F("mirror_url", mirror.URL),
		logging.F("sync_interval", mirror.SyncInterval))

	// 미러에서 패키지 목록 조회
	packageList, err := m.fetchPackageList(ctx, mirror)
	if err != nil {
		return fmt.Errorf("failed to fetch package list: %w", err)
	}

	// 필터링 적용
	filteredPackages := m.applyFilters(packageList, mirror.IncludePattern, mirror.ExcludePattern)

	m.logger.Info("Package list filtered",
		logging.F("total_packages", len(packageList)),
		logging.F("filtered_packages", len(filteredPackages)))

	// 패키지별 동기화 (병렬 처리)
	return m.syncPackages(ctx, mirror, filteredPackages)
}

// HealthCheckGroup 그룹 헬스체크
func (m *DefaultGroupManager) HealthCheckGroup(
	ctx context.Context,
	upstreams []UpstreamConfig,
) (map[string]bool, error) {
	healthResults := make(map[string]bool)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, upstream := range upstreams {
		wg.Add(1)
		go func(upstream UpstreamConfig) {
			defer wg.Done()

			healthy := m.checkSingleUpstream(ctx, upstream)

			mu.Lock()
			healthResults[upstream.URL] = healthy
			mu.Unlock()
		}(upstream)
	}

	wg.Wait()
	return healthResults, nil
}

// fetchFromGroup 그룹에서 데이터 가져오기 (내부 함수)
func (m *DefaultGroupManager) fetchFromGroup(
	ctx context.Context,
	path string,
	upstreams []UpstreamConfig,
	parallel bool,
) ([]GroupResult, error) {
	if parallel {
		return m.fetchParallel(ctx, path, upstreams)
	}

	return m.fetchSequential(ctx, path, upstreams)
}

// fetchParallel 병렬로 여러 업스트림에서 가져오기
func (m *DefaultGroupManager) fetchParallel(
	ctx context.Context,
	path string,
	upstreams []UpstreamConfig,
) ([]GroupResult, error) {
	resultsChan := make(chan GroupResult, len(upstreams))
	var wg sync.WaitGroup

	for _, upstream := range upstreams {
		wg.Add(1)
		go func(upstream UpstreamConfig) {
			defer wg.Done()
			result := m.fetchFromSingleUpstream(ctx, path, upstream)
			resultsChan <- result
		}(upstream)
	}

	wg.Wait()
	close(resultsChan)

	results := make([]GroupResult, 0, len(upstreams))
	for result := range resultsChan {
		results = append(results, result)
	}

	return results, nil
}

// fetchSequential 순차적으로 업스트림에서 가져오기
func (m *DefaultGroupManager) fetchSequential(
	ctx context.Context,
	path string,
	upstreams []UpstreamConfig,
) ([]GroupResult, error) {
	results := make([]GroupResult, 0, len(upstreams))

	for _, upstream := range upstreams {
		result := m.fetchFromSingleUpstream(ctx, path, upstream)
		results = append(results, result)

		// 성공하면 우선순위에 따라 중단할 수 있음
		if result.Error == nil && result.StatusCode == http.StatusOK {
			// 높은 우선순위의 업스트림에서 성공하면 중단
			if upstream.Priority > 0 {
				break
			}
		}
	}

	return results, nil
}

// fetchFromSingleUpstream 단일 업스트림에서 가져오기
func (m *DefaultGroupManager) fetchFromSingleUpstream(
	ctx context.Context,
	path string,
	upstream UpstreamConfig,
) GroupResult {
	start := time.Now()

	result := GroupResult{
		UpstreamURL: upstream.URL,
		Headers:     make(map[string]string),
	}

	fullURL := upstream.URL + path

	// HTTP 요청 생성
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		result.Error = fmt.Errorf("failed to create request: %w", err)
		result.Duration = time.Since(start)
		return result
	}

	// 헤더 설정
	req.Header.Set("User-Agent", m.userAgent)
	for key, value := range upstream.Headers {
		req.Header.Set(key, value)
	}

	// 인증 설정
	if upstream.Username != "" && upstream.Password != "" {
		req.SetBasicAuth(upstream.Username, upstream.Password)
	}

	// 재시도 로직
	var resp *http.Response
	for retry := 0; retry < m.maxRetry; retry++ {
		resp, err = m.client.Do(req)
		if err == nil {
			break
		}

		if retry < m.maxRetry-1 {
			select {
			case <-ctx.Done():
				result.Error = ctx.Err()
				result.Duration = time.Since(start)
				return result
			case <-time.After(time.Duration(retry+1) * time.Second):
				// 재시도 대기
			}
		}
	}

	if err != nil {
		result.Error = fmt.Errorf("request failed after %d retries: %w", m.maxRetry, err)
		result.Duration = time.Since(start)
		return result
	}

	defer func() { _ = resp.Body.Close() }()

	result.StatusCode = resp.StatusCode

	// 응답 헤더 복사
	for key, values := range resp.Header {
		if len(values) > 0 {
			result.Headers[key] = values[0]
		}
	}

	// 응답 바디 읽기
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		result.Error = fmt.Errorf("failed to read response body: %w", err)
	} else {
		result.Body = body
	}

	result.Duration = time.Since(start)
	return result
}

// fetchPackageList 미러에서 패키지 목록 조회
func (m *DefaultGroupManager) fetchPackageList(_ context.Context, mirror MirrorConfig) ([]string, error) {
	// 여기서는 간단한 구현으로 기본 패키지 목록을 반환
	// 실제 구현에서는 미러의 패키지 인덱스를 파싱해야 함
	m.logger.Debug("Fetching package list from mirror",
		logging.F("mirror_url", mirror.URL))

	// TODO: 실제 미러별 패키지 목록 조회 로직 구현
	return []string{}, nil
}

// applyFilters 포함/제외 패턴 필터링 적용
func (m *DefaultGroupManager) applyFilters(packages, includePatterns, excludePatterns []string) []string {
	if len(includePatterns) == 0 && len(excludePatterns) == 0 {
		return packages
	}

	// TODO: 정규식 패턴 매칭 구현
	return packages
}

// syncPackages 패키지 동기화 (병렬 처리)
func (m *DefaultGroupManager) syncPackages(_ context.Context, mirror MirrorConfig, packages []string) error {
	// TODO: 실제 패키지 동기화 로직 구현
	m.logger.Info("Package sync completed",
		logging.F("mirror_url", mirror.URL),
		logging.F("package_count", len(packages)))

	return nil
}

// checkSingleUpstream 단일 업스트림 헬스체크
func (m *DefaultGroupManager) checkSingleUpstream(ctx context.Context, upstream UpstreamConfig) bool {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodHead, upstream.URL, nil)
	if err != nil {
		return false
	}

	// 인증 설정
	if upstream.Username != "" && upstream.Password != "" {
		req.SetBasicAuth(upstream.Username, upstream.Password)
	}

	resp, err := m.client.Do(req)
	if err != nil {
		return false
	}
	defer func() { _ = resp.Body.Close() }()

	return resp.StatusCode < 400
}
