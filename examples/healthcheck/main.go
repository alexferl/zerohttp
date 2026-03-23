package main

import (
	"log"
	"net/http"

	zh "github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp/healthcheck"
)

func main() {
	app := zh.New()

	// Using custom configuration
	cfg := healthcheck.DefaultConfig
	cfg.LivenessEndpoint = "/health/live"
	cfg.ReadinessEndpoint = "/health/ready"
	cfg.ReadinessHandler = customReadinessHandler
	cfg.StartupEndpoint = "/health/startup"
	healthcheck.New(app, cfg)

	log.Fatal(app.Start())
}

func customReadinessHandler(w http.ResponseWriter, _ *http.Request) error {
	if !isReady() {
		return zh.R.Text(w, http.StatusInternalServerError, "not ready")
	}
	return zh.R.Text(w, http.StatusOK, "ready")
}

func isReady() bool {
	// Your readiness logic here (DB connection, cache, etc.)
	return true
}
