package routers

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"proxynd/health"
	"proxynd/internal/config"
)

// healthService 전역 건강 상태 서비스
var healthService *health.HealthService

// InitHealthService 건강 상태 서비스 초기화
func InitHealthService(config *config.UnifiedConfig) {
	// 체크 간격 설정 (기본 30초)
	checkInterval := 30 * time.Second

	healthService = health.NewHealthService(checkInterval)

	// 환경 변수 체커
	healthService.RegisterChecker(health.NewEnvironmentChecker([]string{
		"CONFIG_DIR",
		"STORAGE_DIR",
	}))

	// 디스크 공간 체커
	if storageDir := os.Getenv("STORAGE_DIR"); storageDir != "" {
		// 최소 1GB 여유 공간, 10% 여유 공간
		healthService.RegisterChecker(health.NewDiskSpaceChecker(
			storageDir,
			1024*1024*1024, // 1GB
			10.0,           // 10%
		))

		// 쓰기 가능 체커
		healthService.RegisterChecker(health.NewWritableChecker(storageDir))
	}

	// 캐시 디렉토리 체커
	if config != nil && config.Cache.File.Directory != "" {
		healthService.RegisterChecker(health.NewWritableChecker(config.Cache.File.Directory))
	}

	// 업스트림 레지스트리 체커 (선택적)
	if config != nil {
		// NPM 레지스트리
		if config.Registries.NPM.Enabled {
			healthService.RegisterChecker(health.NewHTTPChecker(
				"npm_registry",
				config.Registries.NPM.Upstream+"/-/ping",
				5*time.Second,
			))
		}

		// PyPI 레지스트리
		if config.Registries.PyPI.Enabled {
			healthService.RegisterChecker(health.NewHTTPChecker(
				"pypi_registry",
				config.Registries.PyPI.Simple,
				5*time.Second,
			))
		}
	}

	// 백그라운드에서 건강 상태 체크 시작
	go healthService.Start(context.Background())
}

// HealthRouter 건강 상태 라우터 (backward compatibility)
func HealthRouter(app *fiber.App) {
	HealthRouterWithConfig(app, nil)
}

// HealthRouterWithConfig 설정을 포함한 건강 상태 라우터
func HealthRouterWithConfig(app *fiber.App, config *config.UnifiedConfig) {
	// 건강 상태 서비스 초기화
	InitHealthService(config)

	// 기본 헬스체크 엔드포인트 (/healthz)
	app.Get("/healthz", func(c *fiber.Ctx) error {
		// 간단한 형식 요청 확인
		if c.Query("format") == "simple" {
			status, _ := healthService.GetStatus()
			return c.JSON(health.SimpleHealthResponse{
				Status:    string(status),
				Timestamp: time.Now().Unix(),
			})
		}

		// 상세 건강 상태
		status, checks := healthService.GetStatus()

		response := health.HealthResponse{
			Status:    status,
			Timestamp: time.Now(),
			Uptime:    formatUptime(healthService.GetUptime()),
			Checks:    checks,
		}

		// 환경 정보 추가
		if env := os.Getenv("ENVIRONMENT"); env != "" {
			response.Environment = env
		}

		// 버전 정보 추가
		if version := os.Getenv("VERSION"); version != "" {
			response.Version = version
		}

		// 상태에 따른 HTTP 상태 코드
		httpStatus := fiber.StatusOK
		if status == health.StatusUnhealthy {
			httpStatus = fiber.StatusServiceUnavailable
		}

		return c.Status(httpStatus).JSON(response)
	})

	// Kubernetes 라이브니스 프로브
	app.Get("/health/live", func(c *fiber.Ctx) error {
		// 프로세스가 살아있으면 OK
		return c.JSON(health.LivenessResponse{
			Status: "alive",
		})
	})

	// Kubernetes 레디니스 프로브
	app.Get("/health/ready", func(c *fiber.Ctx) error {
		status, checks := healthService.GetStatus()

		// 필수 체크 항목
		requiredChecks := []string{"environment", "disk_space"}
		ready := true
		checkResults := make(map[string]bool)

		for _, name := range requiredChecks {
			if check, exists := checks[name]; exists {
				isHealthy := check.Status == health.StatusHealthy || check.Status == health.StatusDegraded
				checkResults[name] = isHealthy
				if !isHealthy {
					ready = false
				}
			} else {
				checkResults[name] = false
				ready = false
			}
		}

		// Degraded 상태도 ready로 간주
		if status == health.StatusUnhealthy {
			ready = false
		}

		response := health.ReadinessResponse{
			Ready:  ready,
			Checks: checkResults,
		}

		if !ready {
			return c.Status(fiber.StatusServiceUnavailable).JSON(response)
		}

		return c.JSON(response)
	})

	// 개별 체크 엔드포인트
	app.Get("/health/check/:name", func(c *fiber.Ctx) error {
		checkName := c.Params("name")
		_, checks := healthService.GetStatus()

		if result, exists := checks[checkName]; exists {
			httpStatus := fiber.StatusOK
			if result.Status == health.StatusUnhealthy {
				httpStatus = fiber.StatusServiceUnavailable
			}

			return c.Status(httpStatus).JSON(result)
		}

		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Check not found",
			"name":  checkName,
		})
	})

	// 디버그 정보 (개발 환경에서만)
	app.Get("/health/debug", func(c *fiber.Ctx) error {
		if os.Getenv("ENVIRONMENT") == "production" {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Debug endpoint is disabled in production",
			})
		}

		status, checks := healthService.GetStatus()

		// 시스템 정보 추가
		debugInfo := fiber.Map{
			"status":      status,
			"checks":      checks,
			"uptime":      healthService.GetUptime().Seconds(),
			"goroutines":  runtime.NumGoroutine(),
			"memory":      getMemoryStats(),
			"environment": getAllEnvVars(),
			"config":      getConfigSummary(config),
		}

		return c.JSON(debugInfo)
	})
}

// getMemoryStats 메모리 통계 반환
func getMemoryStats() map[string]interface{} {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return map[string]interface{}{
		"alloc_mb":       m.Alloc / 1024 / 1024,
		"total_alloc_mb": m.TotalAlloc / 1024 / 1024,
		"sys_mb":         m.Sys / 1024 / 1024,
		"num_gc":         m.NumGC,
		"gc_cpu_percent": m.GCCPUFraction * 100,
	}
}

// getAllEnvVars 모든 환경 변수 반환 (민감한 정보 마스킹)
func getAllEnvVars() map[string]string {
	sensitiveKeys := []string{"PASSWORD", "SECRET", "KEY", "TOKEN"}
	envVars := make(map[string]string)

	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			key := parts[0]
			value := parts[1]

			// 민감한 정보 마스킹
			for _, sensitive := range sensitiveKeys {
				if strings.Contains(strings.ToUpper(key), sensitive) {
					value = "***MASKED***"
					break
				}
			}

			envVars[key] = value
		}
	}

	return envVars
}

// getConfigSummary 설정 요약 반환
func getConfigSummary(config *config.UnifiedConfig) map[string]interface{} {
	if config == nil {
		return nil
	}

	return map[string]interface{}{
		"server": map[string]interface{}{
			"port":        config.Server.Port,
			"tls_enabled": config.Server.TLS.Enabled,
		},
		"cache": map[string]interface{}{
			"backend":     config.Cache.Backend,
			"ttl_seconds": config.Cache.TTL.Seconds(),
		},
		"registries_enabled": map[string]bool{
			"npm":    config.Registries.NPM.Enabled,
			"pypi":   config.Registries.PyPI.Enabled,
			"apt":    config.Registries.APT.Enabled,
			"docker": config.Registries.Docker.Enabled,
			"maven":  config.Registries.Maven.Enabled,
		},
		"metrics_enabled": config.Metrics.Enabled,
		"auth_enabled":    len(config.Security.Authentication.BasicAuth.Users) > 0,
	}
}

// formatUptime 가동 시간 포맷
func formatUptime(d time.Duration) string {
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm %ds", days, hours, minutes, seconds)
	} else if hours > 0 {
		return fmt.Sprintf("%dh %dm %ds", hours, minutes, seconds)
	} else if minutes > 0 {
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	}
	return fmt.Sprintf("%ds", seconds)
}
