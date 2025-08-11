package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"proxynd/cmd/format-multi/formatters"
)

var (
	// 색상 코드
	colorCyan   = "\033[36m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorRed    = "\033[31m"
	colorReset  = "\033[0m"
)

// 지원하는 파일 확장자와 포맷터 매핑
var formattersMap = map[string]formatters.Formatter{
	".go":   &formatters.GoFormatter{},
	".py":   &formatters.PythonFormatter{},
	".js":   &formatters.JavaScriptFormatter{},
	".jsx":  &formatters.JavaScriptFormatter{},
	".ts":   &formatters.JavaScriptFormatter{},
	".tsx":  &formatters.JavaScriptFormatter{},
	".kt":   &formatters.KotlinFormatter{},
	".kts":  &formatters.KotlinFormatter{},
	".sh":   &formatters.ShellFormatter{},
	".bash": &formatters.ShellFormatter{},
	".yml":  &formatters.YAMLFormatter{},
	".yaml": &formatters.YAMLFormatter{},
	".json": &formatters.JSONFormatter{},
	".md":   &formatters.MarkdownFormatter{},
	".html": &formatters.HTMLFormatter{},
	".htm":  &formatters.HTMLFormatter{},
	".css":  &formatters.CSSFormatter{},
	".scss": &formatters.CSSFormatter{},
	".sass": &formatters.CSSFormatter{},
}

func main() {
	var (
		configFile  = flag.String("config", ".formatrc.yaml", "Configuration file path")
		dryRun      = flag.Bool("dry-run", false, "Show what would be formatted without making changes")
		parallel    = flag.Int("parallel", runtime.NumCPU(), "Number of parallel workers")
		verbose     = flag.Bool("verbose", false, "Show verbose output")
		install     = flag.Bool("install", false, "Install all formatters")
		listFormats = flag.Bool("list", false, "List supported file formats")
	)
	flag.Parse()

	// 환경변수 CLAUDE_FILES 처리
	files := flag.Args()
	if len(files) == 0 {
		if claudeFiles := os.Getenv("CLAUDE_FILES"); claudeFiles != "" {
			files = strings.Fields(claudeFiles)
		}
	}

	// 명령 처리
	switch {
	case *install:
		installFormatters()
		return
	case *listFormats:
		listSupportedFormats()
		return
	case len(files) == 0:
		printUsage()
		os.Exit(1)
	}

	// 설정 로드
	config, err := loadConfig(*configFile)
	if err != nil && !os.IsNotExist(err) {
		fmt.Printf("%s❌ Error loading config: %v%s\n", colorRed, err, colorReset)
		os.Exit(1)
	}

	// 파일 포맷팅 실행
	if err := formatFiles(files, config, *dryRun, *parallel, *verbose); err != nil {
		fmt.Printf("%s❌ Error: %v%s\n", colorRed, err, colorReset)
		os.Exit(1)
	}

	fmt.Printf("%s🎉 All files processed!%s\n", colorGreen, colorReset)
}

func formatFiles(files []string, config *Config, dryRun bool, parallel int, verbose bool) error {
	// 작업 채널과 워커 그룹
	jobs := make(chan string, len(files))
	var wg sync.WaitGroup

	// 결과 수집
	results := make(chan FormatResult, len(files))
	var resultWg sync.WaitGroup
	resultWg.Add(1)

	// 결과 출력 고루틴
	go func() {
		defer resultWg.Done()
		successCount := 0
		skipCount := 0
		errorCount := 0

		for result := range results {
			if result.Error != nil {
				errorCount++
				fmt.Printf("%s❌ Error formatting %s: %v%s\n", colorRed, result.File, result.Error, colorReset)
			} else if result.Skipped {
				skipCount++
				if verbose {
					fmt.Printf("%s⚠️  Skipped %s: %s%s\n", colorYellow, result.File, result.Message, colorReset)
				}
			} else {
				successCount++
				fmt.Printf("%s✅ Formatted %s%s\n", colorGreen, result.File, colorReset)
			}
		}

		fmt.Printf("\n%sSummary: %d formatted, %d skipped, %d errors%s\n",
			colorCyan, successCount, skipCount, errorCount, colorReset)
	}()

	// 워커 시작
	for i := 0; i < parallel; i++ {
		wg.Add(1)
		go formatWorker(jobs, results, config, dryRun, verbose, &wg)
	}

	// 작업 추가
	for _, file := range files {
		jobs <- file
	}
	close(jobs)

	// 모든 워커 대기
	wg.Wait()
	close(results)
	resultWg.Wait()

	return nil
}

func formatWorker(
	jobs <-chan string,
	results chan<- FormatResult,
	config *Config,
	dryRun, verbose bool,
	wg *sync.WaitGroup,
) {
	defer wg.Done()

	for file := range jobs {
		result := FormatResult{File: file}

		// 파일 존재 확인
		if _, err := os.Stat(file); os.IsNotExist(err) {
			result.Error = fmt.Errorf("file does not exist")
			results <- result
			continue
		}

		// 확장자로 포맷터 찾기
		ext := strings.ToLower(filepath.Ext(file))
		formatter, ok := formattersMap[ext]
		if !ok {
			result.Skipped = true
			result.Message = "unsupported file type"
			results <- result
			continue
		}

		// 설정에서 언어가 활성화되어 있는지 확인
		langConfig := getLanguageConfig(config, formatter.Language())
		if langConfig != nil && !langConfig.Enabled {
			result.Skipped = true
			result.Message = fmt.Sprintf("%s formatting disabled", formatter.Language())
			results <- result
			continue
		}

		// 포맷터 사용 가능 확인
		if !formatter.IsAvailable() {
			result.Skipped = true
			result.Message = fmt.Sprintf("%s formatter not installed", formatter.Name())
			results <- result
			continue
		}

		// 포맷팅 실행
		if verbose {
			fmt.Printf("%s📝 Formatting %s with %s...%s\n", colorCyan, file, formatter.Name(), colorReset)
		}

		if dryRun {
			fmt.Printf("%s[DRY RUN] Would format %s%s\n", colorYellow, file, colorReset)
		} else {
			if err := formatter.Format(file, langConfig); err != nil {
				result.Error = err
			}
		}

		results <- result
	}
}

func printUsage() {
	fmt.Printf("%sUsage:%s\n", colorCyan, colorReset)
	fmt.Println("  format-multi [options] file1 file2 ...")
	fmt.Println("  CLAUDE_FILES='file1 file2' format-multi [options]")
	fmt.Println()
	fmt.Printf("%sOptions:%s\n", colorCyan, colorReset)
	fmt.Println("  -config string    Configuration file path (default \".formatrc.yaml\")")
	fmt.Println("  -dry-run         Show what would be formatted without making changes")
	fmt.Println("  -parallel int    Number of parallel workers (default: CPU count)")
	fmt.Println("  -verbose         Show verbose output")
	fmt.Println("  -install         Install all formatters")
	fmt.Println("  -list            List supported file formats")
}

func listSupportedFormats() {
	fmt.Printf("%sSupported file formats:%s\n\n", colorCyan, colorReset)

	// 언어별로 그룹화
	langMap := make(map[string][]string)
	for ext, formatter := range formattersMap {
		lang := formatter.Language()
		langMap[lang] = append(langMap[lang], ext)
	}

	for lang, exts := range langMap {
		fmt.Printf("%s%s:%s %s\n", colorGreen, lang, colorReset, strings.Join(exts, ", "))
	}
}

func installFormatters() {
	fmt.Printf("%sInstalling all formatters...%s\n\n", colorCyan, colorReset)

	installed := make(map[string]bool)
	for _, formatter := range formattersMap {
		name := formatter.Name()
		if installed[name] {
			continue
		}
		installed[name] = true

		fmt.Printf("Installing %s...\n", name)
		if err := formatter.Install(); err != nil {
			fmt.Printf("%s❌ Failed to install %s: %v%s\n", colorRed, name, err, colorReset)
		} else {
			fmt.Printf("%s✅ %s installed%s\n", colorGreen, name, colorReset)
		}
	}
}

// FormatResult 는 포맷팅 결과
type FormatResult struct {
	File    string
	Error   error
	Skipped bool
	Message string
}
