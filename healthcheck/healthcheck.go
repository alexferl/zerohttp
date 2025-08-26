package healthcheck

import (
	"net/http"

	zh "github.com/alexferl/zerohttp"
)

// Option is a function that configures the healthcheck
type Option func(*Config)

// Config holds the healthcheck configuration
type Config struct {
	LivenessEndpoint  string
	LivenessHandler   zh.HandlerFunc
	ReadinessEndpoint string
	ReadinessHandler  zh.HandlerFunc
	StartupEndpoint   string
	StartupHandler    zh.HandlerFunc
}

// defaultHandler returns a simple "ok" response
func defaultHandler(w http.ResponseWriter, r *http.Request) error {
	w.WriteHeader(http.StatusOK)
	_, err := w.Write([]byte("ok"))
	return err
}

// defaultConfig returns the default configuration
func defaultConfig() Config {
	return Config{
		LivenessEndpoint:  "/livez",
		LivenessHandler:   defaultHandler,
		ReadinessEndpoint: "/readyz",
		ReadinessHandler:  defaultHandler,
		StartupEndpoint:   "/startupz",
		StartupHandler:    defaultHandler,
	}
}

// WithLivenessEndpoint sets the liveness endpoint path
func WithLivenessEndpoint(endpoint string) Option {
	return func(c *Config) {
		c.LivenessEndpoint = endpoint
	}
}

// WithLivenessHandler sets the liveness handler
func WithLivenessHandler(handler zh.HandlerFunc) Option {
	return func(c *Config) {
		c.LivenessHandler = handler
	}
}

// WithReadinessEndpoint sets the readiness endpoint path
func WithReadinessEndpoint(endpoint string) Option {
	return func(c *Config) {
		c.ReadinessEndpoint = endpoint
	}
}

// WithReadinessHandler sets the readiness handler
func WithReadinessHandler(handler zh.HandlerFunc) Option {
	return func(c *Config) {
		c.ReadinessHandler = handler
	}
}

// WithStartupEndpoint sets the startup endpoint path
func WithStartupEndpoint(endpoint string) Option {
	return func(c *Config) {
		c.StartupEndpoint = endpoint
	}
}

// WithStartupHandler sets the startup handler
func WithStartupHandler(handler zh.HandlerFunc) Option {
	return func(c *Config) {
		c.StartupHandler = handler
	}
}

// New creates and registers all healthcheck endpoints
func New(app *zh.Server, opts ...Option) {
	cfg := defaultConfig()

	for _, opt := range opts {
		opt(&cfg)
	}

	app.GET(cfg.LivenessEndpoint, cfg.LivenessHandler)
	app.GET(cfg.ReadinessEndpoint, cfg.ReadinessHandler)
	app.GET(cfg.StartupEndpoint, cfg.StartupHandler)
}
