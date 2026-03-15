package main

import (
	"log"
	"net/http"

	zh "github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp/config"
)

func main() {
	app := zh.New(config.Config{
		RequestLogger: config.RequestLoggerConfig{
			LogRequestBody:  true,
			LogResponseBody: true,
			MaxBodySize:     1024,
			Fields: []config.LogField{
				config.FieldMethod,
				config.FieldPath,
				config.FieldStatus,
				config.FieldDurationHuman,
				config.FieldRequestBody,
				config.FieldResponseBody,
			},
		},
	})

	app.POST("/api/login", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.JSON(w, http.StatusOK, zh.M{
			"status":  "success",
			"message": "Login successful",
			"token":   "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
		})
	}))

	app.GET("/health", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.JSON(w, http.StatusOK, zh.M{
			"status": "healthy",
		})
	}))

	log.Fatal(app.Start())
}
