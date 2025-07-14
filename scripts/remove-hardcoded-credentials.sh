#!/bin/bash
set -e

echo "🔐 Removing hardcoded credentials..."

# Create backup directory
BACKUP_DIR="backup/credentials-$(date +%Y%m%d-%H%M%S)"
mkdir -p "$BACKUP_DIR"

echo "📁 Backup directory: $BACKUP_DIR"

# Find files with potential hardcoded credentials
echo "🔍 Scanning for hardcoded credentials..."

# Search patterns for different types of credentials
PATTERNS=(
    'password\s*[:=]\s*"[^"]\+"'
    'api[_-]?key\s*[:=]\s*"[^"]\+"'
    'token\s*[:=]\s*"[^"]\+"'
    'secret\s*[:=]\s*"[^"]\+"'
    'username\s*[:=]\s*"[^"]\+"'
    'pass\s*[:=]\s*"[^"]\+"'
)

# Combine patterns with OR
COMBINED_PATTERN=$(IFS='|'; echo "${PATTERNS[*]}")

# Find files with potential issues
FILES=$(grep -rl -E "$COMBINED_PATTERN" --include="*.go" --include="*.yaml" --include="*.yml" --include="*.json" . | \
        grep -v "_test.go" | \
        grep -v ".git/" | \
        grep -v "node_modules/" | \
        grep -v "vendor/" | \
        grep -v "backup/" | \
        sort -u || true)

if [ -z "$FILES" ]; then
    echo "✅ No hardcoded credentials found!"
    exit 0
fi

echo "📄 Found files with potential credentials:"
echo "$FILES"

# Backup files before modification
for file in $FILES; do
    if [ -f "$file" ]; then
        cp "$file" "$BACKUP_DIR/"
        echo "💾 Backed up: $file"
    fi
done

echo ""
echo "🔄 Checking for specific patterns..."

# Check for common insecure patterns
echo "=== JWT Secrets ==="
grep -rn -E 'jwt[_-]?secret\s*[:=]\s*"[^"]\+"' --include="*.go" . | grep -v "_test.go" | head -5 || echo "None found"

echo "=== Database Passwords ==="  
grep -rn -E 'db[_-]?password\s*[:=]\s*"[^"]\+"' --include="*.go" . | grep -v "_test.go" | head -5 || echo "None found"

echo "=== API Keys ==="
grep -rn -E 'api[_-]?key\s*[:=]\s*"[^"]\+"' --include="*.go" . | grep -v "_test.go" | head -5 || echo "None found"

echo "=== OAuth Secrets ==="
grep -rn -E 'client[_-]?secret\s*[:=]\s*"[^"]\+"' --include="*.go" . | grep -v "_test.go" | head -5 || echo "None found"

echo ""
echo "⚠️  Manual Review Required:"
echo "1. Check each file in the backup directory"
echo "2. Replace hardcoded values with environment variables"
echo "3. Use the internal/config package for environment loading"
echo ""
echo "Example replacement:"
echo "  Before: password := \"hardcoded123\""
echo "  After:  password := config.Get().Database.Password"
echo ""
echo "📚 Documentation:"
echo "  - See .env.example for available environment variables"
echo "  - Use internal/config.Load() to initialize environment"
echo "  - Use config.Get() to access environment values"
echo ""

# Create a summary report
REPORT_FILE="$BACKUP_DIR/credentials_scan_report.txt"
echo "Credential Scan Report" > "$REPORT_FILE"
echo "Generated: $(date)" >> "$REPORT_FILE"
echo "Files scanned: $(echo "$FILES" | wc -l)" >> "$REPORT_FILE"
echo "" >> "$REPORT_FILE"
echo "Files with potential credentials:" >> "$REPORT_FILE"
echo "$FILES" >> "$REPORT_FILE"

echo "📄 Report saved: $REPORT_FILE"
echo ""
echo "✅ Credential scan completed!"
echo "🔧 Next steps:"
echo "   1. Review the files in $BACKUP_DIR"
echo "   2. Update hardcoded values to use environment variables"
echo "   3. Test the application with .env file"
echo "   4. Run security scanner to verify removal"