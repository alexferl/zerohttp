package main

import (
	"log"
	"net/http"

	zh "github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp/middleware/mediatype"
)

func main() {
	app := zh.New()

	// Accept multiple vendor media types for API versioning
	app.Use(mediatype.New(mediatype.Config{
		AllowedTypes: []string{
			"application/vnd.api.v1+json",
			"application/vnd.api.v2+json",
		},
		DefaultType: "application/vnd.api.v1+json",
	}))

	// Return different response formats based on Accept header
	app.GET("/api/users", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		accept := r.Header.Get("Accept")

		if accept == "application/vnd.api.v2+json" {
			// V2: full user data with metadata
			return zh.R.JSON(w, http.StatusOK, map[string]any{
				"data": []map[string]any{
					{"id": "1", "name": "Alice", "email": "alice@example.com", "role": "admin"},
					{"id": "2", "name": "Bob", "email": "bob@example.com", "role": "user"},
				},
				"meta": map[string]int{
					"total": 2,
					"page":  1,
				},
			})
		}

		// V1 (default): simple user list
		return zh.R.JSON(w, http.StatusOK, []map[string]string{
			{"id": "1", "name": "Alice"},
			{"id": "2", "name": "Bob"},
		})
	}))

	log.Fatal(app.Start())
}
