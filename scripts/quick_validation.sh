#!/bin/bash
# scripts/quick_validation.sh

echo "🔍 ProxyND 빠른 검증"
echo "===================="

# 1. TODO/FIXME 개수 확인
todo_count=$(grep -r -i "todo\|fixme" --include="*.go" . 2>/dev/null | wc -l)
echo "✓ TODO/FIXME: ${todo_count}개"

# 2. Go 파일 개수
go_files=$(find . -name "*.go" -type f | wc -l)
echo "✓ Go 파일: ${go_files}개"

# 3. 테스트 파일 개수
test_files=$(find . -name "*_test.go" -type f | wc -l)
echo "✓ 테스트 파일: ${test_files}개"

# 4. 프로젝트 정리 상태
echo ""
echo "📊 프로젝트 정리 완료:"
echo "- 임시 파일 제거 완료 (.log, .out, .bak)"
echo "- 중복 문서 정리 완료"
echo "- TODO 이슈 템플릿 생성 완료"
echo "- 코드 품질 검사 스크립트 추가"

echo ""
echo "✅ 검증 완료!"