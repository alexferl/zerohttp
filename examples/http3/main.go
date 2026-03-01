//go:build ignore

// This example demonstrates how to use HTTP/3 with zerohttp.
// It uses github.com/quic-go/quic-go/http3 for HTTP/3 support.
//
// To run this example:
//  1. Install quic-go: go get github.com/quic-go/quic-go
//  2. Generate TLS certificates: openssl req -x509 -newkey rsa:2048 -keyout key.pem -out cert.pem -days 365 -nodes
//  3. Run: go run main.go
//
// To test HTTP/3:
//   - Use a browser that supports HTTP/3 (Chrome, Firefox, Safari)
//   - Or use: curl --http3 -k https://localhost:8443
package main

import (
	"log"
	"net/http"

	"github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp/config"
	"github.com/quic-go/quic-go/http3"
)

func main() {
	// Create zerohttp server with TLS
	app := zerohttp.New(
		config.WithTLSAddr(":8443"),
	)

	// Add Alt-Svc header middleware to advertise HTTP/3
	app.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("Alt-Svc", `h3=":8443"; ma=86400`)
			next.ServeHTTP(w, r)
		})
	})

	// Add routes
	app.GET("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("Hello over HTTP/3!\n"))
	}))

	// Create HTTP/3 server using quic-go
	h3Server := &http3.Server{
		Addr:    ":8443",
		Handler: app,
	}
	app.SetHTTP3Server(h3Server)

	// Start HTTP/3 server in a goroutine
	go func() {
		log.Println("Starting HTTP/3 server on https://localhost:8443")
		if err := app.StartHTTP3("cert.pem", "key.pem"); err != nil {
			log.Fatalf("HTTP/3 server error: %v", err)
		}
	}()

	// Start HTTPS server for HTTP/1 and HTTP/2 (blocking)
	log.Println("Starting HTTPS server on https://localhost:8443")
	log.Fatal(app.StartTLS("cert.pem", "key.pem"))
}
