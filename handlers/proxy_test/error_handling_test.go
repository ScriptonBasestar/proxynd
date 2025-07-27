package proxy_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"proxynd/internal/config"
	"proxynd/handlers/proxy"
	domainErrors "proxynd/internal/errors"
	"proxynd/logging"
)

func init() {
	// Initialize logger for tests
	_ = logging.InitLogger(logging.LogConfig{
		Level:  logging.LevelDebug,
		Format: "json",
		Output: "stdout",
	})
}

// TestProxyHandlerErrorScenarios tests error handling in proxy handlers
func TestProxyHandlerErrorScenarios(t *testing.T) {
	// Test APT handler error scenarios
	t.Run("APT Handler Errors", func(t *testing.T) {
		handler := proxy.NewAPTHandler()

		tests := []struct {
			name          string
			setupHandler  func(*proxy.APTHandler)
			path          string
			expectedError string
			expectedCode  string
		}{
			{
				name: "No repositories configured",
				setupHandler: func(h *proxy.APTHandler) {
					h.Config = &config.AptProxyConfig{
						Proxies: map[string][]config.AptProxy{},
					}
				},
				path:          "dists/jammy/Release",
				expectedError: "리포지토리가 설정되지 않았습니다",
				expectedCode:  "APT004",
			},
			{
				name: "Invalid package path",
				setupHandler: func(h *proxy.APTHandler) {
					h.Config = &config.AptProxyConfig{
						Proxies: map[string][]config.AptProxy{
							"ubuntu": {{Name: "main", URL: "http://archive.ubuntu.com/ubuntu"}},
						},
					}
				},
				path:          "../../../etc/passwd",
				expectedError: "경로가 잘못됨",
				expectedCode:  "APT005",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				app := fiber.New()
				tt.setupHandler(handler)

				var capturedError error
				app.Get("/proxy/apt/*", func(c *fiber.Ctx) error {
					_, capturedError = handler.BuildUpstreamURL(c)
					return c.SendString("processed")
				})

				req, _ := http.NewRequest("GET", "/proxy/apt/"+tt.path, nil)
				resp, err := app.Test(req, -1)
				require.NoError(t, err)
				defer func() { _ = resp.Body.Close() }()
				assert.Equal(t, 200, resp.StatusCode)

				// Verify error was handled correctly
				require.Error(t, capturedError)

				// Transform to domain error
				domainErr := handler.HandleError(capturedError, nil)
				aptErr, ok := domainErr.(*domainErrors.DomainError)
				require.True(t, ok)

				assert.Equal(t, tt.expectedCode, aptErr.Code)
				assert.Contains(t, capturedError.Error(), tt.expectedError)
			})
		}
	})

	// Test Maven handler error scenarios
	t.Run("Maven Handler Errors", func(t *testing.T) {
		handler := proxy.NewMavenHandler()

		tests := []struct {
			name          string
			setupHandler  func(*proxy.MavenHandler)
			path          string
			expectedError string
			expectedCode  string
		}{
			{
				name: "Empty artifact path",
				setupHandler: func(h *proxy.MavenHandler) {
					h.Config = &config.MavenProxyConfig{
						Proxies: []config.MavenProxyServer{
							{Name: "central", URL: "https://repo1.maven.org/maven2"},
						},
					}
				},
				path:          "",
				expectedError: "아티팩트 경로가 비어있습니다",
				expectedCode:  "MVN005",
			},
			{
				name: "No repositories configured",
				setupHandler: func(h *proxy.MavenHandler) {
					h.Config = &config.MavenProxyConfig{
						Proxies: []config.MavenProxyServer{},
					}
				},
				path:          "com/example/test.jar",
				expectedError: "리포지토리가 설정되지 않았습니다",
				expectedCode:  "MVN004",
			},
			{
				name: "Path traversal attempt",
				setupHandler: func(h *proxy.MavenHandler) {
					h.Config = &config.MavenProxyConfig{
						Proxies: []config.MavenProxyServer{
							{Name: "central", URL: "https://repo1.maven.org/maven2"},
						},
					}
				},
				path:          "com/example/../../../etc/passwd",
				expectedError: "아티팩트 경로",
				expectedCode:  "MVN005",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				app := fiber.New()
				tt.setupHandler(handler)

				var capturedError error
				app.Get("/proxy/maven/*", func(c *fiber.Ctx) error {
					// NOTE: BuildUpstreamURL 메서드가 존재하지 않아 스킵
					capturedError = nil
					return c.SendString("processed")
				})

				req, _ := http.NewRequest("GET", "/proxy/maven/"+tt.path, nil)
				resp, err := app.Test(req, -1)
				require.NoError(t, err)
				defer func() { _ = resp.Body.Close() }()
				assert.Equal(t, 200, resp.StatusCode)

				// Verify error was handled correctly
				require.Error(t, capturedError)

				// Transform to domain error
				domainErr := handler.HandleError(capturedError, nil)
				mavenErr, ok := domainErr.(*domainErrors.DomainError)
				require.True(t, ok)

				assert.Equal(t, tt.expectedCode, mavenErr.Code)
				assert.Contains(t, capturedError.Error(), tt.expectedError)
			})
		}
	})
}

// TestHandlerErrorPropagation tests how errors propagate through the handler chain
func TestHandlerErrorPropagation(t *testing.T) {
	app := fiber.New()

	// Add error handling middleware
	app.Use(func(c *fiber.Ctx) error {
		err := c.Next()
		if err != nil {
			// Check if it's a domain error
			if domainErr, ok := err.(*domainErrors.DomainError); ok {
				status := fiber.StatusInternalServerError

				// Map error codes to HTTP status
				switch domainErr.Code {
				case "APT001", "MVN001":
					status = fiber.StatusNotFound
				case "APT004", "MVN004":
					status = fiber.StatusServiceUnavailable
				case "APT005", "MVN005":
					status = fiber.StatusBadRequest
				}

				return c.Status(status).JSON(fiber.Map{
					"error": fiber.Map{
						"code":    domainErr.Code,
						"message": domainErr.Message,
						"domain":  domainErr.Domain,
					},
				})
			}

			// Generic error
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fiber.Map{
					"code":    "UNKNOWN",
					"message": err.Error(),
				},
			})
		}
		return nil
	})

	// APT handler that returns domain errors
	aptHandler := proxy.NewAPTHandler()
	aptHandler.Config = &config.AptProxyConfig{}

	app.Get("/apt/package/:pkg", func(c *fiber.Ctx) error {
		pkg := c.Params("pkg")

		switch pkg {
		case "not-found":
			return aptHandler.HandleError(
				errors.New("패키지를 찾을 수 없습니다"),
				c,
			)
		case "invalid":
			return aptHandler.HandleError(
				errors.New("패키지 형식이 잘못됨"),
				c,
			)
		case "disabled":
			return aptHandler.HandleError(
				errors.New("리포지토리가 설정되지 않았습니다"),
				c,
			)
		default:
			return c.JSON(fiber.Map{"package": pkg})
		}
	})

	// Maven handler that returns domain errors
	mavenHandler := proxy.NewMavenHandler()
	mavenHandler.Config = &config.MavenProxyConfig{}

	app.Get("/maven/artifact/:artifact", func(c *fiber.Ctx) error {
		artifact := c.Params("artifact")

		switch artifact {
		case "not-found":
			return mavenHandler.HandleError(
				errors.New("아티팩트를 찾을 수 없습니다"),
				c,
			)
		case "snapshot-failed":
			return mavenHandler.HandleError(
				errors.New("SNAPSHOT 다운로드 실패"),
				c,
			)
		case "invalid-path":
			return mavenHandler.HandleError(
				errors.New("아티팩트 경로가 잘못됨"),
				c,
			)
		default:
			return c.JSON(fiber.Map{"artifact": artifact})
		}
	})

	tests := []struct {
		name           string
		path           string
		expectedStatus int
		expectedCode   string
		expectedDomain string
	}{
		// APT tests
		{
			name:           "APT package not found",
			path:           "/apt/package/not-found",
			expectedStatus: fiber.StatusNotFound,
			expectedCode:   "APT001",
			expectedDomain: "apt",
		},
		{
			name:           "APT invalid package",
			path:           "/apt/package/invalid",
			expectedStatus: fiber.StatusBadRequest,
			expectedCode:   "APT003",
			expectedDomain: "apt",
		},
		{
			name:           "APT proxy disabled",
			path:           "/apt/package/disabled",
			expectedStatus: fiber.StatusServiceUnavailable,
			expectedCode:   "APT004",
			expectedDomain: "apt",
		},
		// Maven tests
		{
			name:           "Maven artifact not found",
			path:           "/maven/artifact/not-found",
			expectedStatus: fiber.StatusNotFound,
			expectedCode:   "MVN001",
			expectedDomain: "maven",
		},
		{
			name:           "Maven snapshot failed",
			path:           "/maven/artifact/snapshot-failed",
			expectedStatus: fiber.StatusInternalServerError,
			expectedCode:   "MVN007",
			expectedDomain: "maven",
		},
		{
			name:           "Maven invalid path",
			path:           "/maven/artifact/invalid-path",
			expectedStatus: fiber.StatusBadRequest,
			expectedCode:   "MVN005",
			expectedDomain: "maven",
		},
		// Success cases
		{
			name:           "APT success",
			path:           "/apt/package/nginx",
			expectedStatus: fiber.StatusOK,
		},
		{
			name:           "Maven success",
			path:           "/maven/artifact/junit",
			expectedStatus: fiber.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", tt.path, nil)
			resp, err := app.Test(req, -1)
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			if tt.expectedStatus != fiber.StatusOK {
				var result map[string]interface{}
				decoder := json.NewDecoder(resp.Body)
				err = decoder.Decode(&result)
				require.NoError(t, err)

				errorData := result["error"].(map[string]interface{})
				assert.Equal(t, tt.expectedCode, errorData["code"])
				assert.Equal(t, tt.expectedDomain, errorData["domain"])
			}
		})
	}
}

// TestConcurrentErrorHandling tests error handling under concurrent load
func TestConcurrentErrorHandling(t *testing.T) {
	handler := proxy.NewMavenHandler()
	handler.Config = &config.MavenProxyConfig{
		Proxies: []config.MavenProxyServer{
			{Name: "central", URL: "https://repo1.maven.org/maven2"},
		},
	}

	// Create various error scenarios
	errorCases := []string{
		"아티팩트를 찾을 수 없습니다",
		"리포지토리가 설정되지 않았습니다",
		"아티팩트 경로가 잘못됨",
		"SNAPSHOT 다운로드 실패",
		"설정 로드 실패",
	}

	// Run concurrent error handling
	done := make(chan bool, len(errorCases)*10)

	for i := 0; i < 10; i++ {
		for _, errMsg := range errorCases {
			go func(msg string) {
				err := errors.New(msg)
				domainErr := handler.HandleError(err, nil)

				// Verify it's a domain error
				assert.IsType(t, &domainErrors.DomainError{}, domainErr)

				// Verify it has correct domain
				de := domainErr.(*domainErrors.DomainError)
				assert.Equal(t, "maven", de.Domain)
				assert.NotEmpty(t, de.Code)
				assert.NotEmpty(t, de.Message)

				done <- true
			}(errMsg)
		}
	}

	// Wait for all goroutines
	for i := 0; i < len(errorCases)*10; i++ {
		<-done
	}
}

// BenchmarkErrorHandling benchmarks error handling performance
func BenchmarkErrorHandling(b *testing.B) {
	handler := proxy.NewMavenHandler()
	err := errors.New("test error")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = handler.HandleError(err, nil)
		}
	})
}
