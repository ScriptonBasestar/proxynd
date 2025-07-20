package routers

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"

	"proxynd/configs"
	"proxynd/dtos"
	"proxynd/internal/webhook"
	"proxynd/logging"
)

// WebhookRouter 웹훅 관련 라우트 설정
func WebhookRouter(app *fiber.App) {
	logger := logging.GetLogger()

	// 웹훅 설정 로드
	var webhookConfig configs.WebhookConfig
	if webhookConfig.ConfigExists() {
		_ = webhookConfig.ReadConfig()
	} else {
		webhookConfig = configs.GetDefaultWebhookConfig()
	}

	// 웹훅 sender 및 테스터 초기화
	var sender *webhook.WebhookSender
	var historyManager *webhook.WebhookHistoryManager

	if webhookConfig.Enabled {
		if s, err := webhook.NewWebhookSender(webhookConfig); err == nil {
			sender = s
			historyManager = sender.GetHistoryManager()
		} else {
			logger.Error("웹훅 sender 초기화 실패", logging.F(fieldError, err))
		}
	}

	tester := webhook.NewWebhookTester(webhookConfig, sender)

	// API 그룹 생성
	api := app.Group("/api/v1/webhook")

	// 웹훅 테스트 엔드포인트들
	api.Post("/test", testAllWebhooks(tester, logger))
	api.Post("/test/:endpoint", testSingleWebhook(tester, logger))
	api.Get("/test/connectivity/:endpoint", testWebhookConnectivity(tester, logger))
	api.Post("/validate", validateWebhookConfig(logger))

	// 웹훅 상태 및 통계 (기본 응답)
	api.Get("/status", getWebhookStatusSimple(webhookConfig, logger))
	api.Get("/endpoints", getWebhookEndpoints(webhookConfig, logger))

	// 웹훅 이력 및 통계 API
	api.Get("/history", getWebhookHistory(historyManager, logger))
	api.Get("/history/:endpoint", getWebhookHistoryByEndpoint(historyManager, logger))
	api.Get("/statistics", getWebhookStatistics(historyManager, logger))
	api.Get("/statistics/:endpoint", getWebhookStatisticsByEndpoint(historyManager, logger))
	api.Get("/activity/recent", getRecentActivity(historyManager, logger))

	logger.Info("웹훅 라우트 설정 완료", logging.F("endpoints", 11))
}

// testAllWebhooks 모든 웹훅 엔드포인트 테스트
func testAllWebhooks(tester *webhook.WebhookTester, logger logging.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		logger.Info("모든 웹훅 엔드포인트 테스트 요청",
			logging.F("client_ip", c.IP()))

		ctx, cancel := context.WithTimeout(c.Context(), 30*time.Second)
		defer cancel()

		// 백그라운드에서 테스트 실행
		resultChan := make(chan *webhook.TestAllResult, 1)
		errorChan := make(chan error, 1)

		go func() {
			result, err := tester.TestAllEndpoints()
			if err != nil {
				errorChan <- err
			} else {
				resultChan <- result
			}
		}()

		select {
		case result := <-resultChan:
			response := dtos.APIResponse{
				Success: true,
				Message: "웹훅 테스트 완료",
				Data:    result,
			}
			return c.JSON(response)

		case err := <-errorChan:
			logger.Error("웹훅 테스트 실행 오류", logging.F(fieldError, err))
			response := dtos.APIResponse{
				Success: false,
				Message: "웹훅 테스트 실패",
				Error:   err.Error(),
			}
			return c.Status(500).JSON(response)

		case <-ctx.Done():
			response := dtos.APIResponse{
				Success: false,
				Message: "웹훅 테스트 타임아웃",
				Error:   "request timeout",
			}
			return c.Status(408).JSON(response)
		}
	}
}

// testSingleWebhook 단일 웹훅 엔드포인트 테스트
func testSingleWebhook(tester *webhook.WebhookTester, logger logging.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		endpointName := c.Params(fieldEndpoint)
		if endpointName == "" {
			response := dtos.APIResponse{
				Success: false,
				Message: "엔드포인트 이름이 필요합니다",
				Error:   "endpoint parameter is required",
			}
			return c.Status(400).JSON(response)
		}

		logger.Info("단일 웹훅 엔드포인트 테스트 요청",
			logging.F(fieldEndpoint, endpointName),
			logging.F("client_ip", c.IP()))

		result, err := tester.TestSingleEndpoint(endpointName)
		if err != nil {
			logger.Error("웹훅 엔드포인트 테스트 오류",
				logging.F(fieldEndpoint, endpointName),
				logging.F(fieldError, err))

			response := dtos.APIResponse{
				Success: false,
				Message: "웹훅 엔드포인트 테스트 실패",
				Error:   err.Error(),
			}
			return c.Status(404).JSON(response)
		}

		response := dtos.APIResponse{
			Success: result.Success,
			Message: "웹훅 엔드포인트 테스트 완료",
			Data:    result,
		}

		if !result.Success {
			return c.Status(500).JSON(response)
		}

		return c.JSON(response)
	}
}

// testWebhookConnectivity 웹훅 연결성 테스트
func testWebhookConnectivity(tester *webhook.WebhookTester, logger logging.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		endpointName := c.Params(fieldEndpoint)
		if endpointName == "" {
			response := dtos.APIResponse{
				Success: false,
				Message: "엔드포인트 이름이 필요합니다",
				Error:   "endpoint parameter is required",
			}
			return c.Status(400).JSON(response)
		}

		logger.Debug("웹훅 연결성 테스트 요청",
			logging.F(fieldEndpoint, endpointName),
			logging.F("client_ip", c.IP()))

		result, err := tester.TestEndpointConnectivity(endpointName)
		if err != nil {
			logger.Error("웹훅 연결성 테스트 오류",
				logging.F(fieldEndpoint, endpointName),
				logging.F(fieldError, err))

			response := dtos.APIResponse{
				Success: false,
				Message: "웹훅 연결성 테스트 실패",
				Error:   err.Error(),
			}
			return c.Status(404).JSON(response)
		}

		response := dtos.APIResponse{
			Success: result.Success,
			Message: "웹훅 연결성 테스트 완료",
			Data:    result,
		}

		return c.JSON(response)
	}
}

// validateWebhookConfig 웹훅 설정 검증
func validateWebhookConfig(logger logging.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var endpoint configs.WebhookEndpointConfig
		if err := c.BodyParser(&endpoint); err != nil {
			response := dtos.APIResponse{
				Success: false,
				Message: "잘못된 요청 형식",
				Error:   err.Error(),
			}
			return c.Status(400).JSON(response)
		}

		logger.Debug("웹훅 설정 검증 요청",
			logging.F("endpoint_name", endpoint.Name),
			logging.F("client_ip", c.IP()))

		// 더미 테스터로 검증 (설정만 필요)
		tester := &webhook.WebhookTester{}
		errors := tester.ValidateEndpointConfig(endpoint)

		response := dtos.APIResponse{
			Success: len(errors) == 0,
			Message: "웹훅 설정 검증 완료",
			Data: map[string]interface{}{
				"valid":  len(errors) == 0,
				"errors": errors,
			},
		}

		if len(errors) > 0 {
			return c.Status(400).JSON(response)
		}

		return c.JSON(response)
	}
}

// getWebhookStatusSimple 웹훅 시스템 기본 상태 조회
func getWebhookStatusSimple(config configs.WebhookConfig, _ logging.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		enabledCount := 0
		for _, endpoint := range config.Endpoints {
			if endpoint.Enabled {
				enabledCount++
			}
		}

		status := map[string]interface{}{
			"webhook_system_enabled": config.Enabled,
			"total_endpoints":        len(config.Endpoints),
			"enabled_endpoints":      enabledCount,
			"disabled_endpoints":     len(config.Endpoints) - enabledCount,
			"check_time":             time.Now(),
		}

		response := dtos.APIResponse{
			Success: true,
			Message: "웹훅 상태 조회 완료",
			Data:    status,
		}

		return c.JSON(response)
	}
}

// getWebhookEndpoints 웹훅 엔드포인트 목록 조회
func getWebhookEndpoints(config configs.WebhookConfig, _ logging.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// 민감한 정보 제외하고 반환
		endpoints := make([]map[string]interface{}, len(config.Endpoints))

		for i, ep := range config.Endpoints {
			endpoints[i] = map[string]interface{}{
				"name":        ep.Name,
				"url":         ep.URL,
				"enabled":     ep.Enabled,
				"method":      ep.Method,
				"format":      ep.Format,
				"timeout":     ep.Timeout,
				"event_types": ep.EventTypes,
				// 인증 정보는 제외
			}
		}

		response := dtos.APIResponse{
			Success: true,
			Message: "웹훅 엔드포인트 목록 조회 완료",
			Data: map[string]interface{}{
				fieldTotal:  len(endpoints),
				"endpoints": endpoints,
			},
		}

		return c.JSON(response)
	}
}

// getWebhookHistory 웹훅 전송 이력 조회
func getWebhookHistory(historyManager *webhook.WebhookHistoryManager, logger logging.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// 쿼리 파라미터 파싱
		limitStr := c.Query(fieldLimit, "50")
		offsetStr := c.Query(fieldOffset, "0")

		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit <= 0 || limit > 500 {
			limit = 50
		}

		offset, err := strconv.Atoi(offsetStr)
		if err != nil || offset < 0 {
			offset = 0
		}

		if historyManager == nil {
			response := dtos.APIResponse{
				Success: true,
				Message: "웹훅 이력 조회 완료",
				Data: map[string]interface{}{
					"histories": []interface{}{},
					fieldTotal:  0,
					fieldLimit:  limit,
					fieldOffset: offset,
					"note":      "히스토리 매니저가 초기화되지 않음",
				},
			}
			return c.JSON(response)
		}

		// 히스토리 조회
		histories, total, err := historyManager.GetHistory("", limit, offset)
		if err != nil {
			logger.Error("웹훅 이력 조회 실패", logging.F(fieldError, err))
			response := dtos.APIResponse{
				Success: false,
				Message: "웹훅 이력 조회 실패",
				Error:   err.Error(),
			}
			return c.Status(500).JSON(response)
		}

		response := dtos.APIResponse{
			Success: true,
			Message: "웹훅 이력 조회 완료",
			Data: map[string]interface{}{
				"histories": histories,
				fieldTotal:  total,
				fieldLimit:  limit,
				fieldOffset: offset,
			},
		}

		return c.JSON(response)
	}
}

// getWebhookHistoryByEndpoint 특정 엔드포인트의 웹훅 이력 조회
func getWebhookHistoryByEndpoint(historyManager *webhook.WebhookHistoryManager, logger logging.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		endpointName := c.Params(fieldEndpoint)
		if endpointName == "" {
			response := dtos.APIResponse{
				Success: false,
				Message: "엔드포인트 이름이 필요합니다",
				Error:   "endpoint parameter is required",
			}
			return c.Status(400).JSON(response)
		}

		// 쿼리 파라미터 파싱
		limitStr := c.Query(fieldLimit, "50")
		offsetStr := c.Query(fieldOffset, "0")

		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit <= 0 || limit > 500 {
			limit = 50
		}

		offset, err := strconv.Atoi(offsetStr)
		if err != nil || offset < 0 {
			offset = 0
		}

		logger.Debug("특정 엔드포인트 웹훅 이력 조회",
			logging.F(fieldEndpoint, endpointName),
			logging.F(fieldLimit, limit),
			logging.F(fieldOffset, offset))

		if historyManager == nil {
			response := dtos.APIResponse{
				Success: true,
				Message: "엔드포인트별 웹훅 이력 조회 완료",
				Data: map[string]interface{}{
					fieldEndpoint: endpointName,
					"histories":   []interface{}{},
					fieldTotal:    0,
					fieldLimit:    limit,
					fieldOffset:   offset,
					"note":        "히스토리 매니저가 초기화되지 않음",
				},
			}
			return c.JSON(response)
		}

		// 히스토리 조회
		histories, total, err := historyManager.GetHistory(endpointName, limit, offset)
		if err != nil {
			logger.Error("엔드포인트별 웹훅 이력 조회 실패", logging.F(fieldError, err))
			response := dtos.APIResponse{
				Success: false,
				Message: "엔드포인트별 웹훅 이력 조회 실패",
				Error:   err.Error(),
			}
			return c.Status(500).JSON(response)
		}

		response := dtos.APIResponse{
			Success: true,
			Message: "엔드포인트별 웹훅 이력 조회 완료",
			Data: map[string]interface{}{
				fieldEndpoint: endpointName,
				"histories":   histories,
				fieldTotal:    total,
				fieldLimit:    limit,
				fieldOffset:   offset,
			},
		}

		return c.JSON(response)
	}
}

// getWebhookStatistics 웹훅 통계 정보 조회
func getWebhookStatistics(historyManager *webhook.WebhookHistoryManager, logger logging.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// 시간 범위 쿼리 파라미터
		hoursStr := c.Query("hours", "24")
		hours, err := strconv.Atoi(hoursStr)
		if err != nil || hours <= 0 || hours > 24*30 { // 최대 30일
			hours = 24
		}

		logger.Debug("웹훅 통계 조회 요청",
			logging.F("hours", hours),
			logging.F("client_ip", c.IP()))

		if historyManager == nil {
			// 임시 통계 데이터
			now := time.Now()
			startTime := now.Add(-time.Duration(hours) * time.Hour)

			response := dtos.APIResponse{
				Success: true,
				Message: "웹훅 통계 조회 완료",
				Data: map[string]interface{}{
					"total_sent":            0,
					"total_success":         0,
					"total_failed":          0,
					"total_retrying":        0,
					"success_rate":          0.0,
					"average_response_time": "0s",
					"endpoint_stats":        map[string]interface{}{},
					"time_range": map[string]interface{}{
						"start_time": startTime,
						"end_time":   now,
						"duration":   fmt.Sprintf("%dh", hours),
					},
					"last_updated": now,
					"note":         "히스토리 매니저가 초기화되지 않음",
				},
			}
			return c.JSON(response)
		}

		// 시간 범위 설정
		now := time.Now()
		startTime := now.Add(-time.Duration(hours) * time.Hour)
		timeRange := &webhook.TimeRangeStats{
			StartTime: startTime,
			EndTime:   now,
			Duration:  fmt.Sprintf("%dh", hours),
		}

		// 통계 조회
		stats, err := historyManager.GetStatistics("", timeRange)
		if err != nil {
			logger.Error("웹훅 통계 조회 실패", logging.F(fieldError, err))
			response := dtos.APIResponse{
				Success: false,
				Message: "웹훅 통계 조회 실패",
				Error:   err.Error(),
			}
			return c.Status(500).JSON(response)
		}

		response := dtos.APIResponse{
			Success: true,
			Message: "웹훅 통계 조회 완료",
			Data:    stats,
		}

		return c.JSON(response)
	}
}

// getWebhookStatisticsByEndpoint 특정 엔드포인트의 웹훅 통계 조회
func getWebhookStatisticsByEndpoint(historyManager *webhook.WebhookHistoryManager,
	logger logging.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		endpointName := c.Params(fieldEndpoint)
		if endpointName == "" {
			response := dtos.APIResponse{
				Success: false,
				Message: "엔드포인트 이름이 필요합니다",
				Error:   "endpoint parameter is required",
			}
			return c.Status(400).JSON(response)
		}

		// 시간 범위 쿼리 파라미터
		hoursStr := c.Query("hours", "24")
		hours, err := strconv.Atoi(hoursStr)
		if err != nil || hours <= 0 || hours > 24*30 {
			hours = 24
		}

		logger.Debug("특정 엔드포인트 웹훅 통계 조회",
			logging.F(fieldEndpoint, endpointName),
			logging.F("hours", hours))

		if historyManager == nil {
			now := time.Now()
			startTime := now.Add(-time.Duration(hours) * time.Hour)

			response := dtos.APIResponse{
				Success: true,
				Message: "엔드포인트별 웹훅 통계 조회 완료",
				Data: map[string]interface{}{
					fieldEndpoint:           endpointName,
					"total_sent":            0,
					"total_success":         0,
					"total_failed":          0,
					"success_rate":          0.0,
					"average_response_time": "0s",
					"last_sent":             nil,
					"last_success":          nil,
					"last_failure":          nil,
					"time_range": map[string]interface{}{
						"start_time": startTime,
						"end_time":   now,
						"duration":   fmt.Sprintf("%dh", hours),
					},
					"note": "히스토리 매니저가 초기화되지 않음",
				},
			}
			return c.JSON(response)
		}

		// 시간 범위 설정
		now := time.Now()
		startTime := now.Add(-time.Duration(hours) * time.Hour)
		timeRange := &webhook.TimeRangeStats{
			StartTime: startTime,
			EndTime:   now,
			Duration:  fmt.Sprintf("%dh", hours),
		}

		// 특정 엔드포인트 통계 조회
		stats, err := historyManager.GetStatistics(endpointName, timeRange)
		if err != nil {
			logger.Error("엔드포인트별 웹훅 통계 조회 실패", logging.F(fieldError, err))
			response := dtos.APIResponse{
				Success: false,
				Message: "엔드포인트별 웹훅 통계 조회 실패",
				Error:   err.Error(),
			}
			return c.Status(500).JSON(response)
		}

		// 엔드포인트별 통계 추출
		var endpointStats interface{}
		if epStats, exists := stats.EndpointStats[endpointName]; exists {
			endpointStats = epStats
		} else {
			endpointStats = map[string]interface{}{
				"endpoint_name":         endpointName,
				"total_sent":            0,
				"total_success":         0,
				"total_failed":          0,
				"success_rate":          0.0,
				"average_response_time": "0s",
				"last_sent":             nil,
				"last_success":          nil,
				"last_failure":          nil,
			}
		}

		response := dtos.APIResponse{
			Success: true,
			Message: "엔드포인트별 웹훅 통계 조회 완료",
			Data: map[string]interface{}{
				fieldEndpoint: endpointName,
				"statistics":  endpointStats,
				"time_range":  stats.TimeRange,
			},
		}

		return c.JSON(response)
	}
}

// getRecentActivity 최근 웹훅 활동 조회
func getRecentActivity(historyManager *webhook.WebhookHistoryManager, logger logging.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// 제한 개수 파라미터
		limitStr := c.Query(fieldLimit, "20")
		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit <= 0 || limit > 100 {
			limit = 20
		}

		logger.Debug("최근 웹훅 활동 조회",
			logging.F(fieldLimit, limit))

		if historyManager == nil {
			response := dtos.APIResponse{
				Success: true,
				Message: "최근 웹훅 활동 조회 완료",
				Data: map[string]interface{}{
					"activities": []interface{}{},
					fieldLimit:   limit,
					"note":       "히스토리 매니저가 초기화되지 않음",
				},
			}
			return c.JSON(response)
		}

		// 최근 활동 조회
		activities, err := historyManager.GetRecentActivity(limit)
		if err != nil {
			logger.Error("최근 웹훅 활동 조회 실패", logging.F(fieldError, err))
			response := dtos.APIResponse{
				Success: false,
				Message: "최근 웹훅 활동 조회 실패",
				Error:   err.Error(),
			}
			return c.Status(500).JSON(response)
		}

		response := dtos.APIResponse{
			Success: true,
			Message: "최근 웹훅 활동 조회 완료",
			Data: map[string]interface{}{
				"activities": activities,
				fieldLimit:   limit,
				fieldTotal:   len(activities),
			},
		}

		return c.JSON(response)
	}
}
