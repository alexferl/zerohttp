package zerohttp

import (
	"errors"
	"html/template"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alexferl/zerohttp/httpx"
	"github.com/alexferl/zerohttp/zhtest"
)

func TestRenderer_JSON(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		data       any
		expected   string
	}{
		{"simple object", http.StatusOK, M{"message": "hello"}, `{"message":"hello"}`},
		{"array", http.StatusCreated, []string{"a", "b"}, `["a","b"]`},
		{"string", http.StatusAccepted, "test", `"test"`},
		{"number", http.StatusOK, 42, `42`},
		{"boolean", http.StatusOK, true, `true`},
		{"null", http.StatusOK, nil, `null`},
		{"empty M", http.StatusOK, M{}, `{}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()

			zhtest.AssertNoError(t, R.JSON(w, tt.statusCode, tt.data))
			zhtest.AssertWith(t, w).
				Status(tt.statusCode).
				Header(httpx.HeaderContentType, httpx.MIMEApplicationJSONCharset).
				JSONEq(tt.expected)
		})
	}
}

func TestRenderer_JSON_Error(t *testing.T) {
	w := httptest.NewRecorder()
	invalidData := map[string]any{"func": func() {}}

	err := R.JSON(w, http.StatusOK, invalidData)
	zhtest.AssertError(t, err)
}

func TestRenderer_Text(t *testing.T) {
	tests := []struct {
		name string
		data string
	}{
		{"simple", "Hello, World!"},
		{"empty", ""},
		{"multiline", "Line 1\nLine 2"},
		{"unicode", "Hello 世界 🌍"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()

			zhtest.AssertNoError(t, R.Text(w, http.StatusOK, tt.data))
			zhtest.AssertWith(t, w).
				Header(httpx.HeaderContentType, httpx.MIMETextPlainCharset).
				Body(tt.data)
		})
	}
}

func TestRenderer_HTML(t *testing.T) {
	w := httptest.NewRecorder()
	html := "<h1>Test</h1>"

	zhtest.AssertNoError(t, R.HTML(w, http.StatusOK, html))
	zhtest.AssertWith(t, w).
		Header(httpx.HeaderContentType, httpx.MIMETextHTMLCharset).
		Body(html)
}

func TestRenderer_Template(t *testing.T) {
	tmplContent := `{{define "test.html"}}<html><head><title>{{.Title}}</title></head><body><h1>{{.Title}}</h1></body></html>{{end}}`
	tmpl := template.Must(template.New("test").Parse(tmplContent))

	t.Run("renders template successfully", func(t *testing.T) {
		w := httptest.NewRecorder()
		data := map[string]string{"Title": "Test Page"}

		err := R.Template(w, http.StatusOK, tmpl, "test.html", data)
		zhtest.AssertNoError(t, err)
		zhtest.AssertWith(t, w).
			Status(http.StatusOK).
			Header(httpx.HeaderContentType, httpx.MIMETextHTMLCharset).
			BodyContains("<title>Test Page</title>")
	})

	t.Run("returns error for missing template", func(t *testing.T) {
		w := httptest.NewRecorder()

		err := R.Template(w, http.StatusOK, tmpl, "missing.html", nil)
		zhtest.AssertError(t, err)
	})

	t.Run("sets correct status code", func(t *testing.T) {
		w := httptest.NewRecorder()

		err := R.Template(w, http.StatusCreated, tmpl, "test.html", map[string]string{"Title": "Created"})
		zhtest.AssertNoError(t, err)
		zhtest.AssertWith(t, w).Status(http.StatusCreated)
	})

	t.Run("handles nil data", func(t *testing.T) {
		w := httptest.NewRecorder()

		err := R.Template(w, http.StatusOK, tmpl, "test.html", nil)
		zhtest.AssertNoError(t, err)
		zhtest.AssertWith(t, w).BodyContains("<html>")
	})
}

func TestRenderer_Blob(t *testing.T) {
	data := []byte{0x89, 0x50, 0x4E, 0x47} // PNG header
	w := httptest.NewRecorder()

	zhtest.AssertNoError(t, R.Blob(w, http.StatusOK, "image/png", data))
	zhtest.AssertWith(t, w).
		Header(httpx.HeaderContentType, "image/png").
		Body(string(data))
}

func TestRenderer_Stream(t *testing.T) {
	data := "streaming content"
	w := httptest.NewRecorder()

	err := R.Stream(w, http.StatusOK, httpx.MIMETextPlainCharset, strings.NewReader(data))
	zhtest.AssertNoError(t, err)
	zhtest.AssertWith(t, w).Body(data)
}

func TestRenderer_Stream_Error(t *testing.T) {
	w := httptest.NewRecorder()
	errorReader := &errorReader{err: errors.New("read error")}

	err := R.Stream(w, http.StatusOK, httpx.MIMETextPlainCharset, errorReader)
	zhtest.AssertError(t, err)
}

type errorReader struct{ err error }

func (er *errorReader) Read(p []byte) (n int, err error) { return 0, er.err }

func TestRenderer_File(t *testing.T) {
	tests := []struct {
		name        string
		filename    string
		content     string
		contentType string
	}{
		{"text", "test.txt", "Hello!", httpx.MIMETextPlainCharset},
		{"json", "test.json", `{"test":"value"}`, httpx.MIMEApplicationJSONCharset},
		{"html", "test.html", "<h1>Test</h1>", httpx.MIMETextHTMLCharset},
	}

	tempDir := t.TempDir()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := filepath.Join(tempDir, tt.filename)
			zhtest.AssertNoError(t, os.WriteFile(filePath, []byte(tt.content), 0o644))

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/test", nil)

			zhtest.AssertNoError(t, R.File(w, r, filePath))
			zhtest.AssertWith(t, w).
				Status(http.StatusOK).
				Header(httpx.HeaderContentType, tt.contentType).
				Body(tt.content)
		})
	}
}

func TestRenderer_File_NonExistent(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/test", nil)

	err := R.File(w, r, "/nonexistent.txt")
	zhtest.AssertError(t, err)
	zhtest.AssertTrue(t, os.IsNotExist(err))
}

func TestRenderer_File_Range(t *testing.T) {
	tempDir := t.TempDir()
	content := "0123456789ABCDEF"
	filePath := filepath.Join(tempDir, "test.txt")

	zhtest.AssertNoError(t, os.WriteFile(filePath, []byte(content), 0o644))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.Header.Set(httpx.HeaderRange, "bytes=5-10")

	zhtest.AssertNoError(t, R.File(w, r, filePath))
	zhtest.AssertWith(t, w).
		Status(http.StatusPartialContent).
		Body("56789A")
}

func TestRenderer_NoContent(t *testing.T) {
	w := httptest.NewRecorder()

	zhtest.AssertNoError(t, R.NoContent(w))
	zhtest.AssertWith(t, w).
		Status(http.StatusNoContent).
		BodyEmpty().
		HeaderNotExists(httpx.HeaderContentType)
}

func TestRenderer_NotModified(t *testing.T) {
	w := httptest.NewRecorder()

	zhtest.AssertNoError(t, R.NotModified(w))
	zhtest.AssertWith(t, w).
		Status(http.StatusNotModified).
		BodyEmpty().
		HeaderNotExists(httpx.HeaderContentType)
}

func TestRenderer_Redirect(t *testing.T) {
	tests := []struct {
		name       string
		url        string
		statusCode int
	}{
		{"permanent redirect", "/new-location", http.StatusMovedPermanently},
		{"temporary redirect", "/temp-location", http.StatusFound},
		{"see other", "/other", http.StatusSeeOther},
		{"temporary redirect (307)", "/temp", http.StatusTemporaryRedirect},
		{"permanent redirect (308)", "/perm", http.StatusPermanentRedirect},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/original", nil)

			zhtest.AssertNoError(t, R.Redirect(w, r, tt.url, tt.statusCode))
			zhtest.AssertWith(t, w).
				Status(tt.statusCode).
				Header(httpx.HeaderLocation, tt.url).
				BodyContains(tt.url)
		})
	}
}

func TestRenderer_Redirect_AbsoluteURL(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/original", nil)

	absoluteURL := "https://example.com/external"

	zhtest.AssertNoError(t, R.Redirect(w, r, absoluteURL, http.StatusFound))
	zhtest.AssertWith(t, w).Header(httpx.HeaderLocation, absoluteURL)
}

func TestRenderer_Redirect_WithQuery(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/original?param=value", nil)

	redirectURL := "/login?next=/original"

	zhtest.AssertNoError(t, R.Redirect(w, r, redirectURL, http.StatusFound))
	zhtest.AssertWith(t, w).Header(httpx.HeaderLocation, redirectURL)
}

func TestRenderer_ProblemDetail(t *testing.T) {
	t.Run("basic problem detail", func(t *testing.T) {
		w := httptest.NewRecorder()

		problem := NewProblemDetail(http.StatusNotFound, "Resource not found")
		problem.Instance = "/test/resource"

		zhtest.AssertNoError(t, R.ProblemDetail(w, problem))
		zhtest.AssertWith(t, w).
			Status(http.StatusNotFound).
			Header(httpx.HeaderContentType, httpx.MIMEApplicationProblemJSON).
			IsProblemDetail().
			ProblemDetailTitle("Not Found").
			ProblemDetailDetail("Resource not found")
	})

	t.Run("different status codes", func(t *testing.T) {
		tests := []struct {
			name   string
			status int
			title  string
		}{
			{"bad request", http.StatusBadRequest, "Bad Request"},
			{"unauthorized", http.StatusUnauthorized, "Unauthorized"},
			{"forbidden", http.StatusForbidden, "Forbidden"},
			{"internal error", http.StatusInternalServerError, "Internal Server Error"},
			{"service unavailable", http.StatusServiceUnavailable, "Service Unavailable"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				w := httptest.NewRecorder()

				problem := NewProblemDetail(tt.status, "test detail")

				zhtest.AssertNoError(t, R.ProblemDetail(w, problem))
				zhtest.AssertWith(t, w).
					Status(tt.status).
					IsProblemDetail().
					ProblemDetailTitle(tt.title)
			})
		}
	})
}
