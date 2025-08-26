package main

import (
	"log"
	"net/http"

	zh "github.com/alexferl/zerohttp"
)

func main() {
	app := zh.New()

	app.GET("/", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.JSON(w, 200, zh.M{"message": "Hello, World!"})
	}))

	log.Fatal(app.Start())
}
