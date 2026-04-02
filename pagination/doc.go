// Package pagination provides utilities for handling pagination in HTTP APIs.
//
// It supports common pagination patterns including offset-based pagination with
// standardized response headers. The package integrates with zerohttp's query
// binding and validation for easy use in handlers.
//
// Basic usage:
//
//	import "github.com/alexferl/zerohttp/pagination"
//
//	// In your handler
//	func ListItems(w http.ResponseWriter, r *http.Request) error {
//	    var req struct {
//	        pagination.Request
//	    }
//	    if err := zh.B.Query(r, &req); err != nil {
//	        return err
//	    }
//
//	    params := req.Request.Params().Defaults()
//	    items, total := fetchItems(params.Offset(), params.PerPage)
//
//	    params.WriteHeaders(w, r.URL, total)
//	    return zh.R.JSON(w, http.StatusOK, items)
//	}
//
// Response Headers:
//
// The following headers are set by WriteHeaders:
//   - X-Total: Total number of items available
//   - X-Total-Pages: Total number of pages
//   - X-Page: Current page number
//   - X-Per-Page: Items per page
//   - X-Prev-Page: Previous page number (if available)
//   - X-Next-Page: Next page number (if available)
//   - Link: RFC 5988 Link header with first, prev, next, last relations
package pagination
