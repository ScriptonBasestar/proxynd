# APK 클라이언트 설정 가이드

이 문서는 Alpine Linux의 APK 패키지 매니저가 ProxyND를 통해 패키지를 다운로드하도록 설정하는 방법을 설명합니다.

## 지원 버전

- Alpine Linux 3.14 이상
- Edge 브랜치 지원
- 모든 아키텍처 (x86_64, aarch64, armv7 등)

## 설정 방법

### 1. 기존 저장소 설정 백업

```bash
# 기존 저장소 설정 백업
sudo cp /etc/apk/repositories /etc/apk/repositories.backup
```

### 2. ProxyND를 사용하도록 저장소 설정

#### 방법 1: 전체 저장소 교체

```bash
# ProxyND 서버 주소 설정
PROXYND_URL="http://your-proxynd-server:8080/proxy/apk"

# Alpine 버전 확인
ALPINE_VERSION=$(cat /etc/alpine-release | cut -d'.' -f1,2)

# 새로운 저장소 설정 작성
sudo tee /etc/apk/repositories > /dev/null <<EOF
${PROXYND_URL}/v${ALPINE_VERSION}/main
${PROXYND_URL}/v${ALPINE_VERSION}/community
# ${PROXYND_URL}/v${ALPINE_VERSION}/testing
EOF
```

#### 방법 2: 수동 편집

```bash
# 저장소 파일 편집
sudo vi /etc/apk/repositories
```

기존 내용을 다음과 같이 변경:

```
# 기존 (주석 처리)
# https://dl-cdn.alpinelinux.org/alpine/v3.19/main
# https://dl-cdn.alpinelinux.org/alpine/v3.19/community

# ProxyND 사용
http://your-proxynd-server:8080/proxy/apk/v3.19/main
http://your-proxynd-server:8080/proxy/apk/v3.19/community
# http://your-proxynd-server:8080/proxy/apk/v3.19/testing
```

### 3. APK 캐시 갱신 및 테스트

```bash
# APK 캐시 정리
sudo apk update

# 패키지 설치 테스트
sudo apk add curl

# 시스템 업그레이드 테스트
sudo apk upgrade
```

## 고급 설정

### Edge 브랜치 사용

Edge (개발) 브랜치를 사용하려면:

```bash
# Edge 브랜치 저장소 설정
sudo tee /etc/apk/repositories > /dev/null <<EOF
http://your-proxynd-server:8080/proxy/apk/edge/main
http://your-proxynd-server:8080/proxy/apk/edge/community
http://your-proxynd-server:8080/proxy/apk/edge/testing
EOF
```

### 특정 저장소만 프록시 사용

특정 저장소만 ProxyND를 사용하려면:

```bash
# 혼합 설정 예시
sudo tee /etc/apk/repositories > /dev/null <<EOF
# 메인은 ProxyND 사용
http://your-proxynd-server:8080/proxy/apk/v3.19/main

# 커뮤니티는 직접 연결
https://dl-cdn.alpinelinux.org/alpine/v3.19/community
EOF
```

### 아키텍처별 설정

다른 아키텍처용 패키지가 필요한 경우:

```bash
# ARM64 아키텍처 예시
http://your-proxynd-server:8080/proxy/apk/v3.19/main/aarch64
http://your-proxynd-server:8080/proxy/apk/v3.19/community/aarch64
```

## Docker에서 사용

### Dockerfile 설정

```dockerfile
FROM alpine:3.19

# ProxyND APK 저장소 설정
RUN echo "http://your-proxynd-server:8080/proxy/apk/v3.19/main" > /etc/apk/repositories && \
    echo "http://your-proxynd-server:8080/proxy/apk/v3.19/community" >> /etc/apk/repositories

# 패키지 설치
RUN apk update && apk add --no-cache \
    curl \
    bash \
    && rm -rf /var/cache/apk/*
```

### Docker Compose 설정

```yaml
version: '3.8'
services:
  alpine-app:
    image: alpine:3.19
    command: |
      sh -c "
        echo 'http://your-proxynd-server:8080/proxy/apk/v3.19/main' > /etc/apk/repositories &&
        echo 'http://your-proxynd-server:8080/proxy/apk/v3.19/community' >> /etc/apk/repositories &&
        apk update &&
        apk add curl &&
        tail -f /dev/null
      "
```

## 문제 해결

### 1. 연결 오류

```bash
# ProxyND 서버 연결 테스트
curl -I http://your-proxynd-server:8080/healthz

# APK 저장소 직접 테스트
curl -I "http://your-proxynd-server:8080/proxy/apk/v3.19/main/x86_64/APKINDEX.tar.gz"
```

### 2. 저장소 업데이트 실패

```bash
# 상세 로그 출력
sudo apk update --verbose

# 네트워크 문제 확인
ping your-proxynd-server

# DNS 문제 확인
nslookup your-proxynd-server
```

### 3. 패키지 검증 오류

APK 서명 검증 문제가 발생하는 경우:

```bash
# 서명 검증 임시 비활성화 (테스트 목적)
sudo apk --allow-untrusted update

# 또는 저장소별 서명 검증 설정
sudo apk add --repository http://your-proxynd-server:8080/proxy/apk/v3.19/main --allow-untrusted package-name
```

### 4. 캐시 문제

```bash
# APK 캐시 완전 정리
sudo rm -rf /var/cache/apk/*
sudo apk update
```

## 원래 설정으로 복구

```bash
# 백업한 설정으로 복구
sudo mv /etc/apk/repositories.backup /etc/apk/repositories

# 또는 기본 설정으로 복구
sudo tee /etc/apk/repositories > /dev/null <<EOF
https://dl-cdn.alpinelinux.org/alpine/v$(cat /etc/alpine-release | cut -d'.' -f1,2)/main
https://dl-cdn.alpinelinux.org/alpine/v$(cat /etc/alpine-release | cut -d'.' -f1,2)/community
EOF

# 캐시 갱신
sudo apk update
```

## 성능 팁

1. **로컬 네트워크**: ProxyND가 로컬 네트워크에 있을 때 최적 성능
2. **캐시 활용**: 반복적인 빌드 시 대폭적인 시간 단축
3. **병렬 빌드**: Docker 멀티스테이지 빌드에서 캐시 공유 효과
4. **대역폭 절약**: 동일 네트워크의 여러 시스템에서 패키지 공유

## 보안 고려사항

- HTTPS 연결 권장 (ProxyND 서버에서 TLS 설정)
- 신뢰할 수 있는 네트워크에서만 사용
- 정기적인 ProxyND 서버 업데이트
