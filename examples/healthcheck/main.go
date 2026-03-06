package main

import (
	"log"
	"net/http"

	zh "github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp/healthcheck"
)

func customReadinessHandler(w http.ResponseWriter, r *http.Request) error {
	// Check database connection, dependencies, etc.
	if !isReady() {
		return zh.R.Text(w, http.StatusInternalServerError, "not ready")
	}

	return zh.R.Text(w, http.StatusOK, "ready")
}

func main() {
	app := zh.New()

	// Basic usage with defaults
	healthcheck.New(app, healthcheck.DefaultConfig)

	// Or with custom configuration
	cfg := healthcheck.DefaultConfig
	cfg.LivenessEndpoint = "/health/live"
	cfg.ReadinessEndpoint = "/health/ready"
	cfg.ReadinessHandler = customReadinessHandler
	cfg.StartupEndpoint = "/health/startup"
	healthcheck.New(app, cfg)

	log.Fatal(app.Start())
}

func isReady() bool {
	// Your readiness logic here
	return true
}
