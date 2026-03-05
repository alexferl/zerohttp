package zhtest

import (
	"net/http"
	"net/http/httptest"
)

// Serve serves the handler with the given request and returns the response.
// This is a convenience function for testing handlers.
//
// Example:
//
//	req := zhtest.NewRequest(http.MethodGet, "/users").Build()
//	w := zhtest.Serve(router, req)
//	zhtest.AssertWith(t, w).Status(http.StatusOK)
func Serve(handler http.Handler, req *http.Request) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	return w
}

// ServeWithRecorder serves the handler with the given request and returns a Response wrapper.
//
// Example:
//
//	req := zhtest.NewRequest(http.MethodGet, "/users").Build()
//	w := zhtest.ServeWithRecorder(router, req)
//	if !w.IsSuccess() {
//	    t.Error("expected success")
//	}
func ServeWithRecorder(handler http.Handler, req *http.Request) *Response {
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	return &Response{ResponseRecorder: w}
}

// TestHandler serves a handler function with the given request.
// This is a convenience function for testing handler functions directly.
//
// Example:
//
//	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//	    w.WriteHeader(http.StatusOK)
//	    w.Write([]byte("ok"))
//	})
//	req := zhtest.NewRequest(http.MethodGet, "/").Build()
//	w := zhtest.TestHandler(handler, req)
//	zhtest.AssertWith(t, w).Status(http.StatusOK)
func TestHandler(handler http.HandlerFunc, req *http.Request) *httptest.ResponseRecorder {
	return Serve(handler, req)
}

// TestMiddleware tests a middleware with a no-op handler.
// Returns the response from running the middleware with an empty handler.
//
// Example:
//
//	mw := func(next http.Handler) http.Handler {
//	    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//	        w.Header().Set("X-Custom", "value")
//	        next.ServeHTTP(w, r)
//	    })
//	}
//	req := zhtest.NewRequest(http.MethodGet, "/").Build()
//	w := zhtest.TestMiddleware(mw, req)
//	zhtest.AssertWith(t, w).Header("X-Custom", "value")
func TestMiddleware(middleware func(http.Handler) http.Handler, req *http.Request) *httptest.ResponseRecorder {
	// Create a no-op handler that just returns 200
	nopHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Wrap the handler with the middleware
	wrapped := middleware(nopHandler)

	return Serve(wrapped, req)
}

// TestMiddlewareWithHandler tests a middleware with a custom handler.
// This allows you to verify that the middleware properly calls the next handler.
//
// Example:
//
//	mw := func(next http.Handler) http.Handler {
//	    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//	        w.Header().Set("X-Before", "true")
//	        next.ServeHTTP(w, r)
//	        w.Header().Set("X-After", "true")
//	    })
//	}
//	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//	    w.WriteHeader(http.StatusCreated)
//	})
//	req := zhtest.NewRequest(http.MethodPost, "/").Build()
//	w := zhtest.TestMiddlewareWithHandler(mw, handler, req)
//	zhtest.AssertWith(t, w).Status(201).Header("X-Before", "true")
func TestMiddlewareWithHandler(middleware func(http.Handler) http.Handler, handler http.Handler, req *http.Request) *httptest.ResponseRecorder {
	wrapped := middleware(handler)
	return Serve(wrapped, req)
}

// TestMiddlewareChain tests a chain of middleware functions.
// The middleware are applied in order (first middleware is the outermost).
//
// Example:
//
//	mw1 := func(next http.Handler) http.Handler {
//	    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//	        w.Header().Set("X-First", "1")
//	        next.ServeHTTP(w, r)
//	    })
//	}
//	mw2 := func(next http.Handler) http.Handler {
//	    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//	        w.Header().Set("X-Second", "2")
//	        next.ServeHTTP(w, r)
//	    })
//	}
//	req := zhtest.NewRequest(http.MethodGet, "/").Build()
//	w := zhtest.TestMiddlewareChain([]func(http.Handler) http.Handler{mw1, mw2}, req)
//	zhtest.AssertWith(t, w).Header("X-First", "1").Header("X-Second", "2")
func TestMiddlewareChain(middlewares []func(http.Handler) http.Handler, req *http.Request) *httptest.ResponseRecorder {
	// Create a no-op handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Apply middleware in reverse order so the first middleware is outermost
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler).ServeHTTP
	}

	return Serve(handler, req)
}
