//go:build ignore

package main

import (
	"log"
	"net/http"

	"github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp/config"
	"github.com/quic-go/quic-go/http3"
)

func main() {
	certFile, keyFile := "localhost+2.pem", "localhost+2-key.pem"

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
		if err := app.StartHTTP3(certFile, keyFile); err != nil {
			log.Fatalf("HTTP/3 server error: %v", err)
		}
	}()

	// Start HTTPS server for HTTP/1 and HTTP/2 (blocking)
	log.Println("Starting HTTPS server on https://localhost:8443")
	log.Fatal(app.StartTLS(certFile, keyFile))
}
