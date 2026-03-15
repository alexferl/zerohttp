package main

import (
	"log"
	"net/http"
	"time"

	zh "github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/middleware"
)

func main() {
	app := zh.New()

	// Simple API proxy with path stripping
	app.Group(func(api zh.Router) {
		rp, cleanup := middleware.ReverseProxy(config.ReverseProxyConfig{
			Target:      "http://localhost:8081",
			StripPrefix: "/api",
		})
		defer cleanup()
		api.Use(rp)
		api.GET("/api/{path...}", nil)
	})

	// Load balancer with health checks
	app.Group(func(lb zh.Router) {
		rp, cleanup := middleware.ReverseProxy(config.ReverseProxyConfig{
			Targets: []config.Backend{
				{Target: "http://localhost:8081", Weight: 1, Healthy: true},
				{Target: "http://localhost:8082", Weight: 1, Healthy: true},
			},
			LoadBalancer:        config.RoundRobin,
			HealthCheckInterval: 10 * time.Second,
			HealthCheckPath:     "/health",
		})
		defer cleanup()
		lb.Use(rp)
		lb.GET("/lb/{path...}", nil)
	})

	// Direct response (not proxied)
	app.GET("/", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.JSON(w, http.StatusOK, zh.M{"message": "Hello from zerohttp!"})
	}))

	log.Fatal(app.Start())
}
