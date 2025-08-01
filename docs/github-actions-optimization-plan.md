# GitHub Actions 워크플로우 최적화 계획

## 현재 상황 분석

### 워크플로우 현황
- **총 5개 워크플로우**, 1,976줄
- 과도한 중복과 복잡성
- 유지보수 비용 증가

### 주요 문제점
1. **보안 스캔 중복** (ci-cd.yml ↔ security-scan.yml)
2. **Docker 빌드 중복** (ci-cd.yml ↔ release.yml)
3. **배포 로직 중복** (ci-cd.yml ↔ release.yml)
4. **의존성 검사 중복** (security-scan.yml ↔ cleanup.yml)
5. **Go 환경 설정 반복**

## 최적화 전략

### 1. 공통 Action 추출

#### A. Go 환경 설정 Action
```yaml
# .github/actions/setup-go/action.yml
name: 'Setup Go Environment'
description: 'Sets up Go with caching'
inputs:
  go-version:
    description: 'Go version'
    required: false
    default: '1.24'
runs:
  using: 'composite'
  steps:
    - name: Setup Go
      uses: actions/setup-go@v5
      with:
        go-version: ${{ inputs.go-version }}
        cache: true
```

#### B. 보안 스캔 Action
```yaml
# .github/actions/security-scan/action.yml
name: 'Security Scanning'
description: 'Runs comprehensive security scans'
inputs:
  scan-type:
    description: 'Type of scan (code|dependencies|secrets|container)'
    required: false
    default: 'all'
runs:
  using: 'composite'
  steps:
    - name: Gosec
      if: contains(inputs.scan-type, 'code') || inputs.scan-type == 'all'
      # ... Gosec 설정
    - name: Dependency scan
      if: contains(inputs.scan-type, 'dependencies') || inputs.scan-type == 'all'
      # ... 의존성 스캔 설정
```

### 2. 워크플로우 재구성

#### 통합 후 구조
```
main-ci.yml          # 메인 CI (품질 검사, 테스트, 빌드)
├── 코드 품질 검사
├── 단위/통합 테스트  
├── 기본 보안 스캔
└── Docker 빌드 (태그 없이)

release.yml          # 릴리즈 전용
├── Docker 릴리즈 (태그 포함)
├── Helm 차트 릴리즈
├── 릴리즈 노트 생성
└── 프로덕션 배포

security-daily.yml   # 일일 보안 스캔
├── 전체 보안 검사
├── SBOM 생성
└── 보안 리포트

monitoring.yml       # 성능 모니터링 (기존 유지)

maintenance.yml      # 유지보수 (cleanup.yml 개선)
```

### 3. 단계별 마이그레이션

#### Phase 1: 공통 Action 생성
- [ ] Go 환경 설정 Action
- [ ] 보안 스캔 Action  
- [ ] Docker 빌드 Action

#### Phase 2: 메인 CI 통합
- [ ] ci-cd.yml에서 기본 보안 스캔만 유지
- [ ] 배포 로직을 release.yml로 이동
- [ ] 중복 Docker 빌드 제거

#### Phase 3: 전문화된 워크플로우
- [ ] security-scan.yml을 daily 스캔으로 전환
- [ ] release.yml 최적화
- [ ] monitoring.yml 유지

#### Phase 4: 정리 및 테스트
- [ ] 불필요한 중복 제거
- [ ] 전체 파이프라인 테스트
- [ ] 문서 업데이트

## 예상 효과

### 정량적 효과
- **라인 수 감소**: 1,976줄 → 약 1,200줄 (40% 감소)
- **중복 제거**: 5개 주요 중복 영역 해결
- **실행 시간 단축**: 불필요한 중복 작업 제거

### 정성적 효과
- **유지보수성 향상**: 공통 로직의 중앙화
- **일관성 확보**: 표준화된 설정 사용
- **가독성 개선**: 워크플로우 목적 명확화

## 구현 우선순위

### High Priority
1. 공통 Action 추출 (Go 환경, 보안 스캔)
2. ci-cd.yml 중복 제거
3. release.yml 최적화

### Medium Priority
1. security-scan.yml 전환
2. monitoring.yml 연동 개선

### Low Priority
1. cleanup.yml → maintenance.yml 개선
2. 추가 최적화 및 문서화

## 다음 단계

1. **공통 Action 생성**: .github/actions/ 디렉토리 구성
2. **점진적 마이그레이션**: 한 번에 하나씩 변경
3. **테스트 및 검증**: 각 단계별 기능 테스트
4. **문서 업데이트**: 변경사항 문서화
