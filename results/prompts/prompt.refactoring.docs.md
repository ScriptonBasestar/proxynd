# ✅ 프록신디(ProxyND) 문서 정리 최적화 프롬프트

## 프로젝트 분석 결과

### 프로젝트 특성
- **언어**: Go 1.23+
- **프레임워크**: Fiber v2 (고성능 웹 프레임워크)
- **프로젝트 유형**: 패키지 매니저 프록시/미러 서버 (모놀리식 아키텍처)
- **플랫폼**: 서버/컨테이너 (Linux, Docker, Kubernetes)
- **대상 사용자**: 시스템 관리자, DevOps 엔지니어, 개발자
- **라이선스**: 오픈소스

### 현재 문서 현황 (총 55개 Markdown 파일)
1. **루트 레벨**: 10개 - BACKLOG.md, CLAUDE.md, README.md, TODO.md 등
2. **docs/**: 30개 - 기능별 가이드 문서들
3. **하위 디렉토리**: 15개 - integration/, helm/, scripts/ 등의 README.md들

### 중복 및 정리 대상 식별
- **설정 관련**: CONFIGURATION_*.md (3개 파일)
- **테스트 관련**: TESTING_*.md, INTEGRATION_*.md (3개 파일)  
- **CI/CD 관련**: CI_CD.md, CICD_GUIDE.md (중복)
- **클라이언트 설정**: *-setup.md (5개), *-CLIENT-SETUP.md (2개)
- **README 파편화**: 8개 디렉토리에 분산된 README.md들

## 실행 프롬프트

### 1. 최적화된 문서 구조 생성

```bash
# 새로운 문서 디렉토리 구조 생성
mkdir -p docs/{01-getting-started,02-architecture,03-configuration,04-proxy-types,05-development,06-deployment,07-monitoring,08-security,09-troubleshooting,10-reference}
mkdir -p docs/01-getting-started/{installation,first-setup}
mkdir -p docs/02-architecture/{design,adr}
mkdir -p docs/03-configuration/{basic,advanced,migration}
mkdir -p docs/04-proxy-types/{maven,npm,apt,docker,pip,yum,apk}
mkdir -p docs/05-development/{testing,mocking,quality,tools}
mkdir -p docs/06-deployment/{docker,kubernetes,systemd}
mkdir -p docs/07-monitoring/{health,metrics,logging}
mkdir -p docs/08-security/{oauth2,verification,authentication}
mkdir -p docs/09-troubleshooting/{common-issues,debugging}
mkdir -p docs/10-reference/{api,cli,configuration}
```

### 2. 문서 이동 및 통합

```bash
# 시작 가이드 통합
mv README.md docs/01-getting-started/README.md
mv docs/docker-setup.md docs/01-getting-started/installation/docker-setup.md
mv docs/DEPLOYMENT.md docs/01-getting-started/installation/deployment-guide.md

# 아키텍처 문서 정리
mv docs/adr/ docs/02-architecture/
touch docs/02-architecture/README.md

# 설정 문서 통합
mv docs/CONFIGURATION_REFERENCE.md docs/03-configuration/reference.md
mv docs/CONFIGURATION_MIGRATION.md docs/03-configuration/migration.md
mv docs/ENVIRONMENT_VARIABLES.md docs/03-configuration/environment-variables.md
mv docs/HOT_RELOAD_GUIDE.md docs/03-configuration/hot-reload.md

# 프록시 타입별 설정 가이드 통합
mv docs/maven-setup.md docs/04-proxy-types/maven/client-setup.md
mv docs/npm-setup.md docs/04-proxy-types/npm/client-setup.md
mv docs/apt-setup.md docs/04-proxy-types/apt/client-setup.md
mv docs/docker-setup.md docs/04-proxy-types/docker/client-setup.md
mv docs/pip-setup.md docs/04-proxy-types/pip/client-setup.md
mv docs/YUM-CLIENT-SETUP.md docs/04-proxy-types/yum/client-setup.md
mv docs/APK-CLIENT-SETUP.md docs/04-proxy-types/apk/client-setup.md
mv docs/APK-PROXY-REQUIREMENTS.md docs/04-proxy-types/apk/requirements.md
mv docs/APK_MIRROR_SELECTION.md docs/04-proxy-types/apk/mirror-selection.md
mv docs/APK_SIGNATURE_VERIFICATION.md docs/04-proxy-types/apk/signature-verification.md

# 개발 문서 정리
mv docs/TESTING_GUIDE.md docs/05-development/testing/README.md
mv docs/INTEGRATION_TESTING_GUIDE.md docs/05-development/testing/integration.md
mv docs/MOCKING_GUIDE.md docs/05-development/mocking/README.md
mv docs/LINTING_GUIDE.md docs/05-development/quality/linting.md
mv docs/PRE_COMMIT_GUIDE.md docs/05-development/quality/pre-commit.md
mv docs/QUALITY_TARGETS.md docs/05-development/quality/targets.md

# 배포 문서 정리
mv docs/MULTIARCH.md docs/06-deployment/docker/multiarch.md
mv systemd/README.md docs/06-deployment/systemd/README.md
mv helm/README.md docs/06-deployment/kubernetes/README.md

# 모니터링 문서 정리
mv docs/HEALTH_CHECK.md docs/07-monitoring/health/README.md
mv docs/METRICS.md docs/07-monitoring/metrics/README.md
mv docs/LOGGING.md docs/07-monitoring/logging/README.md

# 보안 문서 정리
mv docs/OAUTH2_AUTHENTICATION.md docs/08-security/oauth2.md
mv docs/PACKAGE_VERIFICATION.md docs/08-security/verification.md
mv docs/WEBHOOK_EVENTS.md docs/08-security/webhook-events.md

# CLI 및 참조 문서 정리
mv docs/CLI_USAGE_GUIDE.md docs/10-reference/cli/usage-guide.md
mv docs/CLI_COMPLETION.md docs/10-reference/cli/completion.md
mv docs/CLI-TOOL-REQUIREMENTS.md docs/10-reference/cli/requirements.md

# CI/CD 문서 중복 제거 (CICD_GUIDE.md 우선 유지)
rm docs/CI_CD.md
mv docs/CICD_GUIDE.md docs/05-development/tools/cicd-guide.md
mv docs/RELEASE_WORKFLOW.md docs/05-development/tools/release-workflow.md
```

### 3. 기존 문서 정리

```bash
# 프로젝트 루트 정리 (한국어 컨벤션 적용)
mv BACKLOG.md docs/10-reference/project/BACKLOG.md
mv BACKLOG-TO-TODO-SUMMARY.md docs/10-reference/project/backlog-summary.md
mv TODO.md docs/10-reference/project/TODO.md
mv FEATURES.md docs/10-reference/project/FEATURES.md
mv REF.md docs/10-reference/project/references.md
mv copilot.md docs/10-reference/project/copilot.md

# 통합 테스트 문서 정리
mv integration/README.md docs/05-development/testing/integration-setup.md
mv integration/TEST_MAVEN.md docs/04-proxy-types/maven/test-guide.md
mv tests/integration/README.md docs/05-development/testing/integration-local.md
mv tests/e2e/README.md docs/05-development/testing/e2e.md
mv tests/unit/README.md docs/05-development/testing/unit.md

# 기타 README 정리
mv conf/README.md docs/03-configuration/samples.md
mv scripts/README.md docs/05-development/tools/scripts.md
mv internal/testutil/README.md docs/05-development/testing/test-utilities.md
```

### 4. 새로운 통합 문서 생성

```bash
# 메인 README 생성 (한국어)
cat > README.md << 'EOF'
# 프록신디 (ProxyND)

Go로 작성된 고성능 패키지 매니저 프록시/미러 서버

## 빠른 시작

### 1. 개발 환경 설정
```bash
make dev-prepare  # 의존성 설치
make dev-setup    # 설정 디렉토리 준비
make dev-run      # 개발 서버 실행
```

### 2. Docker 실행
```bash
make docker-build
make docker-run
```

## 지원하는 패키지 매니저
- Maven (Java/Kotlin/Scala)
- NPM (Node.js)
- APT (Ubuntu/Debian)
- Docker Registry
- PyPI (Python)
- YUM (RedHat/CentOS)
- APK (Alpine Linux)

## 문서
- [📚 전체 문서](docs/README.md)
- [🚀 시작하기](docs/01-getting-started/README.md)
- [⚙️ 설정 가이드](docs/03-configuration/README.md)
- [🔧 개발 가이드](docs/05-development/README.md)

## 기여하기
[개발 가이드](docs/05-development/README.md)를 참조하세요.

## 라이선스
[LICENSE](LICENSE) 파일을 참조하세요.
EOF

# 문서 인덱스 생성
cat > docs/README.md << 'EOF'
# 프록신디 문서

## 📋 목차

### 🚀 [01. 시작하기](01-getting-started/README.md)
- [설치 가이드](01-getting-started/installation/)
- [첫 설정](01-getting-started/first-setup/)

### 🏗️ [02. 아키텍처](02-architecture/README.md)
- [설계 문서](02-architecture/design/)
- [아키텍처 결정 기록(ADR)](02-architecture/adr/)

### ⚙️ [03. 설정](03-configuration/README.md)
- [기본 설정](03-configuration/basic/)
- [고급 설정](03-configuration/advanced/)
- [설정 마이그레이션](03-configuration/migration.md)

### 📦 [04. 프록시 타입](04-proxy-types/README.md)
- [Maven](04-proxy-types/maven/)
- [NPM](04-proxy-types/npm/)
- [APT](04-proxy-types/apt/)
- [Docker](04-proxy-types/docker/)
- [PyPI](04-proxy-types/pip/)
- [YUM](04-proxy-types/yum/)
- [APK](04-proxy-types/apk/)

### 👨‍💻 [05. 개발](05-development/README.md)
- [테스트](05-development/testing/)
- [코드 품질](05-development/quality/)
- [개발 도구](05-development/tools/)

### 🚀 [06. 배포](06-deployment/README.md)
- [Docker](06-deployment/docker/)
- [Kubernetes](06-deployment/kubernetes/)
- [Systemd](06-deployment/systemd/)

### 📊 [07. 모니터링](07-monitoring/README.md)
- [헬스체크](07-monitoring/health/)
- [메트릭](07-monitoring/metrics/)
- [로깅](07-monitoring/logging/)

### 🔐 [08. 보안](08-security/README.md)
- [OAuth2 인증](08-security/oauth2.md)
- [패키지 검증](08-security/verification.md)

### 🔧 [09. 문제해결](09-troubleshooting/README.md)
- [일반적인 문제](09-troubleshooting/common-issues/)
- [디버깅 가이드](09-troubleshooting/debugging/)

### 📖 [10. 참조](10-reference/README.md)
- [CLI 도구](10-reference/cli/)
- [설정 참조](10-reference/configuration/)
- [프로젝트 문서](10-reference/project/)
EOF

# 각 섹션별 README 생성
for dir in docs/{01-getting-started,02-architecture,03-configuration,04-proxy-types,05-development,06-deployment,07-monitoring,08-security,09-troubleshooting,10-reference}; do
    if [ ! -f "$dir/README.md" ]; then
        section=$(basename "$dir" | cut -d'-' -f2-)
        echo "# $section" > "$dir/README.md"
        echo "" >> "$dir/README.md"
        echo "이 섹션의 하위 문서를 참조하세요." >> "$dir/README.md"
    fi
done
```

### 5. 파일명 표준화 (kebab-case 적용)

```bash
# 파일명을 kebab-case로 변경
find docs/ -name "*.md" -exec bash -c 'dir=$(dirname "$1"); base=$(basename "$1" .md); new=$(echo "$base" | sed "s/_/-/g" | tr "A-Z" "a-z"); [ "$base" != "$new" ] && mv "$1" "$dir/$new.md"' _ {} \;
```

### 6. 중복 파일 정리

```bash
# 빈 디렉토리 정리
find docs/ -type d -empty -delete

# 중복 제거된 파일들 삭제
rm -f docs/ci-cd.md  # CICD_GUIDE.md로 통합됨

# 임시 파일 정리
find . -name "*.tmp" -delete
find . -name "*~" -delete
```

### 7. 검증 및 확인

```bash
# 문서 구조 확인
tree docs/ -I "*.tmp|*~"

# 깨진 링크 확인 (markdown-link-check 설치 필요)
if command -v markdown-link-check >/dev/null 2>&1; then
    find docs/ -name "*.md" -exec markdown-link-check {} \;
else
    echo "markdown-link-check가 설치되지 않음. 링크 검증을 건너뜁니다."
fi

# 문서 파일 개수 확인
echo "정리 후 문서 파일 수: $(find docs/ -name "*.md" | wc -l)"
echo "루트 레벨 문서: $(find . -maxdepth 1 -name "*.md" | wc -l)"
```

## 실행 순서 및 우선순위

### Phase 1: 긴급 (즉시 실행)
1. 문서 구조 생성 (1번)
2. 메인 문서 이동 (2번 일부)
3. 새로운 README.md 생성 (4번 일부)

### Phase 2: 중요 (1주일 내)
1. 프록시 타입별 문서 정리 (2번)
2. 개발/배포 문서 정리 (2번)
3. 중복 파일 제거 (2번, 6번)

### Phase 3: 보완 (1개월 내)
1. 파일명 표준화 (5번)
2. 통합 문서 완성 (4번)
3. 링크 검증 및 정리 (7번)

## 특별 고려사항

### 한국어 프로젝트 특성
- 문서는 기본적으로 한국어로 작성
- 파일명은 영어 kebab-case 사용
- README.md는 한국어와 영어 병행

### 기술 스택 특성
- Go 프로젝트: godoc 호환성 고려
- Fiber 프레임워크: 성능 중심 문서 구성
- Docker/K8s: 컨테이너 배포 문서 중점

### 사용자 타겟
- 시스템 관리자: 설치/배포 문서 우선
- 개발자: 개발/테스트 가이드 상세화
- DevOps: CI/CD 및 모니터링 문서 강화

이 프롬프트를 단계별로 실행하면 프록신디 프로젝트에 최적화된 문서 구조를 구축할 수 있습니다.