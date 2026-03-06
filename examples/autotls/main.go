//go:build ignore

// This example demonstrates how to use AutoTLS with Let's Encrypt.
//
// To run:
//  1. Install golang.org/x/crypto: go get golang.org/x/crypto
//  2. Update the hosts slice with your actual domain(s)
//  3. Ensure ports 80 and 443 are accessible from the internet
//  4. Run: go run main.go
//
// The first time you run, it will obtain certificates from Let's Encrypt.
// Subsequent runs will use cached certificates.
package main

import (
	"log"
	"net/http"

	zh "github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp/config"
	"golang.org/x/crypto/acme/autocert"
)

var hosts = []string{
	"example.com",     // Your domain
	"www.example.com", // Additional domains
}

func main() {
	// Create autocert manager for automatic Let's Encrypt certificates
	// This requires the golang.org/x/crypto/acme/autocert package
	manager := &autocert.Manager{
		Cache:      autocert.DirCache("/var/cache/certs"), // Certificate cache directory
		Prompt:     autocert.AcceptTOS,                    // Accept Let's Encrypt TOS
		HostPolicy: autocert.HostWhitelist(hosts...),      // Allowed hosts
	}

	app := zh.New(
		config.Config{
			Addr:            ":80",   // HTTP port for ACME challenges
			TLSAddr:         ":443",  // HTTPS port
			AutocertManager: manager, // Enable auto TLS
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
