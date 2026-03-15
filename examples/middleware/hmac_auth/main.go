package main

import (
	"log"
	"net/http"
	"time"

	zh "github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/middleware"
)

func main() {
	credentials := map[string][]string{
		"service-a": {"super-secret-key-at-least-32-bytes-long!!"},
		"service-b": {"another-secret-key-for-service-b-abc123"},
	}

	app := zh.New()

	app.GET("/health", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.JSON(w, http.StatusOK, zh.M{"status": "healthy"})
	}))

	app.Group(func(api zh.Router) {
		api.Use(middleware.HMACAuth(config.HMACAuthConfig{
			CredentialStore: func(accessKeyID string) []string {
				return credentials[accessKeyID]
			},
			MaxSkew: 5 * time.Minute,
		}))

		api.GET("/api/data", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
			accessKeyID := middleware.GetHMACAccessKeyID(r)
			return zh.R.JSON(w, http.StatusOK, zh.M{
				"message":          "Hello from protected API",
				"authenticated_as": accessKeyID,
			})
		}))
	})

	app.Group(func(api zh.Router) {
		api.Use(middleware.HMACAuth(config.HMACAuthConfig{
			CredentialStore: func(accessKeyID string) []string {
				return credentials[accessKeyID]
			},
			MaxSkew:            5 * time.Minute,
			AllowPresignedURLs: true,
		}))

		api.GET("/api/download", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
			accessKeyID := middleware.GetHMACAccessKeyID(r)
			return zh.R.JSON(w, http.StatusOK, zh.M{
				"message":          "Download access granted",
				"authenticated_as": accessKeyID,
			})
		}))
	})

	log.Fatal(app.Start())
}
