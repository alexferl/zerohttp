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

	app.GET("/users/{id}", zh.HandlerFunc(getUser))
	app.GET("/users/{userID}/posts/{postID}", zh.HandlerFunc(getUserPost))
	app.GET("/items/{itemID}", zh.HandlerFunc(getItem))
	app.GET("/products/{$}", zh.HandlerFunc(listProducts))
	app.GET("/products/{category}", zh.HandlerFunc(listProductsByCategory))

	log.Fatal(app.Start())
}

func getUser(w http.ResponseWriter, r *http.Request) error {
	id := zh.Param(r, "id")

	return zh.R.JSON(w, 200, zh.M{
		"user_id": id,
		"type":    "string",
	})
}

func getUserPost(w http.ResponseWriter, r *http.Request) error {
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

func listProducts(w http.ResponseWriter, r *http.Request) error {
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

func listProductsByCategory(w http.ResponseWriter, r *http.Request) error {
	category := zh.ParamOrDefault(r, "category", "all")

	return zh.R.JSON(w, 200, zh.M{
		"category": category,
		"products": []string{fmt.Sprintf("%s Item 1", category), fmt.Sprintf("%s Item 2", category)},
	})
}
