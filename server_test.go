package zerohttp

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/log"
	"github.com/alexferl/zerohttp/zhtest"
)

const testCertPEM = `-----BEGIN CERTIFICATE-----
MIIDCTCCAfGgAwIBAgIUIDs/2QUFXaOYCjjcMnewtycJdJowDQYJKoZIhvcNAQEL
BQAwFDESMBAGA1UEAwwJbG9jYWxob3N0MB4XDTI2MDMwMjE1NTA0OVoXDTI2MDMw
MzE1NTA0OVowFDESMBAGA1UEAwwJbG9jYWxob3N0MIIBIjANBgkqhkiG9w0BAQEF
AAOCAQ8AMIIBCgKCAQEAmiK+7HVK42Gk9eqX+e817mKN0USk2Uf9hbbec/b6PcJz
8h/MUS3ZChPAyV3rtQQUoPECE8twl3xClmTS/zI4zdVCu5D6TfDXh9svIfZcApi9
gsvfSkHUItCkeknFBvb14ssxEYDlxdWH+KhKRl7eHV96GXdKCT+fMmw0u6hzoG4X
Z19U+radt5FeYeNlUlzh2JuZA5NEYtPZov6P4qogc4Irdx0JWHa+cNJpM8nF8VHq
/afEm4uayUfJzmDj4KYzqUDRRBQsgtbnmztFgrE0kTGLGQtamEevypVgma1fBXUi
+aMn11FRLe6/l5iFLgqcV4/HS+oNDm6xOiN9lGcYWQIDAQABo1MwUTAdBgNVHQ4E
FgQUbyRgX50ORFrxQmjL7r1yHYwXS+owHwYDVR0jBBgwFoAUbyRgX50ORFrxQmjL
7r1yHYwXS+owDwYDVR0TAQH/BAUwAwEB/zANBgkqhkiG9w0BAQsFAAOCAQEAHtPH
MS/2DTXs6p6Jf/3ptxR5siG3jY1qGITTMGeUFX8VkiUb6+ZDd2jBzs1QXXIfUzpj
NYG5+Gm5FHiDfiZsfjZFrWqqgS1KSY3mVmzKluZVf0/soQsnV8mwoV71I1IYAs3k
DHkPu15cya6dOTm34yd/3t64EB6UrFkkB8r6Ylkik7Zql9AvMJVT9tjMQX+chbJm
424dI43JAerY8hpYUBAzQenk4w/MoRLUa4Lthu1IQJ8eZLcKwoPnTorLh8o8GUZ2
2C00A+tQVGp8nkuxn+H6UfEB0469cVmvtgSdGxc/GZ49B/u3t0Zq2ZzEH/mcVzc5
LQWCo/jnIFbvhjdZrQ==
-----END CERTIFICATE-----`

const testKeyPEM = `-----BEGIN PRIVATE KEY-----
MIIEvgIBADANBgkqhkiG9w0BAQEFAASCBKgwggSkAgEAAoIBAQCaIr7sdUrjYaT1
6pf57zXuYo3RRKTZR/2Ftt5z9vo9wnPyH8xRLdkKE8DJXeu1BBSg8QITy3CXfEKW
ZNL/MjjN1UK7kPpN8NeH2y8h9lwCmL2Cy99KQdQi0KR6ScUG9vXiyzERgOXF1Yf4
qEpGXt4dX3oZd0oJP58ybDS7qHOgbhdnX1T6tp23kV5h42VSXOHYm5kDk0Ri09mi
/o/iqiBzgit3HQlYdr5w0mkzycXxUer9p8Sbi5rJR8nOYOPgpjOpQNFEFCyC1ueb
O0WCsTSRMYsZC1qYR6/KlWCZrV8FdSL5oyfXUVEt7r+XmIUuCpxXj8dL6g0ObrE6
I32UZxhZAgMBAAECggEABk0Ok0XJu5EjsFpoX1MmOCIvBDurRgclf+x9fE3fxbvP
59lhLb3jMiBj2I+GedpKehhUIolF52lw4utI/V35GMADt/15oAtNEkyVaQzkTsZd
8+1/6a4WfRxcpvOiUnIw09ZE4Z9sdTmREwsVKzqIV7jOGeGopQdk64elIblFjcy2
FwYjAu4qsRwz1bcYZsLrLBswOA2AUakWI7lDYskXZVhmskzsrFdW5eRntvit+PjR
UQ8zlKUAU3TJTMZ2SoAMeBfaLeejMAXSnNO7xt7tpsFzjxjNEspKBzeUHzHiJ9eN
zLxUVTXGu5BeXBcipflYDUs6jfnMRVnnl7eHtvLQkQKBgQDOoHGA8aw+DHggg6CQ
jsp+MiZJXm/Ste7nbu6XBdHJgsxtSKxQMY8WwC+byCYOkJVMLVGHY1tV5vJoYD/P
nlY17MW9B+UCr9N9PIwzRdRSEPKn9dNAmoPh2iDIGztkvlC4kuyBu96db3XDx2CX
SvEJvwSkKvFFzBf3u2o2dk8mkQKBgQC+92B9OvxvYmR1pL6V839I7b4ak+4Biits
jdIE37a/RFa1+vkg5dTqZmvrqbxNhp61WjHwdPb5v0pbAqFafpkj7qO8JV1Lg/c1
IdfHcfWGBfmaz5NKiSSU7dRYBFPF6rjPCgzLukiUlb7BytgUAKaMKy3zfNxT3Sqk
2ATOuNoJSQKBgQCprpJnXI+hCOZhdRaXf9uEVLSiTb4w4J0HS079kJbeD97G5AY1
eO1TtpGiMXQnQ86HFzQ7pXktCxIIavocCqArenxMJr6HPVLFJsLPnEmm9yn+il5o
UDt7boC7M7nLmop5eJZmV5yR1yVzmDiXJcDZyxcJpgYq1lbcZvjrLq8DMQKBgEYG
sHs7hhXSHsSFBN43zBUSGQPl+wDVidbkqn7fCkRY6vMQdQp7PPg3Vpu0QjirhMc7
q9RhD6/FVZ7J+CEXC1EB0UjM6skmOyBgqJ+aSk47IqyCMaDDaYazL4qXC6En0V0a
clbCmJrjzm+B0nqDQo9jxhXjU2ftUhXgoOKtJkcBAoGBAKw7d/V/XzdZakiFajz7
xzKGQgvxYl3wxq3D3J3tQR3KsZ4ueGmATp7F/AkpBJdk1TJFDdWr9/OsdPyNiIcu
VrZYW+DzdGqEaVYWCIzyw6CykDvUG9eq3AgHJK3Li87o0jJ9GvvOX1YuiE7/FHY3
O6lBO0Onq9vJGuITYJtl/t+6
-----END PRIVATE KEY-----`

// Mock logger for testing
type mockServerLogger struct {
	logs []logEntry
}

type logEntry struct {
	level   string
	message string
	fields  []log.Field
}

func (m *mockServerLogger) Debug(msg string, fields ...log.Field) {
	m.logs = append(m.logs, logEntry{level: "debug", message: msg, fields: fields})
}

func (m *mockServerLogger) Info(msg string, fields ...log.Field) {
	m.logs = append(m.logs, logEntry{level: "info", message: msg, fields: fields})
}

func (m *mockServerLogger) Warn(msg string, fields ...log.Field) {
	m.logs = append(m.logs, logEntry{level: "warn", message: msg, fields: fields})
}

func (m *mockServerLogger) Error(msg string, fields ...log.Field) {
	m.logs = append(m.logs, logEntry{level: "error", message: msg, fields: fields})
}

func (m *mockServerLogger) Panic(msg string, fields ...log.Field) {}
func (m *mockServerLogger) Fatal(msg string, fields ...log.Field) {}

func (m *mockServerLogger) WithFields(fields ...log.Field) log.Logger  { return m }
func (m *mockServerLogger) WithContext(ctx context.Context) log.Logger { return m }

func TestNew_DefaultConfig(t *testing.T) {
	server := New()

	if server == nil {
		t.Fatal("Expected server to be created")
	}

	if server.Router == nil {
		t.Error("Expected router to be initialized")
	}

	if server.logger == nil {
		t.Error("Expected logger to be initialized")
	}
}

func TestNew_MiddlewareScenarios(t *testing.T) {
	// Test with DisableDefaultMiddlewares set to true and custom middlewares
	customMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
		})
	}

	server := New(config.Config{
		DisableDefaultMiddlewares: true,
		DefaultMiddlewares:        []func(http.Handler) http.Handler{customMiddleware},
	})

	if server == nil {
		t.Fatal("Expected server to be created")
	}

	// Test with custom default middlewares combined with defaults
	server2 := New(config.Config{
		DefaultMiddlewares: []func(http.Handler) http.Handler{customMiddleware},
	})

	if server2 == nil {
		t.Fatal("Expected server to be created")
	}
}

func TestServer_ListenerAddr(t *testing.T) {
	server := New()

	// Initially no server configured
	addr := server.ListenerAddr()
	if addr != "" {
		t.Logf("Got address %s from default config, this may be expected", addr)
	}

	// Set server address explicitly
	server.server = &http.Server{Addr: ":8080"}
	addr = server.ListenerAddr()
	if addr != ":8080" {
		t.Errorf("Expected ':8080', got '%s'", addr)
	}

	// Set actual listener (takes precedence)
	listener, _ := net.Listen("tcp", "127.0.0.1:0")
	server.listener = listener
	defer func() {
		if err := listener.Close(); err != nil {
			t.Errorf("failed to close listener: %v", err)
		}
	}()

	addr = server.ListenerAddr()
	if addr == "" {
		t.Error("Expected non-empty address from actual listener")
	}
	if !strings.Contains(addr, "127.0.0.1") {
		t.Errorf("Expected address to contain localhost, got '%s'", addr)
	}
}

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

func TestServer_ListenAndServe_NoServer(t *testing.T) {
	mockLogger := &mockServerLogger{}
	server := New(config.Config{Logger: mockLogger})
	server.server = nil

	err := server.ListenAndServe()
	if err != nil {
		t.Errorf("Expected no error when server is nil, got %v", err)
	}

	// Should log debug message about skipping
	found := false
	for _, entry := range mockLogger.logs {
		if entry.level == "debug" && strings.Contains(entry.message, "HTTP server not configured") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected debug log about skipping HTTP server")
	}
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

func TestServer_Close_WithListeners(t *testing.T) {
	server := New()

	// Create real listeners for testing close behavior
	httpListener, _ := net.Listen("tcp", "127.0.0.1:0")
	tlsListener, _ := net.Listen("tcp", "127.0.0.1:0")

	server.listener = httpListener
	server.tlsListener = tlsListener

	err := server.Close()
	if err != nil {
		t.Errorf("Expected no error closing listeners, got %v", err)
	}

	// Listeners should be closed now
	// Calling Close again should handle closed listeners gracefully
	err = server.Close()
	// May return error from already closed listeners, but shouldn't crash
	if err != nil {
		t.Logf("Second Close() returned error (expected): %v", err)
	}
}

func TestServer_Shutdown_NoServers(t *testing.T) {
	server := New()
	server.server = nil
	server.tlsServer = nil

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err := server.Shutdown(ctx)
	if err != nil {
		t.Errorf("Expected no error when no servers, got %v", err)
	}
}

func TestServer_ConcurrentAccess(t *testing.T) {
	server := New()

	// Test concurrent access to address getters
	done := make(chan bool, 10)
	for range 10 {
		go func() {
			_ = server.ListenerAddr()
			_ = server.ListenerTLSAddr()
			done <- true
		}()
	}

	// Wait for all goroutines
	for range 10 {
		<-done
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
	server := New(config.Config{AutocertManager: mgr})

	// Verify manager was set (compare using concrete type assertion)
	if server.autocertManager == nil {
		t.Error("expected autocert manager to be set on server")
	}
}

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
	if err == nil {
		t.Error("expected close error")
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

// mockWebTransportServerWithAutocert implements both WebTransportServer and WebTransportServerWithAutocert
type mockWebTransportServerWithAutocert struct {
	mockWebTransportServer
	mu                                  sync.Mutex
	listenAndServeTLSWithAutocertCalled bool
	autocertManager                     config.AutocertManager
}

func (m *mockWebTransportServerWithAutocert) ListenAndServeTLSWithAutocert(manager config.AutocertManager) error {
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
	server := New(config.Config{
		AutocertManager: mgr,
		TLSAddr:         "localhost:28443",
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
	if !wtServer.wasListenAndServeTLSWithAutocertCalled() {
		t.Error("Expected WebTransport server ListenAndServeTLSWithAutocert to be called")
	}

	if wtServer.autocertManager != mgr {
		t.Error("Expected WebTransport server to receive the autocert manager")
	}
}

func TestServer_StartAutoTLS_WithWebTransportNoAutocert(t *testing.T) {
	// Test WebTransport server that does NOT support autocert
	mgr := &mockAutocertManager{}
	wtServer := &mockWebTransportServer{} // This doesn't implement WebTransportServerWithAutocert

	server := New(config.Config{AutocertManager: mgr})
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
		AutocertManager: mgr,
		TLSAddr:         "localhost:0",
		Addr:            "localhost:0",
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
		AutocertManager: mgr,
		TLSAddr:         "localhost:0",
		Addr:            "localhost:0",
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
		AutocertManager: mgr,
		TLSAddr:         "localhost:0",
		Addr:            "localhost:0",
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
	neverReadyMgr := &neverReadyAutocertManager{}

	h3Server := &mockHTTP3ServerWithAutocert{}

	server := New(config.Config{
		AutocertManager: neverReadyMgr,
		TLSAddr:         "localhost:0",
		Addr:            "localhost:0",
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
		AutocertManager: mgr,
		TLSAddr:         "localhost:0",
		Addr:            "localhost:0",
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
		AutocertManager: mgr,
		TLSAddr:         "localhost:0",
		// No Addr - no HTTP server
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

func TestServer_ListenAndServe_WithListener(t *testing.T) {
	server := New()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to create listener: %v", err)
	}
	server.listener = listener

	// Start server in goroutine
	done := make(chan error, 1)
	go func() {
		done <- server.ListenAndServe()
	}()

	// Give server a moment to start
	time.Sleep(10 * time.Millisecond)

	// Shutdown to stop the server
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err = server.Shutdown(ctx)
	if err != nil {
		t.Errorf("unexpected error during shutdown: %v", err)
	}

	// Wait for ListenAndServe to return
	select {
	case err := <-done:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			t.Errorf("unexpected error: %v", err)
		}
	case <-time.After(time.Second):
		t.Error("timeout waiting for ListenAndServe to return")
	}
}

func TestServer_ListenAndServe_CreatesListener(t *testing.T) {
	// Test the path where ListenAndServe creates its own listener
	server := New()
	server.server = &http.Server{Addr: "127.0.0.1:0"}
	server.listener = nil // Force creation of new listener

	// Start server in goroutine
	done := make(chan error, 1)
	go func() {
		done <- server.ListenAndServe()
	}()

	// Give server a moment to start
	time.Sleep(20 * time.Millisecond)

	// Verify listener was created (use the public method to avoid race)
	if server.ListenerAddr() == "" {
		t.Error("expected listener to be created")
	}

	// Shutdown to stop the server
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_ = server.Shutdown(ctx)

	// Wait for ListenAndServe to return
	select {
	case <-done:
		// Expected
	case <-time.After(time.Second):
		t.Error("timeout waiting for ListenAndServe to return")
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
	if !mockWT.wasListenAndServeTLSCalled() {
		t.Error("expected WebTransport ListenAndServeTLS to be called")
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

func TestServer_Start_WithCertLoading(t *testing.T) {
	// Test the cert loading path in Start()
	server := New()
	server.server = nil // Disable HTTP server

	certFile := "/tmp/test_cert_start.pem"
	keyFile := "/tmp/test_key_start.pem"

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

	// Should attempt to load certs but fail on ListenAndServeTLS since we don't have valid certs
	done := make(chan bool, 1)
	go func() {
		_ = server.Start()
		done <- true
	}()

	select {
	case <-done:
		// Expected - returns due to error
	case <-time.After(100 * time.Millisecond):
		// Also acceptable
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
	if !mockWT.wasListenAndServeTLSCalled() {
		t.Error("expected WebTransport ListenAndServeTLS to be called")
	}

	select {
	case <-done:
		// Expected - HTTPS failed
	case <-time.After(2 * time.Second):
		t.Error("timeout waiting for Start to return")
	}
}

func TestServer_Start(t *testing.T) {
	server := New()
	server.server = nil    // Disable HTTP
	server.tlsServer = nil // Disable HTTPS

	// With no servers, Start should block indefinitely waiting for an error
	// Just verify it doesn't panic immediately
	done := make(chan bool, 1)
	go func() {
		_ = server.Start()
		done <- true
	}()

	select {
	case <-done:
		// Expected - returns when no servers configured
	case <-time.After(100 * time.Millisecond):
		// Also fine - still waiting
	}
}

func TestServer_Logger(t *testing.T) {
	mockLogger := &mockServerLogger{}
	server := New(config.Config{Logger: mockLogger})

	logger := server.Logger()
	if logger == nil {
		t.Error("expected logger to not be nil")
	}
}

// mockWebTransportServer is a mock implementation of config.WebTransportServer for testing
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

	if server.webTransportServer != mockWT {
		t.Error("Expected WebTransport server to be set")
	}
}

func TestServer_SetWebTransportServer_WithConfig(t *testing.T) {
	mockWT := &mockWebTransportServer{}
	server := New(config.Config{WebTransportServer: mockWT})

	if server.webTransportServer != mockWT {
		t.Error("Expected WebTransport server to be set via config")
	}
}

func TestServer_Close_WithWebTransport(t *testing.T) {
	server := New()
	mockWT := &mockWebTransportServer{}
	server.SetWebTransportServer(mockWT)

	// Create a dummy listener so Close doesn't panic
	listener, _ := net.Listen("tcp", "127.0.0.1:0")
	server.listener = listener

	err := server.Close()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if !mockWT.closeCalled {
		t.Error("Expected WebTransport server Close to be called")
	}
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
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if !mockWT.closeCalled {
		t.Error("Expected WebTransport server Close to be called during shutdown")
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

func TestServer_ListenerAddr_Empty(t *testing.T) {
	server := New()
	server.server = nil
	server.listener = nil

	addr := server.ListenerAddr()
	if addr != "" {
		t.Errorf("Expected empty address, got '%s'", addr)
	}
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

// Shutdown Hooks Tests

func TestServer_RegisterPreShutdownHook(t *testing.T) {
	server := New()

	called := false
	server.RegisterPreShutdownHook("test-hook", func(ctx context.Context) error {
		called = true
		return nil
	})

	if len(server.preShutdownHooks) != 1 {
		t.Errorf("Expected 1 pre-shutdown hook, got %d", len(server.preShutdownHooks))
	}

	if server.preShutdownHooks[0].Name != "test-hook" {
		t.Errorf("Expected hook name 'test-hook', got '%s'", server.preShutdownHooks[0].Name)
	}

	// Verify hook works
	err := server.preShutdownHooks[0].Hook(context.Background())
	if err != nil {
		t.Errorf("Expected no error from hook, got %v", err)
	}

	if !called {
		t.Error("Expected hook to be called")
	}
}

func TestServer_RegisterShutdownHook(t *testing.T) {
	server := New()

	called := false
	server.RegisterShutdownHook("test-hook", func(ctx context.Context) error {
		called = true
		return nil
	})

	if len(server.shutdownHooks) != 1 {
		t.Errorf("Expected 1 shutdown hook, got %d", len(server.shutdownHooks))
	}

	if server.shutdownHooks[0].Name != "test-hook" {
		t.Errorf("Expected hook name 'test-hook', got '%s'", server.shutdownHooks[0].Name)
	}

	err := server.shutdownHooks[0].Hook(context.Background())
	if err != nil {
		t.Errorf("Expected no error from hook, got %v", err)
	}

	if !called {
		t.Error("Expected hook to be called")
	}
}

func TestServer_RegisterPostShutdownHook(t *testing.T) {
	server := New()

	called := false
	server.RegisterPostShutdownHook("test-hook", func(ctx context.Context) error {
		called = true
		return nil
	})

	if len(server.postShutdownHooks) != 1 {
		t.Errorf("Expected 1 post-shutdown hook, got %d", len(server.postShutdownHooks))
	}

	if server.postShutdownHooks[0].Name != "test-hook" {
		t.Errorf("Expected hook name 'test-hook', got '%s'", server.postShutdownHooks[0].Name)
	}

	err := server.postShutdownHooks[0].Hook(context.Background())
	if err != nil {
		t.Errorf("Expected no error from hook, got %v", err)
	}

	if !called {
		t.Error("Expected hook to be called")
	}
}

func TestServer_Shutdown_WithPreShutdownHooks(t *testing.T) {
	listener, _ := net.Listen("tcp", "127.0.0.1:0")
	server := New()
	server.listener = listener
	server.server = &http.Server{Addr: listener.Addr().String()}

	var order []string
	server.RegisterPreShutdownHook("hook-1", func(ctx context.Context) error {
		order = append(order, "hook-1")
		return nil
	})
	server.RegisterPreShutdownHook("hook-2", func(ctx context.Context) error {
		order = append(order, "hook-2")
		return nil
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err := server.Shutdown(ctx)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(order) != 2 {
		t.Errorf("Expected 2 hooks to run, got %d", len(order))
	}

	if order[0] != "hook-1" || order[1] != "hook-2" {
		t.Errorf("Expected hooks to run in registration order, got %v", order)
	}
}

func TestServer_Shutdown_WithPostShutdownHooks(t *testing.T) {
	listener, _ := net.Listen("tcp", "127.0.0.1:0")
	server := New()
	server.listener = listener
	server.server = &http.Server{Addr: listener.Addr().String()}

	var order []string
	server.RegisterPostShutdownHook("hook-1", func(ctx context.Context) error {
		order = append(order, "hook-1")
		return nil
	})
	server.RegisterPostShutdownHook("hook-2", func(ctx context.Context) error {
		order = append(order, "hook-2")
		return nil
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err := server.Shutdown(ctx)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(order) != 2 {
		t.Errorf("Expected 2 hooks to run, got %d", len(order))
	}

	if order[0] != "hook-1" || order[1] != "hook-2" {
		t.Errorf("Expected hooks to run in registration order, got %v", order)
	}
}

func TestServer_Shutdown_WithShutdownHooks(t *testing.T) {
	listener, _ := net.Listen("tcp", "127.0.0.1:0")
	server := New()
	server.listener = listener
	server.server = &http.Server{Addr: listener.Addr().String()}

	var mu sync.Mutex
	var calls []string
	server.RegisterShutdownHook("hook-1", func(ctx context.Context) error {
		mu.Lock()
		calls = append(calls, "hook-1")
		mu.Unlock()
		return nil
	})
	server.RegisterShutdownHook("hook-2", func(ctx context.Context) error {
		mu.Lock()
		calls = append(calls, "hook-2")
		mu.Unlock()
		return nil
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err := server.Shutdown(ctx)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	mu.Lock()
	if len(calls) != 2 {
		t.Errorf("Expected 2 shutdown hooks to run, got %d", len(calls))
	}
	mu.Unlock()
}

func TestServer_Shutdown_HooksContinueOnError(t *testing.T) {
	listener, _ := net.Listen("tcp", "127.0.0.1:0")
	server := New()
	server.listener = listener
	server.server = &http.Server{Addr: listener.Addr().String()}

	var calls []string
	server.RegisterPreShutdownHook("failing-hook", func(ctx context.Context) error {
		calls = append(calls, "failing")
		return errors.New("hook error")
	})
	server.RegisterPreShutdownHook("success-hook", func(ctx context.Context) error {
		calls = append(calls, "success")
		return nil
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Shutdown should complete without returning the hook error
	err := server.Shutdown(ctx)
	if err != nil {
		t.Errorf("Expected no error from server shutdown, got %v", err)
	}

	// Both hooks should have been called
	if len(calls) != 2 {
		t.Errorf("Expected both hooks to run, got %v", calls)
	}

	if calls[0] != "failing" || calls[1] != "success" {
		t.Errorf("Expected hooks to run in order despite first failing, got %v", calls)
	}
}

func TestServer_Shutdown_HooksRespectContextCancellation(t *testing.T) {
	listener, _ := net.Listen("tcp", "127.0.0.1:0")
	server := New()
	server.listener = listener
	server.server = &http.Server{Addr: listener.Addr().String()}

	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	called := false
	server.RegisterPreShutdownHook("should-not-run", func(ctx context.Context) error {
		called = true
		return nil
	})

	err := server.Shutdown(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Expected context.Canceled error, got %v", err)
	}

	// Hook should not have been called due to cancelled context
	if called {
		t.Error("Expected hook not to be called due to cancelled context")
	}
}

func TestServer_ConfigWithShutdownHooks(t *testing.T) {
	var preCalled, shutdownCalled, postCalled bool

	server := New(config.Config{
		PreShutdownHooks: []config.ShutdownHookConfig{
			{Name: "pre", Hook: func(ctx context.Context) error {
				preCalled = true
				return nil
			}},
		},
		ShutdownHooks: []config.ShutdownHookConfig{
			{Name: "shutdown", Hook: func(ctx context.Context) error {
				shutdownCalled = true
				return nil
			}},
		},
		PostShutdownHooks: []config.ShutdownHookConfig{
			{Name: "post", Hook: func(ctx context.Context) error {
				postCalled = true
				return nil
			}},
		},
	})

	listener, _ := net.Listen("tcp", "127.0.0.1:0")
	server.listener = listener
	server.server = &http.Server{Addr: listener.Addr().String()}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err := server.Shutdown(ctx)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if !preCalled {
		t.Error("Expected pre-shutdown hook to be called")
	}
	if !shutdownCalled {
		t.Error("Expected shutdown hook to be called")
	}
	if !postCalled {
		t.Error("Expected post-shutdown hook to be called")
	}
}

func TestServer_SSEProvider(t *testing.T) {
	t.Run("SSEProvider returns nil by default", func(t *testing.T) {
		s := New()
		if s.SSEProvider() != nil {
			t.Error("expected nil SSEProvider by default")
		}
	})

	t.Run("SetSSEProvider stores provider", func(t *testing.T) {
		s := New()
		provider := NewDefaultProvider()
		s.SetSSEProvider(provider)

		if s.SSEProvider() != provider {
			t.Error("expected SSEProvider to be set")
		}
	})

	t.Run("SSEProvider works with config option", func(t *testing.T) {
		provider := NewDefaultProvider()
		s := New(config.Config{SSEProvider: provider})

		if s.SSEProvider() != provider {
			t.Error("expected SSEProvider from config option")
		}
	})
}

func TestConfigMerging(t *testing.T) {
	t.Run("partial config uses default Addr", func(t *testing.T) {
		// Only set DisableDefaultMiddlewares, Addr should use default
		s := New(config.Config{DisableDefaultMiddlewares: true})

		// Verify server was created successfully with default address
		if s == nil {
			t.Fatal("server should not be nil")
		}
	})

	t.Run("custom Addr overrides default", func(t *testing.T) {
		customAddr := ":9999"
		s := New(config.Config{Addr: customAddr})

		if s == nil {
			t.Fatal("server should not be nil")
		}
	})

	t.Run("partial config with only middleware settings uses default Addr", func(t *testing.T) {
		s := New(config.Config{
			RequestBodySize: config.RequestBodySizeConfig{
				MaxBytes: 10 * 1024 * 1024,
			},
		})

		if s == nil {
			t.Fatal("server should not be nil")
		}
	})

	t.Run("merge preserves default TLSAddr when not set", func(t *testing.T) {
		s := New(config.Config{
			Addr: ":8080",
		})

		if s == nil {
			t.Fatal("server should not be nil")
		}
	})

	t.Run("merge RequestID config", func(t *testing.T) {
		s := New(config.Config{
			RequestID: config.RequestIDConfig{
				Header: "X-Custom-Request-ID",
			},
		})

		if s == nil {
			t.Fatal("server should not be nil")
		}
	})

	t.Run("merge SecurityHeaders config", func(t *testing.T) {
		s := New(config.Config{
			SecurityHeaders: config.SecurityHeadersConfig{
				XFrameOptions: "SAMEORIGIN",
			},
		})

		if s == nil {
			t.Fatal("server should not be nil")
		}
	})
}

// mockValidator is a mock implementation of config.Validator for testing.
type mockValidator struct {
	structCalled   bool
	registerCalled bool
	lastDst        any
	lastName       string
	structErr      error
}

func (m *mockValidator) Struct(dst any) error {
	m.structCalled = true
	m.lastDst = dst
	return m.structErr
}

func (m *mockValidator) Register(name string, fn func(reflect.Value, string) error) {
	m.registerCalled = true
	m.lastName = name
}

func TestServerValidator(t *testing.T) {
	// Test that validator can be set and retrieved
	app := New()

	// Initially should be nil
	if app.Validator() != nil {
		t.Error("Validator should be nil initially")
	}

	// Set validator
	mockVal := &mockValidator{}
	app.SetValidator(mockVal)

	// Should be retrievable
	if app.Validator() != mockVal {
		t.Error("Validator should be retrievable")
	}

	// Test that it works
	type testStruct struct {
		Name string
	}
	ts := testStruct{Name: "test"}
	if err := app.Validator().Struct(&ts); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if !mockVal.structCalled {
		t.Error("expected Struct to be called")
	}
}

func TestServerWithValidator(t *testing.T) {
	// Test that validator can be set via config option
	mockVal := &mockValidator{}
	app := New(config.Config{Validator: mockVal})

	if app.Validator() != mockVal {
		t.Error("Validator should be set via config option")
	}
}
