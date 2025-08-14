// Package health provides enhanced health checking functionality with comprehensive monitoring
package health

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"proxynd/internal/config"
)

// EnhancedHealthService 강화된 헬스 서비스
type EnhancedHealthService struct {
	*HealthService
	config           *config.RootConfig
	cacheManager     minimalCache
	proxyChecker     *ProxyHealthChecker
	cacheChecker     *CacheSystemHealthChecker
	securityChecker  *SecuritySystemHealthChecker
	systemChecker    *SystemResourceHealthChecker
	enhancedCheckers map[string]HealthChecker
	mutex            sync.RWMutex
}

// minimalCache 헬스 체커에서 필요로 하는 최소 캐시 인터페이스
// 실제 구현체(*cache.Manager)와 테스트용 모크 모두 이 인터페이스를 만족하도록 한다.
type minimalCache interface {
	Put(key string, data []byte, ttl time.Duration) error
	Get(key string) ([]byte, bool)
	Delete(key string) error
}

// EnhancedHealthConfig 강화된 헬스 서비스 설정
type EnhancedHealthConfig struct {
	CheckInterval       time.Duration
	EnableProxyCheck    bool
	EnableCacheCheck    bool
	EnableSecurityCheck bool
	EnableSystemCheck   bool
	SystemCheckPaths    []string
	FastResponseMode    bool // 100ms 미만 응답을 위한 모드
}

// DefaultEnhancedHealthConfig 기본 설정 반환
func DefaultEnhancedHealthConfig() *EnhancedHealthConfig {
	return &EnhancedHealthConfig{
		CheckInterval:       30 * time.Second,
		EnableProxyCheck:    true,
		EnableCacheCheck:    true,
		EnableSecurityCheck: true,
		EnableSystemCheck:   true,
		SystemCheckPaths: []string{
			"/tmp",
			os.Getenv("STORAGE_DIR"),
			os.Getenv("CONFIG_DIR"),
		},
		FastResponseMode: true,
	}
}

// NewEnhancedHealthService 새 강화된 헬스 서비스 생성
func NewEnhancedHealthService(
	config *config.RootConfig,
	cacheManager minimalCache,
	healthConfig *EnhancedHealthConfig,
) *EnhancedHealthService {
	if healthConfig == nil {
		healthConfig = DefaultEnhancedHealthConfig()
	}

	// 기본 헬스 서비스 생성
	baseService := NewHealthService(healthConfig.CheckInterval)

	ehs := &EnhancedHealthService{
		HealthService:    baseService,
		config:           config,
		cacheManager:     cacheManager,
		enhancedCheckers: make(map[string]HealthChecker),
	}

	// 강화된 체커들 초기화
	ehs.initializeEnhancedCheckers(healthConfig)

	return ehs
}

// initializeEnhancedCheckers 강화된 체커들 초기화
func (ehs *EnhancedHealthService) initializeEnhancedCheckers(healthConfig *EnhancedHealthConfig) {
	// 프록시 업스트림 체커
	if healthConfig.EnableProxyCheck && ehs.config != nil {
		ehs.proxyChecker = NewProxyHealthChecker(ehs.config)
		ehs.RegisterChecker(ehs.proxyChecker)
		ehs.enhancedCheckers["proxy_upstreams"] = ehs.proxyChecker
	}

	// 캐시 시스템 체커
	if healthConfig.EnableCacheCheck && ehs.cacheManager != nil {
		ehs.cacheChecker = NewCacheSystemHealthChecker(ehs.config, ehs.cacheManager)
		ehs.RegisterChecker(ehs.cacheChecker)
		ehs.enhancedCheckers["cache_system"] = ehs.cacheChecker
	}

	// 보안 시스템 체커
	if healthConfig.EnableSecurityCheck && ehs.config != nil {
		ehs.securityChecker = NewSecuritySystemHealthChecker(ehs.config)
		ehs.RegisterChecker(ehs.securityChecker)
		ehs.enhancedCheckers["security_system"] = ehs.securityChecker
	}

	// 시스템 리소스 체커
	if healthConfig.EnableSystemCheck {
		// nil 값 필터링
		var validPaths []string
		for _, path := range healthConfig.SystemCheckPaths {
			if path != "" {
				validPaths = append(validPaths, path)
			}
		}
		if len(validPaths) == 0 {
			validPaths = []string{"/tmp"}
		}

		ehs.systemChecker = NewSystemResourceHealthChecker("resources", validPaths)
		ehs.RegisterChecker(ehs.systemChecker)
		ehs.enhancedCheckers["system_resources"] = ehs.systemChecker
	}

	// 기본 체커들도 등록
	ehs.registerBasicCheckers(healthConfig)
}

// registerBasicCheckers 기본 체커들 등록
func (ehs *EnhancedHealthService) registerBasicCheckers(healthConfig *EnhancedHealthConfig) {
	// 환경 변수 체커
	ehs.RegisterChecker(NewEnvironmentChecker([]string{
		"CONFIG_DIR",
		"STORAGE_DIR",
		"SERVER_PORT",
	}))

	// 디스크 공간 체커
	if storageDir := os.Getenv("STORAGE_DIR"); storageDir != "" {
		ehs.RegisterChecker(NewDiskSpaceChecker(storageDir, 1024*1024*1024, 10.0))
		ehs.RegisterChecker(NewWritableChecker(storageDir))
	}

	// 캐시 디렉토리 체커
	if ehs.config != nil && ehs.config.Cache.File.Directory != "" {
		ehs.RegisterChecker(NewWritableChecker(ehs.config.Cache.File.Directory))
	}

	// 메모리 체커
	ehs.RegisterChecker(NewMemoryChecker(85.0, 70.0))

	// 고루틴 체커
	ehs.RegisterChecker(NewGoroutineChecker(10000, 5000))
}

// GetComprehensiveStatus 포괄적인 건강 상태 반환
func (ehs *EnhancedHealthService) GetComprehensiveStatus() (*ComprehensiveHealthStatus, error) {
	start := time.Now()

	// 기본 상태 가져오기
	overallStatus, checks := ehs.GetStatus()

	status := &ComprehensiveHealthStatus{
		OverallStatus:  overallStatus,
		Timestamp:      time.Now(),
		Uptime:         ehs.GetUptime(),
		ResponseTime:   time.Since(start),
		BasicChecks:    checks,
		EnhancedChecks: make(map[string]*CheckResult),
		Summary:        make(map[string]interface{}),
		Metrics:        make(map[string]interface{}),
	}

	// 강화된 체커들의 결과 수집
	ehs.mutex.RLock()
	defer ehs.mutex.RUnlock()

	for name, checker := range ehs.enhancedCheckers {
		if result, exists := checks[checker.Name()]; exists {
			status.EnhancedChecks[name] = result
		}
	}

	// 요약 정보 생성
	status.Summary = ehs.generateSummary(checks)

	// 메트릭 정보 생성
	status.Metrics = ehs.generateMetrics()

	// 환경 정보 추가
	if env := os.Getenv("ENVIRONMENT"); env != "" {
		status.Environment = env
	}
	if version := os.Getenv("VERSION"); version != "" {
		status.Version = version
	}

	return status, nil
}

// ComprehensiveHealthStatus 포괄적인 건강 상태
type ComprehensiveHealthStatus struct {
	OverallStatus  Status                  `json:"status"`
	Timestamp      time.Time               `json:"timestamp"`
	Uptime         time.Duration           `json:"uptime"`
	ResponseTime   time.Duration           `json:"response_time_ms"`
	Environment    string                  `json:"environment,omitempty"`
	Version        string                  `json:"version,omitempty"`
	BasicChecks    map[string]*CheckResult `json:"basic_checks"`
	EnhancedChecks map[string]*CheckResult `json:"enhanced_checks"`
	Summary        map[string]interface{}  `json:"summary"`
	Metrics        map[string]interface{}  `json:"metrics"`
}

// generateSummary 요약 정보 생성
func (ehs *EnhancedHealthService) generateSummary(checks map[string]*CheckResult) map[string]interface{} {
	var healthyCount, degradedCount, unhealthyCount int
	var criticalIssues []string
	var warnings []string

	for _, result := range checks {
		switch result.Status {
		case StatusHealthy:
			healthyCount++
		case StatusDegraded:
			degradedCount++
			warnings = append(warnings, fmt.Sprintf("%s: %s", result.Name, result.Message))
		case StatusUnhealthy:
			unhealthyCount++
			criticalIssues = append(criticalIssues, fmt.Sprintf("%s: %s", result.Name, result.Message))
		}
	}

	summary := map[string]interface{}{
		"total_checks":      len(checks),
		"healthy_checks":    healthyCount,
		"degraded_checks":   degradedCount,
		"unhealthy_checks":  unhealthyCount,
		"health_percentage": float64(healthyCount) / float64(len(checks)) * 100,
	}

	if len(criticalIssues) > 0 {
		summary["critical_issues"] = criticalIssues
	}
	if len(warnings) > 0 {
		summary["warnings"] = warnings
	}

	// 카테고리별 상태 분석
	summary["categories"] = ehs.analyzeCategorizedHealth(checks)

	return summary
}

// analyzeCategorizedHealth 카테고리별 건강 상태 분석
func (ehs *EnhancedHealthService) analyzeCategorizedHealth(checks map[string]*CheckResult) map[string]interface{} {
	categories := map[string][]string{
		"infrastructure": {"environment", "disk_space", "writable_"},
		"performance":    {"memory", "goroutines", "system_"},
		"security":       {"security_"},
		"connectivity":   {"proxy_", "_registry", "cache_"},
	}

	categoryStatus := make(map[string]interface{})

	for category, patterns := range categories {
		var healthy, degraded, unhealthy int
		var matchedChecks []string

		for checkName, result := range checks {
			matched := false
			for _, pattern := range patterns {
				if containsPattern(checkName, pattern) {
					matched = true
					break
				}
			}

			if matched {
				matchedChecks = append(matchedChecks, checkName)
				switch result.Status {
				case StatusHealthy:
					healthy++
				case StatusDegraded:
					degraded++
				case StatusUnhealthy:
					unhealthy++
				}
			}
		}

		if len(matchedChecks) > 0 {
			status := StatusHealthy
			if unhealthy > 0 {
				status = StatusUnhealthy
			} else if degraded > 0 {
				status = StatusDegraded
			}

			categoryStatus[category] = map[string]interface{}{
				"status":          string(status),
				"healthy_count":   healthy,
				"degraded_count":  degraded,
				"unhealthy_count": unhealthy,
				"checks":          matchedChecks,
			}
		}
	}

	return categoryStatus
}

// containsPattern 패턴 매칭 헬퍼
func containsPattern(text, pattern string) bool {
	if pattern == "" {
		return false
	}
	if pattern[0] == '_' {
		// 끝에 패턴이 있는지 확인
		return len(text) >= len(pattern)-1 && text[len(text)-(len(pattern)-1):] == pattern[1:]
	}
	if pattern[len(pattern)-1] == '_' {
		// 시작에 패턴이 있는지 확인
		return len(text) >= len(pattern)-1 && text[:len(pattern)-1] == pattern[:len(pattern)-1]
	}
	// 정확한 매칭
	return text == pattern
}

// generateMetrics 메트릭 정보 생성
func (ehs *EnhancedHealthService) generateMetrics() map[string]interface{} {
	metrics := map[string]interface{}{
		"service_uptime_seconds": ehs.GetUptime().Seconds(),
		"check_count":            len(ehs.checkers),
	}

	// 프록시 메트릭
	if ehs.proxyChecker != nil {
		if cbStatus := ehs.proxyChecker.GetCircuitBreakerStatus(); len(cbStatus) > 0 {
			metrics["circuit_breakers"] = cbStatus
		}
	}

	// 캐시 메트릭
	if ehs.cacheChecker != nil {
		metrics["cache_statistics"] = ehs.cacheChecker.GetCacheStatistics()
	}

	// 보안 메트릭
	if ehs.securityChecker != nil {
		metrics["security_metrics"] = ehs.securityChecker.GetSecurityMetrics()
	}

	// 시스템 리소스 메트릭
	if ehs.systemChecker != nil {
		metrics["resource_summary"] = ehs.systemChecker.GetResourceSummary()
		metrics["resource_trends"] = ehs.systemChecker.GetHistoryData()
		metrics["resource_predictions"] = ehs.systemChecker.PredictResourceExhaustion()
	}

	return metrics
}

// GetFastHealthStatus 빠른 건강 상태 반환 (100ms 미만 목표)
func (ehs *EnhancedHealthService) GetFastHealthStatus() (*FastHealthStatus, error) {
	start := time.Now()

	// 캐시된 결과 사용 또는 필수 체크만 수행
	status, checks := ehs.GetStatus()

	// 핵심 지표만 확인
	coreChecks := []string{"environment", "memory", "goroutines"}
	coreStatus := make(map[string]*CheckResult)

	for _, checkName := range coreChecks {
		if result, exists := checks[checkName]; exists {
			coreStatus[checkName] = result
		}
	}

	fastStatus := &FastHealthStatus{
		Status:       status,
		Timestamp:    time.Now(),
		ResponseTime: time.Since(start),
		CoreChecks:   coreStatus,
		CheckCount:   len(checks),
	}

	// 빠른 요약
	var healthy, total int
	for _, result := range coreStatus {
		total++
		if result.Status == StatusHealthy {
			healthy++
		}
	}

	if total > 0 {
		fastStatus.HealthRatio = float64(healthy) / float64(total) * 100
	} else {
		fastStatus.HealthRatio = 0.0
	}

	return fastStatus, nil
}

// FastHealthStatus 빠른 건강 상태
type FastHealthStatus struct {
	Status       Status                  `json:"status"`
	Timestamp    time.Time               `json:"timestamp"`
	ResponseTime time.Duration           `json:"response_time_ms"`
	CoreChecks   map[string]*CheckResult `json:"core_checks"`
	CheckCount   int                     `json:"total_check_count"`
	HealthRatio  float64                 `json:"health_ratio_percent"`
}

// GetProxyUpstreamStatus 특정 프록시의 업스트림 상태 반환
func (ehs *EnhancedHealthService) GetProxyUpstreamStatus(proxyType string) (*ProxyUpstreamResult, error) {
	if ehs.proxyChecker == nil {
		return nil, fmt.Errorf("proxy health checker not initialized")
	}
	return ehs.proxyChecker.GetUpstreamStatus(proxyType)
}

// RunHealthCheckFor 특정 체커의 헬스체크 실행
func (ehs *EnhancedHealthService) RunHealthCheckFor(checkerName string) (*CheckResult, error) {
	ehs.mutex.RLock()
	defer ehs.mutex.RUnlock()

	checker, exists := ehs.enhancedCheckers[checkerName]
	if !exists {
		return nil, fmt.Errorf("checker not found: %s", checkerName)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return checker.Check(ctx), nil
}

// GetAvailableCheckers 사용 가능한 체커 목록 반환
func (ehs *EnhancedHealthService) GetAvailableCheckers() map[string]interface{} {
	ehs.mutex.RLock()
	defer ehs.mutex.RUnlock()

	checkers := make(map[string]interface{})

	for name, checker := range ehs.enhancedCheckers {
		checkers[name] = map[string]interface{}{
			"name":        checker.Name(),
			"type":        fmt.Sprintf("%T", checker),
			"description": ehs.getCheckerDescription(name),
		}
	}

	return checkers
}

// getCheckerDescription 체커 설명 반환
func (ehs *EnhancedHealthService) getCheckerDescription(name string) string {
	descriptions := map[string]string{
		"proxy_upstreams":  "프록시별 업스트림 레지스트리 연결성 확인",
		"cache_system":     "캐시 백엔드 및 성능 확인",
		"security_system":  "TLS, 인증, 접근제어 등 보안 설정 확인",
		"system_resources": "CPU, 메모리, 디스크 등 시스템 리소스 확인",
	}

	if desc, exists := descriptions[name]; exists {
		return desc
	}
	return "기타 건강 상태 확인"
}

// EnableMaintMode 유지보수 모드 활성화
func (ehs *EnhancedHealthService) EnableMaintMode(reason string) {
	// 유지보수 모드 체커 추가
	maintChecker := &MaintenanceChecker{
		enabled: true,
		reason:  reason,
		since:   time.Now(),
	}

	ehs.RegisterChecker(maintChecker)
}

// MaintenanceChecker 유지보수 모드 체커
type MaintenanceChecker struct {
	enabled bool
	reason  string
	since   time.Time
}

// Name 체커 이름 반환
func (mc *MaintenanceChecker) Name() string {
	return "maintenance_mode"
}

// Check 유지보수 모드 체크
func (mc *MaintenanceChecker) Check(_ context.Context) *CheckResult {
	if !mc.enabled {
		return &CheckResult{
			Name:        mc.Name(),
			Status:      StatusHealthy,
			Message:     "서비스가 정상 운영 중입니다",
			LastChecked: time.Now(),
		}
	}

	return &CheckResult{
		Name:        mc.Name(),
		Status:      StatusDegraded,
		Message:     fmt.Sprintf("유지보수 모드: %s", mc.reason),
		LastChecked: time.Now(),
		Details: map[string]interface{}{
			"maintenance_since": mc.since,
			"duration":          time.Since(mc.since).String(),
			"reason":            mc.reason,
		},
	}
}
