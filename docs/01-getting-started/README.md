# 🚀 프록신디 시작하기

> 프록신디를 처음 설치하고 사용하는 방법을 단계별로 안내합니다.

## 📋 개요

프록신디(ProxyND)는 다양한 패키지 매니저를 위한 고성능 프록시/미러 서버입니다. 기업 환경에서 패키지 다운로드 속도 향상과 대역폭 절약을 위해 설계되었습니다.

### 지원하는 패키지 매니저
- **Maven** (Java/Kotlin/Scala)
- **NPM** (Node.js)
- **APT** (Ubuntu/Debian)
- **Docker Registry**
- **PyPI** (Python)
- **YUM** (RedHat/CentOS)
- **APK** (Alpine Linux)

## 🚀 빠른 설치

### 1. Docker로 실행 (권장)

```bash
# 1. 저장소 클론
git clone https://github.com/your-org/proxynd.git
cd proxynd

# 2. Docker 이미지 빌드
make docker-build

# 3. 실행
make docker-run
```

### 2. 개발 환경에서 실행

```bash
# 1. Go 1.23+ 설치 확인
go version

# 2. 의존성 설치
make dev-prepare

# 3. 개발 환경 설정
make dev-setup

# 4. 개발 서버 실행 (핫 리로드)
make dev-run
```

## ⚙️ 기본 설정

### 환경 변수 설정

`.env` 파일이 자동 생성됩니다:
```bash
CONFIG_DIR=./tmp/config      # 설정 파일 디렉토리
STORAGE_DIR=./tmp/storage    # 캐시 저장 디렉토리
SERVER_PORT=8080            # 서버 포트
```

### 설정 파일 구조

```
./tmp/config/
├── global.yaml           # 글로벌 설정
├── maven-proxy.yaml      # Maven 프록시 설정
├── npm-proxy.yaml        # NPM 프록시 설정
├── apt-proxy.yaml        # APT 프록시 설정
└── ...                   # 기타 프록시 설정
```

## 🔗 다음 단계

### 프록시 설정
각 패키지 매니저별 클라이언트 설정:
- [Maven 설정](../04-proxy-types/maven/client-setup.md)
- [NPM 설정](../04-proxy-types/npm/client-setup.md)
- [APT 설정](../04-proxy-types/apt/client-setup.md)
- [Docker 설정](../04-proxy-types/docker/client-setup.md)

### 고급 설정
- [설정 가이드](../03-configuration/README.md)
- [환경 변수](../03-configuration/environment-variables.md)
- [핫 리로드](../03-configuration/hot-reload.md)

### 배포
- [Docker 배포](../06-deployment/docker/)
- [Kubernetes 배포](../06-deployment/kubernetes/)
- [Systemd 서비스](../06-deployment/systemd/)

## ✅ 확인 방법

### 서버 상태 확인
```bash
# 헬스체크 엔드포인트
curl http://localhost:8080/healthz

# 메트릭 확인
curl http://localhost:8080/metrics
```

### CLI 도구 사용
```bash
# proxyndctl 설치 후
proxyndctl status
proxyndctl cache list
```

## 🆘 문제 해결

일반적인 문제와 해결법:
- [문제해결 가이드](../09-troubleshooting/README.md)
- [헬스체크](../07-monitoring/health/README.md)

---

> 📌 **다음**: [설정 가이드](../03-configuration/README.md)로 이동하여 프록신디를 세부 설정하세요.
