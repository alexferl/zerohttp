package main

import (
	"log"
	"net/http"

	zh "github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/middleware"
)

func main() {
	// SecurityHeaders middleware is enabled by default with secure defaults:
	// - CSP: default-src 'none'; script-src 'self'; ...
	// - X-Frame-Options: DENY
	// - X-Content-Type-Options: nosniff
	// - Referrer-Policy: no-referrer
	// - Permissions-Policy: accelerometer=(), camera=(), ...
	app := zh.New()

	// This endpoint shows default security headers
	app.GET("/", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.JSON(w, http.StatusOK, map[string]string{
			"message": "Check response headers for security headers",
		})
	}))

	// This endpoint has relaxed CSP (API documentation, etc.)
	app.GET("/api/docs", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.JSON(w, http.StatusOK, map[string]string{
			"message": "Relaxed CSP for API docs",
		})
	}),
		middleware.SecurityHeaders(config.SecurityHeadersConfig{
			ContentSecurityPolicy: "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline';",
		}),
	)

	log.Fatal(app.Start())
}
