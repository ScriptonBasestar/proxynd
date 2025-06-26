#!/bin/bash

# ProxyND systemd 설치 스크립트

set -e

# 색상 설정
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# 기본 설정
INSTALL_DIR="/opt/proxynd"
CONFIG_DIR="/etc/proxynd"
DATA_DIR="/var/lib/proxynd"
LOG_DIR="/var/log/proxynd"
USER="proxynd"
GROUP="proxynd"

# 사용법
usage() {
    echo "사용법: $0 [옵션]"
    echo "옵션:"
    echo "  -b, --binary PATH     ProxyND 바이너리 경로 (필수)"
    echo "  -c, --config PATH     설정 디렉토리 경로"
    echo "  -i, --instance NAME   인스턴스 이름 (다중 인스턴스용)"
    echo "  -p, --port PORT       서버 포트 (기본값: 8080)"
    echo "  -u, --uninstall       ProxyND 서비스 제거"
    echo "  -h, --help            도움말 출력"
    exit 1
}

# 로그 함수
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
    exit 1
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

# 루트 권한 확인
check_root() {
    if [ "$EUID" -ne 0 ]; then
        log_error "이 스크립트는 루트 권한으로 실행해야 합니다."
    fi
}

# 시스템 확인
check_system() {
    if ! command -v systemctl &> /dev/null; then
        log_error "systemd가 설치되어 있지 않습니다."
    fi
    
    if ! systemctl --version &> /dev/null; then
        log_error "systemd가 제대로 작동하지 않습니다."
    fi
}

# 사용자 생성
create_user() {
    if ! id "$USER" &> /dev/null; then
        log_info "사용자 생성: $USER"
        useradd --system --user-group --home-dir /var/lib/proxynd --shell /bin/false $USER
    else
        log_info "사용자가 이미 존재합니다: $USER"
    fi
}

# 디렉토리 생성
create_directories() {
    log_info "디렉토리 생성 중..."
    
    # 설치 디렉토리
    mkdir -p "$INSTALL_DIR"
    
    # 설정 디렉토리
    mkdir -p "$CONFIG_DIR"
    if [ -n "$INSTANCE" ]; then
        mkdir -p "$CONFIG_DIR/$INSTANCE"
    fi
    
    # 데이터 디렉토리
    mkdir -p "$DATA_DIR"
    if [ -n "$INSTANCE" ]; then
        mkdir -p "$DATA_DIR/$INSTANCE"
    fi
    
    # 로그 디렉토리
    mkdir -p "$LOG_DIR"
    
    # 권한 설정
    chown -R $USER:$GROUP "$DATA_DIR" "$LOG_DIR"
    chmod 755 "$INSTALL_DIR" "$CONFIG_DIR"
    chmod 750 "$DATA_DIR" "$LOG_DIR"
}

# 바이너리 설치
install_binary() {
    if [ -z "$BINARY_PATH" ]; then
        log_error "바이너리 경로를 지정해주세요 (-b 옵션)"
    fi
    
    if [ ! -f "$BINARY_PATH" ]; then
        log_error "바이너리 파일을 찾을 수 없습니다: $BINARY_PATH"
    fi
    
    log_info "바이너리 설치 중..."
    cp "$BINARY_PATH" "$INSTALL_DIR/proxynd"
    chmod 755 "$INSTALL_DIR/proxynd"
    chown root:root "$INSTALL_DIR/proxynd"
    
    # 버전 확인
    if "$INSTALL_DIR/proxynd" -version &> /dev/null; then
        VERSION=$("$INSTALL_DIR/proxynd" -version)
        log_success "ProxyND $VERSION 설치됨"
    fi
}

# 설정 파일 설치
install_config() {
    log_info "설정 파일 설치 중..."
    
    # 기본 환경 파일
    if [ -n "$INSTANCE" ]; then
        ENV_FILE="/etc/default/proxynd-$INSTANCE"
    else
        ENV_FILE="/etc/default/proxynd"
    fi
    
    if [ ! -f "$ENV_FILE" ]; then
        cat > "$ENV_FILE" << EOF
# ProxyND 환경 변수 설정
SERVER_PORT=${PORT:-8080}
CONFIG_DIR=${CONFIG_DIR}${INSTANCE:+/$INSTANCE}
STORAGE_DIR=${DATA_DIR}${INSTANCE:+/$INSTANCE}
LOG_LEVEL=info
LOG_FORMAT=json
EOF
        chmod 644 "$ENV_FILE"
    else
        log_warning "환경 파일이 이미 존재합니다: $ENV_FILE"
    fi
    
    # 샘플 설정 복사
    if [ -n "$CONFIG_PATH" ] && [ -d "$CONFIG_PATH" ]; then
        log_info "설정 파일 복사 중..."
        if [ -n "$INSTANCE" ]; then
            cp -r "$CONFIG_PATH"/* "$CONFIG_DIR/$INSTANCE/"
        else
            cp -r "$CONFIG_PATH"/* "$CONFIG_DIR/"
        fi
    fi
}

# systemd 서비스 설치
install_service() {
    log_info "systemd 서비스 설치 중..."
    
    if [ -n "$INSTANCE" ]; then
        # 인스턴스 서비스
        SERVICE_FILE="/etc/systemd/system/proxynd@.service"
        if [ ! -f "$SERVICE_FILE" ]; then
            cp "$(dirname "$0")/proxynd@.service" "$SERVICE_FILE"
        fi
        SERVICE_NAME="proxynd@$INSTANCE"
    else
        # 단일 서비스
        SERVICE_FILE="/etc/systemd/system/proxynd.service"
        cp "$(dirname "$0")/proxynd.service" "$SERVICE_FILE"
        SERVICE_NAME="proxynd"
    fi
    
    # systemd 리로드
    systemctl daemon-reload
    
    # 서비스 활성화
    systemctl enable "$SERVICE_NAME"
    log_success "서비스 활성화됨: $SERVICE_NAME"
}

# 서비스 시작
start_service() {
    log_info "서비스 시작 중..."
    
    if [ -n "$INSTANCE" ]; then
        SERVICE_NAME="proxynd@$INSTANCE"
    else
        SERVICE_NAME="proxynd"
    fi
    
    systemctl start "$SERVICE_NAME"
    
    # 상태 확인
    sleep 2
    if systemctl is-active --quiet "$SERVICE_NAME"; then
        log_success "서비스가 성공적으로 시작되었습니다."
        systemctl status "$SERVICE_NAME" --no-pager
    else
        log_error "서비스 시작 실패"
        journalctl -u "$SERVICE_NAME" -n 50 --no-pager
    fi
}

# 제거
uninstall() {
    log_info "ProxyND 제거 중..."
    
    # 서비스 중지
    if [ -n "$INSTANCE" ]; then
        SERVICE_NAME="proxynd@$INSTANCE"
    else
        SERVICE_NAME="proxynd"
        # 모든 인스턴스 중지
        systemctl stop 'proxynd@*' 2>/dev/null || true
    fi
    
    systemctl stop "$SERVICE_NAME" 2>/dev/null || true
    systemctl disable "$SERVICE_NAME" 2>/dev/null || true
    
    # 파일 제거
    if [ -z "$INSTANCE" ]; then
        # 전체 제거
        rm -f /etc/systemd/system/proxynd.service
        rm -f /etc/systemd/system/proxynd@.service
        rm -rf "$INSTALL_DIR"
        rm -f /etc/default/proxynd*
        
        # 사용자에게 확인
        read -p "설정과 데이터도 삭제하시겠습니까? (y/N) " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            rm -rf "$CONFIG_DIR"
            rm -rf "$DATA_DIR"
            rm -rf "$LOG_DIR"
            userdel -r "$USER" 2>/dev/null || true
        fi
    else
        # 인스턴스만 제거
        rm -rf "$CONFIG_DIR/$INSTANCE"
        rm -rf "$DATA_DIR/$INSTANCE"
        rm -f "/etc/default/proxynd-$INSTANCE"
    fi
    
    systemctl daemon-reload
    log_success "ProxyND가 제거되었습니다."
}

# 메인
main() {
    # 옵션 파싱
    while [[ $# -gt 0 ]]; do
        case $1 in
            -b|--binary)
                BINARY_PATH="$2"
                shift 2
                ;;
            -c|--config)
                CONFIG_PATH="$2"
                shift 2
                ;;
            -i|--instance)
                INSTANCE="$2"
                shift 2
                ;;
            -p|--port)
                PORT="$2"
                shift 2
                ;;
            -u|--uninstall)
                UNINSTALL=true
                shift
                ;;
            -h|--help)
                usage
                ;;
            *)
                log_error "알 수 없는 옵션: $1"
                ;;
        esac
    done
    
    # 권한 확인
    check_root
    check_system
    
    # 제거 모드
    if [ "$UNINSTALL" = true ]; then
        uninstall
        exit 0
    fi
    
    # 설치
    log_info "ProxyND 설치 시작..."
    create_user
    create_directories
    install_binary
    install_config
    install_service
    start_service
    
    echo
    log_success "ProxyND 설치가 완료되었습니다!"
    echo
    echo "다음 명령어로 서비스를 관리할 수 있습니다:"
    if [ -n "$INSTANCE" ]; then
        echo "  systemctl status proxynd@$INSTANCE"
        echo "  systemctl stop proxynd@$INSTANCE"
        echo "  systemctl start proxynd@$INSTANCE"
        echo "  systemctl restart proxynd@$INSTANCE"
        echo "  journalctl -u proxynd@$INSTANCE -f"
    else
        echo "  systemctl status proxynd"
        echo "  systemctl stop proxynd"
        echo "  systemctl start proxynd"
        echo "  systemctl restart proxynd"
        echo "  journalctl -u proxynd -f"
    fi
    echo
    echo "설정 파일: $CONFIG_DIR${INSTANCE:+/$INSTANCE}"
    echo "데이터 디렉토리: $DATA_DIR${INSTANCE:+/$INSTANCE}"
    echo "로그: journalctl 또는 $LOG_DIR"
}

# 실행
main "$@"