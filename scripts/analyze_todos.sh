#!/bin/bash
# scripts/analyze_todos.sh

echo "=== TODO/FIXME 분석 결과 ==="
echo ""

# TODO 항목 수집
echo "1. 전체 TODO/FIXME 개수:"
total_count=$(find . -name "*.go" -exec grep -n -i "todo\|fixme" {} + | wc -l | tr -d ' ')
echo "총 ${total_count}개"

echo ""
echo "2. 파일별 분포:"
find . -name "*.go" -exec grep -l -i "todo\|fixme" {} + | while read file; do
    count=$(grep -c -i "todo\|fixme" "$file")
    echo "$file: $count개"
done | sort -t: -k2 -nr | head -10

echo ""
echo "3. 우선순위별 분류:"
echo "Critical (보안/성능):"
find . -name "*.go" -exec grep -n -i "todo.*critical\|fixme.*critical\|todo.*security\|fixme.*security\|todo.*performance\|fixme.*performance" {} +

echo ""
echo "High (기능 미구현):"
find . -name "*.go" -exec grep -n -i "todo.*implement\|fixme.*implement\|todo.*missing\|fixme.*missing" {} +

echo ""
echo "Medium (개선 필요):"
find . -name "*.go" -exec grep -n -i "todo.*improve\|fixme.*improve\|todo.*optimize\|fixme.*optimize" {} +

echo ""
echo "Low (정리 필요):"
find . -name "*.go" -exec grep -n -i "todo.*cleanup\|fixme.*cleanup\|todo.*refactor\|fixme.*refactor" {} +

echo ""
echo "4. 상세 목록:"
find . -name "*.go" -exec grep -n -i "todo\|fixme" {} + | while IFS=: read -r file line content; do
    echo "📁 $file:$line"
    echo "   $content"
    echo ""
done