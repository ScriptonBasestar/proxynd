#!/bin/bash
#
# Performance Benchmark Script for ProxyND Enterprise API
# Tests API response times and validates <500ms target (p95)
#

set -euo pipefail

# Configuration
BASE_URL="${PROXYND_URL:-http://localhost:8080}"
API_BASE="/api/v1/enterprise"
ITERATIONS="${ITERATIONS:-100}"
CONCURRENT="${CONCURRENT:-10}"
P95_TARGET_MS=500
P99_TARGET_MS=1000

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Statistics arrays
declare -a response_times

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}ProxyND Enterprise API Performance Benchmark${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""
echo "Configuration:"
echo "  Base URL: $BASE_URL"
echo "  Iterations: $ITERATIONS"
echo "  Concurrent: $CONCURRENT"
echo "  P95 Target: <${P95_TARGET_MS}ms"
echo "  P99 Target: <${P99_TARGET_MS}ms"
echo ""

# Test endpoint and measure response time
benchmark_endpoint() {
    local name="$1"
    local method="$2"
    local endpoint="$3"
    local iterations="${4:-$ITERATIONS}"

    echo -e "${YELLOW}Testing: $name${NC}"
    echo "  Method: $method"
    echo "  Endpoint: $endpoint"
    echo "  Iterations: $iterations"

    local times=()
    local total_time=0
    local failures=0

    for i in $(seq 1 $iterations); do
        local start=$(date +%s%N)
        local status_code=$(curl -s -w "%{http_code}" -o /dev/null \
            -X "$method" \
            "${BASE_URL}${API_BASE}${endpoint}" 2>/dev/null || echo "000")
        local end=$(date +%s%N)

        local duration_ns=$((end - start))
        local duration_ms=$((duration_ns / 1000000))

        if [[ "$status_code" -ge 200 && "$status_code" -lt 300 ]]; then
            times+=($duration_ms)
            total_time=$((total_time + duration_ms))
        else
            ((failures++)) || true
        fi

        # Progress indicator
        if (( i % 10 == 0 )); then
            echo -n "."
        fi
    done
    echo ""

    # Calculate statistics
    if [ ${#times[@]} -eq 0 ]; then
        echo -e "${RED}  ✗ All requests failed${NC}"
        return 1
    fi

    # Sort times
    IFS=$'\n' sorted_times=($(sort -n <<<"${times[*]}"))
    unset IFS

    local count=${#sorted_times[@]}
    local successful=$((count - failures))
    local avg=$((total_time / count))
    local min=${sorted_times[0]}
    local max=${sorted_times[-1]}

    # Calculate percentiles
    local p50_idx=$(( count * 50 / 100 ))
    local p95_idx=$(( count * 95 / 100 ))
    local p99_idx=$(( count * 99 / 100 ))

    local p50=${sorted_times[$p50_idx]}
    local p95=${sorted_times[$p95_idx]}
    local p99=${sorted_times[$p99_idx]}

    # Display results
    echo "  Results:"
    echo "    Success rate: $successful/$iterations ($(( successful * 100 / iterations ))%)"
    echo "    Min:  ${min}ms"
    echo "    Avg:  ${avg}ms"
    echo "    Max:  ${max}ms"
    echo "    P50:  ${p50}ms"
    echo "    P95:  ${p95}ms"
    echo "    P99:  ${p99}ms"

    # Validate targets
    if [ $p95 -lt $P95_TARGET_MS ]; then
        echo -e "    ${GREEN}✓ P95 target met (<${P95_TARGET_MS}ms)${NC}"
    else
        echo -e "    ${RED}✗ P95 target missed (${p95}ms >= ${P95_TARGET_MS}ms)${NC}"
    fi

    if [ $p99 -lt $P99_TARGET_MS ]; then
        echo -e "    ${GREEN}✓ P99 target met (<${P99_TARGET_MS}ms)${NC}"
    else
        echo -e "    ${YELLOW}⚠ P99 target missed (${p99}ms >= ${P99_TARGET_MS}ms)${NC}"
    fi

    echo ""
}

# Benchmark pagination performance
benchmark_pagination() {
    local name="$1"
    local endpoint="$2"
    local pages="${3:-10}"

    echo -e "${YELLOW}Testing Pagination: $name${NC}"
    echo "  Endpoint: $endpoint"
    echo "  Pages to test: $pages"

    local total_time=0
    local successful=0

    for page in $(seq 1 $pages); do
        local start=$(date +%s%N)
        local status_code=$(curl -s -w "%{http_code}" -o /dev/null \
            "${BASE_URL}${API_BASE}${endpoint}?page=${page}&per_page=20" 2>/dev/null || echo "000")
        local end=$(date +%s%N)

        local duration_ns=$((end - start))
        local duration_ms=$((duration_ns / 1000000))

        if [[ "$status_code" -ge 200 && "$status_code" -lt 300 ]]; then
            total_time=$((total_time + duration_ms))
            ((successful++)) || true
        fi

        echo -n "."
    done
    echo ""

    local avg=$((total_time / successful))
    echo "  Results:"
    echo "    Pages tested: $successful/$pages"
    echo "    Avg time per page: ${avg}ms"

    if [ $avg -lt $P95_TARGET_MS ]; then
        echo -e "    ${GREEN}✓ Pagination performance good (<${P95_TARGET_MS}ms avg)${NC}"
    else
        echo -e "    ${RED}✗ Pagination performance slow (${avg}ms >= ${P95_TARGET_MS}ms)${NC}"
    fi
    echo ""
}

# Concurrent requests benchmark
benchmark_concurrent() {
    local name="$1"
    local endpoint="$2"
    local concurrent="${3:-$CONCURRENT}"

    echo -e "${YELLOW}Testing Concurrent Requests: $name${NC}"
    echo "  Endpoint: $endpoint"
    echo "  Concurrent requests: $concurrent"

    local start=$(date +%s%N)

    # Run concurrent requests
    for i in $(seq 1 $concurrent); do
        curl -s -o /dev/null "${BASE_URL}${API_BASE}${endpoint}" &
    done

    # Wait for all to complete
    wait

    local end=$(date +%s%N)
    local duration_ns=$((end - start))
    local duration_ms=$((duration_ns / 1000000))
    local avg_per_request=$((duration_ms / concurrent))

    echo "  Results:"
    echo "    Total time: ${duration_ms}ms"
    echo "    Avg per request: ${avg_per_request}ms"

    if [ $avg_per_request -lt $P95_TARGET_MS ]; then
        echo -e "    ${GREEN}✓ Concurrent performance good (<${P95_TARGET_MS}ms avg)${NC}"
    else
        echo -e "    ${RED}✗ Concurrent performance slow (${avg_per_request}ms >= ${P95_TARGET_MS}ms)${NC}"
    fi
    echo ""
}

# Main benchmark suite
main() {
    echo -e "${BLUE}Starting Performance Benchmarks...${NC}"
    echo ""

    # RBAC Endpoints
    echo -e "${BLUE}=== RBAC Endpoints ===${NC}"
    benchmark_endpoint "List Roles" "GET" "/rbac/roles" 50
    benchmark_endpoint "Get Role" "GET" "/rbac/roles/role_admin" 50
    benchmark_endpoint "List Permissions" "GET" "/rbac/permissions" 50

    # Audit Endpoints
    echo -e "${BLUE}=== Audit Endpoints ===${NC}"
    benchmark_endpoint "List Events" "GET" "/audit/events" 50
    benchmark_endpoint "Get Stats" "GET" "/audit/stats" 50

    # Analytics Endpoints
    echo -e "${BLUE}=== Analytics Endpoints ===${NC}"
    benchmark_endpoint "Get Overview" "GET" "/analytics/overview" 50
    benchmark_endpoint "Get Usage Stats" "GET" "/analytics/usage" 50
    benchmark_endpoint "Get Performance" "GET" "/analytics/performance" 50

    # Security Endpoints
    echo -e "${BLUE}=== Security Endpoints ===${NC}"
    benchmark_endpoint "List Vulnerabilities" "GET" "/security/vulnerabilities" 50
    benchmark_endpoint "List Licenses" "GET" "/security/licenses" 50

    # Alerts Endpoints
    echo -e "${BLUE}=== Alerts Endpoints ===${NC}"
    benchmark_endpoint "List Alerts" "GET" "/alerts" 50

    # License Endpoints
    echo -e "${BLUE}=== License Endpoints ===${NC}"
    benchmark_endpoint "Get License Info" "GET" "/license/info" 50
    benchmark_endpoint "List Features" "GET" "/license/features" 50

    # Pagination Tests
    echo -e "${BLUE}=== Pagination Tests ===${NC}"
    benchmark_pagination "RBAC Roles Pagination" "/rbac/roles" 10
    benchmark_pagination "Audit Events Pagination" "/audit/events" 10
    benchmark_pagination "Vulnerabilities Pagination" "/security/vulnerabilities" 10

    # Concurrent Tests
    echo -e "${BLUE}=== Concurrent Request Tests ===${NC}"
    benchmark_concurrent "RBAC Concurrent" "/rbac/roles" 20
    benchmark_concurrent "Analytics Concurrent" "/analytics/overview" 20
    benchmark_concurrent "Audit Concurrent" "/audit/events" 20

    echo -e "${GREEN}========================================${NC}"
    echo -e "${GREEN}Benchmark Complete!${NC}"
    echo -e "${GREEN}========================================${NC}"
    echo ""
    echo "Performance targets:"
    echo "  P95: <${P95_TARGET_MS}ms"
    echo "  P99: <${P99_TARGET_MS}ms"
    echo ""
    echo "Review results above to identify slow endpoints."
    echo "For detailed profiling, use: make profile-api"
}

# Run benchmarks
main "$@"
