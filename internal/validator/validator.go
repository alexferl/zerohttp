package validator

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"sync/atomic"
)

var (
	uuidRegex = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	// RFC 1123 hostname regex
	hostnameRegex = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?)*$`)
	hexRegex      = regexp.MustCompile(`^[0-9a-fA-F]+$`)
	// Support #RGB, #RGBA, #RRGGBB, #RRGGBBAA
	hexColorRegex = regexp.MustCompile(`^#([0-9a-fA-F]{3}|[0-9a-fA-F]{4}|[0-9a-fA-F]{6}|[0-9a-fA-F]{8})$`)
	// E.164: + followed by 1-15 digits
	e164Regex = regexp.MustCompile(`^\+[1-9]\d{1,14}$`)
	// Semver regex (simplified)
	semverRegex = regexp.MustCompile(`^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$`)
)

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

// Ensure defaultValidator implements Validator
var _ Validator = (*defaultValidator)(nil)

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

// defaultValidator implements the Validator interface
type defaultValidator struct {
	validators atomic.Value // stores map[string]ValidationFunc
}

// NewValidator creates a new validator instance with built-in validation rules.
func NewValidator() Validator {
	v := &defaultValidator{}
	v.validators.Store(make(map[string]ValidationFunc))
	v.registerBuiltins()
	return v
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
	// Get cached type info
	info := ValidatorRegistry.GetTypeInfo(val.Type())

	// Check if the struct itself implements Validate() error
	// This allows custom validation methods on the struct type
	if info.hasCustomValidate && val.CanAddr() {
		if err := val.Addr().Interface().(interface{ Validate() error }).Validate(); err != nil {
			// Use the prefix if available, otherwise use the struct type name
			key := prefix
			if key == "" {
				key = val.Type().Name()
			}
			errors.Add(key, err.Error())
		}
	}

	for i := range info.fields {
		fi := &info.fields[i]

		// Get the field value using the cached index
		field := val.Field(fi.index)

		// Handle embedded structs - validate them recursively
		if fi.isEmbedded {
			v.validateStruct(field, prefix, errors)
			continue
		}

		// Use cached JSON field name
		fieldName := fi.name
		if prefix != "" {
			fieldName = prefix + "." + fieldName
		}

		// Handle pointer fields
		if fi.isPtr {
			if field.IsNil() {
				// Check if this is a required pointer field using cached info
				if fi.hasRequired {
					errors.Add(fieldName, "required")
				}
				continue
			}
			// Dereference and validate the pointed-to value
			field = field.Elem()
		}

		// Validate field if it has rules
		if len(fi.rules) > 0 {
			v.validateFieldWithInfo(field, fieldName, fi, errors)
		}

		// Recursively validate nested structs
		switch {
		case fi.isStruct && !fi.isTimeTime:
			v.validateStruct(field, fieldName, errors)
		case fi.isSlice, fi.isArray:
			for j := 0; j < field.Len(); j++ {
				elem := field.Index(j)
				elemName := fmt.Sprintf("%s[%d]", fieldName, j)
				if elem.Kind() == reflect.Struct {
					v.validateStruct(elem, elemName, errors)
				} else if elem.Kind() == reflect.Ptr && !elem.IsNil() && elem.Elem().Kind() == reflect.Struct {
					v.validateStruct(elem.Elem(), elemName, errors)
				}
			}
		case fi.isMap:
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

// validateFieldWithInfo validates a single field using pre-parsed field info.
func (v *defaultValidator) validateFieldWithInfo(field reflect.Value, fieldName string, fi *validatedFieldInfo, errors ValidationErrors) {
	// Check for omitempty first using cached value
	if fi.omitempty && isZeroValue(field) {
		return // Skip all other validators
	}

	// Run validators before each on the field itself
	endIndex := len(fi.rules)
	if fi.eachIndex >= 0 {
		endIndex = fi.eachIndex
	}

	validators := v.validators.Load().(map[string]ValidationFunc)

	for i := 0; i < endIndex; i++ {
		rule := fi.rules[i]
		if rule.Name == "omitempty" {
			continue // Already handled above
		}

		fn, exists := validators[rule.Name]

		if !exists {
			errors.Add(fieldName, fmt.Sprintf("unknown validator: %s", rule.Name))
			continue
		}

		if err := fn(field, rule.Param); err != nil {
			errors.Add(fieldName, err.Error())
		}
	}

	// If each is present, validate each element with remaining validators
	if fi.eachIndex >= 0 && fi.eachIndex < len(fi.rules)-1 {
		elementRules := fi.rules[fi.eachIndex+1:]
		v.validateElements(field, fieldName, elementRules, errors)
	}
}

// validateElements validates each element in a slice/array with the given rules.
func (v *defaultValidator) validateElements(field reflect.Value, fieldName string, rules []validationRule, errors ValidationErrors) {
	validators := v.validators.Load().(map[string]ValidationFunc)

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
					fn, exists := validators[rule.Name]

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
					fn, exists := validators[rule.Name]

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
	old := v.validators.Load().(map[string]ValidationFunc)
	newMap := make(map[string]ValidationFunc, len(old)+1)
	for k, v := range old {
		newMap[k] = v
	}
	newMap[name] = fn
	v.validators.Store(newMap)
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
