package adapters

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"proxynd/configs"
	"proxynd/internal/plugins"
	"proxynd/logging"
)

// NPMHandlerAdapter NPM 핸들러를 플러그인 인터페이스로 어댑팅
type NPMHandlerAdapter struct {
	*plugins.BasePackageHandler
	config      *configs.NpmProxyConfig
	groupManager plugins.GroupManager
}

// NPMPlugin NPM 플러그인 구현
type NPMPlugin struct {
	metadata plugins.PluginMetadata
}

// NewNPMPlugin 새로운 NPM 플러그인 생성
func NewNPMPlugin() plugins.Plugin {
	return &NPMPlugin{
		metadata: plugins.PluginMetadata{
			Type:        "npm",
			Name:        "NPM Package Manager",
			Version:     "1.0.0",
			Description: "NPM registry proxy and mirror support",
			Author:      "ProxyND Team",
			License:     "MIT",
			Tags:        []string{"npm", "nodejs", "javascript", "package-manager"},
			Config: map[string]string{
				"default_registry": "https://registry.npmjs.org",
				"cache_ttl":       "24h",
				"max_package_size": "100MB",
			},
		},
	}
}

// GetMetadata 플러그인 메타데이터 반환
func (p *NPMPlugin) GetMetadata() plugins.PluginMetadata {
	return p.metadata
}

// CreateHandler 핸들러 생성
func (p *NPMPlugin) CreateHandler() plugins.PackageHandler {
	handler := &NPMHandlerAdapter{
		BasePackageHandler: plugins.NewBasePackageHandler("npm", "NPM Handler", "1.0.0"),
	}

	// NPM은 프록시와 미러 모드 모두 지원
	handler.SetSupportedModes(map[plugins.OperationMode]bool{
		plugins.ProxyMode:  true,
		plugins.MirrorMode: true,
	})

	// NPM 특화 캐시 정책 설정
	handler.SetCachePolicy(plugins.CachePolicy{
		TTL:              24 * time.Hour,
		MaxSize:          500 * 1024 * 1024, // 500MB
		EnableGzipCache:  true,
		CacheHeaders:     []string{"Content-Type", "ETag", "Last-Modified", "npm-signature"},
		SkipCacheHeaders: []string{"Authorization", "npm-token"},
	})

	return handler
}

// Load 플러그인 로드
func (p *NPMPlugin) Load() error {
	return nil
}

// Unload 플러그인 언로드
func (p *NPMPlugin) Unload() error {
	return nil
}

// GetDependencies 의존성 반환
func (p *NPMPlugin) GetDependencies() []string {
	return []string{} // NPM은 의존성 없음
}

// CheckDependencies 의존성 확인
func (p *NPMPlugin) CheckDependencies() error {
	return nil
}

// Initialize NPM 핸들러 초기화
func (h *NPMHandlerAdapter) Initialize(config interface{}) error {
	if err := h.BasePackageHandler.Initialize(config); err != nil {
		return err
	}

	// NPM 설정 타입 변환
	npmConfig, ok := config.(*configs.NpmProxyConfig)
	if !ok {
		return fmt.Errorf("invalid NPM config type")
	}

	h.config = npmConfig

	// 그룹 매니저 초기화
	h.groupManager = plugins.NewGroupManager(plugins.GroupManagerConfig{
		Timeout:   30 * time.Second,
		MaxRetry:  3,
		UserAgent: "ProxyND-NPM/1.0",
	})

	h.GetLogger().Info("NPM handler initialized",
		logging.F("default_registry", npmConfig.DefaultRegistry),
		logging.F("cache_enabled", npmConfig.CacheEnabled))

	return nil
}

// HandleRequest NPM 요청 처리
func (h *NPMHandlerAdapter) HandleRequest(ctx *fiber.Ctx, mode plugins.OperationMode) error {
	if err := h.BasePackageHandler.HandleRequest(ctx, mode); err != nil && err != plugins.ErrNotImplemented {
		return err
	}

	path := ctx.Path()
	h.GetLogger().Debug("Handling NPM request",
		logging.F("path", path),
		logging.F("mode", string(mode)),
		logging.F("method", ctx.Method()))

	switch mode {
	case plugins.ProxyMode:
		return h.handleProxyRequest(ctx)
	case plugins.MirrorMode:
		return h.handleMirrorRequest(ctx)
	default:
		return fmt.Errorf("unsupported mode: %s", mode)
	}
}

// ValidateConfig NPM 설정 검증
func (h *NPMHandlerAdapter) ValidateConfig(config interface{}) error {
	npmConfig, ok := config.(*configs.NpmProxyConfig)
	if !ok {
		return fmt.Errorf("config must be *configs.NpmProxyConfig")
	}

	if npmConfig.DefaultRegistry == "" {
		return fmt.Errorf("default_registry is required")
	}

	if !strings.HasPrefix(npmConfig.DefaultRegistry, "http://") && 
	   !strings.HasPrefix(npmConfig.DefaultRegistry, "https://") {
		return fmt.Errorf("default_registry must be a valid HTTP/HTTPS URL")
	}

	return nil
}

// GetDefaultConfig NPM 기본 설정 반환
func (h *NPMHandlerAdapter) GetDefaultConfig() interface{} {
	return &configs.NpmProxyConfig{
		DefaultRegistry: "https://registry.npmjs.org",
		CacheEnabled:    true,
		CacheTTL:        24 * time.Hour,
		MaxPackageSize:  100 * 1024 * 1024, // 100MB
		Timeout:         30 * time.Second,
		RetryCount:      3,
		EnableGzip:      true,
		UserAgent:       "ProxyND-NPM/1.0",
	}
}

// HealthCheck NPM 핸들러 헬스체크
func (h *NPMHandlerAdapter) HealthCheck(ctx context.Context) error {
	if err := h.BasePackageHandler.HealthCheck(ctx); err != nil {
		return err
	}

	if h.config == nil {
		return fmt.Errorf("NPM config not initialized")
	}

	// 기본 레지스트리 연결 테스트
	upstreams := []plugins.UpstreamConfig{
		{
			URL:     h.config.DefaultRegistry,
			Timeout: 5 * time.Second,
		},
	}

	healthResults, err := h.groupManager.HealthCheckGroup(ctx, upstreams)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	if !healthResults[h.config.DefaultRegistry] {
		return fmt.Errorf("default registry is not healthy")
	}

	return nil
}

// handleProxyRequest 프록시 모드 요청 처리
func (h *NPMHandlerAdapter) handleProxyRequest(ctx *fiber.Ctx) error {
	path := strings.TrimPrefix(ctx.Path(), "/proxy/npm")
	
	// NPM 레지스트리별 업스트림 설정
	upstreams := []plugins.UpstreamConfig{
		{
			URL:      h.config.DefaultRegistry,
			Priority: 1,
			Timeout:  h.config.Timeout,
			Headers: map[string]string{
				"User-Agent": h.config.UserAgent,
				"Accept":     "application/json",
			},
		},
	}

	// 추가 레지스트리가 설정된 경우
	for i, registry := range h.config.AlternativeRegistries {
		upstreams = append(upstreams, plugins.UpstreamConfig{
			URL:      registry,
			Priority: i + 2,
			Timeout:  h.config.Timeout,
			Headers: map[string]string{
				"User-Agent": h.config.UserAgent,
				"Accept":     "application/json",
			},
		})
	}

	// 그룹에서 데이터 가져오기
	results, err := h.groupManager.FetchOnDemand(ctx.Context(), path, upstreams)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, 
			fmt.Sprintf("Failed to fetch from upstream: %v", err))
	}

	// 첫 번째 성공한 결과 반환
	for _, result := range results {
		if result.Error == nil && result.StatusCode == 200 {
			// 응답 헤더 설정
			for key, value := range result.Headers {
				ctx.Set(key, value)
			}

			// NPM 특화 헤더 추가
			ctx.Set("X-ProxyND-Cache", "MISS")
			ctx.Set("X-ProxyND-Upstream", result.UpstreamURL)

			return ctx.Status(result.StatusCode).Send(result.Body)
		}
	}

	return fiber.NewError(fiber.StatusBadGateway, "All upstreams failed")
}

// handleMirrorRequest 미러 모드 요청 처리
func (h *NPMHandlerAdapter) handleMirrorRequest(ctx *fiber.Ctx) error {
	// 미러 모드에서는 로컬 캐시에서 먼저 조회
	cacheKey := h.BuildCacheKey(ctx)
	
	h.GetLogger().Debug("Handling mirror request",
		logging.F("cache_key", cacheKey),
		logging.F("path", ctx.Path()))

	// TODO: 캐시에서 조회 로직 구현
	// 캐시에 없으면 미러 동기화 트리거

	return fiber.NewError(fiber.StatusNotImplemented, "Mirror mode not yet implemented")
}

// GetSupportedPackageTypes NPM이 지원하는 패키지 타입들
func (h *NPMHandlerAdapter) GetSupportedPackageTypes() []string {
	return []string{
		"@scope/package",  // 스코프 패키지
		"package",         // 일반 패키지
		"@types/package",  // TypeScript 타입 정의
	}
}

// IsValidNPMPackageName NPM 패키지명 유효성 검사
func (h *NPMHandlerAdapter) IsValidNPMPackageName(name string) bool {
	if len(name) == 0 || len(name) > 214 {
		return false
	}

	// NPM 패키지명 규칙 검사
	// - 소문자만 허용
	// - 하이픈, 언더스코어 허용
	// - 스코프 패키지 (@scope/package) 허용
	validChars := "abcdefghijklmnopqrstuvwxyz0123456789-_./@"
	
	for _, char := range name {
		found := false
		for _, validChar := range validChars {
			if char == validChar {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}