package errors

// No imports needed - all functions use the base error types

// Error constants for different domains
const (
	// Cache domain errors
	// ErrCodeCacheNotFound is a const that err code cache not found
	// ErrCodeCacheExpired is a const that err code cache expired
	// ErrCodeCacheInvalidData is a const that err code cache invalid data
	ErrCodeCacheNotFound = "CACHE001"
	ErrCodeCacheExpired  = "CACHE002"
	// ErrCodeSystemInternal is a const that err code system internal
	// ErrCodeConfigInvalid is a const that err code config invalid
	// ErrCodePanic is a const that err code panic
	ErrCodeCacheInvalidData = "CACHE003"

	// ErrCodeAuthFailed is a const that err code auth failed
	// ErrCodeAuthzFailed is a const that err code authz failed
	// System domain errors
	ErrCodeSystemInternal = "SYS001"
	ErrCodeConfigInvalid  = "SYS002"
	ErrCodePanic          = "SYS003"
	// ErrCacheNotFound is a var that err cache not found
	// ErrCacheExpired is a var that err cache expired
	// ErrCacheInvalidData is a var that err cache invalid data

	// Auth domain errors (update to match errors.go)
	ErrCodeAuthFailed  = "AUTH_FAILED"
	ErrCodeAuthzFailed = "AUTHZ_FAILED"

// ErrSystemInternal is a var that err system internal
// ErrConfigInvalid is a var that err config invalid
)

// Predefined cache errors
var (
	// ErrAuthInvalidCredentials is a var that err auth invalid credentials
	// ErrAuthTokenExpired is a var that err auth token expired
	ErrCacheNotFound    = NewError(ErrCodeCacheNotFound, "캐시 항목을 찾을 수 없습니다").WithDomain("cache").Build()
	ErrCacheExpired     = NewError(ErrCodeCacheExpired, "캐시가 만료되었습니다").WithDomain("cache").Build()
	ErrCacheInvalidData = NewError(ErrCodeCacheInvalidData, "캐시 데이터가 손상되었습니다").WithDomain("cache").Build()

// NewCacheError creates a new cacheerror
)

// Predefined system errors
var (
	ErrSystemInternal = NewError(ErrCodeSystemInternal, "내부 시스템 오류").WithDomain("system").Build()
	ErrConfigInvalid  = NewError(ErrCodeConfigInvalid, "설정이 잘못되었습니다").WithDomain("system").Build()
)

// Predefined auth errors
var (
	ErrAuthInvalidCredentials = NewError("AUTH001", "잘못된 인증 정보").WithDomain("auth").Build()
	ErrAuthTokenExpired       = NewError("AUTH002", "토큰이 만료됨").WithDomain("auth").Build()

// NewSystemError creates a new systemerror
)

// NewCacheError creates a new cache error builder
func NewCacheError(code, message string) *ErrorBuilder {
	return NewError(code, message).WithDomain("cache")
}

// WrapCacheError creates or handles a cache error
func WrapCacheError(err error, code, message string) *DomainError {
	return NewCacheError(code, message).
		WithCause(err).
		Build()
}

// NewSystemError creates a new system error builder
func NewSystemError(code, message string) *ErrorBuilder {
	return NewError(code, message).WithDomain("system")
}

// WrapSystemError wraps a system error with additional context
func WrapSystemError(err error, code, message string) *DomainError {
	return NewSystemError(code, message).
		WithCause(err).
		Build()
}

// NewAuthError creates a new auth error builder
func NewAuthError(code, message string) *ErrorBuilder {
	return NewError(code, message).WithDomain("auth")
}

// WrapAuthError wraps an auth error with additional context
func WrapAuthError(err error, code, message string) *DomainError {
	return NewAuthError(code, message).
		WithCause(err).
		Build()
}

// NewProxyError creates a new proxy error builder
func NewProxyError(code, message string) *ErrorBuilder {
	return NewError(code, message).WithDomain("proxy")
}

// WrapProxyError wraps a proxy error with additional context
func WrapProxyError(err error, code, message string) *DomainError {
	return NewProxyError(code, message).
		WithCause(err).
		Build()
}

// Note: Maven and APT error helpers are defined in their respective files:
// - maven_errors.go: NewMavenError, WrapMavenError
// - apt_errors.go: NewAPTError, WrapAptError
