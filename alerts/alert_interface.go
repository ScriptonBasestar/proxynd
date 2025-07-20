package alerts

import (
	"context"
	"time"
)

// AlertLevel 알림 레벨 정의
type AlertLevel string

const (
	// AlertLevelInfo represents the informational alert level
	AlertLevelInfo AlertLevel = "INFO"
	// AlertLevelWarning represents the warning alert level
	AlertLevelWarning AlertLevel = "WARNING"
	// AlertLevelError represents the error alert level
	AlertLevelError AlertLevel = "ERROR"
	// AlertLevelCritical represents the critical alert level
	AlertLevelCritical AlertLevel = "CRITICAL"
)

// AlertEvent 알림 이벤트 구조체
type AlertEvent struct {
	ID          string                 `json:"id"`
	Level       AlertLevel             `json:"level"`
	Type        string                 `json:"type"`
	Title       string                 `json:"title"`
	Message     string                 `json:"message"`
	Source      string                 `json:"source"`
	Timestamp   time.Time              `json:"timestamp"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	PackageInfo *PackageInfo           `json:"package_info,omitempty"`
}

// PackageInfo 패키지 정보
type PackageInfo struct {
	Type         string `json:"type"`          // npm, pip, apt, docker
	Name         string `json:"name"`          // 패키지 이름
	Version      string `json:"version"`       // 패키지 버전
	Path         string `json:"path"`          // 요청 경로
	ExpectedHash string `json:"expected_hash"` // 예상 해시
	ActualHash   string `json:"actual_hash"`   // 실제 해시
	RemoteURL    string `json:"remote_url"`    // 원격 저장소 URL
}

// Alerter 알림 전송 인터페이스
type Alerter interface {
	// Send 알림 전송
	Send(ctx context.Context, event *AlertEvent) error
	// SendBatch 여러 알림 일괄 전송
	SendBatch(ctx context.Context, events []*AlertEvent) error
	// IsEnabled 알림 채널 활성화 여부
	IsEnabled() bool
	// Name 알림 채널 이름
	Name() string
}

// AlertManager 알림 관리자 인터페이스
type AlertManager interface {
	// RegisterAlerter 알림 채널 등록
	RegisterAlerter(alerter Alerter)
	// UnregisterAlerter 알림 채널 해제
	UnregisterAlerter(name string)
	// Send 모든 등록된 채널로 알림 전송
	Send(ctx context.Context, event *AlertEvent) error
	// SendToChannel 특정 채널로 알림 전송
	SendToChannel(ctx context.Context, channelName string, event *AlertEvent) error
	// GetAlerters 등록된 알림 채널 목록
	GetAlerters() []string
}

// AlertConfig 알림 설정
type AlertConfig struct {
	Enabled     bool            `yaml:"enabled" json:"enabled"`
	Channels    []ChannelConfig `yaml:"channels" json:"channels"`
	RateLimit   RateLimitConfig `yaml:"rate_limit" json:"rate_limit"`
	FilterRules []FilterRule    `yaml:"filter_rules" json:"filter_rules"`
}

// ChannelConfig 채널별 설정
type ChannelConfig struct {
	Name    string                 `yaml:"name" json:"name"`
	Type    string                 `yaml:"type" json:"type"` // log, webhook, email, slack
	Enabled bool                   `yaml:"enabled" json:"enabled"`
	Config  map[string]interface{} `yaml:"config" json:"config"`
}

// RateLimitConfig 속도 제한 설정
type RateLimitConfig struct {
	Enabled      bool `yaml:"enabled" json:"enabled"`
	MaxPerMinute int  `yaml:"max_per_minute" json:"max_per_minute"`
	MaxPerHour   int  `yaml:"max_per_hour" json:"max_per_hour"`
	BurstSize    int  `yaml:"burst_size" json:"burst_size"`
}

// FilterRule 필터 규칙
type FilterRule struct {
	Name       string                 `yaml:"name" json:"name"`
	Enabled    bool                   `yaml:"enabled" json:"enabled"`
	Levels     []AlertLevel           `yaml:"levels" json:"levels"`
	Types      []string               `yaml:"types" json:"types"`
	Conditions map[string]interface{} `yaml:"conditions" json:"conditions"`
	Action     string                 `yaml:"action" json:"action"` // allow, deny, redirect
	Target     string                 `yaml:"target" json:"target"` // 리다이렉트 대상 채널
}
