package healthcheck

import (
	"net/http"

	zh "github.com/alexferl/zerohttp"
)

// Config holds the healthcheck configuration
type Config struct {
	// LivenessEndpoint is the path for the liveness probe endpoint.
	// Default: "/livez"
	LivenessEndpoint string

	// LivenessHandler is the handler for the liveness probe.
	// Default: returns "ok" with 200 status
	LivenessHandler zh.HandlerFunc

	// ReadinessEndpoint is the path for the readiness probe endpoint.
	// Default: "/readyz"
	ReadinessEndpoint string

	// ReadinessHandler is the handler for the readiness probe.
	// Default: returns "ok" with 200 status
	ReadinessHandler zh.HandlerFunc

	// StartupEndpoint is the path for the startup probe endpoint.
	// Default: "/startupz"
	StartupEndpoint string

	// StartupHandler is the handler for the startup probe.
	// Default: returns "ok" with 200 status
	StartupHandler zh.HandlerFunc
}

// defaultHandler returns a simple "ok" response
func defaultHandler(w http.ResponseWriter, r *http.Request) error {
	w.WriteHeader(http.StatusOK)
	_, err := w.Write([]byte("ok"))
	return err
}

// DefaultConfig is the default healthcheck configuration.
// Modify this to change system-wide defaults.
var DefaultConfig = Config{
	LivenessEndpoint:  "/livez",
	LivenessHandler:   defaultHandler,
	ReadinessEndpoint: "/readyz",
	ReadinessHandler:  defaultHandler,
	StartupEndpoint:   "/startupz",
	StartupHandler:    defaultHandler,
}

// New creates and registers all healthcheck endpoints with the provided configuration.
// Use DefaultConfig for default values:
//
//	healthcheck.New(app, healthcheck.DefaultConfig)
//
// Or customize specific fields:
//
//	cfg := healthcheck.DefaultConfig
//	cfg.LivenessEndpoint = "/health/live"
//	cfg.ReadinessHandler = myCustomHandler
//	healthcheck.New(app, cfg)
func New(app *zh.Server, cfg Config) {
	app.GET(cfg.LivenessEndpoint, cfg.LivenessHandler)
	app.GET(cfg.ReadinessEndpoint, cfg.ReadinessHandler)
	app.GET(cfg.StartupEndpoint, cfg.StartupHandler)
}
