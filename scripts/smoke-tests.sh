#!/bin/bash

# ProxyND Smoke Tests
# This script runs smoke tests against a deployed ProxyND instance

set -euo pipefail

# Configuration
BASE_URL=""
TIMEOUT=30
VERBOSE=false
PROXY_TYPES=("apt" "maven" "npm")

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test results
TESTS_TOTAL=0
TESTS_PASSED=0
TESTS_FAILED=0
FAILED_TESTS=()

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[PASS]${NC} $1"
}

log_failure() {
    echo -e "${RED}[FAIL]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

# Help function
show_help() {
    cat << EOF
ProxyND Smoke Test Script

Usage: $0 [OPTIONS] <BASE_URL>

Arguments:
    BASE_URL                        Base URL of ProxyND instance (e.g., https://proxynd.example.com)

Options:
    -t, --timeout TIMEOUT          Request timeout in seconds (default: 30)
    -v, --verbose                   Enable verbose output
    -p, --proxy-types TYPES         Comma-separated list of proxy types to test (default: apt,maven,npm)
    -h, --help                     Show this help message

Examples:
    $0 https://proxynd.example.com
    $0 -v -t 60 https://staging.proxynd.example.com
    $0 -p apt,maven https://proxynd.example.com

EOF
}

# Parse command line arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            -t|--timeout)
                TIMEOUT="$2"
                shift 2
                ;;
            -v|--verbose)
                VERBOSE=true
                shift
                ;;
            -p|--proxy-types)
                IFS=',' read -ra PROXY_TYPES <<< "$2"
                shift 2
                ;;
            -h|--help)
                show_help
                exit 0
                ;;
            *)
                if [[ -z "$BASE_URL" ]]; then
                    BASE_URL="$1"
                else
                    log_failure "Unknown option: $1"
                    show_help
                    exit 1
                fi
                shift
                ;;
        esac
    done

    if [[ -z "$BASE_URL" ]]; then
        log_failure "BASE_URL is required"
        show_help
        exit 1
    fi

    # Remove trailing slash
    BASE_URL="${BASE_URL%/}"
}

# Make HTTP request with error handling
make_request() {
    local url="$1"
    local expected_status="${2:-200}"
    local method="${3:-GET}"
    local description="$4"

    TESTS_TOTAL=$((TESTS_TOTAL + 1))

    if [[ "$VERBOSE" == true ]]; then
        log_info "Testing: $description"
        log_info "URL: $url"
        log_info "Expected Status: $expected_status"
    fi

    local response
    local status_code
    local curl_exit_code

    # Make request and capture response
    response=$(curl -s -w "\n%{http_code}" \
        --max-time "$TIMEOUT" \
        --connect-timeout 10 \
        -X "$method" \
        -H "User-Agent: ProxyND-SmokeTest/1.0" \
        "$url" 2>/dev/null) || curl_exit_code=$?

    if [[ ${curl_exit_code:-0} -ne 0 ]]; then
        log_failure "$description - Request failed (curl exit code: ${curl_exit_code:-0})"
        TESTS_FAILED=$((TESTS_FAILED + 1))
        FAILED_TESTS+=("$description")
        return 1
    fi

    # Extract status code from response
    status_code=$(echo "$response" | tail -n1)
    local body=$(echo "$response" | head -n -1)

    if [[ "$status_code" == "$expected_status" ]]; then
        log_success "$description - HTTP $status_code"
        TESTS_PASSED=$((TESTS_PASSED + 1))

        if [[ "$VERBOSE" == true && -n "$body" ]]; then
            echo "Response body: $body"
        fi
        return 0
    else
        log_failure "$description - Expected HTTP $expected_status, got HTTP $status_code"
        TESTS_FAILED=$((TESTS_FAILED + 1))
        FAILED_TESTS+=("$description")

        if [[ "$VERBOSE" == true && -n "$body" ]]; then
            echo "Response body: $body"
        fi
        return 1
    fi
}

# Test health endpoint
test_health() {
    log_info "Testing health endpoints..."

    make_request "$BASE_URL/health" 200 GET "Health check endpoint"
    make_request "$BASE_URL/health/ready" 200 GET "Readiness check endpoint"
    make_request "$BASE_URL/health/live" 200 GET "Liveness check endpoint"
}

# Test metrics endpoint
test_metrics() {
    log_info "Testing metrics endpoint..."

    make_request "$BASE_URL/metrics" 200 GET "Prometheus metrics endpoint"
}

# Test version endpoint
test_version() {
    log_info "Testing version endpoint..."

    make_request "$BASE_URL/version" 200 GET "Version information endpoint"
}

# Test APT proxy endpoints
test_apt_proxy() {
    log_info "Testing APT proxy endpoints..."

    # Test APT repository structure
    make_request "$BASE_URL/proxy/apt/dists/" 200 GET "APT dists directory listing"
    make_request "$BASE_URL/proxy/apt/dists/jammy/Release" 200 GET "APT Release file"
    make_request "$BASE_URL/proxy/apt/dists/jammy/InRelease" 200 GET "APT InRelease file"

    # Test package queries (might return 404 if not cached, which is acceptable)
    make_request "$BASE_URL/proxy/apt/pool/main/a/apt/apt_2.4.8_amd64.deb" 404 GET "APT package download (expected 404 if not cached)"
}

# Test Maven proxy endpoints
test_maven_proxy() {
    log_info "Testing Maven proxy endpoints..."

    # Test Maven repository structure
    make_request "$BASE_URL/proxy/maven/" 200 GET "Maven root directory"
    make_request "$BASE_URL/proxy/maven/org/springframework/" 200 GET "Maven group directory"

    # Test metadata files
    make_request "$BASE_URL/proxy/maven/org/springframework/spring-core/maven-metadata.xml" 200 GET "Maven metadata XML"

    # Test artifact download (might return 404 if not cached)
    make_request "$BASE_URL/proxy/maven/org/springframework/spring-core/5.3.21/spring-core-5.3.21.pom" 404 GET "Maven POM download (expected 404 if not cached)"
}

# Test NPM proxy endpoints
test_npm_proxy() {
    log_info "Testing NPM proxy endpoints..."

    # Test NPM registry endpoints
    make_request "$BASE_URL/proxy/npm/" 200 GET "NPM registry root"
    make_request "$BASE_URL/proxy/npm/express" 200 GET "NPM package metadata"

    # Test package download (might return 404 if not cached)
    make_request "$BASE_URL/proxy/npm/express/-/express-4.18.2.tgz" 404 GET "NPM package download (expected 404 if not cached)"
}

# Test administrative endpoints
test_admin_endpoints() {
    log_info "Testing administrative endpoints..."

    # Test cache statistics
    make_request "$BASE_URL/admin/cache/stats" 200 GET "Cache statistics endpoint"

    # Test system statistics
    make_request "$BASE_URL/admin/stats" 200 GET "System statistics endpoint"

    # Test configuration endpoint
    make_request "$BASE_URL/admin/config" 200 GET "Configuration endpoint"
}

# Test security endpoints (should return 401/403 for unauthorized access)
test_security() {
    log_info "Testing security endpoints..."

    # Test admin endpoints without authentication (should be protected)
    make_request "$BASE_URL/admin/cache/clear" 401 POST "Cache clear endpoint (should require auth)"
    make_request "$BASE_URL/admin/config/reload" 401 POST "Config reload endpoint (should require auth)"
}

# Test performance endpoints
test_performance() {
    log_info "Testing performance endpoints..."

    # Test performance metrics
    make_request "$BASE_URL/admin/performance/metrics" 200 GET "Performance metrics endpoint"
    make_request "$BASE_URL/admin/performance/health" 200 GET "Performance health endpoint"
}

# Test error handling
test_error_handling() {
    log_info "Testing error handling..."

    # Test 404 responses
    make_request "$BASE_URL/nonexistent" 404 GET "Non-existent endpoint (should return 404)"
    make_request "$BASE_URL/proxy/nonexistent/" 404 GET "Non-existent proxy type (should return 404)"

    # Test method not allowed
    make_request "$BASE_URL/health" 405 DELETE "Health endpoint with DELETE method (should return 405)"
}

# Run proxy-specific tests
run_proxy_tests() {
    for proxy_type in "${PROXY_TYPES[@]}"; do
        case "$proxy_type" in
            apt)
                test_apt_proxy
                ;;
            maven)
                test_maven_proxy
                ;;
            npm)
                test_npm_proxy
                ;;
            *)
                log_warning "Unknown proxy type: $proxy_type"
                ;;
        esac
    done
}

# Generate test report
generate_report() {
    echo
    log_info "Smoke Test Report"
    log_info "=================="
    echo -e "Base URL: ${BLUE}$BASE_URL${NC}"
    echo -e "Total Tests: ${BLUE}$TESTS_TOTAL${NC}"
    echo -e "Passed: ${GREEN}$TESTS_PASSED${NC}"
    echo -e "Failed: ${RED}$TESTS_FAILED${NC}"

    local success_rate=0
    if [[ $TESTS_TOTAL -gt 0 ]]; then
        success_rate=$((TESTS_PASSED * 100 / TESTS_TOTAL))
    fi
    echo -e "Success Rate: ${BLUE}${success_rate}%${NC}"

    if [[ $TESTS_FAILED -gt 0 ]]; then
        echo
        log_failure "Failed Tests:"
        for test in "${FAILED_TESTS[@]}"; do
            echo -e "  ${RED}✗${NC} $test"
        done
    fi

    echo
    if [[ $TESTS_FAILED -eq 0 ]]; then
        log_success "All smoke tests passed! 🎉"
        return 0
    else
        log_failure "Some smoke tests failed! 💥"
        return 1
    fi
}

# Main execution
main() {
    log_info "ProxyND Smoke Tests"
    log_info "==================="

    parse_args "$@"

    log_info "Starting smoke tests against: $BASE_URL"
    log_info "Timeout: ${TIMEOUT}s"
    log_info "Proxy types: ${PROXY_TYPES[*]}"
    echo

    # Core functionality tests
    test_health
    test_version
    test_metrics

    # Proxy-specific tests
    run_proxy_tests

    # Administrative tests
    test_admin_endpoints
    test_performance

    # Security tests
    test_security

    # Error handling tests
    test_error_handling

    # Generate final report
    generate_report
}

# Execute main function with all arguments
main "$@"
