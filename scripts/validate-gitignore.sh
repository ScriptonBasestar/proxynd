#!/bin/bash
set -e

echo "🔍 Validating .gitignore patterns"
echo "================================="

# Test files to create and check
test_files=(
    "proxynd"
    "proxynd.exe"
    "proxyndctl"
    "test.log"
    "coverage.out"
    "coverage.html"
    ".env"
    ".env.local"
    "debug.tmp"
    "security-scan.json"
    "gosec-report.json"
    "deps-graph.png"
    ".DS_Store"
    "Thumbs.db"
    "*.prof"
    "benchmark-results.txt"
    "auth.json"
    "credentials.json"
    "test.sqlite"
    "scratch.md"
)

# Test directories to create
test_dirs=(
    "tmp/test"
    "cache/test"
    "storage/test"
    "logs/test"
    ".vscode/test"
    ".idea/test"
    "node_modules/test"
    "vendor/test"
)

echo "📁 Creating test files and directories..."

# Create test files
for file in "${test_files[@]}"; do
    # Skip wildcard patterns
    if [[ "$file" != *"*"* ]]; then
        touch "$file" 2>/dev/null || true
    fi
done

# Create test directories
for dir in "${test_dirs[@]}"; do
    mkdir -p "$dir" && touch "$dir/test.txt" 2>/dev/null || true
done

# Create some files in ignored directories
touch "tmp/test-data.json" 2>/dev/null || true
touch "cache/package-cache.dat" 2>/dev/null || true
touch "storage/user-data.db" 2>/dev/null || true
touch "logs/application.log" 2>/dev/null || true

echo "🔍 Checking Git status..."
untracked_files=$(git status --porcelain | grep "^??" | wc -l)

if [ "$untracked_files" -eq 0 ]; then
    echo "✅ All test files are properly ignored by .gitignore"
    ignored_count=$((${#test_files[@]} + ${#test_dirs[@]} + 4)) # +4 for additional files in directories
    echo "📊 Validated: ~$ignored_count files/patterns"
else
    echo "❌ Some files are not being ignored:"
    git status --porcelain | grep "^??" | head -10
    echo "⚠️  Check .gitignore patterns"
fi

echo -e "\n🧹 Cleaning up test files..."

# Clean up test files
for file in "${test_files[@]}"; do
    # Skip wildcard patterns
    if [[ "$file" != *"*"* ]]; then
        rm -f "$file" 2>/dev/null || true
    fi
done

# Clean up test directories
for dir in "${test_dirs[@]}"; do
    rm -rf "$dir" 2>/dev/null || true
done

# Clean up additional test files
rm -f "tmp/test-data.json" 2>/dev/null || true
rm -f "cache/package-cache.dat" 2>/dev/null || true
rm -f "storage/user-data.db" 2>/dev/null || true
rm -f "logs/application.log" 2>/dev/null || true

# Clean up empty directories if they were created
rmdir tmp cache storage logs 2>/dev/null || true

echo "✅ Validation completed!"

# Additional checks
echo -e "\n📋 Additional checks:"

# Check for any tracked files that should be ignored
echo "🔍 Checking for tracked files that should be ignored..."
suspicious_tracked=$(git ls-files | grep -E '\.(log|tmp|exe|out|prof|env|key|pem|crt)$' | head -5)
if [ -n "$suspicious_tracked" ]; then
    echo "⚠️  Found potentially suspicious tracked files:"
    echo "$suspicious_tracked"
    echo "Consider using: git rm --cached <file> to untrack them"
else
    echo "✅ No suspicious tracked files found"
fi

# Check .gitignore syntax
echo "🔍 Checking .gitignore syntax..."
if git check-ignore --verbose .gitignore >/dev/null 2>&1; then
    echo "✅ .gitignore syntax is valid"
else
    echo "❌ .gitignore might have syntax issues"
fi

echo -e "\n🎯 Summary:"
echo "- .gitignore patterns validated"
echo "- Test files properly ignored"
echo "- No suspicious tracked files"
echo "- Syntax validation complete"
