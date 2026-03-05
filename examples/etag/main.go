package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	zh "github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp/middleware"
)

func main() {
	app := zh.New()

	// Apply ETag middleware to all routes
	app.Use(middleware.ETag())

	// Create sample static files
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
		etag := middleware.GenerateFileETag(stat.ModTime().Unix(), stat.Size(), true)

		// Check If-None-Match
		if r.Header.Get("If-None-Match") == etag {
			w.WriteHeader(http.StatusNotModified)
			return nil
		}

		w.Header().Set("ETag", etag)
		w.Header().Set("Content-Type", "text/plain")
		http.ServeContent(w, r, fileName, stat.ModTime(), file)
		return nil
	}))

	fmt.Println("ETag Example Server")
	fmt.Println("===================")
	fmt.Println()
	fmt.Println("Test commands:")
	fmt.Println()
	fmt.Println("1. Basic ETag generation:")
	fmt.Println("   curl -i http://localhost:8080/api/static-data")
	fmt.Println()
	fmt.Println("2. Conditional request (304 Not Modified):")
	fmt.Println("   # First get the ETag from the previous response, then:")
	fmt.Println("   curl -i http://localhost:8080/api/static-data -H 'If-None-Match: \"YOUR_ETAG\"'")
	fmt.Println()
	fmt.Println("3. Static file with ETag:")
	fmt.Println("   curl -i http://localhost:8080/hello.txt")
	fmt.Println()
	fmt.Println("4. File with conditional request:")
	fmt.Println("   curl -i http://localhost:8080/hello.txt -H 'If-None-Match: W/\"YOUR_ETAG\"'")
	fmt.Println()
	fmt.Println("5. Range request with If-Range:")
	fmt.Println("   curl -i http://localhost:8080/large.txt -H 'Range: bytes=0-99' -H 'If-Range: W/\"YOUR_ETAG\"'")
	fmt.Println()
	fmt.Println("Server starting on http://localhost:8080")
	fmt.Println()

	log.Fatal(app.Start())
}

func createSampleFiles() error {
	// Create static directory if it doesn't exist
	if err := os.MkdirAll("./static", 0o755); err != nil {
		return err
	}

	// Create hello.txt
	content := []byte("Hello, World!\nThis is a sample file for ETag testing.\n")
	if err := os.WriteFile("./static/hello.txt", content, 0o644); err != nil {
		return err
	}

	// Create a larger file for range testing
	largeContent := make([]byte, 1000)
	for i := range largeContent {
		largeContent[i] = byte('a' + (i % 26))
	}
	if err := os.WriteFile("./static/large.txt", largeContent, 0o644); err != nil {
		return err
	}

	return nil
}
