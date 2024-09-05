package main

import (
	"fmt"
	"github.com/joho/godotenv"
	"os"
	"proxynd/routers"
)

func main() {
	e := godotenv.Load()
	if e != nil {
		fmt.Print(e)
	}

	r := routers.BaseRouter()
	routers.HealthRouter(r)
	routers.ProxyRouter(r)

	port := os.Getenv("port")

	// For run on requested port
	if len(os.Args) > 1 {
		reqPort := os.Args[1]
		if reqPort != "" {
			port = reqPort
		}
	}

	if port == "" {
		port = "8080" //localhost
	}
	type Job interface {
		Run()
	}

	fmt.Println("http://localhost:" + port + "\n")
	r.Run(":" + port)
}
