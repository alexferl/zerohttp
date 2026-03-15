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

	// Add CORS middleware with custom configuration
	app.Use(middleware.CORS(config.CORSConfig{
		AllowedOrigins:   []string{"http://localhost:3000", "https://example.com"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"X-Request-Id"},
		AllowCredentials: true,
		MaxAge:           86400,
	}))

	app.GET("/api/data", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.JSON(w, http.StatusOK, map[string]string{
			"message": "CORS-enabled endpoint",
		})
	}))

	app.POST("/api/data", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.JSON(w, http.StatusOK, map[string]string{
			"message": "Data created",
		})
	}))

	log.Fatal(app.Start())
}
