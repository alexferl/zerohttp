# Testing Utilities

The `zhtest` package provides fluent, chainable helpers for testing HTTP handlers and middleware.

## Table of Contents

- [Request Builder](#request-builder)
- [Serving Requests](#serving-requests)
- [Response Assertions](#response-assertions)
- [JSON Assertions](#json-assertions)
- [Problem Details Assertions](#problem-details-assertions)
- [Testing Handlers and Middleware](#testing-handlers-and-middleware)
- [Template Testing](#template-testing)
- [Response Wrapper](#response-wrapper)

## Request Builder

Build test requests fluently:

```go
import "github.com/alexferl/zerohttp/zhtest"

// Simple GET request
req := zhtest.NewRequest(http.MethodGet, "/users").Build()

// With query parameters
req := zhtest.NewRequest(http.MethodGet, "/users").
    WithQuery("page", "1").
    WithQuery("limit", "10").
    Build()

// With headers
req := zhtest.NewRequest(http.MethodGet, "/api/data").
    WithHeader("Authorization", "Bearer token").
    WithHeader("X-Request-ID", "abc123").
    Build()

// With JSON body
req := zhtest.NewRequest(http.MethodPost, "/users").
    WithJSON(zh.M{"name": "John", "email": "john@example.com"}).
    Build()

// With form data
req := zhtest.NewRequest(http.MethodPost, "/login").
    WithForm(url.Values{"username": []string{"john"}}).
    Build()

// With cookies
req := zhtest.NewRequest(http.MethodGet, "/profile").
    WithCookie(&http.Cookie{Name: "session", Value: "abc123"}).
    Build()
```

## Serving Requests

Test handlers with recorded responses:

```go
router := zh.NewRouter()
router.GET("/users/:id", getUserHandler)

// Build and serve request
req := zhtest.NewRequest(http.MethodGet, "/users/123").Build()
w := zhtest.Serve(router, req)

// Assert on response
zhtest.AssertWith(t, w).Status(http.StatusOK))
```

## Response Assertions

Chainable assertions for response validation:

```go
w := zhtest.Serve(handler, req)

// Status assertions
zhtest.AssertWith(t, w).Status(http.StatusOK)
zhtest.AssertWith(t, w).StatusBetween(200, 299)
zhtest.AssertWith(t, w).IsSuccess()
zhtest.AssertWith(t, w).IsClientError()
zhtest.AssertWith(t, w).IsServerError()

// Header assertions
zhtest.AssertWith(t, w).
    Header(zh.HeaderContentType, zh.MIMEApplicationJSON).
    HeaderContains(zh.HeaderContentType, "json").
    HeaderExists(zh.HeaderXRequestID)

// Body assertions
zhtest.AssertWith(t, w).Body("exact match")
zhtest.AssertWith(t, w).BodyContains("partial")
zhtest.AssertWith(t, w).BodyEmpty()
zhtest.AssertWith(t, w).BodyNotEmpty()
```

## JSON Assertions

Decode and assert on JSON responses:

```go
// Decode into struct
var user User
zhtest.AssertWith(t, w).JSON(&user)

// Assert JSON equals expected
zhtest.AssertWith(t, w).JSONEquals(zh.M{"name": "John"})

// Assert JSON path value
zhtest.AssertWith(t, w).JSONPathEqual("user.name", "John")
zhtest.AssertWith(t, w).JSONPathEqual("user.email", "john@example.com")
```

## Problem Details Assertions

Test RFC 9457 Problem Detail responses:

```go
zhtest.AssertWith(t, w).
    IsProblemDetail().
    ProblemDetailStatus(http.StatusUnprocessableEntity).
    ProblemDetailTitle("Unprocessable Entity").
    ProblemDetailDetail("Validation failed").
    ProblemDetailType("https://api.example.com/errors/validation").
    ProblemDetailExtension("errors", []string{"field required"})
```

## Testing Handlers and Middleware

Test handlers and middleware directly:

```go
// Test handler directly
handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    zh.Render.JSON(w, http.StatusOK, zh.M{"ok": true})
})

req := zhtest.NewRequest(http.MethodGet, "/").Build()
w := zhtest.TestHandler(handler, req)
zhtest.AssertWith(t, w).Status(http.StatusOK)

// Test middleware
mw := func(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("X-Custom", "value")
        next.ServeHTTP(w, r)
    })
}

req := zhtest.NewRequest(http.MethodGet, "/").Build()
w := zhtest.TestMiddleware(mw, req)
zhtest.AssertWith(t, w).Header("X-Custom", "value")

// Test middleware chain
mw1 := func(next http.Handler) http.Handler { /* ... */ }
mw2 := func(next http.Handler) http.Handler { /* ... */ }

req := zhtest.NewRequest(http.MethodGet, "/").Build()
w := zhtest.TestMiddlewareChain([]func(http.Handler) http.Handler{mw1, mw2}, req)
```

## Template Testing

Test template rendering:

```go
// Test template rendering
w := zhtest.TestTemplate(`<h1>{{.Title}}</h1>`, map[string]string{"Title": "Hello"})
zhtest.AssertTemplateWith(t, w).
    Contains("<h1>Hello</h1>").
    HasTitle("Page Title")

// Or use the renderer for complex templates
tmpl := template.Must(template.New("test").Parse(templates))
tr := zhtest.NewTemplateRenderer(tmpl)
w := tr.Render("index.html", data)
```

## Response Wrapper

Direct access to response data:

```go
w := zhtest.ServeWithRecorder(handler, req)

// Body access
bodyStr := w.BodyString()
bodyBytes := w.BodyBytes()

// JSON decoding
var result MyStruct
err := w.JSON(&result)

// Cookie access
cookie := w.Cookie("session")
cookieValue := w.CookieValue("session")

// Header access
headerValue := w.HeaderValue(zh.HeaderContentType)

// Status checks
isSuccess := w.IsSuccess()
isRedirect := w.IsRedirect()
isClientError := w.IsClientError()
isServerError := w.IsServerError()
```

## Complete Example

```go
func TestCreateUser(t *testing.T) {
    // Setup router
    router := zh.NewRouter()
    router.POST("/users", createUserHandler)

    // Build request
    req := zhtest.NewRequest(http.MethodPost, "/users").
        WithJSON(zh.M{
            "name":  "John Doe",
            "email": "john@example.com",
        }).
        Build()

    // Serve and get response
    w := zhtest.Serve(router, req)

    // Chain assertions
    zhtest.AssertWith(t, w).
        Status(http.StatusCreated).
        Header(zh.HeaderContentType, zh.MIMEApplicationJSON).
        JSONPathEqual("name", "John Doe").
        JSONPathEqual("email", "john@example.com")
}
```
