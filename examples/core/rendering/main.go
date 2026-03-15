package main

import (
	"bytes"
	"log"
	"net/http"

	zh "github.com/alexferl/zerohttp"
)

func main() {
	app := zh.New()

	// JSON responses
	app.GET("/json", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.JSON(w, http.StatusOK, zh.M{
			"message": "Hello, JSON!",
			"count":   42,
		})
	}))

	// Text responses
	app.GET("/text", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.Text(w, http.StatusOK, "Hello, Text!")
	}))

	// HTML responses
	app.GET("/html", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		html := "<h1>Hello, HTML!</h1><p>This is a paragraph.</p>"
		return zh.R.HTML(w, http.StatusOK, html)
	}))

	// Binary/blob responses
	app.GET("/blob", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		// Simulate PNG data (just bytes)
		pngData := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
		return zh.R.Blob(w, http.StatusOK, "image/png", pngData)
	}))

	// Streaming responses
	app.GET("/stream", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		reader := bytes.NewReader([]byte("This is streaming data line by line\nSecond line\nThird line"))
		return zh.R.Stream(w, http.StatusOK, "text/plain", reader)
	}))

	// File responses
	app.GET("/file", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		// Serve the main.go file itself as an example
		return zh.R.File(w, r, "main.go")
	}))

	// Problem Detail responses (RFC 9457)
	app.GET("/error", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		problem := zh.NewProblemDetail(http.StatusBadRequest, "Invalid input")
		problem.Set("field", "email")
		problem.Set("reason", "Email format is invalid")
		return zh.R.ProblemDetail(w, problem)
	}))

	log.Fatal(app.Start())
}
