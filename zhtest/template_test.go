package zhtest

import (
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alexferl/zerohttp/httpx"
)

func TestTemplateRenderer_Render(t *testing.T) {
	tmplContent := `
{{define "test.html"}}<html><head><title>{{.Title}}</title></head><body><h1>{{.Title}}</h1></body></html>{{end}}
{{define "partial.html"}}<div>{{.Content}}</div>{{end}}
`
	tmpl := template.Must(template.New("test").Parse(tmplContent))
	tr := NewTemplateRenderer(tmpl)

	t.Run("renders template successfully", func(t *testing.T) {
		w := tr.Render("test.html", map[string]string{"Title": "Test Page"})

		AssertEqual(t, http.StatusOK, w.Code)
		AssertTrue(t, strings.Contains(w.BodyString(), "<h1>Test Page</h1>"))
	})

	t.Run("returns error on missing template", func(t *testing.T) {
		w := tr.Render("missing.html", map[string]string{"Title": "Test"})

		AssertEqual(t, http.StatusInternalServerError, w.Code)
	})
}

func TestTemplateRenderer_RenderToString(t *testing.T) {
	tmpl := template.Must(template.New("test").Parse(`<h1>{{.Title}}</h1>`))
	tr := NewTemplateRenderer(tmpl)

	html := tr.RenderToString("test", map[string]string{"Title": "Hello"})

	AssertEqual(t, "<h1>Hello</h1>", html)
}

func TestTemplateRenderer_MustRender(t *testing.T) {
	tmpl := template.Must(template.New("test").Parse(`<h1>{{.Title}}</h1>`))
	tr := NewTemplateRenderer(tmpl)

	w := tr.MustRender("test", map[string]string{"Title": "Hello"})

	AssertEqual(t, http.StatusOK, w.Code)
	AssertEqual(t, "<h1>Hello</h1>", w.BodyString())
}

func TestTestTemplate(t *testing.T) {
	w := TestTemplate(`<h1>{{.Title}}</h1>`, map[string]string{"Title": "Hello"})

	AssertEqual(t, http.StatusOK, w.Code)
	AssertEqual(t, "<h1>Hello</h1>", w.BodyString())
}

func TestTestTemplateToString(t *testing.T) {
	html := TestTemplateToString(`<h1>{{.Title}}</h1>`, map[string]string{"Title": "Hello"})

	AssertEqual(t, "<h1>Hello</h1>", html)
}

func TestAssertTemplate_Contains(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(httpx.HeaderContentType, httpx.MIMETextHTML)
		_, err := w.Write([]byte(`<html><body><h1>Welcome</h1><p>Hello, World!</p></body></html>`))
		AssertNoError(t, err)
	})
	req := NewRequest(http.MethodGet, "/").Build()
	w := Serve(handler, req)

	result := AssertTemplate(w).Contains("<h1>Welcome</h1>")
	AssertNotNil(t, result)
}

func TestAssertTemplate_NotContains(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(httpx.HeaderContentType, httpx.MIMETextHTML)
		_, err := w.Write([]byte(`<html><body><h1>Welcome</h1></body></html>`))
		AssertNoError(t, err)
	})
	req := NewRequest(http.MethodGet, "/").Build()
	w := Serve(handler, req)

	result := AssertTemplate(w).NotContains("error")
	AssertNotNil(t, result)
}

func TestAssertTemplate_HasTitle(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(httpx.HeaderContentType, httpx.MIMETextHTML)
		_, err := w.Write([]byte(`<html><head><title>My Page</title></head><body><h1>Welcome</h1></body></html>`))
		AssertNoError(t, err)
	})
	req := NewRequest(http.MethodGet, "/").Build()
	w := Serve(handler, req)

	result := AssertTemplate(w).HasTitle("My Page")
	AssertNotNil(t, result)
}

func TestAssertTemplate_Equals(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(httpx.HeaderContentType, httpx.MIMETextHTML)
		_, err := w.Write([]byte(`<h1>Hello</h1>`))
		AssertNoError(t, err)
	})
	req := NewRequest(http.MethodGet, "/").Build()
	w := Serve(handler, req)

	result := AssertTemplate(w).Equals("<h1>Hello</h1>")
	AssertNotNil(t, result)
}

func TestAssertTemplate_Status(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(httpx.HeaderContentType, httpx.MIMETextHTML)
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`<h1>Hello</h1>`))
		AssertNoError(t, err)
	})
	req := NewRequest(http.MethodGet, "/").Build()
	w := Serve(handler, req)

	result := AssertTemplate(w).Status(http.StatusOK)
	AssertNotNil(t, result)
}

func TestAssertTemplate_Chaining(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(httpx.HeaderContentType, httpx.MIMETextHTML)
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`<html><head><title>My Page</title></head><body><h1>Welcome</h1></body></html>`))
		AssertNoError(t, err)
	})
	req := NewRequest(http.MethodGet, "/").Build()
	w := Serve(handler, req)

	result := AssertTemplate(w).
		Status(http.StatusOK).
		Contains("<h1>Welcome</h1>").
		NotContains("error").
		HasTitle("My Page")

	AssertNotNil(t, result)
}

func TestAssertTemplateWith(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(httpx.HeaderContentType, httpx.MIMETextHTML)
		_, err := w.Write([]byte(`<h1>Hello</h1>`))
		AssertNoError(t, err)
	})
	req := NewRequest(http.MethodGet, "/").Build()
	w := Serve(handler, req)

	// This will use the actual testing.T - just verify it doesn't panic
	result := AssertTemplateWith(t, w).Contains("<h1>Hello</h1>")
	AssertNotNil(t, result)
}

// Test template assertion failure paths
func TestAssertTemplate_FailurePaths(t *testing.T) {
	t.Run("Contains failure", func(t *testing.T) {
		w := httptest.NewRecorder()
		_, err := w.Write([]byte("hello"))
		AssertNoError(t, err)

		result := AssertTemplate(w).Contains("world")
		AssertNotNil(t, result)
	})

	t.Run("NotContains failure", func(t *testing.T) {
		w := httptest.NewRecorder()
		_, err := w.Write([]byte("hello world"))
		AssertNoError(t, err)

		result := AssertTemplate(w).NotContains("hello")
		AssertNotNil(t, result)
	})

	t.Run("HasTitle failure - wrong title", func(t *testing.T) {
		w := httptest.NewRecorder()
		_, err := w.Write([]byte(`<html><head><title>Wrong</title></head></html>`))
		AssertNoError(t, err)

		result := AssertTemplate(w).HasTitle("Right")
		AssertNotNil(t, result)
	})

	t.Run("HasTitle failure - no title", func(t *testing.T) {
		w := httptest.NewRecorder()
		_, err := w.Write([]byte(`<html><body>no title</body></html>`))
		AssertNoError(t, err)

		result := AssertTemplate(w).HasTitle("Any")
		AssertNotNil(t, result)
	})

	t.Run("Equals failure", func(t *testing.T) {
		w := httptest.NewRecorder()
		_, err := w.Write([]byte("hello"))
		AssertNoError(t, err)

		result := AssertTemplate(w).Equals("world")
		AssertNotNil(t, result)
	})

	t.Run("Status failure", func(t *testing.T) {
		w := httptest.NewRecorder()
		w.WriteHeader(http.StatusNotFound)

		result := AssertTemplate(w).Status(http.StatusOK)
		AssertNotNil(t, result)
	})
}
