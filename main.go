package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"

	"proxynd/logging"
	"proxynd/routers"
)

// Build information variables (set by ldflags at release)
var (
	Version   = "dev"
	BuildTime = "unknown"
	CommitSHA = "unknown"
)

func main() {
	// Define flags
	healthCheck := flag.Bool("health", false, "Run health check and exit")
	version := flag.Bool("version", false, "Show version information and exit")
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
		port := os.Getenv("SERVER_PORT")
		if port == "" {
			port = "8080"
		}
		resp, err := http.Get(fmt.Sprintf("http://localhost:%s/healthz", port))
		if err != nil {
			os.Exit(1)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			os.Exit(1)
		}
		os.Exit(0)
	}

	// Normal execution mode
	e := godotenv.Load()
	if e != nil {
		fmt.Print(e)
	}

	// Initialize structured logging system
	if err := logging.SetupLogging(); err != nil {
		log.Fatalf("Failed to setup logging: %v", err)
	}

	// Get logger
	logger := logging.GetLogger()
	logger.Info("Starting ProxyND server")

	app := routers.BaseRouter()
	routers.HealthRouter(app)
	routers.ProxyRouter(app)
	routers.CacheRouter(app)
	routers.ConfigRouter(app)
	routers.StatusRouter(app)
	routers.UserRouter(app)
	routers.TestRouter(app)
	routers.WebhookRouter(app)

	port := os.Getenv("SERVER_PORT")
	if port == "" {
		logger.Fatal("Error: SERVER_PORT environment variable is not set.")
	}

	// For run on requested port
	if len(os.Args) > 1 {
		reqPort := os.Args[1]
		if reqPort != "" {
			port = reqPort
		}
	}

	url := fmt.Sprintf("http://%s:%s", "0.0.0.0", port)
	logger.Info("Server starting", logging.F("url", url), logging.F("port", port))

	logger.Info("Attempting to start server", logging.F("port", port))
	if err := app.Listen(":" + port); err != nil {
		logger.Fatal("Server failed to start", logging.F("error", err.Error()), logging.F("port", port))
	}
}
