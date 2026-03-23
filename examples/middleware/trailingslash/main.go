package main

import (
	"log"
	"net/http"

	zh "github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp/middleware/trailingslash"
)

func main() {
	app := zh.New()

	// Strip trailing slash - handler always sees /api/users
	app.GET("/api/users", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.JSON(w, http.StatusOK, map[string]string{
			"message": "Users endpoint",
			"path":    r.URL.Path,
		})
	}),
		trailingslash.New(trailingslash.Config{
			Action: trailingslash.StripAction,
		}),
	)

	// Append trailing slash - handler always sees /docs/
	app.GET("/docs", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.JSON(w, http.StatusOK, map[string]string{
			"message": "Docs endpoint",
			"path":    r.URL.Path,
		})
	}),
		trailingslash.New(trailingslash.Config{
			PreferTrailingSlash: true,
			Action:              trailingslash.AppendAction,
		}),
	)

	log.Fatal(app.Start())
}
