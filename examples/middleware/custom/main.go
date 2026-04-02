package main

import (
	"log"
	"net/http"
	"time"

	zh "github.com/alexferl/zerohttp"
)

// requestTimer is a custom middleware that adds timing information to responses.
// This demonstrates the basic pattern for writing your own middleware.
func requestTimer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Call the next handler
		next.ServeHTTP(w, r)

		// Log after the request is processed
		duration := time.Since(start)
		log.Printf("%s %s took %v", r.Method, r.URL.Path, duration)
	})
}

// addHeader is a factory function that creates middleware to add a custom header.
// This pattern allows configuration of middleware at creation time.
func addHeader(key, value string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set(key, value)
			next.ServeHTTP(w, r)
		})
	}
}

func main() {
	app := zh.New()

	// Apply custom middleware globally
	app.Use(requestTimer)
	app.Use(addHeader("X-Custom-Version", "1.0"))

	// Routes
	app.GET("/", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.JSON(w, http.StatusOK, map[string]string{
			"message": "Hello from custom middleware",
		})
	}))

	app.GET("/hello/{name}", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		name := zh.Param(r, "name")
		return zh.R.JSON(w, http.StatusOK, map[string]string{
			"message": "Hello " + name,
		})
	}))

	log.Fatal(app.Start())
}
