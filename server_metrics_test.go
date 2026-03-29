package zerohttp

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/metrics"
	"github.com/alexferl/zerohttp/zhtest"
)

// TestServer_MetricsDisabledByDefault verifies that metrics server does NOT start
// when Enabled is nil (the default).
func TestServer_MetricsDisabledByDefault(t *testing.T) {
	srv := New(Config{
		// Enabled is nil by default - metrics should be disabled
	})

	// Verify metrics addr is empty when disabled
	addr := srv.MetricsAddr()
	zhtest.AssertEmpty(t, addr)

	// Verify metrics registry is nil
	zhtest.AssertNil(t, srv.Metrics())
}

// TestServer_MetricsServerAddrDefault verifies that ServerAddr defaults to localhost:9090
// when metrics are explicitly enabled but ServerAddr is not explicitly set (nil).
func TestServer_MetricsServerAddrDefault(t *testing.T) {
	// Get available port for main server
	mainListener, err := net.Listen("tcp", "localhost:0")
	zhtest.AssertNoError(t, err)
	mainAddr := mainListener.Addr().String()
	zhtest.AssertNoError(t, mainListener.Close())

	srv := New(Config{
		Addr: mainAddr,
		Metrics: metrics.Config{
			Enabled: config.Bool(true), // Explicitly enable metrics
			// ServerAddr not set (nil) - should default to localhost:9090
		},
	})

	// Verify metrics addr is the default
	addr := srv.MetricsAddr()
	zhtest.AssertEqual(t, "localhost:9090", addr)
}

// TestServer_MetricsExplicitEmptyServerAddr verifies that explicitly setting ServerAddr to empty
// string (via config.String("")) serves metrics on the main server.
func TestServer_MetricsExplicitEmptyServerAddr(t *testing.T) {
	srv := New(Config{
		Metrics: metrics.Config{
			Enabled:    config.Bool(true), // Explicitly enable metrics
			ServerAddr: config.String(""), // Explicitly empty - use main server
		},
	})

	// Register a test route
	srv.GET("/test", HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return R.JSON(w, http.StatusOK, M{"ok": true})
	}))

	// Create test server
	server := httptest.NewServer(srv)
	defer server.Close()

	// Make a request
	resp, err := http.Get(server.URL + "/test")
	zhtest.AssertNoError(t, err)
	_ = resp.Body.Close()

	// Metrics should be on main server
	resp, err = http.Get(server.URL + "/metrics")
	zhtest.AssertNoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	zhtest.AssertEqual(t, 200, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	zhtest.AssertContains(t, string(body), "http_requests_total")
}

// TestServer_MetricsDedicatedServer verifies that a dedicated metrics server starts
// on the configured address when ServerAddr is set.
func TestServer_MetricsDedicatedServer(t *testing.T) {
	// Get available ports for both servers
	mainListener, err := net.Listen("tcp", "localhost:0")
	zhtest.AssertNoError(t, err)
	mainAddr := mainListener.Addr().String()
	zhtest.AssertNoError(t, mainListener.Close())

	metricsListener, err := net.Listen("tcp", "localhost:0")
	zhtest.AssertNoError(t, err)
	metricsAddr := metricsListener.Addr().String()
	zhtest.AssertNoError(t, metricsListener.Close())

	srv := New(Config{
		Addr: mainAddr,
		Metrics: metrics.Config{
			Enabled:    config.Bool(true), // Explicitly enable metrics
			ServerAddr: config.String(metricsAddr),
			Endpoint:   "/metrics",
		},
	})

	// Add a test route
	srv.GET("/test", HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		w.WriteHeader(200)
		_, _ = w.Write([]byte("OK"))
		return nil
	}))

	// Start server in background
	go func() {
		_ = srv.Start()
	}()

	// Wait for server to start
	time.Sleep(50 * time.Millisecond)

	// Make a request to the main server
	resp, err := http.Get("http://" + mainAddr + "/test")
	zhtest.AssertNoError(t, err)
	_ = resp.Body.Close()

	// Request metrics from dedicated server
	resp, err = http.Get("http://" + metricsAddr + "/metrics")
	zhtest.AssertNoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	zhtest.AssertEqual(t, 200, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	m := string(body)

	zhtest.AssertContains(t, m, "http_requests_total")

	// Shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	zhtest.AssertNoError(t, srv.Shutdown(ctx))
}

// TestServer_MetricsAddr_NoServer verifies MetricsAddr returns empty string when no metrics server is configured.
func TestServer_MetricsAddr_NoServer(t *testing.T) {
	srv := New(Config{
		Metrics: metrics.Config{
			Enabled: config.Bool(false),
		},
	})

	addr := srv.MetricsAddr()
	zhtest.AssertEmpty(t, addr)
}

// TestServer_MetricsDedicatedServerNotExposedOnMainServer verifies that when using
// a dedicated metrics server, the /metrics endpoint is NOT available on the main server.
func TestServer_MetricsDedicatedServerNotExposedOnMainServer(t *testing.T) {
	// Get available ports for both servers
	mainListener, err := net.Listen("tcp", "localhost:0")
	zhtest.AssertNoError(t, err)
	mainAddr := mainListener.Addr().String()
	zhtest.AssertNoError(t, mainListener.Close())

	metricsListener, err := net.Listen("tcp", "localhost:0")
	zhtest.AssertNoError(t, err)
	metricsAddr := metricsListener.Addr().String()
	zhtest.AssertNoError(t, metricsListener.Close())

	srv := New(Config{
		Addr: mainAddr,
		Metrics: metrics.Config{
			Enabled:    config.Bool(true), // Explicitly enable metrics
			ServerAddr: config.String(metricsAddr),
			Endpoint:   "/metrics",
		},
	})

	// Start server in background
	go func() {
		_ = srv.Start()
	}()

	// Wait for server to start
	time.Sleep(50 * time.Millisecond)

	// Try to access metrics on main server - should 404
	resp, err := http.Get("http://" + mainAddr + "/metrics")
	zhtest.AssertNoError(t, err)
	_ = resp.Body.Close()

	zhtest.AssertEqual(t, 404, resp.StatusCode)

	// But should work on metrics server
	resp2, err := http.Get("http://" + metricsAddr + "/metrics")
	zhtest.AssertNoError(t, err)
	_ = resp2.Body.Close()

	zhtest.AssertEqual(t, 200, resp2.StatusCode)

	// Shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}

func TestServer_RequestContextCancellation(t *testing.T) {
	// This test verifies that when Shutdown is called, the base context
	// is cancelled and requests can detect it via r.Context().Done()

	ctxCancelled := make(chan bool, 1)
	requestStarted := make(chan bool, 1)

	server := New()

	// Add a handler that waits for context cancellation
	server.GET("/wait", HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		close(requestStarted)

		select {
		case <-r.Context().Done():
			ctxCancelled <- true
			w.WriteHeader(http.StatusServiceUnavailable)
			return nil
		case <-time.After(5 * time.Second):
			ctxCancelled <- false
			w.WriteHeader(http.StatusOK)
			return nil
		}
	}))

	// Start server
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	zhtest.AssertNoError(t, err)
	defer func() { _ = listener.Close() }()

	server.listener = listener
	server.server.Addr = listener.Addr().String()

	// Start server in goroutine using ListenAndServe (not Start)
	go func() {
		_ = server.ListenAndServe()
	}()

	// Wait for server to start
	time.Sleep(50 * time.Millisecond)

	// Make a request in the background
	go func() {
		_, _ = http.Get("http://" + listener.Addr().String() + "/wait")
	}()

	// Wait for request to start
	select {
	case <-requestStarted:
		// Good, request started
	case <-time.After(2 * time.Second):
		zhtest.AssertFail(t, "request did not start in time")
	}

	// Small delay to ensure handler is waiting
	time.Sleep(100 * time.Millisecond)

	// Shutdown the server - this should cancel the base context
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	zhtest.AssertNoError(t, server.Shutdown(shutdownCtx))

	// Wait to see if the handler detected context cancellation
	select {
	case wasCancelled := <-ctxCancelled:
		zhtest.AssertTrue(t, wasCancelled)
	case <-time.After(2 * time.Second):
		zhtest.AssertFail(t, "timeout waiting for handler to detect cancellation")
	}
}

// TestServer_MetricsRecordsErrors verifies that 404 and 405 responses are recorded in metrics
func TestServer_MetricsRecordsErrors(t *testing.T) {
	app := New(Config{
		Metrics: metrics.Config{
			Enabled:    config.Bool(true), // Explicitly enable metrics
			ServerAddr: config.String(""), // Empty to use main server for metrics
		},
	})

	// Register a simple GET endpoint
	app.GET("/api/users", HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return R.JSON(w, http.StatusOK, M{"users": []string{"alice", "bob"}})
	}))

	// Create test server
	server := httptest.NewServer(app)
	defer server.Close()

	// Make requests that will result in different status codes
	// 200 - existing route
	resp, err := http.Get(server.URL + "/api/users")
	zhtest.AssertNoError(t, err)
	_ = resp.Body.Close()
	zhtest.AssertEqual(t, http.StatusOK, resp.StatusCode)

	// 404 - non-existent route
	resp, err = http.Get(server.URL + "/nonexistent")
	zhtest.AssertNoError(t, err)
	_ = resp.Body.Close()
	zhtest.AssertEqual(t, http.StatusNotFound, resp.StatusCode)

	// 405 - method not allowed (POST to GET-only route)
	resp, err = http.Post(server.URL+"/api/users", "application/json", nil)
	zhtest.AssertNoError(t, err)
	_ = resp.Body.Close()
	zhtest.AssertEqual(t, http.StatusMethodNotAllowed, resp.StatusCode)

	// Now check that all status codes are recorded in metrics
	resp, err = http.Get(server.URL + "/metrics")
	zhtest.AssertNoError(t, err)

	body, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	zhtest.AssertNoError(t, err)
	m := string(body)

	// Verify http_requests_total contains all expected status codes
	zhtest.AssertContains(t, m, "http_requests_total")

	// Check for 200 status in metrics
	zhtest.AssertContains(t, m, `status="200"`)

	// Check for 404 status in metrics
	zhtest.AssertContains(t, m, `status="404"`)

	// Check for 405 status in metrics
	zhtest.AssertContains(t, m, `status="405"`)

	// Verify the path is recorded for the existing route
	zhtest.AssertContains(t, m, `path="/api/users"`)

	// Verify duration histogram is also present
	zhtest.AssertContains(t, m, "http_request_duration_seconds")
}

// TestServer_MetricsRecordsPanic verifies that panic responses are recorded as 500 in metrics
// and that the recover middleware records its panic counter
func TestServer_MetricsRecordsPanic(t *testing.T) {
	app := New(Config{
		Metrics: metrics.Config{
			Enabled:    config.Bool(true), // Explicitly enable metrics
			ServerAddr: config.String(""), // Empty to use main server for metrics
		},
	})

	// Register a panic endpoint
	app.GET("/panic", HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		panic("intentional panic for testing")
	}))

	// Create test server
	server := httptest.NewServer(app)
	defer server.Close()

	// Make request that will panic
	resp, err := http.Get(server.URL + "/panic")
	zhtest.AssertNoError(t, err)
	_ = resp.Body.Close()

	// Should get 500 status from recover middleware
	zhtest.AssertEqual(t, http.StatusInternalServerError, resp.StatusCode)

	// Now check metrics
	resp, err = http.Get(server.URL + "/metrics")
	zhtest.AssertNoError(t, err)

	body, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	zhtest.AssertNoError(t, err)
	m := string(body)

	// Verify http_requests_total contains status 500
	zhtest.AssertContains(t, m, `status="500"`)

	// Verify recover_panics_total is recorded
	zhtest.AssertContains(t, m, "recover_panics_total")

	// Verify the path is recorded
	zhtest.AssertContains(t, m, `path="/panic"`)
}
