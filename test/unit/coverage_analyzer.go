package unit

import (
	"bufio"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// CoverageAnalyzer 코드 커버리지 분석기
type CoverageAnalyzer struct {
	ProjectRoot  string
	ExcludePaths []string
	FileSet      *token.FileSet
}

// PackageInfo 패키지 정보
type PackageInfo struct {
	Name          string
	Path          string
	Functions     []FunctionInfo
	TestFiles     []string
	CoverageScore float64
	TestCount     int
	FunctionCount int
}

// FunctionInfo 함수 정보
type FunctionInfo struct {
	Name       string
	File       string
	LineStart  int
	LineEnd    int
	IsExported bool
	HasTest    bool
	Complexity int
}

// CoverageReport 커버리지 보고서
type CoverageReport struct {
	TotalPackages   int
	TestedPackages  int
	TotalFunctions  int
	TestedFunctions int
	OverallCoverage float64
	PackageReports  []PackageInfo
	MissingTests    []MissingTestInfo
	Recommendations []string
}

// MissingTestInfo 누락된 테스트 정보
type MissingTestInfo struct {
	Package    string
	Function   string
	File       string
	Line       int
	Priority   string // "high", "medium", "low"
	Complexity int
}

// NewCoverageAnalyzer 새 커버리지 분석기 생성
func NewCoverageAnalyzer(projectRoot string) *CoverageAnalyzer {
	return &CoverageAnalyzer{
		ProjectRoot: projectRoot,
		ExcludePaths: []string{
			"vendor/",
			".git/",
			"test/",
			"test/",
			"_test.go",
			"mocks/",
			"cmd/",
		},
		FileSet: token.NewFileSet(),
	}
}

// AnalyzeCoverage 커버리지 분석 실행
func (ca *CoverageAnalyzer) AnalyzeCoverage() (*CoverageReport, error) {
	packages, err := ca.scanPackages()
	if err != nil {
		return nil, err
	}

	report := &CoverageReport{
		PackageReports: packages,
	}

	ca.calculateOverallCoverage(report)
	ca.generateMissingTests(report)
	ca.generateRecommendations(report)

	return report, nil
}

// scanPackages 패키지 스캔
func (ca *CoverageAnalyzer) scanPackages() ([]PackageInfo, error) {
	var packages []PackageInfo

	err := filepath.Walk(ca.ProjectRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			return nil
		}

		// 제외 경로 확인
		relPath, _ := filepath.Rel(ca.ProjectRoot, path)
		if ca.shouldExclude(relPath) {
			return filepath.SkipDir
		}

		// Go 파일이 있는지 확인
		files, err := os.ReadDir(path)
		if err != nil {
			return err
		}

		var goFiles []string
		var testFiles []string

		for _, file := range files {
			if strings.HasSuffix(file.Name(), ".go") {
				if strings.HasSuffix(file.Name(), "_test.go") {
					testFiles = append(testFiles, file.Name())
				} else {
					goFiles = append(goFiles, file.Name())
				}
			}
		}

		if len(goFiles) > 0 {
			pkg, err := ca.analyzePackage(path, goFiles, testFiles)
			if err == nil && len(pkg.Functions) > 0 {
				packages = append(packages, pkg)
			}
		}

		return nil
	})

	return packages, err
}

// shouldExclude 제외 경로 확인
func (ca *CoverageAnalyzer) shouldExclude(path string) bool {
	for _, exclude := range ca.ExcludePaths {
		if strings.Contains(path, exclude) {
			return true
		}
	}
	return false
}

// analyzePackage 패키지 분석
func (ca *CoverageAnalyzer) analyzePackage(packagePath string, goFiles, testFiles []string) (PackageInfo, error) {
	pkg := PackageInfo{
		Path:      packagePath,
		TestFiles: testFiles,
	}

	// 패키지명 추출
	relPath, _ := filepath.Rel(ca.ProjectRoot, packagePath)
	pkg.Name = strings.ReplaceAll(relPath, "/", ".")

	// 각 Go 파일 분석
	for _, file := range goFiles {
		filePath := filepath.Join(packagePath, file)
		functions, err := ca.analyzeFunctions(filePath)
		if err != nil {
			continue
		}
		pkg.Functions = append(pkg.Functions, functions...)
	}

	// 테스트 함수 매핑
	ca.mapTestFunctions(&pkg)

	// 커버리지 계산
	ca.calculatePackageCoverage(&pkg)

	return pkg, nil
}

// analyzeFunctions 함수 분석
func (ca *CoverageAnalyzer) analyzeFunctions(filePath string) ([]FunctionInfo, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	node, err := parser.ParseFile(ca.FileSet, filePath, content, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	var functions []FunctionInfo

	ast.Inspect(node, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.FuncDecl:
			if x.Name != nil {
				pos := ca.FileSet.Position(x.Pos())
				end := ca.FileSet.Position(x.End())

				function := FunctionInfo{
					Name:       x.Name.Name,
					File:       filepath.Base(filePath),
					LineStart:  pos.Line,
					LineEnd:    end.Line,
					IsExported: ast.IsExported(x.Name.Name),
					Complexity: ca.calculateCyclomaticComplexity(x),
				}

				functions = append(functions, function)
			}
		}
		return true
	})

	return functions, nil
}

// mapTestFunctions 테스트 함수 매핑
func (ca *CoverageAnalyzer) mapTestFunctions(pkg *PackageInfo) {
	testFunctions := make(map[string]bool)

	// 테스트 파일에서 테스트 함수 찾기
	for _, testFile := range pkg.TestFiles {
		filePath := filepath.Join(pkg.Path, testFile)
		functions, err := ca.analyzeFunctions(filePath)
		if err != nil {
			continue
		}

		for _, fn := range functions {
			if strings.HasPrefix(fn.Name, "Test") || strings.HasPrefix(fn.Name, "Benchmark") {
				// Test나 Benchmark로 시작하는 함수명에서 대상 함수명 추출
				targetName := strings.TrimPrefix(fn.Name, "Test")
				targetName = strings.TrimPrefix(targetName, "Benchmark")
				testFunctions[targetName] = true
			}
		}
	}

	// 각 함수에 테스트 존재 여부 표시
	for i := range pkg.Functions {
		fn := &pkg.Functions[i]
		if testFunctions[fn.Name] || testFunctions[strings.Title(fn.Name)] {
			fn.HasTest = true
		}
	}

	pkg.TestCount = len(testFunctions)
	pkg.FunctionCount = len(pkg.Functions)
}

// calculateCyclomaticComplexity 순환 복잡도 계산
func (ca *CoverageAnalyzer) calculateCyclomaticComplexity(fn *ast.FuncDecl) int {
	complexity := 1 // 기본값

	ast.Inspect(fn, func(n ast.Node) bool {
		switch n.(type) {
		case *ast.IfStmt, *ast.ForStmt, *ast.RangeStmt, *ast.SwitchStmt, *ast.TypeSwitchStmt:
			complexity++
		case *ast.CaseClause:
			complexity++
		}
		return true
	})

	return complexity
}

// calculatePackageCoverage 패키지 커버리지 계산
func (ca *CoverageAnalyzer) calculatePackageCoverage(pkg *PackageInfo) {
	if pkg.FunctionCount == 0 {
		pkg.CoverageScore = 0
		return
	}

	testedCount := 0
	for _, fn := range pkg.Functions {
		if fn.HasTest {
			testedCount++
		}
	}

	pkg.CoverageScore = float64(testedCount) / float64(pkg.FunctionCount) * 100
}

// calculateOverallCoverage 전체 커버리지 계산
func (ca *CoverageAnalyzer) calculateOverallCoverage(report *CoverageReport) {
	totalFunctions := 0
	testedFunctions := 0
	testedPackages := 0

	for _, pkg := range report.PackageReports {
		totalFunctions += pkg.FunctionCount

		packageTested := 0
		for _, fn := range pkg.Functions {
			if fn.HasTest {
				testedFunctions++
				packageTested++
			}
		}

		if packageTested > 0 {
			testedPackages++
		}
	}

	report.TotalPackages = len(report.PackageReports)
	report.TestedPackages = testedPackages
	report.TotalFunctions = totalFunctions
	report.TestedFunctions = testedFunctions

	if totalFunctions > 0 {
		report.OverallCoverage = float64(testedFunctions) / float64(totalFunctions) * 100
	}
}

// generateMissingTests 누락된 테스트 생성
func (ca *CoverageAnalyzer) generateMissingTests(report *CoverageReport) {
	var missing []MissingTestInfo

	for _, pkg := range report.PackageReports {
		for _, fn := range pkg.Functions {
			if !fn.HasTest && fn.IsExported {
				priority := "medium"
				if fn.Complexity > 5 {
					priority = "high"
				} else if fn.Complexity < 3 {
					priority = "low"
				}

				missing = append(missing, MissingTestInfo{
					Package:    pkg.Name,
					Function:   fn.Name,
					File:       fn.File,
					Line:       fn.LineStart,
					Priority:   priority,
					Complexity: fn.Complexity,
				})
			}
		}
	}

	// 우선순위별 정렬
	sort.Slice(missing, func(i, j int) bool {
		priorityOrder := map[string]int{"high": 3, "medium": 2, "low": 1}
		return priorityOrder[missing[i].Priority] > priorityOrder[missing[j].Priority]
	})

	report.MissingTests = missing
}

// generateRecommendations 권장사항 생성
func (ca *CoverageAnalyzer) generateRecommendations(report *CoverageReport) {
	var recommendations []string

	// 전체 커버리지 기반 권장사항
	if report.OverallCoverage < 70 {
		recommendations = append(recommendations, "전체 코드 커버리지가 70% 미만입니다. 핵심 기능부터 우선적으로 테스트를 작성하세요.")
	} else if report.OverallCoverage < 90 {
		recommendations = append(recommendations, "코드 커버리지가 90% 목표에 근접했습니다. 누락된 테스트를 완성하여 목표를 달성하세요.")
	} else {
		recommendations = append(recommendations, "우수한 코드 커버리지를 달성했습니다. 현재 품질을 유지하세요.")
	}

	// 패키지별 권장사항
	uncoveredPackages := 0
	for _, pkg := range report.PackageReports {
		if pkg.CoverageScore < 50 {
			uncoveredPackages++
		}
	}

	if uncoveredPackages > 0 {
		recommendations = append(recommendations, fmt.Sprintf("%d개 패키지의 커버리지가 50%% 미만입니다. 이들 패키지의 테스트를 우선 작성하세요.", uncoveredPackages))
	}

	// 복잡도 기반 권장사항
	highComplexityCount := 0
	for _, missing := range report.MissingTests {
		if missing.Priority == "high" {
			highComplexityCount++
		}
	}

	if highComplexityCount > 0 {
		recommendations = append(recommendations, fmt.Sprintf("%d개의 고복잡도 함수에 테스트가 없습니다. 이들 함수를 우선적으로 테스트하세요.", highComplexityCount))
	}

	// Mock 관련 권장사항
	servicePackages := 0
	for _, pkg := range report.PackageReports {
		if strings.Contains(pkg.Name, "services") && pkg.CoverageScore < 80 {
			servicePackages++
		}
	}

	if servicePackages > 0 {
		recommendations = append(recommendations, "서비스 레이어의 테스트 커버리지가 부족합니다. Mock 객체를 활용한 단위 테스트를 강화하세요.")
	}

	report.Recommendations = recommendations
}

// GenerateHTMLReport HTML 보고서 생성
func (ca *CoverageAnalyzer) GenerateHTMLReport(report *CoverageReport, outputPath string) error {
	html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <title>ProxyND Test Coverage Report</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .header { background: #f5f5f5; padding: 15px; border-radius: 5px; margin-bottom: 20px; }
        .summary { display: grid; grid-template-columns: repeat(4, 1fr); gap: 10px; margin-bottom: 20px; }
        .metric { background: white; border: 1px solid #ddd; padding: 15px; text-align: center; border-radius: 5px; }
        .coverage-high { background-color: #d4edda; }
        .coverage-medium { background-color: #fff3cd; }
        .coverage-low { background-color: #f8d7da; }
        table { border-collapse: collapse; width: 100%%; margin: 20px 0; }
        th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
        th { background-color: #f2f2f2; }
        .high-priority { color: #dc3545; font-weight: bold; }
        .medium-priority { color: #ffc107; }
        .low-priority { color: #28a745; }
        .recommendations { background: #e9ecef; padding: 15px; border-radius: 5px; margin-top: 20px; }
    </style>
</head>
<body>
    <div class="header">
        <h1>ProxyND Test Coverage Report</h1>
        <p>Generated: %s</p>
    </div>
    
    <div class="summary">
        <div class="metric %s">
            <h3>Overall Coverage</h3>
            <h2>%.1f%%</h2>
        </div>
        <div class="metric">
            <h3>Packages</h3>
            <h2>%d / %d</h2>
            <p>Tested</p>
        </div>
        <div class="metric">
            <h3>Functions</h3>
            <h2>%d / %d</h2>
            <p>Tested</p>
        </div>
        <div class="metric">
            <h3>Missing Tests</h3>
            <h2>%d</h2>
            <p>Functions</p>
        </div>
    </div>
    
    <h2>Package Coverage</h2>
    <table>
        <tr><th>Package</th><th>Coverage</th><th>Functions</th><th>Tested</th><th>Test Files</th></tr>
        %s
    </table>
    
    <h2>Missing Tests (High Priority)</h2>
    <table>
        <tr><th>Package</th><th>Function</th><th>File</th><th>Line</th><th>Priority</th><th>Complexity</th></tr>
        %s
    </table>
    
    <div class="recommendations">
        <h2>Recommendations</h2>
        <ul>%s</ul>
    </div>
</body>
</html>`,
		fmt.Sprintf("%v", report),
		ca.getCoverageClass(report.OverallCoverage),
		report.OverallCoverage,
		report.TestedPackages, report.TotalPackages,
		report.TestedFunctions, report.TotalFunctions,
		len(report.MissingTests),
		ca.generatePackageTableRows(report.PackageReports),
		ca.generateMissingTestRows(report.MissingTests),
		ca.generateRecommendationsList(report.Recommendations),
	)

	return os.WriteFile(outputPath, []byte(html), 0644)
}

// getCoverageClass 커버리지 클래스 반환
func (ca *CoverageAnalyzer) getCoverageClass(coverage float64) string {
	if coverage >= 80 {
		return "coverage-high"
	} else if coverage >= 60 {
		return "coverage-medium"
	}
	return "coverage-low"
}

// generatePackageTableRows 패키지 테이블 행 생성
func (ca *CoverageAnalyzer) generatePackageTableRows(packages []PackageInfo) string {
	var rows strings.Builder

	for _, pkg := range packages {
		tested := 0
		for _, fn := range pkg.Functions {
			if fn.HasTest {
				tested++
			}
		}

		coverageClass := ca.getCoverageClass(pkg.CoverageScore)
		rows.WriteString(fmt.Sprintf(
			`<tr class="%s"><td>%s</td><td>%.1f%%</td><td>%d</td><td>%d</td><td>%d</td></tr>`,
			coverageClass, pkg.Name, pkg.CoverageScore, pkg.FunctionCount, tested, len(pkg.TestFiles),
		))
	}

	return rows.String()
}

// generateMissingTestRows 누락된 테스트 행 생성
func (ca *CoverageAnalyzer) generateMissingTestRows(missing []MissingTestInfo) string {
	var rows strings.Builder

	count := 0
	for _, test := range missing {
		if count >= 20 { // 상위 20개만 표시
			break
		}

		priorityClass := test.Priority + "-priority"
		rows.WriteString(fmt.Sprintf(
			`<tr><td>%s</td><td>%s</td><td>%s</td><td>%d</td><td class="%s">%s</td><td>%d</td></tr>`,
			test.Package, test.Function, test.File, test.Line, priorityClass, test.Priority, test.Complexity,
		))
		count++
	}

	return rows.String()
}

// generateRecommendationsList 권장사항 리스트 생성
func (ca *CoverageAnalyzer) generateRecommendationsList(recommendations []string) string {
	var items strings.Builder

	for _, rec := range recommendations {
		items.WriteString(fmt.Sprintf("<li>%s</li>", rec))
	}

	return items.String()
}

// PrintSummary 요약 출력
func (ca *CoverageAnalyzer) PrintSummary(report *CoverageReport) {
	fmt.Printf("=== ProxyND Test Coverage Analysis ===\n")
	fmt.Printf("Overall Coverage: %.1f%%\n", report.OverallCoverage)
	fmt.Printf("Packages: %d tested / %d total\n", report.TestedPackages, report.TotalPackages)
	fmt.Printf("Functions: %d tested / %d total\n", report.TestedFunctions, report.TotalFunctions)
	fmt.Printf("Missing Tests: %d functions\n", len(report.MissingTests))
	fmt.Printf("\nTop Missing Tests (High Priority):\n")

	count := 0
	for _, missing := range report.MissingTests {
		if missing.Priority == "high" && count < 10 {
			fmt.Printf("  - %s.%s (complexity: %d)\n", missing.Package, missing.Function, missing.Complexity)
			count++
		}
	}

	fmt.Printf("\nRecommendations:\n")
	for _, rec := range report.Recommendations {
		fmt.Printf("  - %s\n", rec)
	}
}
