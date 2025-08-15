#!/bin/bash

# 스크립트명: 아키텍처 규칙 검증 도구
# 용도: Hexagonal + Clean Architecture 규칙 준수 여부 검증
# 사용법: validate-architecture.sh [--verbose] [--fix]
# 예시: validate-architecture.sh --verbose

set -euo pipefail

# 설정
VERBOSE=false
FIX_MODE=false
EXIT_CODE=0

# 명령행 인수 파싱
while [[ $# -gt 0 ]]; do
    case $1 in
        --verbose)
            VERBOSE=true
            shift
            ;;
        --fix)
            FIX_MODE=true
            shift
            ;;
        --help)
            echo "사용법: $0 [--verbose] [--fix]"
            echo "  --verbose: 상세 출력"
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
NC='\033[0m' # No Color

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

verbose_log() {
    if [[ "$VERBOSE" == "true" ]]; then
        echo -e "${BLUE}[VERBOSE]${NC} $1"
    fi
}

# 아키텍처 검증 시작
log_info "🏛️ ProxyND Architecture Validation"
log_info "Checking Hexagonal + Clean Architecture compliance..."

# 1. 디렉토리 구조 검증
log_info "📁 Checking directory structure..."

check_directory_structure() {
    local required_dirs=(
        "internal/domain"
        "internal/usecase"
        "internal/ports"
        "internal/adapters"
    )

    for dir in "${required_dirs[@]}"; do
        if [[ ! -d "$dir" ]]; then
            log_warning "Missing directory: $dir"
            if [[ "$FIX_MODE" == "true" ]]; then
                mkdir -p "$dir"
                log_info "Created directory: $dir"
            fi
        else
            verbose_log "✓ Directory exists: $dir"
        fi
    done
}

# 2. 의존성 방향 검증
log_info "🔄 Checking dependency direction..."

check_dependency_direction() {
    verbose_log "Analyzing Go import dependencies..."

    # Domain 계층 검증 (아무것도 의존하면 안됨)
    log_info "Checking domain layer independence..."
    domain_violations=$(go list -f '{{.ImportPath}} {{.Imports}}' ./internal/domain/... 2>/dev/null | \
        grep -E "(usecase|ports|adapters)" | head -5 || true)

    if [[ -n "$domain_violations" ]]; then
        log_error "Domain layer has invalid dependencies:"
        echo "$domain_violations" | while read line; do
            log_error "  $line"
        done
    else
        log_success "✓ Domain layer is independent"
    fi

    # Usecase 계층 검증 (adapters 의존 금지)
    log_info "Checking usecase layer dependencies..."
    usecase_violations=$(go list -f '{{.ImportPath}} {{.Imports}}' ./internal/usecase/... 2>/dev/null | \
        grep "adapters" | head -5 || true)

    if [[ -n "$usecase_violations" ]]; then
        log_error "Usecase layer depends on adapters (forbidden):"
        echo "$usecase_violations" | while read line; do
            log_error "  $line"
        done
    else
        log_success "✓ Usecase layer dependencies are clean"
    fi

    # Ports 계층 검증 (adapters 의존 금지)
    log_info "Checking ports layer dependencies..."
    ports_violations=$(go list -f '{{.ImportPath}} {{.Imports}}' ./internal/ports/... 2>/dev/null | \
        grep "adapters" | head -5 || true)

    if [[ -n "$ports_violations" ]]; then
        log_error "Ports layer depends on adapters (forbidden):"
        echo "$ports_violations" | while read line; do
            log_error "  $line"
        done
    else
        log_success "✓ Ports layer dependencies are clean"
    fi
}

# 3. 순환 의존성 검증
log_info "🔄 Checking for circular dependencies..."

check_circular_dependencies() {
    verbose_log "Running Go dependency analysis..."

    # Go 모듈의 순환 의존성 검사
    circular_deps=$(go list -deps ./... 2>&1 | grep -i "import cycle" || true)

    if [[ -n "$circular_deps" ]]; then
        log_error "Circular dependencies detected:"
        echo "$circular_deps" | while read line; do
            log_error "  $line"
        done
    else
        log_success "✓ No circular dependencies found"
    fi

    # 패키지별 의존성 깊이 분석
    if [[ "$VERBOSE" == "true" ]]; then
        log_info "Dependency complexity analysis:"
        go list -f '{{.ImportPath}} {{len .Imports}}' ./internal/... | \
            sort -k2 -nr | head -10 | while read pkg count; do
            verbose_log "  $pkg: $count dependencies"
        done
    fi
}

# 4. 파일 명명 규칙 검증
log_info "📝 Checking file naming conventions..."

check_naming_conventions() {
    # Go 파일 명명 규칙 검증
    invalid_files=$(find . -name "*.go" -not -path "./vendor/*" -not -path "./.git/*" | \
        grep -E "[A-Z]" | head -5 || true)

    if [[ -n "$invalid_files" ]]; then
        log_warning "Files with uppercase names found (Go convention: lowercase):"
        echo "$invalid_files" | while read file; do
            log_warning "  $file"
        done
    else
        log_success "✓ File naming conventions followed"
    fi

    # 테스트 파일 위치 검증
    misplaced_tests=$(find ./internal -name "*_test.go" -not -path "*/tests/*" | head -5 || true)
    if [[ -n "$misplaced_tests" ]]; then
        verbose_log "Test files in source directories (acceptable for Go):"
        echo "$misplaced_tests" | while read file; do
            verbose_log "  $file"
        done
    fi
}

# 5. 인터페이스 정의 검증
log_info "🔌 Checking interface definitions..."

check_interface_definitions() {
    # Ports 디렉토리에 인터페이스가 정의되어 있는지 확인
    if [[ -d "internal/ports" ]]; then
        interface_files=$(find internal/ports -name "*.go" | wc -l)
        if [[ $interface_files -eq 0 ]]; then
            log_warning "No interface files found in internal/ports/"
        else
            log_success "✓ Interface files found in ports layer: $interface_files files"
        fi
    fi

    # 어댑터에서 인터페이스 구현 여부 확인 (정적 분석으로는 한계가 있음)
    if [[ "$VERBOSE" == "true" ]]; then
        log_info "Scanning for interface implementations..."
        grep -r "type.*struct" internal/adapters/ 2>/dev/null | wc -l | \
            xargs -I {} verbose_log "Found {} adapter structs"
    fi
}

# 6. 패키지 구조 검증
log_info "📦 Checking package structure..."

check_package_structure() {
    # 각 계층별 패키지 수 확인
    domain_packages=$(find internal/domain -name "*.go" -type f 2>/dev/null | wc -l || echo "0")
    usecase_packages=$(find internal/usecase -name "*.go" -type f 2>/dev/null | wc -l || echo "0")
    ports_packages=$(find internal/ports -name "*.go" -type f 2>/dev/null | wc -l || echo "0")
    adapters_packages=$(find internal/adapters -name "*.go" -type f 2>/dev/null | wc -l || echo "0")

    log_info "Package distribution:"
    log_info "  Domain:   $domain_packages files"
    log_info "  Usecase:  $usecase_packages files"
    log_info "  Ports:    $ports_packages files"
    log_info "  Adapters: $adapters_packages files"

    # 균형성 검증 (어댑터가 너무 많으면 경고)
    if [[ $adapters_packages -gt $((domain_packages + usecase_packages + ports_packages)) ]]; then
        log_warning "Adapters layer is significantly larger than other layers"
        log_warning "Consider splitting into smaller, focused adapters"
    fi
}

# 7. TODO/FIXME 마커 검증
log_info "📝 Checking TODO/FIXME markers..."

check_todo_markers() {
    if [[ "$VERBOSE" == "true" ]]; then
        # HEXAGONAL_MIGRATION 마커 확인
        migration_todos=$(grep -r "HEXAGONAL_MIGRATION" --include="*.go" . | wc -l || echo "0")
        log_info "Migration TODOs found: $migration_todos"

        # 긴급도별 FIXME 확인
        critical_fixmes=$(grep -r "FIXME(CRITICAL)" --include="*.go" . | wc -l || echo "0")
        if [[ $critical_fixmes -gt 0 ]]; then
            log_warning "Critical FIXMEs found: $critical_fixmes"
        fi
    fi
}

# 검증 실행
check_directory_structure
check_dependency_direction
check_circular_dependencies
check_naming_conventions
check_interface_definitions
check_package_structure
check_todo_markers

# 결과 요약
echo ""
log_info "🎯 Architecture Validation Summary"

if [[ $EXIT_CODE -eq 0 ]]; then
    log_success "✅ All architecture rules passed!"
    log_success "🏛️ Hexagonal + Clean Architecture compliance verified"
else
    log_error "❌ Architecture violations detected"
    log_error "🔧 Please review and fix the issues above"

    if [[ "$FIX_MODE" == "false" ]]; then
        log_info "💡 Run with --fix to attempt automatic fixes"
    fi
fi

# 추가 권장사항
log_info "💡 Recommendations:"
log_info "  1. Run 'make test-unit' to verify functionality"
log_info "  2. Run 'make verify-api' to check API compatibility"
log_info "  3. Review CLAUDE.md for detailed guidelines"

exit $EXIT_CODE
