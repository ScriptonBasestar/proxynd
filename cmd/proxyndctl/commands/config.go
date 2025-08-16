package commands

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
)

// ConfigValidationResponse API 응답 구조체
type ConfigValidationResponse struct {
	Valid    bool                `json:"valid"`
	Errors   []ValidationError   `json:"errors,omitempty"`
	Warnings []ValidationWarning `json:"warnings,omitempty"`
	Summary  ValidationSummary   `json:"summary"`
}

// ValidationError 검증 오류
type ValidationError struct {
	File    string `json:"file"`
	Field   string `json:"field,omitempty"`
	Message string `json:"message"`
	Line    int    `json:"line,omitempty"`
}

// ValidationWarning 검증 경고
type ValidationWarning struct {
	File    string `json:"file"`
	Field   string `json:"field,omitempty"`
	Message string `json:"message"`
	Line    int    `json:"line,omitempty"`
}

// ValidationSummary 검증 요약
type ValidationSummary struct {
	TotalFiles     int               `json:"total_files"`
	ValidFiles     int               `json:"valid_files"`
	ErrorCount     int               `json:"error_count"`
	WarningCount   int               `json:"warning_count"`
	ConfigSources  []ConfigSource    `json:"config_sources"`
	ProxyTypes     []ProxyTypeStatus `json:"proxy_types"`
	LastValidated  time.Time         `json:"last_validated"`
	ValidationTime time.Duration     `json:"validation_time"`
}

// ConfigSource 설정 소스
type ConfigSource struct {
	File        string    `json:"file"`
	Path        string    `json:"path"`
	Size        int64     `json:"size"`
	Modified    time.Time `json:"modified"`
	Exists      bool      `json:"exists"`
	Readable    bool      `json:"readable"`
	Valid       bool      `json:"valid"`
	Description string    `json:"description"`
}

// ProxyTypeStatus 프록시 타입 상태
type ProxyTypeStatus struct {
	Type        string `json:"type"`
	Enabled     bool   `json:"enabled"`
	ConfigFile  string `json:"config_file"`
	ProxyCount  int    `json:"proxy_count"`
	HasUpstream bool   `json:"has_upstream"`
	Status      string `json:"status"` // ok, warning, error
}

// ConfigShowResponse 설정 표시 응답 구조체
type ConfigShowResponse struct {
	Global      interface{}            `json:"global"`
	ProxyTypes  map[string]interface{} `json:"proxy_types"`
	Sources     []ConfigSource         `json:"sources"`
	Environment map[string]string      `json:"environment"`
	Summary     ConfigShowSummary      `json:"summary"`
}

// ConfigShowSummary 설정 표시 요약
type ConfigShowSummary struct {
	TotalProxyTypes   int       `json:"total_proxy_types"`
	EnabledProxyTypes int       `json:"enabled_proxy_types"`
	TotalProxies      int       `json:"total_proxies"`
	ConfigDirectory   string    `json:"config_directory"`
	StorageDirectory  string    `json:"storage_directory"`
	LastModified      time.Time `json:"last_modified"`
}

// NewConfigCmd 설정 관리 명령어 그룹 생성
func NewConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "설정 관리 명령어",
		Long:  "ProxyND 서버의 설정을 관리합니다. 설정 검증 및 조회 기능을 제공합니다.",
	}

	// 서브커맨드 추가
	cmd.AddCommand(newConfigValidateCmd())
	cmd.AddCommand(newConfigShowCmd())

	return cmd
}

// newConfigValidateCmd 설정 검증 명령어
func newConfigValidateCmd() *cobra.Command {
	var (
		detailed bool
		checkNet bool
	)

	cmd := &cobra.Command{
		Use:   "validate",
		Short: "설정 파일 문법 검증",
		Long: `설정 파일의 문법을 검증합니다.
필수 항목 존재 확인, 네트워크 연결성 테스트를 포함합니다.`,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runConfigValidate(detailed, checkNet)
		},
	}

	cmd.Flags().BoolVarP(&detailed, "detailed", "d", false, "상세 검증 정보 표시")
	cmd.Flags().BoolVarP(&checkNet, "check-network", "n", false, "네트워크 연결성 테스트 포함")

	return cmd
}

// newConfigShowCmd 설정 표시 명령어
func newConfigShowCmd() *cobra.Command {
	var (
		proxyType string
		sources   bool
		env       bool
	)

	cmd := &cobra.Command{
		Use:   "show",
		Short: "현재 설정 표시",
		Long: `현재 설정을 표시합니다.
민감한 정보는 마스킹되며, 설정 소스 파일 경로를 표시합니다.`,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runConfigShow(proxyType, sources, env)
		},
	}

	cmd.Flags().StringVarP(&proxyType, "type", "t", "", "특정 프록시 타입만 표시 (apt, npm, maven 등)")
	cmd.Flags().BoolVarP(&sources, "sources", "s", false, "설정 소스 파일 정보 표시")
	cmd.Flags().BoolVarP(&env, "env", "e", false, "환경 변수 정보 표시")

	return cmd
}

// runConfigValidate 설정 검증 실행
func runConfigValidate(detailed, _ bool) error {
	serverURL := getServerURL()
	apiURL := fmt.Sprintf("%s/api/config/validate", serverURL)

	// API 호출
	resp, err := http.Get(apiURL)
	if err != nil {
		return fmt.Errorf(errServerConnection, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf(errAPIRequestStatus, resp.StatusCode)
	}

	// 응답 파싱
	var result ConfigValidationResponse
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
		return outputConfigValidationTable(result, detailed)
	}
}

// runConfigShow 설정 표시 실행
func runConfigShow(proxyType string, showSources, showEnv bool) error {
	serverURL := getServerURL()
	apiURL := fmt.Sprintf("%s/api/config/show", serverURL)

	// API 호출
	resp, err := http.Get(apiURL)
	if err != nil {
		return fmt.Errorf(errServerConnection, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf(errAPIRequestStatus, resp.StatusCode)
	}

	// 응답 파싱
	var result ConfigShowResponse
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
		return outputConfigShowTable(result, proxyType, showSources, showEnv)
	}
}

// outputConfigValidationTable 설정 검증 결과를 테이블 형태로 출력
func outputConfigValidationTable(result ConfigValidationResponse, detailed bool) error {
	// 전체 검증 상태
	status := "✅ 통과"
	if !result.Valid {
		status = "❌ 실패"
	} else if len(result.Warnings) > 0 {
		status = "⚠️ 경고"
	}

	fmt.Printf("🔍 설정 검증 결과: %s\n\n", status)

	// 요약 정보
	summary := result.Summary
	fmt.Printf("📊 검증 요약:\n")
	fmt.Printf("  전체 파일: %d개\n", summary.TotalFiles)
	fmt.Printf("  유효 파일: %d개\n", summary.ValidFiles)
	fmt.Printf("  오류: %d개\n", summary.ErrorCount)
	fmt.Printf("  경고: %d개\n", summary.WarningCount)
	fmt.Printf("  검증 시간: %v\n\n", summary.ValidationTime)

	// 오류 출력
	if len(result.Errors) > 0 {
		fmt.Println("❌ 오류:")
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		_, _ = fmt.Fprintln(w, "파일\t필드\t메시지")
		_, _ = fmt.Fprintln(w, "----\t----\t------")

		for _, err := range result.Errors {
			field := err.Field
			if field == "" {
				field = "-"
			}
			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", err.File, field, err.Message)
		}
		_ = w.Flush()
		fmt.Println()
	}

	// 경고 출력
	if len(result.Warnings) > 0 {
		fmt.Println("⚠️ 경고:")
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		_, _ = fmt.Fprintln(w, "파일\t필드\t메시지")
		_, _ = fmt.Fprintln(w, "----\t----\t------")

		for _, warn := range result.Warnings {
			field := warn.Field
			if field == "" {
				field = "-"
			}
			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", warn.File, field, warn.Message)
		}
		_ = w.Flush()
		fmt.Println()
	}

	// 프록시 타입 상태
	if len(summary.ProxyTypes) > 0 {
		fmt.Println("📦 프록시 타입 상태:")
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		_, _ = fmt.Fprintln(w, "타입\t상태\t설정파일\t프록시수\t상태")
		_, _ = fmt.Fprintln(w, "----\t----\t--------\t-------\t----")

		for _, proxy := range summary.ProxyTypes {
			enabled := statusInactive
			if proxy.Enabled {
				enabled = statusActive
			}

			statusIcon := iconSuccess
			switch proxy.Status {
			case "warning":
				statusIcon = iconWarning
			case "error":
				statusIcon = iconError
			}

			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\n",
				proxy.Type,
				enabled,
				proxy.ConfigFile,
				proxy.ProxyCount,
				statusIcon,
			)
		}
		_ = w.Flush()
		fmt.Println()
	}

	// 상세 정보 출력
	if detailed && len(summary.ConfigSources) > 0 {
		fmt.Println("📄 설정 파일 상세:")
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		_, _ = fmt.Fprintln(w, "파일\t크기\t수정일시\t존재\t읽기\t유효")
		_, _ = fmt.Fprintln(w, "----\t----\t--------\t----\t----\t----")

		for _, source := range summary.ConfigSources {
			exists := boolIcon(source.Exists)
			readable := boolIcon(source.Readable)
			valid := boolIcon(source.Valid)

			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
				source.File,
				formatBytes(source.Size),
				source.Modified.Format("01-02 15:04"),
				exists,
				readable,
				valid,
			)
		}
		_ = w.Flush()
		fmt.Println()
	}

	return nil
}

// outputConfigShowTable 설정 표시 결과를 테이블 형태로 출력
func outputConfigShowTable(result ConfigShowResponse, filterType string, showSources, showEnv bool) error {
	fmt.Println("⚙️ ProxyND 설정 정보")
	fmt.Println()

	// 요약 정보
	summary := result.Summary
	fmt.Printf("📊 요약:\n")
	fmt.Printf("  전체 프록시 타입: %d개\n", summary.TotalProxyTypes)
	fmt.Printf("  활성 프록시 타입: %d개\n", summary.EnabledProxyTypes)
	fmt.Printf("  총 프록시 수: %d개\n", summary.TotalProxies)
	fmt.Printf("  설정 디렉토리: %s\n", summary.ConfigDirectory)
	fmt.Printf("  저장소 디렉토리: %s\n", summary.StorageDirectory)
	if !summary.LastModified.IsZero() {
		fmt.Printf("  마지막 수정: %s\n", summary.LastModified.Format("2006-01-02 15:04:05"))
	}
	fmt.Println()

	// 글로벌 설정 (간단히 표시)
	if result.Global != nil {
		fmt.Println("🌐 글로벌 설정:")
		if globalBytes, err := json.MarshalIndent(result.Global, "  ", "  "); err == nil {
			fmt.Printf("  %s\n", string(globalBytes))
		}
		fmt.Println()
	}

	// 프록시 타입별 설정
	if len(result.ProxyTypes) > 0 {
		fmt.Println("📦 프록시 설정:")

		for proxyType, config := range result.ProxyTypes {
			// 필터링
			if filterType != "" && proxyType != filterType {
				continue
			}

			fmt.Printf("  %s:\n", strings.ToUpper(proxyType))
			if configBytes, err := json.MarshalIndent(config, "    ", "  "); err == nil {
				fmt.Printf("    %s\n", string(configBytes))
			}
		}
		fmt.Println()
	}

	// 설정 소스 파일 정보
	if showSources && len(result.Sources) > 0 {
		fmt.Println("📄 설정 소스 파일:")
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		_, _ = fmt.Fprintln(w, "파일\t경로\t크기\t수정일시\t상태")
		_, _ = fmt.Fprintln(w, "----\t----\t----\t--------\t----")

		for _, source := range result.Sources {
			status := iconError
			if source.Valid {
				status = iconSuccess
			} else if source.Readable {
				status = iconWarning
			}

			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
				source.File,
				truncateString(source.Path, 40),
				formatBytes(source.Size),
				source.Modified.Format("01-02 15:04"),
				status,
			)
		}
		_ = w.Flush()
		fmt.Println()
	}

	// 환경 변수 정보
	if showEnv && len(result.Environment) > 0 {
		fmt.Println("🌍 환경 변수:")
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		_, _ = fmt.Fprintln(w, "변수\t값\t상태")
		_, _ = fmt.Fprintln(w, "----\t---\t----")

		for key, value := range result.Environment {
			status := iconSuccess
			displayValue := value
			if value == "" {
				status = iconWarning
				displayValue = "(미설정)"
			} else if strings.Contains(strings.ToLower(key), "secret") || strings.Contains(strings.ToLower(key), "password") {
				displayValue = "***"
			}

			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", key, displayValue, status)
		}
		_ = w.Flush()
		fmt.Println()
	}

	return nil
}

// boolIcon 불린 값을 아이콘으로 변환
func boolIcon(value bool) string {
	if value {
		return iconSuccess
	}
	return iconError
}
