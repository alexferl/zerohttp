package main

import (
	"log"
	"math/rand"
	"net/http"
	"time"

	zh "github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/middleware"
)

func main() {
	app := zh.New()

	app.Use(middleware.Timeout(config.TimeoutConfig{
		Timeout: 2 * time.Second,
		Message: "Request timeout",
	}))

	app.GET("/fast", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.JSON(w, http.StatusOK, map[string]string{
			"message": "Fast response",
		})
	}))

	app.GET("/slow", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		ctx := r.Context()
		processTime := time.Duration(rand.Intn(4)+1) * time.Second

		select {
		case <-ctx.Done():
			return nil
		case <-time.After(processTime):
			return zh.R.JSON(w, http.StatusOK, map[string]string{
				"message": "Slow response completed",
			})
		}
	}))

	log.Fatal(app.Start())
}
