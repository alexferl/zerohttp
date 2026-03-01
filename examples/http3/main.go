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
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp/config"
	"github.com/quic-go/quic-go/http3"
)

func main() {
	// Create zerohttp server first to get the router
	srv := zerohttp.New(
		config.WithTLSAddr(":8443"),
	)

	// Add routes to zerohttp router
	srv.GET("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello! Protocol: %s\n", r.Proto)
	}))

	// Create HTTP/3 server using quic-go
	// Use the zerohttp server's router (which implements http.Handler)
	h3Server := &http3.Server{
		Addr:    ":8443",
		Handler: srv,
	}

	// Set the HTTP/3 server on the zerohttp server
	srv.SetHTTP3Server(h3Server)

	// Start HTTP/3 server in a goroutine
	go func() {
		fmt.Println("Starting HTTP/3 server on https://localhost:8443")
		if err := srv.StartHTTP3("cert.pem", "key.pem"); err != nil {
			fmt.Printf("HTTP/3 server error: %v\n", err)
		}
	}()

	// Start HTTPS server for HTTP/1 and HTTP/2
	go func() {
		fmt.Println("Starting HTTPS server on https://localhost:8443")
		if err := srv.StartTLS("cert.pem", "key.pem"); err != nil {
			fmt.Printf("HTTPS server error: %v\n", err)
		}
	}()

	// Wait for interrupt signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh

	fmt.Println("\nShutting down...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		fmt.Printf("Shutdown error: %v\n", err)
	}

	fmt.Println("Server stopped")
}
