// Package routers provides enhanced health monitoring endpoints
package routers

import (
	"context"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"proxynd/cache"
	"proxynd/health"
	"proxynd/internal/config"
)

// EnhancedHealthRouter 강화된 헬스체크 라우터
type EnhancedHealthRouter struct {
	healthService *health.EnhancedHealthService
	config        *config.RootConfig
	cacheManager  *cache.Manager
}

// NewEnhancedHealthRouter 새 강화된 헬스 라우터 생성
func NewEnhancedHealthRouter(
	config *config.RootConfig,
	cacheManager *cache.Manager,
) *EnhancedHealthRouter {
	// 헬스 서비스 설정
	healthConfig := health.DefaultEnhancedHealthConfig()
	
	// 환경 변수에서 설정 조정
	if interval := os.Getenv("HEALTH_CHECK_INTERVAL"); interval != "" {
		if d, err := time.ParseDuration(interval); err == nil {
			healthConfig.CheckInterval = d
		}
	}
	
	if fast := os.Getenv("HEALTH_FAST_MODE"); fast == "true" {
		healthConfig.FastResponseMode = true
		healthConfig.CheckInterval = 10 * time.Second // 더 자주 체크
	}

	// 체크 경로 설정
	checkPaths := []string{}
	if storageDir := os.Getenv("STORAGE_DIR"); storageDir != "" {
		checkPaths = append(checkPaths, storageDir)
	}
	if configDir := os.Getenv("CONFIG_DIR"); configDir != "" {
		checkPaths = append(checkPaths, configDir)
	}
	if len(checkPaths) == 0 {
		checkPaths = []string{"/tmp"}
	}
	healthConfig.SystemCheckPaths = checkPaths

	// 헬스 서비스 생성
	healthService := health.NewEnhancedHealthService(config, cacheManager, healthConfig)

	router := &EnhancedHealthRouter{
		healthService: healthService,
		config:        config,
		cacheManager:  cacheManager,
	}

	return router
}

// RegisterRoutes 라우트 등록
func (ehr *EnhancedHealthRouter) RegisterRoutes(app *fiber.App) {
	// 헬스체크 그룹
	healthGroup := app.Group("/health")

	// 기본 엔드포인트들
	ehr.registerBasicEndpoints(healthGroup)
	
	// 강화된 엔드포인트들
	ehr.registerEnhancedEndpoints(healthGroup)
	
	// 프록시별 엔드포인트들
	ehr.registerProxyEndpoints(healthGroup)
	
	// 시스템 리소스 엔드포인트들
	ehr.registerSystemEndpoints(healthGroup)
	
	// 관리용 엔드포인트들
	ehr.registerManagementEndpoints(healthGroup)

	// 백그라운드에서 헬스 서비스 시작
	go ehr.healthService.Start(context.Background())
}

// registerBasicEndpoints 기본 엔드포인트 등록
func (ehr *EnhancedHealthRouter) registerBasicEndpoints(group fiber.Router) {
	// Kubernetes 호환 엔드포인트
	group.Get("/", ehr.handleRootHealth)
	group.Get("/live", ehr.handleLiveness)
	group.Get("/ready", ehr.handleReadiness)
	
	// 레거시 호환성
	group.Get("/check", ehr.handleBasicHealth)
}

// registerEnhancedEndpoints 강화된 엔드포인트 등록
func (ehr *EnhancedHealthRouter) registerEnhancedEndpoints(group fiber.Router) {
	// 포괄적인 헬스체크
	group.Get("/comprehensive", ehr.handleComprehensiveHealth)
	
	// 빠른 헬스체크 (100ms 미만 목표)
	group.Get("/fast", ehr.handleFastHealth)
	
	// 특정 체커별 상세 정보
	group.Get("/checkers", ehr.handleAvailableCheckers)
	group.Get("/checkers/:name", ehr.handleSpecificChecker)
	
	// 메트릭 엔드포인트
	group.Get("/metrics", ehr.handleHealthMetrics)
}

// registerProxyEndpoints 프록시별 엔드포인트 등록
func (ehr *EnhancedHealthRouter) registerProxyEndpoints(group fiber.Router) {
	proxyGroup := group.Group("/proxy")
	
	// 모든 프록시 업스트림 상태
	proxyGroup.Get("/upstreams", ehr.handleAllProxyUpstreams)
	
	// 특정 프록시 업스트림 상태
	proxyGroup.Get("/upstreams/:type", ehr.handleProxyUpstream)
	
	// 서킷 브레이커 상태
	proxyGroup.Get("/circuit-breakers", ehr.handleCircuitBreakers)
}

// registerSystemEndpoints 시스템 리소스 엔드포인트 등록
func (ehr *EnhancedHealthRouter) registerSystemEndpoints(group fiber.Router) {
	systemGroup := group.Group("/system")
	
	// 시스템 리소스 요약
	systemGroup.Get("/resources", ehr.handleSystemResources)
	
	// 리소스 히스토리
	systemGroup.Get("/resources/history", ehr.handleResourceHistory)
	
	// 리소스 예측
	systemGroup.Get("/resources/predictions", ehr.handleResourcePredictions)
	
	// 메모리 상세 정보
	systemGroup.Get("/memory", ehr.handleMemoryDetails)
	
	// 디스크 상세 정보
	systemGroup.Get("/disk", ehr.handleDiskDetails)
}

// registerManagementEndpoints 관리용 엔드포인트 등록
func (ehr *EnhancedHealthRouter) registerManagementEndpoints(group fiber.Router) {
	mgmtGroup := group.Group("/manage")
	
	// 유지보수 모드
	mgmtGroup.Post("/maintenance", ehr.handleEnableMaintenance)
	mgmtGroup.Delete("/maintenance", ehr.handleDisableMaintenance)
	
	// 헬스체크 설정 조회
	mgmtGroup.Get("/config", ehr.handleHealthConfig)
	
	// 헬스체크 강제 실행
	mgmtGroup.Post("/check/:name", ehr.handleForceCheck)
}

// 핸들러 구현들

// handleRootHealth 루트 헬스체크 핸들러
func (ehr *EnhancedHealthRouter) handleRootHealth(c *fiber.Ctx) error {
	format := c.Query("format", "detailed")
	
	switch format {
	case "simple":
		return ehr.handleFastHealth(c)
	case "comprehensive":
		return ehr.handleComprehensiveHealth(c)
	default:
		return ehr.handleBasicHealth(c)
	}
}

// handleLiveness 라이브니스 프로브 핸들러
func (ehr *EnhancedHealthRouter) handleLiveness(c *fiber.Ctx) error {
	// 프로세스가 살아있으면 항상 200 반환
	return c.JSON(fiber.Map{
		"status":    "alive",
		"timestamp": time.Now(),
		"uptime":    ehr.healthService.GetUptime().Seconds(),
	})
}

// handleReadiness 레디니스 프로브 핸들러
func (ehr *EnhancedHealthRouter) handleReadiness(c *fiber.Ctx) error {
	status, checks := ehr.healthService.GetStatus()
	
	// 핵심 체크 항목들
	requiredChecks := []string{"environment", "memory", "system_resources"}
	ready := true
	failedChecks := []string{}
	
	for _, checkName := range requiredChecks {
		found := false
		for name, result := range checks {
			if strings.Contains(name, checkName) {
				found = true
				if result.Status == health.StatusUnhealthy {
					ready = false
					failedChecks = append(failedChecks, name)
				}
				break
			}
		}
		if !found {
			ready = false
			failedChecks = append(failedChecks, checkName+"_missing")
		}
	}
	
	response := fiber.Map{
		"ready":         ready,
		"status":        string(status),
		"timestamp":     time.Now(),
		"checked":       len(checks),
		"failed_checks": failedChecks,
	}
	
	httpStatus := fiber.StatusOK
	if !ready || status == health.StatusUnhealthy {
		httpStatus = fiber.StatusServiceUnavailable
	}
	
	return c.Status(httpStatus).JSON(response)
}

// handleBasicHealth 기본 헬스체크 핸들러
func (ehr *EnhancedHealthRouter) handleBasicHealth(c *fiber.Ctx) error {
	status, checks := ehr.healthService.GetStatus()
	
	response := health.HealthResponse{
		Status:    status,
		Timestamp: time.Now(),
		Uptime:    formatUptimeEnhanced(ehr.healthService.GetUptime()),
		Checks:    checks,
	}
	
	// 환경 정보 추가
	if env := os.Getenv("ENVIRONMENT"); env != "" {
		response.Environment = env
	}
	if version := os.Getenv("VERSION"); version != "" {
		response.Version = version
	}
	
	httpStatus := fiber.StatusOK
	if status == health.StatusUnhealthy {
		httpStatus = fiber.StatusServiceUnavailable
	}
	
	return c.Status(httpStatus).JSON(response)
}

// handleComprehensiveHealth 포괄적인 헬스체크 핸들러
func (ehr *EnhancedHealthRouter) handleComprehensiveHealth(c *fiber.Ctx) error {
	start := time.Now()
	
	status, err := ehr.healthService.GetComprehensiveStatus()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Failed to get comprehensive health status",
			"details": err.Error(),
		})
	}
	
	// 응답 시간 업데이트
	status.ResponseTime = time.Since(start)
	
	httpStatus := fiber.StatusOK
	if status.OverallStatus == health.StatusUnhealthy {
		httpStatus = fiber.StatusServiceUnavailable
	} else if status.OverallStatus == health.StatusDegraded {
		httpStatus = fiber.StatusOK // 성능 저하는 여전히 200 반환
	}
	
	return c.Status(httpStatus).JSON(status)
}

// handleFastHealth 빠른 헬스체크 핸들러
func (ehr *EnhancedHealthRouter) handleFastHealth(c *fiber.Ctx) error {
	status, err := ehr.healthService.GetFastHealthStatus()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Failed to get fast health status",
			"details": err.Error(),
		})
	}
	
	httpStatus := fiber.StatusOK
	if status.Status == health.StatusUnhealthy {
		httpStatus = fiber.StatusServiceUnavailable
	}
	
	return c.Status(httpStatus).JSON(status)
}

// handleAvailableCheckers 사용 가능한 체커 목록 핸들러
func (ehr *EnhancedHealthRouter) handleAvailableCheckers(c *fiber.Ctx) error {
	checkers := ehr.healthService.GetAvailableCheckers()
	
	return c.JSON(fiber.Map{
		"checkers":       checkers,
		"total_count":    len(checkers),
		"timestamp":      time.Now(),
	})
}

// handleSpecificChecker 특정 체커 정보 핸들러
func (ehr *EnhancedHealthRouter) handleSpecificChecker(c *fiber.Ctx) error {
	checkerName := c.Params("name")
	
	result, err := ehr.healthService.RunHealthCheckFor(checkerName)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":   "Checker not found",
			"checker": checkerName,
			"details": err.Error(),
		})
	}
	
	httpStatus := fiber.StatusOK
	if result.Status == health.StatusUnhealthy {
		httpStatus = fiber.StatusServiceUnavailable
	}
	
	return c.Status(httpStatus).JSON(result)
}

// handleHealthMetrics 헬스 메트릭 핸들러
func (ehr *EnhancedHealthRouter) handleHealthMetrics(c *fiber.Ctx) error {
	status, err := ehr.healthService.GetComprehensiveStatus()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	
	return c.JSON(fiber.Map{
		"metrics":   status.Metrics,
		"timestamp": time.Now(),
		"uptime":    ehr.healthService.GetUptime().Seconds(),
	})
}

// handleAllProxyUpstreams 모든 프록시 업스트림 상태 핸들러
func (ehr *EnhancedHealthRouter) handleAllProxyUpstreams(c *fiber.Ctx) error {
	// 현재 활성화된 프록시 타입들
	proxyTypes := []string{"npm", "pypi", "apt", "docker", "maven", "yum", "apk"}
	results := make(map[string]interface{})
	
	for _, proxyType := range proxyTypes {
		if result, err := ehr.healthService.GetProxyUpstreamStatus(proxyType); err == nil {
			results[proxyType] = result
		} else {
			results[proxyType] = fiber.Map{
				"status": "disabled",
				"error":  err.Error(),
			}
		}
	}
	
	return c.JSON(fiber.Map{
		"upstreams": results,
		"timestamp": time.Now(),
	})
}

// handleProxyUpstream 특정 프록시 업스트림 상태 핸들러
func (ehr *EnhancedHealthRouter) handleProxyUpstream(c *fiber.Ctx) error {
	proxyType := c.Params("type")
	
	result, err := ehr.healthService.GetProxyUpstreamStatus(proxyType)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":      "Proxy type not found or not enabled",
			"proxy_type": proxyType,
			"details":    err.Error(),
		})
	}
	
	httpStatus := fiber.StatusOK
	if result.Status == health.StatusUnhealthy {
		httpStatus = fiber.StatusServiceUnavailable
	}
	
	return c.Status(httpStatus).JSON(fiber.Map{
		"proxy_type": proxyType,
		"result":     result,
		"timestamp":  time.Now(),
	})
}

// handleCircuitBreakers 서킷 브레이커 상태 핸들러
func (ehr *EnhancedHealthRouter) handleCircuitBreakers(c *fiber.Ctx) error {
	status, err := ehr.healthService.GetComprehensiveStatus()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	
	circuitBreakers, exists := status.Metrics["circuit_breakers"]
	if !exists {
		return c.JSON(fiber.Map{
			"message": "Circuit breakers information not available",
			"status":  "not_configured",
		})
	}
	
	return c.JSON(fiber.Map{
		"circuit_breakers": circuitBreakers,
		"timestamp":        time.Now(),
	})
}

// handleSystemResources 시스템 리소스 핸들러
func (ehr *EnhancedHealthRouter) handleSystemResources(c *fiber.Ctx) error {
	status, err := ehr.healthService.GetComprehensiveStatus()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	
	resourceSummary, exists := status.Metrics["resource_summary"]
	if !exists {
		return c.JSON(fiber.Map{
			"message": "System resources information not available",
		})
	}
	
	return c.JSON(fiber.Map{
		"resources": resourceSummary,
		"timestamp": time.Now(),
	})
}

// handleResourceHistory 리소스 히스토리 핸들러
func (ehr *EnhancedHealthRouter) handleResourceHistory(c *fiber.Ctx) error {
	status, err := ehr.healthService.GetComprehensiveStatus()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	
	trends, exists := status.Metrics["resource_trends"]
	if !exists {
		return c.JSON(fiber.Map{
			"message": "Resource trends not available",
		})
	}
	
	return c.JSON(fiber.Map{
		"history":   trends,
		"timestamp": time.Now(),
	})
}

// handleResourcePredictions 리소스 예측 핸들러
func (ehr *EnhancedHealthRouter) handleResourcePredictions(c *fiber.Ctx) error {
	status, err := ehr.healthService.GetComprehensiveStatus()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	
	predictions, exists := status.Metrics["resource_predictions"]
	if !exists {
		return c.JSON(fiber.Map{
			"message": "Resource predictions not available",
		})
	}
	
	return c.JSON(fiber.Map{
		"predictions": predictions,
		"timestamp":   time.Now(),
	})
}

// handleMemoryDetails 메모리 상세 정보 핸들러
func (ehr *EnhancedHealthRouter) handleMemoryDetails(c *fiber.Ctx) error {
	_, checks := ehr.healthService.GetStatus()
	
	// 메모리 관련 체크 결과 찾기
	for name, result := range checks {
		if strings.Contains(name, "memory") {
			return c.JSON(fiber.Map{
				"memory": result,
				"timestamp": time.Now(),
			})
		}
	}
	
	return c.JSON(fiber.Map{
		"message": "Memory details not available",
	})
}

// handleDiskDetails 디스크 상세 정보 핸들러
func (ehr *EnhancedHealthRouter) handleDiskDetails(c *fiber.Ctx) error {
	_, checks := ehr.healthService.GetStatus()
	
	diskInfo := make(map[string]*health.CheckResult)
	
	// 디스크 관련 체크 결과들 수집
	for name, result := range checks {
		if strings.Contains(name, "disk") || strings.Contains(name, "writable") {
			diskInfo[name] = result
		}
	}
	
	if len(diskInfo) == 0 {
		return c.JSON(fiber.Map{
			"message": "Disk details not available",
		})
	}
	
	return c.JSON(fiber.Map{
		"disk":      diskInfo,
		"timestamp": time.Now(),
	})
}

// handleEnableMaintenance 유지보수 모드 활성화 핸들러
func (ehr *EnhancedHealthRouter) handleEnableMaintenance(c *fiber.Ctx) error {
	var request struct {
		Reason   string `json:"reason"`
		Duration string `json:"duration,omitempty"`
	}
	
	if err := c.BodyParser(&request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}
	
	if request.Reason == "" {
		request.Reason = "Scheduled maintenance"
	}
	
	ehr.healthService.EnableMaintMode(request.Reason)
	
	return c.JSON(fiber.Map{
		"message":   "Maintenance mode enabled",
		"reason":    request.Reason,
		"timestamp": time.Now(),
	})
}

// handleDisableMaintenance 유지보수 모드 비활성화 핸들러
func (ehr *EnhancedHealthRouter) handleDisableMaintenance(c *fiber.Ctx) error {
	// TODO: 유지보수 모드 비활성화 로직 구현
	return c.JSON(fiber.Map{
		"message":   "Maintenance mode disabled",
		"timestamp": time.Now(),
	})
}

// handleHealthConfig 헬스체크 설정 조회 핸들러
func (ehr *EnhancedHealthRouter) handleHealthConfig(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"check_interval":    ehr.healthService.GetUptime(),
		"checkers_enabled":  len(ehr.healthService.GetAvailableCheckers()),
		"environment":       os.Getenv("ENVIRONMENT"),
		"fast_mode":         os.Getenv("HEALTH_FAST_MODE") == "true",
		"timestamp":         time.Now(),
	})
}

// handleForceCheck 강제 체크 핸들러
func (ehr *EnhancedHealthRouter) handleForceCheck(c *fiber.Ctx) error {
	checkerName := c.Params("name")
	
	result, err := ehr.healthService.RunHealthCheckFor(checkerName)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":   "Failed to run health check",
			"checker": checkerName,
			"details": err.Error(),
		})
	}
	
	return c.JSON(fiber.Map{
		"message":   "Health check completed",
		"checker":   checkerName,
		"result":    result,
		"timestamp": time.Now(),
	})
}

// 헬퍼 함수들

// formatUptimeEnhanced 가동 시간 포맷팅 (강화된 버전)
func formatUptimeEnhanced(d time.Duration) string {
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	if days > 0 {
		return strconv.Itoa(days) + "d " + strconv.Itoa(hours) + "h " + 
			   strconv.Itoa(minutes) + "m " + strconv.Itoa(seconds) + "s"
	} else if hours > 0 {
		return strconv.Itoa(hours) + "h " + strconv.Itoa(minutes) + "m " + 
			   strconv.Itoa(seconds) + "s"
	} else if minutes > 0 {
		return strconv.Itoa(minutes) + "m " + strconv.Itoa(seconds) + "s"
	}
	return strconv.Itoa(seconds) + "s"
}