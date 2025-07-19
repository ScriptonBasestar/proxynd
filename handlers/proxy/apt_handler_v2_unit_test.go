package proxy

import (
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"

	"proxynd/configs"
	"proxynd/internal/errors"
)

// TestAPTHandlerV2_Type tests the Type method
func TestAPTHandlerV2_Type(t *testing.T) {
	handler := &APTHandlerV2{}
	assert.Equal(t, "apt", handler.Type())
}

// TestAPTHandlerV2_IsEnabled tests the IsEnabled method
func TestAPTHandlerV2_IsEnabled(t *testing.T) {
	tests := []struct {
		name     string
		config   *configs.AptProxyConfig
		expected bool
	}{
		{
			name:     "nil config",
			config:   nil,
			expected: false,
		},
		{
			name:     "empty proxies",
			config:   &configs.AptProxyConfig{Proxies: map[string][]configs.AptProxy{}},
			expected: false,
		},
		{
			name: "with proxies",
			config: &configs.AptProxyConfig{
				Proxies: map[string][]configs.AptProxy{
					"ubuntu": {{Name: "main", URL: "http://mirror.example.com"}},
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := &APTHandlerV2{config: tt.config}
			assert.Equal(t, tt.expected, handler.IsEnabled())
		})
	}
}

// TestAPTHandlerV2_GenerateCacheKey tests cache key generation
func TestAPTHandlerV2_GenerateCacheKey(t *testing.T) {
	handler := &APTHandlerV2{}
	app := fiber.New()

	tests := []struct {
		name     string
		path     string
		osType   string
		expected string
	}{
		{
			name:     "Release file",
			path:     "/proxy/apt/ubuntu/dists/focal/Release",
			osType:   "ubuntu",
			expected: "apt:ubuntu:dists_focal_Release",
		},
		{
			name:     "Package file",
			path:     "/proxy/apt/debian/pool/main/v/vim/vim_8.2.deb",
			osType:   "debian",
			expected: "apt:debian:pool_main_v_vim_vim_8.2.deb",
		},
		{
			name:     "Compressed file",
			path:     "/proxy/apt/ubuntu/dists/focal/main/binary-amd64/Packages.gz",
			osType:   "ubuntu",
			expected: "apt:ubuntu:dists_focal_main_binary-amd64_Packages.gz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := app.AcquireCtx(&fasthttp.RequestCtx{})
			defer app.ReleaseCtx(c)

			c.Request().SetRequestURI(tt.path)
			c.Locals("osType", tt.osType)

			key := handler.GenerateCacheKey(c)
			assert.Equal(t, tt.expected, key)
		})
	}
}

// TestAPTHandlerV2_BuildUpstreamURL tests upstream URL building
func TestAPTHandlerV2_BuildUpstreamURL(t *testing.T) {
	app := fiber.New()

	tests := []struct {
		name        string
		config      *configs.AptProxyConfig
		path        string
		osType      string
		expected    string
		expectError bool
	}{
		{
			name: "ubuntu mirror",
			config: &configs.AptProxyConfig{
				Proxies: map[string][]configs.AptProxy{
					"ubuntu": {{Name: "main", URL: "http://archive.ubuntu.com/ubuntu"}},
				},
			},
			path:     "/proxy/apt/ubuntu/dists/focal/Release",
			osType:   "ubuntu",
			expected: "http://archive.ubuntu.com/ubuntu/dists/focal/Release",
		},
		{
			name: "debian mirror with trailing slash",
			config: &configs.AptProxyConfig{
				Proxies: map[string][]configs.AptProxy{
					"debian": {{Name: "main", URL: "http://deb.debian.org/debian/"}},
				},
			},
			path:     "/proxy/apt/debian/pool/main/a/apache2/apache2_2.4.deb",
			osType:   "debian",
			expected: "http://deb.debian.org/debian/pool/main/a/apache2/apache2_2.4.deb",
		},
		{
			name: "no mirror for OS type",
			config: &configs.AptProxyConfig{
				Proxies: map[string][]configs.AptProxy{
					"ubuntu": {{Name: "main", URL: "http://archive.ubuntu.com/ubuntu"}},
				},
			},
			path:        "/proxy/apt/centos/something",
			osType:      "centos",
			expectError: true,
		},
		{
			name:        "nil config",
			config:      nil,
			path:        "/proxy/apt/ubuntu/dists/focal/Release",
			osType:      "ubuntu",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := &APTHandlerV2{config: tt.config}

			c := app.AcquireCtx(&fasthttp.RequestCtx{})
			defer app.ReleaseCtx(c)

			c.Request().SetRequestURI(tt.path)
			c.Locals("osType", tt.osType)

			url, err := handler.BuildUpstreamURL(c)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, url)
			}
		})
	}
}

// TestAPTHandlerV2_TransformRequest tests request transformation
func TestAPTHandlerV2_TransformRequest(t *testing.T) {
	handler := &APTHandlerV2{}
	app := fiber.New()

	c := app.AcquireCtx(&fasthttp.RequestCtx{})
	defer app.ReleaseCtx(c)

	agent := fiber.AcquireAgent()
	defer fiber.ReleaseAgent(agent)

	// 초기 헤더 설정
	agent.Set("User-Agent", "test-agent")

	err := handler.TransformRequest(c, agent)
	assert.NoError(t, err)

	// APT 특정 헤더가 추가되었는지 확인
	req := agent.Request()
	assert.Equal(t, "Debian APT-HTTP/2.0", string(req.Header.UserAgent()))
	assert.Equal(t, "*/*", string(req.Header.Peek("Accept")))
	assert.Equal(t, "gzip, deflate", string(req.Header.Peek("Accept-Encoding")))
}

// TestAPTHandlerV2_TransformResponse tests response transformation
func TestAPTHandlerV2_TransformResponse(t *testing.T) {
	app := fiber.New()

	tests := []struct {
		name     string
		config   *configs.AptProxyConfig
		path     string
		osType   string
		input    []byte
		expected string
	}{
		{
			name: "Release file with mirror replacement",
			config: &configs.AptProxyConfig{
				Proxies: map[string][]configs.AptProxy{
					"ubuntu": {{Name: "custom", URL: "http://custom.mirror.com/ubuntu"}},
				},
			},
			path:   "/proxy/apt/ubuntu/dists/focal/Release",
			osType: "ubuntu",
			input: []byte(`Origin: Ubuntu
Suite: focal
Mirror: http://archive.ubuntu.com/ubuntu
Codename: focal`),
			expected: `Origin: Ubuntu
Suite: focal
Mirror: http://custom.mirror.com/ubuntu
Codename: focal`,
		},
		{
			name: "InRelease file with mirror replacement",
			config: &configs.AptProxyConfig{
				Proxies: map[string][]configs.AptProxy{
					"debian": {{Name: "main", URL: "http://local.debian.mirror"}},
				},
			},
			path:   "/proxy/apt/debian/dists/bullseye/InRelease",
			osType: "debian",
			input: []byte(`-----BEGIN PGP SIGNED MESSAGE-----
Hash: SHA256

Origin: Debian
Mirror: http://deb.debian.org/debian
Suite: bullseye`),
			expected: `-----BEGIN PGP SIGNED MESSAGE-----
Hash: SHA256

Origin: Debian
Mirror: http://local.debian.mirror
Suite: bullseye`,
		},
		{
			name:     "Non-metadata file (no transformation)",
			config:   &configs.AptProxyConfig{},
			path:     "/proxy/apt/ubuntu/pool/main/v/vim/vim_8.2.deb",
			osType:   "ubuntu",
			input:    []byte("binary package data"),
			expected: "binary package data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := &APTHandlerV2{config: tt.config}

			c := app.AcquireCtx(&fasthttp.RequestCtx{})
			defer app.ReleaseCtx(c)

			c.Request().SetRequestURI(tt.path)
			c.Locals("osType", tt.osType)

			output, err := handler.TransformResponse(tt.input, c)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, string(output))
		})
	}
}

// TestAPTHandlerV2_ShouldCache tests caching decision logic
func TestAPTHandlerV2_ShouldCache(t *testing.T) {
	app := fiber.New()
	c := app.AcquireCtx(&fasthttp.RequestCtx{})
	defer app.ReleaseCtx(c)

	tests := []struct {
		name       string
		config     *configs.AptProxyConfig
		statusCode int
		expected   bool
	}{
		{
			name:       "200 OK with cache enabled",
			config:     &configs.AptProxyConfig{UseCache: true},
			statusCode: 200,
			expected:   true,
		},
		{
			name:       "200 OK with cache disabled",
			config:     &configs.AptProxyConfig{UseCache: false},
			statusCode: 200,
			expected:   false,
		},
		{
			name:       "404 Not Found",
			config:     &configs.AptProxyConfig{UseCache: true},
			statusCode: 404,
			expected:   false,
		},
		{
			name:       "500 Server Error",
			config:     &configs.AptProxyConfig{UseCache: true},
			statusCode: 500,
			expected:   false,
		},
		{
			name:       "304 Not Modified",
			config:     &configs.AptProxyConfig{UseCache: true},
			statusCode: 304,
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := &APTHandlerV2{config: tt.config}
			result := handler.ShouldCache(c, tt.statusCode)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestAPTHandlerV2_GetCacheTTL tests cache TTL calculation
func TestAPTHandlerV2_GetCacheTTL(t *testing.T) {
	handler := &APTHandlerV2{}
	app := fiber.New()

	tests := []struct {
		name        string
		path        string
		expectedTTL time.Duration
	}{
		// 메타데이터 파일 - 짧은 TTL
		{
			name:        "Release file",
			path:        "/proxy/apt/ubuntu/dists/focal/Release",
			expectedTTL: 10 * time.Minute,
		},
		{
			name:        "Release.gpg file",
			path:        "/proxy/apt/ubuntu/dists/focal/Release.gpg",
			expectedTTL: 10 * time.Minute,
		},
		{
			name:        "InRelease file",
			path:        "/proxy/apt/debian/dists/bullseye/InRelease",
			expectedTTL: 10 * time.Minute,
		},
		{
			name:        "Packages file",
			path:        "/proxy/apt/ubuntu/dists/focal/main/binary-amd64/Packages",
			expectedTTL: 10 * time.Minute,
		},
		{
			name:        "Sources file",
			path:        "/proxy/apt/ubuntu/dists/focal/main/source/Sources",
			expectedTTL: 10 * time.Minute,
		},
		// 압축 파일 - 중간 TTL
		{
			name:        "Packages.gz",
			path:        "/proxy/apt/ubuntu/dists/focal/main/binary-amd64/Packages.gz",
			expectedTTL: 24 * time.Hour,
		},
		{
			name:        "Sources.xz",
			path:        "/proxy/apt/ubuntu/dists/focal/main/source/Sources.xz",
			expectedTTL: 24 * time.Hour,
		},
		{
			name:        "Packages.bz2",
			path:        "/proxy/apt/ubuntu/dists/focal/universe/binary-amd64/Packages.bz2",
			expectedTTL: 24 * time.Hour,
		},
		// 패키지 파일 - 긴 TTL
		{
			name:        "deb package",
			path:        "/proxy/apt/ubuntu/pool/main/v/vim/vim_8.2.deb",
			expectedTTL: 7 * 24 * time.Hour,
		},
		{
			name:        "udeb package",
			path:        "/proxy/apt/ubuntu/pool/main/l/linux/linux-modules.udeb",
			expectedTTL: 7 * 24 * time.Hour,
		},
		// 기타 파일 - 기본 TTL
		{
			name:        "other file",
			path:        "/proxy/apt/ubuntu/project/trace/archive.ubuntu.com",
			expectedTTL: 1 * time.Hour,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := app.AcquireCtx(&fasthttp.RequestCtx{})
			defer app.ReleaseCtx(c)

			c.Request().SetRequestURI(tt.path)
			ttl := handler.GetCacheTTL(c)
			assert.Equal(t, tt.expectedTTL, ttl)
		})
	}
}

// TestAPTHandlerV2_HandleError tests error handling
func TestAPTHandlerV2_HandleError(t *testing.T) {
	handler := &APTHandlerV2{}
	app := fiber.New()
	c := app.AcquireCtx(&fasthttp.RequestCtx{})
	defer app.ReleaseCtx(c)

	tests := []struct {
		name         string
		inputError   error
		expectedCode string
		expectedMsg  string
	}{
		{
			name:         "upstream connection error",
			inputError:   errors.New("connection refused"),
			expectedCode: errors.ErrCodeProxyUpstreamConnection,
			expectedMsg:  "APT proxy upstream connection failed",
		},
		{
			name:         "upstream timeout",
			inputError:   errors.New("timeout"),
			expectedCode: errors.ErrCodeProxyUpstreamTimeout,
			expectedMsg:  "APT proxy upstream timeout",
		},
		{
			name:         "generic error",
			inputError:   errors.New("something went wrong"),
			expectedCode: errors.ErrCodeProxyGeneric,
			expectedMsg:  "APT proxy error",
		},
		{
			name:         "domain error passthrough",
			inputError:   errors.NewDomainError(errors.ErrCodeCacheRead, "cache error"),
			expectedCode: errors.ErrCodeCacheRead,
			expectedMsg:  "cache error",
		},
		{
			name:         "nil error",
			inputError:   nil,
			expectedCode: "",
			expectedMsg:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := handler.HandleError(tt.inputError, c)

			if tt.inputError == nil {
				assert.NoError(t, err)
			} else {
				require.Error(t, err)
				domainErr, ok := err.(*errors.DomainError)
				require.True(t, ok)
				assert.Equal(t, tt.expectedCode, domainErr.Code)
				assert.Equal(t, tt.expectedMsg, domainErr.Message)
			}
		})
	}
}

// TestAPTHandlerV2_GetUpstreamAuth tests authentication
func TestAPTHandlerV2_GetUpstreamAuth(t *testing.T) {
	handler := &APTHandlerV2{}
	app := fiber.New()
	c := app.AcquireCtx(&fasthttp.RequestCtx{})
	defer app.ReleaseCtx(c)

	username, password, err := handler.GetUpstreamAuth(c)
	assert.NoError(t, err)
	assert.Empty(t, username)
	assert.Empty(t, password)
}

// TestAPTHandlerV2_ValidateClientAuth tests client authentication
func TestAPTHandlerV2_ValidateClientAuth(t *testing.T) {
	handler := &APTHandlerV2{}
	app := fiber.New()
	c := app.AcquireCtx(&fasthttp.RequestCtx{})
	defer app.ReleaseCtx(c)

	err := handler.ValidateClientAuth(c)
	assert.NoError(t, err)
}

// TestAPTHandlerV2_HealthCheck tests health check functionality
func TestAPTHandlerV2_HealthCheck(t *testing.T) {
	tests := []struct {
		name        string
		config      *configs.AptProxyConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "healthy configuration",
			config: &configs.AptProxyConfig{
				Proxies: map[string][]configs.AptProxy{
					"ubuntu": {{Name: "main", URL: "http://archive.ubuntu.com/ubuntu"}},
					"debian": {{Name: "main", URL: "http://deb.debian.org/debian"}},
				},
			},
			expectError: false,
		},
		{
			name:        "nil configuration",
			config:      nil,
			expectError: true,
			errorMsg:    "APT proxy configuration not loaded",
		},
		{
			name: "no proxies configured",
			config: &configs.AptProxyConfig{
				Proxies: map[string][]configs.AptProxy{},
			},
			expectError: true,
			errorMsg:    "no APT proxies configured",
		},
		{
			name: "invalid mirror URL",
			config: &configs.AptProxyConfig{
				Proxies: map[string][]configs.AptProxy{
					"ubuntu": {{Name: "invalid", URL: "not-a-url"}},
				},
			},
			expectError: true,
			errorMsg:    "invalid mirror URL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := &APTHandlerV2{config: tt.config}
			err := handler.HealthCheck()

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// BenchmarkAPTHandlerV2_GenerateCacheKey benchmarks cache key generation
func BenchmarkAPTHandlerV2_GenerateCacheKey(b *testing.B) {
	handler := &APTHandlerV2{}
	app := fiber.New()
	c := app.AcquireCtx(&fasthttp.RequestCtx{})
	defer app.ReleaseCtx(c)

	c.Request().SetRequestURI("/proxy/apt/ubuntu/dists/focal/main/binary-amd64/Packages")
	c.Locals("osType", "ubuntu")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = handler.GenerateCacheKey(c)
	}
}

// BenchmarkAPTHandlerV2_GetCacheTTL benchmarks cache TTL calculation
func BenchmarkAPTHandlerV2_GetCacheTTL(b *testing.B) {
	handler := &APTHandlerV2{}
	app := fiber.New()
	c := app.AcquireCtx(&fasthttp.RequestCtx{})
	defer app.ReleaseCtx(c)

	paths := []string{
		"/proxy/apt/ubuntu/dists/focal/Release",
		"/proxy/apt/ubuntu/pool/main/v/vim/vim_8.2.deb",
		"/proxy/apt/ubuntu/dists/focal/main/binary-amd64/Packages.gz",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Request().SetRequestURI(paths[i%len(paths)])
		_ = handler.GetCacheTTL(c)
	}
}