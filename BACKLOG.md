# 🧭 BACKLOG.md

이 문서는 ProxyND의 향후 버전에 도입될 기능들과 완료된 MVP 작업 이력을 정리한 백로그입니다.  
우선순위와 개발 시점은 실제 사용 피드백 및 운영 환경에 따라 조정될 수 있습니다.

---

## ✅ 완료된 MVP 기능 (v1.0.0)

**이 섹션은 이미 완료되어 FEATURES.md에 문서화된 기능들입니다.**

### 🧭 핵심 구조
- ✅ 프록시 요청 라우터 구성 (`/proxy/:type/*path`)
- ✅ 프록시 정책 처리 미들웨어 (캐시 hit/miss, 인증, 정책 적용)

### 📦 지원 레지스트리 타입 (1차 릴리스)  
- ✅ APT (Ubuntu, Debian 등 APT 프록시 처리)
- ✅ PIP (PyPI)
- ✅ NPM (npmjs)
- ✅ Docker (Docker registry v2)

### ⚙️ 캐시 관리
- ✅ 캐시 스토리지 인터페이스 정의 (CacheBackend)
- ✅ 로컬 파일 시스템 캐시 드라이버
- ✅ S3 캐시 드라이버 (버킷 설정 가능)
- ✅ TTL/만료 정책: LRU, max size 기반 정리 기능

### 🔐 인증/인가/접근 제어
- ✅ IP 화이트리스트 + CIDR 기반 필터
- ✅ BasicAuth (설정파일 기반)
- ✅ 사용자별 권한 정책 (읽기/쓰기/삭제 허용)
- ✅ 허용된 패키지 목록 기반 필터 (AllowList)

### 🛡️ 보안 및 검증
- ✅ SHA256 해시 검증 (패키지 메타와 비교)
- ✅ 요청 응답 기록 로깅 (access.log)
- ✅ 패키지 검증 실패 시 차단/알림

### 🧪 테스트 프레임워크
- ✅ 통합 테스트: 실제 패키지 요청을 모사하는 시나리오 테스트
- ✅ 단위 테스트: 라우팅, 캐시 hit, 인증 등
- ✅ 로컬 e2e 테스트용 Docker Compose 환경 제공

### 📚 설정 및 구성 관리
- ✅ config.yaml 구조 정의 (서버, 레지스트리, 인증, 캐시 설정)
- ✅ 환경 변수 오버라이드 지원
- ✅ 핫리로드 지원 (SIGHUP)

### 🧰 운영 및 관측
- ✅ /metrics 엔드포인트 (Prometheus 포맷)
- ✅ /healthz 상태 체크 API  
- ✅ structured logging (JSON + leveled log)

### 🐳 실행 및 배포
- ✅ 다중 아키텍처 Docker 이미지 빌드 (amd64, arm64)
- ✅ CI 파이프라인에서 자동 테스트 & 빌드
- ✅ Helm chart or systemd 서비스 템플릿 제공

### 📄 문서화
- ✅ README.md (기능, 설치법, 아키텍처 설명)
- ✅ conf/config.example.yaml
- ✅ REF.md: APT/PIP/NPM 패키지 흐름 및 헤더 구조 설명
- ✅ 패키지별 설정 예시 (APT sources.list 등)

---

## 🚀 향후 개발 계획

## 📦 추가 지원 레지스트리

- [ ] `yum` (CentOS, RHEL, Rocky 등)
- [ ] `apk` (Alpine Linux)
- [ ] `maven` (Java 생태계)
- [ ] `cargo` (Rust)
- [ ] `go proxy` (Go modules)
- [ ] `helm` (Helm chart repository)

---

## 🔒 인증 및 권한 고도화

- [ ] OAuth2 지원 (GitHub, GitLab, Google 등)
- [ ] SAML 연동 (SSO 인증)
- [ ] JWT 기반 API 토큰 발급/검증
- [ ] 사용자 그룹/역할(Role) 기반 권한 제어

---

## 🔐 보안 및 감사 기능

- [ ] 취약점 스캔 엔진 연동 (e.g. Trivy, Snyk)
- [ ] SCA (소프트웨어 구성 분석) 리포트 자동 생성
- [ ] 요청 감사 로그 저장 (AuditLog)
- [ ] 패키지 hash 변경 알림 또는 차단

---

## 🧰 관리 및 운영 도구

- [ ] 웹 기반 관리자 대시보드 (React/Svelte 기반 UI)
  - 사용자 관리, 정책 관리, 캐시 통계 시각화
- [ ] CLI 도구 (`proxyndctl`) 제공
  - 사용자 추가/삭제, 캐시 클리어, 상태 확인 등
- [ ] 관리용 REST API 엔드포인트

---

## 🔁 프록시 미러링 / 복제

- [ ] 다중 ProxyND 인스턴스 간 캐시 복제 기능
- [ ] 멀티 리전 대응을 위한 Geo Replication
- [ ] 스냅샷 기반 캐시 백업/복원 기능

---

## 🕒 캐시 및 스토리지 고도화

- [ ] 패키지별 TTL 설정
- [ ] max size 기반 자동 정리 (LRU 외 전략)
- [ ] redis/memcached 기반 인덱스 캐시
- [ ] 삭제 정책: 오래된 버전 자동 제거

---

## 📣 알림 및 통합

- [ ] Webhook 통합 (Slack, Discord, Mattermost)
- [ ] 이메일 알림 (정책 위반, 인증 실패 등)
- [ ] API 서버 상태 Prometheus + Grafana 대시보드 템플릿 제공

---

## 📦 CI/CD 통합

- [ ] prefetch CLI 및 GitHub Actions 통합
  - 특정 커밋에 필요한 패키지 사전 다운로드
- [ ] GitOps 기반 Helm Chart 설치 자동화
- [ ] 패키지 요청 시 Webhook → 승인 → 프록시 처리

---

## 🧱 시스템 패키징 및 배포

- [ ] `.deb`, `.rpm`, `.apk` 패키지 배포
- [ ] Helm Chart 완성형 제공
- [ ] Nix/NixOS 지원
- [ ] Homebrew formula (macOS 지원)

---

## 🎯 연구 및 장기 목표

- [ ] WASM 기반 경량 실행 환경으로 프록시 core 재구성
- [ ] eBPF 기반 트래픽 제어 및 감시
- [ ] 공급망 공격 대응용 SBOM 추출 및 분석
- [ ] Immutable 패키지 정책 + 서명체계 도입

---

> 이 문서의 모든 항목은 구현 전 기술 검토 및 논의를 거쳐 우선순위가 조정될 수 있습니다.
