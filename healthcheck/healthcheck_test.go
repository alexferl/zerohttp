package healthcheck

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	zh "github.com/alexferl/zerohttp"
)

func TestDefaultHealthEndpoints(t *testing.T) {
	app := zh.New()
	New(app)

	server := httptest.NewServer(app)
	defer server.Close()

	endpoints := []string{"/livez", "/readyz", "/startupz"}
	for _, endpoint := range endpoints {
		t.Run(endpoint, func(t *testing.T) {
			resp, err := http.Get(server.URL + endpoint)
			if err != nil {
				t.Fatalf("Failed to get %s: %v", endpoint, err)
			}
			t.Cleanup(func() {
				if err := resp.Body.Close(); err != nil {
					t.Logf("failed to close body: %v", err)
				}
			})

			if resp.StatusCode != http.StatusOK {
				t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
			}

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("Failed to read body: %v", err)
			}

			if string(body) != "ok" {
				t.Errorf("Expected body 'ok', got '%s'", string(body))
			}
		})
	}
}

func TestCustomEndpoints(t *testing.T) {
	app := zh.New()

	New(app,
		WithLivenessEndpoint("/health/live"),
		WithReadinessEndpoint("/health/ready"),
		WithStartupEndpoint("/health/startup"),
	)

	server := httptest.NewServer(app)
	defer server.Close()

	endpoints := []string{"/health/live", "/health/ready", "/health/startup"}
	for _, endpoint := range endpoints {
		t.Run(endpoint, func(t *testing.T) {
			resp, err := http.Get(server.URL + endpoint)
			if err != nil {
				t.Fatalf("Failed to get %s: %v", endpoint, err)
			}
			t.Cleanup(func() {
				if err := resp.Body.Close(); err != nil {
					t.Logf("failed to close body: %v", err)
				}
			})

			if resp.StatusCode != http.StatusOK {
				t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
			}
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

	New(app,
		WithLivenessHandler(livenessHandler),
		WithReadinessHandler(readinessHandler),
		WithStartupHandler(startupHandler),
	)

	server := httptest.NewServer(app)
	defer server.Close()

	t.Run("liveness", func(t *testing.T) {
		resp, err := http.Get(server.URL + "/livez")
		if err != nil {
			t.Fatalf("Failed to get /livez: %v", err)
		}
		t.Cleanup(func() {
			if err := resp.Body.Close(); err != nil {
				t.Logf("failed to close body: %v", err)
			}
		})

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("Failed to read body: %v", err)
		}

		if string(body) != "alive" {
			t.Errorf("Expected body 'alive', got '%s'", string(body))
		}

		if !livenessHandlerCalled {
			t.Error("Liveness handler was not called")
		}
	})

	t.Run("readiness", func(t *testing.T) {
		resp, err := http.Get(server.URL + "/readyz")
		if err != nil {
			t.Fatalf("Failed to get /readyz: %v", err)
		}
		t.Cleanup(func() {
			if err := resp.Body.Close(); err != nil {
				t.Logf("failed to close body: %v", err)
			}
		})

		if resp.StatusCode != http.StatusServiceUnavailable {
			t.Errorf("Expected status %d, got %d", http.StatusServiceUnavailable, resp.StatusCode)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("Failed to read body: %v", err)
		}

		if string(body) != "not ready" {
			t.Errorf("Expected body 'not ready', got '%s'", string(body))
		}

		if !readinessHandlerCalled {
			t.Error("Readiness handler was not called")
		}
	})

	t.Run("startup", func(t *testing.T) {
		resp, err := http.Get(server.URL + "/startupz")
		if err != nil {
			t.Fatalf("Failed to get /startupz: %v", err)
		}
		t.Cleanup(func() {
			if err := resp.Body.Close(); err != nil {
				t.Logf("failed to close body: %v", err)
			}
		})

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("Failed to read body: %v", err)
		}

		if string(body) != "started" {
			t.Errorf("Expected body 'started', got '%s'", string(body))
		}

		if !startupHandlerCalled {
			t.Error("Startup handler was not called")
		}
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

	New(app,
		WithLivenessEndpoint("/custom-livez"),
		WithReadinessHandler(customHandler),
	)

	server := httptest.NewServer(app)
	defer server.Close()

	t.Run("custom endpoint", func(t *testing.T) {
		resp, err := http.Get(server.URL + "/custom-livez")
		if err != nil {
			t.Fatalf("Failed to get /custom-livez: %v", err)
		}
		t.Cleanup(func() {
			if err := resp.Body.Close(); err != nil {
				t.Logf("failed to close body: %v", err)
			}
		})

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("Failed to read body: %v", err)
		}

		if string(body) != "ok" {
			t.Errorf("Expected body 'ok', got '%s'", string(body))
		}
	})

	t.Run("custom handler", func(t *testing.T) {
		resp, err := http.Get(server.URL + "/readyz")
		if err != nil {
			t.Fatalf("Failed to get /readyz: %v", err)
		}
		t.Cleanup(func() {
			if err := resp.Body.Close(); err != nil {
				t.Logf("failed to close body: %v", err)
			}
		})

		if resp.StatusCode != http.StatusTeapot {
			t.Errorf("Expected status %d, got %d", http.StatusTeapot, resp.StatusCode)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("Failed to read body: %v", err)
		}

		if string(body) != "custom" {
			t.Errorf("Expected body 'custom', got '%s'", string(body))
		}

		if !customHandlerCalled {
			t.Error("Custom handler was not called")
		}
	})
}

func TestDefaultConfig(t *testing.T) {
	cfg := defaultConfig()

	if cfg.LivenessEndpoint != "/livez" {
		t.Errorf("Expected LivenessEndpoint '/livez', got '%s'", cfg.LivenessEndpoint)
	}

	if cfg.ReadinessEndpoint != "/readyz" {
		t.Errorf("Expected ReadinessEndpoint '/readyz', got '%s'", cfg.ReadinessEndpoint)
	}

	if cfg.StartupEndpoint != "/startupz" {
		t.Errorf("Expected StartupEndpoint '/startupz', got '%s'", cfg.StartupEndpoint)
	}

	if cfg.LivenessHandler == nil {
		t.Error("Expected LivenessHandler to be set")
	}

	if cfg.ReadinessHandler == nil {
		t.Error("Expected ReadinessHandler to be set")
	}

	if cfg.StartupHandler == nil {
		t.Error("Expected StartupHandler to be set")
	}
}
