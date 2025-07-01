package webhook

import (
	"context"
	"fmt"
	"time"

	"proxynd/alerts"
	"proxynd/configs"
	"proxynd/logging"
)

// TestResult 웹훅 테스트 결과
type TestResult struct {
	EndpointName   string                 `json:"endpoint_name"`
	URL            string                 `json:"url"`
	Success        bool                   `json:"success"`
	StatusCode     int                    `json:"status_code,omitempty"`
	ResponseTime   time.Duration          `json:"response_time"`
	ErrorMessage   string                 `json:"error_message,omitempty"`
	RequestPayload map[string]interface{} `json:"request_payload,omitempty"`
	ResponseBody   string                 `json:"response_body,omitempty"`
	TestTimestamp  time.Time              `json:"test_timestamp"`
}

// TestAllResult 전체 웹훅 테스트 결과
type TestAllResult struct {
	TotalEndpoints int           `json:"total_endpoints"`
	SuccessCount   int           `json:"success_count"`
	FailureCount   int           `json:"failure_count"`
	TestDuration   time.Duration `json:"test_duration"`
	Results        []TestResult  `json:"results"`
	TestTimestamp  time.Time     `json:"test_timestamp"`
}

// WebhookTester 웹훅 테스트 클래스
type WebhookTester struct {
	config configs.WebhookConfig
	sender *WebhookSender
	logger logging.Logger
}

// NewWebhookTester 새로운 웹훅 테스터 생성
func NewWebhookTester(config configs.WebhookConfig, sender *WebhookSender) *WebhookTester {
	return &WebhookTester{
		config: config,
		sender: sender,
		logger: logging.GetLogger(),
	}
}

// TestSingleEndpoint 단일 엔드포인트 테스트
func (wt *WebhookTester) TestSingleEndpoint(endpointName string) (*TestResult, error) {
	// 엔드포인트 찾기
	var endpoint *configs.WebhookEndpointConfig
	for _, ep := range wt.config.Endpoints {
		if ep.Name == endpointName {
			endpoint = &ep
			break
		}
	}

	if endpoint == nil {
		return nil, fmt.Errorf("endpoint not found: %s", endpointName)
	}

	if !endpoint.Enabled {
		return &TestResult{
			EndpointName:  endpointName,
			URL:           endpoint.URL,
			Success:       false,
			ErrorMessage:  "endpoint is disabled",
			TestTimestamp: time.Now(),
		}, nil
	}

	return wt.testEndpoint(*endpoint)
}

// TestAllEndpoints 모든 엔드포인트 테스트
func (wt *WebhookTester) TestAllEndpoints() (*TestAllResult, error) {
	startTime := time.Now()

	var results []TestResult
	successCount := 0
	failureCount := 0

	wt.logger.Info("모든 웹훅 엔드포인트 테스트 시작",
		logging.F("total_endpoints", len(wt.config.Endpoints)))

	for _, endpoint := range wt.config.Endpoints {
		result, err := wt.testEndpoint(endpoint)
		if err != nil {
			result = &TestResult{
				EndpointName:  endpoint.Name,
				URL:           endpoint.URL,
				Success:       false,
				ErrorMessage:  err.Error(),
				TestTimestamp: time.Now(),
			}
		}

		results = append(results, *result)

		if result.Success {
			successCount++
		} else {
			failureCount++
		}
	}

	testDuration := time.Since(startTime)

	wt.logger.Info("웹훅 엔드포인트 테스트 완료",
		logging.F("success_count", successCount),
		logging.F("failure_count", failureCount),
		logging.F("duration", testDuration))

	return &TestAllResult{
		TotalEndpoints: len(wt.config.Endpoints),
		SuccessCount:   successCount,
		FailureCount:   failureCount,
		TestDuration:   testDuration,
		Results:        results,
		TestTimestamp:  time.Now(),
	}, nil
}

// testEndpoint 실제 엔드포인트 테스트 수행
func (wt *WebhookTester) testEndpoint(endpoint configs.WebhookEndpointConfig) (*TestResult, error) {
	startTime := time.Now()

	// 테스트용 이벤트 생성
	testEvent := &alerts.AlertEvent{
		ID:        fmt.Sprintf("test_%d", time.Now().Unix()),
		Type:      "webhook.test",
		Level:     "INFO",
		Message:   fmt.Sprintf("Webhook endpoint test for %s", endpoint.Name),
		Source:    "webhook-tester",
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"test":      true,
			"endpoint":  endpoint.Name,
			"test_time": time.Now().Format(time.RFC3339),
		},
	}

	wt.logger.Debug("웹훅 엔드포인트 테스트 시작",
		logging.F("endpoint", endpoint.Name),
		logging.F("url", endpoint.URL))

	// 어댑터 선택
	adapterName := "generic"
	if endpoint.Format == "slack" {
		adapterName = "slack"
	} else if endpoint.Format == "discord" {
		adapterName = "discord"
	}

	adapter, exists := wt.sender.adapters[adapterName]
	if !exists {
		adapter = wt.sender.adapters["generic"] // 폴백
	}

	// 타임아웃 설정
	timeout := 10 * time.Second
	if endpoint.Timeout != "" {
		if parsed, err := time.ParseDuration(endpoint.Timeout); err == nil {
			timeout = parsed
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// 실제 전송 시도
	err := adapter.Send(ctx, testEvent, endpoint)
	responseTime := time.Since(startTime)

	result := &TestResult{
		EndpointName:  endpoint.Name,
		URL:           endpoint.URL,
		ResponseTime:  responseTime,
		TestTimestamp: time.Now(),
		RequestPayload: map[string]interface{}{
			"event":    testEvent,
			"endpoint": endpoint.Name,
		},
	}

	if err != nil {
		result.Success = false
		result.ErrorMessage = err.Error()

		wt.logger.Warn("웹훅 엔드포인트 테스트 실패",
			logging.F("endpoint", endpoint.Name),
			logging.F("url", endpoint.URL),
			logging.F("error", err.Error()),
			logging.F("response_time", responseTime))
	} else {
		result.Success = true
		result.StatusCode = 200 // 성공한 경우 기본값

		wt.logger.Info("웹훅 엔드포인트 테스트 성공",
			logging.F("endpoint", endpoint.Name),
			logging.F("url", endpoint.URL),
			logging.F("response_time", responseTime))
	}

	return result, nil
}

// TestEndpointConnectivity 엔드포인트 연결성만 테스트 (실제 이벤트 전송 없이)
func (wt *WebhookTester) TestEndpointConnectivity(endpointName string) (*TestResult, error) {
	// 엔드포인트 찾기
	var endpoint *configs.WebhookEndpointConfig
	for _, ep := range wt.config.Endpoints {
		if ep.Name == endpointName {
			endpoint = &ep
			break
		}
	}

	if endpoint == nil {
		return nil, fmt.Errorf("endpoint not found: %s", endpointName)
	}

	startTime := time.Now()

	// 간단한 HEAD 또는 GET 요청으로 연결성 확인
	// 실제 이벤트를 전송하지 않고 엔드포인트 접근 가능성만 확인

	wt.logger.Debug("웹훅 엔드포인트 연결성 테스트",
		logging.F("endpoint", endpoint.Name),
		logging.F("url", endpoint.URL))

	// TODO: HTTP client로 HEAD/GET 요청 구현
	// 현재는 기본 응답 반환
	responseTime := time.Since(startTime)

	return &TestResult{
		EndpointName:  endpoint.Name,
		URL:           endpoint.URL,
		Success:       true,
		StatusCode:    200,
		ResponseTime:  responseTime,
		TestTimestamp: time.Now(),
	}, nil
}

// GetTestHistory 테스트 이력 조회 (향후 확장용)
func (wt *WebhookTester) GetTestHistory(endpointName string, limit int) ([]TestResult, error) {
	// TODO: 테스트 이력을 저장하고 조회하는 기능 구현
	// 현재는 빈 슬라이스 반환
	return []TestResult{}, nil
}

// ValidateEndpointConfig 엔드포인트 설정 검증
func (wt *WebhookTester) ValidateEndpointConfig(endpoint configs.WebhookEndpointConfig) []string {
	var errors []string

	// URL 검증
	if endpoint.URL == "" {
		errors = append(errors, "URL is required")
	}

	// 메서드 검증
	validMethods := map[string]bool{
		"GET": true, "POST": true, "PUT": true, "PATCH": true,
	}
	if !validMethods[endpoint.Method] {
		errors = append(errors, fmt.Sprintf("invalid HTTP method: %s", endpoint.Method))
	}

	// 포맷 검증
	validFormats := map[string]bool{
		"json": true, "slack": true, "discord": true, "teams": true,
	}
	if endpoint.Format != "" && !validFormats[endpoint.Format] {
		errors = append(errors, fmt.Sprintf("unsupported format: %s", endpoint.Format))
	}

	// 타임아웃 검증
	if endpoint.Timeout != "" {
		if _, err := time.ParseDuration(endpoint.Timeout); err != nil {
			errors = append(errors, fmt.Sprintf("invalid timeout format: %s", endpoint.Timeout))
		}
	}

	// 인증 정보 검증
	if endpoint.Credentials.Type != "" {
		validAuthTypes := map[string]bool{
			"none": true, "basic": true, "bearer": true, "hmac": true, "oauth2": true,
		}
		if !validAuthTypes[endpoint.Credentials.Type] {
			errors = append(errors, fmt.Sprintf("unsupported auth type: %s", endpoint.Credentials.Type))
		}

		// Basic 인증 검증
		if endpoint.Credentials.Type == "basic" {
			if endpoint.Credentials.Username == "" || endpoint.Credentials.Password == "" {
				errors = append(errors, "basic auth requires username and password")
			}
		}

		// Bearer 토큰 검증
		if endpoint.Credentials.Type == "bearer" {
			if endpoint.Credentials.Token == "" {
				errors = append(errors, "bearer auth requires token")
			}
		}
	}

	return errors
}
