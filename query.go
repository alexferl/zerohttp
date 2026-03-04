package zerohttp

import (
	"fmt"
	"net/http"
	"strconv"
)

// Query is the default query extractor instance used by the package
var Query = &defaultQueryExtractor{}

// QueryExtractor defines the interface for query parameter extraction
type QueryExtractor interface {
	// QueryParam gets a query parameter by name as a string.
	// Returns empty string if parameter is not found.
	QueryParam(r *http.Request, name string) string

	// QueryParamOrDefault gets a query parameter with a fallback default value.
	QueryParamOrDefault(r *http.Request, name, defaultVal string) string
}

// defaultQueryExtractor implements the QueryExtractor interface
type defaultQueryExtractor struct{}

// QueryParam gets a query parameter by name as a string.
// Returns empty string if parameter is not found.
func (q *defaultQueryExtractor) QueryParam(r *http.Request, name string) string {
	return r.URL.Query().Get(name)
}

// QueryParamOrDefault gets a query parameter with a fallback default value.
func (q *defaultQueryExtractor) QueryParamOrDefault(r *http.Request, name, defaultVal string) string {
	val := r.URL.Query().Get(name)
	if val == "" {
		return defaultVal
	}
	return val
}

// QueryParamAs extracts and converts a query parameter to type T.
// Returns an error if conversion fails.
// For missing parameters, returns the zero value and no error (use pointer types for optional params).
// Supported types: string, int, int8, int16, int32, int64,
// uint, uint8, uint16, uint32, uint64, float32, float64, bool
func QueryParamAs[T ParamType](r *http.Request, name string) (T, error) {
	var zero T
	val := Query.QueryParam(r, name)
	if val == "" {
		return zero, nil
	}

	switch any(zero).(type) {
	case string:
		return any(val).(T), nil
	case int:
		n, err := strconv.Atoi(val)
		if err != nil {
			return zero, fmt.Errorf("query param %q: invalid int: %w", name, err)
		}
		return any(n).(T), nil
	case int8:
		n, err := strconv.ParseInt(val, 10, 8)
		if err != nil {
			return zero, fmt.Errorf("query param %q: invalid int8: %w", name, err)
		}
		return any(int8(n)).(T), nil
	case int16:
		n, err := strconv.ParseInt(val, 10, 16)
		if err != nil {
			return zero, fmt.Errorf("query param %q: invalid int16: %w", name, err)
		}
		return any(int16(n)).(T), nil
	case int32:
		n, err := strconv.ParseInt(val, 10, 32)
		if err != nil {
			return zero, fmt.Errorf("query param %q: invalid int32: %w", name, err)
		}
		return any(int32(n)).(T), nil
	case int64:
		n, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return zero, fmt.Errorf("query param %q: invalid int64: %w", name, err)
		}
		return any(n).(T), nil
	case uint:
		n, err := strconv.ParseUint(val, 10, 0)
		if err != nil {
			return zero, fmt.Errorf("query param %q: invalid uint: %w", name, err)
		}
		return any(uint(n)).(T), nil
	case uint8:
		n, err := strconv.ParseUint(val, 10, 8)
		if err != nil {
			return zero, fmt.Errorf("query param %q: invalid uint8: %w", name, err)
		}
		return any(uint8(n)).(T), nil
	case uint16:
		n, err := strconv.ParseUint(val, 10, 16)
		if err != nil {
			return zero, fmt.Errorf("query param %q: invalid uint16: %w", name, err)
		}
		return any(uint16(n)).(T), nil
	case uint32:
		n, err := strconv.ParseUint(val, 10, 32)
		if err != nil {
			return zero, fmt.Errorf("query param %q: invalid uint32: %w", name, err)
		}
		return any(uint32(n)).(T), nil
	case uint64:
		n, err := strconv.ParseUint(val, 10, 64)
		if err != nil {
			return zero, fmt.Errorf("query param %q: invalid uint64: %w", name, err)
		}
		return any(n).(T), nil
	case float32:
		n, err := strconv.ParseFloat(val, 32)
		if err != nil {
			return zero, fmt.Errorf("query param %q: invalid float32: %w", name, err)
		}
		return any(float32(n)).(T), nil
	case float64:
		n, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return zero, fmt.Errorf("query param %q: invalid float64: %w", name, err)
		}
		return any(n).(T), nil
	case bool:
		n, err := strconv.ParseBool(val)
		if err != nil {
			return zero, fmt.Errorf("query param %q: invalid bool: %w", name, err)
		}
		return any(n).(T), nil
	default:
		return zero, fmt.Errorf("query param %q: unsupported type", name)
	}
}

// QueryParamAsOrDefault extracts and converts a query parameter to type T,
// returning a default value if the parameter is missing or conversion fails.
func QueryParamAsOrDefault[T ParamType](r *http.Request, name string, defaultVal T) T {
	val, err := QueryParamAs[T](r, name)
	if err != nil {
		return defaultVal
	}
	// Check for zero value (missing param) and return default
	var zero T
	if any(val) == any(zero) {
		// Check if param actually exists
		if Query.QueryParam(r, name) == "" {
			return defaultVal
		}
	}
	return val
}

// QueryParam is a convenience function that calls Query.QueryParam
func QueryParam(r *http.Request, name string) string {
	return Query.QueryParam(r, name)
}

// QueryParamOrDefault is a convenience function that calls Query.QueryParamOrDefault
func QueryParamOrDefault(r *http.Request, name, defaultVal string) string {
	return Query.QueryParamOrDefault(r, name, defaultVal)
}
