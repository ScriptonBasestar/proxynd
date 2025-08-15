#!/bin/bash

# 스크립트명: 커밋 전 검증 도구
# 용도: 코드 커밋 전 품질 및 아키텍처 규칙 검증
# 사용법: pre-commit-validation.sh [--skip-tests] [--fix]
# 예시: pre-commit-validation.sh --skip-tests

set -euo pipefail

# 설정
SKIP_TESTS=false
FIX_MODE=false
EXIT_CODE=0

# 명령행 인수 파싱
while [[ $# -gt 0 ]]; do
    case $1 in
        --skip-tests)
            SKIP_TESTS=true
            shift
            ;;
        --fix)
            FIX_MODE=true
            shift
            ;;
        --help)
            echo "사용법: $0 [--skip-tests] [--fix]"
            echo "  --skip-tests: 테스트 스킵 (빠른 검증)"
            echo "  --fix: 자동 수정 시도"
            exit 0
            ;;
        *)
            echo "알 수 없는 옵션: $1" >&2
            exit 1
            ;;
    esac
done

# 색상 정의
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# 로깅 함수
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
    EXIT_CODE=1
}

# 타이머 시작
start_time=$(date +%s)

log_info "🛡️ ProxyND Pre-Commit Validation"
log_info "Ensuring code quality and architecture compliance..."

# 1. Git 상태 확인
log_info "📋 Checking Git status..."

if ! git diff --cached --quiet; then
    log_success "✓ Staged changes detected"
else
    log_warning "No staged changes found - nothing to validate"
    exit 0
fi

# 2. Go 코드 포맷팅 검증/수정
log_info "🎨 Checking Go code formatting..."

unformatted_files=$(gofmt -l . | head -10 || true)
if [[ -n "$unformatted_files" ]]; then
    if [[ "$FIX_MODE" == "true" ]]; then
        log_info "Auto-formatting Go files..."
        gofmt -w .
        log_success "✓ Go files formatted"
    else
        log_error "Unformatted Go files found:"
        echo "$unformatted_files" | while read file; do
            log_error "  $file"
        done
        log_info "💡 Run with --fix to auto-format"
    fi
else
    log_success "✓ All Go files properly formatted"
fi

# 3. 빌드 검증
log_info "🔨 Verifying build..."

if make build > /dev/null 2>&1; then
    log_success "✓ Build successful"
else
    log_error "❌ Build failed"
    log_error "Please fix build errors before committing"
    exit 1
fi

# 4. 아키텍처 규칙 검증
log_info "🏛️ Validating architecture rules..."

if scripts/validate-architecture.sh > /dev/null 2>&1; then
    log_success "✓ Architecture rules compliant"
else
    log_error "❌ Architecture violations detected"
    log_info "Running detailed architecture validation..."
    scripts/validate-architecture.sh
    exit 1
fi

# 5. 단위 테스트 (선택적)
if [[ "$SKIP_TESTS" == "false" ]]; then
    log_info "🧪 Running unit tests..."

    if make test-unit > /dev/null 2>&1; then
        log_success "✓ Unit tests passed"
    else
        log_error "❌ Unit tests failed"
        log_error "Please fix failing tests before committing"
        exit 1
    fi
else
    log_warning "⏭️ Unit tests skipped"
fi

# 6. API 회귀 검증 (CRITICAL)
log_info "🌐 Checking API compatibility..."

if make verify-api > /dev/null 2>&1; then
    log_success "✓ API compatibility verified"
else
    log_error "❌ API regression detected"
    log_error "This is a CRITICAL issue - external API contracts changed"
    log_error "Please ensure external API behavior is maintained"
    exit 1
fi

# 7. 보안 검사 (간단한 패턴 매칭)
log_info "🔒 Basic security check..."

security_issues=$(grep -r "password\|secret\|key" --include="*.go" . | grep -v "_test.go" | grep -i "=.*\"" | head -5 || true)
if [[ -n "$security_issues" ]]; then
    log_warning "Potential hardcoded secrets found:"
    echo "$security_issues" | while read line; do
        log_warning "  $line"
    done
    log_warning "Please review for hardcoded credentials"
fi

# 8. 대용량 파일 검사
log_info "📁 Checking for large files..."

large_files=$(find . -type f -size +1M -not -path "./.git/*" -not -path "./vendor/*" | head -5 || true)
if [[ -n "$large_files" ]]; then
    log_warning "Large files detected:"
    echo "$large_files" | while read file; do
        size=$(du -h "$file" | cut -f1)
        log_warning "  $file ($size)"
    done
fi

# 9. TODO/FIXME 긴급도 검사
log_info "📝 Checking TODO/FIXME markers..."

critical_fixmes=$(grep -r "FIXME(CRITICAL)" --include="*.go" . | head -3 || true)
if [[ -n "$critical_fixmes" ]]; then
    log_warning "Critical FIXMEs found:"
    echo "$critical_fixmes" | while read line; do
        log_warning "  $line"
    done
fi

# 10. 의존성 취약점 간단 검사
log_info "🔍 Basic dependency check..."

if command -v go &> /dev/null; then
    # go.mod가 있는지 확인
    if [[ -f "go.mod" ]]; then
        # 의존성 정리 확인
        if ! go mod tidy -diff > /dev/null 2>&1; then
            log_warning "go.mod needs tidying"
            if [[ "$FIX_MODE" == "true" ]]; then
                go mod tidy
                log_success "✓ go.mod tidied"
            fi
        fi
    fi
fi

# 실행 시간 계산
end_time=$(date +%s)
duration=$((end_time - start_time))

# 결과 요약
echo ""
log_info "🎯 Pre-Commit Validation Summary"
log_info "⏱️ Validation completed in ${duration}s"

if [[ $EXIT_CODE -eq 0 ]]; then
    log_success "✅ All validations passed!"
    log_success "🚀 Code is ready for commit"
else
    log_error "❌ Validation failures detected"
    log_error "🛑 Please fix issues before committing"
    echo ""
    log_info "💡 Quick fixes:"
    log_info "  - Run with --fix for auto-fixes"
    log_info "  - Use --skip-tests for faster validation"
    log_info "  - Check CLAUDE.md for detailed guidelines"
fi

# Git hooks용 출력
if [[ $EXIT_CODE -eq 0 ]]; then
    echo "✅ Pre-commit validation passed"
else
    echo "❌ Pre-commit validation failed"
fi

exit $EXIT_CODE
