# ProxyND 바이너리 배포 워크플로우 가이드

이 문서는 ProxyND의 GitHub Releases를 통한 자동화된 바이너리 배포 워크플로우에 대해 설명합니다.

## 목차

- [개요](#개요)
- [워크플로우 구성](#워크플로우-구성)
- [릴리스 프로세스](#릴리스-프로세스)
- [지원 플랫폼](#지원-플랫폼)
- [로컬 빌드](#로컬-빌드)
- [문제 해결](#문제-해결)

## 개요

ProxyND는 GitHub Actions를 사용하여 다음과 같은 자동화된 릴리스 프로세스를 제공합니다:

- **다중 플랫폼 바이너리 빌드**: Linux, macOS, Windows용 바이너리 자동 생성
- **Docker 이미지 빌드**: 다중 아키텍처 Docker 이미지 자동 빌드 및 배포
- **GitHub Releases**: 바이너리 및 체크섬 파일 자동 업로드
- **Helm 차트**: Kubernetes 배포용 Helm 차트 패키징

## 워크플로우 구성

### 파일 구조

```
.github/workflows/
├── release.yml      # 릴리스 워크플로우
└── build-test.yml   # 빌드 테스트 워크플로우

scripts/
├── build-release.sh # 로컬 빌드 스크립트
├── install-completion.sh
└── install-man-pages.sh
```

### 워크플로우 트리거

1. **자동 트리거**: `v*` 패턴의 Git 태그 푸시 시
2. **수동 트리거**: GitHub UI에서 workflow_dispatch 사용

## 릴리스 프로세스

### 1. 자동 릴리스 (권장)

```bash
# 1. 버전 태그 생성 및 푸시
git tag v1.2.3
git push origin v1.2.3

# 2. GitHub Actions가 자동으로 실행됨
# 3. 완료 후 GitHub Releases에서 확인
```

### 2. 수동 릴리스

GitHub 웹 인터페이스에서:

1. **Actions** 탭으로 이동
2. **Release** 워크플로우 선택
3. **Run workflow** 클릭
4. 릴리스 태그 입력 (예: `v1.2.3`)
5. **Run workflow** 실행

### 3. 릴리스 단계별 실행

#### 단계 1: 바이너리 빌드 (build-binaries)

```yaml
# 지원 플랫폼별 병렬 빌드
strategy:
  matrix:
    include:
      - goos: linux, goarch: amd64
      - goos: linux, goarch: arm64
      - goos: darwin, goarch: amd64
      - goos: darwin, goarch: arm64
      - goos: windows, goarch: amd64
```

**생성되는 파일:**
- `proxynd-v1.2.3-{os}-{arch}.{tar.gz|zip}` (서버 + CLI)
- `proxyndctl-v1.2.3-{os}-{arch}.{tar.gz|zip}` (CLI만)
- `checksums-{os}-{arch}.txt` (체크섬 파일)

#### 단계 2: Docker 이미지 빌드 (docker-release)

```bash
# 생성되는 이미지
ghcr.io/scriptonbasestar/proxynd:1.2.3
ghcr.io/scriptonbasestar/proxynd:latest
scriptonbasestar/proxynd:1.2.3  # Docker Hub (조건부)
scriptonbasestar/proxynd:latest # Docker Hub (조건부)
```

#### 단계 3: GitHub Release 생성 (create-release)

- 모든 플랫폼의 바이너리 수집
- 통합 체크섬 파일 생성 (`checksums.txt`)
- 릴리스 노트 자동 생성
- GitHub Release 생성

#### 단계 4: Helm 차트 패키징 (helm-release)

```bash
# 생성되는 파일
proxynd-1.2.3.tgz  # Helm 차트 패키지
```

## 지원 플랫폼

### 바이너리

| OS | Architecture | 파일 형식 |
|---|---|---|
| Linux | amd64 | tar.gz |
| Linux | arm64 | tar.gz |
| macOS | amd64 | tar.gz |
| macOS | arm64 (Apple Silicon) | tar.gz |
| Windows | amd64 | zip |

### Docker 이미지

| Platform | Support |
|---|---|
| linux/amd64 | ✅ |
| linux/arm64 | ✅ |

## 로컬 빌드

개발 중이거나 로컬에서 릴리스 바이너리를 빌드하려면 제공된 스크립트를 사용하세요.

### 전체 플랫폼 빌드

```bash
# 모든 플랫폼 빌드
./scripts/build-release.sh

# 특정 버전으로 빌드
./scripts/build-release.sh -v v1.2.3

# 특정 디렉토리에 빌드
./scripts/build-release.sh -o ./release
```

### 특정 플랫폼 빌드

```bash
# Linux만 빌드
./scripts/build-release.sh -p linux

# macOS amd64만 빌드
./scripts/build-release.sh -p darwin-amd64

# 빌드 후 테스트 실행
./scripts/build-release.sh -p linux-amd64 -t
```

### 빌드 스크립트 옵션

```bash
Usage: ./scripts/build-release.sh [옵션]

옵션:
    -v, --version VERSION   빌드 버전 (기본값: git tag 또는 dev)
    -o, --output DIR        출력 디렉토리 (기본값: ./dist)
    -p, --platform LIST    빌드할 플랫폼 (기본값: all)
    -c, --clean             기존 빌드 결과 삭제
    -t, --test              빌드 후 테스트 실행
    -h, --help              도움말 출력
```

## 바이너리 사용법

### 다운로드 및 설치

```bash
# 최신 릴리스 다운로드 (Linux amd64 예제)
VERSION=$(curl -s https://api.github.com/repos/scriptonbasestar/proxynd/releases/latest | grep '"tag_name"' | cut -d'"' -f4)
curl -L "https://github.com/scriptonbasestar/proxynd/releases/download/${VERSION}/proxynd-${VERSION}-linux-amd64.tar.gz" | tar xz

# 실행 파일 설치
sudo mv proxynd /usr/local/bin/
sudo mv proxyndctl /usr/local/bin/

# 버전 확인
proxynd --version
proxyndctl --version
```

### 체크섬 검증

```bash
# 체크섬 파일 다운로드
curl -L "https://github.com/scriptonbasestar/proxynd/releases/download/${VERSION}/checksums.txt" -o checksums.txt

# 다운로드한 파일 검증
sha256sum -c checksums.txt --ignore-missing
```

## 빌드 정보 포함

모든 바이너리에는 다음 빌드 정보가 포함됩니다:

```bash
# 서버 버전 정보
proxynd --version
# 출력: ProxyND v1.2.3
#       Build Time: 2024-01-15T10:30:00Z
#       Commit SHA: abc123

# CLI 버전 정보
proxyndctl --version
# 출력: proxyndctl version v1.2.3
```

## 문제 해결

### 빌드 실패

1. **Go 버전 확인**: Go 1.22 이상 필요
2. **의존성 문제**: `go mod download && go mod tidy` 실행
3. **크로스 컴파일 오류**: CGO_ENABLED=0 설정 확인

### 릴리스 워크플로우 실패

1. **권한 문제**: GitHub repository의 Actions 권한 확인
2. **Docker Hub 로그인**: DOCKERHUB_USERNAME, DOCKERHUB_TOKEN 시크릿 설정
3. **태그 충돌**: 기존 태그와 중복되지 않는지 확인

### Docker 이미지 문제

```bash
# 이미지 확인
docker pull ghcr.io/scriptonbasestar/proxynd:latest
docker run --rm ghcr.io/scriptonbasestar/proxynd:latest --version

# 다중 아키텍처 확인
docker buildx imagetools inspect ghcr.io/scriptonbasestar/proxynd:latest
```

### 로컬 빌드 문제

```bash
# 환경 확인
go version
git status

# 의존성 확인
go mod verify
go mod tidy

# 수동 빌드 테스트
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o proxynd main.go
```

## 관련 문서

- [README.md](../README.md) - 프로젝트 개요 및 설치
- [CLI_USAGE_GUIDE.md](CLI_USAGE_GUIDE.md) - CLI 사용법
- [DEPLOYMENT.md](DEPLOYMENT.md) - 배포 가이드
- [CI_CD.md](CI_CD.md) - CI/CD 파이프라인

## 기여하기

릴리스 워크플로우 개선사항이나 버그 리포트는 GitHub Issues를 통해 제보해 주세요.

- 새로운 플랫폼 지원 요청
- 빌드 스크립트 개선
- 문서 업데이트