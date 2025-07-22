# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 프로젝트 개요

프록신디(proxynd)는 Go로 작성된 패키지 매니저 프록시/미러 서버입니다. Maven과 APT 패키지 매니저를 지원하며, 기업 환경에서 패키지 다운로드 속도 향상과 대역폭 절약을 위해 사용됩니다.

**웹 프레임워크**: Fiber v2 (고성능 웹 프레임워크)

## 빌드 및 개발 명령어

### 개발 환경 설정 (필수)
```bash
# 의존성 설치 및 정리 (air 포함)
make dev-prepare

# 설정 디렉토리 준비 (샘플 설정 복사 + .env 생성)
make dev-setup
# 또는 (프로덕션용)
make setup
```

### 개발 실행
```bash
# 로컬 개발 실행 (Air 핫 리로드)
make dev-run

# 직접 실행 (Air 없이)
make dev-run-direct

# Docker로 빌드 및 실행
make docker-build
make docker-run

# 실행 중인 컨테이너 접속
make docker-enter
```

### 테스트
```bash
# 모든 테스트 실행
make dev-test

# 단위 테스트 (특정 패키지)
go test -v ./configs/...
go test -v ./handlers/proxy/...

# 테스트 커버리지 생성 (HTML 리포트 포함)
make test-coverage

# 레이스 컨디션 검사
make test-race

# 벤치마크 테스트
make test-benchmark

# 통합 테스트
make test-integration

# 모든 테스트 (단위 + 레이스 + 서비스)
make test-all
```

### 빌드 및 배포
```bash
# 멀티아키텍처 빌드
make docker-build-multiarch

# 멀티아키텍처 빌드 및 푸시
make docker-build-multiarch-push

# 특정 아키텍처 빌드
make docker-build-amd64
make docker-build-arm64
```

### 환경 변수
```bash
CONFIG_DIR=/config      # 설정 파일 디렉토리
STORAGE_DIR=/storage    # 캐시 저장 디렉토리  
SERVER_PORT=8080       # 서버 포트
```

## 코드 구조 및 아키텍처

### 주요 디렉토리 구조
- `/internal/app`: 애플리케이션 핵심 구조
  - 의존성 주입 컨테이너 (thread-safe 싱글톤)
  - 애플리케이션 생명주기 관리 및 graceful shutdown
  - 헬스체크 및 버전 관리

- `/internal/services`: 비즈니스 로직 계층
  - 프록시 서비스, 설정 관리, 어댑터 패턴
  - Repository 패턴으로 데이터 액세스 분리

- `/internal/domain`: 도메인별 로직 분리
  - 각 패키지 타입별 도메인 로직 (apt, maven, npm 등)
  - 공통 인터페이스 정의

- `/configs`: 각 프록시 타입별 설정 구조체 및 파서
  - 계층형 설정 (글로벌 → 프록시별 → 환경변수 오버라이드)
  - 핫 리로드 지원, YAML/TOML 형식 지원

- `/handlers/proxy`: HTTP 요청 핸들러
  - 통합 프록시 핸들러 (`/proxy/:type/*` 패턴)
  - 각 프록시 타입별 팩토리 패턴 구현

- `/middlewares`: 미들웨어 계층
  - 보안: 인증, 인가, IP 필터링, 속도 제한
  - 운영: 액세스 로그, 메트릭 수집, 프록시 정책

- `/cache`: 다층 캐싱 시스템
  - 인터페이스 기반 설계 (파일시스템, S3 백엔드)
  - LRU 제거 정책, TTL 기반 만료, 통계 수집

- `/alerts` & `/internal/webhook`: 이벤트 기반 알림 시스템
  - 플러그형 알림 인터페이스, 배치 처리
  - 영구 큐, 재시도 로직, 이벤트 히스토리

- `/internal/auth`: 다중 인증 시스템
  - JWT, OAuth2 (GitHub/GitLab/Google), Basic Auth
  - 토큰 갱신 미들웨어

- `/verification`: 패키지 검증 시스템
  - APK 서명 검증, 체크섬 검증, 무결성 확인

### 설정 파일 구조
1. 기본 설정: `global.yaml`
2. 프록시별 설정: `apt-proxy.yaml`, `maven-proxy.yaml` 등
3. 설정 오버라이딩: `default.yaml` > `server1.yaml` 순서로 로딩
4. 지원 형식: yaml, yml, toml (순서대로 탐색)

### 프록시 추가 패턴
새로운 프록시 타입 추가 시:
1. `/configs/`에 설정 구조체 정의 (예: `npm_config.go`)
2. `/handlers/proxy/`에 Fiber 핸들러 구현 (예: `npm_handler.go`)
   - 핸들러 시그니처: `func(c *fiber.Ctx) error`
   - 에러 처리: `return c.Status(code).SendString(msg)`
3. `/routers/proxy_router.go`에 라우팅 추가
   - 라우트 패턴: `app.Get("/proxy/:type/*", handler)`
4. `sample-conf/`에 샘플 설정 파일 추가

## 개발 규칙 (copilot.md 기반)

1. **주석과 문서는 한국어로 작성**
2. **에러 체크 및 타입 검증 구현 필수**
3. **완전한 코드 작성 (placeholder 금지)**
4. **명확한 인라인 주석 포함**

## 테스트 방법

### 단위 테스트
```bash
# 전체 테스트
go test -v ./...

# 커버리지 포함
go test -v -cover ./...
```

### 통합 테스트
- APT 프록시: Docker Ubuntu 이미지로 미러 설정 후 `apt update/upgrade` 테스트
- Maven 프록시: 패키지 다운로드/업로드 테스트 (`integration/TEST_MAVEN.md` 참조)

## 배포

### Docker
```bash
# 빌드
make docker-build

# 레지스트리 푸시
make docker-push
```

### Kubernetes (Helm)
```bash
# Helm 차트 위치: /helm
# 차트 버전: 0.1.3
# Kubernetes 1.20+ 필요
```

## CLI 도구 (proxyndctl)

ProxyND는 강력한 명령줄 관리 도구 `proxyndctl`을 제공합니다:

### 주요 기능
- **캐시 관리**: 캐시 목록 조회, 삭제, 크기 확인
- **설정 검증**: 설정 파일 유효성 검사 및 조회  
- **서버 상태**: 실시간 서버 상태 및 메트릭 조회
- **사용자 관리**: 사용자 추가/삭제/목록 조회
- **프록시 테스트**: 각 프록시 타입별 연결성 테스트

### 사용 예시
```bash
# 캐시 상태 확인
proxyndctl cache list

# 설정 파일 검증
proxyndctl config validate

# 서버 상태 확인
proxyndctl status

# 프록시 테스트
proxyndctl test --proxy npm
```

## 주요 기술 스택

- **Go 1.23+**: 메인 언어
- **Fiber v2**: 고성능 웹 프레임워크 (Express.js 스타일)
- **Prometheus**: 메트릭 수집 (`/metrics` 엔드포인트)
- **구조화 로깅**: JSON 형식 로그 지원
- **Hot Reload**: Air를 사용한 개발 환경

## 코드 품질 및 도구

### 코드 포맷팅 및 린팅
```bash
# 코드 포맷팅 (gofmt + goimports)
make fmt

# 린팅 실행 (golangci-lint 자동 설치)
make lint

# 린팅 이슈 자동 수정
make lint-fix

# 품질 검사 (포맷팅 + 린팅 + 커버리지)
make quality
```

### 보안 및 분석
```bash
# 보안 스캔 (의존성 취약점 + 코드 분석)
make security

# 코드 복잡도 분석
make analyze

# 의존성 관리 및 검증
make deps
```

### 모의 객체 생성
```bash
# Mock 생성 (Mockery 자동 설치)
make generate-mocks

# Mock 업데이트
make update-mocks
```

## 주요 아키텍처 패턴

### 의존성 주입
- `internal/app/container.go`: Thread-safe 싱글톤 컨테이너
- 지연 초기화 및 서비스 팩토리 패턴

### Repository 패턴
- 파일 기반 캐시 및 설정 저장소
- 인터페이스 기반 설계로 테스트 용이성 확보

### 미들웨어 체인
- Fiber 기반 미들웨어 파이프라인
- 보안, 로깅, 메트릭, 인증 계층 분리

### 이벤트 기반 알림
- 비동기 웹훅 처리, 배치 및 재시도 로직
- 영구 큐를 통한 안정성 보장

## 주요 개발 워크플로우

### 빠른 개발 시작
```bash
make dev          # = make dev-prepare + make dev-setup
make dev-run      # 핫 리로드 개발 서버 실행
```

### 코드 변경 후 검증
```bash
make check        # 빠른 검증 (lint + 단위 테스트)
make quality      # 전체 품질 검사 (포맷팅 + 린팅 + 커버리지)
```

### 단일 테스트 실행
```bash
# 특정 패키지 테스트
go test -v ./configs/apt_proxy_config_test.go

# 특정 테스트 함수
go test -v -run TestSpecificFunction ./configs/...

# 벤치마크
go test -bench=BenchmarkName ./internal/services/...
```

## 주의사항

- 테스트 작성 시 `github.com/go-playground/assert/v2` 및 `github.com/stretchr/testify` 사용
- 캐시 TTL 기본값: 3600초 (1시간)
- 개발 시 `make dev-setup` 필수 (환경 변수 및 설정 파일 자동 생성)
- 프로덕션 배포 전 `make quality` 및 `make security` 실행 권장
