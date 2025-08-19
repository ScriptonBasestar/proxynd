# 📚 ProxyND Documentation

## 🎯 개요

ProxyND는 Go로 작성된 고성능 멀티 패키지 매니저 프록시/미러 서버입니다. 7개의 패키지 매니저(Maven, NPM, APT, Docker, PyPI, YUM, APK)를 지원하며, 기업 환경에서 패키지 다운로드 속도 향상과 대역폭 절약을 위해 사용됩니다.

## 🚀 빠른 시작

### 새로운 사용자
- 📖 **[시작하기](01-getting-started/README.md)** - 설치부터 첫 설정까지
- 🏗️ **[아키텍처 개요](02-architecture/README.md)** - 시스템 구조 이해

### 개발자
- 💻 **[개발 환경 설정](05-development/README.md)** - 로컬 개발 환경 구축
- 🧪 **[종합 테스트 가이드](05-development/testing/00-testing-guide.md)** - 완전한 테스트 전략

### 운영자
- 🚀 **[배포 가이드](06-deployment/README.md)** - 프로덕션 배포
- 📊 **[모니터링](07-monitoring/README.md)** - 성능 모니터링 및 알림

## 📋 문서 구조

### 핵심 구성
| 섹션 | 설명 | 대상 사용자 |
|------|------|-------------|
| **[01-getting-started](01-getting-started/)** | 설치 및 기본 설정 | 신규 사용자, 관리자 |
| **[02-architecture](02-architecture/)** | 시스템 아키텍처 및 설계 | 개발자, 아키텍트 |
| **[03-configuration](03-configuration/)** | 상세 설정 가이드 | 시스템 관리자 |
| **[04-proxy-types](04-proxy-types/)** | 패키지 매니저별 가이드 | 모든 사용자 |
| **[05-development](05-development/)** | 개발 및 기여 가이드 | 개발자 |
| **[06-deployment](06-deployment/)** | 배포 및 운영 | DevOps, 운영팀 |
| **[07-monitoring](07-monitoring/)** | 모니터링 및 관찰성 | 운영팀 |
| **[08-security](08-security/)** | 보안 및 인증 | 보안팀, 관리자 |
| **[09-troubleshooting](09-troubleshooting/)** | 문제 해결 | 모든 사용자 |
| **[10-reference](10-reference/)** | 참조 문서 | 고급 사용자 |
| **[11-legal](11-legal/)** | 라이선스 및 법적 사항 | 구매자, 법무팀 |

### 통합된 핵심 가이드

#### 🏗️ 아키텍처 & 성능
- **[Container 아키텍처](02-architecture/20-container-architecture.md)** - 의존성 주입과 Container 패턴
- **[헥사고널 아키텍처](02-architecture/10-hexagonal-architecture.md)** - 포트와 어댑터 패턴
- **[성능 최적화](10-reference/20-performance.md)** - 종합 성능 가이드 및 벤치마크

#### 🛠️ 개발 도구
- **[GitHub Actions 워크플로우](05-development/tools/30-github-actions.md)** - CI/CD 파이프라인 완전 가이드
- **[종합 테스트 가이드](05-development/testing/00-testing-guide.md)** - 4계층 테스트 전략

#### 📊 기술 사양
- **[기술 스택](10-reference/10-tech-stack.md)** - 전체 기술 스택 및 의존성
- **[API 엔드포인트](10-reference/30-api-endpoints.md)** - REST API 참조
- **[CLI 도구](10-reference/cli/10-proxyndctl-reference.md)** - proxyndctl 명령어 참조

## 🔍 빠른 검색

### 사용 목적별
- **설치하기**: [시작하기](01-getting-started/README.md) → [배포 가이드](06-deployment/README.md)
- **설정하기**: [설정 관리](03-configuration/README.md) → [프록시 타입별 가이드](04-proxy-types/)
- **개발하기**: [개발 가이드](05-development/README.md) → [테스트 가이드](05-development/testing/00-testing-guide.md)
- **운영하기**: [배포](06-deployment/README.md) → [모니터링](07-monitoring/README.md) → [문제 해결](09-troubleshooting/README.md)
- **최적화하기**: [성능 가이드](10-reference/20-performance.md) → [아키텍처](02-architecture/)

### 패키지 매니저별
- **Java/Kotlin/Scala**: [Maven](04-proxy-types/maven/)
- **Node.js**: [NPM](04-proxy-types/npm/)
- **Python**: [PyPI](04-proxy-types/pip/)
- **Ubuntu/Debian**: [APT](04-proxy-types/apt/)
- **Container**: [Docker](04-proxy-types/docker/)
- **RedHat/CentOS**: [YUM](04-proxy-types/yum/)
- **Alpine Linux**: [APK](04-proxy-types/apk/)

## 📖 문서 탐색 가이드

### 1단계: 기본 이해 (30분)
1. [프로젝트 개요](01-getting-started/README.md) 읽기
2. [아키텍처 개요](02-architecture/README.md) 훑어보기
3. 사용할 [패키지 매니저 가이드](04-proxy-types/) 확인

### 2단계: 설치 및 설정 (1시간)
1. [설치 가이드](01-getting-started/README.md) 따라하기
2. [기본 설정](03-configuration/README.md) 적용
3. [첫 프록시 설정](04-proxy-types/) 테스트

### 3단계: 고급 활용 (상황별)
- **개발자**: [개발 환경](05-development/README.md) → [테스트](05-development/testing/00-testing-guide.md)
- **운영자**: [배포](06-deployment/README.md) → [모니터링](07-monitoring/README.md)
- **최적화**: [성능 가이드](10-reference/20-performance.md) → [아키텍처 심화](02-architecture/)

## 🎉 최근 문서 개선사항

### 2024년 12월 - 대규모 문서 재구성
- ✅ **통합 및 중복 제거**: Container 아키텍처, 성능, GitHub Actions, 테스트 문서 통합
- ✅ **구조 개선**: 루트 레벨 문서를 적절한 섹션으로 이동
- ✅ **네이밍 표준화**: 모든 파일명을 kebab-case로 통일
- ✅ **내용 동기화**: 코드베이스 변경사항 반영

### 주요 성과
- **문서 수 감소**: 중복 제거로 유지보수 부담 경감
- **탐색성 향상**: 명확한 계층 구조와 번호 체계
- **내용 품질**: 최신 코드베이스와 동기화된 정확한 정보
- **사용성 개선**: 사용자 경험 중심의 문서 구조

## 🤝 기여하기

문서 개선에 기여하고 싶다면:
1. [기여 가이드](05-development/10-contributing.md) 확인
2. 이슈 생성 또는 Pull Request 제출
3. [문서 작성 표준](05-development/README.md) 준수

## 📄 라이선스

이 문서는 ProxyND 프로젝트와 동일한 라이선스를 따릅니다:
- [라이선스 가이드](11-legal/10-licensing-guide.md)
- [상업적 라이선스](11-legal/20-commercial-license.md)

---

**💡 팁**: 특정 주제를 찾고 있다면 [문서 인덱스](00-index.md)를 활용하거나 GitHub 검색 기능을 사용하세요.