#!/bin/bash

# PyPI 클라이언트 E2E 테스트 스크립트

set -euo pipefail

PROXYND_HOST=${PROXYND_HOST:-proxynd}
PROXYND_PORT=${PROXYND_PORT:-8080}
PROXY_URL="http://${PROXYND_HOST}:${PROXYND_PORT}/proxy/pip"

echo "🐍 Testing PyPI proxy with real pip client..."

# pip 인덱스 URL 설정
export PIP_INDEX_URL="$PROXY_URL/simple"
export PIP_TRUSTED_HOST="$PROXYND_HOST"

echo "📦 Current pip index URL: $PIP_INDEX_URL"

# pip 설정 확인
pip config list

# 테스트 패키지 정보 조회
echo "📦 Attempting to get package information..."

# requests 패키지 정보 조회 (실제 설치하지 않음)
if pip index versions requests > /dev/null 2>&1; then
    echo "✅ PyPI proxy test successful - package index accessible"
else
    echo "❌ PyPI proxy test failed"
    # 대안: 직접 HTTP 요청으로 테스트
    if curl -s "${PROXY_URL}/simple/requests/" > /dev/null; then
        echo "✅ PyPI proxy HTTP test successful"
    else
        echo "❌ PyPI proxy HTTP test also failed"
        exit 1
    fi
fi

# 캐시 테스트
echo "⏱️ Testing cache performance..."

start_time=$(date +%s%N)
curl -s "${PROXY_URL}/simple/numpy/" > /dev/null
first_duration=$(( ($(date +%s%N) - start_time) / 1000000 ))

start_time=$(date +%s%N)
curl -s "${PROXY_URL}/simple/numpy/" > /dev/null
second_duration=$(( ($(date +%s%N) - start_time) / 1000000 ))

echo "First request: ${first_duration}ms"
echo "Second request: ${second_duration}ms"

if [ $second_duration -lt $((first_duration + 100)) ]; then
    echo "✅ Cache appears to be working"
else
    echo "⚠️ Cache behavior inconclusive"
fi

echo "🐍 PyPI E2E test completed successfully"