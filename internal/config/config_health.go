package config

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"proxynd/logging"
)

// ConfigHealthMonitor 설정 상태 모니터링 및 자가 치유 시스템
type ConfigHealthMonitor struct {
	mu                sync.RWMutex
	logger            logging.Logger
	validator         *SchemaValidator
	backupManager     *ConfigBackupManager
	healthCheckers    map[string]HealthChecker
	autoHealHandlers  map[string]AutoHealHandler
	monitoringEnabled bool
	checkInterval     time.Duration
	alertThreshold    int
	errorCounts       map[string]int
	lastHealthStatus  *HealthStatus
	ctx               context.Context
	cancel            context.CancelFunc
	wg                sync.WaitGroup
}

// HealthChecker 헬스 체크 인터페이스
type HealthChecker interface {
	Check(config *UnifiedConfig) HealthCheckResult
	Name() string
	Priority() int // 낮을수록 높은 우선순위
}

// AutoHealHandler 자동 복구 핸들러 인터페이스
type AutoHealHandler interface {
	CanHeal(issue HealthIssue) bool
	Heal(config *UnifiedConfig, issue HealthIssue) (*UnifiedConfig, error)
	Name() string
}

// HealthCheckResult 헬스 체크 결과
type HealthCheckResult struct {
	Healthy     bool          `json:"healthy"`
	Issues      []HealthIssue `json:"issues"`
	CheckerName string        `json:"checker_name"`
	Duration    time.Duration `json:"duration"`
	Timestamp   time.Time     `json:"timestamp"`
}

// HealthIssue 헬스 이슈
type HealthIssue struct {
	Type         string      `json:"type"`          // error, warning, info
	Category     string      `json:"category"`      // connectivity, validation, performance, security
	Field        string      `json:"field"`         // 관련 설정 필드
	Message      string      `json:"message"`       // 이슈 설명
	Suggestion   string      `json:"suggestion"`    // 해결 방안
	AutoHealable bool        `json:"auto_healable"` // 자동 복구 가능 여부
	Severity     int         `json:"severity"`      // 1(낮음) ~ 5(높음)
	Details      interface{} `json:"details"`       // 추가 세부사항
}

// HealthStatus 전체 헬스 상태
type HealthStatus struct {
	Overall    string                       `json:"overall"` // healthy, degraded, unhealthy
	Score      int                          `json:"score"`   // 0-100
	LastCheck  time.Time                    `json:"last_check"`
	Duration   time.Duration                `json:"duration"`
	Results    map[string]HealthCheckResult `json:"results"`
	Summary    HealthSummary                `json:"summary"`
	AutoHealed []AutoHealResult             `json:"auto_healed"`
	Trends     map[string][]HealthDataPoint `json:"trends"`
}

// HealthSummary 헬스 요약
type HealthSummary struct {
	TotalChecks     int `json:"total_checks"`
	PassedChecks    int `json:"passed_checks"`
	FailedChecks    int `json:"failed_checks"`
	TotalIssues     int `json:"total_issues"`
	ErrorIssues     int `json:"error_issues"`
	WarningIssues   int `json:"warning_issues"`
	AutoHealedCount int `json:"auto_healed_count"`
}

// AutoHealResult 자동 복구 결과
type AutoHealResult struct {
	Issue       HealthIssue `json:"issue"`
	HandlerName string      `json:"handler_name"`
	Success     bool        `json:"success"`
	Error       string      `json:"error,omitempty"`
	Timestamp   time.Time   `json:"timestamp"`
}

// HealthDataPoint 헬스 데이터 포인트 (트렌드 분석용)
type HealthDataPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Score     int       `json:"score"`
	Issues    int       `json:"issues"`
}

// ConfigBackupManager 설정 백업 관리자
type ConfigBackupManager struct {
	backupDir          string
	maxBackups         int
	compressionEnabled bool
	retentionPeriod    time.Duration
	mu                 sync.Mutex
}

// NewConfigHealthMonitor 새 설정 헬스 모니터 생성
func NewConfigHealthMonitor(logger logging.Logger, validator *SchemaValidator) *ConfigHealthMonitor {
	ctx, cancel := context.WithCancel(context.Background())

	monitor := &ConfigHealthMonitor{
		logger:            logger,
		validator:         validator,
		healthCheckers:    make(map[string]HealthChecker),
		autoHealHandlers:  make(map[string]AutoHealHandler),
		monitoringEnabled: true,
		checkInterval:     5 * time.Minute,
		alertThreshold:    3,
		errorCounts:       make(map[string]int),
		ctx:               ctx,
		cancel:            cancel,
	}

	// 백업 매니저 초기화
	monitor.backupManager = &ConfigBackupManager{
		backupDir:          filepath.Join(os.TempDir(), "proxynd-config-backups"),
		maxBackups:         10,
		compressionEnabled: true,
		retentionPeriod:    7 * 24 * time.Hour, // 7일
	}

	// 기본 헬스 체커 등록
	monitor.registerDefaultHealthCheckers()
	// 기본 자동 복구 핸들러 등록
	monitor.registerDefaultAutoHealHandlers()

	return monitor
}

// registerDefaultHealthCheckers 기본 헬스 체커 등록
func (chm *ConfigHealthMonitor) registerDefaultHealthCheckers() {
	// 설정 검증 체커
	chm.RegisterHealthChecker(&ValidationHealthChecker{validator: chm.validator})

	// 연결성 체커
	chm.RegisterHealthChecker(&ConnectivityHealthChecker{})

	// 디스크 공간 체커
	chm.RegisterHealthChecker(&DiskSpaceHealthChecker{})

	// 권한 체커
	chm.RegisterHealthChecker(&PermissionHealthChecker{})

	// 성능 체커
	chm.RegisterHealthChecker(&PerformanceHealthChecker{})

	// 보안 체커
	chm.RegisterHealthChecker(&SecurityHealthChecker{})
}

// registerDefaultAutoHealHandlers 기본 자동 복구 핸들러 등록
func (chm *ConfigHealthMonitor) registerDefaultAutoHealHandlers() {
	// 디렉토리 생성 핸들러
	chm.RegisterAutoHealHandler(&DirectoryCreationHandler{})

	// 권한 수정 핸들러
	chm.RegisterAutoHealHandler(&PermissionFixHandler{})

	// 기본값 적용 핸들러
	chm.RegisterAutoHealHandler(&DefaultValueHandler{})

	// 캐시 정리 핸들러
	chm.RegisterAutoHealHandler(&CacheCleanupHandler{})
}

// RegisterHealthChecker 헬스 체커 등록
func (chm *ConfigHealthMonitor) RegisterHealthChecker(checker HealthChecker) {
	chm.mu.Lock()
	defer chm.mu.Unlock()
	chm.healthCheckers[checker.Name()] = checker
	chm.logger.Info("Health checker registered", logging.F("name", checker.Name()))
}

// RegisterAutoHealHandler 자동 복구 핸들러 등록
func (chm *ConfigHealthMonitor) RegisterAutoHealHandler(handler AutoHealHandler) {
	chm.mu.Lock()
	defer chm.mu.Unlock()
	chm.autoHealHandlers[handler.Name()] = handler
	chm.logger.Info("Auto heal handler registered", logging.F("name", handler.Name()))
}

// Start 헬스 모니터링 시작
func (chm *ConfigHealthMonitor) Start() error {
	chm.mu.Lock()
	defer chm.mu.Unlock()

	if !chm.monitoringEnabled {
		return fmt.Errorf("monitoring is disabled")
	}

	// 백업 디렉토리 생성
	if err := os.MkdirAll(chm.backupManager.backupDir, 0o755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	chm.wg.Add(1)
	go chm.monitoringLoop()

	chm.logger.Info("Configuration health monitoring started",
		logging.F("interval", chm.checkInterval),
		logging.F("checkers", len(chm.healthCheckers)),
		logging.F("healers", len(chm.autoHealHandlers)))

	return nil
}

// Stop 헬스 모니터링 중지
func (chm *ConfigHealthMonitor) Stop() error {
	chm.cancel()
	chm.wg.Wait()
	chm.logger.Info("Configuration health monitoring stopped")
	return nil
}

// monitoringLoop 모니터링 루프
func (chm *ConfigHealthMonitor) monitoringLoop() {
	defer chm.wg.Done()

	ticker := time.NewTicker(chm.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-chm.ctx.Done():
			return
		case <-ticker.C:
			// 현재 설정 가져오기 (실제 구현에서는 컨테이너에서 가져와야 함)
			// 여기서는 기본 설정으로 테스트
			config := &UnifiedConfig{}
			status := chm.CheckHealth(config)
			chm.handleHealthStatus(config, status)
		}
	}
}

// CheckHealth 설정 헬스 체크 수행
func (chm *ConfigHealthMonitor) CheckHealth(config *UnifiedConfig) *HealthStatus {
	startTime := time.Now()

	chm.mu.Lock()
	checkers := make([]HealthChecker, 0, len(chm.healthCheckers))
	for _, checker := range chm.healthCheckers {
		checkers = append(checkers, checker)
	}
	chm.mu.Unlock()

	status := &HealthStatus{
		LastCheck:  startTime,
		Results:    make(map[string]HealthCheckResult),
		AutoHealed: make([]AutoHealResult, 0),
		Trends:     make(map[string][]HealthDataPoint),
	}

	// 모든 헬스 체커 실행
	var wg sync.WaitGroup
	resultsChan := make(chan HealthCheckResult, len(checkers))

	for _, checker := range checkers {
		wg.Add(1)
		go func(c HealthChecker) {
			defer wg.Done()
			checkStart := time.Now()
			result := c.Check(config)
			result.Duration = time.Since(checkStart)
			result.Timestamp = time.Now()
			resultsChan <- result
		}(checker)
	}

	wg.Wait()
	close(resultsChan)

	// 결과 수집
	allIssues := make([]HealthIssue, 0)
	for result := range resultsChan {
		status.Results[result.CheckerName] = result
		allIssues = append(allIssues, result.Issues...)
	}

	// 자동 복구 시도
	healedIssues := chm.performAutoHealing(config, allIssues)
	status.AutoHealed = healedIssues

	// 상태 계산
	status.Duration = time.Since(startTime)
	chm.calculateOverallStatus(status)

	chm.mu.Lock()
	chm.lastHealthStatus = status
	chm.mu.Unlock()

	return status
}

// performAutoHealing 자동 복구 수행
func (chm *ConfigHealthMonitor) performAutoHealing(config *UnifiedConfig, issues []HealthIssue) []AutoHealResult {
	results := make([]AutoHealResult, 0)

	chm.mu.RLock()
	handlers := make([]AutoHealHandler, 0, len(chm.autoHealHandlers))
	for _, handler := range chm.autoHealHandlers {
		handlers = append(handlers, handler)
	}
	chm.mu.RUnlock()

	for _, issue := range issues {
		if !issue.AutoHealable || issue.Severity < 3 {
			continue // 자동 복구 불가하거나 심각도가 낮은 경우 건너뛰기
		}

		for _, handler := range handlers {
			if handler.CanHeal(issue) {
				result := AutoHealResult{
					Issue:       issue,
					HandlerName: handler.Name(),
					Timestamp:   time.Now(),
				}

				healedConfig, err := handler.Heal(config, issue)
				if err != nil {
					result.Success = false
					result.Error = err.Error()
					chm.logger.Error("Auto healing failed",
						logging.F("handler", handler.Name()),
						logging.F("issue", issue.Message),
						logging.F("error", err))
				} else {
					result.Success = true
					config = healedConfig // 복구된 설정으로 업데이트
					chm.logger.Info("Auto healing succeeded",
						logging.F("handler", handler.Name()),
						logging.F("issue", issue.Message))
				}

				results = append(results, result)
				break // 하나의 핸들러가 처리했으면 다음 이슈로
			}
		}
	}

	return results
}

// calculateOverallStatus 전체 상태 계산
func (chm *ConfigHealthMonitor) calculateOverallStatus(status *HealthStatus) {
	totalScore := 0
	totalChecks := len(status.Results)
	passedChecks := 0
	totalIssues := 0
	errorIssues := 0
	warningIssues := 0

	for _, result := range status.Results {
		if result.Healthy {
			passedChecks++
			totalScore += 100
		} else {
			// 이슈 심각도에 따라 점수 감점
			issueScore := 100
			for _, issue := range result.Issues {
				totalIssues++
				switch issue.Type {
				case "error":
					errorIssues++
					issueScore -= issue.Severity * 10
				case "warning":
					warningIssues++
					issueScore -= issue.Severity * 5
				}
			}
			if issueScore < 0 {
				issueScore = 0
			}
			totalScore += issueScore
		}
	}

	// 평균 점수 계산
	averageScore := 0
	if totalChecks > 0 {
		averageScore = totalScore / totalChecks
	}

	// 전체 상태 결정
	overall := "healthy"
	if averageScore < 50 {
		overall = "unhealthy"
	} else if averageScore < 80 {
		overall = "degraded"
	}

	status.Overall = overall
	status.Score = averageScore
	status.Summary = HealthSummary{
		TotalChecks:     totalChecks,
		PassedChecks:    passedChecks,
		FailedChecks:    totalChecks - passedChecks,
		TotalIssues:     totalIssues,
		ErrorIssues:     errorIssues,
		WarningIssues:   warningIssues,
		AutoHealedCount: len(status.AutoHealed),
	}
}

// handleHealthStatus 헬스 상태 처리
func (chm *ConfigHealthMonitor) handleHealthStatus(config *UnifiedConfig, status *HealthStatus) {
	// 상태가 심각한 경우 백업 생성
	if status.Overall == "unhealthy" || len(status.AutoHealed) > 0 {
		if err := chm.backupManager.CreateBackup(config); err != nil {
			chm.logger.Error("Failed to create config backup", logging.F("error", err))
		}
	}

	// 알림 처리
	chm.handleAlerting(status)

	// 트렌드 데이터 업데이트
	chm.updateTrends(status)
}

// handleAlerting 알림 처리
func (chm *ConfigHealthMonitor) handleAlerting(status *HealthStatus) {
	if status.Overall == "unhealthy" {
		chm.mu.Lock()
		chm.errorCounts["overall"]++
		errorCount := chm.errorCounts["overall"]
		chm.mu.Unlock()

		if errorCount >= chm.alertThreshold {
			chm.logger.Error("Configuration health alert triggered",
				logging.F("status", status.Overall),
				logging.F("score", status.Score),
				logging.F("error_count", errorCount),
				logging.F("total_issues", status.Summary.TotalIssues))

			// 실제 구현에서는 여기서 알림 시스템에 알림 전송
			// webhooks, Slack, email 등
		}
	} else {
		// 상태가 정상이면 에러 카운트 리셋
		chm.mu.Lock()
		chm.errorCounts["overall"] = 0
		chm.mu.Unlock()
	}
}

// updateTrends 트렌드 데이터 업데이트
func (chm *ConfigHealthMonitor) updateTrends(status *HealthStatus) {
	dataPoint := HealthDataPoint{
		Timestamp: status.LastCheck,
		Score:     status.Score,
		Issues:    status.Summary.TotalIssues,
	}

	chm.mu.Lock()
	if chm.lastHealthStatus != nil {
		if chm.lastHealthStatus.Trends == nil {
			chm.lastHealthStatus.Trends = make(map[string][]HealthDataPoint)
		}

		// 최근 24시간 데이터만 유지
		cutoff := time.Now().Add(-24 * time.Hour)
		trends := chm.lastHealthStatus.Trends["overall"]

		// 오래된 데이터 제거
		var filteredTrends []HealthDataPoint
		for _, point := range trends {
			if point.Timestamp.After(cutoff) {
				filteredTrends = append(filteredTrends, point)
			}
		}

		// 새 데이터 추가
		filteredTrends = append(filteredTrends, dataPoint)
		status.Trends["overall"] = filteredTrends
	}
	chm.mu.Unlock()
}

// GetHealthStatus 현재 헬스 상태 반환
func (chm *ConfigHealthMonitor) GetHealthStatus() *HealthStatus {
	chm.mu.RLock()
	defer chm.mu.RUnlock()
	return chm.lastHealthStatus
}

// CreateBackup 설정 백업 생성
func (cbm *ConfigBackupManager) CreateBackup(config *UnifiedConfig) error {
	cbm.mu.Lock()
	defer cbm.mu.Unlock()

	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("config_backup_%s.json", timestamp)
	backupPath := filepath.Join(cbm.backupDir, filename)

	// 설정을 JSON으로 직렬화
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// 백업 파일 작성
	if err := os.WriteFile(backupPath, data, 0o600); err != nil {
		return fmt.Errorf("failed to write backup file: %w", err)
	}

	// 오래된 백업 정리
	if err := cbm.cleanupOldBackups(); err != nil {
		return fmt.Errorf("failed to cleanup old backups: %w", err)
	}

	return nil
}

// cleanupOldBackups 오래된 백업 정리
func (cbm *ConfigBackupManager) cleanupOldBackups() error {
	files, err := os.ReadDir(cbm.backupDir)
	if err != nil {
		return err
	}

	// 백업 파일만 필터링하고 시간순 정렬
	backupFiles := make([]os.FileInfo, 0)
	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".json" {
			info, err := file.Info()
			if err == nil {
				backupFiles = append(backupFiles, info)
			}
		}
	}

	// 최대 백업 수 초과 시 오래된 파일 삭제
	if len(backupFiles) > cbm.maxBackups {
		// 가장 오래된 파일들 삭제
		for i := 0; i < len(backupFiles)-cbm.maxBackups; i++ {
			oldPath := filepath.Join(cbm.backupDir, backupFiles[i].Name())
			if err := os.Remove(oldPath); err != nil {
				return err
			}
		}
	}

	// 보존 기간 초과 파일 삭제
	cutoff := time.Now().Add(-cbm.retentionPeriod)
	for _, file := range backupFiles {
		if file.ModTime().Before(cutoff) {
			oldPath := filepath.Join(cbm.backupDir, file.Name())
			if err := os.Remove(oldPath); err != nil {
				return err
			}
		}
	}

	return nil
}

// RestoreBackup 백업에서 설정 복원
func (cbm *ConfigBackupManager) RestoreBackup(backupName string) (*UnifiedConfig, error) {
	cbm.mu.Lock()
	defer cbm.mu.Unlock()

	backupPath := filepath.Join(cbm.backupDir, backupName)

	data, err := os.ReadFile(backupPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read backup file: %w", err)
	}

	var config UnifiedConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal backup: %w", err)
	}

	return &config, nil
}

// ListBackups 백업 목록 반환
func (cbm *ConfigBackupManager) ListBackups() ([]string, error) {
	files, err := os.ReadDir(cbm.backupDir)
	if err != nil {
		return nil, err
	}

	backups := make([]string, 0)
	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".json" {
			backups = append(backups, file.Name())
		}
	}

	return backups, nil
}

// 기본 헬스 체커 구현들

// ValidationHealthChecker 설정 검증 체커
type ValidationHealthChecker struct {
	validator *SchemaValidator
}

func (v *ValidationHealthChecker) Name() string  { return "validation" }
func (v *ValidationHealthChecker) Priority() int { return 1 }

func (v *ValidationHealthChecker) Check(config *UnifiedConfig) HealthCheckResult {
	result := HealthCheckResult{
		CheckerName: v.Name(),
		Healthy:     true,
		Issues:      make([]HealthIssue, 0),
	}

	if v.validator != nil {
		validationResult := v.validator.Validate(config)

		// 검증 에러를 헬스 이슈로 변환
		for _, err := range validationResult.Errors {
			result.Issues = append(result.Issues, HealthIssue{
				Type:         "error",
				Category:     "validation",
				Field:        err.Field,
				Message:      err.Message,
				AutoHealable: false,
				Severity:     4,
			})
			result.Healthy = false
		}

		// 검증 경고를 헬스 이슈로 변환
		for _, warning := range validationResult.Warnings {
			severity := 2
			if warning.Severity == "high" {
				severity = 3
			} else if warning.Severity == "low" {
				severity = 1
			}

			result.Issues = append(result.Issues, HealthIssue{
				Type:         "warning",
				Category:     warning.Category,
				Field:        warning.Field,
				Message:      warning.Message,
				Suggestion:   warning.Suggestion,
				AutoHealable: warning.Category == "deprecated",
				Severity:     severity,
			})
		}
	}

	return result
}

// ConnectivityHealthChecker 연결성 체커
type ConnectivityHealthChecker struct{}

func (c *ConnectivityHealthChecker) Name() string  { return "connectivity" }
func (c *ConnectivityHealthChecker) Priority() int { return 2 }

func (c *ConnectivityHealthChecker) Check(config *UnifiedConfig) HealthCheckResult {
	result := HealthCheckResult{
		CheckerName: c.Name(),
		Healthy:     true,
		Issues:      make([]HealthIssue, 0),
	}

	// 업스트림 연결 확인 (간단한 예시)
	if config.Registries.NPM.Enabled {
		if err := c.checkURL(config.Registries.NPM.Upstream); err != nil {
			result.Issues = append(result.Issues, HealthIssue{
				Type:         "error",
				Category:     "connectivity",
				Field:        "registries.npm.upstream",
				Message:      fmt.Sprintf("NPM 업스트림 연결 실패: %v", err),
				AutoHealable: false,
				Severity:     3,
			})
			result.Healthy = false
		}
	}

	// Redis 연결 확인 (캐시 백엔드가 Redis인 경우)
	if config.Cache.Backend == "redis" && config.Cache.Redis.Address != "" {
		// 실제 구현에서는 Redis 연결 테스트
		// 여기서는 간단한 예시만
	}

	return result
}

func (c *ConnectivityHealthChecker) checkURL(url string) error {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	return nil
}

// DiskSpaceHealthChecker 디스크 공간 체커
type DiskSpaceHealthChecker struct{}

func (d *DiskSpaceHealthChecker) Name() string  { return "disk_space" }
func (d *DiskSpaceHealthChecker) Priority() int { return 3 }

func (d *DiskSpaceHealthChecker) Check(config *UnifiedConfig) HealthCheckResult {
	result := HealthCheckResult{
		CheckerName: d.Name(),
		Healthy:     true,
		Issues:      make([]HealthIssue, 0),
	}

	// 캐시 디렉토리 공간 확인
	if config.Cache.Backend == "file" && config.Cache.File.Directory != "" {
		// 실제 구현에서는 디스크 사용량 확인
		// 여기서는 간단한 예시
		if _, err := os.Stat(config.Cache.File.Directory); os.IsNotExist(err) {
			result.Issues = append(result.Issues, HealthIssue{
				Type:         "error",
				Category:     "filesystem",
				Field:        "cache.file.directory",
				Message:      "캐시 디렉토리가 존재하지 않음",
				Suggestion:   "디렉토리를 생성하거나 올바른 경로로 수정하세요",
				AutoHealable: true,
				Severity:     4,
			})
			result.Healthy = false
		}
	}

	return result
}

// PermissionHealthChecker 권한 체커
type PermissionHealthChecker struct{}

func (p *PermissionHealthChecker) Name() string  { return "permissions" }
func (p *PermissionHealthChecker) Priority() int { return 3 }

func (p *PermissionHealthChecker) Check(config *UnifiedConfig) HealthCheckResult {
	result := HealthCheckResult{
		CheckerName: p.Name(),
		Healthy:     true,
		Issues:      make([]HealthIssue, 0),
	}

	// TLS 파일 권한 확인
	if config.Server.TLS.Enabled {
		if err := p.checkFilePermissions(config.Server.TLS.CertFile); err != nil {
			result.Issues = append(result.Issues, HealthIssue{
				Type:         "warning",
				Category:     "security",
				Field:        "server.tls.cert_file",
				Message:      fmt.Sprintf("TLS 인증서 파일 권한 문제: %v", err),
				AutoHealable: true,
				Severity:     3,
			})
		}

		if err := p.checkFilePermissions(config.Server.TLS.KeyFile); err != nil {
			result.Issues = append(result.Issues, HealthIssue{
				Type:         "error",
				Category:     "security",
				Field:        "server.tls.key_file",
				Message:      fmt.Sprintf("TLS 키 파일 권한 문제: %v", err),
				AutoHealable: true,
				Severity:     4,
			})
			result.Healthy = false
		}
	}

	return result
}

func (p *PermissionHealthChecker) checkFilePermissions(filePath string) error {
	if filePath == "" {
		return nil
	}

	info, err := os.Stat(filePath)
	if err != nil {
		return err
	}

	// 키 파일은 소유자만 읽을 수 있어야 함
	if info.Mode().Perm() > 0o600 {
		return fmt.Errorf("파일이 너무 개방적인 권한을 가짐: %o", info.Mode().Perm())
	}

	return nil
}

// PerformanceHealthChecker 성능 체커
type PerformanceHealthChecker struct{}

func (p *PerformanceHealthChecker) Name() string  { return "performance" }
func (p *PerformanceHealthChecker) Priority() int { return 4 }

func (p *PerformanceHealthChecker) Check(config *UnifiedConfig) HealthCheckResult {
	result := HealthCheckResult{
		CheckerName: p.Name(),
		Healthy:     true,
		Issues:      make([]HealthIssue, 0),
	}

	// 성능 설정 확인
	if config.Advanced.Performance.MaxConnections > 50000 {
		result.Issues = append(result.Issues, HealthIssue{
			Type:         "warning",
			Category:     "performance",
			Field:        "advanced.performance.max_connections",
			Message:      "매우 높은 최대 연결 수 설정",
			Suggestion:   "시스템 리소스에 맞는 적절한 값으로 조정하세요",
			AutoHealable: false,
			Severity:     2,
		})
	}

	// 타임아웃 설정 확인
	if config.Server.ReadTimeout > 300*time.Second {
		result.Issues = append(result.Issues, HealthIssue{
			Type:         "warning",
			Category:     "performance",
			Field:        "server.read_timeout",
			Message:      "읽기 타임아웃이 너무 김",
			Suggestion:   "더 짧은 타임아웃을 설정하여 리소스 효율성을 높이세요",
			AutoHealable: false,
			Severity:     1,
		})
	}

	return result
}

// SecurityHealthChecker 보안 체커
type SecurityHealthChecker struct{}

func (s *SecurityHealthChecker) Name() string  { return "security" }
func (s *SecurityHealthChecker) Priority() int { return 2 }

func (s *SecurityHealthChecker) Check(config *UnifiedConfig) HealthCheckResult {
	result := HealthCheckResult{
		CheckerName: s.Name(),
		Healthy:     true,
		Issues:      make([]HealthIssue, 0),
	}

	// TLS 활성화 확인
	if !config.Server.TLS.Enabled {
		result.Issues = append(result.Issues, HealthIssue{
			Type:         "warning",
			Category:     "security",
			Field:        "server.tls.enabled",
			Message:      "TLS가 비활성화되어 있음",
			Suggestion:   "프로덕션 환경에서는 TLS를 활성화하세요",
			AutoHealable: false,
			Severity:     3,
		})
	}

	// 인증 확인
	if config.Security.Authentication.BasicAuth.Enabled == nil || !*config.Security.Authentication.BasicAuth.Enabled {
		result.Issues = append(result.Issues, HealthIssue{
			Type:         "warning",
			Category:     "security",
			Field:        "security.authentication.basic_auth.enabled",
			Message:      "인증이 비활성화되어 있음",
			Suggestion:   "인증을 활성화하여 보안을 강화하세요",
			AutoHealable: false,
			Severity:     4,
		})
	}

	// 최소 TLS 버전 확인
	if config.Server.TLS.Enabled && config.Server.TLS.MinVersion < "TLS1.2" {
		result.Issues = append(result.Issues, HealthIssue{
			Type:         "error",
			Category:     "security",
			Field:        "server.tls.min_version",
			Message:      "안전하지 않은 TLS 버전",
			Suggestion:   "TLS 1.2 이상을 사용하세요",
			AutoHealable: true,
			Severity:     4,
		})
		result.Healthy = false
	}

	return result
}

// 기본 자동 복구 핸들러 구현들

// DirectoryCreationHandler 디렉토리 생성 핸들러
type DirectoryCreationHandler struct{}

func (d *DirectoryCreationHandler) Name() string { return "directory_creation" }

func (d *DirectoryCreationHandler) CanHeal(issue HealthIssue) bool {
	return issue.Category == "filesystem" &&
		strings.Contains(issue.Message, "디렉토리가 존재하지 않음")
}

func (d *DirectoryCreationHandler) Heal(config *UnifiedConfig, issue HealthIssue) (*UnifiedConfig, error) {
	if issue.Field == "cache.file.directory" {
		if err := os.MkdirAll(config.Cache.File.Directory, 0o755); err != nil {
			return nil, fmt.Errorf("디렉토리 생성 실패: %w", err)
		}
	}
	return config, nil
}

// PermissionFixHandler 권한 수정 핸들러
type PermissionFixHandler struct{}

func (p *PermissionFixHandler) Name() string { return "permission_fix" }

func (p *PermissionFixHandler) CanHeal(issue HealthIssue) bool {
	return issue.Category == "security" &&
		strings.Contains(issue.Message, "권한 문제")
}

func (p *PermissionFixHandler) Heal(config *UnifiedConfig, issue HealthIssue) (*UnifiedConfig, error) {
	if issue.Field == "server.tls.key_file" {
		if err := os.Chmod(config.Server.TLS.KeyFile, 0o600); err != nil {
			return nil, fmt.Errorf("파일 권한 수정 실패: %w", err)
		}
	}
	return config, nil
}

// DefaultValueHandler 기본값 적용 핸들러
type DefaultValueHandler struct{}

func (d *DefaultValueHandler) Name() string { return "default_value" }

func (d *DefaultValueHandler) CanHeal(issue HealthIssue) bool {
	return issue.Category == "validation" &&
		strings.Contains(issue.Message, "필수 필드가 누락됨")
}

func (d *DefaultValueHandler) Heal(config *UnifiedConfig, issue HealthIssue) (*UnifiedConfig, error) {
	// 실제 구현에서는 필드별로 적절한 기본값 적용
	// 여기서는 간단한 예시만
	return config, nil
}

// CacheCleanupHandler 캐시 정리 핸들러
type CacheCleanupHandler struct{}

func (c *CacheCleanupHandler) Name() string { return "cache_cleanup" }

func (c *CacheCleanupHandler) CanHeal(issue HealthIssue) bool {
	return issue.Category == "performance" &&
		strings.Contains(issue.Message, "캐시")
}

func (c *CacheCleanupHandler) Heal(config *UnifiedConfig, issue HealthIssue) (*UnifiedConfig, error) {
	// 실제 구현에서는 캐시 정리 로직
	return config, nil
}

// GetHealthMetrics 헬스 메트릭 반환 (Prometheus 등을 위한)
func (chm *ConfigHealthMonitor) GetHealthMetrics() map[string]interface{} {
	chm.mu.RLock()
	defer chm.mu.RUnlock()

	if chm.lastHealthStatus == nil {
		return nil
	}

	return map[string]interface{}{
		"overall_score":       chm.lastHealthStatus.Score,
		"total_checks":        chm.lastHealthStatus.Summary.TotalChecks,
		"passed_checks":       chm.lastHealthStatus.Summary.PassedChecks,
		"failed_checks":       chm.lastHealthStatus.Summary.FailedChecks,
		"total_issues":        chm.lastHealthStatus.Summary.TotalIssues,
		"error_issues":        chm.lastHealthStatus.Summary.ErrorIssues,
		"warning_issues":      chm.lastHealthStatus.Summary.WarningIssues,
		"auto_healed_count":   chm.lastHealthStatus.Summary.AutoHealedCount,
		"last_check_duration": chm.lastHealthStatus.Duration.Milliseconds(),
	}
}
