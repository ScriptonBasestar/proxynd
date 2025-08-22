// Package health provides cache system health checking functionality
package health

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"proxynd/internal/config"
)

const (
	backendFile       = "file"
	backendFilesystem = "filesystem"
	backendS3         = "s3"
	backendRedis      = "redis"
)

// CacheSystemHealthChecker 캐시 시스템 전반적인 건강성 체커
type CacheSystemHealthChecker struct {
	config        *config.RootConfig
	cacheManager  minimalCache
	testKeyPrefix string
}

// NewCacheSystemHealthChecker 새 캐시 시스템 헬스체커 생성
func NewCacheSystemHealthChecker(config *config.RootConfig, cacheManager minimalCache) *CacheSystemHealthChecker {
	return &CacheSystemHealthChecker{
		config:        config,
		cacheManager:  cacheManager,
		testKeyPrefix: fmt.Sprintf("health_check_%d", time.Now().Unix()),
	}
}

// Name 체커 이름 반환
func (cshc *CacheSystemHealthChecker) Name() string {
	return "cache_system"
}

// Check 캐시 시스템 전체 건강성 확인
func (cshc *CacheSystemHealthChecker) Check(_ context.Context) *CheckResult {
	start := time.Now()
	result := &CheckResult{
		Name:        cshc.Name(),
		Status:      StatusHealthy,
		LastChecked: time.Now(),
		Details:     make(map[string]interface{}),
	}

	// 캐시 백엔드별 체크
	backendResult := cshc.checkCacheBackend()
	result.Details["backend"] = backendResult

	// 캐시 성능 체크
	perfResult := cshc.checkCachePerformance()
	result.Details["performance"] = perfResult

	// 캐시 용량 체크
	capacityResult := cshc.checkCacheCapacity()
	result.Details["capacity"] = capacityResult

	// 캐시 일관성 체크
	consistencyResult := cshc.checkCacheConsistency()
	result.Details["consistency"] = consistencyResult

	// 전체 상태 결정
	allResults := []map[string]interface{}{
		backendResult, perfResult, capacityResult, consistencyResult,
	}

	var healthyCount, degradedCount, unhealthyCount int
	for _, res := range allResults {
		switch res["status"].(string) {
		case string(StatusHealthy):
			healthyCount++
		case string(StatusDegraded):
			degradedCount++
		case string(StatusUnhealthy):
			unhealthyCount++
		}
	}

	result.Details["summary"] = map[string]interface{}{
		"total_checks":     len(allResults),
		"healthy_checks":   healthyCount,
		"degraded_checks":  degradedCount,
		"unhealthy_checks": unhealthyCount,
	}

	// 상태 결정
	if unhealthyCount > 0 {
		result.Status = StatusUnhealthy
		result.Message = fmt.Sprintf("캐시 시스템에 심각한 문제가 있습니다 (%d개 실패)", unhealthyCount)
	} else if degradedCount > 0 {
		result.Status = StatusDegraded
		result.Message = fmt.Sprintf("캐시 시스템 성능이 저하되었습니다 (%d개 경고)", degradedCount)
	} else {
		result.Message = "캐시 시스템이 정상적으로 작동하고 있습니다"
	}

	result.Duration = time.Since(start)
	return result
}

// checkCacheBackend 캐시 백엔드 연결성 및 기능 확인
func (cshc *CacheSystemHealthChecker) checkCacheBackend() map[string]interface{} {
	start := time.Now()
	result := map[string]interface{}{
		"status":  string(StatusHealthy),
		"backend": cshc.config.Cache.Backend,
		"details": make(map[string]interface{}),
	}

	testKey := fmt.Sprintf("%s_backend_test", cshc.testKeyPrefix)
	testValue := fmt.Sprintf("test_value_%d", time.Now().UnixNano())

	// Set 작업 테스트
	setStart := time.Now()
	err := cshc.cacheManager.Put(testKey, []byte(testValue), 30*time.Second)
	setDuration := time.Since(setStart)

	if err != nil {
		result["status"] = string(StatusUnhealthy)
		result["message"] = fmt.Sprintf("캐시 저장 실패: %v", err)
		result["error"] = err.Error()
		return result
	}

	result["details"].(map[string]interface{})["set_duration_ms"] = setDuration.Milliseconds()

	// Get 작업 테스트
	getStart := time.Now()
	retrievedValue, found := cshc.cacheManager.Get(testKey)
	getDuration := time.Since(getStart)

	if !found {
		result["status"] = string(StatusUnhealthy)
		result["message"] = "캐시 조회 실패: 데이터를 찾을 수 없음"
		result["error"] = "cache miss after set"
		return result
	}

	result["details"].(map[string]interface{})["get_duration_ms"] = getDuration.Milliseconds()

	// 데이터 무결성 확인
	if string(retrievedValue) != testValue {
		result["status"] = string(StatusUnhealthy)
		result["message"] = "캐시 데이터 무결성 오류"
		result["error"] = fmt.Sprintf("expected: %s, got: %s", testValue, string(retrievedValue))
		return result
	}

	// Delete 작업 테스트
	deleteStart := time.Now()
	err = cshc.cacheManager.Delete(testKey)
	deleteDuration := time.Since(deleteStart)

	result["details"].(map[string]interface{})["delete_duration_ms"] = deleteDuration.Milliseconds()

	if err != nil {
		result["status"] = string(StatusDegraded)
		result["message"] = fmt.Sprintf("캐시 삭제 실패: %v", err)
		result["warning"] = err.Error()
	} else {
		result["message"] = "캐시 백엔드가 정상적으로 작동합니다"
	}

	result["duration_ms"] = time.Since(start).Milliseconds()
	return result
}

// checkCachePerformance 캐시 성능 확인
func (cshc *CacheSystemHealthChecker) checkCachePerformance() map[string]interface{} {
	start := time.Now()
	result := map[string]interface{}{
		"status":  string(StatusHealthy),
		"details": make(map[string]interface{}),
	}

	// 여러 개의 키-값 쌍으로 성능 테스트
	numTests := 10
	testKeys := make([]string, numTests)
	testData := make([]byte, 1024) // 1KB 테스트 데이터
	for i := range testData {
		testData[i] = byte(i % 256)
	}

	// 동시 Set 작업 성능 테스트
	setStart := time.Now()
	for i := 0; i < numTests; i++ {
		testKeys[i] = fmt.Sprintf("%s_perf_%d", cshc.testKeyPrefix, i)
		if err := cshc.cacheManager.Put(testKeys[i], testData, 30*time.Second); err != nil {
			result["status"] = string(StatusDegraded)
			result["message"] = fmt.Sprintf("성능 테스트 중 저장 실패: %v", err)
			return result
		}
	}
	avgSetTime := time.Since(setStart) / time.Duration(numTests)

	// 동시 Get 작업 성능 테스트
	getStart := time.Now()
	for _, key := range testKeys {
		if _, found := cshc.cacheManager.Get(key); !found {
			result["status"] = string(StatusDegraded)
			result["message"] = "성능 테스트 중 조회 실패: 데이터를 찾을 수 없음"
			return result
		}
	}
	avgGetTime := time.Since(getStart) / time.Duration(numTests)

	// 정리
	for _, key := range testKeys {
		_ = cshc.cacheManager.Delete(key)
	}

	// 성능 임계값 확인
	result["details"].(map[string]interface{})["avg_set_time_ms"] = avgSetTime.Milliseconds()
	result["details"].(map[string]interface{})["avg_get_time_ms"] = avgGetTime.Milliseconds()
	result["details"].(map[string]interface{})["test_data_size"] = len(testData)
	result["details"].(map[string]interface{})["num_operations"] = numTests

	// 성능 기준 (조정 가능한 임계값)
	slowSetThreshold := 50 * time.Millisecond
	slowGetThreshold := 10 * time.Millisecond

	if avgSetTime > slowSetThreshold || avgGetTime > slowGetThreshold {
		result["status"] = string(StatusDegraded)
		result["message"] = fmt.Sprintf("캐시 성능이 저하되었습니다 (Set: %dms, Get: %dms)",
			avgSetTime.Milliseconds(), avgGetTime.Milliseconds())
	} else {
		result["message"] = "캐시 성능이 정상입니다"
	}

	result["duration_ms"] = time.Since(start).Milliseconds()
	return result
}

// checkCacheCapacity 캐시 용량 및 저장공간 확인
func (cshc *CacheSystemHealthChecker) checkCacheCapacity() map[string]interface{} {
	result := map[string]interface{}{
		"status":  string(StatusHealthy),
		"details": make(map[string]interface{}),
	}

	switch strings.ToLower(cshc.config.Cache.Backend) {
	case backendFile, backendFilesystem:
		return cshc.checkFileSystemCapacity(result)
	case backendS3:
		return cshc.checkS3Capacity(result)
	case backendRedis:
		return cshc.checkRedisCapacity(result)
	default:
		result["status"] = string(StatusUnhealthy)
		result["message"] = fmt.Sprintf("지원하지 않는 캐시 백엔드: %s", cshc.config.Cache.Backend)
		return result
	}
}

// checkFileSystemCapacity 파일시스템 캐시 용량 확인
func (cshc *CacheSystemHealthChecker) checkFileSystemCapacity(result map[string]interface{}) map[string]interface{} {
	cacheDir := cshc.config.Cache.File.Directory
	if cacheDir == "" {
		result["status"] = string(StatusUnhealthy)
		result["message"] = "캐시 디렉토리가 설정되지 않았습니다"
		return result
	}

	// 디스크 사용량 확인
	stats, err := getDiskUsage(cacheDir)
	if err != nil {
		result["status"] = string(StatusUnhealthy)
		result["message"] = fmt.Sprintf("디스크 사용량 확인 실패: %v", err)
		result["error"] = err.Error()
		return result
	}

	result["details"].(map[string]interface{})["cache_directory"] = cacheDir
	result["details"].(map[string]interface{})["disk_total_gb"] = float64(stats.Total) / (1024 * 1024 * 1024)
	result["details"].(map[string]interface{})["disk_free_gb"] = float64(stats.Free) / (1024 * 1024 * 1024)
	result["details"].(map[string]interface{})["disk_used_gb"] = float64(stats.Used) / (1024 * 1024 * 1024)
	result["details"].(map[string]interface{})["disk_free_percent"] = stats.FreePercent

	// 캐시 디렉토리 크기 확인
	cacheSize, fileCount, err := cshc.calculateDirectorySize(cacheDir)
	if err != nil {
		result["warning"] = fmt.Sprintf("캐시 디렉토리 크기 계산 실패: %v", err)
	} else {
		result["details"].(map[string]interface{})["cache_size_gb"] = float64(cacheSize) / (1024 * 1024 * 1024)
		result["details"].(map[string]interface{})["cache_file_count"] = fileCount
	}

	// 용량 경고 확인
	if stats.FreePercent < 10.0 {
		result["status"] = string(StatusUnhealthy)
		result["message"] = fmt.Sprintf("디스크 여유공간 부족: %.1f%%", stats.FreePercent)
	} else if stats.FreePercent < 20.0 {
		result["status"] = string(StatusDegraded)
		result["message"] = fmt.Sprintf("디스크 여유공간 경고: %.1f%%", stats.FreePercent)
	} else {
		result["message"] = fmt.Sprintf("캐시 저장공간이 충분합니다 (%.1f%% 여유)", stats.FreePercent)
	}

	return result
}

// checkS3Capacity S3 캐시 용량 확인
func (cshc *CacheSystemHealthChecker) checkS3Capacity(result map[string]interface{}) map[string]interface{} {
	result["details"].(map[string]interface{})["backend"] = "s3"
	result["details"].(map[string]interface{})["bucket"] = cshc.config.Cache.S3.Bucket
	result["details"].(map[string]interface{})["region"] = cshc.config.Cache.S3.Region

	// S3는 사실상 무제한 저장공간이므로 연결성만 확인
	result["message"] = "S3 캐시 백엔드 (무제한 저장공간)"
	result["details"].(map[string]interface{})["capacity_limit"] = "unlimited"

	return result
}

// checkRedisCapacity Redis 캐시 용량 확인
func (cshc *CacheSystemHealthChecker) checkRedisCapacity(result map[string]interface{}) map[string]interface{} {
	result["details"].(map[string]interface{})["backend"] = "redis"
	result["details"].(map[string]interface{})["address"] = cshc.config.Cache.Redis.Address

	// Redis 메모리 사용량은 실제 Redis 연결이 필요하므로 기본 정보만 제공
	result["message"] = "Redis 캐시 백엔드 (메모리 기반)"
	result["details"].(map[string]interface{})["note"] = "메모리 사용량은 Redis INFO 명령어 필요"

	return result
}

// checkCacheConsistency 캐시 일관성 확인
func (cshc *CacheSystemHealthChecker) checkCacheConsistency() map[string]interface{} {
	start := time.Now()
	result := map[string]interface{}{
		"status":  string(StatusHealthy),
		"details": make(map[string]interface{}),
	}

	// TTL 동작 확인
	testKey := fmt.Sprintf("%s_ttl_test", cshc.testKeyPrefix)
	testValue := "ttl_test_value"
	shortTTL := 2 * time.Second

	// 짧은 TTL로 데이터 저장
	err := cshc.cacheManager.Put(testKey, []byte(testValue), shortTTL)
	if err != nil {
		result["status"] = string(StatusUnhealthy)
		result["message"] = fmt.Sprintf("TTL 테스트 저장 실패: %v", err)
		return result
	}

	// 즉시 조회 (성공해야 함)
	_, found := cshc.cacheManager.Get(testKey)
	if !found {
		result["status"] = string(StatusDegraded)
		result["message"] = "TTL 테스트 중 즉시 조회 실패"
		result["warning"] = "cache miss after set"
	}

	result["details"].(map[string]interface{})["ttl_test_key"] = testKey
	result["details"].(map[string]interface{})["ttl_seconds"] = shortTTL.Seconds()

	// 캐시 키 패턴 일관성 확인 (간단한 샘플링)
	consistency := cshc.checkKeyPatternConsistency()
	result["details"].(map[string]interface{})["key_patterns"] = consistency

	result["message"] = "캐시 일관성 검사 완료"
	result["duration_ms"] = time.Since(start).Milliseconds()

	return result
}

// checkKeyPatternConsistency 캐시 키 패턴 일관성 확인
func (cshc *CacheSystemHealthChecker) checkKeyPatternConsistency() map[string]interface{} {
	consistency := map[string]interface{}{
		"patterns_checked": 0,
		"note":             "키 패턴 일관성은 백엔드 스캔 기능이 필요합니다",
	}

	// 파일시스템 백엔드인 경우 디렉토리 구조 확인
	if strings.ToLower(cshc.config.Cache.Backend) == backendFile {
		if cshc.config.Cache.File.Directory != "" {
			patterns := cshc.analyzeCacheDirectoryPatterns(cshc.config.Cache.File.Directory)
			consistency["file_patterns"] = patterns
		}
	}

	return consistency
}

// analyzeCacheDirectoryPatterns 캐시 디렉토리 패턴 분석
func (cshc *CacheSystemHealthChecker) analyzeCacheDirectoryPatterns(cacheDir string) map[string]interface{} {
	patterns := map[string]interface{}{
		"proxy_types": make(map[string]int),
		"total_dirs":  0,
	}

	err := filepath.Walk(cacheDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // 에러 무시하고 계속
		}

		if info.IsDir() {
			patterns["total_dirs"] = patterns["total_dirs"].(int) + 1

			// 프록시 타입별 디렉토리 카운트
			relPath, _ := filepath.Rel(cacheDir, path)
			if relPath != "." && strings.Count(relPath, string(filepath.Separator)) == 0 {
				// 최상위 서브디렉토리만 카운트
				proxyTypes := patterns["proxy_types"].(map[string]int)
				proxyTypes[relPath]++
				patterns["proxy_types"] = proxyTypes
			}
		}
		return nil
	})
	if err != nil {
		patterns["error"] = err.Error()
	}

	return patterns
}

// calculateDirectorySize 디렉토리 크기 및 파일 수 계산
func (cshc *CacheSystemHealthChecker) calculateDirectorySize(dirPath string) (int64, int, error) {
	var totalSize int64
	var fileCount int

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			totalSize += info.Size()
			fileCount++
		}
		return nil
	})

	return totalSize, fileCount, err
}

// GetCacheStatistics 캐시 통계 정보 반환
func (cshc *CacheSystemHealthChecker) GetCacheStatistics() map[string]interface{} {
	stats := map[string]interface{}{
		"backend":          cshc.config.Cache.Backend,
		"ttl_seconds":      cshc.config.Cache.TTL.Seconds(),
		"max_size":         cshc.config.Cache.MaxSize,
		"cleanup_interval": cshc.config.Cache.CleanupInterval.Seconds(),
		"eviction_policy":  cshc.config.Cache.EvictionPolicy,
	}

	// 백엔드별 추가 정보
	switch strings.ToLower(cshc.config.Cache.Backend) {
	case backendFile:
		stats["directory"] = cshc.config.Cache.File.Directory
		stats["max_file_size"] = cshc.config.Cache.File.MaxFileSize
	case backendS3:
		stats["bucket"] = cshc.config.Cache.S3.Bucket
		stats["region"] = cshc.config.Cache.S3.Region
		stats["use_ssl"] = cshc.config.Cache.S3.UseSSL
	case backendRedis:
		stats["address"] = cshc.config.Cache.Redis.Address
		stats["db"] = cshc.config.Cache.Redis.DB
		stats["key_prefix"] = cshc.config.Cache.Redis.KeyPrefix
	}

	return stats
}
