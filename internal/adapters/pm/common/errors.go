package common

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"proxynd/internal/ports"
)

// ErrorMapper implements ports.ErrorMapper for all package managers
type ErrorMapper struct{}

// NewErrorMapper creates a new error mapper
func NewErrorMapper() ports.ErrorMapper {
	return &ErrorMapper{}
}

// ProxyError represents a standardized proxy error
type ProxyError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	PMType     string `json:"pm_type"`
	HTTPStatus int    `json:"http_status"`
	Retryable  bool   `json:"retryable"`
	Cause      error  `json:"-"`
}

func (e *ProxyError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func (e *ProxyError) Unwrap() error {
	return e.Cause
}

// Standard error codes
const (
	// Network errors
	ErrCodeNetworkTimeout   = "NETWORK_TIMEOUT"
	ErrCodeConnectionFailed = "CONNECTION_FAILED"
	ErrCodeDNSResolution    = "DNS_RESOLUTION_FAILED"

	// Authentication errors
	ErrCodeUnauthorized       = "UNAUTHORIZED"
	ErrCodeForbidden          = "FORBIDDEN"
	ErrCodeInvalidCredentials = "INVALID_CREDENTIALS"

	// Resource errors
	ErrCodeNotFound      = "NOT_FOUND"
	ErrCodeAlreadyExists = "ALREADY_EXISTS"
	ErrCodeInvalidFormat = "INVALID_FORMAT"

	// Validation errors
	ErrCodeInvalidPath     = "INVALID_PATH"
	ErrCodeInvalidVersion  = "INVALID_VERSION"
	ErrCodeInvalidChecksum = "INVALID_CHECKSUM"

	// Service errors
	ErrCodeServiceUnavailable = "SERVICE_UNAVAILABLE"
	ErrCodeRateLimited        = "RATE_LIMITED"
	ErrCodeUpstreamError      = "UPSTREAM_ERROR"

	// Configuration errors
	ErrCodeConfigurationError = "CONFIGURATION_ERROR"
	ErrCodeRepositoryNotFound = "REPOSITORY_NOT_FOUND"

	// Cache errors
	ErrCodeCacheError = "CACHE_ERROR"
	ErrCodeCacheMiss  = "CACHE_MISS"
)

// MapError maps driver-specific errors to standard proxy errors
func (m *ErrorMapper) MapError(pmType string, err error) error {
	if err == nil {
		return nil
	}

	// If it's already a ProxyError, return as-is
	var proxyErr *ProxyError
	if errors.As(err, &proxyErr) {
		return err
	}

	// Map common HTTP errors first
	if httpErr := m.mapHTTPError(pmType, err); httpErr != nil {
		return httpErr
	}

	// Map package manager specific errors
	switch pmType {
	case "maven":
		return m.mapMavenError(err)
	case "npm":
		return m.mapNpmError(err)
	case "apt":
		return m.mapAptError(err)
	case "pypi":
		return m.mapPypiError(err)
	case "yum":
		return m.mapYumError(err)
	case "apk":
		return m.mapApkError(err)
	case "registry":
		return m.mapRegistryError(err)
	default:
		return m.mapGenericError(err)
	}
}

// IsRetryableError checks if error is retryable
func (m *ErrorMapper) IsRetryableError(err error) bool {
	var proxyErr *ProxyError
	if errors.As(err, &proxyErr) {
		return proxyErr.Retryable
	}

	// Check error message for retryable conditions
	errMsg := strings.ToLower(err.Error())
	retryableKeywords := []string{
		"timeout",
		"connection refused",
		"network",
		"temporary",
		"unavailable",
		"rate limit",
		"too many requests",
		"502",
		"503",
		"504",
	}

	for _, keyword := range retryableKeywords {
		if strings.Contains(errMsg, keyword) {
			return true
		}
	}

	return false
}

// IsNotFoundError checks if error represents "not found"
func (m *ErrorMapper) IsNotFoundError(err error) bool {
	var proxyErr *ProxyError
	if errors.As(err, &proxyErr) {
		return proxyErr.Code == ErrCodeNotFound
	}

	errMsg := strings.ToLower(err.Error())
	notFoundKeywords := []string{
		"not found",
		"404",
		"no such",
		"does not exist",
		"missing",
	}

	for _, keyword := range notFoundKeywords {
		if strings.Contains(errMsg, keyword) {
			return true
		}
	}

	return false
}

// IsForbiddenError checks if error represents "forbidden"
func (m *ErrorMapper) IsForbiddenError(err error) bool {
	var proxyErr *ProxyError
	if errors.As(err, &proxyErr) {
		return proxyErr.Code == ErrCodeForbidden || proxyErr.Code == ErrCodeUnauthorized
	}

	errMsg := strings.ToLower(err.Error())
	forbiddenKeywords := []string{
		"forbidden",
		"unauthorized",
		"access denied",
		"401",
		"403",
		"permission denied",
	}

	for _, keyword := range forbiddenKeywords {
		if strings.Contains(errMsg, keyword) {
			return true
		}
	}

	return false
}

// HTTP error mapping
func (m *ErrorMapper) mapHTTPError(pmType string, err error) error {
	errMsg := strings.ToLower(err.Error())

	// Check for HTTP status codes in error message
	if strings.Contains(errMsg, "401") || strings.Contains(errMsg, "unauthorized") {
		return &ProxyError{
			Code:       ErrCodeUnauthorized,
			Message:    "Authentication required",
			PMType:     pmType,
			HTTPStatus: http.StatusUnauthorized,
			Retryable:  false,
			Cause:      err,
		}
	}

	if strings.Contains(errMsg, "403") || strings.Contains(errMsg, "forbidden") {
		return &ProxyError{
			Code:       ErrCodeForbidden,
			Message:    "Access forbidden",
			PMType:     pmType,
			HTTPStatus: http.StatusForbidden,
			Retryable:  false,
			Cause:      err,
		}
	}

	if strings.Contains(errMsg, "404") || strings.Contains(errMsg, "not found") {
		return &ProxyError{
			Code:       ErrCodeNotFound,
			Message:    "Resource not found",
			PMType:     pmType,
			HTTPStatus: http.StatusNotFound,
			Retryable:  false,
			Cause:      err,
		}
	}

	if strings.Contains(errMsg, "429") || strings.Contains(errMsg, "rate limit") {
		return &ProxyError{
			Code:       ErrCodeRateLimited,
			Message:    "Rate limit exceeded",
			PMType:     pmType,
			HTTPStatus: http.StatusTooManyRequests,
			Retryable:  true,
			Cause:      err,
		}
	}

	if strings.Contains(errMsg, "502") || strings.Contains(errMsg, "503") ||
		strings.Contains(errMsg, "504") || strings.Contains(errMsg, "unavailable") {
		return &ProxyError{
			Code:       ErrCodeServiceUnavailable,
			Message:    "Service temporarily unavailable",
			PMType:     pmType,
			HTTPStatus: http.StatusServiceUnavailable,
			Retryable:  true,
			Cause:      err,
		}
	}

	if strings.Contains(errMsg, "timeout") {
		return &ProxyError{
			Code:       ErrCodeNetworkTimeout,
			Message:    "Network timeout",
			PMType:     pmType,
			HTTPStatus: http.StatusGatewayTimeout,
			Retryable:  true,
			Cause:      err,
		}
	}

	return nil
}

// Maven-specific error mapping
func (m *ErrorMapper) mapMavenError(err error) error {
	errMsg := strings.ToLower(err.Error())

	if strings.Contains(errMsg, "invalid pom") || strings.Contains(errMsg, "malformed") {
		return &ProxyError{
			Code:       ErrCodeInvalidFormat,
			Message:    "Invalid Maven POM format",
			PMType:     "maven",
			HTTPStatus: http.StatusBadRequest,
			Retryable:  false,
			Cause:      err,
		}
	}

	if strings.Contains(errMsg, "repository") && strings.Contains(errMsg, "not configured") {
		return &ProxyError{
			Code:       ErrCodeRepositoryNotFound,
			Message:    "Maven repository not configured",
			PMType:     "maven",
			HTTPStatus: http.StatusBadRequest,
			Retryable:  false,
			Cause:      err,
		}
	}

	return m.mapGenericError(err)
}

// NPM-specific error mapping
func (m *ErrorMapper) mapNpmError(err error) error {
	errMsg := strings.ToLower(err.Error())

	if strings.Contains(errMsg, "package.json") && strings.Contains(errMsg, "invalid") {
		return &ProxyError{
			Code:       ErrCodeInvalidFormat,
			Message:    "Invalid package.json format",
			PMType:     "npm",
			HTTPStatus: http.StatusBadRequest,
			Retryable:  false,
			Cause:      err,
		}
	}

	if strings.Contains(errMsg, "registry") && strings.Contains(errMsg, "not configured") {
		return &ProxyError{
			Code:       ErrCodeRepositoryNotFound,
			Message:    "NPM registry not configured",
			PMType:     "npm",
			HTTPStatus: http.StatusBadRequest,
			Retryable:  false,
			Cause:      err,
		}
	}

	return m.mapGenericError(err)
}

// APT-specific error mapping
func (m *ErrorMapper) mapAptError(err error) error {
	errMsg := strings.ToLower(err.Error())

	if strings.Contains(errMsg, "release") && strings.Contains(errMsg, "invalid") {
		return &ProxyError{
			Code:       ErrCodeInvalidFormat,
			Message:    "Invalid APT Release file",
			PMType:     "apt",
			HTTPStatus: http.StatusBadRequest,
			Retryable:  false,
			Cause:      err,
		}
	}

	return m.mapGenericError(err)
}

// PyPI-specific error mapping
func (m *ErrorMapper) mapPypiError(err error) error {
	return m.mapGenericError(err)
}

// YUM-specific error mapping
func (m *ErrorMapper) mapYumError(err error) error {
	return m.mapGenericError(err)
}

// APK-specific error mapping
func (m *ErrorMapper) mapApkError(err error) error {
	return m.mapGenericError(err)
}

// Docker registry-specific error mapping
func (m *ErrorMapper) mapRegistryError(err error) error {
	errMsg := strings.ToLower(err.Error())

	if strings.Contains(errMsg, "manifest") && strings.Contains(errMsg, "invalid") {
		return &ProxyError{
			Code:       ErrCodeInvalidFormat,
			Message:    "Invalid Docker manifest",
			PMType:     "registry",
			HTTPStatus: http.StatusBadRequest,
			Retryable:  false,
			Cause:      err,
		}
	}

	return m.mapGenericError(err)
}

// Generic error mapping
func (m *ErrorMapper) mapGenericError(err error) error {
	errMsg := strings.ToLower(err.Error())

	// Connection errors
	if strings.Contains(errMsg, "connection refused") || strings.Contains(errMsg, "connection failed") {
		return &ProxyError{
			Code:       ErrCodeConnectionFailed,
			Message:    "Connection to upstream failed",
			PMType:     "generic",
			HTTPStatus: http.StatusBadGateway,
			Retryable:  true,
			Cause:      err,
		}
	}

	// DNS errors
	if strings.Contains(errMsg, "no such host") || strings.Contains(errMsg, "dns") {
		return &ProxyError{
			Code:       ErrCodeDNSResolution,
			Message:    "DNS resolution failed",
			PMType:     "generic",
			HTTPStatus: http.StatusBadGateway,
			Retryable:  true,
			Cause:      err,
		}
	}

	// Generic upstream error
	return &ProxyError{
		Code:       ErrCodeUpstreamError,
		Message:    "Upstream error",
		PMType:     "generic",
		HTTPStatus: http.StatusBadGateway,
		Retryable:  true,
		Cause:      err,
	}
}
