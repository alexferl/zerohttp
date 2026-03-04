package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	zh "github.com/alexferl/zerohttp"
)

func main() {
	app := zh.New()

	// Basic string extraction
	app.GET("/users/{id}", zh.HandlerFunc(getUser))

	// Multiple parameters
	app.GET("/users/{userID}/posts/{postID}", zh.HandlerFunc(getUserPost))

	// Numeric IDs with type conversion
	app.GET("/items/{itemID}", zh.HandlerFunc(getItem))

	// Optional path parameter with default value
	app.GET("/products/{$}", zh.HandlerFunc(listProducts))
	app.GET("/products/{category}", zh.HandlerFunc(listProductsByCategory))

	fmt.Println("Try these endpoints:")
	fmt.Println("  GET /users/123                      - String param extraction")
	fmt.Println("  GET /users/42/posts/99              - Multiple params")
	fmt.Println("  GET /items/456                      - Typed int extraction")
	fmt.Println("  GET /products                       - All products")
	fmt.Println("  GET /products/electronics           - Filtered by category")

	log.Fatal(app.Start())
}

// getUser demonstrates basic string parameter extraction
func getUser(w http.ResponseWriter, r *http.Request) error {
	// Simple string extraction
	id := zh.Param(r, "id")

	return zh.R.JSON(w, 200, zh.M{
		"user_id": id,
		"type":    "string",
	})
}

// getUserPost demonstrates extracting multiple path parameters
func getUserPost(w http.ResponseWriter, r *http.Request) error {
	// Extract both parameters as typed ints
	userID, err := zh.ParamAs[int](r, "userID")
	if err != nil {
		return zh.R.ProblemDetail(w, zh.NewProblemDetail(400, "Invalid userID"))
	}

	postID, err := zh.ParamAs[int](r, "postID")
	if err != nil {
		return zh.R.ProblemDetail(w, zh.NewProblemDetail(400, "Invalid postID"))
	}

	return zh.R.JSON(w, 200, zh.M{
		"user_id": userID,
		"post_id": postID,
		"message": fmt.Sprintf("Fetched post %d for user %d", postID, userID),
	})
}

// getItem demonstrates typed parameter extraction with error handling
func getItem(w http.ResponseWriter, r *http.Request) error {
	itemID, err := zh.ParamAs[int](r, "itemID")
	if err != nil {
		return zh.R.ProblemDetail(w, zh.NewProblemDetail(400, "Invalid itemID"))
	}

	return zh.R.JSON(w, 200, zh.M{
		"item_id": itemID,
		"found":   true,
	})
}

// listProducts demonstrates listing all products (no path params)
func listProducts(w http.ResponseWriter, r *http.Request) error {
	// Parse query params for pagination
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}

	return zh.R.JSON(w, 200, zh.M{
		"category": "all",
		"page":     page,
		"products": []string{"Product A", "Product B"},
	})
}

// listProductsByCategory demonstrates ParamOrDefault for optional filtering
func listProductsByCategory(w http.ResponseWriter, r *http.Request) error {
	// Get category with default fallback
	category := zh.ParamOrDefault(r, "category", "all")

	return zh.R.JSON(w, 200, zh.M{
		"category": category,
		"products": []string{fmt.Sprintf("%s Item 1", category), fmt.Sprintf("%s Item 2", category)},
	})
}
