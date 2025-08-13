# 헬스체크 및 관측성 개선 계획

## 📋 개요

ProxyND의 헬스체크 시스템과 관측성을 강화하여 운영 효율성과 신뢰성을 높입니다.

## 🎯 목표

- 포괄적인 헬스체크 시스템 구축
- 실시간 모니터링 및 알림 체계 확립
- 성능 메트릭 수집 및 분석
- 장애 예측 및 자동 복구

## 🏥 헬스체크 시스템 강화

### 현재 헬스체크 현황
**파일**: `routers/health_router.go`

**기존 엔드포인트**:
- `/healthz` - 전체 시스템 상태
- `/health/live` - 라이브니스 프로브  
- `/health/ready` - 레디니스 프로브
- `/health/check/:name` - 개별 체크
- `/health/debug` - 디버그 정보
- `/health/adapters` - 어댑터 상태

### 추가 필요한 헬스체크

#### 1. 프록시별 업스트림 연결성 체크
```go
// NPM 레지스트리 연결 체크
func TestNPMUpstreamHealth(t *testing.T) {
    endpoint := "/health/upstream/npm"
    // registry.npmjs.org 연결성 확인
    // 응답 시간 측정
    // 캐시 상태 확인
}

// Maven Central 연결 체크  
func TestMavenUpstreamHealth(t *testing.T) {
    endpoint := "/health/upstream/maven"
    // repo1.maven.org 연결성 확인
    // 메타데이터 접근 확인
}
```

#### 2. 캐시 시스템 헬스체크
```go
func TestCacheSystemHealth(t *testing.T) {
    endpoint := "/health/cache"
    // 캐시 백엔드 연결성
    // 디스크 사용량
    // 캐시 적중률
    // TTL 만료 처리 상태
}
```

#### 3. 보안 시스템 헬스체크
```go
func TestSecuritySystemHealth(t *testing.T) {
    endpoint := "/health/security"
    // 인증 시스템 상태
    // 레이트 리미터 동작
    // IP 필터링 상태
    // SSL/TLS 인증서 유효성
}
```

## 📊 메트릭 수집 강화

### 현재 메트릭 시스템
**파일**: `metrics/metrics.go`

### 추가 필요한 메트릭

#### 1. 비즈니스 메트릭
```go
// 프록시별 요청 분석
proxynd_requests_by_type{type="npm|maven|docker|..."}

// 패키지별 다운로드 통계
proxynd_package_downloads{proxy_type="npm", package="express"}

// 사용자별 활동 메트릭  
proxynd_user_requests{user_id="...", proxy_type="..."}

// 캐시 효율성 메트릭
proxynd_cache_efficiency{proxy_type="...", cache_hit_ratio="..."}
```

#### 2. 성능 메트릭
```go
// 응답 시간 히스토그램
proxynd_response_time_histogram{proxy_type="...", status="..."}

// 동시 연결 수
proxynd_concurrent_connections

// 큐 길이 (비동기 처리)
proxynd_queue_length{queue_type="..."}

// 고루틴 수
proxynd_goroutines_count
```

#### 3. 에러 메트릭
```go
// 에러율
proxynd_error_rate{proxy_type="...", error_type="..."}

// 업스트림 에러
proxynd_upstream_errors{upstream="...", error_code="..."}

// 재시도 횟수
proxynd_retry_attempts{proxy_type="...", reason="..."}
```

## 🔍 관측성 대시보드

### Grafana 대시보드 설계

#### 1. 운영 대시보드
```json
{
  "dashboard": {
    "title": "ProxyND Operations",
    "panels": [
      {
        "title": "Overall Health",
        "type": "stat",
        "targets": ["proxynd_health_status"]
      },
      {
        "title": "Request Rate",  
        "type": "graph",
        "targets": ["rate(proxynd_requests_total[5m])"]
      },
      {
        "title": "Cache Hit Rate",
        "type": "stat", 
        "targets": ["proxynd_cache_hit_ratio"]
      }
    ]
  }
}
```

#### 2. 프록시별 대시보드
```json
{
  "dashboard": {
    "title": "ProxyND - $proxy_type",
    "templating": {
      "list": [
        {
          "name": "proxy_type",
          "options": ["npm", "maven", "docker", "apt", "pip", "yum", "apk"]
        }
      ]
    },
    "panels": [
      {
        "title": "Requests by Status",
        "targets": ["proxynd_requests_total{proxy_type=\"$proxy_type\"}"]
      }
    ]
  }
}
```

### 알림 규칙 설정

#### 1. 임계 알림
```yaml
# prometheus/rules/proxynd-critical.yml
groups:
  - name: proxynd.critical
    rules:
      - alert: ProxyNDDown
        expr: up{job="proxynd"} == 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "ProxyND is down"
          
      - alert: HighErrorRate
        expr: rate(proxynd_errors_total[5m]) > 0.1
        for: 2m
        labels:
          severity: critical
```

#### 2. 경고 알림
```yaml
# prometheus/rules/proxynd-warning.yml  
groups:
  - name: proxynd.warning
    rules:
      - alert: LowCacheHitRate
        expr: proxynd_cache_hit_ratio < 0.5
        for: 5m
        labels:
          severity: warning
          
      - alert: HighResponseTime
        expr: histogram_quantile(0.95, proxynd_response_time_histogram) > 5
        for: 3m
        labels:
          severity: warning
```

## 🔧 CI에서 헬스체크 활용

### 헬스체크 자동화 테스트

#### 1. CI 단계별 헬스체크
```yaml
# .github/workflows/ci.yml 추가
- name: Health check smoke test
  run: |
    # 서버 시작
    make dev-run &
    SERVER_PID=$!
    
    # 헬스체크 대기
    timeout 30 bash -c 'until curl -f http://localhost:8080/healthz; do sleep 1; done'
    
    # 각 어댑터 헬스체크
    for adapter in npm maven docker apt pip yum apk; do
      echo "Testing $adapter adapter..."
      curl -f "http://localhost:8080/health/check/adapter_$adapter" || exit 1
    done
    
    # 서버 종료
    kill $SERVER_PID
```

#### 2. 어댑터별 초기화 확인
```bash
#!/bin/bash
# scripts/smoke-test-adapters.sh

# 각 프록시 타입별 초기화 및 헬스체크
for proxy_type in npm maven docker apt pip yum apk; do
    echo "Testing $proxy_type proxy..."
    
    # 테스트 요청으로 어댑터 초기화
    make test-$proxy_type-adapter
    
    # 헬스체크 확인
    curl -f "http://localhost:8080/health/adapters" | \
        jq -e ".adapters.$proxy_type.initialized == true"
        
    if [ $? -eq 0 ]; then
        echo "✅ $proxy_type adapter initialized successfully"
    else
        echo "❌ $proxy_type adapter initialization failed"
        exit 1
    fi
done
```

## 🏗️ 모니터링 인프라

### Docker Compose 모니터링 스택
```yaml
# monitoring/docker-compose.yml
version: '3.8'
services:
  prometheus:
    image: prom/prometheus:latest
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
      - ./rules:/etc/prometheus/rules
    ports:
      - "9090:9090"
      
  grafana:
    image: grafana/grafana:latest
    volumes:
      - ./grafana/provisioning:/etc/grafana/provisioning
      - ./grafana/dashboards:/var/lib/grafana/dashboards
    ports:
      - "3000:3000"
    environment:
      GF_SECURITY_ADMIN_PASSWORD: admin
      
  alertmanager:
    image: prom/alertmanager:latest
    volumes:
      - ./alertmanager.yml:/etc/alertmanager/alertmanager.yml
    ports:
      - "9093:9093"
```

### 프로메테우스 설정
```yaml
# monitoring/prometheus.yml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

rule_files:
  - "rules/*.yml"

scrape_configs:
  - job_name: 'proxynd'
    static_configs:
      - targets: ['proxynd:8080']
    metrics_path: '/metrics'
    scrape_interval: 5s
    
alerting:
  alertmanagers:
    - static_configs:
        - targets: ['alertmanager:9093']
```

## 📈 성능 모니터링

### 벤치마크 자동화
```bash
#!/bin/bash
# scripts/performance-monitor.sh

# 정기적인 성능 벤치마크 실행
run_benchmark() {
    local proxy_type=$1
    echo "Running benchmark for $proxy_type..."
    
    go test -bench=Benchmark${proxy_type^}Proxy \
           -benchmem \
           -count=5 \
           -timeout=10m \
           ./tests/integration/ > "benchmarks/${proxy_type}-$(date +%Y%m%d).txt"
}

# 모든 프록시 타입 벤치마크
for proxy_type in npm maven docker apt pip yum apk; do
    run_benchmark $proxy_type
done

# 결과 분석 및 리포트 생성
python scripts/analyze-benchmarks.py
```

### 성능 회귀 검출
```python
# scripts/analyze-benchmarks.py
import json
import re
from datetime import datetime, timedelta

def analyze_performance_regression():
    """성능 회귀 분석 및 알림"""
    
    # 최근 7일간 벤치마크 결과 수집
    recent_results = collect_benchmark_results(days=7)
    
    # 성능 지표 분석
    for proxy_type in ['npm', 'maven', 'docker']:
        current_perf = recent_results[proxy_type][-1]
        baseline_perf = recent_results[proxy_type][0]
        
        # 응답 시간 회귀 검출 (20% 이상 증가)
        if current_perf['response_time'] > baseline_perf['response_time'] * 1.2:
            send_alert(f"Performance regression detected in {proxy_type} proxy")
            
        # 메모리 사용량 회귀 검출
        if current_perf['memory_usage'] > baseline_perf['memory_usage'] * 1.5:
            send_alert(f"Memory usage regression in {proxy_type} proxy")
```

## 📋 체크리스트

### 헬스체크 강화
- [ ] 업스트림 연결성 체크 구현
- [ ] 캐시 시스템 헬스체크 추가
- [ ] 보안 시스템 헬스체크 구현
- [ ] 프록시별 세부 헬스체크 강화

### 메트릭 수집 개선
- [ ] 비즈니스 메트릭 추가
- [ ] 성능 메트릭 강화  
- [ ] 에러 추적 메트릭 구현
- [ ] 사용자 활동 메트릭 수집

### 모니터링 대시보드
- [ ] Grafana 운영 대시보드 구축
- [ ] 프록시별 대시보드 생성
- [ ] 알림 규칙 설정
- [ ] 성능 트렌드 분석 도구

### CI 통합
- [ ] 헬스체크 자동화 테스트 구현
- [ ] 어댑터 초기화 검증 추가
- [ ] 성능 회귀 검출 자동화
- [ ] 모니터링 스택 CI 통합

### 문서화
- [ ] 헬스체크 가이드 작성
- [ ] 모니터링 설정 가이드
- [ ] 알림 설정 가이드  
- [ ] 트러블슈팅 가이드

## 🔗 관련 파일

- `routers/health_router.go` - 헬스체크 라우터
- `health/` - 헬스체크 구현체들
- `metrics/` - 메트릭 수집 시스템
- `monitoring/` - 모니터링 설정 파일들
- `scripts/smoke-test-adapters.sh` - 어댑터 테스트 스크립트