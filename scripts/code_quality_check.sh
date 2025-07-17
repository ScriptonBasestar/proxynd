#!/bin/bash
# scripts/code_quality_check.sh

echo "=== 코드 품질 검사 ==="

# 1. TODO/FIXME 개수 확인
todo_count=$(find . -name "*.go" -exec grep -i "todo\|fixme" {} + | wc -l | tr -d ' ')
echo "남은 TODO/FIXME: ${todo_count}개"

# 2. 임시 코드 확인
temp_code=$(find . -name "*.go" -exec grep -i "temp\|tmp\|hack\|workaround" {} + | wc -l | tr -d ' ')
echo "임시 코드: ${temp_code}개"

# 3. 주석 처리된 코드 확인
commented_code=$(find . -name "*.go" -exec grep "^\s*//.*[{}();]" {} + | wc -l | tr -d ' ')
echo "주석 처리된 코드: ${commented_code}개"

# 4. 미사용 import 확인 (goimports가 설치되어 있을 경우)
if command -v goimports >/dev/null 2>&1; then
    unused_imports=$(goimports -l . 2>/dev/null | wc -l | tr -d ' ')
    echo "잘못된 import: ${unused_imports}개"
else
    echo "goimports가 설치되어 있지 않습니다. 설치 방법: go install golang.org/x/tools/cmd/goimports@latest"
    unused_imports=0
fi

# 5. 하드코딩된 설정 확인
hardcoded_config=$(find . -name "*.go" -exec grep -E "(localhost|127\.0\.0\.1|:8080|:3306)" {} + | wc -l | tr -d ' ')
echo "하드코딩된 설정: ${hardcoded_config}개"

# 6. 에러 처리 누락 확인
missing_error_handling=$(find . -name "*.go" -exec grep -E "_, _ =" {} + | wc -l | tr -d ' ')
echo "에러 처리 누락 가능성: ${missing_error_handling}개"

echo ""
echo "=== 검사 기준 ==="

# 기준 확인
fail_count=0

if [ "$todo_count" -gt 50 ]; then
    echo "❌ TODO 항목이 너무 많습니다 (50개 초과)"
    fail_count=$((fail_count + 1))
else
    echo "✅ TODO 항목: 적정 수준 (${todo_count}/50)"
fi

if [ "$temp_code" -gt 10 ]; then
    echo "❌ 임시 코드가 너무 많습니다 (10개 초과)"
    fail_count=$((fail_count + 1))
else
    echo "✅ 임시 코드: 적정 수준 (${temp_code}/10)"
fi

if [ "$commented_code" -gt 20 ]; then
    echo "❌ 주석 처리된 코드가 너무 많습니다 (20개 초과)"
    fail_count=$((fail_count + 1))
else
    echo "✅ 주석 처리된 코드: 적정 수준 (${commented_code}/20)"
fi

if [ "$hardcoded_config" -gt 5 ]; then
    echo "❌ 하드코딩된 설정이 너무 많습니다 (5개 초과)"
    fail_count=$((fail_count + 1))
else
    echo "✅ 하드코딩된 설정: 적정 수준 (${hardcoded_config}/5)"
fi

echo ""
if [ $fail_count -eq 0 ]; then
    echo "🎉 코드 품질 검사 통과!"
    exit 0
else
    echo "❌ 코드 품질 검사 실패: ${fail_count}개 항목이 기준을 초과했습니다."
    exit 1
fi