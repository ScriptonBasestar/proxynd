#!/bin/bash

# 스크립트명: API 엔드포인트 검증 도구
# 용도: ProxyND 서버의 CLI 필수 API 엔드포인트 연결성 및 응답 검증
# 사용법: verify-api-endpoints.sh [BASE_URL] [--json] [--verbose]
# 예시: verify-api-endpoints.sh http://localhost:8080 --json

set -euo pipefail

# 설정
BASE_URL=${1:-"http://localhost:8080"}
OUTPUT_FORMAT="text"
VERBOSE=false
TIMEOUT=10

# 명령행 인수 파싱
while [[ $# -gt 0 ]]; do
    case $1 in
        --json)
            OUTPUT_FORMAT="json"
            shift
            ;;
        --verbose)
            VERBOSE=true
            shift
            ;;
        --timeout)
            TIMEOUT="$2"
            shift 2
            ;;
        --help)
            echo "사용법: $0 [BASE_URL] [--json] [--verbose] [--timeout SECONDS]"
            echo "  BASE_URL: 서버 URL (기본: http://localhost:8080)"
            echo "  --json: JSON 형식으로 출력"
            echo "  --verbose: 상세 출력"
            echo "  --timeout: 요청 타임아웃 (기본: 10초)"
            exit 0
            ;;
        http*://*/*)
            BASE_URL="$1"
            shift
            ;;
        *)
            echo "알 수 없는 옵션: $1" >&2
            exit 1
            ;;
    esac
done

# 검증할 엔드포인트 목록
declare -a ENDPOINTS=(
    # 캐시 관리 API
    "GET /api/cache/list"
    "GET /api/cache/size"
    "GET /api/cache/stats"
    "GET /api/cache/ttl"

    # 설정 관리 API
    "GET /api/config/validate"
    "GET /api/config/show"
    "GET /api/config/files"

    # 사용자 관리 API
    "GET /api/user/list"

    # 프록시 테스트 API
    "GET /api/test/types"
    "POST /api/test/all"

    # 서버 상태 API
    "GET /api/status"
    "GET /api/status/health"
    "GET /api/status/metrics"
    "GET /api/status/dependencies"
    "GET /api/status/stats"

    # 기존 엔드포인트
    "GET /healthz"
    "GET /metrics"
)

# 색상 정의
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 로그 함수
log_info() {
    if [[ "$OUTPUT_FORMAT" == "text" ]]; then
        echo -e "${BLUE}[INFO]${NC} $1"
    fi
}

log_success() {
    if [[ "$OUTPUT_FORMAT" == "text" ]]; then
        echo -e "${GREEN}[SUCCESS]${NC} $1"
    fi
}

log_warning() {
    if [[ "$OUTPUT_FORMAT" == "text" ]]; then
        echo -e "${YELLOW}[WARNING]${NC} $1"
    fi
}

log_error() {
    if [[ "$OUTPUT_FORMAT" == "text" ]]; then
        echo -e "${RED}[ERROR]${NC} $1"
    fi
}

# JSON 결과 저장 배열
declare -a JSON_RESULTS=()

# 개별 엔드포인트 테스트 함수
test_endpoint() {
    local method="$1"
    local path="$2"
    local url="${BASE_URL}${path}"

    if [[ "$VERBOSE" == "true" && "$OUTPUT_FORMAT" == "text" ]]; then
        echo -n "Testing $method $path ... "
    fi

    # 요청 수행
    local start_time=$(date +%s%N)
    local response
    local http_code
    local success=false
    local error_message=""

    if [[ "$method" == "POST" ]]; then
        # POST 요청의 경우 빈 JSON 바디 전송
        response=$(curl -s -w "%{http_code}" -X "$method" \
                  -H "Content-Type: application/json" \
                  -d '{}' \
                  --connect-timeout "$TIMEOUT" \
                  --max-time "$TIMEOUT" \
                  "$url" 2>/dev/null || echo "000")
    else
        response=$(curl -s -w "%{http_code}" -X "$method" \
                  --connect-timeout "$TIMEOUT" \
                  --max-time "$TIMEOUT" \
                  "$url" 2>/dev/null || echo "000")
    fi

    local end_time=$(date +%s%N)
    local duration=$(((end_time - start_time) / 1000000)) # 밀리초로 변환

    # HTTP 코드 추출
    if [[ ${#response} -ge 3 ]]; then
        http_code="${response: -3}"
        response_body="${response%???}"
    else
        http_code="000"
        response_body=""
    fi

    # 응답 분석
    case "$http_code" in
        200|201)
            success=true
            status="✅ OK"
            ;;
        404)
            success=false
            status="❌ NOT FOUND"
            error_message="엔드포인트가 구현되지 않음"
            ;;
        405)
            success=false
            status="⚠️  METHOD NOT ALLOWED"
            error_message="HTTP 메서드가 지원되지 않음"
            ;;
        500|502|503)
            success=false
            status="❌ SERVER ERROR"
            error_message="서버 내부 오류"
            ;;
        000)
            success=false
            status="❌ CONNECTION FAILED"
            error_message="서버에 연결할 수 없음"
            ;;
        *)
            success=false
            status="⚠️  HTTP $http_code"
            error_message="예상치 못한 응답 코드"
            ;;
    esac

    # 결과 출력
    if [[ "$OUTPUT_FORMAT" == "text" ]]; then
        if [[ "$VERBOSE" == "true" ]]; then
            echo "$status (${duration}ms)"
        else
            printf "%-30s %s\n" "$method $path" "$status"
        fi

        # 상세 정보 출력 (verbose 모드)
        if [[ "$VERBOSE" == "true" && ! "$success" == "true" ]]; then
            echo "  └─ Error: $error_message"
            if [[ -n "$response_body" && ${#response_body} -lt 200 ]]; then
                echo "  └─ Response: $response_body"
            fi
        fi
    fi

    # JSON 결과 저장
    local json_result=$(cat <<EOF
{
    "method": "$method",
    "path": "$path",
    "url": "$url",
    "success": $success,
    "http_code": $http_code,
    "duration_ms": $duration,
    "status": "$status",
    "error": "$error_message"
}
EOF
)
    JSON_RESULTS+=("$json_result")

    # 성공 여부 반환
    if [[ "$success" == "true" ]]; then
        return 0
    else
        return 1
    fi
}

# 메인 실행 함수
main() {
    local total=0
    local passed=0
    local failed=0
    local warnings=0

    if [[ "$OUTPUT_FORMAT" == "text" ]]; then
        log_info "🔍 ProxyND API 엔드포인트 검증 시작"
        log_info "서버 URL: $BASE_URL"
        log_info "타임아웃: ${TIMEOUT}초"
        echo ""
    fi

    # 서버 연결성 사전 체크
    if ! curl -s --connect-timeout 5 "$BASE_URL" >/dev/null 2>&1; then
        if [[ "$OUTPUT_FORMAT" == "text" ]]; then
            log_error "서버에 연결할 수 없습니다: $BASE_URL"
            log_error "서버가 실행 중인지 확인해주세요."
        else
            echo '{"error": "서버 연결 실패", "base_url": "'$BASE_URL'"}'
        fi
        exit 1
    fi

    # 각 엔드포인트 테스트
    for endpoint in "${ENDPOINTS[@]}"; do
        IFS=' ' read -r method path <<< "$endpoint"
        total=$((total + 1))

        if test_endpoint "$method" "$path"; then
            passed=$((passed + 1))
        else
            if [[ "$http_code" == "404" || "$http_code" == "405" ]]; then
                failed=$((failed + 1))
            else
                warnings=$((warnings + 1))
            fi
        fi
    done

    # 결과 출력
    if [[ "$OUTPUT_FORMAT" == "json" ]]; then
        # JSON 형식 출력
        echo "{"
        echo "  \"summary\": {"
        echo "    \"total\": $total,"
        echo "    \"passed\": $passed,"
        echo "    \"failed\": $failed,"
        echo "    \"warnings\": $warnings,"
        echo "    \"success_rate\": $(awk "BEGIN {printf \"%.1f\", $passed/$total*100}")"
        echo "  },"
        echo "  \"base_url\": \"$BASE_URL\","
        echo "  \"timestamp\": \"$(date -u +%Y-%m-%dT%H:%M:%SZ)\","
        echo "  \"results\": ["

        for i in "${!JSON_RESULTS[@]}"; do
            echo "    ${JSON_RESULTS[$i]}"
            if [[ $i -lt $((${#JSON_RESULTS[@]} - 1)) ]]; then
                echo ","
            fi
        done

        echo "  ]"
        echo "}"
    else
        # 텍스트 형식 요약
        echo ""
        log_info "📊 테스트 결과 요약"
        echo "총 엔드포인트: $total"
        echo "성공: $passed"
        echo "실패: $failed"
        echo "경고: $warnings"
        echo "성공률: $(awk "BEGIN {printf \"%.1f%%\", $passed/$total*100}")"

        # 권장사항
        echo ""
        if [[ $failed -gt 0 ]]; then
            log_warning "일부 엔드포인트가 구현되지 않았습니다."
            log_info "프로젝트의 CLAUDE.md를 참조하여 누락된 API를 구현해주세요."
        fi

        if [[ $warnings -gt 0 ]]; then
            log_warning "일부 엔드포인트에서 경고가 발생했습니다."
            log_info "서버 로그를 확인하여 문제를 진단해주세요."
        fi

        if [[ $passed -eq $total ]]; then
            log_success "🎉 모든 API 엔드포인트가 정상적으로 작동합니다!"
        fi
    fi

    # 종료 코드 설정
    if [[ $failed -gt 0 ]]; then
        exit 1
    elif [[ $warnings -gt 0 ]]; then
        exit 2
    else
        exit 0
    fi
}

# 스크립트 실행
main "$@"
