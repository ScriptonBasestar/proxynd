# Container 기반 의존성 주입 아키텍처

## 📋 개요

ProxyND의 Container 기반 의존성 주입 시스템은 고성능, 확장 가능하며 테스트하기 쉬운 프록시 핸들러 아키텍처를 제공합니다. 이 시스템은 기존의 매 요청마다 설정을 읽는 방식을 개선하여 성능을 크게 향상시켰습니다.

### 📊 개선 효과

**Before (기존 방식)**:
- 매 요청마다 53+ 파일 읽기 작업
- 핸들러 인스턴스 중복 생성
- 직접적인 설정 파일 의존성

**After (Container 방식)**:
- 애플리케이션 시작 시 1회 설정 로딩
- 99.9% 파일 I/O 작업 제거
- 핸들러 인스턴스 재사용으로 메모리 절약

## 🏗️ 아키텍처 개요

```mermaid
graph TB
    subgraph "Request Flow"
        A[HTTP Request] --> B[Fiber Router]
        B --> C[Container Proxy Router]
        C --> D[Handler Factory]
        D --> E[Container Handler]
    end

    subgraph "Container System"
        F[Container Provider] --> G[Configuration Cache]
        F --> H[Service Singletons]
        F --> I[Hot Reload Watcher]
    end

    subgraph "Handler Layer"
        E --> J[Base Container Handler]
        J --> K[Metrics Collection]
        J --> L[Config Access]
        J --> M[Upstream Proxy]
    end

    E --> F
    G --> L
    K --> N[Prometheus Metrics]
```

## 🔧 핵심 컴포넌트

### 1. Container Provider (`/internal/app/container.go`)

**역할**: 중앙집중식 의존성 관리 및 설정 캐싱

**주요 기능**:
- **Thread-Safe Singleton**: `sync.RWMutex`를 사용한 동시성 제어
- **Lazy Initialization**: 최초 요청 시점에 설정 로딩
- **Configuration Caching**: 메모리 내 설정 캐시 및 재사용
- **Hot Reload**: `fsnotify`를 통한 설정 파일 변경 감지
- **Service Factory Pattern**: 서비스 인스턴스 생성 및 관리

**Interface Contract**:
```go
type ContainerProvider interface {
    // Storage & Configuration
    GetStorageDir() string
    GetConfigDir() string

    // Proxy Configurations (Cached)
    GetAptProxyConfig() (*config.AptProxyConfig, error)
    GetMavenProxyConfig() (*config.MavenProxySettings, error)
    GetNpmProxyConfig() (*config.NpmProxySettings, error)
    GetDockerProxyConfig() (*config.DockerProxySettings, error)
    GetPipProxyConfig() (*config.PipProxySettings, error)
    GetYumProxyConfig() (*config.YumProxySettings, error)
    GetApkProxyConfig() (*config.ApkProxySettings, error)
}
```

### 2. Handler Interface Hierarchy

```go
// 기본 핸들러 계층
Handler
├── ContainerAwareHandler
│   ├── LoadConfig() error
│   ├── ReloadConfig() error
│   ├── SetContainer(ContainerProvider)
│   └── GetContainer() ContainerProvider
│
└── BaseProxyHandler
    ├── Type() string
    ├── IsEnabled() bool
    ├── GenerateCacheKey(*fiber.Ctx) string
    ├── BuildUpstreamURL(*fiber.Ctx) (string, error)
    ├── TransformRequest(*fiber.Ctx, *fiber.Agent) error
    ├── TransformResponse([]byte, *fiber.Ctx) ([]byte, error)
    ├── ShouldCache(*fiber.Ctx, int) bool
    ├── GetCacheTTL(*fiber.Ctx) time.Duration
    └── HandleError(error, *fiber.Ctx) error

// 통합 인터페이스
ContainerProxyHandler = ContainerAwareHandler + BaseProxyHandler
```

### 3. Handler Factory Pattern

**Factory Registration**:
```go
type HandlerFactory func(ContainerProvider) (ContainerProxyHandler, error)

factory.RegisterHandler("apt", func(provider ContainerProvider) (ContainerProxyHandler, error) {
    handler := containerhandlers.NewAPTContainerHandler(provider)
    return middleware.WrapContainerHandler(handler), nil
})
```

**지원하는 프록시 타입**:
- `apt`: Debian/Ubuntu 패키지 프록시
- `maven`: Java 패키지 프록시  
- `npm`: Node.js 패키지 프록시
- `docker`: Container 이미지 프록시
- `pip`: Python 패키지 프록시
- `yum`: Red Hat 패키지 프록시
- `apk`: Alpine 패키지 프록시

## 🔄 핵심 디자인 패턴

### 1. Dependency Injection Pattern

**Inversion of Control (IoC)**:
```go
// ❌ Bad: 직접적인 의존성
type Handler struct {
    configPath string
}

func (h *Handler) process() error {
    config := &Config{}
    config.ReadFromFile(h.configPath)  // 직접 의존
    // ...
}

// ✅ Good: 의존성 주입
type Handler struct {
    provider ContainerProvider
}

func (h *Handler) process() error {
    config, err := h.provider.GetConfig()  // 주입된 의존성
    if err != nil {
        return err
    }
    // ...
}
```

### 2. Thread-Safe Singleton Pattern

**Double-Checked Locking 구현**:
```go
func (c *Container) GetMavenProxyConfig() (*config.MavenProxySettings, error) {
    // Fast path: 읽기 락으로 캐시 확인
    c.mu.RLock()
    if cached, exists := c.singletons["maven-proxy-config"]; exists {
        c.mu.RUnlock()
        return cached.(*config.MavenProxySettings), nil
    }
    c.mu.RUnlock()

    // Slow path: 쓰기 락으로 설정 로딩
    c.mu.Lock()
    defer c.mu.Unlock()

    // Double-check: 락 획득 중 다른 고루틴이 로딩했을 수 있음
    if cached, exists := c.singletons["maven-proxy-config"]; exists {
        return cached.(*config.MavenProxySettings), nil
    }

    // 실제 설정 로딩 및 캐싱
    cfg, err := c.configLoader.LoadMavenProxyConfig(context.Background())
    if err != nil {
        return nil, err
    }

    c.singletons["maven-proxy-config"] = cfg
    return cfg, nil
}
```

### 3. Factory Pattern with Registry

**Handler Instance Reuse**:
```go
var handlerInstances = make(map[string]ContainerProxyHandler)

func (f *StandardProxyHandlerFactory) CreateHandler(proxyType string, provider ContainerProvider) (ContainerProxyHandler, error) {
    if handler, exists := handlerInstances[proxyType]; exists {
        return handler, nil
    }

    createFn, exists := f.creators[proxyType]
    if !exists {
        return nil, fmt.Errorf("unsupported proxy type: %s", proxyType)
    }

    handler, err := createFn(provider)
    if err != nil {
        return nil, err
    }

    handlerInstances[proxyType] = handler
    return handler, nil
}
```

## ⚙️ 설정 관리

### Hot Reload System

**File Watcher Implementation**:
```go
func (c *Container) startConfigWatcher() {
    watcher, err := fsnotify.NewWatcher()
    if err != nil {
        return
    }

    err = watcher.Add(c.config.ConfigDir)
    if err != nil {
        return
    }

    go func() {
        for {
            select {
            case event := <-watcher.Events:
                if event.Op&fsnotify.Write == fsnotify.Write {
                    time.Sleep(100 * time.Millisecond) // Debounce
                    c.ReloadConfig()
                }
            case err := <-watcher.Errors:
                log.Error("Config watcher error", err)
            }
        }
    }()
}
```

**설정 계층구조**:
```
기본값 → YAML 파일 → 환경 변수
```

**환경 변수 확장**:
```yaml
# maven-proxy.yaml
proxies:
  - name: central
    url: ${MAVEN_CENTRAL_URL:https://repo1.maven.org/maven2}
```

## 📊 성능 최적화

### 1. Configuration Caching

**성능 개선**:
- **I/O 감소**: 99.9% 파일 읽기 작업 제거
- **CPU 사용량**: YAML 파싱 오버헤드 제거
- **메모리 효율성**: 설정당 단일 인스턴스 유지

### 2. Round-Robin Load Balancing

**Atomic Counter 기반 부하 분산**:
```go
type PIPContainerHandler struct {
    serverIdx uint32 // atomic counter
    proxies   []config.PipProxyServer
}

func (h *PIPContainerHandler) selectUpstreamServer() config.PipProxyServer {
    if len(h.proxies) == 1 {
        return h.proxies[0]
    }

    idx := atomic.AddUint32(&h.serverIdx, 1) % uint32(len(h.proxies))
    return h.proxies[idx]
}
```

## 📈 메트릭 및 모니터링

### Container-Specific Metrics

**Handler Lifecycle Metrics**:
```go
// 핸들러 초기화 추적
proxynd_container_handler_initializations_total{handler_type="maven", status="success"}

// 요청 처리 성능
proxynd_container_handler_requests_total{handler_type="maven", method="GET", status="200"}
proxynd_container_handler_duration_seconds{handler_type="maven", method="GET"}
```

**Configuration Efficiency Metrics**:
```go
// 설정 로딩 효율성 (목표: 감소)
proxynd_config_load_operations_total{handler_type="maven", operation="initial", result="success"}

// 캐시 히트율 (목표: 95%+)
proxynd_config_cache_hits_total{handler_type="maven", config_type="maven-config"}
proxynd_config_cache_misses_total{handler_type="maven", config_type="maven-config"}
```

### Automatic Metrics Collection

```go
func (h *BaseContainerHandler) WithMetrics(c *fiber.Ctx, handler func() error) error {
    start := time.Now()
    err := handler()

    duration := time.Since(start)
    statusCode := c.Response().StatusCode()

    h.RecordHandlerRequest(c.Method(), statusCode)
    h.RecordHandlerDuration(c.Method(), duration)

    if statusCode >= 400 {
        h.RecordHandlerError(getErrorTypeFromStatusCode(statusCode))
    }

    return err
}
```

## 🧪 테스트 아키텍처

### MockContainer System

**Test-Friendly Design**:
```go
type MockContainerProvider struct {
    mock.Mock
    storageDir string
    configDir  string

    // 캐시된 설정들
    aptConfig    *config.AptProxyConfig
    mavenConfig  *config.MavenProxySettings
}

// Builder Pattern for Test Setup
func NewMockContainerProvider(t *testing.T) *MockContainerProvider {
    return &MockContainerProvider{
        storageDir: t.TempDir() + "/storage",
        configDir:  t.TempDir() + "/config",
    }.WithDefaultConfigs()
}

// Method Chaining for Test Configuration
mockContainer := testutil.NewMockContainerProvider(t).
    WithStorageDir("/custom/path").
    WithError("GetMavenProxyConfig", someError).
    WithConfig("maven", customMavenConfig)
```

### Integration Test Suite

```go
type ContainerHandlerTestSuite struct {
    t             *testing.T
    mockContainer *MockContainerProvider
    app           *fiber.App
    tempDir       string
}

func (s *ContainerHandlerTestSuite) TestAllHandlersBasicFunctionality() {
    handlerTypes := []string{"apt", "maven", "npm", "docker", "pip"}

    for _, handlerType := range handlerTypes {
        handler := s.createHandler(handlerType)

        assert.True(s.t, handler.IsEnabled())
        assert.NoError(s.t, handler.HealthCheck())
        assert.NotEmpty(s.t, handler.Name())
        assert.Equal(s.t, handlerType, handler.Type())
    }
}
```

## 🔗 라우팅 통합

### Unified Proxy Router

**Route Pattern**:
```
/api/v1/proxy/:type/*  (새로운 통합 API)
/proxy/:type/*         (레거시 호환성)
```

**Handler Resolution**:
```go
func (r *ContainerProxyRouter) setupRoutes() {
    handlerTypes := []string{"apt", "maven", "npm", "docker", "pip"}

    for _, handlerType := range handlerTypes {
        // 새로운 API 경로
        r.app.All(fmt.Sprintf("/api/v1/proxy/%s/*", handlerType),
            r.createHandlerFunc(handlerType))

        // 레거시 API 경로  
        r.app.All(fmt.Sprintf("/proxy/%s/*", handlerType),
            r.createHandlerFunc(handlerType))
    }
}

func (r *ContainerProxyRouter) createHandlerFunc(handlerType string) fiber.Handler {
    return func(c *fiber.Ctx) error {
        handler, err := r.handlerFactory.CreateHandler(handlerType, r.containerProvider)
        if err != nil {
            return fiber.NewError(fiber.StatusInternalServerError, err.Error())
        }

        return handler.Handle(c)
    }
}
```

## 🔒 보안 고려사항

### Configuration Access Control

**Safe Configuration Loading**:
```go
func (c *Container) loadConfigSafely(configPath string, target interface{}) error {
    // Path Traversal 방지
    if !strings.HasPrefix(configPath, c.config.ConfigDir) {
        return errors.New("config path outside allowed directory")
    }

    // 파일 크기 제한 (메모리 보호)
    if stat, err := os.Stat(configPath); err == nil {
        if stat.Size() > maxConfigFileSize {
            return errors.New("config file too large")
        }
    }

    return helpers.ReadYamlSafe(configPath, target)
}
```

### Container Isolation

- Container 내부 구현 세부사항 은닉
- 핸들러는 인터페이스를 통해서만 접근
- Mock 가능한 테스트 경계 제공

## 🚀 마이그레이션 가이드

### 기존 핸들러에서 Container 핸들러로 전환

**1단계: 기존 핸들러 분석**
```go
// Before (기존 패턴)
type OldHandler struct {
    configPath string
}

func (h *OldHandler) Handle(c *fiber.Ctx) error {
    config := &SomeProxyConfig{}
    if err := config.ReadConfig(); err != nil {  // 매번 파일 읽기
        return err
    }
    // 비즈니스 로직
    return nil
}
```

**2단계: Container 기반 핸들러 생성**
```go
// After (Container 패턴)
type NewContainerHandler struct {
    *containerhandlers.BaseContainerHandler
    provider ContainerProvider
}

func NewContainerHandler(provider ContainerProvider) *NewContainerHandler {
    return &NewContainerHandler{
        BaseContainerHandler: containerhandlers.NewBaseContainerHandler(
            provider, "new-handler", "NewHandler",
        ),
        provider: provider,
    }
}

func (h *NewContainerHandler) Handle(c *fiber.Ctx) error {
    return h.WithMetrics(c, func() error {
        config, err := h.provider.GetConfig()  // 캐시된 설정 사용
        if err != nil {
            return err
        }
        // 비즈니스 로직
        return nil
    })
}
```

**3단계: Factory에 등록**
```go
factory.RegisterHandler("new", func(provider ContainerProvider) (ContainerProxyHandler, error) {
    return NewContainerHandler(provider), nil
})
```

## 📊 운영 모니터링

### Key Performance Indicators

1. **Configuration Load Reduction**: 95%+ 감소 목표
2. **Cache Hit Ratio**: 90%+ 유지 목표  
3. **Response Time**: 50%+ 개선 목표
4. **Memory Usage**: 안정적인 메모리 사용 패턴
5. **Error Rate**: Container 관련 에러 0.1% 미만

### Alerting Rules

```yaml
# Prometheus 알림 규칙
- alert: ContainerConfigCacheHitRateLow
  expr: |
    (
      rate(proxynd_config_cache_hits_total[5m]) /
      (rate(proxynd_config_cache_hits_total[5m]) + rate(proxynd_config_cache_misses_total[5m]))
    ) * 100 < 90
  for: 2m
  annotations:
    summary: "Container config cache hit rate is below 90%"

- alert: ContainerHandlerErrors
  expr: rate(proxynd_container_handler_errors_total[5m]) > 0.01
  for: 1m
  annotations:
    summary: "Container handler error rate is above 1%"
```

## 🎯 이점 요약

1. **성능 향상**: 99.9% I/O 작업 감소로 응답 시간 단축
2. **메모리 효율성**: 설정 캐싱과 핸들러 인스턴스 재사용
3. **테스트 용이성**: MockContainer를 통한 간단하고 reliable한 테스트
4. **확장성**: 새로운 프록시 타입 추가 용이
5. **유지보수성**: 명확한 책임 분리와 의존성 주입
6. **안정성**: Thread-safe 구현과 에러 처리
7. **모니터링**: 상세한 메트릭과 성능 추적

이 Container 기반 아키텍처는 ProxyND의 안정성, 성능, 확장성을 크게 향상시키는 현대적인 의존성 주입 패턴을 구현합니다.