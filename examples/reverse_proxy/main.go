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
	// Create server
	app := zh.New()

	// Example 1: Simple API proxy with path stripping
	// Requests to /api/users go to http://localhost:8081/users
	app.Group(func(api zh.Router) {
		api.Use(middleware.ReverseProxy(config.ReverseProxyConfig{
			Target:      "http://localhost:8081",
			StripPrefix: "/api",
		}))
		api.GET("/api/{path...}", nil)
	})

	// Example 2: Load balancer with health checks
	// Distributes requests across multiple backends
	app.Group(func(lb zh.Router) {
		lb.Use(middleware.ReverseProxy(config.ReverseProxyConfig{
			Targets: []config.Backend{
				{Target: "http://backend1:8081", Weight: 1, Healthy: true},
				{Target: "http://backend2:8081", Weight: 1, Healthy: true},
				{Target: "http://backend3:8081", Weight: 1, Healthy: true},
			},
			LoadBalancer:        config.RoundRobin,
			HealthCheckInterval: 10 * time.Second,
			HealthCheckPath:     "/health",
		}))
		lb.GET("/lb/{path...}", nil)
	})

	// Example 3: With custom headers and X-Forwarded-* injection
	app.Group(func(fwd zh.Router) {
		fwd.Use(middleware.ReverseProxy(config.ReverseProxyConfig{
			Target: "http://api.example.com",
			SetHeaders: map[string]string{
				"X-Proxy-By": "zerohttp",
			},
			RemoveHeaders:  []string{"X-Internal-Token"},
			ForwardHeaders: true, // Adds X-Forwarded-For, X-Forwarded-Proto, X-Forwarded-Host
		}))
		fwd.GET("/forward/{path...}", nil)
	})

	// Example 4: With fallback for when all backends are down
	app.Group(func(crit zh.Router) {
		crit.Use(middleware.ReverseProxy(config.ReverseProxyConfig{
			Targets: []config.Backend{
				{Target: "http://backend1:8081", Healthy: true},
				{Target: "http://backend2:8081", Healthy: true},
			},
			LoadBalancer: config.LeastConnections,
			FallbackHandler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusServiceUnavailable)
				_, _ = w.Write([]byte(`{"error": "Service temporarily unavailable"}`))
			}),
		}))
		crit.GET("/critical/{path...}", nil)
	})

	// Example 5: Path rewriting
	app.Group(func(rw zh.Router) {
		rw.Use(middleware.ReverseProxy(config.ReverseProxyConfig{
			Target:      "http://api.example.com",
			StripPrefix: "/rewrite",
			Rewrites: []config.RewriteRule{
				// Rewrite /rewrite/v1/* to /v2/*
				{Pattern: "/v1/*", Replacement: "/v2/path"},
			},
		}))
		rw.GET("/rewrite/{path...}", nil)
	})

	// Regular routes that don't go through proxy
	app.GET("/", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.JSON(w, http.StatusOK, zh.M{"message": "Hello from zerohttp!"})
	}))

	fmt.Println("Server starting on :8080")
	fmt.Println("Routes:")
	fmt.Println("  GET /              - Direct response")
	fmt.Println("  GET /api/*         -> Proxied to localhost:8081")
	fmt.Println("  GET /lb/*          -> Load balanced across backends")
	fmt.Println("  GET /forward/*     -> Proxy with X-Forwarded headers")
	fmt.Println("  GET /critical/*    -> Proxy with fallback handler")
	fmt.Println("  GET /rewrite/*     -> Proxy with path rewriting")

	if err := app.Start(); err != nil {
		panic(err)
	}
}
