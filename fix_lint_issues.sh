#!/bin/bash

# ProxyND Lint Auto-fix Script
# This script helps fix common lint issues automatically

echo "ProxyND Lint Auto-fix Script"
echo "============================"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored output
print_step() {
    echo -e "${GREEN}[STEP]${NC} $1"
}

print_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if golangci-lint is installed
if ! command -v golangci-lint &> /dev/null; then
    print_error "golangci-lint is not installed. Installing..."
    go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
fi

# Phase 1: Format code with gofmt and goimports
print_step "Phase 1: Formatting code..."
go fmt ./...
if command -v goimports &> /dev/null; then
    goimports -w .
else
    print_warn "goimports not found, skipping import formatting"
fi

# Phase 2: Run auto-fixable linters
print_step "Phase 2: Running auto-fixable linters..."
golangci-lint run --fix --disable-all --enable=whitespace,unconvert,misspell,goconst ./...

# Phase 3: Generate report for manual fixes
print_step "Phase 3: Generating lint report..."
golangci-lint run --out-format=line-number > lint_report.txt 2>&1

# Phase 4: Extract specific linter issues
print_step "Phase 4: Analyzing specific linter issues..."

echo "=== ERRCHECK Issues ===" > lint_analysis_detailed.txt
golangci-lint run --disable-all --enable=errcheck >> lint_analysis_detailed.txt 2>&1

echo -e "\n=== STATICCHECK Issues ===" >> lint_analysis_detailed.txt
golangci-lint run --disable-all --enable=staticcheck >> lint_analysis_detailed.txt 2>&1

echo -e "\n=== UNUSED Issues ===" >> lint_analysis_detailed.txt
golangci-lint run --disable-all --enable=unused >> lint_analysis_detailed.txt 2>&1

echo -e "\n=== INEFFASSIGN Issues ===" >> lint_analysis_detailed.txt
golangci-lint run --disable-all --enable=ineffassign >> lint_analysis_detailed.txt 2>&1

echo -e "\n=== GOVET Issues ===" >> lint_analysis_detailed.txt
golangci-lint run --disable-all --enable=govet >> lint_analysis_detailed.txt 2>&1

# Phase 5: Count issues by type
print_step "Phase 5: Summarizing issues..."
echo "=== Lint Issue Summary ===" > lint_summary.txt
echo "Total issues: $(grep -c ":" lint_report.txt 2>/dev/null || echo 0)" >> lint_summary.txt
echo "" >> lint_summary.txt

# Count by linter type
for linter in errcheck staticcheck unused ineffassign govet bodyclose goconst misspell unconvert whitespace gocyclo lll revive; do
    count=$(grep -c "\[$linter\]" lint_report.txt 2>/dev/null || echo 0)
    if [ $count -gt 0 ]; then
        echo "$linter: $count issues" >> lint_summary.txt
    fi
done

# Phase 6: Create fix suggestions
print_step "Phase 6: Creating fix suggestions..."
cat > lint_fix_guide.md << 'EOF'
# Lint Fix Guide

## Quick Fixes Applied
- whitespace issues
- unconvert (unnecessary conversions)
- misspell (spelling errors)
- basic formatting

## Manual Fixes Required

### 1. Error Checking (errcheck)
Look for patterns like:
- `defer file.Close()` → Add error handling
- `defer resp.Body.Close()` → Add error handling
- Unchecked function calls that return errors

### 2. Static Check Issues
- Replace `io/ioutil` with `os` or `io` packages
- Fix deprecated API usage
- Remove dead code

### 3. Unused Code
Review and remove:
- Unused variables
- Unused functions
- Unused struct fields

### 4. Shadow Variables (govet)
Look for `err` being shadowed in nested scopes

### 5. Complex Functions (gocyclo)
Refactor functions with complexity > 30

## Next Steps
1. Review `lint_report.txt` for all issues
2. Review `lint_analysis_detailed.txt` for specific linter output
3. Fix issues by priority (errors → warnings → style)
4. Run `make lint` to verify fixes
EOF

print_step "Lint analysis complete!"
echo ""
echo "Generated files:"
echo "  - lint_report.txt: Full lint report"
echo "  - lint_analysis_detailed.txt: Detailed analysis by linter"
echo "  - lint_summary.txt: Issue count summary"
echo "  - lint_fix_guide.md: Manual fix guide"
echo ""
echo "Run 'make lint' to see current status"
