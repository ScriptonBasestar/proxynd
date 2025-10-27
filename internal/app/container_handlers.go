package app

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/gofiber/fiber/v2"

	"proxynd/internal/adapters/observability/zap"
	"proxynd/internal/adapters/pm/ansible"
	ansibleHandler "proxynd/internal/adapters/http/fiber/handlers/ansible"
	casStorage "proxynd/internal/adapters/storage/cas"
	"proxynd/internal/config"
	"proxynd/internal/container"
	"proxynd/internal/containerhandlers"
	"proxynd/internal/factory"
	"proxynd/internal/handlers"
	"proxynd/internal/logging"
	"proxynd/internal/metrics"
	sqliteRepo "proxynd/internal/repositories/metadata/sqlite"
	"proxynd/internal/types"
)

// GetHandlerFactory returns the proxy handler factory
func (c *Container) GetHandlerFactory() (types.ProxyHandlerFactory, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.handlerFactory != nil {
		return c.handlerFactory, nil
	}

	// Get service factory
	_, err := c.GetServiceFactory()
	if err != nil {
		return nil, err
	}

	// Create handler factory
	factory := types.NewStandardProxyHandlerFactory()

	// Register all handlers (commented out until handlers are implemented)
	/*
		// Maven
		if err := factory.RegisterHandler(
			types.ProxyTypeMaven,
			proxyHandlers.MavenHandlerCreator(serviceFactory),
		); err != nil {
			return nil, fmt.Errorf("failed to register Maven handler: %w", err)
		}

		// APT
		if err := factory.RegisterHandler(
			types.ProxyTypeAPT,
			proxyHandlers.AptHandlerCreator(serviceFactory),
		); err != nil {
			return nil, fmt.Errorf("failed to register APT handler: %w", err)
		}

		// NPM
		if err := factory.RegisterHandler(
			types.ProxyTypeNPM,
			proxyHandlers.NpmHandlerCreator(serviceFactory),
		); err != nil {
			return nil, fmt.Errorf("failed to register NPM handler: %w", err)
		}
	*/

	// TODO: Register other handlers (Docker, PIP, YUM, APK)

	c.handlerFactory = factory
	return factory, nil
}

// GetHandlerAdapterFactory returns the handler adapter factory instance
func (c *Container) GetHandlerAdapterFactory() (*factory.HandlerAdapterFactory, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.handlerAdapterFactory != nil {
		return c.handlerAdapterFactory, nil
	}

	// Create handler adapter factory
	c.handlerAdapterFactory = factory.NewHandlerAdapterFactory()

	c.logger.Info("Handler adapter factory created")
	return c.handlerAdapterFactory, nil
}

// GetContainerProxyHandlerFactory Container 기반 프록시 핸들러 팩토리 반환
func (c *Container) GetContainerProxyHandlerFactory() (*handlers.StandardProxyHandlerFactory, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 싱글톤에서 확인
	if cached, exists := c.singletons["container-proxy-handler-factory"]; exists {
		if factory, ok := cached.(*handlers.StandardProxyHandlerFactory); ok {
			return factory, nil
		}
	}

	// 새로운 팩토리 생성
	factory := handlers.NewStandardProxyHandlerFactory()

	// Container 기반 핸들러들 등록
	if err := c.registerContainerProxyHandlers(factory); err != nil {
		return nil, fmt.Errorf("failed to register container proxy handlers: %w", err)
	}

	// 싱글톤에 저장
	c.singletons["container-proxy-handler-factory"] = factory

	c.logger.Info("Container proxy handler factory created with handlers",
		logging.F("supported_types", factory.SupportedTypes()),
	)

	return factory, nil
}

// registerContainerProxyHandlers Container 기반 프록시 핸들러들을 팩토리에 등록
func (c *Container) registerContainerProxyHandlers(factory *handlers.StandardProxyHandlerFactory) error {
	containerMetrics := metrics.GetContainerMetrics()
	// APT 핸들러 등록
	if err := factory.RegisterHandler("apt", func(provider container.ContainerProvider) (handlers.ContainerProxyHandler, error) { //nolint:lll
		containerMetrics.RecordHandlerFactoryOperation("create", "apt", true)
		// 여기서 별도 패키지의 핸들러를 생성하고 인터페이스로 반환
		return &containerHandlerAdapter{
			name:      "apt-container-handler",
			proxyType: "apt",
			provider:  provider,
		}, nil
	}); err != nil {
		containerMetrics.RecordHandlerFactoryOperation("register", "apt", false)
		return fmt.Errorf("failed to register APT handler: %w", err)
	}
	containerMetrics.RecordHandlerFactoryOperation("register", "apt", true)

	// Maven 핸들러 등록
	if err := factory.RegisterHandler("maven", func(provider container.ContainerProvider) (handlers.ContainerProxyHandler, error) { //nolint:lll
		containerMetrics.RecordHandlerFactoryOperation("create", "maven", true)
		return &containerHandlerAdapter{
			name:      "maven-container-handler",
			proxyType: "maven",
			provider:  provider,
		}, nil
	}); err != nil {
		containerMetrics.RecordHandlerFactoryOperation("register", "maven", false)
		return fmt.Errorf("failed to register Maven handler: %w", err)
	}
	containerMetrics.RecordHandlerFactoryOperation("register", "maven", true)

	// NPM 핸들러 등록
	if err := factory.RegisterHandler("npm", func(provider container.ContainerProvider) (handlers.ContainerProxyHandler, error) { //nolint:lll
		containerMetrics.RecordHandlerFactoryOperation("create", "npm", true)
		return &containerHandlerAdapter{
			name:      "npm-container-handler",
			proxyType: "npm",
			provider:  provider,
		}, nil
	}); err != nil {
		containerMetrics.RecordHandlerFactoryOperation("register", "npm", false)
		return fmt.Errorf("failed to register NPM handler: %w", err)
	}
	containerMetrics.RecordHandlerFactoryOperation("register", "npm", true)

	// Docker 핸들러 등록
	if err := factory.RegisterHandler("docker", func(provider container.ContainerProvider) (handlers.ContainerProxyHandler, error) { //nolint:lll
		containerMetrics.RecordHandlerFactoryOperation("create", "docker", true)
		return &containerHandlerAdapter{
			name:      "docker-container-handler",
			proxyType: "docker",
			provider:  provider,
		}, nil
	}); err != nil {
		containerMetrics.RecordHandlerFactoryOperation("register", "docker", false)
		return fmt.Errorf("failed to register Docker handler: %w", err)
	}
	containerMetrics.RecordHandlerFactoryOperation("register", "docker", true)

	// PIP 핸들러 등록
	if err := factory.RegisterHandler("pip", func(provider container.ContainerProvider) (handlers.ContainerProxyHandler, error) { //nolint:lll
		containerMetrics.RecordHandlerFactoryOperation("create", "pip", true)
		return &containerHandlerAdapter{
			name:      "pip-container-handler",
			proxyType: "pip",
			provider:  provider,
		}, nil
	}); err != nil {
		containerMetrics.RecordHandlerFactoryOperation("register", "pip", false)
		return fmt.Errorf("failed to register PIP handler: %w", err)
	}
	containerMetrics.RecordHandlerFactoryOperation("register", "pip", true)

	// YUM 핸들러 등록
	if err := factory.RegisterHandler("yum", func(provider container.ContainerProvider) (handlers.ContainerProxyHandler, error) { //nolint:lll
		containerMetrics.RecordHandlerFactoryOperation("create", "yum", true)
		return &containerHandlerAdapter{
			name:      "yum-container-handler",
			proxyType: "yum",
			provider:  provider,
		}, nil
	}); err != nil {
		containerMetrics.RecordHandlerFactoryOperation("register", "yum", false)
		return fmt.Errorf("failed to register YUM handler: %w", err)
	}
	containerMetrics.RecordHandlerFactoryOperation("register", "yum", true)

	// APK 핸들러 등록
	if err := factory.RegisterHandler("apk", func(provider container.ContainerProvider) (handlers.ContainerProxyHandler, error) { //nolint:lll
		containerMetrics.RecordHandlerFactoryOperation("create", "apk", true)
		return &containerHandlerAdapter{
			name:      "apk-container-handler",
			proxyType: "apk",
			provider:  provider,
		}, nil
	}); err != nil {
		containerMetrics.RecordHandlerFactoryOperation("register", "apk", false)
		return fmt.Errorf("failed to register APK handler: %w", err)
	}
	containerMetrics.RecordHandlerFactoryOperation("register", "apk", true)

	return nil
}

// containerHandlerAdapter Container 핸들러 어댑터
type containerHandlerAdapter struct {
	name      string
	proxyType string
	provider  container.ContainerProvider
}

func (a *containerHandlerAdapter) Handle(c *fiber.Ctx) error {
	// 실제 Container 기반 핸들러 생성 및 호출
	switch a.proxyType {
	case "apt":
		handler := containerhandlers.NewAPTContainerHandler(a.provider)
		return handler.Handle(c)
	case "maven":
		handler := containerhandlers.NewMavenContainerHandler(a.provider)
		return handler.Handle(c)
	case "npm":
		handler := containerhandlers.NewNPMContainerHandler(a.provider)
		return handler.Handle(c)
	case "docker":
		handler := containerhandlers.NewDockerContainerHandler(a.provider)
		return handler.Handle(c)
	case "pip":
		handler := containerhandlers.NewPIPContainerHandler(a.provider)
		return handler.Handle(c)
	case "yum":
		handler := containerhandlers.NewYUMContainerHandler(a.provider)
		return handler.Handle(c)
	case "apk":
		handler := containerhandlers.NewAPKContainerHandler(a.provider)
		return handler.Handle(c)
	default:
		return c.Status(fiber.StatusNotImplemented).SendString(
			fmt.Sprintf("%s handler not implemented yet", a.proxyType),
		)
	}
}

func (a *containerHandlerAdapter) Name() string {
	return a.name
}

func (a *containerHandlerAdapter) Type() string {
	return a.proxyType
}

func (a *containerHandlerAdapter) SetContainer(provider container.ContainerProvider) {
	a.provider = provider
}

func (a *containerHandlerAdapter) GetContainer() container.ContainerProvider {
	return a.provider
}

func (a *containerHandlerAdapter) LoadConfig() error {
	return nil
}

func (a *containerHandlerAdapter) ReloadConfig() error {
	return nil
}

func (a *containerHandlerAdapter) IsEnabled() bool {
	return true
}

func (a *containerHandlerAdapter) GenerateCacheKey(c *fiber.Ctx) string {
	return fmt.Sprintf("%s_%s_%s", a.proxyType, c.Method(), c.Path())
}

func (a *containerHandlerAdapter) BuildUpstreamURL(c *fiber.Ctx) (string, error) {
	return "", fmt.Errorf("not implemented")
}

func (a *containerHandlerAdapter) TransformRequest(c *fiber.Ctx, upstreamReq *fiber.Agent) error {
	return nil
}

func (a *containerHandlerAdapter) TransformResponse(resp []byte, c *fiber.Ctx) ([]byte, error) {
	return resp, nil
}

func (a *containerHandlerAdapter) ShouldCache(c *fiber.Ctx, statusCode int) bool {
	return statusCode == fiber.StatusOK
}

func (a *containerHandlerAdapter) GetCacheTTL(c *fiber.Ctx) time.Duration {
	return time.Hour
}

func (a *containerHandlerAdapter) HandleError(err error, c *fiber.Ctx) error {
	return err
}

func (a *containerHandlerAdapter) HealthCheck() error {
	return nil
}

func (a *containerHandlerAdapter) IsCacheable(c *fiber.Ctx) bool {
	return c.Method() == fiber.MethodGet
}

func (a *containerHandlerAdapter) GetCacheKey(c *fiber.Ctx) string {
	return a.GenerateCacheKey(c)
}

func (a *containerHandlerAdapter) GetUpstreamURL(c *fiber.Ctx) (string, error) {
	return a.BuildUpstreamURL(c)
}

func (a *containerHandlerAdapter) ModifyRequest(c *fiber.Ctx) error {
	return nil
}

func (a *containerHandlerAdapter) ModifyResponse(c *fiber.Ctx) error {
	return nil
}

// GetUnifiedRouter returns the unified proxy router (placeholder)
func (c *Container) GetUnifiedRouter() (interface{}, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// TODO: Implement unified router when handlers are ready
	/*
		if c.unifiedRouter != nil {
			return c.unifiedRouter, nil
		}

		// Get service factory
		serviceFactory, err := c.GetServiceFactory()
		if err != nil {
			return nil, err
		}

		// Create unified router
		c.unifiedRouter = proxyHandlers.NewUnifiedProxyRouter(serviceFactory)
	*/
	return nil, fmt.Errorf("unified router not implemented yet")
}

// GetAnsibleHandler returns the Ansible Collections handler
func (c *Container) GetAnsibleHandler() (*ansibleHandler.AnsibleHandler, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check singleton cache
	if cached, exists := c.singletons["ansible-handler"]; exists {
		if handler, ok := cached.(*ansibleHandler.AnsibleHandler); ok {
			return handler, nil
		}
	}

	// 1. Initialize Zap logger for structured logging (implements ports.Logger)
	zapLogger, err := zap.NewLogger(zap.DevelopmentLoggerConfig())
	if err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %w", err)
	}

	// 2. Initialize LocalCAS for content-addressable storage
	casDir := filepath.Join(c.config.StorageDir, "ansible", "cas")
	casInstance, err := casStorage.NewLocalCAS(casDir, zapLogger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize CAS: %w", err)
	}
	c.logger.Info("Ansible LocalCAS initialized", logging.F("path", casDir))

	// 3. Initialize SQLite metadata repository
	dbPath := filepath.Join(c.config.StorageDir, "ansible", "metadata.db")
	metadataRepo, err := sqliteRepo.NewRepository(dbPath, zapLogger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize metadata repository: %w", err)
	}
	c.logger.Info("Ansible metadata repository initialized", logging.F("path", dbPath))

	// 4. Create Ansible repository configuration
	ansibleCfg := &config.AnsibleRepositoryConfig{
		Name:              "ansible-collections",
		Type:              "hosted",
		Enabled:           true,
		MaxUploadSize:     10 * 1024 * 1024, // 10MB default
		AllowedNamespaces: []string{"*"},    // Allow all namespaces
		RequireAuth:       false,
	}

	// 5. Create HostedDriver with all dependencies
	hostedDriver, err := ansible.NewHostedDriver(casInstance, metadataRepo, ansibleCfg, zapLogger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize HostedDriver: %w", err)
	}
	c.logger.Info("Ansible HostedDriver initialized")

	// 6. Create AnsibleHandler
	handler := ansibleHandler.NewAnsibleHandler(hostedDriver, ansibleCfg, zapLogger)
	c.logger.Info("Ansible handler created successfully")

	// 7. Store in singletons for caching for re-use on subsequent calls
	c.singletons["ansible-handler"] = handler
	c.singletons["ansible-cas"] = casInstance
	c.singletons["ansible-metadata-repo"] = metadataRepo

	return handler, nil
}
