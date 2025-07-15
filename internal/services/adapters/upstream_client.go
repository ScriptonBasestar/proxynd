package adapters

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"proxynd/internal/services/proxy"
	"proxynd/logging"
)

// HTTPUpstreamClient implements the UpstreamClient interface using HTTP
type HTTPUpstreamClient struct {
	client *http.Client
	logger logging.Logger
}

// NewHTTPUpstreamClient creates a new HTTP upstream client
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

// Fetch retrieves content from upstream
func (c *HTTPUpstreamClient) Fetch(ctx context.Context, url string,
	headers map[string]string) (*proxy.ProxyResponse, error) {
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
		req.Header.Set("User-Agent", "ProxyND/1.0")
	}

	// Log the request
	c.logger.Debug("Fetching from upstream",
		logging.F("url", url),
		logging.F("headers", len(headers)))

	// Execute request
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("upstream request failed: %w", err)
	}

	// Check response status
	if resp.StatusCode >= 400 {
		// Read error body
		body, _ := io.ReadAll(resp.Body)
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
