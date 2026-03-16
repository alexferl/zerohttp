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

	// Add content encoding middleware that allows gzip and deflate
	app.Use(middleware.ContentEncoding(config.ContentEncodingConfig{
		Encodings: []string{httpx.ContentEncodingGzip, httpx.ContentEncodingDeflate},
	}))

	app.POST("/api/data", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.JSON(w, http.StatusOK, map[string]string{
			"message": "Data received with valid encoding",
		})
	}))

	log.Fatal(app.Start())
}
