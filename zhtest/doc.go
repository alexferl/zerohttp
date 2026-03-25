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
//	    Header(httpx.HeaderContentType, "application/json").
//	    JSONPathEqual("name", "John").
//	    JSONPathEqual("email", "john@example.com")
//
// # General Assertions
//
// Use standalone assertion functions for general test assertions:
//
// Error Assertions:
//
//	zhtest.AssertNoError(t, err)
//	zhtest.AssertError(t, err)
//	zhtest.AssertErrorIs(t, err, os.ErrNotExist)
//	zhtest.AssertErrorContains(t, err, "connection refused")
//
// Nil/NotNil Assertions:
//
//	zhtest.AssertNil(t, ptr)
//	zhtest.AssertNotNil(t, result)
//
// Equality Assertions:
//
//	zhtest.AssertEqual(t, 42, result)
//	zhtest.AssertNotEqual(t, "old", result)
//	zhtest.AssertDeepEqual(t, []int{1, 2, 3}, result)
//
// Boolean Assertions:
//
//	zhtest.AssertTrue(t, len(items) > 0)
//	zhtest.AssertFalse(t, len(items) == 0)
//
// Empty/NotEmpty Assertions:
//
//	zhtest.AssertEmpty(t, "")
//	zhtest.AssertEmpty(t, []int{})
//	zhtest.AssertNotEmpty(t, "hello")
//	zhtest.AssertNotEmpty(t, []int{1, 2, 3})
//
// Collection Assertions:
//
//	zhtest.AssertLen(t, []int{1, 2, 3}, 3)
//	zhtest.AssertContains(t, []int{1, 2, 3}, 2)
//	zhtest.AssertNotContains(t, []int{1, 2, 3}, 4)
//
// Type Assertions:
//
//	zhtest.AssertIsType(t, (*MyError)(nil), err)
//	zhtest.AssertImplements(t, (*io.Reader)(nil), myReader)
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
//	mw := cors.New(cors.DefaultConfig)
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
