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

		AssertEqual(t, http.MethodGet, req.Method)
		AssertEqual(t, "/users", req.URL.Path)
	})

	t.Run("with headers", func(t *testing.T) {
		req := NewRequest(http.MethodGet, "/").
			WithHeader(httpx.HeaderAuthorization, "Bearer token").
			WithHeader("X-Custom", "value1").
			WithHeader("X-Custom", "value2").
			Build()

		AssertEqual(t, "Bearer token", req.Header.Get(httpx.HeaderAuthorization))

		values := req.Header["X-Custom"]
		AssertEqual(t, 2, len(values))
		AssertEqual(t, "value1", values[0])
		AssertEqual(t, "value2", values[1])
	})

	t.Run("with headers map", func(t *testing.T) {
		req := NewRequest(http.MethodGet, "/").
			WithHeaders(map[string]string{
				"Authorization": "Bearer token",
				"X-Request-ID":  "abc123",
			}).
			Build()

		AssertEqual(t, "Bearer token", req.Header.Get(httpx.HeaderAuthorization))
		AssertEqual(t, "abc123", req.Header.Get("X-Request-ID"))
	})

	t.Run("with query", func(t *testing.T) {
		req := NewRequest(http.MethodGet, "/users").
			WithQuery("page", "1").
			WithQuery("limit", "10").
			Build()

		AssertEqual(t, "1", req.URL.Query().Get("page"))
		AssertEqual(t, "10", req.URL.Query().Get("limit"))
	})

	t.Run("with query in path", func(t *testing.T) {
		req := NewRequest(http.MethodGet, "/users?sort=name").
			WithQuery("page", "1").
			Build()

		AssertEqual(t, "name", req.URL.Query().Get("sort"))
		AssertEqual(t, "1", req.URL.Query().Get("page"))
	})

	t.Run("with cookie", func(t *testing.T) {
		req := NewRequest(http.MethodGet, "/").
			WithCookie(&http.Cookie{Name: "session", Value: "abc123"}).
			Build()

		cookie, err := req.Cookie("session")
		AssertNoError(t, err)
		AssertNotNil(t, cookie)
		AssertEqual(t, "abc123", cookie.Value)
	})

	t.Run("with body", func(t *testing.T) {
		req := NewRequest(http.MethodPost, "/upload").
			WithBody(strings.NewReader("raw data")).
			Build()

		body := make([]byte, 8)
		n, _ := req.Body.Read(body)
		AssertEqual(t, "raw data", string(body[:n]))
	})

	t.Run("with bytes", func(t *testing.T) {
		req := NewRequest(http.MethodPost, "/upload").
			WithBytes([]byte("raw data")).
			Build()

		body := make([]byte, 8)
		n, _ := req.Body.Read(body)
		AssertEqual(t, "raw data", string(body[:n]))
	})

	t.Run("with JSON", func(t *testing.T) {
		req := NewRequest(http.MethodPost, "/users").
			WithJSON(map[string]string{"name": "John"}).
			Build()

		AssertEqual(t, "application/json", req.Header.Get(httpx.HeaderContentType))

		body := make([]byte, 100)
		n, _ := req.Body.Read(body)
		AssertTrue(t, strings.Contains(string(body[:n]), "John"))
	})

	t.Run("with form", func(t *testing.T) {
		req := NewRequest(http.MethodPost, "/login").
			WithForm(url.Values{"username": []string{"john"}}).
			Build()

		AssertEqual(t, "application/x-www-form-urlencoded", req.Header.Get(httpx.HeaderContentType))

		body := make([]byte, 100)
		n, _ := req.Body.Read(body)
		AssertTrue(t, strings.Contains(string(body[:n]), "username=john"))
	})
}

func TestResponse(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(httpx.HeaderContentType, httpx.MIMEApplicationJSON)
		w.Header().Set("X-Custom", "value")
		http.SetCookie(w, &http.Cookie{Name: "session", Value: "abc123"})
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`{"message": "hello"}`))
		AssertNoError(t, err)
	})

	req := NewRequest(http.MethodGet, "/").Build()
	w := ServeWithRecorder(handler, req)

	t.Run("BodyString", func(t *testing.T) {
		AssertTrue(t, strings.Contains(w.BodyString(), "hello"))
	})

	t.Run("BodyBytes", func(t *testing.T) {
		AssertTrue(t, len(w.BodyBytes()) > 0)
	})

	t.Run("JSON", func(t *testing.T) {
		var result map[string]string
		err := w.JSON(&result)
		AssertNoError(t, err)
		AssertEqual(t, "hello", result["message"])
	})

	t.Run("Cookie", func(t *testing.T) {
		cookie := w.Cookie("session")
		AssertNotNil(t, cookie)
		AssertEqual(t, "abc123", cookie.Value)
	})

	t.Run("CookieValue", func(t *testing.T) {
		AssertEqual(t, "abc123", w.CookieValue("session"))
		AssertEqual(t, "", w.CookieValue("missing"))
	})

	t.Run("HeaderValue", func(t *testing.T) {
		AssertEqual(t, "value", w.HeaderValue("X-Custom"))
		AssertEqual(t, "", w.HeaderValue("Missing"))
	})

	t.Run("IsSuccess", func(t *testing.T) {
		AssertTrue(t, w.IsSuccess())
	})

	t.Run("IsRedirect", func(t *testing.T) {
		redirectHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/new", http.StatusFound)
		})
		req := NewRequest(http.MethodGet, "/").Build()
		redirectW := ServeWithRecorder(redirectHandler, req)
		AssertTrue(t, redirectW.IsRedirect())
	})

	t.Run("IsClientError", func(t *testing.T) {
		notFoundHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		})
		req := NewRequest(http.MethodGet, "/").Build()
		notFoundW := ServeWithRecorder(notFoundHandler, req)
		AssertTrue(t, notFoundW.IsClientError())
	})

	t.Run("IsServerError", func(t *testing.T) {
		errorHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		})
		req := NewRequest(http.MethodGet, "/").Build()
		errorW := ServeWithRecorder(errorHandler, req)
		AssertTrue(t, errorW.IsServerError())
	})
}

func TestNewRecorder(t *testing.T) {
	w := NewRecorder()
	AssertNotNil(t, w)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("hello"))
		AssertNoError(t, err)
	})
	req := NewRequest(http.MethodGet, "/").Build()
	handler.ServeHTTP(w.ResponseRecorder, req)

	AssertEqual(t, 200, w.Code)
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
	AssertTrue(t, err != nil || n == 0)
}

func TestErrorReader(t *testing.T) {
	er := &errorReader{err: errors.New("read error")}
	buf := make([]byte, 10)
	n, err := er.Read(buf)
	AssertEqual(t, 0, n)
	AssertError(t, err)
	AssertErrorContains(t, err, "read error")
}

func TestWithQuery_EdgeCases(t *testing.T) {
	t.Run("invalid URL path", func(t *testing.T) {
		req := NewRequest(http.MethodGet, "/path?invalid=%").
			WithQuery("key", "value").
			Build()

		AssertEqual(t, "value", req.URL.Query().Get("key"))
	})

	t.Run("URL without scheme", func(t *testing.T) {
		req := NewRequest(http.MethodGet, "//host/path").
			WithQuery("key", "value").
			Build()

		AssertEqual(t, "value", req.URL.Query().Get("key"))
	})

	t.Run("overwrite query param", func(t *testing.T) {
		req := NewRequest(http.MethodGet, "/path").
			WithQuery("key", "1").
			WithQuery("key", "2").
			Build()

		AssertEqual(t, "2", req.URL.Query().Get("key"))
	})
}

func TestServeWithRecorder_ErrorCases(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, err := w.Write([]byte("error"))
		AssertNoError(t, err)
	})
	req := NewRequest(http.MethodGet, "/").Build()
	w := ServeWithRecorder(handler, req)

	AssertFalse(t, w.IsSuccess())
	AssertTrue(t, w.IsServerError())
}

func TestJSON_VariousTypes(t *testing.T) {
	t.Run("array", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set(httpx.HeaderContentType, httpx.MIMEApplicationJSON)
			_, err := w.Write([]byte(`[1, 2, 3]`))
			AssertNoError(t, err)
		})
		req := NewRequest(http.MethodGet, "/").Build()
		w := ServeWithRecorder(handler, req)

		var result []int
		err := w.JSON(&result)
		AssertNoError(t, err)
		AssertEqual(t, 3, len(result))
	})

	t.Run("invalid JSON", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set(httpx.HeaderContentType, httpx.MIMEApplicationJSON)
			_, err := w.Write([]byte(`{invalid`))
			AssertNoError(t, err)
		})
		req := NewRequest(http.MethodGet, "/").Build()
		w := ServeWithRecorder(handler, req)

		var result map[string]string
		err := w.JSON(&result)
		AssertError(t, err)
	})
}

func TestBody_Consistency(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("test content"))
		AssertNoError(t, err)
	})
	req := NewRequest(http.MethodGet, "/").Build()
	w := ServeWithRecorder(handler, req)

	str := w.BodyString()
	bytes := w.BodyBytes()

	AssertEqual(t, str, string(bytes))
}

func TestCookieValue_Missing(t *testing.T) {
	w := httptest.NewRecorder()
	resp := &Response{ResponseRecorder: w}

	AssertEqual(t, "", resp.CookieValue("missing"))
}

func TestHeaderValue_Missing(t *testing.T) {
	w := httptest.NewRecorder()
	resp := &Response{ResponseRecorder: w}

	AssertEqual(t, "", resp.HeaderValue("X-NonExistent"))
}

func TestIsRedirect_Codes(t *testing.T) {
	redirectCodes := []int{301, 302, 303, 307, 308}
	for _, code := range redirectCodes {
		t.Run(fmt.Sprintf("status %d", code), func(t *testing.T) {
			w := httptest.NewRecorder()
			w.WriteHeader(code)

			resp := &Response{ResponseRecorder: w}
			AssertTrue(t, resp.IsRedirect())
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

	AssertEqual(t, "raw body content", string(buf[:n]))
}

func TestWithBytes_Content(t *testing.T) {
	req := NewRequest(http.MethodPost, "/upload").
		WithBytes([]byte("byte slice content")).
		Build()

	buf := make([]byte, 100)
	n, _ := req.Body.Read(buf)

	AssertEqual(t, "byte slice content", string(buf[:n]))
}
