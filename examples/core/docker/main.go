package main

import (
	"log"
	"net/http"

	zh "github.com/alexferl/zerohttp"
)

func main() {
	app := zh.New(
		zh.Config{
			// Use :8080 (all interfaces) not localhost:8080
			// Required for Docker or the container won't be accessible
			Addr: ":8080",
		},
	)

	app.GET("/", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.JSON(w, 200, zh.M{"message": "Hello, World!"})
	}))

	log.Fatal(app.Start())
}
