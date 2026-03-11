package main

import (
	"fmt"
	"net/http"

	zh "github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp/config"
)

func main() {
	// Create router with request logger config
	// The router already has a default logger, we just need to configure it
	router := zh.New(config.Config{
		RequestLogger: config.RequestLoggerConfig{
			LogRequestBody:  true,
			LogResponseBody: true,
			MaxBodySize:     1024,
			Fields: []config.LogField{
				config.FieldMethod,
				config.FieldPath,
				config.FieldStatus,
				config.FieldDurationHuman,
				config.FieldRequestBody,
				config.FieldResponseBody,
			},
		},
	})

	// Login endpoint - demonstrates sensitive field masking (password)
	router.POST("/api/login", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.JSON(w, http.StatusOK, zh.M{
			"status":  "success",
			"message": "Login successful",
			"token":   "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
		})
	}))

	// Token endpoint - demonstrates token masking
	router.POST("/api/token", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.JSON(w, http.StatusOK, zh.M{
			"access_token":  "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
			"refresh_token": "dGhpcyBpcyBhIHJlZnJlc2ggdG9rZW4",
			"token_type":    "Bearer",
		})
	}))

	// Users endpoint - demonstrates nested object logging
	router.POST("/api/users", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.JSON(w, http.StatusCreated, zh.M{
			"id":       123,
			"username": "john_doe",
			"email":    "john@example.com",
			"profile": zh.M{
				"bio":      "Software developer",
				"location": "New York",
			},
		})
	}))

	// Payment endpoint - demonstrates custom sensitive field handling
	router.POST("/api/payment", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.JSON(w, http.StatusOK, zh.M{
			"status":      "approved",
			"transaction": "txn_123456",
			"amount":      "49.99",
		})
	}))

	// Health check - no sensitive data
	router.GET("/health", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.JSON(w, http.StatusOK, zh.M{
			"status": "healthy",
		})
	}))

	// Print usage examples
	fmt.Println("Request Logger with Body Logging Example")
	fmt.Println("=========================================")
	fmt.Println()
	fmt.Println("This example demonstrates request/response body logging with")
	fmt.Println("automatic masking of sensitive fields like passwords and tokens.")
	fmt.Println()
	fmt.Println("Try these commands:")
	fmt.Println()
	fmt.Println("1. Login (password field will be masked):")
	fmt.Println("   curl -X POST http://localhost:8080/api/login \\")
	fmt.Println("     -H 'Content-Type: application/json' \\")
	fmt.Println(`     -d '{"username":"john","password":"super_secret_123"}'`)
	fmt.Println()
	fmt.Println("2. Get token (tokens will be masked in response):")
	fmt.Println("   curl -X POST http://localhost:8080/api/token \\")
	fmt.Println("     -H 'Content-Type: application/json' \\")
	fmt.Println(`     -d '{"grant_type":"password"}'`)
	fmt.Println()
	fmt.Println("3. Create user (nested objects):")
	fmt.Println("   curl -X POST http://localhost:8080/api/users \\")
	fmt.Println("     -H 'Content-Type: application/json' \\")
	fmt.Println(`     -d '{"username":"jane","email":"jane@example.com"}'`)
	fmt.Println()
	fmt.Println("4. Health check (no body logging needed):")
	fmt.Println("   curl http://localhost:8080/health")
	fmt.Println()
	fmt.Println("Server starting on :8080")
	fmt.Println()

	if err := router.Start(); err != nil {
		panic(err)
	}
}
