package zhtest

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
)

// Assert creates a new Assertions instance for the given ResponseRecorder.
// This is a convenience function that doesn't require passing *testing.T.
// For automatic test failures, use AssertWith.
//
// Example:
//
//	w := httptest.NewRecorder()
//	router.ServeHTTP(w, req)
//	zhtest.Assert(w).Status(http.StatusOK)
func Assert(w *httptest.ResponseRecorder) *Assertions {
	return &Assertions{resp: &Response{w}, t: nil}
}

// AssertWith creates a new Assertions instance that will automatically fail the test.
//
// Example:
//
//	w := httptest.NewRecorder()
//	router.ServeHTTP(w, req)
//	zhtest.AssertWith(t, w).Status(http.StatusOK)
func AssertWith(t *testing.T, w *httptest.ResponseRecorder) *Assertions {
	return &Assertions{resp: &Response{w}, t: t}
}

// Assertions provides fluent assertion methods for HTTP responses.
type Assertions struct {
	resp *Response
	t    *testing.T
}

// fail reports a test failure if a testing.T is available.
func (a *Assertions) fail(format string, args ...any) {
	if a.t != nil {
		a.t.Errorf(format, args...)
	}
}

// Status asserts that the response status code equals the expected value.
//
// Example:
//
//	zhtest.AssertWith(t, w).Status(http.StatusOK)
func (a *Assertions) Status(expected int) *Assertions {
	if a.resp.Code != expected {
		a.fail("expected status %d, got %d", expected, a.resp.Code)
	}
	return a
}

// StatusNot asserts that the response status code does not equal the given value.
//
// Example:
//
//	zhtest.AssertWith(t, w).StatusNot(http.StatusNotFound)
func (a *Assertions) StatusNot(unexpected int) *Assertions {
	if a.resp.Code == unexpected {
		a.fail("expected status not to be %d, but it was", unexpected)
	}
	return a
}

// StatusBetween asserts that the response status code is within the given range (inclusive).
//
// Example:
//
//	zhtest.AssertWith(t, w).StatusBetween(200, 299)
func (a *Assertions) StatusBetween(min, max int) *Assertions {
	if a.resp.Code < min || a.resp.Code > max {
		a.fail("expected status between %d and %d, got %d", min, max, a.resp.Code)
	}
	return a
}

// Header asserts that the response header with the given key equals the expected value.
// Only checks the first value if multiple values exist.
//
// Example:
//
//	zhtest.AssertWith(t, w).Header("Content-Type", "application/json")
func (a *Assertions) Header(key, expected string) *Assertions {
	actual := a.resp.Header().Get(key)
	if actual != expected {
		a.fail("expected header %q to be %q, got %q", key, expected, actual)
	}
	return a
}

// HeaderContains asserts that the response header with the given key contains the substring.
//
// Example:
//
//	zhtest.AssertWith(t, w).HeaderContains("Content-Type", "json")
func (a *Assertions) HeaderContains(key, substring string) *Assertions {
	actual := a.resp.Header().Get(key)
	if !strings.Contains(actual, substring) {
		a.fail("expected header %q to contain %q, got %q", key, substring, actual)
	}
	return a
}

// HeaderExists asserts that the response header with the given key is present.
//
// Example:
//
//	zhtest.AssertWith(t, w).HeaderExists("X-Request-ID")
func (a *Assertions) HeaderExists(key string) *Assertions {
	if a.resp.Header().Get(key) == "" {
		a.fail("expected header %q to exist, but it was missing or empty", key)
	}
	return a
}

// HeaderNotExists asserts that the response header with the given key is not present.
//
// Example:
//
//	zhtest.AssertWith(t, w).HeaderNotExists("X-Powered-By")
func (a *Assertions) HeaderNotExists(key string) *Assertions {
	if a.resp.Header().Get(key) != "" {
		a.fail("expected header %q to not exist, but it was present", key)
	}
	return a
}

// Body asserts that the response body equals the expected string.
//
// Example:
//
//	zhtest.AssertWith(t, w).Body("Hello, World!")
func (a *Assertions) Body(expected string) *Assertions {
	actual := a.resp.Body.String()
	if actual != expected {
		a.fail("expected body %q, got %q", expected, actual)
	}
	return a
}

// BodyContains asserts that the response body contains the substring.
//
// Example:
//
//	zhtest.AssertWith(t, w).BodyContains("success")
func (a *Assertions) BodyContains(substring string) *Assertions {
	actual := a.resp.Body.String()
	if !strings.Contains(actual, substring) {
		a.fail("expected body to contain %q, got %q", substring, actual)
	}
	return a
}

// BodyNotContains asserts that the response body does not contain the substring.
//
// Example:
//
//	zhtest.AssertWith(t, w).BodyNotContains("error")
func (a *Assertions) BodyNotContains(substring string) *Assertions {
	actual := a.resp.Body.String()
	if strings.Contains(actual, substring) {
		a.fail("expected body to not contain %q, but it did", substring)
	}
	return a
}

// BodyEmpty asserts that the response body is empty.
func (a *Assertions) BodyEmpty() *Assertions {
	if a.resp.Body.Len() > 0 {
		a.fail("expected body to be empty, got %q", a.resp.Body.String())
	}
	return a
}

// BodyNotEmpty asserts that the response body is not empty.
func (a *Assertions) BodyNotEmpty() *Assertions {
	if a.resp.Body.Len() == 0 {
		a.fail("expected body to not be empty")
	}
	return a
}

// JSON asserts that the response body is valid JSON and decodes it into v.
// This is useful when you want to decode and inspect the result.
//
// Example:
//
//	var user User
//	zhtest.AssertWith(t, w).JSON(&user)
func (a *Assertions) JSON(v any) *Assertions {
	if err := json.Unmarshal(a.resp.Body.Bytes(), v); err != nil {
		a.fail("failed to decode JSON: %v\nbody: %s", err, a.resp.Body.String())
	}
	return a
}

// JSONEq asserts that the response body JSON equals the expected JSON string.
// Both are unmarshaled and compared semantically, ignoring whitespace/formatting.
//
// Example:
//
//	zhtest.AssertWith(t, w).JSONEq(`{"name": "John"}`)
func (a *Assertions) JSONEq(expected string) *Assertions {
	var expectedVal, actualVal any
	if err := json.Unmarshal([]byte(expected), &expectedVal); err != nil {
		a.fail("failed to unmarshal expected JSON: %v", err)
		return a
	}
	if err := json.Unmarshal(a.resp.Body.Bytes(), &actualVal); err != nil {
		a.fail("failed to decode response JSON: %v\nbody: %s", err, a.resp.Body.String())
		return a
	}

	if !jsonValuesEqual(expectedVal, actualVal) {
		a.fail("expected JSON %s, got %s", expected, a.resp.Body.String())
	}
	return a
}

// jsonMapsEqual compares two maps for equality recursively.
func jsonMapsEqual(a, b map[string]any) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		bv, ok := b[k]
		if !ok {
			return false
		}
		if !jsonValuesEqual(v, bv) {
			return false
		}
	}
	return true
}

// jsonValuesEqual compares two JSON values for equality.
func jsonValuesEqual(a, b any) bool {
	switch av := a.(type) {
	case map[string]any:
		bv, ok := b.(map[string]any)
		if !ok {
			return false
		}
		return jsonMapsEqual(av, bv)
	case []any:
		bv, ok := b.([]any)
		if !ok {
			return false
		}
		if len(av) != len(bv) {
			return false
		}
		for i := range av {
			if !jsonValuesEqual(av[i], bv[i]) {
				return false
			}
		}
		return true
	default:
		return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
	}
}

// JSONPathEqual asserts that the value at the given JSON path equals the expected value.
// Uses simple dot notation (e.g., "user.name", "items.0.id").
//
// Example:
//
//	zhtest.AssertWith(t, w).JSONPathEqual("user.name", "John")
func (a *Assertions) JSONPathEqual(path string, expected any) *Assertions {
	var data map[string]any
	if err := json.Unmarshal(a.resp.Body.Bytes(), &data); err != nil {
		a.fail("failed to decode JSON: %v\nbody: %s", err, a.resp.Body.String())
		return a
	}

	value, err := getJSONPath(data, path)
	if err != nil {
		a.fail("JSON path error: %v", err)
		return a
	}

	if fmt.Sprintf("%v", value) != fmt.Sprintf("%v", expected) {
		a.fail("expected JSON path %q to be %v, got %v", path, expected, value)
	}
	return a
}

// getJSONPath retrieves a value from a JSON structure using dot notation.
func getJSONPath(data map[string]any, path string) (any, error) {
	parts := strings.Split(path, ".")
	current := any(data)

	for _, part := range parts {
		switch v := current.(type) {
		case map[string]any:
			next, ok := v[part]
			if !ok {
				return nil, fmt.Errorf("key %q not found", part)
			}
			current = next
		case []any:
			// Try to parse as array index
			var index int
			if _, err := fmt.Sscanf(part, "%d", &index); err != nil {
				return nil, fmt.Errorf("expected array index, got %q", part)
			}
			if index < 0 || index >= len(v) {
				return nil, fmt.Errorf("index %d out of bounds", index)
			}
			current = v[index]
		default:
			return nil, fmt.Errorf("cannot traverse %T", current)
		}
	}

	return current, nil
}

// Cookie asserts that a cookie with the given name exists and has the expected value.
//
// Example:
//
//	zhtest.AssertWith(t, w).Cookie("session", "abc123")
func (a *Assertions) Cookie(name, expected string) *Assertions {
	cookie := a.resp.Cookie(name)
	if cookie == nil {
		a.fail("expected cookie %q to exist, but it was not found", name)
		return a
	}
	if cookie.Value != expected {
		a.fail("expected cookie %q to be %q, got %q", name, expected, cookie.Value)
	}
	return a
}

// CookieExists asserts that a cookie with the given name exists.
//
// Example:
//
//	zhtest.AssertWith(t, w).CookieExists("session")
func (a *Assertions) CookieExists(name string) *Assertions {
	if a.resp.Cookie(name) == nil {
		a.fail("expected cookie %q to exist, but it was not found", name)
	}
	return a
}

// CookieNotExists asserts that a cookie with the given name does not exist.
//
// Example:
//
//	zhtest.AssertWith(t, w).CookieNotExists("session")
func (a *Assertions) CookieNotExists(name string) *Assertions {
	if a.resp.Cookie(name) != nil {
		a.fail("expected cookie %q to not exist, but it was found", name)
	}
	return a
}

// Redirect asserts that the response is a redirect to the given location.
//
// Example:
//
//	zhtest.AssertWith(t, w).Redirect("/login")
func (a *Assertions) Redirect(location string) *Assertions {
	if a.resp.Code < 300 || a.resp.Code >= 400 {
		a.fail("expected redirect status (3xx), got %d", a.resp.Code)
		return a
	}
	actual := a.resp.Header().Get("Location")
	if actual != location {
		a.fail("expected redirect to %q, got %q", location, actual)
	}
	return a
}

// IsSuccess asserts that the response status is 2xx.
func (a *Assertions) IsSuccess() *Assertions {
	if !a.resp.IsSuccess() {
		a.fail("expected success status (2xx), got %d", a.resp.Code)
	}
	return a
}

// IsClientError asserts that the response status is 4xx.
func (a *Assertions) IsClientError() *Assertions {
	if !a.resp.IsClientError() {
		a.fail("expected client error status (4xx), got %d", a.resp.Code)
	}
	return a
}

// IsServerError asserts that the response status is 5xx.
func (a *Assertions) IsServerError() *Assertions {
	if !a.resp.IsServerError() {
		a.fail("expected server error status (5xx), got %d", a.resp.Code)
	}
	return a
}

// IsProblemDetail asserts that the response Content-Type is application/problem+json.
//
// Example:
//
//	zhtest.AssertWith(t, w).IsProblemDetail()
func (a *Assertions) IsProblemDetail() *Assertions {
	contentType := a.resp.Header().Get("Content-Type")
	if contentType != "application/problem+json" {
		a.fail("expected Content-Type application/problem+json, got %s", contentType)
	}
	return a
}

// ProblemDetailStatus asserts that the response is a Problem Detail with the given status.
//
// Example:
//
//	zhtest.AssertWith(t, w).ProblemDetailStatus(400)
func (a *Assertions) ProblemDetailStatus(expected int) *Assertions {
	a.IsProblemDetail()

	var problem struct {
		Status int `json:"status"`
	}
	if err := a.resp.JSON(&problem); err != nil {
		a.fail("failed to decode Problem Detail JSON: %v", err)
		return a
	}

	if problem.Status != expected {
		a.fail("expected Problem Detail status %d, got %d", expected, problem.Status)
	}
	return a
}

// ProblemDetailTitle asserts that the response is a Problem Detail with the given title.
//
// Example:
//
//	zhtest.AssertWith(t, w).ProblemDetailTitle("Bad Request")
func (a *Assertions) ProblemDetailTitle(expected string) *Assertions {
	a.IsProblemDetail()

	var problem struct {
		Title string `json:"title"`
	}
	if err := a.resp.JSON(&problem); err != nil {
		a.fail("failed to decode Problem Detail JSON: %v", err)
		return a
	}

	if problem.Title != expected {
		a.fail("expected Problem Detail title %q, got %q", expected, problem.Title)
	}
	return a
}

// ProblemDetailDetail asserts that the response is a Problem Detail with the given detail message.
//
// Example:
//
//	zhtest.AssertWith(t, w).ProblemDetailDetail("The request was invalid")
func (a *Assertions) ProblemDetailDetail(expected string) *Assertions {
	a.IsProblemDetail()

	var problem struct {
		Detail string `json:"detail"`
	}
	if err := a.resp.JSON(&problem); err != nil {
		a.fail("failed to decode Problem Detail JSON: %v", err)
		return a
	}

	if problem.Detail != expected {
		a.fail("expected Problem Detail detail %q, got %q", expected, problem.Detail)
	}
	return a
}

// ProblemDetailType asserts that the response is a Problem Detail with the given type URI.
//
// Example:
//
//	zhtest.AssertWith(t, w).ProblemDetailType("https://api.example.com/errors/invalid-request")
func (a *Assertions) ProblemDetailType(expected string) *Assertions {
	a.IsProblemDetail()

	var problem struct {
		Type string `json:"type"`
	}
	if err := a.resp.JSON(&problem); err != nil {
		a.fail("failed to decode Problem Detail JSON: %v", err)
		return a
	}

	if problem.Type != expected {
		a.fail("expected Problem Detail type %q, got %q", expected, problem.Type)
	}
	return a
}

// ProblemDetailExtension asserts that the response is a Problem Detail with the given extension field value.
//
// Example:
//
//	zhtest.AssertWith(t, w).ProblemDetailExtension("errors", []any{"field required"})
func (a *Assertions) ProblemDetailExtension(key string, expected any) *Assertions {
	a.IsProblemDetail()

	var problem map[string]any
	if err := a.resp.JSON(&problem); err != nil {
		a.fail("failed to decode Problem Detail JSON: %v", err)
		return a
	}

	value, ok := problem[key]
	if !ok {
		a.fail("expected Problem Detail extension %q to exist", key)
		return a
	}

	if fmt.Sprintf("%v", value) != fmt.Sprintf("%v", expected) {
		a.fail("expected Problem Detail extension %q to be %v, got %v", key, expected, value)
	}
	return a
}

// ProblemDetail decodes the response as a Problem Detail and stores it in v.
//
// Example:
//
//	var problem zerohttp.ProblemDetail
//	zhtest.AssertWith(t, w).ProblemDetail(&problem)
func (a *Assertions) ProblemDetail(v any) *Assertions {
	a.IsProblemDetail()

	if err := a.resp.JSON(v); err != nil {
		a.fail("failed to decode Problem Detail JSON: %v", err)
	}
	return a
}
