# ProxyND E2E Testing Environment

ProxyND를 위한 완전한 End-to-End 테스트 환경입니다. Docker Compose를 사용하여 실제 패키지 매니저 클라이언트들과 함께 ProxyND의 모든 기능을 테스트할 수 있습니다.

## 🏗️ 아키텍처

```
┌─────────────────────────────────────────────────────────────┐
│                    E2E Test Environment                     │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐  │
│  │   Clients    │    │   ProxyND    │    │  Upstream    │  │
│  │              │───▶│              │───▶│   Server     │  │
│  │ npm/pip/apt  │    │   (Main)     │    │   (Nginx)    │  │
│  │   /docker    │    │              │    │              │  │
│  └──────────────┘    └──────────────┘    └──────────────┘  │
│                                                             │
│  ┌──────────────┐    ┌──────────────┐                      │
│  │    MinIO     │    │ Test Runner  │                      │
│  │ (S3 Cache)   │    │   (Ginkgo)   │                      │
│  └──────────────┘    └──────────────┘                      │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

## 🚀 빠른 시작

### 1. 환경 시작
```bash
cd tests/e2e
make up
```

### 2. 테스트 실행
```bash
make test
```

### 3. 환경 정리
```bash
make down
```

## 📦 구성 요소

### 서비스들

| 서비스 | 포트 | 설명 |
|--------|------|------|
| **proxynd** | 8080 | 메인 ProxyND 서버 |
| **minio** | 9000, 9001 | S3 호환 스토리지 (캐시 백엔드) |
| **nginx-upstream** | 8081 | 업스트림 서버 시뮬레이션 |
| **ubuntu-client** | - | APT 클라이언트 테스트용 |
| **node-client** | - | NPM 클라이언트 테스트용 |
| **python-client** | - | PyPI 클라이언트 테스트용 |
| **docker-client** | - | Docker 클라이언트 테스트용 |
| **test-runner** | - | E2E 테스트 실행기 |

### 네트워크
- **proxynd-test**: 모든 서비스를 연결하는 브리지 네트워크
- 서브넷: `172.20.0.0/16`

### 볼륨
- **proxynd_storage**: ProxyND 스토리지 데이터
- **proxynd_cache**: 캐시 데이터
- **minio_data**: MinIO 데이터
- **docker_data**: Docker-in-Docker 데이터

## 🧪 테스트 종류

### 1. 기본 E2E 테스트
```bash
make test
```
- 헬스체크
- 기본 프록시 기능
- 캐시 동작
- 부하 테스트

### 2. 클라이언트별 테스트
```bash
make test-npm      # NPM 클라이언트 테스트
make test-pip      # PyPI 클라이언트 테스트
make test-apt      # APT 클라이언트 테스트
make test-docker   # Docker 클라이언트 테스트
```

### 3. 전체 테스트
```bash
make test-all      # 모든 테스트 실행
```

## 🛠️ 개발 워크플로우

### 개발 중 테스트
```bash
# 환경 시작
make up

# 개발 작업...

# 테스트 실행
make test

# 로그 확인
make logs-proxynd

# 환경 재시작 (코드 변경 후)
make restart
```

### 디버깅
```bash
# 서비스 상태 확인
make status

# 로그 실시간 보기
make logs-follow

# 컨테이너에 접속
make shell-proxynd
make shell-ubuntu
make shell-node
```

### 헬스체크
```bash
make health
```

## 📂 파일 구조

```
tests/e2e/
├── config/
│   └── config.yaml              # ProxyND 설정
├── scripts/
│   ├── run-e2e-tests.sh        # 메인 E2E 테스트
│   ├── test-npm.sh             # NPM 클라이언트 테스트
│   ├── test-pip.sh             # PyPI 클라이언트 테스트
│   ├── test-apt.sh             # APT 클라이언트 테스트
│   └── test-docker.sh          # Docker 클라이언트 테스트
├── nginx/
│   └── default.conf            # 업스트림 서버 설정
├── upstream-data/
│   ├── npm/                    # NPM 모의 데이터
│   ├── pip/                    # PyPI 모의 데이터
│   ├── apt/                    # APT 모의 데이터
│   └── docker/                 # Docker 모의 데이터
├── workspace/                  # 클라이언트 작업 공간
├── Dockerfile.test             # 테스트 러너 이미지
├── Makefile                    # 테스트 명령들
└── README.md                   # 이 파일
```

## 🎯 테스트 시나리오

### 1. NPM 프록시 테스트
- 패키지 메타데이터 조회
- 스코프 패키지 처리
- tarball 다운로드
- 캐시 동작 확인

### 2. PyPI 프록시 테스트
- Simple API 조회
- 패키지 인덱스 접근
- wheel/tarball 다운로드
- pip 클라이언트 호환성

### 3. APT 프록시 테스트
- Release 파일 조회
- 패키지 목록 다운로드
- .deb 파일 접근
- apt 클라이언트 호환성

### 4. Docker 프록시 테스트
- Registry API v2 호환성
- 매니페스트 조회
- 블롭 다운로드
- 카탈로그 API

### 5. 캐시 테스트
- 첫 요청 vs 캐시된 요청 성능
- TTL 만료 동작
- 캐시 무효화

### 6. 보안 테스트
- IP 필터링
- Basic 인증
- 해시 검증

## 🔧 설정 커스터마이징

### ProxyND 설정 변경
`tests/e2e/config/config.yaml`을 편집하고 환경을 재시작:
```bash
make restart
```

### 업스트림 데이터 추가
`tests/e2e/upstream-data/` 디렉토리에 새 파일 추가

### 포트 변경
`docker-compose.e2e.yml`에서 포트 매핑 수정

## 🐛 문제 해결

### 포트 충돌
```bash
# 사용 중인 포트 확인
netstat -tulpn | grep :8080

# 다른 포트로 변경
# docker-compose.e2e.yml 편집
```

### 컨테이너 빌드 오류
```bash
# 캐시 없이 재빌드
docker-compose -f ../docker-compose.e2e.yml build --no-cache
```

### 볼륨 권한 문제
```bash
# 볼륨 재생성
make clean
make up
```

### 네트워크 문제
```bash
# 네트워크 정보 확인
make network

# Docker 네트워크 재설정
docker network prune
```

## 📊 성능 모니터링

### 메트릭 수집
```bash
# ProxyND 메트릭 (구현된 경우)
curl http://localhost:8080/metrics

# MinIO 메트릭
curl http://localhost:9000/minio/v2/metrics/cluster
```

### 로그 분석
```bash
# 액세스 로그 분석
make shell-proxynd
tail -f /storage/access.log | jq .
```

## 🔄 CI/CD 통합

### GitHub Actions 예시
```yaml
name: E2E Tests
on: [push, pull_request]
jobs:
  e2e:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Run E2E Tests
        run: |
          cd tests/e2e
          make quick-test
```

### Jenkins Pipeline 예시
```groovy
pipeline {
    agent any
    stages {
        stage('E2E Test') {
            steps {
                dir('tests/e2e') {
                    sh 'make quick-test'
                }
            }
        }
    }
}
```

## 📝 기여 가이드

새로운 테스트 시나리오 추가:
1. `scripts/` 디렉토리에 테스트 스크립트 추가
2. 필요한 경우 `upstream-data/`에 모의 데이터 추가
3. `Makefile`에 새 타겟 추가
4. 문서 업데이트

## 📄 라이선스

이 E2E 테스트 환경은 ProxyND 프로젝트와 동일한 라이선스를 따릅니다.