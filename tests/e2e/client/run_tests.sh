#!/bin/bash
# ProxyND E2E Test Runner
# Usage: ./run_tests.sh [options]

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Default values
PROXYND_URL="${PROXYND_URL:-http://localhost:8080}"
VERBOSE=""
MARKERS=""
TEST_PATH=""

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

usage() {
    cat << EOF
ProxyND E2E Test Runner

Usage: $0 [options] [test_path]

Options:
    -h, --help          Show this help message
    -v, --verbose       Verbose output
    -u, --url URL       ProxyND server URL (default: http://localhost:8080)

    # Test categories
    --public            Run public tests only (no auth required)
    --private           Run private tests only (auth required)
    --slow              Include slow tests

    # Package manager tests
    --npm               Run NPM tests only
    --maven             Run Maven tests only
    --auth              Run authentication tests only

    # Specific test files
    npm/                Run all NPM tests
    maven/              Run all Maven tests
    auth/               Run all auth tests
    npm/test_public_download.py   Run specific test file

Examples:
    $0 --public                     # All public tests
    $0 --npm --public               # NPM public tests only
    $0 --maven                      # All Maven tests
    $0 npm/test_tarball_download.py # Specific test file
    $0 -v --slow                    # Verbose with slow tests

Environment Variables:
    PROXYND_URL         Server URL (default: http://localhost:8080)
    PROXYND_USERNAME    Basic auth username
    PROXYND_PASSWORD    Basic auth password
    PROXYND_TOKEN       Bearer token (JWT)
    PROXYND_API_KEY     API key
EOF
}

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            usage
            exit 0
            ;;
        -v|--verbose)
            VERBOSE="-v"
            shift
            ;;
        -u|--url)
            PROXYND_URL="$2"
            shift 2
            ;;
        --public)
            MARKERS="${MARKERS:+$MARKERS and }public"
            shift
            ;;
        --private)
            MARKERS="${MARKERS:+$MARKERS and }private"
            shift
            ;;
        --slow)
            MARKERS="${MARKERS:+$MARKERS and }slow"
            shift
            ;;
        --npm)
            MARKERS="${MARKERS:+$MARKERS and }npm"
            shift
            ;;
        --maven)
            MARKERS="${MARKERS:+$MARKERS and }maven"
            shift
            ;;
        --auth)
            TEST_PATH="auth/"
            shift
            ;;
        *)
            TEST_PATH="$1"
            shift
            ;;
    esac
done

# Check if uv is installed
if ! command -v uv &> /dev/null; then
    echo -e "${RED}Error: uv is not installed${NC}"
    echo "Install with: curl -LsSf https://astral.sh/uv/install.sh | sh"
    exit 1
fi

# Check server availability
echo -e "${YELLOW}Checking server at $PROXYND_URL...${NC}"
if ! curl -s -o /dev/null -w "" --connect-timeout 5 "$PROXYND_URL/healthz" 2>/dev/null; then
    echo -e "${RED}Warning: Server may not be available at $PROXYND_URL${NC}"
    echo "Tests will be skipped if server is unreachable."
fi

# Build pytest command
PYTEST_CMD="uv run pytest"
[[ -n "$VERBOSE" ]] && PYTEST_CMD="$PYTEST_CMD $VERBOSE"
[[ -n "$MARKERS" ]] && PYTEST_CMD="$PYTEST_CMD -m \"$MARKERS\""
[[ -n "$TEST_PATH" ]] && PYTEST_CMD="$PYTEST_CMD $TEST_PATH"

# Export environment
export PROXYND_URL

# Run tests
echo -e "${GREEN}Running: $PYTEST_CMD${NC}"
echo "---"
eval "$PYTEST_CMD"
