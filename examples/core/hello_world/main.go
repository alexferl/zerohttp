package main

import (
	"log"
	"net/http"

	zh "github.com/alexferl/zerohttp"
)

func main() {
	app := zh.New()

	app.GET("/hello/{name}", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		name := zh.Param(r, "name")
		return zh.Render.JSON(w, http.StatusOK, zh.M{"message": "Hello, " + name + "!"})
	}))

	log.Fatal(app.Start())
}
