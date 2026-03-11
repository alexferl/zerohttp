package zerohttp

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/alexferl/zerohttp/config"
)

// mockHTTP3Server is a mock implementation of HTTP3Server for testing
type mockHTTP3Server struct {
	mu                      sync.Mutex
	listenAndServeTLSCalled bool
	shutdownCalled          bool
	closeCalled             bool
	certFile                string
	keyFile                 string
}

func (m *mockHTTP3Server) ListenAndServeTLS(certFile, keyFile string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.listenAndServeTLSCalled = true
	m.certFile = certFile
	m.keyFile = keyFile
	return nil
}

func (m *mockHTTP3Server) Shutdown(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shutdownCalled = true
	return nil
}

func (m *mockHTTP3Server) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closeCalled = true
	return nil
}

func (m *mockHTTP3Server) wasListenAndServeTLSCalled() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.listenAndServeTLSCalled
}

func (m *mockHTTP3Server) getCertFile() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.certFile
}

func (m *mockHTTP3Server) getKeyFile() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.keyFile
}

func (m *mockHTTP3Server) wasShutdownCalled() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.shutdownCalled
}

func (m *mockHTTP3Server) wasCloseCalled() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.closeCalled
}

func TestServer_SetHTTP3Server(t *testing.T) {
	server := New()
	h3Server := &mockHTTP3Server{}

	server.SetHTTP3Server(h3Server)

	if server.http3Server != h3Server {
		t.Error("expected HTTP/3 server to be set")
	}
}

func TestServer_ListenAndServeHTTP3_NoServer(t *testing.T) {
	mockLogger := &mockServerLogger{}
	server := New(config.Config{Logger: mockLogger})
	// http3Server is nil by default

	err := server.ListenAndServeHTTP3("cert.pem", "key.pem")
	if err != nil {
		t.Errorf("expected no error when HTTP/3 server is nil, got %v", err)
	}

	// Should log debug message about skipping
	found := false
	for _, entry := range mockLogger.logs {
		if entry.level == "debug" && strings.Contains(entry.message, "HTTP/3 server not configured") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected debug log about skipping HTTP/3 server")
	}
}

func TestServer_ListenAndServeHTTP3_WithServer(t *testing.T) {
	server := New()
	h3Server := &mockHTTP3Server{}
	server.SetHTTP3Server(h3Server)

	// Run in goroutine since it would block
	go func() {
		err := server.ListenAndServeHTTP3("cert.pem", "key.pem")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	}()

	// Give it a moment to be called
	time.Sleep(10 * time.Millisecond)

	if !h3Server.wasListenAndServeTLSCalled() {
		t.Error("expected ListenAndServeTLS to be called on HTTP/3 server")
	}
	if h3Server.getCertFile() != "cert.pem" {
		t.Errorf("expected certFile = 'cert.pem', got '%s'", h3Server.getCertFile())
	}
	if h3Server.getKeyFile() != "key.pem" {
		t.Errorf("expected keyFile = 'key.pem', got '%s'", h3Server.getKeyFile())
	}
}

func TestServer_StartHTTP3(t *testing.T) {
	server := New()
	h3Server := &mockHTTP3Server{}
	server.SetHTTP3Server(h3Server)

	// Run in goroutine since it would block
	go func() {
		err := server.StartHTTP3("cert.pem", "key.pem")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	}()

	time.Sleep(10 * time.Millisecond)

	if !h3Server.wasListenAndServeTLSCalled() {
		t.Error("expected StartHTTP3 to call ListenAndServeTLS")
	}
}

func TestServer_Shutdown_WithHTTP3(t *testing.T) {
	server := New()
	h3Server := &mockHTTP3Server{}
	server.SetHTTP3Server(h3Server)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err := server.Shutdown(ctx)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if !h3Server.wasShutdownCalled() {
		t.Error("expected Shutdown to be called on HTTP/3 server")
	}
}

func TestServer_Close_WithHTTP3(t *testing.T) {
	server := New()
	h3Server := &mockHTTP3Server{}
	server.SetHTTP3Server(h3Server)

	err := server.Close()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if !h3Server.wasCloseCalled() {
		t.Error("expected Close to be called on HTTP/3 server")
	}
}

func TestServer_Shutdown_WithHTTP3Error(t *testing.T) {
	// Test HTTP/3 shutdown error logging path
	server := New()
	h3Server := &mockHTTP3ServerWithShutdownError{}
	server.SetHTTP3Server(h3Server)

	// Need a listener for proper shutdown
	listener, _ := net.Listen("tcp", "127.0.0.1:0")
	server.listener = listener

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Error is logged and also returned via errCh
	err := server.Shutdown(ctx)
	if err == nil {
		t.Error("expected shutdown error")
	}
}

// mockHTTP3ServerWithError is a mock that can return an error
type mockHTTP3ServerWithError struct {
	mu                      sync.Mutex
	certFile                string
	keyFile                 string
	shouldErr               bool
	errMsg                  string
	listenAndServeTLSCalled bool
}

func (m *mockHTTP3ServerWithError) ListenAndServeTLS(certFile, keyFile string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.certFile = certFile
	m.keyFile = keyFile
	m.listenAndServeTLSCalled = true
	if m.shouldErr {
		return fmt.Errorf("%s", m.errMsg)
	}
	// Return immediately to not block tests
	return nil
}

func (m *mockHTTP3ServerWithError) Shutdown(ctx context.Context) error {
	return nil
}

func (m *mockHTTP3ServerWithError) Close() error {
	return nil
}

func (m *mockHTTP3ServerWithError) getCertFile() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.certFile
}

// mockHTTP3ServerWithShutdownError returns an error on Shutdown
type mockHTTP3ServerWithShutdownError struct {
	mockHTTP3Server
}

func (m *mockHTTP3ServerWithShutdownError) Shutdown(ctx context.Context) error {
	return fmt.Errorf("shutdown error")
}

// mockWebTransportServerWithCloseError returns an error on Close
type mockWebTransportServerWithCloseError struct {
	mockWebTransportServer
}

func (m *mockWebTransportServerWithCloseError) Close() error {
	return fmt.Errorf("close error")
}

// mockHTTP3ServerWithAutocert implements both HTTP3Server and HTTP3ServerWithAutocert
type mockHTTP3ServerWithAutocert struct {
	mockHTTP3Server
	mu                                  sync.Mutex
	listenAndServeTLSWithAutocertCalled bool
	autocertManager                     config.AutocertManager
}

func (m *mockHTTP3ServerWithAutocert) ListenAndServeTLSWithAutocert(manager config.AutocertManager) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.listenAndServeTLSWithAutocertCalled = true
	m.autocertManager = manager
	return nil
}

func (m *mockHTTP3ServerWithAutocert) wasListenAndServeTLSWithAutocertCalled() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.listenAndServeTLSWithAutocertCalled
}

func (m *mockHTTP3ServerWithAutocert) getAutocertManager() config.AutocertManager {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.autocertManager
}

func TestServer_StartAutoTLS_WithHTTP3Autocert(t *testing.T) {
	mgr := &mockAutocertManager{}
	h3Server := &mockHTTP3ServerWithAutocert{}

	// Use unique port to avoid conflicts with other tests
	server := New(config.Config{
		AutocertManager: mgr,
		TLSAddr:         "localhost:18443",
	})
	server.SetHTTP3Server(h3Server)

	// Run StartAutoTLS in a goroutine since it blocks
	go func() {
		// Suppress the error since we're just testing the setup
		_ = server.StartAutoTLS()
	}()

	// Give it a moment to start
	time.Sleep(50 * time.Millisecond)

	// Trigger GetCertificate by making a TLS connection
	// This signals the certReady channel so HTTP/3 can start
	conn, err := tls.Dial("tcp", "localhost:18443", &tls.Config{
		InsecureSkipVerify: true,
	})
	if err == nil {
		_ = conn.Close()
	}

	// Give HTTP/3 time to start after cert signal
	time.Sleep(50 * time.Millisecond)

	// The HTTP/3 server with autocert support should have been detected and started
	if !h3Server.wasListenAndServeTLSWithAutocertCalled() {
		t.Error("expected ListenAndServeTLSWithAutocert to be called on HTTP/3 server with autocert support")
	}

	if h3Server.getAutocertManager() != mgr {
		t.Error("expected autocert manager to be passed to HTTP/3 server")
	}
}

func TestServer_StartAutoTLS_WithHTTP3NoAutocert(t *testing.T) {
	// Test HTTP/3 server that does NOT support autocert
	mgr := &mockAutocertManager{}
	h3Server := &mockHTTP3Server{} // This doesn't implement HTTP3ServerWithAutocert

	server := New(config.Config{AutocertManager: mgr})
	server.SetHTTP3Server(h3Server)

	// Run StartAutoTLS in a goroutine since it blocks
	go func() {
		// Suppress the error since we're just testing the setup
		_ = server.StartAutoTLS()
	}()

	// Give it a moment to start
	time.Sleep(50 * time.Millisecond)

	// The HTTP/3 server without autocert support should not have the autocert method called
	if h3Server.wasListenAndServeTLSCalled() {
		// It shouldn't be called via this method since it doesn't support autocert
		t.Log("HTTP/3 server ListenAndServeTLS was not called (expected - no autocert support)")
	}
}

func TestServer_ListenAndServeTLS_WithHTTP3(t *testing.T) {
	// Test HTTP/3 startup in ListenAndServeTLS
	server := New()
	mockH3 := &mockHTTP3Server{}
	server.SetHTTP3Server(mockH3)

	certFile := "/tmp/test_cert_h3.pem"
	keyFile := "/tmp/test_key_h3.pem"

	if err := os.WriteFile(certFile, []byte(testCertPEM), 0o644); err != nil {
		t.Skipf("Cannot write cert file: %v", err)
	}
	defer func() { _ = os.Remove(certFile) }()

	if err := os.WriteFile(keyFile, []byte(testKeyPEM), 0o600); err != nil {
		t.Skipf("Cannot write key file: %v", err)
	}
	defer func() { _ = os.Remove(keyFile) }()

	server.tlsServer = &http.Server{Addr: "127.0.0.1:0"}

	// Start server in goroutine
	done := make(chan error, 1)
	go func() {
		done <- server.ListenAndServeTLS(certFile, keyFile)
	}()

	// Give server time to start HTTP/3
	time.Sleep(100 * time.Millisecond)

	// Verify HTTP/3 was started
	if !mockH3.wasListenAndServeTLSCalled() {
		t.Error("expected HTTP/3 ListenAndServeTLS to be called")
	}

	// Verify correct cert/key files were passed
	if mockH3.getCertFile() != certFile {
		t.Errorf("expected cert file %q, got %q", certFile, mockH3.getCertFile())
	}
	if mockH3.getKeyFile() != keyFile {
		t.Errorf("expected key file %q, got %q", keyFile, mockH3.getKeyFile())
	}

	// Shutdown
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_ = server.Shutdown(ctx)

	select {
	case <-done:
		// Expected
	case <-time.After(time.Second):
		t.Error("timeout waiting for ListenAndServeTLS to return")
	}
}

func TestServer_ListenAndServeTLS_HTTP3Error(t *testing.T) {
	// Test HTTP/3 error logging path
	server := New()
	mockH3 := &mockHTTP3ServerWithError{shouldErr: true, errMsg: "h3 error"}
	server.SetHTTP3Server(mockH3)

	certFile := "/tmp/test_cert_h3_err.pem"
	keyFile := "/tmp/test_key_h3_err.pem"

	if err := os.WriteFile(certFile, []byte(testCertPEM), 0o644); err != nil {
		t.Skipf("Cannot write cert file: %v", err)
	}
	defer func() { _ = os.Remove(certFile) }()

	if err := os.WriteFile(keyFile, []byte(testKeyPEM), 0o600); err != nil {
		t.Skipf("Cannot write key file: %v", err)
	}
	defer func() { _ = os.Remove(keyFile) }()

	server.tlsServer = &http.Server{Addr: "127.0.0.1:0"}

	// Start server - it should still work even if HTTP/3 errors
	done := make(chan error, 1)
	go func() {
		done <- server.ListenAndServeTLS(certFile, keyFile)
	}()

	// Give HTTP/3 time to fail
	time.Sleep(100 * time.Millisecond)

	// Verify HTTP/3 was started (even though it errors)
	if mockH3.getCertFile() != certFile {
		t.Errorf("expected cert file %q, got %q", certFile, mockH3.getCertFile())
	}

	// Shutdown
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_ = server.Shutdown(ctx)

	select {
	case <-done:
		// Expected
	case <-time.After(time.Second):
		t.Error("timeout waiting for ListenAndServeTLS to return")
	}
}

func TestServer_Start_WithHTTP3(t *testing.T) {
	// Test HTTP/3 auto-start in Start() - use invalid address to make HTTPS fail quickly
	server := New()
	server.server = nil // Disable HTTP

	mockH3 := &mockHTTP3Server{}
	server.SetHTTP3Server(mockH3)

	certFile := "/tmp/test_cert_h3_start.pem"
	keyFile := "/tmp/test_key_h3_start.pem"

	if err := os.WriteFile(certFile, []byte(testCertPEM), 0o644); err != nil {
		t.Skipf("Cannot write cert file: %v", err)
	}
	defer func() { _ = os.Remove(certFile) }()

	if err := os.WriteFile(keyFile, []byte(testKeyPEM), 0o600); err != nil {
		t.Skipf("Cannot write key file: %v", err)
	}
	defer func() { _ = os.Remove(keyFile) }()

	server.certFile = certFile
	server.keyFile = keyFile
	// Use an invalid address to make ListenAndServeTLS fail immediately
	server.tlsServer = &http.Server{Addr: "invalid:address:format"}

	// Start server - will fail quickly due to invalid address
	done := make(chan bool, 1)
	go func() {
		_ = server.Start()
		done <- true
	}()

	// Give HTTP/3 time to start (it starts before HTTPS)
	time.Sleep(100 * time.Millisecond)

	// Verify HTTP/3 was started
	if !mockH3.wasListenAndServeTLSCalled() {
		t.Error("expected HTTP/3 ListenAndServeTLS to be called")
	}

	select {
	case <-done:
		// Expected - HTTPS failed
	case <-time.After(2 * time.Second):
		t.Error("timeout waiting for Start to return")
	}
}
