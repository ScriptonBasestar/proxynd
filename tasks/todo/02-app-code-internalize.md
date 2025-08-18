# Phase 2: 앱 코드 내부화 TODO

## 개요
- **목표**: 루트의 `routers/`, `handlers/`, `middlewares/`를 `internal/` 하위로 이동하여 캡슐화
- **우선순위**: High
- **예상 소요시간**: 4시간
- **담당자**: Backend/Fullstack

## 선행 작업
- [ ] 01-configs-refactor.md 완료

## 세부 작업 목록

### 1. 브랜치 생성 및 환경 준비
- [ ] **브랜치 생성** (`git checkout -b refactor/app-code-internalize`)
  - 새 작업 브랜치 생성
  - 완료 기준: 브랜치 생성 및 최신 상태 확인
  - 주의사항: 이전 단계 완료 후 최신 develop 기반

### 2. 기존 구조 분석 및 충돌 확인
- [ ] **기존 internal 디렉토리 확인** (`internal/handlers/`, `internal/middleware/` 등)
  - 이동 대상과 기존 파일 충돌 가능성 분석
  - 완료 기준: 충돌 파일 목록 작성 및 병합 계획 수립
  - 주의사항: 파일명/타입명 중복 시 병합 또는 리네임 필요

### 3. 디렉토리 생성 및 구조 준비
- [ ] **대상 디렉토리 생성** (`mkdir -p internal/routers internal/handlers internal/middleware`)
  - 이동 대상 디렉토리 준비
  - 완료 기준: 모든 대상 디렉토리 생성
  - 주의사항: middlewares → middleware (복수형에서 단수형으로 통일)

### 4. 라우터 코드 이동
- [ ] **routers 디렉토리 이동** (`routers/*` → `internal/routers/`)
  - 모든 라우터 파일을 internal로 이동
  - 완료 기준: 루트 routers 디렉토리 비움
  - 주의사항: Git 히스토리 보존을 위해 `git mv` 사용

### 5. 핸들러 코드 이동  
- [ ] **handlers 디렉토리 이동** (`handlers/*` → `internal/handlers/`)
  - 모든 핸들러 파일을 internal로 이동
  - 완료 기준: 루트 handlers 디렉토리 비움
  - 주의사항: 기존 internal/handlers와 병합 필요 시 충돌 해결

### 6. 미들웨어 코드 이동
- [ ] **middlewares 디렉토리 이동** (`middlewares/*` → `internal/middleware/`)
  - 모든 미들웨어 파일을 internal/middleware로 이동
  - 완료 기준: 루트 middlewares 디렉토리 비움
  - 주의사항: 기존 internal/middleware와 충돌 시 병합

### 7. Import 경로 수정
- [ ] **routers import 경로 수정**
  - `grep -RIl "\bproxynd/routers\b" . | xargs -I{} sed -i '' -e 's|proxynd/routers|proxynd/internal/routers|g' {}`
  - 모든 라우터 참조 경로 업데이트
  - 완료 기준: 이전 경로 참조 완전 제거

- [ ] **handlers import 경로 수정**
  - `grep -RIl "\bproxynd/handlers\b" . | xargs -I{} sed -i '' -e 's|proxynd/handlers|proxynd/internal/handlers|g' {}`
  - 모든 핸들러 참조 경로 업데이트
  - 완료 기준: 이전 경로 참조 완전 제거

- [ ] **middlewares import 경로 수정**
  - `grep -RIl "\bproxynd/middlewares\b" . | xargs -I{} sed -i '' -e 's|proxynd/middlewares|proxynd/internal/middleware|g' {}`
  - 모든 미들웨어 참조 경로 업데이트 (복수형 → 단수형)
  - 완료 기준: 이전 경로 참조 완전 제거
  - 주의사항: middlewares → middleware로 디렉토리명 변경

### 8. 패키지명 검증 및 수정
- [ ] **패키지 선언 확인** (각 이동된 파일의 package 선언)
  - 파일 내 package 선언이 디렉토리명과 일치하는지 확인
  - 완료 기준: 모든 파일의 package 선언 적절히 설정
  - 주의사항: 컴파일 오류 방지를 위한 패키지명 정합성

### 9. 순환 의존성 검증
- [ ] **의존성 방향 확인**
  - internal/routers → internal/handlers → internal/middleware 의존성 방향 확인
  - 완료 기준: 순환 의존성 없음을 확인
  - 주의사항: 순환 참조 발견 시 인터페이스 분리 또는 구조 재설계

## 완료 검증

### 1. 빌드 및 테스트 검증
- [ ] **코드 포맷팅** (`go fmt ./...`)
  - 모든 Go 파일 포맷팅
  - 완료 기준: 포맷팅 변경사항 없음

- [ ] **정적 분석** (`go vet ./...`)
  - 정적 분석 통과
  - 완료 기준: vet 오류 없음
  - 주의사항: 순환 의존성 관련 오류 특별 확인

- [ ] **린팅** (`golangci-lint run` - 선택사항)
  - 코드 품질 검증
  - 완료 기준: 린트 오류 해결

- [ ] **빌드 테스트** (`go build ./...`)
  - 전체 프로젝트 빌드
  - 완료 기준: 빌드 성공
  - 주의사항: import 경로 오류 시 재확인

- [ ] **단위 테스트** (`go test ./...`)
  - 모든 단위 테스트 실행
  - 완료 기준: 테스트 통과

### 2. 통합 테스트 검증
- [ ] **라우팅 엔드포인트 테스트** (`tests/integration` 또는 수동 테스트)
  - 주요 API 엔드포인트 동작 확인
  - 완료 기준: 핵심 엔드포인트 정상 응답
  - 주의사항: 라우터-핸들러-미들웨어 연결 상태 확인

### 3. 서버 동작 검증
- [ ] **서버 기동 테스트**
  - 애플리케이션 정상 시작 확인
  - 완료 기준: 서버 정상 기동 및 로그 출력
  - 주의사항: 초기화 오류 여부 확인

- [ ] **핵심 기능 smoke 테스트**
  - 주요 프록시 기능 동작 확인
  - 완료 기준: NPM, Maven, Docker 등 주요 프록시 동작
  - 주의사항: 각 패키지 매니저별 기본 요청/응답 확인

## 관련 파일
- `routers/` (이동 대상)
- `handlers/` (이동 대상) 
- `middlewares/` (이동 대상)
- `internal/routers/` (생성될 디렉토리)
- `internal/handlers/` (병합 대상)
- `internal/middleware/` (병합 대상)
- 모든 Go 소스 파일 (import 경로 수정 대상)
- `main.go` (라우터 초기화 코드)

## 롤백 계획
```bash
git restore --staged -W .
git checkout -- .
git checkout -
git branch -D refactor/app-code-internalize
```

## 완료 후 상태
- [ ] 루트의 `routers/`, `handlers/`, `middlewares/` 디렉토리 제거
- [ ] `internal/` 하위로 모든 앱 코드 이동 완료
- [ ] 모든 import 경로 업데이트 완료
- [ ] 순환 의존성 없음
- [ ] 빌드 및 통합 테스트 성공
- [ ] 서버 정상 기동 및 핵심 기능 동작