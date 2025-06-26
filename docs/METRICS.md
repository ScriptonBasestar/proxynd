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

## 주요 메트릭

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

## 문제 해결

### 메트릭이 수집되지 않음
1. `/metrics` 엔드포인트 접근 가능 여부 확인
2. Prometheus 타겟 상태 확인
3. 방화벽/네트워크 설정 확인

### 메트릭 값이 이상함
1. 시계 동기화 확인
2. 카운터 리셋 여부 확인
3. 레이블 일관성 확인

### 높은 메모리 사용량
1. 메트릭 카디널리티 확인
2. 히스토그램 버킷 수 조정
3. 스크레이프 간격 증가