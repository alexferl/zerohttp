// Package pprof provides Go runtime profiling endpoints.
//
// This package registers standard Go pprof handlers at configurable paths
// with auto-generated authentication for security.
//
// # Quick Start
//
// Use the default configuration:
//
//	app := zh.New()
//	pp := pprof.New(app)
//	log.Printf("pprof credentials: %s / %s", pp.Auth.Username, pp.Auth.Password)
//
// Access profiles at http://localhost:8080/debug/pprof/
//
// # Configuration
//
// Customize endpoints or provide explicit credentials:
//
//	// Override specific fields (uses defaults for rest)
//	pprof.New(app, pprof.Config{
//	    Prefix: "/admin/pprof",
//	})
//
//	// Full custom config
//	pprof.New(app, pprof.Config{
//	    Prefix: "/debug/pprof",
//	    Auth: &pprof.AuthConfig{
//	        Username: "admin",
//	        Password: "secret",
//	    },
//	})
//
// If Auth is nil, a secure password is auto-generated and available via pp.Auth.
// Set Auth to &AuthConfig{} with empty Username/Password to disable auth.
//
// # Available Endpoints
//
// The following profiles are available (when enabled):
//   - /debug/pprof/ - Index page listing all profiles
//   - /debug/pprof/cmdline - Command line arguments
//   - /debug/pprof/profile - CPU profile (30 seconds by default)
//   - /debug/pprof/symbol - Symbol lookup
//   - /debug/pprof/trace - Execution trace
//   - /debug/pprof/heap - Heap profile
//   - /debug/pprof/goroutine - Goroutine profile
//   - /debug/pprof/threadcreate - Thread creation profile
//   - /debug/pprof/block - Block profile
//   - /debug/pprof/mutex - Mutex profile
//   - /debug/pprof/allocs - Allocs profile
package pprof
