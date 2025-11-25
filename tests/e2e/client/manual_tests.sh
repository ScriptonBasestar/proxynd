#!/bin/bash
# ProxyND Manual Test Commands
# Usage: ./manual_tests.sh [test_name]
#
# This script provides curl commands for manual testing.
# Run without arguments to see all available tests.

set -e

PROXYND_URL="${PROXYND_URL:-http://localhost:8080}"
USERNAME="${PROXYND_USERNAME:-testuser}"
PASSWORD="${PROXYND_PASSWORD:-testpass}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

print_header() {
    echo -e "\n${CYAN}=== $1 ===${NC}"
}

print_cmd() {
    echo -e "${YELLOW}$ $1${NC}"
}

run_cmd() {
    print_cmd "$1"
    eval "$1"
    echo ""
}

# ============================================================
# NPM Tests
# ============================================================

test_npm_public_metadata() {
    print_header "NPM: Get Package Metadata"
    run_cmd "curl -s '$PROXYND_URL/proxy/npm/express' | head -c 500"
}

test_npm_scoped_package() {
    print_header "NPM: Get Scoped Package"
    run_cmd "curl -s '$PROXYND_URL/proxy/npm/@types/node' | head -c 500"
}

test_npm_tarball() {
    print_header "NPM: Download Tarball"
    run_cmd "curl -s -o /tmp/express.tgz '$PROXYND_URL/proxy/npm/express/-/express-4.18.2.tgz' && file /tmp/express.tgz"
}

test_npm_not_found() {
    print_header "NPM: Package Not Found (expect 404)"
    run_cmd "curl -s -o /dev/null -w 'HTTP Status: %{http_code}\n' '$PROXYND_URL/proxy/npm/nonexistent-pkg-12345'"
}

test_npm_private_unauthorized() {
    print_header "NPM Private: Unauthorized (expect 401/403)"
    run_cmd "curl -s -o /dev/null -w 'HTTP Status: %{http_code}\n' '$PROXYND_URL/proxy/npm-private/internal-pkg'"
}

test_npm_private_basic_auth() {
    print_header "NPM Private: Basic Auth"
    run_cmd "curl -s -o /dev/null -w 'HTTP Status: %{http_code}\n' -u '$USERNAME:$PASSWORD' '$PROXYND_URL/proxy/npm-private/internal-pkg'"
}

# ============================================================
# Maven Tests
# ============================================================

test_maven_pom() {
    print_header "Maven: Get POM File"
    run_cmd "curl -s '$PROXYND_URL/proxy/maven/junit/junit/4.13.2/junit-4.13.2.pom' | head -20"
}

test_maven_metadata() {
    print_header "Maven: Get Metadata"
    run_cmd "curl -s '$PROXYND_URL/proxy/maven/junit/junit/maven-metadata.xml' | head -20"
}

test_maven_jar() {
    print_header "Maven: Download JAR"
    run_cmd "curl -s -o /tmp/junit.jar '$PROXYND_URL/proxy/maven/junit/junit/4.13.2/junit-4.13.2.jar' && file /tmp/junit.jar"
}

test_maven_not_found() {
    print_header "Maven: Artifact Not Found (expect 404)"
    run_cmd "curl -s -o /dev/null -w 'HTTP Status: %{http_code}\n' '$PROXYND_URL/proxy/maven/nonexistent/artifact/1.0/artifact-1.0.pom'"
}

test_maven_private_unauthorized() {
    print_header "Maven Private: Unauthorized (expect 401/403)"
    run_cmd "curl -s -o /dev/null -w 'HTTP Status: %{http_code}\n' '$PROXYND_URL/proxy/maven-private/com/internal/lib/1.0/lib-1.0.pom'"
}

# ============================================================
# Auth Tests
# ============================================================

test_auth_basic_valid() {
    print_header "Auth: Basic Auth (valid credentials)"
    run_cmd "curl -s -o /dev/null -w 'HTTP Status: %{http_code}\n' -u '$USERNAME:$PASSWORD' '$PROXYND_URL/api/status'"
}

test_auth_basic_invalid() {
    print_header "Auth: Basic Auth (invalid credentials, expect 401)"
    run_cmd "curl -s -o /dev/null -w 'HTTP Status: %{http_code}\n' -u 'invalid:invalid' '$PROXYND_URL/proxy/npm-private/test'"
}

test_auth_bearer_invalid() {
    print_header "Auth: Bearer Token (invalid, expect 401)"
    run_cmd "curl -s -o /dev/null -w 'HTTP Status: %{http_code}\n' -H 'Authorization: Bearer invalid-token' '$PROXYND_URL/proxy/npm-private/test'"
}

test_auth_api_key_invalid() {
    print_header "Auth: API Key (invalid, expect 401)"
    run_cmd "curl -s -o /dev/null -w 'HTTP Status: %{http_code}\n' -H 'X-API-Key: invalid-key' '$PROXYND_URL/proxy/npm-private/test'"
}

# ============================================================
# Server Status
# ============================================================

test_health() {
    print_header "Server: Health Check"
    run_cmd "curl -s '$PROXYND_URL/healthz'"
}

test_status() {
    print_header "Server: Status"
    run_cmd "curl -s '$PROXYND_URL/api/status' | head -50"
}

# ============================================================
# Main
# ============================================================

list_tests() {
    cat << EOF
Available Tests:

NPM Tests:
    npm_public_metadata     - Get NPM package metadata
    npm_scoped_package      - Get scoped package (@types/node)
    npm_tarball             - Download NPM tarball
    npm_not_found           - Test 404 response
    npm_private_unauthorized - Test unauthorized access
    npm_private_basic_auth  - Test Basic Auth access

Maven Tests:
    maven_pom               - Get Maven POM file
    maven_metadata          - Get Maven metadata.xml
    maven_jar               - Download Maven JAR
    maven_not_found         - Test 404 response
    maven_private_unauthorized - Test unauthorized access

Auth Tests:
    auth_basic_valid        - Valid Basic Auth
    auth_basic_invalid      - Invalid Basic Auth (expect 401)
    auth_bearer_invalid     - Invalid Bearer token (expect 401)
    auth_api_key_invalid    - Invalid API key (expect 401)

Server Tests:
    health                  - Health check
    status                  - Server status

Run all:
    all                     - Run all tests

Usage:
    ./manual_tests.sh <test_name>
    ./manual_tests.sh all
    ./manual_tests.sh npm_public_metadata

Environment:
    PROXYND_URL=$PROXYND_URL
    PROXYND_USERNAME=$USERNAME
EOF
}

run_all() {
    test_health
    test_status
    test_npm_public_metadata
    test_npm_scoped_package
    test_npm_tarball
    test_npm_not_found
    test_npm_private_unauthorized
    test_npm_private_basic_auth
    test_maven_pom
    test_maven_metadata
    test_maven_jar
    test_maven_not_found
    test_maven_private_unauthorized
    test_auth_basic_valid
    test_auth_basic_invalid
    test_auth_bearer_invalid
    test_auth_api_key_invalid
}

case "${1:-}" in
    "")
        list_tests
        ;;
    all)
        run_all
        ;;
    *)
        func_name="test_$1"
        if declare -f "$func_name" > /dev/null; then
            "$func_name"
        else
            echo -e "${RED}Unknown test: $1${NC}"
            echo "Run without arguments to see available tests."
            exit 1
        fi
        ;;
esac
