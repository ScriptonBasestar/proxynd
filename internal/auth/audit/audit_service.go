// Package audit provides security audit logging functionality
package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"proxynd/logging"
)

// AuditLevel 감사 로그 레벨
type AuditLevel string

const (
	// LevelInfo 정보성 이벤트
	LevelInfo AuditLevel = "info"
	// LevelWarning 경고 이벤트
	LevelWarning AuditLevel = "warning"
	// LevelCritical 중요 이벤트
	LevelCritical AuditLevel = "critical"
	// LevelSecurity 보안 이벤트
	LevelSecurity AuditLevel = "security"
)

// AuditEventType 감사 이벤트 타입
type AuditEventType string

const (
	// Authentication Events
	EventAuthLogin        AuditEventType = "auth.login"
	EventAuthLogout       AuditEventType = "auth.logout"
	EventAuthFailed       AuditEventType = "auth.failed"
	EventAuthMFASetup     AuditEventType = "auth.mfa.setup"
	EventAuthMFAVerify    AuditEventType = "auth.mfa.verify"
	EventAuthMFAFailed    AuditEventType = "auth.mfa.failed"
	EventAuthTokenRefresh AuditEventType = "auth.token.refresh"

	// Authorization Events
	EventAuthzPermission AuditEventType = "authz.permission"
	EventAuthzDenied     AuditEventType = "authz.denied"
	EventAuthzElevation  AuditEventType = "authz.elevation"

	// Security Events
	EventSecurityDDoS         AuditEventType = "security.ddos"
	EventSecurityRateLimit    AuditEventType = "security.rate_limit"
	EventSecurityBlocked      AuditEventType = "security.blocked"
	EventSecurityMalicious    AuditEventType = "security.malicious"
	EventSecurityVulnDetected AuditEventType = "security.vuln_detected"

	// API Key Events
	EventAPIKeyCreated AuditEventType = "api_key.created"
	EventAPIKeyRevoked AuditEventType = "api_key.revoked"
	EventAPIKeyUsed    AuditEventType = "api_key.used"
	EventAPIKeyAbused  AuditEventType = "api_key.abused"

	// Configuration Events
	EventConfigChanged AuditEventType = "config.changed"
	EventConfigReload  AuditEventType = "config.reload"

	// System Events
	EventSystemStartup  AuditEventType = "system.startup"
	EventSystemShutdown AuditEventType = "system.shutdown"
	EventSystemError    AuditEventType = "system.error"

	// Package Management Events
	EventPackageUpload   AuditEventType = "package.upload"
	EventPackageDownload AuditEventType = "package.download"
	EventPackageDeleted  AuditEventType = "package.deleted"
	EventPackageCorrupt  AuditEventType = "package.corrupt"
)

// AuditEvent 감사 이벤트
type AuditEvent struct {
	ID          string                 `json:"id"`
	Timestamp   time.Time              `json:"timestamp"`
	Level       AuditLevel             `json:"level"`
	EventType   AuditEventType         `json:"event_type"`
	Message     string                 `json:"message"`
	UserID      string                 `json:"user_id,omitempty"`
	UserEmail   string                 `json:"user_email,omitempty"`
	SessionID   string                 `json:"session_id,omitempty"`
	ClientIP    string                 `json:"client_ip,omitempty"`
	UserAgent   string                 `json:"user_agent,omitempty"`
	RequestID   string                 `json:"request_id,omitempty"`
	Resource    string                 `json:"resource,omitempty"`
	Action      string                 `json:"action,omitempty"`
	Result      string                 `json:"result,omitempty"`
	Details     map[string]interface{} `json:"details,omitempty"`
	RiskScore   int                    `json:"risk_score,omitempty"`   // 0-100 위험도 점수
	ThreatLevel string                 `json:"threat_level,omitempty"` // low, medium, high, critical
	Source      string                 `json:"source"`                 // 이벤트 발생 소스
	Tags        []string               `json:"tags,omitempty"`
}

// AuditConfig 감사 로그 설정
type AuditConfig struct {
	Enabled          bool          `json:"enabled"`
	FilePath         string        `json:"file_path"`
	MaxFileSize      int64         `json:"max_file_size"`     // 바이트 단위
	MaxFiles         int           `json:"max_files"`         // 로테이션 파일 수
	BufferSize       int           `json:"buffer_size"`       // 버퍼 크기
	FlushInterval    time.Duration `json:"flush_interval"`    // 플러시 간격
	EnableStructured bool          `json:"enable_structured"` // 구조화된 로깅
	EnableSyslog     bool          `json:"enable_syslog"`     // syslog 전송
	MinLevel         AuditLevel    `json:"min_level"`         // 최소 로그 레벨
	IncludeStack     bool          `json:"include_stack"`     // 스택 트레이스 포함
	EnableMetrics    bool          `json:"enable_metrics"`    // 메트릭 수집
	RetentionDays    int           `json:"retention_days"`    // 로그 보관 기간
}

// DefaultAuditConfig 기본 감사 설정
func DefaultAuditConfig() *AuditConfig {
	return &AuditConfig{
		Enabled:          true,
		FilePath:         "logs/audit.log",
		MaxFileSize:      100 * 1024 * 1024, // 100MB
		MaxFiles:         10,
		BufferSize:       1000,
		FlushInterval:    30 * time.Second,
		EnableStructured: true,
		EnableSyslog:     false,
		MinLevel:         LevelInfo,
		IncludeStack:     false,
		EnableMetrics:    true,
		RetentionDays:    90,
	}
}

// AuditService 감사 로그 서비스
type AuditService struct {
	config  *AuditConfig
	logger  logging.Logger
	buffer  chan *AuditEvent
	file    *os.File
	mutex   sync.RWMutex
	metrics *AuditMetrics
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
}

// AuditMetrics 감사 로그 메트릭
type AuditMetrics struct {
	mutex            sync.RWMutex
	TotalEvents      int64                    `json:"total_events"`
	EventsByType     map[AuditEventType]int64 `json:"events_by_type"`
	EventsByLevel    map[AuditLevel]int64     `json:"events_by_level"`
	EventsByUser     map[string]int64         `json:"events_by_user"`
	HighRiskEvents   int64                    `json:"high_risk_events"`
	SecurityEvents   int64                    `json:"security_events"`
	DroppedEvents    int64                    `json:"dropped_events"`
	LastReset        time.Time                `json:"last_reset"`
	ProcessingErrors int64                    `json:"processing_errors"`
}

// NewAuditService 감사 서비스 생성
func NewAuditService(config *AuditConfig) (*AuditService, error) {
	if config == nil {
		config = DefaultAuditConfig()
	}

	if !config.Enabled {
		return &AuditService{config: config}, nil
	}

	ctx, cancel := context.WithCancel(context.Background())

	service := &AuditService{
		config: config,
		logger: logging.GetLogger(),
		buffer: make(chan *AuditEvent, config.BufferSize),
		metrics: &AuditMetrics{
			EventsByType:  make(map[AuditEventType]int64),
			EventsByLevel: make(map[AuditLevel]int64),
			EventsByUser:  make(map[string]int64),
			LastReset:     time.Now(),
		},
		ctx:    ctx,
		cancel: cancel,
	}

	// 로그 파일 초기화
	if err := service.initLogFile(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to initialize audit log file: %w", err)
	}

	// 백그라운드 처리 시작
	service.wg.Add(1)
	go service.processEvents()

	// 정기적인 플러시
	service.wg.Add(1)
	go service.periodicFlush()

	// 로그 로테이션 및 정리
	service.wg.Add(1)
	go service.logMaintenance()

	service.logger.Info("Audit service started",
		logging.F("file_path", config.FilePath),
		logging.F("buffer_size", config.BufferSize))

	return service, nil
}

// LogEvent 이벤트 로깅
func (s *AuditService) LogEvent(eventType AuditEventType, level AuditLevel, message string) *AuditEventBuilder {
	return NewAuditEventBuilder(s, eventType, level, message)
}

// LogSecurityEvent 보안 이벤트 로깅
func (s *AuditService) LogSecurityEvent(eventType AuditEventType, message string) *AuditEventBuilder {
	return s.LogEvent(eventType, LevelSecurity, message).
		WithTag("security").
		WithRiskScore(70)
}

// LogAuthEvent 인증 이벤트 로깅
func (s *AuditService) LogAuthEvent(eventType AuditEventType, userID, userEmail string, success bool) *AuditEventBuilder {
	level := LevelInfo
	if !success {
		level = LevelWarning
	}

	message := fmt.Sprintf("Authentication event: %s", eventType)
	if !success {
		message += " (failed)"
	}

	builder := s.LogEvent(eventType, level, message).
		WithUser(userID, userEmail).
		WithTag("authentication")

	if !success {
		builder = builder.WithRiskScore(60)
	}

	return builder
}

// LogAPIKeyEvent API 키 이벤트 로깅
func (s *AuditService) LogAPIKeyEvent(eventType AuditEventType, keyID, userID, action string) *AuditEventBuilder {
	return s.LogEvent(eventType, LevelInfo, fmt.Sprintf("API key %s: %s", action, keyID)).
		WithUser(userID, "").
		WithDetail("key_id", keyID).
		WithDetail("action", action).
		WithTag("api_key")
}

// writeEvent 이벤트를 버퍼에 추가
func (s *AuditService) writeEvent(event *AuditEvent) {
	if !s.config.Enabled {
		return
	}

	// 최소 레벨 확인
	if !s.shouldLog(event.Level) {
		return
	}

	// 메트릭 업데이트
	s.updateMetrics(event)

	select {
	case s.buffer <- event:
		// 성공적으로 버퍼에 추가
	default:
		// 버퍼가 가득 찬 경우
		s.metrics.mutex.Lock()
		s.metrics.DroppedEvents++
		s.metrics.mutex.Unlock()

		s.logger.Warn("Audit event buffer full, dropping event",
			logging.F("event_type", string(event.EventType)),
			logging.F("event_id", event.ID))
	}
}

// shouldLog 로그 레벨 확인
func (s *AuditService) shouldLog(level AuditLevel) bool {
	levelOrder := map[AuditLevel]int{
		LevelInfo:     1,
		LevelWarning:  2,
		LevelCritical: 3,
		LevelSecurity: 4,
	}

	return levelOrder[level] >= levelOrder[s.config.MinLevel]
}

// updateMetrics 메트릭 업데이트
func (s *AuditService) updateMetrics(event *AuditEvent) {
	if !s.config.EnableMetrics {
		return
	}

	s.metrics.mutex.Lock()
	defer s.metrics.mutex.Unlock()

	s.metrics.TotalEvents++
	s.metrics.EventsByType[event.EventType]++
	s.metrics.EventsByLevel[event.Level]++

	if event.UserID != "" {
		s.metrics.EventsByUser[event.UserID]++
	}

	if event.Level == LevelSecurity {
		s.metrics.SecurityEvents++
	}

	if event.RiskScore >= 70 {
		s.metrics.HighRiskEvents++
	}
}

// processEvents 이벤트 처리 루프
func (s *AuditService) processEvents() {
	defer s.wg.Done()

	for {
		select {
		case event := <-s.buffer:
			if err := s.writeEventToFile(event); err != nil {
				s.metrics.mutex.Lock()
				s.metrics.ProcessingErrors++
				s.metrics.mutex.Unlock()

				s.logger.Error("Failed to write audit event",
					logging.F("error", err),
					logging.F("event_id", event.ID))
			}

		case <-s.ctx.Done():
			// 남은 이벤트 처리
			for {
				select {
				case event := <-s.buffer:
					s.writeEventToFile(event)
				default:
					return
				}
			}
		}
	}
}

// writeEventToFile 파일에 이벤트 기록
func (s *AuditService) writeEventToFile(event *AuditEvent) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.file == nil {
		return fmt.Errorf("audit log file not open")
	}

	var data []byte
	var err error

	if s.config.EnableStructured {
		data, err = json.Marshal(event)
		if err != nil {
			return fmt.Errorf("failed to marshal event: %w", err)
		}
		data = append(data, '\n')
	} else {
		// 플레인 텍스트 형식
		logLine := fmt.Sprintf("[%s] %s %s: %s",
			event.Timestamp.Format(time.RFC3339),
			event.Level,
			event.EventType,
			event.Message)

		if event.UserID != "" {
			logLine += fmt.Sprintf(" (user: %s)", event.UserID)
		}
		if event.ClientIP != "" {
			logLine += fmt.Sprintf(" (ip: %s)", event.ClientIP)
		}

		data = []byte(logLine + "\n")
	}

	_, err = s.file.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write to audit log: %w", err)
	}

	// 파일 크기 확인 및 로테이션
	if info, err := s.file.Stat(); err == nil && info.Size() > s.config.MaxFileSize {
		go s.rotateLogFile()
	}

	return nil
}

// periodicFlush 정기적 플러시
func (s *AuditService) periodicFlush() {
	defer s.wg.Done()

	ticker := time.NewTicker(s.config.FlushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.flush()
		case <-s.ctx.Done():
			s.flush()
			return
		}
	}
}

// flush 버퍼 플러시
func (s *AuditService) flush() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.file != nil {
		s.file.Sync()
	}
}

// initLogFile 로그 파일 초기화
func (s *AuditService) initLogFile() error {
	// 디렉토리 생성
	dir := filepath.Dir(s.config.FilePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	// 파일 열기
	file, err := os.OpenFile(s.config.FilePath,
		os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("failed to open audit log file: %w", err)
	}

	s.file = file
	return nil
}

// rotateLogFile 로그 파일 로테이션
func (s *AuditService) rotateLogFile() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.file == nil {
		return
	}

	// 현재 파일 닫기
	s.file.Close()

	// 파일명 변경 (타임스탬프 추가)
	timestamp := time.Now().Format("20060102-150405")
	rotatedPath := fmt.Sprintf("%s.%s", s.config.FilePath, timestamp)

	if err := os.Rename(s.config.FilePath, rotatedPath); err != nil {
		s.logger.Error("Failed to rotate audit log",
			logging.F("error", err))
		return
	}

	// 새 파일 생성
	if err := s.initLogFile(); err != nil {
		s.logger.Error("Failed to create new audit log file",
			logging.F("error", err))
		return
	}

	s.logger.Info("Audit log file rotated",
		logging.F("rotated_file", rotatedPath))

	// 오래된 파일 정리
	go s.cleanupOldFiles()
}

// logMaintenance 로그 유지보수
func (s *AuditService) logMaintenance() {
	defer s.wg.Done()

	ticker := time.NewTicker(24 * time.Hour) // 매일 실행
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.cleanupOldFiles()
		case <-s.ctx.Done():
			return
		}
	}
}

// cleanupOldFiles 오래된 파일 정리
func (s *AuditService) cleanupOldFiles() {
	dir := filepath.Dir(s.config.FilePath)
	basename := filepath.Base(s.config.FilePath)

	entries, err := os.ReadDir(dir)
	if err != nil {
		s.logger.Error("Failed to read audit log directory",
			logging.F("error", err))
		return
	}

	var logFiles []os.DirEntry
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), basename+".") {
			logFiles = append(logFiles, entry)
		}
	}

	// 파일 수 제한
	if len(logFiles) > s.config.MaxFiles {
		// 가장 오래된 파일들 삭제
		for i := 0; i < len(logFiles)-s.config.MaxFiles; i++ {
			filePath := filepath.Join(dir, logFiles[i].Name())
			if err := os.Remove(filePath); err != nil {
				s.logger.Warn("Failed to remove old audit log",
					logging.F("file", filePath),
					logging.F("error", err))
			}
		}
	}

	// 보관 기간 확인 및 정리
	cutoff := time.Now().AddDate(0, 0, -s.config.RetentionDays)
	for _, entry := range logFiles {
		info, err := entry.Info()
		if err != nil {
			continue
		}

		if info.ModTime().Before(cutoff) {
			filePath := filepath.Join(dir, entry.Name())
			if err := os.Remove(filePath); err != nil {
				s.logger.Warn("Failed to remove expired audit log",
					logging.F("file", filePath),
					logging.F("error", err))
			} else {
				s.logger.Info("Removed expired audit log",
					logging.F("file", filePath))
			}
		}
	}
}

// GetMetrics 메트릭 조회
func (s *AuditService) GetMetrics() *AuditMetrics {
	if !s.config.EnableMetrics {
		return nil
	}

	s.metrics.mutex.RLock()
	defer s.metrics.mutex.RUnlock()

	// 복사본 반환
	metrics := &AuditMetrics{
		TotalEvents:      s.metrics.TotalEvents,
		EventsByType:     make(map[AuditEventType]int64),
		EventsByLevel:    make(map[AuditLevel]int64),
		EventsByUser:     make(map[string]int64),
		HighRiskEvents:   s.metrics.HighRiskEvents,
		SecurityEvents:   s.metrics.SecurityEvents,
		DroppedEvents:    s.metrics.DroppedEvents,
		LastReset:        s.metrics.LastReset,
		ProcessingErrors: s.metrics.ProcessingErrors,
	}

	for k, v := range s.metrics.EventsByType {
		metrics.EventsByType[k] = v
	}
	for k, v := range s.metrics.EventsByLevel {
		metrics.EventsByLevel[k] = v
	}
	for k, v := range s.metrics.EventsByUser {
		metrics.EventsByUser[k] = v
	}

	return metrics
}

// ResetMetrics 메트릭 초기화
func (s *AuditService) ResetMetrics() {
	if !s.config.EnableMetrics {
		return
	}

	s.metrics.mutex.Lock()
	defer s.metrics.mutex.Unlock()

	s.metrics.TotalEvents = 0
	s.metrics.EventsByType = make(map[AuditEventType]int64)
	s.metrics.EventsByLevel = make(map[AuditLevel]int64)
	s.metrics.EventsByUser = make(map[string]int64)
	s.metrics.HighRiskEvents = 0
	s.metrics.SecurityEvents = 0
	s.metrics.DroppedEvents = 0
	s.metrics.ProcessingErrors = 0
	s.metrics.LastReset = time.Now()

	s.logger.Info("Audit metrics reset")
}

// Close 서비스 종료
func (s *AuditService) Close() error {
	if !s.config.Enabled {
		return nil
	}

	s.cancel()
	s.wg.Wait()

	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.file != nil {
		s.file.Sync()
		s.file.Close()
		s.file = nil
	}

	close(s.buffer)

	s.logger.Info("Audit service stopped")
	return nil
}
