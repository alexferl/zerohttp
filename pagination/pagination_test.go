package pagination

import (
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/alexferl/zerohttp/httpx"
	"github.com/alexferl/zerohttp/zhtest"
)

func TestRequest_Params(t *testing.T) {
	tests := []struct {
		name     string
		request  Request
		expected Params
	}{
		{
			name:     "zero values",
			request:  Request{},
			expected: Params{Page: 0, PerPage: 0},
		},
		{
			name:     "with page only",
			request:  Request{Page: 5},
			expected: Params{Page: 5, PerPage: 0},
		},
		{
			name:     "with per_page only",
			request:  Request{PerPage: 25},
			expected: Params{Page: 0, PerPage: 25},
		},
		{
			name:     "with both values",
			request:  Request{Page: 3, PerPage: 50},
			expected: Params{Page: 3, PerPage: 50},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.request.Params()
			zhtest.AssertEqual(t, tt.expected.Page, result.Page)
			zhtest.AssertEqual(t, tt.expected.PerPage, result.PerPage)
		})
	}
}

func TestDefaultPerPage(t *testing.T) {
	// Save original values and restore after test
	originalDefault := DefaultPerPage
	originalMax := DefaultMaxPerPage
	defer func() {
		DefaultPerPage = originalDefault
		DefaultMaxPerPage = originalMax
	}()

	tests := []struct {
		name           string
		defaultPerPage int
		input          Params
		expectedPage   int
	}{
		{
			name:           "custom default per page is applied",
			defaultPerPage: 50,
			input:          Params{},
			expectedPage:   50,
		},
		{
			name:           "custom default of 10",
			defaultPerPage: 10,
			input:          Params{Page: 2},
			expectedPage:   10,
		},
		{
			name:           "explicit per_page is not overridden",
			defaultPerPage: 50,
			input:          Params{PerPage: 30},
			expectedPage:   30,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			DefaultPerPage = tt.defaultPerPage
			result := tt.input.Defaults()
			zhtest.AssertEqual(t, tt.expectedPage, result.PerPage)
		})
	}
}

func TestDefaultMaxPerPage(t *testing.T) {
	// Save original values and restore after test
	originalDefault := DefaultPerPage
	originalMax := DefaultMaxPerPage
	defer func() {
		DefaultPerPage = originalDefault
		DefaultMaxPerPage = originalMax
	}()

	tests := []struct {
		name        string
		maxPerPage  int
		input       Params
		expectedMax int
	}{
		{
			name:        "custom max per page caps value",
			maxPerPage:  50,
			input:       Params{Page: 1, PerPage: 100},
			expectedMax: 50,
		},
		{
			name:        "custom max of 200",
			maxPerPage:  200,
			input:       Params{Page: 1, PerPage: 250},
			expectedMax: 200,
		},
		{
			name:        "value under custom max is allowed",
			maxPerPage:  50,
			input:       Params{Page: 1, PerPage: 30},
			expectedMax: 30,
		},
		{
			name:        "value at custom max is allowed",
			maxPerPage:  50,
			input:       Params{Page: 1, PerPage: 50},
			expectedMax: 50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			DefaultMaxPerPage = tt.maxPerPage
			result := tt.input.Defaults()
			zhtest.AssertEqual(t, tt.expectedMax, result.PerPage)
		})
	}
}

func TestParams_Defaults(t *testing.T) {
	tests := []struct {
		name     string
		input    Params
		expected Params
	}{
		{
			name:     "zero values get defaults",
			input:    Params{},
			expected: Params{Page: 1, PerPage: 25},
		},
		{
			name:     "negative page gets default",
			input:    Params{Page: -1, PerPage: 10},
			expected: Params{Page: 1, PerPage: 10},
		},
		{
			name:     "negative per_page gets default",
			input:    Params{Page: 5, PerPage: -5},
			expected: Params{Page: 5, PerPage: 25},
		},
		{
			name:     "per_page over max gets capped",
			input:    Params{Page: 1, PerPage: 200},
			expected: Params{Page: 1, PerPage: 100},
		},
		{
			name:     "per_page at max is allowed",
			input:    Params{Page: 1, PerPage: 100},
			expected: Params{Page: 1, PerPage: 100},
		},
		{
			name:     "valid values unchanged",
			input:    Params{Page: 3, PerPage: 25},
			expected: Params{Page: 3, PerPage: 25},
		},
		{
			name:     "page 1 is valid",
			input:    Params{Page: 1, PerPage: 25},
			expected: Params{Page: 1, PerPage: 25},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.input.Defaults()
			zhtest.AssertEqual(t, tt.expected.Page, result.Page)
			zhtest.AssertEqual(t, tt.expected.PerPage, result.PerPage)
		})
	}
}

func TestParams_Offset(t *testing.T) {
	tests := []struct {
		name     string
		params   Params
		expected int
	}{
		{
			name:     "page 1 has offset 0",
			params:   Params{Page: 1, PerPage: 20},
			expected: 0,
		},
		{
			name:     "page 2 with per_page 20 has offset 20",
			params:   Params{Page: 2, PerPage: 20},
			expected: 20,
		},
		{
			name:     "page 3 with per_page 10 has offset 20",
			params:   Params{Page: 3, PerPage: 10},
			expected: 20,
		},
		{
			name:     "page 5 with per_page 50 has offset 200",
			params:   Params{Page: 5, PerPage: 50},
			expected: 200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.params.Offset()
			zhtest.AssertEqual(t, tt.expected, result)
		})
	}
}

func TestParams_LimitSkip(t *testing.T) {
	params := Params{Page: 3, PerPage: 25}
	limit, skip := params.LimitSkip()

	zhtest.AssertEqual(t, 25, limit)
	zhtest.AssertEqual(t, 50, skip)
}

func TestParams_TotalPages(t *testing.T) {
	tests := []struct {
		name     string
		params   Params
		total    int
		expected int
	}{
		{
			name:     "zero total returns 1",
			params:   Params{PerPage: 20},
			total:    0,
			expected: 1,
		},
		{
			name:     "total less than per_page returns 1",
			params:   Params{PerPage: 20},
			total:    15,
			expected: 1,
		},
		{
			name:     "total equal to per_page returns 1",
			params:   Params{PerPage: 20},
			total:    20,
			expected: 1,
		},
		{
			name:     "total slightly more than per_page returns 2",
			params:   Params{PerPage: 20},
			total:    21,
			expected: 2,
		},
		{
			name:     "exact multiple",
			params:   Params{PerPage: 10},
			total:    100,
			expected: 10,
		},
		{
			name:     "large total",
			params:   Params{PerPage: 25},
			total:    1000,
			expected: 40,
		},
		{
			name:     "single item",
			params:   Params{PerPage: 20},
			total:    1,
			expected: 1,
		},
		{
			name:     "negative total returns 1",
			params:   Params{PerPage: 20},
			total:    -5,
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.params.TotalPages(tt.total)
			zhtest.AssertEqual(t, tt.expected, result)
		})
	}
}

func TestParams_Headers(t *testing.T) {
	tests := []struct {
		name     string
		params   Params
		total    int
		expected ResponseHeaders
	}{
		{
			name:   "first page of many",
			params: Params{Page: 1, PerPage: 20},
			total:  100,
			expected: ResponseHeaders{
				Total:      100,
				TotalPages: 5,
				Page:       1,
				PerPage:    20,
				HasPrev:    false,
				HasNext:    true,
				PrevPage:   0,
				NextPage:   2,
			},
		},
		{
			name:   "middle page",
			params: Params{Page: 3, PerPage: 20},
			total:  100,
			expected: ResponseHeaders{
				Total:      100,
				TotalPages: 5,
				Page:       3,
				PerPage:    20,
				HasPrev:    true,
				HasNext:    true,
				PrevPage:   2,
				NextPage:   4,
			},
		},
		{
			name:   "last page",
			params: Params{Page: 5, PerPage: 20},
			total:  100,
			expected: ResponseHeaders{
				Total:      100,
				TotalPages: 5,
				Page:       5,
				PerPage:    20,
				HasPrev:    true,
				HasNext:    false,
				PrevPage:   4,
				NextPage:   0,
			},
		},
		{
			name:   "single page no navigation",
			params: Params{Page: 1, PerPage: 20},
			total:  15,
			expected: ResponseHeaders{
				Total:      15,
				TotalPages: 1,
				Page:       1,
				PerPage:    20,
				HasPrev:    false,
				HasNext:    false,
				PrevPage:   0,
				NextPage:   0,
			},
		},
		{
			name:   "empty result",
			params: Params{Page: 1, PerPage: 20},
			total:  0,
			expected: ResponseHeaders{
				Total:      0,
				TotalPages: 1,
				Page:       1,
				PerPage:    20,
				HasPrev:    false,
				HasNext:    false,
				PrevPage:   0,
				NextPage:   0,
			},
		},
		{
			name:   "page 2 of 2",
			params: Params{Page: 2, PerPage: 10},
			total:  15,
			expected: ResponseHeaders{
				Total:      15,
				TotalPages: 2,
				Page:       2,
				PerPage:    10,
				HasPrev:    true,
				HasNext:    false,
				PrevPage:   1,
				NextPage:   0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.params.Headers(tt.total)

			zhtest.AssertEqual(t, tt.expected.Total, result.Total)
			zhtest.AssertEqual(t, tt.expected.TotalPages, result.TotalPages)
			zhtest.AssertEqual(t, tt.expected.Page, result.Page)
			zhtest.AssertEqual(t, tt.expected.PerPage, result.PerPage)
			zhtest.AssertEqual(t, tt.expected.HasPrev, result.HasPrev)
			zhtest.AssertEqual(t, tt.expected.HasNext, result.HasNext)
			zhtest.AssertEqual(t, tt.expected.PrevPage, result.PrevPage)
			zhtest.AssertEqual(t, tt.expected.NextPage, result.NextPage)
		})
	}
}

func TestParams_WriteHeaders(t *testing.T) {
	u, _ := url.Parse("http://example.com/api/items?page=2&per_page=20")
	rec := httptest.NewRecorder()

	params := Params{Page: 2, PerPage: 20}
	params.WriteHeaders(rec, u, 50)

	headers := rec.Header()

	zhtest.AssertEqual(t, "50", headers.Get(httpx.HeaderXTotal))
	zhtest.AssertEqual(t, "3", headers.Get(httpx.HeaderXTotalPages))
	zhtest.AssertEqual(t, "2", headers.Get(httpx.HeaderXPage))
	zhtest.AssertEqual(t, "20", headers.Get(httpx.HeaderXPerPage))
	zhtest.AssertEqual(t, "1", headers.Get(httpx.HeaderXPrevPage))
	zhtest.AssertEqual(t, "3", headers.Get(httpx.HeaderXNextPage))
	zhtest.AssertNotEmpty(t, headers.Get("Link"))
}

func TestParams_WriteHeaders_FirstPage(t *testing.T) {
	u, _ := url.Parse("http://example.com/api/items?page=1&per_page=20")
	rec := httptest.NewRecorder()

	params := Params{Page: 1, PerPage: 20}
	params.WriteHeaders(rec, u, 50)

	headers := rec.Header()

	zhtest.AssertEmpty(t, headers.Get(httpx.HeaderXPrevPage))
	zhtest.AssertEqual(t, "2", headers.Get(httpx.HeaderXNextPage))
}

func TestParams_WriteHeaders_LastPage(t *testing.T) {
	u, _ := url.Parse("http://example.com/api/items?page=3&per_page=20")
	rec := httptest.NewRecorder()

	params := Params{Page: 3, PerPage: 20}
	params.WriteHeaders(rec, u, 50)

	headers := rec.Header()

	zhtest.AssertEqual(t, "2", headers.Get(httpx.HeaderXPrevPage))
	zhtest.AssertEmpty(t, headers.Get(httpx.HeaderXNextPage))
}

func TestParams_WriteHeaders_SinglePage(t *testing.T) {
	u, _ := url.Parse("http://example.com/api/items?page=1&per_page=20")
	rec := httptest.NewRecorder()

	params := Params{Page: 1, PerPage: 20}
	params.WriteHeaders(rec, u, 10)

	headers := rec.Header()

	zhtest.AssertEmpty(t, headers.Get(httpx.HeaderXPrevPage))
	zhtest.AssertEmpty(t, headers.Get(httpx.HeaderXNextPage))
}

func TestBuildLinkHeader(t *testing.T) {
	tests := []struct {
		name       string
		url        string
		page       int
		perPage    int
		totalPages int
		expected   string
	}{
		{
			name:       "first page includes first, next, last",
			url:        "http://example.com/api/items",
			page:       1,
			perPage:    20,
			totalPages: 5,
			expected:   `<http://example.com/api/items?page=1&per_page=20>; rel="first", <http://example.com/api/items?page=2&per_page=20>; rel="next", <http://example.com/api/items?page=5&per_page=20>; rel="last"`,
		},
		{
			name:       "middle page includes all links",
			url:        "http://example.com/api/items",
			page:       3,
			perPage:    20,
			totalPages: 5,
			expected:   `<http://example.com/api/items?page=1&per_page=20>; rel="first", <http://example.com/api/items?page=2&per_page=20>; rel="prev", <http://example.com/api/items?page=4&per_page=20>; rel="next", <http://example.com/api/items?page=5&per_page=20>; rel="last"`,
		},
		{
			name:       "last page includes first, prev, last",
			url:        "http://example.com/api/items",
			page:       5,
			perPage:    20,
			totalPages: 5,
			expected:   `<http://example.com/api/items?page=1&per_page=20>; rel="first", <http://example.com/api/items?page=4&per_page=20>; rel="prev", <http://example.com/api/items?page=5&per_page=20>; rel="last"`,
		},
		{
			name:       "preserves existing query params",
			url:        "http://example.com/api/items?search=test&category=books",
			page:       2,
			perPage:    10,
			totalPages: 3,
			expected:   `<http://example.com/api/items?category=books&page=1&per_page=10&search=test>; rel="first", <http://example.com/api/items?category=books&page=1&per_page=10&search=test>; rel="prev", <http://example.com/api/items?category=books&page=3&per_page=10&search=test>; rel="next", <http://example.com/api/items?category=books&page=3&per_page=10&search=test>; rel="last"`,
		},
		{
			name:       "single page has no prev or next",
			url:        "http://example.com/api/items",
			page:       1,
			perPage:    20,
			totalPages: 1,
			expected:   `<http://example.com/api/items?page=1&per_page=20>; rel="first", <http://example.com/api/items?page=1&per_page=20>; rel="last"`,
		},
		{
			name:       "with path",
			url:        "http://example.com/v2/resources",
			page:       2,
			perPage:    50,
			totalPages: 10,
			expected:   `<http://example.com/v2/resources?page=1&per_page=50>; rel="first", <http://example.com/v2/resources?page=1&per_page=50>; rel="prev", <http://example.com/v2/resources?page=3&per_page=50>; rel="next", <http://example.com/v2/resources?page=10&per_page=50>; rel="last"`,
		},
		{
			name:       "zero total pages returns empty",
			url:        "http://example.com/api/items",
			page:       1,
			perPage:    20,
			totalPages: 0,
			expected:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, _ := url.Parse(tt.url)
			result := BuildLinkHeader(u, tt.page, tt.perPage, tt.totalPages)
			zhtest.AssertEqual(t, tt.expected, result)
		})
	}
}

func TestCloneURLWithParams(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		page     int
		perPage  int
		expected string
	}{
		{
			name:     "simple URL",
			url:      "http://example.com/api/items",
			page:     2,
			perPage:  20,
			expected: "http://example.com/api/items?page=2&per_page=20",
		},
		{
			name:     "URL with existing params",
			url:      "http://example.com/api/items?search=test",
			page:     3,
			perPage:  10,
			expected: "http://example.com/api/items?page=3&per_page=10&search=test",
		},
		{
			name:     "URL with pagination params to be replaced",
			url:      "http://example.com/api/items?page=1&per_page=5&search=test",
			page:     2,
			perPage:  10,
			expected: "http://example.com/api/items?page=2&per_page=10&search=test",
		},
		{
			name:     "HTTPS URL",
			url:      "https://api.example.com/v1/users",
			page:     1,
			perPage:  25,
			expected: "https://api.example.com/v1/users?page=1&per_page=25",
		},
		{
			name:     "URL with port",
			url:      "http://localhost:8080/api/items",
			page:     5,
			perPage:  50,
			expected: "http://localhost:8080/api/items?page=5&per_page=50",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, _ := url.Parse(tt.url)
			result := cloneURLWithParams(u, tt.page, tt.perPage)
			zhtest.AssertEqual(t, tt.expected, result)
		})
	}
}
