package zerohttp

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alexferl/zerohttp/config"
)

// TestServer_MetricsDedicatedServer verifies that a dedicated metrics server starts
// on the configured address when ServerAddr is set.
func TestServer_MetricsDedicatedServer(t *testing.T) {
	// Get available ports for both servers
	mainListener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("failed to create main listener: %v", err)
	}
	mainAddr := mainListener.Addr().String()
	if err := mainListener.Close(); err != nil {
		t.Fatalf("failed to close main listener: %v", err)
	}

	metricsListener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("failed to create metrics listener: %v", err)
	}
	metricsAddr := metricsListener.Addr().String()
	if err := metricsListener.Close(); err != nil {
		t.Fatalf("failed to close metrics listener: %v", err)
	}

	srv := New(config.Config{
		Addr: mainAddr,
		Metrics: config.MetricsConfig{
			Enabled:    true,
			ServerAddr: metricsAddr,
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
		if err := srv.Start(); err != nil {
			t.Errorf("server start error: %v", err)
		}
	}()

	// Wait for server to start
	time.Sleep(50 * time.Millisecond)

	// Make a request to the main server
	resp, err := http.Get("http://" + mainAddr + "/test")
	if err != nil {
		t.Fatalf("failed to request main server: %v", err)
	}
	_ = resp.Body.Close()

	// Request metrics from dedicated server
	resp, err = http.Get("http://" + metricsAddr + "/metrics")
	if err != nil {
		t.Fatalf("failed to request metrics server: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	metrics := string(body)

	if !strings.Contains(metrics, "http_requests_total") {
		t.Error("metrics should contain http_requests_total")
	}

	// Shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		t.Errorf("shutdown error: %v", err)
	}
}

// TestServer_MetricsAddr_NoServer verifies MetricsAddr returns empty string when no metrics server is configured.
func TestServer_MetricsAddr_NoServer(t *testing.T) {
	srv := New(config.Config{
		Metrics: config.MetricsConfig{
			Enabled: false,
		},
	})

	addr := srv.MetricsAddr()
	if addr != "" {
		t.Errorf("expected empty address, got %s", addr)
	}
}

// TestServer_MetricsDedicatedServerNotExposedOnMainServer verifies that when using
// a dedicated metrics server, the /metrics endpoint is NOT available on the main server.
func TestServer_MetricsDedicatedServerNotExposedOnMainServer(t *testing.T) {
	// Get available ports for both servers
	mainListener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("failed to create main listener: %v", err)
	}
	mainAddr := mainListener.Addr().String()
	if err := mainListener.Close(); err != nil {
		t.Fatalf("failed to close main listener: %v", err)
	}

	metricsListener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("failed to create metrics listener: %v", err)
	}
	metricsAddr := metricsListener.Addr().String()
	if err := metricsListener.Close(); err != nil {
		t.Fatalf("failed to close metrics listener: %v", err)
	}

	srv := New(config.Config{
		Addr: mainAddr,
		Metrics: config.MetricsConfig{
			Enabled:    true,
			ServerAddr: metricsAddr,
			Endpoint:   "/metrics",
		},
	})

	// Start server in background
	go func() {
		if err := srv.Start(); err != nil {
			t.Errorf("server start error: %v", err)
		}
	}()

	// Wait for server to start
	time.Sleep(50 * time.Millisecond)

	// Try to access metrics on main server - should 404
	resp, err := http.Get("http://" + mainAddr + "/metrics")
	if err != nil {
		t.Fatalf("failed to request main server: %v", err)
	}
	_ = resp.Body.Close()

	if resp.StatusCode != 404 {
		t.Errorf("expected status 404 on main server, got %d", resp.StatusCode)
	}

	// But should work on metrics server
	resp2, err := http.Get("http://" + metricsAddr + "/metrics")
	if err != nil {
		t.Fatalf("failed to request metrics server: %v", err)
	}
	_ = resp2.Body.Close()

	if resp2.StatusCode != 200 {
		t.Errorf("expected status 200 on metrics server, got %d", resp2.StatusCode)
	}

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
	if err != nil {
		t.Fatalf("failed to create listener: %v", err)
	}
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
		t.Fatal("request did not start in time")
	}

	// Small delay to ensure handler is waiting
	time.Sleep(100 * time.Millisecond)

	// Shutdown the server - this should cancel the base context
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = server.Shutdown(shutdownCtx)
	if err != nil {
		t.Errorf("shutdown error: %v", err)
	}

	// Wait to see if the handler detected context cancellation
	select {
	case wasCancelled := <-ctxCancelled:
		if !wasCancelled {
			t.Error("handler did not receive context cancellation signal")
		}
		// Success - context was cancelled
	case <-time.After(2 * time.Second):
		t.Error("timeout waiting for handler to detect cancellation")
	}
}

// TestServer_MetricsRecordsErrors verifies that 404 and 405 responses are recorded in metrics
func TestServer_MetricsRecordsErrors(t *testing.T) {
	app := New(config.Config{
		Metrics: config.MetricsConfig{
			Enabled: true,
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
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	// 404 - non-existent route
	resp, err = http.Get(server.URL + "/nonexistent")
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}

	// 405 - method not allowed (POST to GET-only route)
	resp, err = http.Post(server.URL+"/api/users", "application/json", nil)
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", resp.StatusCode)
	}

	// Now check that all status codes are recorded in metrics
	resp, err = http.Get(server.URL + "/metrics")
	if err != nil {
		t.Fatalf("failed to get metrics: %v", err)
	}

	body, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		t.Fatalf("failed to read metrics body: %v", err)
	}
	metrics := string(body)

	// Verify http_requests_total contains all expected status codes
	if !strings.Contains(metrics, "http_requests_total") {
		t.Error("metrics should contain http_requests_total")
	}

	// Check for 200 status in metrics
	if !strings.Contains(metrics, `status="200"`) {
		t.Error("metrics should contain status=200")
	}

	// Check for 404 status in metrics
	if !strings.Contains(metrics, `status="404"`) {
		t.Error("metrics should contain status=404")
	}

	// Check for 405 status in metrics
	if !strings.Contains(metrics, `status="405"`) {
		t.Error("metrics should contain status=405")
	}

	// Verify the path is recorded for the existing route
	if !strings.Contains(metrics, `path="/api/users"`) {
		t.Error("metrics should contain path=/api/users")
	}

	// Verify duration histogram is also present
	if !strings.Contains(metrics, "http_request_duration_seconds") {
		t.Error("metrics should contain http_request_duration_seconds")
	}
}

// TestServer_MetricsRecordsPanic verifies that panic responses are recorded as 500 in metrics
// and that the recover middleware records its panic counter
func TestServer_MetricsRecordsPanic(t *testing.T) {
	app := New(config.Config{
		Metrics: config.MetricsConfig{
			Enabled: true,
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
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}
	_ = resp.Body.Close()

	// Should get 500 status from recover middleware
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", resp.StatusCode)
	}

	// Now check metrics
	resp, err = http.Get(server.URL + "/metrics")
	if err != nil {
		t.Fatalf("failed to get metrics: %v", err)
	}

	body, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		t.Fatalf("failed to read metrics body: %v", err)
	}
	metrics := string(body)

	// Verify http_requests_total contains status 500
	if !strings.Contains(metrics, `status="500"`) {
		t.Error("metrics should contain status=500 for panic request")
	}

	// Verify recover_panics_total is recorded
	if !strings.Contains(metrics, "recover_panics_total") {
		t.Error("metrics should contain recover_panics_total from recover middleware")
	}

	// Verify the path is recorded
	if !strings.Contains(metrics, `path="/panic"`) {
		t.Error("metrics should contain path=/panic")
	}
}
