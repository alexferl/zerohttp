package main

import (
	"log"
	"net/http"
	"time"

	zh "github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp/httpx"
	"github.com/alexferl/zerohttp/middleware/nocache"
)

func main() {
	app := zh.New()

	// Apply no-cache middleware globally to prevent caching on all routes
	app.Use(nocache.New())

	// This endpoint returns sensitive user data - never cached
	app.GET("/api/user/profile", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.JSON(w, http.StatusOK, map[string]any{
			"id":        "user-123",
			"email":     "user@example.com",
			"ssn_last4": "6789",
			"balance":   1234.56,
			"fetchedAt": time.Now().Format(time.RFC3339Nano),
		})
	}))

	// This endpoint returns session info - never cached
	app.GET("/api/session", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.JSON(w, http.StatusOK, map[string]any{
			"session_id": "sess-abc-123",
			"expires_at": time.Now().Add(30 * time.Minute).Format(time.RFC3339),
			"timestamp":  time.Now().Unix(),
		})
	}))

	// This endpoint serves dynamic content that changes frequently
	app.GET("/api/live/status", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.JSON(w, http.StatusOK, map[string]any{
			"status":    "operational",
			"timestamp": time.Now().Format(time.RFC3339Nano),
			"load":      0.75,
			"memory":    "45%",
		})
	}))

	// Example with custom no-cache configuration
	app.GET("/api/admin/secrets", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.JSON(w, http.StatusOK, map[string]string{
			"message": "Ultra-sensitive admin data",
		})
	}),
		nocache.New(nocache.Config{
			Headers: map[string]string{
				httpx.HeaderCacheControl: "no-cache, no-store, must-revalidate, max-age=0",
				httpx.HeaderPragma:       httpx.CacheControlNoCache,
				httpx.HeaderExpires:      "0",
			},
		}),
	)

	log.Fatal(app.Start())
}
