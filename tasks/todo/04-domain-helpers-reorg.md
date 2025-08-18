# Phase 4: 도메인 보조 모듈 정리 TODO

## 개요
- **목표**: 루트의 보조 모듈들(`dtos/`, `helpers/`, `storage/`, `verification/`, `alerts/`, `performance/`)을 적절한 위치로 이동
- **우선순리**: Medium
- **예상 소요시간**: 6시간
- **담당자**: Backend/Fullstack

## 선행 작업
- [ ] 03-observability-consolidation.md 완료

## 세부 작업 목록

### 1. 브랜치 생성 및 환경 준비
- [ ] **브랜치 생성** (`git checkout -b refactor/domain-helpers-reorg`)
  - 새 작업 브랜치 생성
  - 완료 기준: 브랜치 생성 및 최신 상태 확인
  - 주의사항: 이전 단계 완료 후 최신 develop 기반

### 2. 모듈별 분석 및 이동 계획 수립
- [ ] **DTOs 모듈 분석** (`dtos/`)
  - API 응답/요청 구조체 내용 확인
  - 완료 기준: 외부 노출 필요성 판단 및 이동 경로 결정
  - 주의사항: `internal/dto/` 또는 `internal/api/dto/` 중 선택

- [ ] **Helpers 모듈 분석** (`helpers/`)
  - 유틸리티 함수들의 재사용성 분석
  - 완료 기준: 내부용/외부 공개용 분류 완료
  - 주의사항: `internal/helpers/` vs `pkg/helpers/` 결정

- [ ] **Storage 모듈 분석** (`storage/`)
  - 저장소 관련 코드의 성격 분석
  - 완료 기준: 내부 구현체 vs 공개 인터페이스 분류
  - 주의사항: 캐시 구현과의 중복 여부 확인

- [ ] **Verification 모듈 분석** (`verification/`)
  - 패키지 검증 로직 내용 확인
  - 완료 기준: 내부 구현으로 분류 및 `internal/verification/` 이동 계획
  - 주의사항: 외부 패키지 매니저와의 의존성 확인

- [ ] **Alerts 모듈 분석** (`alerts/`)
  - 알림 시스템 구조 확인
  - 완료 기준: `internal/alerts/` 또는 `internal/monitoring/alerts/` 결정
  - 주의사항: 기존 모니터링 시스템과의 통합 고려

- [ ] **Performance 모듈 분석** (`performance/`)
  - 기존 `internal/performance/`와의 중복 확인
  - 완료 기준: 병합 또는 분리 전략 수립
  - 주의사항: 성능 모니터링 중복 구현 제거

### 3. 디렉토리 생성 및 구조 준비
- [ ] **대상 디렉토리 생성**
  - `mkdir -p internal/dto internal/helpers internal/storage internal/verification internal/alerts internal/performance pkg/helpers pkg/storage`
  - 완료 기준: 모든 필요 디렉토리 생성
  - 주의사항: 기존 디렉토리와의 충돌 확인

### 4. DTOs 모듈 이동
- [ ] **DTOs 이동** (`dtos/*` → `internal/dto/`)
  - API 구조체들을 내부 모듈로 이동
  - 완료 기준: 루트 dtos 디렉토리 비움
  - 주의사항: HTTP 응답 구조체의 일관성 유지

- [ ] **DTOs import 경로 수정**
  - `grep -RIl "\bproxynd/dtos\b" . | xargs -I{} sed -i '' -e 's|proxynd/dtos|proxynd/internal/dto|g' {}`
  - 모든 DTO 참조 경로 업데이트
  - 완료 기준: 이전 경로 참조 완전 제거

### 5. Helpers 모듈 이동 및 분류
- [ ] **Helpers 내부 이동** (`helpers/*` → `internal/helpers/`)
  - 우선 모든 helper를 internal로 이동
  - 완료 기준: 루트 helpers 디렉토리 비움
  - 주의사항: 추후 외부 공개 필요 시 pkg로 재이동

- [ ] **Helpers 외부 공개 검토**
  - 재사용 가능한 유틸리티의 pkg 이동 검토
  - 완료 기준: 외부 공개 대상 식별 및 분류
  - 주의사항: URL helper, YAML helper 등의 범용성 고려

- [ ] **Helpers import 경로 수정**
  - `grep -RIl "\bproxynd/helpers\b" . | xargs -I{} sed -i '' -e 's|proxynd/helpers|proxynd/internal/helpers|g' {}`
  - 모든 helper 참조 경로 업데이트
  - 완료 기준: 이전 경로 참조 완전 제거

### 6. Storage 모듈 이동
- [ ] **Storage 이동** (`storage/*` → `internal/storage/`)
  - 저장소 구현을 내부 모듈로 이동
  - 완료 기준: 루트 storage 디렉토리 비움
  - 주의사항: 기존 캐시 시스템과의 중복 확인

- [ ] **Storage import 경로 수정**
  - `grep -RIl "\bproxynd/storage\b" . | xargs -I{} sed -i '' -e 's|proxynd/storage|proxynd/internal/storage|g' {}`
  - 모든 저장소 참조 경로 업데이트
  - 완료 기준: 이전 경로 참조 완전 제거

### 7. Verification 모듈 이동
- [ ] **Verification 이동** (`verification/*` → `internal/verification/`)
  - 패키지 검증 로직을 내부로 이동
  - 완료 기준: 루트 verification 디렉토리 비움
  - 주의사항: 각 패키지 매니저별 검증 로직 유지

- [ ] **Verification import 경로 수정**
  - `grep -RIl "\bproxynd/verification\b" . | xargs -I{} sed -i '' -e 's|proxynd/verification|proxynd/internal/verification|g' {}`
  - 모든 검증 참조 경로 업데이트
  - 완료 기준: 이전 경로 참조 완전 제거

### 8. Alerts 모듈 이동 및 통합
- [ ] **Alerts 이동** (`alerts/*` → `internal/alerts/`)
  - 알림 시스템을 내부로 이동
  - 완료 기준: 루트 alerts 디렉토리 비움
  - 주의사항: 웹훅 알림 및 로그 알림 기능 유지

- [ ] **Alerts import 경로 수정**
  - `grep -RIl "\bproxynd/alerts\b" . | xargs -I{} sed -i '' -e 's|proxynd/alerts|proxynd/internal/alerts|g' {}`
  - 모든 알림 참조 경로 업데이트
  - 완료 기준: 이전 경로 참조 완전 제거

### 9. Performance 모듈 병합
- [ ] **Performance 중복 분석**
  - 루트 `performance/`와 `internal/performance/` 비교
  - 완료 기준: 중복 구현 식별 및 병합 계획 수립
  - 주의사항: 성능 최적화 로직 중복 제거

- [ ] **Performance 병합** (`performance/*` → `internal/performance/`)
  - 중복 제거 후 통합된 성능 모듈 구성
  - 완료 기준: 루트 performance 디렉토리 비움 및 중복 제거
  - 주의사항: 성능 모니터링 기능 정상 동작 유지

- [ ] **Performance import 경로 수정**
  - `grep -RIl "\bproxynd/performance\b" . | xargs -I{} sed -i '' -e 's|proxynd/performance|proxynd/internal/performance|g' {}`
  - 모든 성능 참조 경로 업데이트
  - 완료 기준: 이전 경로 참조 완전 제거

### 10. 패키지명 검증 및 타입 충돌 해결
- [ ] **패키지 선언 통일**
  - 각 이동된 파일의 package 선언 확인 및 수정
  - 완료 기준: 디렉토리명과 패키지명 일치
  - 주의사항: 컴파일 오류 방지

- [ ] **타입명 충돌 해결**
  - 동일 타입명의 중복 정의 해결
  - 완료 기준: 모든 타입명 고유성 확보
  - 주의사항: 명확한 네이밍으로 타입 구분

## 완료 검증

### 1. 빌드 및 테스트 검증
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

- [ ] **단위 테스트** (`go test ./...`)
  - 모든 단위 테스트 실행
  - 완료 기준: 테스트 통과

### 2. 기능별 검증
- [ ] **저장소 기능 테스트**
  - 캐시 저장/조회 기능 정상 동작 확인
  - 완료 기준: 파일시스템 및 S3 캐시 정상 동작
  - 주의사항: 저장소 경로 변경 영향 없음

- [ ] **검증 기능 테스트**
  - 패키지 검증 로직 정상 동작 확인
  - 완료 기준: APK, NPM 등 주요 패키지 검증 동작
  - 주의사항: 서명 검증 및 무결성 검사 유지

- [ ] **알림 기능 테스트**
  - 웹훅 및 로그 알림 기능 확인
  - 완료 기준: 알림 전송 정상 동작
  - 주의사항: 웹훅 이벤트 형식 유지

- [ ] **성능 도구 테스트**
  - 성능 모니터링 및 최적화 도구 동작 확인
  - 완료 기준: 성능 메트릭 수집 정상 동작
  - 주의사항: 캐시 최적화 전략 유지

## 관련 파일
- `dtos/` (이동 대상)
- `helpers/` (이동 대상)
- `storage/` (이동 대상)
- `verification/` (이동 대상)
- `alerts/` (이동 대상)
- `performance/` (이동/병합 대상)
- `internal/performance/` (병합 대상)
- 모든 Go 소스 파일 (import 경로 수정 대상)

## 롤백 계획
```bash
git restore --staged -W .
git checkout -- .
git checkout -
git branch -D refactor/domain-helpers-reorg
```

## 완료 후 상태
- [ ] 모든 보조 모듈이 적절한 위치(`internal/` 또는 `pkg/`)로 이동
- [ ] 루트의 보조 디렉토리들 제거
- [ ] 모든 import 경로 업데이트 완료
- [ ] 중복 구현 및 타입 정리 완료
- [ ] 빌드 및 테스트 성공
- [ ] 모든 기능 정상 동작