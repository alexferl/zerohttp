// Example: Built-in Compression (gzip/deflate)
//
// This example shows the default compression middleware using only
// the built-in gzip and deflate algorithms from Go's standard library.
// No external dependencies required.
//
// Run: go run main.go
package main

import (
	"log"
	"net/http"
	"time"

	"github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp/middleware"
)

func main() {
	app := zerohttp.New()

	// Use default compression (gzip + deflate, level 6)
	app.Use(middleware.Compress())

	app.GET("/", zerohttp.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		w.Header().Set("Content-Type", "text/html")
		_, err := w.Write([]byte(`<!DOCTYPE html>
<html>
<head><title>Compression Demo</title></head>
<body>
<h1>Hello, Compressed World!</h1>
<p>This response is automatically compressed if the client supports it.</p>
<p>Try: curl -H 'Accept-Encoding: gzip' http://localhost:8080/ | gunzip</p>
</body>
</html>`))
		return err
	}))

	app.GET("/api/data", zerohttp.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zerohttp.R.JSON(w, http.StatusOK, map[string]any{
			"message":   "This JSON response is compressed",
			"timestamp": time.Now().Unix(),
		})
	}))

	app.Logger().Info("Starting server with built-in compression (gzip/deflate)")

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
