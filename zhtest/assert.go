package zhtest

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/alexferl/zerohttp/httpx"
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
//	zhtest.AssertWith(t, w).httpx.HeaderContentType, "application/json")
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
//	zhtest.AssertWith(t, w).HeaderContains(httpx.HeaderContentType, "json")
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

// JSONPathNotEqual asserts that the value at the given JSON path is not equal to the unexpected value.
// Uses simple dot notation (e.g., "user.name", "items.0.id").
//
// Example:
//
//	zhtest.AssertWith(t, w).JSONPathNotEqual("user.name", "Jane")
func (a *Assertions) JSONPathNotEqual(path string, unexpected any) *Assertions {
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

	if fmt.Sprintf("%v", value) == fmt.Sprintf("%v", unexpected) {
		a.fail("expected JSON path %q to not be %v", path, unexpected)
	}
	return a
}

// JSONPathExists asserts that the given JSON path exists in the response.
// Uses simple dot notation (e.g., "user.name", "items.0.id").
//
// Example:
//
//	zhtest.AssertWith(t, w).JSONPathExists("user.name")
func (a *Assertions) JSONPathExists(path string) *Assertions {
	var data map[string]any
	if err := json.Unmarshal(a.resp.Body.Bytes(), &data); err != nil {
		a.fail("failed to decode JSON: %v\nbody: %s", err, a.resp.Body.String())
		return a
	}

	_, err := getJSONPath(data, path)
	if err != nil {
		a.fail("expected JSON path %q to exist, but it was not found", path)
	}
	return a
}

// JSONPathNotExists asserts that the given JSON path does not exist in the response.
// Uses simple dot notation (e.g., "user.password", "items.0.secret").
//
// Example:
//
//	zhtest.AssertWith(t, w).JSONPathNotExists("user.password")
func (a *Assertions) JSONPathNotExists(path string) *Assertions {
	var data map[string]any
	if err := json.Unmarshal(a.resp.Body.Bytes(), &data); err != nil {
		a.fail("failed to decode JSON: %v\nbody: %s", err, a.resp.Body.String())
		return a
	}

	_, err := getJSONPath(data, path)
	if err == nil {
		a.fail("expected JSON path %q to not exist, but it was found", path)
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
	actual := a.resp.Header().Get(httpx.HeaderLocation)
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
	contentType := a.resp.Header().Get(httpx.HeaderContentType)
	if contentType != httpx.MIMEApplicationProblemJSON {
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

// General Assertions

// fail reports a test failure if a testing.TB is available.
func fail(t testing.TB, format string, args ...any) {
	if t != nil {
		t.Helper()
		t.Errorf(format, args...)
	}
}

// AssertFail fails the test immediately with the given message.
// This is useful for timeout cases or unrecoverable test failures.
//
// Example:
//
//	select {
//	case <-done:
//	case <-time.After(time.Second):
//		zhtest.AssertFail(t, "timeout waiting for operation")
//	}
func AssertFail(t testing.TB, msg string) {
	t.Helper()
	t.Fatal(msg)
}

// AssertFailf fails the test immediately with a formatted message.
// This is useful for timeout cases or unrecoverable test failures with dynamic values.
//
// Example:
//
//	zhtest.AssertFailf(t, "timeout after %v waiting for %s", duration, name)
func AssertFailf(t testing.TB, format string, args ...any) {
	t.Helper()
	t.Fatalf(format, args...)
}

// AssertNoError fails if err is not nil.
// For automatic test failures, pass a non-nil testing.TB.
//
// Example:
//
//	err := someFunction()
//	zhtest.AssertNoError(t, err)
func AssertNoError(t testing.TB, err error) {
	if err != nil {
		fail(t, "expected no error, got: %v", err)
	}
}

// AssertError fails if err is nil.
// For automatic test failures, pass a non-nil testing.TB.
//
// Example:
//
//	err := someFunctionThatShouldFail()
//	zhtest.AssertError(t, err)
func AssertError(t testing.TB, err error) {
	if err == nil {
		fail(t, "expected an error, got nil")
	}
}

// AssertErrorIs fails if err does not match the target error using errors.Is().
// For automatic test failures, pass a non-nil testing.TB.
//
// Example:
//
//	err := os.Open("nonexistent")
//	zhtest.AssertErrorIs(t, err, os.ErrNotExist)
func AssertErrorIs(t testing.TB, err error, target error) {
	if !errors.Is(err, target) {
		fail(t, "expected error to be %v, got %v", target, err)
	}
}

// AssertErrorContains fails if err is nil or its message does not contain the substring.
// For automatic test failures, pass a non-nil testing.TB.
//
// Example:
//
//	err := errors.New("connection refused")
//	zhtest.AssertErrorContains(t, err, "refused")
func AssertErrorContains(t testing.TB, err error, substring string) {
	if err == nil {
		fail(t, "expected an error containing %q, got nil", substring)
		return
	}
	if !strings.Contains(err.Error(), substring) {
		fail(t, "expected error to contain %q, got %q", substring, err.Error())
	}
}

// AssertNil fails if v is not nil.
// For automatic test failures, pass a non-nil testing.TB.
//
// Example:
//
//	var ptr *MyStruct
//	zhtest.AssertNil(t, ptr)
func AssertNil(t testing.TB, v any) {
	if v != nil {
		// Handle wrapped nil values (typed nil pointers, interfaces, etc.)
		rv := reflect.ValueOf(v)
		switch rv.Kind() {
		case reflect.Ptr, reflect.Slice, reflect.Map, reflect.Chan, reflect.Func, reflect.Interface:
			if !rv.IsNil() {
				fail(t, "expected nil, got %v", v)
			}
		default:
			fail(t, "expected nil, got %v", v)
		}
	}
}

// AssertNotNil fails if v is nil.
// For automatic test failures, pass a non-nil testing.TB.
//
// Example:
//
//	result := someFunction()
//	zhtest.AssertNotNil(t, result)
func AssertNotNil(t testing.TB, v any) {
	if v == nil {
		fail(t, "expected non-nil value, got nil")
		return
	}
	// Handle wrapped nil values (typed nil pointers, interfaces, etc.)
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Ptr, reflect.Slice, reflect.Map, reflect.Chan, reflect.Func, reflect.Interface:
		if rv.IsNil() {
			fail(t, "expected non-nil value, got nil")
		}
	}
}

// AssertEqual fails if expected and actual are not equal.
// Uses reflect.DeepEqual for proper comparison of slices, maps, and structs.
// For numeric types, performs value-based comparison allowing mixed int/float types.
// For automatic test failures, pass a non-nil testing.TB.
//
// Example:
//
//	zhtest.AssertEqual(t, 42, result)
//	zhtest.AssertEqual(t, []int{1, 2, 3}, result)
func AssertEqual(t testing.TB, expected, actual any) {
	// Try numeric comparison first for mixed numeric types
	if isNumeric(expected) && isNumeric(actual) {
		if !numericEqual(expected, actual) {
			fail(t, "expected %v, got %v", expected, actual)
		}
		return
	}
	if !reflect.DeepEqual(expected, actual) {
		fail(t, "expected %v, got %v", expected, actual)
	}
}

// isNumeric returns true if v is a numeric type.
func isNumeric(v any) bool {
	switch v.(type) {
	case int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float32, float64:
		return true
	default:
		return false
	}
}

// numericEqual compares two numeric values for equality.
// Converts both to float64 for comparison to handle mixed types.
func numericEqual(a, b any) bool {
	aFloat := toFloat64Generic(a)
	bFloat := toFloat64Generic(b)
	return aFloat == bFloat
}

// toFloat64Generic converts any numeric value to float64.
func toFloat64Generic(v any) float64 {
	switch val := v.(type) {
	case int:
		return float64(val)
	case int8:
		return float64(val)
	case int16:
		return float64(val)
	case int32:
		return float64(val)
	case int64:
		return float64(val)
	case uint:
		return float64(val)
	case uint8:
		return float64(val)
	case uint16:
		return float64(val)
	case uint32:
		return float64(val)
	case uint64:
		return float64(val)
	case float32:
		return float64(val)
	case float64:
		return val
	default:
		return 0
	}
}

// AssertNotEqual fails if unexpected and actual are equal.
// Uses reflect.DeepEqual for proper comparison of slices, maps, and structs.
// For numeric types, performs value-based comparison allowing mixed int/float types.
// For automatic test failures, pass a non-nil testing.TB.
//
// Example:
//
//	zhtest.AssertNotEqual(t, "old", result)
func AssertNotEqual(t testing.TB, unexpected, actual any) {
	// Try numeric comparison first for mixed numeric types
	if isNumeric(unexpected) && isNumeric(actual) {
		if numericEqual(unexpected, actual) {
			fail(t, "expected value not to be %v", unexpected)
		}
		return
	}
	if reflect.DeepEqual(unexpected, actual) {
		fail(t, "expected value not to be %v", unexpected)
	}
}

// AssertDeepEqual fails if expected and actual are not deeply equal using reflect.DeepEqual.
// Use this for slices, maps, and structs.
// For automatic test failures, pass a non-nil testing.TB.
//
// Example:
//
//	expected := []int{1, 2, 3}
//	zhtest.AssertDeepEqual(t, expected, result)
func AssertDeepEqual(t testing.TB, expected, actual any) {
	if !reflect.DeepEqual(expected, actual) {
		fail(t, "expected %v, got %v", expected, actual)
	}
}

// AssertGreater fails if actual <= expected.
// For automatic test failures, pass a non-nil testing.TB.
//
// Example:
//
//	zhtest.AssertGreater(t, 10, 5)
func AssertGreater(t testing.TB, actual, expected any) {
	// Try to compare as int64
	aInt, aIsInt := toInt64(actual)
	bInt, bIsInt := toInt64(expected)
	if aIsInt && bIsInt {
		if aInt <= bInt {
			fail(t, "expected %v to be greater than %v", actual, expected)
		}
		return
	}

	// Try to compare as float64 (handles float/float and mixed int/float)
	aFloat, aIsFloat := toFloat64(actual)
	bFloat, bIsFloat := toFloat64(expected)

	// Convert ints to floats for mixed comparisons
	if aIsInt {
		aFloat = float64(aInt)
		aIsFloat = true
	}
	if bIsInt {
		bFloat = float64(bInt)
		bIsFloat = true
	}

	if aIsFloat && bIsFloat {
		if aFloat <= bFloat {
			fail(t, "expected %v to be greater than %v", actual, expected)
		}
		return
	}

	fail(t, "AssertGreater requires comparable numeric types, got %T and %T", actual, expected)
}

// AssertLess fails if actual >= expected.
// For automatic test failures, pass a non-nil testing.TB.
//
// Example:
//
//	zhtest.AssertLess(t, 5, 10)
func AssertLess(t testing.TB, actual, expected any) {
	// Try to compare as int64
	aInt, aIsInt := toInt64(actual)
	bInt, bIsInt := toInt64(expected)
	if aIsInt && bIsInt {
		if aInt >= bInt {
			fail(t, "expected %v to be less than %v", actual, expected)
		}
		return
	}

	// Try to compare as float64 (handles float/float and mixed int/float)
	aFloat, aIsFloat := toFloat64(actual)
	bFloat, bIsFloat := toFloat64(expected)

	// Convert ints to floats for mixed comparisons
	if aIsInt {
		aFloat = float64(aInt)
		aIsFloat = true
	}
	if bIsInt {
		bFloat = float64(bInt)
		bIsFloat = true
	}

	if aIsFloat && bIsFloat {
		if aFloat >= bFloat {
			fail(t, "expected %v to be less than %v", actual, expected)
		}
		return
	}

	fail(t, "AssertLess requires comparable numeric types, got %T and %T", actual, expected)
}

// toInt64 attempts to convert a value to int64.
func toInt64(v any) (int64, bool) {
	switch val := v.(type) {
	case int:
		return int64(val), true
	case int8:
		return int64(val), true
	case int16:
		return int64(val), true
	case int32:
		return int64(val), true
	case int64:
		return val, true
	case uint:
		return int64(val), true
	case uint8:
		return int64(val), true
	case uint16:
		return int64(val), true
	case uint32:
		return int64(val), true
	case uint64:
		return int64(val), true
	default:
		return 0, false
	}
}

// toFloat64 attempts to convert a value to float64.
func toFloat64(v any) (float64, bool) {
	switch val := v.(type) {
	case float32:
		return float64(val), true
	case float64:
		return val, true
	default:
		return 0, false
	}
}

// AssertTrue fails if condition is false.
// For automatic test failures, pass a non-nil testing.TB.
//
// Example:
//
//	zhtest.AssertTrue(t, len(items) > 0)
func AssertTrue(t testing.TB, condition bool) {
	if !condition {
		fail(t, "expected condition to be true")
	}
}

// AssertFalse fails if condition is true.
// For automatic test failures, pass a non-nil testing.TB.
//
// Example:
//
//	zhtest.AssertFalse(t, len(items) == 0)
func AssertFalse(t testing.TB, condition bool) {
	if condition {
		fail(t, "expected condition to be false")
	}
}

// AssertEmpty fails if s is not empty.
// Works with strings, slices, maps, and arrays.
// For automatic test failures, pass a non-nil testing.TB.
//
// Example:
//
//	zhtest.AssertEmpty(t, "")
//	zhtest.AssertEmpty(t, []int{})
func AssertEmpty(t testing.TB, s any) {
	if !isEmpty(s) {
		fail(t, "expected empty value, got %v", s)
	}
}

// AssertNotEmpty fails if s is empty.
// Works with strings, slices, maps, and arrays.
// For automatic test failures, pass a non-nil testing.TB.
//
// Example:
//
//	zhtest.AssertNotEmpty(t, "hello")
//	zhtest.AssertNotEmpty(t, []int{1, 2, 3})
func AssertNotEmpty(t testing.TB, s any) {
	if isEmpty(s) {
		fail(t, "expected non-empty value")
	}
}

// isEmpty checks if a value is empty.
func isEmpty(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.String, reflect.Slice, reflect.Map, reflect.Array, reflect.Chan:
		return rv.Len() == 0
	case reflect.Ptr, reflect.Interface:
		if rv.IsNil() {
			return true
		}
		// Dereference and check again
		return isEmpty(rv.Elem().Interface())
	}
	return false
}

// AssertLen fails if collection does not have the expected length.
// Works with strings, slices, maps, arrays, and channels.
// For automatic test failures, pass a non-nil testing.TB.
//
// Example:
//
//	zhtest.AssertLen(t, []int{1, 2, 3}, 3)
func AssertLen(t testing.TB, collection any, expectedLen int) {
	rv := reflect.ValueOf(collection)
	switch rv.Kind() {
	case reflect.String, reflect.Slice, reflect.Map, reflect.Array, reflect.Chan:
		actualLen := rv.Len()
		if actualLen != expectedLen {
			fail(t, "expected length %d, got %d", expectedLen, actualLen)
		}
	default:
		fail(t, "AssertLen requires a collection type, got %T", collection)
	}
}

// AssertContains fails if slice does not contain the element, or if s does not contain substring.
// For slices, uses reflect.DeepEqual for comparison.
// For strings, uses strings.Contains.
// For automatic test failures, pass a non-nil testing.TB.
//
// Example:
//
//	zhtest.AssertContains(t, []int{1, 2, 3}, 2)
//	zhtest.AssertContains(t, "hello world", "world")
func AssertContains(t testing.TB, s any, element any) {
	// Handle string containment
	str, isString := s.(string)
	if isString {
		substr, isSubstr := element.(string)
		if !isSubstr {
			fail(t, "AssertContains requires substring to be a string when s is a string, got %T", element)
			return
		}
		if !strings.Contains(str, substr) {
			fail(t, "expected %q to contain %q", str, substr)
		}
		return
	}

	rv := reflect.ValueOf(s)
	if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
		fail(t, "AssertContains requires a slice, array, or string, got %T", s)
		return
	}

	for i := 0; i < rv.Len(); i++ {
		if reflect.DeepEqual(rv.Index(i).Interface(), element) {
			return
		}
	}
	fail(t, "expected slice to contain %v", element)
}

// AssertNotContains fails if slice contains the element, or if s contains substring.
// For slices, uses reflect.DeepEqual for comparison.
// For strings, uses strings.Contains.
// For automatic test failures, pass a non-nil testing.TB.
//
// Example:
//
//	zhtest.AssertNotContains(t, []int{1, 2, 3}, 4)
//	zhtest.AssertNotContains(t, "hello world", "goodbye")
func AssertNotContains(t testing.TB, s any, element any) {
	// Handle string containment
	str, isString := s.(string)
	if isString {
		substr, isSubstr := element.(string)
		if !isSubstr {
			fail(t, "AssertNotContains requires substring to be a string when s is a string, got %T", element)
			return
		}
		if strings.Contains(str, substr) {
			fail(t, "expected %q to not contain %q", str, substr)
		}
		return
	}

	rv := reflect.ValueOf(s)
	if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
		fail(t, "AssertNotContains requires a slice, array, or string, got %T", s)
		return
	}

	for i := 0; i < rv.Len(); i++ {
		if reflect.DeepEqual(rv.Index(i).Interface(), element) {
			fail(t, "expected slice to not contain %v", element)
			return
		}
	}
}

// AssertIsType fails if actual is not of the expected type.
// For automatic test failures, pass a non-nil testing.TB.
//
// Example:
//
//	zhtest.AssertIsType(t, (*MyError)(nil), err)
func AssertIsType(t testing.TB, expectedType any, actual any) {
	expectedReflectType := reflect.TypeOf(expectedType)
	actualReflectType := reflect.TypeOf(actual)

	// Handle nil pointer types for expected (e.g., (*MyType)(nil))
	if expectedReflectType != nil && expectedReflectType.Kind() == reflect.Ptr && actualReflectType != nil {
		// If expected is a pointer type but actual is not, compare the underlying type
		if actualReflectType.Kind() != reflect.Ptr {
			expectedReflectType = expectedReflectType.Elem()
		}
	}

	if expectedReflectType != actualReflectType {
		if expectedReflectType == nil {
			fail(t, "expected type nil, got %v", actualReflectType)
		} else if actualReflectType == nil {
			fail(t, "expected type %v, got nil", expectedReflectType)
		} else {
			fail(t, "expected type %v, got %v", expectedReflectType, actualReflectType)
		}
	}
}

// AssertImplements fails if actual does not implement the interfaceType.
// The interfaceType should be a pointer to an interface (e.g., (*io.Reader)(nil)).
// For automatic test failures, pass a non-nil testing.TB.
//
// Example:
//
//	zhtest.AssertImplements(t, (*io.Reader)(nil), myReader)
func AssertImplements(t testing.TB, interfaceType any, actual any) {
	interfaceReflectType := reflect.TypeOf(interfaceType)

	if interfaceReflectType == nil || interfaceReflectType.Kind() != reflect.Ptr || interfaceReflectType.Elem().Kind() != reflect.Interface {
		fail(t, "AssertImplements requires a pointer to an interface as the first argument (e.g., (*io.Reader)(nil)), got %T", interfaceType)
		return
	}

	iface := interfaceReflectType.Elem()
	actualReflectType := reflect.TypeOf(actual)

	if actualReflectType == nil {
		fail(t, "expected type to implement %v, but got nil", iface)
		return
	}

	if !actualReflectType.Implements(iface) {
		fail(t, "expected type %v to implement %v", actualReflectType, iface)
	}
}

// AssertPanic fails if f does not panic.
// For automatic test failures, pass a non-nil testing.TB.
//
// Example:
//
//	zhtest.AssertPanic(t, func() {
//	    someFunctionThatPanics()
//	})
func AssertPanic(t testing.TB, f func()) {
	panicked := false
	func() {
		defer func() {
			if recover() != nil {
				panicked = true
			}
		}()
		f()
	}()
	if !panicked {
		fail(t, "expected function to panic, but it did not")
	}
}

// AssertNoPanic fails if f panics.
// For automatic test failures, pass a non-nil testing.TB.
//
// Example:
//
//	zhtest.AssertNoPanic(t, func() {
//	    someFunctionThatShouldNotPanic()
//	})
func AssertNoPanic(t testing.TB, f func()) {
	panicked := false
	var panicValue any
	func() {
		defer func() {
			if r := recover(); r != nil {
				panicked = true
				panicValue = r
			}
		}()
		f()
	}()
	if panicked {
		fail(t, "expected function not to panic, but it panicked with: %v", panicValue)
	}
}

// AssertPanicContains fails if f does not panic with a message containing substring.
// For automatic test failures, pass a non-nil testing.TB.
//
// Example:
//
//	zhtest.AssertPanicContains(t, func() {
//	    someFunctionThatPanicsWithMessage()
//	}, "expected error message")
func AssertPanicContains(t testing.TB, f func(), substring string) {
	panicked := false
	var panicValue any
	func() {
		defer func() {
			if r := recover(); r != nil {
				panicked = true
				panicValue = r
			}
		}()
		f()
	}()
	if !panicked {
		fail(t, "expected function to panic, but it did not")
		return
	}
	panicStr := fmt.Sprintf("%v", panicValue)
	if !strings.Contains(panicStr, substring) {
		fail(t, "expected panic message to contain %q, got %q", substring, panicStr)
	}
}
