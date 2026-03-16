package zerohttp

import (
	"embed"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alexferl/zerohttp/httpx"
	"github.com/alexferl/zerohttp/zhtest"
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

		zhtest.AssertWith(t, w).
			Status(http.StatusOK).
			Header(httpx.HeaderContentType, httpx.MIMETextHTMLCharset).
			BodyContains("Test Page")
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

		zhtest.AssertWith(t, w).Status(http.StatusCreated)
	})
}
