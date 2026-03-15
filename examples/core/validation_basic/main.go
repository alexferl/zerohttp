package main

import (
	"log"
	"net/http"

	zh "github.com/alexferl/zerohttp"
)

type CreateUserRequest struct {
	Name     string `json:"name" validate:"required,min=2,max=50"`
	Email    string `json:"email" validate:"required,email"`
	Age      int    `json:"age" validate:"min=13,max=120"`
	Username string `json:"username" validate:"required,alphanum,min=3,max=20"`
}

type UpdateUserRequest struct {
	Name  *string `json:"name" validate:"omitempty,min=2,max=50"`
	Email *string `json:"email" validate:"omitempty,email"`
	Age   *int    `json:"age" validate:"omitempty,min=13,max=120"`
}

func main() {
	app := zh.New()

	app.POST("/users", zh.HandlerFunc(createUserHandler))
	app.PATCH("/users/{id}", zh.HandlerFunc(updateUserHandler))

	log.Fatal(app.Start())
}

func createUserHandler(w http.ResponseWriter, r *http.Request) error {
	var req CreateUserRequest
	if err := zh.BindAndValidate(r, &req); err != nil {
		return err
	}

	return zh.R.JSON(w, http.StatusCreated, zh.M{
		"message":  "User created",
		"name":     req.Name,
		"email":    req.Email,
		"age":      req.Age,
		"username": req.Username,
	})
}

func updateUserHandler(w http.ResponseWriter, r *http.Request) error {
	var req UpdateUserRequest
	if err := zh.BindAndValidate(r, &req); err != nil {
		return err
	}

	response := zh.M{"message": "User updated"}
	if req.Name != nil {
		response["name"] = *req.Name
	}
	if req.Email != nil {
		response["email"] = *req.Email
	}
	if req.Age != nil {
		response["age"] = *req.Age
	}

	return zh.R.JSON(w, http.StatusOK, response)
}
