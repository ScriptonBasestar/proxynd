# ProxyND 메트릭 가이드

ProxyND는 Prometheus 형식의 메트릭을 제공하여 서비스 상태와 성능을 모니터링할 수 있습니다.

## 메트릭 엔드포인트

### 기본 엔드포인트
- **URL**: `/metrics`
- **형식**: Prometheus text format
- **인증**: 선택적 Basic Auth

### 추가 엔드포인트
- `/api/metrics/cache` - 캐시 통계 (JSON)
- `/api/metrics/proxy` - 프록시 통계 (JSON)
- `/api/metrics/system` - 시스템 메트릭 (JSON)
- `/api/metrics/health` - 건강 상태 (Prometheus format)

### 강화된 메트릭 엔드포인트
- `/api/metrics/business` - 비즈니스 메트릭 (패키지 인기도, 사용자 에이전트 분석)
- `/api/metrics/performance` - 성능 메트릭 (백분위수 지연시간, 처리량)
- `/api/metrics/errors` - 에러 메트릭 (상태 코드 분포, 재시도 추적)
- `/api/metrics/users` - 사용자 활동 메트릭 (지리적 분포, 행동 패턴)
- `/api/metrics/snapshot` - 전체 메트릭 스냅샷
- `/api/metrics/trends` - 시계열 트렌드 분석
- `/api/metrics/alerts` - 임계치 기반 알림 조건 체크

#### 쿼리 파라미터 지원
- `?details=true` - 상세 정보 포함
- `?registry=npm` - 특정 레지스트리 필터링
- `?range=24h` - 시간 범위 지정 (1h, 24h, 7d)
- `?limit=50` - 결과 수 제한

#### 비즈니스 메트릭 API 응답 예시

**GET /api/metrics/business?details=true&registry=npm**
```json
{
  "timestamp": "2024-01-20T10:30:00Z",
  "registry_filter": "npm",
  "popular_packages": [
    {
      "name": "express",
      "downloads": 12543,
      "rank": 1,
      "growth_rate": 5.2
    },
    {
      "name": "react",
      "downloads": 11892,
      "rank": 2,
      "growth_rate": 3.1
    }
  ],
  "user_agents": {
    "npm": {"count": 45230, "percentage": 67.2},
    "yarn": {"count": 18120, "percentage": 26.9},
    "pnpm": {"count": 3960, "percentage": 5.9}
  },
  "cache_efficiency": {
    "hit_rate": 0.847,
    "miss_rate": 0.153,
    "bandwidth_saved_mb": 15420
  }
}
```

#### 성능 메트릭 API 응답 예시

**GET /api/metrics/performance?range=1h**
```json
{
  "timestamp": "2024-01-20T10:30:00Z",
  "range": "1h",
  "latency_percentiles": {
    "p50": 45.3,
    "p90": 156.7,
    "p95": 234.1,
    "p99": 892.4
  },
  "throughput": {
    "rps_1m": 127.3,
    "rps_5m": 134.7,
    "rps_15m": 142.1
  },
  "concurrent_connections": 89,
  "resource_utilization": {
    "cpu_percent": 15.2,
    "memory_percent": 45.8,
    "network_mbps": 23.4
  }
}
```

## 주요 메트릭

### 비즈니스 메트릭

#### 패키지 다운로드/업로드 통계
- **패키지별 다운로드 수**: 레지스트리별 인기 패키지 순위
- **버전별 다운로드 분포**: 특정 패키지의 버전별 사용 현황
- **파일 타입별 요청**: .tgz, .whl, .jar 등 파일 확장자별 분석
- **사용자 에이전트 분석**: npm, yarn, pip, mvn 등 클라이언트 도구별 요청 분포

#### 캐시 효율성 메트릭
- **프록시 타입별 캐시 히트율**: 각 레지스트리의 캐시 성능
- **인기 패키지 캐시 성능**: 상위 100개 패키지의 캐시 효율성
- **캐시 용량 분석**: 디스크 사용량, 파일 개수 모니터링

#### 지리적 사용 분포
- **국가별 요청 분포**: IP 기반 지리적 사용 패턴 (개인정보 보호 고려)
- **시간대별 사용량**: 글로벌 사용 패턴 분석
- **지역별 인기 패키지**: 지역별로 다른 패키지 선호도

### 성능 메트릭

#### 응답시간 백분위수
- **P50, P90, P95, P99**: 지연시간 백분위수 실시간 계산
- **레지스트리별 성능**: 각 프록시 타입의 성능 특성
- **시간대별 성능 변화**: 부하에 따른 성능 변화 추적

#### 처리량 메트릭
- **RPS (Requests Per Second)**: 1분, 5분, 15분 단위 처리량
- **동시 연결 수**: 실시간 동시 연결 추적
- **네트워크 처리량**: 업로드/다운로드 대역폭 사용량

#### 리소스 활용도
- **CPU 사용률**: 고루틴 대 CPU 비율 기반 추정
- **메모리 사용률**: GC 메트릭 포함 메모리 분석
- **네트워크 사용률**: 실시간 네트워크 I/O 모니터링

### 에러 메트릭

#### HTTP 상태 코드 분포
- **2xx 성공률**: 정상 요청 처리율
- **4xx 클라이언트 에러**: 잘못된 요청, 인증 실패 등
- **5xx 서버 에러**: 내부 서버 오류, 업스트림 장애 등
- **상세 에러 분류**: 404, 502, 503 등 구체적 에러 분석

#### 재시도 및 타임아웃 추적
- **재시도 횟수**: 업스트림 요청 재시도 통계
- **재시도 성공률**: 재시도를 통한 복구율
- **타임아웃 유형**: 연결, 읽기, 쓰기 타임아웃 분류
- **업스트림 실패율**: 각 업스트림 서버별 실패율

### 사용자 활동 메트릭

#### 고유 사용자 추적
- **1시간/24시간/7일**: 시간 윈도우별 고유 사용자 수
- **익명화된 사용자 ID**: 해시 기반 개인정보 보호
- **세션 지속시간**: 사용자 세션 패턴 분석

#### 사용자 행동 분석
- **요청 패턴**: 시간대별, 요일별 사용 패턴
- **인증 이벤트**: 로그인 성공/실패, 인증 방식별 통계
- **봇 대 인간**: 자동화 도구와 일반 사용자 구분

### HTTP 요청 메트릭

#### proxynd_http_requests_total
- **타입**: Counter
- **설명**: HTTP 요청 총 개수
- **레이블**:
  - `method`: HTTP 메서드 (GET, POST, PUT, DELETE)
  - `path`: 정규화된 경로
  - `status`: HTTP 상태 코드
  - `registry_type`: 레지스트리 타입 (npm, pypi, apt, docker, maven)

#### proxynd_http_request_duration_seconds
- **타입**: Histogram
- **설명**: HTTP 요청 처리 시간 (초)
- **레이블**: `method`, `path`, `status`, `registry_type`

#### proxynd_http_request_size_bytes
- **타입**: Histogram
- **설명**: HTTP 요청 크기 (바이트)
- **레이블**: `method`, `path`, `registry_type`

#### proxynd_http_response_size_bytes
- **타입**: Histogram
- **설명**: HTTP 응답 크기 (바이트)
- **레이블**: `method`, `path`, `status`, `registry_type`

#### proxynd_http_active_requests
- **타입**: Gauge
- **설명**: 현재 처리 중인 HTTP 요청 수

### 캐시 메트릭

#### proxynd_cache_hits_total
- **타입**: Counter
- **설명**: 캐시 히트 총 개수
- **레이블**:
  - `registry_type`: 레지스트리 타입
  - `cache_backend`: 캐시 백엔드 (file, s3, redis)

#### proxynd_cache_misses_total
- **타입**: Counter
- **설명**: 캐시 미스 총 개수
- **레이블**: `registry_type`, `cache_backend`

#### proxynd_cache_evictions_total
- **타입**: Counter
- **설명**: 캐시 제거 총 개수
- **레이블**:
  - `registry_type`: 레지스트리 타입
  - `cache_backend`: 캐시 백엔드
  - `reason`: 제거 이유 (ttl, size, manual)

#### proxynd_cache_size_bytes
- **타입**: Gauge
- **설명**: 현재 캐시 크기 (바이트)
- **레이블**: `registry_type`, `cache_backend`

#### proxynd_cache_items_count
- **타입**: Gauge
- **설명**: 캐시 아이템 개수
- **레이블**: `registry_type`, `cache_backend`

#### proxynd_cache_bandwidth_saved_bytes
- **타입**: Counter
- **설명**: 캐시로 절약된 대역폭 (바이트)
- **레이블**: `registry_type`

### 프록시 메트릭

#### proxynd_proxy_requests_total
- **타입**: Counter
- **설명**: 프록시 요청 총 개수
- **레이블**:
  - `registry_type`: 레지스트리 타입
  - `upstream`: 업스트림 서버
  - `method`: HTTP 메서드

#### proxynd_proxy_errors_total
- **타입**: Counter
- **설명**: 프록시 오류 총 개수
- **레이블**:
  - `registry_type`: 레지스트리 타입
  - `upstream`: 업스트림 서버
  - `error_type`: 오류 타입 (timeout, bad_gateway, etc.)

#### proxynd_proxy_upstream_duration_seconds
- **타입**: Histogram
- **설명**: 업스트림 요청 처리 시간 (초)
- **레이블**: `registry_type`, `upstream`

#### proxynd_proxy_bytes_transferred_total
- **타입**: Counter
- **설명**: 프록시를 통해 전송된 바이트 수
- **레이블**:
  - `registry_type`: 레지스트리 타입
  - `direction`: 방향 (upload, download)

### 인증 메트릭

#### proxynd_auth_attempts_total
- **타입**: Counter
- **설명**: 인증 시도 총 개수
- **레이블**:
  - `method`: 인증 방법 (basic, token, ldap)
  - `result`: 결과 (success, failure)

#### proxynd_auth_failures_total
- **타입**: Counter
- **설명**: 인증 실패 총 개수
- **레이블**:
  - `method`: 인증 방법
  - `reason`: 실패 이유

#### proxynd_active_sessions
- **타입**: Gauge
- **설명**: 현재 활성 세션 수

### 패키지 검증 메트릭

#### proxynd_package_verifications_total
- **타입**: Counter
- **설명**: 패키지 검증 총 개수
- **레이블**:
  - `registry_type`: 레지스트리 타입
  - `result`: 결과 (success, failure)

#### proxynd_verification_failures_total
- **타입**: Counter
- **설명**: 검증 실패 총 개수
- **레이블**:
  - `registry_type`: 레지스트리 타입
  - `failure_type`: 실패 타입 (hash_mismatch, missing_hash, etc.)

#### proxynd_verification_duration_seconds
- **타입**: Histogram
- **설명**: 패키지 검증 소요 시간 (초)
- **레이블**: `registry_type`

### 시스템 메트릭

#### proxynd_config_reloads_total
- **타입**: Counter
- **설명**: 설정 리로드 총 횟수

#### proxynd_config_reload_failures_total
- **타입**: Counter
- **설명**: 설정 리로드 실패 총 횟수

#### proxynd_uptime_seconds
- **타입**: Counter
- **설명**: 서비스 가동 시간 (초)

#### proxynd_go_goroutines
- **타입**: Gauge
- **설명**: 현재 고루틴 수

#### proxynd_go_memory_alloc_bytes
- **타입**: Gauge
- **설명**: 할당된 메모리 (바이트)

#### proxynd_go_gc_pause_total_seconds
- **타입**: Counter
- **설명**: GC 일시정지 총 시간 (초)

## Prometheus 설정 예시

```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'proxynd'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: '/metrics'
    scrape_interval: 15s

    # Basic Auth 사용 시
    basic_auth:
      username: 'metrics'
      password: 'prometheus'
```

## Grafana 대시보드

ProxyND 메트릭을 시각화하기 위한 Grafana 대시보드 예시:

### 요청 처리량
```promql
# 초당 요청 수
rate(proxynd_http_requests_total[5m])

# 레지스트리별 요청 수
sum by (registry_type) (rate(proxynd_http_requests_total[5m]))
```

### 캐시 효율성
```promql
# 캐시 히트율
sum(rate(proxynd_cache_hits_total[5m])) /
(sum(rate(proxynd_cache_hits_total[5m])) + sum(rate(proxynd_cache_misses_total[5m])))

# 레지스트리별 캐시 히트율
sum by (registry_type) (rate(proxynd_cache_hits_total[5m])) /
sum by (registry_type) (rate(proxynd_cache_hits_total[5m]) + rate(proxynd_cache_misses_total[5m]))
```

### 대역폭 절약
```promql
# 시간당 절약된 대역폭
rate(proxynd_cache_bandwidth_saved_bytes[1h])

# 총 절약된 대역폭
proxynd_cache_bandwidth_saved_bytes
```

### 오류율
```promql
# 오류율 (5xx 응답)
sum(rate(proxynd_http_requests_total{status=~"5.."}[5m])) /
sum(rate(proxynd_http_requests_total[5m]))

# 프록시 오류율
sum(rate(proxynd_proxy_errors_total[5m])) /
sum(rate(proxynd_proxy_requests_total[5m]))
```

### 응답 시간
```promql
# 평균 응답 시간
histogram_quantile(0.5, rate(proxynd_http_request_duration_seconds_bucket[5m]))

# 95 백분위수 응답 시간
histogram_quantile(0.95, rate(proxynd_http_request_duration_seconds_bucket[5m]))

# 레지스트리별 평균 응답 시간
histogram_quantile(0.5, sum by (registry_type, le) (rate(proxynd_http_request_duration_seconds_bucket[5m])))
```

## 알림 규칙 예시

```yaml
# alerts.yml
groups:
  - name: proxynd
    rules:
      # 높은 오류율
      - alert: HighErrorRate
        expr: |
          sum(rate(proxynd_http_requests_total{status=~"5.."}[5m])) /
          sum(rate(proxynd_http_requests_total[5m])) > 0.05
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High error rate detected"
          description: "Error rate is {{ $value | humanizePercentage }}"

      # 낮은 캐시 히트율
      - alert: LowCacheHitRate
        expr: |
          sum(rate(proxynd_cache_hits_total[5m])) /
          (sum(rate(proxynd_cache_hits_total[5m])) + sum(rate(proxynd_cache_misses_total[5m]))) < 0.5
        for: 10m
        labels:
          severity: info
        annotations:
          summary: "Low cache hit rate"
          description: "Cache hit rate is {{ $value | humanizePercentage }}"

      # 높은 메모리 사용량
      - alert: HighMemoryUsage
        expr: proxynd_go_memory_alloc_bytes > 1e9  # 1GB
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High memory usage"
          description: "Memory usage is {{ $value | humanize1024 }}B"

      # 검증 실패율
      - alert: HighVerificationFailureRate
        expr: |
          sum(rate(proxynd_package_verifications_total{result="failure"}[5m])) /
          sum(rate(proxynd_package_verifications_total[5m])) > 0.1
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "High package verification failure rate"
          description: "Verification failure rate is {{ $value | humanizePercentage }}"
```

## 성능 튜닝 가이드

### 메트릭 카디널리티 관리
- 경로 정규화로 카디널리티 감소
- 불필요한 레이블 제거
- 히스토그램 버킷 최적화

### 스크레이프 간격 조정
- 기본: 15초
- 상세 모니터링: 5초
- 리소스 절약: 30초

### 메트릭 보관 정책
- 원시 데이터: 15일
- 5분 평균: 63일
- 1시간 평균: 1년

## 자동 알림 조건 (내장)

강화된 메트릭 시스템은 다음 조건들을 자동으로 감지합니다:

### 성능 알림
- **높은 지연시간**: P99 > 5초
- **높은 에러율**: 에러율 > 5%
- **업스트림 실패**: 실패율 > 10%
- **연결 문제**: 연결 에러 급증
- **캐시 효율성**: 히트율 < 80%

### 리소스 알림
- **메모리 사용량**: 사용률 > 90%
- **디스크 공간**: 여유 공간 < 10%
- **CPU 사용량**: 지속적 사용률 > 80%

### 비즈니스 알림
- **인기 패키지 캐시 미스**: 상위 10개 패키지의 캐시 미스율 > 20%
- **사용자 에이전트 이상**: 특정 클라이언트의 급격한 증가/감소
- **지역별 트래픽 급변**: 평소 대비 200% 이상 증가

## 메트릭 시스템 아키텍처

### Observer + Collector 패턴
- **Thread-Safe 컬렉터**: 각 메트릭 유형별 독립적인 컬렉터
- **실시간 집계**: 백분위수 및 추세 분석 실시간 계산
- **미들웨어 통합**: 기존 미들웨어와 seamless 통합
- **메모리 효율성**: 순환 버퍼 및 TTL 기반 데이터 관리

### 데이터 수집 전략
```
Request → Middleware → Enhanced Collector → Multiple Trackers
                                          ├── PopularityTracker
                                          ├── LatencyTracker  
                                          ├── UserAgentTracker
                                          ├── GeographicTracker
                                          ├── ErrorTracker
                                          └── ... (9 more trackers)
```

### 메모리 관리
- **설정 가능한 보존 기간**: 기본 24시간, 설정 가능
- **자동 정리**: 시간 기반 오래된 데이터 자동 정리
- **버퍼 제한**: 각 컬렉터별 최대 데이터 포인트 제한
- **효율적 집계**: 지연 로딩 및 on-demand 계산

## 성능 영향 및 최적화

### 메모리 사용량
- **기본 설정**: ~50MB RAM (일반적인 워크로드)
- **대용량 환경**: ~200MB RAM (높은 트래픽)
- **순환 버퍼**: 고정 크기, 메모리 증가 제한

### 처리 오버헤드
- **요청 지연시간**: +0.1~0.5ms (무시할 수 있는 수준)
- **백그라운드 처리**: 30초 간격 비동기 집계
- **정리 작업**: 1시간 간격 자동 정리

### 스토리지 효율성
- **인메모리 전용**: 별도 데이터베이스 불필요
- **압축된 메트릭**: 효율적인 데이터 구조
- **설정 가능한 TTL**: 필요에 따라 조정 가능

### 레지스트리별 패키지 파싱
- **NPM**: `@scope/package/-/package-version.tgz` 패턴 지원
- **Maven**: GroupId/ArtifactId/Version 구조 파싱  
- **PyPI**: Wheel (.whl) 및 소스 배포판 (.tar.gz) 구분
- **Docker**: Manifest/Blob 요청 타입 분류
- **APT/YUM/APK**: 패키지명-버전-아키텍처 파싱

## 보안 및 개인정보 보호

### 데이터 보호
- **IP 주소 처리**: 지리적 통계는 선택적
- **사용자 ID 익명화**: 해싱 기반 익명 ID
- **데이터 보존 제한**: 설정 가능한 보존 기간
- **PII 저장 금지**: 개인식별정보 저장 안함

### 접근 제어
- **Basic Auth 지원**: 메트릭 엔드포인트 보호
- **API 키 지원**: 향후 API 키 인증 지원 가능
- **역할 기반 접근**: 엔드포인트별 접근 제어 가능

## 문제 해결

### 메트릭이 수집되지 않음
1. `/metrics` 엔드포인트 접근 가능 여부 확인
2. Prometheus 타겟 상태 확인
3. 방화벽/네트워크 설정 확인
4. Enhanced Collector 초기화 상태 확인

### 메트릭 값이 이상함
1. 시계 동기화 확인
2. 카운터 리셋 여부 확인
3. 레이블 일관성 확인
4. 메트릭 컬렉터 오류 로그 확인

### 높은 메모리 사용량
1. 메트릭 카디널리티 확인
2. 히스토그램 버킷 수 조정
3. 스크레이프 간격 증가
4. Enhanced Collector 보존 기간 단축

### Enhanced Metrics 특정 문제
1. **패키지 파싱 오류**: 레지스트리별 경로 패턴 확인
2. **사용자 에이전트 분류**: Unknown 카테고리 비율 확인
3. **지리적 위치 오류**: IP 지역 데이터베이스 업데이트
4. **백분위수 계산 이상**: 샘플 데이터 충분성 확인
