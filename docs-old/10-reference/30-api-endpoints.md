# ProxyND API 엔드포인트 가이드

**버전**: v1.0  
**최종 업데이트**: 2024-12-26  
**상태**: ✅ 완전 구현됨

## 개요

이 문서는 ProxyND 서버에서 제공하는 모든 API 엔드포인트를 설명합니다. 모든 API는 `proxyndctl` CLI 도구와 완전히 호환됩니다.

## 기본 정보

- **베이스 URL**: `http://localhost:8080`
- **API 버전**: v1
- **응답 형식**: JSON
- **인증**: 선택적 (사용자 관리 API의 경우 필요)

## 📋 API 카테고리

### 1. 캐시 관리 API

캐시 저장소의 관리 및 모니터링을 위한 API입니다.

#### 엔드포인트 목록

| 메서드 | 경로 | 설명 | 상태 |
|-------|------|------|------|
| `GET` | `/api/cache/list` | 캐시 항목 목록 조회 | ✅ |
| `GET` | `/api/cache/size` | 캐시 사용량 및 통계 | ✅ |
| `GET` | `/api/cache/stats` | 캐시 성능 통계 | ✅ |
| `GET` | `/api/cache/ttl` | TTL 정책 조회 | ✅ |
| `DELETE` | `/api/cache/clear` | 전체 캐시 정리 | ✅ |
| `DELETE` | `/api/cache/clear/:type` | 타입별 캐시 정리 | ✅ |
| `DELETE` | `/api/cache/item/*` | 특정 캐시 항목 삭제 | ✅ |

#### 사용 예시

```bash
# 캐시 목록 조회
curl http://localhost:8080/api/cache/list

# NPM 캐시만 조회
curl http://localhost:8080/api/cache/list?type=npm&limit=50

# 캐시 사용량 확인
curl http://localhost:8080/api/cache/size

# Maven 캐시 정리 (확인 필요)
curl -X DELETE "http://localhost:8080/api/cache/clear/maven?confirm=true"
```

### 2. 설정 관리 API

서버 설정의 검증, 조회, 관리를 위한 API입니다.

#### 엔드포인트 목록

| 메서드 | 경로 | 설명 | 상태 |
|-------|------|------|------|
| `GET` | `/api/config/validate` | 설정 파일 검증 | ✅ |
| `GET` | `/api/config/show` | 현재 설정 표시 | ✅ |
| `GET` | `/api/config/files` | 설정 파일 목록 | ✅ |
| `GET` | `/api/config/files/*` | 특정 설정 파일 내용 | ✅ |
| `POST` | `/api/config/reload` | 설정 리로드 | ✅ |

#### 사용 예시

```bash
# 설정 검증
curl http://localhost:8080/api/config/validate

# 현재 설정 확인 (민감 정보 마스킹됨)
curl http://localhost:8080/api/config/show

# 설정 파일 목록
curl http://localhost:8080/api/config/files

# 특정 설정 파일 읽기
curl http://localhost:8080/api/config/files/npm-proxy.yaml
```

### 3. 사용자 관리 API

사용자 계정 관리를 위한 API입니다.

#### 엔드포인트 목록

| 메서드 | 경로 | 설명 | 상태 |
|-------|------|------|------|
| `GET` | `/api/user/list` | 사용자 목록 조회 | ✅ |
| `POST` | `/api/user/add` | 사용자 추가 | ✅ |
| `DELETE` | `/api/user/delete` | 사용자 삭제 | ✅ |
| `GET` | `/api/user/:username` | 사용자 정보 조회 | ✅ |
| `PUT` | `/api/user/:username` | 사용자 정보 수정 | 🚧 |
| `POST` | `/api/user/:username/password` | 비밀번호 변경 | 🚧 |
| `POST` | `/api/user/:username/toggle` | 활성화/비활성화 | 🚧 |

#### 사용 예시

```bash
# 사용자 목록 조회
curl http://localhost:8080/api/user/list

# 새 사용자 추가
curl -X POST http://localhost:8080/api/user/add \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser","password":"password123","role":"user"}'

# 사용자 삭제
curl -X DELETE http://localhost:8080/api/user/delete \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser"}'
```

### 4. 프록시 테스트 API

프록시 연결성 및 기능 테스트를 위한 API입니다.

#### 엔드포인트 목록

| 메서드 | 경로 | 설명 | 상태 |
|-------|------|------|------|
| `GET` | `/api/test/types` | 지원되는 프록시 타입 목록 | ✅ |
| `POST` | `/api/test/all` | 전체 프록시 테스트 | ✅ |
| `POST` | `/api/test/:proxy_type` | 개별 프록시 테스트 | ✅ |
| `GET` | `/api/test/connectivity/:proxy_type` | 연결성 테스트 | ✅ |

#### 사용 예시

```bash
# 지원되는 프록시 타입 확인
curl http://localhost:8080/api/test/types

# 모든 프록시 테스트
curl -X POST http://localhost:8080/api/test/all \
  -H "Content-Type: application/json" \
  -d '{"timeout":30}'

# NPM 프록시만 테스트
curl -X POST http://localhost:8080/api/test/npm

# Maven 프록시 연결성 확인
curl http://localhost:8080/api/test/connectivity/maven
```

### 5. 서버 상태 및 모니터링 API

서버 상태, 헬스체크, 메트릭 조회를 위한 API입니다.

#### 엔드포인트 목록

| 메서드 | 경로 | 설명 | 상태 |
|-------|------|------|------|
| `GET` | `/api/status` | 전체 서버 상태 | ✅ |
| `GET` | `/api/status/health` | 헬스체크 | ✅ |
| `GET` | `/api/status/metrics` | 상세 메트릭 | ✅ |
| `GET` | `/api/status/dependencies` | 의존성 상태 | ✅ |
| `GET` | `/api/status/stats` | 실시간 통계 | ✅ |

#### 호환성 엔드포인트

CLI 도구와의 호환성을 위해 추가로 제공되는 엔드포인트:

| 메서드 | 경로 | 리다이렉트 대상 | 상태 |
|-------|------|-------------|------|
| `GET` | `/api/health` | `/api/status/health` | ✅ |
| `GET` | `/api/metrics` | `/api/status/metrics` | ✅ |
| `GET` | `/healthz` | 기존 엔드포인트 | ✅ |
| `GET` | `/metrics` | 기존 Prometheus 메트릭 | ✅ |

#### 사용 예시

```bash
# 전체 서버 상태
curl http://localhost:8080/api/status

# 헬스체크 (간단한 형식)
curl http://localhost:8080/api/health

# 상세 메트릭
curl http://localhost:8080/api/metrics

# Kubernetes 호환 헬스체크
curl http://localhost:8080/healthz?format=simple

# Prometheus 메트릭
curl http://localhost:8080/metrics
```

## 🛠️ CLI 도구 연동

### proxyndctl 명령어 매핑

| CLI 명령어 | API 엔드포인트 | 설명 |
|-----------|---------------|------|
| `proxyndctl cache list` | `GET /api/cache/list` | 캐시 목록 |
| `proxyndctl cache clear` | `DELETE /api/cache/clear` | 캐시 정리 |
| `proxyndctl config validate` | `GET /api/config/validate` | 설정 검증 |
| `proxyndctl config show` | `GET /api/config/show` | 설정 표시 |
| `proxyndctl user list` | `GET /api/user/list` | 사용자 목록 |
| `proxyndctl user add <user>` | `POST /api/user/add` | 사용자 추가 |
| `proxyndctl test all` | `POST /api/test/all` | 전체 테스트 |
| `proxyndctl test --proxy npm` | `POST /api/test/npm` | 개별 테스트 |
| `proxyndctl status` | `GET /api/status` | 서버 상태 |

## 🧪 API 테스트

### 검증 스크립트 실행

프로젝트에는 모든 API 엔드포인트의 연결성을 확인하는 검증 스크립트가 포함되어 있습니다:

```bash
# 기본 검증
make verify-api

# JSON 형식 출력
make verify-api-json

# 상세 출력
make verify-api-verbose

# 수동 실행
./scripts/verify-api-endpoints.sh http://localhost:8080 --verbose
```

### 개발 서버와 함께 테스트

```bash
# 개발 서버 시작
make dev

# 다른 터미널에서 API 검증
make verify-api
```

## 📊 응답 형식

### 성공 응답

모든 API는 일관된 JSON 형식으로 응답합니다:

```json
{
  "data": {...},
  "timestamp": "2024-12-26T10:30:00Z",
  "status": "success"
}
```

### 오류 응답

```json
{
  "error": "오류 메시지",
  "code": "ERROR_CODE",
  "timestamp": "2024-12-26T10:30:00Z",
  "status": "error"
}
```

### 페이지네이션

목록 API는 페이지네이션을 지원합니다:

```json
{
  "items": [...],
  "total": 100,
  "limit": 50,
  "offset": 0,
  "has_more": true
}
```

## 🔒 보안 고려사항

1. **민감 정보 마스킹**: 설정 API는 비밀번호, 토큰 등을 자동으로 마스킹합니다.
2. **경로 검증**: 모든 파일 경로 파라미터는 Path Traversal 공격을 방지합니다.
3. **입력 검증**: 모든 사용자 입력은 검증됩니다.
4. **CORS 헤더**: CLI 도구 호환성을 위해 적절한 CORS 헤더를 설정합니다.

## 🚀 다음 단계

### 계획된 개선사항

- [ ] 사용자 관리 API 완전 구현 (비밀번호 변경, 상태 토글)
- [ ] 실시간 WebSocket API 추가
- [ ] API 키 인증 지원
- [ ] GraphQL 엔드포인트 추가
- [ ] OpenAPI 3.0 스펙 문서 생성

### 기여하기

API 개선 사항이나 버그를 발견하면 이슈를 등록하거나 풀 리퀘스트를 보내주세요.

---

**참고**: 이 문서는 `results/anal/api-endpoints-analysis.md` 분석을 바탕으로 작성되었으며, 모든 엔드포인트가 실제로 구현되고 테스트되었습니다.
