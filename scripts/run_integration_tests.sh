#!/bin/bash

# 통합 테스트 실행 스크립트

set -e

echo "Running integration tests for ProxyND..."

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 테스트 환경 설정
export TEST_ENV=integration
export CONFIG_DIR=$(pwd)/tests/integration
export STORAGE_DIR=/tmp/proxynd-integration-test

# 캐시 디렉토리 준비
echo -e "${YELLOW}Preparing test environment...${NC}"
rm -rf $STORAGE_DIR
mkdir -p $STORAGE_DIR

# 통합 테스트 실행 함수
run_integration_test() {
    local test_name=$1
    local test_path=$2

    echo -e "\n${GREEN}Running integration test: ${test_name}${NC}"

    if go test -v -timeout 30s -run "$test_name" "$test_path" -count=1; then
        echo -e "${GREEN}✓ ${test_name} passed${NC}"
        return 0
    else
        echo -e "${RED}✗ ${test_name} failed${NC}"
        return 1
    fi
}

# 개별 통합 테스트 실행
echo -e "\n${YELLOW}Running individual proxy flow tests...${NC}"

tests=(
    "TestAPTProxyFlow:./tests/integration"
    "TestMavenProxyFlow:./tests/integration"
    "TestNPMProxyFlow:./tests/integration"
    "TestConcurrentRequests:./tests/integration"
    "TestContextCancellation:./tests/integration"
    "TestLargeFileHandling:./tests/integration"
    "TestAuthenticationFlow:./tests/integration"
    "TestErrorHandling:./tests/integration"
    "TestCacheInvalidation:./tests/integration"
    "TestMetricsCollection:./tests/integration"
    "TestFiberIntegration:./tests/integration"
)

failed_tests=0
passed_tests=0

for test in "${tests[@]}"; do
    IFS=':' read -r test_name test_path <<< "$test"
    if run_integration_test "$test_name" "$test_path"; then
        ((passed_tests++))
    else
        ((failed_tests++))
    fi
done

# 기존 통합 테스트도 실행
echo -e "\n${YELLOW}Running existing integration test suite...${NC}"
if go test -v ./tests/integration -timeout 60s; then
    echo -e "${GREEN}✓ Integration test suite passed${NC}"
else
    echo -e "${RED}✗ Integration test suite failed${NC}"
    ((failed_tests++))
fi

# 커버리지 리포트 생성
echo -e "\n${YELLOW}Generating coverage report...${NC}"
go test -coverprofile=/tmp/integration_coverage.out ./tests/integration
go tool cover -html=/tmp/integration_coverage.out -o /tmp/integration_coverage.html

# 벤치마크 실행 (선택적)
if [[ "$1" == "--bench" ]]; then
    echo -e "\n${YELLOW}Running integration benchmarks...${NC}"
    go test -bench=. -benchmem ./tests/integration
fi

# 결과 요약
echo -e "\n${YELLOW}========== Test Summary ==========${NC}"
echo -e "Total tests run: $((passed_tests + failed_tests))"
echo -e "${GREEN}Passed: ${passed_tests}${NC}"
echo -e "${RED}Failed: ${failed_tests}${NC}"
echo -e "${YELLOW}Coverage report: /tmp/integration_coverage.html${NC}"

# 정리
echo -e "\n${YELLOW}Cleaning up...${NC}"
rm -rf $STORAGE_DIR

if [[ $failed_tests -gt 0 ]]; then
    echo -e "\n${RED}Integration tests failed!${NC}"
    exit 1
else
    echo -e "\n${GREEN}All integration tests passed!${NC}"
    exit 0
fi
