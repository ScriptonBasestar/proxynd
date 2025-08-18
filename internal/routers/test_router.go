package routers

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"proxynd/internal/adapters/pm/common"
	"proxynd/logging"
)

// ProxyTestRequest 프록시 테스트 요청 구조체
type ProxyTestRequest struct {
	ProxyType string            `json:"proxy_type"`
	Target    string            `json:"target,omitempty"`
	Timeout   int               `json:"timeout,omitempty"`
	Options   map[string]string `json:"options,omitempty"`
}

// ProxyTestResult 프록시 테스트 결과 구조체
type ProxyTestResult struct {
	ProxyType    string                 `json:"proxy_type"`
	Status       string                 `json:"status"`
	Message      string                 `json:"message"`
	ResponseTime float64                `json:"response_time_ms"`
	StatusCode   int                    `json:"status_code,omitempty"`
	Details      map[string]interface{} `json:"details,omitempty"`
	Timestamp    time.Time              `json:"timestamp"`
}

// ProxyTestResponse 프록시 테스트 응답 구조체
type ProxyTestResponse struct {
	Results   []ProxyTestResult `json:"results"`
	Summary   TestSummary       `json:"summary"`
	Timestamp time.Time         `json:"timestamp"`
}

// TestSummary 테스트 요약
type TestSummary struct {
	Total   int `json:"total"`
	Passed  int `json:"passed"`
	Failed  int `json:"failed"`
	Skipped int `json:"skipped"`
}

// TestRouter 프록시 테스트 API 라우터 설정
func TestRouter(app *fiber.App) {
	api := app.Group("/api/test")

	// 개별 프록시 테스트
	api.Post("/:proxy_type", testProxyType)

	// 전체 프록시 테스트
	api.Post("/all", testAllProxies)

	// 프록시 연결성 검사
	api.Get("/connectivity/:proxy_type", checkProxyConnectivity)

	// 지원되는 프록시 타입 목록
	api.Get("/types", getSupportedProxyTypes)
}

// testProxyType 특정 프록시 타입 테스트 핸들러
func testProxyType(c *fiber.Ctx) error {
	logger := logging.GetLogger()
	proxyType := c.Params("proxy_type")

	// 요청 검증
	if !isValidProxyType(proxyType) {
		return c.Status(fiber.StatusBadRequest).SendString("지원하지 않는 프록시 타입입니다")
	}

	var req ProxyTestRequest
	if err := c.BodyParser(&req); err != nil {
		// 바디 파싱에 실패해도 기본값으로 진행
		req.ProxyType = proxyType
		req.Timeout = 30 // 기본 타임아웃 30초
	}

	// ProxyType이 없으면 URL 파라미터에서 가져오기
	if req.ProxyType == "" {
		req.ProxyType = proxyType
	}

	// 기본 타임아웃 설정
	if req.Timeout <= 0 {
		req.Timeout = 30
	}

	// 테스트 수행
	result := performProxyTest(proxyType, req)

	logger.Info("Proxy test performed", logging.F("proxy_type", proxyType), logging.F("status", result.Status))

	return c.JSON(result)
}

// testAllProxies 모든 프록시 테스트 핸들러
func testAllProxies(c *fiber.Ctx) error {
	logger := logging.GetLogger()

	var req ProxyTestRequest
	if err := c.BodyParser(&req); err != nil {
		// 바디 파싱에 실패해도 기본값으로 진행
		req.Timeout = 30 // 기본 타임아웃
	}

	// 기본 타임아웃 설정
	if req.Timeout <= 0 {
		req.Timeout = 30
	}

	supportedTypes := getSupportedTypes()
	results := make([]ProxyTestResult, 0, len(supportedTypes))
	summary := TestSummary{Total: len(supportedTypes)}

	// 각 프록시 타입에 대해 테스트 수행
	for _, proxyType := range supportedTypes {
		result := performProxyTest(proxyType, req)
		results = append(results, result)

		// 요약 통계 업데이트
		switch result.Status {
		case statusPassed:
			summary.Passed++
		case statusFailed:
			summary.Failed++
		case "skipped":
			summary.Skipped++
		}
	}

	response := ProxyTestResponse{
		Results:   results,
		Summary:   summary,
		Timestamp: time.Now(),
	}

	logger.Info("All proxy tests completed",
		logging.F("total", summary.Total),
		logging.F(statusPassed, summary.Passed),
		logging.F(statusFailed, summary.Failed))

	return c.JSON(response)
}

// checkProxyConnectivity 프록시 연결성 검사 핸들러
func checkProxyConnectivity(c *fiber.Ctx) error {
	logger := logging.GetLogger()
	proxyType := c.Params("proxy_type")

	if !isValidProxyType(proxyType) {
		return c.Status(fiber.StatusBadRequest).SendString("지원하지 않는 프록시 타입입니다")
	}

	// 연결성 검사 수행
	result := checkConnectivity(proxyType)

	logger.Info("Connectivity check performed", logging.F("proxy_type", proxyType))

	return c.JSON(result)
}

// getSupportedProxyTypes 지원되는 프록시 타입 목록 핸들러
func getSupportedProxyTypes(c *fiber.Ctx) error {
	supportedTypes := getSupportedTypes()

	return c.JSON(fiber.Map{
		"proxy_types": supportedTypes,
		"total":       len(supportedTypes),
		"timestamp":   time.Now(),
	})
}

// performProxyTest 프록시 테스트 수행
func performProxyTest(proxyType string, req ProxyTestRequest) ProxyTestResult {
	start := time.Now()
	result := ProxyTestResult{
		ProxyType: proxyType,
		Timestamp: time.Now(),
		Details:   make(map[string]interface{}),
	}

	// 설정 확인
	if !isProxyConfigured(proxyType) {
		result.Status = "skipped"
		result.Message = fmt.Sprintf("%s 프록시 설정이 없습니다", proxyType)
		result.ResponseTime = float64(time.Since(start).Nanoseconds()) / 1e6
		return result
	}

	// 프록시 타입별 테스트 수행
	switch proxyType {
	case common.PMTypeApt:
		result = testAptProxy(req)
	case common.PMTypeNpm:
		result = testNpmProxy(req)
	case "maven":
		result = testMavenProxy(req)
	case "pip":
		result = testPipProxy(req)
	case "docker":
		result = testDockerProxy(req)
	case "yum":
		result = testYumProxy(req)
	case "apk":
		result = testApkProxy(req)
	default:
		result.Status = statusFailed
		result.Message = fmt.Sprintf("지원하지 않는 프록시 타입: %s", proxyType)
	}

	result.ProxyType = proxyType
	result.ResponseTime = float64(time.Since(start).Nanoseconds()) / 1e6
	result.Timestamp = time.Now()

	return result
}

// checkConnectivity 연결성 검사
func checkConnectivity(proxyType string) ProxyTestResult {
	result := ProxyTestResult{
		ProxyType: proxyType,
		Timestamp: time.Now(),
		Details:   make(map[string]interface{}),
	}

	// 프록시 엔드포인트 URL 구성
	baseURL := getServerBaseURL()
	proxyURL := fmt.Sprintf("%s/proxy/%s/", baseURL, proxyType)

	// HTTP GET 요청으로 연결성 확인
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(proxyURL)
	if err != nil {
		result.Status = statusFailed
		result.Message = fmt.Sprintf("연결 실패: %v", err)
		return result
	}
	defer func() { _ = resp.Body.Close() }()

	result.StatusCode = resp.StatusCode
	result.Details["headers"] = resp.Header

	if resp.StatusCode < 500 {
		result.Status = statusPassed
		result.Message = "연결 성공"
	} else {
		result.Status = statusFailed
		result.Message = fmt.Sprintf("서버 오류: HTTP %d", resp.StatusCode)
	}

	return result
}

// 프록시별 테스트 함수들

func testAptProxy(req ProxyTestRequest) ProxyTestResult {
	result := ProxyTestResult{Status: statusPassed, Message: "APT 프록시 테스트 성공"}

	// 실제 APT 패키지 인덱스 파일 요청 테스트
	testURL := "/proxy/apt/dists/jammy/Release"
	if req.Target != "" {
		testURL = req.Target
	}

	statusCode, err := makeTestRequest(methodGET, testURL, req.Timeout)
	result.StatusCode = statusCode

	if err != nil {
		result.Status = statusFailed
		result.Message = fmt.Sprintf("APT 프록시 테스트 실패: %v", err)
	}

	result.Details["test_url"] = testURL
	return result
}

func testNpmProxy(req ProxyTestRequest) ProxyTestResult {
	result := ProxyTestResult{Status: statusPassed, Message: "NPM 프록시 테스트 성공"}

	// NPM 레지스트리 정보 요청 테스트
	testURL := "/proxy/npm/"
	if req.Target != "" {
		testURL = req.Target
	}

	statusCode, err := makeTestRequest(methodGET, testURL, req.Timeout)
	result.StatusCode = statusCode

	if err != nil {
		result.Status = statusFailed
		result.Message = fmt.Sprintf("NPM 프록시 테스트 실패: %v", err)
	}

	result.Details["test_url"] = testURL
	return result
}

func testMavenProxy(req ProxyTestRequest) ProxyTestResult {
	result := ProxyTestResult{Status: statusPassed, Message: "Maven 프록시 테스트 성공"}

	// Maven 중앙 저장소 메타데이터 요청 테스트
	testURL := "/proxy/maven/maven-metadata.xml"
	if req.Target != "" {
		testURL = req.Target
	}

	statusCode, err := makeTestRequest(methodGET, testURL, req.Timeout)
	result.StatusCode = statusCode

	if err != nil {
		result.Status = statusFailed
		result.Message = fmt.Sprintf("Maven 프록시 테스트 실패: %v", err)
	}

	result.Details["test_url"] = testURL
	return result
}

func testPipProxy(req ProxyTestRequest) ProxyTestResult {
	result := ProxyTestResult{Status: statusPassed, Message: "Pip 프록시 테스트 성공"}

	// PyPI 인덱스 요청 테스트
	testURL := "/proxy/pip/simple/"
	if req.Target != "" {
		testURL = req.Target
	}

	statusCode, err := makeTestRequest(methodGET, testURL, req.Timeout)
	result.StatusCode = statusCode

	if err != nil {
		result.Status = statusFailed
		result.Message = fmt.Sprintf("Pip 프록시 테스트 실패: %v", err)
	}

	result.Details["test_url"] = testURL
	return result
}

func testDockerProxy(req ProxyTestRequest) ProxyTestResult {
	result := ProxyTestResult{Status: statusPassed, Message: "Docker 프록시 테스트 성공"}

	// Docker 레지스트리 v2 API 테스트
	testURL := "/proxy/docker/v2/"
	if req.Target != "" {
		testURL = req.Target
	}

	statusCode, err := makeTestRequest(methodGET, testURL, req.Timeout)
	result.StatusCode = statusCode

	if err != nil {
		result.Status = statusFailed
		result.Message = fmt.Sprintf("Docker 프록시 테스트 실패: %v", err)
	}

	result.Details["test_url"] = testURL
	return result
}

func testYumProxy(req ProxyTestRequest) ProxyTestResult {
	result := ProxyTestResult{Status: statusPassed, Message: "YUM 프록시 테스트 성공"}

	// YUM 저장소 메타데이터 요청 테스트
	testURL := "/proxy/yum/repodata/repomd.xml"
	if req.Target != "" {
		testURL = req.Target
	}

	statusCode, err := makeTestRequest(methodGET, testURL, req.Timeout)
	result.StatusCode = statusCode

	if err != nil {
		result.Status = statusFailed
		result.Message = fmt.Sprintf("YUM 프록시 테스트 실패: %v", err)
	}

	result.Details["test_url"] = testURL
	return result
}

func testApkProxy(req ProxyTestRequest) ProxyTestResult {
	result := ProxyTestResult{Status: statusPassed, Message: "APK 프록시 테스트 성공"}

	// Alpine APK 인덱스 요청 테스트
	testURL := "/proxy/apk/main/APKINDEX.tar.gz"
	if req.Target != "" {
		testURL = req.Target
	}

	statusCode, err := makeTestRequest(methodGET, testURL, req.Timeout)
	result.StatusCode = statusCode

	if err != nil {
		result.Status = statusFailed
		result.Message = fmt.Sprintf("APK 프록시 테스트 실패: %v", err)
	}

	result.Details["test_url"] = testURL
	return result
}

// 유틸리티 함수들

func getSupportedTypes() []string {
	return []string{"apt", "npm", "maven", "pip", "docker", "yum", "apk"}
}

func isValidProxyType(proxyType string) bool {
	supportedTypes := getSupportedTypes()
	for _, t := range supportedTypes {
		if t == proxyType {
			return true
		}
	}
	return false
}

func isProxyConfigured(proxyType string) bool {
	configDir := getConfigDir()
	configFile := fmt.Sprintf("%s/%s-proxy.yaml", configDir, proxyType)

	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		return false
	}

	return true
}

func getConfigDir() string {
	if configDir := os.Getenv("CONFIG_DIR"); configDir != "" {
		return configDir
	}
	return "/config"
}

func getServerBaseURL() string {
	if baseURL := os.Getenv("SERVER_BASE_URL"); baseURL != "" {
		return baseURL
	}
	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8080"
	}
	return fmt.Sprintf("http://localhost:%s", port)
}

func makeTestRequest(method, path string, timeoutSec int) (int, error) {
	if timeoutSec <= 0 {
		timeoutSec = 30
	}

	client := &http.Client{
		Timeout: time.Duration(timeoutSec) * time.Second,
	}

	url := getServerBaseURL() + path
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return 0, err
	}

	resp, err := client.Do(req)
	if err != nil {
		// 연결 실패도 유효한 테스트 결과로 처리
		if strings.Contains(err.Error(), "connection refused") {
			return 0, fmt.Errorf("connection refused - server not running")
		}
		return 0, err
	}
	defer func() { _ = resp.Body.Close() }()

	return resp.StatusCode, nil
}
