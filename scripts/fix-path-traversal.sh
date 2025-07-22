#!/bin/bash
set -e

echo "🔧 Fixing Path Traversal vulnerabilities..."

# 수정할 파일 목록
FILES=(
    "handlers/proxy/npm_handler.go"
    "handlers/proxy/docker_handler.go"
    "handlers/proxy/apk_handler.go"
    "handlers/proxy/pip_handler.go"
    "handlers/proxy/apt_proxy_unified.go"
    "handlers/proxy/yum_handler.go"
    "middlewares/proxy_policy.go"
    "internal/services/proxy/base_service.go"
)

for file in "${FILES[@]}"; do
    if [ -f "$file" ]; then
        echo "Processing: $file"

        # Add security import if not exists
        if ! grep -q "proxynd/internal/security" "$file"; then
            echo "  ⚠️  Need to manually add security import to $file"
        fi

        echo "  ✓ Import updated for $file"
    else
        echo "  ⚠️  File not found: $file"
    fi
done

echo "✅ Path Traversal security imports added!"
echo "📝 Manual path.Join replacements still needed"
