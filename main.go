package main

import (
	"flag"
	"fmt"
	"github.com/joho/godotenv"
	"log"
	"net/http"
	"os"
	"proxynd/logging"
	"proxynd/routers"
)

// 빌드 정보 변수 (릴리스 시 ldflags로 설정됨)
var (
	Version   = "dev"
	BuildTime = "unknown"
	CommitSHA = "unknown"
)

func main() {
	// 플래그 정의
	healthCheck := flag.Bool("health", false, "Run health check and exit")
	version := flag.Bool("version", false, "Show version information and exit")
	flag.Parse()

	// 버전 정보 출력
	if *version {
		fmt.Printf("ProxyND %s\n", Version)
		fmt.Printf("Build Time: %s\n", BuildTime)
		fmt.Printf("Commit SHA: %s\n", CommitSHA)
		os.Exit(0)
	}

	// 헬스체크 모드
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

	// 일반 실행 모드
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
	routers.CacheRouter(app)
	routers.ConfigRouter(app)
	routers.StatusRouter(app)
	routers.UserRouter(app)
	routers.TestRouter(app)

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

	logger.Info("Attempting to start server", logging.F("port", port))
	if err := app.Listen(":" + port); err != nil {
		logger.Fatal("Server failed to start", logging.F("error", err.Error()), logging.F("port", port))
	}
}
