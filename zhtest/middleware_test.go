package zhtest

import (
	"net/http"
	"testing"
)

func TestServe(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("hello")); err != nil {
			t.Errorf("failed to write: %v", err)
		}
	})
	req := NewRequest(http.MethodGet, "/").Build()

	w := Serve(handler, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	if w.Body.String() != "hello" {
		t.Errorf("expected body 'hello', got %s", w.Body.String())
	}
}

func TestServeWithRecorder(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Custom", "value")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("hello")); err != nil {
			t.Errorf("failed to write: %v", err)
		}
	})
	req := NewRequest(http.MethodGet, "/").Build()

	w := ServeWithRecorder(handler, req)

	if !w.IsSuccess() {
		t.Error("expected IsSuccess to be true")
	}
	if w.HeaderValue("X-Custom") != "value" {
		t.Error("expected X-Custom header")
	}
}

func TestTestHandler(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST method, got %s", r.Method)
		}
		w.WriteHeader(http.StatusCreated)
	})
	req := NewRequest(http.MethodPost, "/").Build()

	w := TestHandler(handler, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", w.Code)
	}
}

func TestTestMiddleware(t *testing.T) {
	t.Run("middleware that sets header", func(t *testing.T) {
		mw := func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-Middleware", "applied")
				next.ServeHTTP(w, r)
			})
		}
		req := NewRequest(http.MethodGet, "/").Build()

		w := TestMiddleware(mw, req)

		if w.Header().Get("X-Middleware") != "applied" {
			t.Error("expected X-Middleware header to be set")
		}
		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}
	})

	t.Run("middleware that intercepts request", func(t *testing.T) {
		mw := func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
				// Does not call next
			})
		}
		req := NewRequest(http.MethodGet, "/").Build()

		w := TestMiddleware(mw, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %d", w.Code)
		}
	})
}

func TestTestMiddlewareWithHandler(t *testing.T) {
	mw := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Before", "true")
			next.ServeHTTP(w, r)
			w.Header().Set("X-After", "true")
		})
	}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		if _, err := w.Write([]byte("created")); err != nil {
			t.Errorf("failed to write: %v", err)
		}
	})
	req := NewRequest(http.MethodPost, "/").Build()

	w := TestMiddlewareWithHandler(mw, handler, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", w.Code)
	}
	if w.Header().Get("X-Before") != "true" {
		t.Error("expected X-Before header to be set")
	}
	if w.Header().Get("X-After") != "true" {
		t.Error("expected X-After header to be set")
	}
	if w.Body.String() != "created" {
		t.Errorf("expected body 'created', got %s", w.Body.String())
	}
}

func TestTestMiddlewareChain(t *testing.T) {
	var order []string

	mw1 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "mw1-before")
			next.ServeHTTP(w, r)
			order = append(order, "mw1-after")
		})
	}
	mw2 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "mw2-before")
			next.ServeHTTP(w, r)
			order = append(order, "mw2-after")
		})
	}
	req := NewRequest(http.MethodGet, "/").Build()

	w := TestMiddlewareChain([]func(http.Handler) http.Handler{mw1, mw2}, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	expectedOrder := []string{"mw1-before", "mw2-before", "mw2-after", "mw1-after"}
	if len(order) != len(expectedOrder) {
		t.Fatalf("expected order %v, got %v", expectedOrder, order)
	}
	for i, expected := range expectedOrder {
		if order[i] != expected {
			t.Errorf("expected order[%d] to be %q, got %q", i, expected, order[i])
		}
	}
}
