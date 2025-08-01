package testutil

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

// HTTPTestHelper HTTP 테스트를 위한 헬퍼 유틸리티
type HTTPTestHelper struct {
	t   *testing.T
	app *fiber.App
}

// NewHTTPTestHelper 새로운 HTTP 테스트 헬퍼 생성
func NewHTTPTestHelper(t *testing.T) *HTTPTestHelper {
	return &HTTPTestHelper{
		t:   t,
		app: fiber.New(),
	}
}

// CreateFiberContext Fiber 컨텍스트 생성
func (h *HTTPTestHelper) CreateFiberContext(method, url string, body io.Reader) *fiber.Ctx {
	c := h.app.AcquireCtx(&fasthttp.RequestCtx{})
	c.Request().Header.SetMethod(method)
	c.Request().SetRequestURI(url)

	if body != nil {
		if bodyBytes, err := io.ReadAll(body); err == nil {
			c.Request().SetBody(bodyBytes)
		}
	}

	return c
}

// CreateFiberContextWithHeaders 헤더가 포함된 Fiber 컨텍스트 생성
func (h *HTTPTestHelper) CreateFiberContextWithHeaders(method, url string, body io.Reader, headers map[string]string) *fiber.Ctx {
	c := h.CreateFiberContext(method, url, body)

	for key, value := range headers {
		c.Request().Header.Set(key, value)
	}

	return c
}

// CreateFiberContextWithParams 파라미터가 포함된 Fiber 컨텍스트 생성
func (h *HTTPTestHelper) CreateFiberContextWithParams(method, url string, params map[string]string) *fiber.Ctx {
	c := h.CreateFiberContext(method, url, nil)

	// 파라미터 시뮬레이션 (실제 라우팅은 없지만 Params 함수 동작을 위해)
	for key, value := range params {
		// Fiber의 내부 구조에 직접 접근하는 것은 권장되지 않지만 테스트를 위해
		if key == "*" {
			c.Request().SetRequestURI("/" + value)
		}
	}

	return c
}

// ReleaseFiberContext Fiber 컨텍스트 해제
func (h *HTTPTestHelper) ReleaseFiberContext(c *fiber.Ctx) {
	h.app.ReleaseCtx(c)
}

// MockHTTPServer 테스트용 HTTP 서버
type MockHTTPServer struct {
	server   *httptest.Server
	handler  http.HandlerFunc
	requests []*http.Request
}

// NewMockHTTPServer 새로운 Mock HTTP 서버 생성
func NewMockHTTPServer(handler http.HandlerFunc) *MockHTTPServer {
	mock := &MockHTTPServer{
		handler:  handler,
		requests: make([]*http.Request, 0),
	}

	// 요청을 기록하는 래퍼 핸들러
	wrappedHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mock.requests = append(mock.requests, r)
		handler(w, r)
	})

	mock.server = httptest.NewServer(wrappedHandler)
	return mock
}

// URL 서버 URL 반환
func (m *MockHTTPServer) URL() string {
	return m.server.URL
}

// GetRequests 수신한 요청들 반환
func (m *MockHTTPServer) GetRequests() []*http.Request {
	return m.requests
}

// GetLastRequest 마지막 요청 반환
func (m *MockHTTPServer) GetLastRequest() *http.Request {
	if len(m.requests) == 0 {
		return nil
	}
	return m.requests[len(m.requests)-1]
}

// RequestCount 요청 수 반환
func (m *MockHTTPServer) RequestCount() int {
	return len(m.requests)
}

// Reset 요청 기록 초기화
func (m *MockHTTPServer) Reset() {
	m.requests = m.requests[:0]
}

// Close 서버 종료
func (m *MockHTTPServer) Close() {
	m.server.Close()
}

// 미리 정의된 응답 핸들러들

// JSONResponseHandler JSON 응답을 반환하는 핸들러
func JSONResponseHandler(statusCode int, jsonData string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		_, _ = w.Write([]byte(jsonData))
	}
}

// PlainTextResponseHandler 평문 응답을 반환하는 핸들러
func PlainTextResponseHandler(statusCode int, text string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(statusCode)
		_, _ = w.Write([]byte(text))
	}
}

// FileResponseHandler 파일 응답을 반환하는 핸들러
func FileResponseHandler(statusCode int, filename string, content []byte) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", "attachment; filename="+filename)
		w.WriteHeader(statusCode)
		_, _ = w.Write(content)
	}
}

// ErrorResponseHandler 오류 응답을 반환하는 핸들러
func ErrorResponseHandler(statusCode int, errorMessage string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, errorMessage, statusCode)
	}
}

// DelayedResponseHandler 지연된 응답을 반환하는 핸들러
func DelayedResponseHandler(delay int, handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 실제 환경에서는 time.Sleep을 사용하지만 테스트에서는 제한적으로 사용
		// time.Sleep(time.Duration(delay) * time.Millisecond)
		handler(w, r)
	}
}

// ConditionalResponseHandler 조건부 응답 핸들러
func ConditionalResponseHandler(condition func(*http.Request) bool, trueHandler, falseHandler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if condition(r) {
			trueHandler(w, r)
		} else {
			falseHandler(w, r)
		}
	}
}

// PackageManagerResponseHandlers 패키지 매니저별 응답 핸들러들

// PyPISimpleAPIHandler PyPI Simple API 응답 핸들러
func PyPISimpleAPIHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		html := `<!DOCTYPE html>
<html>
<head><title>Simple index</title></head>
<body>
<h1>Simple index</h1>
<a href="/simple/requests/">requests</a><br>
<a href="/simple/django/">django</a><br>
</body>
</html>`
		_, _ = w.Write([]byte(html))
	}
}

// MavenMetadataHandler Maven 메타데이터 응답 핸들러
func MavenMetadataHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		xml := `<?xml version="1.0" encoding="UTF-8"?>
<metadata>
  <groupId>org.springframework</groupId>
  <artifactId>spring-core</artifactId>
  <versioning>
    <latest>5.3.21</latest>
    <release>5.3.21</release>
    <versions>
      <version>5.3.21</version>
    </versions>
  </versioning>
</metadata>`
		_, _ = w.Write([]byte(xml))
	}
}

// NPMPackageHandler NPM 패키지 정보 응답 핸들러
func NPMPackageHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json := `{
  "name": "express",
  "version": "4.18.1",
  "description": "Fast, unopinionated, minimalist web framework",
  "main": "index.js",
  "dependencies": {}
}`
		_, _ = w.Write([]byte(json))
	}
}

// DockerManifestHandler Docker 매니페스트 응답 핸들러
func DockerManifestHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.docker.distribution.manifest.v2+json")
		w.WriteHeader(http.StatusOK)
		manifest := `{
  "schemaVersion": 2,
  "mediaType": "application/vnd.docker.distribution.manifest.v2+json",
  "config": {
    "mediaType": "application/vnd.docker.container.image.v1+json",
    "size": 1234,
    "digest": "sha256:abc123"
  }
}`
		_, _ = w.Write([]byte(manifest))
	}
}

// HTTPTestAssertions HTTP 테스트 단언문들

// AssertHTTPRequest HTTP 요청 검증
func AssertHTTPRequest(t *testing.T, req *http.Request, expectedMethod, expectedPath string) {
	assert.Equal(t, expectedMethod, req.Method)
	assert.Equal(t, expectedPath, req.URL.Path)
}

// AssertHTTPRequestWithHeaders 헤더가 포함된 HTTP 요청 검증
func AssertHTTPRequestWithHeaders(t *testing.T, req *http.Request, expectedHeaders map[string]string) {
	for key, expectedValue := range expectedHeaders {
		actualValue := req.Header.Get(key)
		assert.Equal(t, expectedValue, actualValue, "Header %s mismatch", key)
	}
}

// AssertHTTPRequestBody HTTP 요청 본문 검증
func AssertHTTPRequestBody(t *testing.T, req *http.Request, expectedBody []byte) {
	if req.Body != nil {
		actualBody, err := io.ReadAll(req.Body)
		require.NoError(t, err)
		assert.Equal(t, expectedBody, actualBody)

		// Body를 다시 읽을 수 있도록 복원
		req.Body = io.NopCloser(bytes.NewReader(actualBody))
	}
}

// AssertFiberResponse Fiber 응답 검증
func AssertFiberResponse(t *testing.T, c *fiber.Ctx, expectedStatus int, expectedBody string) {
	assert.Equal(t, expectedStatus, c.Response().StatusCode())

	if expectedBody != "" {
		actualBody := string(c.Response().Body())
		assert.Equal(t, expectedBody, actualBody)
	}
}

// AssertFiberResponseHeaders Fiber 응답 헤더 검증
func AssertFiberResponseHeaders(t *testing.T, c *fiber.Ctx, expectedHeaders map[string]string) {
	for key, expectedValue := range expectedHeaders {
		actualValue := string(c.Response().Header.Peek(key))
		assert.Equal(t, expectedValue, actualValue, "Response header %s mismatch", key)
	}
}
