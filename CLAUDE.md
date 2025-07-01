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

# 통합 테스트
cd tests/integration && make test

# E2E 테스트  
cd tests/e2e && make test

# 커버리지 포함 테스트
go test -v -cover ./...
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
- `/configs`: 각 프록시 타입별 설정 구조체 및 파서
  - 각 프록시 타입(APT, Maven, NPM 등)별 설정 구현
  - YAML 설정 파일 파싱 로직
  
- `/handlers/proxy`: HTTP 요청 핸들러
  - 각 프록시 타입별 컨트롤러 구현
  - 프록시 요청 처리 및 캐싱 로직
  - Fiber 컨텍스트 기반 핸들러
  
- `/routers`: HTTP 라우팅 설정
  - Fiber 프레임워크 기반 라우터 설정
  - 헬스체크 엔드포인트: `/healthz`
  - HTML 템플릿 렌더링 지원
  
- `/helpers`: 유틸리티 함수
  - URL 조작, YAML 파싱 등 공통 기능

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

- **Go 1.22+**: 메인 언어
- **Fiber v2**: 고성능 웹 프레임워크 (Express.js 스타일)
- **Prometheus**: 메트릭 수집 (`/metrics` 엔드포인트)
- **구조화 로깅**: JSON 형식 로그 지원
- **Hot Reload**: Air를 사용한 개발 환경

## 주의사항

- 현재 **linting 도구가 설정되지 않음** - 코드 품질은 수동 검토 필요
- `go fmt` 실행 권장
- 테스트 작성 시 `github.com/go-playground/assert/v2` 사용
- 캐시 TTL 기본값: 3600초 (1시간)
- 개발 시 `make dev-setup` 필수 (환경 변수 및 설정 파일 자동 생성)