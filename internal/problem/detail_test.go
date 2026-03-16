package problem

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alexferl/zerohttp/httpx"
	"github.com/alexferl/zerohttp/zhtest"
)

func TestNewDetail(t *testing.T) {
	detail := NewDetail(http.StatusNotFound, "Not found")

	if detail.Type != "" {
		t.Errorf("expected empty Type, got %s", detail.Type)
	}

	if detail.Title != "Not Found" {
		t.Errorf("expected title 'Not Found', got %s", detail.Title)
	}

	if detail.Status != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", detail.Status)
	}

	if detail.Detail != "Not found" {
		t.Errorf("expected detail 'Not found', got %s", detail.Detail)
	}

	if detail.Extensions == nil {
		t.Error("expected Extensions to be initialized")
	}
}

func TestDetail_Set(t *testing.T) {
	detail := NewDetail(400, "Bad request")

	result := detail.Set("field", "email").Set("code", "INVALID")

	if result != detail {
		t.Error("expected Set to return same detail for chaining")
	}

	if len(detail.Extensions) != 2 {
		t.Errorf("expected 2 extensions, got %d", len(detail.Extensions))
	}

	if detail.Extensions["field"] != "email" {
		t.Error("expected field extension to be set")
	}
}

func TestDetail_MarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		detail   *Detail
		expected map[string]any
	}{
		{
			"all fields",
			&Detail{
				Type: "https://example.com/error", Title: "Error", Status: http.StatusBadRequest,
				Detail: "Bad request", Instance: "/test",
				Extensions: map[string]any{"code": "ERR001"},
			},
			map[string]any{
				"type": "https://example.com/error", "title": "Error", "status": float64(http.StatusBadRequest),
				"detail": "Bad request", "instance": "/test", "code": "ERR001",
			},
		},
		{
			"required only",
			&Detail{Title: "Error", Status: http.StatusInternalServerError},
			map[string]any{"title": "Error", "status": float64(http.StatusInternalServerError)},
		},
		{
			"with extensions",
			&Detail{
				Title: "Bad Request", Status: http.StatusBadRequest,
				Extensions: map[string]any{"errors": []string{"field required"}},
			},
			map[string]any{
				"title": "Bad Request", "status": float64(http.StatusBadRequest),
				"errors": []string{"field required"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := tt.detail.MarshalJSON()
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

func TestNewValidationDetail(t *testing.T) {
	errs := []ValidationError{
		{Detail: "required", Pointer: "#/name"},
		{Detail: "invalid", Field: "email"},
	}

	detail := NewValidationDetail("Validation failed", errs)

	if detail.Status != http.StatusUnprocessableEntity {
		t.Errorf("expected status 422, got %d", detail.Status)
	}

	if detail.Title != "Unprocessable Entity" {
		t.Errorf("expected title 'Unprocessable Entity', got %s", detail.Title)
	}

	errorsExt, exists := detail.Extensions["errors"]
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

func TestNewValidationDetail_Custom(t *testing.T) {
	type CustomError struct {
		Code string `json:"code"`
		Msg  string `json:"msg"`
	}

	errs := []CustomError{
		{Code: "ERR001", Msg: "Invalid input"},
	}

	detail := NewValidationDetail("Custom validation", errs)

	errorsExt := detail.Extensions["errors"].([]CustomError)
	if len(errorsExt) != 1 || errorsExt[0].Code != "ERR001" {
		t.Error("expected custom error to be preserved")
	}
}

func TestDetail_Set_ExtensionsInitialization(t *testing.T) {
	p := &Detail{
		Title:  "Test Error",
		Status: http.StatusBadRequest,
	}

	if p.Extensions != nil {
		t.Fatal("Expected Extensions to be nil initially")
	}

	result := p.Set("key", "value")

	if p.Extensions == nil {
		t.Fatal("Expected Extensions to be initialized after Set")
	}

	if val, ok := p.Extensions["key"]; !ok || val != "value" {
		t.Errorf("Expected Extensions to contain 'key' with value 'value', got %v", p.Extensions)
	}

	if result != p {
		t.Error("Expected Set to return same Detail instance for chaining")
	}

	p.Set("another", 123).Set("third", true)

	if len(p.Extensions) != 3 {
		t.Errorf("Expected Extensions to contain 3 items, got %d", len(p.Extensions))
	}

	if p.Extensions["another"] != 123 || p.Extensions["third"] != true {
		t.Error("Expected all extension values to be preserved")
	}
}

func TestDetail_Render(t *testing.T) {
	detail := NewDetail(http.StatusNotFound, "Not found")
	detail.Instance = "/users/123"
	detail.Set("timestamp", "2023-01-01T00:00:00Z")

	w := httptest.NewRecorder()

	err := detail.Render(w)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	zhtest.AssertWith(t, w).
		Status(http.StatusNotFound).
		Header(httpx.HeaderContentType, "application/problem+json").
		JSONPathEqual("title", "Not Found").
		JSONPathEqual("status", float64(http.StatusNotFound)).
		JSONPathEqual("detail", "Not found").
		JSONPathEqual("instance", "/users/123").
		JSONPathEqual("timestamp", "2023-01-01T00:00:00Z")
}

func TestDetail_Render_WithExtensions(t *testing.T) {
	detail := NewDetail(http.StatusUnprocessableEntity, "Validation failed")
	detail.Set("errors", []string{"name is required", "email is invalid"})
	detail.Set("code", "VALIDATION_ERROR")

	w := httptest.NewRecorder()

	err := detail.Render(w)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	zhtest.AssertWith(t, w).
		Status(http.StatusUnprocessableEntity).
		Header(httpx.HeaderContentType, "application/problem+json").
		JSONPathEqual("code", "VALIDATION_ERROR")

	var result map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if errors, exists := result["errors"]; !exists {
		t.Error("expected errors extension to exist")
	} else if errorList, ok := errors.([]any); !ok || len(errorList) != 2 {
		t.Errorf("expected errors to be array of 2 items, got %v", errors)
	}
}

func TestDetail_RenderAuto(t *testing.T) {
	tests := []struct {
		name         string
		acceptHeader string
		wantJSON     bool
	}{
		{"accepts JSON", "application/json", true},
		{"accepts problem+json", "application/problem+json", true},
		{"accepts wildcard", "*/*", true},
		{"accepts HTML only", "text/html", false},
		{"empty accept header", "", false},
		{"accepts JSON with quality", "application/json;q=0.9", true},
		{"accepts HTML with wildcard", "text/html,*/*;q=0.8", false},
		{"accepts browser header", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detail := NewDetail(http.StatusBadRequest, "Invalid request")
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.acceptHeader != "" {
				r.Header.Set(httpx.HeaderAccept, tt.acceptHeader)
			}

			err := detail.RenderAuto(w, r)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			if tt.wantJSON {
				zhtest.AssertWith(t, w).
					Status(http.StatusBadRequest).
					Header(httpx.HeaderContentType, "application/problem+json").
					JSONPathEqual("title", "Bad Request").
					JSONPathEqual("detail", "Invalid request")
			} else {
				zhtest.AssertWith(t, w).
					Status(http.StatusBadRequest).
					Header(httpx.HeaderContentType, "text/plain; charset=utf-8").
					BodyContains("Invalid request")
			}
		})
	}
}

func TestDetail_RenderAuto_FallbackToTitle(t *testing.T) {
	detail := &Detail{Title: "Custom Error Title", Status: http.StatusInternalServerError}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set(httpx.HeaderAccept, httpx.MIMETextPlain)

	err := detail.RenderAuto(w, r)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	zhtest.AssertWith(t, w).
		Status(http.StatusInternalServerError).
		Header(httpx.HeaderContentType, "text/plain; charset=utf-8").
		BodyContains("Custom Error Title")
}

func TestAcceptsJSON(t *testing.T) {
	tests := []struct {
		name   string
		header string
		want   bool
	}{
		{"empty header", "", false},
		{"application/json", "application/json", true},
		{"application/problem+json", "application/problem+json", true},
		{"text/html", "text/html", false},
		{"text/plain", "text/plain", false},
		{"*/*", "*/*", true},
		{"wildcard after json", "application/json, text/plain", true},
		{"json after wildcard", "text/plain, application/json", true},
		{"json with quality", "application/json;q=0.9", true},
		{"problem+json with quality", "application/problem+json;q=0.8", true},
		{"html only", "text/html,application/xhtml+xml,application/xml;q=0.9", false},
		{"html with wildcard", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8", false},
		{"browser accept header", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8", false},
		{"json higher priority than html", "application/json;q=0.9,text/html;q=0.8", true},
		{"html higher priority than json", "text/html;q=0.9,application/json;q=0.8", false},
		{"json explicit q=0 refusal", "application/json;q=0, */*", false},
		{"json q=0 with html", "application/json;q=0, text/html", false},
		{"json q=0 with wildcard and html", "application/json;q=0, text/html, */*;q=0.5", false},
		{"empty entry in accept", "application/json,,text/html", true},
		{"invalid quality value", "application/json;q=invalid, */*", true},
		{"quality out of range high", "application/json;q=1.5", true},
		{"quality out of range low", "application/json;q=-0.5", true},
		{"quality exactly 0", "application/json;q=0, text/html", false},
		{"quality exactly 1", "application/json;q=1", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.header != "" {
				req.Header.Set(httpx.HeaderAccept, tt.header)
			}
			if got := AcceptsJSON(req); got != tt.want {
				t.Errorf("AcceptsJSON() = %v, want %v", got, tt.want)
			}
		})
	}
}

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
