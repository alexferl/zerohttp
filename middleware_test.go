package zerohttp

import (
	"context"
	"net/http"
	"testing"

	"github.com/alexferl/zerohttp/log"
	"github.com/alexferl/zerohttp/zhtest"
)

type mockLogger struct {
	debugLogs   []string
	infoLogs    []string
	errorLogs   []string
	errorFields [][]log.Field
}

func (m *mockLogger) Debug(msg string, fields ...log.Field) { m.debugLogs = append(m.debugLogs, msg) }
func (m *mockLogger) Info(msg string, fields ...log.Field)  { m.infoLogs = append(m.infoLogs, msg) }
func (m *mockLogger) Warn(msg string, fields ...log.Field)  {}
func (m *mockLogger) Error(msg string, fields ...log.Field) {
	m.errorLogs = append(m.errorLogs, msg)
	m.errorFields = append(m.errorFields, fields)
}
func (m *mockLogger) Panic(msg string, fields ...log.Field)      {}
func (m *mockLogger) Fatal(msg string, fields ...log.Field)      {}
func (m *mockLogger) WithFields(fields ...log.Field) log.Logger  { return m }
func (m *mockLogger) WithContext(ctx context.Context) log.Logger { return m }

func TestDefaultMiddlewares(t *testing.T) {
	cfg := DefaultConfig
	logger := &mockLogger{}

	middlewares := DefaultMiddlewares(cfg, logger)

	expectedCount := 5
	if len(middlewares) != expectedCount {
		t.Errorf("Expected %d middlewares, got %d", expectedCount, len(middlewares))
	}

	for i, middleware := range middlewares {
		if middleware == nil {
			t.Errorf("Middleware at index %d is nil", i)
		}
	}

	baseHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	var wrappedHandler http.Handler = baseHandler
	for i := len(middlewares) - 1; i >= 0; i-- {
		wrappedHandler = middlewares[i](wrappedHandler)
	}

	if wrappedHandler == nil {
		t.Error("Wrapped handler should not be nil")
	}

	req := zhtest.NewRequest(http.MethodGet, "/").Build()

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Middleware chain panicked: %v", r)
		}
	}()

	w := zhtest.Serve(wrappedHandler, req)

	if w.Code == 0 {
		t.Error("Expected response code to be set")
	}
}

func TestDefaultMiddlewares_NilInputs(t *testing.T) {
	t.Run("nil logger", func(t *testing.T) {
		cfg := Config{}

		defer func() {
			if r := recover(); r != nil {
				t.Errorf("DefaultMiddlewares panicked with nil logger: %v", r)
			}
		}()

		middlewares := DefaultMiddlewares(cfg, nil)

		if len(middlewares) == 0 {
			t.Error("Expected middlewares to be returned even with nil logger")
		}
	})
}
