package main

import (
	"log"
	"net/http"

	zh "github.com/alexferl/zerohttp"
)

type SearchRequest struct {
	Query    string   `query:"q"`
	Category string   `query:"category"`
	Tags     []string `query:"tags"`
}

type PaginationRequest struct {
	Page    int   `query:"page"`
	Limit   int   `query:"limit"`
	Offset  int   `query:"offset"`
	IsAdmin *bool `query:"is_admin"`
}

type FilterRequest struct {
	IDs      []int    `query:"id"`
	Statuses []string `query:"status"`
	MinPrice float64  `query:"min_price"`
	MaxPrice float64  `query:"max_price"`
	InStock  bool     `query:"in_stock"`
}

type EmbeddedPagination struct {
	Page  int `query:"page"`
	Limit int `query:"limit"`
}

type ListRequest struct {
	EmbeddedPagination
	Search string `query:"search"`
}

func main() {
	app := zh.New()

	app.GET("/", zh.HandlerFunc(indexHandler))
	app.GET("/search", zh.HandlerFunc(searchHandler))
	app.GET("/items", zh.HandlerFunc(itemsHandler))
	app.GET("/products", zh.HandlerFunc(productsHandler))
	app.GET("/users", zh.HandlerFunc(listUsersHandler))
	app.GET("/extract", zh.HandlerFunc(extractHandler))

	log.Fatal(app.Start())
}

func indexHandler(w http.ResponseWriter, _ *http.Request) error {
	html := `<!DOCTYPE html>
<html>
<head>
    <title>Query Parameter Binding Examples</title>
    <style>
        body { font-family: sans-serif; max-width: 800px; margin: 40px auto; padding: 20px; }
        h1 { color: #333; }
        code { background: #f4f4f4; padding: 2px 6px; border-radius: 3px; }
        pre { background: #f4f4f4; padding: 15px; border-radius: 5px; }
        .endpoint { margin: 20px 0; padding: 15px; background: #f9f9f9; border-left: 4px solid #28a745; }
        .method { font-weight: bold; color: #28a745; }
    </style>
</head>
<body>
    <h1>Query Parameter Binding Examples</h1>

    <div class="endpoint">
        <span class="method">GET</span> <code>/search</code>
        <p>Basic search with strings and slices.</p>
        <pre>curl "http://localhost:8080/search?q=golang&amp;category=tech&amp;tags=api&amp;tags=web"</pre>
    </div>

    <div class="endpoint">
        <span class="method">GET</span> <code>/items</code>
        <p>Pagination with optional boolean pointer.</p>
        <pre>curl "http://localhost:8080/items?page=2&amp;limit=50&amp;is_admin=true"</pre>
    </div>

    <div class="endpoint">
        <span class="method">GET</span> <code>/products</code>
        <p>Filter with int slices and float values.</p>
        <pre>curl "http://localhost:8080/products?id=1&amp;id=2&amp;status=active&amp;min_price=10.99"</pre>
    </div>

    <div class="endpoint">
        <span class="method">GET</span> <code>/users</code>
        <p>Embedded struct with pagination.</p>
        <pre>curl "http://localhost:8080/users?search=john&amp;page=1&amp;limit=20"</pre>
    </div>

    <div class="endpoint">
        <span class="method">GET</span> <code>/extract</code>
        <p>Individual parameter extraction helpers.</p>
        <pre>curl "http://localhost:8080/extract?user_id=123&amp;active=true&amp;score=95.5"</pre>
    </div>
</body>
</html>`
	return zh.R.HTML(w, http.StatusOK, html)
}

func searchHandler(w http.ResponseWriter, r *http.Request) error {
	var req SearchRequest
	if err := zh.B.Query(r, &req); err != nil {
		return zh.R.ProblemDetail(w, zh.NewProblemDetail(http.StatusBadRequest, err.Error()))
	}

	if req.Category == "" {
		req.Category = "all"
	}

	return zh.R.JSON(w, http.StatusOK, zh.M{
		"query":    req.Query,
		"category": req.Category,
		"tags":     req.Tags,
	})
}

func itemsHandler(w http.ResponseWriter, r *http.Request) error {
	var req PaginationRequest
	if err := zh.B.Query(r, &req); err != nil {
		return zh.R.ProblemDetail(w, zh.NewProblemDetail(http.StatusBadRequest, err.Error()))
	}

	if req.Page < 1 {
		req.Page = 1
	}
	if req.Limit < 1 {
		req.Limit = 20
	}
	if req.Limit > 100 {
		req.Limit = 100
	}

	response := zh.M{
		"page":   req.Page,
		"limit":  req.Limit,
		"offset": req.Offset,
	}

	if req.IsAdmin != nil {
		response["is_admin"] = *req.IsAdmin
		response["is_admin_provided"] = true
	} else {
		response["is_admin_provided"] = false
	}

	return zh.R.JSON(w, http.StatusOK, response)
}

func productsHandler(w http.ResponseWriter, r *http.Request) error {
	var req FilterRequest
	if err := zh.B.Query(r, &req); err != nil {
		return zh.R.ProblemDetail(w, zh.NewProblemDetail(http.StatusBadRequest, err.Error()))
	}

	return zh.R.JSON(w, http.StatusOK, zh.M{
		"ids":       req.IDs,
		"statuses":  req.Statuses,
		"min_price": req.MinPrice,
		"max_price": req.MaxPrice,
		"in_stock":  req.InStock,
	})
}

func listUsersHandler(w http.ResponseWriter, r *http.Request) error {
	var req ListRequest
	if err := zh.B.Query(r, &req); err != nil {
		return zh.R.ProblemDetail(w, zh.NewProblemDetail(http.StatusBadRequest, err.Error()))
	}

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
	})
}

func extractHandler(w http.ResponseWriter, r *http.Request) error {
	userID, err := zh.QueryParamAs[int](r, "user_id")
	if err != nil {
		return zh.R.ProblemDetail(w, zh.NewProblemDetail(http.StatusBadRequest, "Invalid user_id: "+err.Error()))
	}

	score, err := zh.QueryParamAs[float64](r, "score")
	if err != nil {
		return zh.R.ProblemDetail(w, zh.NewProblemDetail(http.StatusBadRequest, "Invalid score: "+err.Error()))
	}

	active, err := zh.QueryParamAs[bool](r, "active")
	if err != nil {
		return zh.R.ProblemDetail(w, zh.NewProblemDetail(http.StatusBadRequest, "Invalid active: "+err.Error()))
	}

	name := zh.QueryParamAsOrDefault(r, "name", "Anonymous")
	limit := zh.QueryParamAsOrDefault(r, "limit", 10)

	return zh.R.JSON(w, http.StatusOK, zh.M{
		"user_id": userID,
		"active":  active,
		"score":   score,
		"name":    name,
		"limit":   limit,
	})
}
