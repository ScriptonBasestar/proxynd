# TODO/FIXME 분석 보고서

## 개요
코드베이스에서 총 44개의 TODO/FIXME 마커가 22개 파일에서 발견되었습니다.

## 카테고리별 분류

### 🔴 긴급 (보안/인증 관련)
1. **middlewares/proxy_policy.go:73**
   - `// TODO: 실제 인증 로직 구현`
   - 보안에 직접적인 영향

### 🟠 중요 (핵심 기능 미구현)

#### 메트릭 및 모니터링
1. **routers/metrics_router.go** (5개)
   - 사용자 정보 로드
   - 메트릭 값 추출
   - Prometheus 메트릭 연동
   - 건강 상태 확인 로직

#### 핸들러 구현
1. **internal/app/providers.go** (3개)
   - Docker, PIP, YUM, APK 핸들러 등록
   - 라우터 구현

2. **internal/app/container.go** (2개)
   - 핸들러 등록 관련

3. **internal/services/proxy/** (3개)
   - Docker, NPM, APT 서비스 로직 구현

#### Maven 브라우저
1. **handlers/proxy/maven_browser_handler.go** (3개)
   - searchInTree 함수 구현
   - buildGAVTree 함수 구현
   - NewFileIndexStorage 함수 구현

### 🟡 중간 (기능 개선)

#### 캐시 관련
1. **cache/eviction.go:153**
   - 디렉토리 스캔으로 키 목록 가져오기

2. **internal/plugins/adapters/npm_adapter.go:282**
   - 캐시 조회 로직 구현

3. **internal/services/docker/cache_manager.go:414**
   - 더 정교한 제거 정책 구현

#### 웹훅 및 알림
1. **internal/webhook/worker.go:157**
   - Dead Letter Queue 구현

2. **internal/webhook/sender.go:124**
   - WebhookHistoryManager 반환 수정

3. **internal/webhook/test_handler.go** (2개)
   - HTTP HEAD/GET 요청 구현
   - 테스트 이력 저장/조회

#### 설정 관리
1. **configs/hot_reload.go** (7개)
   - 로거 레벨/포맷 변경
   - 캐시 TTL 업데이트
   - 메트릭 서버 시작/중지
   - 사용자 재로드
   - IP 화이트리스트 업데이트

### 🟢 낮음 (문서화/정리)

1. **internal/context/timeout.go** (3개)
   - TODO 컨텍스트 관련 (표준 Go 패턴)

2. **configs/config_loader.go:349**
   - 파일 시스템 감시 구현

3. **logging/config.go:133**
   - 통합 설정 구조체에서 로그 설정 추출

## 권장 조치

### 즉시 처리 필요
1. 인증 로직 구현 (보안 관련)
2. 메트릭 시스템 완성
3. 핵심 프록시 서비스 로직 구현

### 단계별 처리
1. Maven 브라우저 관련 함수 구현
2. 캐시 시스템 개선
3. 웹훅 시스템 완성
4. 설정 핫 리로드 기능 완성

### 코드 정리
1. 구현 완료된 TODO 제거
2. 불필요한 TODO 컨텍스트 함수 검토

## 이슈 생성 템플릿

```markdown
### 🔴 긴급: 프록시 인증 로직 구현
**파일**: middlewares/proxy_policy.go:73
**설명**: 실제 인증 로직이 구현되지 않아 보안 위험
**작업**: JWT, OAuth2 등 인증 메커니즘 구현

### 🟠 중요: 메트릭 시스템 완성
**파일**: routers/metrics_router.go
**설명**: Prometheus 메트릭 연동 및 실제 값 추출 로직 미구현
**작업**:
- 사용자 정보 로드
- 메트릭 값 추출
- Prometheus 연동
- 건강 상태 확인

### 🟠 중요: Docker/NPM/APT 프록시 서비스 구현
**파일**: internal/services/proxy/
**설명**: 각 패키지 매니저별 프록시 로직 미구현
**작업**: 각 서비스별 핵심 로직 구현
```
