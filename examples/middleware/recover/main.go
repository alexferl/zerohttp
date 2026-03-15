package main

import (
	"errors"
	"log"
	"net/http"

	zh "github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/middleware"
)

func main() {
	app := zh.New()

	// Note: Recover middleware is enabled by default via DefaultMiddlewares.
	// You don't need to add it manually unless you want custom configuration.

	// This endpoint panics with a nil pointer dereference
	app.GET("/panic/nil", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		var ptr *string
		_ = *ptr // This will panic
		return nil
	}))

	// This endpoint explicitly calls panic()
	app.GET("/panic/explicit", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		panic("something went terribly wrong!")
	}))

	// This endpoint panics with an error
	app.GET("/panic/error", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		panic(errors.New("critical system error"))
	}))

	// This endpoint works normally
	app.GET("/healthy", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.JSON(w, http.StatusOK, map[string]string{
			"status": "ok",
		})
	}))

	// Example with custom recover configuration
	customApp := zh.New()
	customApp.Use(middleware.Recover(customApp.Logger(), config.RecoverConfig{
		EnableStackTrace: false, // Disable stack traces
		StackSize:        1024,  // Smaller stack buffer
	}))

	customApp.GET("/no-stack", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		panic("panic without stack trace")
	}))

	// Use the main app for this example
	_ = customApp

	log.Fatal(app.Start())
}
