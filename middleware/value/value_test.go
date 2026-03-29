package value

import (
	"net/http"
	"testing"

	"github.com/alexferl/zerohttp/zhtest"
)

func TestWithValue(t *testing.T) {
	const key = "testKey"
	const value = "testValue"

	// Next handler verifies value is correctly set in context
	handler := With(key, value)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got, ok := Get[string](r, key)
		zhtest.AssertTrue(t, ok)
		zhtest.AssertEqual(t, value, got)
		w.WriteHeader(http.StatusOK)
	}))

	req := zhtest.NewRequest(http.MethodGet, "/").Build()
	w := zhtest.Serve(handler, req)

	zhtest.AssertWith(t, w).Status(http.StatusOK)
}

func TestGetContextValue_Found(t *testing.T) {
	req := zhtest.NewRequest(http.MethodGet, "/").Build()

	// Set values using WithValue middleware
	handler := With("stringKey", "hello")(
		With("intKey", 42)(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Test string value
				str, ok := Get[string](r, "stringKey")
				zhtest.AssertTrue(t, ok)
				zhtest.AssertEqual(t, "hello", str)

				// Test int value
				num, ok := Get[int](r, "intKey")
				zhtest.AssertTrue(t, ok)
				zhtest.AssertEqual(t, 42, num)
			}),
		),
	)

	zhtest.Serve(handler, req)
}

func TestGetContextValue_NotFound(t *testing.T) {
	req := zhtest.NewRequest(http.MethodGet, "/").Build()

	// Try to get a value that does not exist
	got, ok := Get[string](req, "missingKey")
	zhtest.AssertFalse(t, ok)
	var zero string
	zhtest.AssertEqual(t, zero, got)
}

func TestGetContextValue_WrongType(t *testing.T) {
	req := zhtest.NewRequest(http.MethodGet, "/").Build()

	handler := With("key", "stringValue")(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Try to get string value as int - should fail
			got, ok := Get[int](r, "key")
			zhtest.AssertFalse(t, ok)
			zhtest.AssertEqual(t, 0, got)
		}),
	)

	zhtest.Serve(handler, req)
}

func TestGetContextValue_NilValue(t *testing.T) {
	req := zhtest.NewRequest(http.MethodGet, "/").Build()

	handler := With("key", nil)(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			got, ok := Get[string](r, "key")
			zhtest.AssertFalse(t, ok)
			var zero string
			zhtest.AssertEqual(t, zero, got)
		}),
	)

	zhtest.Serve(handler, req)
}
