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
		if !ok {
			t.Errorf("expected to find value in context, got none")
		}
		if got != value {
			t.Errorf("got %q, want %q", got, value)
		}
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
				if !ok {
					t.Errorf("expected to find string value")
				}
				if str != "hello" {
					t.Errorf("got %q, want %q", str, "hello")
				}

				// Test int value
				num, ok := Get[int](r, "intKey")
				if !ok {
					t.Errorf("expected to find int value")
				}
				if num != 42 {
					t.Errorf("got %d, want %d", num, 42)
				}
			}),
		),
	)

	zhtest.Serve(handler, req)
}

func TestGetContextValue_NotFound(t *testing.T) {
	req := zhtest.NewRequest(http.MethodGet, "/").Build()

	// Try to get a value that does not exist
	got, ok := Get[string](req, "missingKey")
	if ok {
		t.Errorf("expected ok to be false when key is missing")
	}
	var zero string
	if got != zero {
		t.Errorf("expected zero value for type string, got %v", got)
	}
}

func TestGetContextValue_WrongType(t *testing.T) {
	req := zhtest.NewRequest(http.MethodGet, "/").Build()

	handler := With("key", "stringValue")(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Try to get string value as int - should fail
			got, ok := Get[int](r, "key")
			if ok {
				t.Errorf("expected ok to be false when type assertion fails")
			}
			if got != 0 {
				t.Errorf("expected zero value for int, got %v", got)
			}
		}),
	)

	zhtest.Serve(handler, req)
}

func TestGetContextValue_NilValue(t *testing.T) {
	req := zhtest.NewRequest(http.MethodGet, "/").Build()

	handler := With("key", nil)(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			got, ok := Get[string](r, "key")
			if ok {
				t.Errorf("expected ok to be false when value is nil")
			}
			var zero string
			if got != zero {
				t.Errorf("expected zero value for string, got %v", got)
			}
		}),
	)

	zhtest.Serve(handler, req)
}
