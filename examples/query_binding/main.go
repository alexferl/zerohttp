package main

import (
	"log"
	"net/http"

	zh "github.com/alexferl/zerohttp"
)

// SearchRequest demonstrates basic query parameter binding
type SearchRequest struct {
	Query    string   `query:"q"`
	Category string   `query:"category"`
	Tags     []string `query:"tags"`
}

// PaginationRequest demonstrates numeric types and optional params
type PaginationRequest struct {
	Page    int   `query:"page"`
	Limit   int   `query:"limit"`
	Offset  int   `query:"offset"`
	IsAdmin *bool `query:"is_admin"` // Pointer = optional
}

// FilterRequest demonstrates slices and multiple types
type FilterRequest struct {
	IDs      []int    `query:"id"`
	Statuses []string `query:"status"`
	MinPrice float64  `query:"min_price"`
	MaxPrice float64  `query:"max_price"`
	InStock  bool     `query:"in_stock"`
}

// EmbeddedPagination demonstrates embedded structs
type EmbeddedPagination struct {
	Page  int `query:"page"`
	Limit int `query:"limit"`
}

// ListRequest combines embedded pagination with search
type ListRequest struct {
	EmbeddedPagination
	Search string `query:"search"`
}

func main() {
	app := zh.New()

	// Routes
	app.GET("/", zh.HandlerFunc(indexHandler))
	app.GET("/search", zh.HandlerFunc(searchHandler))
	app.GET("/items", zh.HandlerFunc(itemsHandler))
	app.GET("/products", zh.HandlerFunc(productsHandler))
	app.GET("/users", zh.HandlerFunc(listUsersHandler))

	// Individual query parameter extraction
	app.GET("/extract", zh.HandlerFunc(extractHandler))

	log.Println("Server starting on :8080")
	log.Println("Try these endpoints:")
	log.Println("  GET /search?q=golang&category=tech&tags=api&tags=web")
	log.Println("  GET /items?page=2&limit=50&is_admin=true")
	log.Println("  GET /products?id=1&id=2&id=3&status=active&min_price=10.99")
	log.Println("  GET /users?search=john&page=1&limit=20")
	log.Println("  GET /extract?user_id=123&active=true&score=95.5")

	log.Fatal(app.Start())
}

func indexHandler(w http.ResponseWriter, r *http.Request) error {
	html := `<!DOCTYPE html>
<html>
<head>
    <title>Query Parameter Binding Examples</title>
    <style>
        body { font-family: sans-serif; max-width: 800px; margin: 40px auto; padding: 20px; }
        h1 { color: #333; }
        h2 { color: #555; margin-top: 30px; }
        code { background: #f4f4f4; padding: 2px 6px; border-radius: 3px; }
        pre { background: #f4f4f4; padding: 15px; border-radius: 5px; overflow-x: auto; }
        .endpoint { margin: 20px 0; padding: 15px; background: #f9f9f9; border-left: 4px solid #28a745; }
        .method { font-weight: bold; color: #28a745; }
    </style>
</head>
<body>
    <h1>Query Parameter Binding Examples</h1>
    <p>This example demonstrates zerohttp's query parameter binding using <code>Bind.Query()</code>.</p>

    <h2>Available Endpoints</h2>

    <div class="endpoint">
        <span class="method">GET</span> <code>/search</code>
        <p>Basic search with strings and slices.</p>
        <pre>curl "http://localhost:8080/search?q=golang&category=tech&tags=api&tags=web"</pre>
    </div>

    <div class="endpoint">
        <span class="method">GET</span> <code>/items</code>
        <p>Pagination with optional boolean pointer.</p>
        <pre>curl "http://localhost:8080/items?page=2&limit=50&is_admin=true"</pre>
    </div>

    <div class="endpoint">
        <span class="method">GET</span> <code>/products</code>
        <p>Filter with int slices and float values.</p>
        <pre>curl "http://localhost:8080/products?id=1&id=2&id=3&status=active&min_price=10.99"</pre>
    </div>

    <div class="endpoint">
        <span class="method">GET</span> <code>/users</code>
        <p>Embedded struct with pagination.</p>
        <pre>curl "http://localhost:8080/users?search=john&page=1&limit=20"</pre>
    </div>

    <div class="endpoint">
        <span class="method">GET</span> <code>/extract</code>
        <p>Individual parameter extraction helpers.</p>
        <pre>curl "http://localhost:8080/extract?user_id=123&active=true&score=95.5"</pre>
    </div>

    <h2>Features Demonstrated</h2>
    <ul>
        <li><strong>Bind.Query()</strong> - Bind query params to structs using <code>query:"name"</code> tags</li>
        <li><strong>QueryParamAs[T]()</strong> - Type-safe individual parameter extraction</li>
        <li><strong>QueryParamAsOrDefault()</strong> - Extract with default value fallback</li>
        <li><strong>Slice binding</strong> - Multiple values with same param name</li>
        <li><strong>Optional params</strong> - Pointer types for optional values</li>
        <li><strong>Type conversion</strong> - Automatic string to int, float, bool conversion</li>
        <li><strong>Embedded structs</strong> - Pagination structs embedded in request types</li>
    </ul>
</body>
</html>`
	return zh.R.HTML(w, http.StatusOK, html)
}

// searchHandler demonstrates basic query binding with slices
func searchHandler(w http.ResponseWriter, r *http.Request) error {
	var req SearchRequest
	if err := zh.B.Query(r, &req); err != nil {
		return zh.NewProblemDetail(http.StatusBadRequest, err.Error()).Render(w)
	}

	// Set defaults
	if req.Category == "" {
		req.Category = "all"
	}

	return zh.R.JSON(w, http.StatusOK, zh.M{
		"query":    req.Query,
		"category": req.Category,
		"tags":     req.Tags,
		"count":    len(req.Tags),
	})
}

// itemsHandler demonstrates numeric types and optional boolean pointer
func itemsHandler(w http.ResponseWriter, r *http.Request) error {
	var req PaginationRequest
	if err := zh.B.Query(r, &req); err != nil {
		return zh.NewProblemDetail(http.StatusBadRequest, err.Error()).Render(w)
	}

	// Set defaults
	if req.Page < 1 {
		req.Page = 1
	}
	if req.Limit < 1 {
		req.Limit = 20
	}
	if req.Limit > 100 {
		req.Limit = 100 // Max limit
	}

	response := zh.M{
		"page":   req.Page,
		"limit":  req.Limit,
		"offset": req.Offset,
	}

	// Optional pointer field - nil if not provided, true/false if provided
	if req.IsAdmin != nil {
		response["is_admin"] = *req.IsAdmin
		response["is_admin_provided"] = true
	} else {
		response["is_admin"] = nil
		response["is_admin_provided"] = false
	}

	return zh.R.JSON(w, http.StatusOK, response)
}

// productsHandler demonstrates int slices and float types
func productsHandler(w http.ResponseWriter, r *http.Request) error {
	var req FilterRequest
	if err := zh.B.Query(r, &req); err != nil {
		return zh.NewProblemDetail(http.StatusBadRequest, err.Error()).Render(w)
	}

	return zh.R.JSON(w, http.StatusOK, zh.M{
		"ids":             req.IDs,
		"statuses":        req.Statuses,
		"min_price":       req.MinPrice,
		"max_price":       req.MaxPrice,
		"in_stock":        req.InStock,
		"filters_applied": len(req.IDs) + len(req.Statuses),
	})
}

// listUsersHandler demonstrates embedded struct binding
func listUsersHandler(w http.ResponseWriter, r *http.Request) error {
	var req ListRequest
	if err := zh.B.Query(r, &req); err != nil {
		return zh.NewProblemDetail(http.StatusBadRequest, err.Error()).Render(w)
	}

	// Set defaults
	if req.Page < 1 {
		req.Page = 1
	}
	if req.Limit < 1 {
		req.Limit = 20
	}

	return zh.R.JSON(w, http.StatusOK, zh.M{
		"search": req.Search,
		"page":   req.Page,
		"limit":  req.Limit,
		"note":   "Pagination fields come from EmbeddedPagination struct",
	})
}

// extractHandler demonstrates individual parameter extraction helpers
func extractHandler(w http.ResponseWriter, r *http.Request) error {
	// Extract user_id as int with error handling
	userID, err := zh.QueryParamAs[int](r, "user_id")
	if err != nil {
		return zh.NewProblemDetail(http.StatusBadRequest, "Invalid user_id: "+err.Error()).Render(w)
	}

	// Extract score as float64 with error handling
	score, err := zh.QueryParamAs[float64](r, "score")
	if err != nil {
		return zh.NewProblemDetail(http.StatusBadRequest, "Invalid score: "+err.Error()).Render(w)
	}

	// Extract active as bool with error handling
	active, err := zh.QueryParamAs[bool](r, "active")
	if err != nil {
		return zh.NewProblemDetail(http.StatusBadRequest, "Invalid active: "+err.Error()).Render(w)
	}

	// Extract optional name with default
	name := zh.QueryParamAsOrDefault(r, "name", "Anonymous")

	// Extract optional limit with default
	limit := zh.QueryParamAsOrDefault(r, "limit", 10)

	return zh.R.JSON(w, http.StatusOK, zh.M{
		"user_id":   userID,
		"active":    active,
		"score":     score,
		"name":      name,
		"limit":     limit,
		"extracted": "Individual parameters using QueryParamAs[T]()",
	})
}
