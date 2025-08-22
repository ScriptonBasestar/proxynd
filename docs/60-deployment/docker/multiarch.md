# 다중 아키텍처 Docker 이미지 빌드 가이드

ProxyND는 다양한 하드웨어 플랫폼에서 실행할 수 있도록 다중 아키텍처 Docker 이미지를 지원합니다.

## 지원 플랫폼

- `linux/amd64` (x86_64)
- `linux/arm64` (aarch64)

## 빌드 요구사항

### Docker Buildx 설치

Docker 19.03+ 버전과 buildx 플러그인이 필요합니다.

```bash
# buildx 확인
docker buildx version

# buildx가 없다면 설치
# Docker Desktop은 기본적으로 포함되어 있음
# Linux에서는 별도 설치 필요:
docker run --rm --privileged docker/binfmt:latest --install all
```

## 빌드 방법

### 1. Makefile 사용 (권장)

```bash
# 다중 아키텍처 이미지 빌드 (로컬 저장)
make docker-build-multiarch

# 빌드 후 레지스트리에 푸시
make docker-build-multiarch-push

# 특정 아키텍처만 빌드
make docker-build-amd64  # AMD64만
make docker-build-arm64  # ARM64만
```

### 2. 빌드 스크립트 직접 사용

```bash
# 기본 빌드 (amd64, arm64)
./scripts/build-multiarch.sh

# 레지스트리와 버전 지정
./scripts/build-multiarch.sh \
  --registry myregistry \
  --image proxynd \
  --version 1.0.0

# 빌드 후 푸시
./scripts/build-multiarch.sh --push

# 특정 플랫폼만 빌드
./scripts/build-multiarch.sh \
  --platforms linux/arm64 \
  --load
```

### 3. Docker Buildx 직접 사용

```bash
# 빌더 생성
docker buildx create --name proxynd-builder --use

# 다중 플랫폼 빌드
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  -f Dockerfile.multiarch \
  -t myregistry/proxynd:latest \
  --push .

# 빌더 제거
docker buildx rm proxynd-builder
```

## 이미지 확인

### 빌드된 이미지 정보 확인

```bash
# 레지스트리에 푸시된 이미지 확인
docker buildx imagetools inspect scriptonbasestar/proxynd:latest

# 로컬 이미지 확인
docker images | grep proxynd
```

### 플랫폼별 이미지 실행

```bash
# AMD64 플랫폼에서 실행
PLATFORM=linux/amd64 docker-compose -f docker-compose.multiarch.yml up

# ARM64 플랫폼에서 실행 (Apple Silicon Mac, AWS Graviton 등)
PLATFORM=linux/arm64 docker-compose -f docker-compose.multiarch.yml up
```

## Dockerfile.multiarch 특징

### 1. 크로스 컴파일 지원

```dockerfile
ARG BUILDPLATFORM
ARG TARGETPLATFORM
ARG TARGETOS
ARG TARGETARCH

ENV GOOS=$TARGETOS
ENV GOARCH=$TARGETARCH
```

### 2. 최적화된 빌드

- Go 모듈 캐싱으로 빌드 속도 향상
- `-ldflags="-w -s"` 플래그로 바이너리 크기 최소화
- `trimpath`로 빌드 경로 정보 제거

### 3. 보안 강화

- distroless 베이스 이미지 사용
- nonroot 사용자로 실행
- 최소한의 의존성만 포함

### 4. 헬스체크 내장

```dockerfile
HEALTHCHECK --interval=30s --timeout=3s \
  CMD ["/app/proxynd", "-health"]
```

## CI/CD 통합

### GitHub Actions 예시

```yaml
- name: Set up QEMU
  uses: docker/setup-qemu-action@v2

- name: Set up Docker Buildx
  uses: docker/setup-buildx-action@v2

- name: Build and push
  run: make docker-build-multiarch-push
```

### GitLab CI 예시

```yaml
build-multiarch:
  stage: build
  services:
    - docker:dind
  before_script:
    - docker run --rm --privileged multiarch/qemu-user-static --reset -p yes
    - docker buildx create --use
  script:
    - make docker-build-multiarch-push
```

## 문제 해결

### 1. buildx를 찾을 수 없음

```bash
# Docker 버전 확인
docker version

# buildx 플러그인 수동 설치
mkdir -p ~/.docker/cli-plugins
curl -L https://github.com/docker/buildx/releases/latest/download/buildx-linux-amd64 \
  -o ~/.docker/cli-plugins/docker-buildx
chmod +x ~/.docker/cli-plugins/docker-buildx
```

### 2. QEMU 에러

```bash
# QEMU 설치
docker run --rm --privileged multiarch/qemu-user-static --reset -p yes

# 또는
docker run --rm --privileged docker/binfmt:latest --install all
```

### 3. 빌드 속도가 느림

- 빌드 캐시 활용
- 병렬 빌드 수 조정: `docker buildx build --builder-opt max-parallelism=2`
- 특정 플랫폼만 빌드

## 성능 고려사항

### ARM64 최적화

ARM64 플랫폼에서는 다음 사항을 고려하세요:

1. **네이티브 빌드**: 가능하면 ARM64 머신에서 직접 빌드
2. **메모리 사용**: ARM 기반 시스템은 일반적으로 메모리가 적음
3. **Go 런타임**: Go 1.16+ 버전 사용 권장 (ARM64 최적화 포함)

### 이미지 크기 최적화

현재 설정으로 생성되는 이미지 크기:
- AMD64: ~20MB
- ARM64: ~20MB

distroless 이미지 사용으로 최소 크기 달성

## 배포 전략

### 1. 멀티 아키텍처 매니페스트

Docker는 자동으로 실행 환경에 맞는 이미지를 선택합니다:

```bash
# 모든 플랫폼에서 동일한 명령어 사용
docker run scriptonbasestar/proxynd:latest
```

### 2. 플랫폼별 태그

필요시 플랫폼별 태그도 사용 가능:

```bash
# 스크립트에서 태그 생성
docker tag proxynd:latest proxynd:latest-amd64
docker tag proxynd:latest proxynd:latest-arm64
```

## 모니터링

다중 플랫폼 환경에서는 다음을 모니터링하세요:

1. **플랫폼별 메트릭**: `/metrics` 엔드포인트에서 `platform` 레이블 확인
2. **성능 차이**: ARM vs x86 성능 비교
3. **메모리 사용량**: 플랫폼별 메모리 사용 패턴

## 베스트 프랙티스

1. **정기적인 테스트**: 모든 지원 플랫폼에서 정기적으로 테스트
2. **플랫폼별 최적화**: 필요시 플랫폼별 빌드 플래그 사용
3. **캐시 활용**: 빌드 캐시를 적극 활용하여 빌드 시간 단축
4. **보안 업데이트**: 베이스 이미지 정기적 업데이트
