package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	zh "github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/middleware"
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
		middleware.Cache(config.CacheConfig{
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
		middleware.Cache(config.CacheConfig{
			CacheControl: "private, max-age=60",
			DefaultTTL:   time.Minute,
			ETag:         true,
			Vary:         []string{"Accept", "Accept-Encoding"},
		}),
	)

	// Live/health endpoint - never cached
	// Demonstrates exempt paths (or you could just not apply the middleware)
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
		middleware.Cache(config.CacheConfig{
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
		middleware.Cache(config.CacheConfig{
			CacheControl: "public, max-age=120",
			DefaultTTL:   2 * time.Minute,
		}),
	)

	// Stats endpoint - demonstrates custom store usage (in-memory here)
	// This also shows how to exempt specific paths from caching
	app.GET("/api/stats", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.JSON(w, http.StatusOK, map[string]any{
			"requests":    12345,
			"cacheHits":   9876,
			"cacheMisses": 2469,
			"updatedAt":   time.Now().Unix(),
		})
	}),
		middleware.Cache(config.CacheConfig{
			CacheControl: "public, max-age=10",
			DefaultTTL:   10 * time.Second,
			MaxEntries:   1000, // Smaller cache for this route
		}),
	)

	fmt.Println("HTTP Cache Example Server")
	fmt.Println("=========================")
	fmt.Println()
	fmt.Println("Test commands:")
	fmt.Println()
	fmt.Println("1. Basic cached request (30s TTL):")
	fmt.Println("   curl -i http://localhost:8080/api/public/data")
	fmt.Println()
	fmt.Println("2. Conditional request with ETag (returns 304 if not modified):")
	fmt.Println("   curl -i http://localhost:8080/api/public/data -H 'If-None-Match: \"<etag-from-above>\"'")
	fmt.Println()
	fmt.Println("3. User profile (private cache):")
	fmt.Println("   curl -i http://localhost:8080/api/users/123")
	fmt.Println()
	fmt.Println("4. Live endpoint (never cached):")
	fmt.Println("   curl -i http://localhost:8080/api/live")
	fmt.Println("   curl -i http://localhost:8080/api/live")
	fmt.Println("   (Notice the timestamps are always different)")
	fmt.Println()
	fmt.Println("5. Static config (1h TTL):")
	fmt.Println("   curl -i http://localhost:8080/api/config")
	fmt.Println()
	fmt.Println("6. HTML page (2m TTL):")
	fmt.Println("   curl -i http://localhost:8080/page/info")
	fmt.Println()
	fmt.Println("7. Stats endpoint (10s TTL):")
	fmt.Println("   curl -i http://localhost:8080/api/stats")
	fmt.Println()
	fmt.Println("Server starting on http://localhost:8080")
	fmt.Println()

	log.Fatal(app.Start())
}
