package proxy

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"proxynd/internal/logging"
)

// HTTPBestPracticesHandler demonstrates proper HTTP resource management
type HTTPBestPracticesHandler struct {
	client *http.Client
	logger logging.Logger
}

// NewHTTPBestPracticesHandler creates a handler with proper HTTP client configuration
func NewHTTPBestPracticesHandler() *HTTPBestPracticesHandler {
	return &HTTPBestPracticesHandler{
		client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		logger: logging.GetLogger(),
	}
}

// HandleWithProperCleanup demonstrates proper resource cleanup patterns
func (h *HTTPBestPracticesHandler) HandleWithProperCleanup(c *fiber.Ctx) error {
	url := c.Query("url")
	if url == "" {
		return c.Status(fiber.StatusBadRequest).SendString("URL parameter required")
	}

	// Create request with context for timeout control
	ctx, cancel := context.WithTimeout(c.Context(), 15*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		h.logger.Error("Failed to create request", logging.F("error", err))
		return c.Status(fiber.StatusBadRequest).SendString("Invalid URL")
	}

	// Execute request
	resp, err := h.client.Do(req)
	if err != nil {
		h.logger.Error("Request failed", logging.F("error", err))
		return c.Status(fiber.StatusBadGateway).SendString("Upstream request failed")
	}
	// CRITICAL: Always close response body, even on error
	defer func() { _ = resp.Body.Close() }()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		// Read error body for logging (with limit to prevent memory issues)
		errorBody, err := io.ReadAll(io.LimitReader(resp.Body, 1024))
		if err != nil {
			errorBody = []byte("failed to read error body")
		}
		h.logger.Warn("Non-OK response",
			logging.F("status", resp.StatusCode),
			logging.F("body", string(errorBody)))

		return c.Status(resp.StatusCode).SendString("Upstream returned error")
	}

	// Set response headers
	c.Set("Content-Type", resp.Header.Get("Content-Type"))
	c.Set("Content-Length", resp.Header.Get("Content-Length"))

	// Stream response body to client
	return c.SendStream(resp.Body)
}

// HandleMultipleRequests demonstrates cleanup when making multiple HTTP requests
func (h *HTTPBestPracticesHandler) HandleMultipleRequests(c *fiber.Ctx) error {
	urls := c.Query("urls")
	if urls == "" {
		return c.Status(fiber.StatusBadRequest).SendString("URLs parameter required")
	}

	var lastSuccessfulResponse []byte
	var lastError error

	// Try multiple upstream servers
	for _, url := range parseURLs(urls) {
		// Create context for each request
		ctx, cancel := context.WithTimeout(c.Context(), 10*time.Second)

		// Make request
		resp, err := h.makeRequest(ctx, url)
		cancel() // Cancel context after request completes

		if err != nil {
			lastError = err
			h.logger.Warn("Request failed",
				logging.F("url", url),
				logging.F("error", err))
			continue
		}

		// Process successful response
		lastSuccessfulResponse = resp
		lastError = nil
		break
	}

	if lastError != nil {
		return c.Status(fiber.StatusBadGateway).SendString("All upstream requests failed")
	}

	return c.Send(lastSuccessfulResponse)
}

// makeRequest demonstrates proper request handling with cleanup
func (h *HTTPBestPracticesHandler) makeRequest(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := h.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	// Always close response body
	defer func() {
		// Drain and close body to allow connection reuse
		if _, err := io.Copy(io.Discard, resp.Body); err != nil {
			// Log if needed, but don't fail the operation
			_ = err // explicitly ignore error for staticcheck
		}
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("non-OK status: %d", resp.StatusCode)
	}

	// Read response with size limit to prevent memory issues
	body, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024)) // 10MB limit
	if err != nil {
		return nil, fmt.Errorf("failed to read body: %w", err)
	}

	return body, nil
}

// HandleWithRetry demonstrates cleanup with retry logic
func (h *HTTPBestPracticesHandler) HandleWithRetry(c *fiber.Ctx) error {
	url := c.Query("url")
	maxRetries := 3

	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			time.Sleep(time.Duration(attempt*attempt) * time.Second)
		}

		ctx, cancel := context.WithTimeout(c.Context(), 10*time.Second)
		resp, err := h.makeRequestWithCleanup(ctx, url)
		cancel()

		if err == nil {
			return c.Send(resp)
		}

		lastErr = err
		h.logger.Warn("Retry attempt failed",
			logging.F("attempt", attempt+1),
			logging.F("error", err))
	}

	return c.Status(fiber.StatusBadGateway).
		SendString(fmt.Sprintf("Failed after %d attempts: %v", maxRetries, lastErr))
}

// makeRequestWithCleanup shows cleanup pattern for complex scenarios
func (h *HTTPBestPracticesHandler) makeRequestWithCleanup(ctx context.Context, url string) ([]byte, error) {
	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	// Make request
	resp, err := h.client.Do(req)
	if err != nil {
		return nil, err
	}

	// Setup cleanup that always runs
	cleanup := func() {
		if resp != nil && resp.Body != nil {
			// Drain remaining body to allow connection reuse
			if _, err := io.Copy(io.Discard, resp.Body); err != nil {
				// Log if needed, but don't fail the operation
				_ = err // explicitly ignore error for staticcheck
			}
			_ = resp.Body.Close()
		}
	}

	// Ensure cleanup runs no matter what
	defer cleanup()

	// Check status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status: %d", resp.StatusCode)
	}

	// Read body with timeout
	bodyCtx, bodyCancel := context.WithTimeout(ctx, 5*time.Second)
	defer bodyCancel()

	bodyChan := make(chan []byte, 1)
	errChan := make(chan error, 1)

	go func() {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			errChan <- err
			return
		}
		bodyChan <- body
	}()

	select {
	case body := <-bodyChan:
		return body, nil
	case err := <-errChan:
		return nil, err
	case <-bodyCtx.Done():
		return nil, fmt.Errorf("body read timeout")
	}
}

// parseURLs is a helper to parse comma-separated URLs
func parseURLs(urls string) []string {
	// Simple implementation - in production, use proper parsing
	return strings.Split(urls, ",")
}

// Key Best Practices Demonstrated:
// 1. Always use defer resp.Body.Close() immediately after error check
// 2. Use context for timeout control
// 3. Drain response body before closing for connection reuse
// 4. Handle non-OK status codes properly
// 5. Limit response body size to prevent memory issues
// 6. Use proper error handling and logging
// 7. Clean up resources even in error paths
