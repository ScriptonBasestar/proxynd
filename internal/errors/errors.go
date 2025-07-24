// Package errors provides centralized error management for ProxyND
package errors

import (
	"fmt"
)

const (
	// ErrCodeProxyUpstreamConnection represents proxy upstream connection error code
	ErrCodeProxyUpstreamConnection = "PROXY_UPSTREAM_CONNECTION"
	// ErrCodeProxyUpstreamTimeout represents proxy upstream timeout error code
	ErrCodeProxyUpstreamTimeout = "PROXY_UPSTREAM_TIMEOUT"
	// ErrCodeProxyGeneric represents generic proxy error code
	ErrCodeProxyGeneric = "PROXY_GENERIC"

	// ErrCodeCacheRead represents cache read error code
	ErrCodeCacheRead = "CACHE_READ"
	// ErrCodeCacheWrite represents cache write error code
	ErrCodeCacheWrite = "CACHE_WRITE"

	// ErrCodeAuthenticationFailed represents authentication failed error code
	ErrCodeAuthenticationFailed = "AUTH_FAILED"
	// ErrCodeAuthorizationFailed represents authorization failed error code
	ErrCodeAuthorizationFailed = "AUTHZ_FAILED"
)

// New creates a new error with the given message
func New(message string) error {
	return fmt.Errorf("%s", message)
}

// NewDomainError creates a new domain error with the existing DomainError type
func NewDomainError(code, message string, err error) error {
	return NewError(code, message).
		WithCause(err).
		Build()
}
