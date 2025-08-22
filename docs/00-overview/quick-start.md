# 🚀 ProxyND 빠른 시작 가이드

ProxyND를 5분 안에 설치하고 첫 번째 프록시를 설정해보세요.

## 📋 필수 요구사항

- **Go 1.23+**: 모든 기능 지원
- **Linux/macOS/Windows**: 크로스 플랫폼 지원
- **2GB+ RAM**: 기본 운영 권장
- **10GB+ 디스크**: 캐시 저장 공간

## ⚡ 5분 설치

### 1단계: 바이너리 다운로드

```bash
# GitHub Releases에서 최신 버전 다운로드
wget https://github.com/yourusername/proxynd/releases/latest/download/proxynd-linux-amd64.tar.gz
tar -xzf proxynd-linux-amd64.tar.gz
sudo mv proxynd /usr/local/bin/
```

### 2단계: 환경 설정

```bash
# 필수 환경 변수 설정
export CONFIG_DIR="/etc/proxynd"
export STORAGE_DIR="/var/lib/proxynd"
export SERVER_PORT="8080"

# 디렉터리 생성
sudo mkdir -p $CONFIG_DIR $STORAGE_DIR
sudo chown $USER:$USER $CONFIG_DIR $STORAGE_DIR
```

### 3단계: 기본 설정 생성

```bash
# 글로벌 설정 파일 생성
cat > $CONFIG_DIR/global.yaml << EOF
server:
  port: ${SERVER_PORT:-8080}
  host: "0.0.0.0"
  read_timeout: "30s"
  write_timeout: "30s"

logging:
  level: "info"
  format: "json"

cache:
  default_ttl: "1h"
  max_size: "10GB"
  cleanup_interval: "5m"
EOF
```

### 4단계: 첫 번째 프록시 설정 (NPM 예시)

```bash
# NPM 프록시 설정
cat > $CONFIG_DIR/npm-proxy.yaml << EOF
enabled: true
cache:
  type: "filesystem"
  path: "${STORAGE_DIR}/cache/npm"
  ttl: "24h"

upstream:
  url: "https://registry.npmjs.org"
  timeout: "30s"

auth:
  enabled: false
EOF
```

### 5단계: 서버 실행

```bash
# ProxyND 서버 시작
proxynd

# 또는 백그라운드로 실행
nohup proxynd > /var/log/proxynd.log 2>&1 &
```

## ✅ 설치 확인

### 헬스체크 확인
```bash
curl http://localhost:8080/healthz
# 응답: {"status":"ok","timestamp":"2025-01-01T10:30:45Z"}
```

### NPM 프록시 테스트
```bash
# 프록시를 통해 패키지 다운로드 테스트
curl -I http://localhost:8080/api/v1/proxy/npm/express/latest
# 응답: HTTP/1.1 200 OK
```

### 클라이언트 설정 테스트
```bash
# NPM 레지스트리 변경
npm config set registry http://localhost:8080/api/v1/proxy/npm/

# 패키지 설치 테스트
npm install express
```

## 🐳 Docker로 빠른 시작

### Docker Compose 사용

```yaml
# docker-compose.yml
version: '3.8'
services:
  proxynd:
    image: proxynd:latest
    ports:
      - "8080:8080"
    environment:
      - CONFIG_DIR=/etc/proxynd
      - STORAGE_DIR=/var/lib/proxynd
      - SERVER_PORT=8080
    volumes:
      - ./config:/etc/proxynd
      - ./storage:/var/lib/proxynd
    restart: unless-stopped
```

```bash
# 실행
docker-compose up -d

# 로그 확인
docker-compose logs -f proxynd
```

## 📦 지원 패키지 매니저

ProxyND는 7개의 패키지 매니저를 지원합니다:

| 패키지 매니저 | 언어/플랫폼 | API 경로 | 포트 |
|---------------|-------------|----------|------|
| **NPM** | Node.js | `/api/v1/proxy/npm/` | 8080 |
| **Maven** | Java/Kotlin/Scala | `/api/v1/proxy/maven/` | 8080 |
| **PyPI** | Python | `/api/v1/proxy/pip/` | 8080 |
| **APT** | Ubuntu/Debian | `/api/v1/proxy/apt/` | 8080 |
| **Docker** | Container | `/api/v1/proxy/docker/` | 8080 |
| **YUM** | RedHat/CentOS | `/api/v1/proxy/yum/` | 8080 |
| **APK** | Alpine Linux | `/api/v1/proxy/apk/` | 8080 |

## 🔧 다음 단계

### 추가 프록시 설정
```bash
# Maven 프록시 추가
cat > $CONFIG_DIR/maven-proxy.yaml << EOF
enabled: true
cache:
  type: "filesystem"
  path: "${STORAGE_DIR}/cache/maven"
  ttl: "24h"
upstream:
  url: "https://repo1.maven.org/maven2"
  timeout: "30s"
EOF

# 서버 재시작 (설정 자동 리로드)
# ProxyND는 fsnotify로 설정 변경을 자동 감지합니다
```

### 보안 설정
```bash
# OAuth2 인증 설정 (선택사항)
cat >> $CONFIG_DIR/global.yaml << EOF

auth:
  enabled: true
  oauth2:
    provider: "github"
    client_id: "your-github-client-id"
    client_secret: "your-github-client-secret"
    allowed_organizations: ["your-org"]
EOF
```

### 모니터링 설정
```bash
# Prometheus 메트릭 확인
curl http://localhost:8080/metrics

# 구조화된 로그 확인
tail -f /var/log/proxynd.log | jq '.'
```

## 🚨 문제 해결

### 일반적인 문제들

**서버 시작 실패**:
```bash
# 환경 변수 확인
echo $CONFIG_DIR $STORAGE_DIR $SERVER_PORT

# 권한 확인
ls -la $CONFIG_DIR $STORAGE_DIR

# 포트 사용 중 확인
sudo netstat -tulpn | grep :8080
```

**프록시 연결 실패**:
```bash
# 설정 파일 구문 확인
cat $CONFIG_DIR/npm-proxy.yaml

# 업스트림 연결 확인
curl -I https://registry.npmjs.org

# 로그 확인
journalctl -u proxynd -f
```

**캐시 공간 부족**:
```bash
# 캐시 사용량 확인
du -sh $STORAGE_DIR/cache/*

# 캐시 정리
rm -rf $STORAGE_DIR/cache/npm/*
```

## 📚 참고 문서

- **[설정 참조](../20-configuration/configuration-reference.md)** - 상세 설정 옵션
- **[프록시 타입별 가이드](../30-proxy-types/)** - 각 패키지 매니저 설정
- **[보안 설정](../50-security/oauth2-authentication.md)** - 인증 및 보안
- **[배포 가이드](../60-deployment/deployment-checklist.md)** - 프로덕션 배포
- **[문제 해결](../70-operations/troubleshooting.md)** - 상세 문제 해결

---

**🎉 축하합니다!** ProxyND가 성공적으로 설치되었습니다.  
**📞 지원**: 문제가 발생하면 [GitHub Issues](https://github.com/yourusername/proxynd/issues)에 문의하세요.