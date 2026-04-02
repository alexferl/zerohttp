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

	zhtest.AssertEqual(t, len(middlewares), 5)

	for _, middleware := range middlewares {
		zhtest.AssertNotNil(t, middleware)
	}

	baseHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	var wrappedHandler http.Handler = baseHandler
	for i := len(middlewares) - 1; i >= 0; i-- {
		wrappedHandler = middlewares[i](wrappedHandler)
	}

	zhtest.AssertNotNil(t, wrappedHandler)

	req := zhtest.NewRequest(http.MethodGet, "/").Build()

	zhtest.AssertNoPanic(t, func() {
		w := zhtest.Serve(wrappedHandler, req)
		zhtest.AssertTrue(t, w.Code != 0)
	})
}

func TestDefaultMiddlewares_NilInputs(t *testing.T) {
	t.Run("nil logger", func(t *testing.T) {
		cfg := Config{}

		var middlewares []MiddlewareFunc
		zhtest.AssertNoPanic(t, func() {
			middlewares = DefaultMiddlewares(cfg, nil)
		})
		zhtest.AssertNotEmpty(t, middlewares)
	})
}
