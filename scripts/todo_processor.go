// Package main provides utility scripts for development
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// TodoItem represents a todo item
type TodoItem struct {
	File     string `json:"file"`
	Line     int    `json:"line"`
	Priority string `json:"priority"`
	Content  string `json:"content"`
	Type     string `json:"type"` // TODO or FIXME
}

// TodoProcessor processes todo items in source code
type TodoProcessor struct {
	items []TodoItem
}

// ProcessDirectory processes all Go files in a directory for todo items
func (p *TodoProcessor) ProcessDirectory(dir string) error {
	return filepath.Walk(dir, func(path string, _ os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Go 파일만 처리
		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		// vendor, .git 디렉토리 제외
		if strings.Contains(path, "vendor/") || strings.Contains(path, ".git/") {
			return nil
		}

		return p.processFile(path)
	})
}

func (p *TodoProcessor) processFile(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()

	scanner := bufio.NewScanner(file)
	lineNum := 0

	// TODO/FIXME 패턴 매칭 (대소문자 구분 없음, 한국어 포함)
	todoRegex := regexp.MustCompile(`(?i)(todo|fixme)[:：]?\s*(.+)`)

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		if matches := todoRegex.FindStringSubmatch(line); matches != nil {
			item := TodoItem{
				File:     filename,
				Line:     lineNum,
				Type:     strings.ToUpper(matches[1]),
				Content:  strings.TrimSpace(matches[2]),
				Priority: p.determinePriority(matches[2]),
			}

			p.items = append(p.items, item)
		}
	}

	return scanner.Err()
}

func (p *TodoProcessor) determinePriority(content string) string {
	contentLower := strings.ToLower(content)

	// Critical 키워드
	criticalKeywords := []string{"security", "critical", "vulnerability", "safety", "보안", "취약점"}
	for _, keyword := range criticalKeywords {
		if strings.Contains(contentLower, keyword) {
			return "critical"
		}
	}

	// High 키워드
	highKeywords := []string{"implement", "missing", "broken", "필수", "구현", "미구현"}
	for _, keyword := range highKeywords {
		if strings.Contains(contentLower, keyword) {
			return "high"
		}
	}

	// Medium 키워드
	mediumKeywords := []string{"improve", "optimize", "enhance", "better", "개선", "최적화", "향상"}
	for _, keyword := range mediumKeywords {
		if strings.Contains(contentLower, keyword) {
			return "medium"
		}
	}

	// Low 키워드
	lowKeywords := []string{"cleanup", "refactor", "style", "comment", "정리", "리팩토링", "주석"}
	for _, keyword := range lowKeywords {
		if strings.Contains(contentLower, keyword) {
			return "low"
		}
	}

	return "medium" // 기본값
}

// GenerateReport generates a text report of todo items
func (p *TodoProcessor) GenerateReport() {
	priorities := map[string]int{
		"critical": 0,
		"high":     0,
		"medium":   0,
		"low":      0,
	}

	fileDistribution := make(map[string]int)

	for _, item := range p.items {
		priorities[item.Priority]++
		fileDistribution[item.File]++
	}

	fmt.Println("=== TODO/FIXME 우선순위별 분포 ===")
	caser := cases.Title(language.English)
	for priority, count := range priorities {
		fmt.Printf("%s: %d개\n", caser.String(priority), count)
	}

	fmt.Println("\n=== 파일별 분포 (상위 10개) ===")
	// 파일별 분포를 정렬하여 출력 (간단한 구현)
	for file, count := range fileDistribution {
		if count > 1 {
			fmt.Printf("%s: %d개\n", file, count)
		}
	}

	fmt.Printf("\n총 %d개의 TODO/FIXME 항목이 발견되었습니다.\n", len(p.items))
}

// GenerateMarkdownReport generates a markdown report of todo items
func (p *TodoProcessor) GenerateMarkdownReport() {
	fmt.Print("# TODO/FIXME 분석 보고서\n\n")

	// 우선순위별 그룹화
	priorityGroups := make(map[string][]TodoItem)
	for _, item := range p.items {
		priorityGroups[item.Priority] = append(priorityGroups[item.Priority], item)
	}

	// 우선순위 순서 정의
	priorities := []string{"critical", "high", "medium", "low"}

	for _, priority := range priorities {
		items := priorityGroups[priority]
		if len(items) == 0 {
			continue
		}

		caser := cases.Title(language.English)
		fmt.Printf("## %s Priority (%d개)\n\n", caser.String(priority), len(items))

		for _, item := range items {
			fmt.Printf("### %s:%d\n", item.File, item.Line)
			fmt.Printf("**Type**: %s  \n", item.Type)
			fmt.Printf("**Content**: %s\n\n", item.Content)
		}
	}
}

// ExportJSON exports todo items to a JSON file
func (p *TodoProcessor) ExportJSON() error {
	data, err := json.MarshalIndent(p.items, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile("todo_items.json", data, 0644)
}

func main() {
	processor := &TodoProcessor{}

	// 현재 디렉토리에서 TODO 항목 수집
	if err := processor.ProcessDirectory("."); err != nil {
		fmt.Printf("Error processing directory: %v\n", err)
		os.Exit(1)
	}

	// 명령행 인수에 따라 출력 형식 결정
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "-format=json":
			if err := processor.ExportJSON(); err != nil {
				fmt.Printf("Error exporting JSON: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("TODO items exported to todo_items.json")
		case "-format=markdown":
			processor.GenerateMarkdownReport()
		default:
			processor.GenerateReport()
		}
	} else {
		processor.GenerateReport()
	}
}
