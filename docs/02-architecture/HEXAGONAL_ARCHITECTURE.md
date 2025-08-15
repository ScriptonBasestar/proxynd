# Hexagonal Architecture Implementation

ProxyND의 헥사고널 아키텍처 (Ports and Adapters) 구현에 대한 상세 문서입니다.

## 🎯 아키텍처 목표

### 1. 관심사의 분리 (Separation of Concerns)
- **비즈니스 로직**과 **기술적 구현**을 명확히 분리
- 각 계층이 고유한 책임을 가지도록 설계
- 의존성 방향을 단방향으로 제한

### 2. 테스트 가능성 (Testability)
- 모든 외부 의존성을 인터페이스로 추상화
- 단위 테스트에서 쉽게 모킹 가능
- 통합 테스트와 단위 테스트의 명확한 구분

### 3. 유연성 (Flexibility)
- 웹 프레임워크 교체 용이성
- 새로운 패키지 매니저 타입 추가 용이성
- 캐시 백엔드 교체/추가 용이성

## 🏗️ 계층별 상세 구조

### Ports Layer (`internal/ports/`)

인바운드와 아웃바운드 포트를 정의하는 계층입니다.

#### Inbound Ports (Primary Ports)
```go
// HTTP 서버 포트
type HTTPServer interface {
    Start(ctx context.Context, addr string) error
    Stop(ctx context.Context) error
    RegisterRoute(method, path string, handler HTTPHandler)
    Use(middleware HTTPMiddleware)
}

// HTTP 핸들러 포트
type HTTPHandler interface {
    Handle(ctx HTTPContext) error
}
```

#### Outbound Ports (Secondary Ports)
```go
// 패키지 매니저 포트
type PackageManager interface {
    GetPackage(ctx context.Context, req *PackageRequest) (*PackageResponse, error)
    ListPackages(ctx context.Context, repo string) (*PackageListResponse, error)
}

// 캐시 포트
type CacheBackend interface {
    Get(ctx context.Context, key string) ([]byte, error)
    Set(ctx context.Context, key string, data []byte, ttl time.Duration) error
}
```

### Use Cases Layer (`internal/usecase/`)

프레임워크에 독립적인 비즈니스 로직을 구현하는 계층입니다.

#### ProxyService
```go
type ProxyService struct {
    packageManager ports.PackageManager  // 의존성 주입
    cacheManager   ports.CacheManager    // 의존성 주입
    authService    ports.AuthService     // 의존성 주입
    logger         ports.Logger          // 의존성 주입
}

func (ps *ProxyService) HandleProxyRequest(ctx context.Context, req *ProxyRequest) (*ProxyResponse, error) {
    // 1. 권한 검증
    if err := ps.authService.Authorize(ctx, req.UserID, req.Resource); err != nil {
        return nil, err
    }
    
    // 2. 캐시 확인
    if cached, err := ps.cacheManager.Get(ctx, req.CacheKey); err == nil {
        return &ProxyResponse{Content: cached, Cached: true}, nil
    }
    
    // 3. 업스트림에서 가져오기
    resp, err := ps.packageManager.GetPackage(ctx, req.ToPackageRequest())
    if err != nil {
        return nil, err
    }
    
    // 4. 캐시에 저장
    ps.cacheManager.Set(ctx, req.CacheKey, resp.Content, time.Hour*24)
    
    return resp, nil
}
```

#### CacheStrategyService
```go
type CacheStrategyService struct {
    cacheManager ports.CacheManager
    strategies   map[string]ports.CacheStrategy
}

func (cs *CacheStrategyService) DetermineStrategy(ctx context.Context, req *CacheRequest) (*CacheDecision, error) {
    strategy := cs.strategies[req.PackageType]
    if strategy == nil {
        strategy = cs.strategies["default"]
    }
    
    return &CacheDecision{
        ShouldCache: strategy.ShouldCache(req),
        TTL:         strategy.GetTTL(req),
        Backend:     strategy.GetBackend(req),
        CacheKey:    strategy.GenerateKey(req),
    }, nil
}
```

### Adapters Layer (`internal/adapters/`)

외부 시스템과의 인터페이스를 구현하는 계층입니다.

#### HTTP Adapter (`internal/adapters/http/fiber/`)
```go
type Server struct {
    app            *fiber.App
    proxyService   *usecase.ProxyService     // 유스케이스 의존성
    cacheService   *usecase.CacheStrategyService
    healthService  *usecase.HealthService
}

// HTTPServer 포트 구현
func (s *Server) Start(ctx context.Context, addr string) error {
    return s.app.Listen(addr)
}

// 요청 처리 핸들러
func (s *Server) handleProxy(c *fiber.Ctx) error {
    // Fiber Context를 포트 Context로 변환
    ctx := &FiberContext{ctx: c}
    
    // 유스케이스 호출
    req := &usecase.ProxyRequest{
        PackageType: c.Params("type"),
        Repository:  c.Params("repo"),
        Path:        c.Params("*"),
    }
    
    resp, err := s.proxyService.HandleProxyRequest(c.Context(), req)
    if err != nil {
        return ctx.Status(500).JSON(map[string]string{"error": err.Error()})
    }
    
    return ctx.Send(resp.Content)
}
```

## 🔄 의존성 흐름

### 컴파일 타임 의존성
```
외부 시스템 (Web, DB, Cache)
         ↑
    Adapters Layer
         ↑
     Ports Layer
         ↑
   Use Cases Layer
         ↑
    Domain Layer
```

### 런타임 제어 흐름
```
HTTP Request → Fiber Adapter → ProxyService → PackageManager Adapter → External API
                    ↓               ↓                    ↓
              HTTP Response ← Use Case Logic ← Cache/Auth Logic ← External Response
```

## 🧪 테스트 전략

### 단위 테스트
```go
func TestProxyService_HandleProxyRequest(t *testing.T) {
    // 모든 외부 의존성을 모킹
    mockPM := &mocks.PackageManager{}
    mockCache := &mocks.CacheManager{}
    mockAuth := &mocks.AuthService{}
    mockLogger := &mocks.Logger{}
    
    service := usecase.NewProxyService(mockPM, mockCache, mockAuth, mockLogger)
    
    // 모킹 설정
    mockAuth.On("Authorize", mock.Anything, "user123", "repo/package").Return(nil)
    mockCache.On("Get", mock.Anything, "cache-key").Return(nil, errors.New("cache miss"))
    mockPM.On("GetPackage", mock.Anything, mock.Anything).Return(&ports.PackageResponse{
        Content: []byte("package-content"),
    }, nil)
    
    // 테스트 실행
    req := &usecase.ProxyRequest{
        UserID:   "user123",
        Resource: "repo/package",
        CacheKey: "cache-key",
    }
    
    resp, err := service.HandleProxyRequest(context.Background(), req)
    
    // 검증
    assert.NoError(t, err)
    assert.Equal(t, []byte("package-content"), resp.Content)
    mockPM.AssertExpectations(t)
}
```

### 통합 테스트
```go
func TestHTTPServer_Integration(t *testing.T) {
    // 실제 서비스 구성 (테스트용 설정)
    server := setupTestServer(t)
    defer server.Stop(context.Background())
    
    // HTTP 요청 테스트
    resp, err := http.Get("http://localhost:8080/api/v1/proxy/npm/express")
    
    assert.NoError(t, err)
    assert.Equal(t, http.StatusOK, resp.StatusCode)
}
```

## 📦 패키지 구성 원칙

### Import 규칙
```go
// ✅ 허용되는 의존성 방향
internal/adapters/http/fiber → internal/ports
internal/adapters/http/fiber → internal/usecase
internal/usecase → internal/ports
internal/usecase → internal/domain

// ❌ 금지되는 의존성 방향
internal/ports → internal/adapters     // 포트가 어댑터에 의존하면 안됨
internal/usecase → internal/adapters   // 유스케이스가 어댑터에 의존하면 안됨
internal/domain → 모든 다른 internal 패키지  // 도메인은 가장 안쪽 계층
```

### 인터페이스 위치 규칙
- **Inbound Ports**: 유스케이스가 외부로 노출하는 인터페이스 (`internal/ports/`)
- **Outbound Ports**: 유스케이스가 외부 시스템에 의존하는 인터페이스 (`internal/ports/`)
- **구현체**: 해당 어댑터 패키지에 위치

## 🔧 DI Container 설계

### 의존성 주입 컨테이너
```go
type Container struct {
    // Repositories
    cacheRepo     ports.CacheBackend
    configRepo    ports.ConfigRepository
    
    // Services  
    proxyService  *usecase.ProxyService
    cacheService  *usecase.CacheStrategyService
    healthService *usecase.HealthService
    
    // Adapters
    httpServer    ports.HTTPServer
    packageManagers map[string]ports.TypedPackageManager
}

func NewContainer(config *Config) (*Container, error) {
    container := &Container{}
    
    // 1. 어댑터 생성 (가장 바깥쪽)
    container.cacheRepo = filesystem.NewCacheBackend(config.CacheDir)
    container.configRepo = file.NewConfigRepository(config.ConfigDir)
    
    // 2. 유스케이스 생성 (의존성 주입)
    container.proxyService = usecase.NewProxyService(
        container.packageManagers["npm"], // 나중에 팩토리로 교체
        container.cacheRepo,
        container.authService,
        container.logger,
    )
    
    // 3. HTTP 서버 생성 (가장 바깥쪽)
    container.httpServer = fiber.NewServer(
        container.proxyService,
        container.cacheService,
        container.healthService,
        container.logger,
        config.Server,
    )
    
    return container, nil
}
```

## 🚀 마이그레이션 가이드

### 기존 코드 → 새 아키텍처

#### 1단계: 핸들러 마이그레이션
```go
// Before: handlers/proxy/npm_handler.go
func (h *NPMHandler) HandleNPMRequest(c *fiber.Ctx) error {
    // 직접 fiber.Ctx 사용, 비즈니스 로직 섞임
}

// After: internal/usecase/proxy_service.go  
func (ps *ProxyService) HandleNPMRequest(ctx context.Context, req *NPMRequest) (*NPMResponse, error) {
    // 프레임워크 독립적, 순수 비즈니스 로직
}

// After: internal/adapters/http/fiber/handlers/npm.go
func (h *NPMHandler) Handle(ctx ports.HTTPContext) error {
    // HTTP 어댑터, 유스케이스 호출
    req := h.buildNPMRequest(ctx)
    resp, err := h.proxyService.HandleNPMRequest(ctx.Context(), req)
    return h.buildHTTPResponse(ctx, resp, err)
}
```

#### 2단계: 미들웨어 마이그레이션
```go
// Before: middlewares/auth.go
func AuthMiddleware() fiber.Handler {
    return func(c *fiber.Ctx) error {
        // 직접 fiber.Ctx 사용
    }
}

// After: internal/adapters/http/fiber/middleware/auth.go
type AuthMiddleware struct {
    authService ports.AuthService // 포트 의존성
}

func (m *AuthMiddleware) Process(next ports.HTTPHandler) ports.HTTPHandler {
    return &AuthMiddlewareHandler{
        next:        next,
        authService: m.authService,
    }
}
```

## 📊 성능 고려사항

### 계층간 호출 오버헤드
- 인터페이스 호출로 인한 미미한 성능 영향 (< 1%)
- 컴파일러 최적화로 대부분 상쇄
- 테스트 가능성과 유지보수성 이득이 훨씬 큼

### 메모리 사용량
- 의존성 주입으로 인한 추가 포인터 참조
- 전체 메모리 사용량에 미미한 영향
- 가비지 컬렉션 효율성 향상 가능

## 🎉 이점 요약

1. **테스트 용이성**: 모든 의존성을 모킹하여 빠른 단위 테스트
2. **유연성**: 웹 프레임워크나 데이터베이스 교체 용이
3. **확장성**: 새로운 패키지 매니저나 기능 추가 용이
4. **유지보수성**: 각 계층의 책임이 명확하여 코드 이해 쉬움
5. **비즈니스 로직 보호**: 프레임워크 변경이 핵심 로직에 영향 없음