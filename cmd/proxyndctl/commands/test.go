package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
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

// SupportedTypesResponse 지원되는 프록시 타입 응답
type SupportedTypesResponse struct {
	ProxyTypes []string  `json:"proxy_types"`
	Total      int       `json:"total"`
	Timestamp  time.Time `json:"timestamp"`
}

// NewTestCmd 프록시 테스트 명령어 생성
func NewTestCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "test",
		Short: "프록시 기능 테스트 명령어",
		Long:  "ProxyND 서버의 프록시 기능을 테스트합니다. 각 패키지 매니저별 연결성을 확인합니다.",
	}

	cmd.AddCommand(newTestAllCmd())
	cmd.AddCommand(newTestParallelCmd())
	cmd.AddCommand(newTestConnectivityCmd())
	cmd.AddCommand(newTestTypesCmd())

	// 개별 프록시 테스트 명령어들
	cmd.AddCommand(newTestAptCmd())
	cmd.AddCommand(newTestNpmCmd())
	cmd.AddCommand(newTestMavenCmd())
	cmd.AddCommand(newTestPipCmd())
	cmd.AddCommand(newTestDockerCmd())
	cmd.AddCommand(newTestYumCmd())
	cmd.AddCommand(newTestApkCmd())

	return cmd
}

// newTestAllCmd 전체 프록시 테스트 명령어
func newTestAllCmd() *cobra.Command {
	var (
		timeout     int
		showDetails bool
	)

	cmd := &cobra.Command{
		Use:   "all",
		Short: "모든 프록시 테스트",
		Long:  "모든 지원되는 프록시 타입에 대해 테스트를 수행합니다.",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runTestAll(timeout, showDetails)
		},
	}

	cmd.Flags().IntVarP(&timeout, "timeout", "t", 30, "테스트 타임아웃 (초)")
	cmd.Flags().BoolVarP(&showDetails, "details", "d", false, "상세 정보 표시")

	return cmd
}

// newTestConnectivityCmd 연결성 테스트 명령어
func newTestConnectivityCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "connectivity <proxy_type>",
		Short: "프록시 연결성 테스트",
		Long:  "지정된 프록시 타입의 기본 연결성을 테스트합니다.",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runTestConnectivity(args[0])
		},
	}

	return cmd
}

// newTestTypesCmd 지원되는 프록시 타입 조회 명령어
func newTestTypesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "types",
		Short: "지원되는 프록시 타입 조회",
		Long:  "ProxyND에서 지원하는 모든 프록시 타입 목록을 조회합니다.",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runTestTypes()
		},
	}

	return cmd
}

// 개별 프록시 테스트 명령어들

func newTestAptCmd() *cobra.Command {
	return newProxyTestCmd("apt", "APT (Ubuntu/Debian)")
}

func newTestNpmCmd() *cobra.Command {
	return newProxyTestCmd("npm", "NPM (Node.js)")
}

func newTestMavenCmd() *cobra.Command {
	return newProxyTestCmd("maven", "Maven (Java)")
}

func newTestPipCmd() *cobra.Command {
	return newProxyTestCmd("pip", "Pip (Python)")
}

func newTestDockerCmd() *cobra.Command {
	return newProxyTestCmd("docker", "Docker Registry")
}

func newTestYumCmd() *cobra.Command {
	return newProxyTestCmd("yum", "YUM (RHEL/CentOS)")
}

func newTestApkCmd() *cobra.Command {
	return newProxyTestCmd("apk", "APK (Alpine)")
}

// newProxyTestCmd 개별 프록시 테스트 명령어 생성 (공통)
func newProxyTestCmd(proxyType, description string) *cobra.Command {
	var (
		target  string
		timeout int
	)

	cmd := &cobra.Command{
		Use:   proxyType,
		Short: fmt.Sprintf("%s 프록시 테스트", description),
		Long:  fmt.Sprintf("%s 프록시의 기능을 테스트합니다.", description),
		RunE: func(_ *cobra.Command, _ []string) error {
			return runProxyTest(proxyType, target, timeout)
		},
	}

	cmd.Flags().StringVarP(&target, "target", "u", "", "테스트할 특정 경로")
	cmd.Flags().IntVarP(&timeout, "timeout", "t", 30, "테스트 타임아웃 (초)")

	return cmd
}

// runTestAll 전체 프록시 테스트 실행
func runTestAll(timeout int, showDetails bool) error {
	serverURL := getServerURL()
	apiURL := fmt.Sprintf("%s/api/test/all", serverURL)

	// 요청 데이터 준비
	req := ProxyTestRequest{
		Timeout: timeout,
	}

	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("요청 데이터 인코딩 실패: %v", err)
	}

	// 최적화된 HTTP 클라이언트 사용
	client := GetHTTPClient()
	httpReq, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("요청 생성 실패: %v", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(httpReq)
	if err != nil {
		return fmt.Errorf(errServerConnection, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf(errAPIRequest, resp.StatusCode)
	}

	// 응답 파싱
	var result ProxyTestResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf(errResponseParsing, err)
	}

	// 결과 출력
	outputFormat := getOutputFormat()
	switch outputFormat {
	case outputFormatJSON:
		return outputJSON(result)
	case outputFormatYAML:
		return outputYAML(result)
	default:
		return outputTestResultsTable(result, showDetails)
	}
}

// runTestConnectivity 연결성 테스트 실행
func runTestConnectivity(proxyType string) error {
	serverURL := getServerURL()
	apiURL := fmt.Sprintf("%s/api/test/connectivity/%s", serverURL, proxyType)

	// 최적화된 HTTP 클라이언트 사용
	client := GetHTTPClient()
	resp, err := client.Get(apiURL)
	if err != nil {
		return fmt.Errorf(errServerConnection, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf(errAPIRequest, resp.StatusCode)
	}

	// 응답 파싱
	var result ProxyTestResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf(errResponseParsing, err)
	}

	// 결과 출력
	outputFormat := getOutputFormat()
	switch outputFormat {
	case outputFormatJSON:
		return outputJSON(result)
	case outputFormatYAML:
		return outputYAML(result)
	default:
		return outputSingleTestResult(result)
	}
}

// runTestTypes 지원되는 프록시 타입 조회 실행
func runTestTypes() error {
	serverURL := getServerURL()
	apiURL := fmt.Sprintf("%s/api/test/types", serverURL)

	// 최적화된 HTTP 클라이언트 사용
	client := GetHTTPClient()
	resp, err := client.Get(apiURL)
	if err != nil {
		return fmt.Errorf(errServerConnection, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf(errAPIRequest, resp.StatusCode)
	}

	// 응답 파싱
	var result SupportedTypesResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf(errResponseParsing, err)
	}

	// 결과 출력
	outputFormat := getOutputFormat()
	switch outputFormat {
	case outputFormatJSON:
		return outputJSON(result)
	case outputFormatYAML:
		return outputYAML(result)
	default:
		return outputSupportedTypesTable(result)
	}
}

// runProxyTest 개별 프록시 테스트 실행
func runProxyTest(proxyType, target string, timeout int) error {
	serverURL := getServerURL()
	apiURL := fmt.Sprintf("%s/api/test/%s", serverURL, proxyType)

	// 요청 데이터 준비
	req := ProxyTestRequest{
		ProxyType: proxyType,
		Target:    target,
		Timeout:   timeout,
	}

	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("요청 데이터 인코딩 실패: %v", err)
	}

	// 최적화된 HTTP 클라이언트 사용
	client := GetHTTPClient()
	httpReq, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("요청 생성 실패: %v", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(httpReq)
	if err != nil {
		return fmt.Errorf(errServerConnection, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf(errAPIRequest, resp.StatusCode)
	}

	// 응답 파싱
	var result ProxyTestResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf(errResponseParsing, err)
	}

	// 결과 출력
	outputFormat := getOutputFormat()
	switch outputFormat {
	case outputFormatJSON:
		return outputJSON(result)
	case outputFormatYAML:
		return outputYAML(result)
	default:
		return outputSingleTestResult(result)
	}
}

// outputTestResultsTable 테스트 결과를 테이블 형태로 출력
func outputTestResultsTable(result ProxyTestResponse, showDetails bool) error {
	fmt.Printf("🧪 프록시 테스트 결과\n")
	fmt.Printf("테스트 시간: %s\n\n", result.Timestamp.Format(dateTimeFormat))

	// 요약 정보
	summary := result.Summary
	fmt.Printf("📊 요약:\n")
	fmt.Printf("  전체: %d개\n", summary.Total)
	fmt.Printf("  성공: %d개\n", summary.Passed)
	fmt.Printf("  실패: %d개\n", summary.Failed)
	fmt.Printf("  건너뜀: %d개\n\n", summary.Skipped)

	// 개별 테스트 결과
	if len(result.Results) > 0 {
		fmt.Println("🔍 개별 테스트 결과:")
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

		if showDetails {
			_, _ = fmt.Fprintln(w, "타입\t상태\t응답시간\tHTTP코드\t메시지")
			_, _ = fmt.Fprintln(w, "----\t----\t--------\t-------\t------")

			for _, test := range result.Results {
				status := getStatusIcon(test.Status)
				responseTime := fmt.Sprintf("%.1fms", test.ResponseTime)
				httpCode := "-"
				if test.StatusCode > 0 {
					httpCode = fmt.Sprintf("%d", test.StatusCode)
				}

				_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
					test.ProxyType,
					status,
					responseTime,
					httpCode,
					test.Message,
				)
			}
		} else {
			_, _ = fmt.Fprintln(w, "타입\t상태\t응답시간\t메시지")
			_, _ = fmt.Fprintln(w, "----\t----\t--------\t------")

			for _, test := range result.Results {
				status := getStatusIcon(test.Status)
				responseTime := fmt.Sprintf("%.1fms", test.ResponseTime)

				_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
					test.ProxyType,
					status,
					responseTime,
					test.Message,
				)
			}
		}

		_ = w.Flush()
	}

	return nil
}

// outputSingleTestResult 단일 테스트 결과를 출력
func outputSingleTestResult(result ProxyTestResult) error {
	status := getStatusIcon(result.Status)

	fmt.Printf("🧪 %s 프록시 테스트 결과\n\n", strings.ToUpper(result.ProxyType))
	fmt.Printf("상태: %s (%s)\n", status, result.Status)
	fmt.Printf("메시지: %s\n", result.Message)
	fmt.Printf("응답 시간: %.1f ms\n", result.ResponseTime)

	if result.StatusCode > 0 {
		fmt.Printf("HTTP 상태 코드: %d\n", result.StatusCode)
	}

	fmt.Printf("테스트 시간: %s\n", result.Timestamp.Format(dateTimeFormat))

	// 상세 정보 출력 (있는 경우)
	if len(result.Details) > 0 {
		fmt.Printf("\n📋 상세 정보:\n")
		for key, value := range result.Details {
			fmt.Printf("  %s: %v\n", key, value)
		}
	}

	return nil
}

// outputSupportedTypesTable 지원되는 프록시 타입을 테이블 형태로 출력
func outputSupportedTypesTable(result SupportedTypesResponse) error {
	fmt.Printf("📦 지원되는 프록시 타입 (총 %d개)\n", result.Total)
	fmt.Printf("조회 시간: %s\n\n", result.Timestamp.Format(dateTimeFormat))

	if len(result.ProxyTypes) == 0 {
		fmt.Println("지원되는 프록시 타입이 없습니다.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "프록시 타입\t설명")
	_, _ = fmt.Fprintln(w, "----------\t----")

	descriptions := map[string]string{
		"apt":    "APT (Ubuntu/Debian 패키지 매니저)",
		"npm":    "NPM (Node.js 패키지 매니저)",
		"maven":  "Maven (Java 빌드/의존성 관리 도구)",
		"pip":    "Pip (Python 패키지 매니저)",
		"docker": "Docker Registry (컨테이너 이미지 저장소)",
		"yum":    "YUM (RHEL/CentOS 패키지 매니저)",
		"apk":    "APK (Alpine Linux 패키지 매니저)",
	}

	for _, proxyType := range result.ProxyTypes {
		description := descriptions[proxyType]
		if description == "" {
			description = fmt.Sprintf("%s 프록시", strings.ToUpper(proxyType))
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\n", proxyType, description)
	}

	_ = w.Flush()
	return nil
}

// getStatusIcon 상태에 따른 아이콘 반환
func getStatusIcon(status string) string {
	switch status {
	case "passed":
		return "✅"
	case "failed":
		return "❌"
	case "skipped":
		return "⏭️"
	default:
		return "❓"
	}
}
