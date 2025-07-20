package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

// NewDocsCmd 문서 생성 명령어
func NewDocsCmd(rootCmd *cobra.Command) *cobra.Command {
	cmd := &cobra.Command{
		Use:    "docs",
		Short:  "문서 생성 명령어",
		Long:   "ProxyND CLI의 다양한 형식의 문서를 생성합니다 (man pages, markdown, rest, yaml).",
		Hidden: false, // 일반 사용자에게도 표시
	}

	// Man page 생성 서브커맨드
	manCmd := &cobra.Command{
		Use:   "man [경로]",
		Short: "Man page 생성",
		Long: `ProxyND CLI의 man page를 생성합니다.

생성된 man page는 시스템의 man 디렉토리에 설치할 수 있습니다.

예제:
  # 현재 디렉토리에 man page 생성
  $ proxyndctl docs man .

  # 특정 디렉토리에 생성
  $ proxyndctl docs man /tmp/man

  # 시스템에 설치 (관리자 권한 필요)
  $ proxyndctl docs man . && sudo cp ./proxyndctl.1 /usr/share/man/man1/
  $ sudo mandb  # man 데이터베이스 업데이트

  # 설치 확인
  $ man proxyndctl`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			dir := "."
			if len(args) > 0 {
				dir = args[0]
			}
			return generateManPages(rootCmd, dir)
		},
	}

	// Markdown 문서 생성 서브커맨드
	markdownCmd := &cobra.Command{
		Use:   "markdown [경로]",
		Short: "Markdown 문서 생성",
		Long: `ProxyND CLI의 Markdown 형식 문서를 생성합니다.

생성된 문서는 GitHub Wiki나 다른 문서 시스템에서 사용할 수 있습니다.

예제:
  # 현재 디렉토리에 markdown 생성
  $ proxyndctl docs markdown .

  # docs 디렉토리에 생성
  $ proxyndctl docs markdown ./docs/cli`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			dir := "."
			if len(args) > 0 {
				dir = args[0]
			}
			return generateMarkdownDocs(rootCmd, dir)
		},
	}

	// RestructuredText 문서 생성 서브커맨드
	restCmd := &cobra.Command{
		Use:   "rest [경로]",
		Short: "RestructuredText 문서 생성",
		Long: `ProxyND CLI의 RestructuredText 형식 문서를 생성합니다.

생성된 문서는 Sphinx나 Read the Docs에서 사용할 수 있습니다.

예제:
  # 현재 디렉토리에 rest 생성
  $ proxyndctl docs rest .

  # docs 디렉토리에 생성
  $ proxyndctl docs rest ./docs/source`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			dir := "."
			if len(args) > 0 {
				dir = args[0]
			}
			return generateRestDocs(rootCmd, dir)
		},
	}

	// YAML 문서 생성 서브커맨드
	yamlCmd := &cobra.Command{
		Use:   "yaml [경로]",
		Short: "YAML 문서 생성",
		Long: `ProxyND CLI의 구조를 YAML 형식으로 출력합니다.

이 문서는 자동화 도구나 문서 생성 시스템에서 사용할 수 있습니다.

예제:
  # 현재 디렉토리에 YAML 생성
  $ proxyndctl docs yaml .

  # 파일로 저장
  $ proxyndctl docs yaml . > proxyndctl-commands.yaml`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			dir := "."
			if len(args) > 0 {
				dir = args[0]
			}
			return generateYamlDocs(rootCmd, dir)
		},
	}

	cmd.AddCommand(manCmd)
	cmd.AddCommand(markdownCmd)
	cmd.AddCommand(restCmd)
	cmd.AddCommand(yamlCmd)

	return cmd
}

// generateManPages man page 생성
func generateManPages(rootCmd *cobra.Command, dir string) error {
	// 디렉토리 생성
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("디렉토리 생성 실패: %v", err)
	}

	// Man page 헤더 정보
	header := &doc.GenManHeader{
		Title:   "PROXYNDCTL",
		Section: "1",
		Manual:  "ProxyND CLI Manual",
		Source:  "ProxyND " + rootCmd.Version,
	}

	// Man page 생성
	if err := doc.GenManTree(rootCmd, header, dir); err != nil {
		return fmt.Errorf("man page 생성 실패: %v", err)
	}

	fmt.Printf("✅ Man page가 생성되었습니다: %s\n", dir)

	// 생성된 파일 목록 표시
	files, err := filepath.Glob(filepath.Join(dir, "*.1"))
	if err == nil && len(files) > 0 {
		fmt.Println("\n생성된 파일:")
		for _, file := range files {
			fmt.Printf("  - %s\n", filepath.Base(file))
		}

		fmt.Println("\n설치 방법:")
		fmt.Printf("  $ sudo cp %s/*.1 /usr/share/man/man1/\n", dir)
		fmt.Println("  $ sudo mandb")
		fmt.Println("\n확인:")
		fmt.Println("  $ man proxyndctl")
	}

	return nil
}

// generateMarkdownDocs Markdown 문서 생성
func generateMarkdownDocs(rootCmd *cobra.Command, dir string) error {
	// 디렉토리 생성
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("디렉토리 생성 실패: %v", err)
	}

	// Markdown 문서 생성
	if err := doc.GenMarkdownTree(rootCmd, dir); err != nil {
		return fmt.Errorf("markdown 문서 생성 실패: %v", err)
	}

	// 인덱스 파일 생성
	if err := generateMarkdownIndex(rootCmd, dir); err != nil {
		return fmt.Errorf("markdown 인덱스 생성 실패: %v", err)
	}

	fmt.Printf("✅ Markdown 문서가 생성되었습니다: %s\n", dir)

	return nil
}

// generateMarkdownIndex Markdown 인덱스 파일 생성
func generateMarkdownIndex(rootCmd *cobra.Command, dir string) error {
	indexPath := filepath.Join(dir, "README.md")

	content := fmt.Sprintf(`# ProxyND CLI 명령어 참조

이 문서는 ProxyND CLI (proxyndctl) 명령어의 전체 참조 문서입니다.

## 버전 정보

- 버전: %s
- 생성일: %s

## 명령어 목록

### 메인 명령어

- [proxyndctl](proxyndctl.md) - ProxyND 서버 관리를 위한 CLI 도구

### 서브커맨드

#### 캐시 관리
- [proxyndctl cache](proxyndctl_cache.md) - 캐시 관리 명령어
  - [proxyndctl cache list](proxyndctl_cache_list.md) - 캐시 목록 조회
  - [proxyndctl cache clear](proxyndctl_cache_clear.md) - 캐시 삭제
  - [proxyndctl cache size](proxyndctl_cache_size.md) - 캐시 크기 조회

#### 설정 관리
- [proxyndctl config](proxyndctl_config.md) - 설정 관리 명령어
  - [proxyndctl config validate](proxyndctl_config_validate.md) - 설정 검증
  - [proxyndctl config show](proxyndctl_config_show.md) - 설정 조회

#### 서버 상태
- [proxyndctl status](proxyndctl_status.md) - 서버 상태 확인
- [proxyndctl health](proxyndctl_health.md) - 헬스체크 수행
- [proxyndctl metrics](proxyndctl_metrics.md) - 메트릭 조회

#### 사용자 관리
- [proxyndctl user](proxyndctl_user.md) - 사용자 관리 명령어
  - [proxyndctl user list](proxyndctl_user_list.md) - 사용자 목록 조회
  - [proxyndctl user add](proxyndctl_user_add.md) - 사용자 추가
  - [proxyndctl user delete](proxyndctl_user_delete.md) - 사용자 삭제
  - [proxyndctl user info](proxyndctl_user_info.md) - 사용자 정보 조회

#### 프록시 테스트
- [proxyndctl test](proxyndctl_test.md) - 프록시 기능 테스트
  - [proxyndctl test all](proxyndctl_test_all.md) - 모든 프록시 테스트
  - [proxyndctl test types](proxyndctl_test_types.md) - 지원되는 프록시 타입 조회
  - 각 프록시별 테스트 명령어

#### 자동완성
- [proxyndctl completion](proxyndctl_completion.md) - 쉘 자동완성 스크립트 생성

#### 문서
- [proxyndctl docs](proxyndctl_docs.md) - 문서 생성 명령어

## 전역 플래그

모든 명령어에서 사용할 수 있는 전역 플래그:

- `+"`--server`"+` - ProxyND 서버 URL (기본값: http://localhost:8080)
- `+"`--timeout`"+` - 요청 타임아웃 (기본값: 30s)
- `+"`--format`"+` - 출력 포맷 (table, json, yaml) (기본값: table)
- `+"`-v, --verbose`"+` - 상세 출력

## 예제

### 기본 사용법

`+"```bash"+`
# 서버 상태 확인
proxyndctl status

# 캐시 목록 조회
proxyndctl cache list

# JSON 형식으로 출력
proxyndctl cache list --format json

# 다른 서버에 연결
proxyndctl --server http://proxy.example.com:8080 status
`+"```"+`

### 자동완성 설정

`+"```bash"+`
# Bash
source <(proxyndctl completion bash)

# Zsh
source <(proxyndctl completion zsh)

# Fish
proxyndctl completion fish > ~/.config/fish/completions/proxyndctl.fish
`+"```"+`

## 관련 문서

- [ProxyND 서버 문서](https://github.com/scriptonbasestar/proxynd)
- [CLI 자동완성 가이드](/docs/CLI_COMPLETION.md)
`, rootCmd.Version, time.Now().Format("2006-01-02"))

	return os.WriteFile(indexPath, []byte(content), 0644)
}

// generateRestDocs RestructuredText 문서 생성
func generateRestDocs(rootCmd *cobra.Command, dir string) error {
	// 디렉토리 생성
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("디렉토리 생성 실패: %v", err)
	}

	// RestructuredText 문서 생성
	if err := doc.GenReSTTree(rootCmd, dir); err != nil {
		return fmt.Errorf("rest 문서 생성 실패: %v", err)
	}

	fmt.Printf("✅ RestructuredText 문서가 생성되었습니다: %s\n", dir)

	return nil
}

// generateYamlDocs YAML 문서 생성
func generateYamlDocs(rootCmd *cobra.Command, dir string) error {
	// 디렉토리 생성
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("디렉토리 생성 실패: %v", err)
	}

	// YAML 문서 생성
	yamlPath := filepath.Join(dir, "proxyndctl.yaml")
	if err := doc.GenYamlTree(rootCmd, dir); err != nil {
		return fmt.Errorf("yaml 문서 생성 실패: %v", err)
	}

	fmt.Printf("✅ YAML 문서가 생성되었습니다: %s\n", yamlPath)

	return nil
}

// GenerateAllDocs 모든 형식의 문서를 한 번에 생성
func GenerateAllDocs(rootCmd *cobra.Command, baseDir string) error {
	// 각 형식별 디렉토리 생성 및 문서 생성
	formats := map[string]func(*cobra.Command, string) error{
		"man":      generateManPages,
		"markdown": generateMarkdownDocs,
		"rest":     generateRestDocs,
		"yaml":     generateYamlDocs,
	}

	for format, genFunc := range formats {
		dir := filepath.Join(baseDir, format)
		fmt.Printf("\n%s 문서 생성 중...\n", strings.ToUpper(format))

		if err := genFunc(rootCmd, dir); err != nil {
			return fmt.Errorf("%s 문서 생성 실패: %v", format, err)
		}
	}

	fmt.Printf("\n✨ 모든 문서가 생성되었습니다: %s\n", baseDir)
	return nil
}
