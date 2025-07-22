#!/bin/bash

# ProxyND 릴리스 빌드 스크립트
# 로컬에서 릴리스용 바이너리를 빌드하기 위한 스크립트

set -e

# 색상 코드
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 로깅 함수
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_step() {
    echo -e "${BLUE}[STEP]${NC} $1"
}

# 사용법 출력
usage() {
    cat << EOF
ProxyND 릴리스 빌드 스크립트

사용법: $0 [옵션]

옵션:
    -v, --version VERSION   빌드 버전 (기본값: git tag 또는 dev)
    -o, --output DIR        출력 디렉토리 (기본값: ./dist)
    -p, --platform LIST    빌드할 플랫폼 (기본값: all)
    -c, --clean             기존 빌드 결과 삭제
    -t, --test              빌드 후 테스트 실행
    -h, --help              이 도움말 출력

플랫폼 목록:
    all                     모든 플랫폼
    linux                   Linux (amd64, arm64)
    darwin                  macOS (amd64, arm64)
    windows                 Windows (amd64)
    linux-amd64            Linux amd64만
    darwin-amd64           macOS amd64만
    등...

예제:
    $0                                          # 모든 플랫폼 빌드
    $0 -v v1.0.0 -o release                   # 특정 버전으로 release 디렉토리에 빌드
    $0 -p linux -t                           # Linux만 빌드하고 테스트
    $0 -c -p darwin-amd64                     # 기존 결과 삭제 후 macOS amd64만 빌드
EOF
}

# 기본값 설정
VERSION=""
OUTPUT_DIR="./dist"
PLATFORMS="all"
CLEAN=false
TEST=false

# 명령줄 인수 파싱
while [[ $# -gt 0 ]]; do
    case $1 in
        -v|--version)
            VERSION="$2"
            shift 2
            ;;
        -o|--output)
            OUTPUT_DIR="$2"
            shift 2
            ;;
        -p|--platform)
            PLATFORMS="$2"
            shift 2
            ;;
        -c|--clean)
            CLEAN=true
            shift
            ;;
        -t|--test)
            TEST=true
            shift
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        *)
            log_error "알 수 없는 옵션: $1"
            usage
            exit 1
            ;;
    esac
done

# 플랫폼 매트릭스 정의
declare -A PLATFORM_MATRIX=(
    ["linux-amd64"]="linux amd64"
    ["linux-arm64"]="linux arm64"
    ["darwin-amd64"]="darwin amd64"
    ["darwin-arm64"]="darwin arm64"
    ["windows-amd64"]="windows amd64"
)

# 빌드할 플랫폼 목록 생성
get_build_platforms() {
    local platforms=""

    case $PLATFORMS in
        "all")
            platforms="linux-amd64 linux-arm64 darwin-amd64 darwin-arm64 windows-amd64"
            ;;
        "linux")
            platforms="linux-amd64 linux-arm64"
            ;;
        "darwin")
            platforms="darwin-amd64 darwin-arm64"
            ;;
        "windows")
            platforms="windows-amd64"
            ;;
        *)
            # 개별 플랫폼 또는 공백으로 구분된 목록
            platforms="$PLATFORMS"
            ;;
    esac

    echo "$platforms"
}

# 버전 정보 설정
setup_version() {
    if [ -z "$VERSION" ]; then
        # Git 태그에서 버전 추출
        VERSION=$(git describe --tags --exact-match 2>/dev/null || echo "")

        if [ -z "$VERSION" ]; then
            # 최신 태그 + 커밋 정보
            LATEST_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
            COMMIT_COUNT=$(git rev-list --count HEAD ^${LATEST_TAG} 2>/dev/null || echo "0")
            COMMIT_SHA=$(git rev-parse --short HEAD)

            if [ "$COMMIT_COUNT" = "0" ]; then
                VERSION="$LATEST_TAG"
            else
                VERSION="${LATEST_TAG}-dev.${COMMIT_COUNT}.${COMMIT_SHA}"
            fi
        fi
    fi

    BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    COMMIT_SHA=$(git rev-parse --short HEAD)

    log_info "버전: $VERSION"
    log_info "빌드 시간: $BUILD_TIME"
    log_info "커밋 SHA: $COMMIT_SHA"
}

# 환경 확인
check_environment() {
    log_step "환경 확인 중..."

    # Go 설치 확인
    if ! command -v go &> /dev/null; then
        log_error "Go가 설치되지 않았습니다."
        exit 1
    fi

    GO_VERSION=$(go version | cut -d' ' -f3)
    log_info "Go 버전: $GO_VERSION"

    # Git 확인
    if ! command -v git &> /dev/null; then
        log_error "Git이 설치되지 않았습니다."
        exit 1
    fi

    # 프로젝트 루트 확인
    if [ ! -f "go.mod" ] || [ ! -f "main.go" ]; then
        log_error "프로젝트 루트 디렉토리에서 실행해주세요."
        exit 1
    fi

    log_info "환경 확인 완료"
}

# 의존성 설치
install_dependencies() {
    log_step "의존성 설치 중..."
    go mod download
    go mod tidy
    log_info "의존성 설치 완료"
}

# 기존 빌드 결과 정리
clean_build() {
    if [ "$CLEAN" = true ] && [ -d "$OUTPUT_DIR" ]; then
        log_step "기존 빌드 결과 정리 중..."
        rm -rf "$OUTPUT_DIR"
        log_info "정리 완료"
    fi
}

# 바이너리 빌드
build_binary() {
    local platform=$1
    local goos=$(echo $platform | cut -d'-' -f1)
    local goarch=$(echo $platform | cut -d'-' -f2)

    log_step "빌드 중: $platform"

    # 출력 디렉토리 생성
    local platform_dir="$OUTPUT_DIR/$platform"
    mkdir -p "$platform_dir"

    # 바이너리 이름 설정
    local server_name="proxynd"
    local cli_name="proxyndctl"

    if [ "$goos" = "windows" ]; then
        server_name="proxynd.exe"
        cli_name="proxyndctl.exe"
    fi

    # 빌드 플래그
    local ldflags="-w -s -X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME} -X main.CommitSHA=${COMMIT_SHA}"

    # ProxyND 서버 빌드
    log_info "  - ProxyND 서버 빌드 중..."
    CGO_ENABLED=0 GOOS=$goos GOARCH=$goarch go build \
        -ldflags="$ldflags" \
        -trimpath \
        -o "$platform_dir/$server_name" \
        main.go

    # ProxyND CLI 빌드
    log_info "  - ProxyND CLI 빌드 중..."
    CGO_ENABLED=0 GOOS=$goos GOARCH=$goarch go build \
        -ldflags="$ldflags" \
        -trimpath \
        -o "$platform_dir/$cli_name" \
        ./cmd/proxyndctl/main.go

    # 아카이브 생성
    log_info "  - 아카이브 생성 중..."
    cd "$platform_dir"

    if [ "$goos" = "windows" ]; then
        # Windows용 ZIP
        zip "../proxynd-${VERSION}-${platform}.zip" "$server_name" "$cli_name"
        zip "../proxyndctl-${VERSION}-${platform}.zip" "$cli_name"
    else
        # Unix용 tar.gz
        tar czf "../proxynd-${VERSION}-${platform}.tar.gz" "$server_name" "$cli_name"
        tar czf "../proxyndctl-${VERSION}-${platform}.tar.gz" "$cli_name"
    fi

    cd - > /dev/null

    # 파일 크기 정보
    local server_size=$(du -h "$platform_dir/$server_name" | cut -f1)
    local cli_size=$(du -h "$platform_dir/$cli_name" | cut -f1)

    log_info "  - ProxyND 서버: $server_size"
    log_info "  - ProxyND CLI: $cli_size"
    log_info "  ✅ $platform 빌드 완료"
}

# 체크섬 생성
generate_checksums() {
    log_step "체크섬 생성 중..."

    cd "$OUTPUT_DIR"

    # 아카이브 파일들의 체크섬 생성
    sha256sum *.tar.gz *.zip > checksums.txt 2>/dev/null || true

    log_info "체크섬 파일 생성: $OUTPUT_DIR/checksums.txt"

    cd - > /dev/null
}

# 빌드 테스트
test_build() {
    if [ "$TEST" = false ]; then
        return
    fi

    log_step "빌드 테스트 중..."

    # 현재 플랫폼 확인
    local current_os=$(go env GOOS)
    local current_arch=$(go env GOARCH)
    local current_platform="${current_os}-${current_arch}"

    local platform_dir="$OUTPUT_DIR/$current_platform"

    if [ ! -d "$platform_dir" ]; then
        log_warn "현재 플랫폼($current_platform)의 빌드가 없어 테스트를 건너뜁니다."
        return
    fi

    # 바이너리 이름
    local server_name="proxynd"
    local cli_name="proxyndctl"

    if [ "$current_os" = "windows" ]; then
        server_name="proxynd.exe"
        cli_name="proxyndctl.exe"
    fi

    # 실행 권한 확인
    if [ "$current_os" != "windows" ]; then
        chmod +x "$platform_dir/$server_name"
        chmod +x "$platform_dir/$cli_name"
    fi

    # 버전 확인 테스트
    log_info "  - ProxyND 서버 버전 확인..."
    "$platform_dir/$server_name" --version || log_warn "서버 버전 확인 실패"

    log_info "  - ProxyND CLI 버전 확인..."
    "$platform_dir/$cli_name" --version || log_warn "CLI 버전 확인 실패"

    log_info "✅ 빌드 테스트 완료"
}

# 빌드 결과 요약
show_summary() {
    log_step "빌드 결과 요약"

    echo ""
    echo "📦 빌드된 파일:"
    find "$OUTPUT_DIR" -type f \( -name "*.tar.gz" -o -name "*.zip" \) -exec ls -lh {} \; | \
        awk '{printf "  %s %s\n", $9, $5}'

    echo ""
    echo "📋 체크섬 파일:"
    if [ -f "$OUTPUT_DIR/checksums.txt" ]; then
        echo "  $OUTPUT_DIR/checksums.txt"
    fi

    echo ""
    echo "🎉 빌드 완료!"
    echo "출력 디렉토리: $OUTPUT_DIR"
}

# 메인 함수
main() {
    log_info "ProxyND 릴리스 빌드 시작"

    # 환경 확인
    check_environment

    # 버전 정보 설정
    setup_version

    # 기존 빌드 정리
    clean_build

    # 의존성 설치
    install_dependencies

    # 빌드할 플랫폼 목록 가져오기
    local build_platforms=$(get_build_platforms)

    # 각 플랫폼별 빌드
    for platform in $build_platforms; do
        if [[ "${PLATFORM_MATRIX[$platform]}" ]]; then
            build_binary "$platform"
        else
            log_warn "알 수 없는 플랫폼: $platform"
        fi
    done

    # 체크섬 생성
    generate_checksums

    # 빌드 테스트
    test_build

    # 결과 요약
    show_summary
}

# 스크립트 실행
main "$@"
