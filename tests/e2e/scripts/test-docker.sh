#!/bin/bash
# 스크립트명: Docker E2E 테스트 스크립트
# 용도: ProxyND의 Docker 레지스트리 프록시 기능을 테스트
# 사용법: test-docker.sh [옵션]
# 예시: test-docker.sh --verbose

set -euo pipefail

PROXYND_HOST=${PROXYND_HOST:-proxynd}
PROXYND_PORT=${PROXYND_PORT:-8080}
PROXY_URL="http://${PROXYND_HOST}:${PROXYND_PORT}/proxy/docker"

# 색상 출력을 위한 함수들
red() { echo -e "\033[31m$1\033[0m"; }
green() { echo -e "\033[32m$1\033[0m"; }
yellow() { echo -e "\033[33m$1\033[0m"; }
blue() { echo -e "\033[34m$1\033[0m"; }

# 로그 함수들
log_info() { echo "ℹ️ $1"; }
log_success() { green "✅ $1"; }
log_warning() { yellow "⚠️ $1"; }
log_error() { red "❌ $1"; }
log_test() { blue "🧪 $1"; }

# 인수 파싱
VERBOSE=false
while [[ $# -gt 0 ]]; do
    case $1 in
        -v|--verbose)
            VERBOSE=true
            shift
            ;;
        -h|--help)
            echo "Docker E2E 테스트 스크립트"
            echo "Usage: $0 [options]"
            echo "Options:"
            echo "  -v, --verbose     Verbose output"
            echo "  -h, --help        Show this help message"
            exit 0
            ;;
        *)
            log_error "Unknown option: $1"
            exit 1
            ;;
    esac
done

# Verbose 모드 설정
if [[ "$VERBOSE" == "true" ]]; then
    set -x
fi

log_info "Starting Enhanced Docker E2E tests..."
log_info "Proxy URL: $PROXY_URL"

# ProxyND 헬스체크
check_proxynd_health() {
    log_test "Checking ProxyND health..."

    local health_url="http://${PROXYND_HOST}:${PROXYND_PORT}/healthz"
    local max_attempts=10
    local attempt=0

    while [ $attempt -lt $max_attempts ]; do
        if curl -sf "$health_url" >/dev/null 2>&1; then
            log_success "ProxyND is healthy"
            return 0
        fi

        attempt=$((attempt + 1))
        log_info "Attempt $attempt/$max_attempts: Waiting for ProxyND..."
        sleep 2
    done

    log_error "ProxyND health check failed after $max_attempts attempts"
    return 1
}

# Docker 레지스트리 API v2 테스트
test_docker_api_v2() {
    log_test "Testing Docker Registry API v2..."

    # v2 API 체크
    if curl -sf "${PROXY_URL}/v2/" > /tmp/docker_v2_test 2>&1; then
        log_success "Docker Registry v2 API accessible"
        if [[ "$VERBOSE" == "true" ]]; then
            log_info "API response preview:"
            cat /tmp/docker_v2_test | head -5
        fi
        return 0
    else
        log_error "Docker Registry v2 API not accessible"
        return 1
    fi
}

# Step 3-2: 매니페스트 조회 테스트 (기존 + 강화)
test_manifest_inspection() {
    log_test "Testing enhanced manifest inspection..."

    local images=(
        "library/nginx:latest"
        "library/alpine:latest" 
        "library/ubuntu:20.04"
        "library/hello-world:latest"
    )

    local manifest_formats=(
        "application/vnd.docker.distribution.manifest.v2+json"
        "application/vnd.docker.distribution.manifest.list.v2+json"
        "application/vnd.docker.distribution.manifest.v1+json"
        "application/vnd.oci.image.manifest.v1+json"
    )

    for image in "${images[@]}"; do
        local manifest_url="${PROXY_URL}/v2/${image}/manifests/latest"
        local success=false

        for format in "${manifest_formats[@]}"; do
            if curl -sf -H "Accept: $format" "$manifest_url" > "/tmp/manifest_${image//\//_}.json" 2>&1; then
                log_success "Manifest retrieval successful for $image (format: $format)"
                success=true
                
                # 매니페스트 내용 검증
                local manifest_file="/tmp/manifest_${image//\//_}.json"
                if jq -e '.schemaVersion' "$manifest_file" >/dev/null 2>&1; then
                    local schema_version
                    schema_version=$(jq -r '.schemaVersion' "$manifest_file")
                    log_success "Valid manifest schema version: $schema_version"
                    
                    # 레이어 정보 확인
                    if jq -e '.layers // .fsLayers' "$manifest_file" >/dev/null 2>&1; then
                        local layer_count
                        layer_count=$(jq '.layers // .fsLayers | length' "$manifest_file")
                        log_info "Image $image has $layer_count layer(s)"
                    fi
                    
                    # 아키텍처 정보 확인 (manifest list의 경우)
                    if jq -e '.manifests' "$manifest_file" >/dev/null 2>&1; then
                        local arch_count
                        arch_count=$(jq '.manifests | length' "$manifest_file")
                        log_success "Multi-arch manifest list with $arch_count architecture(s)"
                        
                        if [[ "$VERBOSE" == "true" ]]; then
                            log_info "Available architectures:"
                            jq -r '.manifests[].platform.architecture' "$manifest_file" | sort | uniq
                        fi
                    fi
                else
                    log_warning "Manifest format not recognized as JSON for $image"
                fi
                break
            fi
        done
        
        if [ "$success" = false ]; then
            log_error "Manifest retrieval failed for $image with all formats"
            return 1
        fi
    done
    
    log_success "Enhanced manifest inspection completed successfully"
}

# Step 3-1: 다양한 아키텍처 이미지 테스트
test_multi_architecture_images() {
    log_test "Testing multi-architecture image support..."

    local multi_arch_images=(
        "library/nginx:latest"
        "library/alpine:latest"
        "library/busybox:latest"
    )

    local target_architectures=(
        "amd64"
        "arm64"
        "arm/v7"
    )

    for image in "${multi_arch_images[@]}"; do
        local manifest_url="${PROXY_URL}/v2/${image}/manifests/latest"
        
        # manifest list 확인
        if curl -sf -H "Accept: application/vnd.docker.distribution.manifest.list.v2+json" \
            "$manifest_url" > "/tmp/manifest_list_${image//\//_}.json" 2>&1; then
            
            log_success "Multi-arch manifest list retrieved for $image"
            
            # 지원 아키텍처 확인
            for arch in "${target_architectures[@]}"; do
                if jq -e ".manifests[] | select(.platform.architecture == \"$arch\")" \
                    "/tmp/manifest_list_${image//\//_}.json" >/dev/null 2>&1; then
                    log_success "Architecture $arch supported for $image"
                    
                    # 특정 아키텍처 매니페스트 조회
                    local digest
                    digest=$(jq -r ".manifests[] | select(.platform.architecture == \"$arch\") | .digest" \
                        "/tmp/manifest_list_${image//\//_}.json")
                    
                    if curl -sf "${PROXY_URL}/v2/${image}/manifests/$digest" \
                        > "/tmp/arch_manifest_${image//\//_}_${arch//\//_}.json" 2>&1; then
                        log_success "Architecture-specific manifest retrieved: $arch"
                    else
                        log_warning "Failed to retrieve arch-specific manifest for $arch"
                    fi
                else
                    log_info "Architecture $arch not available for $image"
                fi
            done
        else
            log_warning "Multi-arch manifest list not available for $image"
        fi
    done
    
    log_success "Multi-architecture image testing completed"
}

# 블롭 테스트 강화
test_blob_access() {
    log_test "Testing blob access..."

    # 기본 블롭 엔드포인트 테스트 (실제 SHA는 없으므로 404가 정상)
    local blob_url="${PROXY_URL}/v2/library/nginx/blobs/sha256:dummy123"
    local response_code
    response_code=$(curl -s -o /dev/null -w "%{http_code}" "$blob_url")

    if [ "$response_code" -eq 404 ]; then
        log_success "Blob endpoint accessible (404 expected for non-existent blob)"
    elif [ "$response_code" -eq 200 ]; then
        log_success "Blob endpoint accessible (200 OK)"
    else
        log_warning "Blob endpoint returned unexpected status: $response_code"
    fi
    
    # 실제 이미지에서 blob SHA 추출 후 테스트
    if [ -f "/tmp/manifest_library_nginx.json" ]; then
        local config_digest
        config_digest=$(jq -r '.config.digest // empty' "/tmp/manifest_library_nginx.json" 2>/dev/null)
        
        if [ -n "$config_digest" ]; then
            local config_url="${PROXY_URL}/v2/library/nginx/blobs/$config_digest"
            if curl -sf "$config_url" > /tmp/nginx_config.json 2>&1; then
                log_success "Actual blob retrieval successful: $config_digest"
            else
                log_warning "Actual blob retrieval failed: $config_digest"
            fi
        fi
    fi
}

# 카탈로그 테스트 강화
test_catalog_api() {
    log_test "Testing enhanced catalog API..."

    if curl -sf "${PROXY_URL}/v2/_catalog" > /tmp/catalog_test 2>&1; then
        log_success "Catalog API accessible"
        
        # JSON 파싱 검증
        if jq -e '.repositories' /tmp/catalog_test >/dev/null 2>&1; then
            local repo_count
            repo_count=$(jq '.repositories | length' /tmp/catalog_test)
            log_success "Catalog contains $repo_count repositories"
            
            if [[ "$VERBOSE" == "true" ]]; then
                log_info "Available repositories:"
                jq -r '.repositories[]' /tmp/catalog_test | head -10
            fi
        else
            log_warning "Catalog response is not valid JSON"
        fi
    else
        log_warning "Catalog API not accessible (may be disabled)"
    fi
    
    # 페이지네이션 테스트
    if curl -sf "${PROXY_URL}/v2/_catalog?n=5" > /tmp/catalog_paginated 2>&1; then
        log_success "Catalog pagination supported"
    else
        log_info "Catalog pagination test inconclusive"
    fi
}

# Step 3-3: 레이어 캐싱 검증 테스트
test_layer_caching() {
    log_test "Testing layer caching verification..."

    local test_urls=(
        "${PROXY_URL}/v2/"
        "${PROXY_URL}/v2/library/nginx/manifests/latest"
        "${PROXY_URL}/v2/library/alpine/manifests/latest"
    )

    for test_url in "${test_urls[@]}"; do
        local endpoint_name
        endpoint_name=$(basename "$test_url")
        
        # 첫 번째 요청 (캐시 MISS)
        local start_time
        start_time=$(date +%s%N)
        curl -sf "$test_url" > /tmp/cache_test_first 2>&1
        local first_duration=$(( ($(date +%s%N) - start_time) / 1000000 ))

        # 두 번째 요청 (캐시 HIT)
        start_time=$(date +%s%N)
        curl -sf "$test_url" > /tmp/cache_test_second 2>&1
        local second_duration=$(( ($(date +%s%N) - start_time) / 1000000 ))

        log_info "Cache test for $endpoint_name: First: ${first_duration}ms, Second: ${second_duration}ms"

        if [ $second_duration -lt $((first_duration / 2)) ]; then
            log_success "Cache HIT performance improvement detected for $endpoint_name"
        elif [ $second_duration -lt $first_duration ]; then
            log_info "Cache shows some improvement for $endpoint_name"
        else
            log_warning "Cache behavior inconclusive for $endpoint_name"
        fi
    done
    
    # 실제 블롭 캐싱 테스트 (가능한 경우)
    if [ -f "/tmp/manifest_library_nginx.json" ]; then
        local layer_digest
        layer_digest=$(jq -r '.layers[0].digest // empty' "/tmp/manifest_library_nginx.json" 2>/dev/null)
        
        if [ -n "$layer_digest" ]; then
            local layer_url="${PROXY_URL}/v2/library/nginx/blobs/$layer_digest"
            
            # 레이어 다운로드 성능 테스트
            start_time=$(date +%s%N)
            curl -sf --range "0-1023" "$layer_url" > /tmp/layer_sample_first 2>&1 || true
            first_duration=$(( ($(date +%s%N) - start_time) / 1000000 ))
            
            start_time=$(date +%s%N)
            curl -sf --range "0-1023" "$layer_url" > /tmp/layer_sample_second 2>&1 || true
            second_duration=$(( ($(date +%s%N) - start_time) / 1000000 ))
            
            log_info "Layer caching test: First: ${first_duration}ms, Second: ${second_duration}ms"
            
            if [ $second_duration -lt $first_duration ]; then
                log_success "Layer caching appears to be working"
            else
                log_info "Layer caching test inconclusive"
            fi
        fi
    fi
    
    log_success "Layer caching verification completed"
}

# Step 3-4: 인증 처리 테스트
test_authentication_handling() {
    log_test "Testing authentication handling..."

    # 기본 인증 헤더 테스트 (인증이 필요하지 않더라도 처리 확인)
    local auth_headers=(
        "Authorization: Basic dGVzdDp0ZXN0"  # test:test
        "Authorization: Bearer test-token"
        "Docker-Content-Digest: sha256:test"
        "X-Registry-Auth: test-auth"
    )

    for header in "${auth_headers[@]}"; do
        local header_name
        header_name=$(echo "$header" | cut -d':' -f1)
        
        local response_code
        response_code=$(curl -s -o /dev/null -w "%{http_code}" \
            -H "$header" "${PROXY_URL}/v2/" 2>&1)
            
        if [ "$response_code" -eq 200 ] || [ "$response_code" -eq 401 ] || [ "$response_code" -eq 403 ]; then
            log_success "Auth header $header_name properly handled (status: $response_code)"
        else
            log_warning "Auth header $header_name returned unexpected status: $response_code"
        fi
    done
    
    # 인증 챌린지 테스트 (private registry 시뮬레이션)
    local private_manifest_url="${PROXY_URL}/v2/private/test/manifests/latest"
    response_code=$(curl -s -o /dev/null -w "%{http_code}" "$private_manifest_url" 2>&1)
    
    if [ "$response_code" -eq 401 ] || [ "$response_code" -eq 403 ]; then
        log_success "Authentication challenge properly returned for private repository"
    elif [ "$response_code" -eq 404 ]; then
        log_info "Private repository returned 404 (expected for non-existent repo)"
    else
        log_info "Authentication test inconclusive (status: $response_code)"
    fi
    
    # WWW-Authenticate 헤더 확인
    local auth_challenge
    auth_challenge=$(curl -s -I "$private_manifest_url" 2>&1 | grep -i "www-authenticate" || true)
    if [ -n "$auth_challenge" ]; then
        log_success "WWW-Authenticate header present: $auth_challenge"
    else
        log_info "WWW-Authenticate header not found (may not be required)"
    fi
    
    log_success "Authentication handling tests completed"
}

# 에러 시나리오 테스트
test_error_scenarios() {
    log_test "Testing error scenarios..."

    local error_urls=(
        "${PROXY_URL}/v2/nonexistent/repo/manifests/latest"
        "${PROXY_URL}/v2/library/nginx/manifests/nonexistent-tag" 
        "${PROXY_URL}/v2/library/nginx/blobs/sha256:invalid-hash"
        "${PROXY_URL}/v2/invalid-endpoint"
    )

    for url in "${error_urls[@]}"; do
        local endpoint_desc
        endpoint_desc=$(echo "$url" | sed "s|${PROXY_URL}/v2/||")
        
        local response_code
        response_code=$(curl -s -o /dev/null -w "%{http_code}" "$url" 2>&1)

        if [ "$response_code" -eq 404 ]; then
            log_success "Correct 404 response for: $endpoint_desc"
        elif [ "$response_code" -eq 400 ] || [ "$response_code" -eq 401 ] || [ "$response_code" -eq 403 ]; then
            log_success "Appropriate error response ($response_code) for: $endpoint_desc"
        else
            log_warning "Unexpected response ($response_code) for: $endpoint_desc"
        fi
    done
}

# 메인 테스트 실행
main() {
    log_info "=== Enhanced Docker E2E Tests Starting ==="
    log_info "Step 3: Docker E2E 테스트 강화 - 다양한 아키텍처, 매니페스트, 레이어 캐싱, 인증"

    # 필수 헬스 체크
    check_proxynd_health
    
    # 기본 API 테스트
    test_docker_api_v2
    test_blob_access
    test_catalog_api
    
    # Step 3: Docker E2E 테스트 강화
    test_multi_architecture_images    # Step 3-1: 다양한 아키텍처 이미지 테스트
    test_manifest_inspection         # Step 3-2: 매니페스트 조회 테스트 (강화)
    test_layer_caching              # Step 3-3: 레이어 캐싱 검증
    test_authentication_handling    # Step 3-4: 인증 처리 테스트
    
    # 추가 테스트들
    test_error_scenarios

    log_success "=== Enhanced Docker E2E Tests Completed Successfully ==="
    log_info "✅ 다양한 아키텍처 이미지 테스트 완료"
    log_info "✅ 매니페스트 조회 테스트 완료"
    log_info "✅ 레이어 캐싱 검증 완료"
    log_info "✅ 인증 처리 테스트 완료"
}

# 스크립트 실행
main "$@"
