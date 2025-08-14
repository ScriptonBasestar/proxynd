// Package health provides proxy-specific health checking functionality
package health

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"proxynd/internal/config"
)

// ProxyHealthChecker 프록시별 헬스체크를 제공하는 구조체
type ProxyHealthChecker struct {
	config         *config.RootConfig
	client         *http.Client
	circuitBreaker map[string]*CircuitBreaker
	mutex          sync.RWMutex
}

// NewProxyHealthChecker 새 프록시 헬스체커 생성
func NewProxyHealthChecker(config *config.RootConfig) *ProxyHealthChecker {
	return &ProxyHealthChecker{
		config: config,
		client: &http.Client{
			Timeout: 5 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:       10,
				IdleConnTimeout:    30 * time.Second,
				DisableCompression: false,
			},
		},
		circuitBreaker: make(map[string]*CircuitBreaker),
	}
}

// Name 체커 이름 반환
func (phc *ProxyHealthChecker) Name() string {
	return "proxy_upstreams"
}

// Check 모든 활성화된 프록시의 업스트림 연결성 확인
func (phc *ProxyHealthChecker) Check(ctx context.Context) *CheckResult {
	start := time.Now()
	result := &CheckResult{
		Name:        phc.Name(),
		Status:      StatusHealthy,
		LastChecked: time.Now(),
		Details:     make(map[string]interface{}),
	}

	proxyResults := make(map[string]*ProxyUpstreamResult)
	var wg sync.WaitGroup
	var resultsMutex sync.Mutex

	// 활성화된 프록시별로 헬스체크 실행
	proxyCheckers := []struct {
		name      string
		enabled   bool
		checkFunc func(context.Context) *ProxyUpstreamResult
	}{
		{"npm", phc.config.Registries.NPM.Enabled, phc.checkNPMUpstream},
		{"pypi", phc.config.Registries.PyPI.Enabled, phc.checkPyPIUpstream},
		{"apt", phc.config.Registries.APT.Enabled, phc.checkAPTUpstream},
		{"docker", phc.config.Registries.Docker.Enabled, phc.checkDockerUpstream},
		{"maven", phc.config.Registries.Maven.Enabled, phc.checkMavenUpstream},
	}

	for _, checker := range proxyCheckers {
		if !checker.enabled {
			continue
		}

		wg.Add(1)
		go func(name string, checkFunc func(context.Context) *ProxyUpstreamResult) {
			defer wg.Done()

			checkResult := checkFunc(ctx)

			resultsMutex.Lock()
			proxyResults[name] = checkResult
			resultsMutex.Unlock()
		}(checker.name, checker.checkFunc)
	}

	wg.Wait()

	// 결과 분석
	var healthyCount, degradedCount, unhealthyCount int
	for proxyType, proxyResult := range proxyResults {
		result.Details[proxyType] = map[string]interface{}{
			"status":           string(proxyResult.Status),
			"message":          proxyResult.Message,
			"response_time_ms": proxyResult.ResponseTime.Milliseconds(),
			"upstream_url":     proxyResult.UpstreamURL,
			"status_code":      proxyResult.StatusCode,
			"error":            proxyResult.Error,
		}

		switch proxyResult.Status {
		case StatusHealthy:
			healthyCount++
		case StatusDegraded:
			degradedCount++
		case StatusUnhealthy:
			unhealthyCount++
		}
	}

	result.Details["summary"] = map[string]interface{}{
		"total_proxies":     len(proxyResults),
		"healthy_proxies":   healthyCount,
		"degraded_proxies":  degradedCount,
		"unhealthy_proxies": unhealthyCount,
	}

	// 전체 상태 결정
	if unhealthyCount > 0 {
		if unhealthyCount == len(proxyResults) {
			result.Status = StatusUnhealthy
			result.Message = "모든 프록시 업스트림이 비정상 상태입니다"
		} else {
			result.Status = StatusDegraded
			result.Message = fmt.Sprintf("일부 프록시 업스트림이 비정상입니다 (%d/%d)",
				unhealthyCount, len(proxyResults))
		}
	} else if degradedCount > 0 {
		result.Status = StatusDegraded
		result.Message = fmt.Sprintf("일부 프록시 업스트림에서 성능 저하가 감지됩니다 (%d/%d)",
			degradedCount, len(proxyResults))
	} else {
		result.Message = fmt.Sprintf("모든 프록시 업스트림이 정상입니다 (%d개)",
			len(proxyResults))
	}

	result.Duration = time.Since(start)
	return result
}

// ProxyUpstreamResult 프록시 업스트림 체크 결과
type ProxyUpstreamResult struct {
	Status       Status
	Message      string
	UpstreamURL  string
	StatusCode   int
	ResponseTime time.Duration
	Error        string
}

// checkNPMUpstream NPM 업스트림 연결성 확인
func (phc *ProxyHealthChecker) checkNPMUpstream(ctx context.Context) *ProxyUpstreamResult {
	upstreamURL := phc.config.Registries.NPM.Upstream
	if upstreamURL == "" {
		upstreamURL = "https://registry.npmjs.org"
	}

	// NPM registry ping 엔드포인트 사용
	pingURL := strings.TrimSuffix(upstreamURL, "/") + "/-/ping"

	return phc.checkHTTPUpstream("npm", pingURL, []int{200}, func(body string) bool {
		// NPM ping은 빈 객체 {} 또는 간단한 응답을 반환
		return len(body) > 0
	})
}

// checkPyPIUpstream PyPI 업스트림 연결성 확인
func (phc *ProxyHealthChecker) checkPyPIUpstream(ctx context.Context) *ProxyUpstreamResult {
	simpleURL := phc.config.Registries.PyPI.Simple
	if simpleURL == "" {
		simpleURL = "https://pypi.org/simple"
	}

	// PyPI simple 인덱스 페이지 확인
	return phc.checkHTTPUpstream("pypi", simpleURL+"/", []int{200}, func(body string) bool {
		// PyPI simple 페이지는 HTML을 반환하고 "simple" 문자열을 포함
		return strings.Contains(strings.ToLower(body), "simple") ||
			strings.Contains(body, "<html")
	})
}

// checkAPTUpstream APT 업스트림 연결성 확인
func (phc *ProxyHealthChecker) checkAPTUpstream(ctx context.Context) *ProxyUpstreamResult {
	// APT 미러 중 첫 번째 활성 미러 확인
	for distro, mirrors := range phc.config.Registries.APT.Mirrors {
		if len(mirrors) > 0 {
			mirrorURL := mirrors[0].URL
			// Release 파일 존재 확인
			releaseURL := strings.TrimSuffix(mirrorURL, "/") + "/dists/stable/Release"

			result := phc.checkHTTPUpstream("apt", releaseURL, []int{200, 404}, func(body string) bool {
				// Release 파일이 있거나 404이면 정상 (미러가 응답함)
				return true
			})

			result.UpstreamURL = fmt.Sprintf("%s (distro: %s)", mirrorURL, distro)
			return result
		}
	}

	return &ProxyUpstreamResult{
		Status:      StatusUnhealthy,
		Message:     "APT 미러가 설정되지 않았습니다",
		UpstreamURL: "none",
		Error:       "no mirrors configured",
	}
}

// checkDockerUpstream Docker 레지스트리 연결성 확인
func (phc *ProxyHealthChecker) checkDockerUpstream(ctx context.Context) *ProxyUpstreamResult {
	// Docker 레지스트리 중 첫 번째 확인
	if len(phc.config.Registries.Docker.Registries) > 0 {
		registry := phc.config.Registries.Docker.Registries[0]
		// Docker v2 API 버전 확인
		versionURL := strings.TrimSuffix(registry.URL, "/") + "/v2/"

		return phc.checkHTTPUpstream("docker", versionURL, []int{200, 401}, func(body string) bool {
			// Docker registry v2 API는 200 또는 401을 반환 (인증 필요)
			return true
		})
	}

	return &ProxyUpstreamResult{
		Status:      StatusUnhealthy,
		Message:     "Docker 레지스트리가 설정되지 않았습니다",
		UpstreamURL: "none",
		Error:       "no registries configured",
	}
}

// checkMavenUpstream Maven 저장소 연결성 확인
func (phc *ProxyHealthChecker) checkMavenUpstream(ctx context.Context) *ProxyUpstreamResult {
	// Maven 저장소 중 첫 번째 확인
	if len(phc.config.Registries.Maven.Repositories) > 0 {
		repo := phc.config.Registries.Maven.Repositories[0]
		// Maven 중앙 저장소의 메타데이터 확인
		metadataURL := strings.TrimSuffix(repo.URL, "/") + "/org/apache/maven/maven-core/maven-metadata.xml"

		return phc.checkHTTPUpstream("maven", metadataURL, []int{200, 404}, func(body string) bool {
			// 메타데이터가 있거나 404이면 정상 (저장소가 응답함)
			return true
		})
	}

	return &ProxyUpstreamResult{
		Status:      StatusUnhealthy,
		Message:     "Maven 저장소가 설정되지 않았습니다",
		UpstreamURL: "none",
		Error:       "no repositories configured",
	}
}

// checkHTTPUpstream HTTP 업스트림 일반 체크 함수
func (phc *ProxyHealthChecker) checkHTTPUpstream(
	proxyType, checkURL string,
	validStatusCodes []int,
	bodyValidator func(string) bool,
) *ProxyUpstreamResult {
	start := time.Now()
	result := &ProxyUpstreamResult{
		Status:      StatusHealthy,
		UpstreamURL: checkURL,
	}

	// URL 유효성 검사
	if _, err := url.Parse(checkURL); err != nil {
		result.Status = StatusUnhealthy
		result.Message = fmt.Sprintf("잘못된 업스트림 URL: %v", err)
		result.Error = err.Error()
		return result
	}

	// 서킷 브레이커를 통한 요청
	cb := phc.getCircuitBreaker(proxyType)

	err := cb.Execute(context.Background(), func(ctx context.Context) error {
		req, err := http.NewRequestWithContext(ctx, "GET", checkURL, nil)
		if err != nil {
			return err
		}

		// User-Agent 헤더 추가
		req.Header.Set("User-Agent", "ProxyND-HealthCheck/1.0")

		resp, err := phc.client.Do(req)
		if err != nil {
			return err
		}
		defer func() { _ = resp.Body.Close() }()

		result.StatusCode = resp.StatusCode
		result.ResponseTime = time.Since(start)

		// 상태 코드 검증
		validStatus := false
		for _, code := range validStatusCodes {
			if resp.StatusCode == code {
				validStatus = true
				break
			}
		}

		if !validStatus {
			return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}

		// 응답 본문 검증 (선택적)
		if bodyValidator != nil {
			body := make([]byte, 1024) // 처음 1KB만 읽기
			n, _ := resp.Body.Read(body)
			if !bodyValidator(string(body[:n])) {
				return fmt.Errorf("body validation failed")
			}
		}

		return nil
	})

	if err != nil {
		if IsCircuitBreakerError(err) {
			result.Status = StatusDegraded
			result.Message = fmt.Sprintf("%s 업스트림 서킷 브레이커가 열려있습니다", strings.ToUpper(proxyType))
		} else {
			result.Status = StatusUnhealthy
			result.Message = fmt.Sprintf("%s 업스트림 연결 실패: %v", strings.ToUpper(proxyType), err)
		}
		result.Error = err.Error()
	} else {
		result.Message = fmt.Sprintf("%s 업스트림이 정상입니다", strings.ToUpper(proxyType))

		// 응답 시간에 따른 성능 평가
		if result.ResponseTime > 2*time.Second {
			result.Status = StatusDegraded
			result.Message = fmt.Sprintf("%s 업스트림 응답이 느립니다 (%.2fs)",
				strings.ToUpper(proxyType), result.ResponseTime.Seconds())
		}
	}

	return result
}

// getCircuitBreaker 프록시 타입별 서킷 브레이커 반환
func (phc *ProxyHealthChecker) getCircuitBreaker(proxyType string) *CircuitBreaker {
	phc.mutex.Lock()
	defer phc.mutex.Unlock()

	if cb, exists := phc.circuitBreaker[proxyType]; exists {
		return cb
	}

	// 프록시 타입별 서킷 브레이커 생성
	config := DefaultCircuitBreakerConfig(fmt.Sprintf("upstream_%s", proxyType))
	config.FailureThreshold = 3
	config.Timeout = 30 * time.Second

	phc.circuitBreaker[proxyType] = NewCircuitBreaker(config)
	return phc.circuitBreaker[proxyType]
}

// GetUpstreamStatus 특정 프록시의 업스트림 상태 반환
func (phc *ProxyHealthChecker) GetUpstreamStatus(proxyType string) (*ProxyUpstreamResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	switch strings.ToLower(proxyType) {
	case "npm":
		if !phc.config.Registries.NPM.Enabled {
			return nil, fmt.Errorf("NPM proxy is not enabled")
		}
		return phc.checkNPMUpstream(ctx), nil
	case "pypi", "pip":
		if !phc.config.Registries.PyPI.Enabled {
			return nil, fmt.Errorf("PyPI proxy is not enabled")
		}
		return phc.checkPyPIUpstream(ctx), nil
	case "apt":
		if !phc.config.Registries.APT.Enabled {
			return nil, fmt.Errorf("APT proxy is not enabled")
		}
		return phc.checkAPTUpstream(ctx), nil
	case "docker":
		if !phc.config.Registries.Docker.Enabled {
			return nil, fmt.Errorf("docker proxy is not enabled")
		}
		return phc.checkDockerUpstream(ctx), nil
	case "maven":
		if !phc.config.Registries.Maven.Enabled {
			return nil, fmt.Errorf("maven proxy is not enabled")
		}
		return phc.checkMavenUpstream(ctx), nil
	default:
		return nil, fmt.Errorf("unsupported proxy type: %s", proxyType)
	}
}

// GetCircuitBreakerStatus 서킷 브레이커 상태 반환
func (phc *ProxyHealthChecker) GetCircuitBreakerStatus() map[string]interface{} {
	phc.mutex.RLock()
	defer phc.mutex.RUnlock()

	status := make(map[string]interface{})
	for proxyType, cb := range phc.circuitBreaker {
		status[proxyType] = cb.GetStatus()
	}

	return status
}
