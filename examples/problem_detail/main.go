package main

import (
	"log"
	"net/http"

	zh "github.com/alexferl/zerohttp"
)

type MyError struct {
	Code  string `json:"code"`
	Field string `json:"field"`
	Issue string `json:"issue"`
}

func main() {
	app := zh.New()

	app.GET("/", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.JSON(w, 200, zh.M{"message": "Hello, World!"})
	}))

	// Regular ProblemDetail error
	app.GET("/error", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		problem := zh.NewProblemDetail(404, "The requested resource was not found")
		problem.Type = "https://example.com/probs/not-found"
		problem.Instance = "/error"
		return zh.R.ProblemDetail(w, problem)
	}))

	// Default ValidationError example
	app.POST("/validate-simple", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		errors := []zh.ValidationError{
			{Detail: "must be positive", Pointer: "#/age"},
			{Detail: "invalid email", Field: "email"},
		}

		problem := zh.NewValidationProblemDetail("Validation failed", errors)
		return zh.R.ProblemDetail(w, problem)
	}))

	// Custom error example
	app.POST("/validate-custom", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		errors := []MyError{
			{Code: "INVALID_AGE", Field: "age", Issue: "must be positive"},
			{Code: "BAD_EMAIL", Field: "email", Issue: "invalid format"},
		}

		problem := zh.NewValidationProblemDetail("Custom validation failed", errors)
		return zh.R.ProblemDetail(w, problem)
	}))

	log.Fatal(app.Start())
}
