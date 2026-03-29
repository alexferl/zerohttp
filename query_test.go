package zerohttp

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alexferl/zerohttp/zhtest"
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
			zhtest.AssertEqual(t, result, tt.expectedVal)

			// Also test convenience function
			result2 := QueryParam(req, tt.paramName)
			zhtest.AssertEqual(t, result2, tt.expectedVal)
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
			zhtest.AssertEqual(t, result, tt.expectedVal)

			// Also test convenience function
			result2 := QueryParamOrDefault(req, tt.paramName, tt.defaultVal)
			zhtest.AssertEqual(t, result2, tt.expectedVal)
		})
	}
}

func TestQueryParamAs_String(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/?name=John", nil)

	result, err := QueryParamAs[string](req, "name")
	zhtest.AssertNoError(t, err)
	zhtest.AssertEqual(t, result, "John")

	// Missing param returns zero value
	missing, err := QueryParamAs[string](req, "missing")
	zhtest.AssertNoError(t, err)
	zhtest.AssertEqual(t, missing, "")
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
			zhtest.AssertNoError(t, err)
			zhtest.AssertEqual(t, result, tt.expected)
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
			zhtest.AssertNoError(t, err)
			zhtest.AssertEqual(t, result, tt.expected)
		})
	}
}

func TestQueryParamAs_FloatTypes(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/?pi=3.14&big=1.7976931348623157e+308", nil)

	// float32
	result32, err := QueryParamAs[float32](req, "pi")
	zhtest.AssertNoError(t, err)
	zhtest.AssertEqual(t, result32, float32(3.14))

	// float64
	result64, err := QueryParamAs[float64](req, "pi")
	zhtest.AssertNoError(t, err)
	zhtest.AssertEqual(t, result64, 3.14)
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
			zhtest.AssertNoError(t, err)
			zhtest.AssertEqual(t, result, tt.expected)
		})
	}
}

func TestQueryParamAs_MissingParam(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/?other=value", nil)

	// Missing param should return zero value and no error
	result, err := QueryParamAs[int](req, "page")
	zhtest.AssertNoError(t, err)
	zhtest.AssertEqual(t, result, 0)
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
			zhtest.AssertError(t, err)
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
			zhtest.AssertEqual(t, result, tt.expectedVal)
		})
	}
}

func TestQueryParamAsOrDefault_InvalidConversion(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/?page=abc", nil)

	// Invalid conversion returns default
	result := QueryParamAsOrDefault(req, "page", 1)
	zhtest.AssertEqual(t, result, 1)
}

func TestQueryParamAsOrDefault_String(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/?sort=name", nil)

	// Existing value
	result := QueryParamAsOrDefault(req, "sort", "id")
	zhtest.AssertEqual(t, result, "name")

	// Missing value
	result = QueryParamAsOrDefault(req, "order", "asc")
	zhtest.AssertEqual(t, result, "asc")
}

func TestQueryParamAsOrDefault_Bool(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/?active=true", nil)

	// Existing true value
	result := QueryParamAsOrDefault(req, "active", false)
	zhtest.AssertTrue(t, result)

	// Missing value uses default
	result = QueryParamAsOrDefault(req, "enabled", true)
	zhtest.AssertTrue(t, result)
}

func TestQueryParam_CaseSensitivity(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/?Page=10&PAGE=20", nil)

	// Query params are case-sensitive
	page := Query.QueryParam(req, "Page")
	zhtest.AssertEqual(t, page, "10")

	pageUpper := Query.QueryParam(req, "PAGE")
	zhtest.AssertEqual(t, pageUpper, "20")

	pageLower := Query.QueryParam(req, "page")
	zhtest.AssertEqual(t, pageLower, "")
}

func TestQueryParam_MultipleValues(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/?tag=go&tag=web&tag=api", nil)

	// Query().Get() returns the first value
	tag := Query.QueryParam(req, "tag")
	zhtest.AssertEqual(t, tag, "go")

	// Verify all values are accessible via URL.Query()
	allTags := req.URL.Query()["tag"]
	zhtest.AssertEqual(t, len(allTags), 3)
}
