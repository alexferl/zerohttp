package main

import (
	"log"
	"net/http"

	zh "github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp/middleware/setheader"
)

func main() {
	app := zh.New()

	// Set headers globally
	app.Use(setheader.New(setheader.Config{
		Headers: map[string]string{
			"X-Custom-Header": "global-value",
		},
	}))

	// This endpoint has global headers
	app.GET("/", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.JSON(w, http.StatusOK, map[string]string{
			"message": "Check response headers",
		})
	}))

	// This endpoint has additional route-specific headers
	app.GET("/api", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.JSON(w, http.StatusOK, map[string]string{
			"message": "API endpoint",
		})
	}),
		setheader.New(setheader.Config{
			Headers: map[string]string{
				"X-API-Version": "v1",
			},
		}),
	)

	log.Fatal(app.Start())
}
