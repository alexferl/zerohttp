package main

import (
	"embed"
	"io/fs"
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

	// Standard library way (verbose):
	// Create a sub-filesystem from the embedded files
	staticFS, _ := fs.Sub(staticFiles, "static")
	fileServer := http.FileServer(http.FS(staticFS))
	app.GET("/assets/", http.StripPrefix("/assets/", fileServer))

	// Directory serving (standard library way):
	uploadServer := http.FileServer(http.Dir("./uploads"))
	app.GET("/files/", http.StripPrefix("/files/", uploadServer))

	// zerohttp way (concise):
	app.Files("/static/", staticFiles, "static")
	app.FilesDir("/uploads/", "./uploads")

	log.Fatal(app.Start())
}
