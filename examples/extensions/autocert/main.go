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
			Addr: ":80", // HTTP port for ACME challenges
			TLS: config.TLSConfig{
				Addr: ":443", // HTTPS port
			},
			Extensions: config.ExtensionsConfig{
				AutocertManager: manager, // Enable auto TLS
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
