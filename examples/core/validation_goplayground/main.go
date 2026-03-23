package main

import (
	"fmt"
	"log"
	"net/http"
	"reflect"

	zh "github.com/alexferl/zerohttp"
	"github.com/go-playground/validator/v10"
)

// goPlaygroundValidator wraps go-playground/validator to implement zh.Validator
type goPlaygroundValidator struct {
	v *validator.Validate
}

// Struct validates a struct using go-playground/validator
func (g *goPlaygroundValidator) Struct(dst any) error {
	return g.v.Struct(dst)
}

// Register adds a custom validation function
func (g *goPlaygroundValidator) Register(name string, fn func(reflect.Value, string) error) {
	// Wrap the zerohttp validation func for go-playground
	g.v.RegisterValidation(name, func(fl validator.FieldLevel) bool {
		err := fn(fl.Field(), fl.Param())
		return err == nil
	})
}

// UserRequest demonstrates validation with go-playground validator tags
type UserRequest struct {
	Name     string `json:"name" validate:"required,min=2,max=50"`
	Email    string `json:"email" validate:"required,email"`
	Age      int    `json:"age" validate:"gte=13,lte=120"`
	Username string `json:"username" validate:"required,alphanum,min=3,max=20"`
	// go-playground has built-in validators for common patterns
	Phone string `json:"phone,omitempty" validate:"omitempty,e164"`
	URL   string `json:"url,omitempty" validate:"omitempty,url"`
}

var app *zh.Server

func main() {
	gpv := validator.New()

	customValidator := &goPlaygroundValidator{v: gpv}

	app = zh.New(zh.Config{
		Validator: customValidator,
	})

	app.POST("/users", zh.HandlerFunc(createUserHandler))

	log.Fatal(app.Start())
}

func createUserHandler(w http.ResponseWriter, r *http.Request) error {
	var req UserRequest
	if err := zh.B.JSON(r.Body, &req); err != nil {
		return err
	}

	if err := app.Validator().Struct(&req); err != nil {
		if validationErrors, ok := err.(validator.ValidationErrors); ok {
			errors := make(map[string]string)
			for _, e := range validationErrors {
				errors[e.Field()] = fmt.Sprintf("failed %s validation", e.Tag())
			}
			return zh.NewProblemDetail(http.StatusUnprocessableEntity, "Validation failed").
				Set("errors", errors).
				Render(w)
		}
		return err
	}

	return zh.R.JSON(w, http.StatusCreated, zh.M{
		"message":  "User created",
		"name":     req.Name,
		"email":    req.Email,
		"username": req.Username,
	})
}
