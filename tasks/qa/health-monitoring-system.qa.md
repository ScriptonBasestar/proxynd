# ✅ 헬스체크 및 모니터링 시스템 QA 시나리오

## related_tasks
- `/tasks/done/10-testing/04-health-monitoring.md`

## purpose
헬스체크 시스템, 메트릭 수집, 모니터링 대시보드, 알림 시스템이 운영 환경에서 정상 동작하는지 검증

## tags
[qa], [monitoring], [health-check], [metrics], [manual]

---

## 🧪 테스트 시나리오

### 1. 헬스체크 엔드포인트 검증
1. **기본 헬스체크**
   - `GET /healthz` 응답 시간 < 100ms 확인
   - 서비스 상태 정상 응답 확인
2. **상세 헬스체크**
   - `GET /health/detailed` 모든 컴포넌트 상태 확인
   - 프록시별, 캐시, 보안, 시스템 상태 세부 정보 조회
3. **개별 컴포넌트 헬스체크**
   - `/health/proxy/maven`, `/health/proxy/npm` 등 프록시별 상태
   - `/health/cache` 캐시 시스템 상태
   - `/health/security` 보안 시스템 상태
   - `/health/system` 시스템 리소스 상태

### 2. 메트릭 수집 시스템 검증
1. **사용자 활동 메트릭**
   - 활성 사용자 수 (`proxynd_active_users_total`)
   - 요청 수 (`proxynd_user_requests_total`)
   - 대역폭 사용량 (`proxynd_user_bandwidth_bytes_total`)
   - 세션 지속 시간 (`proxynd_user_session_duration_seconds`)
2. **비즈니스 메트릭**
   - 프록시별 요청 통계
   - 패키지 다운로드 통계
   - 지리적 분포 (`proxynd_user_geolocation_total`)
3. **성능 메트릭**
   - 응답 시간 히스토그램
   - 동시 연결 수
   - 에러율 (`proxynd_user_errors_total`)

### 3. Grafana 대시보드 검증
1. **사용자 활동 대시보드**
   - 프록시 타입별 요청 분포 (파이 차트)
   - 활성 사용자 및 일일 사용자 (타임시리즈)
   - 대역폭 사용량 (타임시리즈)
   - 지리적 분포 (파이 차트)
   - 상위 사용자 (테이블)
   - 세션 지속 시간 (히스토그램)
   - 에러율 (타임시리즈)
2. **대시보드 기능**
   - 실시간 데이터 업데이트 (30초 주기)
   - 시간 범위 선택 기능
   - 패널별 드릴다운 가능

### 4. Prometheus 알림 시스템 검증
1. **헬스체크 알림**
   - `ProxyNDHealthCheckFailing`: 컴포넌트 헬스체크 실패
   - `ProxyNDOverallHealthCritical`: 전체 헬스체크 위험
   - `ProxyNDMultipleHealthFailures`: 다중 헬스체크 실패
2. **사용자 활동 알림**
   - `ProxyNDHighUserErrorRate`: 높은 사용자 에러율
   - `ProxyNDUnusualUserActivity`: 비정상적 사용자 활동
   - `ProxyNDHighBandwidthUsage`: 높은 대역폭 사용량
3. **지리적 보안 알림**
   - `ProxyNDGeolocationAnomalies`: 지리적 이상 탐지

### 5. CI/CD 통합 테스트
1. **자동화된 헬스체크 테스트**
   - GitHub Actions의 `health-monitoring.yml` 워크플로우 실행
   - 모든 헬스체크 엔드포인트 자동 검증
   - 부하 테스트 시나리오 실행
2. **성능 기준선 테스트**
   - 헬스체크 응답 시간 임계값 검증
   - 성능 회귀 검출 자동화

---

## ✅ 기대 결과

- **헬스체크 응답 시간**: < 100ms
- **메트릭 수집 누락율**: < 1%
- **알림 정확도**: > 95%
- **대시보드 로딩 시간**: < 3초
- **실시간 데이터 업데이트**: 30초 이내
- **CI 헬스체크 자동화**: 모든 시나리오 통과

---

## 🔍 검증 포인트

1. **헬스체크 엔드포인트**
   - `/healthz`, `/health/detailed`, `/health/status` 등
   - 각 프록시별 헬스체크 엔드포인트
   - 응답 시간 및 상태 코드 확인

2. **메트릭 엔드포인트**
   - `/metrics` Prometheus 메트릭 수집 확인
   - 메트릭 레이블 및 값 정확성 검증

3. **모니터링 스택**
   - Grafana 대시보드 접근성 (`http://localhost:3000`)
   - Prometheus 메트릭 수집 (`http://localhost:9090`)
   - 알림 규칙 설정 및 동작

4. **자동화 테스트**
   - `.github/workflows/health-monitoring.yml` 실행 결과
   - CI에서의 성능 벤치마크 테스트 결과

---

## 🎯 수동 테스트 절차

1. **환경 설정**
   ```bash
   make dev-setup
   make dev-run
   cd monitoring && docker-compose up -d
   ```

2. **헬스체크 테스트**
   ```bash
   curl http://localhost:8081/healthz
   curl http://localhost:8081/health/detailed | jq
   ```

3. **메트릭 수집 확인**
   ```bash
   curl http://localhost:8081/metrics | grep "proxynd_"
   ```

4. **대시보드 접근**
   - Grafana: http://localhost:3000 (admin/admin)
   - Prometheus: http://localhost:9090