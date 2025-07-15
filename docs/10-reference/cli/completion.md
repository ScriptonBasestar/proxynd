# ProxyND CLI 자동완성 가이드

이 문서는 `proxyndctl` 명령어의 자동완성 기능을 설정하는 방법을 설명합니다.

## 지원되는 쉘

ProxyND CLI는 다음 쉘에 대한 자동완성을 지원합니다:

- **Bash** (Linux, macOS)
- **Zsh** (Linux, macOS)
- **Fish** (Linux, macOS)
- **PowerShell** (Windows, Linux, macOS)

## 설치 방법

### Bash

#### 방법 1: 현재 세션에만 적용

```bash
source <(proxyndctl completion bash)
```

#### 방법 2: 영구 설정 (사용자별)

```bash
# Linux
echo 'source <(proxyndctl completion bash)' >> ~/.bashrc

# macOS
echo 'source <(proxyndctl completion bash)' >> ~/.bash_profile
```

#### 방법 3: 시스템 전체 설정

```bash
# Linux (관리자 권한 필요)
proxyndctl completion bash | sudo tee /etc/bash_completion.d/proxyndctl > /dev/null

# macOS (Homebrew 사용 시)
proxyndctl completion bash > $(brew --prefix)/etc/bash_completion.d/proxyndctl
```

### Zsh

#### 방법 1: 현재 세션에만 적용

```zsh
source <(proxyndctl completion zsh)
```

#### 방법 2: Oh My Zsh 사용자

```zsh
proxyndctl completion zsh > ~/.oh-my-zsh/custom/plugins/proxyndctl/_proxyndctl
```

#### 방법 3: 일반 Zsh 설정

```zsh
# 자동완성 디렉토리에 저장
proxyndctl completion zsh > "${fpath[1]}/_proxyndctl"

# 또는 .zshrc에 추가
echo 'source <(proxyndctl completion zsh)' >> ~/.zshrc
```

### Fish

Fish는 자동완성 파일을 특정 디렉토리에 저장하면 자동으로 로드합니다:

```fish
# 사용자별 설정
proxyndctl completion fish > ~/.config/fish/completions/proxyndctl.fish

# 시스템 전체 설정 (관리자 권한 필요)
proxyndctl completion fish | sudo tee /usr/share/fish/completions/proxyndctl.fish > /dev/null
```

### PowerShell

#### 방법 1: 현재 세션에만 적용

```powershell
proxyndctl completion powershell | Out-String | Invoke-Expression
```

#### 방법 2: 영구 설정

```powershell
# PowerShell 프로필에 추가
proxyndctl completion powershell >> $PROFILE

# 프로필이 없는 경우 먼저 생성
New-Item -Path $PROFILE -Type File -Force
```

## 설치 확인

자동완성이 제대로 설치되었는지 확인하려면:

1. 새 터미널/쉘 세션을 시작합니다
2. `proxyndctl` 입력 후 Tab 키를 누릅니다
3. 사용 가능한 명령어 목록이 표시되어야 합니다

### 예시

```bash
$ proxyndctl [TAB]
cache       config      health      metrics     test        
completion  help        status      user        version
```

## 문제 해결

### Bash에서 자동완성이 작동하지 않는 경우

1. bash-completion 패키지가 설치되어 있는지 확인:
   ```bash
   # Ubuntu/Debian
   sudo apt-get install bash-completion
   
   # RHEL/CentOS
   sudo yum install bash-completion
   
   # macOS
   brew install bash-completion
   ```

2. 쉘을 다시 시작하거나 설정 파일을 다시 로드:
   ```bash
   source ~/.bashrc
   ```

### Zsh에서 자동완성이 작동하지 않는 경우

1. compinit이 활성화되어 있는지 확인:
   ```zsh
   echo 'autoload -Uz compinit && compinit' >> ~/.zshrc
   ```

2. fpath에 자동완성 디렉토리가 포함되어 있는지 확인:
   ```zsh
   echo $fpath
   ```

### Fish에서 자동완성이 작동하지 않는 경우

1. 자동완성 파일이 올바른 위치에 있는지 확인:
   ```fish
   ls ~/.config/fish/completions/proxyndctl.fish
   ```

2. Fish 버전이 3.0 이상인지 확인:
   ```fish
   fish --version
   ```

## 고급 기능

### 명령어별 자동완성

ProxyND CLI는 컨텍스트를 인식하는 자동완성을 제공합니다:

- **플래그 자동완성**: `--` 입력 후 Tab 키를 누르면 사용 가능한 플래그 목록이 표시됩니다
- **서브커맨드 자동완성**: 각 명령어의 서브커맨드가 자동으로 제안됩니다
- **값 자동완성**: 일부 플래그는 가능한 값 목록을 제공합니다 (예: `--format` 플래그)

### 예시

```bash
# 캐시 명령어의 서브커맨드 자동완성
$ proxyndctl cache [TAB]
clear  list   size

# 플래그 자동완성
$ proxyndctl cache list --[TAB]
--format   --help     --server   --timeout  --verbose

# 형식 옵션 자동완성
$ proxyndctl cache list --format [TAB]
json   table  yaml
```

## 자동완성 스크립트 업데이트

ProxyND CLI가 업데이트되면 자동완성 스크립트도 업데이트해야 할 수 있습니다:

```bash
# Bash
proxyndctl completion bash > /etc/bash_completion.d/proxyndctl

# Zsh
proxyndctl completion zsh > "${fpath[1]}/_proxyndctl"

# Fish
proxyndctl completion fish > ~/.config/fish/completions/proxyndctl.fish

# PowerShell
proxyndctl completion powershell > $PROFILE
```