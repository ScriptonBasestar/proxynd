# ProxyND 헥사고널 아키텍처 리팩토링 TODO 요약

## 📋 생성된 TODO 목록

### Phase 1: 핵심 인프라 (필수 우선 실행)
1. **[01-configs-refactor.md](01-configs-refactor.md)** - 설정 디렉토리 표준화 (HIGH) - 3시간
   - `configs/`에서 Go 코드 제거, `internal/config/`로 이동
   - 설정 샘플을 `configs/`로 통합
   - **선행작업**: develop 브랜치 그린 상태

2. **[02-app-code-internalize.md](02-app-code-internalize.md)** - 앱 코드 내부화 (HIGH) - 4시간  
   - `routers/`, `handlers/`, `middlewares/`를 `internal/` 하위로 이동
   - 캡슐화 및 순환 의존성 제거
   - **선행작업**: 01-configs-refactor.md 완료

3. **[03-observability-consolidation.md](03-observability-consolidation.md)** - 관측성 정리 (HIGH) - 5시간
   - `logging/`, `metrics/`를 `internal/`로 일원화  
   - 중복 구현 제거 및 단일 진입점 확보
   - **선행작업**: 02-app-code-internalize.md 완료

### Phase 2: 도메인 및 지원 모듈 (병렬 가능)
4. **[04-domain-helpers-reorg.md](04-domain-helpers-reorg.md)** - 도메인 보조 모듈 정리 (MEDIUM) - 6시간
   - `dtos/`, `helpers/`, `storage/`, `verification/`, `alerts/`, `performance/` 정리
   - 적절한 위치(`internal/` 또는 `pkg/`)로 분류 이동
   - **선행작업**: 03-observability-consolidation.md 완료

5. **[05-runtime-artifacts-reorg.md](05-runtime-artifacts-reorg.md)** - 런타임/운영 자산 재배치 (MEDIUM) - 4시간
   - 런타임 산출물(`logs/`, `tmp/`, `cache/`) VCS 제외
   - 운영 자산(`monitoring/`, `systemd/`, `helm/`)을 `deployments/`로 이동
   - **선행작업**: 04-domain-helpers-reorg.md 완료 (권장)

### Phase 3: 마무리 및 검증
6. **[06-tests-naming.md](06-tests-naming.md)** - 테스트 구조 정리 (LOW) - 2시간
   - 테스트 디렉토리 네이밍 표준화
   - CI/CD 파이프라인 및 스크립트 경로 업데이트
   - **선행작업**: 05-runtime-artifacts-reorg.md 완료

## 🎯 전체 프로젝트 개요

### 목표
**표준 Go 프로젝트 레이아웃 적용 + 헥사고널 아키텍처 원칙 준수**
- 루트에 분산된 코드를 `internal/`로 캡슐화
- 운영/배포 자산을 `deployments/`로 정리
- 중복 구현 제거 및 의존성 정리

### 핵심 원칙 (CLAUDE.md 준수)
- **단방향 의존성**: `adapters → ports → usecase → domain`
- **회귀 방지**: 각 단계별 `make verify-api` 필수
- **점진적 리팩토링**: 파일 이동 → import 수정 → 아키텍처 개선 순서
- **즉시 롤백**: 실패 시 바로 이전 상태로 복구

## 📊 실행 통계

### 총 예상 소요시간: **24시간**
- Phase 1 (핵심): 12시간 (50%)
- Phase 2 (확장): 10시간 (42%)  
- Phase 3 (마무리): 2시간 (8%)

### 권장 구현 순서
```
01 → 02 → 03 → 04 → 05 → 06
(순차)  (순차)  (병렬가능)  (순차)
```

### 병렬 작업 가능성
- Phase 1은 반드시 순차 실행 (상호 의존성 높음)
- Phase 2의 04, 05는 부분적 병렬 가능 (서로 다른 모듈 대상)
- Phase 3은 모든 이전 작업 완료 후 실행

## ⚠️ 중요 주의사항

### 회귀 방지 필수 검증
각 TODO 완료 후 반드시 실행:
```bash
make clean && make build
make test-unit  
make verify-api  # 🚨 가장 중요 - API 회귀 감지
```

### 보호 구역 (절대 수정 금지)
- `scripts/verify-api-endpoints.sh`
- `Makefile` 및 `Makefile.*.mk`
- `.github/workflows/ci.yml`
- `docker-compose.e2e.yml`
- `README.md` (워크플로 문서)

### 즉시 롤백 조건
- 빌드 실패
- 단위 테스트 실패  
- **API 회귀 감지** (가장 중요)
- 아키텍처 의존성 위반

## 🛠️ 실행 준비

### 환경 요구사항
- Go 1.23+ 
- Make 툴체인
- golangci-lint (선택사항)
- Docker (E2E 테스트용)

### 사전 검증
```bash
# 현재 상태 그린 확인
make clean && make build && make test-unit && make verify-api

# 브랜치 최신 상태 확인
git checkout develop
git pull --rebase
```

## 📁 완료 후 최종 구조

```
proxynd/
├── cmd/                    # 애플리케이션 진입점
├── configs/                # 설정 파일만 (Go 코드 제거됨)
├── deployments/           # 새로 생성
│   ├── monitoring/        # monitoring/ → 이동
│   ├── systemd/          # systemd/ → 이동  
│   └── helm/             # helm/ → 이동
├── docs/                  # 문서 (기존 유지)
├── examples/              # 설정 예제 (기존 유지)
├── internal/              # 확장됨
│   ├── adapters/         # 기존 유지
│   ├── config/           # configs/*.go → 이동
│   ├── routers/          # routers/ → 이동
│   ├── handlers/         # handlers/ → 이동 (기존과 병합)
│   ├── middleware/       # middlewares/ → 이동 (기존과 병합)
│   ├── logging/          # logging/ → 이동 (기존과 병합)
│   ├── metrics/          # metrics/ → 이동
│   ├── dto/              # dtos/ → 이동
│   ├── helpers/          # helpers/ → 이동
│   ├── storage/          # storage/ → 이동
│   ├── verification/     # verification/ → 이동
│   ├── alerts/           # alerts/ → 이동
│   ├── performance/      # performance/ → 이동 (기존과 병합)
│   └── ...               # 기존 internal 구조 유지
├── pkg/                   # 외부 공개 라이브러리 (기존 유지)
├── scripts/               # 빌드/배포 스크립트 (기존 유지)
├── test/                  # tests/ → 리네이밍 (선택사항)
│   ├── unit/
│   ├── integration/
│   ├── e2e/
│   └── mocks/
├── testdata/              # 테스트 데이터 (기존 유지)
└── main.go                # 메인 진입점 (기존 유지)

# 제거되는 루트 디렉토리들:
# ❌ routers/, handlers/, middlewares/
# ❌ logging/, metrics/  
# ❌ dtos/, helpers/, storage/, verification/, alerts/, performance/
# ❌ monitoring/, systemd/, helm/
# ❌ logs/, tmp/, cache/ (VCS 제외)
```

## 🚀 시작하기

1. **첫 번째 TODO 시작**:
   ```bash
   # tasks/todo/01-configs-refactor.md 참조
   git checkout develop
   git pull --rebase
   git checkout -b refactor/configs-standardize
   ```

2. **각 TODO 완료 후**: 
   - 해당 TODO 파일의 "완료 검증" 섹션 실행
   - `make verify-api` 반드시 성공 확인
   - PR 생성 및 코드 리뷰

3. **전체 완료 후**:
   - 헥사고널 아키텍처 의존성 검증
   - E2E 테스트 스위트 실행
   - 성능 회귀 테스트

---

**⚠️ 중요**: 각 TODO는 독립적으로 커밋 가능한 단위로 설계되었습니다. 문제 발생 시 언제든 롤백하고 다음 단계로 진행할 수 있습니다.