package webhook

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"proxynd/internal/alerts"
	"proxynd/internal/config"
	"proxynd/internal/logging"
)

// Additional webhook type constants not in constants.go
const (
	webhookTypeDiscord = "discord"
	webhookTypeGeneric = "generic"
	webhookTypeTeams   = "teams"
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
	config      config.WebhookConfig
	sender      *WebhookSender
	logger      logging.Logger
	httpClient  *http.Client
	testHistory map[string][]TestResult // endpoint name -> test results
	historyMu   sync.RWMutex            // protects testHistory
	maxHistory  int                     // maximum history entries per endpoint
}

// NewWebhookTester 새로운 웹훅 테스터 생성
func NewWebhookTester(config config.WebhookConfig, sender *WebhookSender) *WebhookTester {
	return &WebhookTester{
		config: config,
		sender: sender,
		logger: logging.GetLogger(),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
				return http.ErrUseLastResponse // Don't follow redirects
			},
		},
		testHistory: make(map[string][]TestResult),
		maxHistory:  100, // Store up to 100 test results per endpoint
	}
}

// TestSingleEndpoint 단일 엔드포인트 테스트
func (wt *WebhookTester) TestSingleEndpoint(endpointName string) (*TestResult, error) {
	// 엔드포인트 찾기
	var endpoint *config.WebhookEndpointConfig
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
func (wt *WebhookTester) testEndpoint(endpoint config.WebhookEndpointConfig) (*TestResult, error) {
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
		logging.F(fieldEndpoint, endpoint.Name),
		logging.F("url", endpoint.URL))

	// 어댑터 선택
	adapterName := webhookTypeGeneric
	switch endpoint.Format {
	case webhookTypeSlack:
		adapterName = webhookTypeSlack
	case webhookTypeDiscord:
		adapterName = webhookTypeDiscord
	}

	adapter, exists := wt.sender.adapters[adapterName]
	if !exists {
		adapter = wt.sender.adapters[webhookTypeGeneric] // 폴백
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
	err := adapter.Send(ctx, endpoint.URL, testEvent)
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
			logging.F(fieldEndpoint, endpoint.Name),
			logging.F("url", endpoint.URL),
			logging.F("error", err.Error()),
			logging.F(fieldResponseTime, responseTime))
	} else {
		result.Success = true
		result.StatusCode = 200 // 성공한 경우 기본값

		wt.logger.Info("웹훅 엔드포인트 테스트 성공",
			logging.F(fieldEndpoint, endpoint.Name),
			logging.F("url", endpoint.URL),
			logging.F(fieldResponseTime, responseTime))
	}

	// Store result in history
	wt.addTestResult(endpoint.Name, *result)

	return result, nil
}

// TestEndpointConnectivity 엔드포인트 연결성만 테스트 (실제 이벤트 전송 없이)
func (wt *WebhookTester) TestEndpointConnectivity(endpointName string) (*TestResult, error) {
	// 엔드포인트 찾기
	var endpoint *config.WebhookEndpointConfig
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

	wt.logger.Debug("웹훅 엔드포인트 연결성 테스트",
		logging.F(fieldEndpoint, endpoint.Name),
		logging.F("url", endpoint.URL))

	// Create HTTP request with timeout
	timeout := 10 * time.Second
	if endpoint.Timeout != "" {
		if parsed, err := time.ParseDuration(endpoint.Timeout); err == nil {
			timeout = parsed
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Try HEAD request first (lightweight), fallback to GET
	var req *http.Request
	var err error

	req, err = http.NewRequestWithContext(ctx, http.MethodHead, endpoint.URL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add custom headers if configured
	req.Header.Set("User-Agent", "ProxyND-Webhook-Tester/1.0")
	for key, value := range endpoint.Headers {
		req.Header.Set(key, value)
	}

	// Send request
	resp, err := wt.httpClient.Do(req)
	responseTime := time.Since(startTime)

	result := &TestResult{
		EndpointName:  endpoint.Name,
		URL:           endpoint.URL,
		ResponseTime:  responseTime,
		TestTimestamp: time.Now(),
	}

	if err != nil {
		result.Success = false
		result.ErrorMessage = err.Error()

		wt.logger.Warn("웹훅 엔드포인트 연결성 테스트 실패",
			logging.F(fieldEndpoint, endpoint.Name),
			logging.F("url", endpoint.URL),
			logging.F("error", err.Error()),
			logging.F(fieldResponseTime, responseTime))

		// Store failed result
		wt.addTestResult(endpoint.Name, *result)
		return result, nil
	}
	defer resp.Body.Close()

	result.StatusCode = resp.StatusCode
	result.Success = resp.StatusCode >= 200 && resp.StatusCode < 400

	if !result.Success {
		result.ErrorMessage = fmt.Sprintf("unexpected status code: %d %s",
			resp.StatusCode, http.StatusText(resp.StatusCode))

		wt.logger.Warn("웹훅 엔드포인트 연결성 테스트 실패",
			logging.F(fieldEndpoint, endpoint.Name),
			logging.F("url", endpoint.URL),
			logging.F("status_code", resp.StatusCode),
			logging.F(fieldResponseTime, responseTime))
	} else {
		wt.logger.Info("웹훅 엔드포인트 연결성 테스트 성공",
			logging.F(fieldEndpoint, endpoint.Name),
			logging.F("url", endpoint.URL),
			logging.F("status_code", resp.StatusCode),
			logging.F(fieldResponseTime, responseTime))
	}

	// Store result in history
	wt.addTestResult(endpoint.Name, *result)

	return result, nil
}

// GetTestHistory 테스트 이력 조회
// endpointName: 특정 엔드포인트의 이력 조회, 빈 문자열이면 모든 엔드포인트
// limit: 반환할 최대 결과 수, 0이면 모든 결과 반환
func (wt *WebhookTester) GetTestHistory(endpointName string, limit int) ([]TestResult, error) {
	wt.historyMu.RLock()
	defer wt.historyMu.RUnlock()

	var results []TestResult

	if endpointName != "" {
		// Get history for specific endpoint
		history, exists := wt.testHistory[endpointName]
		if !exists {
			return []TestResult{}, nil
		}

		if limit > 0 && len(history) > limit {
			// Return most recent 'limit' entries
			results = make([]TestResult, limit)
			copy(results, history[len(history)-limit:])
		} else {
			results = make([]TestResult, len(history))
			copy(results, history)
		}
	} else {
		// Get history for all endpoints
		for _, history := range wt.testHistory {
			results = append(results, history...)
		}

		// Sort by timestamp (newest first)
		// Using simple sort - in production might want to use sort.Slice
		for i := 0; i < len(results)-1; i++ {
			for j := i + 1; j < len(results); j++ {
				if results[i].TestTimestamp.Before(results[j].TestTimestamp) {
					results[i], results[j] = results[j], results[i]
				}
			}
		}

		if limit > 0 && len(results) > limit {
			results = results[:limit]
		}
	}

	return results, nil
}

// ValidateEndpointConfig 엔드포인트 설정 검증
func (wt *WebhookTester) ValidateEndpointConfig(endpoint config.WebhookEndpointConfig) []string {
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
		"json": true, webhookTypeSlack: true, webhookTypeDiscord: true, webhookTypeTeams: true,
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

// addTestResult 테스트 결과를 히스토리에 추가
func (wt *WebhookTester) addTestResult(endpointName string, result TestResult) {
	wt.historyMu.Lock()
	defer wt.historyMu.Unlock()

	history := wt.testHistory[endpointName]
	history = append(history, result)

	// Maintain maximum history size (FIFO)
	if len(history) > wt.maxHistory {
		history = history[1:] // Remove oldest entry
	}

	wt.testHistory[endpointName] = history

	wt.logger.Debug("테스트 결과 저장",
		logging.F(fieldEndpoint, endpointName),
		logging.F("history_size", len(history)),
		logging.F("success", result.Success))
}

// ClearHistory 특정 엔드포인트 또는 전체 테스트 히스토리 삭제
func (wt *WebhookTester) ClearHistory(endpointName string) {
	wt.historyMu.Lock()
	defer wt.historyMu.Unlock()

	if endpointName != "" {
		delete(wt.testHistory, endpointName)
		wt.logger.Info("엔드포인트 테스트 히스토리 삭제",
			logging.F(fieldEndpoint, endpointName))
	} else {
		wt.testHistory = make(map[string][]TestResult)
		wt.logger.Info("전체 테스트 히스토리 삭제")
	}
}

// GetTestStatistics 테스트 통계 조회
func (wt *WebhookTester) GetTestStatistics(endpointName string) map[string]interface{} {
	wt.historyMu.RLock()
	defer wt.historyMu.RUnlock()

	stats := make(map[string]interface{})

	if endpointName != "" {
		// Statistics for specific endpoint
		history, exists := wt.testHistory[endpointName]
		if !exists {
			return stats
		}

		successCount := 0
		totalResponseTime := time.Duration(0)

		for _, result := range history {
			if result.Success {
				successCount++
			}
			totalResponseTime += result.ResponseTime
		}

		stats["endpoint_name"] = endpointName
		stats["total_tests"] = len(history)
		stats["success_count"] = successCount
		stats["failure_count"] = len(history) - successCount
		stats["success_rate"] = float64(successCount) / float64(len(history)) * 100

		if len(history) > 0 {
			stats["avg_response_time"] = totalResponseTime / time.Duration(len(history))
			stats["last_test"] = history[len(history)-1].TestTimestamp
			stats["last_success"] = history[len(history)-1].Success
		}
	} else {
		// Statistics for all endpoints
		totalTests := 0
		totalSuccess := 0
		endpointCount := len(wt.testHistory)

		for _, history := range wt.testHistory {
			totalTests += len(history)
			for _, result := range history {
				if result.Success {
					totalSuccess++
				}
			}
		}

		stats["total_endpoints"] = endpointCount
		stats["total_tests"] = totalTests
		stats["success_count"] = totalSuccess
		stats["failure_count"] = totalTests - totalSuccess

		if totalTests > 0 {
			stats["success_rate"] = float64(totalSuccess) / float64(totalTests) * 100
		}
	}

	return stats
}
