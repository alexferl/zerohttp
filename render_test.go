package zerohttp

import (
	"bytes"
	"errors"
	"html/template"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderer_JSON(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		data       any
		expected   string
	}{
		{"simple object", 200, M{"message": "hello"}, `{"message":"hello"}`},
		{"array", 201, []string{"a", "b"}, `["a","b"]`},
		{"string", 202, "test", `"test"`},
		{"number", 200, 42, `42`},
		{"boolean", 200, true, `true`},
		{"null", 200, nil, `null`},
		{"empty M", 200, M{}, `{}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()

			if err := R.JSON(w, tt.statusCode, tt.data); err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			if w.Code != tt.statusCode {
				t.Errorf("expected status %d, got %d", tt.statusCode, w.Code)
			}

			if ct := w.Header().Get(HeaderContentType); ct != MIMEApplicationJSON {
				t.Errorf("expected Content-Type %s, got %s", MIMEApplicationJSON, ct)
			}

			body := strings.TrimSpace(w.Body.String())
			if body != tt.expected {
				t.Errorf("expected body %s, got %s", tt.expected, body)
			}
		})
	}
}

func TestRenderer_JSON_Error(t *testing.T) {
	w := httptest.NewRecorder()
	invalidData := map[string]any{"func": func() {}}

	err := R.JSON(w, 200, invalidData)
	if err == nil {
		t.Error("expected error for invalid data")
	}

	if w.Code != 200 {
		t.Errorf("expected status 200 even on error, got %d", w.Code)
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

			if err := R.Text(w, 200, tt.data); err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			if ct := w.Header().Get(HeaderContentType); ct != MIMETextPlain {
				t.Errorf("expected Content-Type %s, got %s", MIMETextPlain, ct)
			}

			if body := w.Body.String(); body != tt.data {
				t.Errorf("expected body %s, got %s", tt.data, body)
			}
		})
	}
}

func TestRenderer_HTML(t *testing.T) {
	w := httptest.NewRecorder()
	html := "<h1>Test</h1>"

	if err := R.HTML(w, 200, html); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if ct := w.Header().Get(HeaderContentType); ct != MIMETextHTML {
		t.Errorf("expected Content-Type %s, got %s", MIMETextHTML, ct)
	}

	if body := w.Body.String(); body != html {
		t.Errorf("expected body %s, got %s", html, body)
	}
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

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		if ct := w.Header().Get(HeaderContentType); ct != MIMETextHTML {
			t.Errorf("expected Content-Type %s, got %s", MIMETextHTML, ct)
		}

		body := w.Body.String()
		if !strings.Contains(body, "Test Page") {
			t.Errorf("expected body to contain 'Test Page', got %s", body)
		}
		if !strings.Contains(body, "<title>Test Page</title>") {
			t.Errorf("expected body to contain title tag, got %s", body)
		}
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

		if w.Code != http.StatusCreated {
			t.Errorf("expected status 201, got %d", w.Code)
		}
	})

	t.Run("handles nil data", func(t *testing.T) {
		w := httptest.NewRecorder()

		err := R.Template(w, http.StatusOK, tmpl, "test.html", nil)
		if err != nil {
			t.Fatalf("expected no error with nil data, got %v", err)
		}

		body := w.Body.String()
		if !strings.Contains(body, "<html>") {
			t.Errorf("expected valid HTML structure, got %s", body)
		}
	})
}

func TestRenderer_Blob(t *testing.T) {
	data := []byte{0x89, 0x50, 0x4E, 0x47} // PNG header
	w := httptest.NewRecorder()

	if err := R.Blob(w, 200, "image/png", data); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if ct := w.Header().Get(HeaderContentType); ct != "image/png" {
		t.Errorf("expected Content-Type image/png, got %s", ct)
	}

	if !bytes.Equal(w.Body.Bytes(), data) {
		t.Error("expected body to match data")
	}
}

func TestRenderer_Stream(t *testing.T) {
	data := "streaming content"
	w := httptest.NewRecorder()

	if err := R.Stream(w, 200, "text/plain", strings.NewReader(data)); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if body := w.Body.String(); body != data {
		t.Errorf("expected body %s, got %s", data, body)
	}
}

func TestRenderer_Stream_Error(t *testing.T) {
	w := httptest.NewRecorder()
	errorReader := &errorReader{err: errors.New("read error")}

	err := R.Stream(w, 200, "text/plain", errorReader)
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
		{"text", "test.txt", "Hello!", "text/plain; charset=utf-8"},
		{"json", "test.json", `{"test":"value"}`, "application/json"},
		{"html", "test.html", "<h1>Test</h1>", "text/html; charset=utf-8"},
	}

	tempDir := t.TempDir()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := filepath.Join(tempDir, tt.filename)
			if err := os.WriteFile(filePath, []byte(tt.content), 0o644); err != nil {
				t.Fatalf("failed to create test file: %v", err)
			}

			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/test", nil)

			if err := R.File(w, r, filePath); err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			if w.Code != 200 {
				t.Errorf("expected status 200, got %d", w.Code)
			}

			if ct := w.Header().Get(HeaderContentType); ct != tt.contentType {
				t.Errorf("expected Content-Type %s, got %s", tt.contentType, ct)
			}

			if body := w.Body.String(); body != tt.content {
				t.Errorf("expected body %s, got %s", tt.content, body)
			}
		})
	}
}

func TestRenderer_File_NonExistent(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/test", nil)

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
	r := httptest.NewRequest("GET", "/test", nil)
	r.Header.Set(HeaderRange, "bytes=5-10")

	if err := R.File(w, r, filePath); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if w.Code != http.StatusPartialContent {
		t.Errorf("expected status 206, got %d", w.Code)
	}

	if body := w.Body.String(); body != "56789A" {
		t.Errorf("expected range body 56789A, got %s", body)
	}
}

func TestRenderer_NoContent(t *testing.T) {
	w := httptest.NewRecorder()

	if err := R.NoContent(w); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status %d, got %d", http.StatusNoContent, w.Code)
	}

	if body := w.Body.String(); body != "" {
		t.Errorf("expected empty body, got %s", body)
	}

	// 204 responses should not have Content-Type or Content-Length headers
	if ct := w.Header().Get(HeaderContentType); ct != "" {
		t.Errorf("expected no Content-Type header, got %s", ct)
	}
}

func TestRenderer_NotModified(t *testing.T) {
	w := httptest.NewRecorder()

	if err := R.NotModified(w); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if w.Code != http.StatusNotModified {
		t.Errorf("expected status %d, got %d", http.StatusNotModified, w.Code)
	}

	if body := w.Body.String(); body != "" {
		t.Errorf("expected empty body, got %s", body)
	}

	// 304 responses should not have Content-Type or Content-Length headers
	if ct := w.Header().Get(HeaderContentType); ct != "" {
		t.Errorf("expected no Content-Type header, got %s", ct)
	}
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

			if w.Code != tt.statusCode {
				t.Errorf("expected status %d, got %d", tt.statusCode, w.Code)
			}

			if location := w.Header().Get("Location"); location != tt.url {
				t.Errorf("expected Location header %s, got %s", tt.url, location)
			}

			body := w.Body.String()
			if body == "" {
				t.Error("expected redirect to have body content")
			}

			if !strings.Contains(body, tt.url) {
				t.Errorf("expected body to contain redirect URL %s, got %s", tt.url, body)
			}
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

	if location := w.Header().Get("Location"); location != absoluteURL {
		t.Errorf("expected Location header %s, got %s", absoluteURL, location)
	}
}

func TestRenderer_Redirect_WithQuery(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/original?param=value", nil)

	redirectURL := "/login?next=/original"

	if err := R.Redirect(w, r, redirectURL, http.StatusFound); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if location := w.Header().Get("Location"); location != redirectURL {
		t.Errorf("expected Location header %s, got %s", redirectURL, location)
	}
}
