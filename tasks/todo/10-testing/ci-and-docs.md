# CI 매트릭스 및 문서 개선 계획

## 📋 개요

GitHub Actions CI 매트릭스를 최적화하고 테스트 관련 문서를 개선하여 개발 효율성을 높입니다.

## 🎯 목표

- CI 매트릭스 병렬화 및 최적화
- 테스트 실행 시간 단축
- 포괄적인 테스트 문서 제공
- 개발자 온보딩 개선

## 🚀 CI 매트릭스 개선

### 현재 CI 구성 분석
**파일**: `.github/workflows/ci.yml`

**현재 매트릭스**:
```yaml
strategy:
  matrix:
    proxy-type: [maven, npm, apt, docker, pip, yum, apk]
  fail-fast: false
```

### 개선된 매트릭스 구조

#### 1. 계층화된 테스트 매트릭스
```yaml
# 빠른 피드백을 위한 핵심 테스트
quick-tests:
  matrix:
    test-type: [unit, lint, format]
    
# 프록시별 통합 테스트  
integration-tests:
  matrix:
    proxy-type: [maven, npm, apt, docker, pip, yum, apk]
    go-version: ['1.21', '1.22']
    
# 플랫폼별 E2E 테스트
e2e-tests:
  matrix:
    proxy-type: [maven, npm, docker] # 핵심 타입만
    os: [ubuntu-latest, macos-latest]
```

#### 2. 조건부 실행 최적화
```yaml
# PR에서는 필수 테스트만
on:
  pull_request:
    paths:
      - '**.go'
      - 'go.mod'
      - 'go.sum'
      - '.github/workflows/**'

# 메인 브랜치에서는 전체 테스트      
on:
  push:
    branches: [main, develop]
```

#### 3. 캐시 최적화
```yaml
# Go 모듈 캐시
- uses: actions/cache@v4
  with:
    path: |
      ~/.cache/go-build
      ~/go/pkg/mod
    key: ${{ runner.os }}-go-${{ hashFiles('**/go.sum') }}
    
# 테스트 픽스처 캐시
- uses: actions/cache@v4  
  with:
    path: tests/e2e/upstream-data
    key: e2e-fixtures-${{ hashFiles('tests/e2e/upstream-data/**') }}
```

## 📚 문서 개선 계획

### 1. 테스트 실행 가이드
**위치**: `docs/05-development/testing/`

#### 새로운 문서 구조
```
docs/05-development/testing/
├── README.md                 # 테스트 개요
├── unit-testing.md          # 단위 테스트 가이드  
├── integration-testing.md   # 통합 테스트 가이드
├── e2e-testing.md          # E2E 테스트 가이드
├── performance-testing.md   # 성능 테스트 가이드
├── troubleshooting.md      # 테스트 문제 해결
└── ci-local.md            # 로컬 CI 실행
```

#### README.md 구조
```markdown
# ProxyND 테스트 가이드

## 빠른 시작
- 전체 테스트: `make test-all`
- 단위 테스트: `make test-unit`  
- 통합 테스트: `make test-integration`
- E2E 테스트: `make test-e2e`

## 테스트 타입별 가이드
- [단위 테스트](unit-testing.md)
- [통합 테스트](integration-testing.md)  
- [E2E 테스트](e2e-testing.md)
- [성능 테스트](performance-testing.md)

## 개발 워크플로우
1. 코드 변경
2. 관련 테스트 실행
3. 테스트 추가/수정
4. 전체 테스트 확인
5. PR 제출
```

### 2. 프록시 타입별 테스트 가이드

#### Maven 테스트 가이드
**파일**: `docs/04-proxy-types/maven/test-guide.md`
```markdown
# Maven 프록시 테스트 가이드

## 통합 테스트
- 아티팩트 다운로드: `go test -v -run TestMavenProxy_Artifact`
- SNAPSHOT 처리: `go test -v -run TestMavenProxy_Snapshot`
- 브라우저 인터페이스: `go test -v -run TestMavenProxy_Browser`

## E2E 테스트  
- 실제 mvn 명령어: `tests/e2e/scripts/test-maven.sh`
- Docker 환경: `docker-compose -f tests/e2e/docker-compose.yml up`
```

#### NPM 테스트 가이드
**파일**: `docs/04-proxy-types/npm/test-guide.md`
```markdown
# NPM 프록시 테스트 가이드

## 통합 테스트
- 패키지 메타데이터: `go test -v -run TestNPMProxy_Metadata`
- 타르볼 다운로드: `go test -v -run TestNPMProxy_Tarball`
- 스코프 패키지: `go test -v -run TestNPMProxy_Scoped`

## E2E 테스트
- 실제 npm 명령어: `tests/e2e/scripts/test-npm.sh`
- 레지스트리 설정: `npm config set registry http://localhost:8080/proxy/npm`
```

### 3. CI/CD 문서화

#### CI 가이드
**파일**: `docs/05-development/tools/cicd-guide.md`
```markdown
# CI/CD 파이프라인 가이드

## GitHub Actions 워크플로우
- 품질 검사: 린팅, 포맷팅, 보안 스캔
- 테스트 매트릭스: 프록시 타입별 병렬 실행
- 빌드 검증: 멀티아키텍처 Docker 빌드

## 로컬 CI 실행
```bash
# 전체 CI 파이프라인 로컬 실행
make ci-local

# 특정 워크플로우 실행
act -j test-matrix
```
```

### 4. 성능 테스트 문서

#### 성능 테스트 가이드  
**파일**: `docs/05-development/testing/performance-testing.md`
```markdown
# 성능 테스트 가이드

## 벤치마크 테스트
```bash
# 프록시별 성능 벤치마크
go test -bench=BenchmarkProxy -benchmem ./tests/...

# 특정 프록시 타입
go test -bench=BenchmarkNPMProxy -benchmem ./tests/integration/
```

## 부하 테스트
```bash
# 동시 요청 테스트
go test -v -run TestConcurrentRequests ./tests/integration/

# 장시간 부하 테스트  
go test -v -run TestLongRunning -timeout=30m ./tests/integration/
```

## 성능 메트릭
- 응답 시간: 캐시 HIT < 10ms, MISS < 5s
- 처리량: > 1000 req/sec
- 메모리: 안정적인 메모리 사용량
- CPU: 효율적인 CPU 활용
```

## 🔧 CI 최적화 구현

### 1. 병렬 실행 개선
```yaml
# 현재: 순차 실행 (7 × 실행시간)
# 개선: 병렬 실행 (max(실행시간들))

jobs:
  test-matrix:
    strategy:
      matrix:
        include:
          - proxy: npm
            priority: high
          - proxy: maven  
            priority: high
          - proxy: docker
            priority: medium
          # ...
```

### 2. 조건부 실행 구현
```yaml
# 변경된 파일에 따른 선택적 테스트 실행
- name: Check changed files
  uses: dorny/paths-filter@v2
  with:
    filters: |
      npm:
        - 'internal/services/npm/**'
        - 'handlers/proxy/npm_handler.go'
        - 'tests/integration/npm_*'
      maven:
        - 'internal/services/maven/**'  
        - 'handlers/proxy/maven_handler.go'
        - 'tests/integration/maven_*'
```

### 3. 테스트 결과 리포팅
```yaml
# 테스트 결과 수집 및 리포팅
- name: Collect test results
  uses: actions/upload-artifact@v4
  if: always()
  with:
    name: test-results-${{ matrix.proxy-type }}
    path: |
      coverage.out
      test-results.xml
      logs/
```

## 📊 예상 개선 효과

### CI 실행 시간 개선
- **현재**: 순차 실행으로 20-30분
- **개선 후**: 병렬 실행으로 8-12분
- **절약**: 60-70% 시간 단축

### 리소스 효율성
- **캐시 활용**: Go 모듈 및 빌드 캐시로 빌드 시간 단축  
- **조건부 실행**: 불필요한 테스트 스킵으로 리소스 절약
- **우선순위 기반**: 중요한 테스트 우선 실행

### 개발자 경험 개선
- **빠른 피드백**: 핵심 테스트 우선 실행
- **명확한 문서**: 단계별 테스트 가이드 제공
- **쉬운 디버깅**: 상세한 로그 및 아티팩트 제공

## 📋 체크리스트

### CI 매트릭스 개선
- [ ] 계층화된 테스트 매트릭스 구현
- [ ] 조건부 실행 로직 추가
- [ ] 캐시 최적화 구현
- [ ] 병렬 실행 개선
- [ ] 테스트 결과 수집 자동화

### 문서 개선
- [ ] 테스트 가이드 디렉토리 구조 생성
- [ ] 각 프록시 타입별 테스트 가이드 작성
- [ ] CI/CD 가이드 문서 작성
- [ ] 성능 테스트 가이드 작성
- [ ] 트러블슈팅 가이드 작성

### 도구 개선
- [ ] make ci-local 타겟 구현
- [ ] 테스트 실행 스크립트 개선
- [ ] 로그 수집 및 분석 도구
- [ ] 성능 모니터링 도구

### 검증 및 배포
- [ ] 새로운 CI 매트릭스 테스트
- [ ] 문서 리뷰 및 피드백 반영
- [ ] 팀 교육 및 가이드 배포
- [ ] 성능 개선 효과 측정

## 🔗 관련 파일

- `.github/workflows/ci.yml` - CI 워크플로우 설정
- `docs/05-development/testing/` - 테스트 문서 디렉토리
- `Makefile.test.mk` - 테스트 실행 도구
- `scripts/test-coverage.sh` - 커버리지 분석 스크립트
- `tests/e2e/Makefile` - E2E 테스트 도구