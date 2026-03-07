package config

import (
	"context"
	"errors"
	"net"
	"net/http"
	"reflect"
	"testing"
	"time"
)

func TestDefaultConfigValues(t *testing.T) {
	cfg := DefaultConfig
	if cfg.Addr != "localhost:8080" {
		t.Errorf("expected Addr = localhost:8080, got %s", cfg.Addr)
	}
	if cfg.TLSAddr != "localhost:8443" {
		t.Errorf("expected TLSAddr = localhost:8443, got %s", cfg.TLSAddr)
	}
	if cfg.DisableDefaultMiddlewares {
		t.Error("expected DisableDefaultMiddlewares to be false")
	}
	if cfg.DefaultMiddlewares != nil {
		t.Error("expected DefaultMiddlewares to be nil")
	}
	if cfg.Logger != nil {
		t.Error("expected Logger to be nil")
	}
	if cfg.CertFile != "" {
		t.Errorf("expected CertFile to be empty, got %s", cfg.CertFile)
	}
	if cfg.KeyFile != "" {
		t.Errorf("expected KeyFile to be empty, got %s", cfg.KeyFile)
	}
	if cfg.AutocertManager != nil {
		t.Error("expected AutocertManager to be nil")
	}
	if cfg.Server != nil {
		t.Error("expected Server to be nil in default config")
	}
	if cfg.TLSServer != nil {
		t.Error("expected TLSServer to be nil in default config")
	}
	if cfg.Listener != nil {
		t.Error("expected Listener to be nil")
	}
	if cfg.TLSListener != nil {
		t.Error("expected TLSListener to be nil")
	}

	// Test middleware configs are initialized with defaults
	if cfg.Recover.StackSize != DefaultRecoverConfig.StackSize {
		t.Errorf("expected Recover.StackSize = %d, got %d", DefaultRecoverConfig.StackSize, cfg.Recover.StackSize)
	}
	if cfg.RequestBodySize.MaxBytes != DefaultRequestBodySizeConfig.MaxBytes {
		t.Errorf("expected RequestBodySize.MaxBytes = %d, got %d", DefaultRequestBodySizeConfig.MaxBytes, cfg.RequestBodySize.MaxBytes)
	}
	if cfg.RequestID.Header != DefaultRequestIDConfig.Header {
		t.Errorf("expected RequestID.Header = %s, got %s", DefaultRequestIDConfig.Header, cfg.RequestID.Header)
	}
}

func TestConfigZeroValues(t *testing.T) {
	var cfg Config
	if cfg.Addr != "" {
		t.Error("expected zero value for Addr to be empty string")
	}
	if cfg.TLSAddr != "" {
		t.Error("expected zero value for TLSAddr to be empty string")
	}
	if cfg.DisableDefaultMiddlewares != false {
		t.Error("expected zero value for DisableDefaultMiddlewares to be false")
	}
	if cfg.DefaultMiddlewares != nil {
		t.Error("expected zero value for DefaultMiddlewares to be nil")
	}
	if cfg.Logger != nil {
		t.Error("expected zero value for Logger to be nil")
	}
	if cfg.Server != nil {
		t.Error("expected zero value for Server to be nil")
	}
	if cfg.TLSServer != nil {
		t.Error("expected zero value for TLSServer to be nil")
	}
	if cfg.Listener != nil {
		t.Error("expected zero value for Listener to be nil")
	}
	if cfg.TLSListener != nil {
		t.Error("expected zero value for TLSListener to be nil")
	}
	if cfg.CertFile != "" {
		t.Error("expected zero value for CertFile to be empty string")
	}
	if cfg.KeyFile != "" {
		t.Error("expected zero value for KeyFile to be empty string")
	}
	if cfg.AutocertManager != nil {
		t.Error("expected zero value for AutocertManager to be nil")
	}
}

// mockHTTP3Server is a mock implementation of HTTP3Server for testing
type mockHTTP3Server struct{}

func (m *mockHTTP3Server) ListenAndServeTLS(certFile, keyFile string) error { return nil }
func (m *mockHTTP3Server) Shutdown(ctx context.Context) error               { return nil }
func (m *mockHTTP3Server) Close() error                                     { return nil }

func TestWithHTTP3Server(t *testing.T) {
	cfg := DefaultConfig
	mockServer := &mockHTTP3Server{}
	cfg.HTTP3Server = mockServer

	if cfg.HTTP3Server == nil {
		t.Error("expected HTTP3Server to be set")
	}
}

// mockWebTransportServer is a mock implementation of WebTransportServer for testing
type mockWebTransportServer struct{}

func (m *mockWebTransportServer) ListenAndServeTLS(certFile, keyFile string) error { return nil }
func (m *mockWebTransportServer) Close() error                                     { return nil }

func TestWithWebTransportServer(t *testing.T) {
	cfg := DefaultConfig
	mockServer := &mockWebTransportServer{}
	cfg.WebTransportServer = mockServer

	if cfg.WebTransportServer == nil {
		t.Error("expected WebTransportServer to be set")
	}
}

func TestShutdownHookWithError(t *testing.T) {
	expectedErr := errors.New("hook error")
	hook := func(ctx context.Context) error {
		return expectedErr
	}

	cfg := DefaultConfig
	cfg.ShutdownHooks = []ShutdownHookConfig{{Name: "error-hook", Hook: hook}}

	err := cfg.ShutdownHooks[0].Hook(context.Background())
	if !errors.Is(err, expectedErr) {
		t.Errorf("Expected error '%v', got '%v'", expectedErr, err)
	}
}

func TestShutdownHookContextCancellation(t *testing.T) {
	hook := func(ctx context.Context) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			return nil
		}
	}

	cfg := DefaultConfig
	cfg.ShutdownHooks = []ShutdownHookConfig{{Name: "ctx-hook", Hook: hook}}

	// Test with cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := cfg.ShutdownHooks[0].Hook(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Expected context.Canceled error, got %v", err)
	}
}

// mockValidator is a mock implementation of Validator for testing
type mockValidator struct {
	structCalled   bool
	registerCalled bool
	lastDst        any
	lastName       string
}

func (m *mockValidator) Struct(dst any) error {
	m.structCalled = true
	m.lastDst = dst
	return nil
}

func (m *mockValidator) Register(name string, fn func(reflect.Value, string) error) {
	m.registerCalled = true
	m.lastName = name
}

func TestValidator(t *testing.T) {
	cfg := DefaultConfig
	mockVal := &mockValidator{}
	cfg.Validator = mockVal

	if cfg.Validator == nil {
		t.Error("expected Validator to be set")
	}

	// Test that the mock works
	type testStruct struct {
		Name string
	}
	ts := testStruct{Name: "test"}
	if err := cfg.Validator.Struct(&ts); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if !mockVal.structCalled {
		t.Error("expected Struct to be called")
	}

	// Test Register
	cfg.Validator.Register("custom", func(v reflect.Value, s string) error { return nil })
	if !mockVal.registerCalled {
		t.Error("expected Register to be called")
	}
	if mockVal.lastName != "custom" {
		t.Errorf("expected name to be 'custom', got %s", mockVal.lastName)
	}
}

// mockWebSocketConn is a mock implementation of WebSocketConn for testing
type mockWebSocketConn struct{}

func (m *mockWebSocketConn) ReadMessage() (int, []byte, error)      { return 0, nil, nil }
func (m *mockWebSocketConn) WriteMessage(mt int, data []byte) error { return nil }
func (m *mockWebSocketConn) Close() error                           { return nil }
func (m *mockWebSocketConn) RemoteAddr() net.Addr                   { return nil }

// mockWebSocketUpgrader is a mock implementation of WebSocketUpgrader for testing
type mockWebSocketUpgrader struct{}

func (m *mockWebSocketUpgrader) Upgrade(w http.ResponseWriter, r *http.Request) (WebSocketConn, error) {
	return &mockWebSocketConn{}, nil
}

func TestWebSocketUpgrader(t *testing.T) {
	cfg := DefaultConfig
	mockUpgrader := &mockWebSocketUpgrader{}
	cfg.WebSocketUpgrader = mockUpgrader

	if cfg.WebSocketUpgrader == nil {
		t.Error("expected WebSocketUpgrader to be set")
	}
}

// mockSSEConnection is a mock implementation of SSEConnection for testing
type mockSSEConnection struct{}

func (m *mockSSEConnection) Send(event SSEEvent) error        { return nil }
func (m *mockSSEConnection) SendComment(comment string) error { return nil }
func (m *mockSSEConnection) Close() error                     { return nil }
func (m *mockSSEConnection) SetRetry(d time.Duration) error   { return nil }

// mockSSEProvider is a mock implementation of SSEProvider for testing
type mockSSEProvider struct{}

func (m *mockSSEProvider) NewSSE(w http.ResponseWriter, r *http.Request) (SSEConnection, error) {
	return &mockSSEConnection{}, nil
}

func TestSSEProvider(t *testing.T) {
	cfg := DefaultConfig
	mockProvider := &mockSSEProvider{}
	cfg.SSEProvider = mockProvider

	if cfg.SSEProvider == nil {
		t.Error("expected SSEProvider to be set")
	}
}

func TestCloseError(t *testing.T) {
	tests := []struct {
		name string
		err  CloseError
		want string
	}{
		{
			name: "with reason",
			err:  CloseError{Code: 1000, Reason: "normal closure"},
			want: "websocket: close 1000 normal closure",
		},
		{
			name: "without reason",
			err:  CloseError{Code: 1001},
			want: "websocket: close 1001",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			if got != tt.want {
				t.Errorf("CloseError.Error() = %q, want %q", got, tt.want)
			}
		})
	}
}
