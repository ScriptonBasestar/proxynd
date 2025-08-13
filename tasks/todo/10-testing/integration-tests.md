# 통합 테스트 개선 계획

## 📋 개요

각 패키지 매니저별 통합 테스트를 강화하여 실제 프록시 워크플로우를 철저히 검증합니다.

## 🎯 목표

- 모든 프록시 타입별 통합 테스트 완성
- 캐시 hit/miss 시나리오 검증
- 에러 처리 및 복구 메커니즘 테스트
- 성능 및 안정성 검증

## 📦 패키지별 통합 테스트 작업

### ✅ NPM 통합 테스트
**상태**: 완료 (404 JSON 응답 오류 수정됨)
- 파일: `tests/integration/npm_integration_test.go`
- 커버리지: 패키지 메타데이터, 타르볼 다운로드, 스코프 패키지, 인증 폴백

### 🔄 Maven 통합 테스트
**상태**: 개선 필요
- 파일: `tests/integration/maven_integration_test.go`
- **추가 필요 케이스**:
  - SNAPSHOT 버전 처리
  - 체크섬 엔드포인트 검증
  - 디렉토리 브라우징 토글
  - 메타데이터 병합 테스트

**구현 계획**:
```go
func TestMavenProxy_SnapshotHandling(t *testing.T) {
    // SNAPSHOT 버전의 메타데이터 및 아티팩트 다운로드 테스트
}

func TestMavenProxy_ChecksumValidation(t *testing.T) {
    // .sha1, .md5 파일 검증 테스트
}
```

### 🔄 YUM 통합 테스트  
**상태**: 개선 필요
- 파일: `tests/integration/yum_integration_test.go`
- **추가 필요 케이스**:
  - repodata 신선도 검사
  - gzip/pgp 헤더 처리
  - Range 요청 지원
  - 미러 간 페일오버

### 🔄 APK 통합 테스트
**상태**: 개선 필요  
- 파일: `tests/integration/apk_integration_test.go`
- **추가 필요 케이스**:
  - 서명 검증 실패 경로
  - 미러 자동 전환
  - APKINDEX 압축 처리

### ⏳ PIP 통합 테스트
**상태**: 검토 필요
- 파일: `tests/integration/pip_integration_test.go`
- **확인 필요**:
  - Simple API 호환성
  - 휠 파일 처리
  - 버전 범위 쿼리

### ⏳ Docker 통합 테스트
**상태**: 검토 필요
- 파일: `tests/integration/docker_integration_test.go`
- **확인 필요**:
  - 매니페스트 처리
  - 레이어 다운로드
  - 인증 토큰 갱신

## 🧪 공통 테스트 패턴

### 캐시 검증 패턴
```go
func TestCacheScenarios(t *testing.T) {
    // 1. Cache MISS → 업스트림 요청
    // 2. Cache HIT → 로컬 응답
    // 3. Cache 만료 → 재검증
    // 4. Cache 무효화 → 강제 갱신
}
```

### 에러 처리 패턴
```go
func TestErrorRecovery(t *testing.T) {
    // 1. 업스트림 타임아웃
    // 2. 네트워크 오류
    // 3. 인증 실패
    // 4. 손상된 캐시 데이터
}
```

### 성능 검증 패턴
```go
func TestPerformanceMetrics(t *testing.T) {
    // 1. 응답 시간 측정
    // 2. 동시 요청 처리
    // 3. 메모리 사용량
    // 4. 캐시 효율성
}
```

## 📊 측정 지표

### 응답 시간
- 캐시 HIT: < 10ms
- 캐시 MISS: < 5초 (업스트림 의존)
- 대용량 파일: 다운로드 진행률 추적

### 캐시 효율성
- 캐시 적중률: > 70%
- 캐시 크기 관리: LRU 정책 동작 확인
- TTL 준수: 만료 시간 정확성

### 안정성
- 동시 요청 처리: 최소 100개
- 메모리 누수 없음
- 고루틴 누수 없음

## 🚀 실행 계획

### Phase 1: 기존 테스트 강화 (2-3일)
1. Maven SNAPSHOT 및 체크섬 테스트 추가
2. YUM repodata 및 Range 요청 테스트 추가  
3. APK 서명 검증 테스트 추가

### Phase 2: 새로운 테스트 케이스 (2-3일)
1. 동시성 테스트 구현
2. 성능 벤치마크 추가
3. 장애 복구 시나리오 테스트

### Phase 3: CI 통합 (1일)
1. 매트릭스 빌드 최적화
2. 테스트 결과 리포팅 개선
3. 커버리지 임계값 설정

## 📋 체크리스트

- [ ] Maven: SNAPSHOT 처리 테스트
- [ ] Maven: 체크섬 검증 테스트  
- [ ] YUM: repodata 신선도 테스트
- [ ] YUM: Range 요청 테스트
- [ ] APK: 서명 검증 실패 테스트
- [ ] APK: 미러 전환 테스트
- [ ] 모든 타입: 동시성 테스트
- [ ] 모든 타입: 성능 벤치마크
- [ ] CI: 매트릭스 최적화
- [ ] 문서: 테스트 실행 가이드 업데이트

## 🔗 관련 파일

- `tests/integration/*.go` - 통합 테스트 파일들
- `.github/workflows/ci.yml` - CI 설정
- `Makefile.test.mk` - 테스트 실행 스크립트
- `scripts/test-coverage.sh` - 커버리지 분석