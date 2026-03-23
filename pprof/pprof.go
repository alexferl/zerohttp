package pprof

import (
	"crypto/rand"
	"encoding/base64"
	"net"
	"net/http"
	stdpprof "net/http/pprof"
	"strings"

	zh "github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/httpx"
	zconfig "github.com/alexferl/zerohttp/internal/config"
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
	// nil = use default (true)
	// Default: nil
	EnableIndex *bool

	// EnableCmdline enables the cmdline endpoint
	// nil = use default (true)
	// Default: nil
	EnableCmdline *bool

	// EnableProfile enables the CPU profile endpoint
	// nil = use default (true)
	// Default: nil
	EnableProfile *bool

	// EnableSymbol enables the symbol endpoint
	// nil = use default (true)
	// Default: nil
	EnableSymbol *bool

	// EnableTrace enables the trace endpoint
	// nil = use default (true)
	// Default: nil
	EnableTrace *bool

	// EnableHeap enables the heap profile endpoint
	// nil = use default (true)
	// Default: nil
	EnableHeap *bool

	// EnableGoroutine enables the goroutine profile endpoint
	// nil = use default (true)
	// Default: nil
	EnableGoroutine *bool

	// EnableThreadCreate enables the threadcreate profile endpoint
	// nil = use default (true)
	// Default: nil
	EnableThreadCreate *bool

	// EnableBlock enables the block profile endpoint
	// nil = use default (true)
	// Default: nil
	EnableBlock *bool

	// EnableMutex enables the mutex profile endpoint
	// nil = use default (true)
	// Default: nil
	EnableMutex *bool

	// Auth is the basic auth configuration.
	// If nil, a random password will be generated.
	// Set to &AuthConfig{} with empty Username/Password to disable auth.
	// Default: nil (auto-generates secure password)
	Auth *AuthConfig

	// AllowedIPs restricts access to specific IPs or CIDR ranges.
	// Supports IPv4 and IPv6 addresses and CIDR notation (e.g., "10.0.0.0/8", "192.168.1.100").
	// Default: []string{"127.0.0.1/8", "::1/128"} (localhost only)
	// Set to empty slice to allow any IP (with auth still required).
	AllowedIPs []string
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
	EnableIndex:        config.Bool(true),
	EnableCmdline:      config.Bool(true),
	EnableProfile:      config.Bool(true),
	EnableSymbol:       config.Bool(true),
	EnableTrace:        config.Bool(true),
	EnableHeap:         config.Bool(true),
	EnableGoroutine:    config.Bool(true),
	EnableThreadCreate: config.Bool(true),
	EnableBlock:        config.Bool(true),
	EnableMutex:        config.Bool(true),
	Auth:               nil,
	AllowedIPs:         []string{"127.0.0.1/8", "::1/128"}, // localhost only by default
}

// New creates and registers all pprof endpoints with the provided configuration.
// Uses DefaultConfig if no config is provided, or merges user config with defaults.
// Returns a PProf struct containing the configuration and actual auth credentials used.
//
// See package documentation for usage examples.
func New(app *zh.Server, cfg ...Config) *PProf {
	c := DefaultConfig
	if len(cfg) > 0 {
		zconfig.Merge(&c, cfg[0])
	}

	prefix := c.Prefix
	auth := c.Auth
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

	// Parse allowed IPs (nil means use default localhost-only)
	allowedIPs := c.AllowedIPs
	if allowedIPs == nil {
		allowedIPs = []string{"127.0.0.1/8", "::1/128"}
	}

	var allowedNets []*net.IPNet
	if len(allowedIPs) > 0 {
		var err error
		allowedNets, err = parseAllowedIPs(allowedIPs)
		if err != nil {
			logger.Error("failed to parse allowed IPs, falling back to localhost only",
				log.F("error", err),
			)
			allowedNets, _ = parseAllowedIPs([]string{"127.0.0.1/8", "::1/128"})
		}
	}

	pp := &PProf{
		Config: c,
		Auth:   auth,
	}

	// Build middleware chain: IP check -> Auth -> Handler
	wrapFunc := func(fn http.HandlerFunc) zh.HandlerFunc {
		handler := adaptHandlerFunc(fn)
		if auth != nil {
			handler = authHandlerFunc(auth, fn)
		}
		if len(allowedNets) > 0 {
			handler = ipCheckHandler(handler, allowedNets, logger, prefix)
		}
		return handler
	}

	wrapHandler := func(h http.Handler) zh.HandlerFunc {
		handler := adaptHandler(h)
		if auth != nil {
			handler = authHandler(auth, h)
		}
		if len(allowedNets) > 0 {
			handler = ipCheckHandler(handler, allowedNets, logger, prefix)
		}
		return handler
	}

	if config.BoolOrDefault(c.EnableIndex, true) {
		app.GET(prefix+"/", wrapFunc(stdpprof.Index))
	}
	if config.BoolOrDefault(c.EnableCmdline, true) {
		app.GET(prefix+"/cmdline", wrapFunc(stdpprof.Cmdline))
	}
	if config.BoolOrDefault(c.EnableProfile, true) {
		app.GET(prefix+"/profile", wrapFunc(stdpprof.Profile))
	}
	if config.BoolOrDefault(c.EnableSymbol, true) {
		app.GET(prefix+"/symbol", wrapFunc(stdpprof.Symbol))
		app.POST(prefix+"/symbol", wrapFunc(stdpprof.Symbol))
	}
	if config.BoolOrDefault(c.EnableTrace, true) {
		app.GET(prefix+"/trace", wrapFunc(stdpprof.Trace))
	}
	if config.BoolOrDefault(c.EnableHeap, true) {
		app.GET(prefix+"/heap", wrapHandler(stdpprof.Handler("heap")))
	}
	if config.BoolOrDefault(c.EnableGoroutine, true) {
		app.GET(prefix+"/goroutine", wrapHandler(stdpprof.Handler("goroutine")))
	}
	if config.BoolOrDefault(c.EnableThreadCreate, true) {
		app.GET(prefix+"/threadcreate", wrapHandler(stdpprof.Handler("threadcreate")))
	}
	if config.BoolOrDefault(c.EnableBlock, true) {
		app.GET(prefix+"/block", wrapHandler(stdpprof.Handler("block")))
	}
	if config.BoolOrDefault(c.EnableMutex, true) {
		app.GET(prefix+"/mutex", wrapHandler(stdpprof.Handler("mutex")))
	}

	return pp
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
			w.Header().Set(httpx.HeaderWWWAuthenticate, `Basic realm="pprof"`)
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
			w.Header().Set(httpx.HeaderWWWAuthenticate, `Basic realm="pprof"`)
			w.WriteHeader(http.StatusUnauthorized)
			return nil
		}
		fn(w, r)
		return nil
	}
}

// ipCheckHandler wraps a handler with IP allowlist checking
func ipCheckHandler(next zh.HandlerFunc, allowedNets []*net.IPNet, logger log.Logger, prefix string) zh.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		clientIP := extractClientIP(r, false)

		if !isIPAllowed(clientIP, allowedNets) {
			logger.Warn("pprof access denied: IP not in allowlist",
				log.F("client_ip", clientIP),
				log.F("endpoint", prefix),
			)
			w.WriteHeader(http.StatusForbidden)
			return nil
		}

		return next(w, r)
	}
}

// parseAllowedIPs parses a list of IP addresses or CIDR ranges.
// Returns a slice of *net.IPNet for CIDR matching.
func parseAllowedIPs(ips []string) ([]*net.IPNet, error) {
	nets := make([]*net.IPNet, 0, len(ips))
	for _, ipStr := range ips {
		ipStr = strings.TrimSpace(ipStr)
		if ipStr == "" {
			continue
		}

		// Try parsing as CIDR first
		_, ipNet, err := net.ParseCIDR(ipStr)
		if err == nil {
			nets = append(nets, ipNet)
			continue
		}

		// Try parsing as single IP
		ip := net.ParseIP(ipStr)
		if ip == nil {
			return nil, err
		}

		// Convert single IP to /32 (IPv4) or /128 (IPv6)
		var mask net.IPMask
		if ip.To4() != nil {
			mask = net.CIDRMask(32, 32)
		} else {
			mask = net.CIDRMask(128, 128)
		}
		nets = append(nets, &net.IPNet{IP: ip, Mask: mask})
	}
	return nets, nil
}

// isIPAllowed checks if the given IP is in the allowed list.
func isIPAllowed(clientIP string, allowedNets []*net.IPNet) bool {
	// Parse the client IP
	ip := net.ParseIP(clientIP)
	if ip == nil {
		return false
	}

	// Check if IP matches any allowed network
	for _, ipNet := range allowedNets {
		if ipNet.Contains(ip) {
			return true
		}
	}
	return false
}

// extractClientIP extracts the client IP from the request.
// It handles X-Forwarded-For and X-Real-IP headers when behind a proxy.
func extractClientIP(r *http.Request, trustProxy bool) string {
	if trustProxy {
		// Check X-Forwarded-For header (may contain multiple IPs, use the first)
		if xff := r.Header.Get(httpx.HeaderXForwardedFor); xff != "" {
			ips := strings.Split(xff, ",")
			if len(ips) > 0 {
				return strings.TrimSpace(ips[0])
			}
		}

		// Check X-Real-IP header
		if xri := r.Header.Get(httpx.HeaderXRealIP); xri != "" {
			return xri
		}
	}

	// Fall back to RemoteAddr
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		// If SplitHostPort fails, RemoteAddr might not have a port
		return r.RemoteAddr
	}
	return host
}
