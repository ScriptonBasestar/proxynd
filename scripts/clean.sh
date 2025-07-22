#!/bin/bash
set -e

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

echo "🧹 ProxyND Build Artifacts Cleanup Script"
echo "========================================="

# Function to clean with confirmation
clean_with_confirm() {
    local target=$1
    local description=$2

    if [ -e "$target" ] || [ -d "$target" ]; then
        echo -n "Remove $description? (y/N) "
        read -r response
        if [[ "$response" =~ ^[Yy]$ ]]; then
            rm -rf "$target"
            echo "✓ Removed $description"
        else
            echo "⏭  Skipped $description"
        fi
    fi
}

# Function to clean without confirmation
clean_silent() {
    local target=$1
    if [ -e "$target" ] || [ -d "$target" ]; then
        rm -rf "$target"
    fi
}

cd "$PROJECT_ROOT"

echo -e "\n📦 Cleaning build artifacts..."
clean_silent "proxynd"
clean_silent "proxyndctl"
clean_silent "*.exe"
clean_silent "*.out"
clean_silent "tmp/main"
find . -name "*.test" -type f -delete 2>/dev/null || true
echo "✓ Build artifacts cleaned"

echo -e "\n📊 Cleaning test coverage files..."
clean_silent "coverage.out"
clean_silent "coverage.html"
clean_silent "coverage.txt"
clean_silent "*.prof"
clean_silent "*.trace"
clean_silent "*.pprof"
echo "✓ Coverage files cleaned"

echo -e "\n🔍 Cleaning analysis reports..."
clean_silent "gosec-report.json"
clean_silent "deps-graph.png"
clean_silent "security-report.json"
clean_silent "staticcheck-report.json"
clean_silent "golangci-lint-report.json"
echo "✓ Analysis reports cleaned"

echo -e "\n🗄️ Cleaning caches..."
go clean -cache -testcache -modcache 2>/dev/null || true
echo "✓ Go caches cleaned"

echo -e "\n📝 Cleaning logs (preserving structure)..."
find logs -name "*.log" -type f -exec truncate -s 0 {} \; 2>/dev/null || true
find tests -name "*.log" -type f -exec truncate -s 0 {} \; 2>/dev/null || true
find integration -name "*.log" -type f -exec truncate -s 0 {} \; 2>/dev/null || true
echo "✓ Log files truncated"

echo -e "\n🗑️ Cleaning temporary files..."
find . -name "*.tmp" -o -name "*.temp" -type f -delete 2>/dev/null || true
clean_with_confirm "tmp/storage" "temporary storage"
clean_with_confirm "cache/packages" "package cache"
echo "✓ Temporary files cleaned"

echo -e "\n💾 Disk usage summary:"
du -sh . 2>/dev/null || echo "Unable to calculate"

echo -e "\n✅ Cleanup completed!"
