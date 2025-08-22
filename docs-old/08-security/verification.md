# 패키지 검증 및 알림 시스템

ProxyND의 패키지 검증 시스템은 프록시를 통해 전달되는 패키지의 무결성을 검증하고, 검증 실패 시 적절한 조치를 취하는 기능을 제공합니다.

## 주요 기능

### 1. 해시 기반 검증
- **SHA256, SHA512, SHA1** 등 다양한 해시 알고리즘 지원
- 패키지 타입별 표준 검증 방식 적용
  - NPM: `integrity` 필드 (SHA512)
  - PyPI: SHA256 체크섬
  - APT: Release 파일의 SHA256
  - Docker: Manifest digest (SHA256)
  - Maven: SHA1/SHA256 체크섬

### 2. 실시간 검증
- **다운로드 시**: 원격 저장소에서 받은 패키지 검증
- **업로드 시**: 사용자가 업로드하는 패키지 검증
- **캐시 제공 시**: 캐시된 패키지의 무결성 확인

### 3. 유연한 정책 설정
- **엄격 모드**: 검증 실패 시 패키지 전달 차단
- **경고 모드**: 검증 실패 시 경고만 발생
- **패키지 타입별 설정**: 각 레지스트리 타입별로 다른 정책 적용

### 4. 다중 채널 알림
- **로그 알림**: 파일 또는 콘솔로 알림 기록
- **웹훅 알림**: Slack, Discord 등 외부 서비스 연동
- **이메일 알림**: SMTP를 통한 이메일 전송 (예정)

## 설정 방법

### 1. 기본 설정 (`verification-config.yaml`)

```yaml
verification:
  # 엄격 모드 활성화
  strict_mode: true

  # 검증 실패 시 차단
  block_on_failure: true

  # 검증 실패 시 알림
  alert_on_failure: true

  # 패키지 타입별 설정
  package_types:
    npm:
      enabled: true
      required_hashes: ["sha512"]
      trusted_sources:
        - "https://registry.npmjs.org"
```

### 2. 알림 채널 설정

```yaml
alerts:
  enabled: true

  channels:
    # 로그 기반 알림
    - name: log
      type: log
      enabled: true
      config:
        log_file: "./logs/verification-alerts.log"
        json_format: true

    # 웹훅 알림 (Slack 예시)
    - name: slack
      type: webhook
      enabled: true
      config:
        url: "https://hooks.slack.com/services/YOUR/WEBHOOK/URL"
        method: POST
        timeout: "30s"
        retry_count: 3
```

### 3. 속도 제한 설정

```yaml
alerts:
  rate_limit:
    enabled: true
    max_per_minute: 10     # 분당 최대 알림 수
    max_per_hour: 100      # 시간당 최대 알림 수
    burst_size: 5          # 순간 최대 알림 수
```

## 검증 프로세스

### 1. 다운로드 검증 플로우
```
클라이언트 → ProxyND → 원격 저장소
                ↓
           패키지 다운로드
                ↓
           해시 계산/비교
                ↓
        검증 성공?
        ├─ Yes → 캐시 저장 → 클라이언트 전달
        └─ No  → 알림 발생
                  ↓
              차단 모드?
              ├─ Yes → 403 Forbidden
              └─ No  → 경고 후 전달
```

### 2. 업로드 검증 플로우
```
클라이언트 → ProxyND
        ↓
    요청 헤더 확인
    (Content-SHA256 등)
        ↓
    본문 해시 계산
        ↓
    해시 비교
        ↓
    검증 성공?
    ├─ Yes → 원격 저장소로 전달
    └─ No  → 400 Bad Request
```

## 알림 형식

### 로그 알림 (JSON)
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "level": "CRITICAL",
  "type": "package_verification_failed",
  "title": "Package Verification Failed",
  "message": "Verification failed for npm package: @angular/core",
  "source": "package_verifier",
  "timestamp": "2024-01-20T15:30:45Z",
  "package_info": {
    "type": "npm",
    "name": "@angular/core",
    "version": "17.0.0",
    "path": "@angular/core/-/core-17.0.0.tgz",
    "expected_hash": "sha512-abc123...",
    "actual_hash": "sha512-def456...",
    "remote_url": "https://registry.npmjs.org/@angular/core/-/core-17.0.0.tgz"
  }
}
```

### 웹훅 알림 (Slack)
```json
{
  "text": "🚨 Package Verification Failed",
  "attachments": [{
    "color": "danger",
    "fields": [
      {
        "title": "Package",
        "value": "npm/@angular/core@17.0.0",
        "short": true
      },
      {
        "title": "Reason",
        "value": "Hash mismatch",
        "short": true
      }
    ],
    "footer": "ProxyND",
    "ts": 1705762245
  }]
}
```

## 보안 고려사항

1. **해시 알고리즘 선택**
   - SHA256 이상 권장 (SHA1, MD5는 보안상 취약)
   - 패키지 매니저의 기본 알고리즘 준수

2. **신뢰할 수 있는 소스**
   - 공식 레지스트리 URL만 신뢰
   - HTTPS 연결 필수

3. **검증 우회 방지**
   - 검증 건너뛰기 패턴은 최소화
   - 정기적인 검증 정책 검토

4. **알림 보안**
   - 웹훅 URL 보호 (환경 변수 사용)
   - 민감한 정보 마스킹

## 문제 해결

### 검증이 계속 실패하는 경우
1. 원격 저장소의 체크섬 파일 확인
2. 네트워크 프록시 설정 확인
3. 해시 알고리즘 불일치 확인

### 알림이 전송되지 않는 경우
1. 알림 채널 설정 확인
2. 속도 제한 설정 확인
3. 로그 파일 권한 확인

### 성능 이슈
1. 대용량 파일의 경우 스트리밍 해시 계산
2. 검증 결과 캐싱 고려
3. 비동기 알림 전송

## 향후 개선 사항

- [ ] GPG 서명 검증 (APT)
- [ ] 인증서 기반 검증
- [ ] 검증 결과 대시보드
- [ ] 자동 차단 목록 관리
- [ ] 머신러닝 기반 이상 탐지
