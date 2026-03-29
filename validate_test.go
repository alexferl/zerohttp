package zerohttp

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alexferl/zerohttp/httpx"
	"github.com/alexferl/zerohttp/validator"
	"github.com/alexferl/zerohttp/zhtest"
)

func TestBindAndValidate(t *testing.T) {
	type TestRequest struct {
		Name  string `json:"name" validate:"required,min=2"`
		Email string `json:"email" validate:"required,email"`
	}

	tests := []struct {
		name           string
		contentType    string
		body           string
		method         string
		wantErr        bool
		isBindingError bool
	}{
		{
			name:           "valid JSON",
			contentType:    "application/json",
			body:           `{"name":"John","email":"john@example.com"}`,
			wantErr:        false,
			isBindingError: false,
		},
		{
			name:           "invalid JSON",
			contentType:    "application/json",
			body:           `{"name":}`,
			wantErr:        true,
			isBindingError: true,
		},
		{
			name:           "validation error",
			contentType:    "application/json",
			body:           `{"name":"J","email":"not-an-email"}`,
			wantErr:        true,
			isBindingError: false,
		},
		{
			name:           "form data",
			contentType:    "application/x-www-form-urlencoded",
			body:           "name=John&email=john@example.com",
			wantErr:        false,
			isBindingError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			method := tt.method
			if method == "" {
				method = http.MethodPost
			}
			req := httptest.NewRequest(method, "/test", bytes.NewBufferString(tt.body))
			if tt.contentType != "" {
				req.Header.Set(httpx.HeaderContentType, tt.contentType)
			}

			var dst TestRequest
			err := BindAndValidate(req, &dst)

			if tt.wantErr {
				zhtest.AssertError(t, err)
				if tt.isBindingError {
					zhtest.AssertTrue(t, IsBindError(err))
				} else {
					zhtest.AssertTrue(t, IsValidationError(err))
				}
			} else {
				zhtest.AssertNoError(t, err)
			}
		})
	}
}

func TestRenderAndValidate(t *testing.T) {
	type TestResponse struct {
		Name  string `json:"name" validate:"required,min=2"`
		Email string `json:"email" validate:"required,email"`
	}

	t.Run("valid data renders JSON", func(t *testing.T) {
		w := httptest.NewRecorder()
		data := TestResponse{Name: "John", Email: "john@example.com"}
		err := RenderAndValidate(w, http.StatusOK, data)
		zhtest.AssertNoError(t, err)
		zhtest.AssertEqual(t, http.StatusOK, w.Code)
		body := w.Body.String()
		zhtest.AssertContains(t, body, `"name":"John"`)
	})

	t.Run("invalid data returns error", func(t *testing.T) {
		w := httptest.NewRecorder()
		data := TestResponse{Name: "J", Email: "not-an-email"}
		err := RenderAndValidate(w, http.StatusOK, data)
		zhtest.AssertError(t, err)
		zhtest.AssertContains(t, err.Error(), "invalid response data")
	})

	t.Run("invalid required field", func(t *testing.T) {
		w := httptest.NewRecorder()
		data := TestResponse{Email: "john@example.com"} // missing Name
		err := RenderAndValidate(w, http.StatusOK, data)
		zhtest.AssertError(t, err)
		zhtest.AssertContains(t, err.Error(), "invalid response data")
	})

	t.Run("valid slice of structs", func(t *testing.T) {
		w := httptest.NewRecorder()
		data := []TestResponse{
			{Name: "John", Email: "john@example.com"},
			{Name: "Jane", Email: "jane@example.com"},
		}
		err := RenderAndValidate(w, http.StatusOK, data)
		zhtest.AssertNoError(t, err)
		zhtest.AssertEqual(t, http.StatusOK, w.Code)
	})

	t.Run("invalid slice of structs", func(t *testing.T) {
		w := httptest.NewRecorder()
		data := []TestResponse{
			{Name: "John", Email: "john@example.com"},
			{Name: "J", Email: "invalid"}, // invalid entry
		}
		err := RenderAndValidate(w, http.StatusOK, data)
		zhtest.AssertError(t, err)
		zhtest.AssertContains(t, err.Error(), "invalid response data")
	})

	t.Run("valid pointer to slice of structs", func(t *testing.T) {
		w := httptest.NewRecorder()
		data := &[]TestResponse{
			{Name: "John", Email: "john@example.com"},
			{Name: "Jane", Email: "jane@example.com"},
		}
		err := RenderAndValidate(w, http.StatusOK, data)
		zhtest.AssertNoError(t, err)
		zhtest.AssertEqual(t, http.StatusOK, w.Code)
	})

	t.Run("valid array of structs", func(t *testing.T) {
		w := httptest.NewRecorder()
		data := [2]TestResponse{
			{Name: "John", Email: "john@example.com"},
			{Name: "Jane", Email: "jane@example.com"},
		}
		err := RenderAndValidate(w, http.StatusOK, data)
		zhtest.AssertNoError(t, err)
		zhtest.AssertEqual(t, http.StatusOK, w.Code)
	})

	t.Run("empty slice", func(t *testing.T) {
		w := httptest.NewRecorder()
		data := []TestResponse{}
		err := RenderAndValidate(w, http.StatusOK, data)
		zhtest.AssertNoError(t, err)
		zhtest.AssertEqual(t, http.StatusOK, w.Code)
	})
}

func TestBindError_Unwrap(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader("{invalid"))
	req.Header.Set(httpx.HeaderContentType, httpx.MIMEApplicationJSON)
	var dst struct{ Name string }
	err := BindAndValidate(req, &dst)
	zhtest.AssertError(t, err)
	// Test IsBindError with nil
	zhtest.AssertFalse(t, IsBindError(nil))
	// Test errors.As works with wrapped error
	var bindErr *validator.BindError
	zhtest.AssertTrue(t, errors.As(err, &bindErr))
	// Unwrap should return the inner error
	zhtest.AssertNotNil(t, bindErr.Unwrap())
}

func TestBindAndValidate_MultipartForm(t *testing.T) {
	// Build multipart form request
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	_ = writer.WriteField("name", "John")
	_ = writer.WriteField("email", "john@example.com")
	err := writer.Close()
	zhtest.AssertNoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/test", &buf)
	req.Header.Set(httpx.HeaderContentType, writer.FormDataContentType())

	type TestRequest struct {
		Name  string `form:"name" validate:"required"`
		Email string `form:"email" validate:"required,email"`
	}

	var dst TestRequest
	err = BindAndValidate(req, &dst)
	zhtest.AssertNoError(t, err)
	zhtest.AssertEqual(t, "John", dst.Name)
}

func TestBindAndValidate_QueryBinding(t *testing.T) {
	// GET request with no content-type should bind from query params
	req := httptest.NewRequest(http.MethodGet, "/test?name=John&email=john@example.com", nil)

	type TestRequest struct {
		Name  string `query:"name" validate:"required"`
		Email string `query:"email" validate:"required,email"`
	}

	var dst TestRequest
	err := BindAndValidate(req, &dst)
	zhtest.AssertNoError(t, err)
	zhtest.AssertEqual(t, "John", dst.Name)
}

func TestBindAndValidate_HeadMethod(t *testing.T) {
	// HEAD request with no content-type should also bind from query params
	req := httptest.NewRequest(http.MethodHead, "/test?name=John", nil)

	type TestRequest struct {
		Name string `query:"name" validate:"required"`
	}

	var dst TestRequest
	err := BindAndValidate(req, &dst)
	zhtest.AssertNoError(t, err)
	zhtest.AssertEqual(t, "John", dst.Name)
}

func TestBindAndValidate_DefaultToJSON(t *testing.T) {
	// Unknown content-type on POST should default to JSON
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(`{"name":"John"}`))
	req.Header.Set(httpx.HeaderContentType, "application/xml")

	type TestRequest struct {
		Name string `json:"name" validate:"required"`
	}

	var dst TestRequest
	err := BindAndValidate(req, &dst)
	zhtest.AssertNoError(t, err)
	zhtest.AssertEqual(t, "John", dst.Name)
}

func TestBindAndValidate_NoContentType(t *testing.T) {
	// POST with no content-type header should default to JSON
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(`{"name":"John"}`))

	type TestRequest struct {
		Name string `json:"name" validate:"required"`
	}

	var dst TestRequest
	err := BindAndValidate(req, &dst)
	zhtest.AssertNoError(t, err)
	zhtest.AssertEqual(t, "John", dst.Name)
}

func TestBindAndValidate_ContentTypeWithCharset(t *testing.T) {
	// Content-Type with charset suffix
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(`{"name":"John"}`))
	req.Header.Set(httpx.HeaderContentType, "application/json; charset=utf-8")

	type TestRequest struct {
		Name string `json:"name" validate:"required"`
	}

	var dst TestRequest
	err := BindAndValidate(req, &dst)
	zhtest.AssertNoError(t, err)
	zhtest.AssertEqual(t, "John", dst.Name)
}

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
			zhtest.AssertNoError(t, err)
			defer func() { _ = resp.Body.Close() }()

			zhtest.AssertEqual(t, tt.wantStatus, resp.StatusCode)

			// Check content type for error responses
			if tt.wantStatus >= 400 {
				contentType := resp.Header.Get(httpx.HeaderContentType)
				zhtest.AssertEqual(t, httpx.MIMEApplicationProblemJSON, contentType)

				var result map[string]any
				err := json.NewDecoder(resp.Body).Decode(&result)
				zhtest.AssertNoError(t, err)

				// Check RFC 7807 format
				_, ok := result["title"]
				zhtest.AssertTrue(t, ok)
				_, ok = result["status"]
				zhtest.AssertTrue(t, ok)
				_, ok = result["detail"]
				zhtest.AssertTrue(t, ok)

				// Check specific errors
				e, ok := result["errors"].(map[string]any)
				zhtest.AssertTrue(t, ok)

				for field := range tt.wantErrors {
					_, ok := e[field]
					zhtest.AssertTrue(t, ok)
				}
			}
		})
	}
}

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
			zhtest.AssertNoError(t, err)
			defer func() { _ = resp.Body.Close() }()

			zhtest.AssertEqual(t, tt.wantStatus, resp.StatusCode)
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
	zhtest.AssertNoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	zhtest.AssertEqual(t, http.StatusUnprocessableEntity, resp.StatusCode)

	var result map[string]any
	err = json.NewDecoder(resp.Body).Decode(&result)
	zhtest.AssertNoError(t, err)

	e, ok := result["errors"].(map[string]any)
	zhtest.AssertTrue(t, ok)

	// Error should be on crossFieldOrder (the struct type name)
	_, ok = e["crossFieldOrder"]
	zhtest.AssertTrue(t, ok)
}

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
	zhtest.AssertNoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	zhtest.AssertEqual(t, http.StatusUnprocessableEntity, resp.StatusCode)

	var result map[string]any
	err = json.NewDecoder(resp.Body).Decode(&result)
	zhtest.AssertNoError(t, err)

	e, ok := result["errors"].(map[string]any)
	zhtest.AssertTrue(t, ok)

	// Check each validation errors use JSON field names with index
	_, ok = e["tags[0]"]
	zhtest.AssertTrue(t, ok)
	_, ok = e["tags[1]"]
	zhtest.AssertTrue(t, ok)
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
	zhtest.AssertNoError(t, err)
	defer func() { _ = handlerResp.Body.Close() }()

	appResp, err := http.Post(appServer.URL+"/", "application/json", bytes.NewReader([]byte(body)))
	zhtest.AssertNoError(t, err)
	defer func() { _ = appResp.Body.Close() }()

	// Compare status codes
	zhtest.AssertEqual(t, handlerResp.StatusCode, appResp.StatusCode)
	zhtest.AssertEqual(t, http.StatusUnprocessableEntity, handlerResp.StatusCode)

	// Compare content types
	zhtest.AssertEqual(t, handlerResp.Header.Get(httpx.HeaderContentType), appResp.Header.Get(httpx.HeaderContentType))

	// Compare response bodies
	var handlerResult, appResult map[string]any
	err = json.NewDecoder(handlerResp.Body).Decode(&handlerResult)
	zhtest.AssertNoError(t, err)
	err = json.NewDecoder(appResp.Body).Decode(&appResult)
	zhtest.AssertNoError(t, err)

	// Compare title
	zhtest.AssertEqual(t, handlerResult["title"], appResult["title"])

	// Compare status
	zhtest.AssertEqual(t, handlerResult["status"], appResult["status"])

	// Compare detail
	zhtest.AssertEqual(t, handlerResult["detail"], appResult["detail"])

	// Compare errors
	handlerErrors, _ := handlerResult["errors"].(map[string]any)
	appErrors, _ := appResult["errors"].(map[string]any)

	zhtest.AssertEqual(t, len(handlerErrors), len(appErrors))

	for field := range handlerErrors {
		_, ok := appErrors[field]
		zhtest.AssertTrue(t, ok)
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
	zhtest.AssertNoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	// Should return 500, NOT 422 - this is a server bug, not client error
	zhtest.AssertEqual(t, http.StatusInternalServerError, resp.StatusCode)

	// Verify it's not returning 422 Unprocessable Entity
	zhtest.AssertNotEqual(t, http.StatusUnprocessableEntity, resp.StatusCode)
}
