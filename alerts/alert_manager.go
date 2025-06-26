package alerts

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
	
	"github.com/google/uuid"
)

// DefaultAlertManager 기본 알림 관리자 구현
type DefaultAlertManager struct {
	alerters      map[string]Alerter
	mutex         sync.RWMutex
	config        *AlertConfig
	rateLimiter   *RateLimiter
	filterEngine  *FilterEngine
}

// NewAlertManager 새 알림 관리자 생성
func NewAlertManager(config *AlertConfig) *DefaultAlertManager {
	am := &DefaultAlertManager{
		alerters: make(map[string]Alerter),
		config:   config,
	}
	
	// 속도 제한기 초기화
	if config != nil && config.RateLimit.Enabled {
		am.rateLimiter = NewRateLimiter(
			config.RateLimit.MaxPerMinute,
			config.RateLimit.MaxPerHour,
			config.RateLimit.BurstSize,
		)
	}
	
	// 필터 엔진 초기화
	if config != nil && len(config.FilterRules) > 0 {
		am.filterEngine = NewFilterEngine(config.FilterRules)
	}
	
	return am
}

// RegisterAlerter 알림 채널 등록
func (am *DefaultAlertManager) RegisterAlerter(alerter Alerter) {
	am.mutex.Lock()
	defer am.mutex.Unlock()
	
	name := alerter.Name()
	am.alerters[name] = alerter
	log.Printf("Alert channel registered: %s", name)
}

// UnregisterAlerter 알림 채널 해제
func (am *DefaultAlertManager) UnregisterAlerter(name string) {
	am.mutex.Lock()
	defer am.mutex.Unlock()
	
	delete(am.alerters, name)
	log.Printf("Alert channel unregistered: %s", name)
}

// Send 모든 등록된 채널로 알림 전송
func (am *DefaultAlertManager) Send(ctx context.Context, event *AlertEvent) error {
	// 이벤트 ID 생성
	if event.ID == "" {
		event.ID = uuid.New().String()
	}
	
	// 타임스탬프 설정
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}
	
	// 속도 제한 체크
	if am.rateLimiter != nil && !am.rateLimiter.Allow() {
		return fmt.Errorf("rate limit exceeded")
	}
	
	// 필터 적용
	if am.filterEngine != nil {
		action, target := am.filterEngine.Evaluate(event)
		switch action {
		case "deny":
			return nil // 알림 차단
		case "redirect":
			if target != "" {
				return am.SendToChannel(ctx, target, event)
			}
		}
	}
	
	am.mutex.RLock()
	defer am.mutex.RUnlock()
	
	var errors []error
	successCount := 0
	
	for name, alerter := range am.alerters {
		if !alerter.IsEnabled() {
			continue
		}
		
		if err := alerter.Send(ctx, event); err != nil {
			errors = append(errors, fmt.Errorf("%s: %w", name, err))
			log.Printf("Failed to send alert via %s: %v", name, err)
		} else {
			successCount++
		}
	}
	
	if successCount == 0 && len(errors) > 0 {
		return fmt.Errorf("failed to send alert to any channel: %v", errors)
	}
	
	return nil
}

// SendToChannel 특정 채널로 알림 전송
func (am *DefaultAlertManager) SendToChannel(ctx context.Context, channelName string, event *AlertEvent) error {
	am.mutex.RLock()
	alerter, exists := am.alerters[channelName]
	am.mutex.RUnlock()
	
	if !exists {
		return fmt.Errorf("alert channel not found: %s", channelName)
	}
	
	if !alerter.IsEnabled() {
		return fmt.Errorf("alert channel is disabled: %s", channelName)
	}
	
	return alerter.Send(ctx, event)
}

// GetAlerters 등록된 알림 채널 목록
func (am *DefaultAlertManager) GetAlerters() []string {
	am.mutex.RLock()
	defer am.mutex.RUnlock()
	
	names := make([]string, 0, len(am.alerters))
	for name := range am.alerters {
		names = append(names, name)
	}
	return names
}

// RateLimiter 속도 제한기
type RateLimiter struct {
	maxPerMinute int
	maxPerHour   int
	burstSize    int
	minuteWindow *timeWindow
	hourWindow   *timeWindow
	mutex        sync.Mutex
}

// NewRateLimiter 새 속도 제한기 생성
func NewRateLimiter(maxPerMinute, maxPerHour, burstSize int) *RateLimiter {
	return &RateLimiter{
		maxPerMinute: maxPerMinute,
		maxPerHour:   maxPerHour,
		burstSize:    burstSize,
		minuteWindow: newTimeWindow(time.Minute),
		hourWindow:   newTimeWindow(time.Hour),
	}
}

// Allow 요청 허용 여부 확인
func (rl *RateLimiter) Allow() bool {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()
	
	now := time.Now()
	
	// 분당 제한 체크
	if rl.maxPerMinute > 0 {
		count := rl.minuteWindow.count(now)
		if count >= rl.maxPerMinute {
			return false
		}
	}
	
	// 시간당 제한 체크
	if rl.maxPerHour > 0 {
		count := rl.hourWindow.count(now)
		if count >= rl.maxPerHour {
			return false
		}
	}
	
	// 버스트 체크
	if rl.burstSize > 0 {
		recentCount := rl.minuteWindow.countRecent(now, 10*time.Second)
		if recentCount >= rl.burstSize {
			return false
		}
	}
	
	// 허용된 요청 기록
	rl.minuteWindow.add(now)
	rl.hourWindow.add(now)
	
	return true
}

// timeWindow 시간 창 구현
type timeWindow struct {
	duration   time.Duration
	timestamps []time.Time
}

func newTimeWindow(duration time.Duration) *timeWindow {
	return &timeWindow{
		duration:   duration,
		timestamps: make([]time.Time, 0),
	}
}

func (tw *timeWindow) add(t time.Time) {
	tw.timestamps = append(tw.timestamps, t)
	tw.cleanup(t)
}

func (tw *timeWindow) count(now time.Time) int {
	tw.cleanup(now)
	return len(tw.timestamps)
}

func (tw *timeWindow) countRecent(now time.Time, duration time.Duration) int {
	cutoff := now.Add(-duration)
	count := 0
	for i := len(tw.timestamps) - 1; i >= 0; i-- {
		if tw.timestamps[i].Before(cutoff) {
			break
		}
		count++
	}
	return count
}

func (tw *timeWindow) cleanup(now time.Time) {
	cutoff := now.Add(-tw.duration)
	i := 0
	for i < len(tw.timestamps) && tw.timestamps[i].Before(cutoff) {
		i++
	}
	tw.timestamps = tw.timestamps[i:]
}

// FilterEngine 필터 엔진
type FilterEngine struct {
	rules []FilterRule
}

// NewFilterEngine 새 필터 엔진 생성
func NewFilterEngine(rules []FilterRule) *FilterEngine {
	return &FilterEngine{rules: rules}
}

// Evaluate 필터 규칙 평가
func (fe *FilterEngine) Evaluate(event *AlertEvent) (action string, target string) {
	for _, rule := range fe.rules {
		if !rule.Enabled {
			continue
		}
		
		// 레벨 필터
		if len(rule.Levels) > 0 {
			match := false
			for _, level := range rule.Levels {
				if event.Level == level {
					match = true
					break
				}
			}
			if !match {
				continue
			}
		}
		
		// 타입 필터
		if len(rule.Types) > 0 {
			match := false
			for _, t := range rule.Types {
				if event.Type == t {
					match = true
					break
				}
			}
			if !match {
				continue
			}
		}
		
		// 조건 평가 (추가 구현 필요)
		// TODO: 복잡한 조건 평가 로직
		
		return rule.Action, rule.Target
	}
	
	return "allow", ""
}