package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"proxynd/cmd/proxyndctl/commands"
)

var (
	// 글로벌 설정 변수들
	serverURL string
	timeout   string
	format    string
	verbose   bool
)

// rootCmd는 proxyndctl의 기본 명령어
var rootCmd = &cobra.Command{
	Use:   "proxyndctl",
	Short: "ProxyND 서버 관리를 위한 CLI 도구",
	Long: `ProxyND는 패키지 매니저 프록시/미러 서버입니다.
이 CLI 도구를 사용하여 ProxyND 서버를 효율적으로 관리할 수 있습니다.

지원하는 기능:
- 캐시 관리 (조회, 정리, 통계)
- 설정 검증 및 조회
- 서버 상태 모니터링
- 사용자 관리
- 프록시 기능 테스트`,
	Version: "1.0.0",
}

// cacheCmd는 캐시 관리 명령어 그룹 (commands 패키지에서 가져옴)
var cacheCmd = commands.NewCacheCmd()

// configCmd는 설정 관리 명령어 그룹 (commands 패키지에서 가져옴)
var configCmd = commands.NewConfigCmd()

// userCmd는 사용자 관리 명령어 그룹
var userCmd = &cobra.Command{
	Use:   "user",
	Short: "사용자 관리 명령어",
	Long:  "ProxyND 서버의 사용자 계정을 관리합니다. 사용자 추가, 삭제, 조회 기능을 제공합니다.",
}

// testCmd는 프록시 테스트 명령어 그룹
var testCmd = &cobra.Command{
	Use:   "test",
	Short: "프록시 기능 테스트 명령어",
	Long:  "ProxyND 서버의 프록시 기능을 테스트합니다. 각 패키지 매니저별 연결성을 확인합니다.",
}

// statusCmd는 서버 상태 확인 명령어
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "서버 상태 확인",
	Long:  "ProxyND 서버의 전반적인 상태를 확인합니다.",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("서버 상태 확인 기능을 구현 예정입니다.")
		// TODO: 실제 상태 확인 로직 구현
	},
}

// healthCmd는 헬스체크 명령어
var healthCmd = &cobra.Command{
	Use:   "health",
	Short: "헬스체크 수행",
	Long:  "ProxyND 서버의 헬스체크를 수행하고 업스트림 연결 상태를 확인합니다.",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("헬스체크 기능을 구현 예정입니다.")
		// TODO: 실제 헬스체크 로직 구현
	},
}

// metricsCmd는 메트릭 조회 명령어
var metricsCmd = &cobra.Command{
	Use:   "metrics",
	Short: "메트릭 조회",
	Long:  "ProxyND 서버의 Prometheus 메트릭을 사용자 친화적 형태로 조회합니다.",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("메트릭 조회 기능을 구현 예정입니다.")
		// TODO: 실제 메트릭 조회 로직 구현
	},
}

func init() {
	// 글로벌 플래그 설정
	rootCmd.PersistentFlags().StringVar(&serverURL, "server", "http://localhost:8080", "ProxyND 서버 URL")
	rootCmd.PersistentFlags().StringVar(&timeout, "timeout", "30s", "요청 타임아웃")
	rootCmd.PersistentFlags().StringVar(&format, "format", "table", "출력 포맷 (table, json, yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "상세 출력")

	// 서브커맨드 추가
	rootCmd.AddCommand(cacheCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(userCmd)
	rootCmd.AddCommand(testCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(healthCmd)
	rootCmd.AddCommand(metricsCmd)

	// 캐시 관리 서브커맨드들 (향후 구현)
	// cacheCmd.AddCommand(cacheListCmd)
	// cacheCmd.AddCommand(cacheClearCmd)
	// cacheCmd.AddCommand(cacheSizeCmd)

	// 설정 관리 서브커맨드들 (향후 구현)
	// configCmd.AddCommand(configValidateCmd)
	// configCmd.AddCommand(configShowCmd)

	// 사용자 관리 서브커맨드들 (향후 구현)
	// userCmd.AddCommand(userAddCmd)
	// userCmd.AddCommand(userDeleteCmd)
	// userCmd.AddCommand(userListCmd)

	// 프록시 테스트 서브커맨드들 (향후 구현)
	// testCmd.AddCommand(testAptCmd)
	// testCmd.AddCommand(testNpmCmd)
	// testCmd.AddCommand(testMavenCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
