// Package health provides health monitoring dashboard and reporting
package health

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"

	"proxynd/internal/logging"
)

// HealthDashboard 헬스 대시보드
type HealthDashboard struct {
	healthService   *HealthService
	autoRecovery    *AutoRecoveryService
	circuitBreakers *CircuitBreakerManager
	metrics         *DashboardMetrics
	logger          logging.Logger
	config          *DashboardConfig
	alertManager    *AlertManager
	historicalData  *HistoricalHealthData
	mutex           sync.RWMutex
}

// DashboardConfig 대시보드 설정
type DashboardConfig struct {
	EnableWebUI         bool           `json:"enable_web_ui"`
	RefreshInterval     time.Duration  `json:"refresh_interval"`
	HistoryRetention    time.Duration  `json:"history_retention"`
	AlertThresholds     map[string]int `json:"alert_thresholds"`
	EnableNotifications bool           `json:"enable_notifications"`
}

// DashboardMetrics 대시보드 메트릭
type DashboardMetrics struct {
	mutex            sync.RWMutex
	TotalChecks      int64                    `json:"total_checks"`
	HealthyChecks    int64                    `json:"healthy_checks"`
	DegradedChecks   int64                    `json:"degraded_checks"`
	UnhealthyChecks  int64                    `json:"unhealthy_checks"`
	AvgCheckDuration float64                  `json:"avg_check_duration_ms"`
	LastUpdate       time.Time                `json:"last_update"`
	UptimeStart      time.Time                `json:"uptime_start"`
	ComponentUptime  map[string]time.Duration `json:"component_uptime"`
}

// HistoricalHealthData 과거 헬스 데이터
type HistoricalHealthData struct {
	mutex     sync.RWMutex
	snapshots []*HealthSnapshot
	maxSize   int
}

// HealthSnapshot 헬스 스냅샷
type HealthSnapshot struct {
	Timestamp      time.Time               `json:"timestamp"`
	OverallStatus  Status                  `json:"overall_status"`
	ComponentCount map[Status]int          `json:"component_count"`
	CheckResults   map[string]*CheckResult `json:"check_results"`
	SystemMetrics  map[string]interface{}  `json:"system_metrics"`
}

// AlertRule 알림 규칙
type AlertRule struct {
	Name      string        `json:"name"`
	Component string        `json:"component"`
	Condition string        `json:"condition"` // unhealthy, degraded, timeout
	Duration  time.Duration `json:"duration"`  // 조건 지속 시간
	Severity  string        `json:"severity"`  // low, medium, high, critical
	Enabled   bool          `json:"enabled"`
	LastFired *time.Time    `json:"last_fired"`
	FireCount int           `json:"fire_count"`
}

// Alert 알림
type Alert struct {
	ID          string            `json:"id"`
	Rule        string            `json:"rule"`
	Component   string            `json:"component"`
	Message     string            `json:"message"`
	Severity    string            `json:"severity"`
	Status      string            `json:"status"` // firing, resolved
	StartsAt    time.Time         `json:"starts_at"`
	EndsAt      *time.Time        `json:"ends_at,omitempty"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
}

// AlertManager 알림 관리자
type AlertManager struct {
	rules      []*AlertRule
	alerts     map[string]*Alert
	history    []*Alert
	mutex      sync.RWMutex
	logger     logging.Logger
	webhooks   []string
	maxHistory int
}

// NewHealthDashboard 새 헬스 대시보드 생성
func NewHealthDashboard(healthService *HealthService, config *DashboardConfig) *HealthDashboard {
	if config == nil {
		config = &DashboardConfig{
			EnableWebUI:         true,
			RefreshInterval:     30 * time.Second,
			HistoryRetention:    24 * time.Hour,
			AlertThresholds:     map[string]int{"unhealthy": 3, "degraded": 5},
			EnableNotifications: true,
		}
	}

	dashboard := &HealthDashboard{
		healthService:   healthService,
		autoRecovery:    GetGlobalAutoRecovery(),
		circuitBreakers: GetCircuitBreakerManager(),
		config:          config,
		logger:          logging.GetLogger(),
		metrics: &DashboardMetrics{
			UptimeStart:     time.Now(),
			ComponentUptime: make(map[string]time.Duration),
		},
		alertManager: NewAlertManager(),
		historicalData: &HistoricalHealthData{
			snapshots: make([]*HealthSnapshot, 0),
			maxSize:   1000,
		},
	}

	// 기본 알림 규칙 설정
	dashboard.setupDefaultAlertRules()

	return dashboard
}

// Start 대시보드 시작
func (hd *HealthDashboard) Start(ctx context.Context) {
	go hd.collectMetrics(ctx)
	go hd.monitorAlerts(ctx)

	hd.logger.Info("Health dashboard started",
		logging.F("web_ui_enabled", hd.config.EnableWebUI),
		logging.F("refresh_interval", hd.config.RefreshInterval))
}

// collectMetrics 메트릭 수집
func (hd *HealthDashboard) collectMetrics(ctx context.Context) {
	ticker := time.NewTicker(hd.config.RefreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			hd.updateMetrics()
			hd.captureSnapshot()
		}
	}
}

// updateMetrics 메트릭 업데이트
func (hd *HealthDashboard) updateMetrics() {
	if hd.healthService == nil {
		return
	}

	_, results := hd.healthService.GetStatus()

	hd.metrics.mutex.Lock()
	defer hd.metrics.mutex.Unlock()

	// 상태별 카운트
	var healthyCount, degradedCount, unhealthyCount int64
	var totalDuration time.Duration

	for _, result := range results {
		hd.metrics.TotalChecks++
		totalDuration += result.Duration

		switch result.Status {
		case StatusHealthy:
			healthyCount++
			hd.metrics.HealthyChecks++
		case StatusDegraded:
			degradedCount++
			hd.metrics.DegradedChecks++
		case StatusUnhealthy:
			unhealthyCount++
			hd.metrics.UnhealthyChecks++
		}
	}

	// 평균 체크 시간 계산
	if len(results) > 0 {
		hd.metrics.AvgCheckDuration = float64(totalDuration.Milliseconds()) / float64(len(results))
	}

	hd.metrics.LastUpdate = time.Now()

	// 컴포넌트별 업타임 업데이트
	for name, result := range results {
		if result.Status == StatusHealthy {
			if _, exists := hd.metrics.ComponentUptime[name]; !exists {
				hd.metrics.ComponentUptime[name] = 0
			}
			hd.metrics.ComponentUptime[name] += hd.config.RefreshInterval
		} else {
			hd.metrics.ComponentUptime[name] = 0
		}
	}
}

// captureSnapshot 스냅샷 캡처
func (hd *HealthDashboard) captureSnapshot() {
	if hd.healthService == nil {
		return
	}

	overallStatus, results := hd.healthService.GetStatus()

	// 상태별 카운트
	componentCount := map[Status]int{
		StatusHealthy:   0,
		StatusDegraded:  0,
		StatusUnhealthy: 0,
	}

	for _, result := range results {
		componentCount[result.Status]++
	}

	snapshot := &HealthSnapshot{
		Timestamp:      time.Now(),
		OverallStatus:  overallStatus,
		ComponentCount: componentCount,
		CheckResults:   results,
		SystemMetrics:  hd.collectSystemMetrics(),
	}

	hd.historicalData.mutex.Lock()
	defer hd.historicalData.mutex.Unlock()

	hd.historicalData.snapshots = append(hd.historicalData.snapshots, snapshot)

	// 크기 제한
	if len(hd.historicalData.snapshots) > hd.historicalData.maxSize {
		copy(hd.historicalData.snapshots, hd.historicalData.snapshots[1:])
		hd.historicalData.snapshots = hd.historicalData.snapshots[:hd.historicalData.maxSize]
	}
}

// collectSystemMetrics 시스템 메트릭 수집
func (hd *HealthDashboard) collectSystemMetrics() map[string]interface{} {
	metrics := make(map[string]interface{})

	// 서킷 브레이커 메트릭
	if hd.circuitBreakers != nil {
		cbStatus := hd.circuitBreakers.GetStatus()
		metrics["circuit_breakers"] = cbStatus
	}

	// 자동 복구 메트릭
	if hd.autoRecovery != nil {
		recoveryStatus := hd.autoRecovery.GetRecoveryStatus()
		metrics["auto_recovery"] = recoveryStatus
	}

	// 업타임
	metrics["uptime_seconds"] = time.Since(hd.metrics.UptimeStart).Seconds()

	return metrics
}

// monitorAlerts 알림 모니터링
func (hd *HealthDashboard) monitorAlerts(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			hd.evaluateAlerts()
		}
	}
}

// evaluateAlerts 알림 평가
func (hd *HealthDashboard) evaluateAlerts() {
	if hd.healthService == nil {
		return
	}

	_, results := hd.healthService.GetStatus()
	hd.alertManager.EvaluateRules(results)
}

// GetDashboardData 대시보드 데이터 반환
func (hd *HealthDashboard) GetDashboardData() map[string]interface{} {
	hd.mutex.RLock()
	defer hd.mutex.RUnlock()

	if hd.healthService == nil {
		return map[string]interface{}{
			"error": "Health service not available",
		}
	}

	overallStatus, results := hd.healthService.GetStatus()

	data := map[string]interface{}{
		"overall_status": string(overallStatus),
		"timestamp":      time.Now(),
		"uptime":         hd.healthService.GetUptime().Seconds(),
		"components":     results,
		"metrics":        hd.metrics,
		"alerts":         hd.alertManager.GetActiveAlerts(),
	}

	// 서킷 브레이커 정보
	if hd.circuitBreakers != nil {
		data["circuit_breakers"] = hd.circuitBreakers.GetStatus()
	}

	// 자동 복구 정보
	if hd.autoRecovery != nil {
		data["auto_recovery"] = hd.autoRecovery.GetRecoveryStatus()
	}

	return data
}

// GetHistoricalData 과거 데이터 반환
func (hd *HealthDashboard) GetHistoricalData(hours int) []*HealthSnapshot {
	hd.historicalData.mutex.RLock()
	defer hd.historicalData.mutex.RUnlock()

	if hours <= 0 {
		return hd.historicalData.snapshots
	}

	cutoff := time.Now().Add(-time.Duration(hours) * time.Hour)
	var filtered []*HealthSnapshot

	for _, snapshot := range hd.historicalData.snapshots {
		if snapshot.Timestamp.After(cutoff) {
			filtered = append(filtered, snapshot)
		}
	}

	return filtered
}

// setupDefaultAlertRules 기본 알림 규칙 설정
func (hd *HealthDashboard) setupDefaultAlertRules() {
	rules := []*AlertRule{
		{
			Name:      "ComponentUnhealthy",
			Component: "*",
			Condition: "unhealthy",
			Duration:  2 * time.Minute,
			Severity:  "high",
			Enabled:   true,
		},
		{
			Name:      "ComponentDegraded",
			Component: "*",
			Condition: "degraded",
			Duration:  5 * time.Minute,
			Severity:  "medium",
			Enabled:   true,
		},
		{
			Name:      "CheckTimeout",
			Component: "*",
			Condition: "timeout",
			Duration:  1 * time.Minute,
			Severity:  "medium",
			Enabled:   true,
		},
	}

	for _, rule := range rules {
		hd.alertManager.AddRule(rule)
	}
}

// RegisterHTTPHandlers HTTP 핸들러 등록
func (hd *HealthDashboard) RegisterHTTPHandlers(app *fiber.App) {
	// 대시보드 API 엔드포인트
	app.Get("/health/dashboard", hd.handleDashboard)
	app.Get("/health/dashboard/data", hd.handleDashboardData)
	app.Get("/health/dashboard/history", hd.handleHistoryData)
	app.Get("/health/dashboard/alerts", hd.handleAlerts)
	app.Get("/health/dashboard/metrics", hd.handleMetrics)

	// 관리 엔드포인트
	app.Post("/health/recovery/trigger", hd.handleTriggerRecovery)
	app.Post("/health/circuit-breaker/reset", hd.handleResetCircuitBreaker)
	app.Post("/health/alerts/acknowledge", hd.handleAcknowledgeAlert)

	hd.logger.Info("Health dashboard HTTP handlers registered")
}

// HTTP 핸들러들

func (hd *HealthDashboard) handleDashboard(c *fiber.Ctx) error {
	// 간단한 HTML 대시보드 반환
	html := `
<!DOCTYPE html>
<html>
<head>
    <title>ProxyND Health Dashboard</title>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <script src="https://unpkg.com/htmx.org@1.9.10"></script>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .status { padding: 10px; margin: 10px 0; border-radius: 5px; }
        .healthy { background-color: #d4edda; color: #155724; }
        .degraded { background-color: #fff3cd; color: #856404; }
        .unhealthy { background-color: #f8d7da; color: #721c24; }
        .component { margin: 10px 0; padding: 10px; border: 1px solid #ddd; border-radius: 5px; }
        .refresh-btn {
            background-color: #007bff;
            color: white;
            padding: 10px 20px;
            border: none;
            border-radius: 5px;
            cursor: pointer;
        }
    </style>
</head>
<body>
    <h1>ProxyND Health Dashboard</h1>
    <button class="refresh-btn"
            hx-get="/health/dashboard/data"
            hx-target="#dashboard-content"
            hx-trigger="click">Refresh</button>
    <div id="dashboard-content" hx-get="/health/dashboard/data" hx-trigger="load, every 30s">
        Loading...
    </div>
</body>
</html>`

	c.Set("Content-Type", "text/html")
	return c.SendString(html)
}

func (hd *HealthDashboard) handleDashboardData(c *fiber.Ctx) error {
	data := hd.GetDashboardData()

	// HTML 형태로 데이터 렌더링
	html := hd.renderDashboardHTML(data)
	return c.SendString(html)
}

func (hd *HealthDashboard) handleHistoryData(c *fiber.Ctx) error {
	hours := c.QueryInt("hours", 24)
	data := hd.GetHistoricalData(hours)
	return c.JSON(data)
}

func (hd *HealthDashboard) handleAlerts(c *fiber.Ctx) error {
	alerts := hd.alertManager.GetActiveAlerts()
	return c.JSON(alerts)
}

func (hd *HealthDashboard) handleMetrics(c *fiber.Ctx) error {
	return c.JSON(hd.metrics)
}

func (hd *HealthDashboard) handleTriggerRecovery(c *fiber.Ctx) error {
	var request struct {
		Component string `json:"component"`
		Strategy  string `json:"strategy"`
	}

	if err := c.BodyParser(&request); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request"})
	}

	if hd.autoRecovery == nil {
		return c.Status(503).JSON(fiber.Map{"error": "Auto-recovery service not available"})
	}

	strategy := RecoveryStrategy(request.Strategy)
	err := hd.autoRecovery.TriggerManualRecovery(request.Component, strategy)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"message": "Recovery triggered successfully"})
}

func (hd *HealthDashboard) handleResetCircuitBreaker(c *fiber.Ctx) error {
	var request struct {
		Name string `json:"name"`
	}

	if err := c.BodyParser(&request); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request"})
	}

	if hd.circuitBreakers == nil {
		return c.Status(503).JSON(fiber.Map{"error": "Circuit breaker manager not available"})
	}

	cb, err := hd.circuitBreakers.Get(request.Name)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": err.Error()})
	}

	cb.Reset()
	return c.JSON(fiber.Map{"message": "Circuit breaker reset successfully"})
}

func (hd *HealthDashboard) handleAcknowledgeAlert(c *fiber.Ctx) error {
	var request struct {
		AlertID string `json:"alert_id"`
	}

	if err := c.BodyParser(&request); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request"})
	}

	hd.alertManager.AcknowledgeAlert(request.AlertID)
	return c.JSON(fiber.Map{"message": "Alert acknowledged"})
}

// renderDashboardHTML 대시보드 HTML 렌더링
func (hd *HealthDashboard) renderDashboardHTML(data map[string]interface{}) string {
	overallStatus := data["overall_status"].(string)
	components := data["components"].(map[string]*CheckResult)

	html := fmt.Sprintf(`
<div class="status %s">
    <h2>Overall Status: %s</h2>
    <p>Last Updated: %v</p>
    <p>Uptime: %.2f hours</p>
</div>

<h3>Components (%d)</h3>
`, overallStatus, overallStatus, data["timestamp"], data["uptime"].(float64)/3600, len(components))

	// 컴포넌트들을 상태별로 정렬
	var sortedComponents []*CheckResult
	for _, result := range components {
		sortedComponents = append(sortedComponents, result)
	}
	sort.Slice(sortedComponents, func(i, j int) bool {
		statusOrder := map[Status]int{StatusUnhealthy: 0, StatusDegraded: 1, StatusHealthy: 2}
		return statusOrder[sortedComponents[i].Status] < statusOrder[sortedComponents[j].Status]
	})

	for _, result := range sortedComponents {
		statusClass := string(result.Status)
		html += fmt.Sprintf(`
<div class="component %s">
    <h4>%s</h4>
    <p>Status: %s</p>
    <p>Message: %s</p>
    <p>Duration: %dms</p>
    <p>Last Checked: %v</p>
</div>
`, statusClass, result.Name, result.Status, result.Message, result.Duration.Milliseconds(), result.LastChecked)
	}

	return html
}

// AlertManager 알림 관리자

// NewAlertManager 새 알림 관리자 생성
func NewAlertManager() *AlertManager {
	return &AlertManager{
		rules:      make([]*AlertRule, 0),
		alerts:     make(map[string]*Alert),
		history:    make([]*Alert, 0),
		logger:     logging.GetLogger(),
		webhooks:   make([]string, 0),
		maxHistory: 1000,
	}
}

// AddRule 알림 규칙 추가
func (am *AlertManager) AddRule(rule *AlertRule) {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	am.rules = append(am.rules, rule)
	am.logger.Info("Alert rule added", logging.F("rule", rule.Name))
}

// EvaluateRules 규칙 평가
func (am *AlertManager) EvaluateRules(results map[string]*CheckResult) {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	for _, rule := range am.rules {
		if !rule.Enabled {
			continue
		}

		// 규칙에 해당하는 컴포넌트 찾기
		for _, result := range results {
			if rule.Component != "*" && rule.Component != result.Name {
				continue
			}

			shouldFire := false
			message := ""

			switch rule.Condition {
			case "unhealthy":
				shouldFire = result.Status == StatusUnhealthy
				message = fmt.Sprintf("Component %s is unhealthy: %s", result.Name, result.Message)
			case "degraded":
				shouldFire = result.Status == StatusDegraded
				message = fmt.Sprintf("Component %s is degraded: %s", result.Name, result.Message)
			case "timeout":
				shouldFire = result.Duration > 10*time.Second
				message = fmt.Sprintf("Component %s check timed out: %dms", result.Name, result.Duration.Milliseconds())
			}

			if shouldFire {
				am.fireAlert(rule, result.Name, message)
			} else {
				am.resolveAlert(rule, result.Name)
			}
		}
	}
}

// fireAlert 알림 발생
func (am *AlertManager) fireAlert(rule *AlertRule, component, message string) {
	alertID := fmt.Sprintf("%s_%s", rule.Name, component)

	// 기존 알림 확인
	if alert, exists := am.alerts[alertID]; exists {
		if alert.Status == "firing" {
			return // 이미 발생 중인 알림
		}
	}

	// 새 알림 생성
	alert := &Alert{
		ID:        alertID,
		Rule:      rule.Name,
		Component: component,
		Message:   message,
		Severity:  rule.Severity,
		Status:    "firing",
		StartsAt:  time.Now(),
		Labels: map[string]string{
			"component": component,
			"severity":  rule.Severity,
		},
		Annotations: map[string]string{
			"description": message,
		},
	}

	am.alerts[alertID] = alert
	rule.LastFired = &alert.StartsAt
	rule.FireCount++

	am.logger.Warn("Alert fired",
		logging.F("rule", rule.Name),
		logging.F("component", component),
		logging.F("message", message),
		logging.F("severity", rule.Severity))
}

// resolveAlert 알림 해결
func (am *AlertManager) resolveAlert(rule *AlertRule, component string) {
	alertID := fmt.Sprintf("%s_%s", rule.Name, component)

	if alert, exists := am.alerts[alertID]; exists && alert.Status == "firing" {
		now := time.Now()
		alert.Status = "resolved"
		alert.EndsAt = &now

		// 이력에 추가
		am.history = append(am.history, alert)
		if len(am.history) > am.maxHistory {
			copy(am.history, am.history[1:])
			am.history = am.history[:am.maxHistory]
		}

		// 활성 알림에서 제거
		delete(am.alerts, alertID)

		am.logger.Info("Alert resolved",
			logging.F("rule", rule.Name),
			logging.F("component", component))
	}
}

// GetActiveAlerts 활성 알림 반환
func (am *AlertManager) GetActiveAlerts() []*Alert {
	am.mutex.RLock()
	defer am.mutex.RUnlock()

	alerts := make([]*Alert, 0, len(am.alerts))
	for _, alert := range am.alerts {
		alerts = append(alerts, alert)
	}

	return alerts
}

// AcknowledgeAlert 알림 확인
func (am *AlertManager) AcknowledgeAlert(alertID string) {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	if alert, exists := am.alerts[alertID]; exists {
		alert.Annotations["acknowledged"] = "true"
		alert.Annotations["acknowledged_at"] = time.Now().Format(time.RFC3339)
	}
}
