package metrics

import (
	"testing"
)

func TestTTLCollector_RecordTTLCalculation(t *testing.T) {
	collector := NewTTLCollector(100)

	// 테스트 데이터 기록
	collector.RecordTTLCalculation("express", "npm", 1800, "package_type")
	collector.RecordTTLCalculation("spring-boot-SNAPSHOT", "maven", 300, "pattern_match")
	collector.RecordTTLCalculation("react", "npm", 7200, "cache_header")

	// 엔트리 수 확인
	entries := collector.GetRecentEntries(10)
	if len(entries) != 3 {
		t.Errorf("Expected 3 entries, got %d", len(entries))
	}

	// 통계 강제 업데이트
	collector.mu.Lock()
	collector.updateStatistics()
	collector.mu.Unlock()

	// 글로벌 통계 확인
	stats := collector.GetStats("")
	if stats.TotalCalculations != 3 {
		t.Errorf("Expected 3 total calculations, got %d", stats.TotalCalculations)
	}

	if stats.AverageTTL == 0 {
		t.Error("Expected non-zero average TTL")
	}
}

func TestTTLCollector_Statistics(t *testing.T) {
	collector := NewTTLCollector(100)

	// 다양한 TTL 값 기록
	testData := []struct {
		packageName string
		packageType string
		ttl         int
		source      string
	}{
		{"express", "npm", 1800, "package_type"},
		{"react", "npm", 3600, "package_type"},
		{"vue", "npm", 900, "pattern_match"},
		{"angular", "npm", 7200, "cache_header"},
		{"lodash", "npm", 1800, "package_type"},
	}

	for _, data := range testData {
		collector.RecordTTLCalculation(data.packageName, data.packageType, data.ttl, data.source)
	}

	// 통계 강제 업데이트
	collector.mu.Lock()
	collector.updateStatistics()
	collector.mu.Unlock()

	// NPM 통계 확인
	npmStats := collector.GetStats("npm")
	if npmStats.TotalCalculations != 5 {
		t.Errorf("Expected 5 npm calculations, got %d", npmStats.TotalCalculations)
	}

	if npmStats.MinTTL != 900 {
		t.Errorf("Expected min TTL 900, got %d", npmStats.MinTTL)
	}

	if npmStats.MaxTTL != 7200 {
		t.Errorf("Expected max TTL 7200, got %d", npmStats.MaxTTL)
	}

	// 평균 계산 확인 (1800+3600+900+7200+1800)/5 = 3060
	expectedAvg := 3060.0
	if npmStats.AverageTTL != expectedAvg {
		t.Errorf("Expected average TTL %.1f, got %.1f", expectedAvg, npmStats.AverageTTL)
	}
}

func TestTTLCollector_MaxEntriesLimit(t *testing.T) {
	collector := NewTTLCollector(3) // 최대 3개 항목

	// 5개 항목 추가
	for i := 0; i < 5; i++ {
		collector.RecordTTLCalculation("package", "npm", 1800, "test")
	}

	// 3개만 유지되는지 확인
	entries := collector.GetRecentEntries(10)
	if len(entries) != 3 {
		t.Errorf("Expected 3 entries (max limit), got %d", len(entries))
	}
}

func TestTTLCollector_ExpirationTracking(t *testing.T) {
	collector := NewTTLCollector(100)

	// 일부 패키지 기록
	collector.RecordTTLCalculation("test-package", "npm", 1800, "package_type")
	
	// 만료 기록
	collector.RecordTTLExpiration("npm", "early")

	// 조기 만료가 기록되었는지 확인
	entries := collector.GetRecentEntries(10)
	hasEarlyExpiration := false
	for _, entry := range entries {
		if entry.IsExpired && entry.ExpiryReason == "early_expiration" {
			hasEarlyExpiration = true
			break
		}
	}

	if !hasEarlyExpiration {
		t.Error("Expected early expiration to be recorded")
	}
}

func TestCalculatePercentile(t *testing.T) {
	values := []int{100, 200, 300, 400, 500, 600, 700, 800, 900, 1000}

	// 50th percentile (median) should be 550
	p50 := calculatePercentile(values, 50)
	if p50 != 550 {
		t.Errorf("Expected 50th percentile 550, got %.1f", p50)
	}

	// 90th percentile should be 950
	p90 := calculatePercentile(values, 90)
	if p90 != 950 {
		t.Errorf("Expected 90th percentile 950, got %.1f", p90)
	}

	// Edge cases
	p0 := calculatePercentile(values, 0)
	if p0 != 100 {
		t.Errorf("Expected 0th percentile 100, got %.1f", p0)
	}

	p100 := calculatePercentile(values, 100)
	if p100 != 1000 {
		t.Errorf("Expected 100th percentile 1000, got %.1f", p100)
	}
}

func TestTTLCollector_ConcurrentAccess(t *testing.T) {
	collector := NewTTLCollector(1000)

	// 동시에 여러 고루틴에서 기록
	done := make(chan bool, 10)
	
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 10; j++ {
				collector.RecordTTLCalculation("concurrent", "npm", 1800, "test")
			}
			done <- true
		}(i)
	}

	// 모든 고루틴 완료 대기
	for i := 0; i < 10; i++ {
		<-done
	}

	// 모든 기록이 완료되었는지 확인
	entries := collector.GetRecentEntries(200)
	if len(entries) != 100 {
		t.Errorf("Expected 100 entries from concurrent access, got %d", len(entries))
	}
}