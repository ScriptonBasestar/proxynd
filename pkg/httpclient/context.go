// Package httpclient provides HTTP client utilities
package httpclient

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ContextClient wraps an HTTP client with context-aware methods
type ContextClient struct {
	client *http.Client
}

// NewContextClient creates a new context-aware HTTP client
func NewContextClient(client *http.Client) *ContextClient {
	if client == nil {
		client = NewDefault()
	}
	return &ContextClient{client: client}
}

// NewContextClientWithConfig creates a new context-aware HTTP client with config
func NewContextClientWithConfig(config Config) *ContextClient {
	return &ContextClient{client: New(config)}
}

// Get performs a GET request with context
func (c *ContextClient) Get(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	return c.client.Do(req)
}

// GetWithTimeout performs a GET request with a specific timeout
func (c *ContextClient) GetWithTimeout(url string, timeout time.Duration) (*http.Response, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return c.Get(ctx, url)
}

// Post performs a POST request with context
func (c *ContextClient) Post(ctx context.Context, url string, contentType string,
	body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	return c.client.Do(req)
}

// Do performs a request with context
func (c *ContextClient) Do(req *http.Request) (*http.Response, error) {
	if req.Context() == nil {
		return nil, fmt.Errorf("request must have a context")
	}

	return c.client.Do(req)
}

// DoWithTimeout performs a request with a specific timeout
func (c *ContextClient) DoWithTimeout(req *http.Request, timeout time.Duration) (*http.Response, error) {
	ctx, cancel := context.WithTimeout(req.Context(), timeout)
	defer cancel()

	return c.client.Do(req.WithContext(ctx))
}

// ProxyClient is a specialized client for proxy operations
type ProxyClient struct {
	*ContextClient
	defaultTimeout time.Duration
}

// NewProxyClient creates a new proxy-specific HTTP client
func NewProxyClient() *ProxyClient {
	return &ProxyClient{
		ContextClient:  NewContextClientWithConfig(ProxyConfig()),
		defaultTimeout: 60 * time.Second,
	}
}

// GetWithRetry performs a GET request with retry logic
func (p *ProxyClient) GetWithRetry(ctx context.Context, url string, maxRetries int) (*http.Response, error) {
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			backoff := time.Duration(attempt*attempt) * time.Second
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}

		resp, err := p.Get(ctx, url)
		if err != nil {
			lastErr = err
			continue
		}

		// Success or non-retryable status
		if resp.StatusCode < 500 {
			return resp, nil
		}

		// Close body for retry
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
		lastErr = fmt.Errorf("server error: %d", resp.StatusCode)
	}

	return nil, fmt.Errorf("failed after %d retries: %w", maxRetries, lastErr)
}

// GetProxyDefault performs a GET request with default proxy timeout
func (p *ProxyClient) GetProxyDefault(url string) (*http.Response, error) {
	return p.GetWithTimeout(url, p.defaultTimeout)
}
