package routers

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"

	"proxynd/internal/alerts"
	"proxynd/internal/config"
	"proxynd/internal/container"
	"proxynd/internal/handlers"
	authHandlers "proxynd/internal/adapters/http/fiber/handlers/auth"
	proxyHandlers "proxynd/internal/adapters/http/fiber/handlers/proxy"
	"proxynd/internal/logging"
	middlewares "proxynd/internal/adapters/http/fiber/middleware"
)

// Constants for package managers
const (
	pmAPT   = "apt"
	pmMaven = "maven"
	pmNPM   = "npm"
)

// ContainerProxyRouter Container 기반 프록시 라우터
type ContainerProxyRouter struct {
	container           container.ContainerProvider
	handlerFactory      handlers.ProxyHandlerFactory
	logger              logging.Logger
	alertManager        alerts.AlertManager
	verificationHandler *proxyHandlers.VerificationHandler
}

// NewContainerProxyRouter 새로운 Container 기반 프록시 라우터 생성
func NewContainerProxyRouter(container container.ContainerProvider) *ContainerProxyRouter {
	router := &ContainerProxyRouter{
		container:      container,
		handlerFactory: handlers.NewStandardProxyHandlerFactory(),
		logger:         logging.GetLogger(),
	}

	// Alert 매니저 초기화
	router.initializeAlertManager()

	// 핸들러 등록
	router.registerHandlers()

	return router
}

// initializeAlertManager Alert 매니저 초기화
func (r *ContainerProxyRouter) initializeAlertManager() {
	// Global config 읽기
	globalConfig := config.GlobalConfig{}
	if err := globalConfig.ReadConfig(); err != nil {
		r.logger.Warn("Failed to read global config", logging.F("error", err))
	}

	// Alert 설정 로드
	alertConfig := loadContainerAlertConfig()
	r.alertManager = alerts.NewAlertManager(alertConfig)

	// Log alerter 등록
	if logAlerter, err := alerts.NewLogAlerter(alerts.LogAlerterConfig{
		Enabled:    true,
		LogFile:    "./logs/verification-alerts.log",
		JSONFormat: true,
	}); err == nil {
		r.alertManager.RegisterAlerter(logAlerter)
	}

	// Verification 핸들러 생성
	r.verificationHandler = proxyHandlers.NewVerificationHandler(&globalConfig, r.alertManager)
}

// registerHandlers 핸들러 등록
func (r *ContainerProxyRouter) registerHandlers() {
	// 핸들러들을 동적으로 등록 (순환 참조 방지)
	handlerTypes := []string{"apt", "maven", "npm", "docker", "pip", "yum", "apk"}

	for _, handlerType := range handlerTypes {
		proxyType := handlerType // 클로저 변수 캡처

		if err := r.handlerFactory.RegisterHandler(proxyType,
			func(provider container.ContainerProvider) (handlers.ContainerProxyHandler, error) {
				// 동적으로 핸들러 생성 (실제 구현은 나중에)
				return r.createHandlerByType(proxyType, provider)
			}); err != nil {
			r.logger.Error("Failed to register handler",
				logging.F("type", proxyType),
				logging.F("error", err),
			)
		}
	}

	r.logger.Info("Container proxy handlers registered",
		logging.F("handlers", r.handlerFactory.SupportedTypes()),
	)
}

// createHandlerByType 타입별 핸들러 생성 (순환 참조 방지)
func (r *ContainerProxyRouter) createHandlerByType(
	proxyType string,
	provider container.ContainerProvider,
) (handlers.ContainerProxyHandler, error) {
	switch proxyType {
	case pmAPT:
		return r.createAPTHandler(provider)
	case pmMaven:
		return r.createMavenHandler(provider)
	case pmNPM:
		return r.createNPMHandler(provider)
	case "docker":
		return r.createDockerHandler(provider)
	case "pip":
		return r.createPIPHandler(provider)
	case "yum":
		return r.createYUMHandler(provider)
	case "apk":
		return r.createAPKHandler(provider)
	default:
		return nil, fmt.Errorf("unsupported proxy type: %s", proxyType)
	}
}

// 핸들러 생성 함수들 - V3 legacy handlers wrapped for Container compatibility
func (r *ContainerProxyRouter) createAPTHandler(
	provider container.ContainerProvider,
) (handlers.ContainerProxyHandler, error) {
	legacyHandler := proxyHandlers.NewAPTHandlerV3()
	return newLegacyHandlerAdapter(legacyHandler, pmAPT, provider), nil
}

func (r *ContainerProxyRouter) createMavenHandler(
	provider container.ContainerProvider,
) (handlers.ContainerProxyHandler, error) {
	legacyHandler := proxyHandlers.NewMavenHandlerV3()
	return newLegacyHandlerAdapter(legacyHandler, pmMaven, provider), nil
}

func (r *ContainerProxyRouter) createNPMHandler(
	provider container.ContainerProvider,
) (handlers.ContainerProxyHandler, error) {
	legacyHandler := proxyHandlers.NewNPMHandlerV3()
	return newLegacyHandlerAdapter(legacyHandler, pmNPM, provider), nil
}

func (r *ContainerProxyRouter) createDockerHandler(
	provider container.ContainerProvider,
) (handlers.ContainerProxyHandler, error) {
	legacyHandler := proxyHandlers.NewDockerHandlerV3()
	return newLegacyHandlerAdapter(legacyHandler, "docker", provider), nil
}

func (r *ContainerProxyRouter) createPIPHandler(
	provider container.ContainerProvider,
) (handlers.ContainerProxyHandler, error) {
	legacyHandler := proxyHandlers.NewPipHandlerV3()
	return newLegacyHandlerAdapter(legacyHandler, "pip", provider), nil
}

func (r *ContainerProxyRouter) createYUMHandler(
	provider container.ContainerProvider,
) (handlers.ContainerProxyHandler, error) {
	legacyHandler := proxyHandlers.NewYumHandlerV3()
	return newLegacyHandlerAdapter(legacyHandler, "yum", provider), nil
}

func (r *ContainerProxyRouter) createAPKHandler(
	provider container.ContainerProvider,
) (handlers.ContainerProxyHandler, error) {
	legacyHandler := proxyHandlers.NewApkHandlerV3()
	return newLegacyHandlerAdapter(legacyHandler, "apk", provider), nil
}

// Setup Container 기반 프록시 라우터 설정
func (r *ContainerProxyRouter) Setup(app *fiber.App) {
	// 새로운 Container 기반 API (v2) - 권장
	r.setupV2Routes(app)

	// 기존 API 호환성 유지 (v1) - Deprecated
	r.setupV1Routes(app)
}

// setupV2Routes 새로운 Container 기반 API 라우트 설정 (v2)
func (r *ContainerProxyRouter) setupV2Routes(app *fiber.App) {
	// API v2 그룹 생성
	v2 := app.Group("/api/v2/proxy")

	// 지원되는 프록시 타입별 라우트 설정
	for _, proxyType := range r.handlerFactory.SupportedTypes() {
		proxyGroup := v2.Group(fmt.Sprintf("/%s", proxyType))

		// GET 요청
		proxyGroup.Get("/*",
			r.createContainerHandlerMiddleware(proxyType),
			middlewares.ProxyPolicyMiddleware(),
			middlewares.DefaultAccessLogMiddleware(),
			authHandlers.OptionalAuth(),
			authHandlers.BasicAuthFallback(),
			r.verificationHandler.VerificationMiddleware(),
		)

		// POST 요청 (Docker registry push 등을 위해)
		proxyGroup.Post("/*",
			r.createContainerHandlerMiddleware(proxyType),
			middlewares.ProxyPolicyMiddleware(),
			middlewares.DefaultAccessLogMiddleware(),
			authHandlers.OptionalAuth(),
			authHandlers.BasicAuthFallback(),
			r.verificationHandler.VerificationMiddleware(),
		)

		// PUT 요청 (Docker registry push 등을 위해)
		proxyGroup.Put("/*",
			r.createContainerHandlerMiddleware(proxyType),
			middlewares.ProxyPolicyMiddleware(),
			middlewares.DefaultAccessLogMiddleware(),
			authHandlers.OptionalAuth(),
			authHandlers.BasicAuthFallback(),
			r.verificationHandler.VerificationMiddleware(),
		)

		// DELETE 요청 (향후 확장을 위해)
		proxyGroup.Delete("/*",
			r.createContainerHandlerMiddleware(proxyType),
			middlewares.ProxyPolicyMiddleware(),
			middlewares.DefaultAccessLogMiddleware(),
			authHandlers.OptionalAuth(),
			authHandlers.BasicAuthFallback(),
			r.verificationHandler.VerificationMiddleware(),
		)

		r.logger.Info("Container proxy routes registered",
			logging.F("type", proxyType),
			logging.F("pattern", fmt.Sprintf("/api/v2/proxy/%s/*", proxyType)),
		)
	}
}

// setupV1Routes 기존 API 호환성 유지 라우트 설정 (v1 - Deprecated)
func (r *ContainerProxyRouter) setupV1Routes(app *fiber.App) {
	// 통합 핸들러를 통한 기존 라우트 유지
	methods := []string{"GET", "POST", "PUT", "DELETE"}

	for _, method := range methods {
		app.Add(method, "/proxy/:type/*",
			r.createDeprecationMiddleware(method, "/proxy/:type/*", "/api/v2/proxy/:type/*"),
			r.createLegacyContainerHandlerMiddleware(),
			middlewares.ProxyPolicyMiddleware(),
			middlewares.DefaultAccessLogMiddleware(),
			authHandlers.OptionalAuth(),
			authHandlers.BasicAuthFallback(),
			r.verificationHandler.VerificationMiddleware(),
		)
	}

	r.logger.Info("Legacy proxy routes registered for backward compatibility")
}

// createContainerHandlerMiddleware Container 기반 핸들러 미들웨어 생성
func (r *ContainerProxyRouter) createContainerHandlerMiddleware(proxyType string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// 핸들러 생성
		handler, err := r.handlerFactory.CreateHandler(proxyType, r.container)
		if err != nil {
			r.logger.Error("Failed to create handler",
				logging.F("type", proxyType),
				logging.F("error", err),
			)
			return c.Status(fiber.StatusInternalServerError).SendString(
				fmt.Sprintf("Failed to create %s handler: %v", proxyType, err),
			)
		}

		// 핸들러 실행
		return handler.Handle(c)
	}
}

// createLegacyContainerHandlerMiddleware 레거시 호환성을 위한 통합 핸들러 미들웨어
func (r *ContainerProxyRouter) createLegacyContainerHandlerMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		proxyType := c.Params("type")

		// 지원되는 프록시 타입인지 확인
		if !r.isProxyTypeSupported(proxyType) {
			r.logger.Warn("Unsupported proxy type",
				logging.F("type", proxyType),
			)
			return c.Status(fiber.StatusBadRequest).SendString(fmt.Sprintf("Unsupported proxy type: %s", proxyType))
		}

		// Container 기반 핸들러 사용
		handler, err := r.handlerFactory.CreateHandler(proxyType, r.container)
		if err != nil {
			r.logger.Error("Failed to create handler",
				logging.F("type", proxyType),
				logging.F("error", err),
			)
			return c.Status(fiber.StatusInternalServerError).SendString(fmt.Sprintf("Failed to create handler: %v", err))
		}

		// 핸들러 실행
		return handler.Handle(c)
	}
}

// createDeprecationMiddleware Deprecation 미들웨어 생성
func (r *ContainerProxyRouter) createDeprecationMiddleware(method, oldPath, newPath string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Deprecation 헤더 추가
		c.Set("Deprecation", "true")
		c.Set("Sunset", "2025-12-31")
		c.Set("Link", fmt.Sprintf("<%s>; rel=\"successor-version\"", newPath))
		c.Set("Warning", fmt.Sprintf("299 - \"Deprecated API. Use %s instead\"", newPath))

		// 로그 기록
		r.logger.Warn("Deprecated API used",
			logging.F("method", method),
			logging.F("old_path", oldPath),
			logging.F("new_path", newPath),
			logging.F("client_ip", c.IP()),
			logging.F("user_agent", c.Get("User-Agent")),
		)

		return c.Next()
	}
}

// isProxyTypeSupported 프록시 타입 지원 여부 확인
func (r *ContainerProxyRouter) isProxyTypeSupported(proxyType string) bool {
	supportedTypes := r.handlerFactory.SupportedTypes()
	for _, supportedType := range supportedTypes {
		if supportedType == proxyType {
			return true
		}
	}
	return false
}

// GetHandlerFactory 핸들러 팩토리 반환 (테스트용)
func (r *ContainerProxyRouter) GetHandlerFactory() handlers.ProxyHandlerFactory {
	return r.handlerFactory
}

// RegisterHandler 핸들러 등록 (동적 등록용)
func (r *ContainerProxyRouter) RegisterHandler(
	proxyType string,
	createFn func(container.ContainerProvider) (handlers.ContainerProxyHandler, error),
) error {
	return r.handlerFactory.RegisterHandler(proxyType, createFn)
}

// GetSupportedTypes 지원되는 프록시 타입 목록 반환
func (r *ContainerProxyRouter) GetSupportedTypes() []string {
	return r.handlerFactory.SupportedTypes()
}

// ContainerProxyRouterSetup Container 기반 프록시 라우터 설정 함수 (기존 패턴 호환)
func ContainerProxyRouterSetup(app *fiber.App, container container.ContainerProvider) {
	router := NewContainerProxyRouter(container)
	router.Setup(app)
}

// loadContainerAlertConfig Alert 설정 로드 (기존 함수 복사)
func loadContainerAlertConfig() *alerts.AlertConfig {
	// 기본 Alert 설정 반환
	return &alerts.AlertConfig{
		Enabled: true,
		Channels: []alerts.ChannelConfig{
			{
				Type:    "log",
				Enabled: true,
				Config: map[string]interface{}{
					"file":   "./logs/verification-alerts.log",
					"format": "json",
				},
			},
		},
	}
}

// legacyHandlerAdapter V3 legacy handler를 ContainerProxyHandler 인터페이스로 wrapping하는 어댑터
// 이 어댑터는 기존 V3 핸들러들이 ContainerProxyHandler 인터페이스를 충족하도록 브릿지 역할을 수행
type legacyHandlerAdapter struct {
	legacyHandler interface{} // V3 handler (APTHandlerV3, MavenHandlerV3, etc.)
	proxyType     string
	container     container.ContainerProvider
	logger        logging.Logger
}

// newLegacyHandlerAdapter 새로운 legacy handler adapter 생성
func newLegacyHandlerAdapter(
	legacyHandler interface{},
	proxyType string,
	provider container.ContainerProvider,
) *legacyHandlerAdapter {
	return &legacyHandlerAdapter{
		legacyHandler: legacyHandler,
		proxyType:     proxyType,
		container:     provider,
		logger:        logging.GetLogger(),
	}
}

// Handle implements handlers.Handler.Handle
func (a *legacyHandlerAdapter) Handle(c *fiber.Ctx) error {
	// V3 핸들러의 Handle 메서드 호출
	type handlerWithHandle interface {
		Handle(c *fiber.Ctx) error
	}

	if h, ok := a.legacyHandler.(handlerWithHandle); ok {
		return h.Handle(c)
	}

	return c.Status(fiber.StatusInternalServerError).SendString(
		fmt.Sprintf("Handler for %s does not implement Handle method", a.proxyType),
	)
}

// Name implements handlers.Handler.Name
func (a *legacyHandlerAdapter) Name() string {
	return fmt.Sprintf("%s-handler-v3", a.proxyType)
}

// Type implements handlers.Handler.Type and handlers.BaseProxyHandler.Type
func (a *legacyHandlerAdapter) Type() string {
	return a.proxyType
}

// SetContainer implements handlers.ContainerAwareHandler.SetContainer
func (a *legacyHandlerAdapter) SetContainer(provider container.ContainerProvider) {
	a.container = provider
}

// GetContainer implements handlers.ContainerAwareHandler.GetContainer
func (a *legacyHandlerAdapter) GetContainer() container.ContainerProvider {
	return a.container
}

// LoadConfig implements handlers.ContainerAwareHandler.LoadConfig
func (a *legacyHandlerAdapter) LoadConfig() error {
	// Legacy handlers don't require explicit config loading
	return nil
}

// ReloadConfig implements handlers.ContainerAwareHandler.ReloadConfig
func (a *legacyHandlerAdapter) ReloadConfig() error {
	// Legacy handlers don't support config reloading
	return nil
}

// IsEnabled implements handlers.BaseProxyHandler.IsEnabled
func (a *legacyHandlerAdapter) IsEnabled() bool {
	// Always enabled for registered handlers
	return true
}

// GenerateCacheKey implements handlers.BaseProxyHandler.GenerateCacheKey
func (a *legacyHandlerAdapter) GenerateCacheKey(c *fiber.Ctx) string {
	// Generate cache key from proxy type and path
	return fmt.Sprintf("%s:%s", a.proxyType, c.Path())
}

// BuildUpstreamURL implements handlers.BaseProxyHandler.BuildUpstreamURL
func (a *legacyHandlerAdapter) BuildUpstreamURL(c *fiber.Ctx) (string, error) {
	// For legacy handlers, URL building is handled internally in Handle()
	// Return a placeholder that indicates the handler manages its own URLs
	return fmt.Sprintf("handled-by-%s-handler", a.proxyType), nil
}

// TransformRequest implements handlers.BaseProxyHandler.TransformRequest
func (a *legacyHandlerAdapter) TransformRequest(c *fiber.Ctx, upstreamReq *fiber.Agent) error {
	// Legacy handlers handle request transformation internally
	return nil
}

// TransformResponse implements handlers.BaseProxyHandler.TransformResponse
func (a *legacyHandlerAdapter) TransformResponse(resp []byte, c *fiber.Ctx) ([]byte, error) {
	// Legacy handlers handle response transformation internally
	return resp, nil
}

// ShouldCache implements handlers.BaseProxyHandler.ShouldCache
func (a *legacyHandlerAdapter) ShouldCache(c *fiber.Ctx, statusCode int) bool {
	// Cache successful responses
	return statusCode >= 200 && statusCode < 300
}

// GetCacheTTL implements handlers.BaseProxyHandler.GetCacheTTL
func (a *legacyHandlerAdapter) GetCacheTTL(c *fiber.Ctx) time.Duration {
	// Default TTL for package artifacts
	return 24 * time.Hour
}

// HandleError implements handlers.BaseProxyHandler.HandleError
func (a *legacyHandlerAdapter) HandleError(err error, c *fiber.Ctx) error {
	a.logger.Error("Handler error",
		logging.F("type", a.proxyType),
		logging.F("error", err),
		logging.F("path", c.Path()),
	)
	return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
		"error": err.Error(),
		"type":  a.proxyType,
	})
}
