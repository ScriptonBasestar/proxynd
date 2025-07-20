package alerts

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// LogAlerter 로그 기반 알림 구현
type LogAlerter struct {
	enabled    bool
	logFile    string
	jsonFormat bool
	logger     *log.Logger
	file       *os.File
	mutex      sync.Mutex
}

// LogAlerterConfig 로그 알림 설정
type LogAlerterConfig struct {
	Enabled    bool   `json:"enabled"`
	LogFile    string `json:"log_file"`
	JSONFormat bool   `json:"json_format"`
}

// NewLogAlerter 새 로그 알림 생성
func NewLogAlerter(config LogAlerterConfig) (*LogAlerter, error) {
	la := &LogAlerter{
		enabled:    config.Enabled,
		logFile:    config.LogFile,
		jsonFormat: config.JSONFormat,
	}

	if config.LogFile != "" {
		// 로그 디렉토리 생성
		dir := filepath.Dir(config.LogFile)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create log directory: %w", err)
		}

		// 로그 파일 열기
		file, err := os.OpenFile(config.LogFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file: %w", err)
		}

		la.file = file
		la.logger = log.New(file, "", 0)
	} else {
		// 표준 출력 사용
		la.logger = log.New(os.Stdout, "[ALERT] ", log.LstdFlags)
	}

	return la, nil
}

// Send 알림 전송
func (la *LogAlerter) Send(_ context.Context, event *AlertEvent) error {
	la.mutex.Lock()
	defer la.mutex.Unlock()

	if la.jsonFormat {
		return la.sendJSON(event)
	}
	return la.sendText(event)
}

// sendJSON JSON 형식으로 로그
func (la *LogAlerter) sendJSON(event *AlertEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	la.logger.Println(string(data))
	return nil
}

// sendText 텍스트 형식으로 로그
func (la *LogAlerter) sendText(event *AlertEvent) error {
	// 기본 정보
	msg := fmt.Sprintf("[%s] %s - %s: %s",
		event.Level,
		event.Timestamp.Format(time.RFC3339),
		event.Title,
		event.Message,
	)

	// 패키지 정보 추가
	if event.PackageInfo != nil {
		pkg := event.PackageInfo
		msg += fmt.Sprintf("\n  Package: %s/%s (version: %s)",
			pkg.Type, pkg.Name, pkg.Version,
		)
		if pkg.ExpectedHash != "" && pkg.ActualHash != "" {
			msg += fmt.Sprintf("\n  Expected Hash: %s", pkg.ExpectedHash)
			msg += fmt.Sprintf("\n  Actual Hash:   %s", pkg.ActualHash)
		}
		if pkg.RemoteURL != "" {
			msg += fmt.Sprintf("\n  Remote URL: %s", pkg.RemoteURL)
		}
	}

	// 메타데이터 추가
	if len(event.Metadata) > 0 {
		msg += "\n  Metadata:"
		for k, v := range event.Metadata {
			msg += fmt.Sprintf("\n    %s: %v", k, v)
		}
	}

	la.logger.Println(msg)
	return nil
}

// SendBatch 여러 알림 일괄 전송
func (la *LogAlerter) SendBatch(ctx context.Context, events []*AlertEvent) error {
	for _, event := range events {
		if err := la.Send(ctx, event); err != nil {
			return err
		}
	}
	return nil
}

// IsEnabled 알림 채널 활성화 여부
func (la *LogAlerter) IsEnabled() bool {
	return la.enabled
}

// Name 알림 채널 이름
func (la *LogAlerter) Name() string {
	return "log"
}

// Close 리소스 정리
func (la *LogAlerter) Close() error {
	la.mutex.Lock()
	defer la.mutex.Unlock()

	if la.file != nil {
		return la.file.Close()
	}
	return nil
}
