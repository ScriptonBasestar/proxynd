# APK 서명 검증 가이드

이 문서는 ProxyND의 Alpine APK 프록시에서 지원하는 패키지 서명 검증 기능에 대해 설명합니다.

## 목차

- [개요](#개요)
- [설정 방법](#설정-방법)
- [신뢰할 수 있는 키 관리](#신뢰할-수-있는-키-관리)
- [검증 프로세스](#검증-프로세스)
- [API 엔드포인트](#api-엔드포인트)
- [문제 해결](#문제-해결)

## 개요

Alpine Linux의 APK 패키지는 RSA 서명을 통해 무결성과 신뢰성을 보장합니다. ProxyND는 다운로드되는 APK 패키지의 서명을 자동으로 검증하여 보안을 강화할 수 있습니다.

### 주요 기능

- **자동 서명 검증**: APK 파일 다운로드 시 자동으로 서명 검증 수행
- **신뢰할 수 있는 키 관리**: 여러 공개키를 로드하여 다양한 소스의 패키지 검증
- **유연한 정책**: 검증 실패 시 차단 또는 경고만 수행하도록 설정 가능
- **검증 로깅**: 모든 검증 과정과 결과를 상세하게 로깅
- **API 지원**: RESTful API를 통한 검증 상태 확인 및 수동 검증

## 설정 방법

### 1. 기본 설정

`apk-proxy.yaml` 파일에서 서명 검증을 설정합니다:

```yaml
# APK 서명 검증 설정
verification:
  # 서명 검증 활성화 여부
  enabled: true
  
  # 신뢰할 수 있는 공개키 디렉토리
  key_directory: "/etc/apk/keys"
  
  # 서명 검증 실패 시 요청 차단 여부
  fail_on_invalid: false
  
  # 검증된 패키지만 캐시 여부
  cache_validated: false
```

### 2. 설정 옵션 설명

| 옵션 | 설명 | 기본값 |
|------|------|---------|
| `enabled` | 서명 검증 활성화 여부 | `false` |
| `key_directory` | 신뢰할 수 있는 공개키가 저장된 디렉토리 | `/etc/apk/keys` |
| `fail_on_invalid` | 서명 검증 실패 시 요청 차단 여부 | `false` |
| `cache_validated` | 검증된 패키지만 캐시에 저장할지 여부 | `false` |

### 3. 검증 정책

#### 관대한 정책 (기본값)
```yaml
verification:
  enabled: true
  fail_on_invalid: false
```
- 서명 검증을 수행하지만 실패해도 패키지 제공
- 검증 실패는 로그에 경고로 기록

#### 엄격한 정책
```yaml
verification:
  enabled: true
  fail_on_invalid: true
```
- 서명 검증 실패 시 패키지 요청 차단
- 높은 보안이 필요한 환경에 적합

## 신뢰할 수 있는 키 관리

### 1. Alpine Linux 공식 키 설치

Alpine Linux 시스템에서 APK 키를 복사:

```bash
# Alpine Linux에서 키 복사
sudo mkdir -p /etc/apk/keys
sudo cp /etc/apk/keys/* /path/to/proxynd/keys/

# 또는 ProxyND 서버에 직접 다운로드
sudo mkdir -p /etc/apk/keys
cd /etc/apk/keys
sudo wget https://alpinelinux.org/keys/alpine-devel@lists.alpinelinux.org-*.rsa.pub
```

### 2. 키 디렉토리 구조

```
/etc/apk/keys/
├── alpine-devel@lists.alpinelinux.org-4a6a0840.rsa.pub
├── alpine-devel@lists.alpinelinux.org-5243ef4b.rsa.pub
└── alpine-devel@lists.alpinelinux.org-5261cecb.rsa.pub
```

### 3. 지원되는 키 형식

- **PEM 형식**: `.pem` 확장자
- **Alpine 공개키**: `.pub` 확장자
- **PKCS#1 형식**: RSA 공개키
- **PKIX 형식**: 일반적인 공개키

### 4. 키 검증

키가 올바르게 로드되었는지 확인:

```bash
# API를 통한 키 상태 확인
curl http://localhost:8080/api/apk/verification/status

# 신뢰할 수 있는 키 목록 조회
curl http://localhost:8080/api/apk/verification/keys
```

## 검증 프로세스

### 1. 서명 파일 탐색

APK 파일에 대해 다음 패턴으로 서명 파일을 찾습니다:

- `{package}.apk.SIGN.RSA.*`
- `{package}.rsa`
- `*.SIGN.RSA.*` (동일 디렉토리)

### 2. 서명 검증 단계

1. **APK 파일 해시 계산**: SHA256 해시 생성
2. **서명 파일 읽기**: RSA 서명 데이터 로드
3. **키 매칭**: 로드된 신뢰할 수 있는 키들로 검증 시도
4. **결과 처리**: 검증 결과에 따른 후속 처리

### 3. 검증 결과

```json
{
  "is_valid": true,
  "signature_file": "/path/to/package.apk.SIGN.RSA.alpine",
  "key_fingerprint": "abcd1234efgh5678",
  "details": [
    "서명 파일 발견: package.apk.SIGN.RSA.alpine",
    "APK 해시: a1b2c3d4...",
    "서명 검증 성공 (키: abcd1234efgh5678)"
  ]
}
```

## API 엔드포인트

### 1. 검증 상태 조회

```bash
GET /api/apk/verification/status
```

**응답 예시:**
```json
{
  "enabled": true,
  "key_directory": "/etc/apk/keys",
  "key_dir_exists": true,
  "trusted_key_count": 3,
  "fail_on_invalid": false,
  "cache_validated": false,
  "key_load_error": null,
  "status": "active"
}
```

### 2. 특정 파일 검증

```bash
POST /api/apk/verification/verify
Content-Type: application/json

{
  "file_path": "proxy/apk/v3.18/main/x86_64/curl-8.1.2-r0.apk"
}
```

**응답 예시:**
```json
{
  "file_path": "proxy/apk/v3.18/main/x86_64/curl-8.1.2-r0.apk",
  "verification": {
    "is_valid": true,
    "signature_file": "/storage/proxy/apk/v3.18/main/x86_64/curl-8.1.2-r0.apk.SIGN.RSA.alpine",
    "key_fingerprint": "abcd1234efgh5678",
    "details": ["검증 성공"]
  },
  "trusted_keys": 3
}
```

### 3. 신뢰할 수 있는 키 목록

```bash
GET /api/apk/verification/keys
```

**응답 예시:**
```json
{
  "key_directory": "/etc/apk/keys",
  "key_count": 3,
  "keys": [
    "abcd1234efgh5678",
    "1234567890abcdef", 
    "fedcba0987654321"
  ]
}
```

### 4. 검증 설정 조회

```bash
GET /api/apk/verification/config
```

**응답 예시:**
```json
{
  "verification": {
    "enabled": true,
    "key_directory": "/etc/apk/keys",
    "fail_on_invalid": false,
    "cache_validated": false
  }
}
```

## 로깅

### 1. 검증 성공 로그

```
INFO  APK 서명 검증 성공 file=/storage/proxy/apk/package.apk key=abcd1234
```

### 2. 검증 실패 로그

```
WARN  APK 서명 검증 실패 file=/storage/proxy/apk/package.apk error="서명 검증 실패: 신뢰할 수 있는 키로 검증되지 않음"
```

### 3. 키 로드 로그

```
INFO  신뢰할 수 있는 키 로드 완료 directory=/etc/apk/keys count=3
```

## 문제 해결

### 1. 일반적인 문제

#### 키가 로드되지 않음

**증상:**
```json
{
  "trusted_key_count": 0,
  "key_load_error": "키 디렉토리 탐색 실패"
}
```

**해결 방법:**
1. 키 디렉토리 경로 확인
2. 디렉토리 권한 확인 (`chmod 755`)
3. 키 파일 형식 확인

#### 서명 검증 항상 실패

**증상:**
```
WARN  APK 서명 검증 실패 error="서명 검증 실패: 신뢰할 수 있는 키로 검증되지 않음"
```

**해결 방법:**
1. 올바른 Alpine 키 설치 확인
2. 키 파일 형식 및 권한 확인
3. APK 패키지 출처 확인

### 2. 디버깅

#### 상세 로그 활성화

```yaml
# logging 설정에서 DEBUG 레벨 활성화
logging:
  level: debug
```

#### 검증 상태 확인

```bash
# 전체 상태 확인
curl -s http://localhost:8080/api/apk/verification/status | jq

# 특정 파일 수동 검증
curl -X POST http://localhost:8080/api/apk/verification/verify \
  -H "Content-Type: application/json" \
  -d '{"file_path": "proxy/apk/package.apk"}' | jq
```

#### 키 디렉토리 확인

```bash
# 키 파일 목록 확인
ls -la /etc/apk/keys/

# 키 파일 내용 확인
file /etc/apk/keys/*
```

### 3. 성능 고려사항

- **키 캐싱**: 키는 첫 로드 시에만 읽고 메모리에 캐시됨
- **서명 파일**: 큰 APK 파일의 해시 계산은 CPU 집약적
- **동시성**: 여러 요청이 동시에 검증을 수행할 수 있음

### 4. 보안 고려사항

- **키 무결성**: 신뢰할 수 있는 소스에서만 키 다운로드
- **정기적 갱신**: Alpine 키는 정기적으로 갱신됨
- **접근 제어**: 키 디렉토리에 대한 적절한 권한 설정

## 관련 문서

- [APK 클라이언트 설정 가이드](APK-CLIENT-SETUP.md)
- [APK 프록시 요구사항](APK-PROXY-REQUIREMENTS.md)
- [패키지 검증 가이드](PACKAGE_VERIFICATION.md)

## 참고 자료

- [Alpine Linux APK 문서](https://wiki.alpinelinux.org/wiki/Package_management)
- [Alpine Linux 보안 키](https://alpinelinux.org/keys/)
- [RSA 서명 검증](https://pkg.go.dev/crypto/rsa)