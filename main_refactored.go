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
	Version   = "dev"
	BuildTime = "unknown"
	CommitSHA = "unknown"
)

// Example of how to use the refactored main with dependency injection
func mainRefactored() {
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

	// Create application with dependency injection
	application, err := app.NewRefactored(cfg)
	if err != nil {
		log.Fatalf("Failed to create application: %v", err)
	}

	// Example of how to access dependencies from the container
	container := application.GetContainer()
	
	// Get a service from the container
	serviceFactory, err := container.GetServiceFactory()
	if err != nil {
		log.Fatalf("Failed to get service factory: %v", err)
	}
	
	// Log available services
	log.Printf("Service factory initialized: %v", serviceFactory != nil)

	// Run the application
	if err := application.Run(); err != nil {
		log.Fatalf("Application error: %v", err)
	}
}

// This file demonstrates the refactored main with dependency injection
// The actual main.go would be updated similarly