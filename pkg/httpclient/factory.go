package httpclient

import (
	"net"
	"net/http"
	"time"
)

// Config represents HTTP client configuration
type Config struct {
	// Timeout is the maximum time for a complete request
	Timeout time.Duration

	// ConnectTimeout is the maximum time to establish a connection
	ConnectTimeout time.Duration

	// KeepAliveTimeout is the keep-alive period for an active network connection
	KeepAliveTimeout time.Duration

	// MaxIdleConns controls the maximum number of idle connections across all hosts
	MaxIdleConns int

	// MaxIdleConnsPerHost controls the maximum idle connections to keep per-host
	MaxIdleConnsPerHost int

	// MaxConnsPerHost limits the total connections per host
	MaxConnsPerHost int

	// IdleConnTimeout is the maximum time an idle connection remains idle before closing
	IdleConnTimeout time.Duration

	// DisableCompression disables transport compression
	DisableCompression bool

	// DisableKeepAlives disables HTTP keep-alives
	DisableKeepAlives bool
}

// DefaultConfig returns a production-ready default configuration
func DefaultConfig() Config {
	return Config{
		Timeout:             30 * time.Second,
		ConnectTimeout:      10 * time.Second,
		KeepAliveTimeout:    30 * time.Second,
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		MaxConnsPerHost:     0, // No limit by default
		IdleConnTimeout:     90 * time.Second,
		DisableCompression:  false,
		DisableKeepAlives:   false,
	}
}

// ProxyConfig returns configuration optimized for proxy operations
func ProxyConfig() Config {
	return Config{
		Timeout:             60 * time.Second, // Longer timeout for large files
		ConnectTimeout:      10 * time.Second,
		KeepAliveTimeout:    30 * time.Second,
		MaxIdleConns:        200,
		MaxIdleConnsPerHost: 20,
		MaxConnsPerHost:     50,
		IdleConnTimeout:     90 * time.Second,
		DisableCompression:  true, // Proxy should not compress/decompress
		DisableKeepAlives:   false,
	}
}

// New creates a new HTTP client with the given configuration
func New(config Config) *http.Client {
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   config.ConnectTimeout,
			KeepAlive: config.KeepAliveTimeout,
		}).DialContext,
		MaxIdleConns:          config.MaxIdleConns,
		MaxIdleConnsPerHost:   config.MaxIdleConnsPerHost,
		MaxConnsPerHost:       config.MaxConnsPerHost,
		IdleConnTimeout:       config.IdleConnTimeout,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		DisableCompression:    config.DisableCompression,
		DisableKeepAlives:     config.DisableKeepAlives,
	}

	return &http.Client{
		Transport: transport,
		Timeout:   config.Timeout,
	}
}

// NewDefault creates a new HTTP client with default configuration
func NewDefault() *http.Client {
	return New(DefaultConfig())
}

// NewProxy creates a new HTTP client optimized for proxy operations
func NewProxy() *http.Client {
	return New(ProxyConfig())
}
