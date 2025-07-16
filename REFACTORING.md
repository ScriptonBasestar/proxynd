# 🔧 REFACTORING.md - ProxyND

> 이 문서는 ProxyND 프로젝트의 리팩토링 계획과 진행 상황을 기록합니다. 코드 품질 향상과 유지보수성 개선을 목표로 합니다.

---

## 📌 1. 프로젝트 현황 분석

### 현재 아키텍처
ProxyND는 Go 기반의 패키지 매니저 프록시 서버로, Maven과 APT를 지원하며 다음과 같은 구조를 가지고 있습니다:

```
proxynd/
├── internal/          # 애플리케이션 핵심 (app, services, domain, auth, webhook)
├── configs/           # 프록시별 설정 구조체
├── handlers/          # HTTP 요청 핸들러
├── middlewares/       # Fiber 미들웨어
├── cache/            # 캐싱 시스템
├── alerts/           # 알림 시스템
└── verification/     # 패키지 검증
```

### 식별된 문제점

1. **의존성 관리**: 핸들러에서 `ReadConfig()` 반복 호출 (30+ 곳)
2. **에러 처리**: 507개의 `if err != nil` 패턴, 에러 타입 부족
3. **코드 중복**: 프록시 타입별 유사한 패턴 반복
4. **테스트 커버리지**: 핸들러 0%, 전체 평균 30% 미만
5. **미들웨어 분산**: 15개 파일로 분산된 미들웨어
6. **TODO 누적**: 218개의 TODO/FIXME 코멘트

---

## 🎯 2. 리팩토링 목표 및 원칙

### 목표
- **유지보수성**: 코드 중복 제거, 명확한 책임 분리
- **테스트 가능성**: 의존성 주입, 인터페이스 기반 설계
- **성능**: 불필요한 I/O 제거, 효율적인 캐싱
- **안정성**: 체계적인 에러 처리, 타입 안전성

### 원칙
- **점진적 개선**: 기능 유지하며 단계별 적용
- **하위 호환성**: 기존 API 유지
- **테스트 우선**: 리팩토링 전 테스트 작성
- **문서화**: 변경사항 즉시 문서화

---

## 🔨 3. 주요 개선 영역

### 3.1 의존성 주입 및 설정 관리

#### 현재 상태 (Before)
```go
// handlers/proxy/apt_handler.go
func HandleAPTRequest(c *fiber.Ctx) error {
    config, err := configs.ReadConfig() // 매번 파일 읽기
    if err != nil {
        return c.Status(500).SendString("설정 읽기 실패")
    }
    // ... 핸들러 로직
}
```

#### 개선 방안 (After)
```go
// internal/app/container.go 확장
type Container struct {
    config     *configs.Config
    configOnce sync.Once
    // ... 기타 서비스
}

func (c *Container) Config() *configs.Config {
    c.configOnce.Do(func() {
        c.config = c.loadConfig()
        c.watchConfigChanges() // 핫 리로드
    })
    return c.config
}

// handlers/proxy/apt_handler.go
type APTHandler struct {
    container *app.Container
}

func NewAPTHandler(container *app.Container) *APTHandler {
    return &APTHandler{container: container}
}

func (h *APTHandler) Handle(c *fiber.Ctx) error {
    config := h.container.Config() // 캐시된 설정 사용
    // ... 핸들러 로직
}
```

### 3.2 에러 처리 표준화

#### 현재 상태 (Before)
```go
if err != nil {
    log.Printf("에러: %v", err)
    return c.Status(500).SendString("내부 서버 오류")
}
```

#### 개선 방안 (After)
```go
// internal/errors/domain_errors.go
type DomainError struct {
    Code    string
    Message string
    Domain  string
    Cause   error
}

// APT 도메인 에러
var (
    ErrAPTPackageNotFound = &DomainError{
        Code:    "APT001",
        Message: "패키지를 찾을 수 없습니다",
        Domain:  "apt",
    }
)

// middlewares/error_handler.go
func ErrorHandler() fiber.Handler {
    return func(c *fiber.Ctx) error {
        err := c.Next()
        if err != nil {
            switch e := err.(type) {
            case *DomainError:
                return c.Status(400).JSON(fiber.Map{
                    "error": e.Code,
                    "message": e.Message,
                    "domain": e.Domain,
                })
            default:
                // 기본 에러 처리
            }
        }
        return nil
    }
}
```

### 3.3 코드 중복 제거

#### 현재 상태 (Before)
```go
// APT, Maven, NPM 핸들러가 각각 유사한 캐시 로직 구현
func HandleAPTRequest(c *fiber.Ctx) error {
    // 캐시 확인
    if cached := checkCache(key); cached != nil {
        return c.Send(cached)
    }
    // 원본 서버 요청
    resp := fetchFromUpstream(url)
    // 캐시 저장
    saveToCache(key, resp)
    return c.Send(resp)
}
```

#### 개선 방안 (After)
```go
// internal/handlers/base_proxy_handler.go
type BaseProxyHandler interface {
    GetCacheKey(*fiber.Ctx) string
    GetUpstreamURL(*fiber.Ctx) string
    TransformResponse([]byte) []byte
}

type ProxyHandler struct {
    Base  BaseProxyHandler
    Cache cache.Cache
}

func (h *ProxyHandler) Handle(c *fiber.Ctx) error {
    key := h.Base.GetCacheKey(c)

    // 공통 캐시 로직
    if cached, err := h.Cache.Get(key); err == nil {
        return c.Send(cached)
    }

    // 공통 프록시 로직
    url := h.Base.GetUpstreamURL(c)
    resp, err := h.fetchFromUpstream(url)
    if err != nil {
        return err
    }

    transformed := h.Base.TransformResponse(resp)
    h.Cache.Set(key, transformed)

    return c.Send(transformed)
}

// APT 구현체
type APTProxyHandler struct{}

func (a *APTProxyHandler) GetCacheKey(c *fiber.Ctx) string {
    return fmt.Sprintf("apt:%s", c.Path())
}
// ... 나머지 메서드 구현
```

### 3.4 테스트 커버리지 향상

#### 테스트 전략
```go
// handlers/proxy/apt_handler_test.go
func TestAPTHandler_Handle(t *testing.T) {
    // Given: Mock 설정
    mockContainer := mocks.NewMockContainer(t)
    mockConfig := &configs.Config{
        APTProxy: configs.APTProxyConfig{
            Enabled: true,
            Mirrors: []string{"http://mirror.example.com"},
        },
    }
    mockContainer.On("Config").Return(mockConfig)

    // When: 핸들러 실행
    handler := NewAPTHandler(mockContainer)
    app := fiber.New()
    app.Get("/apt/*", handler.Handle)

    req := httptest.NewRequest("GET", "/apt/dists/focal/Release", nil)
    resp, err := app.Test(req)

    // Then: 검증
    assert.NoError(t, err)
    assert.Equal(t, 200, resp.StatusCode)
    mockContainer.AssertExpectations(t)
}
```

### 3.5 미들웨어 통합

#### 현재 상태 (Before)
```go
// 15개의 개별 미들웨어 파일
app.Use(SecurityMiddleware())
app.Use(SecurityHeadersMiddleware())
app.Use(AccessLogMiddleware())
app.Use(StructuredAccessLogMiddleware())
// ...
```

#### 개선 방안 (After)
```go
// middlewares/chain.go
type MiddlewareChain struct {
    middlewares []fiber.Handler
}

func NewSecurityChain() *MiddlewareChain {
    return &MiddlewareChain{
        middlewares: []fiber.Handler{
            helmet.New(),
            cors.New(),
            RateLimiter(),
            IPFilter(),
        },
    }
}

func NewLoggingChain() *MiddlewareChain {
    return &MiddlewareChain{
        middlewares: []fiber.Handler{
            StructuredLogger(),
            RequestID(),
            ResponseTime(),
        },
    }
}

// main.go
app.Use(NewSecurityChain().Build()...)
app.Use(NewLoggingChain().Build()...)
```

---

## 📅 4. 실행 로드맵

### Phase 1: 기반 구축 (1-2주)
- [ ] Container 패턴 확장 및 설정 관리 중앙화
- [ ] 에러 타입 정의 및 핸들러 구현
- [ ] 기본 테스트 인프라 구축

### Phase 2: 핵심 리팩토링 (2-3주)
- [ ] BaseProxyHandler 구현 및 적용
- [ ] 공통 캐시 레이어 추출
- [ ] 미들웨어 통합

### Phase 3: 품질 개선 (2-3주)
- [ ] 단위 테스트 작성 (목표: 70%+)
- [ ] 통합 테스트 시나리오 확대
- [ ] 성능 프로파일링 및 최적화

### Phase 4: 마무리 (1주)
- [ ] TODO/FIXME 정리
- [ ] 문서 업데이트
- [ ] 최종 검증

---

## 🚨 5. 위험 관리

### 위험 요소
1. **기능 손상**: 리팩토링 중 기존 기능 영향
2. **성능 저하**: 추상화로 인한 오버헤드
3. **호환성**: API 변경으로 인한 클라이언트 영향

### 완화 전략
- **Feature Flag**: 새 기능 점진적 활성화
- **A/B 테스트**: 성능 비교 검증
- **Canary 배포**: 단계적 롤아웃
- **롤백 계획**: 각 단계별 되돌리기 가능

---

## 📊 6. 성공 지표

### 정량적 지표
- 테스트 커버리지: 30% → 70%+
- 코드 중복: 30% 감소
- 빌드 시간: 20% 단축
- 메모리 사용량: 15% 감소

### 정성적 지표
- 신규 프록시 추가 시간: 1일 → 2시간
- 버그 수정 시간: 평균 50% 단축
- 코드 리뷰 시간: 30% 감소

---

## 🔄 7. 지속적 개선

### 코드 리뷰 체크리스트
- [ ] 의존성 주입 패턴 준수
- [ ] 에러 처리 표준 따름
- [ ] 테스트 커버리지 80%+
- [ ] 문서화 완료

### 모니터링
- Prometheus 메트릭 추가
- 에러 발생률 추적
- 성능 지표 대시보드

---

## 👥 참여자

- 리팩토링 주도: ProxyND 팀
- 검토: 아키텍처 팀
- 승인: 기술 리더십

---

> 📅 최종 업데이트: 2025-01-16
>
> 이 문서는 지속적으로 업데이트됩니다. 변경사항은 PR을 통해 반영해주세요.
