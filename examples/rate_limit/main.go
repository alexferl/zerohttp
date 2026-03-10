package main

import (
	"fmt"
	"net/http"
	"time"

	zh "github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/middleware"
)

func main() {
	router := zh.New()

	// Example 1: Default rate limiting (token bucket, 100 req/min)
	router.GET("/api/default", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return responseJSON(w, r)
	}), middleware.RateLimit())

	// Example 2: Strict rate limiting (10 req/second, sliding window)
	router.GET("/api/strict", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return responseJSON(w, r)
	}), middleware.RateLimit(config.RateLimitConfig{
		Rate:           10,
		Window:         time.Second,
		Algorithm:      config.SlidingWindow,
		IncludeHeaders: true,
	}))

	// Example 3: Per-user rate limiting (using header)
	router.GET("/api/user", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return responseJSON(w, r)
	}), middleware.RateLimit(config.RateLimitConfig{
		Rate:           5,
		Window:         time.Minute,
		IncludeHeaders: true,
		KeyExtractor: func(r *http.Request) string {
			// Rate limit per user ID header
			userID := r.Header.Get("X-User-ID")
			if userID == "" {
				return "anonymous"
			}
			return userID
		},
	}))

	// Example 4: Custom store (in-memory with higher limit)
	router.GET("/api/high-volume", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return responseJSON(w, r)
	}), middleware.RateLimit(config.RateLimitConfig{
		Rate:           1000,
		Window:         time.Minute,
		MaxKeys:        100000, // Allow more unique keys
		IncludeHeaders: true,
	}))

	// Example 5: Exempt health check from rate limiting
	router.GET("/health", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.JSON(w, http.StatusOK, zh.M{"status": "healthy"})
	}), middleware.RateLimit(config.RateLimitConfig{
		Rate:           10,
		Window:         time.Second,
		ExemptPaths:    []string{"/health", "/metrics"},
		IncludeHeaders: true,
	}))

	fmt.Println("Rate limit examples:")
	fmt.Println("  Default:       curl http://localhost:8080/api/default")
	fmt.Println("  Strict:        curl http://localhost:8080/api/strict")
	fmt.Println("  Per-user:      curl -H 'X-User-ID: user1' http://localhost:8080/api/user")
	fmt.Println("  High volume:   curl http://localhost:8080/api/high-volume")
	fmt.Println("  Health (exempt): curl http://localhost:8080/health")
	fmt.Println()
	fmt.Println("Server starting on :8080")
	if err := router.Start(); err != nil {
		panic(err)
	}
}

func responseJSON(w http.ResponseWriter, _ *http.Request) error {
	// Return rate limit headers if present
	limit := w.Header().Get("X-RateLimit-Limit")
	remaining := w.Header().Get("X-RateLimit-Remaining")
	reset := w.Header().Get("X-RateLimit-Reset")

	return zh.R.JSON(w, http.StatusOK, zh.M{
		"status":    "ok",
		"limit":     limit,
		"remaining": remaining,
		"reset":     reset,
	})
}
