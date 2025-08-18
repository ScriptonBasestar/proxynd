package http

import (
	"strings"

	"github.com/gofiber/fiber/v2"

	"proxynd/internal/config"
	"proxynd/internal/domain/maven"
	"proxynd/internal/logging"
	mavenServices "proxynd/internal/services/maven"
)

// MavenBrowserAdapter Fiber HTTP 요청을 도메인 서비스로 연결하는 어댑터
type MavenBrowserAdapter struct {
	browserHandler maven.BrowserHandler
	logger         logging.Logger
}

// NewMavenBrowserAdapter 새로운 Maven 브라우저 어댑터 생성
func NewMavenBrowserAdapter(config config.MavenProxySettings, logger logging.Logger) *MavenBrowserAdapter {
	// 설정을 도메인 인터페이스로 래핑
	proxyConfig := maven.NewDefaultProxyConfig(&config)

	// 도메인 서비스 생성
	browserService := mavenServices.NewBrowserService(proxyConfig, logger)

	return &MavenBrowserAdapter{
		browserHandler: browserService,
		logger:         logger,
	}
}

// Handle Fiber HTTP 요청을 처리하여 도메인 서비스로 전달
func (a *MavenBrowserAdapter) Handle(c *fiber.Ctx) error {
	ctx := c.Context()

	// Fiber Context를 도메인 요청으로 변환
	request := a.fiberToDomainRequest(c)

	a.logger.Info("Maven browser request via adapter",
		logging.F("path", request.Path),
		logging.F("userAgent", request.Headers["User-Agent"]),
		logging.F("remoteIP", c.IP()),
	)

	// 브라우저 요청인지 확인
	if !a.browserHandler.IsBrowserRequest(request) {
		return c.Status(fiber.StatusNotAcceptable).SendString("Browser request required")
	}

	// 도메인 서비스로 요청 전달
	response, err := a.browserHandler.Handle(ctx, request)
	if err != nil {
		a.logger.Error("Browser handler failed", logging.F("error", err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to process request",
		})
	}

	// 도메인 응답을 Fiber 응답으로 변환
	return a.domainToFiberResponse(c, response)
}

// fiberToDomainRequest Fiber Context를 도메인 BrowserRequest로 변환
func (a *MavenBrowserAdapter) fiberToDomainRequest(c *fiber.Ctx) *maven.BrowserRequest {
	// 경로 추출
	path := c.Params("*")
	if path == "" {
		path = "/"
	}

	// 쿼리 파라미터 추출
	queryParams := make(map[string]string)
	c.Request().URI().QueryArgs().VisitAll(func(key, value []byte) {
		queryParams[string(key)] = string(value)
	})

	// 헤더 추출
	headers := make(map[string]string)
	c.Request().Header.VisitAll(func(key, value []byte) {
		headers[string(key)] = string(value)
	})

	return &maven.BrowserRequest{
		Path:        path,
		QueryParams: queryParams,
		Headers:     headers,
	}
}

// domainToFiberResponse 도메인 BrowserResponse를 Fiber 응답으로 변환
func (a *MavenBrowserAdapter) domainToFiberResponse(c *fiber.Ctx, response *maven.BrowserResponse) error {
	// 상태 코드 설정
	c.Status(response.StatusCode)

	// Content-Type 설정 (응답에 지정된 경우)
	if response.ContentType != "" {
		c.Set(fiber.HeaderContentType, response.ContentType)
	}

	// Accept 헤더에 따른 응답 형식 결정
	acceptHeader := c.Get("Accept", "")

	if strings.Contains(acceptHeader, "application/json") {
		// JSON 응답
		c.Set(fiber.HeaderContentType, "application/json")
		return c.JSON(response.Data)
	}

	// HTML 템플릿 렌더링 (기본)
	c.Set(fiber.HeaderContentType, "text/html")
	return c.Render("maven-browser", response.Data)
}

// IsBrowserRequest 요청이 브라우저 요청인지 확인 (어댑터 레벨 검증)
func (a *MavenBrowserAdapter) IsBrowserRequest(c *fiber.Ctx) bool {
	userAgent := c.Get("User-Agent", "")
	acceptHeader := c.Get("Accept", "")

	// 일반적인 브라우저 패턴
	browserPatterns := []string{
		"Mozilla", "Chrome", "Safari", "Firefox", "Edge", "Opera",
	}

	for _, pattern := range browserPatterns {
		if strings.Contains(userAgent, pattern) {
			return true
		}
	}

	// Accept 헤더로 HTML 요청 확인
	if strings.Contains(acceptHeader, "text/html") {
		return true
	}

	// JSON 요청도 브라우저로 간주 (AJAX)
	if strings.Contains(acceptHeader, "application/json") {
		return true
	}

	return false
}
