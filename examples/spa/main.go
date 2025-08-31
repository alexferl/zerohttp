package main

import (
	"embed"
	"log"
	"net/http"

	zh "github.com/alexferl/zerohttp"
)

//go:embed dist
var spaFiles embed.FS

func main() {
	app := zh.New()

	// API routes (must be registered before Static)
	app.GET("/api/health", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.JSON(w, 200, zh.M{"status": "healthy"})
	}))

	// Static handler - serves files from embedded dist folder
	app.Static(spaFiles, "dist", true)

	// Or for development with custom API prefix:
	// app.StaticDir("./dist", "/api/v1/", true)

	log.Fatal(app.Start())
}
