package zerohttp

import (
	"bytes"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

// Test structs
type TestUser struct {
	Name  string `validate:"required,min=2,max=50"`
	Email string `validate:"required,email"`
	Age   int    `validate:"min=13,max=120"`
}

type TestOptional struct {
	Name  string `validate:"omitempty,min=2"`
	Email string `validate:"omitempty,email"`
}

type TestPointers struct {
	Name *string `validate:"required,min=2"`
	Age  *int    `validate:"omitempty,min=13"`
}

type TestNested struct {
	User    TestUser `validate:"required"`
	Address struct {
		Street string `validate:"required"`
		City   string `validate:"required"`
	}
}

type TestSlice struct {
	Tags  []string `validate:"min=1,max=5"`
	Items []struct {
		Name  string `validate:"required"`
		Price int    `validate:"min=0"`
	}
}

// Custom validator test
type TestCustom struct {
	Code string `validate:"custom_code"`
}

func TestValidationErrors_ValidUser(t *testing.T) {
	input := TestUser{Name: "John", Email: "john@example.com", Age: 25}
	err := V.Struct(&input)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestValidationErrors_MissingRequired(t *testing.T) {
	input := TestUser{}
	err := V.Struct(&input)
	if err == nil {
		t.Errorf("expected error, got nil")
		return
	}
	var ve ValidationErrors
	ok := errors.As(err, &ve)
	if !ok {
		t.Errorf("expected ValidationErrors, got %T", err)
		return
	}
	if len(ve.FieldErrors("Name")) == 0 {
		t.Errorf("expected Name error, got none")
	}
	if len(ve.FieldErrors("Email")) == 0 {
		t.Errorf("expected Email error, got none")
	}
}

func TestValidationErrors_InvalidEmail(t *testing.T) {
	input := TestUser{Name: "John", Email: "not-an-email", Age: 25}
	err := V.Struct(&input)
	if err == nil {
		t.Errorf("expected error, got nil")
		return
	}
	var ve ValidationErrors
	errors.As(err, &ve)
	errs := ve.FieldErrors("Email")
	found := false
	for _, e := range errs {
		if strings.Contains(e, "invalid email") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected invalid email error, got %v", errs)
	}
}

func TestValidationErrors_MinLength(t *testing.T) {
	input := TestUser{Name: "J", Email: "john@example.com", Age: 25}
	err := V.Struct(&input)
	if err == nil {
		t.Errorf("expected error, got nil")
		return
	}
	var ve ValidationErrors
	errors.As(err, &ve)
	errs := ve.FieldErrors("Name")
	found := false
	for _, e := range errs {
		if strings.Contains(e, "at least 2") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected min length error, got %v", errs)
	}
}

func TestValidationErrors_MaxLength(t *testing.T) {
	input := TestUser{Name: strings.Repeat("a", 51), Email: "john@example.com", Age: 25}
	err := V.Struct(&input)
	if err == nil {
		t.Errorf("expected error, got nil")
		return
	}
	var ve ValidationErrors
	errors.As(err, &ve)
	errs := ve.FieldErrors("Name")
	found := false
	for _, e := range errs {
		if strings.Contains(e, "at most 50") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected max length error, got %v", errs)
	}
}

func TestValidationErrors_Multiple(t *testing.T) {
	input := TestUser{Name: "", Email: "bad", Age: 5}
	err := V.Struct(&input)
	if err == nil {
		t.Errorf("expected error, got nil")
		return
	}
	var ve ValidationErrors
	errors.As(err, &ve)
	if len(ve.FieldErrors("Name")) == 0 {
		t.Errorf("expected Name error")
	}
	if len(ve.FieldErrors("Email")) == 0 {
		t.Errorf("expected Email error")
	}
	if len(ve.FieldErrors("Age")) == 0 {
		t.Errorf("expected Age error")
	}
}

func TestOmitempty(t *testing.T) {
	tests := []struct {
		name    string
		input   TestOptional
		wantErr bool
	}{
		{
			name:    "empty is valid",
			input:   TestOptional{},
			wantErr: false,
		},
		{
			name:    "valid name",
			input:   TestOptional{Name: "John"},
			wantErr: false,
		},
		{
			name:    "valid email",
			input:   TestOptional{Email: "john@example.com"},
			wantErr: false,
		},
		{
			name:    "short name fails",
			input:   TestOptional{Name: "J"},
			wantErr: true,
		},
		{
			name:    "bad email fails",
			input:   TestOptional{Email: "not-an-email"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := V.Struct(&tt.input)
			if tt.wantErr && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("expected no error, got %v", err)
			}
		})
	}
}

func TestPointerFields(t *testing.T) {
	name := "John"
	shortName := "J"
	age := 25

	tests := []struct {
		name    string
		input   TestPointers
		wantErr bool
		errors  map[string][]string
	}{
		{
			name:    "nil required pointer fails",
			input:   TestPointers{},
			wantErr: true,
			errors: map[string][]string{
				"Name": {"required"},
			},
		},
		{
			name:    "valid required pointer",
			input:   TestPointers{Name: &name},
			wantErr: false,
		},
		{
			name:    "short required pointer fails",
			input:   TestPointers{Name: &shortName},
			wantErr: true,
			errors: map[string][]string{
				"Name": {"must be at least 2 characters"},
			},
		},
		{
			name:    "nil optional pointer is ok",
			input:   TestPointers{Name: &name, Age: nil},
			wantErr: false,
		},
		{
			name:    "valid optional pointer",
			input:   TestPointers{Name: &name, Age: &age},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := V.Struct(&tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
					return
				}
				var ve ValidationErrors
				ok := errors.As(err, &ve)
				if !ok {
					t.Errorf("expected ValidationErrors, got %T", err)
					return
				}
				for field, expectedErrs := range tt.errors {
					actualErrs := ve.FieldErrors(field)
					for _, expected := range expectedErrs {
						found := false
						for _, actual := range actualErrs {
							if strings.Contains(actual, expected) {
								found = true
								break
							}
						}
						if !found {
							t.Errorf("expected error containing %q for field %s, got %v", expected, field, actualErrs)
						}
					}
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
			}
		})
	}
}

func TestNestedStructs(t *testing.T) {
	tests := []struct {
		name    string
		input   TestNested
		wantErr bool
		errors  map[string][]string
	}{
		{
			name: "valid nested",
			input: TestNested{
				User: TestUser{Name: "John", Email: "john@example.com", Age: 25},
				Address: struct {
					Street string `validate:"required"`
					City   string `validate:"required"`
				}{Street: "123 Main St", City: "NYC"},
			},
			wantErr: false,
		},
		{
			name:    "missing nested fields",
			input:   TestNested{},
			wantErr: true,
			errors: map[string][]string{
				"User.Name":      {"required"},
				"User.Email":     {"required"},
				"Address.Street": {"required"},
				"Address.City":   {"required"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := V.Struct(&tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
					return
				}
				var ve ValidationErrors
				ok := errors.As(err, &ve)
				if !ok {
					t.Errorf("expected ValidationErrors, got %T", err)
					return
				}
				for field, expectedErrs := range tt.errors {
					actualErrs := ve.FieldErrors(field)
					if len(actualErrs) == 0 {
						t.Errorf("expected errors for field %s, got none. All errors: %v", field, ve)
						continue
					}
					for _, expected := range expectedErrs {
						found := false
						for _, actual := range actualErrs {
							if strings.Contains(actual, expected) {
								found = true
								break
							}
						}
						if !found {
							t.Errorf("expected error containing %q for field %s, got %v", expected, field, actualErrs)
						}
					}
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
			}
		})
	}
}

func TestSliceValidation(t *testing.T) {
	tests := []struct {
		name    string
		input   TestSlice
		wantErr bool
		errors  map[string][]string
	}{
		{
			name:    "empty slice fails min",
			input:   TestSlice{},
			wantErr: true,
			errors: map[string][]string{
				"Tags": {"must have at least 1 items"},
			},
		},
		{
			name: "valid slice",
			input: TestSlice{
				Tags: []string{"a", "b"},
			},
			wantErr: false,
		},
		{
			name: "too many tags",
			input: TestSlice{
				Tags: []string{"a", "b", "c", "d", "e", "f"},
			},
			wantErr: true,
			errors: map[string][]string{
				"Tags": {"must have at most 5 items"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := V.Struct(&tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
					return
				}
				var ve ValidationErrors
				ok := errors.As(err, &ve)
				if !ok {
					t.Errorf("expected ValidationErrors, got %T", err)
					return
				}
				for field, expectedErrs := range tt.errors {
					actualErrs := ve.FieldErrors(field)
					if len(actualErrs) == 0 {
						t.Errorf("expected errors for field %s, got none. All errors: %v", field, ve)
						continue
					}
					for _, expected := range expectedErrs {
						found := false
						for _, actual := range actualErrs {
							if strings.Contains(actual, expected) {
								found = true
								break
							}
						}
						if !found {
							t.Errorf("expected error containing %q for field %s, got %v", expected, field, actualErrs)
						}
					}
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
			}
		})
	}
}

func TestCustomValidator(t *testing.T) {
	// Register custom validator
	V.Register("custom_code", func(value reflect.Value, param string) error {
		if value.Kind() != reflect.String {
			return errors.New("custom_code only supports strings")
		}
		code := value.String()
		if len(code) != 5 {
			return errors.New("code must be 5 characters")
		}
		return nil
	})

	tests := []struct {
		name    string
		input   TestCustom
		wantErr bool
	}{
		{
			name:    "valid code",
			input:   TestCustom{Code: "ABC12"},
			wantErr: false,
		},
		{
			name:    "invalid code length",
			input:   TestCustom{Code: "ABC1"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := V.Struct(&tt.input)
			if tt.wantErr && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("expected no error, got %v", err)
			}
		})
	}
}

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
				req.Header.Set("Content-Type", tt.contentType)
			}

			var dst TestRequest
			err := BindAndValidate(req, &dst)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
					return
				}
				if tt.isBindingError && !IsBindingError(err) {
					t.Errorf("expected binding error, got %T: %v", err, err)
				}
				if !tt.isBindingError && !IsValidationError(err) {
					t.Errorf("expected validation error, got %T: %v", err, err)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
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
		if err != nil {
			t.Errorf("expected no error, got %v", err)
			return
		}
		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
		}
		body := w.Body.String()
		if !strings.Contains(body, `"name":"John"`) {
			t.Errorf("expected JSON to contain name, got %s", body)
		}
	})

	t.Run("invalid data returns error", func(t *testing.T) {
		w := httptest.NewRecorder()
		data := TestResponse{Name: "J", Email: "not-an-email"}
		err := RenderAndValidate(w, http.StatusOK, data)
		if err == nil {
			t.Errorf("expected error, got nil")
			return
		}
		if !strings.Contains(err.Error(), "invalid response data") {
			t.Errorf("expected error to contain 'invalid response data', got %v", err)
		}
	})

	t.Run("invalid required field", func(t *testing.T) {
		w := httptest.NewRecorder()
		data := TestResponse{Email: "john@example.com"} // missing Name
		err := RenderAndValidate(w, http.StatusOK, data)
		if err == nil {
			t.Errorf("expected error for missing required field, got nil")
			return
		}
		if !strings.Contains(err.Error(), "invalid response data") {
			t.Errorf("expected error to contain 'invalid response data', got %v", err)
		}
	})
}

func TestValidationErrors_HasErrors(t *testing.T) {
	ve := ValidationErrors{}
	if ve.HasErrors() {
		t.Error("expected HasErrors to be false for empty errors")
	}
	ve.Add("field", "error")
	if !ve.HasErrors() {
		t.Error("expected HasErrors to be true when errors exist")
	}
}

func TestValidationErrors_ValidationErrors(t *testing.T) {
	ve := ValidationErrors{
		"Name": {"required"},
		"Age":  {"min"},
	}
	errs := ve.ValidationErrors()
	if len(errs["Name"]) != 1 || errs["Name"][0] != "required" {
		t.Errorf("expected Name error, got %v", errs["Name"])
	}
	if len(errs["Age"]) != 1 || errs["Age"][0] != "min" {
		t.Errorf("expected Age error, got %v", errs["Age"])
	}
}

func TestBindError_Unwrap(t *testing.T) {
	inner := errors.New("inner error")
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader("{invalid"))
	req.Header.Set("Content-Type", "application/json")
	var dst struct{ Name string }
	err := BindAndValidate(req, &dst)
	if err == nil {
		t.Fatal("expected error")
	}
	// Test IsBindingError with nil
	if IsBindingError(nil) {
		t.Error("expected IsBindingError(nil) to be false")
	}
	// Test errors.As works with wrapped error
	var bindErr *BindError
	if !errors.As(err, &bindErr) {
		t.Error("expected error to be BindError")
	}
	// Unwrap should return the inner error
	if bindErr.Unwrap() == nil {
		t.Error("expected Unwrap to return the inner error")
	}
	_ = inner // suppress unused warning
}

func TestBindAndValidate_MultipartForm(t *testing.T) {
	// Build multipart form request
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	_ = writer.WriteField("name", "John")
	_ = writer.WriteField("email", "john@example.com")
	if err := writer.Close(); err != nil {
		t.Fatalf("failed to close multipart writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/test", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	type TestRequest struct {
		Name  string `form:"name" validate:"required"`
		Email string `form:"email" validate:"required,email"`
	}

	var dst TestRequest
	err := BindAndValidate(req, &dst)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if dst.Name != "John" {
		t.Errorf("expected Name=John, got %s", dst.Name)
	}
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
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if dst.Name != "John" {
		t.Errorf("expected Name=John, got %s", dst.Name)
	}
}

func TestBindAndValidate_HeadMethod(t *testing.T) {
	// HEAD request with no content-type should also bind from query params
	req := httptest.NewRequest(http.MethodHead, "/test?name=John", nil)

	type TestRequest struct {
		Name string `query:"name" validate:"required"`
	}

	var dst TestRequest
	err := BindAndValidate(req, &dst)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if dst.Name != "John" {
		t.Errorf("expected Name=John, got %s", dst.Name)
	}
}

func TestBindAndValidate_DefaultToJSON(t *testing.T) {
	// Unknown content-type on POST should default to JSON
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(`{"name":"John"}`))
	req.Header.Set("Content-Type", "application/xml")

	type TestRequest struct {
		Name string `json:"name" validate:"required"`
	}

	var dst TestRequest
	err := BindAndValidate(req, &dst)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if dst.Name != "John" {
		t.Errorf("expected Name=John, got %s", dst.Name)
	}
}

func TestBindAndValidate_NoContentType(t *testing.T) {
	// POST with no content-type header should default to JSON
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(`{"name":"John"}`))

	type TestRequest struct {
		Name string `json:"name" validate:"required"`
	}

	var dst TestRequest
	err := BindAndValidate(req, &dst)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if dst.Name != "John" {
		t.Errorf("expected Name=John, got %s", dst.Name)
	}
}

func TestBindAndValidate_ContentTypeWithCharset(t *testing.T) {
	// Content-Type with charset suffix
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(`{"name":"John"}`))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	type TestRequest struct {
		Name string `json:"name" validate:"required"`
	}

	var dst TestRequest
	err := BindAndValidate(req, &dst)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if dst.Name != "John" {
		t.Errorf("expected Name=John, got %s", dst.Name)
	}
}

// rootValidatableOrder is used to test root struct validation
type rootValidatableOrder struct {
	Items    []string `validate:"required,min=1"`
	Total    float64  `validate:"gte=0"`
	Discount float64  `validate:"gte=0"`
}

// Validate implements custom cross-field validation on the root struct
func (o rootValidatableOrder) Validate() error {
	var sum float64
	for range o.Items {
		sum += 10.0 // simplified pricing
	}
	if o.Total != sum {
		return fmt.Errorf("total must equal sum of items")
	}
	if o.Discount > o.Total {
		return fmt.Errorf("discount cannot exceed total")
	}
	return nil
}

// TestRootStructValidate tests that Validate() is called on the root struct itself
func TestRootStructValidate(t *testing.T) {
	t.Run("valid order", func(t *testing.T) {
		input := rootValidatableOrder{
			Items:    []string{"item1", "item2"},
			Total:    20.0,
			Discount: 5.0,
		}
		err := V.Struct(&input)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("invalid total - cross field validation fails", func(t *testing.T) {
		input := rootValidatableOrder{
			Items: []string{"item1", "item2"},
			Total: 100.0, // Wrong total
		}
		err := V.Struct(&input)
		if err == nil {
			t.Error("expected error for mismatched total")
		}
		var ve ValidationErrors
		errors.As(err, &ve)
		// Root-level validation errors use the struct type name
		if errs := ve.FieldErrors("rootValidatableOrder"); len(errs) == 0 {
			t.Errorf("expected root validation error, got %v", ve)
		}
	})

	t.Run("discount exceeds total", func(t *testing.T) {
		input := rootValidatableOrder{
			Items:    []string{"item1"},
			Total:    10.0,
			Discount: 20.0, // More than total
		}
		err := V.Struct(&input)
		if err == nil {
			t.Error("expected error for discount exceeding total")
		}
	})
}

func TestEmbeddedStruct(t *testing.T) {
	type Embedded struct {
		Name string `validate:"required"`
	}
	type TestEmbedded struct {
		Embedded
		Age int `validate:"min=0"`
	}

	t.Run("valid embedded", func(t *testing.T) {
		input := TestEmbedded{Embedded: Embedded{Name: "John"}, Age: 25}
		err := V.Struct(&input)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("invalid embedded field", func(t *testing.T) {
		input := TestEmbedded{Age: 25}
		err := V.Struct(&input)
		if err == nil {
			t.Error("expected error for missing Name in embedded struct")
		}
		var ve ValidationErrors
		errors.As(err, &ve)
		if errs := ve.FieldErrors("Name"); len(errs) == 0 {
			t.Errorf("expected Name error, got %v", ve)
		}
	})
}

func TestNestedPointerValidation(t *testing.T) {
	type Inner struct {
		Name string `validate:"required"`
	}
	type TestNestedPtr struct {
		Inner *Inner `validate:"required"`
	}

	t.Run("nil required pointer", func(t *testing.T) {
		input := TestNestedPtr{Inner: nil}
		err := V.Struct(&input)
		if err == nil {
			t.Error("expected error for nil required pointer")
		}
	})

	t.Run("valid pointer", func(t *testing.T) {
		input := TestNestedPtr{Inner: &Inner{Name: "John"}}
		err := V.Struct(&input)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("invalid inner struct", func(t *testing.T) {
		input := TestNestedPtr{Inner: &Inner{Name: ""}}
		err := V.Struct(&input)
		if err == nil {
			t.Error("expected error for invalid inner struct")
		}
	})
}

func TestPointerWithOmitEmpty(t *testing.T) {
	type TestPtrOmit struct {
		Name  *string `validate:"omitempty,min=2"`
		Email *string `validate:"omitempty,email"`
	}

	t.Run("nil pointer with omitempty is valid", func(t *testing.T) {
		input := TestPtrOmit{}
		err := V.Struct(&input)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("valid pointer with omitempty", func(t *testing.T) {
		name := "John"
		email := "john@example.com"
		input := TestPtrOmit{Name: &name, Email: &email}
		err := V.Struct(&input)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("invalid short name with omitempty", func(t *testing.T) {
		name := "J"
		input := TestPtrOmit{Name: &name}
		err := V.Struct(&input)
		if err == nil {
			t.Error("expected error for short name")
		}
	})
}

func TestValidationErrors_Empty(t *testing.T) {
	// Test empty ValidationErrors.Error() returns simple message
	ve := ValidationErrors{}
	msg := ve.Error()
	if msg != "validation failed" {
		t.Errorf("expected 'validation failed', got %q", msg)
	}
}

// TestJSONFieldNameInErrors verifies that validation errors use json tag names
func TestJSONFieldNameInErrors(t *testing.T) {
	type TestRequest struct {
		UserName string `json:"user_name" validate:"required,min=5"`
		Email    string `json:"email_address" validate:"required,email"`
	}

	input := TestRequest{
		UserName: "ab",  // too short
		Email:    "bad", // invalid email
	}

	err := V.Struct(&input)
	if err == nil {
		t.Fatal("expected error")
	}

	var ve ValidationErrors
	ok := errors.As(err, &ve)
	if !ok {
		t.Fatalf("expected ValidationErrors, got %T", err)
	}

	// Errors should use json tag names, not Go field names
	if errs := ve.FieldErrors("user_name"); len(errs) == 0 {
		t.Errorf("expected error on 'user_name' (json tag), got errors: %v", ve)
	}
	if errs := ve.FieldErrors("Username"); len(errs) > 0 {
		t.Errorf("should not have error on 'Username' (Go field name), got: %v", errs)
	}

	if errs := ve.FieldErrors("email_address"); len(errs) == 0 {
		t.Errorf("expected error on 'email_address' (json tag), got errors: %v", ve)
	}
	if errs := ve.FieldErrors("Email"); len(errs) > 0 {
		t.Errorf("should not have error on 'Email' (Go field name), got: %v", errs)
	}
}

// TestJSONFieldNameInNestedErrors verifies json tag names in nested structs
func TestJSONFieldNameInNestedErrors(t *testing.T) {
	type Address struct {
		Street string `json:"street_address" validate:"required"`
		City   string `json:"city_name" validate:"required"`
	}
	type Person struct {
		Name    string  `json:"full_name" validate:"required"`
		Address Address `json:"home_address"`
	}

	input := Person{
		Name:    "",
		Address: Address{Street: "", City: "NYC"},
	}

	err := V.Struct(&input)
	if err == nil {
		t.Fatal("expected error")
	}

	var ve ValidationErrors
	ok := errors.As(err, &ve)
	if !ok {
		t.Fatalf("expected ValidationErrors, got %T", err)
	}

	// Check nested paths use json tag names
	if errs := ve.FieldErrors("full_name"); len(errs) == 0 {
		t.Errorf("expected error on 'full_name', got errors: %v", ve)
	}
	if errs := ve.FieldErrors("home_address.street_address"); len(errs) == 0 {
		t.Errorf("expected error on 'home_address.street_address', got errors: %v", ve)
	}
}

// TestAnonymousEmbeddedStruct tests validation of anonymous embedded structs
func TestAnonymousEmbeddedStruct(t *testing.T) {
	type Embedded struct {
		Value string `validate:"required"`
	}

	type TestAnonymous struct {
		Embedded
		Name string `validate:"required"`
	}

	tests := []struct {
		name     string
		input    TestAnonymous
		wantErr  bool
		errField string
	}{
		{
			name:    "all valid",
			input:   TestAnonymous{Embedded: Embedded{Value: "embedded"}, Name: "test"},
			wantErr: false,
		},
		{
			name:     "embedded field invalid",
			input:    TestAnonymous{Embedded: Embedded{Value: ""}, Name: "test"},
			wantErr:  true,
			errField: "Value",
		},
		{
			name:     "regular field invalid",
			input:    TestAnonymous{Embedded: Embedded{Value: "embedded"}, Name: ""},
			wantErr:  true,
			errField: "Name",
		},
		{
			name:     "both invalid",
			input:    TestAnonymous{Embedded: Embedded{Value: ""}, Name: ""},
			wantErr:  true,
			errField: "Value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := V.Struct(&tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
					return
				}
				var ve ValidationErrors
				ok := errors.As(err, &ve)
				if !ok {
					t.Errorf("expected ValidationErrors, got %T", err)
					return
				}
				errs := ve.FieldErrors(tt.errField)
				if len(errs) == 0 {
					t.Errorf("expected error for field %s, got none", tt.errField)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
			}
		})
	}
}

// TestAnonymousEmbeddedStructWithJSONTags tests anonymous embedded structs with json tags
func TestAnonymousEmbeddedStructWithJSONTags(t *testing.T) {
	type Embedded struct {
		Value string `json:"embedded_value" validate:"required"`
	}

	type TestAnonymous struct {
		Embedded
		Name string `json:"name" validate:"required"`
	}

	tests := []struct {
		name     string
		input    TestAnonymous
		wantErr  bool
		errField string
	}{
		{
			name:    "all valid",
			input:   TestAnonymous{Embedded: Embedded{Value: "embedded"}, Name: "test"},
			wantErr: false,
		},
		{
			name:     "embedded field uses json tag",
			input:    TestAnonymous{Embedded: Embedded{Value: ""}, Name: "test"},
			wantErr:  true,
			errField: "embedded_value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := V.Struct(&tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
					return
				}
				var ve ValidationErrors
				ok := errors.As(err, &ve)
				if !ok {
					t.Errorf("expected ValidationErrors, got %T", err)
					return
				}
				errs := ve.FieldErrors(tt.errField)
				if len(errs) == 0 {
					t.Errorf("expected error for field %s, got none", tt.errField)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
			}
		})
	}
}

// TestValidationErrorsMethods tests ValidationErrors helper methods
func TestValidationErrorsMethods(t *testing.T) {
	t.Run("HasErrors with errors", func(t *testing.T) {
		type Test struct {
			Value string `validate:"required"`
		}
		input := Test{Value: ""}
		err := V.Struct(&input)
		if err == nil {
			t.Fatal("expected error")
		}
		var ve ValidationErrors
		errors.As(err, &ve)
		if !ve.HasErrors() {
			t.Error("HasErrors should return true when there are errors")
		}
	})

	t.Run("FieldErrors for non-existent field", func(t *testing.T) {
		type Test struct {
			Value string `validate:"required"`
		}
		input := Test{Value: ""}
		err := V.Struct(&input)
		if err == nil {
			t.Fatal("expected error")
		}
		var ve ValidationErrors
		errors.As(err, &ve)
		errs := ve.FieldErrors("nonexistent")
		if len(errs) != 0 {
			t.Errorf("expected empty slice for non-existent field, got: %v", errs)
		}
	})

	t.Run("Add error manually", func(t *testing.T) {
		ve := make(ValidationErrors)
		ve.Add("field1", "error 1")
		ve.Add("field1", "error 2")
		ve.Add("field2", "error 3")

		if len(ve["field1"]) != 2 {
			t.Errorf("expected 2 errors for field1, got: %d", len(ve["field1"]))
		}
		if len(ve["field2"]) != 1 {
			t.Errorf("expected 1 error for field2, got: %d", len(ve["field2"]))
		}
	})

	t.Run("Error string format", func(t *testing.T) {
		ve := make(ValidationErrors)
		ve.Add("field1", "error 1")
		ve.Add("field2", "error 2")

		msg := ve.Error()
		if msg == "" {
			t.Error("Error() should return non-empty string")
		}
		if msg == "validation failed" {
			t.Error("Error() should include field details when there are errors")
		}
	})

	t.Run("Error on empty ValidationErrors", func(t *testing.T) {
		ve := ValidationErrors{}
		msg := ve.Error()
		if msg != "validation failed" {
			t.Errorf("expected 'validation failed', got: %s", msg)
		}
	})

	t.Run("ValidationErrors map accessor", func(t *testing.T) {
		ve := make(ValidationErrors)
		ve.Add("field", "error")

		m := ve.ValidationErrors()
		if len(m["field"]) != 1 {
			t.Errorf("expected 1 error, got: %d", len(m["field"]))
		}
	})
}
