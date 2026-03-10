package zerohttp

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/mail"
	"net/url"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	zerrors "github.com/alexferl/zerohttp/internal/errors"
)

// Validate is the default validator instance used by the package
var Validate = NewValidator()

// V is a short alias for Validate for convenience
var V = Validate

// ValidationFunc is a custom validation function.
// It receives the field value and returns an error if validation fails.
type ValidationFunc func(value reflect.Value, param string) error

// Validator handles struct validation using reflection and struct tags.
type Validator interface {
	// Struct validates a struct using `validate` struct tags.
	// It returns a ValidationErrors containing all validation failures,
	// or nil if the struct is valid.
	Struct(dst any) error

	// Register adds a custom validation function with the given name.
	// The name can be used in struct tags like `validate:"customName"`.
	Register(name string, fn ValidationFunc)
}

// defaultValidator implements the Validator interface
// Ensure defaultValidator implements Validator
var _ Validator = (*defaultValidator)(nil)

type defaultValidator struct {
	mu         sync.RWMutex
	validators map[string]ValidationFunc
}

// NewValidator creates a new validator instance with built-in validation rules.
func NewValidator() Validator {
	v := &defaultValidator{
		validators: make(map[string]ValidationFunc),
	}

	v.registerBuiltins()

	return v
}

// ValidationErrors holds all validation errors for a struct.
// The key is the field path (e.g., "Name", "Address.City", "Items[0].Name").
type ValidationErrors map[string][]string

// Error implements the error interface.
func (ve ValidationErrors) Error() string {
	if len(ve) == 0 {
		return "validation failed"
	}

	var parts []string
	for field, errs := range ve {
		parts = append(parts, fmt.Sprintf("%s: %s", field, strings.Join(errs, ", ")))
	}
	return "validation failed: " + strings.Join(parts, "; ")
}

// HasErrors returns true if there are any validation errors.
func (ve ValidationErrors) HasErrors() bool {
	return len(ve) > 0
}

// FieldErrors returns all errors for a specific field.
func (ve ValidationErrors) FieldErrors(field string) []string {
	return ve[field]
}

// Add adds an error for a specific field.
func (ve ValidationErrors) Add(field, err string) {
	ve[field] = append(ve[field], err)
}

// ValidationErrors returns the errors map (implements ValidationErrorer interface).
func (ve ValidationErrors) ValidationErrors() map[string][]string {
	return ve
}

// ValidationErrorer is implemented by validation error types.
// The default error handler uses this to detect validation errors
// and return 422 Unprocessable Entity with proper formatting.
type ValidationErrorer interface {
	error
	ValidationErrors() map[string][]string
}

// Ensure ValidationErrors implements ValidationErrorer
var _ ValidationErrorer = (ValidationErrors)(nil)

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
	contentType := r.Header.Get(HeaderContentType)

	// Strip charset suffix if present
	if idx := strings.Index(contentType, ";"); idx > 0 {
		contentType = strings.TrimSpace(contentType[:idx])
	}

	var bindErr error
	switch contentType {
	case MIMEApplicationJSON:
		bindErr = Bind.JSON(r.Body, dst)
	case MIMEApplicationFormURLEncoded:
		bindErr = Bind.Form(r, dst)
	case MIMEMultipartFormData:
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
		return &zerrors.BindError{Err: bindErr}
	}

	// Validate
	if valErr := V.Struct(dst); valErr != nil {
		return valErr // Already ValidationErrors (422)
	}

	return nil
}

// BindError is an alias for internal/errors.BindError
type BindError = zerrors.BindError

// IsBindError checks if an error is a binding error (should return 400).
func IsBindError(err error) bool {
	return zerrors.IsBindError(err)
}

// IsBindingError checks if an error is a binding error (should return 400).
func IsBindingError(err error) bool {
	return IsBindError(err)
}

// IsValidationError checks if an error is a validation error (should return 422).
func IsValidationError(err error) bool {
	var validationErrorer ValidationErrorer
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
	// Validate the response data
	if err := V.Struct(data); err != nil {
		// Log error for developers - this is a server-side bug
		return fmt.Errorf("invalid response data: %w", err)
	}
	return R.JSON(w, status, data)
}

// Struct validates a struct using `validate` struct tags.
func (v *defaultValidator) Struct(dst any) error {
	if dst == nil {
		return ValidationErrors{"": {"nil value"}}
	}

	val := reflect.ValueOf(dst)
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return ValidationErrors{"": {"nil pointer"}}
		}
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return ValidationErrors{"": {fmt.Sprintf("expected struct, got %s", val.Kind())}}
	}

	errs := make(ValidationErrors)
	v.validateStruct(val, "", errs)

	if len(errs) > 0 {
		return errs
	}
	return nil
}

// validateStruct recursively validates a struct and its fields.
func (v *defaultValidator) validateStruct(val reflect.Value, prefix string, errors ValidationErrors) {
	typ := val.Type()

	// Check if the struct itself implements Validate() error
	// This allows custom validation methods on the struct type
	if val.CanAddr() {
		if validator, ok := val.Addr().Interface().(interface{ Validate() error }); ok {
			if err := validator.Validate(); err != nil {
				// Use the prefix if available, otherwise use the struct type name
				key := prefix
				if key == "" {
					key = typ.Name()
				}
				errors.Add(key, err.Error())
			}
		}
	}

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		// Skip unexported fields
		if !field.CanInterface() {
			continue
		}

		// Handle embedded structs - validate them recursively
		if fieldType.Anonymous && field.Kind() == reflect.Struct {
			v.validateStruct(field, prefix, errors)
			continue
		}

		// Use json tag name if available, otherwise use struct field name
		fieldName := getJSONFieldName(fieldType)
		if prefix != "" {
			fieldName = prefix + "." + fieldName
		}

		// Handle pointer fields
		if field.Kind() == reflect.Ptr {
			if field.IsNil() {
				// Check if this is a required pointer field
				tag := fieldType.Tag.Get("validate")
				if tag != "" && tag != "-" {
					rules := parseTag(tag)
					for _, rule := range rules {
						if rule.Name == "required" {
							errors.Add(fieldName, "required")
							break
						}
					}
				}
				continue
			}
			// Dereference and validate the pointed-to value
			field = field.Elem()
		}

		// Get validate tag
		tag := fieldType.Tag.Get("validate")
		if tag != "" && tag != "-" {
			v.validateField(field, fieldName, tag, errors)
		}

		// Recursively validate nested structs
		switch field.Kind() {
		case reflect.Struct:
			// Skip time.Time and other common types
			if field.Type().String() == "time.Time" {
				continue
			}
			v.validateStruct(field, fieldName, errors)
		case reflect.Slice, reflect.Array:
			for j := 0; j < field.Len(); j++ {
				elem := field.Index(j)
				elemName := fmt.Sprintf("%s[%d]", fieldName, j)
				if elem.Kind() == reflect.Struct {
					v.validateStruct(elem, elemName, errors)
				} else if elem.Kind() == reflect.Ptr && !elem.IsNil() && elem.Elem().Kind() == reflect.Struct {
					v.validateStruct(elem.Elem(), elemName, errors)
				}
			}
		case reflect.Map:
			// Recursively validate struct values in maps
			for _, key := range field.MapKeys() {
				elem := field.MapIndex(key)
				elemName := fmt.Sprintf("%s[%v]", fieldName, key)
				if elem.Kind() == reflect.Struct {
					v.validateStruct(elem, elemName, errors)
				} else if elem.Kind() == reflect.Ptr && !elem.IsNil() && elem.Elem().Kind() == reflect.Struct {
					v.validateStruct(elem.Elem(), elemName, errors)
				}
			}
		default:
			// No recursive validation for other kinds
		}
	}
}

// validateField validates a single field based on its validate tag.
func (v *defaultValidator) validateField(field reflect.Value, fieldName, tag string, errors ValidationErrors) {
	// Parse and execute each validation rule
	rules := parseTag(tag)

	// Check for omitempty first
	for _, rule := range rules {
		if rule.Name == "omitempty" {
			if isZeroValue(field) {
				return // Skip all other validators
			}
			break
		}
	}

	// Find each position to handle slice/array element validation
	eachIndex := -1
	for i, rule := range rules {
		if rule.Name == "each" {
			eachIndex = i
			break
		}
	}

	// Run validators before each on the field itself
	endIndex := len(rules)
	if eachIndex >= 0 {
		endIndex = eachIndex
	}

	for i := 0; i < endIndex; i++ {
		rule := rules[i]
		if rule.Name == "omitempty" {
			continue // Already handled above
		}

		v.mu.RLock()
		fn, exists := v.validators[rule.Name]
		v.mu.RUnlock()

		if !exists {
			errors.Add(fieldName, fmt.Sprintf("unknown validator: %s", rule.Name))
			continue
		}

		if err := fn(field, rule.Param); err != nil {
			errors.Add(fieldName, err.Error())
		}
	}

	// If each is present, validate each element with remaining validators
	if eachIndex >= 0 && eachIndex < len(rules)-1 {
		elementRules := rules[eachIndex+1:]
		v.validateElements(field, fieldName, elementRules, errors)
	}
}

// validateElements validates each element in a slice/array with the given rules.
func (v *defaultValidator) validateElements(field reflect.Value, fieldName string, rules []validationRule, errors ValidationErrors) {
	switch field.Kind() {
	case reflect.Slice, reflect.Array:
		for j := 0; j < field.Len(); j++ {
			elem := field.Index(j)
			elemName := fmt.Sprintf("%s[%d]", fieldName, j)

			// Handle pointer elements
			if elem.Kind() == reflect.Ptr && !elem.IsNil() {
				elem = elem.Elem()
			}

			// For struct elements, validate their fields
			if elem.Kind() == reflect.Struct {
				v.validateStruct(elem, elemName, errors)
			} else {
				// For primitive elements, apply the validation rules
				for _, rule := range rules {
					v.mu.RLock()
					fn, exists := v.validators[rule.Name]
					v.mu.RUnlock()

					if !exists {
						errors.Add(elemName, fmt.Sprintf("unknown validator: %s", rule.Name))
						continue
					}

					if err := fn(elem, rule.Param); err != nil {
						errors.Add(elemName, err.Error())
					}
				}
			}
		}
	case reflect.Map:
		// For maps, iterate over values
		for _, key := range field.MapKeys() {
			elem := field.MapIndex(key)
			elemName := fmt.Sprintf("%s[%v]", fieldName, key)

			if elem.Kind() == reflect.Struct {
				v.validateStruct(elem, elemName, errors)
			} else {
				for _, rule := range rules {
					v.mu.RLock()
					fn, exists := v.validators[rule.Name]
					v.mu.RUnlock()

					if !exists {
						errors.Add(elemName, fmt.Sprintf("unknown validator: %s", rule.Name))
						continue
					}

					if err := fn(elem, rule.Param); err != nil {
						errors.Add(elemName, err.Error())
					}
				}
			}
		}
	default:
		// Not a collection type, nothing to validate
	}
}

// validationRule represents a single validation rule parsed from a tag.
type validationRule struct {
	Name  string
	Param string
}

// parseTag parses a validate tag into individual rules.
// Example: "required,min=3,max=10" -> [{required ""} {min "3"} {max "10"}]
func parseTag(tag string) []validationRule {
	var rules []validationRule

	parts := strings.Split(tag, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Check for parameter (e.g., "min=3")
		if idx := strings.Index(part, "="); idx > 0 {
			rules = append(rules, validationRule{
				Name:  part[:idx],
				Param: part[idx+1:],
			})
		} else {
			rules = append(rules, validationRule{Name: part})
		}
	}

	return rules
}

// Register adds a custom validation function.
func (v *defaultValidator) Register(name string, fn ValidationFunc) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.validators[name] = fn
}

// registerBuiltins registers all built-in validation functions.
func (v *defaultValidator) registerBuiltins() {
	// required - field must not be zero value
	v.validators["required"] = func(value reflect.Value, tag string) error {
		if isZeroValue(value) {
			return fmt.Errorf("required")
		}
		return nil
	}

	// omitempty - skip validation if zero value (handled in validateField)
	v.validators["omitempty"] = func(value reflect.Value, tag string) error {
		return nil // Handled in validateField before calling validators
	}

	// min - minimum value for numbers, minimum length for strings/slices/arrays/maps
	v.validators["min"] = func(value reflect.Value, tag string) error {
		switch value.Kind() {
		case reflect.String:
			length, err := strconv.Atoi(tag)
			if err != nil {
				return fmt.Errorf("invalid min parameter: %s", tag)
			}
			if utf8.RuneCountInString(value.String()) < length {
				return fmt.Errorf("must be at least %d characters", length)
			}
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			target, err := strconv.ParseInt(tag, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid min parameter: %s", tag)
			}
			if value.Int() < target {
				return fmt.Errorf("must be at least %d", target)
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			target, err := strconv.ParseUint(tag, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid min parameter: %s", tag)
			}
			if value.Uint() < target {
				return fmt.Errorf("must be at least %d", target)
			}
		case reflect.Float32, reflect.Float64:
			target, err := strconv.ParseFloat(tag, 64)
			if err != nil {
				return fmt.Errorf("invalid min parameter: %s", tag)
			}
			if value.Float() < target {
				return fmt.Errorf("must be at least %v", target)
			}
		case reflect.Slice, reflect.Array, reflect.Map:
			length, err := strconv.Atoi(tag)
			if err != nil {
				return fmt.Errorf("invalid min parameter: %s", tag)
			}
			if value.Len() < length {
				return fmt.Errorf("must have at least %d items", length)
			}
		default:
			return fmt.Errorf("min not supported for type %s", value.Kind())
		}
		return nil
	}

	// max - maximum value for numbers, maximum length for strings/slices/arrays/maps
	v.validators["max"] = func(value reflect.Value, tag string) error {
		switch value.Kind() {
		case reflect.String:
			length, err := strconv.Atoi(tag)
			if err != nil {
				return fmt.Errorf("invalid max parameter: %s", tag)
			}
			if utf8.RuneCountInString(value.String()) > length {
				return fmt.Errorf("must be at most %d characters", length)
			}
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			target, err := strconv.ParseInt(tag, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid max parameter: %s", tag)
			}
			if value.Int() > target {
				return fmt.Errorf("must be at most %d", target)
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			target, err := strconv.ParseUint(tag, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid max parameter: %s", tag)
			}
			if value.Uint() > target {
				return fmt.Errorf("must be at most %d", target)
			}
		case reflect.Float32, reflect.Float64:
			target, err := strconv.ParseFloat(tag, 64)
			if err != nil {
				return fmt.Errorf("invalid max parameter: %s", tag)
			}
			if value.Float() > target {
				return fmt.Errorf("must be at most %v", target)
			}
		case reflect.Slice, reflect.Array, reflect.Map:
			length, err := strconv.Atoi(tag)
			if err != nil {
				return fmt.Errorf("invalid max parameter: %s", tag)
			}
			if value.Len() > length {
				return fmt.Errorf("must have at most %d items", length)
			}
		default:
			return fmt.Errorf("max not supported for type %s", value.Kind())
		}
		return nil
	}

	// len - exact length for strings/slices/arrays
	v.validators["len"] = func(value reflect.Value, tag string) error {
		length, err := strconv.Atoi(tag)
		if err != nil {
			return fmt.Errorf("invalid len parameter: %s", tag)
		}

		switch value.Kind() {
		case reflect.String:
			if utf8.RuneCountInString(value.String()) != length {
				return fmt.Errorf("must be exactly %d characters", length)
			}
		case reflect.Slice, reflect.Array, reflect.Map:
			if value.Len() != length {
				return fmt.Errorf("must have exactly %d items", length)
			}
		default:
			return fmt.Errorf("len not supported for type %s", value.Kind())
		}
		return nil
	}

	// email - validates email address format
	v.validators["email"] = func(value reflect.Value, tag string) error {
		if value.Kind() != reflect.String {
			return fmt.Errorf("email validator only supports strings")
		}
		email := value.String()
		if email == "" {
			return nil // Let "required" handle empty strings
		}
		_, err := mail.ParseAddress(email)
		if err != nil {
			return fmt.Errorf("invalid email format")
		}
		return nil
	}

	// url - validates URL format
	v.validators["url"] = func(value reflect.Value, tag string) error {
		if value.Kind() != reflect.String {
			return fmt.Errorf("url validator only supports strings")
		}
		urlStr := value.String()
		if urlStr == "" {
			return nil // Let "required" handle empty strings
		}
		// Simple URL validation
		if !strings.HasPrefix(urlStr, "http://") && !strings.HasPrefix(urlStr, "https://") {
			return fmt.Errorf("invalid URL format")
		}
		return nil
	}

	// alpha - only alphabetic characters
	v.validators["alpha"] = func(value reflect.Value, tag string) error {
		if value.Kind() != reflect.String {
			return fmt.Errorf("alpha validator only supports strings")
		}
		s := value.String()
		if s == "" {
			return nil
		}
		for _, r := range s {
			if !unicode.IsLetter(r) {
				return fmt.Errorf("must contain only letters")
			}
		}
		return nil
	}

	// alphanum - only alphanumeric characters
	v.validators["alphanum"] = func(value reflect.Value, tag string) error {
		if value.Kind() != reflect.String {
			return fmt.Errorf("alphanum validator only supports strings")
		}
		s := value.String()
		if s == "" {
			return nil
		}
		for _, r := range s {
			if !unicode.IsLetter(r) && !unicode.IsNumber(r) {
				return fmt.Errorf("must contain only letters and numbers")
			}
		}
		return nil
	}

	// numeric - only numeric characters
	v.validators["numeric"] = func(value reflect.Value, tag string) error {
		if value.Kind() != reflect.String {
			return fmt.Errorf("numeric validator only supports strings")
		}
		s := value.String()
		if s == "" {
			return nil
		}
		for _, r := range s {
			if !unicode.IsNumber(r) {
				return fmt.Errorf("must contain only numbers")
			}
		}
		return nil
	}

	// oneof - value must be one of the specified options
	v.validators["oneof"] = func(value reflect.Value, tag string) error {
		options := strings.Fields(tag)
		if len(options) == 0 {
			return nil
		}

		switch value.Kind() {
		case reflect.String:
			s := value.String()
			if s == "" {
				return nil
			}
			for _, opt := range options {
				if s == opt {
					return nil
				}
			}
			return fmt.Errorf("must be one of: %s", strings.Join(options, ", "))
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			n := value.Int()
			for _, opt := range options {
				if i, err := strconv.ParseInt(opt, 10, 64); err == nil && n == i {
					return nil
				}
			}
			return fmt.Errorf("must be one of: %s", tag)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			n := value.Uint()
			for _, opt := range options {
				if i, err := strconv.ParseUint(opt, 10, 64); err == nil && n == i {
					return nil
				}
			}
			return fmt.Errorf("must be one of: %s", tag)
		default:
			return fmt.Errorf("oneof not supported for type %s", value.Kind())
		}
	}

	// eq - equal to value
	v.validators["eq"] = func(value reflect.Value, tag string) error {
		switch value.Kind() {
		case reflect.String:
			if value.String() != tag {
				return fmt.Errorf("must be equal to %s", tag)
			}
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			n, err := strconv.ParseInt(tag, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid eq parameter: %s", tag)
			}
			if value.Int() != n {
				return fmt.Errorf("must be equal to %d", n)
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			n, err := strconv.ParseUint(tag, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid eq parameter: %s", tag)
			}
			if value.Uint() != n {
				return fmt.Errorf("must be equal to %d", n)
			}
		case reflect.Float32, reflect.Float64:
			n, err := strconv.ParseFloat(tag, 64)
			if err != nil {
				return fmt.Errorf("invalid eq parameter: %s", tag)
			}
			if value.Float() != n {
				return fmt.Errorf("must be equal to %v", n)
			}
		default:
			return fmt.Errorf("eq not supported for type %s", value.Kind())
		}
		return nil
	}

	// ne - not equal to value
	v.validators["ne"] = func(value reflect.Value, tag string) error {
		switch value.Kind() {
		case reflect.String:
			if value.String() == tag {
				return fmt.Errorf("must not be equal to %s", tag)
			}
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			n, err := strconv.ParseInt(tag, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid ne parameter: %s", tag)
			}
			if value.Int() == n {
				return fmt.Errorf("must not be equal to %d", n)
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			n, err := strconv.ParseUint(tag, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid ne parameter: %s", tag)
			}
			if value.Uint() == n {
				return fmt.Errorf("must not be equal to %d", n)
			}
		case reflect.Float32, reflect.Float64:
			n, err := strconv.ParseFloat(tag, 64)
			if err != nil {
				return fmt.Errorf("invalid ne parameter: %s", tag)
			}
			if value.Float() == n {
				return fmt.Errorf("must not be equal to %v", n)
			}
		default:
			return fmt.Errorf("ne not supported for type %s", value.Kind())
		}
		return nil
	}

	// gt - greater than
	v.validators["gt"] = func(value reflect.Value, tag string) error {
		switch value.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			target, err := strconv.ParseInt(tag, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid gt parameter: %s", tag)
			}
			if value.Int() <= target {
				return fmt.Errorf("must be greater than %d", target)
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			target, err := strconv.ParseUint(tag, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid gt parameter: %s", tag)
			}
			if value.Uint() <= target {
				return fmt.Errorf("must be greater than %d", target)
			}
		case reflect.Float32, reflect.Float64:
			target, err := strconv.ParseFloat(tag, 64)
			if err != nil {
				return fmt.Errorf("invalid gt parameter: %s", tag)
			}
			if value.Float() <= target {
				return fmt.Errorf("must be greater than %v", target)
			}
		default:
			return fmt.Errorf("gt not supported for type %s", value.Kind())
		}
		return nil
	}

	// lt - less than
	v.validators["lt"] = func(value reflect.Value, tag string) error {
		switch value.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			target, err := strconv.ParseInt(tag, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid lt parameter: %s", tag)
			}
			if value.Int() >= target {
				return fmt.Errorf("must be less than %d", target)
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			target, err := strconv.ParseUint(tag, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid lt parameter: %s", tag)
			}
			if value.Uint() >= target {
				return fmt.Errorf("must be less than %d", target)
			}
		case reflect.Float32, reflect.Float64:
			target, err := strconv.ParseFloat(tag, 64)
			if err != nil {
				return fmt.Errorf("invalid lt parameter: %s", tag)
			}
			if value.Float() >= target {
				return fmt.Errorf("must be less than %v", target)
			}
		default:
			return fmt.Errorf("lt not supported for type %s", value.Kind())
		}
		return nil
	}

	// gte - greater than or equal
	v.validators["gte"] = func(value reflect.Value, tag string) error {
		switch value.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			target, err := strconv.ParseInt(tag, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid gte parameter: %s", tag)
			}
			if value.Int() < target {
				return fmt.Errorf("must be greater than or equal to %d", target)
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			target, err := strconv.ParseUint(tag, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid gte parameter: %s", tag)
			}
			if value.Uint() < target {
				return fmt.Errorf("must be greater than or equal to %d", target)
			}
		case reflect.Float32, reflect.Float64:
			target, err := strconv.ParseFloat(tag, 64)
			if err != nil {
				return fmt.Errorf("invalid gte parameter: %s", tag)
			}
			if value.Float() < target {
				return fmt.Errorf("must be greater than or equal to %v", target)
			}
		default:
			return fmt.Errorf("gte not supported for type %s", value.Kind())
		}
		return nil
	}

	// lte - less than or equal
	v.validators["lte"] = func(value reflect.Value, tag string) error {
		switch value.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			target, err := strconv.ParseInt(tag, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid lte parameter: %s", tag)
			}
			if value.Int() > target {
				return fmt.Errorf("must be less than or equal to %d", target)
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			target, err := strconv.ParseUint(tag, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid lte parameter: %s", tag)
			}
			if value.Uint() > target {
				return fmt.Errorf("must be less than or equal to %d", target)
			}
		case reflect.Float32, reflect.Float64:
			target, err := strconv.ParseFloat(tag, 64)
			if err != nil {
				return fmt.Errorf("invalid lte parameter: %s", tag)
			}
			if value.Float() > target {
				return fmt.Errorf("must be less than or equal to %v", target)
			}
		default:
			return fmt.Errorf("lte not supported for type %s", value.Kind())
		}
		return nil
	}

	// uuid - validates UUID format
	v.validators["uuid"] = func(value reflect.Value, tag string) error {
		if value.Kind() != reflect.String {
			return fmt.Errorf("uuid validator only supports strings")
		}
		uuid := value.String()
		if uuid == "" {
			return nil
		}
		uuidRegex := regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
		if !uuidRegex.MatchString(uuid) {
			return fmt.Errorf("invalid UUID format")
		}
		return nil
	}

	// datetime - validates datetime format
	v.validators["datetime"] = func(value reflect.Value, tag string) error {
		if value.Kind() != reflect.String {
			return fmt.Errorf("datetime validator only supports strings")
		}
		dt := value.String()
		if dt == "" {
			return nil
		}
		// Default to RFC3339 if no format specified
		format := tag
		if format == "" {
			format = time.RFC3339
		}
		_, err := time.Parse(format, dt)
		if err != nil {
			return fmt.Errorf("invalid datetime format, expected %s", format)
		}
		return nil
	}

	// ===== String Content Validators =====

	// contains - must contain substring
	v.validators["contains"] = func(value reflect.Value, tag string) error {
		if value.Kind() != reflect.String {
			return fmt.Errorf("contains validator only supports strings")
		}
		s := value.String()
		if s == "" {
			return nil
		}
		if !strings.Contains(s, tag) {
			return fmt.Errorf("must contain %s", tag)
		}
		return nil
	}

	// startswith - must start with prefix
	v.validators["startswith"] = func(value reflect.Value, tag string) error {
		if value.Kind() != reflect.String {
			return fmt.Errorf("startswith validator only supports strings")
		}
		s := value.String()
		if s == "" {
			return nil
		}
		if !strings.HasPrefix(s, tag) {
			return fmt.Errorf("must start with %s", tag)
		}
		return nil
	}

	// endswith - must end with suffix
	v.validators["endswith"] = func(value reflect.Value, tag string) error {
		if value.Kind() != reflect.String {
			return fmt.Errorf("endswith validator only supports strings")
		}
		s := value.String()
		if s == "" {
			return nil
		}
		if !strings.HasSuffix(s, tag) {
			return fmt.Errorf("must end with %s", tag)
		}
		return nil
	}

	// excludes - must not contain substring
	v.validators["excludes"] = func(value reflect.Value, tag string) error {
		if value.Kind() != reflect.String {
			return fmt.Errorf("excludes validator only supports strings")
		}
		s := value.String()
		if s == "" {
			return nil
		}
		if strings.Contains(s, tag) {
			return fmt.Errorf("must not contain %s", tag)
		}
		return nil
	}

	// lowercase - must be all lowercase
	v.validators["lowercase"] = func(value reflect.Value, tag string) error {
		if value.Kind() != reflect.String {
			return fmt.Errorf("lowercase validator only supports strings")
		}
		s := value.String()
		if s == "" {
			return nil
		}
		if s != strings.ToLower(s) {
			return fmt.Errorf("must be lowercase")
		}
		return nil
	}

	// uppercase - must be all uppercase
	v.validators["uppercase"] = func(value reflect.Value, tag string) error {
		if value.Kind() != reflect.String {
			return fmt.Errorf("uppercase validator only supports strings")
		}
		s := value.String()
		if s == "" {
			return nil
		}
		if s != strings.ToUpper(s) {
			return fmt.Errorf("must be uppercase")
		}
		return nil
	}

	// ascii - ASCII characters only
	v.validators["ascii"] = func(value reflect.Value, tag string) error {
		if value.Kind() != reflect.String {
			return fmt.Errorf("ascii validator only supports strings")
		}
		s := value.String()
		if s == "" {
			return nil
		}
		for _, r := range s {
			if r > 127 {
				return fmt.Errorf("must contain only ASCII characters")
			}
		}
		return nil
	}

	// printascii - printable ASCII only
	v.validators["printascii"] = func(value reflect.Value, tag string) error {
		if value.Kind() != reflect.String {
			return fmt.Errorf("printascii validator only supports strings")
		}
		s := value.String()
		if s == "" {
			return nil
		}
		for _, r := range s {
			if r < 32 || r > 126 {
				return fmt.Errorf("must contain only printable ASCII characters")
			}
		}
		return nil
	}

	// boolean - parseable boolean string
	v.validators["boolean"] = func(value reflect.Value, tag string) error {
		if value.Kind() != reflect.String {
			return fmt.Errorf("boolean validator only supports strings")
		}
		s := strings.ToLower(value.String())
		if s == "" {
			return nil
		}
		valid := []string{"true", "false", "1", "0", "yes", "no", "on", "off"}
		for _, v := range valid {
			if s == v {
				return nil
			}
		}
		return fmt.Errorf("must be a valid boolean value")
	}

	// json - valid JSON string
	v.validators["json"] = func(value reflect.Value, tag string) error {
		if value.Kind() != reflect.String {
			return fmt.Errorf("json validator only supports strings")
		}
		s := value.String()
		if s == "" {
			return nil
		}
		var js any
		if err := json.Unmarshal([]byte(s), &js); err != nil {
			return fmt.Errorf("must be valid JSON")
		}
		return nil
	}

	// ===== Format / Network Validators =====

	// ip - valid IP address (v4 or v6)
	v.validators["ip"] = func(value reflect.Value, tag string) error {
		if value.Kind() != reflect.String {
			return fmt.Errorf("ip validator only supports strings")
		}
		s := value.String()
		if s == "" {
			return nil
		}
		if net.ParseIP(s) == nil {
			return fmt.Errorf("must be a valid IP address")
		}
		return nil
	}

	// ipv4 - valid IPv4 address
	v.validators["ipv4"] = func(value reflect.Value, tag string) error {
		if value.Kind() != reflect.String {
			return fmt.Errorf("ipv4 validator only supports strings")
		}
		s := value.String()
		if s == "" {
			return nil
		}
		ip := net.ParseIP(s)
		if ip == nil || ip.To4() == nil {
			return fmt.Errorf("must be a valid IPv4 address")
		}
		return nil
	}

	// ipv6 - valid IPv6 address
	v.validators["ipv6"] = func(value reflect.Value, tag string) error {
		if value.Kind() != reflect.String {
			return fmt.Errorf("ipv6 validator only supports strings")
		}
		s := value.String()
		if s == "" {
			return nil
		}
		ip := net.ParseIP(s)
		if ip == nil || ip.To4() != nil {
			return fmt.Errorf("must be a valid IPv6 address")
		}
		return nil
	}

	// cidr - valid CIDR notation
	v.validators["cidr"] = func(value reflect.Value, tag string) error {
		if value.Kind() != reflect.String {
			return fmt.Errorf("cidr validator only supports strings")
		}
		s := value.String()
		if s == "" {
			return nil
		}
		_, _, err := net.ParseCIDR(s)
		if err != nil {
			return fmt.Errorf("must be valid CIDR notation")
		}
		return nil
	}

	// hostname - valid hostname (RFC 1123)
	v.validators["hostname"] = func(value reflect.Value, tag string) error {
		if value.Kind() != reflect.String {
			return fmt.Errorf("hostname validator only supports strings")
		}
		s := value.String()
		if s == "" {
			return nil
		}
		// RFC 1123 hostname regex
		hostnameRegex := regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?)*$`)
		if !hostnameRegex.MatchString(s) || len(s) > 253 {
			return fmt.Errorf("must be a valid hostname")
		}
		return nil
	}

	// uri - valid URI (any scheme)
	v.validators["uri"] = func(value reflect.Value, tag string) error {
		if value.Kind() != reflect.String {
			return fmt.Errorf("uri validator only supports strings")
		}
		s := value.String()
		if s == "" {
			return nil
		}
		u, err := url.Parse(s)
		if err != nil || u.Scheme == "" || u.Host == "" {
			return fmt.Errorf("must be a valid URI")
		}
		return nil
	}

	// base64 - valid base64 string
	v.validators["base64"] = func(value reflect.Value, tag string) error {
		if value.Kind() != reflect.String {
			return fmt.Errorf("base64 validator only supports strings")
		}
		s := value.String()
		if s == "" {
			return nil
		}
		if _, err := base64.StdEncoding.DecodeString(s); err != nil {
			return fmt.Errorf("must be valid base64")
		}
		return nil
	}

	// hexadecimal - valid hex string
	v.validators["hexadecimal"] = func(value reflect.Value, tag string) error {
		if value.Kind() != reflect.String {
			return fmt.Errorf("hexadecimal validator only supports strings")
		}
		s := value.String()
		if s == "" {
			return nil
		}
		hexRegex := regexp.MustCompile(`^[0-9a-fA-F]+$`)
		if !hexRegex.MatchString(s) {
			return fmt.Errorf("must be valid hexadecimal")
		}
		return nil
	}

	// hexcolor - valid hex color code
	v.validators["hexcolor"] = func(value reflect.Value, tag string) error {
		if value.Kind() != reflect.String {
			return fmt.Errorf("hexcolor validator only supports strings")
		}
		s := value.String()
		if s == "" {
			return nil
		}
		// Support #RGB, #RGBA, #RRGGBB, #RRGGBBAA
		hexColorRegex := regexp.MustCompile(`^#([0-9a-fA-F]{3}|[0-9a-fA-F]{4}|[0-9a-fA-F]{6}|[0-9a-fA-F]{8})$`)
		if !hexColorRegex.MatchString(s) {
			return fmt.Errorf("must be valid hex color code")
		}
		return nil
	}

	// e164 - E.164 phone number format
	v.validators["e164"] = func(value reflect.Value, tag string) error {
		if value.Kind() != reflect.String {
			return fmt.Errorf("e164 validator only supports strings")
		}
		s := value.String()
		if s == "" {
			return nil
		}
		// E.164: + followed by 1-15 digits
		e164Regex := regexp.MustCompile(`^\+[1-9]\d{1,14}$`)
		if !e164Regex.MatchString(s) {
			return fmt.Errorf("must be valid E.164 phone number")
		}
		return nil
	}

	// semver - semantic version string
	v.validators["semver"] = func(value reflect.Value, tag string) error {
		if value.Kind() != reflect.String {
			return fmt.Errorf("semver validator only supports strings")
		}
		s := value.String()
		if s == "" {
			return nil
		}
		// Semver regex (simplified)
		semverRegex := regexp.MustCompile(`^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$`)
		if !semverRegex.MatchString(s) {
			return fmt.Errorf("must be valid semantic version")
		}
		return nil
	}

	// jwt - valid JWT format (header.payload.signature)
	v.validators["jwt"] = func(value reflect.Value, tag string) error {
		if value.Kind() != reflect.String {
			return fmt.Errorf("jwt validator only supports strings")
		}
		s := value.String()
		if s == "" {
			return nil
		}
		parts := strings.Split(s, ".")
		if len(parts) != 3 {
			return fmt.Errorf("must be valid JWT format")
		}
		// Check each part is valid base64
		for _, part := range parts {
			if _, err := base64.RawURLEncoding.DecodeString(part); err != nil {
				return fmt.Errorf("must be valid JWT format")
			}
		}
		return nil
	}

	// ===== Collection Validators =====

	// unique - all elements must be unique
	v.validators["unique"] = func(value reflect.Value, tag string) error {
		switch value.Kind() {
		case reflect.Slice, reflect.Array:
			seen := make(map[any]struct{})
			for i := 0; i < value.Len(); i++ {
				elem := value.Index(i).Interface()
				if _, exists := seen[elem]; exists {
					return fmt.Errorf("must have unique elements")
				}
				seen[elem] = struct{}{}
			}
			return nil
		case reflect.Map:
			// Maps always have unique keys by definition
			return nil
		default:
			return fmt.Errorf("unique not supported for type %s", value.Kind())
		}
	}

	// each - validate each element in collection (handled in validateField)
	v.validators["each"] = func(value reflect.Value, tag string) error {
		// each is a special marker that tells validateField to recurse into elements
		// The actual validation is handled in validateField
		return nil
	}
}

// isZeroValue checks if a value is its zero value.
func isZeroValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.String:
		return v.String() == ""
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Slice, reflect.Map, reflect.Array:
		return v.Len() == 0
	case reflect.Ptr, reflect.Interface:
		return v.IsNil()
	case reflect.Struct:
		// Check if struct is zero by comparing with a new instance
		return reflect.DeepEqual(v.Interface(), reflect.Zero(v.Type()).Interface())
	default:
		return false
	}
}

// getJSONFieldName returns the json tag name if available, otherwise the struct field name.
// It handles json tags with options like `json:"name,omitempty"` by stripping the options.
func getJSONFieldName(field reflect.StructField) string {
	jsonTag := field.Tag.Get("json")
	if jsonTag == "" || jsonTag == "-" {
		return field.Name
	}

	// Handle json tags with options (e.g., "name,omitempty")
	if idx := strings.Index(jsonTag, ","); idx >= 0 {
		jsonTag = jsonTag[:idx]
	}

	// If the json tag is empty after stripping options, use field name
	if jsonTag == "" {
		return field.Name
	}

	return jsonTag
}
