package main

import (
	"fmt"
	"net/http"

	zh "github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/middleware"
)

func main() {
	router := zh.New()

	// Example 1: Basic host validation - exact match only
	router.GET("/api/basic", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.JSON(w, http.StatusOK, zh.M{
			"status": "ok",
			"host":   r.Host,
		})
	}), middleware.HostValidation(config.HostValidationConfig{
		AllowedHosts: []string{"api.example.com"},
	}))

	// Example 2: Allow subdomains
	router.GET("/api/subdomains", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.JSON(w, http.StatusOK, zh.M{
			"status": "ok",
			"host":   r.Host,
		})
	}), middleware.HostValidation(config.HostValidationConfig{
		AllowedHosts:    []string{"example.com"},
		AllowSubdomains: true, // Allows *.example.com
	}))

	// Example 3: Multiple allowed hosts
	router.GET("/api/multi", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.JSON(w, http.StatusOK, zh.M{
			"status": "ok",
			"host":   r.Host,
		})
	}), middleware.HostValidation(config.HostValidationConfig{
		AllowedHosts: []string{"api.example.com", "app.example.com", "example.com"},
	}))

	// Example 4: Strict port validation
	// When running on non-standard port, require Host header to include it
	router.GET("/api/strict-port", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
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
	router.GET("/health", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
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
	router.GET("/api/custom", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.JSON(w, http.StatusOK, zh.M{
			"status": "ok",
			"host":   r.Host,
		})
	}), middleware.HostValidation(config.HostValidationConfig{
		AllowedHosts: []string{"secure.example.com"},
		StatusCode:   http.StatusForbidden,
		Message:      "Access denied: invalid host",
	}))

	fmt.Println("Host validation examples:")
	fmt.Println()
	fmt.Println("1. Basic (only api.example.com):")
	fmt.Println("   curl -H 'Host: api.example.com' http://localhost:8080/api/basic")
	fmt.Println("   curl -H 'Host: evil.com' http://localhost:8080/api/basic       # rejected")
	fmt.Println()
	fmt.Println("2. Subdomains allowed (any *.example.com):")
	fmt.Println("   curl -H 'Host: api.example.com' http://localhost:8080/api/subdomains")
	fmt.Println("   curl -H 'Host: v1.api.example.com' http://localhost:8080/api/subdomains")
	fmt.Println("   curl -H 'Host: evil.com' http://localhost:8080/api/subdomains  # rejected")
	fmt.Println()
	fmt.Println("3. Multiple allowed hosts:")
	fmt.Println("   curl -H 'Host: api.example.com' http://localhost:8080/api/multi")
	fmt.Println("   curl -H 'Host: app.example.com' http://localhost:8080/api/multi")
	fmt.Println()
	fmt.Println("4. Strict port (requires :8080 in Host):")
	fmt.Println("   curl -H 'Host: localhost:8080' http://localhost:8080/api/strict-port")
	fmt.Println("   curl -H 'Host: localhost' http://localhost:8080/api/strict-port      # rejected")
	fmt.Println()
	fmt.Println("5. Exempt path (no validation on /health):")
	fmt.Println("   curl -H 'Host: anything.com' http://localhost:8080/health")
	fmt.Println()
	fmt.Println("6. Custom error:")
	fmt.Println("   curl -H 'Host: evil.com' http://localhost:8080/api/custom")
	fmt.Println()
	fmt.Println("Server starting on :8080")
	if err := router.Start(); err != nil {
		panic(err)
	}
}
