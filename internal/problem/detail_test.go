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

	zhtest.AssertEmpty(t, detail.Type)
	zhtest.AssertEqual(t, "Not Found", detail.Title)
	zhtest.AssertEqual(t, http.StatusNotFound, detail.Status)
	zhtest.AssertEqual(t, "Not found", detail.Detail)
	zhtest.AssertNotNil(t, detail.Extensions)
}

func TestDetail_Set(t *testing.T) {
	detail := NewDetail(400, "Bad request")

	result := detail.Set("field", "email").Set("code", "INVALID")

	zhtest.AssertEqual(t, detail, result)
	zhtest.AssertEqual(t, 2, len(detail.Extensions))
	zhtest.AssertEqual(t, "email", detail.Extensions["field"])
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
			zhtest.AssertNoError(t, err)

			var result map[string]any
			zhtest.AssertNoError(t, json.Unmarshal(data, &result))

			for key, expected := range tt.expected {
				actual, exists := result[key]
				zhtest.AssertTrue(t, exists)
				zhtest.AssertTrue(t, equalValues(actual, expected))
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

	zhtest.AssertEqual(t, http.StatusUnprocessableEntity, detail.Status)
	zhtest.AssertEqual(t, "Unprocessable Entity", detail.Title)

	errorsExt, exists := detail.Extensions["errors"]
	zhtest.AssertTrue(t, exists)

	validationErrors, ok := errorsExt.([]ValidationError)
	zhtest.AssertTrue(t, ok)
	zhtest.AssertEqual(t, 2, len(validationErrors))
	zhtest.AssertEqual(t, "required", validationErrors[0].Detail)
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
	zhtest.AssertEqual(t, 1, len(errorsExt))
	zhtest.AssertEqual(t, "ERR001", errorsExt[0].Code)
}

func TestDetail_Set_ExtensionsInitialization(t *testing.T) {
	p := &Detail{
		Title:  "Test Error",
		Status: http.StatusBadRequest,
	}

	zhtest.AssertNil(t, p.Extensions)

	result := p.Set("key", "value")

	zhtest.AssertNotNil(t, p.Extensions)
	val, ok := p.Extensions["key"]
	zhtest.AssertTrue(t, ok)
	zhtest.AssertEqual(t, "value", val)
	zhtest.AssertEqual(t, p, result)

	p.Set("another", 123).Set("third", true)

	zhtest.AssertEqual(t, 3, len(p.Extensions))
	zhtest.AssertEqual(t, 123, p.Extensions["another"])
	zhtest.AssertEqual(t, true, p.Extensions["third"])
}

func TestDetail_Render(t *testing.T) {
	detail := NewDetail(http.StatusNotFound, "Not found")
	detail.Instance = "/users/123"
	detail.Set("timestamp", "2023-01-01T00:00:00Z")

	w := httptest.NewRecorder()

	err := detail.Render(w)
	zhtest.AssertNoError(t, err)

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
	zhtest.AssertNoError(t, err)

	zhtest.AssertWith(t, w).
		Status(http.StatusUnprocessableEntity).
		Header(httpx.HeaderContentType, "application/problem+json").
		JSONPathEqual("code", "VALIDATION_ERROR")

	var result map[string]any
	zhtest.AssertNoError(t, json.Unmarshal(w.Body.Bytes(), &result))

	errors, exists := result["errors"]
	zhtest.AssertTrue(t, exists)
	errorList, ok := errors.([]any)
	zhtest.AssertTrue(t, ok)
	zhtest.AssertEqual(t, 2, len(errorList))
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
		{"empty accept header", "", true},
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
			zhtest.AssertNoError(t, err)

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
	zhtest.AssertNoError(t, err)

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
		{"empty header", "", true},
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
		{"vendor json type", "application/vnd.api.v1+json", true},
		{"vendor json with quality", "application/vnd.api.v1+json;q=0.9", true},
		{"vendor json vs html", "text/html;q=0.9,application/vnd.api.v1+json;q=0.8", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.header != "" {
				req.Header.Set(httpx.HeaderAccept, tt.header)
			}
			zhtest.AssertEqual(t, tt.want, AcceptsJSON(req))
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
