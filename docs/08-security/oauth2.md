# OAuth2 Authentication Design Document

## 개요

ProxyND는 OAuth2 프로토콜을 통한 인증 시스템을 지원하여 기업 환경에서 기존 ID 제공자(Identity Provider)와 통합할 수 있습니다. 이 문서는 OAuth2 인증 플로우의 설계와 구현 방법을 설명합니다.

## 지원하는 OAuth2 플로우

### 1. Authorization Code Flow (웹 애플리케이션용)

**사용 사례**: 대시보드, 관리 웹 인터페이스 접근

**플로우**:
1. 사용자가 ProxyND 대시보드 접근 시도
2. 로그인하지 않은 경우 OAuth2 제공자의 인증 페이지로 리다이렉트
3. 사용자가 ID 제공자에서 인증 완료
4. Authorization Code를 받아 Access Token으로 교환
5. 사용자 정보 조회 후 세션 생성

```
[사용자] -> [ProxyND] -> [OAuth2 Provider] -> [사용자]
    |           |              |                 |
    |           |              |                 |
    v           v              v                 v
로그인 요청 -> 인증 페이지 -> 사용자 인증 -> Authorization Code
    |           |                              |
    |           v                              v
    |    Access Token 교환 <- Authorization Code
    |           |
    v           v
세션 생성 <- 사용자 정보 조회
```

### 2. Client Credentials Flow (서비스간 통신용)

**사용 사례**: CI/CD 파이프라인, API 클라이언트, 자동화 도구

**플로우**:
1. 클라이언트가 Client ID와 Client Secret으로 직접 인증
2. Access Token 발급
3. API 요청 시 Bearer Token으로 인증

```
[API 클라이언트] -> [ProxyND] -> [OAuth2 Provider]
        |              |              |
        |              |              |
        v              v              v
Client Credentials -> Token 요청 -> Access Token
        |                             |
        |                             v
        v                      API 요청 with Token
    패키지 다운로드 <-------------- 인증 성공
```

## 지원하는 OAuth2 제공자

### 1. GitHub OAuth2
- **Authorization URL**: `https://github.com/login/oauth/authorize`
- **Token URL**: `https://github.com/login/oauth/access_token`
- **User Info URL**: `https://api.github.com/user`
- **Scopes**: `user:email`

### 2. GitLab OAuth2
- **Authorization URL**: `https://gitlab.com/oauth/authorize`
- **Token URL**: `https://gitlab.com/oauth/token`
- **User Info URL**: `https://gitlab.com/api/v4/user`
- **Scopes**: `read_user`

### 3. Google OAuth2
- **Authorization URL**: `https://accounts.google.com/o/oauth2/v2/auth`
- **Token URL**: `https://oauth2.googleapis.com/token`
- **User Info URL**: `https://www.googleapis.com/oauth2/v2/userinfo`
- **Scopes**: `openid email profile`

### 4. Generic OAuth2
- 커스텀 OAuth2 제공자 지원 (예: Azure AD, Okta 등)
- 설정 가능한 엔드포인트와 스코프

## 설정 구조

```yaml
# global.yaml
oauth2:
  enabled: true
  default_provider: github

  providers:
    github:
      client_id: "${GITHUB_CLIENT_ID}"
      client_secret: "${GITHUB_CLIENT_SECRET}"
      redirect_uri: "https://proxynd.example.com/auth/callback/github"
      scopes: ["user:email"]

    gitlab:
      client_id: "${GITLAB_CLIENT_ID}"
      client_secret: "${GITLAB_CLIENT_SECRET}"
      redirect_uri: "https://proxynd.example.com/auth/callback/gitlab"
      scopes: ["read_user"]

    google:
      client_id: "${GOOGLE_CLIENT_ID}"
      client_secret: "${GOOGLE_CLIENT_SECRET}"
      redirect_uri: "https://proxynd.example.com/auth/callback/google"
      scopes: ["openid", "email", "profile"]

  # JWT 토큰 설정
  jwt:
    secret: "${JWT_SECRET}"
    access_token_ttl: 3600    # 1시간
    refresh_token_ttl: 604800 # 7일

  # 사용자 매핑 설정
  user_mapping:
    auto_create: true
    default_role: "viewer"
    admin_users: ["admin@example.com"]
    admin_organizations: ["my-org"]
```

## 인증 엔드포인트

### 1. 로그인 시작
```
GET /auth/login/:provider
```
- 지정된 OAuth2 제공자의 인증 페이지로 리다이렉트

### 2. 콜백 처리
```
GET /auth/callback/:provider?code=xxx&state=xxx
```
- Authorization Code를 Access Token으로 교환
- 사용자 정보 조회 후 JWT 토큰 발급

### 3. 토큰 갱신
```
POST /auth/refresh
Authorization: Bearer <refresh_token>
```
- Refresh Token으로 새로운 Access Token 발급

### 4. 로그아웃
```
POST /auth/logout
Authorization: Bearer <access_token>
```
- 토큰 무효화 및 세션 정리

## JWT 토큰 구조

### Access Token Claims
```json
{
  "sub": "user123",
  "email": "user@example.com",
  "name": "User Name",
  "provider": "github",
  "roles": ["viewer"],
  "iat": 1234567890,
  "exp": 1234571490,
  "iss": "proxynd",
  "aud": "proxynd-api"
}
```

### Refresh Token Claims
```json
{
  "sub": "user123",
  "type": "refresh",
  "iat": 1234567890,
  "exp": 1235172690,
  "iss": "proxynd",
  "aud": "proxynd-api"
}
```

## 권한 매핑

### 역할 기반 접근 제어
- **admin**: 전체 관리 권한
- **maintainer**: 캐시 관리, 설정 조회
- **developer**: 패키지 업로드, 다운로드
- **viewer**: 읽기 전용 접근

### 자동 권한 할당 규칙
1. **이메일 기반**: 특정 이메일 주소에 admin 권한 부여
2. **조직 기반**: GitHub/GitLab 조직 멤버십으로 권한 결정
3. **기본 권한**: 신규 사용자에게 viewer 권한 부여

## 보안 고려사항

### 1. PKCE (Proof Key for Code Exchange)
- Authorization Code Flow에서 코드 가로채기 공격 방지
- Code Verifier와 Code Challenge 사용

### 2. State Parameter
- CSRF 공격 방지를 위한 랜덤 state 값 검증

### 3. Secure Cookie
- 세션 쿠키는 HttpOnly, Secure, SameSite 속성 설정

### 4. Token 저장
- Access Token은 메모리나 HttpOnly 쿠키에 저장
- Refresh Token은 안전한 저장소에 보관

### 5. Token 검증
- JWT 서명 검증
- 만료 시간 확인
- 토큰 무효화 목록 확인

## 기존 인증과의 통합

### BasicAuth 호환성
- OAuth2와 BasicAuth 동시 지원
- 설정에 따라 우선순위 결정

### 미들웨어 체인
```
Request -> CORS -> Rate Limiting -> Auth Middleware -> Handler
                                         |
                                         v
                                   [OAuth2 JWT] or [BasicAuth]
```

### 인증 우선순위
1. Authorization 헤더의 Bearer 토큰 (OAuth2)
2. Authorization 헤더의 Basic 인증
3. 세션 쿠키 (웹 대시보드)

## 에러 처리

### OAuth2 에러 응답
```json
{
  "error": "invalid_grant",
  "error_description": "The provided authorization grant is invalid",
  "error_uri": "https://docs.example.com/oauth2/errors#invalid_grant"
}
```

### 일반적인 에러 시나리오
- **invalid_client**: Client ID/Secret 불일치
- **unauthorized_client**: 허용되지 않은 클라이언트
- **access_denied**: 사용자가 인증 거부
- **unsupported_response_type**: 지원하지 않는 응답 타입
- **invalid_scope**: 잘못된 스코프 요청

## 모니터링 및 로깅

### 메트릭
- OAuth2 인증 시도 수
- 성공/실패율
- 제공자별 통계
- 토큰 갱신 빈도

### 로그 이벤트
- 인증 성공/실패
- 토큰 발급/갱신
- 권한 승격 시도
- 의심스러운 활동

## 배포 고려사항

### 환경 변수
```bash
# OAuth2 클라이언트 자격증명
GITHUB_CLIENT_ID=xxx
GITHUB_CLIENT_SECRET=xxx
GITLAB_CLIENT_ID=xxx
GITLAB_CLIENT_SECRET=xxx
GOOGLE_CLIENT_ID=xxx
GOOGLE_CLIENT_SECRET=xxx

# JWT 서명 키
JWT_SECRET=your-super-secret-key

# Redis (토큰 저장소)
REDIS_URL=redis://localhost:6379
```

### HTTPS 필수
- OAuth2는 HTTPS 환경에서만 안전하게 동작
- 개발 환경에서도 HTTPS 인증서 사용 권장

### 콜백 URL 등록
- 각 OAuth2 제공자에서 콜백 URL 사전 등록 필요
- 예: `https://proxynd.example.com/auth/callback/github`

## 테스트 전략

### 단위 테스트
- JWT 토큰 생성/검증
- OAuth2 플로우 시뮬레이션
- 권한 매핑 로직

### 통합 테스트
- 실제 OAuth2 제공자와의 연동
- 토큰 라이프사이클 테스트
- 권한 기반 API 접근 테스트

### E2E 테스트
- 웹 브라우저를 통한 전체 인증 플로우
- 다양한 제공자에서의 인증 테스트

이 설계 문서를 바탕으로 ProxyND의 OAuth2 인증 시스템을 단계별로 구현할 예정입니다.
