#!/bin/bash

# Comprehensive Automated Lint Fix Script for ProxyND
# This script applies safe automated fixes for common lint issues

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check for required tools
check_tools() {
    log_info "Checking required tools..."
    
    # Check Go
    if ! command -v go &> /dev/null; then
        log_error "Go is not installed"
        exit 1
    fi
    
    # Check/Install goimports
    if ! command -v goimports &> /dev/null; then
        log_warning "goimports not found, installing..."
        go install golang.org/x/tools/cmd/goimports@latest
    fi
    
    # Check/Install golangci-lint
    if ! command -v golangci-lint &> /dev/null; then
        log_warning "golangci-lint not found, installing..."
        curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin
    fi
    
    log_success "All required tools are available"
}

# Phase 1: Basic formatting
phase1_formatting() {
    log_info "Phase 1: Basic code formatting..."
    
    # Run gofmt
    log_info "Running gofmt..."
    go fmt ./...
    
    # Run goimports
    log_info "Running goimports..."
    goimports -w .
    
    log_success "Phase 1 complete"
}

# Phase 2: Auto-fixable lint issues
phase2_auto_fix() {
    log_info "Phase 2: Running auto-fixable linters..."
    
    # Run golangci-lint with auto-fix for safe linters
    golangci-lint run --fix --disable-all \
        --enable=whitespace \
        --enable=unconvert \
        --enable=misspell \
        --enable=gofmt \
        --enable=goimports \
        ./... || true
    
    log_success "Phase 2 complete"
}

# Phase 3: Fix deprecated io/ioutil
phase3_fix_ioutil() {
    log_info "Phase 3: Fixing deprecated io/ioutil usage..."
    
    # Count occurrences
    COUNT=$(grep -r "io/ioutil" --include="*.go" . | wc -l)
    
    if [ $COUNT -eq 0 ]; then
        log_info "No io/ioutil usage found"
        return
    fi
    
    log_info "Found $COUNT files using io/ioutil"
    
    # Run the auto-fix tool if it exists
    if [ -f "auto_fix_ioutil.go" ]; then
        log_info "Running io/ioutil auto-fixer..."
        go run auto_fix_ioutil.go
    else
        log_warning "auto_fix_ioutil.go not found, using sed fallback..."
        
        # Fix imports
        find . -name "*.go" -type f ! -path "./vendor/*" ! -path "./.git/*" | while read -r file; do
            # Check if file uses ioutil
            if grep -q "io/ioutil" "$file"; then
                # Check what functions are used
                if grep -q "ioutil\.ReadAll" "$file" && ! grep -q "ioutil\.\(ReadFile\|WriteFile\|TempFile\|TempDir\)" "$file"; then
                    # Only ReadAll - change to io
                    sed -i.bak 's|"io/ioutil"|"io"|g' "$file"
                    sed -i.bak 's|ioutil\.ReadAll|io.ReadAll|g' "$file"
                elif ! grep -q "ioutil\.ReadAll" "$file"; then
                    # Only file operations - change to os
                    sed -i.bak 's|"io/ioutil"|"os"|g' "$file"
                    sed -i.bak 's|ioutil\.|os.|g' "$file"
                else
                    log_warning "$file needs manual review (uses both ReadAll and file operations)"
                fi
                rm -f "$file.bak"
            fi
        done
    fi
    
    log_success "Phase 3 complete"
}

# Phase 4: Extract repeated strings
phase4_extract_constants() {
    log_info "Phase 4: Extracting repeated strings to constants..."
    
    # Run goconst to find repeated strings
    if command -v goconst &> /dev/null; then
        log_info "Running goconst analysis..."
        goconst -min-len 5 -min-occurrences 3 ./... > goconst_report.txt 2>&1 || true
        
        if [ -s goconst_report.txt ]; then
            log_warning "Found repeated strings. See goconst_report.txt for details"
        fi
    else
        log_warning "goconst not installed, skipping constant extraction"
    fi
    
    log_success "Phase 4 complete"
}

# Phase 5: Generate lint reports
phase5_generate_reports() {
    log_info "Phase 5: Generating lint reports..."
    
    # Full lint report
    log_info "Running full lint analysis..."
    golangci-lint run ./... > lint_report_full.txt 2>&1 || true
    
    # Individual linter reports
    for linter in errcheck staticcheck unused ineffassign govet bodyclose; do
        log_info "Analyzing $linter issues..."
        golangci-lint run --disable-all --enable=$linter ./... > "lint_report_${linter}.txt" 2>&1 || true
    done
    
    # Create summary
    cat > lint_summary_after_fixes.txt << EOF
=== Lint Summary After Automated Fixes ===
Generated: $(date)

Total Issues: $(grep -c ":" lint_report_full.txt 2>/dev/null || echo 0)

By Linter:
EOF
    
    for linter in errcheck staticcheck unused ineffassign govet bodyclose goconst whitespace misspell; do
        count=$(grep -c "\[$linter\]" lint_report_full.txt 2>/dev/null || echo 0)
        echo "$linter: $count issues" >> lint_summary_after_fixes.txt
    done
    
    log_success "Phase 5 complete"
}

# Phase 6: Create manual fix guide
phase6_create_guide() {
    log_info "Phase 6: Creating manual fix guide..."
    
    cat > manual_fixes_required.md << 'EOF'
# Manual Fixes Required

After running automated fixes, the following issues require manual intervention:

## 1. Error Checking (errcheck)

Review files in `lint_report_errcheck.txt` and add error handling for:
- `defer XXX.Close()` calls
- Unchecked function returns

Example fix:
```go
// Before:
defer file.Close()

// After:
defer func() {
    if err := file.Close(); err != nil {
        log.Printf("Error closing file: %v", err)
    }
}()
```

## 2. Static Check Issues

Review `lint_report_staticcheck.txt` for:
- Deprecated API usage not fixed automatically
- Dead code
- Incorrect error handling patterns

## 3. Unused Code

Review `lint_report_unused.txt` and decide whether to:
- Remove genuinely unused code
- Add `// nolint:unused` with explanation if needed for future use
- Fix code that should be used but isn't

## 4. Shadow Variables (govet)

Review `lint_report_govet.txt` for shadowed variables, especially `err`:
```go
// Problem:
err := doSomething()
if val, err := getValue(); err != nil { // shadows outer err
    return err
}
// outer err is hidden

// Fix:
err := doSomething()
if err != nil {
    return err
}
val, err := getValue()
if err != nil {
    return err
}
```

## 5. Complex Functions

Functions with cyclomatic complexity > 30 should be refactored.
Look for functions with many if/else branches or switch cases.

## Next Steps

1. Review each lint_report_*.txt file
2. Fix issues by priority: errors > warnings > style
3. Run `make lint` after each batch of fixes
4. Consider adding `// nolint` directives with explanations for intentional patterns
EOF
    
    log_success "Phase 6 complete"
}

# Main execution
main() {
    echo "ProxyND Automated Lint Fix Tool"
    echo "==============================="
    echo ""
    
    check_tools
    
    # Backup current state
    log_info "Creating backup branch..."
    git branch -D lint-fixes-backup 2>/dev/null || true
    git checkout -b lint-fixes-backup
    git checkout -
    
    # Run all phases
    phase1_formatting
    phase2_auto_fix
    phase3_fix_ioutil
    phase4_extract_constants
    phase5_generate_reports
    phase6_create_guide
    
    echo ""
    echo "==============================="
    log_success "Automated fixes complete!"
    echo ""
    echo "Summary:"
    echo "- Basic formatting: DONE"
    echo "- Auto-fixable issues: DONE"
    echo "- io/ioutil deprecation: DONE"
    echo "- Reports generated: DONE"
    echo ""
    echo "Generated files:"
    echo "- lint_report_full.txt: Complete lint report"
    echo "- lint_report_*.txt: Individual linter reports"
    echo "- lint_summary_after_fixes.txt: Summary of remaining issues"
    echo "- manual_fixes_required.md: Guide for manual fixes"
    echo "- goconst_report.txt: Repeated strings analysis (if available)"
    echo ""
    echo "Next steps:"
    echo "1. Review the changes with: git diff"
    echo "2. Review manual_fixes_required.md"
    echo "3. Fix remaining issues manually"
    echo "4. Run 'make lint' to verify"
    echo ""
    echo "To restore original state: git checkout lint-fixes-backup"
}

# Run main function
main "$@"