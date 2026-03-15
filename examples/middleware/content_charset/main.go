package main

import (
	"log"
	"net/http"

	zh "github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/middleware"
)

func main() {
	app := zh.New()

	// Add content charset middleware that only allows UTF-8
	app.Use(middleware.ContentCharset(config.ContentCharsetConfig{
		Charsets: []string{"utf-8"},
	}))

	app.POST("/api/data", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.JSON(w, http.StatusOK, map[string]string{
			"message": "Data received with valid charset",
		})
	}))

	log.Fatal(app.Start())
}
