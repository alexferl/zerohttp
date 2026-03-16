package zhtest

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/alexferl/zerohttp/httpx"
)

func TestNewRequest(t *testing.T) {
	t.Run("basic request", func(t *testing.T) {
		req := NewRequest(http.MethodGet, "/users").Build()

		if req.Method != http.MethodGet {
			t.Errorf("expected method GET, got %s", req.Method)
		}
		if req.URL.Path != "/users" {
			t.Errorf("expected path /users, got %s", req.URL.Path)
		}
	})

	t.Run("with headers", func(t *testing.T) {
		req := NewRequest(http.MethodGet, "/").
			WithHeader(httpx.HeaderAuthorization, "Bearer token").
			WithHeader("X-Custom", "value1").
			WithHeader("X-Custom", "value2").
			Build()

		if req.Header.Get(httpx.HeaderAuthorization) != "Bearer token" {
			t.Error("expected Authorization header")
		}

		values := req.Header["X-Custom"]
		if len(values) != 2 || values[0] != "value1" || values[1] != "value2" {
			t.Error("expected X-Custom header with two values")
		}
	})

	t.Run("with headers map", func(t *testing.T) {
		req := NewRequest(http.MethodGet, "/").
			WithHeaders(map[string]string{
				"Authorization": "Bearer token",
				"X-Request-ID":  "abc123",
			}).
			Build()

		if req.Header.Get(httpx.HeaderAuthorization) != "Bearer token" {
			t.Error("expected Authorization header")
		}
		if req.Header.Get("X-Request-ID") != "abc123" {
			t.Error("expected X-Request-ID header")
		}
	})

	t.Run("with query", func(t *testing.T) {
		req := NewRequest(http.MethodGet, "/users").
			WithQuery("page", "1").
			WithQuery("limit", "10").
			Build()

		if req.URL.Query().Get("page") != "1" {
			t.Errorf("expected page=1, got %s", req.URL.Query().Get("page"))
		}
		if req.URL.Query().Get("limit") != "10" {
			t.Errorf("expected limit=10, got %s", req.URL.Query().Get("limit"))
		}
	})

	t.Run("with query in path", func(t *testing.T) {
		req := NewRequest(http.MethodGet, "/users?sort=name").
			WithQuery("page", "1").
			Build()

		if req.URL.Query().Get("sort") != "name" {
			t.Error("expected sort=name from path")
		}
		if req.URL.Query().Get("page") != "1" {
			t.Error("expected page=1 from WithQuery")
		}
	})

	t.Run("with cookie", func(t *testing.T) {
		req := NewRequest(http.MethodGet, "/").
			WithCookie(&http.Cookie{Name: "session", Value: "abc123"}).
			Build()

		cookie, err := req.Cookie("session")
		if err != nil {
			t.Fatalf("expected session cookie, got error: %v", err)
		}
		if cookie.Value != "abc123" {
			t.Errorf("expected cookie value abc123, got %s", cookie.Value)
		}
	})

	t.Run("with body", func(t *testing.T) {
		req := NewRequest(http.MethodPost, "/upload").
			WithBody(strings.NewReader("raw data")).
			Build()

		body := make([]byte, 8)
		n, _ := req.Body.Read(body)
		if string(body[:n]) != "raw data" {
			t.Errorf("expected body 'raw data', got %s", string(body[:n]))
		}
	})

	t.Run("with bytes", func(t *testing.T) {
		req := NewRequest(http.MethodPost, "/upload").
			WithBytes([]byte("raw data")).
			Build()

		body := make([]byte, 8)
		n, _ := req.Body.Read(body)
		if string(body[:n]) != "raw data" {
			t.Errorf("expected body 'raw data', got %s", string(body[:n]))
		}
	})

	t.Run("with JSON", func(t *testing.T) {
		req := NewRequest(http.MethodPost, "/users").
			WithJSON(map[string]string{"name": "John"}).
			Build()

		if req.Header.Get(httpx.HeaderContentType) != "application/json" {
			t.Error("expected Content-Type application/json")
		}

		body := make([]byte, 100)
		n, _ := req.Body.Read(body)
		if !strings.Contains(string(body[:n]), "John") {
			t.Errorf("expected body to contain 'John', got %s", string(body[:n]))
		}
	})

	t.Run("with form", func(t *testing.T) {
		req := NewRequest(http.MethodPost, "/login").
			WithForm(url.Values{"username": []string{"john"}}).
			Build()

		if req.Header.Get(httpx.HeaderContentType) != "application/x-www-form-urlencoded" {
			t.Error("expected Content-Type application/x-www-form-urlencoded")
		}

		body := make([]byte, 100)
		n, _ := req.Body.Read(body)
		if !strings.Contains(string(body[:n]), "username=john") {
			t.Errorf("expected body to contain 'username=john', got %s", string(body[:n]))
		}
	})
}

func TestResponse(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(httpx.HeaderContentType, httpx.MIMEApplicationJSON)
		w.Header().Set("X-Custom", "value")
		http.SetCookie(w, &http.Cookie{Name: "session", Value: "abc123"})
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{"message": "hello"}`)); err != nil {
			t.Errorf("failed to write: %v", err)
		}
	})

	req := NewRequest(http.MethodGet, "/").Build()
	w := ServeWithRecorder(handler, req)

	t.Run("BodyString", func(t *testing.T) {
		if !strings.Contains(w.BodyString(), "hello") {
			t.Errorf("expected body to contain 'hello', got %s", w.BodyString())
		}
	})

	t.Run("BodyBytes", func(t *testing.T) {
		if len(w.BodyBytes()) == 0 {
			t.Error("expected non-empty body bytes")
		}
	})

	t.Run("JSON", func(t *testing.T) {
		var result map[string]string
		if err := w.JSON(&result); err != nil {
			t.Errorf("expected no error decoding JSON, got %v", err)
		}
		if result["message"] != "hello" {
			t.Errorf("expected message 'hello', got %s", result["message"])
		}
	})

	t.Run("Cookie", func(t *testing.T) {
		cookie := w.Cookie("session")
		if cookie == nil {
			t.Fatal("expected session cookie")
		}
		if cookie.Value != "abc123" {
			t.Errorf("expected cookie value abc123, got %s", cookie.Value)
		}
	})

	t.Run("CookieValue", func(t *testing.T) {
		if w.CookieValue("session") != "abc123" {
			t.Error("expected cookie value abc123")
		}
		if w.CookieValue("missing") != "" {
			t.Error("expected empty value for missing cookie")
		}
	})

	t.Run("HeaderValue", func(t *testing.T) {
		if w.HeaderValue("X-Custom") != "value" {
			t.Error("expected X-Custom header value 'value'")
		}
		if w.HeaderValue("Missing") != "" {
			t.Error("expected empty value for missing header")
		}
	})

	t.Run("IsSuccess", func(t *testing.T) {
		if !w.IsSuccess() {
			t.Error("expected IsSuccess to be true for status 200")
		}
	})

	t.Run("IsRedirect", func(t *testing.T) {
		redirectHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/new", http.StatusFound)
		})
		req := NewRequest(http.MethodGet, "/").Build()
		redirectW := ServeWithRecorder(redirectHandler, req)
		if !redirectW.IsRedirect() {
			t.Error("expected IsRedirect to be true for status 302")
		}
	})

	t.Run("IsClientError", func(t *testing.T) {
		notFoundHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		})
		req := NewRequest(http.MethodGet, "/").Build()
		notFoundW := ServeWithRecorder(notFoundHandler, req)
		if !notFoundW.IsClientError() {
			t.Error("expected IsClientError to be true for status 404")
		}
	})

	t.Run("IsServerError", func(t *testing.T) {
		errorHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		})
		req := NewRequest(http.MethodGet, "/").Build()
		errorW := ServeWithRecorder(errorHandler, req)
		if !errorW.IsServerError() {
			t.Error("expected IsServerError to be true for status 500")
		}
	})
}

func TestNewRecorder(t *testing.T) {
	w := NewRecorder()
	if w == nil {
		t.Fatal("expected recorder to not be nil")
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("hello")); err != nil {
			t.Errorf("failed to write: %v", err)
		}
	})
	req := NewRequest(http.MethodGet, "/").Build()
	handler.ServeHTTP(w.ResponseRecorder, req)

	if w.Code != 200 {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestWithJSON_Error(t *testing.T) {
	type badJSON struct {
		Func func()
	}

	req := NewRequest(http.MethodPost, "/").
		WithJSON(badJSON{Func: func() {}}).
		Build()

	body := make([]byte, 100)
	n, err := req.Body.Read(body)
	if err == nil || n > 0 {
		t.Error("expected error when reading body with JSON marshal error")
	}
}

func TestErrorReader(t *testing.T) {
	er := &errorReader{err: errors.New("read error")}
	buf := make([]byte, 10)
	n, err := er.Read(buf)
	if n != 0 {
		t.Errorf("expected 0 bytes read, got %d", n)
	}
	if err == nil || err.Error() != "read error" {
		t.Errorf("expected 'read error', got %v", err)
	}
}

func TestWithQuery_EdgeCases(t *testing.T) {
	t.Run("invalid URL path", func(t *testing.T) {
		req := NewRequest(http.MethodGet, "/path?invalid=%").
			WithQuery("key", "value").
			Build()

		if req.URL.Query().Get("key") != "value" {
			t.Error("expected query param to be set")
		}
	})

	t.Run("URL without scheme", func(t *testing.T) {
		req := NewRequest(http.MethodGet, "//host/path").
			WithQuery("key", "value").
			Build()

		if req.URL.Query().Get("key") != "value" {
			t.Error("expected query param to be set")
		}
	})

	t.Run("overwrite query param", func(t *testing.T) {
		req := NewRequest(http.MethodGet, "/path").
			WithQuery("key", "1").
			WithQuery("key", "2").
			Build()

		if req.URL.Query().Get("key") != "2" {
			t.Errorf("expected key=2, got %s", req.URL.Query().Get("key"))
		}
	})
}

func TestServeWithRecorder_ErrorCases(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		if _, err := w.Write([]byte("error")); err != nil {
			t.Errorf("failed to write: %v", err)
		}
	})
	req := NewRequest(http.MethodGet, "/").Build()
	w := ServeWithRecorder(handler, req)

	if w.IsSuccess() {
		t.Error("expected IsSuccess to be false for 500")
	}
	if !w.IsServerError() {
		t.Error("expected IsServerError to be true for 500")
	}
}

func TestJSON_VariousTypes(t *testing.T) {
	t.Run("array", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set(httpx.HeaderContentType, httpx.MIMEApplicationJSON)
			if _, err := w.Write([]byte(`[1, 2, 3]`)); err != nil {
				t.Errorf("failed to write: %v", err)
			}
		})
		req := NewRequest(http.MethodGet, "/").Build()
		w := ServeWithRecorder(handler, req)

		var result []int
		if err := w.JSON(&result); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if len(result) != 3 {
			t.Errorf("expected 3 elements, got %d", len(result))
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set(httpx.HeaderContentType, httpx.MIMEApplicationJSON)
			if _, err := w.Write([]byte(`{invalid`)); err != nil {
				t.Errorf("failed to write: %v", err)
			}
		})
		req := NewRequest(http.MethodGet, "/").Build()
		w := ServeWithRecorder(handler, req)

		var result map[string]string
		err := w.JSON(&result)
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})
}

func TestBody_Consistency(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write([]byte("test content")); err != nil {
			t.Errorf("failed to write: %v", err)
		}
	})
	req := NewRequest(http.MethodGet, "/").Build()
	w := ServeWithRecorder(handler, req)

	str := w.BodyString()
	bytes := w.BodyBytes()

	if str != string(bytes) {
		t.Error("BodyString and BodyBytes should return consistent results")
	}
}

func TestCookieValue_Missing(t *testing.T) {
	w := httptest.NewRecorder()
	resp := &Response{ResponseRecorder: w}

	if resp.CookieValue("missing") != "" {
		t.Error("expected empty string for missing cookie")
	}
}

func TestHeaderValue_Missing(t *testing.T) {
	w := httptest.NewRecorder()
	resp := &Response{ResponseRecorder: w}

	if resp.HeaderValue("X-NonExistent") != "" {
		t.Error("expected empty string for non-existent header")
	}
}

func TestIsRedirect_Codes(t *testing.T) {
	redirectCodes := []int{301, 302, 303, 307, 308}
	for _, code := range redirectCodes {
		t.Run(fmt.Sprintf("status %d", code), func(t *testing.T) {
			w := httptest.NewRecorder()
			w.WriteHeader(code)

			resp := &Response{ResponseRecorder: w}
			if !resp.IsRedirect() {
				t.Errorf("expected IsRedirect to be true for status %d", code)
			}
		})
	}
}

func TestWithBody_Content(t *testing.T) {
	bodyContent := strings.NewReader("raw body content")
	req := NewRequest(http.MethodPost, "/upload").
		WithBody(bodyContent).
		Build()

	buf := make([]byte, 100)
	n, _ := req.Body.Read(buf)

	if string(buf[:n]) != "raw body content" {
		t.Errorf("expected 'raw body content', got '%s'", string(buf[:n]))
	}
}

func TestWithBytes_Content(t *testing.T) {
	req := NewRequest(http.MethodPost, "/upload").
		WithBytes([]byte("byte slice content")).
		Build()

	buf := make([]byte, 100)
	n, _ := req.Body.Read(buf)

	if string(buf[:n]) != "byte slice content" {
		t.Errorf("expected 'byte slice content', got '%s'", string(buf[:n]))
	}
}
