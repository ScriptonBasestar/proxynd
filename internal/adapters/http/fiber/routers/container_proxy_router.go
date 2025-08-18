package routers

import (
	"fmt"

	"github.com/gofiber/fiber/v2"

	"proxynd/alerts"
	"proxynd/internal/config"
	"proxynd/internal/container"
	"proxynd/internal/handlers"
	authHandlers "proxynd/internal/handlers-legacy/auth"
	proxyHandlers "proxynd/internal/handlers-legacy/proxy"
	"proxynd/internal/logging"
	middlewares "proxynd/internal/middleware-legacy"
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
	case "apt":
		return r.createAPTHandler(provider)
	case "maven":
		return r.createMavenHandler(provider)
	case "npm":
		return r.createNPMHandler(provider)
	case "docker":
		return r.createDockerHandler(provider)
	case "pip":
		return r.createPIPHandler(provider)
	default:
		return nil, fmt.Errorf("unsupported proxy type: %s", proxyType)
	}
}

// 핸들러 생성 함수들 (플레이스홀더)
func (r *ContainerProxyRouter) createAPTHandler(
	provider container.ContainerProvider,
) (handlers.ContainerProxyHandler, error) {
	// TODO: 실제 APT 핸들러 생성 로직 구현
	return nil, fmt.Errorf("APT handler not implemented yet")
}

func (r *ContainerProxyRouter) createMavenHandler(
	provider container.ContainerProvider,
) (handlers.ContainerProxyHandler, error) {
	// TODO: 실제 Maven 핸들러 생성 로직 구현
	return nil, fmt.Errorf("maven handler not implemented yet")
}

func (r *ContainerProxyRouter) createNPMHandler(
	provider container.ContainerProvider,
) (handlers.ContainerProxyHandler, error) {
	// TODO: 실제 NPM 핸들러 생성 로직 구현
	return nil, fmt.Errorf("NPM handler not implemented yet")
}

func (r *ContainerProxyRouter) createDockerHandler(
	provider container.ContainerProvider,
) (handlers.ContainerProxyHandler, error) {
	// TODO: 실제 Docker 핸들러 생성 로직 구현
	return nil, fmt.Errorf("docker handler not implemented yet")
}

func (r *ContainerProxyRouter) createPIPHandler(
	provider container.ContainerProvider,
) (handlers.ContainerProxyHandler, error) {
	// TODO: 실제 PIP 핸들러 생성 로직 구현
	return nil, fmt.Errorf("PIP handler not implemented yet")
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
			r.logger.Warn("Unsupported proxy type in legacy route",
				logging.F("type", proxyType),
			)
			// 기존 UnifiedProxyHandler로 폴백
			return proxyHandlers.UnifiedProxyHandler(c)
		}

		// Container 기반 핸들러 사용
		handler, err := r.handlerFactory.CreateHandler(proxyType, r.container)
		if err != nil {
			r.logger.Error("Failed to create handler in legacy route",
				logging.F("type", proxyType),
				logging.F("error", err),
			)
			// 기존 UnifiedProxyHandler로 폴백
			return proxyHandlers.UnifiedProxyHandler(c)
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
