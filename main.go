package main

import (
	"fmt"
	"github.com/joho/godotenv"
	"log"
	"os"
	"proxynd/routers"
)

func main() {
	e := godotenv.Load()
	if e != nil {
		fmt.Print(e)
	}

	app := routers.BaseRouter()
	routers.HealthRouter(app)
	routers.ProxyRouter(app)

	port := os.Getenv("SERVER_PORT")
	if port == "" {
		log.Fatalln("Error: SERVER_PORT environment variable is not set.")
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
	fmt.Println(url)
	log.Fatal(app.Listen(":" + port))
}
