package health

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Status 건강 상태
type Status string

const (
	StatusHealthy   Status = "healthy"
	StatusDegraded  Status = "degraded"
	StatusUnhealthy Status = "unhealthy"
)

// CheckResult 체크 결과
type CheckResult struct {
	Name        string                 `json:"name"`
	Status      Status                 `json:"status"`
	Message     string                 `json:"message,omitempty"`
	Duration    time.Duration          `json:"duration_ms"`
	Details     map[string]interface{} `json:"details,omitempty"`
	LastChecked time.Time              `json:"last_checked"`
}

// HealthChecker 건강 상태 체크 인터페이스
type HealthChecker interface {
	Check(ctx context.Context) *CheckResult
	Name() string
}

// HealthService 건강 상태 서비스
type HealthService struct {
	checkers      []HealthChecker
	checkInterval time.Duration
	results       map[string]*CheckResult
	resultsMutex  sync.RWMutex
	startTime     time.Time
}

// NewHealthService 새 건강 상태 서비스 생성
func NewHealthService(checkInterval time.Duration) *HealthService {
	return &HealthService{
		checkers:      make([]HealthChecker, 0),
		checkInterval: checkInterval,
		results:       make(map[string]*CheckResult),
		startTime:     time.Now(),
	}
}

// RegisterChecker 체커 등록
func (hs *HealthService) RegisterChecker(checker HealthChecker) {
	hs.checkers = append(hs.checkers, checker)
}

// Start 주기적 체크 시작
func (hs *HealthService) Start(ctx context.Context) {
	// 초기 체크
	hs.runChecks(ctx)

	// 주기적 체크
	ticker := time.NewTicker(hs.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			hs.runChecks(ctx)
		}
	}
}

// runChecks 모든 체크 실행
func (hs *HealthService) runChecks(ctx context.Context) {
	var wg sync.WaitGroup

	for _, checker := range hs.checkers {
		wg.Add(1)
		go func(c HealthChecker) {
			defer wg.Done()

			// 타임아웃 설정
			checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()

			result := c.Check(checkCtx)

			hs.resultsMutex.Lock()
			hs.results[c.Name()] = result
			hs.resultsMutex.Unlock()
		}(checker)
	}

	wg.Wait()
}

// GetStatus 전체 건강 상태 반환
func (hs *HealthService) GetStatus() (Status, map[string]*CheckResult) {
	hs.resultsMutex.RLock()
	defer hs.resultsMutex.RUnlock()

	// 결과 복사
	results := make(map[string]*CheckResult)
	overallStatus := StatusHealthy
	unhealthyCount := 0
	degradedCount := 0

	for name, result := range hs.results {
		results[name] = result

		switch result.Status {
		case StatusUnhealthy:
			unhealthyCount++
		case StatusDegraded:
			degradedCount++
		}
	}

	// 전체 상태 결정
	if unhealthyCount > 0 {
		overallStatus = StatusUnhealthy
	} else if degradedCount > 0 {
		overallStatus = StatusDegraded
	}

	return overallStatus, results
}

// GetUptime 가동 시간 반환
func (hs *HealthService) GetUptime() time.Duration {
	return time.Since(hs.startTime)
}

// 기본 체커 구현들

// EnvironmentChecker 환경 변수 체커
type EnvironmentChecker struct {
	requiredVars []string
}

func NewEnvironmentChecker(vars []string) *EnvironmentChecker {
	return &EnvironmentChecker{requiredVars: vars}
}

func (ec *EnvironmentChecker) Name() string {
	return "environment"
}

func (ec *EnvironmentChecker) Check(ctx context.Context) *CheckResult {
	start := time.Now()
	result := &CheckResult{
		Name:        ec.Name(),
		Status:      StatusHealthy,
		LastChecked: time.Now(),
		Details:     make(map[string]interface{}),
	}

	missingVars := []string{}
	for _, varName := range ec.requiredVars {
		value := os.Getenv(varName)
		result.Details[varName] = value != ""
		if value == "" {
			missingVars = append(missingVars, varName)
		}
	}

	if len(missingVars) > 0 {
		result.Status = StatusUnhealthy
		result.Message = fmt.Sprintf("Missing environment variables: %v", missingVars)
	} else {
		result.Message = "All required environment variables are set"
	}

	result.Duration = time.Since(start)
	return result
}

// DiskSpaceChecker 디스크 공간 체커
type DiskSpaceChecker struct {
	path           string
	minFreeBytes   uint64
	minFreePercent float64
}

func NewDiskSpaceChecker(path string, minFreeBytes uint64, minFreePercent float64) *DiskSpaceChecker {
	return &DiskSpaceChecker{
		path:           path,
		minFreeBytes:   minFreeBytes,
		minFreePercent: minFreePercent,
	}
}

func (dc *DiskSpaceChecker) Name() string {
	return "disk_space"
}

func (dc *DiskSpaceChecker) Check(ctx context.Context) *CheckResult {
	start := time.Now()
	result := &CheckResult{
		Name:        dc.Name(),
		Status:      StatusHealthy,
		LastChecked: time.Now(),
		Details:     make(map[string]interface{}),
	}

	// 디스크 사용량 확인
	stats, err := getDiskUsage(dc.path)
	if err != nil {
		result.Status = StatusUnhealthy
		result.Message = fmt.Sprintf("Failed to get disk usage: %v", err)
		result.Duration = time.Since(start)
		return result
	}

	result.Details["path"] = dc.path
	result.Details["total_bytes"] = stats.Total
	result.Details["free_bytes"] = stats.Free
	result.Details["used_bytes"] = stats.Used
	result.Details["free_percent"] = stats.FreePercent

	// 최소 공간 체크
	if stats.Free < dc.minFreeBytes {
		result.Status = StatusUnhealthy
		result.Message = fmt.Sprintf("Insufficient disk space: %d bytes free (minimum: %d)",
			stats.Free, dc.minFreeBytes)
	} else if stats.FreePercent < dc.minFreePercent {
		result.Status = StatusDegraded
		result.Message = fmt.Sprintf("Low disk space: %.1f%% free (minimum: %.1f%%)",
			stats.FreePercent, dc.minFreePercent)
	} else {
		result.Message = fmt.Sprintf("Disk space OK: %.1f%% free", stats.FreePercent)
	}

	result.Duration = time.Since(start)
	return result
}

// WritableChecker 쓰기 가능 체커
type WritableChecker struct {
	path string
}

func NewWritableChecker(path string) *WritableChecker {
	return &WritableChecker{path: path}
}

func (wc *WritableChecker) Name() string {
	return "writable_" + filepath.Base(wc.path)
}

func (wc *WritableChecker) Check(ctx context.Context) *CheckResult {
	start := time.Now()
	result := &CheckResult{
		Name:        wc.Name(),
		Status:      StatusHealthy,
		LastChecked: time.Now(),
		Details: map[string]interface{}{
			"path": wc.path,
		},
	}

	// 디렉토리 존재 확인
	info, err := os.Stat(wc.path)
	if err != nil {
		if os.IsNotExist(err) {
			// 디렉토리 생성 시도
			if err := os.MkdirAll(wc.path, 0755); err != nil {
				result.Status = StatusUnhealthy
				result.Message = fmt.Sprintf("Cannot create directory: %v", err)
				result.Duration = time.Since(start)
				return result
			}
		} else {
			result.Status = StatusUnhealthy
			result.Message = fmt.Sprintf("Cannot access directory: %v", err)
			result.Duration = time.Since(start)
			return result
		}
	} else if !info.IsDir() {
		result.Status = StatusUnhealthy
		result.Message = "Path exists but is not a directory"
		result.Duration = time.Since(start)
		return result
	}

	// 쓰기 테스트
	testFile := filepath.Join(wc.path, ".health_check")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		result.Status = StatusUnhealthy
		result.Message = fmt.Sprintf("Cannot write to directory: %v", err)
	} else {
		os.Remove(testFile)
		result.Message = "Directory is writable"
	}

	result.Duration = time.Since(start)
	return result
}

// HTTPChecker HTTP 엔드포인트 체커
type HTTPChecker struct {
	name    string
	url     string
	timeout time.Duration
	client  *http.Client
}

func NewHTTPChecker(name, url string, timeout time.Duration) *HTTPChecker {
	return &HTTPChecker{
		name:    name,
		url:     url,
		timeout: timeout,
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

func (hc *HTTPChecker) Name() string {
	return hc.name
}

func (hc *HTTPChecker) Check(ctx context.Context) *CheckResult {
	start := time.Now()
	result := &CheckResult{
		Name:        hc.Name(),
		Status:      StatusHealthy,
		LastChecked: time.Now(),
		Details: map[string]interface{}{
			"url": hc.url,
		},
	}

	req, err := http.NewRequestWithContext(ctx, "GET", hc.url, nil)
	if err != nil {
		result.Status = StatusUnhealthy
		result.Message = fmt.Sprintf("Failed to create request: %v", err)
		result.Duration = time.Since(start)
		return result
	}

	resp, err := hc.client.Do(req)
	if err != nil {
		result.Status = StatusUnhealthy
		result.Message = fmt.Sprintf("Request failed: %v", err)
		result.Duration = time.Since(start)
		return result
	}
	defer resp.Body.Close()

	result.Details["status_code"] = resp.StatusCode

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		result.Message = fmt.Sprintf("Endpoint is healthy (status: %d)", resp.StatusCode)
	} else {
		result.Status = StatusUnhealthy
		result.Message = fmt.Sprintf("Endpoint returned status %d", resp.StatusCode)
	}

	result.Duration = time.Since(start)
	return result
}

// CacheBackendChecker 캐시 백엔드 체커
type CacheBackendChecker struct {
	backendType string
	checkFunc   func(ctx context.Context) error
}

func NewCacheBackendChecker(backendType string, checkFunc func(ctx context.Context) error) *CacheBackendChecker {
	return &CacheBackendChecker{
		backendType: backendType,
		checkFunc:   checkFunc,
	}
}

func (cc *CacheBackendChecker) Name() string {
	return "cache_" + cc.backendType
}

func (cc *CacheBackendChecker) Check(ctx context.Context) *CheckResult {
	start := time.Now()
	result := &CheckResult{
		Name:        cc.Name(),
		Status:      StatusHealthy,
		LastChecked: time.Now(),
		Details: map[string]interface{}{
			"backend": cc.backendType,
		},
	}

	if err := cc.checkFunc(ctx); err != nil {
		result.Status = StatusUnhealthy
		result.Message = fmt.Sprintf("Cache backend check failed: %v", err)
	} else {
		result.Message = "Cache backend is healthy"
	}

	result.Duration = time.Since(start)
	return result
}
