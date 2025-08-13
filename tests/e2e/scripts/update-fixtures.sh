#!/bin/bash
# 스크립트명: 테스트 픽스처 자동 업데이트 스크립트
# 용도: E2E 테스트용 업스트림 데이터 자동 업데이트
# 사용법: update-fixtures.sh [--type TYPE] [--force]
# 예시: update-fixtures.sh --type npm --force

set -euo pipefail

# Step 4-3: 테스트 픽스처 자동 업데이트 스크립트

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
UPSTREAM_DATA_DIR="${SCRIPT_DIR}/../upstream-data"
VERBOSE=${VERBOSE:-false}
FORCE=${FORCE:-false}

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
log_test() { blue "🔧 $1"; }

# 헬프 메시지
show_help() {
    echo "테스트 픽스처 자동 업데이트 스크립트"
    echo ""
    echo "Usage: $0 [options]"
    echo ""
    echo "Options:"
    echo "  --type TYPE       Update specific proxy type (npm, maven, docker, apt, pip, yum, apk)"
    echo "  --all             Update all proxy types"
    echo "  --force           Force update even if files exist"
    echo "  --dry-run         Show what would be updated without making changes"
    echo "  -v, --verbose     Verbose output"
    echo "  -h, --help        Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0 --type npm"
    echo "  $0 --all --force"
    echo "  $0 --dry-run"
}

# 인수 파싱
PROXY_TYPE=""
UPDATE_ALL=false
DRY_RUN=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --type)
            PROXY_TYPE="$2"
            shift 2
            ;;
        --all)
            UPDATE_ALL=true
            shift
            ;;
        --force)
            FORCE=true
            shift
            ;;
        --dry-run)
            DRY_RUN=true
            shift
            ;;
        -v|--verbose)
            VERBOSE=true
            shift
            ;;
        -h|--help)
            show_help
            exit 0
            ;;
        *)
            log_error "Unknown option: $1"
            show_help
            exit 1
            ;;
    esac
done

# Verbose 모드 설정
if [[ "$VERBOSE" == "true" ]]; then
    set -x
fi

log_info "Starting test fixture update..."
log_info "Upstream data directory: $UPSTREAM_DATA_DIR"

# 디렉토리 생성 함수
ensure_directory() {
    local dir="$1"
    if [[ "$DRY_RUN" == "true" ]]; then
        log_info "Would create directory: $dir"
    else
        mkdir -p "$dir"
        log_success "Directory ensured: $dir"
    fi
}

# NPM 픽스처 업데이트
update_npm_fixtures() {
    log_test "Updating NPM fixtures..."
    
    local npm_dir="$UPSTREAM_DATA_DIR/npm"
    ensure_directory "$npm_dir"
    
    local packages=(
        "express"
        "lodash"
        "react" 
        "@types/node"
        "@angular/core"
    )
    
    for package in "${packages[@]}"; do
        local package_dir="$npm_dir/$package"
        ensure_directory "$package_dir"
        
        if [[ "$DRY_RUN" == "true" ]]; then
            log_info "Would update NPM package: $package"
        else
            # 패키지 메타데이터 생성 (실제 npm registry 구조 모사)
            cat > "$package_dir/package.json" << EOF
{
  "name": "$package",
  "description": "Test fixture for $package",
  "version": "1.0.0-test",
  "main": "index.js",
  "repository": {
    "type": "git",
    "url": "https://github.com/test/$package.git"
  },
  "keywords": ["test", "fixture"],
  "license": "MIT"
}
EOF
            
            # 간단한 인덱스 파일 생성
            echo "module.exports = { name: '$package', version: '1.0.0-test' };" > "$package_dir/index.js"
            
            log_success "Updated NPM fixture: $package"
        fi
    done
    
    log_success "NPM fixtures update completed"
}

# Maven 픽스처 업데이트
update_maven_fixtures() {
    log_test "Updating Maven fixtures..."
    
    local maven_dir="$UPSTREAM_DATA_DIR/maven"
    ensure_directory "$maven_dir"
    
    # 새로운 인기 라이브러리 추가
    local artifacts=(
        "com/google/guava/guava/31.1-jre"
        "org/slf4j/slf4j-api/2.0.6"
        "com/fasterxml/jackson/core/jackson-core/2.14.2"
        "org/mockito/mockito-core/5.1.1"
    )
    
    for artifact in "${artifacts[@]}"; do
        local artifact_dir="$maven_dir/$artifact"
        ensure_directory "$artifact_dir"
        
        if [[ "$DRY_RUN" == "true" ]]; then
            log_info "Would update Maven artifact: $artifact"
        else
            local artifact_name=$(basename "$artifact")
            local version=$(basename "$(dirname "$artifact")")
            
            # POM 파일 생성
            cat > "$artifact_dir/$artifact_name.pom" << EOF
<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0">
    <modelVersion>4.0.0</modelVersion>
    <groupId>$(echo "$artifact" | cut -d'/' -f1-3 | tr '/' '.')</groupId>
    <artifactId>$artifact_name</artifactId>
    <version>$version</version>
    <packaging>jar</packaging>
</project>
EOF
            
            # 더미 JAR 파일 생성
            echo "Test JAR content for $artifact_name" > "$artifact_dir/$artifact_name.jar"
            
            log_success "Updated Maven artifact: $artifact"
        fi
    done
    
    log_success "Maven fixtures update completed"
}

# Docker 픽스처 업데이트
update_docker_fixtures() {
    log_test "Updating Docker registry fixtures..."
    
    local docker_dir="$UPSTREAM_DATA_DIR/docker"
    ensure_directory "$docker_dir"
    
    # Docker registry v2 API 응답 모사
    local registries=(
        "library/nginx"
        "library/alpine"
        "library/ubuntu"
        "library/hello-world"
    )
    
    for registry in "${registries[@]}"; do
        local registry_dir="$docker_dir/v2/$registry"
        ensure_directory "$registry_dir/manifests"
        ensure_directory "$registry_dir/blobs"
        
        if [[ "$DRY_RUN" == "true" ]]; then
            log_info "Would update Docker registry: $registry"
        else
            # 매니페스트 파일 생성
            cat > "$registry_dir/manifests/latest" << EOF
{
   "schemaVersion": 2,
   "mediaType": "application/vnd.docker.distribution.manifest.v2+json",
   "config": {
      "digest": "sha256:test-config-digest",
      "mediaType": "application/vnd.docker.container.image.v1+json",
      "size": 1234
   },
   "layers": [
      {
         "digest": "sha256:test-layer-digest-1",
         "mediaType": "application/vnd.docker.image.rootfs.diff.tar.gzip",
         "size": 5678
      }
   ]
}
EOF
            
            log_success "Updated Docker registry: $registry"
        fi
    done
    
    log_success "Docker fixtures update completed"
}

# APT 픽스처 업데이트
update_apt_fixtures() {
    log_test "Updating APT repository fixtures..."
    
    local apt_dir="$UPSTREAM_DATA_DIR/apt/ubuntu/dists/jammy"
    ensure_directory "$apt_dir/main/binary-amd64"
    ensure_directory "$apt_dir/universe/binary-amd64"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log_info "Would update APT fixtures"
    else
        # Release 파일 업데이트
        cat > "$apt_dir/Release" << EOF
Origin: Ubuntu
Label: Ubuntu
Suite: jammy
Version: 22.04
Codename: jammy
Date: $(date -u +"%a, %d %b %Y %H:%M:%S %Z")
Architectures: amd64
Components: main universe
Description: Ubuntu Jammy 22.04
EOF
        
        # Packages 파일 생성
        cat > "$apt_dir/main/binary-amd64/Packages" << EOF
Package: curl
Version: 7.81.0-1ubuntu1.4
Architecture: amd64
Maintainer: Ubuntu Developers
Section: web
Filename: pool/main/c/curl/curl_7.81.0-1ubuntu1.4_amd64.deb
Size: 194432
SHA256: test-sha256-curl
Description: command line tool for transferring data with URL syntax

Package: nginx
Version: 1.18.0-6ubuntu14.4
Architecture: amd64
Maintainer: Ubuntu Developers
Section: httpd
Filename: pool/main/n/nginx/nginx_1.18.0-6ubuntu14.4_amd64.deb
Size: 1024000
SHA256: test-sha256-nginx
Description: small, powerful, scalable web/proxy server
EOF
        
        log_success "Updated APT fixtures"
    fi
    
    log_success "APT fixtures update completed"
}

# 업데이트 실행 함수
run_updates() {
    local types_to_update=()
    
    if [[ "$UPDATE_ALL" == "true" ]]; then
        types_to_update=("npm" "maven" "docker" "apt")
    elif [[ -n "$PROXY_TYPE" ]]; then
        types_to_update=("$PROXY_TYPE")
    else
        log_error "No proxy type specified. Use --type TYPE or --all"
        show_help
        exit 1
    fi
    
    for type in "${types_to_update[@]}"; do
        case "$type" in
            npm)
                update_npm_fixtures
                ;;
            maven)
                update_maven_fixtures
                ;;
            docker)
                update_docker_fixtures
                ;;
            apt)
                update_apt_fixtures
                ;;
            *)
                log_warning "Unsupported proxy type: $type"
                ;;
        esac
    done
}

# 백업 생성 (force가 아닌 경우)
create_backup() {
    if [[ "$FORCE" == "false" && -d "$UPSTREAM_DATA_DIR" && "$DRY_RUN" == "false" ]]; then
        local backup_dir="$UPSTREAM_DATA_DIR.backup.$(date +%Y%m%d_%H%M%S)"
        log_info "Creating backup: $backup_dir"
        cp -r "$UPSTREAM_DATA_DIR" "$backup_dir"
        log_success "Backup created: $backup_dir"
    fi
}

# 메인 실행
main() {
    log_info "=== Test Fixture Update Starting ==="
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log_info "DRY RUN MODE - No changes will be made"
    fi
    
    create_backup
    run_updates
    
    if [[ "$DRY_RUN" == "false" ]]; then
        log_success "=== Test Fixture Update Completed Successfully ==="
        log_info "Updated fixtures are available in: $UPSTREAM_DATA_DIR"
    else
        log_info "=== Dry Run Completed - No Changes Made ==="
    fi
}

# 스크립트 실행
main "$@"