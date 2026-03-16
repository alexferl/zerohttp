// Package healthcheck provides [Kubernetes]-style health probe endpoints.
//
// This package implements liveness, readiness, and startup probes
// following the Kubernetes health check conventions.
//
// # Quick Start
//
// Use the default configuration:
//
//	app := zh.New()
//	healthcheck.Register(app)
//
// Endpoints:
//   - GET /livez - Liveness probe (is the process running?)
//   - GET /readyz - Readiness probe (is the app ready to receive traffic?)
//   - GET /startupz - Startup probe (has the app started successfully?)
//
// # Custom Handlers
//
// Provide custom health check logic:
//
//	config := healthcheck.Config{
//	    ReadinessHandler: func(w http.ResponseWriter, r *http.Request) error {
//	        if !db.IsConnected() {
//	            return zh.NewProblemDetail(http.StatusServiceUnavailable, "Database not ready")
//	        }
//	        return zh.Render.Plain(w, http.StatusOK, "ready")
//	    },
//	}
//	healthcheck.Register(app, config)
//
// # Configuration
//
// Customize endpoints and handlers:
//
//	config := healthcheck.Config{
//	    LivenessEndpoint:  "/health",
//	    ReadinessEndpoint: "/ready",
//	    StartupEndpoint:   "/started",
//	}
//	healthcheck.Register(app, config)
//
// [Kubernetes]: https://kubernetes.io/docs/concepts/configuration/liveness-readiness-startup-probes/
package healthcheck
