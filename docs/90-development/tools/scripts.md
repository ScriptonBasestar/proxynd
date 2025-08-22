# ProxyND Scripts

이 디렉토리는 ProxyND 프로젝트의 빌드 및 배포 스크립트를 포함합니다.

## 스크립트 목록

### build-multiarch.sh
다중 아키텍처 Docker 이미지를 빌드하는 스크립트입니다.

**사용법:**
```bash
./build-multiarch.sh [옵션]
```

**주요 옵션:**
- `-r, --registry`: Docker 레지스트리 지정
- `-v, --version`: 버전 태그 지정
- `-p, --platforms`: 빌드할 플랫폼 지정 (기본값: linux/amd64,linux/arm64)
- `--push`: 빌드 후 레지스트리에 푸시
- `--load`: 로컬 Docker에 이미지 로드

**예시:**
```bash
# 기본 빌드
./build-multiarch.sh

# 특정 버전으로 빌드 후 푸시
./build-multiarch.sh --version 1.0.0 --push

# ARM64만 빌드하여 로컬에 로드
./build-multiarch.sh --platforms linux/arm64 --load
```

## 권한 설정

스크립트 실행 권한이 필요합니다:
```bash
chmod +x build-multiarch.sh
```
