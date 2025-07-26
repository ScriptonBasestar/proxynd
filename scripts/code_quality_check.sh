#!/bin/bash
# scripts/code_quality_check.sh

echo "=== 코드 품질 검사 ==="

# 1. TODO/FIXME 개수 확인
todo_count=$(grep -r -i "todo\|fixme" --include="*.go" . 2>/dev/null | wc -l)
echo "남은 TODO/FIXME: $todo_count개"

# 2. 임시 코드 확인
temp_code=$(grep -r -i "temp\|tmp\|hack\|workaround" --include="*.go" . 2>/dev/null | wc -l)
echo "임시 코드: $temp_code개"

# 3. 주석 처리된 코드 확인
commented_code=$(grep -r "^\s*//.*[{}();]" --include="*.go" . 2>/dev/null | wc -l)
echo "주석 처리된 코드: $commented_code개"

# 4. 미사용 import 확인
if command -v goimports &> /dev/null; then
    unused_imports=$(goimports -l . 2>/dev/null | wc -l)
    echo "잘못된 import: $unused_imports개"
else
    echo "goimports가 설치되지 않아 import 검사를 건너뜁니다"
fi

# 기준 확인
if [ $todo_count -gt 50 ]; then
    echo "❌ TODO 항목이 너무 많습니다 (50개 초과)"
    exit 1
fi

if [ $temp_code -gt 10 ]; then
    echo "❌ 임시 코드가 너무 많습니다 (10개 초과)"
    exit 1
fi

echo "✅ 코드 품질 검사 통과"
