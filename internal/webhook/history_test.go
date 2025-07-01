package webhook

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewWebhookHistoryManager(t *testing.T) {
	tempDir := filepath.Join(os.TempDir(), "webhook_history_test")
	defer os.RemoveAll(tempDir)

	manager := NewWebhookHistoryManager(tempDir, 1000, 24*time.Hour)
	if manager == nil {
		t.Fatal("Expected non-nil manager")
	}

	if manager.storageDir != tempDir {
		t.Errorf("Expected storageDir %s, got %s", tempDir, manager.storageDir)
	}

	if manager.maxHistories != 1000 {
		t.Errorf("Expected maxHistories 1000, got %d", manager.maxHistories)
	}

	if manager.retentionTTL != 24*time.Hour {
		t.Errorf("Expected retentionTTL 24h, got %v", manager.retentionTTL)
	}

	// 디렉토리가 생성되었는지 확인
	if _, err := os.Stat(tempDir); os.IsNotExist(err) {
		t.Error("Expected storage directory to be created")
	}
}

func TestWebhookHistoryManager_AddHistory(t *testing.T) {
	tempDir := filepath.Join(os.TempDir(), "webhook_history_add_test")
	defer os.RemoveAll(tempDir)

	manager := NewWebhookHistoryManager(tempDir, 1000, 24*time.Hour)

	// 테스트 이력 아이템 생성
	item := &WebhookHistoryItem{
		EndpointName: "test-endpoint",
		URL:          "https://example.com/webhook",
		EventID:      "event-123",
		EventType:    "test.event",
		Status:       "success",
		Timestamp:    time.Now(),
		ResponseTime: 150 * time.Millisecond,
		StatusCode:   200,
		RetryCount:   0,
	}

	// 이력 추가
	err := manager.AddHistory(item)
	if err != nil {
		t.Fatalf("Failed to add history: %v", err)
	}

	// ID가 자동 생성되었는지 확인
	if item.ID == "" {
		t.Error("Expected ID to be auto-generated")
	}

	// 파일이 생성되었는지 확인
	dateStr := item.Timestamp.Format("2006-01-02")
	expectedFile := filepath.Join(tempDir, "webhook_history_"+dateStr+".json")
	if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
		t.Error("Expected history file to be created")
	}
}

func TestWebhookHistoryManager_GetHistory(t *testing.T) {
	tempDir := filepath.Join(os.TempDir(), "webhook_history_get_test")
	defer os.RemoveAll(tempDir)

	manager := NewWebhookHistoryManager(tempDir, 1000, 24*time.Hour)

	// 테스트 이력 추가
	now := time.Now()
	items := []*WebhookHistoryItem{
		{
			EndpointName: "endpoint-1",
			URL:          "https://example.com/webhook1",
			EventID:      "event-1",
			Status:       "success",
			Timestamp:    now,
		},
		{
			EndpointName: "endpoint-2",
			URL:          "https://example.com/webhook2",
			EventID:      "event-2",
			Status:       "failed",
			Timestamp:    now.Add(1 * time.Minute),
		},
		{
			EndpointName: "endpoint-1",
			URL:          "https://example.com/webhook1",
			EventID:      "event-3",
			Status:       "success",
			Timestamp:    now.Add(2 * time.Minute),
		},
	}

	for _, item := range items {
		if err := manager.AddHistory(item); err != nil {
			t.Fatalf("Failed to add history: %v", err)
		}
	}

	// 전체 이력 조회
	histories, total, err := manager.GetHistory("", 10, 0)
	if err != nil {
		t.Fatalf("Failed to get history: %v", err)
	}

	if total != 3 {
		t.Errorf("Expected total 3, got %d", total)
	}

	if len(histories) != 3 {
		t.Errorf("Expected 3 histories, got %d", len(histories))
	}

	// 최신순 정렬 확인 (event-3가 첫 번째)
	if histories[0].EventID != "event-3" {
		t.Errorf("Expected first history to be event-3, got %s", histories[0].EventID)
	}

	// 특정 엔드포인트 필터링
	endpoint1Histories, total1, err := manager.GetHistory("endpoint-1", 10, 0)
	if err != nil {
		t.Fatalf("Failed to get filtered history: %v", err)
	}

	if total1 != 2 {
		t.Errorf("Expected total 2 for endpoint-1, got %d", total1)
	}

	if len(endpoint1Histories) != 2 {
		t.Errorf("Expected 2 histories for endpoint-1, got %d", len(endpoint1Histories))
	}

	// 페이징 테스트
	paged, totalPaged, err := manager.GetHistory("", 1, 1)
	if err != nil {
		t.Fatalf("Failed to get paged history: %v", err)
	}

	if totalPaged != 3 {
		t.Errorf("Expected total 3 in paged result, got %d", totalPaged)
	}

	if len(paged) != 1 {
		t.Errorf("Expected 1 history in paged result, got %d", len(paged))
	}

	// 두 번째 항목이 반환되어야 함 (event-2)
	if paged[0].EventID != "event-2" {
		t.Errorf("Expected paged result to be event-2, got %s", paged[0].EventID)
	}
}

func TestWebhookHistoryManager_GetStatistics(t *testing.T) {
	tempDir := filepath.Join(os.TempDir(), "webhook_history_stats_test")
	defer os.RemoveAll(tempDir)

	manager := NewWebhookHistoryManager(tempDir, 1000, 24*time.Hour)

	// 테스트 이력 추가
	now := time.Now()
	items := []*WebhookHistoryItem{
		{
			EndpointName: "endpoint-1",
			Status:       "success",
			Timestamp:    now.Add(-1 * time.Hour),
			ResponseTime: 100 * time.Millisecond,
		},
		{
			EndpointName: "endpoint-1",
			Status:       "failed",
			Timestamp:    now.Add(-30 * time.Minute),
			ResponseTime: 200 * time.Millisecond,
		},
		{
			EndpointName: "endpoint-2",
			Status:       "success",
			Timestamp:    now.Add(-15 * time.Minute),
			ResponseTime: 150 * time.Millisecond,
		},
		{
			EndpointName: "endpoint-1",
			Status:       "success",
			Timestamp:    now.Add(-5 * time.Minute),
			ResponseTime: 120 * time.Millisecond,
		},
	}

	for _, item := range items {
		if err := manager.AddHistory(item); err != nil {
			t.Fatalf("Failed to add history: %v", err)
		}
	}

	// 전체 통계 조회
	stats, err := manager.GetStatistics("", nil)
	if err != nil {
		t.Fatalf("Failed to get statistics: %v", err)
	}

	if stats.TotalSent != 4 {
		t.Errorf("Expected TotalSent 4, got %d", stats.TotalSent)
	}

	if stats.TotalSuccess != 3 {
		t.Errorf("Expected TotalSuccess 3, got %d", stats.TotalSuccess)
	}

	if stats.TotalFailed != 1 {
		t.Errorf("Expected TotalFailed 1, got %d", stats.TotalFailed)
	}

	expectedSuccessRate := 75.0 // 3/4 * 100
	if stats.SuccessRate != expectedSuccessRate {
		t.Errorf("Expected SuccessRate %.1f, got %.1f", expectedSuccessRate, stats.SuccessRate)
	}

	// 평균 응답 시간 확인 (100+200+150+120)/4 = 142.5ms
	expectedAvgResponseTime := 142500 * time.Microsecond // 142.5ms
	if stats.AverageResponseTime < expectedAvgResponseTime-time.Millisecond ||
		stats.AverageResponseTime > expectedAvgResponseTime+time.Millisecond {
		t.Errorf("Expected AverageResponseTime around %v, got %v", expectedAvgResponseTime, stats.AverageResponseTime)
	}

	// 엔드포인트별 통계 확인
	if len(stats.EndpointStats) != 2 {
		t.Errorf("Expected 2 endpoint stats, got %d", len(stats.EndpointStats))
	}

	ep1Stats, exists := stats.EndpointStats["endpoint-1"]
	if !exists {
		t.Error("Expected endpoint-1 stats to exist")
	} else {
		if ep1Stats.TotalSent != 3 {
			t.Errorf("Expected endpoint-1 TotalSent 3, got %d", ep1Stats.TotalSent)
		}
		if ep1Stats.TotalSuccess != 2 {
			t.Errorf("Expected endpoint-1 TotalSuccess 2, got %d", ep1Stats.TotalSuccess)
		}
		if ep1Stats.TotalFailed != 1 {
			t.Errorf("Expected endpoint-1 TotalFailed 1, got %d", ep1Stats.TotalFailed)
		}
		expectedEp1SuccessRate := 66.67 // 2/3 * 100 (반올림)
		if ep1Stats.SuccessRate < expectedEp1SuccessRate-0.1 || ep1Stats.SuccessRate > expectedEp1SuccessRate+0.1 {
			t.Errorf("Expected endpoint-1 SuccessRate around %.2f, got %.2f", expectedEp1SuccessRate, ep1Stats.SuccessRate)
		}
	}

	ep2Stats, exists := stats.EndpointStats["endpoint-2"]
	if !exists {
		t.Error("Expected endpoint-2 stats to exist")
	} else {
		if ep2Stats.TotalSent != 1 {
			t.Errorf("Expected endpoint-2 TotalSent 1, got %d", ep2Stats.TotalSent)
		}
		if ep2Stats.TotalSuccess != 1 {
			t.Errorf("Expected endpoint-2 TotalSuccess 1, got %d", ep2Stats.TotalSuccess)
		}
		if ep2Stats.SuccessRate != 100.0 {
			t.Errorf("Expected endpoint-2 SuccessRate 100.0, got %.2f", ep2Stats.SuccessRate)
		}
	}
}

func TestWebhookHistoryManager_GetStatistics_WithTimeRange(t *testing.T) {
	tempDir := filepath.Join(os.TempDir(), "webhook_history_timerange_test")
	defer os.RemoveAll(tempDir)

	manager := NewWebhookHistoryManager(tempDir, 1000, 24*time.Hour)

	// 시간 범위별 테스트 이력 추가
	now := time.Now()
	items := []*WebhookHistoryItem{
		{
			EndpointName: "endpoint-1",
			Status:       "success",
			Timestamp:    now.Add(-2 * time.Hour), // 범위 밖
		},
		{
			EndpointName: "endpoint-1",
			Status:       "success",
			Timestamp:    now.Add(-30 * time.Minute), // 범위 안
		},
		{
			EndpointName: "endpoint-1",
			Status:       "failed",
			Timestamp:    now.Add(-15 * time.Minute), // 범위 안
		},
	}

	for _, item := range items {
		if err := manager.AddHistory(item); err != nil {
			t.Fatalf("Failed to add history: %v", err)
		}
	}

	// 최근 1시간 통계 조회
	timeRange := &TimeRangeStats{
		StartTime: now.Add(-1 * time.Hour),
		EndTime:   now,
	}

	stats, err := manager.GetStatistics("", timeRange)
	if err != nil {
		t.Fatalf("Failed to get time-range statistics: %v", err)
	}

	// 시간 범위 내의 항목만 포함되어야 함 (2개)
	if stats.TotalSent != 2 {
		t.Errorf("Expected TotalSent 2 in time range, got %d", stats.TotalSent)
	}

	if stats.TotalSuccess != 1 {
		t.Errorf("Expected TotalSuccess 1 in time range, got %d", stats.TotalSuccess)
	}

	if stats.TotalFailed != 1 {
		t.Errorf("Expected TotalFailed 1 in time range, got %d", stats.TotalFailed)
	}

	// 시간 범위 정보 확인
	if stats.TimeRange == nil {
		t.Error("Expected TimeRange to be set")
	} else {
		if !stats.TimeRange.StartTime.Equal(timeRange.StartTime) {
			t.Error("TimeRange StartTime mismatch")
		}
		if !stats.TimeRange.EndTime.Equal(timeRange.EndTime) {
			t.Error("TimeRange EndTime mismatch")
		}
	}
}

func TestWebhookHistoryManager_GetRecentActivity(t *testing.T) {
	tempDir := filepath.Join(os.TempDir(), "webhook_history_recent_test")
	defer os.RemoveAll(tempDir)

	manager := NewWebhookHistoryManager(tempDir, 1000, 24*time.Hour)

	// 테스트 이력 추가
	now := time.Now()
	items := []*WebhookHistoryItem{
		{
			EndpointName: "endpoint-1",
			EventID:      "event-1",
			Status:       "success",
			Timestamp:    now,
		},
		{
			EndpointName: "endpoint-2",
			EventID:      "event-2",
			Status:       "failed",
			Timestamp:    now.Add(1 * time.Minute),
		},
		{
			EndpointName: "endpoint-3",
			EventID:      "event-3",
			Status:       "success",
			Timestamp:    now.Add(2 * time.Minute),
		},
	}

	for _, item := range items {
		if err := manager.AddHistory(item); err != nil {
			t.Fatalf("Failed to add history: %v", err)
		}
	}

	// 최근 활동 조회 (limit=2)
	activities, err := manager.GetRecentActivity(2)
	if err != nil {
		t.Fatalf("Failed to get recent activity: %v", err)
	}

	if len(activities) != 2 {
		t.Errorf("Expected 2 recent activities, got %d", len(activities))
	}

	// 최신순 정렬 확인
	if activities[0].EventID != "event-3" {
		t.Errorf("Expected first activity to be event-3, got %s", activities[0].EventID)
	}

	if activities[1].EventID != "event-2" {
		t.Errorf("Expected second activity to be event-2, got %s", activities[1].EventID)
	}
}

func TestWebhookHistoryManager_CleanupOldHistories(t *testing.T) {
	tempDir := filepath.Join(os.TempDir(), "webhook_history_cleanup_test")
	defer os.RemoveAll(tempDir)

	// 짧은 보관 기간으로 매니저 생성 (테스트용)
	manager := NewWebhookHistoryManager(tempDir, 1000, 1*time.Hour)

	// 오래된 이력 추가 (3일 전 - 다른 날짜 파일에 저장되도록)
	oldDate := time.Now().Add(-3 * 24 * time.Hour)
	oldItem := &WebhookHistoryItem{
		EndpointName: "endpoint-1",
		Status:       "success",
		Timestamp:    oldDate,
	}

	// 최근 이력 추가 (오늘)
	recentItem := &WebhookHistoryItem{
		EndpointName: "endpoint-1",
		Status:       "success",
		Timestamp:    time.Now(),
	}

	if err := manager.AddHistory(oldItem); err != nil {
		t.Fatalf("Failed to add old history: %v", err)
	}

	if err := manager.AddHistory(recentItem); err != nil {
		t.Fatalf("Failed to add recent history: %v", err)
	}

	// 정리 전 파일 확인
	files, _ := filepath.Glob(filepath.Join(tempDir, "webhook_history_*.json"))
	beforeCleanup := len(files)

	// 최소 2개 파일이 있어야 함 (오늘 + 3일 전)
	if beforeCleanup < 2 {
		t.Logf("Warning: Expected at least 2 files before cleanup, got %d", beforeCleanup)
	}

	// 정리 실행
	err := manager.CleanupOldHistories()
	if err != nil {
		t.Fatalf("Failed to cleanup old histories: %v", err)
	}

	// 정리 후 파일 확인
	files, _ = filepath.Glob(filepath.Join(tempDir, "webhook_history_*.json"))
	afterCleanup := len(files)

	// 오래된 파일이 삭제되었는지 확인 (최소 1개는 삭제되어야 함)
	if beforeCleanup >= 2 && afterCleanup >= beforeCleanup {
		t.Errorf("Expected fewer files after cleanup, before: %d, after: %d", beforeCleanup, afterCleanup)
	}

	// 최근 이력은 여전히 조회 가능해야 함
	histories, total, err := manager.GetHistory("", 10, 0)
	if err != nil {
		t.Fatalf("Failed to get history after cleanup: %v", err)
	}

	if total == 0 {
		t.Error("Expected recent history to still exist after cleanup")
	}

	// 최근 항목이 남아있는지 확인
	foundRecent := false
	for _, history := range histories {
		if history.Timestamp.After(time.Now().Add(-30 * time.Minute)) {
			foundRecent = true
			break
		}
	}

	if !foundRecent {
		t.Error("Expected recent history to be preserved after cleanup")
	}

	// 오래된 항목이 제거되었는지 확인
	foundOld := false
	for _, history := range histories {
		if history.Timestamp.Before(time.Now().Add(-2 * 24 * time.Hour)) {
			foundOld = true
			break
		}
	}

	if foundOld {
		t.Error("Expected old history to be removed after cleanup")
	}
}