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
		if result == nil {
			t.Error("expected result to not be nil")
		}
	})
}

func TestAssert_StatusNot(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	req := NewRequest(http.MethodGet, "/").Build()
	w := Serve(handler, req)

	result := Assert(w).StatusNot(http.StatusNotFound)
	if result == nil {
		t.Error("expected result to not be nil")
	}
}

func TestAssert_StatusBetween(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})
	req := NewRequest(http.MethodGet, "/").Build()
	w := Serve(handler, req)

	result := Assert(w).StatusBetween(200, 299)
	if result == nil {
		t.Error("expected result to not be nil")
	}
}

func TestAssert_Header(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(httpx.HeaderContentType, httpx.MIMEApplicationJSON)
		w.WriteHeader(http.StatusOK)
	})
	req := NewRequest(http.MethodGet, "/").Build()
	w := Serve(handler, req)

	result := Assert(w).Header(httpx.HeaderContentType, "application/json")
	if result == nil {
		t.Error("expected result to not be nil")
	}
}

func TestAssert_HeaderContains(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(httpx.HeaderContentType, "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
	})
	req := NewRequest(http.MethodGet, "/").Build()
	w := Serve(handler, req)

	result := Assert(w).HeaderContains(httpx.HeaderContentType, "json")
	if result == nil {
		t.Error("expected result to not be nil")
	}
}

func TestAssert_HeaderExists(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Custom", "value")
		w.WriteHeader(http.StatusOK)
	})
	req := NewRequest(http.MethodGet, "/").Build()
	w := Serve(handler, req)

	result := Assert(w).HeaderExists("X-Custom")
	if result == nil {
		t.Error("expected result to not be nil")
	}
}

func TestAssert_HeaderNotExists(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	req := NewRequest(http.MethodGet, "/").Build()
	w := Serve(handler, req)

	result := Assert(w).HeaderNotExists("X-Custom")
	if result == nil {
		t.Error("expected result to not be nil")
	}
}

func TestAssert_Body(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write([]byte("hello")); err != nil {
			t.Errorf("failed to write: %v", err)
		}
	})
	req := NewRequest(http.MethodGet, "/").Build()
	w := Serve(handler, req)

	result := Assert(w).Body("hello")
	if result == nil {
		t.Error("expected result to not be nil")
	}
}

func TestAssert_BodyContains(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write([]byte("hello world")); err != nil {
			t.Errorf("failed to write: %v", err)
		}
	})
	req := NewRequest(http.MethodGet, "/").Build()
	w := Serve(handler, req)

	result := Assert(w).BodyContains("world")
	if result == nil {
		t.Error("expected result to not be nil")
	}
}

func TestAssert_BodyNotContains(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write([]byte("hello world")); err != nil {
			t.Errorf("failed to write: %v", err)
		}
	})
	req := NewRequest(http.MethodGet, "/").Build()
	w := Serve(handler, req)

	result := Assert(w).BodyNotContains("error")
	if result == nil {
		t.Error("expected result to not be nil")
	}
}

func TestAssert_BodyEmpty(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	req := NewRequest(http.MethodGet, "/").Build()
	w := Serve(handler, req)

	result := Assert(w).BodyEmpty()
	if result == nil {
		t.Error("expected result to not be nil")
	}
}

func TestAssert_BodyNotEmpty(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write([]byte("content")); err != nil {
			t.Errorf("failed to write: %v", err)
		}
	})
	req := NewRequest(http.MethodGet, "/").Build()
	w := Serve(handler, req)

	result := Assert(w).BodyNotEmpty()
	if result == nil {
		t.Error("expected result to not be nil")
	}
}

func TestAssert_JSON(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(httpx.HeaderContentType, httpx.MIMEApplicationJSON)
		if _, err := w.Write([]byte(`{"name": "John", "age": 30}`)); err != nil {
			t.Errorf("failed to write: %v", err)
		}
	})
	req := NewRequest(http.MethodGet, "/").Build()
	w := Serve(handler, req)

	var result struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	a := Assert(w).JSON(&result)
	if a == nil {
		t.Error("expected result to not be nil")
	}
	if result.Name != "John" {
		t.Errorf("expected name 'John', got %s", result.Name)
	}
	if result.Age != 30 {
		t.Errorf("expected age 30, got %d", result.Age)
	}
}

func TestAssert_JSONEq(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(httpx.HeaderContentType, httpx.MIMEApplicationJSON)
		if _, err := w.Write([]byte(`{"name": "John"}`)); err != nil {
			t.Errorf("failed to write: %v", err)
		}
	})
	req := NewRequest(http.MethodGet, "/").Build()
	w := Serve(handler, req)

	result := Assert(w).JSONEq(`{"name": "John"}`)
	if result == nil {
		t.Error("expected result to not be nil")
	}
}

func TestAssert_JSONPathEqual(t *testing.T) {
	t.Run("works with nested object", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set(httpx.HeaderContentType, httpx.MIMEApplicationJSON)
			if _, err := w.Write([]byte(`{"user": {"name": "John"}}`)); err != nil {
				t.Errorf("failed to write: %v", err)
			}
		})
		req := NewRequest(http.MethodGet, "/").Build()
		w := Serve(handler, req)

		result := Assert(w).JSONPathEqual("user.name", "John")
		if result == nil {
			t.Error("expected result to not be nil")
		}
	})

	t.Run("works with array index", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set(httpx.HeaderContentType, httpx.MIMEApplicationJSON)
			if _, err := w.Write([]byte(`{"items": [{"id": 1}, {"id": 2}]}`)); err != nil {
				t.Errorf("failed to write: %v", err)
			}
		})
		req := NewRequest(http.MethodGet, "/").Build()
		w := Serve(handler, req)

		result := Assert(w).JSONPathEqual("items.0.id", "1")
		if result == nil {
			t.Error("expected result to not be nil")
		}
	})
}

func TestAssert_JSONPathNotEqual(t *testing.T) {
	t.Run("value is not equal", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set(httpx.HeaderContentType, httpx.MIMEApplicationJSON)
			if _, err := w.Write([]byte(`{"user": {"name": "John"}}`)); err != nil {
				t.Errorf("failed to write: %v", err)
			}
		})
		req := NewRequest(http.MethodGet, "/").Build()
		w := Serve(handler, req)

		result := Assert(w).JSONPathNotEqual("user.name", "Jane")
		if result == nil {
			t.Error("expected result to not be nil")
		}
	})

	t.Run("works with array index", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set(httpx.HeaderContentType, httpx.MIMEApplicationJSON)
			if _, err := w.Write([]byte(`{"items": [{"id": 1}, {"id": 2}]}`)); err != nil {
				t.Errorf("failed to write: %v", err)
			}
		})
		req := NewRequest(http.MethodGet, "/").Build()
		w := Serve(handler, req)

		result := Assert(w).JSONPathNotEqual("items.0.id", "999")
		if result == nil {
			t.Error("expected result to not be nil")
		}
	})
}

func TestAssert_JSONPathExists(t *testing.T) {
	t.Run("path exists", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set(httpx.HeaderContentType, httpx.MIMEApplicationJSON)
			if _, err := w.Write([]byte(`{"user": {"name": "John"}}`)); err != nil {
				t.Errorf("failed to write: %v", err)
			}
		})
		req := NewRequest(http.MethodGet, "/").Build()
		w := Serve(handler, req)

		result := Assert(w).JSONPathExists("user.name")
		if result == nil {
			t.Error("expected result to not be nil")
		}
	})

	t.Run("nested path exists", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set(httpx.HeaderContentType, httpx.MIMEApplicationJSON)
			if _, err := w.Write([]byte(`{"user": {"profile": {"name": "John"}}}`)); err != nil {
				t.Errorf("failed to write: %v", err)
			}
		})
		req := NewRequest(http.MethodGet, "/").Build()
		w := Serve(handler, req)

		result := Assert(w).JSONPathExists("user.profile.name")
		if result == nil {
			t.Error("expected result to not be nil")
		}
	})

	t.Run("array index exists", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set(httpx.HeaderContentType, httpx.MIMEApplicationJSON)
			if _, err := w.Write([]byte(`{"items": [{"id": 1}]}`)); err != nil {
				t.Errorf("failed to write: %v", err)
			}
		})
		req := NewRequest(http.MethodGet, "/").Build()
		w := Serve(handler, req)

		result := Assert(w).JSONPathExists("items.0.id")
		if result == nil {
			t.Error("expected result to not be nil")
		}
	})
}

func TestAssert_JSONPathNotExists(t *testing.T) {
	t.Run("path does not exist", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set(httpx.HeaderContentType, httpx.MIMEApplicationJSON)
			if _, err := w.Write([]byte(`{"user": {"name": "John"}}`)); err != nil {
				t.Errorf("failed to write: %v", err)
			}
		})
		req := NewRequest(http.MethodGet, "/").Build()
		w := Serve(handler, req)

		result := Assert(w).JSONPathNotExists("user.password")
		if result == nil {
			t.Error("expected result to not be nil")
		}
	})

	t.Run("nested path does not exist", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set(httpx.HeaderContentType, httpx.MIMEApplicationJSON)
			if _, err := w.Write([]byte(`{"user": {"profile": {"name": "John"}}}`)); err != nil {
				t.Errorf("failed to write: %v", err)
			}
		})
		req := NewRequest(http.MethodGet, "/").Build()
		w := Serve(handler, req)

		result := Assert(w).JSONPathNotExists("user.profile.email")
		if result == nil {
			t.Error("expected result to not be nil")
		}
	})

	t.Run("array index out of bounds", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set(httpx.HeaderContentType, httpx.MIMEApplicationJSON)
			if _, err := w.Write([]byte(`{"items": [{"id": 1}]}`)); err != nil {
				t.Errorf("failed to write: %v", err)
			}
		})
		req := NewRequest(http.MethodGet, "/").Build()
		w := Serve(handler, req)

		result := Assert(w).JSONPathNotExists("items.5.id")
		if result == nil {
			t.Error("expected result to not be nil")
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set(httpx.HeaderContentType, httpx.MIMEApplicationJSON)
			if _, err := w.Write([]byte(`not json`)); err != nil {
				t.Errorf("failed to write: %v", err)
			}
		})
		req := NewRequest(http.MethodGet, "/").Build()
		w := Serve(handler, req)

		// This will fail to decode JSON, but we still return the assertion
		result := Assert(w).JSONPathNotExists("any.path")
		if result == nil {
			t.Error("expected result to not be nil")
		}
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
	if result == nil {
		t.Error("expected result to not be nil")
	}
}

func TestAssert_CookieExists(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "session", Value: "abc123"})
		w.WriteHeader(http.StatusOK)
	})
	req := NewRequest(http.MethodGet, "/").Build()
	w := Serve(handler, req)

	result := Assert(w).CookieExists("session")
	if result == nil {
		t.Error("expected result to not be nil")
	}
}

func TestAssert_CookieNotExists(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	req := NewRequest(http.MethodGet, "/").Build()
	w := Serve(handler, req)

	result := Assert(w).CookieNotExists("session")
	if result == nil {
		t.Error("expected result to not be nil")
	}
}

func TestAssert_Redirect(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/login", http.StatusFound)
	})
	req := NewRequest(http.MethodGet, "/").Build()
	w := Serve(handler, req)

	result := Assert(w).Redirect("/login")
	if result == nil {
		t.Error("expected result to not be nil")
	}
}

func TestAssert_IsSuccess(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})
	req := NewRequest(http.MethodGet, "/").Build()
	w := Serve(handler, req)

	result := Assert(w).IsSuccess()
	if result == nil {
		t.Error("expected result to not be nil")
	}
}

func TestAssert_IsClientError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	req := NewRequest(http.MethodGet, "/").Build()
	w := Serve(handler, req)

	result := Assert(w).IsClientError()
	if result == nil {
		t.Error("expected result to not be nil")
	}
}

func TestAssert_IsServerError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	req := NewRequest(http.MethodGet, "/").Build()
	w := Serve(handler, req)

	result := Assert(w).IsServerError()
	if result == nil {
		t.Error("expected result to not be nil")
	}
}

func TestAssert_Chaining(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(httpx.HeaderContentType, httpx.MIMEApplicationJSON)
		w.WriteHeader(http.StatusCreated)
		if _, err := w.Write([]byte(`{"message": "created"}`)); err != nil {
			t.Errorf("failed to write: %v", err)
		}
	})
	req := NewRequest(http.MethodPost, "/").Build()
	w := Serve(handler, req)

	// Test chaining multiple assertions
	result := Assert(w).
		Status(http.StatusCreated).
		Header(httpx.HeaderContentType, "application/json").
		BodyContains("created")

	if result == nil {
		t.Error("expected result to not be nil")
	}
}

func TestAssertWith(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	req := NewRequest(http.MethodGet, "/").Build()
	w := Serve(handler, req)

	// This will use the actual testing.T - just verify it doesn't panic
	result := AssertWith(t, w).Status(http.StatusOK)
	if result == nil {
		t.Error("expected result to not be nil")
	}
}

// Test failure paths - these use t.Errorf but we can't easily capture that,
// so we just verify they don't panic and the chain continues

func TestAssert_FailurePaths(t *testing.T) {
	// Test all failure paths to ensure they don't panic and chain continues
	t.Run("Status failure", func(t *testing.T) {
		w := httptest.NewRecorder()
		w.WriteHeader(http.StatusNotFound)

		result := Assert(w).Status(http.StatusOK)
		if result == nil {
			t.Error("expected chain to continue after failure")
		}
	})

	t.Run("StatusNot failure", func(t *testing.T) {
		w := httptest.NewRecorder()
		w.WriteHeader(http.StatusNotFound)

		result := Assert(w).StatusNot(http.StatusNotFound)
		if result == nil {
			t.Error("expected chain to continue after failure")
		}
	})

	t.Run("StatusBetween failure", func(t *testing.T) {
		w := httptest.NewRecorder()
		w.WriteHeader(http.StatusNotFound)

		result := Assert(w).StatusBetween(200, 299)
		if result == nil {
			t.Error("expected chain to continue after failure")
		}
	})

	t.Run("Header failure", func(t *testing.T) {
		w := httptest.NewRecorder()
		w.Header().Set(httpx.HeaderContentType, httpx.MIMETextPlain)

		result := Assert(w).Header(httpx.HeaderContentType, "application/json")
		if result == nil {
			t.Error("expected chain to continue after failure")
		}
	})

	t.Run("HeaderContains failure", func(t *testing.T) {
		w := httptest.NewRecorder()
		w.Header().Set(httpx.HeaderContentType, httpx.MIMETextPlain)

		result := Assert(w).HeaderContains(httpx.HeaderContentType, "json")
		if result == nil {
			t.Error("expected chain to continue after failure")
		}
	})

	t.Run("HeaderExists failure", func(t *testing.T) {
		w := httptest.NewRecorder()

		result := Assert(w).HeaderExists("X-Missing")
		if result == nil {
			t.Error("expected chain to continue after failure")
		}
	})

	t.Run("HeaderNotExists failure", func(t *testing.T) {
		w := httptest.NewRecorder()
		w.Header().Set("X-Custom", "value")

		result := Assert(w).HeaderNotExists("X-Custom")
		if result == nil {
			t.Error("expected chain to continue after failure")
		}
	})

	t.Run("Body failure", func(t *testing.T) {
		w := httptest.NewRecorder()
		if _, err := w.Write([]byte("hello")); err != nil {
			t.Errorf("failed to write: %v", err)
		}

		result := Assert(w).Body("world")
		if result == nil {
			t.Error("expected chain to continue after failure")
		}
	})

	t.Run("BodyContains failure", func(t *testing.T) {
		w := httptest.NewRecorder()
		if _, err := w.Write([]byte("hello")); err != nil {
			t.Errorf("failed to write: %v", err)
		}

		result := Assert(w).BodyContains("world")
		if result == nil {
			t.Error("expected chain to continue after failure")
		}
	})

	t.Run("BodyNotContains failure", func(t *testing.T) {
		w := httptest.NewRecorder()
		if _, err := w.Write([]byte("hello world")); err != nil {
			t.Errorf("failed to write: %v", err)
		}

		result := Assert(w).BodyNotContains("hello")
		if result == nil {
			t.Error("expected chain to continue after failure")
		}
	})

	t.Run("BodyEmpty failure", func(t *testing.T) {
		w := httptest.NewRecorder()
		if _, err := w.Write([]byte("content")); err != nil {
			t.Errorf("failed to write: %v", err)
		}

		result := Assert(w).BodyEmpty()
		if result == nil {
			t.Error("expected chain to continue after failure")
		}
	})

	t.Run("BodyNotEmpty failure", func(t *testing.T) {
		w := httptest.NewRecorder()

		result := Assert(w).BodyNotEmpty()
		if result == nil {
			t.Error("expected chain to continue after failure")
		}
	})

	t.Run("JSON decode failure", func(t *testing.T) {
		w := httptest.NewRecorder()
		if _, err := w.Write([]byte("not json")); err != nil {
			t.Errorf("failed to write: %v", err)
		}

		var result map[string]string
		a := Assert(w).JSON(&result)
		if a == nil {
			t.Error("expected chain to continue after failure")
		}
	})

	t.Run("JSONEq failure", func(t *testing.T) {
		w := httptest.NewRecorder()
		if _, err := w.Write([]byte(`{"name": "John"}`)); err != nil {
			t.Errorf("failed to write: %v", err)
		}

		result := Assert(w).JSONEq(`{"name": "Jane"}`)
		if result == nil {
			t.Error("expected chain to continue after failure")
		}
	})

	t.Run("JSONEq unmarshal failure", func(t *testing.T) {
		w := httptest.NewRecorder()
		if _, err := w.Write([]byte("not json")); err != nil {
			t.Errorf("failed to write: %v", err)
		}

		result := Assert(w).JSONEq(`{}`)
		if result == nil {
			t.Error("expected chain to continue after failure")
		}
	})

	t.Run("JSONPathEqual failure", func(t *testing.T) {
		w := httptest.NewRecorder()
		if _, err := w.Write([]byte(`{"user": {"name": "John"}}`)); err != nil {
			t.Errorf("failed to write: %v", err)
		}

		result := Assert(w).JSONPathEqual("user.name", "Jane")
		if result == nil {
			t.Error("expected chain to continue after failure")
		}
	})

	t.Run("JSONPathEqual invalid JSON", func(t *testing.T) {
		w := httptest.NewRecorder()
		if _, err := w.Write([]byte("not json")); err != nil {
			t.Errorf("failed to write: %v", err)
		}

		result := Assert(w).JSONPathEqual("user", "value")
		if result == nil {
			t.Error("expected chain to continue after failure")
		}
	})

	t.Run("JSONPathEqual missing key", func(t *testing.T) {
		w := httptest.NewRecorder()
		if _, err := w.Write([]byte(`{"user": {}}`)); err != nil {
			t.Errorf("failed to write: %v", err)
		}

		result := Assert(w).JSONPathEqual("user.name", "John")
		if result == nil {
			t.Error("expected chain to continue after failure")
		}
	})

	t.Run("JSONPathNotEqual failure", func(t *testing.T) {
		w := httptest.NewRecorder()
		if _, err := w.Write([]byte(`{"user": {"name": "John"}}`)); err != nil {
			t.Errorf("failed to write: %v", err)
		}

		result := Assert(w).JSONPathNotEqual("user.name", "John")
		if result == nil {
			t.Error("expected chain to continue after failure")
		}
	})

	t.Run("JSONPathNotEqual invalid JSON", func(t *testing.T) {
		w := httptest.NewRecorder()
		if _, err := w.Write([]byte("not json")); err != nil {
			t.Errorf("failed to write: %v", err)
		}

		result := Assert(w).JSONPathNotEqual("user", "value")
		if result == nil {
			t.Error("expected chain to continue after failure")
		}
	})

	t.Run("JSONPathNotEqual missing key", func(t *testing.T) {
		w := httptest.NewRecorder()
		if _, err := w.Write([]byte(`{"user": {}}`)); err != nil {
			t.Errorf("failed to write: %v", err)
		}

		result := Assert(w).JSONPathNotEqual("user.name", "John")
		if result == nil {
			t.Error("expected chain to continue after failure")
		}
	})

	t.Run("JSONPathExists failure", func(t *testing.T) {
		w := httptest.NewRecorder()
		if _, err := w.Write([]byte(`{"user": {}}`)); err != nil {
			t.Errorf("failed to write: %v", err)
		}

		result := Assert(w).JSONPathExists("user.name")
		if result == nil {
			t.Error("expected chain to continue after failure")
		}
	})

	t.Run("JSONPathExists invalid JSON", func(t *testing.T) {
		w := httptest.NewRecorder()
		if _, err := w.Write([]byte("not json")); err != nil {
			t.Errorf("failed to write: %v", err)
		}

		result := Assert(w).JSONPathExists("user")
		if result == nil {
			t.Error("expected chain to continue after failure")
		}
	})

	t.Run("JSONPathNotExists failure", func(t *testing.T) {
		w := httptest.NewRecorder()
		if _, err := w.Write([]byte(`{"user": {"name": "John"}}`)); err != nil {
			t.Errorf("failed to write: %v", err)
		}

		result := Assert(w).JSONPathNotExists("user.name")
		if result == nil {
			t.Error("expected chain to continue after failure")
		}
	})

	t.Run("JSONPathNotExists invalid JSON", func(t *testing.T) {
		w := httptest.NewRecorder()
		if _, err := w.Write([]byte("not json")); err != nil {
			t.Errorf("failed to write: %v", err)
		}

		result := Assert(w).JSONPathNotExists("user")
		if result == nil {
			t.Error("expected chain to continue after failure")
		}
	})

	t.Run("Cookie wrong value", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.SetCookie(w, &http.Cookie{Name: "session", Value: "wrong"})
			w.WriteHeader(http.StatusOK)
		})
		req := NewRequest(http.MethodGet, "/").Build()
		rec := Serve(handler, req)

		result := Assert(rec).Cookie("session", "expected")
		if result == nil {
			t.Error("expected chain to continue after failure")
		}
	})

	t.Run("Cookie missing", func(t *testing.T) {
		w := httptest.NewRecorder()

		result := Assert(w).Cookie("session", "value")
		if result == nil {
			t.Error("expected chain to continue after failure")
		}
	})

	t.Run("CookieExists failure", func(t *testing.T) {
		w := httptest.NewRecorder()

		result := Assert(w).CookieExists("session")
		if result == nil {
			t.Error("expected chain to continue after failure")
		}
	})

	t.Run("CookieNotExists failure", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.SetCookie(w, &http.Cookie{Name: "session", Value: "abc"})
			w.WriteHeader(http.StatusOK)
		})
		req := NewRequest(http.MethodGet, "/").Build()
		rec := Serve(handler, req)

		result := Assert(rec).CookieNotExists("session")
		if result == nil {
			t.Error("expected chain to continue after failure")
		}
	})

	t.Run("Redirect not a redirect", func(t *testing.T) {
		w := httptest.NewRecorder()
		w.WriteHeader(http.StatusOK)

		result := Assert(w).Redirect("/other")
		if result == nil {
			t.Error("expected chain to continue after failure")
		}
	})

	t.Run("Redirect wrong location", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/actual", http.StatusFound)
		})
		req := NewRequest(http.MethodGet, "/").Build()
		rec := Serve(handler, req)

		result := Assert(rec).Redirect("/expected")
		if result == nil {
			t.Error("expected chain to continue after failure")
		}
	})

	t.Run("IsSuccess failure", func(t *testing.T) {
		w := httptest.NewRecorder()
		w.WriteHeader(http.StatusNotFound)

		result := Assert(w).IsSuccess()
		if result == nil {
			t.Error("expected chain to continue after failure")
		}
	})

	t.Run("IsClientError failure", func(t *testing.T) {
		w := httptest.NewRecorder()
		w.WriteHeader(http.StatusOK)

		result := Assert(w).IsClientError()
		if result == nil {
			t.Error("expected chain to continue after failure")
		}
	})

	t.Run("IsServerError failure", func(t *testing.T) {
		w := httptest.NewRecorder()
		w.WriteHeader(http.StatusOK)

		result := Assert(w).IsServerError()
		if result == nil {
			t.Error("expected chain to continue after failure")
		}
	})
}

// Test jsonValuesEqual edge cases
func TestJSONValuesEqual(t *testing.T) {
	t.Run("different length maps", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set(httpx.HeaderContentType, httpx.MIMEApplicationJSON)
			if _, err := w.Write([]byte(`{"x": 1}`)); err != nil {
				t.Errorf("failed to write: %v", err)
			}
		})
		req := NewRequest(http.MethodGet, "/").Build()
		rec := Serve(handler, req)

		// This should fail because expected has 2 fields but actual has 1
		result := Assert(rec).JSONEq(`{"x": 1, "y": 2}`)
		if result == nil {
			t.Error("expected chain to continue")
		}
	})

	t.Run("different values in maps", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set(httpx.HeaderContentType, httpx.MIMEApplicationJSON)
			if _, err := w.Write([]byte(`{"x": 1}`)); err != nil {
				t.Errorf("failed to write: %v", err)
			}
		})
		req := NewRequest(http.MethodGet, "/").Build()
		rec := Serve(handler, req)

		result := Assert(rec).JSONEq(`{"x": 2}`)
		if result == nil {
			t.Error("expected chain to continue")
		}
	})

	t.Run("different length arrays", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set(httpx.HeaderContentType, httpx.MIMEApplicationJSON)
			if _, err := w.Write([]byte(`[1, 2]`)); err != nil {
				t.Errorf("failed to write: %v", err)
			}
		})
		req := NewRequest(http.MethodGet, "/").Build()
		rec := Serve(handler, req)

		result := Assert(rec).JSONEq(`{"items": [1, 2, 3]}`)
		if result == nil {
			t.Error("expected chain to continue")
		}
	})

	t.Run("nested map with missing key", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set(httpx.HeaderContentType, httpx.MIMEApplicationJSON)
			if _, err := w.Write([]byte(`{"user": {"name": "John"}}`)); err != nil {
				t.Errorf("failed to write: %v", err)
			}
		})
		req := NewRequest(http.MethodGet, "/").Build()
		rec := Serve(handler, req)

		result := Assert(rec).JSONEq(`{"user": {"name": "John", "age": 30}}`)
		if result == nil {
			t.Error("expected chain to continue")
		}
	})

	t.Run("array element mismatch", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set(httpx.HeaderContentType, httpx.MIMEApplicationJSON)
			if _, err := w.Write([]byte(`[1, 2, 4]`)); err != nil {
				t.Errorf("failed to write: %v", err)
			}
		})
		req := NewRequest(http.MethodGet, "/").Build()
		rec := Serve(handler, req)

		result := Assert(rec).JSONEq(`[1, 2, 3]`)
		if result == nil {
			t.Error("expected chain to continue")
		}
	})
}

// Test JSONPathEqual edge cases
func TestJSONPathEqual_EdgeCases(t *testing.T) {
	t.Run("traverse into non-map", func(t *testing.T) {
		w := httptest.NewRecorder()
		if _, err := w.Write([]byte(`{"value": "string"}`)); err != nil {
			t.Errorf("failed to write: %v", err)
		}

		result := Assert(w).JSONPathEqual("value.property", "x")
		if result == nil {
			t.Error("expected chain to continue after failure")
		}
	})

	t.Run("invalid array index", func(t *testing.T) {
		w := httptest.NewRecorder()
		if _, err := w.Write([]byte(`{"items": [1, 2, 3]}`)); err != nil {
			t.Errorf("failed to write: %v", err)
		}

		result := Assert(w).JSONPathEqual("items.invalid", "x")
		if result == nil {
			t.Error("expected chain to continue after failure")
		}
	})

	t.Run("out of bounds array index", func(t *testing.T) {
		w := httptest.NewRecorder()
		if _, err := w.Write([]byte(`{"items": [1, 2, 3]}`)); err != nil {
			t.Errorf("failed to write: %v", err)
		}

		result := Assert(w).JSONPathEqual("items.99", "x")
		if result == nil {
			t.Error("expected chain to continue after failure")
		}
	})

	t.Run("deep path", func(t *testing.T) {
		w := httptest.NewRecorder()
		if _, err := w.Write([]byte(`{"a": {"b": {"c": {"d": "deep"}}}}`)); err != nil {
			t.Errorf("failed to write: %v", err)
		}

		result := Assert(w).JSONPathEqual("a.b.c.d", "wrong")
		if result == nil {
			t.Error("expected chain to continue after failure")
		}
	})
}

// Test Problem ProblemDetail failure paths
func TestProblemDetail_FailurePaths(t *testing.T) {
	t.Run("IsProblemDetail failure", func(t *testing.T) {
		w := httptest.NewRecorder()
		w.Header().Set(httpx.HeaderContentType, httpx.MIMEApplicationJSON)
		if _, err := w.Write([]byte(`{}`)); err != nil {
			t.Errorf("failed to write: %v", err)
		}

		result := Assert(w).IsProblemDetail()
		if result == nil {
			t.Error("expected chain to continue after failure")
		}
	})

	t.Run("ProblemDetailStatus invalid JSON", func(t *testing.T) {
		w := httptest.NewRecorder()
		w.Header().Set(httpx.HeaderContentType, "application/problem+json")
		if _, err := w.Write([]byte("not json")); err != nil {
			t.Errorf("failed to write: %v", err)
		}

		result := Assert(w).ProblemDetailStatus(http.StatusBadRequest)
		if result == nil {
			t.Error("expected chain to continue after failure")
		}
	})

	t.Run("ProblemDetailStatus wrong status", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			detail := problem.NewDetail(http.StatusBadRequest, "Invalid")
			if err := detail.Render(w); err != nil {
				t.Errorf("failed to render: %v", err)
			}
		})
		req := NewRequest(http.MethodGet, "/").Build()
		rec := Serve(handler, req)

		result := Assert(rec).ProblemDetailStatus(http.StatusInternalServerError)
		if result == nil {
			t.Error("expected chain to continue after failure")
		}
	})

	t.Run("ProblemDetailTitle invalid JSON", func(t *testing.T) {
		w := httptest.NewRecorder()
		w.Header().Set(httpx.HeaderContentType, "application/problem+json")
		if _, err := w.Write([]byte("not json")); err != nil {
			t.Errorf("failed to write: %v", err)
		}

		result := Assert(w).ProblemDetailTitle("Bad Request")
		if result == nil {
			t.Error("expected chain to continue after failure")
		}
	})

	t.Run("ProblemDetailTitle wrong title", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			detail := problem.NewDetail(http.StatusBadRequest, "Invalid")
			if err := detail.Render(w); err != nil {
				t.Errorf("failed to render: %v", err)
			}
		})
		req := NewRequest(http.MethodGet, "/").Build()
		rec := Serve(handler, req)

		result := Assert(rec).ProblemDetailTitle("Wrong Title")
		if result == nil {
			t.Error("expected chain to continue after failure")
		}
	})

	t.Run("ProblemDetailDetail invalid JSON", func(t *testing.T) {
		w := httptest.NewRecorder()
		w.Header().Set(httpx.HeaderContentType, "application/problem+json")
		if _, err := w.Write([]byte("not json")); err != nil {
			t.Errorf("failed to write: %v", err)
		}

		result := Assert(w).ProblemDetailDetail("detail")
		if result == nil {
			t.Error("expected chain to continue after failure")
		}
	})

	t.Run("ProblemDetailDetail wrong detail", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			detail := problem.NewDetail(http.StatusBadRequest, "Actual detail")
			if err := detail.Render(w); err != nil {
				t.Errorf("failed to render: %v", err)
			}
		})
		req := NewRequest(http.MethodGet, "/").Build()
		rec := Serve(handler, req)

		result := Assert(rec).ProblemDetailDetail("Expected detail")
		if result == nil {
			t.Error("expected chain to continue after failure")
		}
	})

	t.Run("ProblemDetailType invalid JSON", func(t *testing.T) {
		w := httptest.NewRecorder()
		w.Header().Set(httpx.HeaderContentType, "application/problem+json")
		if _, err := w.Write([]byte("not json")); err != nil {
			t.Errorf("failed to write: %v", err)
		}

		result := Assert(w).ProblemDetailType("type")
		if result == nil {
			t.Error("expected chain to continue after failure")
		}
	})

	t.Run("ProblemDetailType wrong type", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			detail := problem.NewDetail(http.StatusBadRequest, "Invalid")
			detail.Type = "https://api.example.com/actual"
			if err := detail.Render(w); err != nil {
				t.Errorf("failed to render: %v", err)
			}
		})
		req := NewRequest(http.MethodGet, "/").Build()
		rec := Serve(handler, req)

		result := Assert(rec).ProblemDetailType("https://api.example.com/expected")
		if result == nil {
			t.Error("expected chain to continue after failure")
		}
	})

	t.Run("ProblemDetailExtension invalid JSON", func(t *testing.T) {
		w := httptest.NewRecorder()
		w.Header().Set(httpx.HeaderContentType, "application/problem+json")
		if _, err := w.Write([]byte("not json")); err != nil {
			t.Errorf("failed to write: %v", err)
		}

		result := Assert(w).ProblemDetailExtension("key", "value")
		if result == nil {
			t.Error("expected chain to continue after failure")
		}
	})

	t.Run("ProblemDetailExtension missing key", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			detail := problem.NewDetail(http.StatusBadRequest, "Invalid")
			if err := detail.Render(w); err != nil {
				t.Errorf("failed to render: %v", err)
			}
		})
		req := NewRequest(http.MethodGet, "/").Build()
		rec := Serve(handler, req)

		result := Assert(rec).ProblemDetailExtension("missing", "value")
		if result == nil {
			t.Error("expected chain to continue after failure")
		}
	})

	t.Run("ProblemDetailExtension wrong value", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			detail := problem.NewDetail(http.StatusBadRequest, "Invalid")
			detail.Set("key", "actual")
			if err := detail.Render(w); err != nil {
				t.Errorf("failed to render: %v", err)
			}
		})
		req := NewRequest(http.MethodGet, "/").Build()
		rec := Serve(handler, req)

		result := Assert(rec).ProblemDetailExtension("key", "expected")
		if result == nil {
			t.Error("expected chain to continue after failure")
		}
	})

	t.Run("ProblemDetail invalid JSON", func(t *testing.T) {
		w := httptest.NewRecorder()
		w.Header().Set(httpx.HeaderContentType, "application/problem+json")
		if _, err := w.Write([]byte("not json")); err != nil {
			t.Errorf("failed to write: %v", err)
		}

		var detail problem.Detail

		result := Assert(w).ProblemDetail(detail)
		if result == nil {
			t.Error("expected chain to continue after failure")
		}
	})
}

func TestAssert_IsProblemDetail(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		detail := problem.NewDetail(http.StatusBadRequest, "Invalid request")
		if err := detail.Render(w); err != nil {
			t.Errorf("failed to render problem: %v", err)
		}
	})
	req := NewRequest(http.MethodGet, "/").Build()
	w := Serve(handler, req)

	result := Assert(w).IsProblemDetail()
	if result == nil {
		t.Error("expected result to not be nil")
	}
}

func TestAssert_ProblemDetailStatus(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		detail := problem.NewDetail(http.StatusBadRequest, "Invalid request")
		if err := detail.Render(w); err != nil {
			t.Errorf("failed to render problem: %v", err)
		}
	})
	req := NewRequest(http.MethodGet, "/").Build()
	w := Serve(handler, req)

	result := Assert(w).ProblemDetailStatus(http.StatusBadRequest)
	if result == nil {
		t.Error("expected result to not be nil")
	}
}

func TestAssert_ProblemDetailTitle(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		detail := problem.NewDetail(http.StatusBadRequest, "Invalid request")
		if err := detail.Render(w); err != nil {
			t.Errorf("failed to render problem: %v", err)
		}
	})
	req := NewRequest(http.MethodGet, "/").Build()
	w := Serve(handler, req)

	result := Assert(w).ProblemDetailTitle("Bad Request")
	if result == nil {
		t.Error("expected result to not be nil")
	}
}

func TestAssert_ProblemDetailDetail(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		detail := problem.NewDetail(http.StatusBadRequest, "Invalid request")
		if err := detail.Render(w); err != nil {
			t.Errorf("failed to render problem: %v", err)
		}
	})
	req := NewRequest(http.MethodGet, "/").Build()
	w := Serve(handler, req)

	result := Assert(w).ProblemDetailDetail("Invalid request")
	if result == nil {
		t.Error("expected result to not be nil")
	}
}

func TestAssert_ProblemDetailType(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		detail := problem.NewDetail(http.StatusBadRequest, "Invalid request")
		detail.Type = "https://api.example.com/errors/invalid-request"
		if err := detail.Render(w); err != nil {
			t.Errorf("failed to render problem: %v", err)
		}
	})
	req := NewRequest(http.MethodGet, "/").Build()
	w := Serve(handler, req)

	result := Assert(w).ProblemDetailType("https://api.example.com/errors/invalid-request")
	if result == nil {
		t.Error("expected result to not be nil")
	}
}

func TestAssert_ProblemDetailExtension(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		detail := problem.NewDetail(http.StatusUnprocessableEntity, "Validation failed")
		detail.Set("errors", []string{"field1 is required", "field2 is invalid"})
		if err := detail.Render(w); err != nil {
			t.Errorf("failed to render problem: %v", err)
		}
	})
	req := NewRequest(http.MethodGet, "/").Build()
	w := Serve(handler, req)

	result := Assert(w).ProblemDetailExtension("errors", []string{"field1 is required", "field2 is invalid"})
	if result == nil {
		t.Error("expected result to not be nil")
	}
}

func TestAssert_ProblemDetail(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		detail := problem.NewDetail(http.StatusBadRequest, "Invalid request")
		if err := detail.Render(w); err != nil {
			t.Errorf("failed to render problem: %v", err)
		}
	})
	req := NewRequest(http.MethodGet, "/").Build()
	w := Serve(handler, req)

	var detail problem.Detail
	result := Assert(w).ProblemDetail(&detail)
	if result == nil {
		t.Error("expected result to not be nil")
	}
	if detail.Status != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", detail.Status)
	}
	if detail.Detail != "Invalid request" {
		t.Errorf("expected detail 'Invalid request', got %s", detail.Detail)
	}
}

func TestAssert_ProblemDetailChaining(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		detail := problem.NewDetail(http.StatusBadRequest, "Invalid request")
		detail.Type = "https://api.example.com/errors/invalid-request"
		if err := detail.Render(w); err != nil {
			t.Errorf("failed to render problem: %v", err)
		}
	})
	req := NewRequest(http.MethodGet, "/").Build()
	w := Serve(handler, req)

	result := Assert(w).
		IsProblemDetail().
		ProblemDetailStatus(http.StatusBadRequest).
		ProblemDetailTitle("Bad Request").
		ProblemDetailDetail("Invalid request").
		ProblemDetailType("https://api.example.com/errors/invalid-request")

	if result == nil {
		t.Error("expected result to not be nil")
	}
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

func TestAssertNotEqual(t *testing.T) {
	AssertNotEqual(t, 42, 43)
	AssertNotEqual(t, "hello", "world")
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
		AssertNotEmpty(t, []int{1, 2, 3})
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
}

func TestAssertNotContains(t *testing.T) {
	t.Run("int slice", func(t *testing.T) {
		AssertNotContains(t, []int{1, 2, 3}, 4)
	})

	t.Run("string slice", func(t *testing.T) {
		AssertNotContains(t, []string{"a", "b", "c"}, "d")
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
}
