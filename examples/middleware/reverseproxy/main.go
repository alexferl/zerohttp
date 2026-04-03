package main

import (
	"log"
	"net/http"
	"time"

	zh "github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/middleware/reverseproxy"
)

func main() {
	app := zh.New()

	// Simple API proxy with path stripping
	app.Group(func(api zh.Router) {
		rp, cleanup := reverseproxy.New(reverseproxy.Config{
			Target:      "http://localhost:8081",
			StripPrefix: "/api",
		})
		defer cleanup()
		api.Use(rp)
		api.GET("/api/{path...}", nil)
	})

	// Load balancer with health checks
	app.Group(func(lb zh.Router) {
		rp, cleanup := reverseproxy.New(reverseproxy.Config{
			Targets: []reverseproxy.Backend{
				{Target: "http://localhost:8081", Weight: 1, Healthy: config.Bool(true)},
				{Target: "http://localhost:8082", Weight: 1, Healthy: config.Bool(true)},
			},
			LoadBalancer:        reverseproxy.RoundRobin,
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
