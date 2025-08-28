package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWithValue(t *testing.T) {
	const key = "testKey"
	const value = "testValue"

	// Next handler verifies value is correctly set in context
	handler := WithValue(key, value)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got, ok := GetContextValue[string](r, key)
		if !ok {
			t.Errorf("expected to find value in context, got none")
		}
		if got != value {
			t.Errorf("got %q, want %q", got, value)
		}
		w.WriteHeader(http.StatusOK)
	}))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	handler.ServeHTTP(rr, req)

	if status := rr.Result().StatusCode; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
}

func TestGetContextValue_Found(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	// Set values using WithValue middleware
	handler := WithValue("stringKey", "hello")(
		WithValue("intKey", 42)(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Test string value
				str, ok := GetContextValue[string](r, "stringKey")
				if !ok {
					t.Errorf("expected to find string value")
				}
				if str != "hello" {
					t.Errorf("got %q, want %q", str, "hello")
				}

				// Test int value
				num, ok := GetContextValue[int](r, "intKey")
				if !ok {
					t.Errorf("expected to find int value")
				}
				if num != 42 {
					t.Errorf("got %d, want %d", num, 42)
				}
			}),
		),
	)

	handler.ServeHTTP(httptest.NewRecorder(), req)
}

func TestGetContextValue_NotFound(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	// Try to get a value that does not exist
	got, ok := GetContextValue[string](req, "missingKey")
	if ok {
		t.Errorf("expected ok to be false when key is missing")
	}
	var zero string
	if got != zero {
		t.Errorf("expected zero value for type string, got %v", got)
	}
}

func TestGetContextValue_WrongType(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	handler := WithValue("key", "stringValue")(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Try to get string value as int - should fail
			got, ok := GetContextValue[int](r, "key")
			if ok {
				t.Errorf("expected ok to be false when type assertion fails")
			}
			if got != 0 {
				t.Errorf("expected zero value for int, got %v", got)
			}
		}),
	)

	handler.ServeHTTP(httptest.NewRecorder(), req)
}

func TestGetContextValue_NilValue(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	handler := WithValue("key", nil)(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			got, ok := GetContextValue[string](r, "key")
			if ok {
				t.Errorf("expected ok to be false when value is nil")
			}
			var zero string
			if got != zero {
				t.Errorf("expected zero value for string, got %v", got)
			}
		}),
	)

	handler.ServeHTTP(httptest.NewRecorder(), req)
}
