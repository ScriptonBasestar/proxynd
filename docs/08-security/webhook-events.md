# Webhook 이벤트 타입 가이드

ProxyND는 다양한 시스템 이벤트에 대해 웹훅 알림을 지원합니다. 이 문서는 지원되는 모든 이벤트 타입과 각각의 의미를 설명합니다.

## 이벤트 카테고리

### 1. 캐시 관련 이벤트 (`cache.*`)

| 이벤트 타입 | 설명 | 레벨 | 예시 상황 |
|------------|------|------|----------|
| `cache.expiry` | 캐시 항목이 TTL에 의해 만료됨 | INFO | npm 패키지가 3600초 후 만료 |
| `cache.miss` | 요청된 항목이 캐시에 없음 | INFO | 새로운 패키지 요청 시 |
| `cache.eviction` | 메모리 부족으로 캐시 항목 축출 | WARNING | 캐시 용량 초과 시 |
| `cache.full` | 캐시 저장소가 가득 참 | ERROR | 디스크 공간 부족 |
| `cache.error` | 캐시 시스템 오류 | ERROR | 캐시 백엔드 연결 실패 |
| `cache.cleared` | 캐시 수동 정리 | INFO | 관리자가 캐시 정리 실행 |
| `cache.corruption` | 캐시 데이터 손상 감지 | ERROR | 파일 해시 불일치 |

### 2. 인증 및 권한 관련 이벤트 (`auth.*`)

| 이벤트 타입 | 설명 | 레벨 | 예시 상황 |
|------------|------|------|----------|
| `auth.failure` | 인증 실패 | WARNING | 잘못된 사용자명/비밀번호 |
| `auth.success` | 인증 성공 | INFO | 정상 로그인 |
| `auth.blocked` | 계정 차단 | ERROR | 반복적인 인증 실패 |
| `auth.rate_limit` | 인증 시도 속도 제한 | WARNING | 짧은 시간 내 많은 로그인 시도 |
| `auth.permission_denied` | 권한 거부 | WARNING | 접근 권한 없는 리소스 요청 |
| `auth.unauthorized` | 무권한 접근 시도 | WARNING | 토큰 없이 보호된 API 호출 |

### 3. 정책 위반 관련 이벤트 (`policy.*`)

| 이벤트 타입 | 설명 | 레벨 | 예시 상황 |
|------------|------|------|----------|
| `policy.violation` | 일반적인 정책 위반 | WARNING | 정의된 규칙 위반 |
| `policy.package_blocked` | 패키지 차단 | ERROR | 금지된 패키지 다운로드 시도 |
| `policy.size_exceeded` | 크기 제한 초과 | WARNING | 최대 패키지 크기 초과 |
| `policy.rate_limited` | 속도 제한 적용 | WARNING | API 호출 한도 초과 |
| `policy.ip_blocked` | IP 주소 차단 | ERROR | 블랙리스트 IP에서 접근 |
| `policy.quota_exceeded` | 할당량 초과 | WARNING | 사용자별 다운로드 한도 초과 |

### 4. 서버 상태 변경 이벤트 (`server.*`)

| 이벤트 타입 | 설명 | 레벨 | 예시 상황 |
|------------|------|------|----------|
| `server.started` | 서버 시작 | INFO | ProxyND 프로세스 시작 |
| `server.stopped` | 서버 중지 | INFO | 정상적인 서버 종료 |
| `server.restarted` | 서버 재시작 | INFO | 설정 변경 후 재시작 |
| `server.health_failed` | 헬스체크 실패 | ERROR | `/healthz` 엔드포인트 응답 없음 |
| `server.health_passed` | 헬스체크 성공 | INFO | 헬스체크 정상 복구 |
| `server.config_reloaded` | 설정 재로드 | INFO | 런타임 설정 변경 |
| `server.config_error` | 설정 오류 | ERROR | 잘못된 설정 파일 |

### 5. 패키지 관련 이벤트 (`package.*`)

| 이벤트 타입 | 설명 | 레벨 | 예시 상황 |
|------------|------|------|----------|
| `package.downloaded` | 패키지 다운로드 완료 | INFO | 업스트림에서 패키지 다운로드 |
| `package.uploaded` | 패키지 업로드 완료 | INFO | 사용자가 패키지 업로드 |
| `package.corrupted` | 패키지 손상 감지 | ERROR | 다운로드 중 파일 손상 |
| `package.verify_failed` | 패키지 검증 실패 | ERROR | 메타데이터 검증 실패 |
| `package.signature_invalid` | 서명 검증 실패 | ERROR | APK 서명 불일치 |
| `package.hash_mismatch` | 해시 불일치 | ERROR | 체크섬 검증 실패 |

### 6. 미러 및 프록시 이벤트 (`mirror.*`, `proxy.*`, `upstream.*`)

| 이벤트 타입 | 설명 | 레벨 | 예시 상황 |
|------------|------|------|----------|
| `mirror.down` | 미러 서버 다운 | ERROR | 업스트림 미러 응답 없음 |
| `mirror.up` | 미러 서버 복구 | INFO | 다운된 미러 서버 복구 |
| `mirror.slow` | 미러 서버 응답 지연 | WARNING | 느린 응답 시간 |
| `mirror.error` | 미러 서버 오류 | ERROR | HTTP 5xx 응답 |
| `proxy.fallback` | 프록시 폴백 | WARNING | 주 서버 실패로 대체 서버 사용 |
| `upstream.timeout` | 업스트림 타임아웃 | WARNING | 업스트림 응답 시간 초과 |

### 7. 보안 관련 이벤트 (`security.*`)

| 이벤트 타입 | 설명 | 레벨 | 예시 상황 |
|------------|------|------|----------|
| `security.threat` | 보안 위협 감지 | WARNING | 의심스러운 활동 패턴 |
| `security.malware` | 악성코드 감지 | CRITICAL | 바이러스 스캐너가 위험 파일 발견 |
| `security.vulnerability` | 취약점 발견 | WARNING | 알려진 CVE가 있는 패키지 |
| `security.audit_full` | 감사 로그 가득 참 | WARNING | 로그 저장소 용량 부족 |
| `security.suspicious` | 의심스러운 활동 | WARNING | 비정상적인 접근 패턴 |

### 8. 시스템 리소스 이벤트 (`system.*`)

| 이벤트 타입 | 설명 | 레벨 | 예시 상황 |
|------------|------|------|----------|
| `system.disk_full` | 디스크 가득 참 | CRITICAL | 디스크 사용률 95% 이상 |
| `system.disk_low` | 디스크 부족 | WARNING | 디스크 사용률 80% 이상 |
| `system.memory_high` | 메모리 사용량 높음 | WARNING | 메모리 사용률 85% 이상 |
| `system.cpu_high` | CPU 사용량 높음 | WARNING | CPU 사용률 90% 이상 |
| `system.network_error` | 네트워크 오류 | ERROR | 네트워크 연결 실패 |

### 9. 사용자 및 관리 이벤트 (`user.*`, `admin.*`)

| 이벤트 타입 | 설명 | 레벨 | 예시 상황 |
|------------|------|------|----------|
| `user.created` | 사용자 생성 | INFO | 새 사용자 계정 생성 |
| `user.deleted` | 사용자 삭제 | INFO | 사용자 계정 삭제 |
| `user.modified` | 사용자 정보 수정 | INFO | 사용자 권한 변경 |
| `admin.config_changed` | 설정 변경 | INFO | 관리자 설정 수정 |
| `admin.backup_completed` | 백업 완료 | INFO | 자동 백업 성공 |
| `admin.backup_failed` | 백업 실패 | ERROR | 백업 프로세스 실패 |

## 이벤트 구조

모든 웹훅 이벤트는 다음과 같은 공통 구조를 가집니다:

```json
{
  "id": "evt_1234567890",
  "level": "WARNING",
  "type": "auth.failure",
  "title": "Authentication Failed",
  "message": "User 'testuser' authentication failed from IP 192.168.1.100",
  "source": "proxynd",
  "timestamp": 1640995200,
  "metadata": {
    "username": "testuser",
    "client_ip": "192.168.1.100",
    "user_agent": "npm/8.0.0",
    "request_path": "/npm/express"
  },
  "package": {
    "type": "npm",
    "name": "express",
    "version": "4.18.0",
    "path": "/npm/express",
    "remote_url": "https://registry.npmjs.org"
  }
}
```

## 이벤트 필터링

웹훅 설정에서 다음과 같은 방법으로 이벤트를 필터링할 수 있습니다:

### 1. 이벤트 타입 필터링

```yaml
endpoints:
  - name: "security-alerts"
    event_types:
      - "security.*"      # 모든 보안 이벤트
      - "auth.failure"    # 특정 인증 실패 이벤트
      - "system.disk_*"   # 디스크 관련 이벤트
```

### 2. 레벨 필터링

```yaml
endpoints:
  - name: "critical-alerts"
    filters:
      min_level: "ERROR"  # ERROR, CRITICAL 레벨만
```

### 3. 패키지 타입 필터링

```yaml
endpoints:
  - name: "npm-alerts"
    filters:
      package_types:
        - "npm"
        - "pip"
```

### 4. 제외 필터

```yaml
endpoints:
  - name: "important-alerts"
    filters:
      exclude_types:
        - "cache.miss"    # 캐시 미스는 제외 (너무 빈번)
        - "auth.success"  # 성공적인 인증은 제외
```

## 플랫폼별 포맷

### Slack 포맷

```json
{
  "text": "🚨 ProxyND Alert",
  "attachments": [
    {
      "color": "warning",
      "title": "Authentication Failed",
      "text": "User 'testuser' authentication failed from IP 192.168.1.100",
      "fields": [
        {
          "title": "Event Type",
          "value": "auth.failure",
          "short": true
        },
        {
          "title": "Client IP",
          "value": "192.168.1.100",
          "short": true
        }
      ],
      "ts": 1640995200
    }
  ]
}
```

### Discord 포맷

```json
{
  "embeds": [
    {
      "title": "ProxyND Alert",
      "description": "Authentication Failed",
      "color": 16744448,
      "fields": [
        {
          "name": "Event Type",
          "value": "auth.failure",
          "inline": true
        },
        {
          "name": "Username",
          "value": "testuser",
          "inline": true
        }
      ],
      "timestamp": "2021-12-31T12:00:00.000Z"
    }
  ]
}
```

### Microsoft Teams 포맷

```json
{
  "@type": "MessageCard",
  "@context": "http://schema.org/extensions",
  "themeColor": "FF6347",
  "summary": "ProxyND Alert",
  "sections": [
    {
      "activityTitle": "Authentication Failed",
      "activitySubtitle": "auth.failure",
      "facts": [
        {
          "name": "Username",
          "value": "testuser"
        },
        {
          "name": "Client IP",
          "value": "192.168.1.100"
        }
      ]
    }
  ]
}
```

## 이벤트 생성 예제

개발자를 위한 이벤트 생성 예제입니다:

```go
import "proxynd/alerts"

// 캐시 만료 이벤트
cacheEvent := alerts.CreateCacheEvent(
    alerts.EventCacheExpiry,
    "/npm/express/4.18.0", 
    1024000, 
    "express 패키지 캐시가 만료되었습니다"
)

// 인증 실패 이벤트
authEvent := alerts.CreateAuthEvent(
    alerts.EventAuthFailure,
    "testuser",
    "192.168.1.100",
    false,
    "사용자 인증에 실패했습니다"
)

// 보안 위협 이벤트
securityEvent := alerts.CreateSecurityEvent(
    alerts.EventSecurityThreat,
    "high",
    "의심스러운 활동이 감지되었습니다",
    map[string]interface{}{
        "source_ip": "192.168.1.100",
        "attack_type": "brute_force",
        "attempts": 10,
    }
)

// 이벤트 전송
alertManager.Send(context.Background(), cacheEvent)
```

## 베스트 프랙티스

1. **중요도에 따른 필터링**: 중요한 알림만 즉시 전송하고, 정보성 이벤트는 집계하여 전송
2. **중복 제거**: 같은 유형의 이벤트가 반복될 때 중복 제거 설정 활용
3. **속도 제한**: 과도한 알림으로 인한 스팸 방지를 위해 적절한 속도 제한 설정
4. **재시도 설정**: 네트워크 오류 등으로 인한 일시적 실패에 대비한 재시도 설정
5. **보안 고려**: 민감한 정보가 포함된 웹훅의 경우 HTTPS 및 인증 설정 필수