//go:build ignore

// This example demonstrates HTTP/3 with Let's Encrypt AutoTLS.
//
// To run:
//  1. Install quic-go: go get github.com/quic-go/quic-go
//  2. Update the domain below to your actual domain
//  3. Ensure port 443 is accessible from the internet
//  4. Run: go run autotls.go
//
// The server will obtain certificates from Let's Encrypt automatically.
package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp/config"
	"github.com/quic-go/quic-go/http3"
	"golang.org/x/crypto/acme/autocert"
)

var (
	_ config.HTTP3Server             = (*HTTP3AutocertServer)(nil)
	_ config.HTTP3ServerWithAutocert = (*HTTP3AutocertServer)(nil)
)

// HTTP3AutocertServer wraps quic-go's http3.Server to implement
// config.HTTP3ServerWithAutocert interface
type HTTP3AutocertServer struct {
	Server *http3.Server
}

func (s *HTTP3AutocertServer) ListenAndServeTLS(certFile, keyFile string) error {
	return s.Server.ListenAndServeTLS(certFile, keyFile)
}

func (s *HTTP3AutocertServer) Shutdown(ctx context.Context) error {
	return s.Server.Shutdown(ctx)
}

func (s *HTTP3AutocertServer) Close() error {
	return s.Server.Close()
}

func (s *HTTP3AutocertServer) ListenAndServeTLSWithAutocert(manager config.AutocertManager) error {
	if s.Server.TLSConfig == nil {
		s.Server.TLSConfig = &tls.Config{}
	}
	s.Server.TLSConfig.GetCertificate = manager.GetCertificate
	return s.Server.ListenAndServeTLS("", "")
}

func main() {
	// Your domain must be publicly accessible on port 443 for Let's Encrypt
	const domain = "example.com"

	// Create autocert manager for automatic certificates
	autocertManager := &autocert.Manager{
		Cache:      autocert.DirCache("/var/cache/certs"),
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(domain),
	}

	// Create zerohttp server with autocert manager
	srv := zerohttp.New(
		config.WithAutocertManager(autocertManager),
	)

	// Add routes
	srv.GET("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello over %s!\n", r.Proto)
	}))

	// Create HTTP/3 server with autocert support
	h3Server := &HTTP3AutocertServer{
		Server: &http3.Server{
			Addr:    ":443",
			Handler: srv,
		},
	}

	// Set the HTTP/3 server
	srv.SetHTTP3Server(h3Server)

	// Start server with AutoTLS (HTTP, HTTPS, and HTTP/3)
	go func() {
		fmt.Printf("Starting server with AutoTLS for %s...\n", domain)
		fmt.Println("HTTP/3 will be available automatically!")

		// This starts:
		// - HTTP server on :80 (for ACME challenges and redirects)
		// - HTTPS server on :443 (HTTP/1 and HTTP/2 with AutoTLS)
		// - HTTP/3 server on :443 (if HTTP3Server implements HTTP3ServerWithAutocert)
		if err := srv.StartAutoTLS(); err != nil {
			fmt.Printf("Server error: %v\n", err)
		}
	}()

	// Wait for interrupt
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh

	fmt.Println("\nShutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		fmt.Printf("Shutdown error: %v\n", err)
	}

	fmt.Println("Server stopped")
}
