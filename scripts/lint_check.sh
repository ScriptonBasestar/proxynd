#!/bin/bash

# Lint check script for ProxyND

set -e

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${YELLOW}ProxyND Linting Check${NC}"
echo "====================="

# Check if golangci-lint is installed
if ! command -v golangci-lint &> /dev/null; then
    echo -e "${RED}❌ golangci-lint is not installed${NC}"
    echo "Installing golangci-lint..."
    curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin
fi

# Get golangci-lint version
LINT_VERSION=$(golangci-lint --version | head -n 1)
echo -e "${GREEN}✓ $LINT_VERSION${NC}"

# Run linting with stats
echo -e "\n${YELLOW}Running linting checks...${NC}"

# Create temp file for output
LINT_OUTPUT=$(mktemp)

# Run golangci-lint and capture output
if golangci-lint run --out-format=tab ./... > "$LINT_OUTPUT" 2>&1; then
    echo -e "${GREEN}✅ No linting issues found!${NC}"
    ISSUES_COUNT=0
else
    ISSUES_COUNT=$(wc -l < "$LINT_OUTPUT")
    echo -e "${RED}❌ Found $ISSUES_COUNT linting issues${NC}"
fi

# Show summary by linter
if [ $ISSUES_COUNT -gt 0 ]; then
    echo -e "\n${YELLOW}Issues by linter:${NC}"
    awk -F'\t' '{print $2}' "$LINT_OUTPUT" | sort | uniq -c | sort -rn | while read count linter; do
        echo -e "  ${RED}$count${NC} $linter"
    done

    echo -e "\n${YELLOW}Top 10 issues:${NC}"
    head -n 10 "$LINT_OUTPUT"

    echo -e "\n${YELLOW}To see all issues, run:${NC}"
    echo "  make lint"
    echo -e "\n${YELLOW}To auto-fix issues, run:${NC}"
    echo "  make lint-fix"
fi

# Cleanup
rm -f "$LINT_OUTPUT"

# Show enabled linters
echo -e "\n${YELLOW}Enabled linters:${NC}"
golangci-lint linters | grep -E "^E" | wc -l | xargs echo -e "${GREEN}Total enabled:${NC}"

# Exit with appropriate code
if [ $ISSUES_COUNT -gt 0 ]; then
    exit 1
else
    exit 0
fi
