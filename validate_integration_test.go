package zerohttp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alexferl/zerohttp/httpx"
)

// TestValidationHTTPResponse tests that validation errors are returned as proper HTTP responses
func TestValidationHTTPResponse(t *testing.T) {
	type CreateUserRequest struct {
		Name  string `json:"name" validate:"required,min=2,max=50"`
		Email string `json:"email" validate:"required,email"`
		Age   int    `json:"age" validate:"min=13,max=120"`
	}

	handler := HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		var req CreateUserRequest
		if err := BindAndValidate(r, &req); err != nil {
			return err
		}
		return R.JSON(w, http.StatusCreated, M{"name": req.Name, "email": req.Email})
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantErrors map[string]bool // field names that should have errors
	}{
		{
			name:       "valid request",
			body:       `{"name":"John Doe","email":"john@example.com","age":25}`,
			wantStatus: http.StatusCreated,
			wantErrors: nil,
		},
		{
			name:       "validation errors",
			body:       `{"name":"J","email":"bad","age":5}`,
			wantStatus: http.StatusUnprocessableEntity,
			wantErrors: map[string]bool{"name": true, "email": true, "age": true},
		},
		{
			name:       "missing required fields",
			body:       `{}`,
			wantStatus: http.StatusUnprocessableEntity,
			wantErrors: map[string]bool{"name": true, "email": true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := http.Post(server.URL, "application/json", bytes.NewReader([]byte(tt.body)))
			if err != nil {
				t.Fatalf("failed to make request: %v", err)
			}
			defer func() { _ = resp.Body.Close() }()

			if resp.StatusCode != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, resp.StatusCode)
			}

			// Check content type for error responses
			if tt.wantStatus >= 400 {
				contentType := resp.Header.Get(httpx.HeaderContentType)
				if contentType != httpx.MIMEApplicationProblemJSON {
					t.Errorf("expected application/problem+json, got %s", contentType)
				}

				var result map[string]any
				if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}

				// Check RFC 7807 format
				if _, ok := result["title"]; !ok {
					t.Error("expected title field in error response")
				}
				if _, ok := result["status"]; !ok {
					t.Error("expected status field in error response")
				}
				if _, ok := result["detail"]; !ok {
					t.Error("expected detail field in error response")
				}

				// Check specific errors
				errors, ok := result["errors"].(map[string]any)
				if !ok {
					t.Fatalf("expected errors object, got %T", result["errors"])
				}

				for field := range tt.wantErrors {
					if _, ok := errors[field]; !ok {
						t.Errorf("expected error for field %s, got errors: %v", field, errors)
					}
				}
			}
		})
	}
}

// TestBindingHTTPResponse tests that binding errors return 400
func TestBindingHTTPResponse(t *testing.T) {
	type TestRequest struct {
		Name string `json:"name" validate:"required"`
	}

	handler := HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		var req TestRequest
		if err := BindAndValidate(r, &req); err != nil {
			return err
		}
		return R.JSON(w, http.StatusOK, M{"name": req.Name})
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{
			name:       "invalid json",
			body:       `{"name":}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "wrong json type",
			body:       `[]`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "not json",
			body:       `not json at all`,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := http.Post(server.URL, "application/json", bytes.NewReader([]byte(tt.body)))
			if err != nil {
				t.Fatalf("failed to make request: %v", err)
			}
			defer func() { _ = resp.Body.Close() }()

			if resp.StatusCode != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, resp.StatusCode)
			}
		})
	}
}

// crossFieldOrder for testing cross-field validation
type crossFieldOrder struct {
	Items []string `json:"items" validate:"required,min=1"`
	Total float64  `json:"total" validate:"gte=0"`
}

// Validate checks that total matches number of items
func (o crossFieldOrder) Validate() error {
	expected := float64(len(o.Items)) * 10.0
	if o.Total != expected {
		return fmt.Errorf("total must equal %.2f (based on %d items)", expected, len(o.Items))
	}
	return nil
}

// TestCrossFieldValidationHTTP tests cross-field validation via Validate() method
func TestCrossFieldValidationHTTP(t *testing.T) {
	handler := HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		var req crossFieldOrder
		if err := BindAndValidate(r, &req); err != nil {
			return err
		}
		return R.JSON(w, http.StatusCreated, M{"total": req.Total})
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	// Request with mismatched total
	body := `{"items":["item1","item2"],"total":100.00}`
	resp, err := http.Post(server.URL, "application/json", bytes.NewReader([]byte(body)))
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("expected 422 for cross-field validation error, got %d", resp.StatusCode)
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	errors, ok := result["errors"].(map[string]any)
	if !ok {
		t.Fatalf("expected errors object, got %T", result["errors"])
	}

	// Error should be on crossFieldOrder (the struct type name)
	if _, ok := errors["crossFieldOrder"]; !ok {
		t.Errorf("expected crossFieldOrder error, got errors: %v", errors)
	}
}

// TestEachValidationHTTP tests each validator with slice elements
func TestEachValidationHTTP(t *testing.T) {
	type BulkRequest struct {
		Tags []string `json:"tags" validate:"each,min=3,max=10"`
	}

	handler := HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		var req BulkRequest
		if err := BindAndValidate(r, &req); err != nil {
			return err
		}
		return R.JSON(w, http.StatusCreated, M{"tags": req.Tags})
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	// Request with invalid tags
	body := `{"tags":["a","way-too-long"]}`
	resp, err := http.Post(server.URL, "application/json", bytes.NewReader([]byte(body)))
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d", resp.StatusCode)
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	errors, ok := result["errors"].(map[string]any)
	if !ok {
		t.Fatalf("expected errors object, got %T", result["errors"])
	}

	// Check each validation errors use JSON field names with index
	if _, ok := errors["tags[0]"]; !ok {
		t.Errorf("expected tags[0] error, got errors: %v", errors)
	}
	if _, ok := errors["tags[1]"]; !ok {
		t.Errorf("expected tags[1] error, got errors: %v", errors)
	}
}

// TestValidationWithAndWithoutRecoverMiddleware verifies that validation errors
// are handled the same way whether or not the Recover middleware is enabled.
func TestValidationWithAndWithoutRecoverMiddleware(t *testing.T) {
	type TestRequest struct {
		Name  string `json:"name" validate:"required,min=3"`
		Email string `json:"email" validate:"required,email"`
	}

	handler := HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		var req TestRequest
		if err := BindAndValidate(r, &req); err != nil {
			return err
		}
		return R.JSON(w, http.StatusOK, M{"name": req.Name, "email": req.Email})
	})

	// Test without Recover middleware (direct handler)
	handlerServer := httptest.NewServer(handler)
	defer handlerServer.Close()

	// Test with Recover middleware (via App)
	app := New()
	app.POST("/", handler)
	appServer := httptest.NewServer(app)
	defer appServer.Close()

	body := `{"name":"Jo","email":"not-an-email"}`

	// Make requests to both servers
	handlerResp, err := http.Post(handlerServer.URL, "application/json", bytes.NewReader([]byte(body)))
	if err != nil {
		t.Fatalf("failed to make request to handler: %v", err)
	}
	defer func() { _ = handlerResp.Body.Close() }()

	appResp, err := http.Post(appServer.URL+"/", "application/json", bytes.NewReader([]byte(body)))
	if err != nil {
		t.Fatalf("failed to make request to app: %v", err)
	}
	defer func() { _ = appResp.Body.Close() }()

	// Compare status codes
	if handlerResp.StatusCode != appResp.StatusCode {
		t.Errorf("status codes differ: handler=%d, app=%d", handlerResp.StatusCode, appResp.StatusCode)
	}
	if handlerResp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got handler=%d, app=%d", handlerResp.StatusCode, appResp.StatusCode)
	}

	// Compare content types
	if handlerResp.Header.Get(httpx.HeaderContentType) != appResp.Header.Get(httpx.HeaderContentType) {
		t.Errorf("content types differ: handler=%s, app=%s",
			handlerResp.Header.Get(httpx.HeaderContentType), appResp.Header.Get(httpx.HeaderContentType))
	}

	// Compare response bodies
	var handlerResult, appResult map[string]any
	if err := json.NewDecoder(handlerResp.Body).Decode(&handlerResult); err != nil {
		t.Fatalf("failed to decode handler response: %v", err)
	}
	if err := json.NewDecoder(appResp.Body).Decode(&appResult); err != nil {
		t.Fatalf("failed to decode app response: %v", err)
	}

	// Compare title
	if handlerResult["title"] != appResult["title"] {
		t.Errorf("titles differ: handler=%v, app=%v", handlerResult["title"], appResult["title"])
	}

	// Compare status
	if handlerResult["status"] != appResult["status"] {
		t.Errorf("statuses differ: handler=%v, app=%v", handlerResult["status"], appResult["status"])
	}

	// Compare detail
	if handlerResult["detail"] != appResult["detail"] {
		t.Errorf("details differ: handler=%v, app=%v", handlerResult["detail"], appResult["detail"])
	}

	// Compare errors
	handlerErrors, _ := handlerResult["errors"].(map[string]any)
	appErrors, _ := appResult["errors"].(map[string]any)

	if len(handlerErrors) != len(appErrors) {
		t.Errorf("error counts differ: handler=%d, app=%d", len(handlerErrors), len(appErrors))
	}

	for field := range handlerErrors {
		if _, ok := appErrors[field]; !ok {
			t.Errorf("app response missing error for field %s", field)
		}
	}
}

// TestRenderAndValidate_Returns500 tests that RenderAndValidate returns 500 (not 422)
// when response validation fails. This is a server-side bug, not a client error.
func TestRenderAndValidate_Returns500(t *testing.T) {
	type TestResponse struct {
		Name  string `json:"name" validate:"required,min=2"`
		Email string `json:"email" validate:"required,email"`
	}

	handler := HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		// Create invalid response data (simulating a server bug)
		data := TestResponse{Name: "J", Email: "not-an-email"}
		return RenderAndValidate(w, http.StatusOK, data)
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Should return 500, NOT 422 - this is a server bug, not client error
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected status %d (Internal Server Error), got %d", http.StatusInternalServerError, resp.StatusCode)
	}

	// Verify it's not returning 422 Unprocessable Entity
	if resp.StatusCode == http.StatusUnprocessableEntity {
		t.Error("RenderAndValidate incorrectly returned 422 Unprocessable Entity - should be 500 for server-side bugs")
	}
}
