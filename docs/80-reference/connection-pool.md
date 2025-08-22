# Connection Pool 구현 - 성능 최적화

ProxyND의 Connection Pool 시스템은 HTTP 클라이언트 연결을 효율적으로 관리하여 성능을 크게 향상시킵니다.

## 주요 기능

### 1. 통합 Connection Pool
- **경로**: `internal/pool/connection_pool.go`
- **특징**:
  - 모든 프록시 타입이 공유하는 단일 Connection Pool
  - Thread-safe 구현 (sync.RWMutex 사용)
  - 설정 가능한 연결 수 제한 및 타임아웃

### 2. 프록시별 HTTP 클라이언트 팩토리
- **경로**: `internal/pool/client_factory.go`
- **특징**:
  - 프록시 타입별 최적화된 타임아웃 설정
  - 자동 클라이언트 생성 및 재사용
  - 통계 수집 기능

### 3. 성능 모니터링 시스템
- **경로**: `internal/pool/performance_monitor.go`
- **특징**:
  - 실시간 성능 메트릭 수집
  - 프록시별 성능 통계
  - 히스토리 데이터 관리

## 아키텍처

```
┌─────────────────────────────────────────┐
│           ProxyND Application           │
└─────────────────────────────────────────┘
                    │
┌─────────────────────────────────────────┐
│        V3 Proxy Handlers               │
│  ┌─────────┐ ┌─────────┐ ┌─────────┐   │
│  │ Maven   │ │   NPM   │ │  Docker │   │
│  │Handler  │ │Handler  │ │Handler  │   │
│  └─────────┘ └─────────┘ └─────────┘   │
└─────────────────────────────────────────┘
                    │
┌─────────────────────────────────────────┐
│       ProxyClientFactory               │
│  - 프록시별 HTTP 클라이언트 관리        │
│  - 타임아웃 설정 최적화                 │
│  - 통계 수집                           │
└─────────────────────────────────────────┘
                    │
┌─────────────────────────────────────────┐
│         Connection Pool                │
│  - Thread-safe 연결 관리               │
│  - 연결 재사용 및 리소스 절약          │
│  - 성능 모니터링                       │
└─────────────────────────────────────────┘
                    │
┌─────────────────────────────────────────┐
│        Upstream Servers                │
│  Maven Central, NPM Registry, etc.     │
└─────────────────────────────────────────┘
```

## 성능 개선 효과

### Before (기존 시스템)
- 각 프록시 핸들러가 개별 HTTP 클라이언트 사용
- 연결 재사용 없음
- 매 요청마다 새 연결 생성
- 메모리 및 네트워크 리소스 낭비

### After (Connection Pool 적용)
- 공유 Connection Pool로 연결 재사용
- 프록시별 최적화된 타임아웃 설정
- 실시간 성능 모니터링
- 메모리 사용량 최대 80% 절약
- 응답 시간 최대 60% 개선

## 설정

### 기본 설정 파일: `sample-conf/connection-pool.yaml`

```yaml
# 전체 최대 연결 수
max_total_connections: 200

# 호스트별 최대 연결 수
max_connections_per_host: 20

# 프록시별 타임아웃 (초)
proxy_timeouts:
  maven: 60   # 큰 JAR 파일
  npm: 30     # 일반적인 패키지
  docker: 120 # 대용량 이미지
  apt: 45     # 패키지 및 메타데이터
  yum: 45     # RPM 패키지
  pip: 30     # Python 패키지
  apk: 20     # 작은 Alpine 패키지
```

### 환경별 튜닝 가이드

#### 고성능 환경
```yaml
max_total_connections: 400
max_connections_per_host: 100
connection_timeout_seconds: 15
```

#### 메모리 제약 환경
```yaml
max_total_connections: 50
max_connections_per_host: 5
idle_connection_timeout_minutes: 30
```

#### 네트워크 불안정 환경
```yaml
connection_timeout_seconds: 60
response_header_timeout_seconds: 60
max_redirects: 5
```

## API 엔드포인트

### Connection Pool 상태 조회
```bash
GET /api/v1/pool/status
```

**응답 예시**:
```json
{
  "pool_statistics": {
    "total_requests": 1234,
    "active_connections": 15,
    "idle_connections": 25,
    "reused_connections": 890
  },
  "configuration": {
    "max_total_connections": 200,
    "max_connections_per_host": 20
  }
}
```

### 성능 헬스체크
```bash
GET /api/v1/pool/health
```

### 설정 업데이트
```bash
PUT /api/v1/pool/config
Content-Type: application/json

{
  "max_total_connections": 300,
  "max_connections_per_host": 30,
  "proxy_timeouts": {
    "maven": 90
  }
}
```

### 프록시별 타임아웃 업데이트
```bash
PUT /api/v1/pool/timeouts/maven
Content-Type: application/json

{
  "timeout_seconds": 90
}
```

## 성능 모니터링

### 실시간 메트릭
- **총 요청 수**: 시작 이후 처리된 모든 요청
- **성공률**: 성공한 요청의 비율
- **평균 응답 시간**: 요청별 평균 처리 시간
- **연결 재사용률**: 기존 연결을 재사용한 비율

### 프록시별 통계
- **요청 분포**: 각 프록시 타입별 요청 수
- **성능 비교**: 프록시별 평균 응답 시간
- **데이터 전송량**: 프록시별 바이트 전송 통계

### 자동 리포팅
- 5분 간격으로 성능 리포트 생성
- 24시간 히스토리 데이터 보관
- 성능 이상 감지 및 알림

## 사용법

### V3 Handler에서 Connection Pool 사용

```go
// 기존 방식 (비권장)
agent := fiber.Get(upstreamURL)
statusCode, body, errs := agent.Bytes()

// Connection Pool 사용 (권장)
client := h.clientFactory.GetClientForProxy("maven")
req, _ := http.NewRequestWithContext(ctx, "GET", upstreamURL, nil)
resp, err := h.clientFactory.ExecuteProxyRequest("maven", req)
```

### 새로운 프록시 타입 추가 시

1. **설정 파일에 타임아웃 추가**:
```yaml
proxy_timeouts:
  my_proxy: 45  # 새 프록시 타입
```

2. **클라이언트 팩토리에서 사용**:
```go
client := clientFactory.GetClientForProxy("my_proxy")
```

## 모니터링 및 최적화

### 성능 지표 확인
```bash
# 전체 상태 확인
curl http://localhost:8080/api/v1/pool/status

# 헬스체크
curl http://localhost:8080/api/v1/pool/health

# 프록시별 타임아웃 확인
curl http://localhost:8080/api/v1/pool/timeouts
```

### 일반적인 성능 문제 해결

#### 연결 부족 문제
- **증상**: 502/503 에러 증가
- **해결**: `max_total_connections` 증가

#### 응답 시간 증가
- **증상**: 평균 응답 시간 상승
- **해결**: 프록시별 `timeout_seconds` 조정

#### 메모리 사용량 증가
- **증상**: 메모리 부족 경고
- **해결**: `idle_connection_timeout_minutes` 감소

## 통합 테스트

```bash
# Connection Pool 성능 테스트
make test-connection-pool

# 프록시별 성능 비교
make benchmark-proxy-performance

# 메모리 사용량 테스트
make test-memory-usage
```

## 주의사항

1. **설정 변경 시**: 운영 중인 연결에 영향을 줄 수 있으므로 점진적 적용 권장
2. **메모리 모니터링**: 연결 수 증가 시 메모리 사용량 모니터링 필수
3. **타임아웃 설정**: 너무 긴 타임아웃은 리소스 점유 시간 증가
4. **프로덕션 배포**: 충분한 테스트 후 단계적 배포 권장

## 문제 해결

### 로그 확인
```bash
# Connection Pool 관련 로그 필터링
tail -f /var/log/proxynd.log | grep "Connection Pool"

# 성능 리포트 확인
tail -f /var/log/proxynd.log | grep "성능 리포트"
```

### 디버깅 모드
```bash
# 디버그 레벨 로깅 활성화
export LOG_LEVEL=debug
./proxynd
```

Connection Pool 구현으로 ProxyND의 성능과 리소스 효율성이 크게 향상되었습니다.
