package middlewares

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/helmet"

	"proxynd/logging"
)

// SecurityMiddlewareConfig 보안 미들웨어 통합 설정
type SecurityMiddlewareConfig struct {
	// Rate Limiting
	EnableRateLimit bool                    `json:"enable_rate_limit"`
	RateLimitConfig EnhancedRateLimitConfig `json:"rate_limit_config"`

	// Input Validation
	EnableInputValidation    bool             `json:"enable_input_validation"`
	ValidationConfig         ValidationConfig `json:"validation_config"`
	EnableEnhancedValidation bool             `json:"enable_enhanced_validation"`

	// Security Headers
	EnableSecurityHeaders bool                  `json:"enable_security_headers"`
	SecurityHeadersConfig SecurityHeadersConfig `json:"security_headers_config"`

	// CORS
	EnableCORS     bool     `json:"enable_cors"`
	AllowedOrigins []string `json:"allowed_origins"`

	// IP Filtering
	EnableIPFiltering bool     `json:"enable_ip_filtering"`
	BlockedCIDRs      []string `json:"blocked_cidrs"`
	AllowedCIDRs      []string `json:"allowed_cidrs"`

	// Logging
	EnableSecurityLogging bool `json:"enable_security_logging"`
}

// DefaultSecurityMiddlewareConfig 기본 보안 미들웨어 설정
func DefaultSecurityMiddlewareConfig() SecurityMiddlewareConfig {
	return SecurityMiddlewareConfig{
		EnableRateLimit:          true,
		RateLimitConfig:          DefaultEnhancedRateLimitConfig(),
		EnableInputValidation:    true,
		ValidationConfig:         DefaultValidationConfig(),
		EnableEnhancedValidation: true,
		EnableSecurityHeaders:    true,
		SecurityHeadersConfig:    DefaultSecurityHeadersConfig(),
		EnableCORS:               false,
		AllowedOrigins:           []string{},
		EnableIPFiltering:        false,
		BlockedCIDRs:             []string{},
		AllowedCIDRs:             []string{},
		EnableSecurityLogging:    true,
	}
}

// ProductionSecurityMiddlewareConfig 프로덕션용 보안 미들웨어 설정
func ProductionSecurityMiddlewareConfig() SecurityMiddlewareConfig {
	return SecurityMiddlewareConfig{
		EnableRateLimit: true,
		RateLimitConfig: EnhancedRateLimitConfig{
			Rate:           "500-H", // 시간당 500개 요청으로 제한
			Burst:          50,
			TrustedProxies: []string{"127.0.0.1", "::1"},
			WhitelistIPs:   []string{},
			BlacklistIPs:   []string{},
			EnableLogging:  true,
			PathConfigs: map[string]PathRateConfig{
				"/auth":  {Rate: "3-M", Burst: 1},    // 인증: 분당 3개 요청
				"/api":   {Rate: "60-M", Burst: 10},  // API: 분당 60개 요청
				"/proxy": {Rate: "300-M", Burst: 30}, // 프록시: 분당 300개 요청
			},
		},
		EnableInputValidation:    true,
		ValidationConfig:         StrictValidationConfig(),
		EnableEnhancedValidation: true,
		EnableSecurityHeaders:    true,
		SecurityHeadersConfig:    ProductionSecurityHeadersConfig(),
		EnableCORS:               false,
		AllowedOrigins:           []string{},
		EnableIPFiltering:        false,
		BlockedCIDRs:             []string{},
		AllowedCIDRs:             []string{},
		EnableSecurityLogging:    true,
	}
}

// SetupSecurityMiddlewares 보안 미들웨어 설정
func SetupSecurityMiddlewares(app *fiber.App, config SecurityMiddlewareConfig) {
	logger := logging.GetLogger()

	if config.EnableSecurityLogging {
		logger.Info("Setting up security middlewares",
			logging.F("rate_limit", config.EnableRateLimit),
			logging.F("input_validation", config.EnableInputValidation),
			logging.F("enhanced_validation", config.EnableEnhancedValidation),
			logging.F("security_headers", config.EnableSecurityHeaders),
			logging.F("cors", config.EnableCORS),
			logging.F("ip_filtering", config.EnableIPFiltering))
	}

	// 1. IP 필터링 (가장 먼저)
	if config.EnableIPFiltering && (len(config.BlockedCIDRs) > 0 || len(config.AllowedCIDRs) > 0) {
		app.Use(IPValidation(config.BlockedCIDRs, config.AllowedCIDRs))
		if config.EnableSecurityLogging {
			logger.Info("IP filtering enabled",
				logging.F("blocked_cidrs", len(config.BlockedCIDRs)),
				logging.F("allowed_cidrs", len(config.AllowedCIDRs)))
		}
	}

	// 2. Rate Limiting
	if config.EnableRateLimit {
		app.Use(NewEnhancedRateLimiter(config.RateLimitConfig))
		if config.EnableSecurityLogging {
			logger.Info("Rate limiting enabled",
				logging.F("rate", config.RateLimitConfig.Rate),
				logging.F("burst", config.RateLimitConfig.Burst))
		}
	}

	// 3. Security Headers
	if config.EnableSecurityHeaders {
		app.Use(SecurityHeaders(config.SecurityHeadersConfig))
		if config.EnableSecurityLogging {
			logger.Info("Security headers enabled")
		}
	}

	// 4. CORS (보안 헤더 다음에)
	if config.EnableCORS {
		if len(config.AllowedOrigins) > 0 {
			app.Use(CORSSecurityHeaders(config.AllowedOrigins))
		} else {
			// 기본 CORS 설정
			app.Use(cors.New(cors.Config{
				AllowOrigins:     "*",
				AllowMethods:     "GET,POST,HEAD,PUT,DELETE,PATCH,OPTIONS",
				AllowHeaders:     "Origin,Content-Type,Accept,Authorization,X-Requested-With",
				AllowCredentials: false,
				MaxAge:           3600,
			}))
		}
		if config.EnableSecurityLogging {
			logger.Info("CORS enabled",
				logging.F("allowed_origins", len(config.AllowedOrigins)))
		}
	}

	// 5. Input Validation
	if config.EnableInputValidation {
		if config.EnableEnhancedValidation {
			app.Use(EnhancedInputValidation(config.ValidationConfig))
			if config.EnableSecurityLogging {
				logger.Info("Enhanced input validation enabled")
			}
		} else {
			app.Use(InputValidation(config.ValidationConfig))
			if config.EnableSecurityLogging {
				logger.Info("Basic input validation enabled")
			}
		}
	}

	// 6. Helmet (추가 보안 헤더)
	app.Use(helmet.New())

	if config.EnableSecurityLogging {
		logger.Info("Security middlewares setup completed")
	}
}

// SetupAPISecurityMiddlewares API 전용 보안 미들웨어 설정
func SetupAPISecurityMiddlewares(apiGroup fiber.Router, config SecurityMiddlewareConfig) {
	logger := logging.GetLogger()

	// API 전용 보안 헤더
	apiGroup.Use(APISecurityHeaders())

	// 엄격한 Rate Limiting
	strictRateConfig := config.RateLimitConfig
	strictRateConfig.Rate = "100-H" // 시간당 100개로 제한
	strictRateConfig.Burst = 10
	apiGroup.Use(NewEnhancedRateLimiter(strictRateConfig))

	// 향상된 입력 검증
	apiGroup.Use(EnhancedInputValidation(StrictValidationConfig()))

	if config.EnableSecurityLogging {
		logger.Info("API security middlewares setup completed")
	}
}

// SetupAuthSecurityMiddlewares 인증 엔드포인트 전용 보안 미들웨어
func SetupAuthSecurityMiddlewares(authGroup fiber.Router, config SecurityMiddlewareConfig) {
	logger := logging.GetLogger()

	// 인증 전용 보안 헤더
	authGroup.Use(APISecurityHeaders())

	// 매우 엄격한 Rate Limiting
	authGroup.Use(AuthRateLimiter())

	// 최대한 엄격한 입력 검증
	strictConfig := StrictValidationConfig()
	strictConfig.MaxParamLength = 50
	strictConfig.MaxQueryLength = 200
	authGroup.Use(EnhancedInputValidation(strictConfig))

	if config.EnableSecurityLogging {
		logger.Info("Auth security middlewares setup completed")
	}
}

// SetupProxySecurityMiddlewares 프록시 엔드포인트 전용 보안 미들웨어
func SetupProxySecurityMiddlewares(proxyGroup fiber.Router, config SecurityMiddlewareConfig) {
	logger := logging.GetLogger()

	// 공개 컨텐츠용 보안 헤더
	proxyGroup.Use(PublicSecurityHeaders())

	// 관대한 Rate Limiting
	proxyGroup.Use(ProxyRateLimiter())

	// 기본 입력 검증 (파일 업로드 고려)
	proxyGroup.Use(InputValidation(config.ValidationConfig))

	if config.EnableSecurityLogging {
		logger.Info("Proxy security middlewares setup completed")
	}
}
