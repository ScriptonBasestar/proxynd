// Package commands provides CLI command implementations for proxyndctl
package commands

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"

	"github.com/spf13/cobra"
)

// MavenStatusResponse represents Maven proxy status
type MavenStatusResponse struct {
	Enabled      bool              `json:"enabled"`
	CacheSize    int64             `json:"cache_size"`
	TotalItems   int               `json:"total_items"`
	Repositories []MavenRepository `json:"repositories"`
}

// MavenRepository represents a Maven repository configuration
type MavenRepository struct {
	Name    string `json:"name"`
	URL     string `json:"url"`
	Enabled bool   `json:"enabled"`
}

// NewMavenCmd creates the Maven command group
func NewMavenCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "maven",
		Short: "Maven package manager commands",
		Long: `Manage Maven proxy operations including repository configuration,
index management, and backup operations.`,
		Example: `  # Check Maven proxy status
  proxyndctl maven status

  # List Maven repositories
  proxyndctl maven repos

  # Build Maven index (future)
  proxyndctl maven index build

  # Create Maven backup (future)
  proxyndctl maven backup create`,
	}

	// Add subcommands
	cmd.AddCommand(newMavenStatusCmd())
	cmd.AddCommand(newMavenReposCmd())

	return cmd
}

// newMavenStatusCmd creates the maven status command
func newMavenStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show Maven proxy status",
		Long:  "Display the current status of the Maven proxy including cache size and repository configuration.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			serverURL, _ := cmd.Flags().GetString("server")
			format, _ := cmd.Flags().GetString("format")

			return runMavenStatus(serverURL, format)
		},
	}
}

// newMavenReposCmd creates the maven repos command
func newMavenReposCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "repos",
		Short: "List Maven repositories",
		Long:  "List all configured Maven repositories and their status.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			serverURL, _ := cmd.Flags().GetString("server")
			format, _ := cmd.Flags().GetString("format")

			return runMavenRepos(serverURL, format)
		},
	}
}

// runMavenStatus fetches and displays Maven proxy status
func runMavenStatus(serverURL, format string) error {
	// Build URL
	u, err := url.Parse(serverURL)
	if err != nil {
		return fmt.Errorf("invalid server URL: %w", err)
	}
	u.Path = "/api/v1/proxies/maven/status"

	// Make request
	resp, err := http.Get(u.String())
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	var status MavenStatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	// Output based on format
	switch format {
	case outputFormatJSON:
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(status)
	case outputFormatYAML:
		// Simple YAML output
		fmt.Printf("enabled: %v\n", status.Enabled)
		fmt.Printf("cache_size: %d\n", status.CacheSize)
		fmt.Printf("total_items: %d\n", status.TotalItems)
		fmt.Println("repositories:")
		for _, repo := range status.Repositories {
			fmt.Printf("  - name: %s\n", repo.Name)
			fmt.Printf("    url: %s\n", repo.URL)
			fmt.Printf("    enabled: %v\n", repo.Enabled)
		}
	default:
		// Table format
		fmt.Println("Maven Proxy Status")
		fmt.Println("==================")
		fmt.Printf("Enabled:      %v\n", status.Enabled)
		fmt.Printf("Cache Size:   %s\n", formatBytes(status.CacheSize))
		fmt.Printf("Total Items:  %d\n", status.TotalItems)
		fmt.Printf("Repositories: %d\n", len(status.Repositories))
	}

	return nil
}

// runMavenRepos lists Maven repositories
func runMavenRepos(serverURL, format string) error {
	// Build URL
	u, err := url.Parse(serverURL)
	if err != nil {
		return fmt.Errorf("invalid server URL: %w", err)
	}
	u.Path = "/api/v1/proxies/maven/status"

	// Make request
	resp, err := http.Get(u.String())
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	var status MavenStatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	// Output repositories based on format
	switch format {
	case outputFormatJSON:
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(status.Repositories)
	case outputFormatYAML:
		fmt.Println("repositories:")
		for _, repo := range status.Repositories {
			fmt.Printf("  - name: %s\n", repo.Name)
			fmt.Printf("    url: %s\n", repo.URL)
			fmt.Printf("    enabled: %v\n", repo.Enabled)
		}
	default:
		// Table format
		fmt.Println("NAME\tURL\tENABLED")
		fmt.Println("────\t───\t───────")
		for _, repo := range status.Repositories {
			fmt.Printf("%s\t%s\t%v\n", repo.Name, repo.URL, repo.Enabled)
		}
	}

	return nil
}
