package unit

import (
	"log"
	"path/filepath"
	"testing"
)

// RunCoverageAnalysis 커버리지 분석 실행
func RunCoverageAnalysis(t *testing.T) {
	// 프로젝트 루트 경로 계산
	projectRoot := filepath.Join("..", "..")

	// 커버리지 분석기 생성
	analyzer := NewCoverageAnalyzer(projectRoot)

	// 커버리지 분석 실행
	report, err := analyzer.AnalyzeCoverage()
	if err != nil {
		t.Fatalf("Failed to run coverage analysis: %v", err)
	}

	// 콘솔 요약 출력
	analyzer.PrintSummary(report)

	// HTML 보고서 생성
	htmlOutputPath := filepath.Join(projectRoot, "tests", "coverage_report.html")
	err = analyzer.GenerateHTMLReport(report, htmlOutputPath)
	if err != nil {
		log.Printf("Failed to generate HTML report: %v", err)
	} else {
		t.Logf("HTML coverage report generated: %s", htmlOutputPath)
	}

	// 90% 커버리지 목표 검증
	if report.OverallCoverage < 90.0 {
		t.Logf("WARNING: Overall coverage (%.1f%%) is below 90%% target", report.OverallCoverage)

		// 높은 우선순위 누락 테스트 출력
		highPriorityCount := 0
		for _, missing := range report.MissingTests {
			if missing.Priority == "high" {
				highPriorityCount++
				if highPriorityCount <= 10 {
					t.Logf("  High priority missing test: %s.%s (complexity: %d)",
						missing.Package, missing.Function, missing.Complexity)
				}
			}
		}

		if highPriorityCount > 10 {
			t.Logf("  ... and %d more high priority missing tests", highPriorityCount-10)
		}
	} else {
		t.Logf("SUCCESS: Achieved %.1f%% coverage target", report.OverallCoverage)
	}

	// 권장사항 출력
	for _, recommendation := range report.Recommendations {
		t.Logf("RECOMMENDATION: %s", recommendation)
	}
}

// TestCoverageAnalysis 커버리지 분석 테스트
func TestCoverageAnalysis(t *testing.T) {
	RunCoverageAnalysis(t)
}
