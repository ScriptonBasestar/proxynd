package main

import (
	"fmt"
	"github.com/joho/godotenv"
	"log"
	"os"
	"proxynd/logging"
	"proxynd/routers"
)

func main() {
	e := godotenv.Load()
	if e != nil {
		fmt.Print(e)
	}

	// 구조화된 로깅 시스템 초기화
	if err := logging.SetupLogging(); err != nil {
		log.Fatalf("Failed to setup logging: %v", err)
	}

	// 로거 가져오기
	logger := logging.GetLogger()
	logger.Info("Starting ProxyND server")

	app := routers.BaseRouter()
	routers.HealthRouter(app)
	routers.ProxyRouter(app)

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

	type Job interface {
		Run()
	}

	url := fmt.Sprintf("http://%s:%s", "0.0.0.0", port)
	logger.Info("Server starting", logging.F("url", url), logging.F("port", port))
	
	if err := app.Listen(":" + port); err != nil {
		logger.Fatal("Server failed to start", logging.F("error", err))
	}
}
