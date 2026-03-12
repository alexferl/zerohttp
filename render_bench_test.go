package zerohttp

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alexferl/zerohttp/internal/problem"
)

// BenchmarkRenderer_JSON_Baseline compares JSON() method vs stdlib json.NewEncoder directly
func BenchmarkRenderer_JSON_Baseline(b *testing.B) {
	data := M{"message": "hello", "status": "ok"}

	b.Run("Stdlib_NewEncoder", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			w := httptest.NewRecorder()
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(data)
		}
	})

	b.Run("Renderer_JSON", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			w := httptest.NewRecorder()
			_ = R.JSON(w, http.StatusOK, data)
		}
	})

	b.Run("Stdlib_Marshal", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			w := httptest.NewRecorder()
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			bytes, _ := json.Marshal(data)
			_, _ = w.Write(bytes)
		}
	})
}

// BenchmarkRenderer_JSON_PayloadSizes measures allocation patterns for different payload sizes
func BenchmarkRenderer_JSON_PayloadSizes(b *testing.B) {
	sizes := []struct {
		name string
		data any
	}{
		{
			name: "Tiny_50B",
			data: M{"msg": "hi"},
		},
		{
			name: "Small_500B",
			data: M{
				"id":      "12345",
				"name":    "Test User",
				"email":   "test@example.com",
				"status":  "active",
				"message": strings.Repeat("a", 400),
			},
		},
		{
			name: "Medium_5KB",
			data: func() any {
				items := make([]M, 50)
				for i := range 50 {
					items[i] = M{
						"id":    i,
						"name":  fmt.Sprintf("Item %d", i),
						"value": fmt.Sprintf("Value %d with some padding", i),
					}
				}
				return M{"items": items, "count": 50}
			}(),
		},
		{
			name: "Medium_50KB",
			data: func() any {
				items := make([]M, 500)
				for i := range 500 {
					items[i] = M{
						"id":    i,
						"name":  fmt.Sprintf("Item %d", i),
						"value": fmt.Sprintf("Value %d with some padding text here", i),
						"data":  strings.Repeat("x", 50),
					}
				}
				return M{"items": items, "count": 500}
			}(),
		},
		{
			name: "Large_100KB",
			data: func() any {
				items := make([]M, 1000)
				for i := range 1000 {
					items[i] = M{
						"id":    i,
						"name":  fmt.Sprintf("Item %d", i),
						"value": fmt.Sprintf("Value %d", i),
						"data":  strings.Repeat("y", 80),
					}
				}
				return M{"items": items, "count": 1000}
			}(),
		},
	}

	for _, s := range sizes {
		b.Run(s.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				w := httptest.NewRecorder()
				_ = R.JSON(w, http.StatusOK, s.data)
			}
		})
	}
}

// BenchmarkRenderer_JSON_DataTypes compares different Go types for JSON rendering
func BenchmarkRenderer_JSON_DataTypes(b *testing.B) {
	type User struct {
		ID    int    `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	types := []struct {
		name string
		data any
	}{
		{
			name: "Map",
			data: M{"id": 1, "name": "test", "email": "test@example.com"},
		},
		{
			name: "Struct",
			data: User{ID: 1, Name: "test", Email: "test@example.com"},
		},
		{
			name: "Slice",
			data: []string{"a", "b", "c", "d", "e"},
		},
		{
			name: "String",
			data: "hello world",
		},
		{
			name: "Number",
			data: 42,
		},
		{
			name: "Bool",
			data: true,
		},
		{
			name: "Nil",
			data: nil,
		},
	}

	for _, t := range types {
		b.Run(t.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				w := httptest.NewRecorder()
				_ = R.JSON(w, http.StatusOK, t.data)
			}
		})
	}
}

// BenchmarkRenderer_ProblemDetail measures problem detail rendering overhead
func BenchmarkRenderer_ProblemDetail(b *testing.B) {
	b.Run("Simple", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			w := httptest.NewRecorder()
			problem := NewProblemDetail(http.StatusNotFound, "Resource not found")
			_ = R.ProblemDetail(w, problem)
		}
	})

	b.Run("WithExtensions", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			w := httptest.NewRecorder()
			problem := NewProblemDetail(http.StatusBadRequest, "Validation failed")
			problem.Set("field", "email")
			problem.Set("constraint", "required")
			_ = R.ProblemDetail(w, problem)
		}
	})

	b.Run("ValidationErrors", func(b *testing.B) {
		errors := []problem.ValidationError{
			{Detail: "Name is required", Field: "name"},
			{Detail: "Invalid email format", Field: "email"},
			{Detail: "Age must be positive", Field: "age"},
		}

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			w := httptest.NewRecorder()
			problem := problem.NewValidationDetail("Validation failed", errors)
			_ = R.ProblemDetail(w, problem)
		}
	})

	b.Run("StatusCodes", func(b *testing.B) {
		codes := []int{
			http.StatusBadRequest,
			http.StatusUnauthorized,
			http.StatusForbidden,
			http.StatusNotFound,
			http.StatusInternalServerError,
		}

		for _, code := range codes {
			b.Run(fmt.Sprintf("Status%d", code), func(b *testing.B) {
				b.ReportAllocs()
				b.ResetTimer()

				for b.Loop() {
					w := httptest.NewRecorder()
					problem := NewProblemDetail(code, "Test error message")
					_ = R.ProblemDetail(w, problem)
				}
			})
		}
	})
}

// BenchmarkRenderer_Text compares Text rendering as a baseline
func BenchmarkRenderer_Text(b *testing.B) {
	messages := []struct {
		name string
		text string
	}{
		{"Short", "OK"},
		{"Medium", "Hello, this is a medium length message"},
		{"Large_1KB", strings.Repeat("x", 1024)},
	}

	for _, m := range messages {
		b.Run(m.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				w := httptest.NewRecorder()
				_ = R.Text(w, http.StatusOK, m.text)
			}
		})
	}
}

// BenchmarkRenderer_HTML compares HTML rendering
func BenchmarkRenderer_HTML(b *testing.B) {
	html := []struct {
		name string
		data string
	}{
		{"Simple", "<h1>Hello</h1>"},
		{"Medium", "<html><body><h1>Title</h1><p>Some content here</p></body></html>"},
		{"Large_1KB", "<html><body>" + strings.Repeat("<p>Paragraph</p>", 50) + "</body></html>"},
	}

	for _, h := range html {
		b.Run(h.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				w := httptest.NewRecorder()
				_ = R.HTML(w, http.StatusOK, h.data)
			}
		})
	}
}

// BenchmarkRenderer_Template measures template rendering performance
func BenchmarkRenderer_Template(b *testing.B) {
	simpleTmpl := template.Must(template.New("simple").Parse("<h1>{{.Title}}</h1>"))
	complexTmpl := template.Must(template.New("complex").Parse(`
<html>
<head><title>{{.Title}}</title></head>
<body>
<h1>{{.Title}}</h1>
<p>{{.Description}}</p>
<ul>
{{range .Items}}<li>{{.}}</li>{{end}}
</ul>
</body>
</html>`))

	b.Run("Simple", func(b *testing.B) {
		data := map[string]string{"Title": "Test Page"}

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			w := httptest.NewRecorder()
			_ = R.Template(w, http.StatusOK, simpleTmpl, "simple", data)
		}
	})

	b.Run("Complex", func(b *testing.B) {
		data := map[string]any{
			"Title":       "Complex Page",
			"Description": "This is a more complex template",
			"Items":       []string{"Item 1", "Item 2", "Item 3", "Item 4", "Item 5"},
		}

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			w := httptest.NewRecorder()
			_ = R.Template(w, http.StatusOK, complexTmpl, "complex", data)
		}
	})
}

// BenchmarkRenderer_Blob measures Blob rendering as a binary baseline
func BenchmarkRenderer_Blob(b *testing.B) {
	sizes := []int{100, 1024, 10240, 102400}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
			data := make([]byte, size)

			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				w := httptest.NewRecorder()
				_ = R.Blob(w, http.StatusOK, "application/octet-stream", data)
			}
		})
	}
}

// BenchmarkRenderer_NoContent measures NoContent rendering
func BenchmarkRenderer_NoContent(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		w := httptest.NewRecorder()
		_ = R.NoContent(w)
	}
}

// BenchmarkRenderer_NotModified measures NotModified rendering
func BenchmarkRenderer_NotModified(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		w := httptest.NewRecorder()
		_ = R.NotModified(w)
	}
}

// BenchmarkRenderer_Redirect measures redirect rendering
func BenchmarkRenderer_Redirect(b *testing.B) {
	r := httptest.NewRequest(http.MethodGet, "/original", nil)

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		w := httptest.NewRecorder()
		_ = R.Redirect(w, r, "/new-location", http.StatusFound)
	}
}

// BenchmarkRenderer_JSON_NestedStructs measures deeply nested structures
func BenchmarkRenderer_JSON_NestedStructs(b *testing.B) {
	type Address struct {
		Street string `json:"street"`
		City   string `json:"city"`
	}

	type Person struct {
		Name    string  `json:"name"`
		Address Address `json:"address"`
	}

	type Company struct {
		CEO     Person   `json:"ceo"`
		CTO     Person   `json:"cto"`
		Offices []string `json:"offices"`
	}

	nested := Company{
		CEO: Person{
			Name: "John CEO",
			Address: Address{
				Street: "123 Main St",
				City:   "New York",
			},
		},
		CTO: Person{
			Name: "Jane CTO",
			Address: Address{
				Street: "456 Tech Ave",
				City:   "San Francisco",
			},
		},
		Offices: []string{"NYC", "SF", "London"},
	}

	b.Run("NestedStruct", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			w := httptest.NewRecorder()
			_ = R.JSON(w, http.StatusOK, nested)
		}
	})

	// Compare with map
	nestedMap := M{
		"ceo": M{
			"name": "John CEO",
			"address": M{
				"street": "123 Main St",
				"city":   "New York",
			},
		},
		"cto": M{
			"name": "Jane CTO",
			"address": M{
				"street": "456 Tech Ave",
				"city":   "San Francisco",
			},
		},
		"offices": []string{"NYC", "SF", "London"},
	}

	b.Run("NestedMap", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			w := httptest.NewRecorder()
			_ = R.JSON(w, http.StatusOK, nestedMap)
		}
	})
}

// BenchmarkRenderer_JSON_Concurrent measures concurrent JSON rendering
func BenchmarkRenderer_JSON_Concurrent(b *testing.B) {
	data := M{"message": "hello", "status": "ok", "count": 42}

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			w := httptest.NewRecorder()
			_ = R.JSON(w, http.StatusOK, data)
		}
	})
}

// BenchmarkRenderer_JSON_StatusCodes measures different status codes
func BenchmarkRenderer_JSON_StatusCodes(b *testing.B) {
	data := M{"result": "success"}

	codes := []int{
		http.StatusOK,
		http.StatusCreated,
		http.StatusAccepted,
		http.StatusNoContent,
		http.StatusBadRequest,
		http.StatusUnauthorized,
		http.StatusNotFound,
		http.StatusInternalServerError,
	}

	for _, code := range codes {
		b.Run(fmt.Sprintf("Status%d", code), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				w := httptest.NewRecorder()
				_ = R.JSON(w, code, data)
			}
		})
	}
}
