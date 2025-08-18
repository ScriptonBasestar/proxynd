## 리팩토링: 도메인 보조 모듈 정리(dto/helpers/storage/…)

### 배경/목표
- 루트에 `dtos/`, `helpers/`, `storage/`, `verification/`, `alerts/`, `performance/` 등이 산재해 있습니다.
- 목표: 내부 전용 모듈은 `internal/`로, 외부 공개 필요 라이브러리는 `pkg/`로 이동해 의도를 명확히 합니다.

### 범위
- 이동 후보 및 기본 권장 경로
  - `dtos/` → `internal/dto/` (또는 `internal/api/dto/`)
  - `helpers/` → 내부 전용이면 `internal/helpers/`, 재사용 라이브러리면 `pkg/helpers/`
  - `storage/` → 내부 전용이면 `internal/storage/`, 공개 필요 시 `pkg/storage/`
  - `verification/` → `internal/verification/`
  - `alerts/` → `internal/alerts/` (또는 `internal/monitoring/alerts/`)
  - `performance/` → `internal/performance/` (이미 `internal/performance/`가 있다면 병합)

### 단계별 작업 지침
1) 브랜치 생성
```bash
git checkout -b refactor/domain-helpers-reorg
```

2) 디렉토리 생성 및 이동(필요 시 병합)
```bash
mkdir -p internal/dto internal/helpers internal/storage internal/verification internal/alerts internal/performance pkg/helpers pkg/storage

# 필요에 따라 선택 이동 (아래는 예)
# DTO
git mv dtos/* internal/dto/ 2>/dev/null || mv dtos/* internal/dto/

# Helpers: 우선 internal로 이동 후, 외부 공개 여부 판단해 pkg로 재배치
git mv helpers/* internal/helpers/ 2>/dev/null || mv helpers/* internal/helpers/

# Storage: 내부 전용 가정
git mv storage/* internal/storage/ 2>/dev/null || mv storage/* internal/storage/

# Verification
git mv verification/* internal/verification/ 2>/dev/null || mv verification/* internal/verification/

# Alerts
git mv alerts/* internal/alerts/ 2>/dev/null || mv alerts/* internal/alerts/

# Performance
git mv performance/* internal/performance/ 2>/dev/null || mv performance/* internal/performance/
```

3) import 경로 업데이트
```bash
grep -RIl "\bproxynd/dtos\b" . | xargs -I{} sed -i '' -e 's|proxynd/dtos|proxynd/internal/dto|g' {}

grep -RIl "\bproxynd/helpers\b" . | xargs -I{} sed -i '' -e 's|proxynd/helpers|proxynd/internal/helpers|g' {}

grep -RIl "\bproxynd/storage\b" . | xargs -I{} sed -i '' -e 's|proxynd/storage|proxynd/internal/storage|g' {}

grep -RIl "\bproxynd/verification\b" . | xargs -I{} sed -i '' -e 's|proxynd/verification|proxynd/internal/verification|g' {}

grep -RIl "\bproxynd/alerts\b" . | xargs -I{} sed -i '' -e 's|proxynd/alerts|proxynd/internal/alerts|g' {}

grep -RIl "\bproxynd/performance\b" . | xargs -I{} sed -i '' -e 's|proxynd/performance|proxynd/internal/performance|g' {}
```

4) 포맷/빌드/테스트
```bash
go fmt ./...
go vet ./...
if command -v golangci-lint >/dev/null 2>&1; then golangci-lint run; fi

go build ./...
go test ./...
```

### 코드 영향 및 주의사항
- 외부 공개가 필요한 유틸리티/스토리지는 `pkg/`로 이동해야 import 가능성이 열립니다. 반대로 내부 전용은 반드시 `internal/`에 위치시켜 외부 의존을 차단하세요.
- 동일 명칭의 타입/함수가 `internal/`에 이미 있을 수 있습니다. 충돌 시 명확한 네이밍으로 병합/분리하세요.

### 검증 방법
- 전체 빌드/테스트 통과
- 주요 기능(저장소 접근, 검증, 알림, 성능 도구)이 정상 동작하는지 수동 확인

### 롤백 전략
```bash
git restore --staged -W .
git checkout -- .
git checkout -
git branch -D refactor/domain-helpers-reorg
```

### 완료 기준 체크리스트
- [ ] 보조 모듈이 의도에 맞는 경로(`internal/` 또는 `pkg/`)로 이동
- [ ] import 경로 정리 및 빌드/테스트 통과
- [ ] 중복 구현/타입 정리
