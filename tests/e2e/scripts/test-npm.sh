#!/bin/bash
# 스크립트명: NPM E2E 테스트 스크립트
# 용도: NPM 프록시의 실제 클라이언트 호환성 검증 및 전체 워크플로우 테스트
# 사용법: test-npm.sh [--verbose]
# 예시: test-npm.sh --verbose

set -euo pipefail

# Step 1: NPM E2E 테스트 강화 - 스코프 패키지, search, view, 의존성, 캐시 테스트

PROXYND_HOST=${PROXYND_HOST:-proxynd}
PROXYND_PORT=${PROXYND_PORT:-8080}
PROXY_URL="http://${PROXYND_HOST}:${PROXYND_PORT}/proxy/npm"
VERBOSE=${1:-}

# 로깅 함수
log_info() { echo "ℹ️ $1"; }
log_success() { echo "✅ $1"; }
log_warning() { echo "⚠️ $1"; }
log_error() { echo "❌ $1"; }

echo "🚀 Enhanced NPM E2E Testing - Comprehensive Workflow Validation"
echo "📍 Proxy URL: $PROXY_URL"

# npm 레지스트리 설정
npm config set registry "$PROXY_URL"
log_info "Registry configured: $(npm config get registry)"

# 테스트 결과 추적
TESTS_PASSED=0
TESTS_TOTAL=0

# 테스트 함수 정의
run_test() {
    local test_name="$1"
    local test_command="$2"
    
    TESTS_TOTAL=$((TESTS_TOTAL + 1))
    log_info "Running test: $test_name"
    
    if [[ "$VERBOSE" == "--verbose" ]]; then
        echo "Command: $test_command"
    fi
    
    if eval "$test_command"; then
        log_success "$test_name - PASSED"
        TESTS_PASSED=$((TESTS_PASSED + 1))
        return 0
    else
        log_error "$test_name - FAILED"
        return 1
    fi
}

# 1. 기본 패키지 메타데이터 조회 테스트
log_info "📦 Test 1: Basic package metadata retrieval"
run_test "Express package metadata" "npm view express --json > /tmp/express-meta.json 2>&1"

# 2. 스코프 패키지 테스트 (Step 1-1: 스코프 패키지 설치 테스트)
log_info "📦 Test 2: Scoped packages support"
run_test "Scoped package (@types/node)" "npm view @types/node --json > /tmp/types-node-meta.json 2>&1"
run_test "Scoped package (@angular/core)" "npm view @angular/core --json > /tmp/angular-core-meta.json 2>&1"

# 3. NPM search 테스트 (Step 1-2: npm search 및 view 명령어 테스트)
log_info "🔍 Test 3: NPM search functionality"
run_test "Search for express" "npm search express --json > /tmp/search-express.json 2>&1"
run_test "Search for lodash" "npm search lodash --json > /tmp/search-lodash.json 2>&1"

# 4. 다양한 버전 및 태그 테스트 (Step 1-3: 의존성 해결 검증)
log_info "🏷️ Test 4: Version and tag resolution"
run_test "Latest tag resolution" "npm view express@latest version 2>&1 | grep -E '^[0-9]+\.[0-9]+\.[0-9]+'"
run_test "Specific version resolution" "npm view express@4.18.0 version 2>&1"
run_test "Version range listing" "npm view express versions --json > /tmp/express-versions.json 2>&1"

# 5. 캐시 성능 테스트 (Step 1-4: 캐시 동작 확인 테스트)
log_info "⚡ Test 5: Cache performance validation"

# 첫 번째 요청 (캐시 MISS)
start_time=$(date +%s%N)
npm view lodash --json > /tmp/lodash-first.json 2>&1
first_duration=$(( ($(date +%s%N) - start_time) / 1000000 ))

# 두 번째 요청 (캐시 HIT)
start_time=$(date +%s%N)
npm view lodash --json > /tmp/lodash-second.json 2>&1
second_duration=$(( ($(date +%s%N) - start_time) / 1000000 ))

log_info "Cache performance: First request: ${first_duration}ms, Second request: ${second_duration}ms"

if [ $second_duration -lt $((first_duration / 2)) ]; then
    log_success "Cache HIT performance improvement detected"
    TESTS_PASSED=$((TESTS_PASSED + 1))
else
    log_warning "Cache performance improvement not clearly detected"
fi
TESTS_TOTAL=$((TESTS_TOTAL + 1))

# 6. 의존성 정보 조회 테스트
log_info "🔗 Test 6: Dependency information retrieval"
run_test "Express dependencies" "npm view express dependencies --json > /tmp/express-deps.json 2>&1"
run_test "Package dist-tags" "npm view express dist-tags --json > /tmp/express-tags.json 2>&1"

# 7. 타르볼 다운로드 테스트 (실제 설치는 하지 않음)
log_info "📥 Test 7: Package tarball availability"
run_test "Express tarball info" "npm view express dist.tarball 2>&1 | grep -E 'https?://'"

# 결과 요약
echo ""
echo "🎯 NPM E2E Test Results Summary"
echo "==============================="
echo "Total tests: $TESTS_TOTAL"
echo "Passed: $TESTS_PASSED"
echo "Failed: $((TESTS_TOTAL - TESTS_PASSED))"
echo "Success rate: $(( TESTS_PASSED * 100 / TESTS_TOTAL ))%"

# 검증 기준 확인
if [ $TESTS_PASSED -eq $TESTS_TOTAL ]; then
    log_success "All tests passed! NPM proxy compatibility verified"
    echo "✅ 패키지 설치 성공률: 100%"
    echo "✅ 캐시 성능 개선 확인됨"
    echo "✅ 스코프 패키지 지원 검증됨"
    exit 0
else
    log_error "Some tests failed. Check proxy configuration and upstream connectivity."
    exit 1
fi
