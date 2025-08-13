# E2E 테스트 개선 계획

## 📋 개요

실제 패키지 매니저 클라이언트를 사용한 엔드투엔드 테스트를 강화하여 실제 사용 시나리오를 검증합니다.

## 🎯 목표

- 실제 클라이언트 도구와의 호환성 검증
- 전체 워크플로우 테스트 (설치 → 사용 → 캐시)
- 다양한 환경에서의 동작 확인
- 사용자 관점에서의 기능 검증

## 🛠️ E2E 테스트 현황

### ✅ 기존 스크립트들
- `tests/e2e/scripts/test-npm.sh` ✅
- `tests/e2e/scripts/test-maven.sh` ✅  
- `tests/e2e/scripts/test-apt.sh` ✅
- `tests/e2e/scripts/test-docker.sh` ✅
- `tests/e2e/scripts/test-pip.sh` ✅
- `tests/e2e/scripts/test-yum.sh` ✅
- `tests/e2e/scripts/test-apk.sh` ✅

### 🔄 개선이 필요한 영역

## 📦 패키지별 E2E 테스트 강화

### NPM E2E 테스트
**현재 상태**: 기본 스크립트 존재
**개선 계획**:
```bash
# 추가 테스트 시나리오
- npm install express --registry=http://localhost:8080/proxy/npm
- npm install @types/node --registry=http://localhost:8080/proxy/npm
- npm view express --registry=http://localhost:8080/proxy/npm
- npm search express --registry=http://localhost:8080/proxy/npm
```

**검증 항목**:
- [ ] 패키지 설치 성공
- [ ] 스코프 패키지 지원
- [ ] 버전 범위 해석
- [ ] 의존성 해결
- [ ] 캐시 동작 확인

### Maven E2E 테스트  
**현재 상태**: 기본 스크립트 존재
**개선 계획**:
```bash
# 추가 테스트 시나리오
- mvn dependency:get -Dartifact=junit:junit:4.13.2
- mvn install -Dmaven.repo.local=/tmp/maven-test
- mvn dependency:tree (의존성 트리 확인)
- 브라우저 인터페이스 테스트
```

**검증 항목**:
- [ ] 아티팩트 다운로드
- [ ] SNAPSHOT 버전 처리
- [ ] 의존성 해결  
- [ ] 체크섬 검증
- [ ] 브라우저 인터페이스

### Docker E2E 테스트
**현재 상태**: 기본 스크립트 존재  
**개선 계획**:
```bash
# 추가 테스트 시나리오
- docker pull nginx (through proxy)
- docker pull hello-world
- docker manifest inspect nginx
- 다양한 아키텍처 이미지
```

**검증 항목**:
- [ ] 이미지 풀 성공
- [ ] 매니페스트 조회
- [ ] 레이어 캐싱
- [ ] 인증 처리

### APT E2E 테스트
**개선 계획**:
```bash
# 실제 apt 명령어 테스트
- apt update (sources.list 수정)
- apt install curl
- apt search nginx
- apt show nginx
```

**검증 항목**:
- [ ] 패키지 목록 업데이트
- [ ] 패키지 설치
- [ ] 검색 기능
- [ ] 의존성 해결

## 🏗️ E2E 테스트 인프라

### 테스트 환경 설정
```bash
# 컨테이너 기반 테스트 환경
docker-compose -f tests/e2e/docker-compose.yml up -d

# 서비스 구성:
# - proxynd: 메인 프록시 서버
# - nginx: 업스트림 모의 서버  
# - test-runner: 테스트 실행 환경
```

### 업스트림 모의 서버
**위치**: `tests/e2e/upstream-data/`
**구성**:
- 각 패키지 타입별 최소한의 테스트 데이터
- 메타데이터, 패키지 파일, 서명 파일
- 다양한 응답 시나리오 (200, 404, 500)

### 테스트 픽스처 관리
```bash
# 픽스처 업데이트 스크립트
tests/e2e/scripts/update-fixtures.sh

# 픽스처 검증 스크립트  
tests/e2e/scripts/validate-fixtures.sh
```

## 🧪 고급 E2E 테스트 시나리오

### 장애 복구 테스트
```bash
# 업스트림 서버 장애 시뮬레이션
- 네트워크 지연 주입
- 서버 오류 응답
- 부분적 파일 전송
- 연결 타임아웃
```

### 캐시 검증 테스트
```bash
# 캐시 동작 확인
- 첫 번째 요청: 업스트림에서 다운로드
- 두 번째 요청: 캐시에서 응답
- 캐시 만료 후: 재검증
- 캐시 무효화: 강제 갱신
```

### 성능 테스트
```bash
# 동시 요청 처리
- 동일 패키지 동시 요청
- 다른 패키지 병렬 요청
- 대용량 파일 다운로드
- 메모리 사용량 모니터링
```

### 보안 테스트
```bash
# 보안 시나리오
- 경로 순회 공격 방어
- 악성 패키지 차단
- 인증 우회 시도
- SSL/TLS 검증
```

## 📊 E2E 테스트 메트릭

### 기능 검증 메트릭
- 패키지 설치 성공률: 100%
- 캐시 적중률: > 70%
- 응답 시간: 업스트림 대비 < 110%

### 호환성 메트릭
- 클라이언트 버전 호환성
- 프로토콜 표준 준수
- 에러 메시지 일관성

### 안정성 메트릭
- 장시간 실행 안정성
- 메모리 누수 없음
- 동시 연결 처리

## 🚀 실행 계획

### Phase 1: 기존 스크립트 강화 (2-3일)
1. 각 스크립트에 추가 시나리오 구현
2. 검증 로직 강화
3. 로깅 및 리포팅 개선

### Phase 2: 고급 시나리오 구현 (3-4일)  
1. 장애 복구 테스트 추가
2. 성능 테스트 구현
3. 보안 테스트 추가

### Phase 3: CI 통합 (1-2일)
1. GitHub Actions 매트릭스 확장
2. 테스트 결과 아티팩트 수집
3. 실패 시 자동 디버깅 정보 수집

### Phase 4: 문서화 (1일)
1. E2E 테스트 실행 가이드
2. 새로운 테스트 추가 방법
3. 트러블슈팅 가이드

## 📋 체크리스트

### 스크립트 개선
- [ ] NPM: 스코프 패키지 및 검색 테스트 추가
- [ ] Maven: SNAPSHOT 및 브라우저 인터페이스 테스트
- [ ] Docker: 다중 아키텍처 이미지 테스트
- [ ] APT: 실제 apt 명령어 테스트  
- [ ] YUM: 실제 yum/dnf 명령어 테스트
- [ ] APK: 실제 apk 명령어 테스트
- [ ] PIP: 실제 pip 명령어 테스트

### 인프라 개선
- [ ] Docker Compose 환경 구축
- [ ] 업스트림 모의 서버 개선
- [ ] 테스트 픽스처 자동 업데이트
- [ ] 테스트 환경 격리

### 고급 시나리오
- [ ] 장애 복구 테스트 구현
- [ ] 성능 벤치마크 추가
- [ ] 보안 테스트 추가
- [ ] 호환성 테스트 확장

### CI/CD 통합  
- [ ] GitHub Actions 매트릭스 확장
- [ ] 테스트 결과 리포팅 개선
- [ ] 아티팩트 수집 자동화
- [ ] 실패 시 디버깅 정보 수집

## 🔗 관련 파일

- `tests/e2e/scripts/*.sh` - E2E 테스트 스크립트들
- `tests/e2e/docker-compose.yml` - 테스트 환경 설정  
- `tests/e2e/upstream-data/` - 테스트 픽스처
- `tests/e2e/Makefile` - E2E 테스트 실행 도구
- `.github/workflows/ci.yml` - CI 설정