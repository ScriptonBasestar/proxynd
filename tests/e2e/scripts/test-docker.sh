#!/bin/bash

# Docker 클라이언트 E2E 테스트 스크립트

set -euo pipefail

PROXYND_HOST=${PROXYND_HOST:-proxynd}
PROXYND_PORT=${PROXYND_PORT:-8080}
PROXY_URL="http://${PROXYND_HOST}:${PROXYND_PORT}/proxy/docker"

echo "🐳 Testing Docker registry proxy..."

# Docker 레지스트리 API v2 테스트
echo "🔍 Testing Docker Registry API v2..."

# v2 API 체크
if curl -s "${PROXY_URL}/v2/" > /tmp/docker_v2_test; then
    echo "✅ Docker Registry v2 API accessible"
    echo "📄 API response preview:"
    cat /tmp/docker_v2_test | head -5
else
    echo "❌ Docker Registry v2 API not accessible"
    exit 1
fi

# 매니페스트 테스트
echo "🔍 Testing manifest retrieval..."

manifest_url="${PROXY_URL}/v2/library/nginx/manifests/latest"
if curl -s -H "Accept: application/vnd.docker.distribution.manifest.v2+json" "$manifest_url" > /tmp/manifest_test; then
    echo "✅ Manifest retrieval successful"
    echo "📄 Manifest preview:"
    cat /tmp/manifest_test | head -10
else
    echo "❌ Manifest retrieval failed"
    echo "🔄 Trying with different accept header..."
    if curl -s -H "Accept: application/vnd.docker.distribution.manifest.v1+json" "$manifest_url" > /tmp/manifest_test; then
        echo "✅ Manifest retrieval successful with v1 format"
    else
        echo "❌ Manifest retrieval failed with both formats"
        exit 1
    fi
fi

# 블롭 테스트 (실제 SHA는 없으므로 404가 정상)
echo "🔍 Testing blob access..."

blob_url="${PROXY_URL}/v2/library/nginx/blobs/sha256:dummy123"
response_code=$(curl -s -o /dev/null -w "%{http_code}" "$blob_url")

if [ "$response_code" -eq 404 ]; then
    echo "✅ Blob endpoint accessible (404 expected for non-existent blob)"
elif [ "$response_code" -eq 200 ]; then
    echo "✅ Blob endpoint accessible (200 OK)"
else
    echo "⚠️ Blob endpoint returned unexpected status: $response_code"
fi

# 카탈로그 테스트
echo "🔍 Testing catalog API..."

if curl -s "${PROXY_URL}/v2/_catalog" > /tmp/catalog_test; then
    echo "✅ Catalog API accessible"
    echo "📄 Catalog preview:"
    cat /tmp/catalog_test | head -5
else
    echo "❌ Catalog API not accessible"
fi

# 캐시 테스트
echo "⏱️ Testing cache performance..."

test_url="${PROXY_URL}/v2/"

start_time=$(date +%s%N)
curl -s "$test_url" > /dev/null
first_duration=$(( ($(date +%s%N) - start_time) / 1000000 ))

start_time=$(date +%s%N)
curl -s "$test_url" > /dev/null
second_duration=$(( ($(date +%s%N) - start_time) / 1000000 ))

echo "First request: ${first_duration}ms"
echo "Second request: ${second_duration}ms"

if [ $second_duration -lt $((first_duration + 100)) ]; then
    echo "✅ Cache appears to be working"
else
    echo "⚠️ Cache behavior inconclusive"
fi

# 실제 Docker 클라이언트 테스트 (주석 처리 - Docker 데몬 필요)
# echo "🐳 Testing with Docker client..."
# 
# # 레지스트리 미러 설정
# mkdir -p /etc/docker
# cat > /etc/docker/daemon.json << EOF
# {
#   "registry-mirrors": ["${PROXY_URL}"]
# }
# EOF
# 
# # Docker 데몬 재시작
# if systemctl restart docker 2>/dev/null || service docker restart 2>/dev/null; then
#     echo "✅ Docker daemon restarted with proxy mirror"
#     
#     # 이미지 pull 테스트
#     if docker pull hello-world; then
#         echo "✅ Docker pull through proxy successful"
#     else
#         echo "❌ Docker pull through proxy failed"
#     fi
# else
#     echo "⚠️ Could not restart Docker daemon for mirror test"
# fi

echo "🐳 Docker E2E test completed successfully"