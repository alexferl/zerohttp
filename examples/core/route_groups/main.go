package main

import (
	"log"
	"net/http"

	zh "github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp/middleware/basicauth"
	"github.com/alexferl/zerohttp/middleware/ratelimit"
	"github.com/alexferl/zerohttp/middleware/requestid"
)

func main() {
	app := zh.New()

	// Public routes
	app.GET("/", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.Text(w, http.StatusOK, "Welcome to the public API")
	}))

	// API group with common middleware
	app.Group(func(api zh.Router) {
		api.Use(requestid.New(zh.DefaultConfig.RequestID))

		// Users endpoints
		api.GET("/users", zh.HandlerFunc(listUsers))
		api.POST("/users", zh.HandlerFunc(createUser))
		api.GET("/users/{id}", zh.HandlerFunc(getUser))
		api.PUT("/users/{id}", zh.HandlerFunc(updateUser))
		api.DELETE("/users/{id}", zh.HandlerFunc(deleteUser))
	})

	// Admin group with authentication middleware
	adminCfg := basicauth.Config{}
	adminCfg.Credentials = map[string]string{"admin": "admin"}

	app.Group(func(admin zh.Router) {
		admin.Use(basicauth.New(adminCfg))

		admin.GET("/admin/dashboard", zh.HandlerFunc(dashboard))
		admin.GET("/admin/settings", zh.HandlerFunc(settings))
		admin.POST("/admin/users/{id}/ban", zh.HandlerFunc(banUser))
	})

	// Nested group example - API v2 with rate limiting
	app.Group(func(v2 zh.Router) {
		v2.Use(ratelimit.New())

		// Public v2 endpoints
		v2.GET("/v2/public/status", zh.HandlerFunc(status))

		// Authenticated v2 endpoints (nested group)
		v2.Group(func(auth zh.Router) {
			auth.Use(basicauth.New(adminCfg))

			auth.GET("/v2/profile", zh.HandlerFunc(getProfile))
			auth.PUT("/v2/profile", zh.HandlerFunc(updateProfile))
		})
	})

	log.Fatal(app.Start())
}

func listUsers(w http.ResponseWriter, r *http.Request) error {
	return zh.R.JSON(w, http.StatusOK, zh.M{"users": []string{"alice", "bob"}})
}

func createUser(w http.ResponseWriter, r *http.Request) error {
	return zh.R.JSON(w, http.StatusCreated, zh.M{"id": "123", "created": true})
}

func getUser(w http.ResponseWriter, r *http.Request) error {
	id := r.PathValue("id")
	return zh.R.JSON(w, http.StatusOK, zh.M{"id": id, "name": "User " + id})
}

func updateUser(w http.ResponseWriter, r *http.Request) error {
	id := r.PathValue("id")
	return zh.R.JSON(w, http.StatusOK, zh.M{"id": id, "updated": true})
}

func deleteUser(w http.ResponseWriter, r *http.Request) error {
	return zh.R.JSON(w, http.StatusOK, zh.M{"deleted": true})
}

func dashboard(w http.ResponseWriter, r *http.Request) error {
	return zh.R.JSON(w, http.StatusOK, zh.M{"admin": true, "page": "dashboard"})
}

func settings(w http.ResponseWriter, r *http.Request) error {
	return zh.R.JSON(w, http.StatusOK, zh.M{"admin": true, "page": "settings"})
}

func banUser(w http.ResponseWriter, r *http.Request) error {
	id := r.PathValue("id")
	return zh.R.JSON(w, http.StatusOK, zh.M{"banned": id})
}

func status(w http.ResponseWriter, r *http.Request) error {
	return zh.R.JSON(w, http.StatusOK, zh.M{"version": "2.0", "status": "ok"})
}

func getProfile(w http.ResponseWriter, r *http.Request) error {
	return zh.R.JSON(w, http.StatusOK, zh.M{"user": "current", "profile": "data"})
}

func updateProfile(w http.ResponseWriter, r *http.Request) error {
	return zh.R.JSON(w, http.StatusOK, zh.M{"updated": true})
}
