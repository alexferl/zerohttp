package main

import (
	"log"
	"net/http"

	zh "github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp/config"
)

func main() {
	// Create autocert manager for automatic Let's Encrypt certificates
	manager := zh.NewAutocertManager(
		"/var/cache/certs", // Certificate cache directory
		"example.com",      // Your domain
		"www.example.com",  // Additional domains
	)

	app := zh.New(
		config.WithAddr(":80"),              // HTTP port for ACME challenges
		config.WithTLSAddr(":443"),          // HTTPS port
		config.WithAutocertManager(manager), // Enable auto TLS
	)

	app.GET("/", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.JSON(w, 200, zh.M{
			"message": "Hello, Auto TLS World!",
			"tls":     r.TLS != nil,
			"host":    r.Host,
		})
	}))

	// StartAutoTLS handles both HTTP (for ACME challenges + redirects) and HTTPS
	log.Fatal(app.StartAutoTLS("example.com", "www.example.com"))
}
