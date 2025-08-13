---
priority: high
severity: medium
file_type: testing
source: tasks/plan/10-testing/integration-tests.md
---

# 통합 테스트 개선

## 목표
각 패키지 매니저별 통합 테스트를 강화하여 실제 프록시 워크플로우를 철저히 검증합니다.

## 작업 항목

### Step 1: Maven 통합 테스트 강화
- [x] SNAPSHOT 버전 처리 테스트 추가
- [x] 체크섬 엔드포인트 검증 테스트
- [x] 디렉토리 브라우징 토글 테스트
- [x] 메타데이터 병합 테스트

### Step 2: YUM 통합 테스트 개선
- [x] repodata 신선도 검사 테스트
- [x] gzip/pgp 헤더 처리 테스트
- [x] Range 요청 지원 테스트
- [x] 미러 간 페일오버 테스트

### Step 3: APK 통합 테스트 개선
- [x] 서명 검증 실패 경로 테스트
- [x] 미러 자동 전환 테스트
- [x] APKINDEX 압축 처리 테스트
- [x] APK 패키지 다운로드 검증

### Step 4: 공통 테스트 패턴 구현
- [x] 캐시 시나리오 검증 패턴 구현
- [x] 에러 처리 및 복구 패턴 테스트
- [x] 성능 검증 패턴 구현
- [x] CI 매트릭스 최적화 및 통합

## 검증 기준
- 응답 시간: 캐시 HIT < 10ms, MISS < 5초
- 캐시 적중률: > 70%
- 동시 요청 처리: 최소 100개

## 관련 파일
- `tests/integration/*.go`
- `.github/workflows/ci.yml`
- `Makefile.test.mk`
