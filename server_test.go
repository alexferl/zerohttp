package zerohttp

import (
	"context"
	"errors"
	"net"
	"net/http"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

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

	zhtest.AssertNotNil(t, server)
	zhtest.AssertNotNil(t, server.Router)
	zhtest.AssertNotNil(t, server.logger)
}

func TestNew_MiddlewareScenarios(t *testing.T) {
	// Test with DisableDefaultMiddlewares set to true and custom middlewares
	customMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
		})
	}

	server := New(Config{
		DisableDefaultMiddlewares: true,
		DefaultMiddlewares:        []func(http.Handler) http.Handler{customMiddleware},
	})

	zhtest.AssertNotNil(t, server)

	// Test with custom default middlewares combined with defaults
	server2 := New(Config{
		DefaultMiddlewares: []func(http.Handler) http.Handler{customMiddleware},
	})

	zhtest.AssertNotNil(t, server2)
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
	zhtest.AssertEqual(t, ":8080", addr)

	// Set actual listener (takes precedence)
	listener, _ := net.Listen("tcp", "127.0.0.1:0")
	server.listener = listener
	defer func() {
		zhtest.AssertNoError(t, listener.Close())
	}()

	addr = server.ListenerAddr()
	zhtest.AssertNotEmpty(t, addr)
	zhtest.AssertTrue(t, strings.Contains(addr, "127.0.0.1"))
}

func TestServer_ListenAndServe_NoServer(t *testing.T) {
	mockLogger := &mockServerLogger{}
	server := New(Config{Logger: mockLogger})
	server.server = nil

	err := server.ListenAndServe()
	zhtest.AssertNoError(t, err)

	// Should log debug message about skipping
	found := false
	for _, entry := range mockLogger.logs {
		if entry.level == "debug" && strings.Contains(entry.message, "HTTP server not configured") {
			found = true
			break
		}
	}
	zhtest.AssertTrue(t, found)
}

func TestServer_Close_WithServers(t *testing.T) {
	// Test Close() with servers configured (new behavior - uses server.Close() not listener.Close())
	server := New()

	// Create http.Servers directly (simulating Start() path where listener is nil)
	server.server = &http.Server{Addr: "127.0.0.1:0"}
	server.tlsServer = &http.Server{Addr: "127.0.0.1:0"}

	// Close should work even with nil listeners (uses server.Close() internally)
	err := server.Close()
	zhtest.AssertNoError(t, err)

	// Calling Close again should handle closed servers gracefully
	_ = server.Close()
	// May return error from already closed servers, but shouldn't crash
}

func TestServer_Start_NoServersConfigured(t *testing.T) {
	// When no servers are configured, Start() should return immediately without hanging
	server := New()
	server.server = nil
	server.metricsServer = nil
	// tlsServer will be nil since we didn't configure TLS

	done := make(chan error, 1)
	go func() {
		done <- server.Start()
	}()

	select {
	case err := <-done:
		zhtest.AssertNoError(t, err)
	case <-time.After(time.Second):
		zhtest.AssertFail(t, "Start() hung when no servers configured - expected immediate return")
	}
}

func TestServer_Shutdown_NoServers(t *testing.T) {
	server := New()
	server.server = nil
	server.tlsServer = nil

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err := server.Shutdown(ctx)
	zhtest.AssertNoError(t, err)
}

func TestServer_Shutdown_AfterStart(t *testing.T) {
	// Test that Shutdown works when server was started via Start()
	// (where listener fields are nil)
	server := New()
	server.server = &http.Server{Addr: "127.0.0.1:0"}
	// Note: server.listener is nil - Start() creates its own listener

	done := make(chan error, 1)
	go func() {
		done <- server.Start()
	}()

	// Give server time to start
	time.Sleep(50 * time.Millisecond)

	// Shutdown should work even though server.listener is nil
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err := server.Shutdown(ctx)
	zhtest.AssertNoError(t, err)

	// Wait for Start() to return
	select {
	case <-done:
		// Expected
	case <-time.After(time.Second):
		zhtest.AssertFail(t, "timeout waiting for Start() to return after shutdown")
	}
}

func TestServer_Close_AfterStart(t *testing.T) {
	// Test that Close works when server was started via Start()
	// (where listener fields are nil)
	server := New()
	server.server = &http.Server{Addr: "127.0.0.1:0"}
	// Note: server.listener is nil - Start() creates its own listener

	done := make(chan error, 1)
	go func() {
		done <- server.Start()
	}()

	// Give server time to start
	time.Sleep(50 * time.Millisecond)

	// Close should work even though server.listener is nil
	err := server.Close()
	zhtest.AssertNoError(t, err)

	// Wait for Start() to return
	select {
	case <-done:
		// Expected
	case <-time.After(time.Second):
		zhtest.AssertFail(t, "timeout waiting for Start() to return after close")
	}
}

func TestServer_Start_MultipleServers_CleanShutdown(t *testing.T) {
	// Test Start() with multiple servers and verify clean shutdown returns nil
	server := New()
	server.server = &http.Server{Addr: "127.0.0.1:0"}
	server.metricsServer = &http.Server{Addr: "127.0.0.1:0"}

	done := make(chan error, 1)
	go func() {
		done <- server.Start()
	}()

	// Give servers time to start
	time.Sleep(50 * time.Millisecond)

	// Clean shutdown should make Start() return nil
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err := server.Shutdown(ctx)
	zhtest.AssertNoError(t, err)

	select {
	case startErr := <-done:
		zhtest.AssertNoError(t, startErr)
	case <-time.After(time.Second):
		zhtest.AssertFail(t, "timeout waiting for Start() to return after shutdown")
	}
}

func TestServer_Shutdown_DrainsHookErrors(t *testing.T) {
	// Test that Shutdown properly drains hookErrCh and logs errors
	server := New()
	server.server = &http.Server{Addr: "127.0.0.1:0"}

	// Register shutdown hooks that return errors
	hook1Called := false
	hook2Called := false
	server.RegisterShutdownHook("error-hook-1", func(ctx context.Context) error {
		hook1Called = true
		return errors.New("hook 1 error")
	})
	server.RegisterShutdownHook("error-hook-2", func(ctx context.Context) error {
		hook2Called = true
		return errors.New("hook 2 error")
	})

	done := make(chan error, 1)
	go func() {
		done <- server.Start()
	}()

	time.Sleep(50 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Shutdown should complete without deadlock even with hook errors
	err := server.Shutdown(ctx)
	zhtest.AssertNoError(t, err)

	// Wait for Start() to complete
	select {
	case <-done:
		// Expected
	case <-time.After(time.Second):
		zhtest.AssertFail(t, "timeout waiting for Start() to return")
	}

	// Verify hooks were called
	zhtest.AssertTrue(t, hook1Called)
	zhtest.AssertTrue(t, hook2Called)
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

func TestServer_ListenAndServe_WithListener(t *testing.T) {
	server := New()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	zhtest.AssertNoError(t, err)
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
	zhtest.AssertNoError(t, err)

	// Wait for ListenAndServe to return
	select {
	case err := <-done:
		// ErrServerClosed is expected after shutdown
		isExpectedErr := err == nil || errors.Is(err, http.ErrServerClosed)
		zhtest.AssertTrue(t, isExpectedErr)
	case <-time.After(time.Second):
		zhtest.AssertFail(t, "timeout waiting for ListenAndServe to return")
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
	zhtest.AssertNotEmpty(t, server.ListenerAddr())

	// Shutdown to stop the server
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_ = server.Shutdown(ctx)

	// Wait for ListenAndServe to return
	select {
	case <-done:
		// Expected
	case <-time.After(time.Second):
		zhtest.AssertFail(t, "timeout waiting for ListenAndServe to return")
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
	server := New(Config{Logger: mockLogger})

	zhtest.AssertNotNil(t, server.Logger())
}

func TestServer_ListenerAddr_Empty(t *testing.T) {
	server := New()
	server.server = nil
	server.listener = nil

	addr := server.ListenerAddr()
	zhtest.AssertEmpty(t, addr)
}

// mockValidator is a mock implementation of Validator for testing.
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
	zhtest.AssertNil(t, app.Validator())

	// Set validator
	mockVal := &mockValidator{}
	app.SetValidator(mockVal)

	// Should be retrievable
	zhtest.AssertEqual(t, mockVal, app.Validator())

	// Test that it works
	type testStruct struct {
		Name string
	}
	ts := testStruct{Name: "test"}
	zhtest.AssertNoError(t, app.Validator().Struct(&ts))
	zhtest.AssertTrue(t, mockVal.structCalled)
}

func TestServerWithValidator(t *testing.T) {
	// Test that validator can be set via config option
	mockVal := &mockValidator{}
	app := New(Config{Validator: mockVal})

	zhtest.AssertEqual(t, mockVal, app.Validator())
}

func TestDefaultTimeoutConstants(t *testing.T) {
	zhtest.AssertEqual(t, 10*time.Second, DefaultReadTimeout)
	zhtest.AssertEqual(t, 15*time.Second, DefaultWriteTimeout)
	zhtest.AssertEqual(t, 60*time.Second, DefaultIdleTimeout)
}

func TestServer_DefaultTimeoutsApplied(t *testing.T) {
	server := New()

	zhtest.AssertNotNil(t, server.server)
	zhtest.AssertEqual(t, DefaultReadTimeout, server.server.ReadTimeout)
	zhtest.AssertEqual(t, DefaultWriteTimeout, server.server.WriteTimeout)
	zhtest.AssertEqual(t, DefaultIdleTimeout, server.server.IdleTimeout)
}
