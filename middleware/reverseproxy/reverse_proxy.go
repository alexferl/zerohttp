package reverseproxy

import (
	"context"
	"net/http"
	"net/http/httputil"
	"net/url"
	"path"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/httpx"
	"github.com/alexferl/zerohttp/internal/mwutil"
	"github.com/alexferl/zerohttp/metrics"
)

// reverseProxy manages the proxy state including load balancing
type reverseProxy struct {
	cfg        Config
	backends   []*backend
	current    atomic.Uint64 // for round-robin
	transport  http.RoundTripper
	cancelFunc context.CancelFunc // For stopping health checks on shutdown
}

// proxyResponseRecorder wraps http.ResponseWriter to capture status code
type proxyResponseRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (rec *proxyResponseRecorder) WriteHeader(code int) {
	rec.statusCode = code
	rec.ResponseWriter.WriteHeader(code)
}

// Flush implements http.Flusher to support streaming responses like SSE.
func (rec *proxyResponseRecorder) Flush() {
	if f, ok := rec.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// backend represents a single upstream with health tracking
type backend struct {
	Backend
	targetURL   *url.URL
	activeConns atomic.Int64
	healthy     atomic.Int32
	proxy       *httputil.ReverseProxy
}

// New creates a reverse proxy middleware with the provided configuration
// It returns the middleware handler and a cleanup function to stop health checks.
// The cleanup function should be called on server shutdown to prevent goroutine leaks.
func New(cfg Config) (func(http.Handler) http.Handler, func()) {
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
			rp.initBackend(b.Target, weight, config.BoolOrDefault(b.Healthy, true))
		}
	} else {
		panic("reverse proxy: Target or Targets is required")
	}

	if cfg.HealthCheckInterval > 0 {
		ctx, cancel := context.WithCancel(context.Background())
		rp.cancelFunc = cancel
		go rp.healthCheckLoop(ctx)
	}

	cleanup := func() {
		if rp.cancelFunc != nil {
			rp.cancelFunc()
		}
	}

	mwutil.ValidatePathConfig(cfg.ExcludedPaths, cfg.IncludedPaths, "ReverseProxy")

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reg := metrics.SafeRegistry(metrics.GetRegistry(r.Context()))

			if !mwutil.ShouldProcessMiddleware(r.URL.Path, cfg.IncludedPaths, cfg.ExcludedPaths) {
				next.ServeHTTP(w, r)
				return
			}

			b := rp.selectBackend()
			if b == nil {
				if cfg.FallbackHandler != nil {
					cfg.FallbackHandler.ServeHTTP(w, r)
				} else {
					rp.handleError(w, r, http.ErrHandlerTimeout)
				}
				return
			}

			if cfg.LoadBalancer == LeastConnections {
				b.activeConns.Add(1)
				defer b.activeConns.Add(-1)
			}

			rec := &proxyResponseRecorder{ResponseWriter: w, statusCode: http.StatusOK}
			start := time.Now()

			b.proxy.ServeHTTP(rec, r)

			duration := time.Since(start).Seconds()
			reg.Counter("proxy_requests_total", "target", "status").WithLabelValues(b.Target, strconv.Itoa(rec.statusCode)).Inc()
			reg.Histogram("proxy_request_duration_seconds", []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}, "target").WithLabelValues(b.Target).Observe(duration)
		})
	}, cleanup
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
	//lint:ignore SA1019 Setting Director to nil is required before using Rewrite - this is intentional
	proxy.Director = nil
	proxy.Rewrite = func(r *httputil.ProxyRequest) {
		r.SetURL(targetURL)
		if config.BoolOrDefault(rp.cfg.ForwardHeaders, true) {
			r.SetXForwarded()
		}
		// Preserve query parameters from original request
		if r.In.URL.RawQuery != "" {
			r.Out.URL.RawQuery = r.In.URL.RawQuery
		}
		rp.applyModifications(r.Out)
	}

	b := &backend{
		Backend: Backend{
			Target: target,
			Weight: weight,
		},
		targetURL: targetURL,
		proxy:     proxy,
	}
	if healthy {
		b.healthy.Store(1)
	}
	rp.backends = append(rp.backends, b)
}

// selectBackend chooses a backend based on the load balancer algorithm
func (rp *reverseProxy) selectBackend() *backend {
	switch rp.cfg.LoadBalancer {
	case RoundRobin:
		return rp.roundRobin()
	case Random:
		return rp.random()
	case LeastConnections:
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
	next := rp.current.Add(1) - 1
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
		conns := b.activeConns.Load()
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
		if b.healthy.Load() == 1 {
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

	if config.BoolOrDefault(cfg.ForwardHeaders, true) {
		clientIP := r.RemoteAddr
		if xff := r.Header.Get(httpx.HeaderXForwardedFor); xff != "" {
			clientIP = xff + ", " + clientIP
		}
		r.Header.Set(httpx.HeaderXForwardedFor, clientIP)

		if r.TLS != nil {
			r.Header.Set(httpx.HeaderXForwardedProto, "https")
		} else {
			r.Header.Set(httpx.HeaderXForwardedProto, "http")
		}

		// Only set X-Forwarded-Host if it's not already set (e.g., by SetXForwarded)
		if r.Header.Get(httpx.HeaderXForwardedHost) == "" && r.Host != "" {
			r.Header.Set(httpx.HeaderXForwardedHost, r.Host)
		}
	}

	if cfg.ModifyRequest != nil {
		cfg.ModifyRequest(r)
	}
}

// healthCheckLoop periodically checks backend health
func (rp *reverseProxy) healthCheckLoop(ctx context.Context) {
	ticker := time.NewTicker(rp.cfg.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rp.checkHealth(ctx)
		case <-ctx.Done():
			return
		}
	}
}

// checkHealth performs health checks on all backends
func (rp *reverseProxy) checkHealth(ctx context.Context) {
	var wg sync.WaitGroup
	for _, b := range rp.backends {
		wg.Add(1)
		go func(be *backend) {
			defer wg.Done()
			healthy := rp.checkBackendHealth(ctx, be)
			// Don't update health status if context was cancelled (e.g., during shutdown)
			if ctx.Err() != nil {
				return
			}
			if healthy {
				be.healthy.Store(1)
			} else {
				be.healthy.Store(0)
			}
		}(b)
	}
	wg.Wait()
}

// checkBackendHealth checks a single backend
func (rp *reverseProxy) checkBackendHealth(ctx context.Context, b *backend) bool {
	client := &http.Client{
		Timeout:   rp.cfg.HealthCheckTimeout,
		Transport: rp.transport,
	}

	healthURL := b.targetURL.Scheme + "://" + b.targetURL.Host + rp.cfg.HealthCheckPath
	reqCtx, cancel := context.WithTimeout(ctx, rp.cfg.HealthCheckTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, healthURL, nil)
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
