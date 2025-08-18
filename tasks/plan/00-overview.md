## 리팩토링 개요 및 실행 순서

### 목적
- 루트에 분산된 앱/운영 리소스를 표준 Go 레이아웃에 맞게 정리하여 유지보수성과 가독성을 높입니다.
- 충돌/회귀를 최소화하기 위해 작업 단위를 나누고, 각 단계별 검증 게이트를 둡니다.

### 선행 조건
- 현재 `develop`이 그린 상태(빌드/테스트 통과)
- 작업 전 최신 반영: `git pull --rebase`

### 권장 실행 순서
1. `01-configs-refactor.md`: 설정 디렉토리 표준화
2. `02-app-code-relocation.md`: 라우터/핸들러/미들웨어 내부화
3. `03-observability-refactor.md`: 로깅/메트릭 일원화
4. `05-domain-and-helpers-refactor.md`: DTO/헬퍼/스토리지/검증/알림/성능 정리
5. `04-runtime-artifacts-and-ops.md`: 런타임 산출물 및 운영/배포 자산 재배치
6. `06-tests-and-naming.md`: 테스트 디렉토리 네이밍/구조 정리

### 공통 작업 원칙
- 작은 브랜치로 작게 머지(문서 각 항목에 브랜치 제안 포함)
- 이동 후 즉시 import 경로 정리 → 포맷/빌드/테스트 → PR
- 충돌 시 우선 타입/패키지 네이밍을 명확하게 조정하여 순환 의존을 방지
- 운영/배포 경로 변경은 CI/CD 스크립트와 문서까지 동시 반영

### 검증 게이트(각 단계 공통)
- `go fmt ./...` / `go vet ./...` / `go build ./...` / `go test ./...`
- 가능하면 `golangci-lint run`
- 통합/E2E 테스트 스위트 실행(해당 시)
- 수동 스모크 테스트: 서버 기동, 핵심 엔드포인트 확인, 로그/메트릭 정상 여부 확인

### 롤백 가이드(공통)
- 단계별로 작업 브랜치에서 되돌리기: `git restore --staged -W . && git checkout -- .`
- 이전 브랜치로 이동 후 작업 브랜치 삭제: `git checkout - && git branch -D <branch>`

### 산출물
- `tasks/refactoring/01-06` 문서에 각 단계의 세부 지침, 명령어, 체크리스트 포함
