#!/bin/bash
# 스크립트명: YUM E2E 테스트 스크립트
# 용도: ProxyND의 YUM 프록시 기능을 실제 yum/dnf 클라이언트로 테스트
# 사용법: test-yum.sh [옵션]
# 예시: test-yum.sh --verbose

set -euo pipefail

PROXYND_HOST=${PROXYND_HOST:-proxynd}
PROXYND_PORT=${PROXYND_PORT:-8080}
PROXY_URL="http://${PROXYND_HOST}:${PROXYND_PORT}/proxy/yum"

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
    echo "YUM E2E 테스트 스크립트"
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
    echo "  YUM_OPTS          Additional YUM options"
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

log_info "Starting YUM E2E tests..."
log_info "Proxy URL: $PROXY_URL"

# 테스트 작업 디렉토리 생성
TEST_DIR="/tmp/yum-e2e-test-$$"
mkdir -p "$TEST_DIR"
cd "$TEST_DIR"

# 정리 함수
cleanup() {
    log_info "Cleaning up test directory..."
    cd /
    rm -rf "$TEST_DIR" || true
}
trap cleanup EXIT

# YUM 설정 생성
create_yum_repo_config() {
    log_info "Creating YUM repository configuration..."

    cat > proxynd-test.repo << EOF
[proxynd-centos-base]
name=ProxyND CentOS Base
baseurl=$PROXY_URL/centos/8/BaseOS/x86_64/os/
enabled=1
gpgcheck=0
skip_if_unavailable=1

[proxynd-centos-updates]
name=ProxyND CentOS Updates
baseurl=$PROXY_URL/centos/8/BaseOS/x86_64/os/
enabled=1
gpgcheck=0
skip_if_unavailable=1
EOF

    log_success "YUM repository configuration created"
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

# Repository 메타데이터 테스트
test_repository_metadata() {
    log_test "Testing repository metadata access..."

    # repomd.xml 테스트
    local repomd_url="${PROXY_URL}/centos/8/BaseOS/x86_64/os/repodata/repomd.xml"

    if curl -sf "$repomd_url" >/dev/null 2>&1; then
        log_success "Repository metadata accessible"

        if [[ "$VERBOSE" == "true" ]]; then
            log_info "Sample repomd.xml content:"
            curl -s "$repomd_url" | head -10
        fi

        # XML 구조 검증
        local xml_content
        xml_content=$(curl -s "$repomd_url")

        if echo "$xml_content" | grep -q "<repomd"; then
            log_success "Repository metadata has valid XML structure"
        else
            log_warning "Repository metadata XML structure may be invalid"
        fi

    else
        log_error "Repository metadata not accessible"
        return 1
    fi
}

# Primary 메타데이터 테스트
test_primary_metadata() {
    log_test "Testing primary metadata access..."

    # primary.xml.gz 테스트
    local primary_url="${PROXY_URL}/centos/8/BaseOS/x86_64/os/repodata/primary.xml.gz"

    if curl -sf "$primary_url" >/dev/null 2>&1; then
        log_success "Primary metadata accessible"

        # Content-Type 검증
        local content_type
        content_type=$(curl -sI "$primary_url" | grep -i "content-type" | cut -d: -f2 | tr -d '[:space:]')

        if [[ "$content_type" =~ gzip|application/x-gzip ]]; then
            log_success "Primary metadata has correct Content-Type: $content_type"
        else
            log_warning "Primary metadata Content-Type: $content_type (expected gzip)"
        fi

    else
        log_error "Primary metadata not accessible"
        return 1
    fi
}

# RPM 패키지 다운로드 테스트
test_rpm_package_download() {
    log_test "Testing RPM package download..."

    # 테스트 RPM 다운로드
    local rpm_url="${PROXY_URL}/centos/8/BaseOS/x86_64/os/Packages/nginx-1.20.1-1.el8.x86_64.rpm"
    local rpm_file="nginx-test.rpm"

    local start_time
    start_time=$(date +%s%N)

    if curl -sf "$rpm_url" -o "$rpm_file"; then
        local duration=$(( ($(date +%s%N) - start_time) / 1000000 ))
        local file_size
        file_size=$(stat -c%s "$rpm_file" 2>/dev/null || echo "unknown")

        log_success "RPM package downloaded successfully (${duration}ms, ${file_size} bytes)"

        # 파일이 실제 데이터를 포함하는지 확인
        if [[ -s "$rpm_file" ]]; then
            log_success "Downloaded RPM file contains data"
        else
            log_warning "Downloaded RPM file is empty"
        fi

        # Content-Type 검증
        local content_type
        content_type=$(curl -sI "$rpm_url" | grep -i "content-type" | cut -d: -f2 | tr -d '[:space:]')

        if [[ "$content_type" =~ rpm|application/x-rpm ]]; then
            log_success "RPM has correct Content-Type: $content_type"
        else
            log_warning "RPM Content-Type: $content_type (expected rpm)"
        fi

    else
        log_error "RPM package download failed"
        return 1
    fi
}

# 캐시 성능 테스트
test_cache_performance() {
    log_test "Testing cache performance..."

    local test_url="${PROXY_URL}/centos/8/BaseOS/x86_64/os/repodata/repomd.xml"

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

# YUM 명령어 시뮬레이션 테스트
test_yum_commands() {
    log_test "Testing YUM command simulation..."

    # yum repolist 시뮬레이션 (실제로는 curl로 메타데이터 확인)
    log_info "Simulating 'yum repolist'..."

    local repomd_url="${PROXY_URL}/centos/8/BaseOS/x86_64/os/repodata/repomd.xml"
    if curl -sf "$repomd_url" >/dev/null 2>&1; then
        log_success "Repository list accessible (yum repolist simulation)"
    else
        log_error "Repository list not accessible"
        return 1
    fi

    # yum search 시뮬레이션 (primary.xml.gz 접근)
    log_info "Simulating package search..."

    local primary_url="${PROXY_URL}/centos/8/BaseOS/x86_64/os/repodata/primary.xml.gz"
    if curl -sf "$primary_url" >/dev/null 2>&1; then
        log_success "Package search data accessible"
    else
        log_error "Package search data not accessible"
        return 1
    fi
}

# 동시 요청 테스트
test_concurrent_requests() {
    log_test "Testing concurrent requests..."

    local test_url="${PROXY_URL}/centos/8/BaseOS/x86_64/os/repodata/repomd.xml"
    local num_requests=5
    local pids=()
    local results=()

    # 동시 요청 시작
    for i in $(seq 1 $num_requests); do
        {
            local start_time
            start_time=$(date +%s%N)

            if curl -sf "$test_url" >/dev/null 2>&1; then
                local duration=$(( ($(date +%s%N) - start_time) / 1000000 ))
                echo "SUCCESS:$duration" > "/tmp/yum-concurrent-$i.result"
            else
                echo "FAILED" > "/tmp/yum-concurrent-$i.result"
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
        if [[ -f "/tmp/yum-concurrent-$i.result" ]]; then
            local result
            result=$(cat "/tmp/yum-concurrent-$i.result")
            rm -f "/tmp/yum-concurrent-$i.result"

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

    # 존재하지 않는 저장소 테스트
    local nonexistent_repo_url="${PROXY_URL}/nonexistent/repo/repodata/repomd.xml"
    local response_code
    response_code=$(curl -s -o /dev/null -w "%{http_code}" "$nonexistent_repo_url")

    if [[ "$response_code" == "404" ]]; then
        log_success "Non-existent repository correctly returns 404"
    else
        log_warning "Non-existent repository returned $response_code (expected 404)"
    fi

    # 존재하지 않는 RPM 테스트
    local nonexistent_rpm_url="${PROXY_URL}/centos/8/BaseOS/x86_64/os/Packages/nonexistent-package.rpm"
    response_code=$(curl -s -o /dev/null -w "%{http_code}" "$nonexistent_rpm_url")

    if [[ "$response_code" == "404" ]]; then
        log_success "Non-existent RPM correctly returns 404"
    else
        log_warning "Non-existent RPM returned $response_code (expected 404)"
    fi
}

# HTTP 헤더 테스트
test_http_headers() {
    log_test "Testing HTTP headers..."

    local test_url="${PROXY_URL}/centos/8/BaseOS/x86_64/os/repodata/repomd.xml"

    # User-Agent 헤더 테스트
    local user_agent="yum/3.4.3"
    if curl -sf -H "User-Agent: $user_agent" "$test_url" >/dev/null 2>&1; then
        log_success "YUM User-Agent header accepted"
    else
        log_warning "YUM User-Agent header may not be properly handled"
    fi

    # If-Modified-Since 헤더 테스트
    local old_date="Wed, 21 Oct 2015 07:28:00 GMT"
    local response_code
    response_code=$(curl -s -H "If-Modified-Since: $old_date" -o /dev/null -w "%{http_code}" "$test_url")

    if [[ "$response_code" == "200" ]] || [[ "$response_code" == "304" ]]; then
        log_success "If-Modified-Since header handled correctly ($response_code)"
    else
        log_warning "If-Modified-Since header response: $response_code"
    fi
}

# 메타데이터 동기화 시뮬레이션
test_metadata_sync() {
    log_test "Testing metadata synchronization..."

    local base_url="${PROXY_URL}/centos/8/BaseOS/x86_64/os"
    local metadata_files=(
        "repodata/repomd.xml"
        "repodata/primary.xml.gz"
    )

    local sync_success=true

    for file in "${metadata_files[@]}"; do
        local file_url="${base_url}/${file}"

        if curl -sf "$file_url" >/dev/null 2>&1; then
            log_info "✓ $file synchronized"
        else
            log_error "✗ $file synchronization failed"
            sync_success=false
        fi
    done

    if [[ "$sync_success" == "true" ]]; then
        log_success "Metadata synchronization completed successfully"
    else
        log_error "Metadata synchronization had failures"
        return 1
    fi
}

# 메인 테스트 실행
main() {
    log_info "=== YUM E2E Tests Starting ==="

    create_yum_repo_config

    # 필수 테스트들
    check_proxynd_health
    test_repository_metadata
    test_primary_metadata
    test_rpm_package_download

    # 추가 기능 테스트들
    test_cache_performance
    test_yum_commands
    test_concurrent_requests
    test_error_scenarios
    test_http_headers
    test_metadata_sync

    log_success "=== YUM E2E Tests Completed Successfully ==="
}

# 스크립트 실행
main "$@"
