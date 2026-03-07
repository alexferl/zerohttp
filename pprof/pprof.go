package pprof

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	stdpprof "net/http/pprof"

	zh "github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp/log"
)

// PProf holds the pprof configuration and runtime information
type PProf struct {
	// Config is the configuration used
	Config Config
	// Auth contains the actual authentication config used (auto-generated or provided)
	Auth *AuthConfig
}

// Config holds the pprof configuration
type Config struct {
	// Prefix is the base path for all pprof endpoints
	// Default: "/debug/pprof"
	Prefix string

	// EnableIndex enables the index page listing all profiles
	// Default: true
	EnableIndex bool

	// EnableCmdline enables the cmdline endpoint
	// Default: true
	EnableCmdline bool

	// EnableProfile enables the CPU profile endpoint
	// Default: true
	EnableProfile bool

	// EnableSymbol enables the symbol endpoint
	// Default: true
	EnableSymbol bool

	// EnableTrace enables the trace endpoint
	// Default: true
	EnableTrace bool

	// EnableHeap enables the heap profile endpoint
	// Default: true
	EnableHeap bool

	// EnableGoroutine enables the goroutine profile endpoint
	// Default: true
	EnableGoroutine bool

	// EnableThreadCreate enables the threadcreate profile endpoint
	// Default: true
	EnableThreadCreate bool

	// EnableBlock enables the block profile endpoint
	// Default: true
	EnableBlock bool

	// EnableMutex enables the mutex profile endpoint
	// Default: true
	EnableMutex bool

	// Auth is the basic auth configuration.
	// If nil, a random password will be generated.
	// Set to &AuthConfig{} with empty Username/Password to disable auth.
	// Default: nil (auto-generates secure password)
	Auth *AuthConfig
}

// AuthConfig holds basic authentication configuration
type AuthConfig struct {
	// Username for basic auth
	// Default: "pprof"
	Username string
	// Password for basic auth
	// Default: auto-generated secure random password
	Password string
}

// DefaultConfig is the default pprof configuration.
// Modify this to change system-wide defaults.
var DefaultConfig = Config{
	Prefix:             "/debug/pprof",
	EnableIndex:        true,
	EnableCmdline:      true,
	EnableProfile:      true,
	EnableSymbol:       true,
	EnableTrace:        true,
	EnableHeap:         true,
	EnableGoroutine:    true,
	EnableThreadCreate: true,
	EnableBlock:        true,
	EnableMutex:        true,
	Auth:               nil,
}

// generateRandomPassword generates a secure random password
func generateRandomPassword() string {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		// Fallback to a default if crypto/rand fails (extremely unlikely)
		return "pprof-fallback-password-change-me"
	}
	return base64.URLEncoding.EncodeToString(b)
}

// New creates and registers all pprof endpoints with the provided configuration.
// Returns a PProf struct containing the configuration and actual auth credentials used.
//
// Use DefaultConfig for default values:
//
//	pp := pprof.New(app, pprof.DefaultConfig)
//	// Access auto-generated credentials:
//	// pp.Auth.Username, pp.Auth.Password
//
// Or customize specific fields:
//
//	cfg := pprof.DefaultConfig
//	cfg.Prefix = "/admin/pprof"
//	pp := pprof.New(app, cfg)
//
// To disable authentication:
//
//	cfg := pprof.DefaultConfig
//	cfg.Auth = &pprof.AuthConfig{}  // empty = no auth
//	pp := pprof.New(app, cfg)
func New(app *zh.Server, cfg Config) *PProf {
	prefix := cfg.Prefix
	auth := cfg.Auth
	logger := app.Logger()

	if auth == nil {
		password := generateRandomPassword()
		auth = &AuthConfig{
			Username: "pprof",
			Password: password,
		}
	} else if auth.Username == "" && auth.Password == "" {
		auth = nil
		logger.Warn("pprof endpoints enabled without authentication",
			log.F("endpoint", prefix),
		)
	}

	pp := &PProf{
		Config: cfg,
		Auth:   auth,
	}

	wrapFunc := func(fn http.HandlerFunc) zh.HandlerFunc {
		if auth != nil {
			return authHandlerFunc(auth, fn)
		}
		return adaptHandlerFunc(fn)
	}

	wrapHandler := func(h http.Handler) zh.HandlerFunc {
		if auth != nil {
			return authHandler(auth, h)
		}
		return adaptHandler(h)
	}

	if cfg.EnableIndex {
		app.GET(prefix+"/", wrapFunc(stdpprof.Index))
	}
	if cfg.EnableCmdline {
		app.GET(prefix+"/cmdline", wrapFunc(stdpprof.Cmdline))
	}
	if cfg.EnableProfile {
		app.GET(prefix+"/profile", wrapFunc(stdpprof.Profile))
	}
	if cfg.EnableSymbol {
		app.GET(prefix+"/symbol", wrapFunc(stdpprof.Symbol))
		app.POST(prefix+"/symbol", wrapFunc(stdpprof.Symbol))
	}
	if cfg.EnableTrace {
		app.GET(prefix+"/trace", wrapFunc(stdpprof.Trace))
	}
	if cfg.EnableHeap {
		app.GET(prefix+"/heap", wrapHandler(stdpprof.Handler("heap")))
	}
	if cfg.EnableGoroutine {
		app.GET(prefix+"/goroutine", wrapHandler(stdpprof.Handler("goroutine")))
	}
	if cfg.EnableThreadCreate {
		app.GET(prefix+"/threadcreate", wrapHandler(stdpprof.Handler("threadcreate")))
	}
	if cfg.EnableBlock {
		app.GET(prefix+"/block", wrapHandler(stdpprof.Handler("block")))
	}
	if cfg.EnableMutex {
		app.GET(prefix+"/mutex", wrapHandler(stdpprof.Handler("mutex")))
	}

	return pp
}

// adaptHandler adapts a standard http.Handler to zerohttp's HandlerFunc
func adaptHandler(h http.Handler) zh.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		h.ServeHTTP(w, r)
		return nil
	}
}

// adaptHandlerFunc adapts a standard http.HandlerFunc to zerohttp's HandlerFunc
func adaptHandlerFunc(fn http.HandlerFunc) zh.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		fn(w, r)
		return nil
	}
}

// authHandler wraps a handler with basic authentication
func authHandler(auth *AuthConfig, h http.Handler) zh.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		user, pass, ok := r.BasicAuth()
		if !ok || user != auth.Username || pass != auth.Password {
			w.Header().Set("WWW-Authenticate", `Basic realm="pprof"`)
			w.WriteHeader(http.StatusUnauthorized)
			return nil
		}
		h.ServeHTTP(w, r)
		return nil
	}
}

// authHandlerFunc wraps a handler function with basic authentication
func authHandlerFunc(auth *AuthConfig, fn http.HandlerFunc) zh.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		user, pass, ok := r.BasicAuth()
		if !ok || user != auth.Username || pass != auth.Password {
			w.Header().Set("WWW-Authenticate", `Basic realm="pprof"`)
			w.WriteHeader(http.StatusUnauthorized)
			return nil
		}
		fn(w, r)
		return nil
	}
}
