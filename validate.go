package zerohttp

import (
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/alexferl/zerohttp/httpx"
	"github.com/alexferl/zerohttp/validator"
)

// Validate is the default [Validator] instance used by the package.
// Use it to validate structs using struct tags:
//
//	type CreateUserRequest struct {
//	    Name  string `json:"name" validate:"required,min=2"`
//	    Email string `json:"email" validate:"required,email"`
//	}
//
//	if err := zh.Validate.Struct(&req); err != nil {
//	    // Returns 422 Unprocessable Entity with field errors
//	    return err
//	}
//
// For convenience, use the [V] alias or [BindAndValidate] for combined binding and validation.
var Validate = validator.New()

// V is a short alias for [Validate].
//
//	if err := zh.V.Struct(&req); err != nil {
//	    return err
//	}
var V = Validate

// DefaultMultipartMaxMemory is the default max memory for multipart form parsing in BindAndValidate.
// This can be changed globally. Default is 32MB.
var DefaultMultipartMaxMemory int64 = 32 << 20

// BindAndValidate binds request data based on Content-Type and validates the result.
// It returns appropriate errors:
//   - 400 Bad Request for binding failures (malformed JSON, type mismatches)
//   - 422 Unprocessable Entity for validation failures
//
// Supported Content-Types:
//   - application/json
//   - application/x-www-form-urlencoded
//   - multipart/form-data
//   - (no content-type) - parses query parameters
//
// Example:
//
//	func handler(w http.ResponseWriter, r *http.Request) error {
//	    var req CreateUserRequest
//	    if err := zh.BindAndValidate(r, &req); err != nil {
//	        return err  // 400 or 422 auto-detected
//	    }
//	    // ...
//	}
func BindAndValidate(r *http.Request, dst any) error {
	contentType := r.Header.Get(httpx.HeaderContentType)

	// Strip charset suffix if present
	if idx := strings.Index(contentType, ";"); idx > 0 {
		contentType = strings.TrimSpace(contentType[:idx])
	}

	var bindErr error
	switch contentType {
	case httpx.MIMEApplicationJSON:
		bindErr = Bind.JSON(r.Body, dst)
	case httpx.MIMEApplicationFormURLEncoded:
		bindErr = Bind.Form(r, dst)
	case httpx.MIMEMultipartFormData:
		// Use default max memory for multipart forms
		bindErr = Bind.MultipartForm(r, dst, DefaultMultipartMaxMemory)
	default:
		// No content-type or unknown - try query binding for GET/HEAD, JSON for others
		if r.Method == http.MethodGet || r.Method == http.MethodHead {
			bindErr = Bind.Query(r, dst)
		} else {
			bindErr = Bind.JSON(r.Body, dst)
		}
	}

	if bindErr != nil {
		// Wrap as binding error (400)
		return &validator.BindError{Err: bindErr}
	}

	if valErr := V.Struct(dst); valErr != nil {
		return valErr // Already ValidationErrors (422)
	}

	return nil
}

// IsBindError checks if an error is a binding error (should return 400).
func IsBindError(err error) bool {
	return validator.IsBindError(err)
}

// IsValidationError checks if an error is a validation error (should return 422).
func IsValidationError(err error) bool {
	var validationErrorer validator.ValidationErrorer
	ok := errors.As(err, &validationErrorer)
	return ok
}

// RenderAndValidate renders JSON response after validating the data.
// This catches server-side bugs like missing required fields before sending responses.
//
// If validation fails, it returns a 500 Internal Server Error (server bug).
//
// Example:
//
//	func handler(w http.ResponseWriter, r *http.Request) error {
//	    user := User{ID: "...", Name: "John"}
//	    return zh.RenderAndValidate(w, http.StatusOK, user)
//	}
func RenderAndValidate(w http.ResponseWriter, status int, data any) error {
	if err := validateResponseData(data); err != nil {
		// Log error for developers - this is a server-side bug
		// Use %v (not %w) to prevent errors.As from matching ValidationErrorer
		return fmt.Errorf("invalid response data: %v", err)
	}
	return R.JSON(w, status, data)
}

// validateResponseData validates data for response rendering.
// It handles both structs and slices of structs.
func validateResponseData(data any) error {
	v := reflect.ValueOf(data)

	// Handle pointer
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	// Handle slices - validate each element
	if v.Kind() == reflect.Slice || v.Kind() == reflect.Array {
		for i := 0; i < v.Len(); i++ {
			elem := v.Index(i).Interface()
			if err := V.Struct(elem); err != nil {
				return err
			}
		}
		return nil
	}

	// Handle structs
	return V.Struct(data)
}
