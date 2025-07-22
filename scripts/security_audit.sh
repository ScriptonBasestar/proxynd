#!/bin/bash

# Security Audit Script for ProxyND
# This script performs comprehensive security checks on the ProxyND codebase

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
REPORTS_DIR="${PROJECT_ROOT}/security-reports"

# Create reports directory
mkdir -p "${REPORTS_DIR}"

# Functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

check_tool() {
    local tool="$1"
    local install_cmd="$2"

    if ! command -v "${tool}" &> /dev/null; then
        log_warn "${tool} not found. Installing..."
        eval "${install_cmd}"
        if ! command -v "${tool}" &> /dev/null; then
            log_error "Failed to install ${tool}"
            return 1
        fi
    fi
    log_success "${tool} is available"
    return 0
}

# Security audit functions
audit_code_security() {
    log_info "Running code security analysis..."

    # Run gosec
    if check_tool "gosec" "go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest"; then
        log_info "Running gosec security scanner..."
        gosec -fmt json -out "${REPORTS_DIR}/gosec-report.json" ./...
        gosec -fmt text -out "${REPORTS_DIR}/gosec-report.txt" ./...
        log_success "Gosec scan completed"
    fi

    # Run staticcheck
    if check_tool "staticcheck" "go install honnef.co/go/tools/cmd/staticcheck@latest"; then
        log_info "Running staticcheck analysis..."
        staticcheck ./... > "${REPORTS_DIR}/staticcheck-report.txt" 2>&1 || true
        log_success "Staticcheck analysis completed"
    fi

    # Run govulncheck
    if check_tool "govulncheck" "go install golang.org/x/vuln/cmd/govulncheck@latest"; then
        log_info "Running govulncheck..."
        govulncheck -json ./... > "${REPORTS_DIR}/govulncheck-report.json" 2>&1 || true
        govulncheck ./... > "${REPORTS_DIR}/govulncheck-report.txt" 2>&1 || true
        log_success "Govulncheck completed"
    fi
}

audit_dependencies() {
    log_info "Running dependency security analysis..."

    # Check for nancy
    if check_tool "nancy" "go install github.com/sonatypecommunity/nancy@latest"; then
        log_info "Running nancy vulnerability scanner..."
        go list -json -deps ./... | nancy sleuth --loud --output-format=json > "${REPORTS_DIR}/nancy-report.json" 2>&1 || true
        go list -json -deps ./... | nancy sleuth --loud > "${REPORTS_DIR}/nancy-report.txt" 2>&1 || true
        log_success "Nancy scan completed"
    fi

    # Check for known vulnerabilities in go.mod
    log_info "Checking go.mod for known vulnerabilities..."
    go mod tidy
    go list -m all > "${REPORTS_DIR}/dependencies-list.txt"

    # Check for outdated dependencies
    log_info "Checking for outdated dependencies..."
    go list -u -m all > "${REPORTS_DIR}/outdated-dependencies.txt" 2>&1 || true
}

audit_secrets() {
    log_info "Running secret detection..."

    # Check for common secret patterns
    log_info "Scanning for hardcoded secrets..."

    # Common secret patterns
    local patterns=(
        "password\s*=\s*[\"'][^\"']*[\"']"
        "api_key\s*=\s*[\"'][^\"']*[\"']"
        "secret\s*=\s*[\"'][^\"']*[\"']"
        "token\s*=\s*[\"'][^\"']*[\"']"
        "-----BEGIN [A-Z ]*PRIVATE KEY-----"
        "[A-Za-z0-9+/]{40,}"
    )

    for pattern in "${patterns[@]}"; do
        if grep -rn --include="*.go" --include="*.yaml" --include="*.yml" --include="*.json" -E "${pattern}" "${PROJECT_ROOT}" > /dev/null 2>&1; then
            log_warn "Potential secret found: ${pattern}"
            grep -rn --include="*.go" --include="*.yaml" --include="*.yml" --include="*.json" -E "${pattern}" "${PROJECT_ROOT}" >> "${REPORTS_DIR}/potential-secrets.txt" 2>&1 || true
        fi
    done

    # Check for environment variables that might contain secrets
    log_info "Checking for environment variables..."
    grep -rn --include="*.go" "os\.Getenv" "${PROJECT_ROOT}" > "${REPORTS_DIR}/env-vars.txt" 2>&1 || true
}

audit_docker() {
    log_info "Running Docker security audit..."

    if [ -f "${PROJECT_ROOT}/Dockerfile" ]; then
        log_info "Checking Dockerfile for security best practices..."

        # Check for non-root user
        if ! grep -q "USER " "${PROJECT_ROOT}/Dockerfile"; then
            log_warn "Dockerfile does not specify a non-root user"
            echo "WARNING: No USER instruction found in Dockerfile" >> "${REPORTS_DIR}/docker-security.txt"
        fi

        # Check for HEALTHCHECK
        if ! grep -q "HEALTHCHECK" "${PROJECT_ROOT}/Dockerfile"; then
            log_warn "Dockerfile does not include HEALTHCHECK instruction"
            echo "WARNING: No HEALTHCHECK instruction found in Dockerfile" >> "${REPORTS_DIR}/docker-security.txt"
        fi

        # Check for minimal base image
        if grep -q "FROM.*:latest" "${PROJECT_ROOT}/Dockerfile"; then
            log_warn "Dockerfile uses :latest tag which is not recommended"
            echo "WARNING: Using :latest tag in Dockerfile" >> "${REPORTS_DIR}/docker-security.txt"
        fi

        log_success "Docker security audit completed"
    else
        log_warn "Dockerfile not found"
    fi
}

audit_configuration() {
    log_info "Running configuration security audit..."

    # Check for insecure default configurations
    find "${PROJECT_ROOT}/configs" -name "*.yaml" -o -name "*.yml" -o -name "*.toml" | while read -r config_file; do
        if [ -f "$config_file" ]; then
            log_info "Checking configuration file: ${config_file}"

            # Check for insecure settings
            if grep -q "debug.*true" "$config_file" 2>/dev/null; then
                log_warn "Debug mode enabled in ${config_file}"
                echo "WARNING: Debug mode enabled in ${config_file}" >> "${REPORTS_DIR}/config-security.txt"
            fi

            if grep -q "tls.*false" "$config_file" 2>/dev/null; then
                log_warn "TLS disabled in ${config_file}"
                echo "WARNING: TLS disabled in ${config_file}" >> "${REPORTS_DIR}/config-security.txt"
            fi

            if grep -q "auth.*false" "$config_file" 2>/dev/null; then
                log_warn "Authentication disabled in ${config_file}"
                echo "WARNING: Authentication disabled in ${config_file}" >> "${REPORTS_DIR}/config-security.txt"
            fi
        fi
    done
}

audit_permissions() {
    log_info "Running file permissions audit..."

    # Check for world-writable files
    find "${PROJECT_ROOT}" -type f -perm /002 > "${REPORTS_DIR}/world-writable-files.txt" 2>/dev/null || true

    # Check for executable files that shouldn't be
    find "${PROJECT_ROOT}" -name "*.go" -executable > "${REPORTS_DIR}/executable-go-files.txt" 2>/dev/null || true

    # Check for setuid/setgid files
    find "${PROJECT_ROOT}" -type f \( -perm -4000 -o -perm -2000 \) > "${REPORTS_DIR}/setuid-setgid-files.txt" 2>/dev/null || true

    log_success "File permissions audit completed"
}

generate_summary() {
    log_info "Generating security audit summary..."

    local summary_file="${REPORTS_DIR}/security-audit-summary.txt"

    {
        echo "ProxyND Security Audit Summary"
        echo "==============================="
        echo "Date: $(date)"
        echo "Git Commit: $(git rev-parse HEAD 2>/dev/null || echo 'unknown')"
        echo ""

        echo "Files Generated:"
        ls -la "${REPORTS_DIR}/"
        echo ""

        echo "Critical Findings:"
        echo "-----------------"

        # Check for critical issues
        if [ -f "${REPORTS_DIR}/gosec-report.json" ]; then
            local high_issues=$(jq -r '.Issues[] | select(.Severity == "HIGH") | .What' "${REPORTS_DIR}/gosec-report.json" 2>/dev/null | wc -l)
            echo "High severity code issues: ${high_issues}"
        fi

        if [ -f "${REPORTS_DIR}/nancy-report.json" ]; then
            local vulnerable_deps=$(jq -r '.[] | select(.Vulnerabilities != null) | .Coordinates' "${REPORTS_DIR}/nancy-report.json" 2>/dev/null | wc -l)
            echo "Vulnerable dependencies: ${vulnerable_deps}"
        fi

        if [ -f "${REPORTS_DIR}/potential-secrets.txt" ]; then
            local secret_matches=$(wc -l < "${REPORTS_DIR}/potential-secrets.txt")
            echo "Potential secret matches: ${secret_matches}"
        fi

        echo ""
        echo "Recommendations:"
        echo "---------------"
        echo "1. Review all high severity issues in gosec-report.txt"
        echo "2. Update vulnerable dependencies listed in nancy-report.txt"
        echo "3. Verify potential secrets in potential-secrets.txt"
        echo "4. Address configuration security issues in config-security.txt"
        echo "5. Fix Docker security issues in docker-security.txt"

    } > "${summary_file}"

    log_success "Security audit summary generated: ${summary_file}"
}

# Main execution
main() {
    log_info "Starting ProxyND Security Audit..."
    log_info "Project Root: ${PROJECT_ROOT}"
    log_info "Reports Directory: ${REPORTS_DIR}"

    cd "${PROJECT_ROOT}"

    # Clean previous reports
    rm -rf "${REPORTS_DIR}"
    mkdir -p "${REPORTS_DIR}"

    # Run security audits
    audit_code_security
    audit_dependencies
    audit_secrets
    audit_docker
    audit_configuration
    audit_permissions

    # Generate summary
    generate_summary

    log_success "Security audit completed successfully!"
    log_info "Review the reports in: ${REPORTS_DIR}"

    # Display summary
    echo ""
    echo "=== SECURITY AUDIT SUMMARY ==="
    cat "${REPORTS_DIR}/security-audit-summary.txt"
}

# Run main function
main "$@"
