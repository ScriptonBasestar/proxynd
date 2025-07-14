#!/bin/bash
set -e

# ProxyND 테스트 커버리지 스크립트

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

echo "🧪 ProxyND Test Coverage Analysis"
echo "=================================="

cd "$PROJECT_ROOT"

# 설정
COVERAGE_OUT="coverage.out"
COVERAGE_HTML="coverage.html"
COVERAGE_TXT="coverage.txt"
MIN_COVERAGE=70

# 클린업 함수
cleanup() {
    echo "🧹 Cleaning up temporary files..."
    rm -f "$COVERAGE_OUT" "$COVERAGE_TXT" 2>/dev/null || true
}

# 인터럽트 시 클린업
trap cleanup EXIT

echo "📋 Running tests with coverage..."

# 패키지별 커버리지 수집 (빌드 실패 무시)
echo "🔍 Collecting coverage data..."
go test -short -coverprofile="$COVERAGE_OUT" ./... 2>/dev/null || {
    echo "⚠️  Some tests failed, but continuing with coverage analysis..."
    # 커버리지 파일이 없으면 생성
    if [ ! -f "$COVERAGE_OUT" ]; then
        echo "mode: set" > "$COVERAGE_OUT"
    fi
}

# 커버리지 파일이 비어있는지 확인
if [ ! -s "$COVERAGE_OUT" ]; then
    echo "❌ No coverage data collected"
    exit 1
fi

echo "📊 Generating coverage reports..."

# HTML 리포트 생성
go tool cover -html="$COVERAGE_OUT" -o "$COVERAGE_HTML"
echo "✅ HTML coverage report: $COVERAGE_HTML"

# 텍스트 리포트 생성
go tool cover -func="$COVERAGE_OUT" > "$COVERAGE_TXT"

# 전체 커버리지 추출
total_coverage=$(go tool cover -func="$COVERAGE_OUT" | grep total | awk '{print $3}' | sed 's/%//')

echo ""
echo "📈 Coverage Summary:"
echo "==================="

# 패키지별 커버리지 상위 10개
echo "🏆 Top 10 packages by coverage:"
grep -v "total" "$COVERAGE_TXT" | sort -k3 -nr | head -10 | while read line; do
    package=$(echo "$line" | awk '{print $1}' | sed 's/.*\///')
    coverage=$(echo "$line" | awk '{print $3}')
    echo "  $package: $coverage"
done

echo ""

# 커버리지가 낮은 패키지 상위 10개
echo "⚠️  Bottom 10 packages by coverage:"
grep -v "total" "$COVERAGE_TXT" | grep -v "0.0%" | sort -k3 -n | head -10 | while read line; do
    package=$(echo "$line" | awk '{print $1}' | sed 's/.*\///')
    coverage=$(echo "$line" | awk '{print $3}')
    echo "  $package: $coverage"
done

echo ""
echo "🎯 Overall Coverage: $total_coverage%"

# 커버리지 목표 달성 여부 확인
if (( $(echo "$total_coverage >= $MIN_COVERAGE" | bc -l) )); then
    echo "✅ Coverage target achieved! ($total_coverage% >= $MIN_COVERAGE%)"
    exit_code=0
else
    echo "❌ Coverage target not met ($total_coverage% < $MIN_COVERAGE%)"
    echo "💡 Focus on improving coverage in low-coverage packages"
    exit_code=1
fi

# 커버리지가 0%인 패키지 찾기
zero_coverage_packages=$(grep "0.0%" "$COVERAGE_TXT" | wc -l)
if [ "$zero_coverage_packages" -gt 0 ]; then
    echo ""
    echo "🚨 Packages with 0% coverage ($zero_coverage_packages packages):"
    grep "0.0%" "$COVERAGE_TXT" | awk '{print $1}' | sed 's/.*\///' | head -10
    if [ "$zero_coverage_packages" -gt 10 ]; then
        echo "  ... and $((zero_coverage_packages - 10)) more"
    fi
fi

# 통계 정보
echo ""
echo "📊 Statistics:"
total_packages=$(grep -v "total" "$COVERAGE_TXT" | wc -l)
covered_packages=$(grep -v "total" "$COVERAGE_TXT" | grep -v "0.0%" | wc -l)
echo "  Total packages: $total_packages"
echo "  Packages with tests: $covered_packages"
echo "  Packages without tests: $((total_packages - covered_packages))"

# 커버리지 추세 (이전 실행과 비교)
COVERAGE_HISTORY=".coverage_history"
if [ -f "$COVERAGE_HISTORY" ]; then
    last_coverage=$(tail -1 "$COVERAGE_HISTORY" | awk '{print $2}')
    if [ -n "$last_coverage" ]; then
        diff=$(echo "$total_coverage - $last_coverage" | bc -l)
        if (( $(echo "$diff > 0" | bc -l) )); then
            echo "📈 Coverage improved by ${diff}% since last run"
        elif (( $(echo "$diff < 0" | bc -l) )); then
            echo "📉 Coverage decreased by ${diff#-}% since last run"
        else
            echo "➡️  Coverage unchanged since last run"
        fi
    fi
fi

# 현재 커버리지 기록
echo "$(date '+%Y-%m-%d %H:%M:%S') $total_coverage" >> "$COVERAGE_HISTORY"

echo ""
echo "🎉 Coverage analysis complete!"
echo "📁 View detailed report: open $COVERAGE_HTML"

exit $exit_code