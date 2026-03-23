package zerohttp

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/alexferl/zerohttp/middleware/recover"
	"github.com/alexferl/zerohttp/middleware/requestbodysize"
	"github.com/alexferl/zerohttp/middleware/requestid"
	"github.com/alexferl/zerohttp/sse"
)

func TestDefaultConfigValues(t *testing.T) {
	cfg := DefaultConfig
	if cfg.Addr != "localhost:8080" {
		t.Errorf("expected Addr = localhost:8080, got %s", cfg.Addr)
	}
	if cfg.TLS.Addr != "localhost:8443" {
		t.Errorf("expected Addr = localhost:8443, got %s", cfg.TLS.Addr)
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
	if cfg.TLS.CertFile != "" {
		t.Errorf("expected CertFile to be empty, got %s", cfg.TLS.CertFile)
	}
	if cfg.TLS.KeyFile != "" {
		t.Errorf("expected KeyFile to be empty, got %s", cfg.TLS.KeyFile)
	}
	if cfg.Extensions.AutocertManager != nil {
		t.Error("expected AutocertManager to be nil")
	}
	if cfg.Server != nil {
		t.Error("expected Server to be nil in default config")
	}
	if cfg.TLS.Server != nil {
		t.Error("expected Server to be nil in default config")
	}
	if cfg.Listener != nil {
		t.Error("expected Listener to be nil")
	}
	if cfg.TLS.Listener != nil {
		t.Error("expected Listener to be nil")
	}

	// Test middleware configs are initialized with defaults
	if cfg.Recover.StackSize != recover.DefaultConfig.StackSize {
		t.Errorf("expected Recover.StackSize = %d, got %d", recover.DefaultConfig.StackSize, cfg.Recover.StackSize)
	}
	if cfg.RequestBodySize.MaxBytes != requestbodysize.DefaultConfig.MaxBytes {
		t.Errorf("expected RequestBodySize.MaxBytes = %d, got %d", requestbodysize.DefaultConfig.MaxBytes, cfg.RequestBodySize.MaxBytes)
	}
	if cfg.RequestID.Header != requestid.DefaultConfig.Header {
		t.Errorf("expected RequestID.Header = %s, got %s", requestid.DefaultConfig.Header, cfg.RequestID.Header)
	}
}

func TestConfigZeroValues(t *testing.T) {
	var cfg Config
	if cfg.Addr != "" {
		t.Error("expected zero value for Addr to be empty string")
	}
	if cfg.TLS.Addr != "" {
		t.Error("expected zero value for Addr to be empty string")
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
	if cfg.TLS.Server != nil {
		t.Error("expected zero value for Server to be nil")
	}
	if cfg.Listener != nil {
		t.Error("expected zero value for Listener to be nil")
	}
	if cfg.TLS.Listener != nil {
		t.Error("expected zero value for Listener to be nil")
	}
	if cfg.TLS.CertFile != "" {
		t.Error("expected zero value for CertFile to be empty string")
	}
	if cfg.TLS.KeyFile != "" {
		t.Error("expected zero value for KeyFile to be empty string")
	}
	if cfg.Extensions.AutocertManager != nil {
		t.Error("expected zero value for AutocertManager to be nil")
	}
}

// // mockHTTP3Server is a mock implementation of HTTP3Server for testing
// type mockHTTP3Server struct{}
//
// func (m *mockHTTP3Server) ListenAndServeTLS(certFile, keyFile string) error { return nil }
// func (m *mockHTTP3Server) Shutdown(ctx context.Context) error               { return nil }
// func (m *mockHTTP3Server) Close() error                                     { return nil }
//
//	func TestWithHTTP3Server(t *testing.T) {
//		cfg := DefaultConfig
//		mockServer := &mockHTTP3Server{}
//		cfg.Extensions.HTTP3Server = mockServer
//
//		if cfg.Extensions.HTTP3Server == nil {
//			t.Error("expected HTTP3Server to be set")
//		}
//	}
//
// // mockWebTransportServer is a mock implementation of WebTransportServer for testing
// type mockWebTransportServer struct{}
//
// func (m *mockWebTransportServer) ListenAndServeTLS(certFile, keyFile string) error { return nil }
// func (m *mockWebTransportServer) Close() error                                     { return nil }
func TestWithWebTransportServer(t *testing.T) {
	cfg := DefaultConfig
	mockServer := &mockWebTransportServer{}
	cfg.Extensions.WebTransportServer = mockServer

	if cfg.Extensions.WebTransportServer == nil {
		t.Error("expected WebTransportServer to be set")
	}
}

func TestShutdownHookWithError(t *testing.T) {
	expectedErr := errors.New("hook error")
	hook := func(ctx context.Context) error {
		return expectedErr
	}

	cfg := DefaultConfig
	cfg.Lifecycle.ShutdownHooks = []ShutdownHookConfig{{Name: "error-hook", Hook: hook}}

	err := cfg.Lifecycle.ShutdownHooks[0].Hook(context.Background())
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
	cfg.Lifecycle.ShutdownHooks = []ShutdownHookConfig{{Name: "ctx-hook", Hook: hook}}

	// Test with cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := cfg.Lifecycle.ShutdownHooks[0].Hook(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Expected context.Canceled error, got %v", err)
	}
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

func TestWebSocketUpgrader(t *testing.T) {
	cfg := DefaultConfig
	mockUpgrader := &mockWebSocketUpgrader{}
	cfg.Extensions.WebSocketUpgrader = mockUpgrader

	if cfg.Extensions.WebSocketUpgrader == nil {
		t.Error("expected WebSocketUpgrader to be set")
	}
}

// mockSSEConnection is a mock implementation of SSEConnection for testing
type mockSSEConnection struct{}

func (m *mockSSEConnection) Send(event sse.Event) error       { return nil }
func (m *mockSSEConnection) SendComment(comment string) error { return nil }
func (m *mockSSEConnection) Close() error                     { return nil }
func (m *mockSSEConnection) SetRetry(d time.Duration) error   { return nil }

// mockSSEProvider is a mock implementation of SSEProvider for testing
type mockSSEProvider struct{}

func (m *mockSSEProvider) New(w http.ResponseWriter, r *http.Request) (sse.Connection, error) {
	return &mockSSEConnection{}, nil
}

func TestSSEProvider(t *testing.T) {
	cfg := DefaultConfig
	mockProvider := &mockSSEProvider{}
	cfg.Extensions.SSEProvider = mockProvider

	if cfg.Extensions.SSEProvider == nil {
		t.Error("expected SSEProvider to be set")
	}
}
