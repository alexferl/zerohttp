package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/httpx"
	"github.com/alexferl/zerohttp/metrics"
	"github.com/alexferl/zerohttp/zhtest"
)

func TestTimeout_Scenarios(t *testing.T) {
	tests := []struct {
		name       string
		delay      time.Duration
		timeout    time.Duration
		statusCode int
		message    string
		wantStatus int
		wantBody   string
	}{
		{"success", 10 * time.Millisecond, 50 * time.Millisecond, 0, "", http.StatusOK, "ok"},
		{"timeout", 100 * time.Millisecond, 50 * time.Millisecond, 0, "timeout", http.StatusGatewayTimeout, "timeout"},
		{"custom status", 100 * time.Millisecond, 50 * time.Millisecond, http.StatusRequestTimeout, "custom", http.StatusRequestTimeout, "custom"},
		{"empty message", 100 * time.Millisecond, 50 * time.Millisecond, 0, "", http.StatusGatewayTimeout, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				select {
				case <-r.Context().Done():
					return
				case <-time.After(tt.delay):
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte("ok"))
				}
			})

			cfg := config.TimeoutConfig{Timeout: tt.timeout}
			if tt.statusCode != 0 {
				cfg.StatusCode = tt.statusCode
			}
			if tt.message != "" {
				cfg.Message = tt.message
			}
			middleware := Timeout(cfg)(handler)

			req := zhtest.NewRequest(http.MethodGet, "/").Build()
			w := zhtest.Serve(middleware, req)

			if tt.name == "success" {
				zhtest.AssertWith(t, w).Status(tt.wantStatus).Body(tt.wantBody)
			} else {
				// Timeout cases return ProblemDetail
				// Test JSON response
				req = zhtest.NewRequest(http.MethodGet, "/").WithHeader("Accept", "application/json").Build()
				w = zhtest.Serve(middleware, req)
				zhtest.AssertWith(t, w).Status(tt.wantStatus).IsProblemDetail()

				// Test plain text response
				req = zhtest.NewRequest(http.MethodGet, "/").Build()
				w = zhtest.Serve(middleware, req)
				zhtest.AssertWith(t, w).Status(tt.wantStatus).Header(httpx.HeaderContentType, "text/plain; charset=utf-8")
			}
		})
	}
}

func TestTimeout_DefaultValues(t *testing.T) {
	tests := []struct {
		name   string
		cfg    config.TimeoutConfig
		delay  time.Duration
		expect func(t *testing.T, w *httptest.ResponseRecorder)
	}{
		{
			"zero timeout uses default",
			config.TimeoutConfig{Timeout: 0},
			10 * time.Millisecond,
			func(t *testing.T, w *httptest.ResponseRecorder) {
				zhtest.AssertWith(t, w).Status(http.StatusOK)
			},
		},
		{
			"zero status code uses default",
			config.TimeoutConfig{Timeout: 50 * time.Millisecond, StatusCode: 0},
			100 * time.Millisecond,
			func(t *testing.T, w *httptest.ResponseRecorder) {
				zhtest.AssertWith(t, w).Status(http.StatusGatewayTimeout)
			},
		},
		{
			"nil exempt paths uses default",
			config.TimeoutConfig{Timeout: 50 * time.Millisecond, ExemptPaths: nil},
			100 * time.Millisecond,
			func(t *testing.T, w *httptest.ResponseRecorder) {
				zhtest.AssertWith(t, w).Status(http.StatusGatewayTimeout)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				select {
				case <-r.Context().Done():
					return
				case <-time.After(tt.delay):
					_, _ = w.Write([]byte("ok"))
				}
			})
			middleware := Timeout(tt.cfg)(handler)

			req := zhtest.NewRequest(http.MethodGet, "/").Build()
			w := zhtest.Serve(middleware, req)

			tt.expect(t, w)
		})
	}
}

func TestTimeout_AllDefaults(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Millisecond)
		_, _ = w.Write([]byte("success"))
	})
	middleware := Timeout()(handler)

	req := zhtest.NewRequest(http.MethodGet, "/").Build()
	w := zhtest.Serve(middleware, req)

	zhtest.AssertWith(t, w).Status(http.StatusOK).Body("success")
}

func TestTimeout_ExemptPaths(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		_, _ = w.Write([]byte("done"))
	})
	middleware := Timeout(config.TimeoutConfig{
		Timeout:     50 * time.Millisecond,
		ExemptPaths: []string{"/health"},
	})(handler)

	req := zhtest.NewRequest(http.MethodGet, "/health").Build()
	w := zhtest.Serve(middleware, req)

	zhtest.AssertWith(t, w).Status(http.StatusOK).Body("done")
}

func TestTimeout_HeadersPreserved(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Custom", "test")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("created"))
	})
	middleware := Timeout()(handler)

	req := zhtest.NewRequest(http.MethodGet, "/").Build()
	w := zhtest.Serve(middleware, req)

	zhtest.AssertWith(t, w).Status(http.StatusCreated).Header("X-Custom", "test")
}

func TestTimeout_PanicPropagation(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})
	middleware := Timeout()(handler)

	req := zhtest.NewRequest(http.MethodGet, "/").Build()

	defer func() {
		if r := recover(); r != "test panic" {
			t.Errorf("panic = %v, want 'test panic'", r)
		}
	}()
	zhtest.Serve(middleware, req)

	t.Error("should not reach here")
}

func TestTimeout_NoRaceCondition(t *testing.T) {
	for range 20 {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			select {
			case <-r.Context().Done():
				return
			case <-time.After(45 * time.Millisecond):
				_, err := w.Write([]byte("done"))
				// Ignore timeout middleware write errors - this is expected behavior
				if err != nil && !errors.Is(err, ErrTimeoutWrite) {
					t.Fatalf("unexpected write error: %v", err)
				}
			}
		})
		middleware := Timeout(config.TimeoutConfig{Timeout: 50 * time.Millisecond})(handler)

		req := zhtest.NewRequest(http.MethodGet, "/").Build()
		w := zhtest.Serve(middleware, req)

		body := w.Body.String()
		if w.Code == http.StatusOK && body != "done" {
			t.Fatalf("race detected: status 200 but body %q", body)
		}
		if w.Code == http.StatusGatewayTimeout && body == "done" {
			t.Fatal("race detected: timeout status but success body")
		}
	}
}

func TestTimeout_Metrics(t *testing.T) {
	reg := metrics.NewRegistry()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
			return
		case <-time.After(100 * time.Millisecond):
			w.WriteHeader(http.StatusOK)
		}
	})

	// Wrap with metrics middleware to provide registry in context
	metricsMw := metrics.NewMiddleware(reg, config.MetricsConfig{
		Enabled:       config.Bool(true),
		PathLabelFunc: func(p string) string { return p },
	})
	timeoutMw := Timeout(config.TimeoutConfig{Timeout: 50 * time.Millisecond})

	// Chain: metrics -> timeout -> handler
	wrapped := metricsMw(timeoutMw(handler))

	req := zhtest.NewRequest(http.MethodGet, "/test").Build()
	w := zhtest.Serve(wrapped, req)

	// Should timeout
	if w.Code != http.StatusGatewayTimeout {
		t.Errorf("expected status %d, got %d", http.StatusGatewayTimeout, w.Code)
	}

	// Check that timeout metric was recorded
	families := reg.Gather()

	var timeoutCounter *metrics.MetricFamily
	for _, f := range families {
		if f.Name == "timeout_requests_total" {
			timeoutCounter = &f
			break
		}
	}

	if timeoutCounter == nil {
		t.Fatal("expected timeout_requests_total metric")
	}

	if len(timeoutCounter.Metrics) != 1 {
		t.Errorf("expected 1 timeout metric, got %d", len(timeoutCounter.Metrics))
	}
}
