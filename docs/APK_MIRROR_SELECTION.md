# Alpine APK 미러 자동 선택 가이드

이 문서는 ProxyND의 Alpine APK 프록시에서 지원하는 미러 자동 선택 기능에 대해 설명합니다.

## 목차

- [개요](#개요)
- [설정 방법](#설정-방법)
- [미러 선택 알고리즘](#미러-선택-알고리즘)
- [지역 감지](#지역-감지)
- [헬스체크](#헬스체크)
- [API 엔드포인트](#api-엔드포인트)
- [모니터링](#모니터링)
- [문제 해결](#문제-해결)

## 개요

Alpine APK 미러 자동 선택 기능은 클라이언트의 요청에 따라 최적의 미러를 자동으로 선택하여 패키지 다운로드 성능을 향상시킵니다.

### 주요 기능

- **지능적인 미러 선택**: Alpine 버전, 지역, 미러 상태를 고려한 최적 미러 선택
- **실시간 헬스체크**: 주기적인 미러 상태 모니터링 및 자동 페일오버
- **지역 기반 최적화**: 클라이언트 지역에 따른 미러 우선순위 조정
- **로드 밸런싱**: 응답 시간과 에러율을 고려한 지능적 로드 분산
- **자동 페일오버**: 장애 미러 자동 제외 및 복구 감지

## 설정 방법

### 1. 기본 설정

`apk-proxy.yaml` 파일에서 미러 자동 선택을 설정합니다:

```yaml
# APK 미러 자동 선택 설정
mirror_selection:
  # 미러 자동 선택 활성화 여부
  enabled: true
  
  # 헬스체크 간격 (예: "5m", "10m", "1h")
  health_check_interval: "5m"
  
  # 헬스체크 타임아웃 (예: "10s", "30s")
  health_check_timeout: "10s"
  
  # 선호하는 지역 목록 (우선순위 순)
  preferred_regions:
    - "korea"      # 한국
    - "japan"      # 일본
    - "asia"       # 아시아
    - "official"   # 공식
    - "global"     # 글로벌
  
  # 지역 미러가 모두 실패할 경우 글로벌 미러로 폴백
  fallback_to_global: true
  
  # 미러를 비활성화하기 전 최대 허용 에러 횟수
  max_error_count: 3
  
  # 지역 감지 모드
  region_detection_mode: "auto"
```

### 2. 설정 옵션 설명

| 옵션 | 설명 | 기본값 | 예시 |
|------|------|---------|------|
| `enabled` | 미러 자동 선택 활성화 여부 | `false` | `true` |
| `health_check_interval` | 헬스체크 수행 간격 | `5m` | `"10m"`, `"1h"` |
| `health_check_timeout` | 헬스체크 타임아웃 | `10s` | `"5s"`, `"30s"` |
| `preferred_regions` | 선호하는 지역 목록 (우선순위 순) | `[]` | `["korea", "asia"]` |
| `fallback_to_global` | 글로벌 미러로 폴백 여부 | `true` | `false` |
| `max_error_count` | 미러 비활성화 임계값 | `3` | `5` |
| `region_detection_mode` | 지역 감지 모드 | `auto` | `manual`, `disabled` |

### 3. 미러 설정 예시

```yaml
proxies:
  # 한국 미러들
  - name: kakao
    url: https://mirror.kakao.com/alpine
  - name: naver
    url: https://mirror.navercorp.com/alpine
  
  # 아시아 미러들
  - name: riken-japan
    url: https://ftp.riken.jp/alpine
  - name: singapore
    url: https://mirrors.nus.edu.sg/alpine
  
  # 공식 및 글로벌 미러들
  - name: alpine-official
    url: https://dl-cdn.alpinelinux.org/alpine
  - name: fastly
    url: https://alpine.global.ssl.fastly.net/alpine
```

## 미러 선택 알고리즘

### 1. 선택 기준

미러 선택은 다음 요소들을 종합적으로 고려합니다:

#### 우선순위 점수 계산
```
총점 = 기본점수(100) + 지역점수 + 응답시간점수 + 우선순위점수 - 에러점수 - 건강도점수
```

- **기본 점수**: 100점
- **지역 점수**: 0-30점 (클라이언트와 미러 지역 근접도)
- **응답 시간 점수**: 0-30점 (빠른 응답일수록 높은 점수)
- **우선순위 점수**: 0-20점 (설정된 우선순위)
- **에러 점수**: 에러 횟수 × 5점 (차감)
- **건강도 점수**: 비건강 미러 50점 차감

### 2. 선택 프로세스

1. **Alpine 버전 감지**: 요청 경로에서 Alpine 버전 추출
2. **건강한 미러 필터링**: 헬스체크를 통과한 미러만 선택
3. **점수 계산**: 각 미러의 종합 점수 계산
4. **순위 결정**: 점수 순으로 미러 정렬
5. **결과 반환**: 정렬된 미러 목록 반환

### 3. Alpine 버전 감지

요청 경로에서 Alpine 버전을 자동으로 감지합니다:

```
/proxy/apk/v3.18/main/x86_64/package.apk → v3.18
/proxy/apk/v3.19/community/aarch64/package.apk → v3.19
/proxy/apk/edge/testing/x86_64/package.apk → edge
```

## 지역 감지

### 1. 자동 지역 감지

URL 패턴을 분석하여 미러의 지역을 자동으로 감지합니다:

| 패턴 | 지역 | 예시 |
|------|------|------|
| `kakao.com`, `naver`, `.kr` | korea | `mirror.kakao.com` |
| `japan`, `.jp`, `riken` | japan | `ftp.riken.jp` |
| `china`, `.cn`, `tsinghua` | china | `mirrors.tuna.tsinghua.edu.cn` |
| `singapore`, `.sg`, `asia` | asia | `mirrors.nus.edu.sg` |
| `alpinelinux.org`, `dl-cdn` | official | `dl-cdn.alpinelinux.org` |
| `fastly`, `america`, `.us` | north_america | `alpine.global.ssl.fastly.net` |
| `europe`, `dotsrc`, `.dk` | europe | `mirrors.dotsrc.org` |

### 2. 지역별 우선순위

한국 클라이언트 기준 지역별 점수:

| 지역 | 점수 | 설명 |
|------|------|------|
| korea | 30 | 동일 지역 |
| japan, asia | 25 | 인접 지역 |
| china | 20 | 근접 지역 |
| official | 15 | 공식 미러 |
| global | 10 | 글로벌 미러 |
| 기타 | 5 | 원격 지역 |

## 헬스체크

### 1. 헬스체크 방식

각 미러에 대해 주기적으로 헬스체크를 수행합니다:

```http
HEAD /v3.19/main/x86_64/APKINDEX.tar.gz HTTP/1.1
Host: mirror.example.com
```

### 2. 헬스체크 기준

- **성공**: HTTP 200 응답
- **실패**: 타임아웃, 연결 오류, 4xx/5xx 응답
- **응답 시간**: 헤드 요청 완료까지의 시간
- **연속 실패**: 설정된 최대 에러 횟수 초과 시 비활성화

### 3. 자동 복구

비활성화된 미러도 계속 헬스체크하여 복구를 감지합니다:

```
에러 → 계속 헬스체크 → 성공 감지 → 자동 복구
```

## API 엔드포인트

### 1. 미러 상태 조회

```bash
GET /api/apk/mirror/status
```

**응답 예시:**
```json
{
  "enabled": true,
  "config": {
    "enabled": true,
    "health_check_interval": "5m",
    "preferred_regions": ["korea", "japan"]
  },
  "statistics": {
    "total_mirrors": 6,
    "healthy_mirrors": 5,
    "unhealthy_mirrors": 1,
    "health_rate": 83.33,
    "avg_response_time": "150ms"
  },
  "region_counts": {
    "korea": 2,
    "japan": 1,
    "asia": 1,
    "official": 1,
    "global": 1
  }
}
```

### 2. 미러 헬스 정보

```bash
GET /api/apk/mirror/health
```

**응답 예시:**
```json
{
  "healthy_mirrors": [
    {
      "name": "kakao",
      "url": "https://mirror.kakao.com/alpine",
      "region": "korea",
      "priority": 1,
      "response_time": "50ms",
      "last_check": "2024-01-15T10:30:00Z",
      "error_count": 0
    }
  ],
  "unhealthy_mirrors": [
    {
      "name": "slow-mirror",
      "url": "https://slow.mirror.com/alpine",
      "region": "global",
      "error_count": 5,
      "last_check": "2024-01-15T10:25:00Z"
    }
  ],
  "summary": {
    "total": 6,
    "healthy": 5,
    "unhealthy": 1
  }
}
```

### 3. 미러 선택 테스트

```bash
POST /api/apk/mirror/select
Content-Type: application/json

{
  "request_path": "/proxy/apk/v3.18/main/x86_64/curl-8.1.2-r0.apk"
}
```

**응답 예시:**
```json
{
  "request_path": "/proxy/apk/v3.18/main/x86_64/curl-8.1.2-r0.apk",
  "selected_mirrors": [
    {
      "rank": 1,
      "name": "kakao",
      "url": "https://mirror.kakao.com/alpine",
      "region": "korea",
      "healthy": true,
      "response_time": "50ms"
    },
    {
      "rank": 2,
      "name": "naver",
      "url": "https://mirror.navercorp.com/alpine",
      "region": "korea",
      "healthy": true,
      "response_time": "60ms"
    }
  ],
  "total_available": 6
}
```

### 4. 미러 설정 조회

```bash
GET /api/apk/mirror/config
```

**응답 예시:**
```json
{
  "mirror_selection": {
    "enabled": true,
    "health_check_interval": "5m",
    "health_check_timeout": "10s",
    "preferred_regions": ["korea", "japan", "asia"]
  },
  "proxies": [
    {
      "name": "kakao",
      "url": "https://mirror.kakao.com/alpine"
    }
  ]
}
```

## 모니터링

### 1. 로그 모니터링

미러 선택 관련 로그를 모니터링합니다:

```bash
# 미러 선택 로그
grep "미러 선택" /var/log/proxynd/proxynd.log

# 헬스체크 로그
grep "미러 헬스체크" /var/log/proxynd/proxynd.log

# 미러 에러 로그
grep "미러 에러" /var/log/proxynd/proxynd.log
```

### 2. 메트릭 수집

Prometheus 메트릭으로 미러 상태를 모니터링:

```
# 미러별 헬스 상태
proxynd_mirror_health{mirror="kakao",region="korea"} 1

# 미러별 응답 시간
proxynd_mirror_response_time{mirror="kakao"} 0.05

# 미러별 에러 카운트
proxynd_mirror_errors_total{mirror="slow-mirror"} 5
```

### 3. 알림 설정

미러 장애 시 알림 설정:

```yaml
# Alertmanager 룰 예시
- alert: MirrorDown
  expr: proxynd_mirror_health == 0
  for: 5m
  labels:
    severity: warning
  annotations:
    summary: "Alpine mirror {{ $labels.mirror }} is down"
```

## 성능 최적화

### 1. 헬스체크 최적화

```yaml
mirror_selection:
  # 헬스체크 간격 조정 (너무 빈번하면 부하 증가)
  health_check_interval: "10m"
  
  # 타임아웃 조정 (너무 길면 느린 미러 감지 지연)
  health_check_timeout: "5s"
  
  # 에러 허용 횟수 조정 (너무 낮으면 일시적 장애로 미러 제외)
  max_error_count: 5
```

### 2. 지역 설정 최적화

```yaml
mirror_selection:
  # 클라이언트 지역에 맞는 선호 지역 설정
  preferred_regions:
    - "korea"      # 1순위: 한국
    - "japan"      # 2순위: 일본  
    - "asia"       # 3순위: 아시아
    - "official"   # 4순위: 공식
```

### 3. 미러 순서 최적화

빠른 미러를 앞쪽에 배치:

```yaml
proxies:
  # 빠른 한국 미러들
  - name: kakao
    url: https://mirror.kakao.com/alpine
  - name: naver  
    url: https://mirror.navercorp.com/alpine
  
  # 백업 미러들
  - name: official
    url: https://dl-cdn.alpinelinux.org/alpine
```

## 문제 해결

### 1. 일반적인 문제

#### 미러 선택이 작동하지 않음

**증상:**
- 항상 같은 미러만 사용
- 장애 미러를 계속 시도

**해결 방법:**
1. 미러 선택 활성화 확인
```bash
curl http://localhost:8080/api/apk/mirror/status
```

2. 헬스체크 상태 확인
```bash
curl http://localhost:8080/api/apk/mirror/health
```

3. 설정 파일 검증
```yaml
mirror_selection:
  enabled: true  # 반드시 true로 설정
```

#### 헬스체크 실패

**증상:**
- 모든 미러가 unhealthy로 표시
- 헬스체크 에러 로그

**해결 방법:**
1. 네트워크 연결 확인
```bash
curl -I https://mirror.kakao.com/alpine/v3.19/main/x86_64/APKINDEX.tar.gz
```

2. 타임아웃 설정 확인
```yaml
mirror_selection:
  health_check_timeout: "30s"  # 타임아웃 증가
```

3. 방화벽 및 프록시 설정 확인

### 2. 성능 문제

#### 느린 미러 선택

**원인:**
- 헬스체크 오버헤드
- 잘못된 지역 감지

**해결 방법:**
1. 헬스체크 간격 조정
```yaml
mirror_selection:
  health_check_interval: "10m"  # 간격 증가
```

2. 선호 지역 설정 최적화
```yaml
mirror_selection:
  preferred_regions: ["korea"]  # 지역 제한
```

#### 미러 순환 문제

**증상:**
- 같은 점수의 미러들이 계속 순환
- 불필요한 미러 변경

**해결 방법:**
1. 우선순위 명확화
2. 응답 시간 차이 확대
3. 스티키 세션 구현 (향후 버전)

### 3. 디버깅

#### 상세 로깅 활성화

```yaml
logging:
  level: debug  # 디버그 로그 활성화
```

#### 미러 선택 테스트

```bash
# 특정 경로에 대한 미러 선택 테스트
curl -X POST http://localhost:8080/api/apk/mirror/select \
  -H "Content-Type: application/json" \
  -d '{"request_path": "/proxy/apk/v3.18/main/x86_64/test.apk"}' | jq
```

#### 헬스체크 상태 모니터링

```bash
# 실시간 헬스체크 모니터링
watch -n 30 'curl -s http://localhost:8080/api/apk/mirror/health | jq ".summary"'
```

## 관련 문서

- [APK 클라이언트 설정 가이드](APK-CLIENT-SETUP.md)
- [APK 서명 검증 가이드](APK_SIGNATURE_VERIFICATION.md)
- [헬스체크 시스템](HEALTH_CHECK.md)
- [메트릭 및 모니터링](METRICS.md)

## 참고 자료

- [Alpine Linux 미러 목록](https://mirrors.alpinelinux.org/)
- [Alpine Linux 패키지 저장소 구조](https://wiki.alpinelinux.org/wiki/Alpine_Package_Keeper)
- [로드 밸런싱 전략](https://en.wikipedia.org/wiki/Load_balancing_(computing))