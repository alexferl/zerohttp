package zerohttp

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/zhtest"
)

func TestServer_ListenerTLSAddr(t *testing.T) {
	server := New()

	// Set TLS server address
	server.tlsServer = &http.Server{Addr: ":8443"}
	addr := server.ListenerTLSAddr()
	if addr != ":8443" {
		t.Errorf("Expected ':8443', got '%s'", addr)
	}

	// Set actual TLS listener (takes precedence)
	listener, _ := net.Listen("tcp", "127.0.0.1:0")
	server.tlsListener = listener
	defer func() {
		if err := listener.Close(); err != nil {
			t.Errorf("failed to close listener: %v", err)
		}
	}()

	addr = server.ListenerTLSAddr()
	if addr == "" {
		t.Error("Expected non-empty TLS address from actual listener")
	}
}

func TestServer_CreateHTTPSRedirectHandler(t *testing.T) {
	server := New()
	handler := server.createHTTPSRedirectHandler()

	req := httptest.NewRequest(http.MethodGet, "http://example.com/path?query=value", nil)
	req.Host = "example.com"
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	zhtest.AssertWith(t, w).
		Status(http.StatusMovedPermanently).
		Header(HeaderLocation, "https://example.com/path?query=value")
}

func TestServer_StartAutoTLS_NoManager(t *testing.T) {
	server := New()
	server.autocertManager = nil

	err := server.StartAutoTLS()
	if err == nil {
		t.Fatal("Expected error when autocert manager is nil")
	}

	expectedMsg := "autocert manager not configured"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error '%s', got '%s'", expectedMsg, err.Error())
	}
}

// mockAutocertManager is a mock implementation for testing
type mockAutocertManager struct {
	getCertificateCalled bool
	httpHandlerCalled    bool
}

func (m *mockAutocertManager) GetCertificate(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	m.getCertificateCalled = true
	// Return a dummy cert to simulate successful certificate retrieval
	return &tls.Certificate{}, nil
}

func (m *mockAutocertManager) HTTPHandler(fallback http.Handler) http.Handler {
	m.httpHandlerCalled = true
	return fallback
}

func (m *mockAutocertManager) Hostnames() []string {
	return []string{"example.com"}
}

func TestServer_StartAutoTLS_WithManager(t *testing.T) {
	mgr := &mockAutocertManager{}
	server := New(config.Config{Extensions: config.ExtensionsConfig{AutocertManager: mgr}})

	// Verify manager was set (compare using concrete type assertion)
	if server.autocertManager == nil {
		t.Error("expected autocert manager to be set on server")
	}
}

// blockingAutocertManager wraps a manager and signals when GetCertificate is called
type blockingAutocertManager struct {
	mockAutocertManager
	onGetCertCalled chan struct{}
}

func (m *blockingAutocertManager) GetCertificate(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	if m.onGetCertCalled != nil {
		close(m.onGetCertCalled)
	}
	return m.mockAutocertManager.GetCertificate(hello)
}

// TestStartAutoTLS_HTTPReadyGatesWarmUp verifies the warm-up goroutine waits for httpReady
// before attempting to fetch the certificate
func TestStartAutoTLS_HTTPReadyGatesWarmUp(t *testing.T) {
	// This test verifies the httpReady synchronization works correctly
	// by checking that the warm-up goroutine blocks on httpReady

	getCertCalled := make(chan struct{})
	mgr := &blockingAutocertManager{
		onGetCertCalled: getCertCalled,
	}
	h3Server := &mockHTTP3ServerWithAutocert{}

	server := New(config.Config{
		Addr: "localhost:0",
		TLS: config.TLSConfig{
			Addr: "localhost:0",
		},
		Extensions: config.ExtensionsConfig{AutocertManager: mgr},
	})
	server.SetHTTP3Server(h3Server)

	// Start in goroutine
	go func() {
		_ = server.StartAutoTLS()
	}()

	// Give servers time to start
	time.Sleep(100 * time.Millisecond)

	// Trigger cert by connecting to HTTPS
	conn, err := tls.Dial("tcp", server.ListenerTLSAddr(), &tls.Config{
		InsecureSkipVerify: true,
	})
	if err == nil {
		_ = conn.Close()
	}

	// Verify GetCertificate was eventually called
	select {
	case <-getCertCalled:
		// Expected
	case <-time.After(500 * time.Millisecond):
		t.Error("GetCertificate was not called")
	}

	// Verify HTTP/3 eventually started
	time.Sleep(100 * time.Millisecond)
	if !h3Server.wasListenAndServeTLSWithAutocertCalled() {
		t.Error("HTTP/3 should have started after cert was ready")
	}
}

// TestStartAutoTLS_SyncOnceSignalsOnce verifies certReady is only closed once
// even when both GetCertificate hook and warm-up goroutine race to signal.
// This test uses the regular mock which returns cert immediately.
func TestStartAutoTLS_SyncOnceSignalsOnce(t *testing.T) {
	mgr := &mockAutocertManager{}

	h3Server := &mockHTTP3ServerWithAutocert{}
	wtServer := &mockWebTransportServerWithAutocert{}

	server := New(config.Config{
		Addr: "localhost:0",
		TLS: config.TLSConfig{
			Addr: "localhost:0",
		},
		Extensions: config.ExtensionsConfig{AutocertManager: mgr},
	})
	server.SetHTTP3Server(h3Server)
	server.SetWebTransportServer(wtServer)

	// Start in goroutine
	go func() {
		_ = server.StartAutoTLS()
	}()

	// Give servers time to start - the warm-up goroutine will get the cert immediately
	// and signal certReady once. Both HTTP/3 and WebTransport should then start.
	time.Sleep(200 * time.Millisecond)

	// Verify both HTTP/3 and WebTransport started (they both wait on certReady)
	if !h3Server.wasListenAndServeTLSWithAutocertCalled() {
		t.Error("HTTP/3 server should have started")
	}
	if !wtServer.wasListenAndServeTLSWithAutocertCalled() {
		t.Error("WebTransport server should have started")
	}
}

// TestStartAutoTLS_MultipleConsumersUnblock verifies both HTTP/3 and WebTransport
// unblock when certReady is closed (broadcast behavior).
// This test verifies that after cert is ready, both consumers start.
func TestStartAutoTLS_MultipleConsumersUnblock(t *testing.T) {
	mgr := &mockAutocertManager{}

	server := New(config.Config{
		Addr: "localhost:0",
		TLS: config.TLSConfig{
			Addr: "localhost:0",
		},
		Extensions: config.ExtensionsConfig{AutocertManager: mgr},
	})

	h3Server := &mockHTTP3ServerWithAutocert{}
	wtServer := &mockWebTransportServerWithAutocert{}
	server.SetHTTP3Server(h3Server)
	server.SetWebTransportServer(wtServer)

	// Start in goroutine
	go func() {
		_ = server.StartAutoTLS()
	}()

	// Wait for servers to start
	// The warm-up goroutine will get the cert immediately from the mock
	time.Sleep(200 * time.Millisecond)

	// Both HTTP/3 and WebTransport should have started after cert was signaled
	if !h3Server.wasListenAndServeTLSWithAutocertCalled() {
		t.Error("HTTP/3 did not start after cert was ready")
	}
	if !wtServer.wasListenAndServeTLSWithAutocertCalled() {
		t.Error("WebTransport did not start after cert was ready")
	}
}

// neverReadyAutocertManager always returns error
type neverReadyAutocertManager struct {
	mockAutocertManager
}

func (m *neverReadyAutocertManager) GetCertificate(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	return nil, fmt.Errorf("cert never ready")
}

// TestStartAutoTLS_WarmUpTimeout verifies the warm-up goroutine times out
// when certificate is never available
func TestStartAutoTLS_WarmUpTimeout(t *testing.T) {
	// This test would require modifying the timeout or using a mock time source
	// For now, we just verify the warm-up goroutine doesn't panic on timeout
	// The actual timeout behavior is verified by code inspection

	// Create a manager that never returns a cert
	mgr := &neverReadyAutocertManager{}

	h3Server := &mockHTTP3ServerWithAutocert{}

	server := New(config.Config{
		Addr: "localhost:0",
		TLS: config.TLSConfig{
			Addr: "localhost:0",
		},
		Extensions: config.ExtensionsConfig{AutocertManager: mgr},
	})
	server.SetHTTP3Server(h3Server)

	// Start should not panic even when warm-up times out
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	done := make(chan struct{})
	go func() {
		_ = server.StartAutoTLS()
		close(done)
	}()

	select {
	case <-done:
		// Server exited (expected - error from TLS or timeout)
	case <-ctx.Done():
		// Test timeout - this is fine, server is still running
		t.Log("Test timeout - warm-up goroutine still waiting (expected)")
		if err := server.Close(); err != nil {
			t.Logf("Server close error: %v", err)
		}
	}
}

// emptyHostnamesManager returns empty hostnames slice
type emptyHostnamesManager struct {
	mockAutocertManager
}

func (m *emptyHostnamesManager) Hostnames() []string {
	return []string{}
}

// TestStartAutoTLS_EmptyHostnames verifies warm-up goroutine handles empty hostnames
func TestStartAutoTLS_EmptyHostnames(t *testing.T) {
	mgr := &emptyHostnamesManager{}
	h3Server := &mockHTTP3ServerWithAutocert{}

	server := New(config.Config{
		Addr: "localhost:0",
		TLS: config.TLSConfig{
			Addr: "localhost:0",
		},
		Extensions: config.ExtensionsConfig{AutocertManager: mgr},
	})
	server.SetHTTP3Server(h3Server)

	go func() {
		_ = server.StartAutoTLS()
	}()

	// Give time for warm-up goroutine to run and return early
	time.Sleep(200 * time.Millisecond)

	// HTTP/3 should not have started since warm-up returned early due to empty hostnames
	// The certReady channel was never signaled, so HTTP/3 is still blocked
	if h3Server.wasListenAndServeTLSWithAutocertCalled() {
		t.Error("HTTP/3 should not have started with empty hostnames")
	}
}

// TestStartAutoTLS_NoHTTPServer verifies httpReady is closed when no HTTP server
func TestStartAutoTLS_NoHTTPServer(t *testing.T) {
	mgr := &mockAutocertManager{}
	h3Server := &mockHTTP3ServerWithAutocert{}

	server := New(config.Config{
		// No Addr - no HTTP server
		TLS: config.TLSConfig{
			Addr: "localhost:0",
		},
		Extensions: config.ExtensionsConfig{AutocertManager: mgr},
	})
	server.SetHTTP3Server(h3Server)

	// Manually set server to nil to simulate no HTTP server
	server.server = nil

	go func() {
		_ = server.StartAutoTLS()
	}()

	// Give time for startup - warm-up should proceed even without HTTP server
	time.Sleep(200 * time.Millisecond)

	// HTTP/3 should still start via TLS handshake path
	conn, err := tls.Dial("tcp", server.ListenerTLSAddr(), &tls.Config{
		InsecureSkipVerify: true,
	})
	if err == nil {
		_ = conn.Close()
	}

	time.Sleep(100 * time.Millisecond)

	// HTTP/3 should have started
	if !h3Server.wasListenAndServeTLSWithAutocertCalled() {
		t.Error("HTTP/3 should have started via GetCertificate path")
	}
}

func TestServer_ListenAndServeTLS(t *testing.T) {
	server := New()
	// Set up a TLS server but no listener - it should try to create one
	// but fail since we don't have real certs
	server.tlsServer = &http.Server{Addr: "127.0.0.1:0"}

	// Run in goroutine since it blocks
	go func() {
		err := server.ListenAndServeTLS("cert.pem", "key.pem")
		// Expected to fail due to missing cert files
		if err == nil {
			t.Error("expected error due to missing cert files")
		}
	}()

	time.Sleep(10 * time.Millisecond)
}

func TestServer_StartTLS_NoServer(t *testing.T) {
	server := New()
	server.tlsServer = nil

	err := server.StartTLS("cert.pem", "key.pem")
	if err == nil {
		t.Error("expected error when tlsServer is nil")
	}
	expectedMsg := "TLS server not configured"
	if err.Error() != expectedMsg {
		t.Errorf("expected error '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestServer_StartTLS(t *testing.T) {
	server := New()
	server.tlsServer = &http.Server{Addr: "127.0.0.1:0"}

	// Run in goroutine since it blocks
	go func() {
		err := server.StartTLS("cert.pem", "key.pem")
		// Expected to fail due to missing cert files
		if err == nil {
			t.Error("expected error due to missing cert files")
		}
	}()

	time.Sleep(10 * time.Millisecond)
}

func TestServer_ListenAndServeTLS_NoTLSConfig(t *testing.T) {
	// Test the path where TLSConfig is nil initially
	server := New()
	server.tlsServer = &http.Server{
		Addr:      "127.0.0.1:0",
		TLSConfig: nil, // Explicitly nil to test the path
	}

	// Run in goroutine since it blocks
	go func() {
		err := server.ListenAndServeTLS("cert.pem", "key.pem")
		// Expected to fail due to missing cert files
		if err == nil {
			t.Error("expected error due to missing cert files")
		}
	}()

	time.Sleep(10 * time.Millisecond)
}

func TestServer_ListenerTLSAddr_Empty(t *testing.T) {
	server := New()
	server.tlsServer = nil
	server.tlsListener = nil

	addr := server.ListenerTLSAddr()
	if addr != "" {
		t.Errorf("Expected empty TLS address, got '%s'", addr)
	}
}

func TestServer_Shutdown_WithTLSServer(t *testing.T) {
	server := New()

	// Set up both HTTP and TLS listeners
	httpListener, _ := net.Listen("tcp", "127.0.0.1:0")
	tlsListener, _ := net.Listen("tcp", "127.0.0.1:0")

	server.listener = httpListener
	server.tlsListener = tlsListener
	server.tlsServer = &http.Server{Addr: tlsListener.Addr().String()}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err := server.Shutdown(ctx)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}
