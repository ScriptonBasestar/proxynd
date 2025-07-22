#!/bin/bash

# ProxyND CLI man page 설치 스크립트

set -e

# 색상 코드
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
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

# 사용법 출력
usage() {
    cat << EOF
ProxyND CLI man page 설치 스크립트

사용법: $0 [옵션]

옵션:
    -g, --generate      man page 생성만 (설치하지 않음)
    -i, --install       생성된 man page 설치 (기본값)
    -d, --dir DIR       man page 생성 디렉토리 (기본값: ./man)
    -h, --help          이 도움말 출력

예제:
    $0                  # man page 생성 및 설치
    $0 -g               # man page 생성만
    $0 -d /tmp/man -i   # /tmp/man에 생성 후 설치
EOF
}

# proxyndctl 명령어 확인
check_proxyndctl() {
    if ! command -v proxyndctl &> /dev/null; then
        log_error "proxyndctl 명령어를 찾을 수 없습니다."
        log_error "먼저 proxyndctl을 빌드하거나 PATH에 추가해주세요."
        exit 1
    fi
}

# man page 생성
generate_man_pages() {
    local dir=$1

    log_info "Man page 생성 중..."

    # 디렉토리 생성
    mkdir -p "$dir"

    # man page 생성
    if proxyndctl docs man "$dir"; then
        log_info "Man page가 성공적으로 생성되었습니다: $dir"

        # 생성된 파일 목록
        local count=$(find "$dir" -name "*.1" | wc -l)
        log_info "생성된 man page 수: $count개"
    else
        log_error "Man page 생성 실패"
        return 1
    fi
}

# man page 설치
install_man_pages() {
    local dir=$1

    log_info "Man page 설치 중..."

    # 생성된 man page 확인
    if [ ! -d "$dir" ] || [ -z "$(ls -A $dir/*.1 2>/dev/null)" ]; then
        log_error "설치할 man page가 없습니다: $dir"
        return 1
    fi

    # 설치 디렉토리 확인
    local man_dir="/usr/share/man/man1"
    if [ ! -d "$man_dir" ]; then
        log_warn "시스템 man 디렉토리가 없습니다: $man_dir"
        log_warn "다른 위치에 설치하려면 수동으로 복사하세요."
        return 1
    fi

    # 관리자 권한 확인
    if [ "$EUID" -ne 0 ]; then
        log_info "관리자 권한이 필요합니다. sudo로 재실행합니다..."
        sudo cp "$dir"/*.1 "$man_dir/" || {
            log_error "Man page 설치 실패"
            return 1
        }
    else
        cp "$dir"/*.1 "$man_dir/" || {
            log_error "Man page 설치 실패"
            return 1
        }
    fi

    # man 데이터베이스 업데이트
    log_info "Man 데이터베이스 업데이트 중..."
    if command -v mandb &> /dev/null; then
        if [ "$EUID" -ne 0 ]; then
            sudo mandb
        else
            mandb
        fi
    elif command -v makewhatis &> /dev/null; then
        if [ "$EUID" -ne 0 ]; then
            sudo makewhatis
        else
            makewhatis
        fi
    else
        log_warn "man 데이터베이스 업데이트 명령어를 찾을 수 없습니다."
    fi

    log_info "Man page 설치 완료!"
    log_info "사용 예: man proxyndctl"
}

# 메인 함수
main() {
    local generate_only=false
    local install=true
    local man_dir="./man"

    # 명령줄 인수 파싱
    while [[ $# -gt 0 ]]; do
        case $1 in
            -g|--generate)
                generate_only=true
                install=false
                shift
                ;;
            -i|--install)
                install=true
                generate_only=false
                shift
                ;;
            -d|--dir)
                man_dir="$2"
                shift 2
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

    # proxyndctl 확인
    check_proxyndctl

    # man page 생성
    generate_man_pages "$man_dir" || exit 1

    # 설치 (옵션에 따라)
    if [ "$install" = true ] && [ "$generate_only" = false ]; then
        install_man_pages "$man_dir"
    fi
}

# 스크립트 실행
main "$@"
