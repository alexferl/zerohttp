package zerohttp

import (
	"embed"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

//go:embed testdata/templates/*.html
var testTemplates embed.FS

func TestNewTemplateManager(t *testing.T) {
	tm := NewTemplateManager(testTemplates, "testdata/templates/*.html")
	if tm == nil {
		t.Fatal("expected TemplateRenderer to be created")
	}
}

func TestTemplateManager_Render(t *testing.T) {
	tm := NewTemplateManager(testTemplates, "testdata/templates/*.html")

	t.Run("renders template successfully", func(t *testing.T) {
		w := httptest.NewRecorder()
		data := map[string]string{"Title": "Test Page"}

		err := tm.Render(w, http.StatusOK, "test.html", data)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "Test Page") {
			t.Errorf("expected body to contain 'Test Page', got %s", w.Body.String())
		}
		if w.Header().Get("Content-Type") != "text/html; charset=utf-8" {
			t.Errorf("expected HTML content type, got %s", w.Header().Get("Content-Type"))
		}
	})

	t.Run("returns error for missing template", func(t *testing.T) {
		w := httptest.NewRecorder()

		err := tm.Render(w, http.StatusOK, "missing.html", nil)

		if err == nil {
			t.Fatal("expected error for missing template")
		}
	})

	t.Run("sets correct status code", func(t *testing.T) {
		w := httptest.NewRecorder()

		err := tm.Render(w, http.StatusCreated, "test.html", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if w.Code != http.StatusCreated {
			t.Errorf("expected status 201, got %d", w.Code)
		}
	})
}
