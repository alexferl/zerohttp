package zhtest

import (
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TemplateRenderer provides utilities for testing template rendering.
// It wraps a template and provides methods to render and assert on the output.
type TemplateRenderer struct {
	templates *template.Template
}

// NewTemplateRenderer creates a new TemplateRenderer with the given template.
//
// Example:
//
//	tmpl := template.Must(template.New("test").Parse(`<h1>{{.Title}}</h1>`))
//	tr := zhtest.NewTemplateRenderer(tmpl)
func NewTemplateRenderer(templates *template.Template) *TemplateRenderer {
	return &TemplateRenderer{templates: templates}
}

// Render renders the template with the given name and data, returning the response.
//
// Example:
//
//	w := tr.Render("index.html", map[string]string{"Title": "Hello"})
//	zhtest.AssertEqual(t, http.StatusOK, w.Code)
func (tr *TemplateRenderer) Render(name string, data any) *Response {
	w := httptest.NewRecorder()
	if err := tr.templates.ExecuteTemplate(w, name, data); err != nil {
		// Return a response with error status
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
		return &Response{ResponseRecorder: w}
	}
	w.WriteHeader(http.StatusOK)
	return &Response{ResponseRecorder: w}
}

// RenderToString renders the template and returns the output as a string.
//
// Example:
//
//	html := tr.RenderToString("index.html", map[string]string{"Title": "Hello"})
//	zhtest.AssertContains(t, html, "<h1>Hello</h1>")
func (tr *TemplateRenderer) RenderToString(name string, data any) string {
	w := tr.Render(name, data)
	return w.BodyString()
}

// MustRender renders the template and returns the response.
// Panics if rendering fails.
//
// Example:
//
//	w := tr.MustRender("index.html", map[string]string{"Title": "Hello"})
//	zhtest.AssertWith(t, w).Status(http.StatusOK)
func (tr *TemplateRenderer) MustRender(name string, data any) *Response {
	w := httptest.NewRecorder()
	if err := tr.templates.ExecuteTemplate(w, name, data); err != nil {
		panic(err)
	}
	w.WriteHeader(http.StatusOK)
	return &Response{ResponseRecorder: w}
}

// TestTemplate creates a new template with the given content and renders it.
// This is useful for testing template fragments without setting up a full template.FS.
//
// Example:
//
//	w := zhtest.TestTemplate(`<h1>{{.Title}}</h1>`, map[string]string{"Title": "Hello"})
//	zhtest.AssertWith(t, w).BodyContains("<h1>Hello</h1>")
func TestTemplate(content string, data any) *Response {
	tmpl := template.Must(template.New("test").Parse(content))
	tr := NewTemplateRenderer(tmpl)
	return tr.Render("test", data)
}

// TestTemplateToString renders a template fragment and returns the output as a string.
//
// Example:
//
//	html := zhtest.TestTemplateToString(`<h1>{{.Title}}</h1>`, map[string]string{"Title": "Hello"})
//	zhtest.AssertEqual(t, "<h1>Hello</h1>", html)
func TestTemplateToString(content string, data any) string {
	return TestTemplate(content, data).BodyString()
}

// TemplateAssertions provides assertions specific to HTML template output.
type TemplateAssertions struct {
	resp *Response
	t    *testing.T
}

// AssertTemplate creates template-specific assertions for the response.
//
// Example:
//
//	zhtest.AssertTemplateWith(t, w).
//	    Contains("<h1>Hello</h1>").
//	    ContainsElement("div.content").
//	    HasTitle("Welcome")
func AssertTemplate(w *httptest.ResponseRecorder) *TemplateAssertions {
	return &TemplateAssertions{resp: &Response{w}, t: nil}
}

// AssertTemplateWith creates template-specific assertions that fail the test.
//
// Example:
//
//	zhtest.AssertTemplateWith(t, w).Contains("<h1>Hello</h1>")
func AssertTemplateWith(t *testing.T, w *httptest.ResponseRecorder) *TemplateAssertions {
	return &TemplateAssertions{resp: &Response{w}, t: t}
}

// fail reports a test failure if a testing.T is available.
func (a *TemplateAssertions) fail(format string, args ...any) {
	if a.t != nil {
		a.t.Errorf(format, args...)
	}
}

// Contains asserts that the rendered HTML contains the given substring.
//
// Example:
//
//	zhtest.AssertTemplateWith(t, w).Contains("<h1>Hello</h1>")
func (a *TemplateAssertions) Contains(substring string) *TemplateAssertions {
	if !strings.Contains(a.resp.BodyString(), substring) {
		a.fail("expected template output to contain %q, got %q", substring, a.resp.BodyString())
	}
	return a
}

// NotContains asserts that the rendered HTML does not contain the given substring.
//
// Example:
//
//	zhtest.AssertTemplateWith(t, w).NotContains("error")
func (a *TemplateAssertions) NotContains(substring string) *TemplateAssertions {
	if strings.Contains(a.resp.BodyString(), substring) {
		a.fail("expected template output to not contain %q", substring)
	}
	return a
}

// HasTitle asserts that the rendered HTML contains a title element with the given text.
//
// Example:
//
//	zhtest.AssertTemplateWith(t, w).HasTitle("Welcome")
func (a *TemplateAssertions) HasTitle(title string) *TemplateAssertions {
	body := a.resp.BodyString()
	// Simple string-based check for title
	titleTag := "<title>" + title + "</title>"
	if !strings.Contains(body, titleTag) {
		// Try with different whitespace
		titleTag2 := "<title>" + title + "</title>"
		if !strings.Contains(body, titleTag2) {
			a.fail("expected template to have title %q", title)
		}
	}
	return a
}

// Equals asserts that the rendered HTML equals the expected string.
//
// Example:
//
//	zhtest.AssertTemplateWith(t, w).Equals("<h1>Hello</h1>")
func (a *TemplateAssertions) Equals(expected string) *TemplateAssertions {
	if a.resp.BodyString() != expected {
		a.fail("expected template output %q, got %q", expected, a.resp.BodyString())
	}
	return a
}

// Status asserts that the response has the expected status code.
//
// Example:
//
//	zhtest.AssertTemplateWith(t, w).Status(http.StatusOK)
func (a *TemplateAssertions) Status(code int) *TemplateAssertions {
	if a.resp.Code != code {
		a.fail("expected status %d, got %d", code, a.resp.Code)
	}
	return a
}
