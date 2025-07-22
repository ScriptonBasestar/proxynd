// Package commands provides CLI command implementations for proxyndctl
package commands

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
)

// 출력 형식 상수
const (
	outputFormatJSON  = "json"
	outputFormatYAML  = "yaml"
	outputFormatTable = "table"
)

// CacheListResponse API 응답 구조체
type CacheListResponse struct {
	Items []CacheItem `json:"items"`
	Total int         `json:"total"`
}

// CacheItem 캐시 항목 정보
type CacheItem struct {
	Key         string    `json:"key"`
	ProxyType   string    `json:"proxy_type"`
	Path        string    `json:"path"`
	Size        int64     `json:"size"`
	CreatedAt   time.Time `json:"created_at"`
	AccessedAt  time.Time `json:"accessed_at"`
	TTL         string    `json:"ttl"`
	ContentType string    `json:"content_type"`
}

// CacheSizeResponse 캐시 크기 응답 구조체
type CacheSizeResponse struct {
	TotalSize   int64                    `json:"total_size"`
	TotalItems  int64                    `json:"total_items"`
	SizeByType  map[string]int64         `json:"size_by_type"`
	ItemsByType map[string]int64         `json:"items_by_type"`
	DiskUsage   DiskUsageInfo            `json:"disk_usage"`
	LastClear   *time.Time               `json:"last_clear,omitempty"`
	TypeDetails map[string]ProxyTypeInfo `json:"type_details"`
}

// DiskUsageInfo 디스크 사용량 정보
type DiskUsageInfo struct {
	Used      int64   `json:"used"`
	Available int64   `json:"available"`
	Total     int64   `json:"total"`
	UsedPct   float64 `json:"used_percent"`
}

// ProxyTypeInfo 프록시 타입별 상세 정보
type ProxyTypeInfo struct {
	Enabled    bool   `json:"enabled"`
	ConfigPath string `json:"config_path"`
	CachePath  string `json:"cache_path"`
	ProxyCount int    `json:"proxy_count"`
}

// NewCacheCmd 캐시 관리 명령어 그룹 생성
func NewCacheCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cache",
		Short: "캐시 관리 명령어",
		Long:  "ProxyND 서버의 캐시를 관리합니다. 캐시 조회, 정리, 통계를 제공합니다.",
	}

	// 서브커맨드 추가
	cmd.AddCommand(newCacheListCmd())
	cmd.AddCommand(newCacheClearCmd())
	cmd.AddCommand(newCacheSizeCmd())

	return cmd
}

// newCacheListCmd 캐시 목록 조회 명령어
func newCacheListCmd() *cobra.Command {
	var (
		proxyType string
		limit     int
		offset    int
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "캐시된 패키지 목록 조회",
		Long: `캐시된 패키지 목록을 조회합니다.
패키지 타입별 필터링을 지원하며, 크기, 생성 시간, 마지막 접근 시간을 표시합니다.`,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runCacheList(proxyType, limit, offset)
		},
	}

	cmd.Flags().StringVarP(&proxyType, "type", "t", "", "프록시 타입 필터 (apt, npm, maven, pip, docker, yum, gem, apk)")
	cmd.Flags().IntVarP(&limit, "limit", "l", 50, "조회할 항목 수")
	cmd.Flags().IntVarP(&offset, "offset", "o", 0, "시작 오프셋")

	return cmd
}

// newCacheClearCmd 캐시 정리 명령어
func newCacheClearCmd() *cobra.Command {
	var (
		proxyType string
		force     bool
		confirm   bool
	)

	cmd := &cobra.Command{
		Use:   "clear",
		Short: "캐시 정리",
		Long: `캐시를 정리합니다.
전체 또는 특정 패키지 타입별 정리가 가능하며, 확인 프롬프트를 제공합니다.`,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runCacheClear(proxyType, force, confirm)
		},
	}

	cmd.Flags().StringVarP(&proxyType, "type", "t", "", "정리할 프록시 타입 (지정하지 않으면 전체)")
	cmd.Flags().BoolVarP(&force, "force", "f", false, "확인 없이 강제 실행")
	cmd.Flags().BoolVarP(&confirm, "yes", "y", false, "확인 프롬프트 건너뛰기")

	return cmd
}

// newCacheSizeCmd 캐시 크기 조회 명령어
func newCacheSizeCmd() *cobra.Command {
	var detailed bool

	cmd := &cobra.Command{
		Use:   "size",
		Short: "캐시 사용량 통계",
		Long: `캐시 사용량 통계를 조회합니다.
패키지 타입별 사용량, 디스크 사용률 및 여유 공간을 표시합니다.`,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runCacheSize(detailed)
		},
	}

	cmd.Flags().BoolVarP(&detailed, "detailed", "d", false, "상세 정보 표시")

	return cmd
}

// runCacheList 캐시 목록 조회 실행
func runCacheList(proxyType string, limit, offset int) error {
	serverURL := getServerURL()

	// API 요청 URL 구성
	apiURL := fmt.Sprintf("%s/api/cache/list", serverURL)
	params := url.Values{}
	if proxyType != "" {
		params.Set("type", proxyType)
	}
	params.Set("limit", strconv.Itoa(limit))
	params.Set("offset", strconv.Itoa(offset))

	if len(params) > 0 {
		apiURL += "?" + params.Encode()
	}

	// API 호출
	resp, err := http.Get(apiURL)
	if err != nil {
		return fmt.Errorf(errServerConnection, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API 요청 실패: HTTP %d", resp.StatusCode)
	}

	// 응답 파싱
	var result CacheListResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("응답 파싱 실패: %v", err)
	}

	// 결과 출력
	outputFormat := getOutputFormat()
	switch outputFormat {
	case "json":
		return outputJSON(result)
	case "yaml":
		return outputYAML(result)
	default:
		return outputCacheListTable(result)
	}
}

// runCacheClear 캐시 정리 실행
func runCacheClear(proxyType string, force, confirm bool) error {
	// 확인 프롬프트
	if !force && !confirm {
		message := "전체 캐시를 정리하시겠습니까?"
		if proxyType != "" {
			message = fmt.Sprintf("%s 캐시를 정리하시겠습니까?", proxyType)
		}

		fmt.Printf("%s (y/N): ", message)
		var response string
		if _, err := fmt.Scanln(&response); err != nil {
			fmt.Printf("입력을 읽는 중 오류 발생: %v\n", err)
			return err
		}
		if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
			fmt.Println("캐시 정리가 취소되었습니다.")
			return nil
		}
	}

	serverURL := getServerURL()
	var apiURL string

	if proxyType != "" {
		apiURL = fmt.Sprintf("%s/api/cache/clear/%s?confirm=true", serverURL, proxyType)
	} else {
		apiURL = fmt.Sprintf("%s/api/cache/clear?confirm=true", serverURL)
	}

	// DELETE 요청 생성
	req, err := http.NewRequest("DELETE", apiURL, nil)
	if err != nil {
		return fmt.Errorf("요청 생성 실패: %v", err)
	}

	// API 호출
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf(errServerConnection, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("캐시 정리 실패: HTTP %d", resp.StatusCode)
	}

	// 성공 메시지 출력
	if proxyType != "" {
		fmt.Printf("✅ %s 캐시가 성공적으로 정리되었습니다.\n", proxyType)
	} else {
		fmt.Println("✅ 전체 캐시가 성공적으로 정리되었습니다.")
	}

	return nil
}

// runCacheSize 캐시 크기 조회 실행
func runCacheSize(detailed bool) error {
	serverURL := getServerURL()
	apiURL := fmt.Sprintf("%s/api/cache/size", serverURL)

	// API 호출
	resp, err := http.Get(apiURL)
	if err != nil {
		return fmt.Errorf(errServerConnection, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API 요청 실패: HTTP %d", resp.StatusCode)
	}

	// 응답 파싱
	var result CacheSizeResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("응답 파싱 실패: %v", err)
	}

	// 결과 출력
	outputFormat := getOutputFormat()
	switch outputFormat {
	case "json":
		return outputJSON(result)
	case "yaml":
		return outputYAML(result)
	default:
		return outputCacheSizeTable(result, detailed)
	}
}

// outputCacheListTable 캐시 목록을 테이블 형태로 출력
func outputCacheListTable(result CacheListResponse) error {
	if len(result.Items) == 0 {
		fmt.Println("캐시된 항목이 없습니다.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "타입\t경로\t크기\t생성일시\t접근일시\tTTL")
	_, _ = fmt.Fprintln(w, "----\t----\t----\t-------\t-------\t---")

	for _, item := range result.Items {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			item.ProxyType,
			truncateString(item.Path, 50),
			formatBytes(item.Size),
			item.CreatedAt.Format("01-02 15:04"),
			item.AccessedAt.Format("01-02 15:04"),
			item.TTL,
		)
	}

	_, _ = fmt.Fprintf(w, "\n총 %d개 항목\n", result.Total)
	return w.Flush()
}

// outputCacheSizeTable 캐시 크기를 테이블 형태로 출력
func outputCacheSizeTable(result CacheSizeResponse, detailed bool) error {
	fmt.Printf("📊 캐시 사용량 통계\n\n")

	// 전체 통계
	fmt.Printf("전체 크기: %s (%d 바이트)\n", formatBytes(result.TotalSize), result.TotalSize)
	fmt.Printf("전체 항목: %d개\n\n", result.TotalItems)

	// 프록시 타입별 크기
	if len(result.SizeByType) > 0 {
		fmt.Println("📦 프록시 타입별 사용량:")
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		_, _ = fmt.Fprintln(w, "타입\t크기\t항목수\t상태\t설정파일")
		_, _ = fmt.Fprintln(w, "----\t----\t-----\t----\t--------")

		for proxyType, size := range result.SizeByType {
			items := result.ItemsByType[proxyType]
			detail := result.TypeDetails[proxyType]
			status := "❌ 비활성"
			if detail.Enabled {
				status = "✅ 활성"
			}

			_, _ = fmt.Fprintf(w, "%s\t%s\t%d\t%s\t%s\n",
				proxyType,
				formatBytes(size),
				items,
				status,
				detail.ConfigPath,
			)
		}
		_ = w.Flush()
		fmt.Println()
	}

	// 상세 정보
	if detailed {
		fmt.Println("🔍 상세 정보:")
		if result.LastClear != nil {
			fmt.Printf("마지막 정리: %s\n", result.LastClear.Format("2006-01-02 15:04:05"))
		}

		// 디스크 사용량 (사용 가능한 경우)
		if result.DiskUsage.Total > 0 {
			fmt.Printf("디스크 사용률: %.1f%% (%s/%s)\n",
				result.DiskUsage.UsedPct,
				formatBytes(result.DiskUsage.Used),
				formatBytes(result.DiskUsage.Total))
		}
	}

	return nil
}

// 헬퍼 함수들
func getServerURL() string {
	// 환경 변수 또는 기본값 사용 (나중에 글로벌 플래그와 연동)
	if url := os.Getenv("PROXYNDCTL_SERVER_URL"); url != "" {
		return url
	}
	return "http://localhost:8080"
}

func getOutputFormat() string {
	// 환경 변수 또는 기본값 사용 (나중에 글로벌 플래그와 연동)
	if format := os.Getenv("PROXYNDCTL_OUTPUT_FORMAT"); format != "" {
		return format
	}
	return outputFormatTable
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func truncateString(s string, length int) string {
	if len(s) <= length {
		return s
	}
	return s[:length-3] + "..."
}

func outputJSON(data interface{}) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

func outputYAML(data interface{}) error {
	// YAML 출력은 간단히 JSON으로 대체 (실제로는 yaml 라이브러리 사용)
	return outputJSON(data)
}
