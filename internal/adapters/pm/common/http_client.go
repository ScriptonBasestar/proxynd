package common

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"proxynd/internal/ports"
)

// HTTPClient implements ports.HTTPClient with connection pooling and best practices
type HTTPClient struct {
	client *http.Client
	config *HTTPClientConfig
}

// HTTPClientConfig represents HTTP client configuration
type HTTPClientConfig struct {
	Timeout             time.Duration `json:"timeout" yaml:"timeout"`
	MaxIdleConns        int           `json:"max_idle_conns" yaml:"max_idle_conns"`
	MaxIdleConnsPerHost int           `json:"max_idle_conns_per_host" yaml:"max_idle_conns_per_host"`
	MaxConnsPerHost     int           `json:"max_conns_per_host" yaml:"max_conns_per_host"`
	IdleConnTimeout     time.Duration `json:"idle_conn_timeout" yaml:"idle_conn_timeout"`
	TLSHandshakeTimeout time.Duration `json:"tls_handshake_timeout" yaml:"tls_handshake_timeout"`
	ResponseHeaderTimeout time.Duration `json:"response_header_timeout" yaml:"response_header_timeout"`
	ExpectContinueTimeout time.Duration `json:"expect_continue_timeout" yaml:"expect_continue_timeout"`
	DialTimeout         time.Duration `json:"dial_timeout" yaml:"dial_timeout"`
	KeepAlive           time.Duration `json:"keep_alive" yaml:"keep_alive"`
	DisableCompression  bool          `json:"disable_compression" yaml:"disable_compression"`
	DisableKeepAlives   bool          `json:"disable_keep_alives" yaml:"disable_keep_alives"`
	UserAgent           string        `json:"user_agent" yaml:"user_agent"`
}

// DefaultHTTPClientConfig returns default HTTP client configuration
func DefaultHTTPClientConfig() *HTTPClientConfig {
	return &HTTPClientConfig{
		Timeout:               30 * time.Second,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
		MaxConnsPerHost:       50,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		DialTimeout:           10 * time.Second,
		KeepAlive:             30 * time.Second,
		DisableCompression:    false,
		DisableKeepAlives:     false,
		UserAgent:             "ProxyND/1.0",
	}
}

// NewHTTPClient creates a new HTTP client with connection pooling
func NewHTTPClient(config *HTTPClientConfig) ports.HTTPClient {
	if config == nil {
		config = DefaultHTTPClientConfig()
	}
	
	transport := &http.Transport{
		MaxIdleConns:          config.MaxIdleConns,
		MaxIdleConnsPerHost:   config.MaxIdleConnsPerHost,
		MaxConnsPerHost:       config.MaxConnsPerHost,
		IdleConnTimeout:       config.IdleConnTimeout,
		TLSHandshakeTimeout:   config.TLSHandshakeTimeout,
		ResponseHeaderTimeout: config.ResponseHeaderTimeout,
		ExpectContinueTimeout: config.ExpectContinueTimeout,
		DisableCompression:    config.DisableCompression,
		DisableKeepAlives:     config.DisableKeepAlives,
		ForceAttemptHTTP2:     true,
	}
	
	client := &http.Client{
		Transport: transport,
		Timeout:   config.Timeout,
	}
	
	return &HTTPClient{
		client: client,
		config: config,
	}
}

// Get performs a GET request
func (c *HTTPClient) Get(ctx context.Context, url string, headers map[string]string) (*ports.ProxyHTTPResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create GET request: %w", err)
	}
	
	return c.doRequest(req, headers)
}

// Post performs a POST request
func (c *HTTPClient) Post(ctx context.Context, url string, body io.Reader, headers map[string]string) (*ports.ProxyHTTPResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create POST request: %w", err)
	}
	
	return c.doRequest(req, headers)
}

// Put performs a PUT request
func (c *HTTPClient) Put(ctx context.Context, url string, body io.Reader, headers map[string]string) (*ports.ProxyHTTPResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create PUT request: %w", err)
	}
	
	return c.doRequest(req, headers)
}

// Delete performs a DELETE request
func (c *HTTPClient) Delete(ctx context.Context, url string, headers map[string]string) (*ports.ProxyHTTPResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create DELETE request: %w", err)
	}
	
	return c.doRequest(req, headers)
}

// Head performs a HEAD request
func (c *HTTPClient) Head(ctx context.Context, url string, headers map[string]string) (*ports.ProxyHTTPResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create HEAD request: %w", err)
	}
	
	return c.doRequest(req, headers)
}

// doRequest executes the HTTP request with common handling
func (c *HTTPClient) doRequest(req *http.Request, headers map[string]string) (*ports.ProxyHTTPResponse, error) {
	// Set headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	
	// Set default User-Agent if not provided
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", c.config.UserAgent)
	}
	
	// Set common headers for proxy requests
	if req.Header.Get("Accept") == "" {
		req.Header.Set("Accept", "*/*")
	}
	
	// Add compression support if not disabled
	if !c.config.DisableCompression && req.Header.Get("Accept-Encoding") == "" {
		req.Header.Set("Accept-Encoding", "gzip, deflate")
	}
	
	// Execute request
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request failed: %w", err)
	}
	
	// Convert response headers to map
	responseHeaders := make(map[string]string)
	for key, values := range resp.Header {
		if len(values) > 0 {
			responseHeaders[key] = values[0]
		}
	}
	
	// Get content length
	var size int64
	if resp.ContentLength > 0 {
		size = resp.ContentLength
	}
	
	return &ports.ProxyHTTPResponse{
		Body:       resp.Body,
		StatusCode: resp.StatusCode,
		Headers:    responseHeaders,
		Size:       size,
	}, nil
}

// Close closes the HTTP client and releases resources
func (c *HTTPClient) Close() error {
	if transport, ok := c.client.Transport.(*http.Transport); ok {
		transport.CloseIdleConnections()
	}
	return nil
}

// GetConfig returns the HTTP client configuration
func (c *HTTPClient) GetConfig() *HTTPClientConfig {
	return c.config
}

// GetStats returns HTTP client statistics
func (c *HTTPClient) GetStats() map[string]interface{} {
	stats := make(map[string]interface{})
	
	if transport, ok := c.client.Transport.(*http.Transport); ok {
		stats["max_idle_conns"] = transport.MaxIdleConns
		stats["max_idle_conns_per_host"] = transport.MaxIdleConnsPerHost
		stats["max_conns_per_host"] = transport.MaxConnsPerHost
		stats["idle_conn_timeout"] = transport.IdleConnTimeout
		stats["tls_handshake_timeout"] = transport.TLSHandshakeTimeout
		stats["response_header_timeout"] = transport.ResponseHeaderTimeout
	}
	
	stats["timeout"] = c.client.Timeout
	stats["user_agent"] = c.config.UserAgent
	
	return stats
}

// TODO: Integration points with existing codebase
// 1. Replace fiber.Agent usage in handlers/proxy/ with this unified client
// 2. Integrate with internal/pool/ connection pool if needed
// 3. Add metrics collection hooks for monitoring
// 4. Add circuit breaker pattern for resilience
// 5. Add request/response logging hooks
// 6. Add retry mechanism with exponential backoff
// 7. Add request rate limiting
// 8. Add request tracing hooks for observability