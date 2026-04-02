package pagination

import (
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/alexferl/zerohttp/httpx"
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
			if result.Page != tt.expected.Page {
				t.Errorf("expected Page %d, got %d", tt.expected.Page, result.Page)
			}
			if result.PerPage != tt.expected.PerPage {
				t.Errorf("expected PerPage %d, got %d", tt.expected.PerPage, result.PerPage)
			}
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
			expected: Params{Page: 1, PerPage: 20},
		},
		{
			name:     "negative page gets default",
			input:    Params{Page: -1, PerPage: 10},
			expected: Params{Page: 1, PerPage: 10},
		},
		{
			name:     "negative per_page gets default",
			input:    Params{Page: 5, PerPage: -5},
			expected: Params{Page: 5, PerPage: 20},
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
			input:    Params{Page: 1, PerPage: 20},
			expected: Params{Page: 1, PerPage: 20},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.input.Defaults()
			if result.Page != tt.expected.Page {
				t.Errorf("expected Page %d, got %d", tt.expected.Page, result.Page)
			}
			if result.PerPage != tt.expected.PerPage {
				t.Errorf("expected PerPage %d, got %d", tt.expected.PerPage, result.PerPage)
			}
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
			if result != tt.expected {
				t.Errorf("expected offset %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestParams_LimitSkip(t *testing.T) {
	params := Params{Page: 3, PerPage: 25}
	limit, skip := params.LimitSkip()

	if limit != 25 {
		t.Errorf("expected limit 25, got %d", limit)
	}
	if skip != 50 {
		t.Errorf("expected skip 50, got %d", skip)
	}
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
			if result != tt.expected {
				t.Errorf("expected %d pages, got %d", tt.expected, result)
			}
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

			if result.Total != tt.expected.Total {
				t.Errorf("expected Total %d, got %d", tt.expected.Total, result.Total)
			}
			if result.TotalPages != tt.expected.TotalPages {
				t.Errorf("expected TotalPages %d, got %d", tt.expected.TotalPages, result.TotalPages)
			}
			if result.Page != tt.expected.Page {
				t.Errorf("expected Page %d, got %d", tt.expected.Page, result.Page)
			}
			if result.PerPage != tt.expected.PerPage {
				t.Errorf("expected PerPage %d, got %d", tt.expected.PerPage, result.PerPage)
			}
			if result.HasPrev != tt.expected.HasPrev {
				t.Errorf("expected HasPrev %v, got %v", tt.expected.HasPrev, result.HasPrev)
			}
			if result.HasNext != tt.expected.HasNext {
				t.Errorf("expected HasNext %v, got %v", tt.expected.HasNext, result.HasNext)
			}
			if result.PrevPage != tt.expected.PrevPage {
				t.Errorf("expected PrevPage %d, got %d", tt.expected.PrevPage, result.PrevPage)
			}
			if result.NextPage != tt.expected.NextPage {
				t.Errorf("expected NextPage %d, got %d", tt.expected.NextPage, result.NextPage)
			}
		})
	}
}

func TestParams_WriteHeaders(t *testing.T) {
	u, _ := url.Parse("http://example.com/api/items?page=2&per_page=20")
	rec := httptest.NewRecorder()

	params := Params{Page: 2, PerPage: 20}
	params.WriteHeaders(rec, u, 50)

	headers := rec.Header()

	if headers.Get(httpx.HeaderXTotal) != "50" {
		t.Errorf("expected X-Total header '50', got '%s'", headers.Get(httpx.HeaderXTotal))
	}
	if headers.Get(httpx.HeaderXTotalPages) != "3" {
		t.Errorf("expected X-Total-Pages header '3', got '%s'", headers.Get(httpx.HeaderXTotalPages))
	}
	if headers.Get(httpx.HeaderXPage) != "2" {
		t.Errorf("expected X-Page header '2', got '%s'", headers.Get(httpx.HeaderXPage))
	}
	if headers.Get(httpx.HeaderXPerPage) != "20" {
		t.Errorf("expected X-Per-Page header '20', got '%s'", headers.Get(httpx.HeaderXPerPage))
	}
	if headers.Get(httpx.HeaderXPrevPage) != "1" {
		t.Errorf("expected X-Prev-Page header '1', got '%s'", headers.Get(httpx.HeaderXPrevPage))
	}
	if headers.Get(httpx.HeaderXNextPage) != "3" {
		t.Errorf("expected X-Next-Page header '3', got '%s'", headers.Get(httpx.HeaderXNextPage))
	}
	if headers.Get("Link") == "" {
		t.Error("expected Link header to be set")
	}
}

func TestParams_WriteHeaders_FirstPage(t *testing.T) {
	u, _ := url.Parse("http://example.com/api/items?page=1&per_page=20")
	rec := httptest.NewRecorder()

	params := Params{Page: 1, PerPage: 20}
	params.WriteHeaders(rec, u, 50)

	headers := rec.Header()

	if headers.Get(httpx.HeaderXPrevPage) != "" {
		t.Errorf("expected no X-Prev-Page header on first page, got '%s'", headers.Get(httpx.HeaderXPrevPage))
	}
	if headers.Get(httpx.HeaderXNextPage) != "2" {
		t.Errorf("expected X-Next-Page header '2', got '%s'", headers.Get(httpx.HeaderXNextPage))
	}
}

func TestParams_WriteHeaders_LastPage(t *testing.T) {
	u, _ := url.Parse("http://example.com/api/items?page=3&per_page=20")
	rec := httptest.NewRecorder()

	params := Params{Page: 3, PerPage: 20}
	params.WriteHeaders(rec, u, 50)

	headers := rec.Header()

	if headers.Get(httpx.HeaderXPrevPage) != "2" {
		t.Errorf("expected X-Prev-Page header '2', got '%s'", headers.Get(httpx.HeaderXPrevPage))
	}
	if headers.Get(httpx.HeaderXNextPage) != "" {
		t.Errorf("expected no X-Next-Page header on last page, got '%s'", headers.Get(httpx.HeaderXNextPage))
	}
}

func TestParams_WriteHeaders_SinglePage(t *testing.T) {
	u, _ := url.Parse("http://example.com/api/items?page=1&per_page=20")
	rec := httptest.NewRecorder()

	params := Params{Page: 1, PerPage: 20}
	params.WriteHeaders(rec, u, 10)

	headers := rec.Header()

	if headers.Get(httpx.HeaderXPrevPage) != "" {
		t.Errorf("expected no X-Prev-Page header for single page, got '%s'", headers.Get(httpx.HeaderXPrevPage))
	}
	if headers.Get(httpx.HeaderXNextPage) != "" {
		t.Errorf("expected no X-Next-Page header for single page, got '%s'", headers.Get(httpx.HeaderXNextPage))
	}
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
			if result != tt.expected {
				t.Errorf("expected:\n%s\ngot:\n%s", tt.expected, result)
			}
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
			if result != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}
