package main

import (
	"embed"
	"log"
	"net/http"

	zh "github.com/alexferl/zerohttp"
)

//go:embed public
var publicFiles embed.FS

func main() {
	app := zh.New()

	// Static website mode - missing files use custom 404
	app.Static(publicFiles, "public", false)

	app.NotFound(zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.Text(w, http.StatusNotFound, "The page you're looking for has gone on vacation. Try a different path!")
	}))

	log.Fatal(app.Start())
}
