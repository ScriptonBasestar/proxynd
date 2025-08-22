# ProxyND 문서 인덱스

ProxyND 프로젝트의 전체 문서를 체계적으로 탐색할 수 있는 중앙 인덱스입니다.

## 🚀 빠른 시작

### 처음 사용자
- [📘 시작하기](01-getting-started/README.md) - ProxyND 설치 및 첫 설정
- [🏗️ 아키텍처 개요](02-architecture/README.md) - 전체 구조 이해

### 개발자
- [💻 개발 환경 설정](05-development/README.md) - 로컬 개발 환경 구축
- [🧪 테스트 가이드](05-development/testing/README.md) - 테스트 실행 및 작성

### 운영자
- [🚀 배포 가이드](06-deployment/README.md) - 프로덕션 배포
- [📊 모니터링](07-monitoring/README.md) - 성능 모니터링 및 알림

## 📚 주요 문서 섹션

### 1. [시작하기](01-getting-started/)
- **대상**: 신규 사용자, 시스템 관리자
- **내용**: 설치, 기본 설정, 첫 프록시 구성
- **소요 시간**: 30분 - 1시간

### 2. [아키텍처](02-architecture/)
- **대상**: 개발자, 아키텍트, 고급 사용자
- **내용**: 시스템 구조, 설계 결정, ADR 문서
- **주요 문서**:
  - [ADR 목록](02-architecture/adr/README.md) - 설계 결정 기록

### 3. [설정 관리](03-configuration/)
- **대상**: 시스템 관리자, DevOps 엔지니어
- **내용**: 상세 설정 옵션, 환경 변수, 핫 리로드
- **주요 문서**:
  - [설정 참조](03-configuration/reference.md) - 전체 설정 옵션
  - [환경 변수](03-configuration/environment-variables.md) - 환경별 설정

### 4. [프록시 타입별 가이드](04-proxy-types/)
각 패키지 매니저별 상세 설정 및 클라이언트 구성:
- [Maven](04-proxy-types/maven/) - Java/Kotlin/Scala 패키지
- [NPM](04-proxy-types/npm/) - Node.js 패키지  
- [APT](04-proxy-types/apt/) - Ubuntu/Debian 패키지
- [Docker](04-proxy-types/docker/) - 컨테이너 이미지
- [PyPI](04-proxy-types/pip/) - Python 패키지
- [YUM](04-proxy-types/yum/) - RedHat/CentOS 패키지
- [APK](04-proxy-types/apk/) - Alpine Linux 패키지

### 5. [개발 가이드](05-development/)
- **대상**: 기여자, 개발자
- **내용**: 개발 환경, 테스트, 코드 품질, CI/CD
- **주요 문서**:
  - [테스트](05-development/testing/) - 단위/통합/E2E 테스트
  - [품질 관리](05-development/quality/) - 린팅, 포맷팅, 프리커밋 훅

### 6. [배포](06-deployment/)
- **대상**: DevOps 엔지니어, 시스템 관리자
- **내용**: Docker, Kubernetes, Systemd 배포
- **주요 문서**:
  - [Kubernetes](06-deployment/kubernetes/) - K8s 배포 가이드
  - [Docker](06-deployment/docker/) - 컨테이너 배포

### 7. [모니터링](07-monitoring/)
- **대상**: SRE, 운영팀
- **내용**: 메트릭, 로깅, 헬스체크, 알림
- **주요 문서**:
  - [메트릭](07-monitoring/metrics/) - Prometheus 메트릭
  - [로깅](07-monitoring/logging/) - 구조화된 로깅

### 8. [보안](08-security/)
- **대상**: 보안 담당자, 시스템 관리자
- **내용**: 인증, 권한, OAuth2, 보안 검증
- **주요 문서**:
  - [OAuth2](08-security/oauth2.md) - OAuth2 설정
  - [검증](08-security/verification.md) - 패키지 서명 검증

### 9. [문제 해결](09-troubleshooting/)
- **대상**: 모든 사용자
- **내용**: 일반적인 문제, 디버깅, FAQ
- **빠른 해결**: 자주 발생하는 문제들의 해결책

### 10. [참조 문서](10-reference/)
- **대상**: 모든 사용자 (참조용)
- **내용**: CLI 도구, API, 성능 가이드
- **주요 문서**:
  - [CLI 도구](10-reference/cli/) - proxyndctl 명령어 참조
  - [Maven 인덱스 관리](10-reference/cli/maven-index.md) - Maven 검색 인덱스
  - [Maven 백업 관리](10-reference/cli/maven-backup.md) - Maven 증분 백업
  - [성능 최적화](10-reference/performance-optimization.md) - 성능 튜닝
  - **🚀 미래 확장 계획**:
    - [로드맵](10-reference/future-roadmap.md) - 장기 확장 계획
    - [배치 시스템](10-reference/batch-system-design.md) - 작업 자동화
    - [플러그인 시스템](10-reference/plugin-system-design.md) - 확장성
    - [대화형 모드](10-reference/interactive-mode-design.md) - 사용성 개선

## 🎯 사용자 유형별 학습 경로

### 🆕 신규 사용자 (처음 사용)
1. [빠른 시작](01-getting-started/README.md) ⏱️ 30분
2. [프록시 타입 선택](04-proxy-types/README.md) ⏱️ 15분
3. [클라이언트 설정](04-proxy-types/*/client-setup.md) ⏱️ 15분
4. [기본 모니터링](07-monitoring/README.md) ⏱️ 15분

### 👨‍💻 개발자 (기여 희망)
1. [아키텍처 이해](02-architecture/README.md) ⏱️ 1시간
2. [개발 환경 구축](05-development/README.md) ⏱️ 30분
3. [테스트 실행](05-development/testing/README.md) ⏱️ 30분
4. [기여 가이드](CONTRIBUTING.md) ⏱️ 15분

### 🔧 운영자 (프로덕션 배포)
1. [배포 가이드](06-deployment/README.md) ⏱️ 1시간
2. [설정 참조](03-configuration/reference.md) ⏱️ 45분
3. [모니터링 설정](07-monitoring/README.md) ⏱️ 45분
4. [보안 설정](08-security/README.md) ⏱️ 30분
5. [문제 해결](09-troubleshooting/README.md) ⏱️ 30분

### 🛠️ 고급 사용자 (최적화 및 커스터마이징)
1. [ADR 문서들](02-architecture/adr/README.md) ⏱️ 2시간
2. [성능 최적화](10-reference/performance-optimization.md) ⏱️ 1시간
3. [CLI 도구 고급 사용법](10-reference/cli/10-proxyndctl-reference.md) ⏱️ 1시간
4. [확장 개발](05-development/README.md) ⏱️ 시간 가변

## 🔍 빠른 검색

### 명령어로 찾기
- **설치**: [빠른 시작 → 설치](01-getting-started/README.md#설치)
- **설정**: [설정 관리](03-configuration/README.md)
- **테스트**: [개발 → 테스트](05-development/testing/README.md)
- **배포**: [배포 가이드](06-deployment/README.md)
- **문제 해결**: [트러블슈팅](09-troubleshooting/README.md)

### 기술별 찾기
- **Docker**: [Docker 프록시](04-proxy-types/docker/) | [Docker 배포](06-deployment/docker/)
- **Kubernetes**: [K8s 배포](06-deployment/kubernetes/)
- **Maven**: [Maven 프록시](04-proxy-types/maven/)
- **NPM**: [NPM 프록시](04-proxy-types/npm/)
- **CLI**: [proxyndctl 참조](10-reference/cli/)

### 역할별 찾기
- **관리자**: [설정](03-configuration/) | [배포](06-deployment/) | [모니터링](07-monitoring/)
- **개발자**: [개발 가이드](05-development/) | [아키텍처](02-architecture/)
- **사용자**: [시작하기](01-getting-started/) | [클라이언트 설정](04-proxy-types/*/client-setup.md)

## 📖 추가 리소스

### 외부 링크
- [GitHub 저장소](https://github.com/scriptonbasestar/proxynd)
- [이슈 트래커](https://github.com/scriptonbasestar/proxynd/issues)
- [릴리스 노트](https://github.com/scriptonbasestar/proxynd/releases)

### 커뮤니티
- [기여 가이드](../CONTRIBUTING.md)
- [코드 오브 컨덕트](../CODE_OF_CONDUCT.md)
- [라이선스](../LICENSE)

---

**💡 팁**: 이 인덱스는 정기적으로 업데이트됩니다. 문서를 찾기 어렵거나 개선 제안이 있으시면 [이슈를 생성](https://github.com/scriptonbasestar/proxynd/issues/new)해 주세요.

**📅 마지막 업데이트**: 2025-01-01  
**📝 문서 버전**: v2.0.0
