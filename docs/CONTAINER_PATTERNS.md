# Container 패턴 및 베스트 프랙티스

## 패턴 개요

ProxyND의 Container 기반 의존성 주입 시스템에서 사용되는 핵심 디자인 패턴과 구현 베스트 프랙티스를 설명합니다.

## 핵심 디자인 패턴

### 1. Dependency Injection Container 패턴

#### 의존성 주입의 핵심 원칙

**Inversion of Control (IoC)**:
```go
// Bad: 직접적인 의존성
type Handler struct {
    configPath string
}

func (h *Handler) process() error {
    config := &Config{}
    config.ReadFromFile(h.configPath)  // 직접 의존
    // ...
}

// Good: 의존성 주입
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

**Interface Segregation**:
```go
// 큰 인터페이스를 작은 단위로 분리
type ConfigProvider interface {
    GetAptProxyConfig() (*config.AptProxyConfig, error)
    GetMavenProxyConfig() (*config.MavenProxySettings, error)
}

type StorageProvider interface {
    GetStorageDir() string
    GetConfigDir() string
}

// 합성된 인터페이스
type ContainerProvider interface {
    ConfigProvider
    StorageProvider
}
```

### 2. Singleton 패턴 (Thread-Safe)

#### Double-Checked Locking 구현
```go
type Container struct {
    mu         sync.RWMutex
    singletons map[string]interface{}
}

func (c *Container) GetMavenProxyConfig() (*config.MavenProxySettings, error) {
    // 1차 체크: 읽기 락 (Fast Path)
    c.mu.RLock()
    if cached, exists := c.singletons["maven-proxy-config"]; exists {
        c.mu.RUnlock()
        return cached.(*config.MavenProxySettings), nil
    }
    c.mu.RUnlock()
    
    // 2차 체크: 쓰기 락 (Slow Path)  
    c.mu.Lock()
    defer c.mu.Unlock()
    
    // Double-check: 락 대기 중 다른 고루틴이 생성했을 수 있음
    if cached, exists := c.singletons["maven-proxy-config"]; exists {
        return cached.(*config.MavenProxySettings), nil
    }
    
    // 실제 생성 및 캐싱
    cfg, err := c.loadMavenConfig()
    if err != nil {
        return nil, err
    }
    
    c.singletons["maven-proxy-config"] = cfg
    return cfg, nil
}
```

#### 지연 초기화 (Lazy Initialization)
```go
type LazyContainer struct {
    once   sync.Once
    config *GlobalConfig
    err    error
}

func (c *LazyContainer) GetGlobalConfig() (*GlobalConfig, error) {
    c.once.Do(func() {
        c.config, c.err = loadGlobalConfig()
    })
    return c.config, c.err
}
```

### 3. Factory 패턴

#### Abstract Factory 구현
```go
type HandlerFactory interface {
    CreateHandler(proxyType string) (ContainerProxyHandler, error)
    SupportedTypes() []string
}

type StandardProxyHandlerFactory struct {
    creators map[string]HandlerCreator
    provider ContainerProvider
}

type HandlerCreator func(ContainerProvider) (ContainerProxyHandler, error)

func (f *StandardProxyHandlerFactory) RegisterHandler(
    proxyType string, 
    creator HandlerCreator,
) error {
    if f.creators == nil {
        f.creators = make(map[string]HandlerCreator)
    }
    
    f.creators[proxyType] = creator
    return nil
}

func (f *StandardProxyHandlerFactory) CreateHandler(
    proxyType string,
) (ContainerProxyHandler, error) {
    creator, exists := f.creators[proxyType]
    if !exists {
        return nil, fmt.Errorf("unsupported proxy type: %s", proxyType)
    }
    
    return creator(f.provider)
}
```

#### 팩토리 등록 패턴
```go
// 초기화 시점에 모든 핸들러 등록
func (c *Container) initializeHandlerFactory() error {
    factory := NewStandardProxyHandlerFactory(c)
    
    // 각 핸들러 타입별 등록
    handlers := map[string]HandlerCreator{
        "apt":    func(p ContainerProvider) (ContainerProxyHandler, error) {
            return containerhandlers.NewAPTContainerHandler(p), nil
        },
        "maven":  func(p ContainerProvider) (ContainerProxyHandler, error) {
            return containerhandlers.NewMavenContainerHandler(p), nil
        },
        "npm":    func(p ContainerProvider) (ContainerProxyHandler, error) {
            return containerhandlers.NewNPMContainerHandler(p), nil
        },
    }
    
    for proxyType, creator := range handlers {
        if err := factory.RegisterHandler(proxyType, creator); err != nil {
            return fmt.Errorf("failed to register %s handler: %w", proxyType, err)
        }
    }
    
    c.handlerFactory = factory
    return nil
}
```

### 4. Adapter 패턴

#### 레거시 핸들러와 Container 핸들러 통합
```go
type containerHandlerAdapter struct {
    name      string
    proxyType string
    provider  ContainerProvider
    instance  ContainerProxyHandler
    once      sync.Once
}

func (a *containerHandlerAdapter) Handle(c *fiber.Ctx) error {
    handler, err := a.getOrCreateHandler()
    if err != nil {
        return err
    }
    
    return handler.Handle(c)
}

func (a *containerHandlerAdapter) getOrCreateHandler() (ContainerProxyHandler, error) {
    var err error
    a.once.Do(func() {
        switch a.proxyType {
        case "apt":
            a.instance = containerhandlers.NewAPTContainerHandler(a.provider)
        case "maven":
            a.instance = containerhandlers.NewMavenContainerHandler(a.provider)
        case "npm":
            a.instance = containerhandlers.NewNPMContainerHandler(a.provider)
        case "docker":
            a.instance = containerhandlers.NewDockerContainerHandler(a.provider)
        case "pip":
            a.instance = containerhandlers.NewPIPContainerHandler(a.provider)
        case "yum":
            a.instance = containerhandlers.NewYUMContainerHandler(a.provider)
        case "apk":
            a.instance = containerhandlers.NewAPKContainerHandler(a.provider)
        default:
            err = fmt.Errorf("unsupported proxy type: %s", a.proxyType)
        }
    })
    
    return a.instance, err
}
```

### 5. Decorator 패턴 (Metrics Wrapper)

#### 메트릭 수집을 위한 데코레이터
```go
type ContainerHandlerMetricsWrapper struct {
    handler          ContainerProxyHandler
    containerMetrics *metrics.ContainerMetrics
    handlerType      string
}

func WrapWithMetrics(handler ContainerProxyHandler) ContainerProxyHandler {
    return &ContainerHandlerMetricsWrapper{
        handler:          handler,
        containerMetrics: metrics.GetContainerMetrics(),
        handlerType:      handler.Type(),
    }
}

func (w *ContainerHandlerMetricsWrapper) Handle(c *fiber.Ctx) error {
    start := time.Now()
    
    // 원본 핸들러 호출
    err := w.handler.Handle(c)
    
    // 메트릭 수집
    duration := time.Since(start).Seconds()
    statusCode := c.Response().StatusCode()
    
    w.containerMetrics.RecordHandlerRequest(w.handlerType, c.Method(), statusCode)
    w.containerMetrics.RecordHandlerDuration(w.handlerType, c.Method(), duration)
    
    if statusCode >= 400 {
        w.containerMetrics.RecordHandlerError(w.handlerType, "http_error")
    }
    
    return err
}

// 다른 메서드들도 동일하게 위임 + 메트릭 수집
func (w *ContainerHandlerMetricsWrapper) GenerateCacheKey(c *fiber.Ctx) string {
    start := time.Now()
    key := w.handler.GenerateCacheKey(c)
    duration := time.Since(start).Seconds()
    
    w.containerMetrics.RecordCacheKeyGeneration(w.handlerType, duration)
    return key
}
```

### 6. Template Method 패턴

#### 공통 핸들러 로직 추상화
```go
type BaseContainerHandler struct {
    containerProvider ContainerProvider
    containerMetrics  *metrics.ContainerMetrics
    handlerType       string
    handlerName       string
}

// Template Method: 공통 요청 처리 흐름
func (h *BaseContainerHandler) ProcessRequest(c *fiber.Ctx) error {
    // 1. 사전 검증
    if err := h.validateRequest(c); err != nil {
        return err
    }
    
    // 2. 설정 확인
    if !h.IsConfigured() {
        return errors.New("handler not configured")
    }
    
    // 3. 캐시 확인
    if h.IsCacheable(c) {
        if cached := h.getCachedResponse(c); cached != nil {
            return h.sendCachedResponse(c, cached)
        }
    }
    
    // 4. 업스트림 처리 (서브클래스에서 구현)
    response, err := h.processUpstream(c)
    if err != nil {
        return h.handleError(c, err)
    }
    
    // 5. 응답 후처리
    return h.postProcess(c, response)
}

// 서브클래스에서 구현해야 하는 추상 메서드
func (h *BaseContainerHandler) processUpstream(c *fiber.Ctx) (*UpstreamResponse, error) {
    panic("must be implemented by subclass")
}
```

## 베스트 프랙티스

### 1. Configuration Management

#### 설정 캐싱 전략
```go
type ConfigurationCache struct {
    cache   map[string]*CacheEntry
    mu      sync.RWMutex
    maxAge  time.Duration
    maxSize int
}

type CacheEntry struct {
    value     interface{}
    timestamp time.Time
    hitCount  int64
}

func (c *ConfigurationCache) Get(key string) (interface{}, bool) {
    c.mu.RLock()
    defer c.mu.RUnlock()
    
    entry, exists := c.cache[key]
    if !exists {
        return nil, false
    }
    
    // TTL 체크
    if time.Since(entry.timestamp) > c.maxAge {
        delete(c.cache, key)
        return nil, false
    }
    
    // 히트 카운트 증가 (atomic)
    atomic.AddInt64(&entry.hitCount, 1)
    return entry.value, true
}

func (c *ConfigurationCache) Set(key string, value interface{}) {
    c.mu.Lock()
    defer c.mu.Unlock()
    
    // 캐시 크기 제한
    if len(c.cache) >= c.maxSize {
        c.evictLRU()
    }
    
    c.cache[key] = &CacheEntry{
        value:     value,
        timestamp: time.Now(),
        hitCount:  0,
    }
}
```

#### 환경별 설정 관리
```go
type EnvironmentConfig struct {
    Environment string `yaml:"environment"`
    Debug       bool   `yaml:"debug"`
    
    // 환경별 오버라이드
    Development *EnvOverride `yaml:"development,omitempty"`
    Production  *EnvOverride `yaml:"production,omitempty"`
    Test        *EnvOverride `yaml:"test,omitempty"`
}

func (c *EnvironmentConfig) ApplyEnvironmentOverrides() {
    switch c.Environment {
    case "development":
        if c.Development != nil {
            c.applyOverride(c.Development)
        }
    case "production":
        if c.Production != nil {
            c.applyOverride(c.Production)
        }
    case "test":
        if c.Test != nil {
            c.applyOverride(c.Test)
        }
    }
}
```

### 2. Error Handling Patterns

#### Graceful Degradation
```go
type FallbackConfigProvider struct {
    primary   ContainerProvider
    secondary ContainerProvider
    circuit   *CircuitBreaker
}

func (f *FallbackConfigProvider) GetMavenProxyConfig() (*config.MavenProxySettings, error) {
    // Circuit Breaker 상태 확인
    if f.circuit.IsOpen() {
        return f.secondary.GetMavenProxyConfig()
    }
    
    cfg, err := f.primary.GetMavenProxyConfig()
    if err != nil {
        f.circuit.RecordFailure()
        // Fallback to secondary
        return f.secondary.GetMavenProxyConfig()
    }
    
    f.circuit.RecordSuccess()
    return cfg, nil
}
```

#### Retry with Exponential Backoff
```go
type RetryableContainer struct {
    inner       ContainerProvider
    maxRetries  int
    baseDelay   time.Duration
    maxDelay    time.Duration
}

func (r *RetryableContainer) GetConfigWithRetry(configKey string) (interface{}, error) {
    var lastErr error
    
    for attempt := 0; attempt <= r.maxRetries; attempt++ {
        config, err := r.loadConfig(configKey)
        if err == nil {
            return config, nil
        }
        
        lastErr = err
        
        if attempt < r.maxRetries {
            delay := r.calculateDelay(attempt)
            time.Sleep(delay)
        }
    }
    
    return nil, fmt.Errorf("failed after %d attempts: %w", r.maxRetries, lastErr)
}

func (r *RetryableContainer) calculateDelay(attempt int) time.Duration {
    delay := r.baseDelay * time.Duration(1<<attempt) // 2^attempt
    if delay > r.maxDelay {
        delay = r.maxDelay
    }
    return delay
}
```

### 3. Performance Optimization

#### Object Pooling for Handlers
```go
type HandlerPool struct {
    pool    sync.Pool
    factory func() ContainerProxyHandler
}

func NewHandlerPool(factory func() ContainerProxyHandler) *HandlerPool {
    return &HandlerPool{
        pool: sync.Pool{
            New: func() interface{} {
                return factory()
            },
        },
        factory: factory,
    }
}

func (p *HandlerPool) Get() ContainerProxyHandler {
    return p.pool.Get().(ContainerProxyHandler)
}

func (p *HandlerPool) Put(handler ContainerProxyHandler) {
    // 핸들러 상태 리셋
    if resettable, ok := handler.(interface{ Reset() }); ok {
        resettable.Reset()
    }
    
    p.pool.Put(handler)
}
```

#### Memory-Efficient String Interning
```go
type StringInterner struct {
    strings map[string]string
    mu      sync.RWMutex
}

func (s *StringInterner) Intern(str string) string {
    s.mu.RLock()
    if interned, exists := s.strings[str]; exists {
        s.mu.RUnlock()
        return interned
    }
    s.mu.RUnlock()
    
    s.mu.Lock()
    defer s.mu.Unlock()
    
    // Double-check
    if interned, exists := s.strings[str]; exists {
        return interned
    }
    
    s.strings[str] = str
    return str
}

// 핸들러에서 사용
func (h *Handler) GenerateCacheKey(c *fiber.Ctx) string {
    key := fmt.Sprintf("%s:%s:%s", h.Type(), c.Method(), c.Path())
    return stringInterner.Intern(key)  // 메모리 절약
}
```

### 4. Testing Patterns

#### Dependency Injection for Tests
```go
type TestContainer struct {
    configs   map[string]interface{}
    errors    map[string]error
    callCount map[string]int
    mu        sync.RWMutex
}

func NewTestContainer() *TestContainer {
    return &TestContainer{
        configs:   make(map[string]interface{}),
        errors:    make(map[string]error),
        callCount: make(map[string]int),
    }
}

func (t *TestContainer) WithConfig(key string, config interface{}) *TestContainer {
    t.mu.Lock()
    defer t.mu.Unlock()
    t.configs[key] = config
    return t
}

func (t *TestContainer) WithError(method string, err error) *TestContainer {
    t.mu.Lock()
    defer t.mu.Unlock()
    t.errors[method] = err
    return t
}

func (t *TestContainer) GetMavenProxyConfig() (*config.MavenProxySettings, error) {
    t.mu.Lock()
    defer t.mu.Unlock()
    
    t.callCount["GetMavenProxyConfig"]++
    
    if err, exists := t.errors["GetMavenProxyConfig"]; exists {
        return nil, err
    }
    
    if cfg, exists := t.configs["maven"]; exists {
        return cfg.(*config.MavenProxySettings), nil
    }
    
    return &config.MavenProxySettings{}, nil
}

// 테스트에서 사용
func TestHandlerWithMockContainer(t *testing.T) {
    mockContainer := NewTestContainer().
        WithConfig("maven", &config.MavenProxySettings{
            Path: "test-maven",
            Proxies: []config.MavenProxyServer{
                {Name: "central", URL: "https://test.maven.org"},
            },
        })
    
    handler := containerhandlers.NewMavenContainerHandler(mockContainer)
    
    assert.True(t, handler.IsEnabled())
    assert.Equal(t, "maven", handler.Type())
}
```

#### Integration Test Helpers
```go
type IntegrationTestSuite struct {
    container     ContainerProvider
    tempDir       string
    configFiles   map[string]string
    t             *testing.T
}

func NewIntegrationTestSuite(t *testing.T) *IntegrationTestSuite {
    suite := &IntegrationTestSuite{
        t:           t,
        tempDir:     t.TempDir(),
        configFiles: make(map[string]string),
    }
    
    suite.setupEnvironment()
    return suite
}

func (s *IntegrationTestSuite) setupEnvironment() {
    // 임시 설정 파일들 생성
    s.createConfigFile("maven-proxy.yaml", `
path: maven
use_cache: true
proxies:
  - name: central
    url: https://repo1.maven.org/maven2
`)
    
    // Container 초기화
    s.container = app.NewContainer(&app.Config{
        ConfigDir:  filepath.Join(s.tempDir, "config"),
        StorageDir: filepath.Join(s.tempDir, "storage"),
    })
}

func (s *IntegrationTestSuite) createConfigFile(name, content string) {
    configDir := filepath.Join(s.tempDir, "config")
    os.MkdirAll(configDir, 0755)
    
    configPath := filepath.Join(configDir, name)
    err := ioutil.WriteFile(configPath, []byte(content), 0644)
    require.NoError(s.t, err)
    
    s.configFiles[name] = configPath
}

func (s *IntegrationTestSuite) TestAllHandlersLoadConfig() {
    handlerTypes := []string{"apt", "maven", "npm", "docker", "pip", "yum", "apk"}
    
    for _, handlerType := range handlerTypes {
        s.t.Run(handlerType, func(t *testing.T) {
            factory, err := s.container.GetContainerProxyHandlerFactory()
            require.NoError(t, err)
            
            handler, err := factory.CreateHandler(handlerType, s.container)
            require.NoError(t, err)
            
            assert.True(t, handler.IsEnabled())
            assert.NoError(t, handler.LoadConfig())
        })
    }
}
```

### 5. Monitoring and Observability

#### Structured Logging
```go
type ContainerLogger struct {
    logger       *logrus.Logger
    containerID  string
    handlerType  string
}

func (l *ContainerLogger) LogConfigLoad(operation string, duration time.Duration, err error) {
    fields := logrus.Fields{
        "container_id": l.containerID,
        "handler_type": l.handlerType,
        "operation":    operation,
        "duration_ms":  duration.Milliseconds(),
    }
    
    if err != nil {
        fields["error"] = err.Error()
        l.logger.WithFields(fields).Error("Config load failed")
    } else {
        l.logger.WithFields(fields).Info("Config load successful")
    }
}

func (l *ContainerLogger) LogHandlerRequest(method, path string, statusCode int, duration time.Duration) {
    l.logger.WithFields(logrus.Fields{
        "container_id": l.containerID,
        "handler_type": l.handlerType,
        "method":       method,
        "path":         path,
        "status_code":  statusCode,
        "duration_ms":  duration.Milliseconds(),
    }).Info("Handler request processed")
}
```

#### Health Check Implementation
```go
type HealthChecker struct {
    container      ContainerProvider
    handlerFactory ProxyHandlerFactory
    checks         map[string]HealthCheck
}

type HealthCheck func() error

func (h *HealthChecker) RegisterCheck(name string, check HealthCheck) {
    h.checks[name] = check
}

func (h *HealthChecker) CheckAll() map[string]error {
    results := make(map[string]error)
    
    // Container 자체 헬스체크
    results["container"] = h.checkContainer()
    
    // 각 핸들러 헬스체크
    for _, handlerType := range h.handlerFactory.SupportedTypes() {
        results[handlerType] = h.checkHandler(handlerType)
    }
    
    // 커스텀 헬스체크
    for name, check := range h.checks {
        results[name] = check()
    }
    
    return results
}

func (h *HealthChecker) checkHandler(handlerType string) error {
    handler, err := h.handlerFactory.CreateHandler(handlerType, h.container)
    if err != nil {
        return fmt.Errorf("handler creation failed: %w", err)
    }
    
    if !handler.IsEnabled() {
        return fmt.Errorf("handler not enabled")
    }
    
    if healthCheck, ok := handler.(interface{ HealthCheck() error }); ok {
        return healthCheck.HealthCheck()
    }
    
    return nil
}
```

이러한 패턴과 베스트 프랙티스를 따르면 확장 가능하고 유지보수가 용이한 Container 기반 시스템을 구축할 수 있습니다.