// Package routers provides API route definitions
package routers

import (
	"github.com/gofiber/fiber/v2"

	"proxynd/internal/config"
	"proxynd/internal/mirror"
	"proxynd/logging"
)

// APKMirrorRouter APK 미러 선택 관련 API 라우터
func APKMirrorRouter(app *fiber.App) {
	api := app.Group("/api/apk/mirror")

	// 미러 상태 조회
	api.Get("/status", getMirrorStatus)

	// 미러 헬스 정보 조회
	api.Get("/health", getMirrorHealth)

	// 미러 선택 테스트
	api.Post("/select", testMirrorSelection)

	// 미러 선택 설정 조회
	api.Get("/config", getMirrorConfig)

	// 미러 헬스체크 강제 실행
	api.Post("/health/refresh", refreshMirrorHealth)
}

// getMirrorStatus 미러 선택기 전체 상태 조회
func getMirrorStatus(c *fiber.Ctx) error {
	logger := logging.GetLogger()

	// APK 설정 읽기
	apkConfig := config.ApkProxySettings{}
	if err := apkConfig.ReadConfig(); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to read APK config",
		})
	}

	// 미러 선택기가 활성화되어 있는지 확인
	if !apkConfig.MirrorSelection.Enabled {
		return c.JSON(fiber.Map{
			"enabled": false,
			"message": "Mirror selection is disabled",
		})
	}

	// 미러 선택기 상태 가져오기
	selector := mirror.GetAlpineMirrorSelector()
	healthStats := selector.GetHealthStats()
	mirrorHealth := selector.GetMirrorHealth()

	// 지역별 미러 분포 계산
	regionCounts := make(map[string]int)
	healthyRegions := make(map[string]int)

	for _, health := range mirrorHealth {
		regionCounts[health.Region]++
		if health.IsHealthy {
			healthyRegions[health.Region]++
		}
	}

	logger.Info("미러 상태 조회 요청")

	return c.JSON(fiber.Map{
		"enabled":         true,
		"config":          apkConfig.MirrorSelection,
		"statistics":      healthStats,
		"region_counts":   regionCounts,
		"healthy_regions": healthyRegions,
		"total_mirrors":   len(apkConfig.Proxies),
	})
}

// getMirrorHealth 개별 미러 헬스 정보 조회
func getMirrorHealth(c *fiber.Ctx) error {
	logger := logging.GetLogger()

	// APK 설정 읽기
	apkConfig := config.ApkProxySettings{}
	if err := apkConfig.ReadConfig(); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to read APK config",
		})
	}

	if !apkConfig.MirrorSelection.Enabled {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Mirror selection is disabled",
		})
	}

	selector := mirror.GetAlpineMirrorSelector()
	mirrorHealth := selector.GetMirrorHealth()

	// 상태별로 분류
	healthyMirrors := make([]interface{}, 0)
	unhealthyMirrors := make([]interface{}, 0)

	for _, health := range mirrorHealth {
		healthInfo := fiber.Map{
			"name":          health.Name,
			"url":           health.URL,
			"region":        health.Region,
			"priority":      health.Priority,
			"response_time": health.ResponseTime,
			"last_check":    health.LastCheck,
			"error_count":   health.ErrorCount,
		}

		if health.IsHealthy {
			healthyMirrors = append(healthyMirrors, healthInfo)
		} else {
			unhealthyMirrors = append(unhealthyMirrors, healthInfo)
		}
	}

	logger.Info("미러 헬스 정보 조회 요청")

	return c.JSON(fiber.Map{
		"healthy_mirrors":   healthyMirrors,
		"unhealthy_mirrors": unhealthyMirrors,
		"summary": fiber.Map{
			"total":     len(mirrorHealth),
			"healthy":   len(healthyMirrors),
			"unhealthy": len(unhealthyMirrors),
		},
	})
}

// MirrorSelectionRequest 미러 선택 테스트 요청
type MirrorSelectionRequest struct {
	RequestPath string `json:"request_path" validate:"required"`
}

// testMirrorSelection 특정 경로에 대한 미러 선택 테스트
func testMirrorSelection(c *fiber.Ctx) error {
	logger := logging.GetLogger()

	var req MirrorSelectionRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request format",
		})
	}

	if req.RequestPath == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "request_path is required",
		})
	}

	// APK 설정 읽기
	apkConfig := config.ApkProxySettings{}
	if err := apkConfig.ReadConfig(); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to read APK config",
		})
	}

	if !apkConfig.MirrorSelection.Enabled {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Mirror selection is disabled",
		})
	}

	// 미러 선택 수행
	selector := mirror.GetAlpineMirrorSelector()
	selectedMirrors := selector.SelectBestMirror(req.RequestPath, apkConfig.Proxies)

	// 선택된 미러 정보 구성
	mirrorInfo := make([]fiber.Map, 0, len(selectedMirrors))
	for i, selectedMirror := range selectedMirrors {
		health := selector.GetMirrorHealth()[selectedMirror.Name]
		info := fiber.Map{
			"rank":     i + 1,
			"name":     selectedMirror.Name,
			"url":      selectedMirror.URL,
			"region":   "",
			"healthy":  true,
			"priority": 0,
		}

		if health != nil {
			info["region"] = health.Region
			info["healthy"] = health.IsHealthy
			info["priority"] = health.Priority
			info["response_time"] = health.ResponseTime
			info["error_count"] = health.ErrorCount
		}

		mirrorInfo = append(mirrorInfo, info)
	}

	logger.Info("미러 선택 테스트 수행",
		logging.F("path", req.RequestPath),
		logging.F("selected_count", len(selectedMirrors)))

	return c.JSON(fiber.Map{
		"request_path":     req.RequestPath,
		"selected_mirrors": mirrorInfo,
		"total_available":  len(apkConfig.Proxies),
		"selection_time":   "calculated_on_demand",
	})
}

// getMirrorConfig 미러 선택 설정 조회
func getMirrorConfig(c *fiber.Ctx) error {
	// APK 설정 읽기
	apkConfig := config.ApkProxySettings{}
	if err := apkConfig.ReadConfig(); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to read APK config",
		})
	}

	return c.JSON(fiber.Map{
		"mirror_selection": apkConfig.MirrorSelection,
		"proxies":          apkConfig.Proxies,
	})
}

// refreshMirrorHealth 미러 헬스체크 강제 새로고침
func refreshMirrorHealth(c *fiber.Ctx) error {
	logger := logging.GetLogger()

	// APK 설정 읽기
	apkConfig := config.ApkProxySettings{}
	if err := apkConfig.ReadConfig(); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to read APK config",
		})
	}

	if !apkConfig.MirrorSelection.Enabled {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Mirror selection is disabled",
		})
	}

	// 강제 헬스체크는 현재 구현에서 지원하지 않음
	// 향후 확장 가능
	logger.Info("미러 헬스체크 새로고침 요청")

	return c.JSON(fiber.Map{
		"message": "Health check refresh requested",
		"note":    "Health checks are performed automatically at configured intervals",
	})
}
