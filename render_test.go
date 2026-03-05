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

			if err := R.JSON(w, tt.statusCode, tt.data); err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			zhtest.AssertWith(t, w).
				Status(tt.statusCode).
				Header(HeaderContentType, MIMEApplicationJSON).
				JSONEq(tt.expected)
		})
	}
}

func TestRenderer_JSON_Error(t *testing.T) {
	w := httptest.NewRecorder()
	invalidData := map[string]any{"func": func() {}}

	err := R.JSON(w, http.StatusOK, invalidData)
	if err == nil {
		t.Error("expected error for invalid data")
	}
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

			if err := R.Text(w, http.StatusOK, tt.data); err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			zhtest.AssertWith(t, w).
				Header(HeaderContentType, MIMETextPlain).
				Body(tt.data)
		})
	}
}

func TestRenderer_HTML(t *testing.T) {
	w := httptest.NewRecorder()
	html := "<h1>Test</h1>"

	if err := R.HTML(w, http.StatusOK, html); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	zhtest.AssertWith(t, w).
		Header(HeaderContentType, MIMETextHTML).
		Body(html)
}

func TestRenderer_Template(t *testing.T) {
	tmplContent := `{{define "test.html"}}<html><head><title>{{.Title}}</title></head><body><h1>{{.Title}}</h1></body></html>{{end}}`
	tmpl := template.Must(template.New("test").Parse(tmplContent))

	t.Run("renders template successfully", func(t *testing.T) {
		w := httptest.NewRecorder()
		data := map[string]string{"Title": "Test Page"}

		err := R.Template(w, http.StatusOK, tmpl, "test.html", data)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		zhtest.AssertWith(t, w).
			Status(http.StatusOK).
			Header(HeaderContentType, MIMETextHTML).
			BodyContains("<title>Test Page</title>")
	})

	t.Run("returns error for missing template", func(t *testing.T) {
		w := httptest.NewRecorder()

		err := R.Template(w, http.StatusOK, tmpl, "missing.html", nil)

		if err == nil {
			t.Fatal("expected error for missing template")
		}
	})

	t.Run("sets correct status code", func(t *testing.T) {
		w := httptest.NewRecorder()

		err := R.Template(w, http.StatusCreated, tmpl, "test.html", map[string]string{"Title": "Created"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		zhtest.AssertWith(t, w).Status(http.StatusCreated)
	})

	t.Run("handles nil data", func(t *testing.T) {
		w := httptest.NewRecorder()

		err := R.Template(w, http.StatusOK, tmpl, "test.html", nil)
		if err != nil {
			t.Fatalf("expected no error with nil data, got %v", err)
		}

		zhtest.AssertWith(t, w).BodyContains("<html>")
	})
}

func TestRenderer_Blob(t *testing.T) {
	data := []byte{0x89, 0x50, 0x4E, 0x47} // PNG header
	w := httptest.NewRecorder()

	if err := R.Blob(w, http.StatusOK, "image/png", data); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	zhtest.AssertWith(t, w).
		Header(HeaderContentType, "image/png").
		Body(string(data))
}

func TestRenderer_Stream(t *testing.T) {
	data := "streaming content"
	w := httptest.NewRecorder()

	if err := R.Stream(w, http.StatusOK, MIMETextPlain, strings.NewReader(data)); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	zhtest.AssertWith(t, w).Body(data)
}

func TestRenderer_Stream_Error(t *testing.T) {
	w := httptest.NewRecorder()
	errorReader := &errorReader{err: errors.New("read error")}

	err := R.Stream(w, http.StatusOK, MIMETextPlain, errorReader)
	if err == nil {
		t.Error("expected error from reader")
	}
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
		{"text", "test.txt", "Hello!", MIMETextPlain},
		{"json", "test.json", `{"test":"value"}`, MIMEApplicationJSON},
		{"html", "test.html", "<h1>Test</h1>", MIMETextHTML},
	}

	tempDir := t.TempDir()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := filepath.Join(tempDir, tt.filename)
			if err := os.WriteFile(filePath, []byte(tt.content), 0o644); err != nil {
				t.Fatalf("failed to create test file: %v", err)
			}

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/test", nil)

			if err := R.File(w, r, filePath); err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			zhtest.AssertWith(t, w).
				Status(http.StatusOK).
				Header(HeaderContentType, tt.contentType).
				Body(tt.content)
		})
	}
}

func TestRenderer_File_NonExistent(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/test", nil)

	err := R.File(w, r, "/nonexistent.txt")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}

	if !os.IsNotExist(err) {
		t.Errorf("expected file not found error, got %v", err)
	}
}

func TestRenderer_File_Range(t *testing.T) {
	tempDir := t.TempDir()
	content := "0123456789ABCDEF"
	filePath := filepath.Join(tempDir, "test.txt")

	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.Header.Set(HeaderRange, "bytes=5-10")

	if err := R.File(w, r, filePath); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	zhtest.AssertWith(t, w).
		Status(http.StatusPartialContent).
		Body("56789A")
}

func TestRenderer_NoContent(t *testing.T) {
	w := httptest.NewRecorder()

	if err := R.NoContent(w); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	zhtest.AssertWith(t, w).
		Status(http.StatusNoContent).
		BodyEmpty().
		HeaderNotExists(HeaderContentType)
}

func TestRenderer_NotModified(t *testing.T) {
	w := httptest.NewRecorder()

	if err := R.NotModified(w); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	zhtest.AssertWith(t, w).
		Status(http.StatusNotModified).
		BodyEmpty().
		HeaderNotExists(HeaderContentType)
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

			if err := R.Redirect(w, r, tt.url, tt.statusCode); err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			zhtest.AssertWith(t, w).
				Status(tt.statusCode).
				Header(HeaderLocation, tt.url).
				BodyContains(tt.url)
		})
	}
}

func TestRenderer_Redirect_AbsoluteURL(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/original", nil)

	absoluteURL := "https://example.com/external"

	if err := R.Redirect(w, r, absoluteURL, http.StatusFound); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	zhtest.AssertWith(t, w).Header(HeaderLocation, absoluteURL)
}

func TestRenderer_Redirect_WithQuery(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/original?param=value", nil)

	redirectURL := "/login?next=/original"

	if err := R.Redirect(w, r, redirectURL, http.StatusFound); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	zhtest.AssertWith(t, w).Header(HeaderLocation, redirectURL)
}
