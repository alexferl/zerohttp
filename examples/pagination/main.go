package main

import (
	"log"
	"net/http"
	"strconv"
	"strings"

	zh "github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp/pagination"
)

// Product represents a product in our store
type Product struct {
	ID    int     `json:"id"`
	Name  string  `json:"name"`
	Price float64 `json:"price"`
}

// ProductStore holds our products
type ProductStore struct {
	products []Product
}

func NewProductStore() *ProductStore {
	// Generate sample products
	products := make([]Product, 100)
	for i := 0; i < 100; i++ {
		products[i] = Product{
			ID:    i + 1,
			Name:  "Product " + strconv.Itoa(i+1),
			Price: float64(i+1) * 10.99,
		}
	}
	return &ProductStore{products: products}
}

// ListProducts handles GET /products with pagination
func (s *ProductStore) ListProducts(w http.ResponseWriter, r *http.Request) error {
	var req struct {
		pagination.Request
	}

	if err := zh.B.Query(r, &req); err != nil {
		return zh.R.ProblemDetail(w, zh.NewProblemDetail(http.StatusBadRequest, err.Error()))
	}

	params := req.Request.Params().Defaults()

	// Calculate slice bounds
	start := params.Offset()
	end := start + params.PerPage
	if end > len(s.products) {
		end = len(s.products)
	}

	// Get the page of products
	items := s.products[start:end]

	// Write pagination headers
	params.WriteHeaders(w, r.URL, len(s.products))

	return zh.R.JSON(w, http.StatusOK, items)
}

// SearchProducts handles GET /products/search with filtering and pagination
func (s *ProductStore) SearchProducts(w http.ResponseWriter, r *http.Request) error {
	var req struct {
		pagination.Request
		Query string `query:"q"`
	}

	if err := zh.B.Query(r, &req); err != nil {
		return zh.R.ProblemDetail(w, zh.NewProblemDetail(http.StatusBadRequest, err.Error()))
	}

	params := req.Request.Params().Defaults()

	// Filter products by name
	var filtered []Product
	if req.Query != "" {
		for _, p := range s.products {
			if strings.Contains(strings.ToLower(p.Name), strings.ToLower(req.Query)) {
				filtered = append(filtered, p)
			}
		}
	} else {
		filtered = s.products
	}

	// Calculate slice bounds
	start := params.Offset()
	end := start + params.PerPage
	if end > len(filtered) {
		end = len(filtered)
	}
	if start > len(filtered) {
		start = len(filtered)
	}

	// Get the page of products
	items := filtered[start:end]

	// Write pagination headers
	params.WriteHeaders(w, r.URL, len(filtered))

	return zh.R.JSON(w, http.StatusOK, zh.M{
		"query":   req.Query,
		"results": items,
		"total":   len(filtered),
	})
}

func main() {
	store := NewProductStore()

	app := zh.New()

	app.GET("/products", zh.HandlerFunc(store.ListProducts))
	app.GET("/products/search", zh.HandlerFunc(store.SearchProducts))

	log.Fatal(app.Start())
}
