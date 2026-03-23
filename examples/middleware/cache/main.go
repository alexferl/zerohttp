package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	zh "github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp/httpx"
	"github.com/alexferl/zerohttp/middleware/cache"
)

func main() {
	app := zh.New()

	// Public API endpoint - cached for 30 seconds
	// Demonstrates automatic ETag and Last-Modified generation
	app.GET("/api/public/data", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		data := map[string]any{
			"timestamp": time.Now().Unix(),
			"message":   "This response is cached with ETag support",
			"hits":      1,
		}
		return zh.R.JSON(w, http.StatusOK, data)
	}),
		cache.New(cache.Config{
			CacheControl: "public, max-age=30",
			DefaultTTL:   30 * time.Second,
			ETag:         true,
			LastModified: true,
		}),
	)

	// User profile - cached privately per user
	// Demonstrates Vary header support (different cache per Accept)
	app.GET("/api/users/{id}", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		userID := r.PathValue("id")
		data := map[string]any{
			"id":        userID,
			"name":      "User " + userID,
			"email":     fmt.Sprintf("user%s@example.com", userID),
			"fetchedAt": time.Now().Format(time.RFC3339),
		}
		return zh.R.JSON(w, http.StatusOK, data)
	}),
		cache.New(cache.Config{
			CacheControl: "private, max-age=60",
			DefaultTTL:   time.Minute,
			ETag:         true,
			Vary:         []string{httpx.HeaderAccept, httpx.HeaderAcceptEncoding},
		}),
	)

	// Live/health endpoint - never cached
	// Demonstrates excluded paths (or you could just not apply the middleware)
	app.GET("/api/live", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.JSON(w, http.StatusOK, map[string]string{
			"status":    "ok",
			"timestamp": time.Now().Format(time.RFC3339Nano),
		})
	}),
	// No cache middleware - this endpoint is never cached
	)

	// Static content - aggressively cached
	// Demonstrates long-term caching with immutable directive
	app.GET("/api/config", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.JSON(w, http.StatusOK, map[string]any{
			"version":     "1.0.0",
			"features":    []string{"cache", "etag", "ratelimit"},
			"maintenance": false,
		})
	}),
		cache.New(cache.Config{
			CacheControl: "public, max-age=3600, immutable",
			DefaultTTL:   time.Hour,
		}),
	)

	// HTML response - demonstrates text/html caching
	app.GET("/page/info", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><title>Cache Demo</title></head>
<body>
<h1>Cached HTML Page</h1>
<p>Generated at: %s</p>
<p>This page is cached for 2 minutes.</p>
</body>
</html>`, time.Now().Format(time.RFC3339))
		return zh.R.HTML(w, http.StatusOK, html)
	}),
		cache.New(cache.Config{
			CacheControl: "public, max-age=120",
			DefaultTTL:   2 * time.Minute,
		}),
	)

	// Stats endpoint - demonstrates custom store usage (in-memory here)
	app.GET("/api/stats", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.JSON(w, http.StatusOK, map[string]any{
			"requests":    12345,
			"cacheHits":   9876,
			"cacheMisses": 2469,
			"updatedAt":   time.Now().Unix(),
		})
	}),
		cache.New(cache.Config{
			CacheControl: "public, max-age=10",
			DefaultTTL:   10 * time.Second,
			MaxEntries:   1000, // Smaller cache for this route
		}),
	)

	log.Fatal(app.Start())
}
