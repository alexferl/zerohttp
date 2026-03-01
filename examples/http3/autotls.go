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
	"log"
	"net/http"

	"github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp/config"
	"github.com/quic-go/quic-go/http3"
	"golang.org/x/crypto/acme/autocert"
)

var (
	_ config.HTTP3Server             = (*http3AutocertServer)(nil)
	_ config.HTTP3ServerWithAutocert = (*http3AutocertServer)(nil)
)

// http3AutocertServer wraps quic-go's http3.Server to implement
// config.HTTP3ServerWithAutocert interface
type http3AutocertServer struct {
	server *http3.Server
}

func (h *http3AutocertServer) ListenAndServeTLS(certFile, keyFile string) error {
	return h.server.ListenAndServeTLS(certFile, keyFile)
}

func (h *http3AutocertServer) Shutdown(ctx context.Context) error {
	return h.server.Shutdown(ctx)
}

func (h *http3AutocertServer) Close() error {
	return nil
}

func (h *http3AutocertServer) ListenAndServeTLSWithAutocert(manager config.AutocertManager) error {
	tlsConfig := &tls.Config{
		GetCertificate: manager.GetCertificate,
		NextProtos:     []string{"h3"},
	}
	h.server.TLSConfig = tlsConfig

	err := h.server.ListenAndServe()
	if err != nil {
		log.Printf("[ERROR] HTTP/3 server failed: %v", err)
	}
	return err
}

func main() {
	// Your domain must be publicly accessible on port 443 for Let's Encrypt
	const domain = "example.com"

	// Create autocert manager for automatic certificates
	manager := &autocert.Manager{
		Cache:      autocert.DirCache("/var/cache/certs"),
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(domain),
	}

	// Create zerohttp server with autocert manager
	app := zerohttp.New(
		config.WithAutocertManager(manager),
	)

	// Add Alt-Svc header to advertise HTTP/3 support
	app.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("Alt-Svc", `h3=":443"; ma=86400`)
			next.ServeHTTP(w, r)
		})
	})

	// Add routes
	app.GET("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("Hello over HTTP/3!\n"))
	}))

	// Create HTTP/3 server with autocert support
	h3Server := &http3AutocertServer{
		server: &http3.Server{
			Addr:    ":443",
			Handler: app,
		},
	}
	app.SetHTTP3Server(h3Server)

	// Start server with AutoTLS (HTTP, HTTPS, and HTTP/3)
	// This starts:
	// - HTTP server on :80 (for ACME challenges and redirects)
	// - HTTPS server on :443 (HTTP/1 and HTTP/2 with AutoTLS)
	// - HTTP/3 server on :443 (if HTTP3Server implements HTTP3ServerWithAutocert)
	log.Fatal(app.StartAutoTLS())
}
