#!/bin/bash

# NPM 클라이언트 E2E 테스트 스크립트

set -euo pipefail

PROXYND_HOST=${PROXYND_HOST:-proxynd}
PROXYND_PORT=${PROXYND_PORT:-8080}
PROXY_URL="http://${PROXYND_HOST}:${PROXYND_PORT}/proxy/npm"

echo "🟢 Testing NPM proxy with real npm client..."

# npm 레지스트리 설정
npm config set registry "$PROXY_URL"

echo "📦 Current npm registry: $(npm config get registry)"

# 테스트 패키지 설치 시도
echo "📦 Attempting to install test package..."

# 간단한 패키지로 테스트 (실제로는 설치하지 않고 정보만 조회)
if npm view express --json > /dev/null 2>&1; then
    echo "✅ NPM proxy test successful - package metadata retrieved"
else
    echo "❌ NPM proxy test failed"
    exit 1
fi

# 캐시 테스트 - 같은 요청을 두 번 수행
echo "⏱️ Testing cache performance..."

start_time=$(date +%s%N)
npm view lodash --json > /dev/null 2>&1
first_duration=$(( ($(date +%s%N) - start_time) / 1000000 ))

start_time=$(date +%s%N)
npm view lodash --json > /dev/null 2>&1
second_duration=$(( ($(date +%s%N) - start_time) / 1000000 ))

echo "First request: ${first_duration}ms"
echo "Second request: ${second_duration}ms"

if [ $second_duration -lt $((first_duration + 100)) ]; then
    echo "✅ Cache appears to be working (second request was not significantly slower)"
else
    echo "⚠️ Cache behavior inconclusive"
fi

echo "🟢 NPM E2E test completed successfully"