package main

import (
	"embed"
	"log"
	"net/http"

	zh "github.com/alexferl/zerohttp"
)

//go:embed static
var staticFiles embed.FS

func main() {
	app := zh.New()

	// API routes
	app.GET("/api/health", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.JSON(w, 200, zh.M{"status": "healthy"})
	}))

	// Serve embedded files
	app.Files("/static/", staticFiles, "static")

	// Serve directory files
	app.FilesDir("/uploads/", "./upload_dir")

	log.Fatal(app.Start())
}
