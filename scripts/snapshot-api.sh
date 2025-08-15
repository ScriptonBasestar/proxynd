#!/bin/bash

# 스크립트명: API 상태 스냅샷 도구
# 용도: 리팩토링 전후 API 호환성 비교를 위한 기준점 생성
# 사용법: snapshot-api.sh [output_file] [--format json|text]
# 예시: snapshot-api.sh baseline.json --format json

set -euo pipefail

# 기본 설정
OUTPUT_FILE=${1:-"api-snapshot-$(date +%Y%m%d-%H%M%S).txt"}
FORMAT="text"
BASE_URL="http://localhost:8080"
TIMEOUT=10

# 명령행 인수 파싱
while [[ $# -gt 0 ]]; do
    case $1 in
        --format)
            FORMAT="$2"
            shift 2
            ;;
        --base-url)
            BASE_URL="$2"
            shift 2
            ;;
        --timeout)
            TIMEOUT="$2"
            shift 2
            ;;
        --help)
            echo "사용법: $0 [output_file] [--format json|text] [--base-url URL] [--timeout SECONDS]"
            echo "  output_file: 스냅샷 저장 파일 (기본: api-snapshot-TIMESTAMP.txt)"
            echo "  --format: 출력 형식 (json|text, 기본: text)"
            echo "  --base-url: 서버 URL (기본: http://localhost:8080)"
            echo "  --timeout: 요청 타임아웃 (기본: 10초)"
            exit 0
            ;;
        *.json)
            OUTPUT_FILE="$1"
            FORMAT="json"
            shift
            ;;
        *.txt)
            OUTPUT_FILE="$1"
            FORMAT="text"
            shift
            ;;
        *)
            if [[ ! "$1" == --* ]]; then
                OUTPUT_FILE="$1"
            fi
            shift
            ;;
    esac
done

# 색상 정의
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

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
}

# 서버 연결 확인
check_server() {
    log_info "🌐 Checking server connectivity..."

    if curl -s --max-time "$TIMEOUT" "$BASE_URL/health" > /dev/null; then
        log_success "✓ Server is reachable at $BASE_URL"
    else
        log_error "❌ Server not reachable at $BASE_URL"
        log_error "Please ensure the server is running with 'make dev-run'"
        exit 1
    fi
}

# API 엔드포인트 정의 (verify-api-endpoints.sh 기반)
declare -a ENDPOINTS=(
    "/health|GET|Health check endpoint"
    "/ready|GET|Readiness check endpoint"
    "/metrics|GET|Prometheus metrics endpoint"
    "/|GET|Dashboard root endpoint"
    "/api/health|GET|API health status"
    "/api/metrics/cache|GET|Cache metrics API"
    "/api/metrics/health|GET|Health metrics API"
    "/api/search|GET|Package search API"
    "/proxy/npm/express|GET|NPM package proxy test"
    "/proxy/maven/org/springframework/spring-core|GET|Maven package proxy test"
    "/proxy/apt/ubuntu/dists/jammy/Release|GET|APT repository proxy test"
    "/proxy/docker/library/nginx/manifests/latest|GET|Docker registry proxy test"
    "/config/reload|POST|Configuration reload endpoint"
)

# 스냅샷 데이터 수집
collect_api_snapshot() {
    local timestamp=$(date -Iseconds)
    local snapshot_data=""

    log_info "📸 Collecting API snapshot..."

    if [[ "$FORMAT" == "json" ]]; then
        snapshot_data='{"timestamp":"'$timestamp'","base_url":"'$BASE_URL'","endpoints":['
    else
        snapshot_data="ProxyND API Snapshot\n"
        snapshot_data+="Generated: $timestamp\n"
        snapshot_data+="Base URL: $BASE_URL\n"
        snapshot_data+="===========================================\n\n"
    fi

    local endpoint_count=0
    local success_count=0

    for endpoint_info in "${ENDPOINTS[@]}"; do
        IFS='|' read -r path method description <<< "$endpoint_info"

        log_info "Testing: $method $path"

        # HTTP 요청 실행
        local url="$BASE_URL$path"
        local response_code
        local response_time
        local content_type
        local content_length

        # curl로 상세 정보 수집
        local curl_output=$(curl -s -w "%{http_code}|%{time_total}|%{content_type}|%{size_download}" \
                               --max-time "$TIMEOUT" \
                               -X "$method" \
                               "$url" 2>/dev/null || echo "000|0|unknown|0")

        IFS='|' read -r response_code response_time content_type content_length <<< "$curl_output"

        # 응답 바디 샘플 (처음 200자만)
        local response_sample=$(curl -s --max-time "$TIMEOUT" -X "$method" "$url" 2>/dev/null | head -c 200 || echo "")

        endpoint_count=$((endpoint_count + 1))

        if [[ "$response_code" =~ ^[2-3][0-9][0-9]$ ]]; then
            success_count=$((success_count + 1))
            local status="SUCCESS"
        else
            local status="FAILED"
        fi

        # 형식에 따른 데이터 추가
        if [[ "$FORMAT" == "json" ]]; then
            if [[ $endpoint_count -gt 1 ]]; then
                snapshot_data+=","
            fi
            snapshot_data+='{"path":"'$path'","method":"'$method'","description":"'$description'",'
            snapshot_data+='"status_code":'$response_code',"response_time":'$response_time','
            snapshot_data+='"content_type":"'$content_type'","content_length":'$content_length','
            snapshot_data+='"status":"'$status'","sample":"'$(echo "$response_sample" | sed 's/"/\\"/g' | tr -d '\n')'"}'
        else
            snapshot_data+="[$endpoint_count] $method $path\n"
            snapshot_data+="    Description: $description\n"
            snapshot_data+="    Status Code: $response_code\n"
            snapshot_data+="    Response Time: ${response_time}s\n"
            snapshot_data+="    Content Type: $content_type\n"
            snapshot_data+="    Content Length: $content_length bytes\n"
            snapshot_data+="    Status: $status\n"
            if [[ -n "$response_sample" ]]; then
                snapshot_data+="    Sample: $(echo "$response_sample" | tr -d '\n' | cut -c1-100)...\n"
            fi
            snapshot_data+="\n"
        fi
    done

    # 요약 정보 추가
    if [[ "$FORMAT" == "json" ]]; then
        snapshot_data+='],"summary":{"total_endpoints":'$endpoint_count',"successful_endpoints":'$success_count',"success_rate":'$(echo "scale=2; $success_count * 100 / $endpoint_count" | bc)''}}'
    else
        snapshot_data+="\n===========================================\n"
        snapshot_data+="SUMMARY\n"
        snapshot_data+="===========================================\n"
        snapshot_data+="Total Endpoints: $endpoint_count\n"
        snapshot_data+="Successful: $success_count\n"
        snapshot_data+="Failed: $((endpoint_count - success_count))\n"
        snapshot_data+="Success Rate: $(echo "scale=2; $success_count * 100 / $endpoint_count" | bc)%\n"
    fi

    echo -e "$snapshot_data"
}

# 스냅샷 비교 기능
compare_snapshots() {
    local baseline_file="$1"
    local current_file="$2"

    if [[ ! -f "$baseline_file" ]]; then
        log_error "Baseline file not found: $baseline_file"
        return 1
    fi

    log_info "🔍 Comparing API snapshots..."

    # 간단한 diff 비교
    if diff -u "$baseline_file" "$current_file" > /dev/null; then
        log_success "✅ No API changes detected"
    else
        log_warning "⚠️ API changes detected:"
        diff -u "$baseline_file" "$current_file" | head -20
    fi
}

# 메인 실행
main() {
    log_info "📸 ProxyND API Snapshot Tool"
    log_info "Output file: $OUTPUT_FILE"
    log_info "Format: $FORMAT"

    # 서버 연결 확인
    check_server

    # 스냅샷 수집
    local snapshot_content=$(collect_api_snapshot)

    # 파일 저장
    echo "$snapshot_content" > "$OUTPUT_FILE"

    # 포맷 검증 (JSON인 경우)
    if [[ "$FORMAT" == "json" ]]; then
        if command -v jq &> /dev/null; then
            if jq empty "$OUTPUT_FILE" > /dev/null 2>&1; then
                log_success "✓ Valid JSON format"
            else
                log_warning "⚠️ Invalid JSON format, but file saved"
            fi
        fi
    fi

    log_success "✅ API snapshot saved to: $OUTPUT_FILE"

    # 파일 크기 정보
    local file_size=$(du -h "$OUTPUT_FILE" | cut -f1)
    log_info "📁 File size: $file_size"

    # 사용법 안내
    echo ""
    log_info "💡 Usage examples:"
    log_info "  Compare with baseline: diff baseline.txt $OUTPUT_FILE"
    log_info "  Verify API: make verify-api"
    log_info "  View JSON: cat $OUTPUT_FILE | jq ."
}

# 스크립트 실행
main
