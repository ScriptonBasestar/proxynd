#!/bin/bash
# 스크립트명: 통합 테스트 커버리지 분석 스크립트
# 용도: ProxyND 프로젝트의 전체 테스트 커버리지 분석 및 HTML 리포트 생성
# 사용법: scripts/test-coverage.sh [옵션]
# 예시: scripts/test-coverage.sh --verbose --html

set -euo pipefail

# 스크립트 경로 확인
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
cd "${PROJECT_ROOT}"

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
OUTPUT_FILE="coverage.out"
HTML_FILE="coverage.html"

# 헬프 메시지
show_help() {
    echo "ProxyND 테스트 커버리지 분석 스크립트"
    echo ""
    echo "Usage: $0 [options]"
    echo ""
    echo "Options:"
    echo "  -v, --verbose     상세 출력 모드"
    echo "  -h, --html        HTML 리포트 생성 (기본값: true)"
    echo "  --no-html         HTML 리포트 생성 안함"
    echo "  -t, --threshold   최소 커버리지 임계값 (기본값: 75%)"
    echo "  --help            이 도움말 표시"
    echo ""
    echo "Examples:"
    echo "  $0                           # 기본 커버리지 분석"
    echo "  $0 --verbose --html          # 상세 출력 + HTML 리포트"
    echo "  $0 --threshold 80            # 80% 임계값 설정"
    echo ""
}

# 인수 파싱
while [[ $# -gt 0 ]]; do
    case $1 in
        -v|--verbose)
            VERBOSE=true
            shift
            ;;
        -h|--html)
            HTML_REPORT=true
            shift
            ;;
        --no-html)
            HTML_REPORT=false
            shift
            ;;
        -t|--threshold)
            THRESHOLD="$2"
            shift 2
            ;;
        --help)
            show_help
            exit 0
            ;;
        *)
            log_error "알 수 없는 옵션: $1"
            show_help
            exit 1
            ;;
    esac
done

# 커버리지 디렉토리 생성
mkdir -p "${COVERAGE_DIR}"

log_info "ProxyND 테스트 커버리지 분석 시작..."

# Go 버전 확인
GO_VERSION=$(go version | awk '{print $3}')
log_info "Go 버전: ${GO_VERSION}"

# 의존성 확인
log_info "의존성 확인 중..."
go mod download
go mod verify

# 테스트 실행 및 커버리지 수집
log_info "테스트 실행 및 커버리지 데이터 수집 중..."

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
if [[ "$VERBOSE" == "true" ]]; then
    log_info "상세 모드로 테스트 실행..."
    CONFIG_DIR=./tmp/config STORAGE_DIR=./tmp/storage go test -v -race -coverprofile="${COVERAGE_DIR}/${OUTPUT_FILE}" -covermode=atomic ./... || {
        log_error "테스트 실행 실패"
        exit 1
    }
else
    log_info "테스트 실행 중..."
    CONFIG_DIR=./tmp/config STORAGE_DIR=./tmp/storage go test -race -coverprofile="${COVERAGE_DIR}/${OUTPUT_FILE}" -covermode=atomic ./... || {
        log_error "테스트 실행 실패"
        exit 1
    }
fi

# 커버리지 파일 존재 확인
if [[ ! -f "${COVERAGE_DIR}/${OUTPUT_FILE}" ]]; then
    log_error "커버리지 파일이 생성되지 않았습니다: ${COVERAGE_DIR}/${OUTPUT_FILE}"
    exit 1
fi

# 커버리지 통계 출력
log_info "커버리지 통계 계산 중..."
COVERAGE_PERCENTAGE=$(go tool cover -func="${COVERAGE_DIR}/${OUTPUT_FILE}" | grep "total:" | awk '{print $3}' | sed 's/%//')

echo ""
blue "📊 테스트 커버리지 결과"
echo "======================================"
echo "전체 커버리지: ${COVERAGE_PERCENTAGE}%"
echo "임계값: ${THRESHOLD}%"

# 패키지별 커버리지 상세 정보
if [[ "$VERBOSE" == "true" ]]; then
    echo ""
    blue "📦 패키지별 커버리지 상세"
    echo "======================================"
    go tool cover -func="${COVERAGE_DIR}/${OUTPUT_FILE}" | head -20
    echo ""
fi

# HTML 리포트 생성
if [[ "$HTML_REPORT" == "true" ]]; then
    log_info "HTML 커버리지 리포트 생성 중..."
    go tool cover -html="${COVERAGE_DIR}/${OUTPUT_FILE}" -o "${COVERAGE_DIR}/${HTML_FILE}"
    log_success "HTML 리포트 생성 완료: ${COVERAGE_DIR}/${HTML_FILE}"
fi

# 임계값 검사
if (( $(echo "${COVERAGE_PERCENTAGE} >= ${THRESHOLD}" | bc -l) )); then
    log_success "커버리지 임계값 통과: ${COVERAGE_PERCENTAGE}% >= ${THRESHOLD}%"
else
    log_error "커버리지 임계값 미달: ${COVERAGE_PERCENTAGE}% < ${THRESHOLD}%"
    exit 1
fi

# 추가 정보 출력
echo ""
echo "커버리지 파일: ${COVERAGE_DIR}/${OUTPUT_FILE}"
if [[ "$HTML_REPORT" == "true" ]]; then
    echo "HTML 리포트: ${COVERAGE_DIR}/${HTML_FILE}"
    echo ""
    echo "HTML 리포트를 브라우저에서 열려면:"
    echo "  open ${COVERAGE_DIR}/${HTML_FILE}  # macOS"
    echo "  xdg-open ${COVERAGE_DIR}/${HTML_FILE}  # Linux"
fi

log_success "테스트 커버리지 분석 완료!"