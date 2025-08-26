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

	// Using standard net/http TimeoutHandler - more verbose
	app.GET("/slow-std", http.TimeoutHandler(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			processTime := time.Duration(rand.Intn(4)+1) * time.Second

			select {
			case <-ctx.Done():
				return
			case <-time.After(processTime):
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(`{"message": "Completed with stdlib TimeoutHandler"}`))
				if err != nil {
					log.Fatalf("failed to write response: %v", err)
				}
			}
		}),
		2*time.Second,
		`Request timeout from stdlib`,
	))

	// Using zerohttp timeout middleware - more concise
	app.Group(func(r zh.Router) {
		r.Use(middleware.Timeout(
			config.WithTimeoutDuration(2*time.Second),
			config.WithTimeoutMessage(`Request timeout from zerohttp`),
		))

		r.GET("/slow", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
			ctx := r.Context()
			processTime := time.Duration(rand.Intn(4)+1) * time.Second

			select {
			case <-ctx.Done():
				return nil
			case <-time.After(processTime):
				return zh.Render.JSON(w, 200, zh.M{"message": "Completed with zerohttp timeout middleware"})
			}
		}))
	})

	log.Println("Test both timeout approaches:")
	log.Println("  - GET /slow-std  (stdlib TimeoutHandler)")
	log.Println("  - GET /slow      (zerohttp timeout middleware)")

	log.Fatal(app.Start())
}
