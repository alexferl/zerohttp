package pagination

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/alexferl/zerohttp/httpx"
)

// Request is an embeddable struct for pagination query parameters.
// Use this in your request structs to automatically bind page and per_page
// from query strings with validation.
//
// Example:
//
//	type ListRequest struct {
//	    pagination.Request
//	    Search string `query:"search"`
//	}
type Request struct {
	Page    int `query:"page" validate:"omitempty,min=1"`
	PerPage int `query:"per_page" validate:"omitempty,min=1,max=100"`
}

// Params returns pagination Params from the request values.
func (r Request) Params() Params {
	return Params(r)
}

// Params holds pagination parameters after binding and validation.
type Params struct {
	Page    int
	PerPage int
}

// Defaults returns pagination params with defaults applied.
// If Page is less than 1, it defaults to 1.
// If PerPage is less than 1, it defaults to 20.
// If PerPage is greater than 100, it is capped at 100.
func (p Params) Defaults() Params {
	result := p
	if result.Page < 1 {
		result.Page = 1
	}
	if result.PerPage < 1 {
		result.PerPage = 20
	}
	if result.PerPage > 100 {
		result.PerPage = 100
	}
	return result
}

// Offset calculates the database skip offset for the current page.
// Returns (Page-1) * PerPage.
func (p Params) Offset() int {
	return (p.Page - 1) * p.PerPage
}

// LimitSkip returns perPage and offset values for database queries.
// This is a convenience method that returns the common parameters
// needed for LIMIT/OFFSET style queries.
func (p Params) LimitSkip() (limit, skip int) {
	return p.PerPage, p.Offset()
}

// TotalPages calculates total pages from total items count.
// Returns at least 1 even for empty results to maintain consistent
// pagination semantics.
func (p Params) TotalPages(total int) int {
	if total == 0 {
		return 1
	}
	pages := (total + p.PerPage - 1) / p.PerPage
	if pages < 1 {
		return 1
	}
	return pages
}

// ResponseHeaders contains pagination header values.
type ResponseHeaders struct {
	Total      int
	TotalPages int
	Page       int
	PerPage    int
	HasPrev    bool
	HasNext    bool
	PrevPage   int
	NextPage   int
}

// Headers calculates pagination headers from params and total.
// This computes all the values needed for response headers without
// writing them, allowing inspection or modification before sending.
func (p Params) Headers(total int) ResponseHeaders {
	totalPages := p.TotalPages(total)
	headers := ResponseHeaders{
		Total:      total,
		TotalPages: totalPages,
		Page:       p.Page,
		PerPage:    p.PerPage,
	}

	if p.Page > 1 {
		headers.HasPrev = true
		headers.PrevPage = p.Page - 1
	}
	if p.Page < totalPages {
		headers.HasNext = true
		headers.NextPage = p.Page + 1
	}

	return headers
}

// WriteHeaders writes all pagination headers to the response.
// This sets X-Total, X-Total-Pages, X-Page, X-Per-Page, X-Prev-Page,
// X-Next-Page, and Link headers based on the current pagination state.
func (p Params) WriteHeaders(w http.ResponseWriter, u *url.URL, total int) {
	headers := p.Headers(total)

	w.Header().Set(httpx.HeaderXTotal, strconv.Itoa(headers.Total))
	w.Header().Set(httpx.HeaderXTotalPages, strconv.Itoa(headers.TotalPages))
	w.Header().Set(httpx.HeaderXPage, strconv.Itoa(headers.Page))
	w.Header().Set(httpx.HeaderXPerPage, strconv.Itoa(headers.PerPage))

	if headers.HasPrev {
		w.Header().Set(httpx.HeaderXPrevPage, strconv.Itoa(headers.PrevPage))
	}
	if headers.HasNext {
		w.Header().Set(httpx.HeaderXNextPage, strconv.Itoa(headers.NextPage))
	}

	links := BuildLinkHeader(u, p.Page, p.PerPage, headers.TotalPages)
	if links != "" {
		w.Header().Set(httpx.HeaderLink, links)
	}
}

// BuildLinkHeader creates the RFC 5988 Link header for pagination.
// Returns a string with first, prev, next, and last relations as appropriate.
// The returned string is empty if totalPages is less than 1.
func BuildLinkHeader(u *url.URL, page, perPage, totalPages int) string {
	if totalPages < 1 {
		return ""
	}

	var links []string

	// First page
	first := cloneURLWithParams(u, 1, perPage)
	links = append(links, fmt.Sprintf("<%s>; rel=\"first\"", first))

	// Previous page
	if page > 1 {
		prev := cloneURLWithParams(u, page-1, perPage)
		links = append(links, fmt.Sprintf("<%s>; rel=\"prev\"", prev))
	}

	// Next page
	if page < totalPages {
		next := cloneURLWithParams(u, page+1, perPage)
		links = append(links, fmt.Sprintf("<%s>; rel=\"next\"", next))
	}

	// Last page
	last := cloneURLWithParams(u, totalPages, perPage)
	links = append(links, fmt.Sprintf("<%s>; rel=\"last\"", last))

	return strings.Join(links, ", ")
}

// cloneURLWithParams creates a URL with updated pagination params.
// Preserves all existing query parameters and updates page and per_page.
func cloneURLWithParams(original *url.URL, page, perPage int) string {
	u := &url.URL{
		Scheme: original.Scheme,
		Host:   original.Host,
		Path:   original.Path,
	}
	q := original.Query()
	q.Set("page", strconv.Itoa(page))
	q.Set("per_page", strconv.Itoa(perPage))
	u.RawQuery = q.Encode()
	return u.String()
}
