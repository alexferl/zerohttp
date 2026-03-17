// Package zhtest provides testing utilities for zerohttp applications.
//
// This package uses only Go's standard library (net/http/httptest) to maintain
// the zero external dependencies constraint.
//
// # Quick Start
//
// Build and serve test requests:
//
//	func TestGetUser(t *testing.T) {
//	    router := setupRouter()
//
//	    req := zhtest.NewRequest(http.MethodGet, "/users/123").Build()
//	    w := zhtest.Serve(router, req)
//
//	    zhtest.AssertWith(t, w).Status(http.StatusOK)
//	}
//
// # Request Builder
//
// Build requests with a fluent API:
//
//	req := zhtest.NewRequest(http.MethodPost, "/users").
//	    WithHeader(httpx.HeaderAuthorization, "Bearer token").
//	    WithJSON(zh.M{"name": "John", "email": "john@example.com"}).
//	    Build()
//
// # Response Assertions
//
// Chain assertions for readable tests:
//
//	zhtest.AssertWith(t, w).
//	    Status(http.StatusCreated).
//	    httpx.HeaderContentType, "application/json").
//	    JSONPathEqual("name", "John").
//	    JSONPathEqual("email", "john@example.com")
//
// # Testing Problem Details
//
// Assert RFC 9457 Problem Detail responses:
//
//	zhtest.AssertWith(t, w).
//	    IsProblemDetail().
//	    ProblemDetailStatus(http.StatusUnprocessableEntity).
//	    ProblemDetailTitle("Unprocessable Entity")
//
// # Testing Middleware
//
// Test middleware in isolation:
//
//	mw := middleware.CORS(config.DefaultCORSConfig)
//	w := zhtest.TestMiddleware(mw, req)
//	zhtest.AssertWith(t, w).Header(httpx.HeaderAccessControlAllowOrigin, "*")
//
// # Direct Handler Testing
//
// Test handlers directly:
//
//	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//	    zh.Render.JSON(w, http.StatusOK, zh.M{"ok": true})
//	})
//	w := zhtest.TestHandler(handler, req)
//	zhtest.AssertWith(t, w).Status(http.StatusOK)
package zhtest
