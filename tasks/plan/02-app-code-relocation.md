## 리팩토링: 앱 코드 내부화(routers/handlers/middleware)

### 배경/목표
- 루트에 있는 앱 코드(`routers/`, `handlers/`, `middlewares/`)를 `internal/`로 이동해 캡슐화합니다.
- 기존에 `internal/middleware/`가 존재하므로 `middlewares/`는 `internal/middleware/`로 합칩니다(복수 → 단수).

### 범위
- 이동 대상
  - `routers/` → `internal/routers/`
  - `handlers/` → `internal/handlers/`
  - `middlewares/` → `internal/middleware/`
- import 경로 변경
  - `proxynd/routers` → `proxynd/internal/routers`
  - `proxynd/handlers` → `proxynd/internal/handlers`
  - `proxynd/middlewares` → `proxynd/internal/middleware`

### 단계별 작업 지침
1) 브랜치 생성
```bash
git checkout -b refactor/app-code-internalize
```

2) 이동(디렉토리 생성 및 병합 주의)
```bash
mkdir -p internal/routers internal/handlers internal/middleware
# 중복 파일명/타입 충돌이 있으면 우선 비교 후 병합

git mv routers/* internal/routers/ 2>/dev/null || mv routers/* internal/routers/

git mv handlers/* internal/handlers/ 2>/dev/null || mv handlers/* internal/handlers/

# middlewares → middleware (디렉토리명 변경)
git mv middlewares/* internal/middleware/ 2>/dev/null || mv middlewares/* internal/middleware/
```

3) import 경로 일괄 수정(macOS BSD sed)
```bash
grep -RIl "\bproxynd/routers\b" . | xargs -I{} sed -i '' -e 's|proxynd/routers|proxynd/internal/routers|g' {}

grep -RIl "\bproxynd/handlers\b" . | xargs -I{} sed -i '' -e 's|proxynd/handlers|proxynd/internal/handlers|g' {}

grep -RIl "\bproxynd/middlewares\b" . | xargs -I{} sed -i '' -e 's|proxynd/middlewares|proxynd/internal/middleware|g' {}
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
- 패키지명이 디렉토리명과 불일치할 경우 컴파일 에러가 발생할 수 있습니다. 필요한 경우 `package` 선언을 정리하세요.
- 순환 의존이 없는지 확인하세요. `internal/routers`가 `internal/handlers`를, `internal/middleware`가 `internal/routers`를 다시 참조하는 등 순환 구조는 금지합니다.

### 검증 방법
- 라우팅/핸들러 엔드포인트에 대한 통합 테스트(또는 `tests/integration`) 실행
- 서버 기동 후 핵심 엔드포인트 수동 점검

### 롤백 전략
```bash
git restore --staged -W .
git checkout -- .
# 브랜치 폐기
git checkout -
git branch -D refactor/app-code-internalize
```

### 완료 기준 체크리스트
- [ ] 루트의 `routers/`, `handlers/`, `middlewares/`가 비어있거나 제거됨
- [ ] `internal/` 하위로 이동 및 import 경로 정리 완료
- [ ] 빌드/테스트 통과 및 순환 의존 없음
