package main

import (
	"log"
	"net/http"

	zh "github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/middleware"
)

func main() {
	app := zh.New()

	// Example 1: Basic host validation - exact match only
	app.GET("/api/basic", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.JSON(w, http.StatusOK, zh.M{
			"status": "ok",
			"host":   r.Host,
		})
	}), middleware.HostValidation(config.HostValidationConfig{
		AllowedHosts: []string{"api.example.com"},
	}))

	// Example 2: Allow subdomains
	app.GET("/api/subdomains", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.JSON(w, http.StatusOK, zh.M{
			"status": "ok",
			"host":   r.Host,
		})
	}), middleware.HostValidation(config.HostValidationConfig{
		AllowedHosts:    []string{"example.com"},
		AllowSubdomains: true, // Allows *.example.com
	}))

	// Example 3: Multiple allowed hosts
	app.GET("/api/multi", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.JSON(w, http.StatusOK, zh.M{
			"status": "ok",
			"host":   r.Host,
		})
	}), middleware.HostValidation(config.HostValidationConfig{
		AllowedHosts: []string{"api.example.com", "app.example.com", "example.com"},
	}))

	// Example 4: Strict port validation
	// When running on non-standard port, require Host header to include it
	app.GET("/api/strict-port", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.JSON(w, http.StatusOK, zh.M{
			"status": "ok",
			"host":   r.Host,
		})
	}), middleware.HostValidation(config.HostValidationConfig{
		AllowedHosts: []string{"localhost"},
		StrictPort:   true,
		Port:         8080, // Server port - Host must include :8080
	}))

	// Example 5: Exempt paths (health check bypasses validation)
	app.GET("/health", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.JSON(w, http.StatusOK, zh.M{
			"status": "healthy",
			"host":   r.Host,
			"note":   "This endpoint bypasses host validation",
		})
	}), middleware.HostValidation(config.HostValidationConfig{
		AllowedHosts: []string{"api.example.com"},
		ExemptPaths:  []string{"/health"},
	}))

	// Example 6: Custom error response
	app.GET("/api/custom", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.JSON(w, http.StatusOK, zh.M{
			"status": "ok",
			"host":   r.Host,
		})
	}), middleware.HostValidation(config.HostValidationConfig{
		AllowedHosts: []string{"secure.example.com"},
		StatusCode:   http.StatusForbidden,
		Message:      "Access denied: invalid host",
	}))

	log.Fatal(app.Start())
}
