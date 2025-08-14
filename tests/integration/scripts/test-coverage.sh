#!/bin/bash
# 스크립트명: 통합 테스트 커버리지 분석 스크립트
# 용도: ProxyND 프로젝트의 전체 테스트 커버리지 분석 및 HTML 리포트 생성
# 사용법: scripts/test-coverage.sh [옵션]
# 예시: scripts/test-coverage.sh --verbose --html

set -euo pipefail

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

# 기본 설정
VERBOSE=false
HTML_REPORT=true
THRESHOLD=75
COVERAGE_DIR="coverage"

log_info "ProxyND 테스트 커버리지 분석 시작..."

# 커버리지 디렉토리 생성
mkdir -p "${COVERAGE_DIR}"

# 개발 환경 설정
if [[ ! -d "./tmp/config" ]]; then
    log_info "개발 환경 설정 중..."
    mkdir -p ./tmp/storage ./tmp/config
    if [[ -d "./examples" ]]; then
        cp -r examples/* ./tmp/config/ 2>/dev/null || true
    fi
    echo "CONFIG_DIR=./tmp/config" > .env
    echo "STORAGE_DIR=./tmp/storage" >> .env
    echo "SERVER_PORT=8080" >> .env
fi

# 커버리지 테스트 실행
log_info "테스트 실행 중..."
CONFIG_DIR=./tmp/config STORAGE_DIR=./tmp/storage go test -race -coverprofile="${COVERAGE_DIR}/coverage.out" -covermode=atomic ./... || {
    log_error "테스트 실행 실패"
    exit 1
}

# 커버리지 통계 출력
if [[ -f "${COVERAGE_DIR}/coverage.out" ]]; then
    COVERAGE_PERCENTAGE=$(go tool cover -func="${COVERAGE_DIR}/coverage.out" | grep "total:" | awk '{print $3}' | sed 's/%//')
    echo ""
    blue "📊 테스트 커버리지 결과: ${COVERAGE_PERCENTAGE}%"

    # HTML 리포트 생성
    go tool cover -html="${COVERAGE_DIR}/coverage.out" -o "${COVERAGE_DIR}/coverage.html"
    log_success "HTML 리포트 생성 완료: ${COVERAGE_DIR}/coverage.html"
else
    log_error "커버리지 파일이 생성되지 않았습니다"
    exit 1
fi

log_success "테스트 커버리지 분석 완료!"
