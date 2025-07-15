# 📚 프록신디 문서 INDEX

> **ProxyND Documentation Index** - 프록신디의 모든 문서를 체계적으로 정리한 통합 인덱스입니다.

---

## 📋 전체 목차

### 🚀 [01. 시작하기](01-getting-started/README.md)
프록신디를 처음 사용하는 사용자를 위한 가이드
- [📥 설치 가이드](01-getting-started/installation/)
- [⚙️ 첫 설정](01-getting-started/first-setup/)

### 🏗️ [02. 아키텍처](02-architecture/README.md)
프록신디의 설계와 구조에 대한 문서
- [🎯 설계 문서](02-architecture/design/)
- [📝 아키텍처 결정 기록(ADR)](02-architecture/adr/)

### ⚙️ [03. 설정](03-configuration/README.md)
프록신디의 설정 방법과 옵션들
- [🔧 기본 설정](03-configuration/basic/)
- [🔬 고급 설정](03-configuration/advanced/)
- [📊 설정 참조](03-configuration/reference.md)
- [🔄 설정 마이그레이션](03-configuration/migration.md)
- [🌍 환경 변수](03-configuration/environment-variables.md)
- [🔥 핫 리로드](03-configuration/hot-reload.md)
- [📋 설정 샘플](03-configuration/samples.md)

### 📦 [04. 프록시 타입](04-proxy-types/README.md)
지원하는 각 패키지 매니저별 설정 가이드
- [☕ Maven](04-proxy-types/maven/) - Java/Kotlin/Scala 패키지
  - [📖 클라이언트 설정](04-proxy-types/maven/client-setup.md)
  - [🧪 테스트 가이드](04-proxy-types/maven/test-guide.md)
- [📦 NPM](04-proxy-types/npm/) - Node.js 패키지
  - [📖 클라이언트 설정](04-proxy-types/npm/client-setup.md)
- [🐧 APT](04-proxy-types/apt/) - Ubuntu/Debian 패키지
  - [📖 클라이언트 설정](04-proxy-types/apt/client-setup.md)
- [🐳 Docker](04-proxy-types/docker/) - 컨테이너 이미지
  - [📖 클라이언트 설정](04-proxy-types/docker/client-setup.md)
- [🐍 PyPI](04-proxy-types/pip/) - Python 패키지
  - [📖 클라이언트 설정](04-proxy-types/pip/client-setup.md)
- [📦 YUM](04-proxy-types/yum/) - RedHat/CentOS 패키지
  - [📖 클라이언트 설정](04-proxy-types/yum/client-setup.md)
- [🏔️ APK](04-proxy-types/apk/) - Alpine Linux 패키지
  - [📖 클라이언트 설정](04-proxy-types/apk/client-setup.md)
  - [📋 요구사항](04-proxy-types/apk/requirements.md)
  - [🪞 미러 선택](04-proxy-types/apk/mirror-selection.md)
  - [🔐 서명 검증](04-proxy-types/apk/signature-verification.md)

### 👨‍💻 [05. 개발](05-development/README.md)
프록신디 개발과 기여를 위한 가이드
- [🧪 테스트](05-development/testing/)
  - [📖 테스트 가이드](05-development/testing/README.md)
  - [🔗 통합 테스트](05-development/testing/integration.md)
  - [⚙️ 통합 테스트 설정](05-development/testing/integration-setup.md)
  - [🏠 로컬 통합 테스트](05-development/testing/integration-local.md)
  - [🌐 E2E 테스트](05-development/testing/e2e.md)
  - [🔧 단위 테스트](05-development/testing/unit.md)
  - [🛠️ 테스트 유틸리티](05-development/testing/test-utilities.md)
- [🎭 모킹](05-development/mocking/)
  - [📖 모킹 가이드](05-development/mocking/README.md)
- [✨ 코드 품질](05-development/quality/)
  - [🔍 린팅](05-development/quality/linting.md)
  - [✅ Pre-commit](05-development/quality/pre-commit.md)
  - [🎯 품질 목표](05-development/quality/targets.md)
- [🛠️ 개발 도구](05-development/tools/)
  - [🚀 CI/CD 가이드](05-development/tools/cicd-guide.md)
  - [📦 릴리스 워크플로](05-development/tools/release-workflow.md)
  - [📜 스크립트](05-development/tools/scripts.md)

### 🚀 [06. 배포](06-deployment/README.md)
프록신디의 다양한 배포 방법
- [🐳 Docker](06-deployment/docker/)
  - [🏗️ 멀티아키텍처](06-deployment/docker/multiarch.md)
- [☸️ Kubernetes](06-deployment/kubernetes/)
  - [📖 Helm 차트](06-deployment/kubernetes/README.md)
- [⚙️ Systemd](06-deployment/systemd/)
  - [📖 시스템 서비스](06-deployment/systemd/README.md)

### 📊 [07. 모니터링](07-monitoring/README.md)
프록신디의 상태 모니터링과 관찰
- [💚 헬스체크](07-monitoring/health/)
  - [📖 헬스체크 가이드](07-monitoring/health/README.md)
- [📈 메트릭](07-monitoring/metrics/)
  - [📖 메트릭 수집](07-monitoring/metrics/README.md)
- [📝 로깅](07-monitoring/logging/)
  - [📖 로그 설정](07-monitoring/logging/README.md)

### 🔐 [08. 보안](08-security/README.md)
인증, 인가 및 보안 관련 설정
- [🔑 OAuth2 인증](08-security/oauth2.md)
- [✅ 패키지 검증](08-security/verification.md)
- [🔗 웹훅 이벤트](08-security/webhook-events.md)

### 🔧 [09. 문제해결](09-troubleshooting/README.md)
일반적인 문제와 해결 방법
- [❓ 일반적인 문제](09-troubleshooting/common-issues/)
- [🐛 디버깅 가이드](09-troubleshooting/debugging/)

### 📖 [10. 참조](10-reference/README.md)
API, CLI 및 프로젝트 참조 문서
- [🖥️ CLI 도구](10-reference/cli/)
  - [📖 사용법 가이드](10-reference/cli/usage-guide.md)
  - [⌨️ 자동완성](10-reference/cli/completion.md)
  - [📋 요구사항](10-reference/cli/requirements.md)
- [🔧 API](10-reference/api/)
- [⚙️ 설정 참조](10-reference/configuration/)
- [📁 프로젝트 문서](10-reference/project/)
  - [📋 백로그](10-reference/project/backlog.md)
  - [📝 백로그 요약](10-reference/project/backlog-summary.md)
  - [✅ TODO](10-reference/project/todo.md)
  - [🎯 기능 목록](10-reference/project/features.md)
  - [🔗 참조 자료](10-reference/project/references.md)
  - [🤖 Copilot 가이드](10-reference/project/copilot.md)

---

## 🗂️ 문서 분류

### 📚 사용자 가이드
신규 사용자와 시스템 관리자를 위한 문서
- [시작하기](01-getting-started/README.md)
- [설정 가이드](03-configuration/README.md)
- [프록시 타입별 설정](04-proxy-types/README.md)
- [배포 가이드](06-deployment/README.md)

### 🔧 개발자 가이드
개발자와 기여자를 위한 문서
- [아키텍처](02-architecture/README.md)
- [개발 가이드](05-development/README.md)
- [참조 문서](10-reference/README.md)

### 🚨 운영 가이드
시스템 운영자를 위한 문서
- [모니터링](07-monitoring/README.md)
- [보안](08-security/README.md)
- [문제해결](09-troubleshooting/README.md)

---

## 🔍 빠른 찾기

### 자주 찾는 문서
- **새로 시작하기**: [시작하기 가이드](01-getting-started/README.md)
- **Docker 설정**: [Docker 배포](06-deployment/docker/)
- **Maven 프록시**: [Maven 클라이언트 설정](04-proxy-types/maven/client-setup.md)
- **테스트 실행**: [테스트 가이드](05-development/testing/README.md)
- **CLI 사용법**: [CLI 가이드](10-reference/cli/usage-guide.md)

### 트러블슈팅
- **일반적인 문제**: [문제해결 가이드](09-troubleshooting/README.md)
- **설정 오류**: [설정 참조](03-configuration/reference.md)
- **연결 문제**: [헬스체크](07-monitoring/health/README.md)

---

> 📌 **팁**: 이 문서는 정기적으로 업데이트됩니다. 최신 버전은 GitHub에서 확인하세요.

**마지막 업데이트**: 2025-07-10