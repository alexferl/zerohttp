package zerohttp

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
)

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

func TestProblemDetail_Set_ExtensionsInitialization(t *testing.T) {
	p := &ProblemDetail{
		Title:  "Test Error",
		Status: 400,
	}

	// Verify Extensions starts as nil
	if p.Extensions != nil {
		t.Fatal("Expected Extensions to be nil initially")
	}

	// Call Set - this should initialize Extensions
	result := p.Set("key", "value")

	// Verify Extensions was initialized
	if p.Extensions == nil {
		t.Fatal("Expected Extensions to be initialized after Set")
	}

	// Verify the value was set correctly
	if val, ok := p.Extensions["key"]; !ok || val != "value" {
		t.Errorf("Expected Extensions to contain 'key' with value 'value', got %v", p.Extensions)
	}

	// Verify method chaining works
	if result != p {
		t.Error("Expected Set to return same ProblemDetail instance for chaining")
	}

	// Test multiple Sets to ensure Extensions remains functional
	p.Set("another", 123).Set("third", true)

	if len(p.Extensions) != 3 {
		t.Errorf("Expected Extensions to contain 3 items, got %d", len(p.Extensions))
	}

	if p.Extensions["another"] != 123 || p.Extensions["third"] != true {
		t.Error("Expected all extension values to be preserved")
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
