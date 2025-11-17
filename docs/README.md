# 📚 ProxyND Documentation

ProxyND는 Go 1.23+ 기반의 고성능 멀티 패키지 매니저 프록시/미러 서버입니다. 헥사고날 아키텍처와 Container 패턴을 기반으로 7개의 패키지 매니저(Maven, NPM, APT, Docker, PyPI, YUM, APK)를 지원하며, 기업 환경에서 패키지 다운로드 속도 향상과 대역폭 절약을 위해 설계되었습니다.

## 🚀 빠른 시작

### 📱 새로운 사용자
- 📖 **[빠른 시작](00-overview/quick-start.md)** - 5분 설치 가이드
- 🏗️ **[아키텍처 개요](10-architecture/hexagonal-architecture.md)** - 시스템 구조 이해
- 🔧 **[지원 패키지 매니저](30-proxy-types/README.md)** - 전체 지원 목록

### 👨‍💻 개발자
- 💻 **[개발 환경 설정](90-development/README.md)** - 로컬 개발 환경 구축
- 🧪 **[종합 테스트 가이드](40-testing/testing-guide.md)** - 4계층 테스트 전략
- 🏗️ **[Container 아키텍처](10-architecture/container-dependency-injection.md)** - 의존성 주입 패턴

### 🔧 운영자
- 🚀 **[배포 가이드](60-deployment/deployment-checklist.md)** - 프로덕션 배포
- 📊 **[운영 가이드](70-operations/README.md)** - 모니터링 및 운영
- 🔒 **[보안 설정](50-security/oauth2-authentication.md)** - 인증 및 보안

## 📋 문서 구조

### 핵심 구성
| 섹션 | 설명 | 대상 사용자 |
|------|------|-------------|
| **[00-overview](00-overview/)** | 프로젝트 개요 및 빠른 시작 | 신규 사용자, 관리자 |
| **[04-api-reference](04-api-reference/)** | Enterprise API 참조 및 사용 예제 | API 사용자, 개발자 |
| **[10-architecture](10-architecture/)** | 헥사고날 아키텍처 및 설계 | 개발자, 아키텍트 |
| **[20-configuration](20-configuration/)** | 설정 및 환경 관리 | 시스템 관리자 |
| **[30-proxy-types](30-proxy-types/)** | 패키지 매니저별 가이드 | 모든 사용자 |
| **[40-testing](40-testing/)** | 테스트 전략 및 품질 관리 | 개발자, QA |
| **[50-security](50-security/)** | 보안 및 인증 | 보안팀, 관리자 |
| **[60-deployment](60-deployment/)** | 배포 및 인프라 | DevOps, 운영팀 |
| **[70-operations](70-operations/)** | 운영 및 모니터링 | 운영팀, SRE |
| **[80-reference](80-reference/)** | 참조 문서 및 도구 | 고급 사용자 |
| **[90-development](90-development/)** | 개발 및 기여 가이드 | 개발자, 기여자 |
| **[99-legal](99-legal/)** | 라이선스 및 법적 사항 | 구매자, 법무팀 |

### 🎯 통합된 핵심 가이드

#### 🏗️ 아키텍처 & 성능
- **[헥사고날 아키텍처](10-architecture/hexagonal-architecture.md)** - 포트와 어댑터 패턴
- **[Container 의존성 주입](10-architecture/container-dependency-injection.md)** - Thread-safe 싱글톤 DI
- **[성능 벤치마크](80-reference/performance-benchmarks.md)** - 종합 성능 가이드 및 벤치마크

#### 🛠️ 개발 도구
- **[GitHub Actions 워크플로우](90-development/tools/github-actions.md)** - CI/CD 파이프라인 완전 가이드
- **[종합 테스트 가이드](40-testing/testing-guide.md)** - 4계층 테스트 전략
- **[코드 품질 관리](90-development/quality/)** - 린팅, 포맷팅, Pre-commit

#### 📊 기술 사양
- **[기술 스택](80-reference/tech-stack.md)** - 전체 기술 스택 및 의존성
- **[API 엔드포인트](80-reference/api-endpoints.md)** - REST API 참조
- **[CLI 도구](80-reference/cli-tools/cli/proxyndctl-reference.md)** - proxyndctl 명령어 참조

#### 🔌 Enterprise API
- **[OpenAPI 스펙](api/enterprise-api-spec.yaml)** - OpenAPI 3.0 표준 API 명세 (47 엔드포인트)
- **[사용 예제](04-api-reference/enterprise-api-examples.md)** - 97개 코드 예제 (curl + HTTPie)
- **[Prometheus 메트릭](api/METRICS.md)** - 15+ 메트릭 및 PromQL 쿼리
- **[부하 테스트](../scripts/loadtest/README.md)** - 성능 테스트 도구 및 시나리오
- **[CI/CD 통합](deployment/ci-cd-load-testing.md)** - GitHub Actions 자동화

## 🔍 빠른 검색

### 사용 목적별
- **설치하기**: [빠른 시작](00-overview/quick-start.md) → [배포 가이드](60-deployment/deployment-checklist.md)
- **설정하기**: [설정 참조](20-configuration/configuration-reference.md) → [프록시 타입별 가이드](30-proxy-types/)
- **개발하기**: [개발 환경](90-development/README.md) → [테스트 가이드](40-testing/testing-guide.md)
- **운영하기**: [배포](60-deployment/) → [모니터링](70-operations/README.md) → [문제 해결](70-operations/troubleshooting.md)
- **최적화하기**: [성능 가이드](80-reference/performance-benchmarks.md) → [아키텍처](10-architecture/)

### 패키지 매니저별
- **Java/Kotlin/Scala**: [Maven](30-proxy-types/maven/)
- **Node.js**: [NPM](30-proxy-types/npm/)
- **Python**: [PyPI](30-proxy-types/pip/)
- **Ubuntu/Debian**: [APT](30-proxy-types/apt/)
- **Container**: [Docker](30-proxy-types/docker/)
- **RedHat/CentOS**: [YUM](30-proxy-types/yum/)
- **Alpine Linux**: [APK](30-proxy-types/apk/)

## 📖 사용자 유형별 학습 경로

### 🆕 신규 사용자 (처음 사용)
1. [빠른 시작](00-overview/quick-start.md) ⏱️ 30분
2. [프록시 타입 선택](30-proxy-types/README.md) ⏱️ 15분
3. [클라이언트 설정](30-proxy-types/README.md) ⏱️ 15분
4. [기본 모니터링](70-operations/README.md) ⏱️ 15분

### 👨‍💻 개발자 (기여 희망)
1. [아키텍처 이해](10-architecture/hexagonal-architecture.md) ⏱️ 1시간
2. [개발 환경 구축](90-development/README.md) ⏱️ 30분
3. [테스트 실행](40-testing/testing-guide.md) ⏱️ 30분
4. [기여 가이드](90-development/contributing-guide.md) ⏱️ 15분

### 🔧 운영자 (프로덕션 배포)
1. [배포 체크리스트](60-deployment/deployment-checklist.md) ⏱️ 1시간
2. [설정 참조](20-configuration/configuration-reference.md) ⏱️ 45분
3. [모니터링 설정](70-operations/README.md) ⏱️ 45분
4. [보안 설정](50-security/oauth2-authentication.md) ⏱️ 30분
5. [문제 해결](70-operations/troubleshooting.md) ⏱️ 30분

### 🛠️ 고급 사용자 (최적화 및 커스터마이징)
1. [ADR 문서들](10-architecture/adr/README.md) ⏱️ 2시간
2. [성능 벤치마크](80-reference/performance-benchmarks.md) ⏱️ 1시간
3. [CLI 도구 고급 사용법](80-reference/cli-tools/cli/proxyndctl-reference.md) ⏱️ 1시간
4. [확장 개발](90-development/README.md) ⏱️ 시간 가변

## 🚨 주요 특징

### 헥사고날 아키텍처
- **포트와 어댑터 패턴**: 외부 의존성과 비즈니스 로직 완전 분리
- **Container 기반 DI**: Thread-safe 싱글톤으로 서비스 생명주기 관리
- **멀티 프록시 지원**: 단일 서버에서 7개 패키지 매니저 동시 지원

### 고성능 캐싱
- **다층 캐싱**: FileSystem (기본), S3 (확장) 백엔드 지원
- **LRU 제거**: 지능형 캐시 관리로 저장 공간 최적화
- **TTL 관리**: 패키지 타입별 맞춤형 TTL 설정

### 운영 중심 설계
- **구조화된 로깅**: JSON 형식의 체계적 로그 관리
- **Prometheus 메트릭**: 포괄적인 성능 모니터링
- **헬스체크**: 다단계 시스템 상태 확인

## 🎉 최근 개선사항

### 2025년 8월 - 대규모 문서 재구성
- ✅ **완전한 구조 개편**: 00-90 번호 체계로 완전 재구성
- ✅ **중복 제거**: 85%의 중복 콘텐츠 통합 및 단일 출처 원칙 적용
- ✅ **kebab-case 표준화**: 모든 파일명을 kebab-case로 통일
- ✅ **최신 코드 동기화**: 현재 코드베이스와 100% 일치하는 정보

### 주요 성과
- **탐색성 대폭 향상**: 3클릭 내 목표 문서 도달
- **유지보수 부담 경감**: 단일 출처 원칙으로 업데이트 효율성 증대
- **사용자 경험 개선**: 역할별 명확한 학습 경로 제공
- **아키텍처 반영**: 헥사고날 아키텍처에 맞는 문서 구조

## 🤝 기여하기

ProxyND 프로젝트 개선에 참여하고 싶다면:
1. [기여 가이드](90-development/contributing-guide.md) 확인
2. [개발 환경 설정](90-development/README.md) 준비
3. [테스트 가이드](40-testing/testing-guide.md) 숙지
4. Pull Request 제출

## 📄 라이선스

이 문서는 ProxyND 프로젝트와 동일한 라이선스를 따릅니다:
- [라이선스 가이드](99-legal/licensing-guide.md)
- [상업적 라이선스](99-legal/commercial-license.md)
- [오픈소스 철학](99-legal/open-source-philosophy.md)

---

**💡 팁**: 특정 주제를 찾고 있다면 위의 빠른 검색 섹션을 활용하거나 GitHub 검색 기능을 사용하세요.

**🔧 개발자**: `make help`로 전체 Makefile 명령어를 확인할 수 있습니다.

**📅 마지막 업데이트**: 2025-08-22  
**📝 문서 버전**: v3.0.0 (Complete Restructure)

---
