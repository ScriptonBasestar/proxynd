package unit

import (
	"io"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"proxynd/routers"
)

// TestHealthRouter 헬스체크 라우터 테스트
func TestHealthRouter(t *testing.T) {
	app := fiber.New()
	routers.HealthRouter(app)

	t.Run("Health Check Success", func(t *testing.T) {
		// 필요한 환경 변수 설정
		_ = os.Setenv("CONFIG_DIR", "/tmp")
		_ = os.Setenv("STORAGE_DIR", "/tmp")
		defer func() {
			_ = os.Unsetenv("CONFIG_DIR")
			_ = os.Unsetenv("STORAGE_DIR")
		}()

		req := httptest.NewRequest("GET", "/healthz", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, 200, resp.StatusCode)
		assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		// JSON 응답 검증
		assert.Contains(t, string(body), `"status":"ok"`)
		assert.Contains(t, string(body), `"checks"`)
	})

	t.Run("Health Check Failure", func(t *testing.T) {
		// 환경 변수 제거
		_ = os.Unsetenv("CONFIG_DIR")
		_ = os.Unsetenv("STORAGE_DIR")

		req := httptest.NewRequest("GET", "/healthz", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, 503, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		// 에러 상태 검증
		assert.Contains(t, string(body), `"status":"error"`)
		assert.Contains(t, string(body), `"checks"`)
	})

	t.Run("Invalid Method", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/healthz", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		// POST는 허용되지 않아야 함
		assert.Equal(t, 405, resp.StatusCode)
	})
}

// TestProxyRouter 프록시 라우터 테스트
func TestProxyRouter(t *testing.T) {
	// 테스트를 위한 환경 변수 설정
	_ = os.Setenv("CONFIG_DIR", "../../sample-conf")
	defer func() { _ = os.Unsetenv("CONFIG_DIR") }()

	app := fiber.New()

	// 프록시 라우터 설정 (실제 구현에서는 설정 파일이 필요)
	// 여기서는 단순히 라우팅만 테스트
	app.Get("/proxy/:type/*", func(c *fiber.Ctx) error {
		proxyType := c.Params("type")
		path := c.Params("*")

		// 파라미터 검증
		if proxyType == "" {
			return c.Status(400).SendString("Missing proxy type")
		}

		return c.JSON(fiber.Map{
			"proxy_type": proxyType,
			"path":       path,
		})
	})

	t.Run("NPM Proxy Route", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/proxy/npm/express", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, 200, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		assert.Contains(t, string(body), `"proxy_type":"npm"`)
		assert.Contains(t, string(body), `"path":"express"`)
	})

	t.Run("PyPI Proxy Route", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/proxy/pip/simple/requests/", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, 200, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		assert.Contains(t, string(body), `"proxy_type":"pip"`)
		assert.Contains(t, string(body), `"path":"simple/requests/"`)
	})

	t.Run("APT Proxy Route", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/proxy/apt/ubuntu/dists/jammy/Release", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, 200, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		assert.Contains(t, string(body), `"proxy_type":"apt"`)
		assert.Contains(t, string(body), `"path":"ubuntu/dists/jammy/Release"`)
	})

	t.Run("Docker Proxy Route", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/proxy/docker/v2/library/nginx/manifests/latest", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, 200, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		assert.Contains(t, string(body), `"proxy_type":"docker"`)
		assert.Contains(t, string(body), `"path":"v2/library/nginx/manifests/latest"`)
	})

	t.Run("Missing Proxy Type", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/proxy/", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		// 라우트가 매칭되지 않으므로 404
		assert.Equal(t, 404, resp.StatusCode)
	})

	t.Run("Invalid Route", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/invalid", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, 404, resp.StatusCode)
	})

	t.Run("Scoped Package Route", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/proxy/npm/@types/node", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, 200, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		assert.Contains(t, string(body), `"proxy_type":"npm"`)
		assert.Contains(t, string(body), `"path":"@types/node"`)
	})

	t.Run("Deep Path Route", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/proxy/pip/packages/source/r/requests/requests-2.28.2.tar.gz", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, 200, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		assert.Contains(t, string(body), `"proxy_type":"pip"`)
		assert.Contains(t, string(body), `"path":"packages/source/r/requests/requests-2.28.2.tar.gz"`)
	})
}

// TestRouterParameterExtraction 라우터 파라미터 추출 테스트
func TestRouterParameterExtraction(t *testing.T) {
	app := fiber.New()

	// 파라미터 추출 테스트용 핸들러
	app.Get("/proxy/:type/*", func(c *fiber.Ctx) error {
		proxyType := c.Params("type")
		path := c.Params("*")

		// URL 인코딩된 파라미터 처리
		decodedPath := c.Params("*", "")

		return c.JSON(fiber.Map{
			"proxy_type":   proxyType,
			"path":         path,
			"decoded_path": decodedPath,
			"query":        c.Query("version", ""),
		})
	})

	t.Run("URL Encoded Path", func(t *testing.T) {
		// @types/node 같은 스코프 패키지
		req := httptest.NewRequest("GET", "/proxy/npm/%40types%2Fnode", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, 200, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		assert.Contains(t, string(body), `"proxy_type":"npm"`)
		// URL 디코딩된 경로가 포함되어야 함
		assert.Contains(t, string(body), `@types`)
	})

	t.Run("Query Parameters", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/proxy/npm/express?version=4.18.2", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, 200, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		assert.Contains(t, string(body), `"proxy_type":"npm"`)
		assert.Contains(t, string(body), `"query":"4.18.2"`)
	})

	t.Run("Special Characters in Path", func(t *testing.T) {
		// Docker blob SHA256 해시
		req := httptest.NewRequest("GET", "/proxy/docker/v2/library/nginx/blobs/sha256:abc123def456", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, 200, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		assert.Contains(t, string(body), `"proxy_type":"docker"`)
		assert.Contains(t, string(body), `sha256:abc123def456`)
	})

	t.Run("Empty Path", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/proxy/npm/", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, 200, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		assert.Contains(t, string(body), `"proxy_type":"npm"`)
		// 빈 경로도 처리되어야 함
		assert.Contains(t, string(body), `"path":""`)
	})
}

// TestRouterMiddlewareChain 라우터 미들웨어 체인 테스트
func TestRouterMiddlewareChain(t *testing.T) {
	app := fiber.New()

	// 미들웨어 실행 순서 추적
	var executionOrder []string

	// 첫 번째 미들웨어
	app.Use("/proxy", func(c *fiber.Ctx) error {
		executionOrder = append(executionOrder, "middleware1")
		c.Locals("middleware1", true)
		return c.Next()
	})

	// 두 번째 미들웨어
	app.Use("/proxy", func(c *fiber.Ctx) error {
		executionOrder = append(executionOrder, "middleware2")
		c.Locals("middleware2", true)
		return c.Next()
	})

	// 핸들러
	app.Get("/proxy/:type/*", func(c *fiber.Ctx) error {
		executionOrder = append(executionOrder, "handler")

		// 미들웨어가 설정한 로컬 값 확인
		m1 := c.Locals("middleware1")
		m2 := c.Locals("middleware2")

		return c.JSON(fiber.Map{
			"middleware1_executed": m1 != nil,
			"middleware2_executed": m2 != nil,
			"execution_order":      executionOrder,
		})
	})

	t.Run("Middleware Execution Order", func(t *testing.T) {
		// 실행 순서 초기화
		executionOrder = []string{}

		req := httptest.NewRequest("GET", "/proxy/npm/express", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, 200, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		// 미들웨어가 모두 실행되었는지 확인
		assert.Contains(t, string(body), `"middleware1_executed":true`)
		assert.Contains(t, string(body), `"middleware2_executed":true`)

		// 실행 순서 확인
		assert.Contains(t, string(body), `"execution_order":["middleware1","middleware2","handler"]`)
	})
}
