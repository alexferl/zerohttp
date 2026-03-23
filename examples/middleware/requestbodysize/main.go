package main

import (
	"io"
	"log"
	"net/http"

	zh "github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp/middleware/requestbodysize"
)

func main() {
	// Configure request body size limit via server config
	// This affects the default RequestBodySize middleware
	app := zh.New(zh.Config{
		RequestBodySize: requestbodysize.Config{
			MaxBytes: 100 * 1024, // 100KB limit (default is 1MB)
		},
	})

	// This endpoint reads the body - middleware auto-returns 413 if too large
	app.POST("/api/data", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			return err
		}

		return zh.R.JSON(w, http.StatusOK, map[string]any{
			"message": "Data received successfully",
			"size":    len(body),
		})
	}))

	// This endpoint is also protected by the 100KB limit
	app.POST("/api/webhook", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		_, err := io.ReadAll(r.Body)
		if err != nil {
			return err
		}
		return zh.R.JSON(w, http.StatusOK, map[string]string{
			"message": "Webhook received",
		})
	}))

	log.Fatal(app.Start())
}
