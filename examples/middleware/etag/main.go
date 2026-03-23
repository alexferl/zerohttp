package main

import (
	"log"
	"net/http"
	"os"
	"time"

	zh "github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp/httpx"
	"github.com/alexferl/zerohttp/middleware/etag"
)

func main() {
	app := zh.New()

	app.Use(etag.New())

	if err := createSampleFiles(); err != nil {
		log.Fatal(err)
	}

	// API routes with dynamic content
	app.GET("/api/data", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		data := map[string]any{
			"timestamp": time.Now().Unix(),
			"message":   "This response has an ETag header",
		}
		return zh.R.JSON(w, http.StatusOK, data)
	}))

	// Endpoint that returns consistent content (good for 304 testing)
	app.GET("/api/static-data", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		data := map[string]any{
			"version": "1.0.0",
			"name":    "ETag Demo",
		}
		return zh.R.JSON(w, http.StatusOK, data)
	}))

	// Serve static files with file-based ETags
	// The middleware will handle ETag generation for these too
	app.StaticDir("./static", false)

	// Custom file handler with file-based ETag
	app.GET("/file/{name}", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		// Extract filename from URL path
		// Note: In real code, sanitize the filename to prevent directory traversal
		fileName := r.URL.Path[len("/file/"):]
		filePath := "./static/" + fileName

		file, err := os.Open(filePath)
		if err != nil {
			return zh.R.Text(w, http.StatusNotFound, "File not found")
		}
		defer func() { _ = file.Close() }()

		stat, err := file.Stat()
		if err != nil {
			return zh.R.Text(w, http.StatusInternalServerError, "Error reading file")
		}

		// Generate file-based ETag using modTime and size
		eTag := etag.GenerateFromFile(stat.ModTime().Unix(), stat.Size(), true)

		// Check If-None-Matc
		if r.Header.Get(httpx.HeaderIfNoneMatch) == eTag {
			w.WriteHeader(http.StatusNotModified)
			return nil
		}

		w.Header().Set(httpx.HeaderETag, eTag)
		w.Header().Set(httpx.HeaderContentType, httpx.MIMETextPlain)
		http.ServeContent(w, r, fileName, stat.ModTime(), file)
		return nil
	}))

	log.Fatal(app.Start())
}

func createSampleFiles() error {
	if err := os.MkdirAll("./static", 0o755); err != nil {
		return err
	}

	content := []byte("Hello, World!\nThis is a sample file for ETag testing.\n")
	if err := os.WriteFile("./static/hello.txt", content, 0o644); err != nil {
		return err
	}

	largeContent := make([]byte, 1000)
	for i := range largeContent {
		largeContent[i] = byte('a' + (i % 26))
	}
	if err := os.WriteFile("./static/large.txt", largeContent, 0o644); err != nil {
		return err
	}

	return nil
}
