# ProxyND 설정 가이드

이 디렉토리는 ProxyND 서버의 설정 파일들을 포함합니다.

## 설정 파일

### config.example.yaml
완전한 설정 예시 파일입니다. 모든 가능한 설정 옵션과 기본값을 포함하고 있습니다.

```bash
# 예시 파일을 복사하여 사용
cp conf/config.example.yaml /etc/proxynd/config.yaml

# 필요한 설정만 주석 해제하고 수정
vim /etc/proxynd/config.yaml
```

## 설정 파일 구조

### 1. 서버 설정
```yaml
server:
  port: 8080
  host: "0.0.0.0"
  readTimeout: 30s
  writeTimeout: 30s
```

### 2. 로깅 설정
```yaml
logging:
  level: info
  format: json
  output: stdout
```

### 3. 캐시 설정
```yaml
cache:
  type: filesystem  # filesystem, s3, memory
  ttl: 3600
  maxSize: 10737418240  # 10GB
```

### 4. 프록시 설정
각 패키지 매니저별로 개별 설정 가능:

```yaml
npm:
  enabled: true
  upstream: "https://registry.npmjs.org"
  allowedPackages: ["*"]

pip:
  enabled: true
  upstream: "https://pypi.org/simple"

apt:
  enabled: true
  distributions:
    - name: ubuntu
      upstream: "http://archive.ubuntu.com/ubuntu"

docker:
  enabled: true
  upstream: "https://registry-1.docker.io"
```

### 5. 인증 및 보안
```yaml
auth:
  enabled: true
  type: basic
  users:
    - username: admin
      password: "$2a$10$..."  # bcrypt 해시
      permissions: ["read", "write", "delete"]

ipFilter:
  enabled: true
  allowedNetworks:
    - "192.168.0.0/16"
    - "10.0.0.0/8"
```

## 환경 변수 오버라이드

설정 값은 환경 변수로 오버라이드할 수 있습니다:

```bash
# 서버 포트 설정
export SERVER_PORT=8080

# 로그 레벨 설정
export LOG_LEVEL=debug

# 캐시 타입 설정
export CACHE_TYPE=s3

# S3 설정
export S3_BUCKET=proxynd-cache
export S3_REGION=ap-northeast-2
```

## 설정 검증

설정 파일의 문법을 검증할 수 있습니다:

```bash
# 설정 파일 검증
proxynd -validate-config -config /etc/proxynd/config.yaml

# 설정 파일 출력 (환경변수 적용된 최종 설정)
proxynd -dump-config -config /etc/proxynd/config.yaml
```

## 설정 리로드

실행 중인 서버의 설정을 리로드할 수 있습니다:

```bash
# SIGHUP 시그널로 리로드
kill -HUP $(pgrep proxynd)

# systemd 서비스인 경우
sudo systemctl reload proxynd
```

## 보안 고려사항

1. **패스워드 해싱**: 사용자 패스워드는 반드시 bcrypt로 해싱하여 저장
2. **파일 권한**: 설정 파일의 권한을 600으로 제한
3. **환경 변수**: 민감한 정보는 환경 변수로 관리
4. **TLS 인증서**: HTTPS 사용시 적절한 인증서 관리

```bash
# 설정 파일 권한 설정
chmod 600 /etc/proxynd/config.yaml
chown proxynd:proxynd /etc/proxynd/config.yaml

# bcrypt 패스워드 생성 (Go)
echo 'package main; import("fmt"; "golang.org/x/crypto/bcrypt"); func main() { hash, _ := bcrypt.GenerateFromPassword([]byte("password"), 10); fmt.Println(string(hash)) }' | go run -
```

## 예시 설정

### 기본 설정
```yaml
server:
  port: 8080

logging:
  level: info
  format: json

cache:
  type: filesystem
  ttl: 3600

npm:
  enabled: true
  upstream: "https://registry.npmjs.org"

pip:
  enabled: true
  upstream: "https://pypi.org/simple"
```

### 프로덕션 설정
```yaml
server:
  port: 8080
  tls:
    enabled: true
    certFile: "/etc/ssl/certs/proxynd.crt"
    keyFile: "/etc/ssl/private/proxynd.key"

logging:
  level: warn
  format: json
  output: file
  file:
    path: "/var/log/proxynd/proxynd.log"

cache:
  type: s3
  s3:
    bucket: "proxynd-cache"
    region: "ap-northeast-2"

auth:
  enabled: true
  type: basic
  users:
    - username: admin
      password: "$2a$10$..."
      permissions: ["read", "write", "delete"]

ipFilter:
  enabled: true
  allowedNetworks:
    - "10.0.0.0/8"
    - "192.168.0.0/16"

metrics:
  enabled: true
  auth:
    enabled: true
    username: "metrics"
    password: "secret"
```

### 개발 설정
```yaml
server:
  port: 8080

logging:
  level: debug
  format: console

cache:
  type: memory
  maxSize: 1073741824  # 1GB

debug:
  enabled: true
  pprof:
    enabled: true
    address: "localhost:6060"
```