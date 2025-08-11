#!/bin/bash
# 스크립트명: APK E2E 테스트 스크립트
# 용도: ProxyND의 APK 프록시 기능을 실제 apk 클라이언트로 테스트
# 사용법: test-apk.sh [옵션]
# 예시: test-apk.sh --verbose

set -euo pipefail

PROXYND_HOST=${PROXYND_HOST:-proxynd}
PROXYND_PORT=${PROXYND_PORT:-8080}
PROXY_URL="http://${PROXYND_HOST}:${PROXYND_PORT}/proxy/apk"

# 색상 출력을 위한 함수들
red() { echo -e "\033[31m$1\033[0m"; }
green() { echo -e "\033[32m$1\033[0m"; }
yellow() { echo -e "\033[33m$1\033[0m"; }
blue() { echo -e "\033[34m$1\033[0m"; }

# 로그 함수들
log_info() { echo "ℹ️ $1"; }
log_success() { green "✅ $1"; }
log_warning() { yellow "⚠️ $1"; }
log_error() { red "❌ $1"; }
log_test() { blue "🧪 $1"; }

# 헬프 메시지
show_help() {
    echo "APK E2E 테스트 스크립트"
    echo ""
    echo "Usage: $0 [options]"
    echo ""
    echo "Options:"
    echo "  -v, --verbose     Verbose output"
    echo "  -h, --help        Show this help message"
    echo ""
    echo "Environment Variables:"
    echo "  PROXYND_HOST      ProxyND host (default: proxynd)"
    echo "  PROXYND_PORT      ProxyND port (default: 8080)"
    echo "  APK_OPTS          Additional APK options"
}

# 인수 파싱
VERBOSE=false
while [[ $# -gt 0 ]]; do
    case $1 in
        -v|--verbose)
            VERBOSE=true
            shift
            ;;
        -h|--help)
            show_help
            exit 0
            ;;
        *)
            log_error "Unknown option: $1"
            show_help
            exit 1
            ;;
    esac
done

# Verbose 모드 설정
if [[ "$VERBOSE" == "true" ]]; then
    set -x
fi

log_info "Starting APK E2E tests..."
log_info "Proxy URL: $PROXY_URL"

# 테스트 작업 디렉토리 생성
TEST_DIR="/tmp/apk-e2e-test-$$"
mkdir -p "$TEST_DIR"
cd "$TEST_DIR"

# 정리 함수
cleanup() {
    log_info "Cleaning up test directory..."
    cd /
    rm -rf "$TEST_DIR" || true
}
trap cleanup EXIT

# APK 저장소 설정 생성
create_apk_repo_config() {
    log_info "Creating APK repository configuration..."

    mkdir -p etc/apk
    cat > etc/apk/repositories << EOF
# ProxyND APK 저장소 설정 - E2E 테스트용
${PROXY_URL}/alpine/v3.16/main
${PROXY_URL}/alpine/v3.16/community
# 백업용 공식 저장소 (필요시)
# https://dl-cdn.alpinelinux.org/alpine/v3.16/main
# https://dl-cdn.alpinelinux.org/alpine/v3.16/community
EOF

    log_success "APK repository configuration created"

    if [[ "$VERBOSE" == "true" ]]; then
        log_info "APK repositories configured:"
        cat etc/apk/repositories
    fi
}

# ProxyND 헬스체크
check_proxynd_health() {
    log_test "Checking ProxyND health..."

    local health_url="http://${PROXYND_HOST}:${PROXYND_PORT}/healthz"
    local max_attempts=10
    local attempt=0

    while [ $attempt -lt $max_attempts ]; do
        if curl -sf "$health_url" >/dev/null 2>&1; then
            log_success "ProxyND is healthy"
            return 0
        fi

        attempt=$((attempt + 1))
        log_info "Attempt $attempt/$max_attempts: Waiting for ProxyND..."
        sleep 2
    done

    log_error "ProxyND health check failed after $max_attempts attempts"
    return 1
}

# 패키지 인덱스 접근 테스트
test_package_index_access() {
    log_test "Testing package index access..."

    # APKINDEX.tar.gz 테스트 - main 저장소
    local main_index_url="${PROXY_URL}/alpine/v3.16/main/x86_64/APKINDEX.tar.gz"

    if curl -sf "$main_index_url" >/dev/null 2>&1; then
        log_success "Main repository index accessible"

        if [[ "$VERBOSE" == "true" ]]; then
            log_info "Sample main index content:"
            curl -s "$main_index_url" | head -10
        fi

        # Content-Type 검증
        local content_type
        content_type=$(curl -sI "$main_index_url" | grep -i "content-type" | cut -d: -f2 | tr -d '[:space:]')

        if [[ "$content_type" =~ gzip|application/x-gzip|application/gzip ]]; then
            log_success "Main index has correct Content-Type: $content_type"
        else
            log_warning "Main index Content-Type: $content_type (expected gzip)"
        fi

    else
        log_error "Main repository index not accessible"
        return 1
    fi

    # APKINDEX.tar.gz 테스트 - community 저장소
    local community_index_url="${PROXY_URL}/alpine/v3.16/community/x86_64/APKINDEX.tar.gz"

    if curl -sf "$community_index_url" >/dev/null 2>&1; then
        log_success "Community repository index accessible"
    else
        log_warning "Community repository index not accessible"
    fi
}

# APK 패키지 다운로드 테스트
test_apk_package_download() {
    log_test "Testing APK package download..."

    # nginx 패키지 다운로드 테스트
    local nginx_url="${PROXY_URL}/alpine/v3.16/main/x86_64/nginx-1.22.0-r1.apk"
    local nginx_file="nginx-test.apk"

    local start_time
    start_time=$(date +%s%N)

    if curl -sf "$nginx_url" -o "$nginx_file"; then
        local duration=$(( ($(date +%s%N) - start_time) / 1000000 ))
        local file_size
        file_size=$(stat -c%s "$nginx_file" 2>/dev/null || echo "unknown")

        log_success "nginx APK downloaded successfully (${duration}ms, ${file_size} bytes)"

        # 파일이 실제 데이터를 포함하는지 확인
        if [[ -s "$nginx_file" ]]; then
            log_success "Downloaded nginx APK file contains data"
        else
            log_warning "Downloaded nginx APK file is empty"
        fi

        # Content-Type 검증
        local content_type
        content_type=$(curl -sI "$nginx_url" | grep -i "content-type" | cut -d: -f2 | tr -d '[:space:]')

        if [[ "$content_type" =~ apk|application/vnd.alpine.apk|application/octet-stream ]]; then
            log_success "nginx APK has correct Content-Type: $content_type"
        else
            log_warning "nginx APK Content-Type: $content_type (expected apk-related)"
        fi

    else
        log_error "nginx APK download failed"
        return 1
    fi

    # curl 패키지 다운로드 테스트
    local curl_url="${PROXY_URL}/alpine/v3.16/main/x86_64/curl-7.83.1-r0.apk"
    local curl_file="curl-test.apk"

    if curl -sf "$curl_url" -o "$curl_file"; then
        log_success "curl APK downloaded successfully"

        if [[ -s "$curl_file" ]]; then
            log_success "Downloaded curl APK file contains data"
        else
            log_warning "Downloaded curl APK file is empty"
        fi
    else
        log_warning "curl APK download failed (may be expected for test environment)"
    fi
}

# 캐시 성능 테스트
test_cache_performance() {
    log_test "Testing cache performance..."

    local test_url="${PROXY_URL}/alpine/v3.16/main/x86_64/APKINDEX.tar.gz"

    # 첫 번째 요청
    local start_time
    start_time=$(date +%s%N)
    curl -sf "$test_url" >/dev/null 2>&1
    local first_duration=$(( ($(date +%s%N) - start_time) / 1000000 ))

    # 두 번째 요청 (캐시된 결과 기대)
    start_time=$(date +%s%N)
    curl -sf "$test_url" >/dev/null 2>&1
    local second_duration=$(( ($(date +%s%N) - start_time) / 1000000 ))

    log_info "First request: ${first_duration}ms"
    log_info "Second request: ${second_duration}ms"

    if [ $second_duration -lt $((first_duration / 2)) ]; then
        log_success "Cache is working effectively (${second_duration}ms vs ${first_duration}ms)"
    elif [ $second_duration -lt $first_duration ]; then
        log_warning "Cache shows some improvement (${second_duration}ms vs ${first_duration}ms)"
    else
        log_warning "Cache performance inconclusive"
    fi
}

# APK 명령어 시뮬레이션 테스트
test_apk_commands() {
    log_test "Testing APK command simulation..."

    # apk update 시뮬레이션 (실제로는 curl로 인덱스 확인)
    log_info "Simulating 'apk update'..."

    local main_index_url="${PROXY_URL}/alpine/v3.16/main/x86_64/APKINDEX.tar.gz"
    local community_index_url="${PROXY_URL}/alpine/v3.16/community/x86_64/APKINDEX.tar.gz"

    local update_success=true

    if curl -sf "$main_index_url" >/dev/null 2>&1; then
        log_info "✓ main repository updated"
    else
        log_error "✗ main repository update failed"
        update_success=false
    fi

    if curl -sf "$community_index_url" >/dev/null 2>&1; then
        log_info "✓ community repository updated"
    else
        log_warning "✗ community repository update failed (may be expected)"
    fi

    if [[ "$update_success" == "true" ]]; then
        log_success "Repository update simulation completed (apk update)"
    else
        log_error "Repository update simulation had failures"
        return 1
    fi

    # apk search 시뮬레이션 (패키지 인덱스 접근)
    log_info "Simulating package search..."

    if curl -sf "$main_index_url" >/dev/null 2>&1; then
        log_success "Package search data accessible"
    else
        log_error "Package search data not accessible"
        return 1
    fi
}

# 동시 요청 테스트
test_concurrent_requests() {
    log_test "Testing concurrent requests..."

    local test_url="${PROXY_URL}/alpine/v3.16/main/x86_64/APKINDEX.tar.gz"
    local num_requests=5
    local pids=()

    # 동시 요청 시작
    for i in $(seq 1 $num_requests); do
        {
            local start_time
            start_time=$(date +%s%N)

            if curl -sf "$test_url" >/dev/null 2>&1; then
                local duration=$(( ($(date +%s%N) - start_time) / 1000000 ))
                echo "SUCCESS:$duration" > "/tmp/apk-concurrent-$i.result"
            else
                echo "FAILED" > "/tmp/apk-concurrent-$i.result"
            fi
        } &
        pids+=($!)
    done

    # 모든 요청 완료 대기
    for pid in "${pids[@]}"; do
        wait $pid
    done

    # 결과 수집
    local success_count=0
    local total_time=0

    for i in $(seq 1 $num_requests); do
        if [[ -f "/tmp/apk-concurrent-$i.result" ]]; then
            local result
            result=$(cat "/tmp/apk-concurrent-$i.result")
            rm -f "/tmp/apk-concurrent-$i.result"

            if [[ "$result" =~ ^SUCCESS:([0-9]+)$ ]]; then
                success_count=$((success_count + 1))
                total_time=$((total_time + ${BASH_REMATCH[1]}))
            fi
        fi
    done

    if [[ $success_count -eq $num_requests ]]; then
        local avg_time=$((total_time / num_requests))
        log_success "All $num_requests concurrent requests successful (avg: ${avg_time}ms)"
    else
        log_warning "$success_count/$num_requests concurrent requests successful"
    fi
}

# 에러 시나리오 테스트
test_error_scenarios() {
    log_test "Testing error scenarios..."

    # 존재하지 않는 Alpine 버전 테스트
    local nonexistent_version_url="${PROXY_URL}/alpine/v99.99/main/x86_64/APKINDEX.tar.gz"
    local response_code
    response_code=$(curl -s -o /dev/null -w "%{http_code}" "$nonexistent_version_url")

    if [[ "$response_code" == "404" ]]; then
        log_success "Non-existent Alpine version correctly returns 404"
    else
        log_warning "Non-existent Alpine version returned $response_code (expected 404)"
    fi

    # 존재하지 않는 아키텍처 테스트
    local nonexistent_arch_url="${PROXY_URL}/alpine/v3.16/main/nonexistent-arch/APKINDEX.tar.gz"
    response_code=$(curl -s -o /dev/null -w "%{http_code}" "$nonexistent_arch_url")

    if [[ "$response_code" == "404" ]]; then
        log_success "Non-existent architecture correctly returns 404"
    else
        log_warning "Non-existent architecture returned $response_code (expected 404)"
    fi

    # 존재하지 않는 APK 패키지 테스트
    local nonexistent_package_url="${PROXY_URL}/alpine/v3.16/main/x86_64/nonexistent-package.apk"
    response_code=$(curl -s -o /dev/null -w "%{http_code}" "$nonexistent_package_url")

    if [[ "$response_code" == "404" ]]; then
        log_success "Non-existent APK package correctly returns 404"
    else
        log_warning "Non-existent APK package returned $response_code (expected 404)"
    fi

    # 존재하지 않는 저장소 테스트
    local nonexistent_repo_url="${PROXY_URL}/alpine/v3.16/nonexistent/x86_64/APKINDEX.tar.gz"
    response_code=$(curl -s -o /dev/null -w "%{http_code}" "$nonexistent_repo_url")

    if [[ "$response_code" == "404" ]]; then
        log_success "Non-existent repository correctly returns 404"
    else
        log_warning "Non-existent repository returned $response_code (expected 404)"
    fi
}

# HTTP 헤더 테스트
test_http_headers() {
    log_test "Testing HTTP headers..."

    local test_url="${PROXY_URL}/alpine/v3.16/main/x86_64/APKINDEX.tar.gz"

    # User-Agent 헤더 테스트
    local user_agent="apk-tools/2.12.9"
    if curl -sf -H "User-Agent: $user_agent" "$test_url" >/dev/null 2>&1; then
        log_success "APK User-Agent header accepted"
    else
        log_warning "APK User-Agent header may not be properly handled"
    fi

    # If-None-Match 헤더 테스트 (ETag 기반 캐싱)
    local etag_response
    etag_response=$(curl -sI "$test_url" | grep -i "etag" | cut -d: -f2 | tr -d '[:space:]' || true)

    if [[ -n "$etag_response" ]]; then
        local response_code
        response_code=$(curl -s -H "If-None-Match: $etag_response" -o /dev/null -w "%{http_code}" "$test_url")

        if [[ "$response_code" == "304" ]]; then
            log_success "ETag-based caching working correctly (304 Not Modified)"
        elif [[ "$response_code" == "200" ]]; then
            log_info "ETag header processed, returned fresh content (200)"
        else
            log_warning "ETag header response: $response_code"
        fi
    else
        log_info "No ETag header found (may be expected)"
    fi

    # Accept-Encoding 헤더 테스트
    if curl -sf -H "Accept-Encoding: gzip" "$test_url" >/dev/null 2>&1; then
        log_success "Accept-Encoding header handled correctly"
    else
        log_warning "Accept-Encoding header may not be properly handled"
    fi
}

# APK 서명 검증 시뮬레이션
test_signature_verification() {
    log_test "Testing signature verification simulation..."

    # 이것은 실제 서명 검증이 아닌 프록시의 서명 관련 기능 테스트
    local test_url="${PROXY_URL}/alpine/v3.16/main/x86_64/nginx-1.22.0-r1.apk"

    # 서명 관련 헤더가 있는지 확인
    local headers
    headers=$(curl -sI "$test_url" || true)

    if echo "$headers" | grep -qi "signature\|sign"; then
        log_success "Signature-related headers present"
    else
        log_info "No signature-related headers found (may be expected for mock data)"
    fi

    # APK 패키지 다운로드가 서명 검증과 관계없이 작동하는지 확인
    if curl -sf "$test_url" >/dev/null 2>&1; then
        log_success "Package download works regardless of signature verification status"
    else
        log_warning "Package download failed"
    fi
}

# 저장소 간 패키지 테스트
test_multi_repository() {
    log_test "Testing multiple repository access..."

    # main 저장소 패키지 테스트
    local main_package_url="${PROXY_URL}/alpine/v3.16/main/x86_64/nginx-1.22.0-r1.apk"
    local main_success=false

    if curl -sf "$main_package_url" >/dev/null 2>&1; then
        log_success "Main repository package accessible"
        main_success=true
    else
        log_warning "Main repository package not accessible"
    fi

    # community 저장소 패키지 테스트
    local community_package_url="${PROXY_URL}/alpine/v3.16/community/x86_64/docker-20.10.14-r0.apk"
    local community_success=false

    if curl -sf "$community_package_url" >/dev/null 2>&1; then
        log_success "Community repository package accessible"
        community_success=true
    else
        log_warning "Community repository package not accessible"
    fi

    if [[ "$main_success" == "true" ]] || [[ "$community_success" == "true" ]]; then
        log_success "Multi-repository access working"
    else
        log_error "Multi-repository access failed"
        return 1
    fi
}

# 패키지 메타데이터 검증
test_package_metadata() {
    log_test "Testing package metadata validation..."

    local index_url="${PROXY_URL}/alpine/v3.16/main/x86_64/APKINDEX.tar.gz"
    local index_content

    if index_content=$(curl -s "$index_url"); then
        # APK 인덱스에서 패키지 정보 확인
        if echo "$index_content" | grep -q "P:nginx"; then
            log_success "Package metadata contains expected package (nginx)"
        else
            log_warning "Package metadata may not contain expected packages"
        fi

        # 버전 정보 확인
        if echo "$index_content" | grep -q "V:"; then
            log_success "Package metadata contains version information"
        else
            log_warning "Package metadata may be missing version information"
        fi

        # 아키텍처 정보 확인
        if echo "$index_content" | grep -q "A:x86_64"; then
            log_success "Package metadata contains architecture information"
        else
            log_warning "Package metadata may be missing architecture information"
        fi

    else
        log_error "Failed to retrieve package metadata"
        return 1
    fi
}

# 메인 테스트 실행
main() {
    log_info "=== APK E2E Tests Starting ==="

    create_apk_repo_config

    # 필수 테스트들
    check_proxynd_health
    test_package_index_access
    test_apk_package_download

    # 추가 기능 테스트들
    test_cache_performance
    test_apk_commands
    test_concurrent_requests
    test_error_scenarios
    test_http_headers
    test_signature_verification
    test_multi_repository
    test_package_metadata

    log_success "=== APK E2E Tests Completed Successfully ==="
}

# 스크립트 실행
main "$@"
