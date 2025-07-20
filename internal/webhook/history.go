package webhook

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"proxynd/logging"
)

// WebhookHistoryItem 웹훅 전송 이력 항목
type WebhookHistoryItem struct {
	ID           string                 `json:"id"`
	EndpointName string                 `json:"endpoint_name"`
	URL          string                 `json:"url"`
	EventID      string                 `json:"event_id"`
	EventType    string                 `json:"event_type"`
	Status       string                 `json:"status"` // success, failed, retrying
	Timestamp    time.Time              `json:"timestamp"`
	ResponseTime time.Duration          `json:"response_time"`
	StatusCode   int                    `json:"status_code,omitempty"`
	ErrorMessage string                 `json:"error_message,omitempty"`
	RetryCount   int                    `json:"retry_count"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// WebhookStatistics 웹훅 통계 정보
type WebhookStatistics struct {
	TotalSent           int64                          `json:"total_sent"`
	TotalSuccess        int64                          `json:"total_success"`
	TotalFailed         int64                          `json:"total_failed"`
	TotalRetrying       int64                          `json:"total_retrying"`
	SuccessRate         float64                        `json:"success_rate"`
	AverageResponseTime time.Duration                  `json:"average_response_time"`
	EndpointStats       map[string]*EndpointStatistics `json:"endpoint_stats"`
	LastUpdated         time.Time                      `json:"last_updated"`
	TimeRange           *TimeRangeStats                `json:"time_range,omitempty"`
}

// EndpointStatistics 엔드포인트별 통계
type EndpointStatistics struct {
	EndpointName        string        `json:"endpoint_name"`
	TotalSent           int64         `json:"total_sent"`
	TotalSuccess        int64         `json:"total_success"`
	TotalFailed         int64         `json:"total_failed"`
	SuccessRate         float64       `json:"success_rate"`
	AverageResponseTime time.Duration `json:"average_response_time"`
	LastSent            time.Time     `json:"last_sent"`
	LastSuccess         time.Time     `json:"last_success"`
	LastFailure         time.Time     `json:"last_failure"`
}

// TimeRangeStats 시간대별 통계
type TimeRangeStats struct {
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Duration  string    `json:"duration"`
}

// WebhookHistoryManager 웹훅 이력 관리자
type WebhookHistoryManager struct {
	storageDir   string
	logger       logging.Logger
	mu           sync.RWMutex
	maxHistories int           // 최대 보관 이력 수
	retentionTTL time.Duration // 이력 보관 기간
}

// NewWebhookHistoryManager 새로운 이력 관리자 생성
func NewWebhookHistoryManager(storageDir string, maxHistories int, retentionTTL time.Duration) *WebhookHistoryManager {
	manager := &WebhookHistoryManager{
		storageDir:   storageDir,
		logger:       logging.GetLogger(),
		maxHistories: maxHistories,
		retentionTTL: retentionTTL,
	}

	// 저장 디렉토리 생성
	if err := os.MkdirAll(storageDir, 0755); err != nil {
		manager.logger.Error("웹훅 이력 저장 디렉토리 생성 실패",
			logging.F("dir", storageDir),
			logging.F("error", err))
	}

	return manager
}

// AddHistory 이력 추가
func (whm *WebhookHistoryManager) AddHistory(item *WebhookHistoryItem) error {
	whm.mu.Lock()
	defer whm.mu.Unlock()

	// ID가 없으면 생성
	if item.ID == "" {
		item.ID = fmt.Sprintf("history_%d_%s", time.Now().UnixNano(), item.EndpointName)
	}

	// 파일 경로 생성 (날짜별 분할 저장)
	dateStr := item.Timestamp.Format("2006-01-02")
	filename := fmt.Sprintf("webhook_history_%s.json", dateStr)
	filepath := filepath.Join(whm.storageDir, filename)

	// 기존 이력 로드
	var histories []WebhookHistoryItem
	if data, err := os.ReadFile(filepath); err == nil {
		if err := json.Unmarshal(data, &histories); err != nil {
			whm.logger.Error("웹훅 이력 파일 파싱 실패",
				logging.F("file", filepath),
				logging.F("error", err))
		}
	}

	// 새 이력 추가
	histories = append(histories, *item)

	// 최대 개수 제한 (파일별)
	maxPerFile := whm.maxHistories / 30 // 30일 기준 분할
	if maxPerFile < 100 {
		maxPerFile = 100 // 최소 100개
	}

	if len(histories) > maxPerFile {
		// 오래된 것부터 제거
		sort.Slice(histories, func(i, j int) bool {
			return histories[i].Timestamp.After(histories[j].Timestamp)
		})
		histories = histories[:maxPerFile]
	}

	// 파일에 저장
	data, err := json.MarshalIndent(histories, "", "  ")
	if err != nil {
		return fmt.Errorf("웹훅 이력 직렬화 실패: %w", err)
	}

	if err := os.WriteFile(filepath, data, 0644); err != nil {
		return fmt.Errorf("웹훅 이력 저장 실패: %w", err)
	}

	whm.logger.Debug("웹훅 이력 추가됨",
		logging.F("id", item.ID),
		logging.F("endpoint", item.EndpointName),
		logging.F("status", item.Status))

	return nil
}

// GetHistory 이력 조회 (페이징 지원)
func (whm *WebhookHistoryManager) GetHistory(endpointName string, limit int,
	offset int) ([]WebhookHistoryItem, int, error) {
	whm.mu.RLock()
	defer whm.mu.RUnlock()

	var allHistories []WebhookHistoryItem

	// 모든 히스토리 파일 로드 (최근 30일)
	for i := 0; i < 30; i++ {
		date := time.Now().AddDate(0, 0, -i)
		dateStr := date.Format("2006-01-02")
		filename := fmt.Sprintf("webhook_history_%s.json", dateStr)
		filepath := filepath.Join(whm.storageDir, filename)

		if data, err := os.ReadFile(filepath); err == nil {
			var histories []WebhookHistoryItem
			if err := json.Unmarshal(data, &histories); err == nil {
				allHistories = append(allHistories, histories...)
			}
		}
	}

	// 엔드포인트별 필터링
	var filteredHistories []WebhookHistoryItem
	for _, history := range allHistories {
		if endpointName == "" || history.EndpointName == endpointName {
			filteredHistories = append(filteredHistories, history)
		}
	}

	// 시간순 정렬 (최신순)
	sort.Slice(filteredHistories, func(i, j int) bool {
		return filteredHistories[i].Timestamp.After(filteredHistories[j].Timestamp)
	})

	total := len(filteredHistories)

	// 페이징 적용
	if offset >= total {
		return []WebhookHistoryItem{}, total, nil
	}

	end := offset + limit
	if end > total {
		end = total
	}

	return filteredHistories[offset:end], total, nil
}

// GetStatistics 통계 정보 조회
func (whm *WebhookHistoryManager) GetStatistics(endpointName string,
	timeRange *TimeRangeStats) (*WebhookStatistics, error) {
	whm.mu.RLock()
	defer whm.mu.RUnlock()

	// 시간 범위 설정
	var startTime, endTime time.Time
	if timeRange != nil {
		startTime = timeRange.StartTime
		endTime = timeRange.EndTime
	} else {
		// 기본값: 최근 24시간
		endTime = time.Now()
		startTime = endTime.Add(-24 * time.Hour)
	}

	var allHistories []WebhookHistoryItem

	// 필요한 날짜 범위의 파일들만 로드
	current := startTime.Truncate(24 * time.Hour)
	endTruncated := endTime.Truncate(24 * time.Hour)

	for !current.After(endTruncated) {
		dateStr := current.Format("2006-01-02")
		filename := fmt.Sprintf("webhook_history_%s.json", dateStr)
		filepath := filepath.Join(whm.storageDir, filename)

		if data, err := os.ReadFile(filepath); err == nil {
			var histories []WebhookHistoryItem
			if err := json.Unmarshal(data, &histories); err == nil {
				// 시간 범위 필터링
				for _, history := range histories {
					if (history.Timestamp.After(startTime) || history.Timestamp.Equal(startTime)) &&
						(history.Timestamp.Before(endTime) || history.Timestamp.Equal(endTime)) {
						if endpointName == "" || history.EndpointName == endpointName {
							allHistories = append(allHistories, history)
						}
					}
				}
			}
		}
		current = current.Add(24 * time.Hour)
	}

	// 통계 계산
	stats := &WebhookStatistics{
		EndpointStats: make(map[string]*EndpointStatistics),
		LastUpdated:   time.Now(),
		TimeRange: &TimeRangeStats{
			StartTime: startTime,
			EndTime:   endTime,
			Duration:  endTime.Sub(startTime).String(),
		},
	}

	var totalResponseTime time.Duration
	responseTimeCount := 0

	for _, history := range allHistories {
		stats.TotalSent++

		switch history.Status {
		case "success":
			stats.TotalSuccess++
		case "failed":
			stats.TotalFailed++
		case "retrying":
			stats.TotalRetrying++
		}

		// 응답 시간 누적
		if history.ResponseTime > 0 {
			totalResponseTime += history.ResponseTime
			responseTimeCount++
		}

		// 엔드포인트별 통계
		epStats, exists := stats.EndpointStats[history.EndpointName]
		if !exists {
			epStats = &EndpointStatistics{
				EndpointName: history.EndpointName,
			}
			stats.EndpointStats[history.EndpointName] = epStats
		}

		epStats.TotalSent++
		if history.Status == "success" {
			epStats.TotalSuccess++
			if history.Timestamp.After(epStats.LastSuccess) {
				epStats.LastSuccess = history.Timestamp
			}
		}
		if history.Status == "failed" {
			epStats.TotalFailed++
			if history.Timestamp.After(epStats.LastFailure) {
				epStats.LastFailure = history.Timestamp
			}
		}

		if history.Timestamp.After(epStats.LastSent) {
			epStats.LastSent = history.Timestamp
		}

		if history.ResponseTime > 0 {
			// 엔드포인트별 평균 응답 시간 계산을 위한 누적
			epStats.AverageResponseTime = ((epStats.AverageResponseTime *
				time.Duration(epStats.TotalSent-1)) + history.ResponseTime) /
				time.Duration(epStats.TotalSent)
		}
	}

	// 전체 성공률 계산
	if stats.TotalSent > 0 {
		stats.SuccessRate = float64(stats.TotalSuccess) / float64(stats.TotalSent) * 100
	}

	// 전체 평균 응답 시간
	if responseTimeCount > 0 {
		stats.AverageResponseTime = totalResponseTime / time.Duration(responseTimeCount)
	}

	// 엔드포인트별 성공률 계산
	for _, epStats := range stats.EndpointStats {
		if epStats.TotalSent > 0 {
			epStats.SuccessRate = float64(epStats.TotalSuccess) / float64(epStats.TotalSent) * 100
		}
	}

	return stats, nil
}

// CleanupOldHistories 오래된 이력 정리
func (whm *WebhookHistoryManager) CleanupOldHistories() error {
	whm.mu.Lock()
	defer whm.mu.Unlock()

	cutoffTime := time.Now().Add(-whm.retentionTTL)
	cutoffDate := cutoffTime.Format("2006-01-02")

	files, err := filepath.Glob(filepath.Join(whm.storageDir, "webhook_history_*.json"))
	if err != nil {
		return fmt.Errorf("이력 파일 목록 조회 실패: %w", err)
	}

	deletedCount := 0
	for _, file := range files {
		filename := filepath.Base(file)
		// webhook_history_YYYY-MM-DD.json 형식에서 날짜 추출
		if len(filename) >= 30 {
			dateStr := filename[16:26] // YYYY-MM-DD 부분
			if dateStr < cutoffDate {
				if err := os.Remove(file); err != nil {
					whm.logger.Error("오래된 이력 파일 삭제 실패",
						logging.F("file", file),
						logging.F("error", err))
				} else {
					deletedCount++
					whm.logger.Debug("오래된 이력 파일 삭제됨",
						logging.F("file", filename))
				}
			}
		}
	}

	if deletedCount > 0 {
		whm.logger.Info("오래된 웹훅 이력 정리 완료",
			logging.F("deleted_files", deletedCount),
			logging.F("cutoff_date", cutoffDate))
	}

	return nil
}

// GetRecentActivity 최근 활동 조회 (대시보드용)
func (whm *WebhookHistoryManager) GetRecentActivity(limit int) ([]WebhookHistoryItem, error) {
	histories, _, err := whm.GetHistory("", limit, 0)
	return histories, err
}
