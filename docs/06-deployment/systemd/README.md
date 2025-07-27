# ProxyND Systemd 서비스

ProxyND를 systemd 서비스로 실행하기 위한 템플릿과 설치 스크립트입니다.

## 파일 구성

- `proxynd.service` - 단일 인스턴스 서비스 유닛 파일
- `proxynd@.service` - 다중 인스턴스 서비스 템플릿
- `proxynd.env` - 환경 변수 설정 예시
- `install.sh` - 자동 설치 스크립트

## 빠른 시작

### 1. 단일 인스턴스 설치

```bash
# ProxyND 바이너리 빌드
go build -o proxynd main.go

# 서비스 설치
sudo ./systemd/install.sh -b ./proxynd -c ./examples

# 서비스 확인
sudo systemctl status proxynd
```

### 2. 다중 인스턴스 설치

여러 개의 ProxyND 인스턴스를 다른 포트에서 실행:

```bash
# 첫 번째 인스턴스 (포트 8081)
sudo ./systemd/install.sh -b ./proxynd -i prod -p 8081

# 두 번째 인스턴스 (포트 8082)
sudo ./systemd/install.sh -b ./proxynd -i dev -p 8082

# 서비스 확인
sudo systemctl status proxynd@prod
sudo systemctl status proxynd@dev
```

## 수동 설치

### 1. 사용자 생성

```bash
sudo useradd --system --user-group --home-dir /var/lib/proxynd --shell /bin/false proxynd
```

### 2. 디렉토리 생성

```bash
# 설치 디렉토리
sudo mkdir -p /opt/proxynd

# 설정 디렉토리
sudo mkdir -p /etc/proxynd

# 데이터 디렉토리
sudo mkdir -p /var/lib/proxynd
sudo chown proxynd:proxynd /var/lib/proxynd

# 로그 디렉토리
sudo mkdir -p /var/log/proxynd
sudo chown proxynd:proxynd /var/log/proxynd
```

### 3. 바이너리 설치

```bash
sudo cp proxynd /opt/proxynd/
sudo chmod 755 /opt/proxynd/proxynd
```

### 4. 설정 파일 복사

```bash
# 설정 파일
sudo cp -r examples/* /etc/proxynd/

# 환경 변수 파일
sudo cp systemd/proxynd.env /etc/default/proxynd
```

### 5. Systemd 서비스 설치

```bash
# 서비스 파일 복사
sudo cp systemd/proxynd.service /etc/systemd/system/

# systemd 리로드
sudo systemctl daemon-reload

# 서비스 활성화 및 시작
sudo systemctl enable proxynd
sudo systemctl start proxynd
```

## 서비스 관리

### 기본 명령어

```bash
# 서비스 시작
sudo systemctl start proxynd

# 서비스 중지
sudo systemctl stop proxynd

# 서비스 재시작
sudo systemctl restart proxynd

# 서비스 상태 확인
sudo systemctl status proxynd

# 서비스 로그 확인
sudo journalctl -u proxynd -f

# 설정 리로드 (SIGHUP)
sudo systemctl reload proxynd
```

### 다중 인스턴스 관리

```bash
# 특정 인스턴스 시작
sudo systemctl start proxynd@prod

# 모든 인스턴스 중지
sudo systemctl stop 'proxynd@*'

# 인스턴스 상태 확인
sudo systemctl status 'proxynd@*'
```

## 환경 변수 설정

`/etc/default/proxynd` 파일을 편집하여 환경 변수를 설정할 수 있습니다:

```bash
# 서버 설정
SERVER_PORT=8080
SERVER_HOST=0.0.0.0

# 디렉토리 설정
CONFIG_DIR=/etc/proxynd
STORAGE_DIR=/var/lib/proxynd

# 로깅 설정
LOG_LEVEL=info
LOG_FORMAT=json
```

## 보안 강화

systemd 서비스는 다음과 같은 보안 설정이 적용되어 있습니다:

- `NoNewPrivileges=true` - 권한 상승 방지
- `PrivateTmp=true` - 격리된 임시 디렉토리
- `ProtectSystem=strict` - 시스템 디렉토리 보호
- `ProtectHome=true` - 홈 디렉토리 접근 차단
- `ReadWritePaths` - 쓰기 가능한 경로 제한
- 전용 시스템 사용자로 실행

## 문제 해결

### 서비스가 시작되지 않는 경우

```bash
# 상세 로그 확인
sudo journalctl -u proxynd -n 100 --no-pager

# 설정 파일 문법 확인
sudo /opt/proxynd/proxynd -validate-config

# 권한 확인
ls -la /var/lib/proxynd
ls -la /var/log/proxynd
```

### 포트 충돌

```bash
# 사용 중인 포트 확인
sudo ss -tlnp | grep 8080

# 환경 파일에서 포트 변경
sudo vi /etc/default/proxynd
```

### 메모리 부족

```bash
# 서비스 파일 편집
sudo systemctl edit proxynd

# 메모리 제한 추가
[Service]
MemoryLimit=2G
MemoryHigh=1500M
```

## 모니터링

### Prometheus 메트릭

```bash
# 메트릭 엔드포인트 확인
curl http://localhost:8080/metrics
```

### 로그 분석

```bash
# 에러 로그만 보기
sudo journalctl -u proxynd -p err

# 특정 시간 범위 로그
sudo journalctl -u proxynd --since "1 hour ago"

# JSON 로그 파싱
sudo journalctl -u proxynd -o json | jq '.'
```

## 업그레이드

```bash
# 서비스 중지
sudo systemctl stop proxynd

# 바이너리 교체
sudo cp new-proxynd /opt/proxynd/proxynd

# 서비스 시작
sudo systemctl start proxynd
```

## 제거

```bash
# 자동 제거
sudo ./systemd/install.sh -u

# 수동 제거
sudo systemctl stop proxynd
sudo systemctl disable proxynd
sudo rm /etc/systemd/system/proxynd.service
sudo systemctl daemon-reload
```

## 고급 설정

### 자동 재시작 정책

```ini
[Service]
Restart=always
RestartSec=10
StartLimitInterval=300
StartLimitBurst=5
```

### CPU 친화도 설정

```ini
[Service]
CPUAffinity=0-3
```

### 네트워크 네임스페이스

```ini
[Service]
PrivateNetwork=true
```

### 사용자 정의 cgroup

```ini
[Service]
Slice=proxynd.slice
```
