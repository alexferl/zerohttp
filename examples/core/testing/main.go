package main

import (
	"log"
	"net/http"

	zh "github.com/alexferl/zerohttp"
)

type User struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

var users = map[string]User{
	"1": {ID: "1", Name: "Alice", Email: "alice@example.com"},
	"2": {ID: "2", Name: "Bob", Email: "bob@example.com"},
}

func main() {
	app := zh.New()

	app.GET("/users/{id}", zh.HandlerFunc(getUser))
	app.POST("/users", zh.HandlerFunc(createUser))
	app.GET("/health", zh.HandlerFunc(healthCheck))

	log.Fatal(app.Start())
}

func getUser(w http.ResponseWriter, r *http.Request) error {
	id := zh.Param(r, "id")

	user, ok := users[id]
	if !ok {
		return zh.R.ProblemDetail(w, zh.NewProblemDetail(http.StatusNotFound, "User not found"))
	}

	return zh.R.JSON(w, http.StatusOK, user)
}

func createUser(w http.ResponseWriter, r *http.Request) error {
	var req struct {
		Name  string `json:"name" validate:"required"`
		Email string `json:"email" validate:"required,email"`
	}

	if err := zh.B.JSON(r.Body, &req); err != nil {
		return err
	}

	if err := zh.Validate.Struct(&req); err != nil {
		return err
	}

	user := User{
		ID:    "3",
		Name:  req.Name,
		Email: req.Email,
	}

	return zh.R.JSON(w, http.StatusCreated, user)
}

func healthCheck(w http.ResponseWriter, _ *http.Request) error {
	return zh.R.JSON(w, http.StatusOK, zh.M{"status": "ok"})
}
