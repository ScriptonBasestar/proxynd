# ProxyND 환경 변수 레퍼런스

## 개요

ProxyND는 환경 변수를 통한 설정 오버라이드를 지원합니다. 환경 변수는 설정 파일보다 우선순위가 높으며, 컨테이너 환경에서 특히 유용합니다.

## 환경 변수 네이밍 규칙

- 기본 prefix: `PROXYND_` (Viper 사용 시)
- 중첩 구조는 언더스코어(`_`)로 구분
- 예: `server.port` → `PROXYND_SERVER_PORT`

## 환경 변수 목록

### 기본 설정

| 환경 변수 | 설정 경로 | 타입 | 기본값 | 설명 |
|-----------|-----------|------|--------|------|
| `PROXYND_ENV` | - | string | development | 실행 환경 (development, staging, production) |

### 서버 설정

| 환경 변수 | 설정 경로 | 타입 | 기본값 | 설명 |
|-----------|-----------|------|--------|------|
| `SERVER_HOST` | server.host | string | 0.0.0.0 | 서버 바인딩 주소 |
| `SERVER_PORT` | server.port | int | 8080 | 서버 포트 |
| `SERVER_READ_TIMEOUT` | server.read_timeout | duration | 30s | 읽기 타임아웃 |
| `SERVER_WRITE_TIMEOUT` | server.write_timeout | duration | 30s | 쓰기 타임아웃 |
| `SERVER_IDLE_TIMEOUT` | server.idle_timeout | duration | 120s | 유휴 연결 타임아웃 |

### TLS 설정

| 환경 변수 | 설정 경로 | 타입 | 기본값 | 설명 |
|-----------|-----------|------|--------|------|
| `TLS_ENABLED` | server.tls.enabled | bool | false | HTTPS 활성화 |
| `TLS_CERT_FILE` | server.tls.cert_file | string | - | TLS 인증서 파일 경로 |
| `TLS_KEY_FILE` | server.tls.key_file | string | - | TLS 개인키 파일 경로 |

### 캐시 설정

| 환경 변수 | 설정 경로 | 타입 | 기본값 | 설명 |
|-----------|-----------|------|--------|------|
| `CACHE_BACKEND` | cache.backend | string | file | 캐시 백엔드 (file, s3, redis) |
| `CACHE_TTL` | cache.ttl | duration | 3600s | 캐시 TTL |
| `CACHE_MAX_SIZE` | cache.max_size | string | 10GB | 최대 캐시 크기 |
| `STORAGE_DIR` | cache.file.directory | string | /var/cache/proxynd | 파일 캐시 디렉토리 |

### S3 캐시 설정

| 환경 변수 | 설정 경로 | 타입 | 기본값 | 설명 |
|-----------|-----------|------|--------|------|
| `S3_ENDPOINT` | cache.s3.endpoint | string | s3.amazonaws.com | S3 엔드포인트 |
| `S3_BUCKET` | cache.s3.bucket | string | - | S3 버킷 이름 |
| `S3_REGION` | cache.s3.region | string | us-east-1 | S3 리전 |
| `AWS_ACCESS_KEY_ID` | cache.s3.access_key_id | string | - | AWS 액세스 키 |
| `AWS_SECRET_ACCESS_KEY` | cache.s3.secret_access_key | string | - | AWS 시크릿 키 |

### Redis 캐시 설정

| 환경 변수 | 설정 경로 | 타입 | 기본값 | 설명 |
|-----------|-----------|------|--------|------|
| `REDIS_ADDRESS` | cache.redis.address | string | localhost:6379 | Redis 주소 |
| `REDIS_PASSWORD` | cache.redis.password | string | - | Redis 비밀번호 |
| `REDIS_DB` | cache.redis.db | int | 0 | Redis 데이터베이스 번호 |

### 레지스트리 설정

| 환경 변수 | 설정 경로 | 타입 | 기본값 | 설명 |
|-----------|-----------|------|--------|------|
| `NPM_ENABLED` | registries.npm.enabled | bool | true | NPM 레지스트리 활성화 |
| `NPM_UPSTREAM` | registries.npm.upstream | string | https://registry.npmjs.org | NPM 업스트림 URL |
| `PYPI_ENABLED` | registries.pypi.enabled | bool | true | PyPI 레지스트리 활성화 |
| `PYPI_UPSTREAM` | registries.pypi.upstream | string | https://pypi.org | PyPI 업스트림 URL |
| `APT_ENABLED` | registries.apt.enabled | bool | true | APT 레지스트리 활성화 |
| `DOCKER_ENABLED` | registries.docker.enabled | bool | true | Docker 레지스트리 활성화 |
| `MAVEN_ENABLED` | registries.maven.enabled | bool | true | Maven 레지스트리 활성화 |

### 보안 설정

| 환경 변수 | 설정 경로 | 타입 | 기본값 | 설명 |
|-----------|-----------|------|--------|------|
| `AUTH_ENABLED` | security.authentication.basic_auth.enabled | bool | false | 기본 인증 활성화 |
| `AUTH_USERS_FILE` | security.authentication.basic_auth.users_file | string | users.yml | 사용자 파일 경로 |
| `IP_WHITELIST_ENABLED` | security.access_control.ip_whitelist.enabled | bool | false | IP 화이트리스트 활성화 |

### 로깅 설정

| 환경 변수 | 설정 경로 | 타입 | 기본값 | 설명 |
|-----------|-----------|------|--------|------|
| `LOG_LEVEL` | logging.level | string | info | 로그 레벨 (debug, info, warn, error) |
| `LOG_FORMAT` | logging.format | string | json | 로그 포맷 (json, text) |
| `LOG_OUTPUT` | logging.output | string | stdout | 로그 출력 (stdout, stderr, file) |
| `LOG_FILE` | logging.file.path | string | - | 로그 파일 경로 |
| `ACCESS_LOG_ENABLED` | logging.access_log.enabled | bool | true | 액세스 로그 활성화 |
| `ACCESS_LOG_PATH` | logging.access_log.path | string | - | 액세스 로그 파일 경로 |

### 메트릭 설정

| 환경 변수 | 설정 경로 | 타입 | 기본값 | 설명 |
|-----------|-----------|------|--------|------|
| `METRICS_ENABLED` | metrics.enabled | bool | false | 메트릭 활성화 |
| `METRICS_PATH` | metrics.path | string | /metrics | 메트릭 엔드포인트 경로 |
| `METRICS_PORT` | metrics.port | int | 0 | 메트릭 포트 (0=메인 포트) |

### 패키지 검증

| 환경 변수 | 설정 경로 | 타입 | 기본값 | 설명 |
|-----------|-----------|------|--------|------|
| `VERIFICATION_ENABLED` | verification.strict_mode | bool | false | 엄격 모드 활성화 |
| `VERIFICATION_BLOCK` | verification.block_on_failure | bool | false | 검증 실패 시 차단 |
| `VERIFICATION_ALERT` | verification.alert_on_failure | bool | true | 검증 실패 시 알림 |

### 알림 설정

| 환경 변수 | 설정 경로 | 타입 | 기본값 | 설명 |
|-----------|-----------|------|--------|------|
| `ALERTS_ENABLED` | alerts.enabled | bool | false | 알림 활성화 |
| `WEBHOOK_URL` | alerts.channels[1].config.url | string | - | 웹훅 URL |

### 고급 설정

| 환경 변수 | 설정 경로 | 타입 | 기본값 | 설명 |
|-----------|-----------|------|--------|------|
| `MAX_CONNECTIONS` | advanced.performance.max_connections | int | 1000 | 최대 연결 수 |
| `CONNECTION_TIMEOUT` | advanced.performance.connection_timeout | duration | 30s | 연결 타임아웃 |
| `RETRY_MAX_ATTEMPTS` | advanced.retry.max_attempts | int | 3 | 최대 재시도 횟수 |
| `CIRCUIT_BREAKER_ENABLED` | advanced.circuit_breaker.enabled | bool | false | 회로 차단기 활성화 |

## 사용 예제

### Docker Compose

```yaml
version: '3.8'
services:
  proxynd:
    image: proxynd:latest
    environment:
      - PROXYND_ENV=production
      - SERVER_PORT=80
      - CACHE_BACKEND=redis
      - REDIS_ADDRESS=redis:6379
      - LOG_LEVEL=warn
      - METRICS_ENABLED=true
      - AUTH_ENABLED=true
```

### Kubernetes

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: proxynd-env
data:
  PROXYND_ENV: "production"
  SERVER_PORT: "80"
  CACHE_BACKEND: "redis"
  REDIS_ADDRESS: "redis-service:6379"
  LOG_LEVEL: "warn"
  METRICS_ENABLED: "true"
  AUTH_ENABLED: "true"
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: proxynd
spec:
  template:
    spec:
      containers:
      - name: proxynd
        image: proxynd:latest
        envFrom:
        - configMapRef:
            name: proxynd-env
        env:
        - name: REDIS_PASSWORD
          valueFrom:
            secretKeyRef:
              name: redis-secret
              key: password
```

### Shell

```bash
# 개발 환경
export PROXYND_ENV=development
export SERVER_PORT=8080
export LOG_LEVEL=debug
./proxynd

# 프로덕션 환경
PROXYND_ENV=production \
SERVER_PORT=80 \
CACHE_BACKEND=redis \
REDIS_ADDRESS=redis.internal:6379 \
LOG_LEVEL=warn \
AUTH_ENABLED=true \
./proxynd
```

## 타입 설명

### duration

Go의 시간 형식을 사용합니다:
- `300ms`: 300 밀리초
- `1.5s`: 1.5초
- `2m`: 2분
- `1h`: 1시간
- `1h30m`: 1시간 30분

### string (크기)

크기 표현:
- `100MB`: 100 메가바이트
- `10GB`: 10 기가바이트
- `1TB`: 1 테라바이트

### bool

불린 값:
- `true`, `1`, `yes`, `on`: true
- `false`, `0`, `no`, `off`: false

## 주의사항

1. **대소문자**: 환경 변수 이름은 대소문자를 구분합니다.
2. **우선순위**: 환경 변수 > .env 파일 > 설정 파일 > 기본값
3. **타입 변환**: 잘못된 타입의 값은 무시되고 기본값이 사용됩니다.
4. **보안**: 민감한 정보(비밀번호, 토큰 등)는 환경 변수나 시크릿 관리 도구를 사용하세요.

## 디버깅

환경 변수가 제대로 적용되었는지 확인:

```bash
# 환경 변수 확인
env | grep -E "PROXYND_|SERVER_|CACHE_|LOG_"

# 설정 디버그 모드로 실행
./proxynd --config-debug

# 최종 설정 확인
./proxynd config show
```