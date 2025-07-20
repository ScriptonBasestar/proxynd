package errors

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCacheErrors(t *testing.T) {
	t.Run("Cache error constants", func(t *testing.T) {
		// Test predefined cache errors
		assert.Equal(t, "CACHE001", ErrCacheNotFound.Code)
		assert.Equal(t, "CACHE002", ErrCacheExpired.Code)
		assert.Equal(t, "CACHE003", ErrCacheInvalidData.Code)

		// Test errorabout:blank#blocked code constants
		assert.Equal(t, "CACHE_READ", ErrCodeCacheRead)
		assert.Equal(t, "CACHE_WRITE", ErrCodeCacheWrite)
	})

	t.Run("NewCacheError", func(t *testing.T) {
		domainErr := NewCacheError("CACHE_CUSTOM", "Custom cache error").Build()

		assert.Equal(t, "CACHE_CUSTOM", domainErr.Code)
		assert.Equal(t, "cache", domainErr.Domain)
		assert.Equal(t, "Custom cache error", domainErr.Message)
	})

	t.Run("WrapCacheError", func(t *testing.T) {
		originalErr := errors.New("redis: connection refused")
		domainErr := WrapCacheError(originalErr, ErrCodeCacheRead, "Failed to read from cache")

		assert.Equal(t, ErrCodeCacheRead, domainErr.Code)
		assert.Equal(t, "cache", domainErr.Domain)
		assert.Equal(t, "Failed to read from cache", domainErr.Message)
		assert.Equal(t, originalErr, domainErr.Cause)
	})

	t.Run("Cache error scenarios", func(t *testing.T) {
		scenarios := []struct {
			name    string
			code    string
			message string
			cause   error
			details map[string]interface{}
		}{
			{
				name:    "Cache miss",
				code:    ErrCodeCacheRead,
				message: "Key not found in cache",
				details: map[string]interface{}{
					"key":       "user:12345",
					"namespace": "users",
				},
			},
			{
				name:    "Cache write failure",
				code:    ErrCodeCacheWrite,
				message: "Failed to write to cache",
				cause:   errors.New("disk full"),
				details: map[string]interface{}{
					"key":  "session:abc123",
					"size": 1024,
				},
			},
			{
				name:    "Cache invalidation",
				code:    "CACHE_INVALIDATE",
				message: "Failed to invalidate cache entries",
				details: map[string]interface{}{
					"pattern": "user:*",
					"count":   150,
				},
			},
		}

		for _, sc := range scenarios {
			t.Run(sc.name, func(t *testing.T) {
				builder := NewCacheError(sc.code, sc.message)

				if sc.cause != nil {
					builder = builder.WithCause(sc.cause)
				}

				if sc.details != nil {
					builder = builder.WithDetails(sc.details)
				}

				domainErr := builder.Build()

				assert.Equal(t, sc.code, domainErr.Code)
				assert.Equal(t, "cache", domainErr.Domain)
				assert.Equal(t, sc.message, domainErr.Message)

				if sc.cause != nil {
					assert.Equal(t, sc.cause, domainErr.Cause)
				}

				if sc.details != nil {
					details := domainErr.Details.(map[string]interface{})
					for k, v := range sc.details {
						assert.Equal(t, v, details[k])
					}
				}
			})
		}
	})
}

func TestSystemErrors(t *testing.T) {
	t.Run("System error constants", func(t *testing.T) {
		assert.Equal(t, "SYS001", ErrSystemInternal.Code)
		assert.Equal(t, "SYS002", ErrConfigInvalid.Code)
		assert.Equal(t, "SYS003", ErrCodePanic)
	})

	t.Run("NewSystemError", func(t *testing.T) {
		domainErr := NewSystemError("SYS_CUSTOM", "Custom system error").Build()

		assert.Equal(t, "SYS_CUSTOM", domainErr.Code)
		assert.Equal(t, "system", domainErr.Domain)
	})

	t.Run("System error scenarios", func(t *testing.T) {
		scenarios := []struct {
			name    string
			code    string
			message string
			level   ErrorLevel
			details map[string]interface{}
		}{
			{
				name:    "Out of memory",
				code:    "SYS_OOM",
				message: "System out of memory",
				level:   ErrorLevelCritical,
				details: map[string]interface{}{
					"available": "100MB",
					"required":  "1GB",
				},
			},
			{
				name:    "File system error",
				code:    "SYS_FS_ERROR",
				message: "File system operation failed",
				level:   ErrorLevelError,
				details: map[string]interface{}{
					"operation": "write",
					"path":      "/var/log/app.log",
					"errno":     28, // ENOSPC
				},
			},
			{
				name:    "Configuration missing",
				code:    "CONFIG_MISSING",
				message: "Required configuration not found",
				level:   ErrorLevelCritical,
				details: map[string]interface{}{
					"config_key": "database.url",
					"file":       "config.yaml",
				},
			},
		}

		for _, sc := range scenarios {
			t.Run(sc.name, func(t *testing.T) {
				domainErr := NewSystemError(sc.code, sc.message).
					WithLevel(sc.level).
					WithDetails(sc.details).
					Build()

				assert.Equal(t, sc.code, domainErr.Code)
				assert.Equal(t, "system", domainErr.Domain)
				assert.Equal(t, sc.level, domainErr.Level)

				details := domainErr.Details.(map[string]interface{})
				for k, v := range sc.details {
					assert.Equal(t, v, details[k])
				}
			})
		}
	})

	t.Run("Panic error creation", func(t *testing.T) {
		panicValue := "test panic"
		stackTrace := "goroutine 1 [running]:\nmain.main()\n\t/app/main.go:10"

		domainErr := NewSystemError(ErrCodePanic, "Panic recovered").
			WithLevel(ErrorLevelCritical).
			WithDetails(map[string]interface{}{
				"panic": panicValue,
				"stack": stackTrace,
			}).
			Build()
		assert.Equal(t, ErrCodePanic, domainErr.Code)
		assert.Equal(t, ErrorLevelCritical, domainErr.Level)

		details := domainErr.Details.(map[string]interface{})
		assert.Equal(t, panicValue, details["panic"])
		assert.Equal(t, stackTrace, details["stack"])
	})
}

func TestAuthErrors(t *testing.T) {
	t.Run("Auth error constants", func(t *testing.T) {
		assert.Equal(t, "AUTH001", ErrAuthInvalidCredentials.Code)
		assert.Equal(t, "AUTH002", ErrAuthTokenExpired.Code)
		assert.Equal(t, "AUTH_FAILED", ErrCodeAuthFailed)
		assert.Equal(t, "AUTHZ_FAILED", ErrCodeAuthzFailed)
	})

	t.Run("NewAuthError", func(t *testing.T) {
		domainErr := NewAuthError("AUTH_CUSTOM", "Custom auth error").Build()

		assert.Equal(t, "AUTH_CUSTOM", domainErr.Code)
		assert.Equal(t, "auth", domainErr.Domain)
	})

	t.Run("Auth error scenarios", func(t *testing.T) {
		scenarios := []struct {
			name    string
			code    string
			message string
			details map[string]interface{}
		}{
			{
				name:    "Invalid credentials",
				code:    ErrCodeAuthFailed,
				message: "Invalid username or password",
				details: map[string]interface{}{
					"username": "john.doe",
					"ip":       "192.168.1.100",
					"attempts": 3,
				},
			},
			{
				name:    "Token expired",
				code:    "AUTH_TOKEN_EXPIRED",
				message: "Authentication token has expired",
				details: map[string]interface{}{
					"token_id":   "abc123",
					"expired_at": "2024-01-15T10:30:00Z",
					"issued_at":  "2024-01-15T09:30:00Z",
				},
			},
			{
				name:    "Insufficient permissions",
				code:    ErrCodeAuthzFailed,
				message: "User does not have required permissions",
				details: map[string]interface{}{
					"user_id":  "user123",
					"resource": "/api/admin/users",
					"action":   "DELETE",
					"required": []string{"admin", "user:delete"},
					"actual":   []string{"user"},
				},
			},
			{
				name:    "MFA required",
				code:    "AUTH_MFA_REQUIRED",
				message: "Multi-factor authentication required",
				details: map[string]interface{}{
					"user_id":      "user456",
					"mfa_methods":  []string{"totp", "sms"},
					"challenge_id": "challenge789",
				},
			},
		}

		for _, sc := range scenarios {
			t.Run(sc.name, func(t *testing.T) {
				domainErr := NewAuthError(sc.code, sc.message).
					WithDetails(sc.details).
					Build()

				assert.Equal(t, sc.code, domainErr.Code)
				assert.Equal(t, "auth", domainErr.Domain)
				assert.Equal(t, sc.message, domainErr.Message)

				if sc.details != nil {
					assert.Equal(t, sc.details, domainErr.Details)
				}
			})
		}
	})

	t.Run("OAuth error scenarios", func(t *testing.T) {
		domainErr := NewAuthError("OAUTH_INVALID_GRANT", "Invalid authorization grant").
			WithDetails(map[string]interface{}{
				"grant_type":   "authorization_code",
				"client_id":    "client123",
				"redirect_uri": "https://app.example.com/callback",
			}).
			Build()
		assert.Equal(t, "OAUTH_INVALID_GRANT", domainErr.Code)
		assert.Equal(t, "auth", domainErr.Domain)
	})
}

func TestProxyErrors(t *testing.T) {
	t.Run("Proxy error constants", func(t *testing.T) {
		assert.Equal(t, "PROXY_UPSTREAM_CONNECTION", ErrCodeProxyUpstreamConnection)
		assert.Equal(t, "PROXY_UPSTREAM_TIMEOUT", ErrCodeProxyUpstreamTimeout)
		assert.Equal(t, "PROXY_GENERIC", ErrCodeProxyGeneric)
	})

	t.Run("Proxy error scenarios", func(t *testing.T) {
		scenarios := []struct {
			name    string
			code    string
			message string
			cause   error
			details map[string]interface{}
		}{
			{
				name:    "Upstream connection failed",
				code:    ErrCodeProxyUpstreamConnection,
				message: "Failed to connect to upstream server",
				cause:   errors.New("dial tcp: connection refused"),
				details: map[string]interface{}{
					"upstream_url": "https://repo.maven.org/maven2",
					"proxy_type":   "maven",
					"retry_count":  3,
				},
			},
			{
				name:    "Upstream timeout",
				code:    ErrCodeProxyUpstreamTimeout,
				message: "Upstream server request timeout",
				details: map[string]interface{}{
					"upstream_url": "https://archive.ubuntu.com/ubuntu",
					"proxy_type":   "apt",
					"timeout":      "30s",
					"package":      "nginx",
				},
			},
			{
				name:    "Invalid upstream response",
				code:    "PROXY_INVALID_RESPONSE",
				message: "Upstream server returned invalid response",
				details: map[string]interface{}{
					"status_code":   502,
					"content_type":  "text/html",
					"expected_type": "application/json",
				},
			},
		}

		for _, sc := range scenarios {
			t.Run(sc.name, func(t *testing.T) {
				builder := NewError(sc.code, sc.message).
					WithDomain("proxy").
					WithDetails(sc.details)

				if sc.cause != nil {
					builder = builder.WithCause(sc.cause)
				}

				domainErr := builder.Build()

				assert.Equal(t, sc.code, domainErr.Code)
				assert.Equal(t, "proxy", domainErr.Domain)

				if sc.cause != nil {
					assert.Equal(t, sc.cause, domainErr.Cause)
				}
			})
		}
	})
}

func TestDomainErrorHelperMethods(t *testing.T) {
	t.Run("NewDomainError", func(t *testing.T) {
		cause := errors.New("underlying error")
		err := NewDomainError("TEST001", "Test error", cause)

		domainErr, ok := err.(*DomainError)
		require.True(t, ok)

		assert.Equal(t, "TEST001", domainErr.Code)
		assert.Equal(t, "Test error", domainErr.Message)
		assert.Equal(t, cause, domainErr.Cause)
	})

	t.Run("Domain-specific wrap methods", func(t *testing.T) {
		originalErr := errors.New("original error")

		// Test each domain's wrap method
		tests := []struct {
			name     string
			wrapFunc func(error, string, string) *DomainError
			domain   string
		}{
			{
				name:     "WrapCacheError",
				wrapFunc: WrapCacheError,
				domain:   "cache",
			},
			{
				name:     "WrapMavenError",
				wrapFunc: WrapMavenError,
				domain:   "maven",
			},
			{
				name:     "WrapAPTError",
				wrapFunc: WrapAPTError,
				domain:   "apt",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				wrapped := tt.wrapFunc(originalErr, "CODE001", "Wrapped message")

				assert.Equal(t, "CODE001", wrapped.Code)
				assert.Equal(t, "Wrapped message", wrapped.Message)
				assert.Equal(t, tt.domain, wrapped.Domain)
				assert.Equal(t, originalErr, wrapped.Cause)
			})
		}
	})
}
