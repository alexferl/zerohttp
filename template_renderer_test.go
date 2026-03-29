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
	zhtest.AssertNotNil(t, tm)
}

func TestTemplateManager_Render(t *testing.T) {
	tm := NewTemplateManager(testTemplates, "testdata/templates/*.html")

	t.Run("renders template successfully", func(t *testing.T) {
		w := httptest.NewRecorder()
		data := map[string]string{"Title": "Test Page"}

		err := tm.Render(w, http.StatusOK, "test.html", data)
		zhtest.AssertNoError(t, err)

		zhtest.AssertWith(t, w).
			Status(http.StatusOK).
			Header(httpx.HeaderContentType, httpx.MIMETextHTMLCharset).
			BodyContains("Test Page")
	})

	t.Run("returns error for missing template", func(t *testing.T) {
		w := httptest.NewRecorder()

		err := tm.Render(w, http.StatusOK, "missing.html", nil)
		zhtest.AssertError(t, err)
	})

	t.Run("sets correct status code", func(t *testing.T) {
		w := httptest.NewRecorder()

		err := tm.Render(w, http.StatusCreated, "test.html", nil)
		zhtest.AssertNoError(t, err)

		zhtest.AssertWith(t, w).Status(http.StatusCreated)
	})
}
