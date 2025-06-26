# CI/CD 파이프라인 가이드

ProxyND는 GitHub Actions와 GitLab CI를 통한 자동화된 CI/CD 파이프라인을 제공합니다.

## 📋 목차

- [GitHub Actions](#github-actions)
- [GitLab CI](#gitlab-ci)
- [로컬 테스트](#로컬-테스트)
- [릴리스 프로세스](#릴리스-프로세스)
- [보안 스캔](#보안-스캔)

## GitHub Actions

### 워크플로우 구성

#### 1. CI 워크플로우 (`.github/workflows/ci.yml`)

모든 push와 pull request에서 실행되는 메인 CI 파이프라인입니다.

**단계:**
- **Lint**: 코드 스타일 검사 (golangci-lint, go fmt, go vet)
- **Test**: 단위 테스트 실행 (다중 Go 버전)
- **Build**: 크로스 플랫폼 바이너리 빌드
- **Docker**: 다중 아키텍처 Docker 이미지 빌드
- **Integration Test**: 통합 테스트 실행
- **E2E Test**: End-to-End 테스트 실행
- **Security Scan**: Trivy를 사용한 취약점 스캔

```yaml
# 수동 실행
gh workflow run ci.yml
```

#### 2. 릴리스 워크플로우 (`.github/workflows/release.yml`)

태그 푸시 또는 수동 실행 시 릴리스를 생성합니다.

**기능:**
- 다중 플랫폼 바이너리 빌드
- Docker 이미지 빌드 및 푸시 (ghcr.io, Docker Hub)
- GitHub Release 생성
- Helm 차트 패키징

```bash
# 태그로 릴리스
git tag v1.0.0
git push origin v1.0.0

# 수동 릴리스
gh workflow run release.yml -f version=v1.0.0
```

#### 3. 의존성 업데이트 (`.github/workflows/dependency-update.yml`)

매주 월요일 자동으로 Go 모듈 의존성을 업데이트합니다.

**기능:**
- Go 모듈 업데이트
- 테스트 실행
- PR 자동 생성

#### 4. CodeQL 분석 (`.github/workflows/codeql.yml`)

코드 보안 취약점을 자동으로 스캔합니다.

### GitHub Actions 시크릿 설정

다음 시크릿을 리포지토리에 설정해야 합니다:

```bash
# Docker Hub (선택사항)
DOCKERHUB_USERNAME
DOCKERHUB_TOKEN

# 기타 필요한 시크릿들은 기본적으로 제공됨
# GITHUB_TOKEN - 자동 제공
```

## GitLab CI

### 파이프라인 구성

`.gitlab-ci.yml` 파일은 다음 스테이지로 구성됩니다:

1. **lint**: 코드 품질 검사
2. **test**: 단위 및 통합 테스트
3. **build**: 바이너리 및 Helm 차트 빌드
4. **docker**: Docker 이미지 빌드
5. **deploy**: Kubernetes 배포 (수동)

### GitLab CI 변수 설정

프로젝트 설정에서 다음 변수를 설정하세요:

```bash
# Kubernetes 배포용 (선택사항)
KUBE_CONFIG - Kubernetes 설정
STAGING_URL - 스테이징 환경 URL
PRODUCTION_URL - 프로덕션 환경 URL
```

### 캐싱 전략

빌드 속도 향상을 위한 캐싱:

```yaml
cache:
  key: ${CI_COMMIT_REF_SLUG}
  paths:
    - .go/pkg/mod/
    - .go/build-cache/
```

## 로컬 테스트

CI 파이프라인을 로컬에서 테스트하려면:

### 1. Linting

```bash
# golangci-lint 설치
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | \
  sh -s -- -b $(go env GOPATH)/bin v1.55.2

# 린트 실행
golangci-lint run

# go fmt 확인
gofmt -l .

# go vet 실행
go vet ./...
```

### 2. 테스트

```bash
# 단위 테스트
go test -v -race -coverprofile=coverage.txt ./...

# 커버리지 확인
go tool cover -html=coverage.txt

# 통합 테스트
cd tests/integration
make test

# E2E 테스트
cd tests/e2e
make test
```

### 3. 빌드

```bash
# 로컬 빌드
go build -o proxynd main.go

# 크로스 컴파일
GOOS=linux GOARCH=amd64 go build -o proxynd-linux-amd64 main.go
GOOS=darwin GOARCH=arm64 go build -o proxynd-darwin-arm64 main.go
```

## 릴리스 프로세스

### 1. 버전 태깅

```bash
# 버전 태그 생성
git tag -a v1.0.0 -m "Release v1.0.0"

# 태그 푸시
git push origin v1.0.0
```

### 2. 자동 릴리스

태그가 푸시되면 자동으로:
- 바이너리가 빌드됩니다
- Docker 이미지가 생성되고 푸시됩니다
- GitHub Release가 생성됩니다
- Changelog가 생성됩니다

### 3. 수동 릴리스

GitHub UI에서:
1. Actions 탭으로 이동
2. Release 워크플로우 선택
3. Run workflow 클릭
4. 버전 입력 (예: v1.0.0)

## 보안 스캔

### 1. 의존성 스캔

```bash
# Nancy (Sonatype)
go list -json -deps ./... | docker run --rm -i sonatypecorp/nancy:latest sleuth

# govulncheck
go install golang.org/x/vuln/cmd/govulncheck@latest
govulncheck ./...
```

### 2. 컨테이너 스캔

```bash
# Trivy
trivy image proxynd:latest

# Grype
grype proxynd:latest
```

### 3. 코드 스캔

- CodeQL: 자동으로 실행됨
- SonarQube: 별도 설정 필요

## 모니터링

### GitHub Actions

- 리포지토리의 Actions 탭에서 모든 워크플로우 확인
- 실패한 워크플로우는 이메일로 알림

### GitLab CI

- 프로젝트의 CI/CD > Pipelines에서 확인
- 파이프라인 상태는 Slack/Discord로 알림 가능

## 최적화 팁

### 1. 빌드 캐싱

Docker 빌드 캐싱:
```dockerfile
# go.mod 먼저 복사하여 캐싱 활용
COPY go.mod go.sum ./
RUN go mod download
```

### 2. 병렬 실행

```yaml
# GitHub Actions
strategy:
  matrix:
    os: [ubuntu-latest, macos-latest]
    go: ['1.21', '1.22']

# GitLab CI
parallel:
  matrix:
    - GOOS: linux
      GOARCH: [amd64, arm64]
```

### 3. 조건부 실행

```yaml
# PR에서만 실행
if: github.event_name == 'pull_request'

# 특정 파일 변경 시에만 실행
if: contains(github.event.head_commit.modified, 'go.mod')
```

## 문제 해결

### 1. 권한 오류

```bash
# GitHub Actions
Error: Permission denied to github-actions[bot]
해결: Settings > Actions > General > Workflow permissions 확인

# GitLab CI
Error: unauthorized: authentication required
해결: CI/CD 변수에 레지스트리 인증 정보 추가
```

### 2. 빌드 실패

```bash
# Go 모듈 캐시 문제
go clean -modcache
go mod download

# Docker buildx 문제
docker buildx rm mybuilder
docker buildx create --use
```

### 3. 테스트 실패

```bash
# 레이스 컨디션
go test -race -count=1 ./...

# 타임아웃
go test -timeout 30s ./...
```

## 베스트 프랙티스

1. **브랜치 보호**: main/master 브랜치는 직접 푸시 금지
2. **PR 필수**: 모든 변경사항은 PR을 통해 머지
3. **테스트 커버리지**: 최소 80% 이상 유지
4. **시크릿 관리**: 환경 변수로 민감한 정보 관리
5. **캐싱 활용**: 빌드 시간 단축을 위한 적극적 캐싱
6. **병렬 처리**: 가능한 작업은 병렬로 실행
7. **실패 알림**: 중요한 워크플로우는 실패 시 알림 설정