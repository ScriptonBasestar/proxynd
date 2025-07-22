#!/bin/bash

# ProxyND 다중 아키텍처 Docker 이미지 빌드 스크립트
# 지원 플랫폼: linux/amd64, linux/arm64

set -e

# 색상 설정
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# 기본 설정
REGISTRY="${DOCKER_REGISTRY:-scriptonbasestar}"
IMAGE_NAME="${IMAGE_NAME:-proxynd}"
VERSION="${VERSION:-latest}"
PLATFORMS="${PLATFORMS:-linux/amd64,linux/arm64}"

# 도움말 출력
usage() {
    echo "사용법: $0 [옵션]"
    echo "옵션:"
    echo "  -r, --registry REGISTRY    Docker 레지스트리 (기본값: $REGISTRY)"
    echo "  -i, --image IMAGE          이미지 이름 (기본값: $IMAGE_NAME)"
    echo "  -v, --version VERSION      버전 태그 (기본값: $VERSION)"
    echo "  -p, --platforms PLATFORMS  빌드 플랫폼 (기본값: $PLATFORMS)"
    echo "  --push                     빌드 후 레지스트리에 푸시"
    echo "  --load                     로컬 도커에 이미지 로드 (단일 플랫폼만 가능)"
    echo "  -h, --help                 도움말 출력"
    exit 1
}

# 옵션 파싱
PUSH=false
LOAD=false

while [[ $# -gt 0 ]]; do
    case $1 in
        -r|--registry)
            REGISTRY="$2"
            shift 2
            ;;
        -i|--image)
            IMAGE_NAME="$2"
            shift 2
            ;;
        -v|--version)
            VERSION="$2"
            shift 2
            ;;
        -p|--platforms)
            PLATFORMS="$2"
            shift 2
            ;;
        --push)
            PUSH=true
            shift
            ;;
        --load)
            LOAD=true
            shift
            ;;
        -h|--help)
            usage
            ;;
        *)
            echo -e "${RED}알 수 없는 옵션: $1${NC}"
            usage
            ;;
    esac
done

# Docker buildx 확인
echo -e "${BLUE}Docker buildx 확인 중...${NC}"
if ! docker buildx version &> /dev/null; then
    echo -e "${RED}Docker buildx가 설치되어 있지 않습니다.${NC}"
    echo "설치 방법: https://docs.docker.com/buildx/working-with-buildx/"
    exit 1
fi

# 빌더 생성 또는 사용
BUILDER_NAME="proxynd-multiarch-builder"
if ! docker buildx inspect $BUILDER_NAME &> /dev/null; then
    echo -e "${YELLOW}Buildx 빌더 생성 중: $BUILDER_NAME${NC}"
    docker buildx create --name $BUILDER_NAME --driver docker-container --use
    docker buildx inspect --bootstrap
else
    echo -e "${GREEN}기존 빌더 사용: $BUILDER_NAME${NC}"
    docker buildx use $BUILDER_NAME
fi

# 이미지 태그 설정
FULL_IMAGE_NAME="$REGISTRY/$IMAGE_NAME"
TAGS=(
    "$FULL_IMAGE_NAME:$VERSION"
    "$FULL_IMAGE_NAME:latest"
)

# Git 커밋 해시 추가 (선택적)
if git rev-parse --git-dir > /dev/null 2>&1; then
    GIT_COMMIT=$(git rev-parse --short HEAD)
    TAGS+=("$FULL_IMAGE_NAME:$GIT_COMMIT")
fi

# 빌드 명령어 구성
BUILD_CMD="docker buildx build"
BUILD_CMD="$BUILD_CMD --platform $PLATFORMS"
BUILD_CMD="$BUILD_CMD -f Dockerfile.multiarch"

for tag in "${TAGS[@]}"; do
    BUILD_CMD="$BUILD_CMD -t $tag"
done

# 빌드 옵션
BUILD_CMD="$BUILD_CMD --build-arg VERSION=$VERSION"
BUILD_CMD="$BUILD_CMD --build-arg BUILD_DATE=$(date -u +'%Y-%m-%dT%H:%M:%SZ')"

# 푸시 또는 로드 옵션
if [ "$PUSH" = true ]; then
    BUILD_CMD="$BUILD_CMD --push"
elif [ "$LOAD" = true ]; then
    if [[ "$PLATFORMS" == *","* ]]; then
        echo -e "${RED}--load 옵션은 단일 플랫폼에서만 사용 가능합니다.${NC}"
        echo "단일 플랫폼 예시: -p linux/amd64"
        exit 1
    fi
    BUILD_CMD="$BUILD_CMD --load"
fi

# 빌드 실행
echo -e "${BLUE}다중 아키텍처 이미지 빌드 시작...${NC}"
echo -e "${YELLOW}플랫폼: $PLATFORMS${NC}"
echo -e "${YELLOW}태그: ${TAGS[*]}${NC}"

eval $BUILD_CMD .

if [ $? -eq 0 ]; then
    echo -e "${GREEN}빌드 성공!${NC}"

    # 이미지 정보 출력
    if [ "$PUSH" = true ]; then
        echo -e "${GREEN}이미지가 레지스트리에 푸시되었습니다.${NC}"
        echo -e "${BLUE}이미지 정보 확인:${NC}"
        docker buildx imagetools inspect "$FULL_IMAGE_NAME:$VERSION"
    elif [ "$LOAD" = true ]; then
        echo -e "${GREEN}이미지가 로컬에 로드되었습니다.${NC}"
        docker images | grep "$IMAGE_NAME"
    fi
else
    echo -e "${RED}빌드 실패!${NC}"
    exit 1
fi

# 정리 (선택적)
# docker buildx rm $BUILDER_NAME
