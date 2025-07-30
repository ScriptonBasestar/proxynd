package adapters

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"proxynd/internal/pool"
	"proxynd/internal/services/proxy"
	"proxynd/logging"
)

// HTTPUpstreamClient implements the UpstreamClient interface using HTTP
type HTTPUpstreamClient struct {
	client             *http.Client
	logger             logging.Logger
	clientFactory      *pool.ProxyClientFactory
	performanceMonitor *pool.PerformanceMonitor
	proxyType          string
}

// NewHTTPUpstreamClient creates a new HTTP upstream client (레거시 호환용)
func NewHTTPUpstreamClient(timeout time.Duration) *HTTPUpstreamClient {
	return &HTTPUpstreamClient{
		client: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		logger: logging.GetLogger(),
	}
}

// NewPooledHTTPUpstreamClient Connection Pool을 사용하는 새로운 업스트림 클라이언트 생성
func NewPooledHTTPUpstreamClient(proxyType string, timeout time.Duration) *HTTPUpstreamClient {
	clientFactory := pool.GetGlobalClientFactory()

	return &HTTPUpstreamClient{
		client:             clientFactory.GetClientWithTimeout(proxyType, timeout),
		logger:             logging.GetLogger(),
		clientFactory:      clientFactory,
		performanceMonitor: pool.GetGlobalPerformanceMonitor(),
		proxyType:          proxyType,
	}
}

// Fetch retrieves content from upstream
func (c *HTTPUpstreamClient) Fetch(ctx context.Context, url string,
	headers map[string]string,
) (*proxy.ProxyResponse, error) {
	startTime := time.Now()

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// Set default User-Agent if not provided
	if req.Header.Get("User-Agent") == "" {
		userAgent := "ProxyND/1.0"
		if c.proxyType != "" {
			userAgent = fmt.Sprintf("ProxyND/1.0 %s-Proxy", c.proxyType)
		}
		req.Header.Set("User-Agent", userAgent)
	}

	// Log the request
	c.logger.Debug("Fetching from upstream",
		logging.F("url", url),
		logging.F("proxy_type", c.proxyType),
		logging.F("headers", len(headers)))

	// Execute request (Connection Pool 사용 또는 레거시)
	var resp *http.Response
	if c.clientFactory != nil && c.proxyType != "" {
		// Connection Pool을 통한 요청 실행
		resp, err = c.clientFactory.ExecuteProxyRequest(c.proxyType, req)
	} else {
		// 레거시 직접 실행
		resp, err = c.client.Do(req)
	}

	responseTime := time.Since(startTime)
	success := err == nil && resp != nil && resp.StatusCode < 400

	// 성능 모니터링 기록
	if c.performanceMonitor != nil && c.proxyType != "" {
		bytesReceived := int64(0)
		if resp != nil && resp.ContentLength > 0 {
			bytesReceived = resp.ContentLength
		}

		c.performanceMonitor.RecordRequest(c.proxyType, success, responseTime, bytesReceived, int64(req.ContentLength))

		if err != nil {
			c.performanceMonitor.RecordTimeout(c.proxyType)
		}
	}

	if err != nil {
		return nil, fmt.Errorf("upstream request failed: %w", err)
	}

	// Check response status
	if resp.StatusCode >= 400 {
		// Read error body
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			body = []byte("failed to read error response")
		}
		_ = resp.Body.Close()

		return &proxy.ProxyResponse{
			Body:        io.NopCloser(strings.NewReader(string(body))),
			StatusCode:  resp.StatusCode,
			Headers:     c.extractHeaders(resp),
			ContentType: resp.Header.Get("Content-Type"),
			Cached:      false,
		}, fmt.Errorf("upstream returned error: %d - %s", resp.StatusCode, string(body))
	}

	// Extract filename from Content-Disposition header if available
	filename := c.extractFilename(resp)

	// Build successful response
	return &proxy.ProxyResponse{
		Body:        resp.Body,
		StatusCode:  resp.StatusCode,
		Headers:     c.extractHeaders(resp),
		ContentType: resp.Header.Get("Content-Type"),
		FileName:    filename,
		Cached:      false,
	}, nil
}

// extractHeaders extracts relevant headers from the HTTP response
func (c *HTTPUpstreamClient) extractHeaders(resp *http.Response) map[string]string {
	headers := make(map[string]string)

	// Extract all headers
	for key, values := range resp.Header {
		if len(values) > 0 {
			headers[key] = values[0]
		}
	}

	return headers
}

// extractFilename attempts to extract filename from Content-Disposition header
func (c *HTTPUpstreamClient) extractFilename(resp *http.Response) string {
	contentDisposition := resp.Header.Get("Content-Disposition")
	if contentDisposition == "" {
		return ""
	}

	// Simple extraction - in production, use a proper parser
	const filenamePrefix = "filename="
	start := len(contentDisposition) - len(filenamePrefix)
	for i := 0; i < start; i++ {
		if contentDisposition[i:i+len(filenamePrefix)] == filenamePrefix {
			filename := contentDisposition[i+len(filenamePrefix):]
			// Remove quotes if present
			if len(filename) > 2 && filename[0] == '"' && filename[len(filename)-1] == '"' {
				filename = filename[1 : len(filename)-1]
			}
			return filename
		}
	}

	return ""
}

// Ensure HTTPUpstreamClient implements the UpstreamClient interface
var _ proxy.UpstreamClient = (*HTTPUpstreamClient)(nil)
