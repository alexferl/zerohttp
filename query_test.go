package zerohttp

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestQueryExtractor_QueryParam(t *testing.T) {
	tests := []struct {
		name        string
		query       string
		paramName   string
		expectedVal string
	}{
		{
			name:        "existing param",
			query:       "page=10&limit=20",
			paramName:   "page",
			expectedVal: "10",
		},
		{
			name:        "another existing param",
			query:       "page=10&limit=20",
			paramName:   "limit",
			expectedVal: "20",
		},
		{
			name:        "missing param",
			query:       "page=10",
			paramName:   "limit",
			expectedVal: "",
		},
		{
			name:        "empty query",
			query:       "",
			paramName:   "page",
			expectedVal: "",
		},
		{
			name:        "empty param value",
			query:       "page=",
			paramName:   "page",
			expectedVal: "",
		},
		{
			name:        "url encoded value",
			query:       "search=hello%20world",
			paramName:   "search",
			expectedVal: "hello world",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/?"+tt.query, nil)

			result := Query.QueryParam(req, tt.paramName)
			if result != tt.expectedVal {
				t.Errorf("QueryParam(%q) = %q, want %q", tt.paramName, result, tt.expectedVal)
			}

			// Also test convenience function
			result2 := QueryParam(req, tt.paramName)
			if result2 != tt.expectedVal {
				t.Errorf("QueryParam() convenience function = %q, want %q", result2, tt.expectedVal)
			}
		})
	}
}

func TestQueryExtractor_QueryParamOrDefault(t *testing.T) {
	tests := []struct {
		name        string
		query       string
		paramName   string
		defaultVal  string
		expectedVal string
	}{
		{
			name:        "existing param uses value",
			query:       "page=10",
			paramName:   "page",
			defaultVal:  "1",
			expectedVal: "10",
		},
		{
			name:        "missing param uses default",
			query:       "other=value",
			paramName:   "page",
			defaultVal:  "1",
			expectedVal: "1",
		},
		{
			name:        "empty param uses default",
			query:       "page=",
			paramName:   "page",
			defaultVal:  "1",
			expectedVal: "1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/?"+tt.query, nil)

			result := Query.QueryParamOrDefault(req, tt.paramName, tt.defaultVal)
			if result != tt.expectedVal {
				t.Errorf("QueryParamOrDefault(%q, %q) = %q, want %q",
					tt.paramName, tt.defaultVal, result, tt.expectedVal)
			}

			// Also test convenience function
			result2 := QueryParamOrDefault(req, tt.paramName, tt.defaultVal)
			if result2 != tt.expectedVal {
				t.Errorf("QueryParamOrDefault() convenience function = %q, want %q", result2, tt.expectedVal)
			}
		})
	}
}

func TestQueryParamAs_String(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/?name=John", nil)

	result, err := QueryParamAs[string](req, "name")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result != "John" {
		t.Errorf("expected 'John', got %q", result)
	}

	// Missing param returns zero value
	missing, err := QueryParamAs[string](req, "missing")
	if err != nil {
		t.Errorf("unexpected error for missing param: %v", err)
	}
	if missing != "" {
		t.Errorf("expected empty string for missing param, got %q", missing)
	}
}

func TestQueryParamAs_IntTypes(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/?num=42&neg=-10&big=10000000000", nil)

	tests := []struct {
		name     string
		fn       func() (any, error)
		expected any
	}{
		{
			name: "int",
			fn: func() (any, error) {
				return QueryParamAs[int](req, "num")
			},
			expected: 42,
		},
		{
			name: "int8",
			fn: func() (any, error) {
				return QueryParamAs[int8](req, "num")
			},
			expected: int8(42),
		},
		{
			name: "int16",
			fn: func() (any, error) {
				return QueryParamAs[int16](req, "num")
			},
			expected: int16(42),
		},
		{
			name: "int32",
			fn: func() (any, error) {
				return QueryParamAs[int32](req, "num")
			},
			expected: int32(42),
		},
		{
			name: "int64",
			fn: func() (any, error) {
				return QueryParamAs[int64](req, "big")
			},
			expected: int64(10000000000),
		},
		{
			name: "negative int",
			fn: func() (any, error) {
				return QueryParamAs[int](req, "neg")
			},
			expected: -10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.fn()
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestQueryParamAs_UintTypes(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/?num=42&big=10000000000", nil)

	tests := []struct {
		name     string
		fn       func() (any, error)
		expected any
	}{
		{
			name: "uint",
			fn: func() (any, error) {
				return QueryParamAs[uint](req, "num")
			},
			expected: uint(42),
		},
		{
			name: "uint8",
			fn: func() (any, error) {
				return QueryParamAs[uint8](req, "num")
			},
			expected: uint8(42),
		},
		{
			name: "uint16",
			fn: func() (any, error) {
				return QueryParamAs[uint16](req, "num")
			},
			expected: uint16(42),
		},
		{
			name: "uint32",
			fn: func() (any, error) {
				return QueryParamAs[uint32](req, "num")
			},
			expected: uint32(42),
		},
		{
			name: "uint64",
			fn: func() (any, error) {
				return QueryParamAs[uint64](req, "big")
			},
			expected: uint64(10000000000),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.fn()
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestQueryParamAs_FloatTypes(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/?pi=3.14&big=1.7976931348623157e+308", nil)

	// float32
	result32, err := QueryParamAs[float32](req, "pi")
	if err != nil {
		t.Errorf("unexpected error for float32: %v", err)
	}
	if result32 != 3.14 {
		t.Errorf("expected 3.14, got %f", result32)
	}

	// float64
	result64, err := QueryParamAs[float64](req, "pi")
	if err != nil {
		t.Errorf("unexpected error for float64: %v", err)
	}
	if result64 != 3.14 {
		t.Errorf("expected 3.14, got %f", result64)
	}
}

func TestQueryParamAs_Bool(t *testing.T) {
	tests := []struct {
		value    string
		expected bool
	}{
		{"true", true},
		{"True", true},
		{"TRUE", true},
		{"1", true},
		{"t", true},
		{"T", true},
		{"false", false},
		{"False", false},
		{"FALSE", false},
		{"0", false},
		{"f", false},
		{"F", false},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/?active="+tt.value, nil)

			result, err := QueryParamAs[bool](req, "active")
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestQueryParamAs_MissingParam(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/?other=value", nil)

	// Missing param should return zero value and no error
	result, err := QueryParamAs[int](req, "page")
	if err != nil {
		t.Errorf("unexpected error for missing param: %v", err)
	}
	if result != 0 {
		t.Errorf("expected 0 for missing param, got %d", result)
	}
}

func TestQueryParamAs_InvalidValues(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/?num=abc&float=xyz&bool=maybe", nil)

	tests := []struct {
		name string
		fn   func() (any, error)
	}{
		{
			name: "invalid int",
			fn: func() (any, error) {
				return QueryParamAs[int](req, "num")
			},
		},
		{
			name: "invalid int8",
			fn: func() (any, error) {
				return QueryParamAs[int8](req, "num")
			},
		},
		{
			name: "invalid int16",
			fn: func() (any, error) {
				return QueryParamAs[int16](req, "num")
			},
		},
		{
			name: "invalid int32",
			fn: func() (any, error) {
				return QueryParamAs[int32](req, "num")
			},
		},
		{
			name: "invalid int64",
			fn: func() (any, error) {
				return QueryParamAs[int64](req, "num")
			},
		},
		{
			name: "invalid uint",
			fn: func() (any, error) {
				return QueryParamAs[uint](req, "num")
			},
		},
		{
			name: "invalid uint8",
			fn: func() (any, error) {
				return QueryParamAs[uint8](req, "num")
			},
		},
		{
			name: "invalid uint16",
			fn: func() (any, error) {
				return QueryParamAs[uint16](req, "num")
			},
		},
		{
			name: "invalid uint32",
			fn: func() (any, error) {
				return QueryParamAs[uint32](req, "num")
			},
		},
		{
			name: "invalid uint64",
			fn: func() (any, error) {
				return QueryParamAs[uint64](req, "num")
			},
		},
		{
			name: "invalid float32",
			fn: func() (any, error) {
				return QueryParamAs[float32](req, "float")
			},
		},
		{
			name: "invalid float64",
			fn: func() (any, error) {
				return QueryParamAs[float64](req, "float")
			},
		},
		{
			name: "invalid bool",
			fn: func() (any, error) {
				return QueryParamAs[bool](req, "bool")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.fn()
			if err == nil {
				t.Error("expected error but got none")
			}
		})
	}
}

func TestQueryParamAsOrDefault(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/?page=10&limit=", nil)

	tests := []struct {
		name        string
		paramName   string
		defaultVal  int
		expectedVal int
	}{
		{
			name:        "existing value uses it",
			paramName:   "page",
			defaultVal:  1,
			expectedVal: 10,
		},
		{
			name:        "missing uses default",
			paramName:   "offset",
			defaultVal:  0,
			expectedVal: 0,
		},
		{
			name:        "empty value uses default",
			paramName:   "limit",
			defaultVal:  20,
			expectedVal: 20,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := QueryParamAsOrDefault(req, tt.paramName, tt.defaultVal)
			if result != tt.expectedVal {
				t.Errorf("QueryParamAsOrDefault(%q, %d) = %d, want %d",
					tt.paramName, tt.defaultVal, result, tt.expectedVal)
			}
		})
	}
}

func TestQueryParamAsOrDefault_InvalidConversion(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/?page=abc", nil)

	// Invalid conversion returns default
	result := QueryParamAsOrDefault(req, "page", 1)
	if result != 1 {
		t.Errorf("expected default 1 for invalid conversion, got %d", result)
	}
}

func TestQueryParamAsOrDefault_String(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/?sort=name", nil)

	// Existing value
	result := QueryParamAsOrDefault(req, "sort", "id")
	if result != "name" {
		t.Errorf("expected 'name', got %q", result)
	}

	// Missing value
	result = QueryParamAsOrDefault(req, "order", "asc")
	if result != "asc" {
		t.Errorf("expected 'asc', got %q", result)
	}
}

func TestQueryParamAsOrDefault_Bool(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/?active=true", nil)

	// Existing true value
	result := QueryParamAsOrDefault(req, "active", false)
	if !result {
		t.Error("expected true")
	}

	// Missing value uses default
	result = QueryParamAsOrDefault(req, "enabled", true)
	if !result {
		t.Error("expected default true for missing param")
	}
}

func TestQueryParam_CaseSensitivity(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/?Page=10&PAGE=20", nil)

	// Query params are case-sensitive
	page := Query.QueryParam(req, "Page")
	if page != "10" {
		t.Errorf("expected Page='10', got %q", page)
	}

	pageUpper := Query.QueryParam(req, "PAGE")
	if pageUpper != "20" {
		t.Errorf("expected PAGE='20', got %q", pageUpper)
	}

	pageLower := Query.QueryParam(req, "page")
	if pageLower != "" {
		t.Errorf("expected page='', got %q (query params are case-sensitive)", pageLower)
	}
}

func TestQueryParam_MultipleValues(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/?tag=go&tag=web&tag=api", nil)

	// Query().Get() returns the first value
	tag := Query.QueryParam(req, "tag")
	if tag != "go" {
		t.Errorf("expected first value 'go', got %q", tag)
	}

	// Verify all values are accessible via URL.Query()
	allTags := req.URL.Query()["tag"]
	if len(allTags) != 3 {
		t.Errorf("expected 3 values, got %d", len(allTags))
	}
}
