package zhtest

import (
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		if !strings.Contains(w.BodyString(), "<h1>Test Page</h1>") {
			t.Errorf("expected body to contain '<h1>Test Page</h1>', got %s", w.BodyString())
		}
	})

	t.Run("returns error on missing template", func(t *testing.T) {
		w := tr.Render("missing.html", map[string]string{"Title": "Test"})

		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected status 500, got %d", w.Code)
		}
	})
}

func TestTemplateRenderer_RenderToString(t *testing.T) {
	tmpl := template.Must(template.New("test").Parse(`<h1>{{.Title}}</h1>`))
	tr := NewTemplateRenderer(tmpl)

	html := tr.RenderToString("test", map[string]string{"Title": "Hello"})

	if html != "<h1>Hello</h1>" {
		t.Errorf("expected '<h1>Hello</h1>', got %s", html)
	}
}

func TestTemplateRenderer_MustRender(t *testing.T) {
	tmpl := template.Must(template.New("test").Parse(`<h1>{{.Title}}</h1>`))
	tr := NewTemplateRenderer(tmpl)

	w := tr.MustRender("test", map[string]string{"Title": "Hello"})

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	if w.BodyString() != "<h1>Hello</h1>" {
		t.Errorf("expected '<h1>Hello</h1>', got %s", w.BodyString())
	}
}

func TestTestTemplate(t *testing.T) {
	w := TestTemplate(`<h1>{{.Title}}</h1>`, map[string]string{"Title": "Hello"})

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	if w.BodyString() != "<h1>Hello</h1>" {
		t.Errorf("expected '<h1>Hello</h1>', got %s", w.BodyString())
	}
}

func TestTestTemplateToString(t *testing.T) {
	html := TestTemplateToString(`<h1>{{.Title}}</h1>`, map[string]string{"Title": "Hello"})

	if html != "<h1>Hello</h1>" {
		t.Errorf("expected '<h1>Hello</h1>', got %s", html)
	}
}

func TestAssertTemplate_Contains(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		if _, err := w.Write([]byte(`<html><body><h1>Welcome</h1><p>Hello, World!</p></body></html>`)); err != nil {
			t.Errorf("failed to write: %v", err)
		}
	})
	req := NewRequest(http.MethodGet, "/").Build()
	w := Serve(handler, req)

	result := AssertTemplate(w).Contains("<h1>Welcome</h1>")
	if result == nil {
		t.Error("expected result to not be nil")
	}
}

func TestAssertTemplate_NotContains(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		if _, err := w.Write([]byte(`<html><body><h1>Welcome</h1></body></html>`)); err != nil {
			t.Errorf("failed to write: %v", err)
		}
	})
	req := NewRequest(http.MethodGet, "/").Build()
	w := Serve(handler, req)

	result := AssertTemplate(w).NotContains("error")
	if result == nil {
		t.Error("expected result to not be nil")
	}
}

func TestAssertTemplate_HasTitle(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		if _, err := w.Write([]byte(`<html><head><title>My Page</title></head><body><h1>Welcome</h1></body></html>`)); err != nil {
			t.Errorf("failed to write: %v", err)
		}
	})
	req := NewRequest(http.MethodGet, "/").Build()
	w := Serve(handler, req)

	result := AssertTemplate(w).HasTitle("My Page")
	if result == nil {
		t.Error("expected result to not be nil")
	}
}

func TestAssertTemplate_Equals(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		if _, err := w.Write([]byte(`<h1>Hello</h1>`)); err != nil {
			t.Errorf("failed to write: %v", err)
		}
	})
	req := NewRequest(http.MethodGet, "/").Build()
	w := Serve(handler, req)

	result := AssertTemplate(w).Equals("<h1>Hello</h1>")
	if result == nil {
		t.Error("expected result to not be nil")
	}
}

func TestAssertTemplate_Status(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`<h1>Hello</h1>`)); err != nil {
			t.Errorf("failed to write: %v", err)
		}
	})
	req := NewRequest(http.MethodGet, "/").Build()
	w := Serve(handler, req)

	result := AssertTemplate(w).Status(http.StatusOK)
	if result == nil {
		t.Error("expected result to not be nil")
	}
}

func TestAssertTemplate_Chaining(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`<html><head><title>My Page</title></head><body><h1>Welcome</h1></body></html>`)); err != nil {
			t.Errorf("failed to write: %v", err)
		}
	})
	req := NewRequest(http.MethodGet, "/").Build()
	w := Serve(handler, req)

	result := AssertTemplate(w).
		Status(http.StatusOK).
		Contains("<h1>Welcome</h1>").
		NotContains("error").
		HasTitle("My Page")

	if result == nil {
		t.Error("expected result to not be nil")
	}
}

func TestAssertTemplateWith(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		if _, err := w.Write([]byte(`<h1>Hello</h1>`)); err != nil {
			t.Errorf("failed to write: %v", err)
		}
	})
	req := NewRequest(http.MethodGet, "/").Build()
	w := Serve(handler, req)

	// This will use the actual testing.T - just verify it doesn't panic
	result := AssertTemplateWith(t, w).Contains("<h1>Hello</h1>")
	if result == nil {
		t.Error("expected result to not be nil")
	}
}

// Test template assertion failure paths
func TestAssertTemplate_FailurePaths(t *testing.T) {
	t.Run("Contains failure", func(t *testing.T) {
		w := httptest.NewRecorder()
		if _, err := w.Write([]byte("hello")); err != nil {
			t.Errorf("failed to write: %v", err)
		}

		result := AssertTemplate(w).Contains("world")
		if result == nil {
			t.Error("expected chain to continue after failure")
		}
	})

	t.Run("NotContains failure", func(t *testing.T) {
		w := httptest.NewRecorder()
		if _, err := w.Write([]byte("hello world")); err != nil {
			t.Errorf("failed to write: %v", err)
		}

		result := AssertTemplate(w).NotContains("hello")
		if result == nil {
			t.Error("expected chain to continue after failure")
		}
	})

	t.Run("HasTitle failure - wrong title", func(t *testing.T) {
		w := httptest.NewRecorder()
		if _, err := w.Write([]byte(`<html><head><title>Wrong</title></head></html>`)); err != nil {
			t.Errorf("failed to write: %v", err)
		}

		result := AssertTemplate(w).HasTitle("Right")
		if result == nil {
			t.Error("expected chain to continue after failure")
		}
	})

	t.Run("HasTitle failure - no title", func(t *testing.T) {
		w := httptest.NewRecorder()
		if _, err := w.Write([]byte(`<html><body>no title</body></html>`)); err != nil {
			t.Errorf("failed to write: %v", err)
		}

		result := AssertTemplate(w).HasTitle("Any")
		if result == nil {
			t.Error("expected chain to continue after failure")
		}
	})

	t.Run("Equals failure", func(t *testing.T) {
		w := httptest.NewRecorder()
		if _, err := w.Write([]byte("hello")); err != nil {
			t.Errorf("failed to write: %v", err)
		}

		result := AssertTemplate(w).Equals("world")
		if result == nil {
			t.Error("expected chain to continue after failure")
		}
	})

	t.Run("Status failure", func(t *testing.T) {
		w := httptest.NewRecorder()
		w.WriteHeader(http.StatusNotFound)

		result := AssertTemplate(w).Status(http.StatusOK)
		if result == nil {
			t.Error("expected chain to continue after failure")
		}
	})
}
