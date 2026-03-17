package healthcheck

import (
	"net/http"

	zh "github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp/internal/config"
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
// Uses DefaultConfig if no config is provided, or merges user config with defaults.
//
// Examples:
//
//	// Use defaults
//	healthcheck.New(app)
//
//	// Override specific fields
//	healthcheck.New(app, healthcheck.Config{
//	    LivenessEndpoint: "/health/live",
//	})
//
//	// Full custom config
//	healthcheck.New(app, healthcheck.Config{
//	    LivenessEndpoint:  "/livez",
//	    LivenessHandler:   myCustomHandler,
//	    ReadinessEndpoint: "/readyz",
//	    ReadinessHandler:  myCustomHandler,
//	    StartupEndpoint:   "/startupz",
//	    StartupHandler:    myCustomHandler,
//	})
func New(app *zh.Server, cfg ...Config) {
	c := DefaultConfig
	if len(cfg) > 0 {
		config.Merge(&c, cfg[0])
	}
	app.GET(c.LivenessEndpoint, c.LivenessHandler)
	app.GET(c.ReadinessEndpoint, c.ReadinessHandler)
	app.GET(c.StartupEndpoint, c.StartupHandler)
}
