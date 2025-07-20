package app

import (
	"fmt"
	"net/http"
	"os"
	"time"
)

// RunHealthCheck performs a health check against the running server
func RunHealthCheck() error {
	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8080"
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	// Perform health check
	url := fmt.Sprintf("http://localhost:%s/healthz", port)
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check returned status %d", resp.StatusCode)
	}

	return nil
}
