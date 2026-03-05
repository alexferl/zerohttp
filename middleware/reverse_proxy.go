package middleware

import (
	"context"
	"net/http"
	"net/http/httputil"
	"net/url"
	"path"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/alexferl/zerohttp/config"
)

// reverseProxy manages the proxy state including load balancing
type reverseProxy struct {
	cfg       config.ReverseProxyConfig
	backends  []*backend
	current   uint64 // for round-robin
	transport http.RoundTripper
}

// backend represents a single upstream with health tracking
type backend struct {
	config.Backend
	targetURL   *url.URL
	activeConns int64
	healthy     int32 // atomic access
	proxy       *httputil.ReverseProxy
}

// ReverseProxy creates a reverse proxy middleware from the given configuration
func ReverseProxy(cfg config.ReverseProxyConfig) func(http.Handler) http.Handler {
	rp := &reverseProxy{
		cfg:       cfg,
		transport: cfg.Transport,
	}

	if rp.transport == nil {
		rp.transport = http.DefaultTransport
	}

	if cfg.Target != "" {
		// Single target mode
		rp.initBackend(cfg.Target, 1, true)
	} else if len(cfg.Targets) > 0 {
		// Load balancer mode
		for _, b := range cfg.Targets {
			weight := b.Weight
			if weight <= 0 {
				weight = 1
			}
			rp.initBackend(b.Target, weight, b.Healthy)
		}
	} else {
		panic("reverse proxy: Target or Targets is required")
	}

	if cfg.HealthCheckInterval > 0 {
		go rp.healthCheckLoop()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for _, exempt := range cfg.ExemptPaths {
				if pathMatches(r.URL.Path, exempt) {
					next.ServeHTTP(w, r)
					return
				}
			}

			// Get healthy backend
			b := rp.selectBackend()
			if b == nil {
				// No healthy backends - use fallback or error
				if cfg.FallbackHandler != nil {
					cfg.FallbackHandler.ServeHTTP(w, r)
				} else {
					rp.handleError(w, r, http.ErrHandlerTimeout)
				}
				return
			}

			// Track active connections for least-connections LB
			if cfg.LoadBalancer == config.LeastConnections {
				atomic.AddInt64(&b.activeConns, 1)
				defer atomic.AddInt64(&b.activeConns, -1)
			}

			b.proxy.ServeHTTP(w, r)
		})
	}
}

// initBackend initializes a single backend
func (rp *reverseProxy) initBackend(target string, weight int, healthy bool) {
	targetURL, err := url.Parse(target)
	if err != nil {
		panic("reverse proxy: invalid target URL: " + err.Error())
	}

	proxy := httputil.NewSingleHostReverseProxy(targetURL)
	proxy.Transport = rp.transport
	proxy.FlushInterval = rp.cfg.FlushInterval

	if rp.cfg.ErrorHandler != nil {
		proxy.ErrorHandler = rp.cfg.ErrorHandler
	} else {
		proxy.ErrorHandler = rp.defaultErrorHandler
	}

	if rp.cfg.ModifyResponse != nil {
		proxy.ModifyResponse = rp.cfg.ModifyResponse
	}

	// Clear Director before setting Rewrite (only one can be set)
	proxy.Director = nil
	proxy.Rewrite = func(r *httputil.ProxyRequest) {
		r.SetURL(targetURL)
		if rp.cfg.ForwardHeaders {
			r.SetXForwarded()
		}
		// Preserve query parameters from original request
		if r.In.URL.RawQuery != "" {
			r.Out.URL.RawQuery = r.In.URL.RawQuery
		}
		rp.applyModifications(r.Out)
	}

	b := &backend{
		Backend: config.Backend{
			Target: target,
			Weight: weight,
		},
		targetURL: targetURL,
		proxy:     proxy,
	}
	if healthy {
		atomic.StoreInt32(&b.healthy, 1)
	}
	rp.backends = append(rp.backends, b)
}

// selectBackend chooses a backend based on the load balancer algorithm
func (rp *reverseProxy) selectBackend() *backend {
	switch rp.cfg.LoadBalancer {
	case config.RoundRobin:
		return rp.roundRobin()
	case config.Random:
		return rp.random()
	case config.LeastConnections:
		return rp.leastConnections()
	default:
		return rp.roundRobin()
	}
}

// roundRobin selects backends in order
func (rp *reverseProxy) roundRobin() *backend {
	healthy := rp.healthyBackends()
	if len(healthy) == 0 {
		return nil
	}
	next := atomic.AddUint64(&rp.current, 1) - 1
	return healthy[int(next)%len(healthy)]
}

// random selects a random healthy backend
func (rp *reverseProxy) random() *backend {
	healthy := rp.healthyBackends()
	if len(healthy) == 0 {
		return nil
	}
	// Simple pseudo-random: use current timestamp nanos
	idx := time.Now().UnixNano() % int64(len(healthy))
	return healthy[idx]
}

// leastConnections selects backend with fewest active connections
func (rp *reverseProxy) leastConnections() *backend {
	healthy := rp.healthyBackends()
	if len(healthy) == 0 {
		return nil
	}
	var selected *backend
	var minConns int64 = -1
	for _, b := range healthy {
		conns := atomic.LoadInt64(&b.activeConns)
		if minConns == -1 || conns < minConns {
			selected = b
			minConns = conns
		}
	}
	return selected
}

// healthyBackends returns only healthy backends
func (rp *reverseProxy) healthyBackends() []*backend {
	var healthy []*backend
	for _, b := range rp.backends {
		if atomic.LoadInt32(&b.healthy) == 1 {
			healthy = append(healthy, b)
		}
	}
	return healthy
}

// applyModifications applies all configured modifications to the request
func (rp *reverseProxy) applyModifications(r *http.Request) {
	cfg := rp.cfg

	if cfg.StripPrefix != "" {
		r.URL.Path = strings.TrimPrefix(r.URL.Path, cfg.StripPrefix)
		if r.URL.RawPath != "" {
			r.URL.RawPath = strings.TrimPrefix(r.URL.RawPath, cfg.StripPrefix)
		}
		if r.URL.Path == "" {
			r.URL.Path = "/"
		}
	}

	if cfg.AddPrefix != "" {
		r.URL.Path = cfg.AddPrefix + r.URL.Path
		if r.URL.RawPath != "" {
			r.URL.RawPath = cfg.AddPrefix + r.URL.RawPath
		}
	}

	for _, rule := range cfg.Rewrites {
		if matched, _ := path.Match(rule.Pattern, r.URL.Path); matched {
			r.URL.Path = rule.Replacement
			if r.URL.RawPath != "" {
				r.URL.RawPath = rule.Replacement
			}
			break
		}
	}

	for _, header := range cfg.RemoveHeaders {
		r.Header.Del(header)
	}

	for key, value := range cfg.SetHeaders {
		r.Header.Set(key, value)
	}

	if cfg.ForwardHeaders {
		clientIP := r.RemoteAddr
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			clientIP = xff + ", " + clientIP
		}
		r.Header.Set("X-Forwarded-For", clientIP)

		if r.TLS != nil {
			r.Header.Set("X-Forwarded-Proto", "https")
		} else {
			r.Header.Set("X-Forwarded-Proto", "http")
		}

		// Only set X-Forwarded-Host if it's not already set (e.g., by SetXForwarded)
		if r.Header.Get("X-Forwarded-Host") == "" && r.Host != "" {
			r.Header.Set("X-Forwarded-Host", r.Host)
		}
	}

	if cfg.ModifyRequest != nil {
		cfg.ModifyRequest(r)
	}
}

// healthCheckLoop periodically checks backend health
func (rp *reverseProxy) healthCheckLoop() {
	ticker := time.NewTicker(rp.cfg.HealthCheckInterval)
	defer ticker.Stop()

	for range ticker.C {
		rp.checkHealth()
	}
}

// checkHealth performs health checks on all backends
func (rp *reverseProxy) checkHealth() {
	var wg sync.WaitGroup
	for _, b := range rp.backends {
		wg.Add(1)
		go func(be *backend) {
			defer wg.Done()
			healthy := rp.checkBackendHealth(be)
			if healthy {
				atomic.StoreInt32(&be.healthy, 1)
			} else {
				atomic.StoreInt32(&be.healthy, 0)
			}
		}(b)
	}
	wg.Wait()
}

// checkBackendHealth checks a single backend
func (rp *reverseProxy) checkBackendHealth(b *backend) bool {
	client := &http.Client{
		Timeout:   rp.cfg.HealthCheckTimeout,
		Transport: rp.transport,
	}

	healthURL := b.targetURL.Scheme + "://" + b.targetURL.Host + rp.cfg.HealthCheckPath
	ctx, cancel := context.WithTimeout(context.Background(), rp.cfg.HealthCheckTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, healthURL, nil)
	if err != nil {
		return false
	}

	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer func() { _ = resp.Body.Close() }()

	return resp.StatusCode < 500
}

// handleError handles proxy errors
func (rp *reverseProxy) handleError(w http.ResponseWriter, r *http.Request, err error) {
	if rp.cfg.ErrorHandler != nil {
		rp.cfg.ErrorHandler(w, r, err)
	} else {
		rp.defaultErrorHandler(w, r, err)
	}
}

// defaultErrorHandler handles proxy errors
func (rp *reverseProxy) defaultErrorHandler(w http.ResponseWriter, r *http.Request, err error) {
	w.WriteHeader(http.StatusBadGateway)
	_, _ = w.Write([]byte("Bad Gateway"))
}
