// Package health provides health checking functionality for ProxyND.
// It includes interfaces and implementations for monitoring various system components
// and exposing their health status through HTTP endpoints.
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

// Status represents the health status of a component.
type Status string

const (
	// StatusHealthy indicates the component is functioning normally
	StatusHealthy Status = "healthy"
	// StatusDegraded indicates the component is functioning but with issues
	StatusDegraded Status = "degraded"
	// StatusUnhealthy indicates the component is not functioning properly
	StatusUnhealthy Status = "unhealthy"
)

// CheckResult contains the result of a health check.
type CheckResult struct {
	Name        string                 `json:"name"`
	Status      Status                 `json:"status"`
	Message     string                 `json:"message,omitempty"`
	Duration    time.Duration          `json:"duration_ms"`
	Details     map[string]interface{} `json:"details,omitempty"`
	LastChecked time.Time              `json:"last_checked"`
}

// Checker is an alias for HealthChecker to avoid stuttering
type Checker = HealthChecker

// HealthChecker is the interface that health check implementations must satisfy.
type HealthChecker interface {
	Check(ctx context.Context) *CheckResult
	Name() string
}

// Service is an alias for HealthService to avoid stuttering
type Service = HealthService

// HealthService manages multiple health checkers and provides aggregated health status.
type HealthService struct {
	checkers      []HealthChecker
	checkInterval time.Duration
	results       map[string]*CheckResult
	resultsMutex  sync.RWMutex
	startTime     time.Time
}

// NewHealthService creates a new health service with the specified check interval.
// The service will periodically run all registered health checks.
func NewHealthService(checkInterval time.Duration) *HealthService {
	return &HealthService{
		checkers:      make([]HealthChecker, 0),
		checkInterval: checkInterval,
		results:       make(map[string]*CheckResult),
		startTime:     time.Now(),
	}
}

// RegisterChecker registers a health checker with the service
func (hs *HealthService) RegisterChecker(checker HealthChecker) {
	hs.checkers = append(hs.checkers, checker)
}

// Start begins periodic health checks
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

// runChecks executes all registered health checks
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

// GetStatus returns the overall health status and individual check results
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

// GetUptime returns the service uptime duration
func (hs *HealthService) GetUptime() time.Duration {
	return time.Since(hs.startTime)
}

// 기본 체커 구현들

// EnvironmentChecker checks for required environment variables
type EnvironmentChecker struct {
	requiredVars []string
}

// NewEnvironmentChecker creates a new instance of EnvironmentChecker
func NewEnvironmentChecker(vars []string) *EnvironmentChecker {
	return &EnvironmentChecker{requiredVars: vars}
}

// Name returns the name of the component
func (ec *EnvironmentChecker) Name() string {
	return "environment"
}

// Check performs the environment variable check
func (ec *EnvironmentChecker) Check(_ context.Context) *CheckResult {
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

// DiskSpaceChecker checks for available disk space
type DiskSpaceChecker struct {
	path           string
	minFreeBytes   uint64
	minFreePercent float64
}

// NewDiskSpaceChecker creates a new disk space checker
func NewDiskSpaceChecker(path string, minFreeBytes uint64, minFreePercent float64) *DiskSpaceChecker {
	return &DiskSpaceChecker{
		path:           path,
		minFreeBytes:   minFreeBytes,
		minFreePercent: minFreePercent,
	}
}

// Name returns the name of the disk space checker
func (dc *DiskSpaceChecker) Name() string {
	return "disk_space"
}

// Check performs the disk space check
func (dc *DiskSpaceChecker) Check(_ context.Context) *CheckResult {
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

// WritableChecker checks if a directory is writable
type WritableChecker struct {
	path string
}

// NewWritableChecker creates a new WritableChecker
func NewWritableChecker(path string) *WritableChecker {
	return &WritableChecker{path: path}
}

// Name returns the name of the writable checker
func (wc *WritableChecker) Name() string {
	return "writable_" + filepath.Base(wc.path)
}

// Check performs the writable check
func (wc *WritableChecker) Check(_ context.Context) *CheckResult {
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
			if err := os.MkdirAll(wc.path, 0o755); err != nil {
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
	if err := os.WriteFile(testFile, []byte("test"), 0o644); err != nil {
		result.Status = StatusUnhealthy
		result.Message = fmt.Sprintf("Cannot write to directory: %v", err)
	} else {
		_ = os.Remove(testFile)
		result.Message = "Directory is writable"
	}

	result.Duration = time.Since(start)
	return result
}

// HTTPChecker checks HTTP endpoint health
type HTTPChecker struct {
	name    string
	url     string
	timeout time.Duration
	client  *http.Client
}

// NewHTTPChecker creates a new HTTP health checker
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

// Name returns the name of the HTTP checker
func (hc *HTTPChecker) Name() string {
	return hc.name
}

// Check performs the HTTP health check
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
	defer func() { _ = resp.Body.Close() }()

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

// CacheBackendChecker checks cache backend health
type CacheBackendChecker struct {
	backendType string
	checkFunc   func(ctx context.Context) error
}

// NewCacheBackendChecker creates a new cache backend checker
func NewCacheBackendChecker(backendType string, checkFunc func(ctx context.Context) error) *CacheBackendChecker {
	return &CacheBackendChecker{
		backendType: backendType,
		checkFunc:   checkFunc,
	}
}

// Name returns the name of the cache backend checker
func (cc *CacheBackendChecker) Name() string {
	return "cache_" + cc.backendType
}

// Check performs the cache backend health check
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
