package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alexferl/zerohttp/config"
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
		{"success", 10 * time.Millisecond, 50 * time.Millisecond, 0, "", 200, "ok"},
		{"timeout", 100 * time.Millisecond, 50 * time.Millisecond, 0, "timeout", 504, "timeout"},
		{"custom status", 100 * time.Millisecond, 50 * time.Millisecond, 408, "custom", 408, "custom"},
		{"empty message", 100 * time.Millisecond, 50 * time.Millisecond, 0, "", 504, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				select {
				case <-r.Context().Done():
					return
				case <-time.After(tt.delay):
					w.WriteHeader(200)
					_, err := w.Write([]byte("ok"))
					if err != nil {
						t.Fatalf("write failed: %v", err)
					}
				}
			})

			opts := []config.TimeoutOption{config.WithTimeoutDuration(tt.timeout)}
			if tt.statusCode != 0 {
				opts = append(opts, config.WithTimeoutStatusCode(tt.statusCode))
			}
			if tt.message != "" {
				opts = append(opts, config.WithTimeoutMessage(tt.message))
			}
			middleware := Timeout(opts...)(handler)

			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/", nil)
			middleware.ServeHTTP(w, r)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
			if w.Body.String() != tt.wantBody {
				t.Errorf("body = %q, want %q", w.Body.String(), tt.wantBody)
			}
		})
	}
}

func TestTimeout_DefaultValues(t *testing.T) {
	tests := []struct {
		name   string
		opts   []config.TimeoutOption
		delay  time.Duration
		expect func(t *testing.T, w *httptest.ResponseRecorder)
	}{
		{
			"zero timeout uses default",
			[]config.TimeoutOption{config.WithTimeoutDuration(0)},
			10 * time.Millisecond,
			func(t *testing.T, w *httptest.ResponseRecorder) {
				if w.Code != 200 {
					t.Errorf("expected success with default timeout, got %d", w.Code)
				}
			},
		},
		{
			"zero status code uses default",
			[]config.TimeoutOption{config.WithTimeoutDuration(50 * time.Millisecond), config.WithTimeoutStatusCode(0)},
			100 * time.Millisecond,
			func(t *testing.T, w *httptest.ResponseRecorder) {
				if w.Code != http.StatusGatewayTimeout {
					t.Errorf("expected default status 504, got %d", w.Code)
				}
			},
		},
		{
			"nil exempt paths uses default",
			[]config.TimeoutOption{config.WithTimeoutDuration(50 * time.Millisecond), config.WithTimeoutExemptPaths(nil)},
			100 * time.Millisecond,
			func(t *testing.T, w *httptest.ResponseRecorder) {
				if w.Code != http.StatusGatewayTimeout {
					t.Errorf("expected timeout with nil exempt paths, got %d", w.Code)
				}
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
					_, err := w.Write([]byte("ok"))
					if err != nil {
						t.Fatalf("write failed: %v", err)
					}
				}
			})
			middleware := Timeout(tt.opts...)(handler)

			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/", nil)
			middleware.ServeHTTP(w, r)

			tt.expect(t, w)
		})
	}
}

func TestTimeout_AllDefaults(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Millisecond)
		_, err := w.Write([]byte("success"))
		if err != nil {
			t.Fatalf("write failed: %v", err)
		}
	})
	middleware := Timeout()(handler)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)

	middleware.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Errorf("expected success with all defaults, got %d", w.Code)
	}
	if w.Body.String() != "success" {
		t.Errorf("expected 'success', got %q", w.Body.String())
	}
}

func TestTimeout_ExemptPaths(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		_, err := w.Write([]byte("done"))
		if err != nil {
			t.Fatalf("write failed: %v", err)
		}
	})
	middleware := Timeout(
		config.WithTimeoutDuration(50*time.Millisecond),
		config.WithTimeoutExemptPaths([]string{"/health"}),
	)(handler)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/health", nil)

	middleware.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Errorf("exempt path failed, status = %d", w.Code)
	}
	if w.Body.String() != "done" {
		t.Errorf("exempt path body = %q, want %q", w.Body.String(), "done")
	}
}

func TestTimeout_HeadersPreserved(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Custom", "test")
		w.WriteHeader(201)
		_, err := w.Write([]byte("created"))
		if err != nil {
			t.Fatalf("write failed: %v", err)
		}
	})
	middleware := Timeout()(handler)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	middleware.ServeHTTP(w, r)

	if w.Code != 201 {
		t.Errorf("status = %d, want 201", w.Code)
	}
	if w.Header().Get("X-Custom") != "test" {
		t.Error("custom header not preserved")
	}
}

func TestTimeout_PanicPropagation(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})
	middleware := Timeout()(handler)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)

	defer func() {
		if r := recover(); r != "test panic" {
			t.Errorf("panic = %v, want 'test panic'", r)
		}
	}()
	middleware.ServeHTTP(w, r)

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
				if err != nil {
					t.Fatalf("write failed: %v", err)
				}
			}
		})
		middleware := Timeout(config.WithTimeoutDuration(50 * time.Millisecond))(handler)

		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/", nil)
		middleware.ServeHTTP(w, r)

		body := w.Body.String()
		if w.Code == 200 && body != "done" {
			t.Fatalf("race detected: status 200 but body %q", body)
		}
		if w.Code == 504 && body == "done" {
			t.Fatal("race detected: timeout status but success body")
		}
	}
}
