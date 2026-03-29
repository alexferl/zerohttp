package healthcheck

import (
	"net/http"
	"testing"

	zh "github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp/zhtest"
)

func TestDefaultHealthEndpoints(t *testing.T) {
	app := zh.New()
	New(app, DefaultConfig)

	endpoints := []string{"/livez", "/readyz", "/startupz"}
	for _, endpoint := range endpoints {
		t.Run(endpoint, func(t *testing.T) {
			req := zhtest.NewRequest(http.MethodGet, endpoint).Build()
			w := zhtest.Serve(app, req)
			zhtest.AssertWith(t, w).Status(http.StatusOK).Body("ok")
		})
	}
}

func TestCustomEndpoints(t *testing.T) {
	app := zh.New()

	cfg := DefaultConfig
	cfg.LivenessEndpoint = "/health/live"
	cfg.ReadinessEndpoint = "/health/ready"
	cfg.StartupEndpoint = "/health/startup"
	New(app, cfg)

	endpoints := []string{"/health/live", "/health/ready", "/health/startup"}
	for _, endpoint := range endpoints {
		t.Run(endpoint, func(t *testing.T) {
			req := zhtest.NewRequest(http.MethodGet, endpoint).Build()
			w := zhtest.Serve(app, req)
			zhtest.AssertWith(t, w).Status(http.StatusOK)
		})
	}
}

func TestCustomHandlers(t *testing.T) {
	var livenessHandlerCalled bool
	var readinessHandlerCalled bool
	var startupHandlerCalled bool

	livenessHandler := func(w http.ResponseWriter, r *http.Request) error {
		livenessHandlerCalled = true
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("alive"))
		return err
	}

	readinessHandler := func(w http.ResponseWriter, r *http.Request) error {
		readinessHandlerCalled = true
		w.WriteHeader(http.StatusServiceUnavailable)
		_, err := w.Write([]byte("not ready"))
		return err
	}

	startupHandler := func(w http.ResponseWriter, r *http.Request) error {
		startupHandlerCalled = true
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("started"))
		return err
	}

	app := zh.New()

	cfg := DefaultConfig
	cfg.LivenessHandler = livenessHandler
	cfg.ReadinessHandler = readinessHandler
	cfg.StartupHandler = startupHandler
	New(app, cfg)

	t.Run("liveness", func(t *testing.T) {
		req := zhtest.NewRequest(http.MethodGet, "/livez").Build()
		w := zhtest.Serve(app, req)
		zhtest.AssertWith(t, w).Status(http.StatusOK).Body("alive")

		zhtest.AssertTrue(t, livenessHandlerCalled)
	})

	t.Run("readiness", func(t *testing.T) {
		req := zhtest.NewRequest(http.MethodGet, "/readyz").Build()
		w := zhtest.Serve(app, req)
		zhtest.AssertWith(t, w).Status(http.StatusServiceUnavailable).Body("not ready")

		zhtest.AssertTrue(t, readinessHandlerCalled)
	})

	t.Run("startup", func(t *testing.T) {
		req := zhtest.NewRequest(http.MethodGet, "/startupz").Build()
		w := zhtest.Serve(app, req)
		zhtest.AssertWith(t, w).Status(http.StatusOK).Body("started")

		zhtest.AssertTrue(t, startupHandlerCalled)
	})
}

func TestMixedOptions(t *testing.T) {
	var customHandlerCalled bool

	customHandler := func(w http.ResponseWriter, r *http.Request) error {
		customHandlerCalled = true
		w.WriteHeader(http.StatusTeapot)
		_, err := w.Write([]byte("custom"))
		return err
	}

	app := zh.New()

	cfg := DefaultConfig
	cfg.LivenessEndpoint = "/custom-livez"
	cfg.ReadinessHandler = customHandler
	New(app, cfg)

	t.Run("custom endpoint", func(t *testing.T) {
		req := zhtest.NewRequest(http.MethodGet, "/custom-livez").Build()
		w := zhtest.Serve(app, req)
		zhtest.AssertWith(t, w).Status(http.StatusOK).Body("ok")
	})

	t.Run("custom handler", func(t *testing.T) {
		req := zhtest.NewRequest(http.MethodGet, "/readyz").Build()
		w := zhtest.Serve(app, req)
		zhtest.AssertWith(t, w).Status(http.StatusTeapot).Body("custom")

		zhtest.AssertTrue(t, customHandlerCalled)
	})
}

func TestDefaultConfig(t *testing.T) {
	zhtest.AssertEqual(t, "/livez", DefaultConfig.LivenessEndpoint)
	zhtest.AssertEqual(t, "/readyz", DefaultConfig.ReadinessEndpoint)
	zhtest.AssertEqual(t, "/startupz", DefaultConfig.StartupEndpoint)
	zhtest.AssertNotNil(t, DefaultConfig.LivenessHandler)
	zhtest.AssertNotNil(t, DefaultConfig.ReadinessHandler)
	zhtest.AssertNotNil(t, DefaultConfig.StartupHandler)
}

func TestNoConfig(t *testing.T) {
	// Test calling New without any config (uses defaults)
	app := zh.New()
	New(app)

	endpoints := []string{"/livez", "/readyz", "/startupz"}
	for _, endpoint := range endpoints {
		t.Run(endpoint, func(t *testing.T) {
			req := zhtest.NewRequest(http.MethodGet, endpoint).Build()
			w := zhtest.Serve(app, req)
			zhtest.AssertWith(t, w).Status(http.StatusOK).Body("ok")
		})
	}
}

func TestPartialConfig(t *testing.T) {
	// Test partial config merging with defaults
	app := zh.New()

	// Only override liveness endpoint, rest should use defaults
	New(app, Config{
		LivenessEndpoint: "/health/live",
	})

	// Custom endpoint should work
	t.Run("custom liveness", func(t *testing.T) {
		req := zhtest.NewRequest(http.MethodGet, "/health/live").Build()
		w := zhtest.Serve(app, req)
		zhtest.AssertWith(t, w).Status(http.StatusOK).Body("ok")
	})

	// Default endpoints should still work
	t.Run("default readiness", func(t *testing.T) {
		req := zhtest.NewRequest(http.MethodGet, "/readyz").Build()
		w := zhtest.Serve(app, req)
		zhtest.AssertWith(t, w).Status(http.StatusOK).Body("ok")
	})

	t.Run("default startup", func(t *testing.T) {
		req := zhtest.NewRequest(http.MethodGet, "/startupz").Build()
		w := zhtest.Serve(app, req)
		zhtest.AssertWith(t, w).Status(http.StatusOK).Body("ok")
	})
}
