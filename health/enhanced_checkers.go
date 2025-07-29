// Package health provides enhanced health checkers with predictive monitoring
package health

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"runtime"
	"sync"
	"time"
)

// MemoryChecker 메모리 사용률 체커
type MemoryChecker struct {
	thresholdPercent float64
	warningPercent   float64
}

// NewMemoryChecker 새 메모리 체커 생성
func NewMemoryChecker(thresholdPercent, warningPercent float64) *MemoryChecker {
	return &MemoryChecker{
		thresholdPercent: thresholdPercent,
		warningPercent:   warningPercent,
	}
}

// Name 체커 이름 반환
func (mc *MemoryChecker) Name() string {
	return "memory"
}

// Check 메모리 상태 확인
func (mc *MemoryChecker) Check(_ context.Context) *CheckResult {
	start := time.Now()
	result := &CheckResult{
		Name:        mc.Name(),
		Status:      StatusHealthy,
		LastChecked: time.Now(),
		Details:     make(map[string]interface{}),
	}

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// 메모리 사용률 계산
	usedMB := float64(m.Sys) / 1024 / 1024
	allocMB := float64(m.Alloc) / 1024 / 1024
	totalMB := float64(m.Sys) / 1024 / 1024

	var usagePercent float64
	if totalMB > 0 {
		usagePercent = (allocMB / totalMB) * 100
	}

	result.Details["alloc_mb"] = allocMB
	result.Details["sys_mb"] = usedMB
	result.Details["usage_percent"] = usagePercent
	result.Details["num_gc"] = m.NumGC
	result.Details["gc_pause_ns"] = m.PauseTotalNs

	// 상태 결정
	if usagePercent >= mc.thresholdPercent {
		result.Status = StatusUnhealthy
		result.Message = fmt.Sprintf("High memory usage: %.1f%% (threshold: %.1f%%)", usagePercent, mc.thresholdPercent)
	} else if usagePercent >= mc.warningPercent {
		result.Status = StatusDegraded
		result.Message = fmt.Sprintf("Memory usage warning: %.1f%% (warning: %.1f%%)", usagePercent, mc.warningPercent)
	} else {
		result.Message = fmt.Sprintf("Memory usage normal: %.1f%%", usagePercent)
	}

	result.Duration = time.Since(start)
	return result
}

// GoroutineChecker 고루틴 수 체커
type GoroutineChecker struct {
	maxGoroutines     int
	warningGoroutines int
}

// NewGoroutineChecker 새 고루틴 체커 생성
func NewGoroutineChecker(maxGoroutines, warningGoroutines int) *GoroutineChecker {
	return &GoroutineChecker{
		maxGoroutines:     maxGoroutines,
		warningGoroutines: warningGoroutines,
	}
}

// Name 체커 이름 반환
func (gc *GoroutineChecker) Name() string {
	return "goroutines"
}

// Check 고루틴 상태 확인
func (gc *GoroutineChecker) Check(_ context.Context) *CheckResult {
	start := time.Now()
	result := &CheckResult{
		Name:        gc.Name(),
		Status:      StatusHealthy,
		LastChecked: time.Now(),
		Details:     make(map[string]interface{}),
	}

	numGoroutines := runtime.NumGoroutine()
	result.Details["count"] = numGoroutines
	result.Details["max_allowed"] = gc.maxGoroutines

	// 상태 결정
	if numGoroutines >= gc.maxGoroutines {
		result.Status = StatusUnhealthy
		result.Message = fmt.Sprintf("Too many goroutines: %d (max: %d)", numGoroutines, gc.maxGoroutines)
	} else if numGoroutines >= gc.warningGoroutines {
		result.Status = StatusDegraded
		result.Message = fmt.Sprintf("High goroutine count: %d (warning: %d)", numGoroutines, gc.warningGoroutines)
	} else {
		result.Message = fmt.Sprintf("Goroutine count normal: %d", numGoroutines)
	}

	result.Duration = time.Since(start)
	return result
}

// DatabaseConnectionChecker 데이터베이스 연결 체커 인터페이스
type DatabaseConnectionChecker struct {
	name      string
	pingFunc  func(ctx context.Context) error
	statsFunc func() (int, int, time.Duration) // active, idle, maxLifetime
}

// NewDatabaseConnectionChecker 새 DB 연결 체커 생성
func NewDatabaseConnectionChecker(name string, pingFunc func(ctx context.Context) error, statsFunc func() (int, int, time.Duration)) *DatabaseConnectionChecker {
	return &DatabaseConnectionChecker{
		name:      name,
		pingFunc:  pingFunc,
		statsFunc: statsFunc,
	}
}

// Name 체커 이름 반환
func (dcc *DatabaseConnectionChecker) Name() string {
	return "database_" + dcc.name
}

// Check 데이터베이스 연결 상태 확인
func (dcc *DatabaseConnectionChecker) Check(ctx context.Context) *CheckResult {
	start := time.Now()
	result := &CheckResult{
		Name:        dcc.Name(),
		Status:      StatusHealthy,
		LastChecked: time.Now(),
		Details:     make(map[string]interface{}),
	}

	// Ping 테스트
	if err := dcc.pingFunc(ctx); err != nil {
		result.Status = StatusUnhealthy
		result.Message = fmt.Sprintf("Database ping failed: %v", err)
		result.Duration = time.Since(start)
		return result
	}

	// 연결 통계
	if dcc.statsFunc != nil {
		active, idle, maxLifetime := dcc.statsFunc()
		result.Details["active_connections"] = active
		result.Details["idle_connections"] = idle
		result.Details["max_lifetime"] = maxLifetime.Seconds()

		total := active + idle
		result.Details["total_connections"] = total

		// 연결 상태 평가
		if active > 80 || total > 100 {
			result.Status = StatusDegraded
			result.Message = fmt.Sprintf("High database connection usage: %d active, %d total", active, total)
		} else {
			result.Message = fmt.Sprintf("Database connections healthy: %d active, %d idle", active, idle)
		}
	} else {
		result.Message = "Database ping successful"
	}

	result.Duration = time.Since(start)
	return result
}

// ExternalServiceChecker 외부 서비스 체커
type ExternalServiceChecker struct {
	name           string
	url            string
	timeout        time.Duration
	expectedStatus int
	client         *http.Client
	circuitBreaker *CircuitBreaker
}

// NewExternalServiceChecker 새 외부 서비스 체커 생성
func NewExternalServiceChecker(name, url string, timeout time.Duration, expectedStatus int) *ExternalServiceChecker {
	checker := &ExternalServiceChecker{
		name:           name,
		url:            url,
		timeout:        timeout,
		expectedStatus: expectedStatus,
		client: &http.Client{
			Timeout: timeout,
		},
	}

	// 서킷 브레이커 설정
	cbConfig := DefaultCircuitBreakerConfig(fmt.Sprintf("external_%s", name))
	cbConfig.FailureThreshold = 3
	cbConfig.Timeout = 60 * time.Second
	checker.circuitBreaker = NewCircuitBreaker(cbConfig)

	return checker
}

// Name 체커 이름 반환
func (esc *ExternalServiceChecker) Name() string {
	return "external_" + esc.name
}

// Check 외부 서비스 상태 확인
func (esc *ExternalServiceChecker) Check(ctx context.Context) *CheckResult {
	start := time.Now()
	result := &CheckResult{
		Name:        esc.Name(),
		Status:      StatusHealthy,
		LastChecked: time.Now(),
		Details: map[string]interface{}{
			"url":             esc.url,
			"expected_status": esc.expectedStatus,
		},
	}

	// 서킷 브레이커를 통해 요청 수행
	err := esc.circuitBreaker.Execute(ctx, func(ctx context.Context) error {
		req, err := http.NewRequestWithContext(ctx, "GET", esc.url, nil)
		if err != nil {
			return err
		}

		resp, err := esc.client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		result.Details["actual_status"] = resp.StatusCode
		result.Details["response_time_ms"] = time.Since(start).Milliseconds()

		if resp.StatusCode != esc.expectedStatus {
			return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}

		return nil
	})

	if err != nil {
		if IsCircuitBreakerError(err) {
			result.Status = StatusDegraded
			result.Message = fmt.Sprintf("External service circuit breaker is open: %s", esc.name)
		} else {
			result.Status = StatusUnhealthy
			result.Message = fmt.Sprintf("External service check failed: %v", err)
		}
	} else {
		result.Message = fmt.Sprintf("External service %s is healthy", esc.name)
	}

	// 서킷 브레이커 상태 추가
	cbStatus := esc.circuitBreaker.GetStatus()
	result.Details["circuit_breaker"] = cbStatus

	result.Duration = time.Since(start)
	return result
}

// CacheHealthChecker 캐시 시스템 건강성 체커
type CacheHealthChecker struct {
	name       string
	testKey    string
	testValue  string
	setFunc    func(key, value string, ttl time.Duration) error
	getFunc    func(key string) (string, error)
	deleteFunc func(key string) error
	statsFunc  func() map[string]interface{}
}

// NewCacheHealthChecker 새 캐시 헬스 체커 생성
func NewCacheHealthChecker(name string, setFunc func(string, string, time.Duration) error, getFunc func(string) (string, error), deleteFunc func(string) error, statsFunc func() map[string]interface{}) *CacheHealthChecker {
	return &CacheHealthChecker{
		name:       name,
		testKey:    fmt.Sprintf("health_check_%s_%d", name, time.Now().Unix()),
		testValue:  "test_value",
		setFunc:    setFunc,
		getFunc:    getFunc,
		deleteFunc: deleteFunc,
		statsFunc:  statsFunc,
	}
}

// Name 체커 이름 반환
func (chc *CacheHealthChecker) Name() string {
	return "cache_" + chc.name
}

// Check 캐시 상태 확인
func (chc *CacheHealthChecker) Check(ctx context.Context) *CheckResult {
	start := time.Now()
	result := &CheckResult{
		Name:        chc.Name(),
		Status:      StatusHealthy,
		LastChecked: time.Now(),
		Details:     make(map[string]interface{}),
	}

	// Set 테스트
	if err := chc.setFunc(chc.testKey, chc.testValue, 10*time.Second); err != nil {
		result.Status = StatusUnhealthy
		result.Message = fmt.Sprintf("Cache set operation failed: %v", err)
		result.Duration = time.Since(start)
		return result
	}

	// Get 테스트
	value, err := chc.getFunc(chc.testKey)
	if err != nil {
		result.Status = StatusUnhealthy
		result.Message = fmt.Sprintf("Cache get operation failed: %v", err)
		result.Duration = time.Since(start)
		return result
	}

	if value != chc.testValue {
		result.Status = StatusUnhealthy
		result.Message = fmt.Sprintf("Cache value mismatch: expected %s, got %s", chc.testValue, value)
		result.Duration = time.Since(start)
		return result
	}

	// Delete 테스트
	if err := chc.deleteFunc(chc.testKey); err != nil {
		result.Status = StatusDegraded
		result.Message = fmt.Sprintf("Cache delete operation failed: %v", err)
	} else {
		result.Message = fmt.Sprintf("Cache %s is healthy", chc.name)
	}

	// 통계 정보 추가
	if chc.statsFunc != nil {
		stats := chc.statsFunc()
		result.Details["stats"] = stats
	}

	result.Duration = time.Since(start)
	return result
}

// PredictiveHealthChecker 예측적 헬스 체커
type PredictiveHealthChecker struct {
	name            string
	baseChecker     HealthChecker
	history         []float64
	maxHistorySize  int
	degradationRate float64 // 성능 저하율 임계값
	mutex           sync.RWMutex
}

// NewPredictiveHealthChecker 새 예측적 헬스 체커 생성
func NewPredictiveHealthChecker(name string, baseChecker HealthChecker, degradationRate float64) *PredictiveHealthChecker {
	return &PredictiveHealthChecker{
		name:            name,
		baseChecker:     baseChecker,
		history:         make([]float64, 0),
		maxHistorySize:  50,
		degradationRate: degradationRate,
	}
}

// Name 체커 이름 반환
func (phc *PredictiveHealthChecker) Name() string {
	return "predictive_" + phc.name
}

// Check 예측적 헬스 체크 수행
func (phc *PredictiveHealthChecker) Check(ctx context.Context) *CheckResult {
	// 기본 체크 수행
	baseResult := phc.baseChecker.Check(ctx)

	// 기본 결과를 복사하여 예측 정보 추가
	result := &CheckResult{
		Name:        phc.Name(),
		Status:      baseResult.Status,
		Message:     baseResult.Message,
		Duration:    baseResult.Duration,
		Details:     make(map[string]interface{}),
		LastChecked: baseResult.LastChecked,
	}

	// 기본 상세 정보 복사
	for k, v := range baseResult.Details {
		result.Details[k] = v
	}

	// 성능 이력 업데이트
	phc.updateHistory(baseResult.Duration.Seconds())

	// 예측 분석 수행
	prediction := phc.predictTrend()
	result.Details["prediction"] = prediction

	// 예측에 따른 상태 조정
	if prediction["trend"] == "degrading" && baseResult.Status == StatusHealthy {
		result.Status = StatusDegraded
		result.Message = fmt.Sprintf("%s (predictive: performance degrading)", baseResult.Message)
	}

	return result
}

// updateHistory 이력 업데이트
func (phc *PredictiveHealthChecker) updateHistory(duration float64) {
	phc.mutex.Lock()
	defer phc.mutex.Unlock()

	phc.history = append(phc.history, duration)

	// 크기 제한
	if len(phc.history) > phc.maxHistorySize {
		copy(phc.history, phc.history[1:])
		phc.history = phc.history[:phc.maxHistorySize]
	}
}

// predictTrend 트렌드 예측
func (phc *PredictiveHealthChecker) predictTrend() map[string]interface{} {
	phc.mutex.RLock()
	defer phc.mutex.RUnlock()

	if len(phc.history) < 5 {
		return map[string]interface{}{
			"trend":      "insufficient_data",
			"confidence": 0.0,
		}
	}

	// 최근 5개와 이전 5개 비교
	recentLen := min(5, len(phc.history))
	recent := phc.history[len(phc.history)-recentLen:]

	var recentAvg, overallAvg float64
	for _, v := range recent {
		recentAvg += v
	}
	recentAvg /= float64(len(recent))

	for _, v := range phc.history {
		overallAvg += v
	}
	overallAvg /= float64(len(phc.history))

	// 트렌드 계산
	changePercent := ((recentAvg - overallAvg) / overallAvg) * 100
	var trend string
	var confidence float64

	if changePercent > phc.degradationRate {
		trend = "degrading"
		confidence = math.Min(changePercent/phc.degradationRate, 1.0)
	} else if changePercent < -phc.degradationRate {
		trend = "improving"
		confidence = math.Min((-changePercent)/phc.degradationRate, 1.0)
	} else {
		trend = "stable"
		confidence = 1.0 - (abs(changePercent) / phc.degradationRate)
	}

	return map[string]interface{}{
		"trend":          trend,
		"confidence":     confidence,
		"change_percent": changePercent,
		"recent_avg":     recentAvg,
		"overall_avg":    overallAvg,
		"sample_size":    len(phc.history),
	}
}

// CompoundHealthChecker 복합 헬스 체커
type CompoundHealthChecker struct {
	name            string
	checkers        []HealthChecker
	requiredHealthy int // 최소 정상 체커 수
}

// NewCompoundHealthChecker 새 복합 헬스 체커 생성
func NewCompoundHealthChecker(name string, checkers []HealthChecker, requiredHealthy int) *CompoundHealthChecker {
	return &CompoundHealthChecker{
		name:            name,
		checkers:        checkers,
		requiredHealthy: requiredHealthy,
	}
}

// Name 체커 이름 반환
func (chc *CompoundHealthChecker) Name() string {
	return "compound_" + chc.name
}

// Check 복합 헬스 체크 수행
func (chc *CompoundHealthChecker) Check(ctx context.Context) *CheckResult {
	start := time.Now()
	result := &CheckResult{
		Name:        chc.Name(),
		Status:      StatusHealthy,
		LastChecked: time.Now(),
		Details:     make(map[string]interface{}),
	}

	var wg sync.WaitGroup
	results := make(map[string]*CheckResult)
	var resultsMutex sync.Mutex

	// 병렬로 모든 체커 실행
	for _, checker := range chc.checkers {
		wg.Add(1)
		go func(c HealthChecker) {
			defer wg.Done()

			checkResult := c.Check(ctx)

			resultsMutex.Lock()
			results[c.Name()] = checkResult
			resultsMutex.Unlock()
		}(checker)
	}

	wg.Wait()

	// 결과 분석
	var healthyCount, degradedCount, unhealthyCount int
	checkDetails := make(map[string]interface{})

	for name, checkResult := range results {
		checkDetails[name] = map[string]interface{}{
			"status":   string(checkResult.Status),
			"message":  checkResult.Message,
			"duration": checkResult.Duration.Milliseconds(),
		}

		switch checkResult.Status {
		case StatusHealthy:
			healthyCount++
		case StatusDegraded:
			degradedCount++
		case StatusUnhealthy:
			unhealthyCount++
		}
	}

	result.Details["checks"] = checkDetails
	result.Details["healthy_count"] = healthyCount
	result.Details["degraded_count"] = degradedCount
	result.Details["unhealthy_count"] = unhealthyCount
	result.Details["total_count"] = len(chc.checkers)
	result.Details["required_healthy"] = chc.requiredHealthy

	// 전체 상태 결정
	if healthyCount >= chc.requiredHealthy {
		if unhealthyCount > 0 {
			result.Status = StatusDegraded
			result.Message = fmt.Sprintf("Compound check degraded: %d/%d healthy (required: %d), %d unhealthy",
				healthyCount, len(chc.checkers), chc.requiredHealthy, unhealthyCount)
		} else if degradedCount > 0 {
			result.Status = StatusDegraded
			result.Message = fmt.Sprintf("Compound check degraded: %d/%d healthy, %d degraded",
				healthyCount, len(chc.checkers), degradedCount)
		} else {
			result.Message = fmt.Sprintf("Compound check healthy: %d/%d checks passed",
				healthyCount, len(chc.checkers))
		}
	} else {
		result.Status = StatusUnhealthy
		result.Message = fmt.Sprintf("Compound check failed: only %d/%d healthy (required: %d)",
			healthyCount, len(chc.checkers), chc.requiredHealthy)
	}

	result.Duration = time.Since(start)
	return result
}

// 헬퍼 함수들

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
