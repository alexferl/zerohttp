package zerohttp

import (
	"fmt"
	"net/http"
	"strconv"
)

// Params is the default params extractor instance used by the package
var Params = &defaultParamsExtractor{}

// ParamType is a type constraint for supported path parameter types.
// Supported types: string, int, int8, int16, int32, int64,
// uint, uint8, uint16, uint32, uint64, float32, float64, bool
type ParamType interface {
	~string | ~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 |
		~float32 | ~float64 |
		~bool
}

// ParamExtractor defines the interface for path parameter extraction
type ParamExtractor interface {
	// Param gets a path parameter by name as a string.
	// Returns empty string if parameter is not found.
	Param(r *http.Request, name string) string

	// ParamOrDefault gets a path parameter with a fallback default value.
	ParamOrDefault(r *http.Request, name, defaultVal string) string
}

// Ensure defaultParamsExtractor implements ParamExtractor
var _ ParamExtractor = (*defaultParamsExtractor)(nil)

// defaultParamsExtractor implements the ParamExtractor interface
type defaultParamsExtractor struct{}

// Param gets a path parameter by name as a string.
// Returns empty string if parameter is not found.
func (p *defaultParamsExtractor) Param(r *http.Request, name string) string {
	return r.PathValue(name)
}

// ParamOrDefault gets a path parameter with a fallback default value.
func (p *defaultParamsExtractor) ParamOrDefault(r *http.Request, name, defaultVal string) string {
	val := r.PathValue(name)
	if val == "" {
		return defaultVal
	}
	return val
}

// ParamAs extracts and converts a path parameter to type T.
// Returns an error if the parameter is missing or conversion fails.
// Supported types: string, int, int8, int16, int32, int64,
// uint, uint8, uint16, uint32, uint64, float32, float64, bool
func ParamAs[T ParamType](r *http.Request, name string) (T, error) {
	var zero T
	val := Params.Param(r, name)
	if val == "" {
		return zero, fmt.Errorf("parameter %q not found", name)
	}

	switch any(zero).(type) {
	case string:
		return any(val).(T), nil
	case int:
		n, err := strconv.Atoi(val)
		if err != nil {
			return zero, fmt.Errorf("parameter %q: invalid int: %w", name, err)
		}
		return any(n).(T), nil
	case int8:
		n, err := strconv.ParseInt(val, 10, 8)
		if err != nil {
			return zero, fmt.Errorf("parameter %q: invalid int8: %w", name, err)
		}
		return any(int8(n)).(T), nil
	case int16:
		n, err := strconv.ParseInt(val, 10, 16)
		if err != nil {
			return zero, fmt.Errorf("parameter %q: invalid int16: %w", name, err)
		}
		return any(int16(n)).(T), nil
	case int32:
		n, err := strconv.ParseInt(val, 10, 32)
		if err != nil {
			return zero, fmt.Errorf("parameter %q: invalid int32: %w", name, err)
		}
		return any(int32(n)).(T), nil
	case int64:
		n, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return zero, fmt.Errorf("parameter %q: invalid int64: %w", name, err)
		}
		return any(n).(T), nil
	case uint:
		n, err := strconv.ParseUint(val, 10, 0)
		if err != nil {
			return zero, fmt.Errorf("parameter %q: invalid uint: %w", name, err)
		}
		return any(uint(n)).(T), nil
	case uint8:
		n, err := strconv.ParseUint(val, 10, 8)
		if err != nil {
			return zero, fmt.Errorf("parameter %q: invalid uint8: %w", name, err)
		}
		return any(uint8(n)).(T), nil
	case uint16:
		n, err := strconv.ParseUint(val, 10, 16)
		if err != nil {
			return zero, fmt.Errorf("parameter %q: invalid uint16: %w", name, err)
		}
		return any(uint16(n)).(T), nil
	case uint32:
		n, err := strconv.ParseUint(val, 10, 32)
		if err != nil {
			return zero, fmt.Errorf("parameter %q: invalid uint32: %w", name, err)
		}
		return any(uint32(n)).(T), nil
	case uint64:
		n, err := strconv.ParseUint(val, 10, 64)
		if err != nil {
			return zero, fmt.Errorf("parameter %q: invalid uint64: %w", name, err)
		}
		return any(n).(T), nil
	case float32:
		n, err := strconv.ParseFloat(val, 32)
		if err != nil {
			return zero, fmt.Errorf("parameter %q: invalid float32: %w", name, err)
		}
		return any(float32(n)).(T), nil
	case float64:
		n, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return zero, fmt.Errorf("parameter %q: invalid float64: %w", name, err)
		}
		return any(n).(T), nil
	case bool:
		n, err := strconv.ParseBool(val)
		if err != nil {
			return zero, fmt.Errorf("parameter %q: invalid bool: %w", name, err)
		}
		return any(n).(T), nil
	default:
		return zero, fmt.Errorf("parameter %q: unsupported type", name)
	}
}

// ParamAsOrDefault extracts and converts a path parameter to type T,
// returning a default value if the parameter is missing or conversion fails.
func ParamAsOrDefault[T ParamType](r *http.Request, name string, defaultVal T) T {
	val, err := ParamAs[T](r, name)
	if err != nil {
		return defaultVal
	}
	return val
}

// Param is a convenience function that calls Params.Param
func Param(r *http.Request, name string) string {
	return Params.Param(r, name)
}

// ParamOrDefault is a convenience function that calls Params.ParamOrDefault
func ParamOrDefault(r *http.Request, name, defaultVal string) string {
	return Params.ParamOrDefault(r, name, defaultVal)
}
