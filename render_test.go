package zerohttp

import (
	"bytes"
	"encoding/json"
	"errors"
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

func TestNewProblemDetail(t *testing.T) {
	problem := NewProblemDetail(404, "Not found")

	if problem.Type != "" {
		t.Errorf("expected empty Type, got %s", problem.Type)
	}

	if problem.Title != "Not Found" {
		t.Errorf("expected title 'Not Found', got %s", problem.Title)
	}

	if problem.Status != 404 {
		t.Errorf("expected status 404, got %d", problem.Status)
	}

	if problem.Detail != "Not found" {
		t.Errorf("expected detail 'Not found', got %s", problem.Detail)
	}

	if problem.Extensions == nil {
		t.Error("expected Extensions to be initialized")
	}
}

func TestProblemDetail_Set(t *testing.T) {
	problem := NewProblemDetail(400, "Bad request")

	result := problem.Set("field", "email").Set("code", "INVALID")

	if result != problem {
		t.Error("expected Set to return same problem for chaining")
	}

	if len(problem.Extensions) != 2 {
		t.Errorf("expected 2 extensions, got %d", len(problem.Extensions))
	}

	if problem.Extensions["field"] != "email" {
		t.Error("expected field extension to be set")
	}
}

func TestProblemDetail_MarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		problem  *ProblemDetail
		expected map[string]any
	}{
		{
			"all fields",
			&ProblemDetail{
				Type: "https://example.com/error", Title: "Error", Status: 400,
				Detail: "Bad request", Instance: "/test",
				Extensions: map[string]any{"code": "ERR001"},
			},
			map[string]any{
				"type": "https://example.com/error", "title": "Error", "status": float64(400),
				"detail": "Bad request", "instance": "/test", "code": "ERR001",
			},
		},
		{
			"required only",
			&ProblemDetail{Title: "Error", Status: 500},
			map[string]any{"title": "Error", "status": float64(500)},
		},
		{
			"with extensions",
			&ProblemDetail{
				Title: "Bad Request", Status: 400,
				Extensions: map[string]any{"errors": []string{"field required"}},
			},
			map[string]any{
				"title": "Bad Request", "status": float64(400),
				"errors": []string{"field required"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := tt.problem.MarshalJSON()
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			var result map[string]any
			if err := json.Unmarshal(data, &result); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			for key, expected := range tt.expected {
				if actual, exists := result[key]; !exists {
					t.Errorf("expected field %s to exist", key)
				} else if !equalValues(actual, expected) {
					t.Errorf("expected %s to be %v, got %v", key, expected, actual)
				}
			}
		})
	}
}

func TestRenderer_ProblemDetail(t *testing.T) {
	problem := NewProblemDetail(404, "Not found")
	problem.Instance = "/users/123"
	problem.Set("timestamp", "2023-01-01T00:00:00Z")

	w := httptest.NewRecorder()

	if err := R.ProblemDetail(w, problem); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if w.Code != 404 {
		t.Errorf("expected status 404, got %d", w.Code)
	}

	if ct := w.Header().Get(HeaderContentType); ct != MIMEApplicationProblem {
		t.Errorf("expected Content-Type %s, got %s", MIMEApplicationProblem, ct)
	}

	var result map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	expected := map[string]any{
		"title": "Not Found", "status": float64(404), "detail": "Not found",
		"instance": "/users/123", "timestamp": "2023-01-01T00:00:00Z",
	}

	for key, expectedValue := range expected {
		if actual, exists := result[key]; !exists {
			t.Errorf("expected field %s to exist", key)
		} else if actual != expectedValue {
			t.Errorf("expected %s to be %v, got %v", key, expectedValue, actual)
		}
	}
}

func TestNewValidationProblemDetail(t *testing.T) {
	errs := []ValidationError{
		{Detail: "required", Pointer: "#/name"},
		{Detail: "invalid", Field: "email"},
	}

	problem := NewValidationProblemDetail("Validation failed", errs)

	if problem.Status != 422 {
		t.Errorf("expected status 422, got %d", problem.Status)
	}

	if problem.Title != "Unprocessable Entity" {
		t.Errorf("expected title 'Unprocessable Entity', got %s", problem.Title)
	}

	errorsExt, exists := problem.Extensions["errors"]
	if !exists {
		t.Fatal("expected errors extension to exist")
	}

	validationErrors, ok := errorsExt.([]ValidationError)
	if !ok || len(validationErrors) != 2 {
		t.Fatalf("expected 2 ValidationError items, got %T with len %d", errorsExt, len(validationErrors))
	}

	if validationErrors[0].Detail != "required" {
		t.Errorf("expected first error detail 'required', got %s", validationErrors[0].Detail)
	}
}

func TestNewValidationProblemDetail_Custom(t *testing.T) {
	type CustomError struct {
		Code string `json:"code"`
		Msg  string `json:"msg"`
	}

	errs := []CustomError{
		{Code: "ERR001", Msg: "Invalid input"},
	}

	problem := NewValidationProblemDetail("Custom validation", errs)

	errorsExt := problem.Extensions["errors"].([]CustomError)
	if len(errorsExt) != 1 || errorsExt[0].Code != "ERR001" {
		t.Error("expected custom error to be preserved")
	}
}

// Helper function for comparing values in tests
func equalValues(a, b any) bool {
	if slice, ok := b.([]string); ok {
		aSlice, ok := a.([]any)
		if !ok || len(aSlice) != len(slice) {
			return false
		}
		for i, v := range slice {
			if aSlice[i] != v {
				return false
			}
		}
		return true
	}
	return a == b
}
