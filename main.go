package main

import (
	"dohoarding/routers"
	"fmt"
	"github.com/joho/godotenv"
	"os"
)

func main() {
	e := godotenv.Load()
	if e != nil {
		fmt.Print(e)
	}

	r := routers.SetupRouter()
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

	r.Run(":" + port)
}
