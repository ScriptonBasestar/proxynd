#!/bin/bash
# Load Testing Script for ProxyND Enterprise API
# Supports both wrk (if installed) and custom Go-based load tester

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default configuration
BASE_URL="${BASE_URL:-http://localhost:8080}"
DURATION="${DURATION:-60s}"
CONCURRENCY="${CONCURRENCY:-100}"
TOKEN="${ENTERPRISE_TOKEN:-}"

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Usage information
usage() {
    cat <<EOF
Usage: $0 [OPTIONS] [SCENARIO]

Load testing script for ProxyND Enterprise API

SCENARIOS:
  quick          Quick validation test (10s, 10 concurrent)
  standard       Standard load test (60s, 100 concurrent)
  stress         Stress test (60s, 500 concurrent)
  ramp-up        Gradual ramp-up test (120s, 10-1000 concurrent)
  sustained      Sustained load test (300s, 100 concurrent)

OPTIONS:
  -u, --url URL          Base URL (default: http://localhost:8080)
  -d, --duration TIME    Test duration (default: 60s)
  -c, --concurrency N    Concurrent connections (default: 100)
  -t, --token TOKEN      Enterprise license token
  -h, --help             Show this help message

ENVIRONMENT VARIABLES:
  BASE_URL              Override base URL
  DURATION              Override duration
  CONCURRENCY           Override concurrency
  ENTERPRISE_TOKEN      Enterprise license token

EXAMPLES:
  # Quick test
  $0 quick

  # Standard test with custom token
  $0 -t "your-token-here" standard

  # Stress test with 1000 concurrent users
  $0 -c 1000 stress

  # Custom test
  $0 -u http://localhost:8080 -d 30s -c 50

EOF
    exit 0
}

# Parse command line arguments
SCENARIO="standard"
while [[ $# -gt 0 ]]; do
    case $1 in
        -u|--url)
            BASE_URL="$2"
            shift 2
            ;;
        -d|--duration)
            DURATION="$2"
            shift 2
            ;;
        -c|--concurrency)
            CONCURRENCY="$2"
            shift 2
            ;;
        -t|--token)
            TOKEN="$2"
            shift 2
            ;;
        -h|--help)
            usage
            ;;
        quick|standard|stress|ramp-up|sustained)
            SCENARIO="$1"
            shift
            ;;
        *)
            echo -e "${RED}Unknown option: $1${NC}"
            usage
            ;;
    esac
done

# Apply scenario-specific settings
case $SCENARIO in
    quick)
        DURATION="10s"
        CONCURRENCY=10
        ;;
    standard)
        DURATION="60s"
        CONCURRENCY=100
        ;;
    stress)
        DURATION="60s"
        CONCURRENCY=500
        ;;
    ramp-up)
        DURATION="120s"
        CONCURRENCY=1000
        ;;
    sustained)
        DURATION="300s"
        CONCURRENCY=100
        ;;
esac

# Print test configuration
echo -e "${BLUE}╔════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║  ProxyND Enterprise API Load Test         ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════╝${NC}"
echo ""
echo -e "${GREEN}Scenario:${NC}     $SCENARIO"
echo -e "${GREEN}Base URL:${NC}     $BASE_URL"
echo -e "${GREEN}Duration:${NC}     $DURATION"
echo -e "${GREEN}Concurrency:${NC}  $CONCURRENCY"
echo -e "${GREEN}Token:${NC}        ${TOKEN:+***configured***}${TOKEN:-not provided}"
echo ""

# Check if server is running
echo -e "${YELLOW}Checking server availability...${NC}"
if ! curl -s -f "$BASE_URL/health" > /dev/null; then
    echo -e "${RED}✗ Server is not responding at $BASE_URL${NC}"
    echo -e "${YELLOW}  Please start the ProxyND server first:${NC}"
    echo -e "${YELLOW}    make dev-run${NC}"
    exit 1
fi
echo -e "${GREEN}✓ Server is running${NC}"
echo ""

# Detect available load testing tool
TOOL=""
if command -v wrk &> /dev/null; then
    TOOL="wrk"
    echo -e "${GREEN}Using wrk (high-performance HTTP benchmarking tool)${NC}"
elif [ -f "$SCRIPT_DIR/loadtest.go" ]; then
    TOOL="go"
    echo -e "${GREEN}Using Go-based load tester${NC}"
else
    echo -e "${RED}✗ No load testing tool available${NC}"
    echo -e "${YELLOW}  Install wrk or ensure loadtest.go exists${NC}"
    exit 1
fi
echo ""

# Run tests for different Enterprise API endpoints
echo -e "${BLUE}════════════════════════════════════════════${NC}"
echo -e "${BLUE}  Running Load Tests${NC}"
echo -e "${BLUE}════════════════════════════════════════════${NC}"
echo ""

# Test endpoints
ENDPOINTS=(
    "/api/v1/enterprise/analytics/overview:Analytics Overview"
    "/api/v1/enterprise/rbac/roles:RBAC Roles"
    "/api/v1/enterprise/audit/events:Audit Events"
    "/api/v1/enterprise/security/vulnerabilities:Security Scan"
)

# Run tests
for endpoint_spec in "${ENDPOINTS[@]}"; do
    IFS=':' read -r endpoint name <<< "$endpoint_spec"

    echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${YELLOW}Testing: ${name}${NC}"
    echo -e "${YELLOW}Endpoint: ${endpoint}${NC}"
    echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""

    URL="${BASE_URL}${endpoint}"

    if [ "$TOOL" = "wrk" ]; then
        # Use wrk
        if [ -n "$TOKEN" ]; then
            wrk -t4 -c"$CONCURRENCY" -d"$DURATION" \
                -H "Authorization: Bearer $TOKEN" \
                --latency \
                "$URL"
        else
            wrk -t4 -c"$CONCURRENCY" -d"$DURATION" \
                --latency \
                "$URL"
        fi
    else
        # Use Go-based tool
        TOKEN_FLAG=""
        if [ -n "$TOKEN" ]; then
            TOKEN_FLAG="-token $TOKEN"
        fi

        go run "$SCRIPT_DIR/loadtest.go" \
            -url "$URL" \
            -duration "$DURATION" \
            -concurrency "$CONCURRENCY" \
            $TOKEN_FLAG
    fi

    echo ""
    echo ""
done

echo -e "${BLUE}════════════════════════════════════════════${NC}"
echo -e "${GREEN}✓ Load testing completed${NC}"
echo -e "${BLUE}════════════════════════════════════════════${NC}"
echo ""

# Performance recommendations
echo -e "${YELLOW}Performance Recommendations:${NC}"
echo -e "  • Target latency: < 200ms (p95)"
echo -e "  • Target throughput: > 1000 rps"
echo -e "  • Target error rate: < 0.1%"
echo ""
echo -e "${YELLOW}Next Steps:${NC}"
echo -e "  • Review metrics at: ${BASE_URL}/metrics"
echo -e "  • Check logs for errors"
echo -e "  • Monitor cache hit rates"
echo -e "  • Adjust rate limiting if needed"
echo ""
