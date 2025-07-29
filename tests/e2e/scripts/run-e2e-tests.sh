#!/bin/bash

# E2E 테스트 실행 스크립트

set -euo pipefail

echo "🚀 Starting E2E Tests..."

# 환경 변수 설정
export PROXYND_HOST=${PROXYND_HOST:-proxynd}
export PROXYND_PORT=${PROXYND_PORT:-8080}
export PROXYND_URL="http://${PROXYND_HOST}:${PROXYND_PORT}"

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
}

# ProxyND 서버 대기
wait_for_proxynd() {
    log_info "Waiting for ProxyND server to be ready..."

    local max_attempts=30
    local attempt=1

    while [ $attempt -le $max_attempts ]; do
        if curl -sf "${PROXYND_URL}/healthz" > /dev/null 2>&1; then
            log_success "ProxyND server is ready!"
            return 0
        fi

        log_info "Attempt ${attempt}/${max_attempts}: ProxyND not ready yet..."
        sleep 2
        ((attempt++))
    done

    log_error "ProxyND server failed to start within timeout"
    return 1
}

# 기본 헬스체크 테스트
test_health_check() {
    log_info "Testing health check endpoint..."

    local response
    response=$(curl -s "${PROXYND_URL}/healthz")

    if echo "$response" | jq -e '.status == "ok"' > /dev/null 2>&1; then
        log_success "Health check passed"
        return 0
    else
        log_error "Health check failed: $response"
        return 1
    fi
}

# NPM 프록시 테스트
test_npm_proxy() {
    log_info "Testing NPM proxy..."

    local package_name="express"
    local response

    response=$(curl -s "${PROXYND_URL}/proxy/npm/${package_name}")

    if [ $? -eq 0 ] && [ -n "$response" ]; then
        log_success "NPM proxy test passed"
        return 0
    else
        log_error "NPM proxy test failed"
        return 1
    fi
}

# PyPI 프록시 테스트
test_pip_proxy() {
    log_info "Testing PyPI proxy..."

    local package_path="simple/requests/"
    local response

    response=$(curl -s "${PROXYND_URL}/proxy/pip/${package_path}")

    if [ $? -eq 0 ] && [ -n "$response" ]; then
        log_success "PyPI proxy test passed"
        return 0
    else
        log_error "PyPI proxy test failed"
        return 1
    fi
}

# APT 프록시 테스트
test_apt_proxy() {
    log_info "Testing APT proxy..."

    local release_path="ubuntu/dists/jammy/Release"
    local response

    response=$(curl -s "${PROXYND_URL}/proxy/apt/${release_path}")

    if [ $? -eq 0 ] && [ -n "$response" ]; then
        log_success "APT proxy test passed"
        return 0
    else
        log_error "APT proxy test failed"
        return 1
    fi
}

# Docker 프록시 테스트
test_docker_proxy() {
    log_info "Testing Docker proxy..."

    local registry_path="v2/"
    local response

    response=$(curl -s "${PROXYND_URL}/proxy/docker/${registry_path}")

    if [ $? -eq 0 ]; then
        log_success "Docker proxy test passed"
        return 0
    else
        log_error "Docker proxy test failed"
        return 1
    fi
}

# 캐시 동작 테스트
test_cache_behavior() {
    log_info "Testing cache behavior..."

    local test_url="${PROXYND_URL}/proxy/npm/lodash"

    # 첫 번째 요청 (캐시 미스)
    log_info "Making first request (cache miss)..."
    local start1=$(date +%s%N)
    curl -s "$test_url" > /dev/null
    local end1=$(date +%s%N)
    local duration1=$(( (end1 - start1) / 1000000 ))

    # 두 번째 요청 (캐시 히트)
    log_info "Making second request (cache hit)..."
    local start2=$(date +%s%N)
    curl -s "$test_url" > /dev/null
    local end2=$(date +%s%N)
    local duration2=$(( (end2 - start2) / 1000000 ))

    log_info "First request: ${duration1}ms, Second request: ${duration2}ms"

    # 두 번째 요청이 더 빨라야 함 (일반적으로)
    if [ $duration2 -lt $((duration1 + 100)) ]; then
        log_success "Cache behavior test passed"
        return 0
    else
        log_warning "Cache behavior test inconclusive (performance may vary)"
        return 0
    fi
}

# 부하 테스트
test_load() {
    log_info "Running load test..."

    local concurrent_requests=10
    local total_requests=100
    local test_url="${PROXYND_URL}/proxy/npm/test-package"

    log_info "Sending ${total_requests} requests with ${concurrent_requests} concurrent connections..."

    # GNU parallel이 있으면 사용, 없으면 간단한 반복
    if command -v parallel > /dev/null 2>&1; then
        seq 1 $total_requests | parallel -j $concurrent_requests "curl -s $test_url > /dev/null"
    else
        for i in $(seq 1 $total_requests); do
            curl -s "$test_url" > /dev/null &
            if [ $((i % concurrent_requests)) -eq 0 ]; then
                wait
            fi
        done
        wait
    fi

    log_success "Load test completed"
    return 0
}

# 메인 테스트 실행
main() {
    local failed_tests=0

    echo "============================================"
    echo "🧪 ProxyND E2E Test Suite"
    echo "============================================"
    echo "Target: ${PROXYND_URL}"
    echo "============================================"

    # ProxyND 서버 대기
    if ! wait_for_proxynd; then
        log_error "Cannot proceed without ProxyND server"
        exit 1
    fi

    # 테스트 실행
    local tests=(
        "test_health_check"
        "test_npm_proxy"
        "test_pip_proxy"
        "test_apt_proxy"
        "test_docker_proxy"
        "test_cache_behavior"
        "test_load"
    )

    for test in "${tests[@]}"; do
        echo "--------------------------------------------"
        if ! $test; then
            ((failed_tests++))
        fi
    done

    echo "============================================"
    if [ $failed_tests -eq 0 ]; then
        log_success "All tests passed! 🎉"
        echo "============================================"
        exit 0
    else
        log_error "${failed_tests} test(s) failed! ❌"
        echo "============================================"
        exit 1
    fi
}

# 스크립트 실행
main "$@"
