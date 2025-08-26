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
	healthcheck.New(app)

	// Or with custom options
	healthcheck.New(app,
		healthcheck.WithLivenessEndpoint("/health/live"),
		healthcheck.WithReadinessEndpoint("/health/ready"),
		healthcheck.WithReadinessHandler(customReadinessHandler),
		healthcheck.WithStartupEndpoint("/health/startup"),
	)

	log.Fatal(app.Start())
}

func isReady() bool {
	// Your readiness logic here
	return true
}
