package main

import (
	"crypto/tls"
	"log"
	"net/http"

	zh "github.com/alexferl/zerohttp"
	"golang.org/x/crypto/acme/autocert"
)

var hosts = []string{
	"example.com",     // Your domain
	"www.example.com", // Additional domains
}

// autocertManagerWrapper wraps golangacme.Manager to implement autocert.Manager
type autocertManagerWrapper struct {
	mgr *autocert.Manager
}

func (a *autocertManagerWrapper) Hostnames() []string {
	return hosts
}

func (a *autocertManagerWrapper) GetCertificate(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	return a.mgr.GetCertificate(hello)
}

func (a *autocertManagerWrapper) HTTPHandler(fallback http.Handler) http.Handler {
	return a.mgr.HTTPHandler(fallback)
}

func main() {
	// Create autocert manager for automatic Let's Encrypt certificates
	// This requires the golang.org/x/crypto/acme/autocert package
	manager := &autocert.Manager{
		Cache:      autocert.DirCache("/var/cache/certs"), // Certificate cache directory
		Prompt:     autocert.AcceptTOS,                    // Accept Let's Encrypt TOS
		HostPolicy: autocert.HostWhitelist(hosts...),      // Allowed hosts
	}

	// Wrap the manager to implement autocert.Manager
	wrappedManager := &autocertManagerWrapper{
		mgr: manager,
	}

	app := zh.New(
		zh.Config{
			Addr: ":80", // HTTP port for ACME challenges
			TLS: zh.TLSConfig{
				Addr: ":443", // HTTPS port
			},
			Extensions: zh.ExtensionsConfig{
				AutocertManager: wrappedManager, // Enable auto TLS
			},
		},
	)

	app.GET("/", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.JSON(w, 200, zh.M{
			"message": "Hello, Auto TLS World!",
			"tls":     r.TLS != nil,
			"host":    r.Host,
		})
	}))

	// StartAutoTLS handles both HTTP (for ACME challenges + redirects) and HTTPS
	log.Fatal(app.StartAutoTLS())
}
