## 리팩토링: configs 디렉토리 표준화

### 배경/목표
- Community 표준 레이아웃에서는 `configs/`는 설정 파일 템플릿(YAML/TOML/JSON 등)만 두고, 로딩/검증 로직은 코드 디렉토리(`internal/config/` 등)에 둡니다.
- 현재 상태:
  - `configs/connection_pool_config.go`에 Go 소스가 존재(비표준)
  - `sample-conf/connection-pool.yaml`로 설정 샘플이 분리되어 있음(중복/분산)
- 목표:
  - `configs/`에는 설정 파일만 남기고, Go 코드는 `internal/config/`로 이동
  - 설정 샘플은 `configs/`로 통합, `sample-conf/` 제거

### 범위
- 이동 대상
  - `configs/connection_pool_config.go` → `internal/config/connection_pool_config.go`
  - `sample-conf/connection-pool.yaml` → `configs/connection-pool.yaml`
- 검색/수정 대상
  - 코드 내 import 경로: `proxynd/configs` → `proxynd/internal/config`
  - 혹시 존재한다면 `proxynd/sample-conf` 참조 제거

### 단계별 작업 지침
1) 브랜치 생성
```bash
git checkout -b refactor/configs-standardize
```

2) 파일 이동(존재하지 않으면 디렉토리 생성)
```bash
mkdir -p internal/config
# Go 코드 이동 (패키지명이 맞지 않을 경우, 파일 상단 package 선언을 internal/config 구조에 맞게 수정 필요)
git mv configs/connection_pool_config.go internal/config/connection_pool_config.go || mv configs/connection_pool_config.go internal/config/connection_pool_config.go

# 설정 샘플 이동
mkdir -p configs
git mv sample-conf/connection-pool.yaml configs/connection-pool.yaml || mv sample-conf/connection-pool.yaml configs/connection-pool.yaml
```

3) import 경로 일괄 수정(macOS BSD sed 기준)
```bash
# configs 패키지를 import 하던 경우 internal/config로 교체
grep -RIl "\bproxynd/configs\b" . | xargs -I{} sed -i '' -e 's|proxynd/configs|proxynd/internal/config|g' {}

# 혹시 남아있는 sample-conf 경로 문자열이 문서/스크립트에 있으면 정리(선택)
grep -RIl "sample-conf" . | xargs -I{} sed -i '' -e 's|sample-conf/|configs/|g' {}
```

4) 의존성/포맷 정리 및 빌드/테스트
```bash
go mod tidy
go fmt ./...
go vet ./...
# golangci-lint가 설치되어 있다면
if command -v golangci-lint >/dev/null 2>&1; then golangci-lint run; fi

go build ./...
go test ./...
```

### 코드 영향 및 주의사항
- `configs/connection_pool_config.go`의 package 선언이 `package configs`였다면, 이동 후 `package config` 등으로 변경되어야 할 수 있습니다. 컴파일 에러가 나면 패키지명/경로를 우선 확인하세요.
- `internal/config`에 이미 유사 구조가 존재하므로, 파일 충돌/타입 중복 여부를 먼저 확인하세요. 충돌 시 파일을 분할/병합하고, 타입/함수명을 명확하게 리네임하세요.

### 검증 방법
- `go build ./...`가 성공해야 합니다.
- 설정 로딩 경로가 바뀌었다면, `main.go` 또는 초기화 로직에서 실제 로드되는지 통합 테스트로 확인하세요.
- 연결 풀 설정이 사용되는 라우터/미들웨어 경로에서 정상 동작을 수동 테스트하세요.

### 롤백 전략
```bash
git restore --staged -W .
git checkout -- .
# 혹은 브랜치 삭제
git checkout -
git branch -D refactor/configs-standardize
```

### 완료 기준 체크리스트
- [ ] `configs/`에는 코드가 없고 설정 파일만 남음
- [ ] `sample-conf/`가 제거되고 샘플 파일이 `configs/`로 통합됨
- [ ] import 경로 업데이트로 컴파일 및 테스트가 통과됨
- [ ] 충돌/중복 타입 정리 완료
