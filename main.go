// Package main provides the entry point for the ProxyND server.
// ProxyND is a high-performance package manager proxy/mirror server
// supporting multiple package managers including APT, Maven, NPM, and others.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"proxynd/internal/app"
)

// Build information variables (set by ldflags at release)
var (
	// Version is the semantic version of ProxyND (e.g., "1.2.3")
	Version = "dev"
	// BuildTime is the timestamp when the binary was built
	BuildTime = "unknown"
	// CommitSHA is the git commit SHA at build time
	CommitSHA = "unknown"
)

func main() {
	// Define flags
	healthCheck := flag.Bool("health", false, "Run health check and exit")
	version := flag.Bool("version", false, "Show version information and exit")
	port := flag.String("port", "", "Override server port")
	flag.Parse()

	// Print version information
	if *version {
		fmt.Printf("ProxyND %s\n", Version)
		fmt.Printf("Build Time: %s\n", BuildTime)
		fmt.Printf("Commit SHA: %s\n", CommitSHA)
		os.Exit(0)
	}

	// Health check mode
	if *healthCheck {
		if err := app.RunHealthCheck(); err != nil {
			fmt.Fprintf(os.Stderr, "Health check failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Health check passed")
		os.Exit(0)
	}

	// Create application config
	cfg := &app.Config{
		Version:   Version,
		BuildTime: BuildTime,
		CommitSHA: CommitSHA,
	}

	// Override port if provided via flag or args
	if *port != "" {
		cfg.Port = *port
	} else if len(flag.Args()) > 0 {
		cfg.Port = flag.Args()[0]
	}

	// Create and run application
	application, err := app.New(cfg)
	if err != nil {
		log.Fatalf("Failed to create application: %v", err)
	}

	if err := application.Run(); err != nil {
		log.Fatalf("Application error: %v", err)
	}
}
