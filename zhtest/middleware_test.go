package zhtest

import (
	"net/http"
	"testing"
)

func TestServe(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("hello"))
		AssertNoError(t, err)
	})
	req := NewRequest(http.MethodGet, "/").Build()

	w := Serve(handler, req)

	AssertEqual(t, http.StatusOK, w.Code)
	AssertEqual(t, "hello", w.Body.String())
}

func TestServeWithRecorder(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Custom", "value")
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("hello"))
		AssertNoError(t, err)
	})
	req := NewRequest(http.MethodGet, "/").Build()

	w := ServeWithRecorder(handler, req)

	AssertTrue(t, w.IsSuccess())
	AssertEqual(t, "value", w.HeaderValue("X-Custom"))
}

func TestTestHandler(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		AssertEqual(t, http.MethodPost, r.Method)
		w.WriteHeader(http.StatusCreated)
	})
	req := NewRequest(http.MethodPost, "/").Build()

	w := TestHandler(handler, req)

	AssertEqual(t, http.StatusCreated, w.Code)
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

		AssertEqual(t, "applied", w.Header().Get("X-Middleware"))
		AssertEqual(t, http.StatusOK, w.Code)
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

		AssertEqual(t, http.StatusUnauthorized, w.Code)
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
		_, err := w.Write([]byte("created"))
		AssertNoError(t, err)
	})
	req := NewRequest(http.MethodPost, "/").Build()

	w := TestMiddlewareWithHandler(mw, handler, req)

	AssertEqual(t, http.StatusCreated, w.Code)
	AssertEqual(t, "true", w.Header().Get("X-Before"))
	AssertEqual(t, "true", w.Header().Get("X-After"))
	AssertEqual(t, "created", w.Body.String())
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

	AssertEqual(t, http.StatusOK, w.Code)

	expectedOrder := []string{"mw1-before", "mw2-before", "mw2-after", "mw1-after"}
	AssertEqual(t, len(expectedOrder), len(order))
	for i, expected := range expectedOrder {
		AssertEqual(t, expected, order[i])
	}
}
