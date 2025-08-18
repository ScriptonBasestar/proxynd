## 리팩토링: 관측성 정리(logging/metrics)

### 배경/목표
- 루트의 `logging/`, `metrics/`는 앱 내부 구현으로 보이며, `internal/logging/`도 존재합니다.
- 목표: 로깅/메트릭 구현을 `internal/`로 일원화하고, 중복/충돌을 제거합니다.

### 범위
- 이동 대상
  - `logging/` → `internal/logging/` (이미 존재하는 경우 병합)
  - `metrics/` → `internal/metrics/`
- import 경로 변경
  - `proxynd/logging` → `proxynd/internal/logging`
  - `proxynd/metrics` → `proxynd/internal/metrics`

### 단계별 작업 지침
1) 브랜치 생성
```bash
git checkout -b refactor/observability-consolidation
```

2) 파일 이동 및 병합
```bash
mkdir -p internal/logging internal/metrics
# 충돌 가능성 높음: 동일 파일명/타입 존재 시 diff 확인 후 병합

git mv logging/* internal/logging/ 2>/dev/null || mv logging/* internal/logging/

git mv metrics/* internal/metrics/ 2>/dev/null || mv metrics/* internal/metrics/
```

3) import 경로 수정
```bash
grep -RIl "\bproxynd/logging\b" . | xargs -I{} sed -i '' -e 's|proxynd/logging|proxynd/internal/logging|g' {}

grep -RIl "\bproxynd/metrics\b" . | xargs -I{} sed -i '' -e 's|proxynd/metrics|proxynd/internal/metrics|g' {}
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
- 로깅 초기화(예: zap/zerolog) 및 글로벌 로거 주입 방식이 바뀌면 앱 전역 영향이 큽니다. 초기화 경로를 문서화하고 단일 진입점으로 정리하세요.
- 메트릭 네임스페이스/라벨 변경 시 모니터링 대시보드에 영향이 있으니, 변경 내역을 `monitoring/`(차후 `deployments/monitoring/`) 쪽에도 반영하세요.

### 검증 방법
- 단위/통합 테스트 실행, 서버 기동 후 로그 출력/메트릭 엔드포인트(`/metrics`) 확인

### 롤백 전략
```bash
git restore --staged -W .
git checkout -- .
git checkout -
git branch -D refactor/observability-consolidation
```

### 완료 기준 체크리스트
- [ ] `internal/logging/`, `internal/metrics/`로 일원화
- [ ] import 경로 정리 후 빌드/테스트 통과
- [ ] 모니터링 대시보드 영향 검토 완료
