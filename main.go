// Package main provides the entry point for the ProxyND server.
// ProxyND is a high-performance package manager proxy/mirror server
// supporting multiple package managers including APT, Maven, NPM, and others.
//
// @title ProxyND API
// @version 1.0
// @description High-performance package manager proxy/mirror server supporting Maven, NPM, APT, Docker Registry, PyPI, YUM, and APK
// @description
// @description ProxyND provides caching, authentication, and enterprise features for package management.
// @description Enterprise features include RBAC, audit logging, analytics, security scanning, and alerting.
//
// @contact.name ProxyND Support
// @contact.url https://github.com/ScriptonBasestar/proxynd/issues
// @contact.email support@proxynd.io
//
// @license.name AGPL-3.0
// @license.url https://www.gnu.org/licenses/agpl-3.0.html
//
// @host localhost:8080
// @BasePath /
//
// @schemes http https
//
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Enter your license token in the format: Bearer {token}
//
// @tag.name Health
// @tag.description Health check and system status endpoints
//
// @tag.name RBAC
// @tag.description Role-Based Access Control - manage roles, permissions, and user assignments
//
// @tag.name Audit
// @tag.description Audit logging and compliance reporting
//
// @tag.name Analytics
// @tag.description Usage statistics, performance metrics, and custom reporting
//
// @tag.name Security
// @tag.description Vulnerability scanning, license compliance, and malware detection
//
// @tag.name Alerts
// @tag.description Alert management and notification rules
//
// @tag.name License
// @tag.description Enterprise license management and feature validation
//
// @tag.name system
// @tag.description System information and status
//
// @tag.name package-managers
// @tag.description Package manager management - list, toggle, and configure package managers
//
// @tag.name configuration
// @tag.description Configuration management - reload and validate configuration
//
// @tag.name cache
// @tag.description Cache management - statistics, TTL policies, and cache operations
//
// @tag.name plugins
// @tag.description Plugin system health monitoring and status
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
