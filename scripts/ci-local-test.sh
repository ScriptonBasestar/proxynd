#!/bin/bash

# ProxyND 로컬 CI 테스트 스크립트
# CI 파이프라인을 로컬에서 검증하기 위한 스크립트

set -e

# 색상 설정
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# 헬퍼 함수
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

# 시작
echo -e "${GREEN}=== ProxyND 로컬 CI 테스트 시작 ===${NC}"
echo ""

# Go 버전 확인
log_info "Go 버전 확인..."
go version
echo ""

# 1. 의존성 설치
log_info "의존성 설치..."
go mod download
go mod verify
log_success "의존성 설치 완료"
echo ""

# 2. 코드 포맷 검사
log_info "코드 포맷 검사..."
fmt_files=$(gofmt -l .)
if [ -n "$fmt_files" ]; then
    log_error "다음 파일들이 포맷팅이 필요합니다:"
    echo "$fmt_files"
    exit 1
else
    log_success "코드 포맷 검사 통과"
fi
echo ""

# 3. go vet 실행
log_info "go vet 실행..."
if go vet ./...; then
    log_success "go vet 통과"
else
    log_error "go vet 실패"
    exit 1
fi
echo ""

# 4. golangci-lint 실행 (설치되어 있는 경우)
if command -v golangci-lint &> /dev/null; then
    log_info "golangci-lint 실행..."
    if golangci-lint run --timeout=5m; then
        log_success "golangci-lint 통과"
    else
        log_warning "golangci-lint 경고/오류 발견"
    fi
else
    log_warning "golangci-lint가 설치되어 있지 않습니다. 건너뜁니다."
fi
echo ""

# 5. 단위 테스트
log_info "단위 테스트 실행..."
if go test -v -race -coverprofile=coverage.txt -covermode=atomic ./...; then
    log_success "단위 테스트 통과"
    
    # 커버리지 출력
    log_info "테스트 커버리지:"
    go tool cover -func=coverage.txt | tail -1
else
    log_error "단위 테스트 실패"
    exit 1
fi
echo ""

# 6. 빌드 테스트
log_info "빌드 테스트..."
platforms=(
    "linux/amd64"
    "linux/arm64"
    "darwin/amd64"
    "darwin/arm64"
    "windows/amd64"
)

for platform in "${platforms[@]}"; do
    GOOS=${platform%/*}
    GOARCH=${platform#*/}
    
    log_info "빌드 중: $GOOS/$GOARCH"
    
    output_name="proxynd-test-$GOOS-$GOARCH"
    if [ "$GOOS" = "windows" ]; then
        output_name="${output_name}.exe"
    fi
    
    if CGO_ENABLED=0 GOOS=$GOOS GOARCH=$GOARCH go build -o /tmp/$output_name main.go; then
        log_success "$platform 빌드 성공"
        rm -f /tmp/$output_name
    else
        log_error "$platform 빌드 실패"
        exit 1
    fi
done
echo ""

# 7. Docker 빌드 테스트 (Docker가 설치된 경우)
if command -v docker &> /dev/null; then
    log_info "Docker 빌드 테스트..."
    
    # Dockerfile 존재 확인
    if [ -f "Dockerfile.multiarch" ]; then
        # 빌드만 테스트 (푸시하지 않음)
        if docker build -f Dockerfile.multiarch -t proxynd:ci-test .; then
            log_success "Docker 빌드 성공"
            docker rmi proxynd:ci-test
        else
            log_error "Docker 빌드 실패"
            exit 1
        fi
    else
        log_warning "Dockerfile.multiarch를 찾을 수 없습니다"
    fi
else
    log_warning "Docker가 설치되어 있지 않습니다. Docker 빌드 테스트를 건너뜁니다."
fi
echo ""

# 8. 보안 스캔 (선택사항)
if command -v govulncheck &> /dev/null; then
    log_info "보안 취약점 스캔..."
    if govulncheck ./...; then
        log_success "보안 스캔 통과"
    else
        log_warning "보안 취약점이 발견되었습니다"
    fi
else
    log_warning "govulncheck가 설치되어 있지 않습니다. 보안 스캔을 건너뜁니다."
fi
echo ""

# 정리
log_info "임시 파일 정리..."
rm -f coverage.txt
log_success "정리 완료"
echo ""

# 완료
echo -e "${GREEN}=== 모든 CI 테스트 통과! ===${NC}"
echo ""
echo "다음 단계:"
echo "1. git add 및 commit"
echo "2. PR 생성 또는 push"
echo "3. CI 파이프라인 확인"