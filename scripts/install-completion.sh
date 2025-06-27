#!/bin/bash

# ProxyND CLI 자동완성 설치 스크립트

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
ProxyND CLI 자동완성 설치 스크립트

사용법: $0 [옵션]

옵션:
    -s, --shell SHELL    설치할 쉘 지정 (bash, zsh, fish, powershell)
    -u, --user           사용자 모드로 설치 (기본값)
    -g, --global         시스템 전체 설치 (관리자 권한 필요)
    -h, --help           이 도움말 출력

예제:
    $0                   # 현재 쉘 자동 감지하여 설치
    $0 -s bash           # Bash용 자동완성 설치
    $0 -s zsh -g         # Zsh용 자동완성 시스템 전체 설치
EOF
}

# proxyndctl 명령어 확인
check_proxyndctl() {
    if ! command -v proxyndctl &> /dev/null; then
        log_error "proxyndctl 명령어를 찾을 수 없습니다."
        log_error "먼저 proxyndctl을 설치하거나 PATH에 추가해주세요."
        exit 1
    fi
}

# 현재 쉘 감지
detect_shell() {
    local shell_name=""
    
    if [ -n "$ZSH_VERSION" ]; then
        shell_name="zsh"
    elif [ -n "$BASH_VERSION" ]; then
        shell_name="bash"
    elif [ -n "$FISH_VERSION" ]; then
        shell_name="fish"
    else
        # $SHELL 변수에서 추출
        shell_name=$(basename "$SHELL")
    fi
    
    echo "$shell_name"
}

# Bash 자동완성 설치
install_bash() {
    local mode=$1
    
    if [ "$mode" = "global" ]; then
        log_info "Bash 자동완성을 시스템 전체에 설치합니다..."
        
        local completion_dir="/etc/bash_completion.d"
        if [ -d "$completion_dir" ]; then
            if proxyndctl completion bash | sudo tee "$completion_dir/proxyndctl" > /dev/null; then
                log_info "설치 완료: $completion_dir/proxyndctl"
            else
                log_error "설치 실패"
                return 1
            fi
        else
            log_error "bash-completion 디렉토리를 찾을 수 없습니다: $completion_dir"
            log_warn "bash-completion 패키지를 설치해주세요:"
            log_warn "  Ubuntu/Debian: sudo apt-get install bash-completion"
            log_warn "  RHEL/CentOS: sudo yum install bash-completion"
            return 1
        fi
    else
        log_info "Bash 자동완성을 사용자 모드로 설치합니다..."
        
        local bashrc="$HOME/.bashrc"
        if [ -f "$HOME/.bash_profile" ] && [ "$(uname)" = "Darwin" ]; then
            bashrc="$HOME/.bash_profile"
        fi
        
        if ! grep -q "proxyndctl completion bash" "$bashrc" 2>/dev/null; then
            echo "" >> "$bashrc"
            echo "# ProxyND CLI 자동완성" >> "$bashrc"
            echo 'source <(proxyndctl completion bash)' >> "$bashrc"
            log_info "설치 완료: $bashrc"
        else
            log_warn "이미 설치되어 있습니다: $bashrc"
        fi
    fi
    
    log_info "새 터미널을 열거나 다음 명령어를 실행하세요:"
    log_info "  source ~/.bashrc"
}

# Zsh 자동완성 설치
install_zsh() {
    local mode=$1
    
    if [ "$mode" = "global" ]; then
        log_info "Zsh 자동완성을 시스템 전체에 설치합니다..."
        
        # fpath 디렉토리 찾기
        local completion_dir="/usr/local/share/zsh/site-functions"
        if [ ! -d "$completion_dir" ]; then
            completion_dir="/usr/share/zsh/site-functions"
        fi
        
        if [ -d "$completion_dir" ]; then
            if proxyndctl completion zsh | sudo tee "$completion_dir/_proxyndctl" > /dev/null; then
                log_info "설치 완료: $completion_dir/_proxyndctl"
            else
                log_error "설치 실패"
                return 1
            fi
        else
            log_error "Zsh 자동완성 디렉토리를 찾을 수 없습니다"
            return 1
        fi
    else
        log_info "Zsh 자동완성을 사용자 모드로 설치합니다..."
        
        # Oh My Zsh 확인
        if [ -d "$HOME/.oh-my-zsh" ]; then
            local omz_dir="$HOME/.oh-my-zsh/custom/plugins/proxyndctl"
            mkdir -p "$omz_dir"
            if proxyndctl completion zsh > "$omz_dir/_proxyndctl"; then
                log_info "Oh My Zsh에 설치 완료: $omz_dir/_proxyndctl"
            else
                log_error "설치 실패"
                return 1
            fi
        else
            # 일반 Zsh 설정
            local zshrc="$HOME/.zshrc"
            if ! grep -q "proxyndctl completion zsh" "$zshrc" 2>/dev/null; then
                echo "" >> "$zshrc"
                echo "# ProxyND CLI 자동완성" >> "$zshrc"
                echo 'source <(proxyndctl completion zsh)' >> "$zshrc"
                log_info "설치 완료: $zshrc"
            else
                log_warn "이미 설치되어 있습니다: $zshrc"
            fi
        fi
    fi
    
    log_info "새 터미널을 열거나 다음 명령어를 실행하세요:"
    log_info "  source ~/.zshrc"
}

# Fish 자동완성 설치
install_fish() {
    local mode=$1
    
    if [ "$mode" = "global" ]; then
        log_info "Fish 자동완성을 시스템 전체에 설치합니다..."
        
        local completion_dir="/usr/share/fish/completions"
        if [ -d "$completion_dir" ]; then
            if proxyndctl completion fish | sudo tee "$completion_dir/proxyndctl.fish" > /dev/null; then
                log_info "설치 완료: $completion_dir/proxyndctl.fish"
            else
                log_error "설치 실패"
                return 1
            fi
        else
            log_error "Fish 자동완성 디렉토리를 찾을 수 없습니다: $completion_dir"
            return 1
        fi
    else
        log_info "Fish 자동완성을 사용자 모드로 설치합니다..."
        
        local fish_dir="$HOME/.config/fish/completions"
        mkdir -p "$fish_dir"
        
        if proxyndctl completion fish > "$fish_dir/proxyndctl.fish"; then
            log_info "설치 완료: $fish_dir/proxyndctl.fish"
        else
            log_error "설치 실패"
            return 1
        fi
    fi
    
    log_info "새 터미널을 열면 자동완성이 활성화됩니다."
}

# PowerShell 자동완성 설치
install_powershell() {
    log_info "PowerShell 자동완성을 설치합니다..."
    
    if command -v pwsh &> /dev/null; then
        pwsh -Command "proxyndctl completion powershell >> \$PROFILE"
        log_info "설치 완료: PowerShell 프로필에 추가됨"
        log_info "새 PowerShell 세션을 시작하면 자동완성이 활성화됩니다."
    else
        log_error "PowerShell을 찾을 수 없습니다."
        log_info "Windows에서는 다음 명령어를 PowerShell에서 실행하세요:"
        log_info "  proxyndctl completion powershell >> \$PROFILE"
        return 1
    fi
}

# 메인 함수
main() {
    local shell_type=""
    local install_mode="user"
    
    # 명령줄 인수 파싱
    while [[ $# -gt 0 ]]; do
        case $1 in
            -s|--shell)
                shell_type="$2"
                shift 2
                ;;
            -u|--user)
                install_mode="user"
                shift
                ;;
            -g|--global)
                install_mode="global"
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
    
    # proxyndctl 확인
    check_proxyndctl
    
    # 쉘 타입이 지정되지 않았으면 자동 감지
    if [ -z "$shell_type" ]; then
        shell_type=$(detect_shell)
        log_info "감지된 쉘: $shell_type"
    fi
    
    # 쉘별 설치 수행
    case $shell_type in
        bash)
            install_bash "$install_mode"
            ;;
        zsh)
            install_zsh "$install_mode"
            ;;
        fish)
            install_fish "$install_mode"
            ;;
        powershell|pwsh)
            install_powershell
            ;;
        *)
            log_error "지원하지 않는 쉘입니다: $shell_type"
            log_info "지원되는 쉘: bash, zsh, fish, powershell"
            exit 1
            ;;
    esac
}

# 스크립트 실행
main "$@"