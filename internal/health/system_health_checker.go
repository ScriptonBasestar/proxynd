//go:build linux

// Package health provides system resource health checking functionality
package health

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"sync"
	"syscall"
	"time"
)

// SystemResourceHealthChecker 시스템 리소스 건강성 체커
type SystemResourceHealthChecker struct {
	name               string
	cpuThreshold       float64
	memoryThreshold    float64
	diskThreshold      float64
	goroutineThreshold int
	fdThreshold        int
	checkPaths         []string
	history            *ResourceHistory
	mutex              sync.RWMutex
}

// ResourceHistory 리소스 사용량 이력
type ResourceHistory struct {
	CPUUsage    []float64
	MemoryUsage []float64
	DiskUsage   []float64
	Timestamps  []time.Time
	MaxSize     int
}

// NewSystemResourceHealthChecker 새 시스템 리소스 헬스체커 생성
func NewSystemResourceHealthChecker(name string, checkPaths []string) *SystemResourceHealthChecker {
	return &SystemResourceHealthChecker{
		name:               name,
		cpuThreshold:       80.0,  // 80% CPU 사용률
		memoryThreshold:    85.0,  // 85% 메모리 사용률
		diskThreshold:      90.0,  // 90% 디스크 사용률
		goroutineThreshold: 10000, // 10,000개 고루틴
		fdThreshold:        1000,  // 1,000개 파일 디스크립터
		checkPaths:         checkPaths,
		history: &ResourceHistory{
			CPUUsage:    make([]float64, 0),
			MemoryUsage: make([]float64, 0),
			DiskUsage:   make([]float64, 0),
			Timestamps:  make([]time.Time, 0),
			MaxSize:     100, // 최근 100개 기록 유지
		},
	}
}

// Name 체커 이름 반환
func (srh *SystemResourceHealthChecker) Name() string {
	return "system_" + srh.name
}

// Check 시스템 리소스 전체 건강성 확인
func (srh *SystemResourceHealthChecker) Check(ctx context.Context) *CheckResult {
	start := time.Now()
	result := &CheckResult{
		Name:        srh.Name(),
		Status:      StatusHealthy,
		LastChecked: time.Now(),
		Details:     make(map[string]interface{}),
	}

	// CPU 사용률 확인
	cpuResult := srh.checkCPUUsage()
	result.Details["cpu"] = cpuResult

	// 메모리 사용량 확인
	memoryResult := srh.checkMemoryUsage()
	result.Details["memory"] = memoryResult

	// 디스크 사용량 확인
	diskResult := srh.checkDiskUsage()
	result.Details["disk"] = diskResult

	// 고루틴 수 확인
	goroutineResult := srh.checkGoroutineCount()
	result.Details["goroutines"] = goroutineResult

	// 파일 디스크립터 확인
	fdResult := srh.checkFileDescriptors()
	result.Details["file_descriptors"] = fdResult

	// 시스템 부하 확인 (Linux만)
	loadResult := srh.checkSystemLoad()
	result.Details["load_average"] = loadResult

	// 전체 상태 결정
	allResults := []map[string]interface{}{
		cpuResult, memoryResult, diskResult,
		goroutineResult, fdResult, loadResult,
	}

	var healthyCount, degradedCount, unhealthyCount int
	var criticalIssues []string
	var warnings []string

	for _, res := range allResults {
		status, ok := res["status"].(string)
		if !ok {
			continue
		}

		switch status {
		case string(StatusHealthy):
			healthyCount++
		case string(StatusDegraded):
			degradedCount++
			if warning, ok := res["message"].(string); ok {
				warnings = append(warnings, warning)
			}
		case string(StatusUnhealthy):
			unhealthyCount++
			if issue, ok := res["message"].(string); ok {
				criticalIssues = append(criticalIssues, issue)
			}
		}
	}

	// 이력 업데이트
	srh.updateHistory(cpuResult, memoryResult, diskResult)

	result.Details["summary"] = map[string]interface{}{
		"total_checks":     len(allResults),
		"healthy_checks":   healthyCount,
		"degraded_checks":  degradedCount,
		"unhealthy_checks": unhealthyCount,
		"critical_issues":  criticalIssues,
		"warnings":         warnings,
		"trend_analysis":   srh.analyzeTrends(),
	}

	// 상태 결정
	if unhealthyCount > 0 {
		result.Status = StatusUnhealthy
		result.Message = fmt.Sprintf("시스템 리소스에 심각한 문제가 있습니다 (%d개 실패)", unhealthyCount)
	} else if degradedCount > 0 {
		result.Status = StatusDegraded
		result.Message = fmt.Sprintf("시스템 리소스 사용량이 높습니다 (%d개 경고)", degradedCount)
	} else {
		result.Message = "시스템 리소스가 정상 범위 내에 있습니다"
	}

	result.Duration = time.Since(start)
	return result
}

// checkCPUUsage CPU 사용률 확인
func (srh *SystemResourceHealthChecker) checkCPUUsage() map[string]interface{} {
	result := map[string]interface{}{
		"status":  string(StatusHealthy),
		"details": make(map[string]interface{}),
	}

	// Go 런타임 정보
	numCPU := runtime.NumCPU()
	numGoroutine := runtime.NumGoroutine()

	result["details"].(map[string]interface{})["num_cpu"] = numCPU
	result["details"].(map[string]interface{})["num_goroutines"] = numGoroutine
	result["details"].(map[string]interface{})["gomaxprocs"] = runtime.GOMAXPROCS(0)

	// CPU 사용률 추정 (고루틴 수와 CPU 수의 비율로 간접 측정)
	cpuLoadRatio := float64(numGoroutine) / float64(numCPU)
	result["details"].(map[string]interface{})["goroutine_cpu_ratio"] = cpuLoadRatio

	// 임계값 확인 (고루틴 기반 추정)
	if cpuLoadRatio > 100 { // 고루틴이 CPU 수의 100배 이상
		result["status"] = string(StatusUnhealthy)
		result["message"] = fmt.Sprintf("CPU 부하가 매우 높습니다 (고루틴/CPU 비율: %.1f)", cpuLoadRatio)
	} else if cpuLoadRatio > 50 { // 고루틴이 CPU 수의 50배 이상
		result["status"] = string(StatusDegraded)
		result["message"] = fmt.Sprintf("CPU 부하가 높습니다 (고루틴/CPU 비율: %.1f)", cpuLoadRatio)
	} else {
		result["message"] = fmt.Sprintf("CPU 부하가 정상입니다 (%d CPU, %d 고루틴)", numCPU, numGoroutine)
	}

	// GC 정보 추가
	var gcStats runtime.MemStats
	runtime.ReadMemStats(&gcStats)
	result["details"].(map[string]interface{})["gc_cpu_fraction"] = gcStats.GCCPUFraction * 100

	return result
}

// checkMemoryUsage 메모리 사용량 확인
func (srh *SystemResourceHealthChecker) checkMemoryUsage() map[string]interface{} {
	result := map[string]interface{}{
		"status":  string(StatusHealthy),
		"details": make(map[string]interface{}),
	}

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// 메모리 정보 (바이트를 MB로 변환)
	allocMB := float64(m.Alloc) / 1024 / 1024
	sysMB := float64(m.Sys) / 1024 / 1024
	totalAllocMB := float64(m.TotalAlloc) / 1024 / 1024

	result["details"].(map[string]interface{})["alloc_mb"] = allocMB
	result["details"].(map[string]interface{})["sys_mb"] = sysMB
	result["details"].(map[string]interface{})["total_alloc_mb"] = totalAllocMB
	result["details"].(map[string]interface{})["num_gc"] = m.NumGC
	result["details"].(map[string]interface{})["gc_pause_total_ns"] = m.PauseTotalNs
	result["details"].(map[string]interface{})["heap_objects"] = m.HeapObjects

	// 메모리 사용률 계산 (시스템 할당 기준)
	var usagePercent float64
	if sysMB > 0 {
		usagePercent = (allocMB / sysMB) * 100
	}

	result["details"].(map[string]interface{})["usage_percent"] = usagePercent

	// 임계값 확인
	if usagePercent >= srh.memoryThreshold {
		result["status"] = string(StatusUnhealthy)
		result["message"] = fmt.Sprintf("메모리 사용률이 매우 높습니다 (%.1f%%)", usagePercent)
	} else if usagePercent >= srh.memoryThreshold*0.8 { // 68% 이상 (85%의 80%)
		result["status"] = string(StatusDegraded)
		result["message"] = fmt.Sprintf("메모리 사용률이 높습니다 (%.1f%%)", usagePercent)
	} else {
		result["message"] = fmt.Sprintf("메모리 사용률이 정상입니다 (%.1f%%, %.1fMB/%.1fMB)",
			usagePercent, allocMB, sysMB)
	}

	// GC 빈도 분석
	if m.NumGC > 0 {
		avgPauseNs := m.PauseTotalNs / uint64(m.NumGC)
		result["details"].(map[string]interface{})["avg_gc_pause_ns"] = avgPauseNs

		// GC 일시 정지 시간이 1ms를 초과하면 경고
		if avgPauseNs > 1000000 { // 1ms = 1,000,000 nanoseconds
			result["status"] = string(StatusDegraded)
			result["message"] = fmt.Sprintf("GC 일시 정지 시간이 깁니다 (평균: %dμs)", avgPauseNs/1000)
		}
	}

	return result
}

// checkDiskUsage 디스크 사용량 확인
func (srh *SystemResourceHealthChecker) checkDiskUsage() map[string]interface{} {
	result := map[string]interface{}{
		"status":  string(StatusHealthy),
		"details": make(map[string]interface{}),
		"paths":   make(map[string]interface{}),
	}

	if len(srh.checkPaths) == 0 {
		result["message"] = "디스크 사용량 확인 경로가 설정되지 않았습니다"
		return result
	}

	overallMinFreePercent := 100.0
	var criticalPaths []string
	var warningPaths []string

	// 각 경로별 디스크 사용량 확인
	for _, path := range srh.checkPaths {
		pathStats, err := getDiskUsage(path)
		if err != nil {
			result["paths"].(map[string]interface{})[path] = map[string]interface{}{
				"error":  err.Error(),
				"status": "error",
			}
			continue
		}

		pathResult := map[string]interface{}{
			"total_gb":     float64(pathStats.Total) / (1024 * 1024 * 1024),
			"free_gb":      float64(pathStats.Free) / (1024 * 1024 * 1024),
			"used_gb":      float64(pathStats.Used) / (1024 * 1024 * 1024),
			"free_percent": pathStats.FreePercent,
		}

		// 경로별 상태 판단
		if pathStats.FreePercent < (100.0 - srh.diskThreshold) { // 10% 미만 여유공간
			pathResult["status"] = "critical"
			criticalPaths = append(criticalPaths, path)
		} else if pathStats.FreePercent < (100.0 - srh.diskThreshold*0.8) { // 28% 미만 여유공간
			pathResult["status"] = "warning"
			warningPaths = append(warningPaths, path)
		} else {
			pathResult["status"] = "healthy"
		}

		result["paths"].(map[string]interface{})[path] = pathResult

		// 전체 최소 여유공간 업데이트
		if pathStats.FreePercent < overallMinFreePercent {
			overallMinFreePercent = pathStats.FreePercent
		}
	}

	result["details"].(map[string]interface{})["min_free_percent"] = overallMinFreePercent
	result["details"].(map[string]interface{})["critical_paths"] = criticalPaths
	result["details"].(map[string]interface{})["warning_paths"] = warningPaths

	// 전체 상태 결정
	if len(criticalPaths) > 0 {
		result["status"] = string(StatusUnhealthy)
		result["message"] = fmt.Sprintf("디스크 여유공간이 부족합니다 (%d개 경로 위험)", len(criticalPaths))
	} else if len(warningPaths) > 0 {
		result["status"] = string(StatusDegraded)
		result["message"] = fmt.Sprintf("디스크 여유공간이 부족해지고 있습니다 (%d개 경로 경고)", len(warningPaths))
	} else {
		result["message"] = fmt.Sprintf("디스크 여유공간이 충분합니다 (최소 %.1f%% 여유)", overallMinFreePercent)
	}

	return result
}

// checkGoroutineCount 고루틴 수 확인
func (srh *SystemResourceHealthChecker) checkGoroutineCount() map[string]interface{} {
	result := map[string]interface{}{
		"status":  string(StatusHealthy),
		"details": make(map[string]interface{}),
	}

	numGoroutines := runtime.NumGoroutine()
	result["details"].(map[string]interface{})["count"] = numGoroutines
	result["details"].(map[string]interface{})["threshold"] = srh.goroutineThreshold

	if numGoroutines >= srh.goroutineThreshold {
		result["status"] = string(StatusUnhealthy)
		result["message"] = fmt.Sprintf("고루틴 수가 매우 많습니다 (%d개, 임계값: %d)",
			numGoroutines, srh.goroutineThreshold)
	} else if numGoroutines >= srh.goroutineThreshold/2 {
		result["status"] = string(StatusDegraded)
		result["message"] = fmt.Sprintf("고루틴 수가 많습니다 (%d개)", numGoroutines)
	} else {
		result["message"] = fmt.Sprintf("고루틴 수가 정상입니다 (%d개)", numGoroutines)
	}

	return result
}

// checkFileDescriptors 파일 디스크립터 확인 (Linux 전용)
func (srh *SystemResourceHealthChecker) checkFileDescriptors() map[string]interface{} {
	result := map[string]interface{}{
		"status":  string(StatusHealthy),
		"details": make(map[string]interface{}),
	}

	// Linux에서만 파일 디스크립터 수를 확인할 수 있음
	fdCount, err := getFileDescriptorCount()
	if err != nil {
		result["message"] = fmt.Sprintf("파일 디스크립터 확인 불가: %v", err)
		result["note"] = "Linux 시스템에서만 지원됩니다"
		return result
	}

	result["details"].(map[string]interface{})["count"] = fdCount
	result["details"].(map[string]interface{})["threshold"] = srh.fdThreshold

	if fdCount >= srh.fdThreshold {
		result["status"] = string(StatusUnhealthy)
		result["message"] = fmt.Sprintf("파일 디스크립터 수가 매우 많습니다 (%d개, 임계값: %d)",
			fdCount, srh.fdThreshold)
	} else if fdCount >= srh.fdThreshold/2 {
		result["status"] = string(StatusDegraded)
		result["message"] = fmt.Sprintf("파일 디스크립터 수가 많습니다 (%d개)", fdCount)
	} else {
		result["message"] = fmt.Sprintf("파일 디스크립터 수가 정상입니다 (%d개)", fdCount)
	}

	return result
}

// checkSystemLoad 시스템 부하 확인 (Linux 전용)
func (srh *SystemResourceHealthChecker) checkSystemLoad() map[string]interface{} {
	result := map[string]interface{}{
		"status":  string(StatusHealthy),
		"details": make(map[string]interface{}),
	}

	load1, load5, load15, err := getSystemLoad()
	if err != nil {
		result["message"] = fmt.Sprintf("시스템 부하 확인 불가: %v", err)
		result["note"] = "Linux 시스템에서만 지원됩니다"
		return result
	}

	numCPU := runtime.NumCPU()
	result["details"].(map[string]interface{})["load_1m"] = load1
	result["details"].(map[string]interface{})["load_5m"] = load5
	result["details"].(map[string]interface{})["load_15m"] = load15
	result["details"].(map[string]interface{})["num_cpu"] = numCPU

	// CPU 대비 부하율 계산
	loadRatio1 := load1 / float64(numCPU)
	loadRatio5 := load5 / float64(numCPU)

	result["details"].(map[string]interface{})["load_ratio_1m"] = loadRatio1
	result["details"].(map[string]interface{})["load_ratio_5m"] = loadRatio5

	// 부하 평가 (1분 평균 기준)
	if loadRatio1 > 2.0 { // CPU 대비 200% 이상 부하
		result["status"] = string(StatusUnhealthy)
		result["message"] = fmt.Sprintf("시스템 부하가 매우 높습니다 (1분: %.2f, CPU 대비: %.0f%%)",
			load1, loadRatio1*100)
	} else if loadRatio1 > 1.0 { // CPU 대비 100% 이상 부하
		result["status"] = string(StatusDegraded)
		result["message"] = fmt.Sprintf("시스템 부하가 높습니다 (1분: %.2f, CPU 대비: %.0f%%)",
			load1, loadRatio1*100)
	} else {
		result["message"] = fmt.Sprintf("시스템 부하가 정상입니다 (1분: %.2f, 5분: %.2f, 15분: %.2f)",
			load1, load5, load15)
	}

	return result
}

// updateHistory 리소스 사용량 이력 업데이트
func (srh *SystemResourceHealthChecker) updateHistory(cpu, memory, disk map[string]interface{}) {
	srh.mutex.Lock()
	defer srh.mutex.Unlock()

	now := time.Now()

	// CPU 사용률 (고루틴 비율로 추정)
	if ratio, ok := cpu["details"].(map[string]interface{})["goroutine_cpu_ratio"].(float64); ok {
		srh.history.CPUUsage = append(srh.history.CPUUsage, ratio)
	}

	// 메모리 사용률
	if usage, ok := memory["details"].(map[string]interface{})["usage_percent"].(float64); ok {
		srh.history.MemoryUsage = append(srh.history.MemoryUsage, usage)
	}

	// 디스크 사용률 (최소 여유공간 기준)
	if freePercent, ok := disk["details"].(map[string]interface{})["min_free_percent"].(float64); ok {
		diskUsage := 100.0 - freePercent
		srh.history.DiskUsage = append(srh.history.DiskUsage, diskUsage)
	}

	srh.history.Timestamps = append(srh.history.Timestamps, now)

	// 크기 제한
	if len(srh.history.CPUUsage) > srh.history.MaxSize {
		srh.history.CPUUsage = srh.history.CPUUsage[1:]
		srh.history.MemoryUsage = srh.history.MemoryUsage[1:]
		srh.history.DiskUsage = srh.history.DiskUsage[1:]
		srh.history.Timestamps = srh.history.Timestamps[1:]
	}
}

// analyzeTrends 리소스 사용량 트렌드 분석
func (srh *SystemResourceHealthChecker) analyzeTrends() map[string]interface{} {
	srh.mutex.RLock()
	defer srh.mutex.RUnlock()

	if len(srh.history.CPUUsage) < 5 {
		return map[string]interface{}{
			"status":       "insufficient_data",
			"sample_count": len(srh.history.CPUUsage),
		}
	}

	trends := map[string]interface{}{}

	// 최근 5개와 전체 평균 비교
	recentSize := 5
	if len(srh.history.CPUUsage) < recentSize {
		recentSize = len(srh.history.CPUUsage)
	}

	// CPU 트렌드
	cpuTrend := analyzeTrend(srh.history.CPUUsage, recentSize)
	trends["cpu"] = cpuTrend

	// 메모리 트렌드
	memoryTrend := analyzeTrend(srh.history.MemoryUsage, recentSize)
	trends["memory"] = memoryTrend

	// 디스크 트렌드
	diskTrend := analyzeTrend(srh.history.DiskUsage, recentSize)
	trends["disk"] = diskTrend

	trends["analysis_period"] = fmt.Sprintf("최근 %d회 측정", recentSize)
	trends["total_samples"] = len(srh.history.CPUUsage)

	return trends
}

// analyzeTrend 개별 메트릭 트렌드 분석
func analyzeTrend(data []float64, recentSize int) map[string]interface{} {
	if len(data) == 0 {
		return map[string]interface{}{"status": "no_data"}
	}

	// 전체 평균
	var total float64
	for _, value := range data {
		total += value
	}
	overallAvg := total / float64(len(data))

	// 최근 평균
	recentStart := len(data) - recentSize
	if recentStart < 0 {
		recentStart = 0
	}

	var recentTotal float64
	for i := recentStart; i < len(data); i++ {
		recentTotal += data[i]
	}
	recentAvg := recentTotal / float64(len(data)-recentStart)

	// 변화율 계산
	changePercent := ((recentAvg - overallAvg) / overallAvg) * 100

	trend := map[string]interface{}{
		"overall_avg":    overallAvg,
		"recent_avg":     recentAvg,
		"change_percent": changePercent,
	}

	const (
		trendIncreasing     = "increasing"
		trendDecreasing     = "decreasing"
		trendStable         = "stable"
		severitySignificant = "significant"
		severityModerate    = "moderate"
		severityNone        = "none"
	)

	if changePercent > 20 {
		trend["direction"] = trendIncreasing
		trend["severity"] = severitySignificant
	} else if changePercent > 10 {
		trend["direction"] = trendIncreasing
		trend["severity"] = severityModerate
	} else if changePercent < -20 {
		trend["direction"] = trendDecreasing
		trend["severity"] = severitySignificant
	} else if changePercent < -10 {
		trend["direction"] = trendDecreasing
		trend["severity"] = severityModerate
	} else {
		trend["direction"] = trendStable
		trend["severity"] = severityNone
	}

	return trend
}

// Linux 전용 시스템 함수들

// getFileDescriptorCount 파일 디스크립터 수 반환 (Linux)
func getFileDescriptorCount() (int, error) {
	// /proc/self/fd 디렉토리의 파일 수로 추정
	fdDir, err := os.Open("/proc/self/fd")
	if err != nil {
		return 0, fmt.Errorf("cannot open /proc/self/fd: %w", err)
	}
	//nolint:errcheck // best effort close in test-friendly utility
	defer fdDir.Close()

	entries, err := fdDir.Readdir(-1)
	if err != nil {
		return 0, fmt.Errorf("cannot read /proc/self/fd: %w", err)
	}

	return len(entries), nil
}

// getSystemLoad 시스템 부하 평균 반환 (Linux)
func getSystemLoad() (float64, float64, float64, error) {
	// syscall.Sysinfo()를 사용한 시스템 부하 조회
	var sysinfo syscall.Sysinfo_t
	if err := syscall.Sysinfo(&sysinfo); err != nil {
		return 0, 0, 0, fmt.Errorf("sysinfo failed: %w", err)
	}

	// 부하 평균은 65536로 나누어 실제 값으로 변환
	load1 := float64(sysinfo.Loads[0]) / 65536.0
	load5 := float64(sysinfo.Loads[1]) / 65536.0
	load15 := float64(sysinfo.Loads[2]) / 65536.0

	return load1, load5, load15, nil
}

// GetResourceSummary 리소스 요약 정보 반환
func (srh *SystemResourceHealthChecker) GetResourceSummary() map[string]interface{} {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return map[string]interface{}{
		"cpu_count":         runtime.NumCPU(),
		"goroutine_count":   runtime.NumGoroutine(),
		"memory_alloc_mb":   float64(m.Alloc) / 1024 / 1024,
		"memory_sys_mb":     float64(m.Sys) / 1024 / 1024,
		"gc_count":          m.NumGC,
		"gc_pause_total_ns": m.PauseTotalNs,
		"check_paths":       srh.checkPaths,
		"thresholds": map[string]interface{}{
			"cpu_threshold":       srh.cpuThreshold,
			"memory_threshold":    srh.memoryThreshold,
			"disk_threshold":      srh.diskThreshold,
			"goroutine_threshold": srh.goroutineThreshold,
			"fd_threshold":        srh.fdThreshold,
		},
	}
}

// SetThresholds 임계값 설정
func (srh *SystemResourceHealthChecker) SetThresholds(cpu, memory, disk float64, goroutine, fd int) {
	srh.mutex.Lock()
	defer srh.mutex.Unlock()

	if cpu > 0 {
		srh.cpuThreshold = cpu
	}
	if memory > 0 {
		srh.memoryThreshold = memory
	}
	if disk > 0 {
		srh.diskThreshold = disk
	}
	if goroutine > 0 {
		srh.goroutineThreshold = goroutine
	}
	if fd > 0 {
		srh.fdThreshold = fd
	}
}

// GetHistoryData 이력 데이터 반환
func (srh *SystemResourceHealthChecker) GetHistoryData() *ResourceHistory {
	srh.mutex.RLock()
	defer srh.mutex.RUnlock()

	// 복사본 반환
	return &ResourceHistory{
		CPUUsage:    append([]float64(nil), srh.history.CPUUsage...),
		MemoryUsage: append([]float64(nil), srh.history.MemoryUsage...),
		DiskUsage:   append([]float64(nil), srh.history.DiskUsage...),
		Timestamps:  append([]time.Time(nil), srh.history.Timestamps...),
		MaxSize:     srh.history.MaxSize,
	}
}

// PredictResourceExhaustion 리소스 고갈 예측
func (srh *SystemResourceHealthChecker) PredictResourceExhaustion() map[string]interface{} {
	srh.mutex.RLock()
	defer srh.mutex.RUnlock()

	predictions := map[string]interface{}{}

	if len(srh.history.CPUUsage) < 10 {
		predictions["status"] = "insufficient_data"
		return predictions
	}

	// 간단한 선형 트렌드 기반 예측
	predictions["cpu"] = predictExhaustion(srh.history.CPUUsage, srh.cpuThreshold)
	predictions["memory"] = predictExhaustion(srh.history.MemoryUsage, srh.memoryThreshold)
	predictions["disk"] = predictExhaustion(srh.history.DiskUsage, srh.diskThreshold)

	return predictions
}

// predictExhaustion 개별 리소스 고갈 예측
func predictExhaustion(data []float64, threshold float64) map[string]interface{} {
	if len(data) < 5 {
		return map[string]interface{}{"status": "insufficient_data"}
	}

	// 최근 5개 데이터로 트렌드 계산
	recentStart := len(data) - 5
	var slope float64

	for i := 1; i < 5; i++ {
		slope += data[recentStart+i] - data[recentStart+i-1]
	}
	slope /= 4.0 // 평균 기울기

	current := data[len(data)-1]
	prediction := map[string]interface{}{
		"current_value": current,
		"threshold":     threshold,
		"trend_slope":   slope,
	}

	if slope <= 0 {
		prediction["status"] = "stable_or_decreasing"
		prediction["exhaustion_time"] = "not_predicted"
	} else {
		// 임계값 도달 시간 예측 (단위: 체크 간격)
		checksToThreshold := (threshold - current) / slope
		if checksToThreshold > 0 && checksToThreshold < 1000 {
			prediction["status"] = "increasing"
			prediction["checks_to_threshold"] = int(checksToThreshold)
			prediction["warning"] = checksToThreshold < 10
		} else {
			prediction["status"] = "slow_increase"
			prediction["checks_to_threshold"] = "long_term"
		}
	}

	return prediction
}
