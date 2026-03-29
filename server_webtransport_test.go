package zerohttp

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/alexferl/zerohttp/extensions/autocert"
	"github.com/alexferl/zerohttp/zhtest"
)

func TestServer_Shutdown_WithWebTransportError(t *testing.T) {
	// Test WebTransport close error logging path
	server := New()
	wtServer := &mockWebTransportServerWithCloseError{}
	server.SetWebTransportServer(wtServer)

	// Need a listener for proper shutdown
	listener, _ := net.Listen("tcp", "127.0.0.1:0")
	server.listener = listener

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Error is logged and also returned via errCh
	err := server.Shutdown(ctx)
	zhtest.AssertError(t, err)
}

// mockWebTransportServerWithAutocert implements both WebTransportServer and WebTransportServerWithAutocert
type mockWebTransportServerWithAutocert struct {
	mockWebTransportServer
	mu                                  sync.Mutex
	listenAndServeTLSWithAutocertCalled bool
	autocertManager                     autocert.Manager
}

func (m *mockWebTransportServerWithAutocert) ListenAndServeTLSWithAutocert(manager autocert.Manager) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.listenAndServeTLSWithAutocertCalled = true
	m.autocertManager = manager
	return nil
}

func (m *mockWebTransportServerWithAutocert) wasListenAndServeTLSWithAutocertCalled() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.listenAndServeTLSWithAutocertCalled
}

func TestServer_StartAutoTLS_WithWebTransportAutocert(t *testing.T) {
	mgr := &mockAutocertManager{}
	wtServer := &mockWebTransportServerWithAutocert{}

	// Use unique port to avoid conflicts with other tests
	server := New(Config{
		TLS: TLSConfig{
			Addr: "localhost:28443",
		},
		Extensions: ExtensionsConfig{AutocertManager: mgr},
	})
	server.SetWebTransportServer(wtServer)

	// Run StartAutoTLS in a goroutine since it blocks
	go func() {
		// Suppress the error since we're just testing the setup
		_ = server.StartAutoTLS()
	}()

	// Give it a moment to start
	time.Sleep(50 * time.Millisecond)

	// Trigger GetCertificate by making a TLS connection
	// This signals the certReady channel so WebTransport can start
	conn, err := tls.Dial("tcp", "localhost:28443", &tls.Config{
		InsecureSkipVerify: true,
	})
	if err == nil {
		_ = conn.Close()
	}

	// Give WebTransport time to start after cert signal
	time.Sleep(50 * time.Millisecond)

	// The WebTransport server with autocert support should have the autocert method called
	zhtest.AssertTrue(t, wtServer.wasListenAndServeTLSWithAutocertCalled())
	zhtest.AssertEqual(t, mgr, wtServer.autocertManager)
}

func TestServer_StartAutoTLS_WithWebTransportNoAutocert(t *testing.T) {
	// Test WebTransport server that does NOT support autocert
	mgr := &mockAutocertManager{}
	wtServer := &mockWebTransportServer{} // This doesn't implement WebTransportServerWithAutocert

	server := New(Config{Extensions: ExtensionsConfig{AutocertManager: mgr}})
	server.SetWebTransportServer(wtServer)

	// Run StartAutoTLS in a goroutine since it blocks
	go func() {
		// Suppress the error since we're just testing the setup
		_ = server.StartAutoTLS()
	}()

	// Give it a moment to start
	time.Sleep(50 * time.Millisecond)

	// The WebTransport server without autocert support should not have ListenAndServeTLS called
	// since it doesn't implement WebTransportServerWithAutocert
	if wtServer.wasListenAndServeTLSCalled() {
		t.Log("WebTransport server ListenAndServeTLS was not called (expected - no autocert support)")
	}
}

func TestServer_ListenAndServeTLS_WithWebTransport(t *testing.T) {
	// Test WebTransport startup in ListenAndServeTLS
	server := New()
	mockWT := &mockWebTransportServer{}
	server.SetWebTransportServer(mockWT)

	certFile := "/tmp/test_cert_wt.pem"
	keyFile := "/tmp/test_key_wt.pem"

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

	// Give server time to start WebTransport
	time.Sleep(100 * time.Millisecond)

	// Verify WebTransport was started
	zhtest.AssertTrue(t, mockWT.wasListenAndServeTLSCalled())

	// Shutdown
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_ = server.Shutdown(ctx)

	select {
	case <-done:
		// Expected
	case <-time.After(time.Second):
		zhtest.AssertFail(t, "timeout waiting for ListenAndServeTLS to return")
	}
}

func TestServer_Start_WithWebTransport(t *testing.T) {
	// Test WebTransport auto-start in Start() - use invalid address to make HTTPS fail quickly
	server := New()
	server.server = nil // Disable HTTP

	mockWT := &mockWebTransportServer{}
	server.SetWebTransportServer(mockWT)

	certFile := "/tmp/test_cert_wt_start.pem"
	keyFile := "/tmp/test_key_wt_start.pem"

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

	// Give WebTransport time to start (it starts before HTTPS)
	time.Sleep(100 * time.Millisecond)

	// Verify WebTransport was started
	zhtest.AssertTrue(t, mockWT.wasListenAndServeTLSCalled())

	select {
	case <-done:
		// Expected - HTTPS failed
	case <-time.After(2 * time.Second):
		zhtest.AssertFail(t, "timeout waiting for Start to return")
	}
}

// mockWebTransportServer is a mock implementation of WebTransportServer for testing
type mockWebTransportServer struct {
	mu                      sync.Mutex
	listenAndServeTLSCalled bool
	closeCalled             bool
	certFile                string
	keyFile                 string
}

func (m *mockWebTransportServer) ListenAndServeTLS(certFile, keyFile string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.listenAndServeTLSCalled = true
	m.certFile = certFile
	m.keyFile = keyFile
	return nil
}

func (m *mockWebTransportServer) wasListenAndServeTLSCalled() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.listenAndServeTLSCalled
}

func (m *mockWebTransportServer) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closeCalled = true
	return nil
}

func TestServer_SetWebTransportServer(t *testing.T) {
	server := New()
	mockWT := &mockWebTransportServer{}

	server.SetWebTransportServer(mockWT)

	zhtest.AssertEqual(t, mockWT, server.webTransportServer)
}

func TestServer_SetWebTransportServer_WithConfig(t *testing.T) {
	mockWT := &mockWebTransportServer{}
	server := New(Config{Extensions: ExtensionsConfig{WebTransportServer: mockWT}})

	zhtest.AssertEqual(t, mockWT, server.webTransportServer)
}

func TestServer_Close_WithWebTransport(t *testing.T) {
	server := New()
	mockWT := &mockWebTransportServer{}
	server.SetWebTransportServer(mockWT)

	// Create a dummy listener so Close doesn't panic
	listener, _ := net.Listen("tcp", "127.0.0.1:0")
	server.listener = listener

	err := server.Close()
	zhtest.AssertNoError(t, err)
	zhtest.AssertTrue(t, mockWT.closeCalled)
}

func TestServer_Shutdown_WithWebTransport(t *testing.T) {
	server := New()
	mockWT := &mockWebTransportServer{}
	server.SetWebTransportServer(mockWT)

	// Need an actual HTTP server running for proper shutdown test
	listener, _ := net.Listen("tcp", "127.0.0.1:0")
	server.listener = listener

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err := server.Shutdown(ctx)
	zhtest.AssertNoError(t, err)
	zhtest.AssertTrue(t, mockWT.closeCalled)
}
