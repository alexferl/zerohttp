package zerohttp

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/log"
)

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

	req := httptest.NewRequest("GET", "http://example.com/path?query=value", nil)
	req.Host = "example.com"
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusMovedPermanently {
		t.Errorf("Expected status %d, got %d", http.StatusMovedPermanently, w.Code)
	}

	// http.Redirect sets the Location header
	location := w.Header().Get("Location")
	expectedLocation := "https://example.com/path?query=value"
	if location != expectedLocation {
		t.Errorf("Expected Location '%s', got '%s'", expectedLocation, location)
	}
}

func TestServer_ListenAndServe_NoServer(t *testing.T) {
	mockLogger := &mockServerLogger{}
	server := New(config.WithLogger(mockLogger))
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

	err := server.StartAutoTLS("example.com")
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

func TestNewAutocertManager(t *testing.T) {
	cacheDir := "/tmp/autocert"
	hosts := []string{"example.com", "www.example.com"}

	manager := NewAutocertManager(cacheDir, hosts...)

	if manager == nil {
		t.Fatal("Expected autocert manager to be created")
	}

	if manager.Cache == nil {
		t.Error("Expected cache to be set")
	}

	if manager.Prompt == nil {
		t.Error("Expected prompt to be set")
	}

	if manager.HostPolicy == nil {
		t.Error("Expected host policy to be set")
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
