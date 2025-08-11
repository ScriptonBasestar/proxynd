#!/bin/bash
# 스크립트명: smoke-test-adapters.sh
# 용도: CI/CD 파이프라인에서 어댑터 헬스체크 수행 및 빌드 실패 처리
# 사용법: smoke-test-adapters.sh [--endpoint=URL] [--timeout=SECONDS] [--verbose]
# 예시: smoke-test-adapters.sh --endpoint=http://localhost:8080 --timeout=30 --verbose

set -euo pipefail

# 기본값 설정
ENDPOINT="http://localhost:8080"
TIMEOUT=30
VERBOSE=false
EXIT_CODE=0

# 색상 코드
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 도움말 함수
show_help() {
    echo "어댑터 헬스체크 스모크 테스트 스크립트"
    echo ""
    echo "사용법: $0 [옵션]"
    echo ""
    echo "옵션:"
    echo "  --endpoint=URL      ProxyND 서버 엔드포인트 (기본값: http://localhost:8080)"
    echo "  --timeout=SECONDS   헬스체크 타임아웃 (기본값: 30초)"
    echo "  --verbose          상세 출력 활성화"
    echo "  --help             이 도움말 표시"
    echo ""
    echo "예시:"
    echo "  $0 --endpoint=https://staging.proxynd.example.com --timeout=60 --verbose"
}

# 인자 파싱
for arg in "$@"; do
    case $arg in
        --endpoint=*)
            ENDPOINT="${arg#*=}"
            shift
            ;;
        --timeout=*)
            TIMEOUT="${arg#*=}"
            shift
            ;;
        --verbose)
            VERBOSE=true
            shift
            ;;
        --help)
            show_help
            exit 0
            ;;
        *)
            echo -e "${RED}❌ 알 수 없는 옵션: $arg${NC}"
            show_help
            exit 1
            ;;
    esac
done

# 로깅 함수들
log_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

log_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

log_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

log_error() {
    echo -e "${RED}❌ $1${NC}"
}

log_verbose() {
    if [ "$VERBOSE" = true ]; then
        echo -e "${BLUE}🔍 $1${NC}"
    fi
}

# HTTP 요청 함수
make_request() {
    local url="$1"
    local timeout="$2"

    if command -v curl >/dev/null 2>&1; then
        curl -s -f --max-time "$timeout" --connect-timeout 10 "$url"
    elif command -v wget >/dev/null 2>&1; then
        wget -qO- --timeout="$timeout" "$url"
    else
        log_error "curl 또는 wget이 필요합니다"
        exit 1
    fi
}

# JSON 파싱 함수 (jq가 없는 경우를 위한 간단한 파서)
get_json_value() {
    local json="$1"
    local key="$2"

    if command -v jq >/dev/null 2>&1; then
        echo "$json" | jq -r ".$key // empty"
    else
        # jq가 없는 경우 간단한 grep/sed를 사용한 파싱
        echo "$json" | grep -o "\"$key\"[[:space:]]*:[[:space:]]*[^,}]*" | sed 's/.*:[[:space:]]*//' | tr -d '"'
    fi
}

# 배열 값 추출 함수
get_json_array_values() {
    local json="$1"
    local key="$2"

    if command -v jq >/dev/null 2>&1; then
        echo "$json" | jq -r ".$key[]? // empty"
    else
        # jq가 없는 경우 간단한 파싱
        echo "$json" | grep -o "\"$key\"[[:space:]]*:[[:space:]]*\[[^]]*\]" | sed 's/.*\[//' | sed 's/\].*//' | tr -d '"' | tr ',' '\n'
    fi
}

# 메인 헬스체크 함수
perform_health_check() {
    log_info "어댑터 헬스체크 시작..."
    log_verbose "엔드포인트: $ENDPOINT"
    log_verbose "타임아웃: ${TIMEOUT}초"

    # 기본 헬스체크
    log_info "기본 헬스체크 수행 중..."
    local basic_health_url="$ENDPOINT/healthz"
    if ! basic_response=$(make_request "$basic_health_url" "$TIMEOUT"); then
        log_error "기본 헬스체크 실패: $basic_health_url"
        return 1
    fi

    local basic_status
    basic_status=$(get_json_value "$basic_response" "status")
    log_verbose "기본 헬스체크 상태: $basic_status"

    if [ "$basic_status" != "healthy" ] && [ "$basic_status" != "degraded" ]; then
        log_error "기본 헬스체크 상태가 비정상입니다: $basic_status"
        return 1
    fi

    log_success "기본 헬스체크 통과"

    # 어댑터 헬스체크
    log_info "어댑터 헬스체크 수행 중..."
    local adapter_health_url="$ENDPOINT/health/adapters"
    if ! adapter_response=$(make_request "$adapter_health_url" "$TIMEOUT"); then
        log_error "어댑터 헬스체크 실패: $adapter_health_url"
        return 1
    fi

    local adapter_status
    adapter_status=$(get_json_value "$adapter_response" "status")
    log_verbose "어댑터 헬스체크 상태: $adapter_status"

    local total_count
    total_count=$(get_json_value "$adapter_response" "total_count")
    log_verbose "총 어댑터 개수: $total_count"

    local healthy_count
    healthy_count=$(get_json_value "$adapter_response" "healthy_count")
    log_verbose "건강한 어댑터 개수: $healthy_count"

    if [ "$adapter_status" != "healthy" ]; then
        log_warning "일부 어댑터가 비정상 상태입니다"

        # 비정상 어댑터 목록 출력
        local unhealthy_adapters
        unhealthy_adapters=$(get_json_array_values "$adapter_response" "unhealthy_adapters")

        if [ -n "$unhealthy_adapters" ]; then
            log_error "비정상 어댑터 목록:"
            echo "$unhealthy_adapters" | while read -r adapter; do
                [ -n "$adapter" ] && log_error "  - $adapter"
            done
        fi

        return 1
    fi

    log_success "모든 어댑터가 정상 상태입니다 ($healthy_count/$total_count)"

    # 개별 어댑터 상세 체크 (verbose 모드)
    if [ "$VERBOSE" = true ]; then
        log_info "개별 어댑터 상세 체크..."

        local adapters="maven npm apt docker pip yum apk"
        for adapter in $adapters; do
            local individual_url="$ENDPOINT/health/adapters/$adapter"
            if individual_response=$(make_request "$individual_url" "$TIMEOUT" 2>/dev/null); then
                local individual_status
                individual_status=$(get_json_value "$individual_response" "status.is_healthy")
                local initialized
                initialized=$(get_json_value "$individual_response" "status.initialized")

                if [ "$individual_status" = "true" ] && [ "$initialized" = "true" ]; then
                    log_success "  $adapter: 정상"
                else
                    log_warning "  $adapter: 비정상 (건강: $individual_status, 초기화: $initialized)"
                fi
            else
                log_verbose "  $adapter: 개별 체크 실패 또는 지원되지 않음"
            fi
        done
    fi

    return 0
}

# 메인 실행 함수
main() {
    echo -e "${BLUE}🧪 ProxyND 어댑터 헬스체크 스모크 테스트${NC}"
    echo "=================================================="

    if ! perform_health_check; then
        EXIT_CODE=1
        log_error "헬스체크 실패!"
        echo ""
        echo -e "${RED}🚨 빌드를 중단합니다. 어댑터 상태를 확인하세요.${NC}"
    else
        log_success "모든 헬스체크 통과!"
        echo ""
        echo -e "${GREEN}🚀 빌드를 계속 진행할 수 있습니다.${NC}"
    fi

    echo "=================================================="
    exit $EXIT_CODE
}

# 스크립트 실행
main "$@"
