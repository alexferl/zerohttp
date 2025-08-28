package main

import (
	"context"
	"log"
	"net/http"

	zh "github.com/alexferl/zerohttp"
	"github.com/danielgtaylor/huma/v2"
)

// GreetingInput represents the greeting operation input.
type GreetingInput struct {
	Name string `path:"name" maxLength:"30" example:"world" doc:"Name to greet"`
}

// GreetingOutput represents the greeting operation output.
type GreetingOutput struct {
	Body struct {
		Message string `json:"message" example:"Hello, world!" doc:"Greeting message"`
	}
}

func main() {
	// Create zerohttp server
	app := zh.New()

	// Add a regular route to test middleware
	app.GET("/", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.JSON(w, 200, zh.M{"message": "Hello, World!"})
	}))

	// Create Huma API with zerohttp
	config := huma.DefaultConfig("My API", "1.0.0")
	api := New(app, config)

	// Register operation
	huma.Register(api, huma.Operation{
		OperationID: "greeting",
		Method:      http.MethodGet,
		Path:        "/greeting/{name}",
		Summary:     "Get a greeting",
	}, func(ctx context.Context, input *GreetingInput) (*GreetingOutput, error) {
		resp := &GreetingOutput{}
		resp.Body.Message = "Hello, " + input.Name + "!"
		return resp, nil
	})

	// Start server
	log.Fatal(app.Start())
}
