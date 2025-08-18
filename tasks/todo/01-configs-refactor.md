# Phase 1: 설정 디렉토리 표준화 TODO

## 개요
- **목표**: `configs/` 디렉토리에서 Go 코드를 제거하고 `internal/config/`로 이동, 설정 파일은 `configs/`로 통합
- **우선순위**: High
- **예상 소요시간**: 3시간
- **담당자**: Backend/Fullstack

## 선행 작업
- [ ] develop 브랜치가 그린 상태 확인

## 세부 작업 목록

### 1. 브랜치 생성 및 환경 준비
- [ ] **브랜치 생성** (`git checkout -b refactor/configs-standardize`)
  - 새 작업 브랜치 생성
  - 완료 기준: 브랜치가 생성되고 최신 develop 기반으로 설정됨
  - 주의사항: 작업 전 `git pull --rebase` 수행

### 2. 디렉토리 구조 분석
- [ ] **기존 구조 확인** (`configs/connection_pool_config.go` 및 `sample-conf/connection-pool.yaml`)
  - 현재 configs 디렉토리의 Go 파일 존재 확인
  - sample-conf 디렉토리 내용 확인
  - 완료 기준: 이동 대상 파일 목록 명확화
  - 주의사항: internal/config 디렉토리의 기존 파일과 충돌 가능성 확인

### 3. Go 코드 이동
- [ ] **internal/config 디렉토리 생성** (`mkdir -p internal/config`)
  - 대상 디렉토리 준비
  - 완료 기준: 디렉토리 생성 완료
  
- [ ] **Go 파일 이동** (`configs/connection_pool_config.go` → `internal/config/connection_pool_config.go`)
  - Git을 이용한 파일 이동
  - 완료 기준: 파일이 새 위치로 이동되고 Git 히스토리 보존
  - 주의사항: 기존 internal/config의 파일과 충돌 시 병합 필요

### 4. 설정 파일 통합
- [ ] **configs 디렉토리 준비** (`mkdir -p configs`)
  - 설정 파일 대상 디렉토리 확인/생성
  - 완료 기준: configs 디렉토리 존재
  
- [ ] **설정 샘플 이동** (`sample-conf/connection-pool.yaml` → `configs/connection-pool.yaml`)
  - 설정 파일을 표준 위치로 이동
  - 완료 기준: 설정 파일이 configs 디렉토리로 이동
  - 주의사항: sample-conf 디렉토리 완전 제거 계획

### 5. Import 경로 수정
- [ ] **import 경로 일괄 수정**
  - `grep -RIl "\bproxynd/configs\b" . | xargs -I{} sed -i '' -e 's|proxynd/configs|proxynd/internal/config|g' {}`
  - 모든 소스 파일에서 import 경로 업데이트
  - 완료 기준: grep으로 검색 시 이전 경로 참조 없음
  - 주의사항: macOS sed 문법 사용

- [ ] **sample-conf 참조 제거**
  - `grep -RIl "sample-conf" . | xargs -I{} sed -i '' -e 's|sample-conf/|configs/|g' {}`
  - 문서나 스크립트의 경로 참조 업데이트
  - 완료 기준: sample-conf 참조가 모두 configs로 변경

### 6. 패키지명 수정
- [ ] **package 선언 확인 및 수정** (`internal/config/connection_pool_config.go`)
  - 파일 상단의 package 선언을 적절히 수정
  - 완료 기준: package config 또는 적절한 패키지명으로 설정
  - 주의사항: internal/config 디렉토리의 기존 패키지 구조와 일치

## 완료 검증

### 1. 빌드 및 테스트 검증
- [ ] **의존성 정리** (`go mod tidy`)
  - 모듈 의존성 정리
  - 완료 기준: go.mod 파일 정리 완료
  
- [ ] **코드 포맷팅** (`go fmt ./...`)
  - 모든 Go 파일 포맷팅
  - 완료 기준: 포맷팅 변경사항 없음

- [ ] **정적 분석** (`go vet ./...`)
  - 정적 분석 통과
  - 완료 기준: vet 오류 없음

- [ ] **린팅** (`golangci-lint run` - 선택사항)
  - 코드 품질 검증
  - 완료 기준: 린트 오류 해결

- [ ] **빌드 테스트** (`go build ./...`)
  - 전체 프로젝트 빌드
  - 완료 기준: 빌드 성공
  - 주의사항: 컴파일 오류 시 import 경로 재확인

- [ ] **단위 테스트** (`go test ./...`)
  - 모든 단위 테스트 실행
  - 완료 기준: 테스트 통과
  - 주의사항: 설정 로딩 관련 테스트 특별 확인

### 2. 기능 검증
- [ ] **설정 로딩 검증**
  - 애플리케이션 시작 시 설정 정상 로딩 확인
  - 완료 기준: 연결풀 설정이 정상적으로 로드됨
  - 주의사항: 새 경로의 설정 파일 사용 확인

- [ ] **연결풀 기능 테스트**
  - 연결풀이 사용되는 핸들러/미들웨어 동작 확인
  - 완료 기준: 관련 엔드포인트 정상 응답
  - 주의사항: 수동 smoke 테스트 필수

## 관련 파일
- `configs/connection_pool_config.go` (이동 대상)
- `sample-conf/connection-pool.yaml` (이동 대상)
- `internal/config/` (대상 디렉토리)
- 모든 Go 소스 파일 (import 경로 수정 대상)
- `.gitignore` (필요시 수정)

## 롤백 계획
```bash
git restore --staged -W .
git checkout -- .
git checkout -
git branch -D refactor/configs-standardize
```

## 완료 후 상태
- [ ] `configs/`에는 설정 파일만 존재
- [ ] `sample-conf/` 디렉토리 제거
- [ ] `internal/config/`에 Go 코드 위치
- [ ] 모든 import 경로 업데이트 완료
- [ ] 빌드 및 테스트 성공