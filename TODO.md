# 🎉 TODO.md - MVP 완료!

**ProxyND v1.0.0 MVP의 모든 필수 기능이 완료되었습니다!**

---

## 📋 현재 작업 목록

- [x] `yum` 프록시 지원 추가 (CentOS, RHEL, Rocky Linux 등) ✅

### 🏔️ Alpine APK 프록시 지원
- [x] APK 프록시 요구사항 정리 (APK 레포지토리 구조 분석, APKINDEX 포맷 이해)
- [x] `/configs/apk_proxy_config.go` 설정 구조체 구현
- [x] `/handlers/proxy/apk_proxy_controller.go` 핸들러 구현 (APKINDEX.tar.gz 처리 포함)
- [x] 통합 프록시 핸들러에 APK 라우팅 추가
- [x] `/sample-conf/apk-proxy.yaml` 샘플 설정 작성 (main, community, testing 저장소)
- [x] `/docs/APK-CLIENT-SETUP.md` 클라이언트 설정 가이드 작성
- [ ] APK 서명 검증 로직 구현 (`.SIGN.RSA.*` 파일 처리)
- [ ] Alpine 버전별 미러 자동 선택 기능 구현

### 🛠️ ProxyND CLI 도구 (proxyndctl)
- [ ] CLI 도구 요구사항 정의 (주요 명령어 목록, 사용자 시나리오 정리)
- [ ] `/cmd/proxyndctl/main.go` 기본 구조 생성 (cobra/urfave 라이브러리 선택)
- [ ] 캐시 관리 명령어 구현 (`cache list`, `cache clear`, `cache size`)
- [ ] 설정 검증 명령어 구현 (`config validate`, `config show`)
- [ ] 서버 상태 확인 명령어 구현 (`status`, `health`, `metrics`)
- [ ] 사용자 관리 명령어 구현 (`user add`, `user delete`, `user list`)
- [ ] 프록시 테스트 명령어 구현 (`test apt`, `test npm` 등)
- [ ] 자동완성 스크립트 생성 (bash, zsh, fish)
- [ ] man page 및 help 문서 작성
- [ ] 바이너리 배포 워크플로우 구성 (GitHub Releases)

### 🔔 Webhook 알림 시스템
- [ ] Webhook 이벤트 타입 정의 (캐시 만료, 인증 실패, 정책 위반, 서버 상태 변경 등)
- [ ] `/configs/webhook_config.go` 설정 구조체 구현 (URL, 이벤트 필터, 재시도 정책)
- [ ] `/internal/webhook/sender.go` 핵심 전송 로직 구현 (비동기 큐, 재시도 메커니즘)
- [ ] Slack webhook 어댑터 구현 (`/internal/webhook/adapters/slack.go`)
- [ ] Discord webhook 어댑터 구현 (`/internal/webhook/adapters/discord.go`)
- [ ] Generic webhook 어댑터 구현 (커스텀 JSON 포맷)
- [ ] 이벤트 버퍼링 및 배치 전송 기능 구현 (rate limiting 대응)
- [ ] Webhook 전송 실패 시 로컬 저장 및 재시도 큐 구현
- [ ] Webhook 설정 테스트 엔드포인트 추가 (`/api/webhook/test`)
- [ ] Webhook 전송 이력 및 통계 API 구현

### ⏱️ 패키지별 TTL 고도화
- [ ] 글로벌 설정에 패키지 타입별 기본 TTL 추가 (`apt: 3600`, `npm: 1800` 등)
- [ ] 패키지 패턴별 TTL 오버라이드 설정 구현 (`*-SNAPSHOT: 300`, `*-dev: 600`)
- [ ] 메타데이터 파일 전용 TTL 설정 (`repomd.xml: 300`, `Packages.gz: 600`)
- [ ] 캐시 헤더 기반 동적 TTL 계산 로직 구현 (Cache-Control, Expires 헤더 파싱)
- [ ] TTL 만료 시 백그라운드 갱신 옵션 추가 (stale-while-revalidate)
- [ ] `/api/cache/ttl` 엔드포인트로 현재 TTL 정책 조회 API 구현
- [ ] TTL 통계 메트릭 수집 (평균 TTL, 조기 만료율 등)

### 🔐 OAuth2 인증 시스템
- [ ] OAuth2 인증 플로우 설계 문서 작성 (Authorization Code, Client Credentials 플로우 선택)
- [ ] `/configs/oauth2_config.go` 설정 구조체 구현 (provider 정보, client ID/secret, 콜백 URL)
- [ ] `/internal/auth/oauth2/provider.go` 공통 인터페이스 정의
- [ ] GitHub OAuth2 프로바이더 구현 (`/internal/auth/oauth2/github.go`)
- [ ] GitLab OAuth2 프로바이더 구현 (`/internal/auth/oauth2/gitlab.go`)
- [ ] Google OAuth2 프로바이더 구현 (`/internal/auth/oauth2/google.go`)
- [ ] OAuth2 콜백 핸들러 구현 (`/auth/callback/:provider`)
- [ ] JWT 토큰 발급 및 검증 로직 구현 (access/refresh token)
- [ ] 기존 BasicAuth와 OAuth2 통합 미들웨어 구현
- [ ] OAuth2 사용자 정보 매핑 및 권한 동기화 로직 구현
- [ ] 토큰 만료 시 자동 갱신 메커니즘 구현
- [ ] OAuth2 로그인 페이지 UI 템플릿 작성

### 🛡️ 취약점 스캔 통합
- [ ] 취약점 스캔 요구사항 분석 (스캔 대상, 정책, 차단 기준 정의)
- [ ] `/configs/security_scan_config.go` 설정 구조체 구현 (스캐너 선택, 정책 설정)
- [ ] `/internal/security/scanner.go` 공통 스캐너 인터페이스 정의
- [ ] Trivy 스캐너 통합 구현 (`/internal/security/scanners/trivy.go`)
- [ ] 컨테이너 이미지 스캔 미들웨어 구현 (Docker 프록시 전용)
- [ ] 패키지 다운로드 시 실시간 취약점 검사 로직 구현
- [ ] 취약점 발견 시 차단/경고 정책 엔진 구현
- [ ] 스캔 결과 캐싱 메커니즘 구현 (동일 패키지 반복 스캔 방지)
- [ ] `/api/security/scan/:type/:package` 수동 스캔 API 구현
- [ ] 취약점 리포트 생성 기능 구현 (JSON, HTML, PDF 포맷)
- [ ] 취약점 데이터베이스 자동 업데이트 스케줄러 구현
- [ ] 스캔 성능 메트릭 수집 (스캔 시간, 발견 취약점 수 등)

### 🎨 웹 관리자 대시보드
- [ ] 대시보드 UI/UX 설계 및 와이어프레임 작성
- [ ] 프론트엔드 프레임워크 선정 및 프로젝트 초기화 (React/Svelte 중 선택)
- [ ] 관리 API 엔드포인트 설계 (`/api/admin/*`)
- [ ] 대시보드 메인 페이지 구현 (서버 상태, 캐시 히트율, 활성 사용자 수)
- [ ] 캐시 관리 페이지 구현 (사용량 차트, 히트/미스 통계, 수동 제거)
- [ ] 사용자 관리 페이지 구현 (목록, 추가/삭제, 권한 편집)
- [ ] 프록시 설정 관리 페이지 구현 (실시간 설정 편집, 검증)
- [ ] 실시간 로그 뷰어 구현 (WebSocket 기반 스트리밍)
- [ ] 패키지 검색 및 상세 정보 페이지 구현
- [ ] 알림 설정 관리 페이지 구현 (Webhook, 이메일 설정)
- [ ] 다크 모드 및 반응형 디자인 구현
- [ ] 역할 기반 접근 제어 (관리자/뷰어 권한 분리)
- [ ] 대시보드 성능 최적화 (lazy loading, 데이터 페이징)

### 🚀 CI/CD Prefetch 통합
- [ ] Prefetch 요구사항 분석 (지원할 패키지 매니저, 의존성 파일 포맷)
- [ ] `/cmd/proxyndprefetch/main.go` CLI 도구 기본 구조 구현
- [ ] 의존성 파일 파서 구현 (package.json, requirements.txt, pom.xml, go.mod)
- [ ] ProxyND 서버와 통신하는 클라이언트 라이브러리 구현
- [ ] `prefetch` 명령어 구현 (병렬 다운로드, 진행률 표시)
- [ ] `prefetch-analyze` 명령어 구현 (의존성 트리 분석, 크기 예측)
- [ ] GitHub Actions 워크플로우 템플릿 작성 (`action.yml`)
- [ ] GitHub Actions 마켓플레이스 배포 준비 (문서, 아이콘, 메타데이터)
- [ ] GitLab CI/CD 템플릿 작성 (`.gitlab-ci.yml` 예제)
- [ ] Jenkins Pipeline 스크립트 템플릿 작성
- [ ] 캐시 워밍업 스케줄러 구현 (정기적 prefetch 실행)
- [ ] Prefetch 통계 API 구현 (절약된 시간, 대역폭 등)

### ☕ Maven 프록시 고도화
- [ ] 기존 Maven 프록시 기능 분석 및 개선점 도출
- [ ] Maven 메타데이터 병합 로직 구현 (다중 저장소 통합)
- [ ] SNAPSHOT 버전 자동 갱신 메커니즘 구현
- [ ] POM 파일 의존성 분석 및 선행 다운로드 기능 구현
- [ ] Maven 저장소 인덱스 생성 및 검색 기능 구현
- [ ] 아티팩트 서명 검증 로직 구현 (`.asc` 파일 처리)
- [ ] Maven 설정 자동 생성기 구현 (`settings.xml` 템플릿)
- [ ] 빌드 캐시 통합 지원 (Gradle Enterprise 캐시 호환)
- [ ] 아티팩트 업로드 지원 (deploy 명령어 프록시)
- [ ] Maven Central 동기화 스케줄러 구현

### 🔄 멀티 인스턴스 캐시 복제
- [ ] 캐시 복제 아키텍처 설계 문서 작성 (복제 토폴로지, 충돌 해결 전략)
- [ ] `/configs/replication_config.go` 설정 구조체 구현 (피어 목록, 복제 정책)
- [ ] 인스턴스 디스커버리 메커니즘 구현 (정적 설정, DNS-SD, Consul 연동)
- [ ] 캐시 변경 이벤트 감지 시스템 구현 (파일시스템 워처, S3 이벤트)
- [ ] P2P 복제 프로토콜 구현 (gRPC 기반 스트리밍)
- [ ] 복제 큐 및 우선순위 관리 시스템 구현
- [ ] 대역폭 제한 및 스케줄링 기능 구현 (피크 시간 회피)
- [ ] 복제 상태 모니터링 API 구현 (`/api/replication/status`)
- [ ] 부분 복제 및 필터링 기능 구현 (특정 패키지 타입만 복제)
- [ ] 복제 충돌 해결 로직 구현 (타임스탬프, 버전 비교)
- [ ] 복제 실패 시 재시도 및 복구 메커니즘 구현
- [ ] 네트워크 분할 상황 대응 로직 구현

### 📧 이메일 알림 시스템
- [ ] 이메일 알림 요구사항 정의 (알림 유형, 템플릿, 수신자 관리)
- [ ] `/configs/email_config.go` 설정 구조체 구현 (SMTP 서버, 인증 정보)
- [ ] `/internal/notification/email/sender.go` 이메일 발송 엔진 구현
- [ ] 이메일 템플릿 시스템 구현 (HTML/텍스트 듀얼 포맷)
- [ ] 알림 이벤트별 이메일 템플릿 작성 (인증 실패, 정책 위반, 시스템 알림)
- [ ] 이메일 큐잉 시스템 구현 (대량 발송 시 rate limiting)
- [ ] 수신자 그룹 관리 기능 구현 (관리자, 사용자, 커스텀 그룹)
- [ ] 이메일 발송 이력 저장 및 조회 API 구현
- [ ] 반송 메일 처리 로직 구현 (bounce handling)
- [ ] 이메일 구독 관리 페이지 구현 (opt-in/out)
- [ ] 다국어 이메일 템플릿 지원
- [ ] 이메일 발송 통계 대시보드 구현

### 📝 감사 로그 시스템
- [ ] 감사 로그 요구사항 분석 (규정 준수, 보관 기간, 검색 요구사항)
- [ ] `/internal/audit/logger.go` 감사 로그 핵심 엔진 구현
- [ ] 감사 이벤트 타입 정의 (인증, 인가, 데이터 접근, 설정 변경)
- [ ] 구조화된 감사 로그 포맷 설계 (JSON, CEF, LEEF 지원)
- [ ] 감사 로그 저장소 어댑터 구현 (파일, 데이터베이스, S3)
- [ ] 감사 로그 무결성 보장 메커니즘 구현 (체크섬, 디지털 서명)
- [ ] 감사 로그 검색 API 구현 (`/api/audit/search`)
- [ ] 감사 로그 보관 및 순환 정책 구현 (자동 아카이빙)
- [ ] 실시간 감사 로그 스트리밍 엔드포인트 구현 (WebSocket)
- [ ] 감사 로그 분석 대시보드 구현 (이상 패턴 감지)
- [ ] SIEM 통합을 위한 로그 포워딩 기능 구현
- [ ] 감사 로그 익스포트 기능 구현 (CSV, PDF 리포트)

### 🦀 Cargo (Rust) 프록시 지원
- [ ] Cargo 레지스트리 프로토콜 분석 (crates.io API, 인덱스 구조)
- [ ] `/configs/cargo_proxy_config.go` 설정 구조체 구현
- [ ] `/handlers/proxy/cargo_proxy_controller.go` 핸들러 구현
- [ ] Cargo 인덱스 Git 저장소 미러링 기능 구현
- [ ] Crate 다운로드 API 엔드포인트 구현 (`/api/v1/crates`)
- [ ] Cargo 검색 API 프록시 구현 (`/api/v1/crates/search`)
- [ ] `.crate` 파일 캐싱 및 체크섬 검증 구현
- [ ] 통합 프록시 핸들러에 Cargo 라우팅 추가
- [ ] `/sample-conf/cargo-proxy.yaml` 샘플 설정 작성
- [ ] `/docs/CARGO-CLIENT-SETUP.md` 클라이언트 설정 가이드 작성
- [ ] Sparse 인덱스 프로토콜 지원 추가 (Rust 1.68+)
- [ ] 프라이빗 크레이트 레지스트리 지원

### 🐹 Go Modules 프록시 지원
- [ ] Go 모듈 프록시 프로토콜 분석 (GOPROXY 사양, 체크섬 데이터베이스)
- [ ] `/configs/go_proxy_config.go` 설정 구조체 구현
- [ ] `/handlers/proxy/go_proxy_controller.go` 핸들러 구현
- [ ] Go 모듈 메타데이터 API 구현 (`/@v/list`, `/@v/{version}.info`)
- [ ] 모듈 다운로드 엔드포인트 구현 (`/@v/{version}.mod`, `/@v/{version}.zip`)
- [ ] Go 체크섬 데이터베이스 프록시 구현 (sum.golang.org)
- [ ] `go.sum` 항목 검증 및 캐싱 로직 구현
- [ ] 프라이빗 모듈 인증 지원 (Git 자격 증명 통합)
- [ ] 통합 프록시 핸들러에 Go 라우팅 추가
- [ ] `/sample-conf/go-proxy.yaml` 샘플 설정 작성
- [ ] `/docs/GO-CLIENT-SETUP.md` 클라이언트 설정 가이드 작성
- [ ] 모듈 버전 목록 자동 갱신 기능 구현
- [ ] VCS 직접 프록시 지원 (GitHub, GitLab 등)

### ⎈ Helm Chart 레포지토리 프록시
- [ ] Helm 차트 레포지토리 프로토콜 분석 (index.yaml, 차트 패키징)
- [ ] `/configs/helm_proxy_config.go` 설정 구조체 구현
- [ ] `/handlers/proxy/helm_proxy_controller.go` 핸들러 구현
- [ ] Helm 인덱스 파일 병합 로직 구현 (다중 레포지토리 통합)
- [ ] 차트 다운로드 및 캐싱 엔드포인트 구현
- [ ] 차트 버전 메타데이터 API 구현
- [ ] 차트 서명 검증 로직 구현 (`.prov` 파일 처리)
- [ ] OCI 레지스트리 기반 Helm 차트 지원
- [ ] 통합 프록시 핸들러에 Helm 라우팅 추가
- [ ] `/sample-conf/helm-proxy.yaml` 샘플 설정 작성
- [ ] `/docs/HELM-CLIENT-SETUP.md` 클라이언트 설정 가이드 작성
- [ ] 차트 의존성 자동 다운로드 기능 구현
- [ ] 프라이빗 차트 레포지토리 인증 지원

### 👥 역할 기반 접근 제어 (RBAC)
- [ ] RBAC 모델 설계 문서 작성 (역할, 권한, 리소스 정의)
- [ ] `/internal/auth/rbac/model.go` 역할 및 권한 모델 구현
- [ ] 기본 역할 정의 (Admin, Maintainer, Developer, Viewer)
- [ ] 권한 매트릭스 구현 (리소스 × 작업 × 역할)
- [ ] 사용자-역할 매핑 저장소 구현 (DB/파일 기반)
- [ ] 그룹 관리 시스템 구현 (LDAP/AD 통합 준비)
- [ ] 권한 검증 미들웨어 구현 (세밀한 접근 제어)
- [ ] 역할 상속 및 계층 구조 지원
- [ ] `/api/rbac/*` 역할 관리 REST API 구현
- [ ] 권한 위임 기능 구현 (임시 권한 부여)
- [ ] 감사 로그와 RBAC 통합 (권한 변경 추적)
- [ ] RBAC 정책 백업 및 복원 기능
- [ ] 권한 충돌 검사 및 리포트 도구

### 🚀 고성능 인덱스 캐시 시스템
- [ ] 인덱스 캐시 요구사항 분석 (성능 목표, 캐시 키 설계)
- [ ] `/internal/cache/index/interface.go` 캐시 인터페이스 정의
- [ ] Redis 어댑터 구현 (`/internal/cache/index/redis.go`)
- [ ] Memcached 어댑터 구현 (`/internal/cache/index/memcached.go`)
- [ ] 캐시 키 생성 전략 구현 (패키지 타입별 최적화)
- [ ] 메타데이터 직렬화/역직렬화 최적화 (protobuf/msgpack)
- [ ] 캐시 워밍업 기능 구현 (자주 사용되는 패키지 사전 로드)
- [ ] 캐시 무효화 전략 구현 (이벤트 기반, TTL 기반)
- [ ] 분산 캐시 일관성 보장 메커니즘 구현
- [ ] 캐시 히트율 모니터링 및 자동 튜닝
- [ ] 폴백 메커니즘 구현 (캐시 서버 장애 시)
- [ ] 캐시 압축 옵션 구현 (메모리 효율성)

### 📊 Prometheus + Grafana 모니터링 대시보드
- [ ] ProxyND 메트릭 체계 설계 (핵심 지표 정의)
- [ ] 커스텀 Prometheus 메트릭 추가 구현
- [ ] 메트릭 레이블 표준화 (proxy_type, cache_status, user 등)
- [ ] Grafana 대시보드 JSON 템플릿 작성 - 개요 대시보드
- [ ] Grafana 대시보드 JSON 템플릿 작성 - 캐시 성능 대시보드
- [ ] Grafana 대시보드 JSON 템플릿 작성 - 사용자 활동 대시보드
- [ ] Grafana 대시보드 JSON 템플릿 작성 - 에러 및 알림 대시보드
- [ ] 알림 규칙 템플릿 작성 (높은 에러율, 캐시 부족 등)
- [ ] 메트릭 문서화 작성 (`/docs/METRICS.md`)
- [ ] Docker Compose 모니터링 스택 구성 파일 작성
- [ ] Prometheus 서비스 디스커버리 설정 예제
- [ ] 장기 메트릭 보관을 위한 remote storage 설정 가이드

### 🔑 SAML SSO 인증 시스템
- [ ] SAML 2.0 프로토콜 요구사항 분석 (IdP 지원 범위, 속성 매핑)
- [ ] `/configs/saml_config.go` 설정 구조체 구현 (IdP 메타데이터, SP 설정)
- [ ] `/internal/auth/saml/provider.go` SAML 서비스 프로바이더 구현
- [ ] SAML 메타데이터 생성 엔드포인트 구현 (`/saml/metadata`)
- [ ] SAML SSO 엔드포인트 구현 (`/saml/sso`, `/saml/acs`)
- [ ] SAML 어설션 검증 및 속성 추출 로직 구현
- [ ] 주요 IdP 통합 가이드 작성 (Okta, Azure AD, Ping Identity)
- [ ] SAML 로그아웃 (SLO) 지원 구현
- [ ] 사용자 속성 매핑 및 자동 프로비저닝 구현
- [ ] SAML 인증서 관리 시스템 구현 (자동 갱신)
- [ ] 기존 인증 시스템과 SAML 통합 미들웨어 구현
- [ ] SAML 디버깅 도구 및 로그 구현

### 🎫 JWT 기반 API 토큰 시스템
- [ ] JWT 토큰 요구사항 정의 (클레임 구조, 만료 정책, 갱신 전략)
- [ ] `/internal/auth/jwt/token.go` JWT 생성 및 검증 엔진 구현
- [ ] RSA/ECDSA 키 쌍 관리 시스템 구현 (자동 로테이션)
- [ ] API 토큰 발급 엔드포인트 구현 (`/api/auth/token`)
- [ ] 토큰 갱신 엔드포인트 구현 (`/api/auth/refresh`)
- [ ] 토큰 폐기 목록(Revocation List) 관리 시스템 구현
- [ ] JWT 미들웨어 구현 (토큰 검증, 권한 확인)
- [ ] 토큰 스코프 및 권한 시스템 구현
- [ ] 개인 액세스 토큰(PAT) 관리 UI 구현
- [ ] 토큰 사용 통계 및 감사 로그 구현
- [ ] JWK(JSON Web Key) 엔드포인트 구현 (`/.well-known/jwks.json`)
- [ ] 토큰 만료 알림 시스템 구현

### 📦 시스템 패키지 배포
- [ ] 패키지 배포 요구사항 분석 (대상 OS, 버전, 의존성)
- [ ] Debian 패키지(.deb) 빌드 스크립트 작성 (`/build/deb/`)
- [ ] RPM 패키지 빌드 스크립트 작성 (`/build/rpm/`)
- [ ] Alpine 패키지(.apk) 빌드 스크립트 작성 (`/build/apk/`)
- [ ] 패키지별 post-install/pre-remove 스크립트 작성
- [ ] systemd 서비스 유닛 파일 생성 및 패키징
- [ ] 패키지 서명 자동화 (GPG 키 관리)
- [ ] 패키지 저장소 구성 (APT/YUM 레포지토리)
- [ ] Nix 패키지 정의 파일 작성 (`default.nix`)
- [ ] Homebrew Formula 작성 및 탭 설정
- [ ] 패키지 자동 빌드 CI/CD 파이프라인 구성
- [ ] 패키지 테스트 자동화 (설치/업그레이드/제거)
- [ ] 패키지 배포 문서 작성 (`/docs/INSTALLATION.md`)

### 🎮 관리용 REST API
- [ ] 관리 API 요구사항 및 스펙 정의 (OpenAPI 3.0)
- [ ] `/api/v1/admin/*` 라우트 구조 설계
- [ ] 서버 상태 관리 API 구현 (`GET/POST /api/v1/admin/server`)
- [ ] 캐시 관리 API 구현 (`GET/DELETE /api/v1/admin/cache`)
- [ ] 설정 관리 API 구현 (`GET/PUT /api/v1/admin/config`)
- [ ] 사용자 관리 API 구현 (`CRUD /api/v1/admin/users`)
- [ ] 프록시 정책 관리 API 구현 (`CRUD /api/v1/admin/policies`)
- [ ] 통계 조회 API 구현 (`GET /api/v1/admin/stats`)
- [ ] 벌크 작업 API 구현 (일괄 사용자 추가, 캐시 정리 등)
- [ ] API 버전 관리 및 하위 호환성 유지 전략 구현
- [ ] OpenAPI 문서 자동 생성 및 Swagger UI 통합
- [ ] API 클라이언트 SDK 생성 (Go, Python, JavaScript)

### 🔍 SCA 리포트 자동 생성
- [ ] SCA 요구사항 분석 (SBOM 포맷, 리포트 유형, 규정 준수)
- [ ] `/internal/sca/analyzer.go` SCA 분석 엔진 구현
- [ ] SBOM 생성기 구현 (SPDX, CycloneDX 포맷 지원)
- [ ] 패키지 의존성 그래프 분석 로직 구현
- [ ] 라이선스 분석 및 충돌 검사 기능 구현
- [ ] 알려진 취약점 매핑 기능 구현 (CVE 데이터베이스 연동)
- [ ] 리포트 템플릿 시스템 구현 (HTML, PDF, JSON)
- [ ] 정기 스캔 스케줄러 구현 (cron 기반)
- [ ] `/api/sca/generate` 온디맨드 리포트 생성 API
- [ ] 리포트 이력 관리 및 비교 기능 구현
- [ ] 컴플라이언스 정책 검증 엔진 구현
- [ ] SCA 대시보드 UI 구현 (리포트 조회, 트렌드 분석)

### 🛡️ 패키지 무결성 보호
- [ ] 패키지 해시 변경 감지 요구사항 정의 (정책, 대응 방안)
- [ ] `/internal/integrity/monitor.go` 해시 모니터링 엔진 구현
- [ ] 패키지 해시 이력 저장소 구현 (버전별 해시 추적)
- [ ] 실시간 해시 검증 미들웨어 구현
- [ ] 해시 변경 감지 시 알림 시스템 구현 (이메일, Webhook)
- [ ] 변경된 패키지 자동 차단 메커니즘 구현
- [ ] 해시 화이트리스트 관리 기능 구현 (예외 처리)
- [ ] 변경 이유 분석 도구 구현 (정상/비정상 구분)
- [ ] `/api/integrity/verify` 수동 검증 API 구현
- [ ] 해시 변경 이력 조회 대시보드 구현
- [ ] 정책 기반 대응 엔진 구현 (경고/차단/격리)
- [ ] 포렌식 데이터 수집 기능 구현 (변경 시점, 소스 등)

### 🌍 Geo Replication 시스템
- [ ] 멀티 리전 아키텍처 설계 문서 작성 (토폴로지, 데이터 흐름)
- [ ] `/internal/geo/replication.go` 지역 복제 엔진 구현
- [ ] 리전별 노드 디스커버리 메커니즘 구현
- [ ] 지리적 위치 기반 라우팅 로직 구현 (GeoDNS 연동)
- [ ] 리전 간 데이터 동기화 프로토콜 구현 (최적화된 대역폭)
- [ ] 리전별 캐시 정책 차별화 기능 구현
- [ ] 크로스 리전 장애 복구(Failover) 메커니즘 구현
- [ ] 리전별 성능 메트릭 수집 및 분석 시스템
- [ ] 데이터 지역성 규정 준수 기능 구현 (GDPR 등)
- [ ] 리전 간 일관성 보장 메커니즘 구현 (최종 일관성)
- [ ] 글로벌 대시보드 구현 (전체 리전 상태 모니터링)
- [ ] 리전별 비용 최적화 엔진 구현

### 💾 캐시 스냅샷 백업/복원
- [ ] 스냅샷 요구사항 분석 (백업 주기, 보관 정책, 복원 시나리오)
- [ ] `/internal/snapshot/manager.go` 스냅샷 관리 엔진 구현
- [ ] 증분 백업 알고리즘 구현 (변경된 부분만 백업)
- [ ] 스냅샷 압축 및 암호화 기능 구현
- [ ] 다중 백업 대상 지원 (로컬, S3, NFS, FTP)
- [ ] 스냅샷 메타데이터 관리 시스템 구현
- [ ] 백업 스케줄러 구현 (정기 백업, 이벤트 기반 백업)
- [ ] 스냅샷 검증 도구 구현 (무결성 확인)
- [ ] 선택적 복원 기능 구현 (특정 패키지/시점 복원)
- [ ] 백업 보관 주기 자동 관리 (오래된 스냅샷 정리)
- [ ] `/api/snapshot/*` 백업/복원 관리 API 구현
- [ ] 재해 복구 시뮬레이션 도구 구현
- [ ] 백업 성능 최적화 (병렬 처리, 중복 제거)

### 🧹 고급 캐시 정리 전략
- [ ] 캐시 정리 전략 요구사항 분석 (사용 패턴, 성능 목표)
- [ ] `/internal/cache/eviction/strategy.go` 전략 인터페이스 정의
- [ ] LFU (Least Frequently Used) 알고리즘 구현
- [ ] FIFO (First In First Out) 알고리즘 구현
- [ ] 2Q (Two Queue) 알고리즘 구현
- [ ] ARC (Adaptive Replacement Cache) 알고리즘 구현
- [ ] 패키지 중요도 기반 가중치 시스템 구현
- [ ] 예측 기반 사전 정리 기능 구현 (ML 모델 활용)
- [ ] 다단계 캐시 정리 정책 구현 (경고→정리→긴급정리)
- [ ] 캐시 사용 패턴 분석 도구 구현
- [ ] 전략별 성능 비교 벤치마크 도구
- [ ] 동적 전략 전환 메커니즘 구현
- [ ] 캐시 파편화 방지 및 정리 기능

### 🗑️ 오래된 버전 자동 정리
- [ ] 버전 정리 정책 요구사항 정의 (보관 기준, 예외 규칙)
- [ ] `/internal/cache/version/cleaner.go` 버전 정리 엔진 구현
- [ ] 패키지별 버전 히스토리 추적 시스템 구현
- [ ] 시맨틱 버전 기반 정리 규칙 엔진 구현 (메이저/마이너/패치)
- [ ] 최신 N개 버전 유지 정책 구현
- [ ] 시간 기반 버전 정리 정책 구현 (X일 이상 된 버전)
- [ ] 의존성 체인 분석 기능 구현 (사용 중인 버전 보호)
- [ ] 버전 정리 예외 목록 관리 기능 (중요 버전 보호)
- [ ] 정리 작업 스케줄러 구현 (저사용 시간대 실행)
- [ ] 정리 전 백업 옵션 구현
- [ ] `/api/version/cleanup` 수동 정리 트리거 API
- [ ] 정리 작업 시뮬레이션 모드 구현 (dry-run)
- [ ] 버전 사용 통계 기반 지능형 정리

### 🔄 GitOps Helm 자동화
- [ ] GitOps 워크플로우 요구사항 정의 (Git 저장소 구조, 승인 프로세스)
- [ ] Helm Chart 템플릿 고도화 (values 파일 구조화)
- [ ] ArgoCD Application 매니페스트 작성
- [ ] Flux v2 Kustomization 리소스 작성
- [ ] Git 저장소 Webhook 리스너 구현
- [ ] Helm values 자동 생성기 구현 (환경별 설정)
- [ ] 배포 승인 워크플로우 구현 (PR 기반)
- [ ] 롤백 자동화 스크립트 작성
- [ ] 배포 상태 모니터링 대시보드 통합
- [ ] 시크릿 관리 통합 (Sealed Secrets, SOPS)
- [ ] 멀티 클러스터 배포 지원
- [ ] 배포 이력 및 감사 로그 통합
- [ ] Canary/Blue-Green 배포 전략 지원

### ✋ 패키지 승인 워크플로우
- [ ] 승인 워크플로우 요구사항 정의 (승인자, 정책, SLA)
- [ ] `/internal/approval/engine.go` 승인 엔진 구현
- [ ] 패키지 요청 큐 시스템 구현 (대기/승인/거부 상태)
- [ ] 승인 정책 규칙 엔진 구현 (자동/수동 승인 조건)
- [ ] 승인자 알림 시스템 구현 (이메일, Slack, Teams)
- [ ] 승인 요청 UI 페이지 구현 (요청 상세, 위험도 표시)
- [ ] 일괄 승인/거부 기능 구현
- [ ] 승인 위임 및 에스컬레이션 메커니즘
- [ ] 승인 이력 및 감사 추적 구현
- [ ] 자동 승인 화이트리스트 관리
- [ ] 승인 대기 시간 초과 처리
- [ ] 긴급 승인 바이패스 옵션 (break-glass)
- [ ] 승인 통계 및 SLA 모니터링

---

## ✅ 완료 상태

**모든 MVP 작업이 성공적으로 완료**되어 다음 문서로 이관되었습니다:

### 📋 문서 구조
- **FEATURES.md**: 완료된 모든 기능의 사용자 중심 설명서
- **BACKLOG.md**: 완료된 MVP 작업 이력 + 향후 개발 계획
- **REF.md**: 기술 참조 문서 (패키지 흐름, 헤더 구조)
- **docs/**: 패키지별 클라이언트 설정 가이드

---

## 🚀 주요 달성 사항

✅ **4개 패키지 레지스트리 지원** (APT, NPM, PIP, Docker)  
✅ **멀티 백엔드 캐시 시스템** (파일시스템, S3)  
✅ **포괄적인 보안 기능** (인증, 권한, 패키지 검증)  
✅ **운영 도구** (메트릭, 헬스체크, 구조화된 로깅)  
✅ **멀티플랫폼 배포** (Docker, Kubernetes, systemd)  
✅ **완전한 문서화** (설정 가이드, API 참조)  

---

## 📚 다음 단계

1. **FEATURES.md** 확인 → 구현된 모든 기능 살펴보기
2. **docs/** 폴더 → 각 패키지 매니저별 클라이언트 설정 방법
3. **BACKLOG.md** → 향후 버전에서 구현할 추가 기능들

---

> **🎯 이제 ProxyND는 프로덕션 환경에서 사용할 준비가 완료되었습니다!**  
> 추가 기능 요청이나 개선 사항은 BACKLOG.md에 추가하여 향후 버전에서 검토합니다.
