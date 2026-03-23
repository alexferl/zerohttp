package main

import (
	"log"
	"net/http"

	zh "github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp/middleware/basicauth"
)

func main() {
	app := zh.New()

	// Add basic auth middleware with credentials
	app.Use(basicauth.New(basicauth.Config{
		Realm: "Protected Area",
		Credentials: map[string]string{
			"admin": "secret",
			"user":  "password",
		},
	}))

	app.GET("/", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.JSON(w, http.StatusOK, map[string]string{
			"message": "Welcome to the protected area!",
		})
	}))

	app.GET("/api/data", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.JSON(w, http.StatusOK, map[string]string{
			"data": "This is protected data",
		})
	}))

	log.Fatal(app.Start())
}
