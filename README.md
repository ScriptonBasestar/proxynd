# 📦 ProxyND - 라이브러리 전용 프록시 & 캐시 레지스트리

## 소개
ProxyND는 Docker, apt, pip, npm, yum 등 다양한 패키지 매니저의 다운로드 속도 향상과 인증 제거를 위한 프록시 및 캐시 레지스트리 서버입니다.  
사내 네트워크 환경에서 외부 레지스트리에 직접 접근하지 않고, 내부 ProxyND만 우선 연결하도록 설정할 수 있습니다.

---

## ✅ 주요 기능

1. **프록시 다운로드 (On‑demand)**
   - 클라이언트 요청시 등록된 레지스트리에 순차 및 페일오버 방식으로 접근 후 다운로드, 사용자에게 전달

2. **사전 다운로드 (Prefetch)**
   - 사전 정의된 패키지 목록을 미리 내려받아 캐싱 가능  
   - 빠른 배포를 위한 안정적 인프라 제공

3. **기업 정책 관리**
   - 허용된 패키지만 사용하도록 접근 제한 가능  
   - IP 허용 목록, BasicAuth, OAuth 등 인증 방식 지원

4. **보안 검사**
   - 다운로드된 패키지에 대한 무결성 및 취약점 스캔 (예: 해시 검증, SCA 등)  
   - 정책 위반 패키지 차단

5. **스토리지 백엔드**
   - AWS S3 같은 오브젝트 스토리지 연동 가능  
   - 로컬 또는 원격 저장소에 파일 저장 및 조회

---

## 🚀 아키텍처 구성

```
[ 클라이언트 ]
   │
[ ProxyND 서버 ]
   ├─ 요청 라우팅 (Docker, apt, pip, npm 등)
   ├─ 캐시 저장소 (파일 시스템 / S3)
   ├─ 보안 & 인증 모듈
   ├─ 사전 다운로드 스케줄러
   └─ 관리자 웹/CLI (설정 및 접근 제어)
   │
[ 외부 레지스트리 ]
```

---

## 📂 저장소 구성

- **README.md** – 프로젝트 개요 및 기능 설명  
- **go.mod** – Go 모듈 버전 및 의존성 관리  
- **REF.md** – 디자인 참고자료, 패키지 매니저별 플로우, 프로토콜 설명  
- **Dockerfile** – ProxyND 도커 이미지 빌드 설정  
- **cmd/** – `main.go` 포함, 서버 엔트리포인트  
- **pkg/** – 각 기능 모듈(프록시 핸들러, 캐시, 인증, 보안 검사 등)  
- **conf/** – 설정 파일 (YAML/JSON), 기본 포트 및 백엔드 설정  
- **scripts/** – 초기화, 마이그레이션, 관리 스크립트

---

## ⚙️ 설치 및 실행

1. 프로젝트 클론 및 빌드  
   ```bash
   git clone https://github.com/ScriptonBasestar/proxynd.git
   cd proxynd
   go build ./cmd/proxynd
   ```

2. Docker 이미지 생성  
   ```bash
   docker build -t proxynd:latest .
   docker run -d \
     -p 80:80 -p 443:443 \
     -v /data/cache:/var/cache/proxynd \
     -e AUTH_MODE=basicauth \
     -e STORAGE_BACKEND=s3 \
     -e S3_BUCKET=your-bucket \
     proxynd:latest
   ```

3. 설정 (`conf/config.yaml`)  
   ```yaml
   http:
     port: 80
     tls:
       certFile: /path/to/cert
       keyFile: /path/to/key

   registry:
     apt: true
     pip: true
     npm: true
     yum: true

   upstream:
     apt:
       - http://archive.ubuntu.com/ubuntu
     pip:
       - https://pypi.org/simple
     npm:
       - https://registry.npmjs.org

   auth:
     mode: basicauth
     users:
       - username: alice
         password: "$2a$..."
     allowedIPs:
       - 192.168.0.0/16

   cache:
     backend: s3
     s3:
       bucket: your-bucket
       region: ap-northeast-2
   ```

---

## 📖 구성 요소 상세

### 인증 모듈 (auth)
- IP 허용 목록, BasicAuth, OAuth2 를 통한 사용자 인증/인가 처리

### 프록시 핸들러 (proxy)
- HTTP 프로토콜을 중계하며, 각 패키지 매니저 유형에 맞춰 URL 재작성 및 헤더 조작

### 캐시 저장 및 사전 다운로드
- 파일 시스템 / S3에 패키지 저장  
- 비동기 prefetch 스케줄러로 사전 다운로드 수행

### 보안 검사 (security)
- SHA256 해시 검증, 취약점 스캔 도구 연동(SCA), 무결성 및 정책 위반 여부 판단

---

## 🛠️ 확장 및 향후 기능

- **OAuth2 / SAML** 기반 인증 추가  
- **UI 대시보드**로 다운로드 통계, 캐시 상태, 보안 이벤트 리스트 시각화  
- **Webhook 알림**, Slack/Teams 연동  
- **ProxyND간 복제**, 멀티 리전 배포  
- **CI/CD 플러그인** 제공: 자동 사전 다운로드 및 테스트 통합

---

## 🧪 개발자 가이드

- 코드 포맷: `go fmt`, `golangci-lint` 사용  
- 구조: `cmd/`, `pkg/`, `conf/`, `scripts/` 디렉토리 기준  
- 의존성: `go.mod` 통해 관리 (의존성 정리는 `go mod tidy`)  
- Docker: 로컬 테스트는 `docker build` → `docker run`, CI는 `Dockerfile` 기반 이미지 배포

---

## 🎯 기여 안내

- 이슈/요청은 GitHub Issues 활용  
- PR 템플릿 포함 (`.github/` 디렉토리 내부 참고)  
- 리뷰 통과, CI 성공 시 병합

---

## 📄 라이선스

MIT License

---

## 📞 연락처

문의: `contact@scriptonbasestar.com`  
깃허브: [ScriptonBasestar/proxynd](https://github.com/ScriptonBasestar/proxynd)

감사합니다! 😊
