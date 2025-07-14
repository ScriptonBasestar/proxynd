#!/bin/bash
set -e

echo "🔍 Finding tracked files that should be ignored..."

# Files to remove from tracking
FILES_TO_REMOVE=(
    "tmp/main"
    "proxynd"
    "proxyndctl"
    "coverage.out"
    "coverage.html"
    "*.log"
    "*.prof"
    "*.trace"
    "*.test"
    "*.exe"
    "*.out"
    "gosec-report.json"
    "staticcheck-report.json"
    "golangci-lint-report.json"
    "deps-graph.png"
    "security-report.json"
)

found_files=false

for pattern in "${FILES_TO_REMOVE[@]}"; do
    files=$(git ls-files "$pattern" 2>/dev/null || true)
    if [ -n "$files" ]; then
        echo "Removing from Git: $files"
        git rm --cached $files 2>/dev/null || true
        found_files=true
    fi
done

if [ "$found_files" = false ]; then
    echo "✓ No unwanted files found in Git tracking"
else
    echo "✅ Git cleanup completed"
    echo "📝 Don't forget to commit these changes"
fi