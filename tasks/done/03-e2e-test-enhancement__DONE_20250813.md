---
priority: high
severity: medium
file_type: testing
source: tasks/plan/10-testing/e2e-tests.md
---

# E2E 테스트 강화

## 목표
실제 패키지 매니저 클라이언트와의 호환성을 검증하고 전체 워크플로우를 테스트합니다.

## 작업 항목

### Step 1: NPM E2E 테스트 강화
- [ ] 스코프 패키지 설치 테스트 추가
- [ ] npm search 및 view 명령어 테스트
- [ ] 의존성 해결 검증 테스트
- [ ] 캐시 동작 확인 테스트

### Step 2: Maven E2E 테스트 강화
- [ ] SNAPSHOT 버전 처리 테스트 추가
- [ ] 의존성 트리 확인 테스트
- [ ] 브라우저 인터페이스 테스트
- [ ] 체크섬 검증 테스트

### Step 3: Docker E2E 테스트 강화
- [ ] 다양한 아키텍처 이미지 테스트
- [ ] 매니페스트 조회 테스트
- [ ] 레이어 캐싱 검증
- [ ] 인증 처리 테스트

### Step 4: 테스트 인프라 개선
- [ ] Docker Compose 테스트 환경 구축
- [ ] 업스트림 모의 서버 개선
- [ ] 테스트 픽스처 자동 업데이트 스크립트
- [ ] CI 통합 및 매트릭스 확장

## 검증 기준
- 패키지 설치 성공률: 100%
- 캐시 적중률: > 70%
- 응답 시간: 업스트림 대비 < 110%

## 관련 파일
- `tests/e2e/scripts/*.sh`
- `tests/e2e/docker-compose.yml`
- `tests/e2e/upstream-data/`