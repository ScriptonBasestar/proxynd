# Phase 5: 런타임 산출물 및 운영 자산 재배치 TODO

## 개요
- **목표**: 런타임 산출물(`logs/`, `tmp/`, `cache/`)과 운영/배포 자산(`monitoring/`, `systemd/`, `helm/`)을 표준 위치로 재배치
- **우선순위**: Medium
- **예상 소요시간**: 4시간
- **담당자**: Backend/DevOps

## 선행 작업
- [ ] 04-domain-helpers-reorg.md 완료

## 세부 작업 목록

### 1. 브랜치 생성 및 환경 준비
- [ ] **브랜치 생성** (`git checkout -b refactor/ops-assets-reorg`)
  - 새 작업 브랜치 생성
  - 완료 기준: 브랜치 생성 및 최신 상태 확인
  - 주의사항: 이전 단계 완료 후 최신 develop 기반

### 2. 현재 구조 분석 및 이동 계획
- [ ] **런타임 산출물 분석**
  - `logs/`, `tmp/`, `cache/` 디렉토리 내용 및 사용 패턴 확인
  - 완료 기준: 각 디렉토리의 용도 및 VCS 제외 필요성 파악
  - 주의사항: 실행 중인 애플리케이션에서 사용 중인지 확인

- [ ] **운영/배포 자산 분석**
  - `monitoring/`, `systemd/`, `helm/` 디렉토리 내용 확인
  - 완료 기준: 각 자산의 용도 및 의존성 파악
  - 주의사항: CI/CD 파이프라인에서 참조하는 경로 확인

### 3. 배포 자산 디렉토리 생성 및 구조 준비
- [ ] **deployments 디렉토리 생성**
  - `mkdir -p deployments/monitoring deployments/systemd deployments/helm`
  - 완료 기준: 모든 배포 관련 대상 디렉토리 생성
  - 주의사항: 기존 파일과 충돌 없는지 확인

- [ ] **런타임 디렉토리 생성 (선택사항)**
  - `mkdir -p var/logs var/tmp var/cache`
  - 완료 기준: 런타임 산출물 대상 디렉토리 생성
  - 주의사항: VCS 추적 제외 대상으로 관리

### 4. 모니터링 자산 이동
- [ ] **monitoring 디렉토리 이동** (`monitoring/*` → `deployments/monitoring/`)
  - Prometheus, Grafana, Loki 설정 등을 deployments로 이동
  - 완료 기준: 루트 monitoring 디렉토리 비움
  - 주의사항: Git 히스토리 보존을 위해 `git mv` 사용

- [ ] **monitoring 경로 참조 업데이트**
  - `grep -RIl "\bmonitoring/\b" . | xargs -I{} sed -i '' -e 's|monitoring/|deployments/monitoring/|g' {}`
  - 문서 및 스크립트의 monitoring 경로 업데이트
  - 완료 기준: 모든 monitoring 경로 참조 변경
  - 주의사항: Makefile, README.md 등의 경로 확인

### 5. systemd 자산 이동
- [ ] **systemd 디렉토리 이동** (`systemd/*` → `deployments/systemd/`)
  - systemd 서비스 파일 및 설치 스크립트 이동
  - 완료 기준: 루트 systemd 디렉토리 비움
  - 주의사항: 서비스 파일 경로 변경이 설치에 미치는 영향 고려

- [ ] **systemd 경로 참조 업데이트**
  - `grep -RIl "\bsystemd/\b" . | xargs -I{} sed -i '' -e 's|systemd/|deployments/systemd/|g' {}`
  - 문서 및 스크립트의 systemd 경로 업데이트
  - 완료 기준: 모든 systemd 경로 참조 변경
  - 주의사항: 설치 스크립트 및 문서 업데이트

### 6. Helm 자산 이동
- [ ] **helm 디렉토리 이동** (`helm/*` → `deployments/helm/`)
  - Helm 차트 및 values 파일 이동
  - 완료 기준: 루트 helm 디렉토리 비움
  - 주의사항: Helm 차트 구조 및 의존성 유지

- [ ] **helm 경로 참조 업데이트**
  - `grep -RIl "\bhelm/\b" . | xargs -I{} sed -i '' -e 's|helm/|deployments/helm/|g' {}`
  - 문서 및 스크립트의 helm 경로 업데이트
  - 완료 기준: 모든 helm 경로 참조 변경
  - 주의사항: Kubernetes 배포 스크립트 및 CI/CD 파이프라인 확인

### 7. .gitignore 업데이트
- [ ] **런타임 산출물 제외 설정**
  - `.gitignore`에 런타임 디렉토리 추가
  - 완료 기준: logs/, tmp/, cache/, var/ 디렉토리가 VCS에서 제외
  - 주의사항: 기존 .gitignore 설정과 중복되지 않도록 확인

### 8. CI/CD 스크립트 업데이트
- [ ] **GitHub Actions 워크플로우 업데이트** (`.github/workflows/`)
  - CI/CD 파이프라인에서 참조하는 경로 업데이트
  - 완료 기준: 모든 워크플로우 파일의 경로 변경 완료
  - 주의사항: 빌드 및 배포 스크립트의 경로 의존성 확인

- [ ] **Makefile 업데이트**
  - Makefile 타겟에서 참조하는 경로 업데이트
  - 완료 기준: 모든 make 타겟 정상 동작
  - 주의사항: docker-build, deploy 등 주요 타겟 검증

### 9. 문서 업데이트
- [ ] **README.md 업데이트**
  - 메인 README의 경로 참조 업데이트
  - 완료 기준: 모든 경로가 새로운 구조를 반영
  - 주의사항: 사용자 가이드 및 예시 명령어 업데이트

- [ ] **docs/ 디렉토리 업데이트**
  - 문서 내의 경로 참조 업데이트
  - 완료 기준: 모든 문서의 경로 정확성 확인
  - 주의사항: 배포 가이드 및 모니터링 설정 문서 특별 확인

## 완료 검증

### 1. 경로 참조 검증
- [ ] **grep을 통한 잔여 참조 확인**
  - 이전 경로 참조가 남아있지 않은지 전체 검색
  - 완료 기준: `monitoring/`, `systemd/`, `helm/` 참조 없음
  - 주의사항: 주석이나 문서 내용도 포함하여 확인

### 2. 빌드 시스템 검증
- [ ] **Makefile 타겟 테스트**
  - 주요 make 타겟들의 정상 동작 확인
  - 완료 기준: `make docker-build`, `make deploy` 등 성공
  - 주의사항: 경로 의존적인 타겟들 우선 확인

- [ ] **Docker 빌드 테스트**
  - Docker 이미지 빌드 정상 동작 확인
  - 완료 기준: 이미지 빌드 성공 및 런타임 오류 없음
  - 주의사항: 복사되는 파일 경로 확인

### 3. 배포 스크립트 검증
- [ ] **Helm 차트 검증** (`helm template` 또는 `helm lint`)
  - Helm 차트 문법 및 구조 정확성 확인
  - 완료 기준: 차트 템플릿 렌더링 성공
  - 주의사항: values.yaml 파일 경로 및 설정 확인

- [ ] **systemd 서비스 파일 검증**
  - 서비스 파일 문법 및 경로 정확성 확인
  - 완료 기준: systemd-analyze verify 통과
  - 주의사항: ExecStart 경로 등 절대 경로 사용 확인

### 4. 모니터링 설정 검증
- [ ] **Prometheus 설정 검증**
  - prometheus.yml 파일 문법 확인
  - 완료 기준: Prometheus 설정 로드 성공
  - 주의사항: 스크래핑 타겟 및 경로 정확성

- [ ] **Grafana 대시보드 검증**
  - 대시보드 JSON 파일 정확성 확인
  - 완료 기준: 대시보드 임포트 성공
  - 주의사항: 데이터 소스 및 메트릭 쿼리 정확성

### 5. 런타임 동작 검증
- [ ] **애플리케이션 실행 테스트**
  - 새로운 구조에서 애플리케이션 정상 시작 확인
  - 완료 기준: 서버 정상 기동 및 로그 출력
  - 주의사항: 런타임 디렉토리 생성 및 권한 확인

- [ ] **로그 및 캐시 생성 확인**
  - 런타임 중 파일 생성 위치 확인
  - 완료 기준: 적절한 위치에 파일 생성
  - 주의사항: 환경변수 설정에 따른 경로 변경 반영

## 관련 파일
- `monitoring/` (이동 대상)
- `systemd/` (이동 대상)
- `helm/` (이동 대상)
- `logs/`, `tmp/`, `cache/` (VCS 제외 대상)
- `deployments/` (생성될 디렉토리)
- `.gitignore` (업데이트 대상)
- `.github/workflows/` (경로 업데이트 대상)
- `Makefile` (경로 업데이트 대상)
- `README.md` 및 `docs/` (문서 업데이트 대상)

## 롤백 계획
```bash
git restore --staged -W .
git checkout -- .
git checkout -
git branch -D refactor/ops-assets-reorg
```

## 완료 후 상태
- [ ] 운영/배포 자산이 `deployments/` 하위로 정리
- [ ] 런타임 산출물이 VCS에서 적절히 제외
- [ ] 모든 경로 참조가 새 구조를 반영
- [ ] CI/CD 파이프라인 정상 동작
- [ ] 배포 스크립트 및 모니터링 설정 정상 동작
- [ ] 애플리케이션 런타임 동작 정상