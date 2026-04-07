package timeout

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

			cfg := Config{Duration: tt.timeout}
			if tt.statusCode != 0 {
				cfg.StatusCode = tt.statusCode
			}
			if tt.message != "" {
				cfg.Message = tt.message
			}
			middleware := New(cfg)(handler)

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

				// Test JSON response without Accept header (defaults to JSON)
				req = zhtest.NewRequest(http.MethodGet, "/").Build()
				w = zhtest.Serve(middleware, req)
				zhtest.AssertWith(t, w).Status(tt.wantStatus).Header(httpx.HeaderContentType, "application/problem+json")
			}
		})
	}
}

func TestTimeout_DefaultValues(t *testing.T) {
	tests := []struct {
		name   string
		cfg    Config
		delay  time.Duration
		expect func(t *testing.T, w *httptest.ResponseRecorder)
	}{
		{
			"zero timeout uses default",
			Config{Duration: 0},
			10 * time.Millisecond,
			func(t *testing.T, w *httptest.ResponseRecorder) {
				zhtest.AssertWith(t, w).Status(http.StatusOK)
			},
		},
		{
			"zero status code uses default",
			Config{Duration: 50 * time.Millisecond, StatusCode: 0},
			100 * time.Millisecond,
			func(t *testing.T, w *httptest.ResponseRecorder) {
				zhtest.AssertWith(t, w).Status(http.StatusGatewayTimeout)
			},
		},
		{
			"nil excluded paths uses default",
			Config{Duration: 50 * time.Millisecond, ExcludedPaths: nil},
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
			middleware := New(tt.cfg)(handler)

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
	middleware := New()(handler)

	req := zhtest.NewRequest(http.MethodGet, "/").Build()
	w := zhtest.Serve(middleware, req)

	zhtest.AssertWith(t, w).Status(http.StatusOK).Body("success")
}

func TestTimeout_ExcludedPaths(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		_, _ = w.Write([]byte("done"))
	})
	middleware := New(Config{
		Duration:      50 * time.Millisecond,
		ExcludedPaths: []string{"/health"},
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
	middleware := New()(handler)

	req := zhtest.NewRequest(http.MethodGet, "/").Build()
	w := zhtest.Serve(middleware, req)

	zhtest.AssertWith(t, w).Status(http.StatusCreated).Header("X-Custom", "test")
}

func TestTimeout_PanicPropagation(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})
	middleware := New()(handler)

	req := zhtest.NewRequest(http.MethodGet, "/").Build()

	zhtest.AssertPanic(t, func() {
		zhtest.Serve(middleware, req)
	})
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
					zhtest.AssertFailf(t, "unexpected write error: %v", err)
				}
			}
		})
		middleware := New(Config{Duration: 50 * time.Millisecond})(handler)

		req := zhtest.NewRequest(http.MethodGet, "/").Build()
		w := zhtest.Serve(middleware, req)

		body := w.Body.String()
		if w.Code == http.StatusOK && body != "done" {
			zhtest.AssertFailf(t, "race detected: status 200 but body %q", body)
		}
		if w.Code == http.StatusGatewayTimeout && body == "done" {
			zhtest.AssertFail(t, "race detected: timeout status but success body")
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
	metricsMw := metrics.NewMiddleware(reg, metrics.Config{
		Enabled:       config.Bool(true),
		PathLabelFunc: func(p string) string { return p },
	})
	timeoutMw := New(Config{Duration: 50 * time.Millisecond})

	// Chain: metrics -> timeout -> handler
	wrapped := metricsMw(timeoutMw(handler))

	req := zhtest.NewRequest(http.MethodGet, "/test").Build()
	w := zhtest.Serve(wrapped, req)

	// Should timeout
	zhtest.AssertEqual(t, http.StatusGatewayTimeout, w.Code)

	// Check that timeout metric was recorded
	families := reg.Gather()

	var timeoutCounter *metrics.MetricFamily
	for _, f := range families {
		if f.Name == "timeout_requests_total" {
			timeoutCounter = &f
			break
		}
	}

	zhtest.AssertNotNil(t, timeoutCounter)
	zhtest.AssertEqual(t, 1, len(timeoutCounter.Metrics))
}

type flusherRecorder struct {
	*httptest.ResponseRecorder
	flushed bool
}

func (f *flusherRecorder) Flush() {
	f.flushed = true
}

func TestTimeout_Flush(t *testing.T) {
	tests := []struct {
		name              string
		underlyingFlusher bool
		expectFlushCalled bool
	}{
		{
			name:              "flush passes through to underlying Flusher",
			underlyingFlusher: true,
			expectFlushCalled: true,
		},
		{
			name:              "flush no-op when underlying doesn't implement Flusher",
			underlyingFlusher: false,
			expectFlushCalled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var base http.ResponseWriter
			var flushCalled *bool

			if tt.underlyingFlusher {
				rec := &flusherRecorder{ResponseRecorder: httptest.NewRecorder()}
				base = rec
				flushCalled = &rec.flushed
			} else {
				rec := httptest.NewRecorder()
				base = rec
				flushCalled = new(bool)
			}

			// Create timeoutWriter
			tw := &timeoutWriter{
				w:   base,
				req: httptest.NewRequest(http.MethodGet, "/", nil),
				h:   make(http.Header),
			}

			// Call Flush
			tw.Flush()

			zhtest.AssertEqual(t, tt.expectFlushCalled, *flushCalled)
		})
	}
}

func TestTimeout_Flush_WritesBufferedData(t *testing.T) {
	rec := httptest.NewRecorder()

	// Create timeoutWriter
	tw := &timeoutWriter{
		w:   rec,
		req: httptest.NewRequest(http.MethodGet, "/", nil),
		h:   make(http.Header),
	}

	// Write some data (buffered)
	_, _ = tw.Write([]byte("hello"))

	// Buffered data should not be written yet
	zhtest.AssertEqual(t, "", rec.Body.String())

	// Flush should write buffered data
	tw.Flush()

	// Now data should be written
	zhtest.AssertEqual(t, "hello", rec.Body.String())
}

func TestTimeout_IncludedPaths(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		_, _ = w.Write([]byte("done"))
	})

	middleware := New(Config{
		Duration:      50 * time.Millisecond,
		IncludedPaths: []string{"/api/", "/admin"},
	})(handler)

	tests := []struct {
		name       string
		path       string
		wantStatus int
		wantBody   string
	}{
		{"allowed path - timeout applies", "/api/users", http.StatusGatewayTimeout, ""},
		{"allowed exact path", "/admin", http.StatusGatewayTimeout, ""},
		{"non-allowed path - no timeout", "/health", http.StatusOK, "done"},
		{"non-allowed path 2", "/metrics", http.StatusOK, "done"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := zhtest.NewRequest(http.MethodGet, tt.path).Build()
			w := zhtest.Serve(middleware, req)

			if tt.wantStatus == http.StatusOK {
				zhtest.AssertWith(t, w).Status(tt.wantStatus).Body(tt.wantBody)
			} else {
				// Timeout cases
				zhtest.AssertWith(t, w).Status(tt.wantStatus)
			}
		})
	}
}

func TestTimeout_BothExcludedAndIncludedPathsPanics(t *testing.T) {
	zhtest.AssertPanic(t, func() {
		_ = New(Config{
			Duration:      50 * time.Millisecond,
			ExcludedPaths: []string{"/health"},
			IncludedPaths: []string{"/api"},
		})
	})
}
