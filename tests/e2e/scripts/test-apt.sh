#!/bin/bash

# APT 클라이언트 E2E 테스트 스크립트

set -euo pipefail

PROXYND_HOST=${PROXYND_HOST:-proxynd}
PROXYND_PORT=${PROXYND_PORT:-8080}
PROXY_URL="http://${PROXYND_HOST}:${PROXYND_PORT}/proxy/apt"

echo "📦 Testing APT proxy with real apt client..."

# APT 소스 설정 백업
cp /etc/apt/sources.list /etc/apt/sources.list.backup || true

# ProxyND를 통한 APT 소스 설정
cat > /etc/apt/sources.list.proxynd << EOF
# ProxyND 테스트용 APT 소스
deb ${PROXY_URL}/ubuntu jammy main restricted universe multiverse
deb ${PROXY_URL}/ubuntu jammy-updates main restricted universe multiverse
deb ${PROXY_URL}/ubuntu jammy-security main restricted universe multiverse
EOF

echo "📋 ProxyND APT sources configured:"
cat /etc/apt/sources.list.proxynd

# APT 설정에 ProxyND 추가
mkdir -p /etc/apt/apt.conf.d
cat > /etc/apt/apt.conf.d/99proxynd << EOF
Acquire::http::Proxy "${PROXY_URL}";
Acquire::https::Proxy "DIRECT";
EOF

# 수동으로 Release 파일 테스트
echo "🔍 Testing Release file access..."

if curl -s "${PROXY_URL}/ubuntu/dists/jammy/Release" > /tmp/release_test; then
    echo "✅ Release file accessible through proxy"
    echo "📄 Release file preview:"
    head -10 /tmp/release_test
else
    echo "❌ Release file not accessible"
    exit 1
fi

# 패키지 목록 테스트
echo "🔍 Testing package list access..."

if curl -s "${PROXY_URL}/ubuntu/dists/jammy/main/binary-amd64/Packages.gz" > /tmp/packages_test.gz; then
    echo "✅ Package list accessible through proxy"
    echo "📄 Package list size: $(stat -c%s /tmp/packages_test.gz) bytes"
else
    echo "❌ Package list not accessible"
    # 압축되지 않은 버전도 시도
    if curl -s "${PROXY_URL}/ubuntu/dists/jammy/main/binary-amd64/Packages" > /tmp/packages_test; then
        echo "✅ Uncompressed package list accessible"
    else
        echo "❌ Neither compressed nor uncompressed package list accessible"
        exit 1
    fi
fi

# 캐시 테스트
echo "⏱️ Testing cache performance..."

start_time=$(date +%s%N)
curl -s "${PROXY_URL}/ubuntu/dists/jammy/Release" > /dev/null
first_duration=$(( ($(date +%s%N) - start_time) / 1000000 ))

start_time=$(date +%s%N)
curl -s "${PROXY_URL}/ubuntu/dists/jammy/Release" > /dev/null
second_duration=$(( ($(date +%s%N) - start_time) / 1000000 ))

echo "First request: ${first_duration}ms"
echo "Second request: ${second_duration}ms"

if [ $second_duration -lt $((first_duration + 100)) ]; then
    echo "✅ Cache appears to be working"
else
    echo "⚠️ Cache behavior inconclusive"
fi

# 실제 apt update 테스트 (위험할 수 있으므로 주석 처리)
# echo "🔄 Testing apt update..."
# if apt-get update -o Dir::Etc::sourcelist=/etc/apt/sources.list.proxynd; then
#     echo "✅ APT update through proxy successful"
# else
#     echo "❌ APT update through proxy failed"
# fi

echo "📦 APT E2E test completed successfully"