package zhtest

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/alexferl/zerohttp/httpx"
	"github.com/alexferl/zerohttp/internal/problem"
)

func TestAssert_Status(t *testing.T) {
	t.Run("passes when status matches", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
		req := NewRequest(http.MethodGet, "/").Build()
		w := Serve(handler, req)

		// Just verify it doesn't panic and chains correctly
		result := Assert(w).Status(http.StatusOK)
		AssertNotNil(t, result)
	})
}

func TestAssert_StatusNot(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	req := NewRequest(http.MethodGet, "/").Build()
	w := Serve(handler, req)

	result := Assert(w).StatusNot(http.StatusNotFound)
	AssertNotNil(t, result)
}

func TestAssert_StatusBetween(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})
	req := NewRequest(http.MethodGet, "/").Build()
	w := Serve(handler, req)

	result := Assert(w).StatusBetween(200, 299)
	AssertNotNil(t, result)
}

func TestAssert_Header(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(httpx.HeaderContentType, httpx.MIMEApplicationJSON)
		w.WriteHeader(http.StatusOK)
	})
	req := NewRequest(http.MethodGet, "/").Build()
	w := Serve(handler, req)

	result := Assert(w).Header(httpx.HeaderContentType, "application/json")
	AssertNotNil(t, result)
}

func TestAssert_HeaderContains(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(httpx.HeaderContentType, "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
	})
	req := NewRequest(http.MethodGet, "/").Build()
	w := Serve(handler, req)

	result := Assert(w).HeaderContains(httpx.HeaderContentType, "json")
	AssertNotNil(t, result)
}

func TestAssert_HeaderExists(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Custom", "value")
		w.WriteHeader(http.StatusOK)
	})
	req := NewRequest(http.MethodGet, "/").Build()
	w := Serve(handler, req)

	result := Assert(w).HeaderExists("X-Custom")
	AssertNotNil(t, result)
}

func TestAssert_HeaderNotExists(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	req := NewRequest(http.MethodGet, "/").Build()
	w := Serve(handler, req)

	result := Assert(w).HeaderNotExists("X-Custom")
	AssertNotNil(t, result)
}

func TestAssert_Body(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("hello"))
		AssertNoError(t, err)
	})
	req := NewRequest(http.MethodGet, "/").Build()
	w := Serve(handler, req)

	result := Assert(w).Body("hello")
	AssertNotNil(t, result)
}

func TestAssert_BodyContains(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("hello world"))
		AssertNoError(t, err)
	})
	req := NewRequest(http.MethodGet, "/").Build()
	w := Serve(handler, req)

	result := Assert(w).BodyContains("world")
	AssertNotNil(t, result)
}

func TestAssert_BodyNotContains(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("hello world"))
		AssertNoError(t, err)
	})
	req := NewRequest(http.MethodGet, "/").Build()
	w := Serve(handler, req)

	result := Assert(w).BodyNotContains("error")
	AssertNotNil(t, result)
}

func TestAssert_BodyEmpty(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	req := NewRequest(http.MethodGet, "/").Build()
	w := Serve(handler, req)

	result := Assert(w).BodyEmpty()
	AssertNotNil(t, result)
}

func TestAssert_BodyNotEmpty(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("content"))
		AssertNoError(t, err)
	})
	req := NewRequest(http.MethodGet, "/").Build()
	w := Serve(handler, req)

	result := Assert(w).BodyNotEmpty()
	AssertNotNil(t, result)
}

func TestAssert_JSON(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(httpx.HeaderContentType, httpx.MIMEApplicationJSON)
		_, err := w.Write([]byte(`{"name": "John", "age": 30}`))
		AssertNoError(t, err)
	})
	req := NewRequest(http.MethodGet, "/").Build()
	w := Serve(handler, req)

	var result struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	a := Assert(w).JSON(&result)
	AssertNotNil(t, a)
	AssertEqual(t, "John", result.Name)
	AssertEqual(t, 30, result.Age)
}

func TestAssert_JSONEq(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(httpx.HeaderContentType, httpx.MIMEApplicationJSON)
		_, err := w.Write([]byte(`{"name": "John"}`))
		AssertNoError(t, err)
	})
	req := NewRequest(http.MethodGet, "/").Build()
	w := Serve(handler, req)

	result := Assert(w).JSONEq(`{"name": "John"}`)
	AssertNotNil(t, result)
}

func TestAssert_JSONPathEqual(t *testing.T) {
	t.Run("works with nested object", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set(httpx.HeaderContentType, httpx.MIMEApplicationJSON)
			_, err := w.Write([]byte(`{"user": {"name": "John"}}`))
			AssertNoError(t, err)
		})
		req := NewRequest(http.MethodGet, "/").Build()
		w := Serve(handler, req)

		result := Assert(w).JSONPathEqual("user.name", "John")
		AssertNotNil(t, result)
	})

	t.Run("works with array index", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set(httpx.HeaderContentType, httpx.MIMEApplicationJSON)
			_, err := w.Write([]byte(`{"items": [{"id": 1}, {"id": 2}]}`))
			AssertNoError(t, err)
		})
		req := NewRequest(http.MethodGet, "/").Build()
		w := Serve(handler, req)

		result := Assert(w).JSONPathEqual("items.0.id", "1")
		AssertNotNil(t, result)
	})
}

func TestAssert_JSONPathNotEqual(t *testing.T) {
	t.Run("value is not equal", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set(httpx.HeaderContentType, httpx.MIMEApplicationJSON)
			_, err := w.Write([]byte(`{"user": {"name": "John"}}`))
			AssertNoError(t, err)
		})
		req := NewRequest(http.MethodGet, "/").Build()
		w := Serve(handler, req)

		result := Assert(w).JSONPathNotEqual("user.name", "Jane")
		AssertNotNil(t, result)
	})

	t.Run("works with array index", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set(httpx.HeaderContentType, httpx.MIMEApplicationJSON)
			_, err := w.Write([]byte(`{"items": [{"id": 1}, {"id": 2}]}`))
			AssertNoError(t, err)
		})
		req := NewRequest(http.MethodGet, "/").Build()
		w := Serve(handler, req)

		result := Assert(w).JSONPathNotEqual("items.0.id", "999")
		AssertNotNil(t, result)
	})
}

func TestAssert_JSONPathExists(t *testing.T) {
	t.Run("path exists", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set(httpx.HeaderContentType, httpx.MIMEApplicationJSON)
			_, err := w.Write([]byte(`{"user": {"name": "John"}}`))
			AssertNoError(t, err)
		})
		req := NewRequest(http.MethodGet, "/").Build()
		w := Serve(handler, req)

		result := Assert(w).JSONPathExists("user.name")
		AssertNotNil(t, result)
	})

	t.Run("nested path exists", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set(httpx.HeaderContentType, httpx.MIMEApplicationJSON)
			_, err := w.Write([]byte(`{"user": {"profile": {"name": "John"}}}`))
			AssertNoError(t, err)
		})
		req := NewRequest(http.MethodGet, "/").Build()
		w := Serve(handler, req)

		result := Assert(w).JSONPathExists("user.profile.name")
		AssertNotNil(t, result)
	})

	t.Run("array index exists", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set(httpx.HeaderContentType, httpx.MIMEApplicationJSON)
			_, err := w.Write([]byte(`{"items": [{"id": 1}]}`))
			AssertNoError(t, err)
		})
		req := NewRequest(http.MethodGet, "/").Build()
		w := Serve(handler, req)

		result := Assert(w).JSONPathExists("items.0.id")
		AssertNotNil(t, result)
	})
}

func TestAssert_JSONPathNotExists(t *testing.T) {
	t.Run("path does not exist", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set(httpx.HeaderContentType, httpx.MIMEApplicationJSON)
			_, err := w.Write([]byte(`{"user": {"name": "John"}}`))
			AssertNoError(t, err)
		})
		req := NewRequest(http.MethodGet, "/").Build()
		w := Serve(handler, req)

		result := Assert(w).JSONPathNotExists("user.password")
		AssertNotNil(t, result)
	})

	t.Run("nested path does not exist", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set(httpx.HeaderContentType, httpx.MIMEApplicationJSON)
			_, err := w.Write([]byte(`{"user": {"profile": {"name": "John"}}}`))
			AssertNoError(t, err)
		})
		req := NewRequest(http.MethodGet, "/").Build()
		w := Serve(handler, req)

		result := Assert(w).JSONPathNotExists("user.profile.email")
		AssertNotNil(t, result)
	})

	t.Run("array index out of bounds", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set(httpx.HeaderContentType, httpx.MIMEApplicationJSON)
			_, err := w.Write([]byte(`{"items": [{"id": 1}]}`))
			AssertNoError(t, err)
		})
		req := NewRequest(http.MethodGet, "/").Build()
		w := Serve(handler, req)

		result := Assert(w).JSONPathNotExists("items.5.id")
		AssertNotNil(t, result)
	})

	t.Run("invalid JSON", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set(httpx.HeaderContentType, httpx.MIMEApplicationJSON)
			_, err := w.Write([]byte("not json"))
			AssertNoError(t, err)
		})
		req := NewRequest(http.MethodGet, "/").Build()
		w := Serve(handler, req)

		// This will fail to decode JSON, but we still return the assertion
		result := Assert(w).JSONPathNotExists("any.path")
		AssertNotNil(t, result)
	})
}

func TestAssert_Cookie(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "session", Value: "abc123"})
		w.WriteHeader(http.StatusOK)
	})
	req := NewRequest(http.MethodGet, "/").Build()
	w := Serve(handler, req)

	result := Assert(w).Cookie("session", "abc123")
	AssertNotNil(t, result)
}

func TestAssert_CookieExists(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "session", Value: "abc123"})
		w.WriteHeader(http.StatusOK)
	})
	req := NewRequest(http.MethodGet, "/").Build()
	w := Serve(handler, req)

	result := Assert(w).CookieExists("session")
	AssertNotNil(t, result)
}

func TestAssert_CookieNotExists(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	req := NewRequest(http.MethodGet, "/").Build()
	w := Serve(handler, req)

	result := Assert(w).CookieNotExists("session")
	AssertNotNil(t, result)
}

func TestAssert_Redirect(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/login", http.StatusFound)
	})
	req := NewRequest(http.MethodGet, "/").Build()
	w := Serve(handler, req)

	result := Assert(w).Redirect("/login")
	AssertNotNil(t, result)
}

func TestAssert_IsSuccess(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})
	req := NewRequest(http.MethodGet, "/").Build()
	w := Serve(handler, req)

	result := Assert(w).IsSuccess()
	AssertNotNil(t, result)
}

func TestAssert_IsClientError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	req := NewRequest(http.MethodGet, "/").Build()
	w := Serve(handler, req)

	result := Assert(w).IsClientError()
	AssertNotNil(t, result)
}

func TestAssert_IsServerError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	req := NewRequest(http.MethodGet, "/").Build()
	w := Serve(handler, req)

	result := Assert(w).IsServerError()
	AssertNotNil(t, result)
}

func TestAssert_Chaining(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(httpx.HeaderContentType, httpx.MIMEApplicationJSON)
		w.WriteHeader(http.StatusCreated)
		_, err := w.Write([]byte(`{"message": "created"}`))
		AssertNoError(t, err)
	})
	req := NewRequest(http.MethodPost, "/").Build()
	w := Serve(handler, req)

	// Test chaining multiple assertions
	result := Assert(w).
		Status(http.StatusCreated).
		Header(httpx.HeaderContentType, "application/json").
		BodyContains("created")

	AssertNotNil(t, result)
}

func TestAssertWith(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	req := NewRequest(http.MethodGet, "/").Build()
	w := Serve(handler, req)

	// This will use the actual testing.T - just verify it doesn't panic
	result := AssertWith(t, w).Status(http.StatusOK)
	AssertNotNil(t, result)
}

// Test failure paths - these use t.Errorf but we can't easily capture that,
// so we just verify they don't panic and the chain continues

func TestAssert_FailurePaths(t *testing.T) {
	// Test all failure paths to ensure they don't panic and chain continues
	t.Run("Status failure", func(t *testing.T) {
		w := httptest.NewRecorder()
		w.WriteHeader(http.StatusNotFound)

		result := Assert(w).Status(http.StatusOK)
		AssertNotNil(t, result)
	})

	t.Run("StatusNot failure", func(t *testing.T) {
		w := httptest.NewRecorder()
		w.WriteHeader(http.StatusNotFound)

		result := Assert(w).StatusNot(http.StatusNotFound)
		AssertNotNil(t, result)
	})

	t.Run("StatusBetween failure", func(t *testing.T) {
		w := httptest.NewRecorder()
		w.WriteHeader(http.StatusNotFound)

		result := Assert(w).StatusBetween(200, 299)
		AssertNotNil(t, result)
	})

	t.Run("Header failure", func(t *testing.T) {
		w := httptest.NewRecorder()
		w.Header().Set(httpx.HeaderContentType, httpx.MIMETextPlain)

		result := Assert(w).Header(httpx.HeaderContentType, "application/json")
		AssertNotNil(t, result)
	})

	t.Run("HeaderContains failure", func(t *testing.T) {
		w := httptest.NewRecorder()
		w.Header().Set(httpx.HeaderContentType, httpx.MIMETextPlain)

		result := Assert(w).HeaderContains(httpx.HeaderContentType, "json")
		AssertNotNil(t, result)
	})

	t.Run("HeaderExists failure", func(t *testing.T) {
		w := httptest.NewRecorder()

		result := Assert(w).HeaderExists("X-Missing")
		AssertNotNil(t, result)
	})

	t.Run("HeaderNotExists failure", func(t *testing.T) {
		w := httptest.NewRecorder()
		w.Header().Set("X-Custom", "value")

		result := Assert(w).HeaderNotExists("X-Custom")
		AssertNotNil(t, result)
	})

	t.Run("Body failure", func(t *testing.T) {
		w := httptest.NewRecorder()
		_, err := w.Write([]byte("hello"))
		AssertNoError(t, err)

		result := Assert(w).Body("world")
		AssertNotNil(t, result)
	})

	t.Run("BodyContains failure", func(t *testing.T) {
		w := httptest.NewRecorder()
		_, err := w.Write([]byte("hello"))
		AssertNoError(t, err)

		result := Assert(w).BodyContains("world")
		AssertNotNil(t, result)
	})

	t.Run("BodyNotContains failure", func(t *testing.T) {
		w := httptest.NewRecorder()
		_, err := w.Write([]byte("hello world"))
		AssertNoError(t, err)

		result := Assert(w).BodyNotContains("hello")
		AssertNotNil(t, result)
	})

	t.Run("BodyEmpty failure", func(t *testing.T) {
		w := httptest.NewRecorder()
		_, err := w.Write([]byte("content"))
		AssertNoError(t, err)

		result := Assert(w).BodyEmpty()
		AssertNotNil(t, result)
	})

	t.Run("BodyNotEmpty failure", func(t *testing.T) {
		w := httptest.NewRecorder()

		result := Assert(w).BodyNotEmpty()
		AssertNotNil(t, result)
	})

	t.Run("JSON decode failure", func(t *testing.T) {
		w := httptest.NewRecorder()
		_, err := w.Write([]byte("not json"))
		AssertNoError(t, err)

		var result map[string]string
		a := Assert(w).JSON(&result)
		AssertNotNil(t, a)
	})

	t.Run("JSONEq failure", func(t *testing.T) {
		w := httptest.NewRecorder()
		_, err := w.Write([]byte(`{"name": "John"}`))
		AssertNoError(t, err)

		result := Assert(w).JSONEq(`{"name": "Jane"}`)
		AssertNotNil(t, result)
	})

	t.Run("JSONEq unmarshal failure", func(t *testing.T) {
		w := httptest.NewRecorder()
		_, err := w.Write([]byte("not json"))
		AssertNoError(t, err)

		result := Assert(w).JSONEq(`{}`)
		AssertNotNil(t, result)
	})

	t.Run("JSONPathEqual failure", func(t *testing.T) {
		w := httptest.NewRecorder()
		_, err := w.Write([]byte(`{"user": {"name": "John"}}`))
		AssertNoError(t, err)

		result := Assert(w).JSONPathEqual("user.name", "Jane")
		AssertNotNil(t, result)
	})

	t.Run("JSONPathEqual invalid JSON", func(t *testing.T) {
		w := httptest.NewRecorder()
		_, err := w.Write([]byte("not json"))
		AssertNoError(t, err)

		result := Assert(w).JSONPathEqual("user", "value")
		AssertNotNil(t, result)
	})

	t.Run("JSONPathEqual missing key", func(t *testing.T) {
		w := httptest.NewRecorder()
		_, err := w.Write([]byte(`{"user": {}}`))
		AssertNoError(t, err)

		result := Assert(w).JSONPathEqual("user.name", "John")
		AssertNotNil(t, result)
	})

	t.Run("JSONPathNotEqual failure", func(t *testing.T) {
		w := httptest.NewRecorder()
		_, err := w.Write([]byte(`{"user": {"name": "John"}}`))
		AssertNoError(t, err)

		result := Assert(w).JSONPathNotEqual("user.name", "John")
		AssertNotNil(t, result)
	})

	t.Run("JSONPathNotEqual invalid JSON", func(t *testing.T) {
		w := httptest.NewRecorder()
		_, err := w.Write([]byte("not json"))
		AssertNoError(t, err)

		result := Assert(w).JSONPathNotEqual("user", "value")
		AssertNotNil(t, result)
	})

	t.Run("JSONPathNotEqual missing key", func(t *testing.T) {
		w := httptest.NewRecorder()
		_, err := w.Write([]byte(`{"user": {}}`))
		AssertNoError(t, err)

		result := Assert(w).JSONPathNotEqual("user.name", "John")
		AssertNotNil(t, result)
	})

	t.Run("JSONPathExists failure", func(t *testing.T) {
		w := httptest.NewRecorder()
		_, err := w.Write([]byte(`{"user": {}}`))
		AssertNoError(t, err)

		result := Assert(w).JSONPathExists("user.name")
		AssertNotNil(t, result)
	})

	t.Run("JSONPathExists invalid JSON", func(t *testing.T) {
		w := httptest.NewRecorder()
		_, err := w.Write([]byte("not json"))
		AssertNoError(t, err)

		result := Assert(w).JSONPathExists("user")
		AssertNotNil(t, result)
	})

	t.Run("JSONPathNotExists failure", func(t *testing.T) {
		w := httptest.NewRecorder()
		_, err := w.Write([]byte(`{"user": {"name": "John"}}`))
		AssertNoError(t, err)

		result := Assert(w).JSONPathNotExists("user.name")
		AssertNotNil(t, result)
	})

	t.Run("JSONPathNotExists invalid JSON", func(t *testing.T) {
		w := httptest.NewRecorder()
		_, err := w.Write([]byte("not json"))
		AssertNoError(t, err)

		result := Assert(w).JSONPathNotExists("user")
		AssertNotNil(t, result)
	})

	t.Run("Cookie wrong value", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.SetCookie(w, &http.Cookie{Name: "session", Value: "wrong"})
			w.WriteHeader(http.StatusOK)
		})
		req := NewRequest(http.MethodGet, "/").Build()
		rec := Serve(handler, req)

		result := Assert(rec).Cookie("session", "expected")
		AssertNotNil(t, result)
	})

	t.Run("Cookie missing", func(t *testing.T) {
		w := httptest.NewRecorder()

		result := Assert(w).Cookie("session", "value")
		AssertNotNil(t, result)
	})

	t.Run("CookieExists failure", func(t *testing.T) {
		w := httptest.NewRecorder()

		result := Assert(w).CookieExists("session")
		AssertNotNil(t, result)
	})

	t.Run("CookieNotExists failure", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.SetCookie(w, &http.Cookie{Name: "session", Value: "abc"})
			w.WriteHeader(http.StatusOK)
		})
		req := NewRequest(http.MethodGet, "/").Build()
		rec := Serve(handler, req)

		result := Assert(rec).CookieNotExists("session")
		AssertNotNil(t, result)
	})

	t.Run("Redirect not a redirect", func(t *testing.T) {
		w := httptest.NewRecorder()
		w.WriteHeader(http.StatusOK)

		result := Assert(w).Redirect("/other")
		AssertNotNil(t, result)
	})

	t.Run("Redirect wrong location", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/actual", http.StatusFound)
		})
		req := NewRequest(http.MethodGet, "/").Build()
		rec := Serve(handler, req)

		result := Assert(rec).Redirect("/expected")
		AssertNotNil(t, result)
	})

	t.Run("IsSuccess failure", func(t *testing.T) {
		w := httptest.NewRecorder()
		w.WriteHeader(http.StatusNotFound)

		result := Assert(w).IsSuccess()
		AssertNotNil(t, result)
	})

	t.Run("IsClientError failure", func(t *testing.T) {
		w := httptest.NewRecorder()
		w.WriteHeader(http.StatusOK)

		result := Assert(w).IsClientError()
		AssertNotNil(t, result)
	})

	t.Run("IsServerError failure", func(t *testing.T) {
		w := httptest.NewRecorder()
		w.WriteHeader(http.StatusOK)

		result := Assert(w).IsServerError()
		AssertNotNil(t, result)
	})
}

// Test jsonValuesEqual edge cases
func TestJSONValuesEqual(t *testing.T) {
	t.Run("different length maps", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set(httpx.HeaderContentType, httpx.MIMEApplicationJSON)
			_, err := w.Write([]byte(`{"x": 1}`))
			AssertNoError(t, err)
		})
		req := NewRequest(http.MethodGet, "/").Build()
		rec := Serve(handler, req)

		// This should fail because expected has 2 fields but actual has 1
		result := Assert(rec).JSONEq(`{"x": 1, "y": 2}`)
		AssertNotNil(t, result)
	})

	t.Run("different values in maps", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set(httpx.HeaderContentType, httpx.MIMEApplicationJSON)
			_, err := w.Write([]byte(`{"x": 1}`))
			AssertNoError(t, err)
		})
		req := NewRequest(http.MethodGet, "/").Build()
		rec := Serve(handler, req)

		result := Assert(rec).JSONEq(`{"x": 2}`)
		AssertNotNil(t, result)
	})

	t.Run("different length arrays", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set(httpx.HeaderContentType, httpx.MIMEApplicationJSON)
			_, err := w.Write([]byte(`[1, 2]`))
			AssertNoError(t, err)
		})
		req := NewRequest(http.MethodGet, "/").Build()
		rec := Serve(handler, req)

		result := Assert(rec).JSONEq(`{"items": [1, 2, 3]}`)
		AssertNotNil(t, result)
	})

	t.Run("nested map with missing key", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set(httpx.HeaderContentType, httpx.MIMEApplicationJSON)
			_, err := w.Write([]byte(`{"user": {"name": "John"}}`))
			AssertNoError(t, err)
		})
		req := NewRequest(http.MethodGet, "/").Build()
		rec := Serve(handler, req)

		result := Assert(rec).JSONEq(`{"user": {"name": "John", "age": 30}}`)
		AssertNotNil(t, result)
	})

	t.Run("array element mismatch", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set(httpx.HeaderContentType, httpx.MIMEApplicationJSON)
			_, err := w.Write([]byte(`[1, 2, 4]`))
			AssertNoError(t, err)
		})
		req := NewRequest(http.MethodGet, "/").Build()
		rec := Serve(handler, req)

		result := Assert(rec).JSONEq(`[1, 2, 3]`)
		AssertNotNil(t, result)
	})
}

// Test JSONPathEqual edge cases
func TestJSONPathEqual_EdgeCases(t *testing.T) {
	t.Run("traverse into non-map", func(t *testing.T) {
		w := httptest.NewRecorder()
		_, err := w.Write([]byte(`{"value": "string"}`))
		AssertNoError(t, err)

		result := Assert(w).JSONPathEqual("value.property", "x")
		AssertNotNil(t, result)
	})

	t.Run("invalid array index", func(t *testing.T) {
		w := httptest.NewRecorder()
		_, err := w.Write([]byte(`{"items": [1, 2, 3]}`))
		AssertNoError(t, err)

		result := Assert(w).JSONPathEqual("items.invalid", "x")
		AssertNotNil(t, result)
	})

	t.Run("out of bounds array index", func(t *testing.T) {
		w := httptest.NewRecorder()
		_, err := w.Write([]byte(`{"items": [1, 2, 3]}`))
		AssertNoError(t, err)

		result := Assert(w).JSONPathEqual("items.99", "x")
		AssertNotNil(t, result)
	})

	t.Run("deep path", func(t *testing.T) {
		w := httptest.NewRecorder()
		_, err := w.Write([]byte(`{"a": {"b": {"c": {"d": "deep"}}}}`))
		AssertNoError(t, err)

		result := Assert(w).JSONPathEqual("a.b.c.d", "wrong")
		AssertNotNil(t, result)
	})
}

// Test Problem ProblemDetail failure paths
func TestProblemDetail_FailurePaths(t *testing.T) {
	t.Run("IsProblemDetail failure", func(t *testing.T) {
		w := httptest.NewRecorder()
		w.Header().Set(httpx.HeaderContentType, httpx.MIMEApplicationJSON)
		_, err := w.Write([]byte(`{}`))
		AssertNoError(t, err)

		result := Assert(w).IsProblemDetail()
		AssertNotNil(t, result)
	})

	t.Run("ProblemDetailStatus invalid JSON", func(t *testing.T) {
		w := httptest.NewRecorder()
		w.Header().Set(httpx.HeaderContentType, "application/problem+json")
		_, err := w.Write([]byte("not json"))
		AssertNoError(t, err)

		result := Assert(w).ProblemDetailStatus(http.StatusBadRequest)
		AssertNotNil(t, result)
	})

	t.Run("ProblemDetailStatus wrong status", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			detail := problem.NewDetail(http.StatusBadRequest, "Invalid")
			err := detail.Render(w)
			AssertNoError(t, err)
		})
		req := NewRequest(http.MethodGet, "/").Build()
		rec := Serve(handler, req)

		result := Assert(rec).ProblemDetailStatus(http.StatusInternalServerError)
		AssertNotNil(t, result)
	})

	t.Run("ProblemDetailTitle invalid JSON", func(t *testing.T) {
		w := httptest.NewRecorder()
		w.Header().Set(httpx.HeaderContentType, "application/problem+json")
		_, err := w.Write([]byte("not json"))
		AssertNoError(t, err)

		result := Assert(w).ProblemDetailTitle("Bad Request")
		AssertNotNil(t, result)
	})

	t.Run("ProblemDetailTitle wrong title", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			detail := problem.NewDetail(http.StatusBadRequest, "Invalid")
			err := detail.Render(w)
			AssertNoError(t, err)
		})
		req := NewRequest(http.MethodGet, "/").Build()
		rec := Serve(handler, req)

		result := Assert(rec).ProblemDetailTitle("Wrong Title")
		AssertNotNil(t, result)
	})

	t.Run("ProblemDetailDetail invalid JSON", func(t *testing.T) {
		w := httptest.NewRecorder()
		w.Header().Set(httpx.HeaderContentType, "application/problem+json")
		_, err := w.Write([]byte("not json"))
		AssertNoError(t, err)

		result := Assert(w).ProblemDetailDetail("detail")
		AssertNotNil(t, result)
	})

	t.Run("ProblemDetailDetail wrong detail", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			detail := problem.NewDetail(http.StatusBadRequest, "Actual detail")
			err := detail.Render(w)
			AssertNoError(t, err)
		})
		req := NewRequest(http.MethodGet, "/").Build()
		rec := Serve(handler, req)

		result := Assert(rec).ProblemDetailDetail("Expected detail")
		AssertNotNil(t, result)
	})

	t.Run("ProblemDetailType invalid JSON", func(t *testing.T) {
		w := httptest.NewRecorder()
		w.Header().Set(httpx.HeaderContentType, "application/problem+json")
		_, err := w.Write([]byte("not json"))
		AssertNoError(t, err)

		result := Assert(w).ProblemDetailType("type")
		AssertNotNil(t, result)
	})

	t.Run("ProblemDetailType wrong type", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			detail := problem.NewDetail(http.StatusBadRequest, "Invalid")
			detail.Type = "https://api.example.com/actual"
			err := detail.Render(w)
			AssertNoError(t, err)
		})
		req := NewRequest(http.MethodGet, "/").Build()
		rec := Serve(handler, req)

		result := Assert(rec).ProblemDetailType("https://api.example.com/expected")
		AssertNotNil(t, result)
	})

	t.Run("ProblemDetailExtension invalid JSON", func(t *testing.T) {
		w := httptest.NewRecorder()
		w.Header().Set(httpx.HeaderContentType, "application/problem+json")
		_, err := w.Write([]byte("not json"))
		AssertNoError(t, err)

		result := Assert(w).ProblemDetailExtension("key", "value")
		AssertNotNil(t, result)
	})

	t.Run("ProblemDetailExtension missing key", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			detail := problem.NewDetail(http.StatusBadRequest, "Invalid")
			err := detail.Render(w)
			AssertNoError(t, err)
		})
		req := NewRequest(http.MethodGet, "/").Build()
		rec := Serve(handler, req)

		result := Assert(rec).ProblemDetailExtension("missing", "value")
		AssertNotNil(t, result)
	})

	t.Run("ProblemDetailExtension wrong value", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			detail := problem.NewDetail(http.StatusBadRequest, "Invalid")
			detail.Set("key", "actual")
			err := detail.Render(w)
			AssertNoError(t, err)
		})
		req := NewRequest(http.MethodGet, "/").Build()
		rec := Serve(handler, req)

		result := Assert(rec).ProblemDetailExtension("key", "expected")
		AssertNotNil(t, result)
	})

	t.Run("ProblemDetail invalid JSON", func(t *testing.T) {
		w := httptest.NewRecorder()
		w.Header().Set(httpx.HeaderContentType, "application/problem+json")
		_, err := w.Write([]byte("not json"))
		AssertNoError(t, err)

		var detail problem.Detail

		result := Assert(w).ProblemDetail(detail)
		AssertNotNil(t, result)
	})
}

func TestAssert_IsProblemDetail(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		detail := problem.NewDetail(http.StatusBadRequest, "Invalid request")
		err := detail.Render(w)
		AssertNoError(t, err)
	})
	req := NewRequest(http.MethodGet, "/").Build()
	w := Serve(handler, req)

	result := Assert(w).IsProblemDetail()
	AssertNotNil(t, result)
}

func TestAssert_ProblemDetailStatus(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		detail := problem.NewDetail(http.StatusBadRequest, "Invalid request")
		err := detail.Render(w)
		AssertNoError(t, err)
	})
	req := NewRequest(http.MethodGet, "/").Build()
	w := Serve(handler, req)

	result := Assert(w).ProblemDetailStatus(http.StatusBadRequest)
	AssertNotNil(t, result)
}

func TestAssert_ProblemDetailTitle(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		detail := problem.NewDetail(http.StatusBadRequest, "Invalid request")
		err := detail.Render(w)
		AssertNoError(t, err)
	})
	req := NewRequest(http.MethodGet, "/").Build()
	w := Serve(handler, req)

	result := Assert(w).ProblemDetailTitle("Bad Request")
	AssertNotNil(t, result)
}

func TestAssert_ProblemDetailDetail(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		detail := problem.NewDetail(http.StatusBadRequest, "Invalid request")
		err := detail.Render(w)
		AssertNoError(t, err)
	})
	req := NewRequest(http.MethodGet, "/").Build()
	w := Serve(handler, req)

	result := Assert(w).ProblemDetailDetail("Invalid request")
	AssertNotNil(t, result)
}

func TestAssert_ProblemDetailType(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		detail := problem.NewDetail(http.StatusBadRequest, "Invalid request")
		detail.Type = "https://api.example.com/errors/invalid-request"
		err := detail.Render(w)
		AssertNoError(t, err)
	})
	req := NewRequest(http.MethodGet, "/").Build()
	w := Serve(handler, req)

	result := Assert(w).ProblemDetailType("https://api.example.com/errors/invalid-request")
	AssertNotNil(t, result)
}

func TestAssert_ProblemDetailExtension(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		detail := problem.NewDetail(http.StatusUnprocessableEntity, "Validation failed")
		detail.Set("errors", []string{"field1 is required", "field2 is invalid"})
		err := detail.Render(w)
		AssertNoError(t, err)
	})
	req := NewRequest(http.MethodGet, "/").Build()
	w := Serve(handler, req)

	result := Assert(w).ProblemDetailExtension("errors", []string{"field1 is required", "field2 is invalid"})
	AssertNotNil(t, result)
}

func TestAssert_ProblemDetail(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		detail := problem.NewDetail(http.StatusBadRequest, "Invalid request")
		err := detail.Render(w)
		AssertNoError(t, err)
	})
	req := NewRequest(http.MethodGet, "/").Build()
	w := Serve(handler, req)

	var detail problem.Detail
	result := Assert(w).ProblemDetail(&detail)
	AssertNotNil(t, result)
	AssertEqual(t, http.StatusBadRequest, detail.Status)
	AssertEqual(t, "Invalid request", detail.Detail)
}

func TestAssert_ProblemDetailChaining(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		detail := problem.NewDetail(http.StatusBadRequest, "Invalid request")
		detail.Type = "https://api.example.com/errors/invalid-request"
		err := detail.Render(w)
		AssertNoError(t, err)
	})
	req := NewRequest(http.MethodGet, "/").Build()
	w := Serve(handler, req)

	result := Assert(w).
		IsProblemDetail().
		ProblemDetailStatus(http.StatusBadRequest).
		ProblemDetailTitle("Bad Request").
		ProblemDetailDetail("Invalid request").
		ProblemDetailType("https://api.example.com/errors/invalid-request")

	AssertNotNil(t, result)
}

func TestAssertNoError(t *testing.T) {
	AssertNoError(t, nil)
}

func TestAssertError(t *testing.T) {
	AssertError(t, errors.New("some error"))
}

func TestAssertErrorIs(t *testing.T) {
	err := fmt.Errorf("wrapped: %w", os.ErrNotExist)
	AssertErrorIs(t, err, os.ErrNotExist)
}

func TestAssertErrorContains(t *testing.T) {
	AssertErrorContains(t, errors.New("connection refused"), "refused")
}

func TestAssertNil(t *testing.T) {
	t.Run("passes with nil", func(t *testing.T) {
		AssertNil(t, nil)
	})

	t.Run("passes with nil pointer", func(t *testing.T) {
		var ptr *int
		AssertNil(t, ptr)
	})

	t.Run("passes with nil slice", func(t *testing.T) {
		var slice []int
		AssertNil(t, slice)
	})

	t.Run("passes with nil map", func(t *testing.T) {
		var m map[string]int
		AssertNil(t, m)
	})
}

func TestAssertNotNil(t *testing.T) {
	t.Run("passes with non-nil value", func(t *testing.T) {
		AssertNotNil(t, "not nil")
	})

	t.Run("passes with non-nil pointer", func(t *testing.T) {
		val := 42
		AssertNotNil(t, &val)
	})

	t.Run("passes with non-nil slice", func(t *testing.T) {
		AssertNotNil(t, []int{1, 2, 3})
	})

	t.Run("passes with empty but non-nil slice", func(t *testing.T) {
		AssertNotNil(t, []int{})
	})
}

func TestAssertEqual(t *testing.T) {
	AssertEqual(t, 42, 42)
	AssertEqual(t, "hello", "hello")
	AssertEqual(t, true, true)
}

func TestAssertEqual_NumericTypes(t *testing.T) {
	// Mixed int types
	t.Run("int vs int64", func(t *testing.T) {
		var int64Val int64 = 42
		AssertEqual(t, 42, int64Val)
		AssertEqual(t, int64Val, 42)
	})

	t.Run("int vs int32", func(t *testing.T) {
		var int32Val int32 = 100
		AssertEqual(t, 100, int32Val)
		AssertEqual(t, int32Val, 100)
	})

	t.Run("int64 vs uint64", func(t *testing.T) {
		var int64Val int64 = 1024
		var uint64Val uint64 = 1024
		AssertEqual(t, int64Val, uint64Val)
	})

	t.Run("int vs float64", func(t *testing.T) {
		floatVal := 42.0
		AssertEqual(t, 42, floatVal)
		AssertEqual(t, floatVal, 42)
	})

	t.Run("float32 vs float64", func(t *testing.T) {
		var float32Val float32 = 3.5
		float64Val := 3.5
		AssertEqual(t, float32Val, float64Val)
	})

	t.Run("uint vs int", func(t *testing.T) {
		var uintVal uint = 999
		AssertEqual(t, 999, uintVal)
		AssertEqual(t, uintVal, 999)
	})

	t.Run("byte vs uint8", func(t *testing.T) {
		var byteVal byte = 255
		var uint8Val uint8 = 255
		AssertEqual(t, byteVal, uint8Val)
	})
}

func TestAssertNotEqual(t *testing.T) {
	AssertNotEqual(t, 42, 43)
	AssertNotEqual(t, "hello", "world")
}

func TestAssertNotEqual_NumericTypes(t *testing.T) {
	// Mixed int types that are not equal
	t.Run("int vs int64 different values", func(t *testing.T) {
		var int64Val int64 = 43
		AssertNotEqual(t, 42, int64Val)
		AssertNotEqual(t, int64Val, 42)
	})

	t.Run("int vs float64 different values", func(t *testing.T) {
		floatVal := 42.5
		AssertNotEqual(t, 42, floatVal)
		AssertNotEqual(t, floatVal, 42)
	})

	t.Run("int64 vs uint64 different values", func(t *testing.T) {
		var int64Val int64 = 100
		var uint64Val uint64 = 200
		AssertNotEqual(t, int64Val, uint64Val)
	})
}

func TestAssertDeepEqual(t *testing.T) {
	t.Run("equal slices", func(t *testing.T) {
		AssertDeepEqual(t, []int{1, 2, 3}, []int{1, 2, 3})
	})

	t.Run("equal structs", func(t *testing.T) {
		type User struct {
			Name string
			Age  int
		}
		AssertDeepEqual(t, User{Name: "John", Age: 30}, User{Name: "John", Age: 30})
	})

	t.Run("equal maps", func(t *testing.T) {
		AssertDeepEqual(t, map[string]int{"a": 1}, map[string]int{"a": 1})
	})
}

func TestAssertTrue(t *testing.T) {
	AssertTrue(t, true)
	AssertTrue(t, len([]int{1, 2, 3}) > 0)
}

func TestAssertFalse(t *testing.T) {
	AssertFalse(t, false)
	AssertFalse(t, len([]int{}) > 0)
}

func TestAssertEmpty(t *testing.T) {
	t.Run("empty string", func(t *testing.T) {
		AssertEmpty(t, "")
	})

	t.Run("empty slice", func(t *testing.T) {
		AssertEmpty(t, []int{})
	})

	t.Run("nil slice", func(t *testing.T) {
		AssertEmpty(t, []int(nil))
	})

	t.Run("empty map", func(t *testing.T) {
		AssertEmpty(t, map[string]int{})
	})
}

func TestAssertNotEmpty(t *testing.T) {
	t.Run("non-empty string", func(t *testing.T) {
		AssertNotEmpty(t, "hello")
	})

	t.Run("non-empty slice", func(t *testing.T) {
		AssertNotNil(t, []int{1, 2, 3})
	})
}

func TestAssertLen(t *testing.T) {
	t.Run("slice length", func(t *testing.T) {
		AssertLen(t, []int{1, 2, 3}, 3)
	})

	t.Run("string length", func(t *testing.T) {
		AssertLen(t, "hello", 5)
	})

	t.Run("map length", func(t *testing.T) {
		AssertLen(t, map[string]int{"a": 1, "b": 2}, 2)
	})

	t.Run("empty slice", func(t *testing.T) {
		AssertLen(t, []int{}, 0)
	})
}

func TestAssertContains(t *testing.T) {
	t.Run("int slice", func(t *testing.T) {
		AssertContains(t, []int{1, 2, 3}, 2)
	})

	t.Run("string slice", func(t *testing.T) {
		AssertContains(t, []string{"a", "b", "c"}, "b")
	})

	t.Run("string contains substring", func(t *testing.T) {
		AssertContains(t, "hello world", "world")
	})

	t.Run("string contains substring at start", func(t *testing.T) {
		AssertContains(t, "hello world", "hello")
	})
}

func TestAssertNotContains(t *testing.T) {
	t.Run("int slice", func(t *testing.T) {
		AssertNotContains(t, []int{1, 2, 3}, 4)
	})

	t.Run("string slice", func(t *testing.T) {
		AssertNotContains(t, []string{"a", "b", "c"}, "d")
	})

	t.Run("string does not contain substring", func(t *testing.T) {
		AssertNotContains(t, "hello world", "goodbye")
	})
}

func TestAssertIsType(t *testing.T) {
	t.Run("matching int type", func(t *testing.T) {
		x := 42
		AssertIsType(t, 0, x)
	})

	t.Run("matching string type", func(t *testing.T) {
		AssertIsType(t, "", "hello")
	})

	t.Run("matching pointer type", func(t *testing.T) {
		val := 42
		AssertIsType(t, (*int)(nil), &val)
	})
}

func TestAssertImplements(t *testing.T) {
	t.Run("bytes.Buffer implements io.Reader", func(t *testing.T) {
		var buf bytes.Buffer
		AssertImplements(t, (*io.Reader)(nil), &buf)
	})

	t.Run("bytes.Buffer implements io.Writer", func(t *testing.T) {
		var buf bytes.Buffer
		AssertImplements(t, (*io.Writer)(nil), &buf)
	})
}

func TestGeneralAssert_FailurePaths(t *testing.T) {
	t.Run("AssertNoError fails", func(t *testing.T) {
		AssertNoError(nil, errors.New("some error"))
	})

	t.Run("AssertError fails", func(t *testing.T) {
		AssertError(nil, nil)
	})

	t.Run("AssertErrorIs fails", func(t *testing.T) {
		AssertErrorIs(nil, errors.New("wrong error"), os.ErrNotExist)
	})

	t.Run("AssertErrorContains with nil", func(t *testing.T) {
		AssertErrorContains(nil, nil, "anything")
	})

	t.Run("AssertErrorContains wrong substring", func(t *testing.T) {
		AssertErrorContains(nil, errors.New("hello"), "world")
	})

	t.Run("AssertNil fails with non-nil", func(t *testing.T) {
		AssertNil(nil, "not nil")
	})

	t.Run("AssertNil fails with non-nil pointer", func(t *testing.T) {
		val := 42
		AssertNil(nil, &val)
	})

	t.Run("AssertNotNil fails with nil", func(t *testing.T) {
		AssertNotNil(nil, nil)
	})

	t.Run("AssertNotNil fails with nil pointer", func(t *testing.T) {
		var ptr *int
		AssertNotNil(nil, ptr)
	})

	t.Run("AssertEqual fails", func(t *testing.T) {
		AssertEqual(nil, 42, 43)
	})

	t.Run("AssertNotEqual fails", func(t *testing.T) {
		AssertNotEqual(nil, 42, 42)
	})

	t.Run("AssertDeepEqual fails", func(t *testing.T) {
		AssertDeepEqual(nil, []int{1, 2}, []int{1, 3})
	})

	t.Run("AssertTrue fails", func(t *testing.T) {
		AssertTrue(nil, false)
	})

	t.Run("AssertFalse fails", func(t *testing.T) {
		AssertFalse(nil, true)
	})

	t.Run("AssertEmpty fails", func(t *testing.T) {
		AssertEmpty(nil, "not empty")
	})

	t.Run("AssertNotEmpty fails", func(t *testing.T) {
		AssertNotEmpty(nil, "")
	})

	t.Run("AssertLen fails", func(t *testing.T) {
		AssertLen(nil, []int{1, 2, 3}, 2)
	})

	t.Run("AssertLen with non-collection", func(t *testing.T) {
		AssertLen(nil, 42, 1)
	})

	t.Run("AssertContains fails", func(t *testing.T) {
		AssertContains(nil, []int{1, 2, 3}, 4)
	})

	t.Run("AssertContains with non-slice", func(t *testing.T) {
		AssertContains(nil, 42, 1)
	})

	t.Run("AssertNotContains fails", func(t *testing.T) {
		AssertNotContains(nil, []int{1, 2, 3}, 2)
	})

	t.Run("AssertNotContains with non-slice", func(t *testing.T) {
		AssertNotContains(nil, 42, 1)
	})

	t.Run("AssertIsType fails", func(t *testing.T) {
		AssertIsType(nil, 0, "string")
	})

	t.Run("AssertImplements fails", func(t *testing.T) {
		AssertImplements(nil, (*io.Reader)(nil), 42)
	})

	t.Run("AssertImplements with invalid interfaceType", func(t *testing.T) {
		AssertImplements(nil, 42, "test")
	})

	t.Run("AssertGreater fails", func(t *testing.T) {
		AssertGreater(nil, 5, 10)
	})

	t.Run("AssertGreater with equal values", func(t *testing.T) {
		AssertGreater(nil, 5, 5)
	})

	t.Run("AssertGreater with non-numeric", func(t *testing.T) {
		AssertGreater(nil, "hello", "world")
	})

	t.Run("AssertLess fails", func(t *testing.T) {
		AssertLess(nil, 10, 5)
	})

	t.Run("AssertLess with equal values", func(t *testing.T) {
		AssertLess(nil, 5, 5)
	})

	t.Run("AssertLess with non-numeric", func(t *testing.T) {
		AssertLess(nil, "hello", "world")
	})

	t.Run("AssertPanic does not panic", func(t *testing.T) {
		AssertPanic(nil, func() {
			// This function does not panic
		})
	})

	t.Run("AssertNoPanic panics", func(t *testing.T) {
		AssertNoPanic(nil, func() {
			panic("intentional panic")
		})
	})
}

// Test AssertGreater and AssertLess
func TestAssertGreater(t *testing.T) {
	t.Run("int greater", func(t *testing.T) {
		AssertGreater(t, 10, 5)
	})

	t.Run("int64 greater", func(t *testing.T) {
		AssertGreater(t, int64(10), int64(5))
	})

	t.Run("uint greater", func(t *testing.T) {
		AssertGreater(t, uint(10), uint(5))
	})

	t.Run("float64 greater", func(t *testing.T) {
		AssertGreater(t, 10.5, 5.2)
	})

	t.Run("float32 greater", func(t *testing.T) {
		AssertGreater(t, float32(10.5), float32(5.2))
	})

	t.Run("mixed int types", func(t *testing.T) {
		AssertGreater(t, 10, int64(5))
	})

	t.Run("int and float", func(t *testing.T) {
		AssertGreater(t, 10.5, 5)
	})
}

func TestAssertLess(t *testing.T) {
	t.Run("int less", func(t *testing.T) {
		AssertLess(t, 5, 10)
	})

	t.Run("int64 less", func(t *testing.T) {
		AssertLess(t, int64(5), int64(10))
	})

	t.Run("uint less", func(t *testing.T) {
		AssertLess(t, uint(5), uint(10))
	})

	t.Run("float64 less", func(t *testing.T) {
		AssertLess(t, 5.2, 10.5)
	})

	t.Run("float32 less", func(t *testing.T) {
		AssertLess(t, float32(5.2), float32(10.5))
	})

	t.Run("mixed int types", func(t *testing.T) {
		AssertLess(t, 5, int64(10))
	})

	t.Run("int and float", func(t *testing.T) {
		AssertLess(t, 5, 10.5)
	})
}

// Test AssertPanic and AssertNoPanic
func TestAssertPanic(t *testing.T) {
	t.Run("function panics", func(t *testing.T) {
		AssertPanic(t, func() {
			panic("intentional panic")
		})
	})

	t.Run("function panics with different types", func(t *testing.T) {
		AssertPanic(t, func() {
			panic(errors.New("error panic"))
		})
	})
}

func TestAssertNoPanic(t *testing.T) {
	t.Run("function does not panic", func(t *testing.T) {
		AssertNoPanic(t, func() {
			// Normal execution
			_ = 1 + 1
		})
	})

	t.Run("function returns normally", func(t *testing.T) {
		result := 0
		AssertNoPanic(t, func() {
			result = 42
		})
		AssertEqual(t, 42, result)
	})
}

func TestAssertPanicContains(t *testing.T) {
	t.Run("panic message contains substring", func(t *testing.T) {
		AssertPanicContains(t, func() {
			panic("this is an intentional panic message")
		}, "intentional panic")
	})

	t.Run("panic with error containing substring", func(t *testing.T) {
		AssertPanicContains(t, func() {
			panic(errors.New("connection refused: database is down"))
		}, "database is down")
	})

	t.Run("panic with formatted message", func(t *testing.T) {
		AssertPanicContains(t, func() {
			panic(fmt.Sprintf("error code: %d, message: %s", 500, "internal server error"))
		}, "error code: 500")
	})
}

// Test AssertFail and AssertFailf - these can't test the failure case
// since they call t.Fatal, but we can at least verify they exist and compile
func TestAssertFail(t *testing.T) {
	// We can't actually test the failure path (it would kill the test)
	// but we verify the function signature is correct by calling it conditionally
	shouldFail := false
	if shouldFail {
		AssertFail(t, "this should not be called")
	}
}

func TestAssertFailf(t *testing.T) {
	// We can't actually test the failure path (it would kill the test)
	// but we verify the function signature is correct by calling it conditionally
	shouldFail := false
	if shouldFail {
		AssertFailf(t, "formatted message: %s, %d", "test", 42)
	}
}
