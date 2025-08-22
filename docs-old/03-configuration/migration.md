# ProxyND 설정 마이그레이션 가이드

## 개요

ProxyND는 이제 Viper를 사용하여 더욱 유연한 설정 관리를 지원합니다. 이 가이드는 기존 설정 시스템에서 새로운 설정 시스템으로 마이그레이션하는 방법을 설명합니다.

## 주요 변경사항

### 1. 통합 설정 파일
- 이전: 각 프록시 타입별 개별 설정 파일 (apt-proxy.yaml, maven-proxy.yaml 등)
- 현재: 통합 설정 파일 (config.yaml) + 환경별 오버라이드

### 2. 환경별 설정 분리
- 개발/스테이징/프로덕션 환경별 설정 파일 지원
- 환경 변수를 통한 설정 오버라이드
- .env 파일 지원

### 3. 설정 우선순위
1. 환경 변수
2. .env 파일
3. 환경별 설정 파일 (config.{환경}.yaml)
4. 기본 설정 파일 (config.yaml)
5. 코드에 정의된 기본값

## 마이그레이션 단계

### 1단계: 통합 설정 파일 생성

기존 개별 설정 파일들을 하나의 통합 설정 파일로 합칩니다:

```yaml
# config.yaml
server:
  port: 8080

cache:
  backend: file
  file:
    directory: /var/cache/proxynd

registries:
  npm:
    enabled: true
    upstream: https://registry.npmjs.org

  maven:
    enabled: true
    repositories:
      - id: central
        url: https://repo1.maven.org/maven2
```

### 2단계: 환경별 설정 파일 생성

환경별로 다른 설정이 필요한 경우 별도 파일을 생성합니다:

```yaml
# config.development.yaml
server:
  port: 8080

logging:
  level: debug

# config.production.yaml
server:
  port: 80
  tls:
    enabled: true

logging:
  level: warn
```

### 3단계: 환경 변수 설정

`.env` 파일을 생성하여 환경 변수를 관리합니다:

```bash
# .env
PROXYND_ENV=development
SERVER_PORT=8080
STORAGE_DIR=/var/cache/proxynd
LOG_LEVEL=info
```

### 4단계: 애플리케이션 시작

환경을 지정하여 애플리케이션을 시작합니다:

```bash
# 개발 환경
PROXYND_ENV=development ./proxynd

# 프로덕션 환경
PROXYND_ENV=production ./proxynd

# 또는 설정 파일 직접 지정
./proxynd --config=/etc/proxynd/config.yaml
```

## 설정 예제

### 기존 설정 (레거시)

```yaml
# apt-proxy.yaml
path: /apt
use_cache: true
proxies:
  - name: ubuntu
    url: http://archive.ubuntu.com/ubuntu

# maven-proxy.yaml
path: /maven
use_cache: true
repositories:
  - id: central
    url: https://repo1.maven.org/maven2
```

### 새로운 통합 설정

```yaml
# config.yaml
registries:
  apt:
    enabled: true
    user_cache: true
    mirrors:
      ubuntu:
        - name: main
          url: http://archive.ubuntu.com/ubuntu

  maven:
    enabled: true
    repositories:
      - id: central
        url: https://repo1.maven.org/maven2
```

## 환경 변수 매핑

| 기존 설정 | 환경 변수 | 설정 경로 |
|----------|-----------|-----------|
| `storage_dir` | `STORAGE_DIR` | `cache.file.directory` |
| `cache_ttl` | `CACHE_TTL` | `cache.ttl` |
| `log_level` | `LOG_LEVEL` | `logging.level` |
| `server_port` | `SERVER_PORT` | `server.port` |

## 주의사항

1. **레거시 호환성**: 기존 개별 설정 파일도 계속 지원되지만, 새로운 통합 설정 사용을 권장합니다.

2. **환경 변수 네이밍**:
   - 기본 prefix: `PROXYND_`
   - 중첩 구조는 언더스코어로 구분: `PROXYND_CACHE_BACKEND`

3. **설정 검증**: 애플리케이션 시작 시 자동으로 설정이 검증됩니다.

4. **핫 리로드**: 설정 파일 변경 시 자동으로 재로드됩니다 (프로덕션에서는 비활성화 권장).

## 문제 해결

### 설정이 적용되지 않는 경우

1. 환경 변수 확인:
   ```bash
   env | grep PROXYND
   ```

2. 설정 파일 경로 확인:
   ```bash
   ./proxynd --config-debug
   ```

3. 설정 검증:
   ```bash
   ./proxynd config validate
   ```

### 마이그레이션 도구

기존 설정을 새로운 형식으로 변환하는 도구:

```bash
./proxynd config migrate --old-config-dir=/etc/proxynd/old --output=/etc/proxynd/config.yaml
```

## 추가 리소스

- [설정 참조 문서](./CONFIGURATION_REFERENCE.md)
- [환경 변수 목록](./ENVIRONMENT_VARIABLES.md)
- [예제 설정 파일](../../examples/)
