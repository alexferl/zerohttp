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
	"github.com/alexferl/zerohttp/zhtest"
)

func TestDefaultConfigValues(t *testing.T) {
	cfg := DefaultConfig
	zhtest.AssertEqual(t, cfg.Addr, "localhost:8080")
	zhtest.AssertEqual(t, cfg.TLS.Addr, "localhost:8443")
	zhtest.AssertFalse(t, cfg.DisableDefaultMiddlewares)
	zhtest.AssertNil(t, cfg.DefaultMiddlewares)
	zhtest.AssertNil(t, cfg.Logger)
	zhtest.AssertEqual(t, cfg.TLS.CertFile, "")
	zhtest.AssertEqual(t, cfg.TLS.KeyFile, "")
	zhtest.AssertNil(t, cfg.Extensions.AutocertManager)
	zhtest.AssertNil(t, cfg.Server)
	zhtest.AssertNil(t, cfg.TLS.Server)
	zhtest.AssertNil(t, cfg.Listener)
	zhtest.AssertNil(t, cfg.TLS.Listener)

	// Test middleware configs are initialized with defaults
	zhtest.AssertEqual(t, cfg.Recover.StackSize, recover.DefaultConfig.StackSize)
	zhtest.AssertEqual(t, cfg.RequestBodySize.MaxBytes, requestbodysize.DefaultConfig.MaxBytes)
	zhtest.AssertEqual(t, cfg.RequestID.Header, requestid.DefaultConfig.Header)
}

func TestConfigZeroValues(t *testing.T) {
	var cfg Config
	zhtest.AssertEqual(t, cfg.Addr, "")
	zhtest.AssertEqual(t, cfg.TLS.Addr, "")
	zhtest.AssertFalse(t, cfg.DisableDefaultMiddlewares)
	zhtest.AssertNil(t, cfg.DefaultMiddlewares)
	zhtest.AssertNil(t, cfg.Logger)
	zhtest.AssertNil(t, cfg.Server)
	zhtest.AssertNil(t, cfg.TLS.Server)
	zhtest.AssertNil(t, cfg.Listener)
	zhtest.AssertNil(t, cfg.TLS.Listener)
	zhtest.AssertEqual(t, cfg.TLS.CertFile, "")
	zhtest.AssertEqual(t, cfg.TLS.KeyFile, "")
	zhtest.AssertNil(t, cfg.Extensions.AutocertManager)
}

func TestWithWebTransportServer(t *testing.T) {
	cfg := DefaultConfig
	mockServer := &mockWebTransportServer{}
	cfg.Extensions.WebTransportServer = mockServer

	zhtest.AssertNotNil(t, cfg.Extensions.WebTransportServer)
}

func TestShutdownHookWithError(t *testing.T) {
	expectedErr := errors.New("hook error")
	hook := func(ctx context.Context) error {
		return expectedErr
	}

	cfg := DefaultConfig
	cfg.Lifecycle.ShutdownHooks = []ShutdownHookConfig{{Name: "error-hook", Hook: hook}}

	err := cfg.Lifecycle.ShutdownHooks[0].Hook(context.Background())
	zhtest.AssertErrorIs(t, err, expectedErr)
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
	zhtest.AssertErrorIs(t, err, context.Canceled)
}

func TestValidator(t *testing.T) {
	cfg := DefaultConfig
	mockVal := &mockValidator{}
	cfg.Validator = mockVal

	zhtest.AssertNotNil(t, cfg.Validator)

	// Test that the mock works
	type testStruct struct {
		Name string
	}
	ts := testStruct{Name: "test"}
	zhtest.AssertNoError(t, cfg.Validator.Struct(&ts))
	zhtest.AssertTrue(t, mockVal.structCalled)

	// Test Register
	cfg.Validator.Register("custom", func(v reflect.Value, s string) error { return nil })
	zhtest.AssertTrue(t, mockVal.registerCalled)
	zhtest.AssertEqual(t, mockVal.lastName, "custom")
}

func TestWebSocketUpgrader(t *testing.T) {
	cfg := DefaultConfig
	mockUpgrader := &mockWebSocketUpgrader{}
	cfg.Extensions.WebSocketUpgrader = mockUpgrader

	zhtest.AssertNotNil(t, cfg.Extensions.WebSocketUpgrader)
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

	zhtest.AssertNotNil(t, cfg.Extensions.SSEProvider)
}
