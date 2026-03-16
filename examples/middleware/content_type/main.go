package main

import (
	"log"
	"net/http"

	zh "github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/httpx"
	"github.com/alexferl/zerohttp/middleware"
)

func main() {
	app := zh.New()

	// Add content type middleware that only allows JSON
	app.Use(middleware.ContentType(config.ContentTypeConfig{
		ContentTypes: []string{httpx.MIMEApplicationJSON},
	}))

	app.POST("/api/data", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.JSON(w, http.StatusOK, map[string]string{
			"message": "JSON data received",
		})
	}))

	log.Fatal(app.Start())
}
