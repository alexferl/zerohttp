package main

import (
	"log"
	"net/http"

	zh "github.com/alexferl/zerohttp"
)

func main() {
	app := zh.New()

	app.GET("/", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.Text(w, http.StatusOK, "Hello, World!")
	}))

	app.NotFound(zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.Text(w, http.StatusNotFound, "The page you're looking for has gone on vacation. Try a different path!")
	}))

	app.MethodNotAllowed(zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.Text(w, http.StatusMethodNotAllowed, "That HTTP method isn't welcome here. Check the allowed methods and try again.")
	}))

	log.Fatal(app.Start())
}
