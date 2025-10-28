# 프로덕션용 멀티스테이지 Dockerfile
# Build arguments
ARG GO_VERSION=1.23
ARG ALPINE_VERSION=3.19
ARG BUILD_DATE
ARG VCS_REF
ARG VERSION
ARG GO_BUILD_TAGS=""
ARG SKIP_TESTS=false

# === Builder Stage ===
FROM golang:${GO_VERSION}-alpine${ALPINE_VERSION} AS builder

ARG SKIP_TESTS

ENV GOTOOLCHAIN=auto

# 보안 패키지 및 빌드 도구 설치
RUN apk add --no-cache \
    git \
    ca-certificates \
    tzdata \
    gcc \
    musl-dev \
    make

# 작업 디렉토리 설정
WORKDIR /build

# 의존성 파일들 먼저 복사 (캐시 최적화)
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# 소스 코드 복사
COPY . .

# 린팅 및 테스트 실행 (선택적)
RUN if [ "$SKIP_TESTS" != "true" ]; then \
        go vet ./... && \
        go test -short ./...; \
    fi

# 바이너리 빌드 (최적화된 설정)
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -tags="${GO_BUILD_TAGS}" \
    -ldflags="-s -w -X main.Version=${VERSION:-dev} -X main.BuildTime=${BUILD_DATE} -X main.CommitSHA=${VCS_REF}" \
    -trimpath \
    -o proxynd \
    .

# === Runtime Stage ===
FROM alpine:${ALPINE_VERSION} AS runtime

# 메타데이터 라벨
LABEL maintainer="ProxyND Team" \
      org.opencontainers.image.title="ProxyND" \
      org.opencontainers.image.description="Package Manager Proxy Server" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.created="${BUILD_DATE}" \
      org.opencontainers.image.revision="${VCS_REF}" \
      org.opencontainers.image.vendor="ProxyND" \
      org.opencontainers.image.source="https://github.com/your-org/proxynd"

# 런타임 패키지 설치
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    curl \
    dumb-init \
    && update-ca-certificates

# 비-루트 사용자 생성
RUN addgroup -g 1001 -S proxynd && \
    adduser -u 1001 -S proxynd -G proxynd -h /app -s /bin/sh

# 디렉토리 생성 및 권한 설정
RUN mkdir -p /app /config /storage /var/log/proxynd && \
    chown -R proxynd:proxynd /app /config /storage /var/log/proxynd

# 바이너리 복사
COPY --from=builder --chown=proxynd:proxynd /build/proxynd /app/

# 설정 예제 복사 (선택적)
COPY --chown=proxynd:proxynd examples /app/examples/

# 헬스체크 스크립트 추가
COPY --chown=proxynd:proxynd <<EOF /app/healthcheck.sh
#!/bin/sh
# ProxyND 헬스체크 스크립트
set -e

HEALTH_URL="http://localhost:${SERVER_PORT:-8080}/health"
TIMEOUT=10

# curl을 사용한 헬스체크
if command -v curl >/dev/null 2>&1; then
    curl -f --max-time $TIMEOUT "$HEALTH_URL" >/dev/null 2>&1
    exit $?
fi

# wget fallback
if command -v wget >/dev/null 2>&1; then
    wget --timeout=$TIMEOUT --tries=1 "$HEALTH_URL" -O /dev/null >/dev/null 2>&1
    exit $?
fi

echo "No HTTP client available for health check"
exit 1
EOF

RUN chmod +x /app/healthcheck.sh

# 환경 변수 설정
ENV SERVER_PORT=8080 \
    CONFIG_DIR=/config \
    STORAGE_DIR=/storage \
    LOG_LEVEL=info \
    LOG_FORMAT=json \
    GOMAXPROCS=0 \
    GOMEMLIMIT=256MiB

# 포트 노출
EXPOSE $SERVER_PORT

# 볼륨 마운트 포인트
VOLUME ["/config", "/storage", "/var/log/proxynd"]

# 작업 디렉토리 설정
WORKDIR /app

# 사용자 전환
USER proxynd

# 헬스체크 설정
HEALTHCHECK --interval=30s --timeout=10s --start-period=60s --retries=3 \
    CMD /app/healthcheck.sh

# dumb-init을 사용한 시그널 처리
ENTRYPOINT ["/usr/bin/dumb-init", "--"]
CMD ["/app/proxynd"]
