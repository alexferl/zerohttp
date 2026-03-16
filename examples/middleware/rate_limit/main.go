package main

import (
	"log"
	"net/http"
	"time"

	zh "github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/httpx"
	"github.com/alexferl/zerohttp/middleware"
)

func main() {
	app := zh.New()

	// Example 1: Default rate limiting (token bucket, 100 req/min)
	app.GET("/api/default", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return responseJSON(w, r)
	}), middleware.RateLimit())

	// Example 2: Strict rate limiting (10 req/second, sliding window)
	app.GET("/api/strict", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return responseJSON(w, r)
	}), middleware.RateLimit(config.RateLimitConfig{
		Rate:           10,
		Window:         time.Second,
		Algorithm:      config.SlidingWindow,
		IncludeHeaders: true,
	}))

	// Example 3: Per-user rate limiting (using header)
	app.GET("/api/user", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return responseJSON(w, r)
	}), middleware.RateLimit(config.RateLimitConfig{
		Rate:           5,
		Window:         time.Minute,
		IncludeHeaders: true,
		KeyExtractor: func(r *http.Request) string {
			userID := r.Header.Get("X-User-ID")
			if userID == "" {
				return "anonymous"
			}
			return userID
		},
	}))

	// Example 4: Custom store (in-memory with higher limit)
	app.GET("/api/high-volume", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return responseJSON(w, r)
	}), middleware.RateLimit(config.RateLimitConfig{
		Rate:           1000,
		Window:         time.Minute,
		MaxKeys:        100000,
		IncludeHeaders: true,
	}))

	// Example 5: Exempt health check from rate limiting
	app.GET("/health", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.JSON(w, http.StatusOK, zh.M{"status": "healthy"})
	}), middleware.RateLimit(config.RateLimitConfig{
		Rate:           10,
		Window:         time.Second,
		ExemptPaths:    []string{"/health", "/metrics"},
		IncludeHeaders: true,
	}))

	log.Fatal(app.Start())
}

func responseJSON(w http.ResponseWriter, _ *http.Request) error {
	limit := w.Header().Get(httpx.HeaderXRateLimitLimit)
	remaining := w.Header().Get(httpx.HeaderXRateLimitRemaining)
	reset := w.Header().Get(httpx.HeaderXRateLimitReset)

	return zh.R.JSON(w, http.StatusOK, zh.M{
		"status":    "ok",
		"limit":     limit,
		"remaining": remaining,
		"reset":     reset,
	})
}
